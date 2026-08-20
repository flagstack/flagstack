package database

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	coreenvironment "github.com/switchonyourcode/switchonyourcode/backend/internal/environment"
)

func TestEnvironmentRepositoryIntegration(t *testing.T) {
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
		t.Fatalf("reset environment tables: %v", err)
	}

	var organisationID string
	if err := pool.QueryRow(ctx, `INSERT INTO organisations (name, slug) VALUES ('Example', 'example') RETURNING id::text`).Scan(&organisationID); err != nil {
		t.Fatalf("create organisation: %v", err)
	}

	var projectID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO projects (organisation_id, name, key)
		VALUES ($1, 'Web application', 'web-app')
		RETURNING id::text
	`, organisationID).Scan(&projectID); err != nil {
		t.Fatalf("create project: %v", err)
	}

	repository := NewEnvironmentRepository(entClient)
	created, err := repository.Create(ctx, organisationID, projectID, coreenvironment.CreateInput{
		Name:        "Production",
		Key:         "production",
		Description: "Customer-facing environment",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.OrganisationID != organisationID || created.ProjectID != projectID || created.Key != "production" {
		t.Fatalf("Create() environment = %#v", created)
	}

	if _, err := repository.Create(ctx, organisationID, projectID, coreenvironment.CreateInput{Name: "Duplicate", Key: "production"}); !errors.Is(err, coreenvironment.ErrKeyConflict) {
		t.Fatalf("duplicate Create() error = %v, want ErrKeyConflict", err)
	}

	if _, err := repository.Create(ctx, organisationID, "00000000-0000-0000-0000-000000000000", coreenvironment.CreateInput{Name: "Missing", Key: "missing"}); !errors.Is(err, coreenvironment.ErrProjectNotFound) {
		t.Fatalf("missing project Create() error = %v, want ErrProjectNotFound", err)
	}

	environments, err := repository.List(ctx, organisationID, projectID)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(environments) != 1 {
		t.Fatalf("List() length = %d, want 1", len(environments))
	}
	if environments[0].ID != created.ID || environments[0].Description != "Customer-facing environment" {
		t.Fatalf("List() environment = %#v", environments[0])
	}
}
