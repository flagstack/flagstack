package database

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	coreproject "github.com/switchonyourcode/switchonyourcode/backend/internal/project"
)

func TestProjectRepositoryIntegration(t *testing.T) {
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

	if _, err := pool.Exec(ctx, `TRUNCATE projects, organisations CASCADE`); err != nil {
		t.Fatalf("reset project tables: %v", err)
	}

	var organisationID string
	if err := pool.QueryRow(ctx, `INSERT INTO organisations (name, slug) VALUES ('Example', 'example') RETURNING id::text`).Scan(&organisationID); err != nil {
		t.Fatalf("create organisation: %v", err)
	}

	repository := NewProjectRepository(entClient)
	created, err := repository.Create(ctx, organisationID, coreproject.CreateInput{
		Name:        "Web application",
		Key:         "web-app",
		Description: "Frontend",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if _, err := repository.Create(ctx, organisationID, coreproject.CreateInput{Name: "Duplicate", Key: "web-app"}); !errors.Is(err, coreproject.ErrKeyConflict) {
		t.Fatalf("duplicate Create() error = %v, want ErrKeyConflict", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO environments (organisation_id, project_id, name, key)
		VALUES ($1, $2, 'Production', 'production')
	`, organisationID, created.ID); err != nil {
		t.Fatalf("create environment: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO feature_flags (organisation_id, project_id, name, key, kind, default_value)
		VALUES ($1, $2, 'New dashboard', 'new-dashboard', 'boolean', 'false'::jsonb)
	`, organisationID, created.ID); err != nil {
		t.Fatalf("create feature flag: %v", err)
	}

	projects, err := repository.List(ctx, organisationID)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("List() length = %d, want 1", len(projects))
	}
	if projects[0].EnvironmentCount != 1 || projects[0].FeatureFlagCount != 1 {
		t.Fatalf("List() counts = environments %d, flags %d", projects[0].EnvironmentCount, projects[0].FeatureFlagCount)
	}
}
