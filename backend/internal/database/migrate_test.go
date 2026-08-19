package database

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestMigrateIntegration(t *testing.T) {
	databaseURL := os.Getenv("FLAGSTACK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("FLAGSTACK_TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pool, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer pool.Close()

	client := NewEntClient(pool)
	defer client.Close()

	if err := Migrate(ctx, pool, client); err != nil {
		t.Fatalf("Migrate() first error = %v", err)
	}
	if err := Migrate(ctx, pool, client); err != nil {
		t.Fatalf("Migrate() second error = %v", err)
	}

	var gooseMetadataExists bool
	if err := pool.QueryRow(ctx, `SELECT to_regclass('public.goose_db_version') IS NOT NULL`).Scan(&gooseMetadataExists); err != nil {
		t.Fatalf("check Goose metadata: %v", err)
	}
	if gooseMetadataExists {
		t.Fatal("goose_db_version still exists after Ent migration")
	}

	if _, err := pool.Exec(ctx, `TRUNCATE organisations CASCADE`); err != nil {
		t.Fatalf("reset migration test tables: %v", err)
	}

	var firstOrganisationID, secondOrganisationID, projectID string
	if err := pool.QueryRow(ctx, `INSERT INTO organisations (name, slug) VALUES ('First', 'first') RETURNING id::text`).Scan(&firstOrganisationID); err != nil {
		t.Fatalf("create first organisation: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO organisations (name, slug) VALUES ('Second', 'second') RETURNING id::text`).Scan(&secondOrganisationID); err != nil {
		t.Fatalf("create second organisation: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO projects (organisation_id, name, key)
		VALUES ($1, 'First project', 'first-project')
		RETURNING id::text
	`, firstOrganisationID).Scan(&projectID); err != nil {
		t.Fatalf("create project: %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO environments (organisation_id, project_id, name, key)
		VALUES ($1, $2, 'Invalid', 'invalid')
	`, secondOrganisationID, projectID)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
		t.Fatalf("cross-tenant environment insert error = %v, want foreign-key violation", err)
	}
}
