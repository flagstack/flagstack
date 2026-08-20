package database

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	coreflagconfig "github.com/switchonyourcode/switchonyourcode/backend/internal/flagconfig"
)

func TestFlagConfigRepositoryIntegration(t *testing.T) {
	databaseURL := os.Getenv("SWITCHONYOURCODE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("SWITCHONYOURCODE_TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer pool.Close()

	entClient := NewEntClient(pool)
	defer entClient.Close()

	if _, err := pool.Exec(ctx, `TRUNCATE environment_flag_configs, feature_flags, environments, projects, organisations CASCADE`); err != nil {
		t.Fatalf("reset flag config tables: %v", err)
	}

	var organisationID, projectID, environmentID, featureFlagID string
	if err := pool.QueryRow(ctx, `INSERT INTO organisations (name, slug) VALUES ('Example', 'example') RETURNING id::text`).Scan(&organisationID); err != nil {
		t.Fatalf("create organisation: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO projects (organisation_id, name, key)
		VALUES ($1, 'Web application', 'web-app')
		RETURNING id::text
	`, organisationID).Scan(&projectID); err != nil {
		t.Fatalf("create project: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO environments (organisation_id, project_id, name, key)
		VALUES ($1, $2, 'Production', 'production')
		RETURNING id::text
	`, organisationID, projectID).Scan(&environmentID); err != nil {
		t.Fatalf("create environment: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO feature_flags (organisation_id, project_id, name, key, kind, default_value)
		VALUES ($1, $2, 'Checkout', 'checkout.new-flow', 'boolean', $3)
		RETURNING id::text
	`, organisationID, projectID, json.RawMessage(`false`)).Scan(&featureFlagID); err != nil {
		t.Fatalf("create feature flag: %v", err)
	}

	repository := NewFlagConfigRepository(entClient)
	configs, err := repository.List(ctx, organisationID, projectID)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(configs) != 0 {
		t.Fatalf("List() length = %d, want 0 for untouched sparse config", len(configs))
	}

	unchanged, err := repository.SetEnabled(ctx, organisationID, projectID, environmentID, featureFlagID, false)
	if err != nil {
		t.Fatalf("SetEnabled(false) sparse error = %v", err)
	}
	if unchanged.Enabled || unchanged.Revision != 0 {
		t.Fatalf("SetEnabled(false) sparse config = %#v", unchanged)
	}

	enabled, err := repository.SetEnabled(ctx, organisationID, projectID, environmentID, featureFlagID, true)
	if err != nil {
		t.Fatalf("SetEnabled(true) error = %v", err)
	}
	if !enabled.Enabled || enabled.Revision != 1 {
		t.Fatalf("SetEnabled(true) config = %#v", enabled)
	}

	enabledAgain, err := repository.SetEnabled(ctx, organisationID, projectID, environmentID, featureFlagID, true)
	if err != nil {
		t.Fatalf("SetEnabled(true) idempotent error = %v", err)
	}
	if enabledAgain.Revision != 1 {
		t.Fatalf("SetEnabled(true) idempotent revision = %d, want 1", enabledAgain.Revision)
	}

	disabled, err := repository.SetEnabled(ctx, organisationID, projectID, environmentID, featureFlagID, false)
	if err != nil {
		t.Fatalf("SetEnabled(false) error = %v", err)
	}
	if disabled.Enabled || disabled.Revision != 2 {
		t.Fatalf("SetEnabled(false) config = %#v", disabled)
	}

	configs, err = repository.List(ctx, organisationID, projectID)
	if err != nil {
		t.Fatalf("List() after updates error = %v", err)
	}
	if len(configs) != 1 || configs[0].Revision != 2 {
		t.Fatalf("List() configs = %#v", configs)
	}

	if _, err := repository.SetEnabled(ctx, organisationID, projectID, "00000000-0000-0000-0000-000000000000", featureFlagID, true); !errors.Is(err, coreflagconfig.ErrEnvironmentNotFound) {
		t.Fatalf("missing environment error = %v, want ErrEnvironmentNotFound", err)
	}
	if _, err := repository.SetEnabled(ctx, organisationID, projectID, environmentID, "00000000-0000-0000-0000-000000000000", true); !errors.Is(err, coreflagconfig.ErrFeatureFlagNotFound) {
		t.Fatalf("missing feature flag error = %v, want ErrFeatureFlagNotFound", err)
	}
}
