package database

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestMigrateIntegration(t *testing.T) {
	databaseURL := os.Getenv("SWITCHONYOURCODE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("SWITCHONYOURCODE_TEST_DATABASE_URL is not set")
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

func TestMigrateUpgradesLegacyGooseSchema(t *testing.T) {
	databaseURL := os.Getenv("SWITCHONYOURCODE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("SWITCHONYOURCODE_TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	adminPool, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("Open() admin error = %v", err)
	}
	defer adminPool.Close()

	const databaseName = "switchonyourcode_legacy_ent_test"
	if _, err := adminPool.Exec(ctx, `DROP DATABASE IF EXISTS switchonyourcode_legacy_ent_test WITH (FORCE)`); err != nil {
		t.Fatalf("drop stale legacy database: %v", err)
	}
	if _, err := adminPool.Exec(ctx, `CREATE DATABASE switchonyourcode_legacy_ent_test`); err != nil {
		t.Fatalf("create legacy database: %v", err)
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = adminPool.Exec(cleanupCtx, `DROP DATABASE IF EXISTS switchonyourcode_legacy_ent_test WITH (FORCE)`)
	}()

	legacyURL, err := databaseURLForDatabase(databaseURL, databaseName)
	if err != nil {
		t.Fatalf("build legacy database URL: %v", err)
	}
	legacyPool, err := Open(ctx, legacyURL)
	if err != nil {
		t.Fatalf("Open() legacy error = %v", err)
	}

	legacySchema := `
CREATE TABLE organisation_memberships (
    organisation_id uuid NOT NULL,
    user_id uuid NOT NULL,
    role text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (organisation_id, user_id)
);

CREATE TABLE local_credentials (
    user_id uuid PRIMARY KEY,
    password_hash text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    password_changed_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE environment_flag_configs (
    organisation_id uuid NOT NULL,
    project_id uuid NOT NULL,
    environment_id uuid NOT NULL,
    feature_flag_id uuid NOT NULL,
    enabled boolean NOT NULL DEFAULT false,
    value jsonb,
    revision bigint NOT NULL DEFAULT 1,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (environment_id, feature_flag_id)
);

CREATE TABLE goose_db_version (
    id serial PRIMARY KEY,
    version_id bigint NOT NULL,
    is_applied boolean NOT NULL,
    tstamp timestamp NOT NULL DEFAULT now()
);
INSERT INTO goose_db_version (version_id, is_applied) VALUES (2, true);
`
	if _, err := legacyPool.Exec(ctx, legacySchema); err != nil {
		legacyPool.Close()
		t.Fatalf("create legacy Goose-era schema: %v", err)
	}

	legacyClient := NewEntClient(legacyPool)
	if err := Migrate(ctx, legacyPool, legacyClient); err != nil {
		legacyClient.Close()
		legacyPool.Close()
		t.Fatalf("Migrate() legacy schema error = %v", err)
	}
	if err := Migrate(ctx, legacyPool, legacyClient); err != nil {
		legacyClient.Close()
		legacyPool.Close()
		t.Fatalf("Migrate() legacy schema second error = %v", err)
	}

	for _, table := range []string{"organisation_memberships", "local_credentials", "environment_flag_configs"} {
		var hasID bool
		if err := legacyPool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = 'public' AND table_name = $1 AND column_name = 'id'
			)
		`, table).Scan(&hasID); err != nil {
			legacyClient.Close()
			legacyPool.Close()
			t.Fatalf("check %s id column: %v", table, err)
		}
		if !hasID {
			legacyClient.Close()
			legacyPool.Close()
			t.Fatalf("%s has no Ent id column after migration", table)
		}
	}

	var gooseMetadataExists bool
	if err := legacyPool.QueryRow(ctx, `SELECT to_regclass('public.goose_db_version') IS NOT NULL`).Scan(&gooseMetadataExists); err != nil {
		legacyClient.Close()
		legacyPool.Close()
		t.Fatalf("check legacy Goose metadata: %v", err)
	}
	if gooseMetadataExists {
		legacyClient.Close()
		legacyPool.Close()
		t.Fatal("legacy goose_db_version still exists after migration")
	}

	legacyClient.Close()
	legacyPool.Close()
}

func databaseURLForDatabase(databaseURL, databaseName string) (string, error) {
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return "", fmt.Errorf("parse database URL: %w", err)
	}
	parsed.Path = "/" + databaseName
	return parsed.String(), nil
}
