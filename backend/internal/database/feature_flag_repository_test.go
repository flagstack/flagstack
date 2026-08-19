package database

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	corefeatureflag "github.com/flagstack/flagstack/backend/internal/featureflag"
)

func TestFeatureFlagRepositoryIntegration(t *testing.T) {
	databaseURL := os.Getenv("FLAGSTACK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("FLAGSTACK_TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, `TRUNCATE feature_flags, projects, organisations CASCADE`); err != nil {
		t.Fatalf("reset feature flag tables: %v", err)
	}

	var organisationID, projectID string
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

	repository := NewFeatureFlagRepository(pool)
	created, err := repository.Create(ctx, organisationID, projectID, corefeatureflag.CreateInput{
		Name:         "Checkout flow",
		Key:          "checkout.new-flow",
		Description:  "Controls the new checkout experience.",
		Kind:         "boolean",
		DefaultValue: json.RawMessage(`false`),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Key != "checkout.new-flow" || string(created.DefaultValue) != "false" {
		t.Fatalf("Create() feature flag = %#v", created)
	}

	if _, err := repository.Create(ctx, organisationID, projectID, corefeatureflag.CreateInput{
		Name: "Duplicate", Key: "checkout.new-flow", Kind: "boolean", DefaultValue: json.RawMessage(`true`),
	}); !errors.Is(err, corefeatureflag.ErrKeyConflict) {
		t.Fatalf("duplicate Create() error = %v, want ErrKeyConflict", err)
	}

	if _, err := repository.Create(ctx, organisationID, "00000000-0000-0000-0000-000000000000", corefeatureflag.CreateInput{
		Name: "Missing project", Key: "missing-project", Kind: "boolean", DefaultValue: json.RawMessage(`false`),
	}); !errors.Is(err, corefeatureflag.ErrProjectNotFound) {
		t.Fatalf("missing-project Create() error = %v, want ErrProjectNotFound", err)
	}

	if _, err := repository.Create(ctx, organisationID, projectID, corefeatureflag.CreateInput{
		Name: "Configuration", Key: "checkout.config", Kind: "json", DefaultValue: json.RawMessage(`{"percentage":50}`),
	}); err != nil {
		t.Fatalf("Create() json error = %v", err)
	}

	flags, err := repository.List(ctx, organisationID, projectID)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(flags) != 2 {
		t.Fatalf("List() length = %d, want 2", len(flags))
	}
	if flags[0].ProjectID != projectID || flags[0].OrganisationID != organisationID {
		t.Fatalf("List() tenancy = %#v", flags[0])
	}
}
