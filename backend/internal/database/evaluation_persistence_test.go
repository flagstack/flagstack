package database

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestEvaluationPersistenceSchemaIntegration(t *testing.T) {
	databaseURL := os.Getenv("SWITCHONYOURCODE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("SWITCHONYOURCODE_TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer pool.Close()

	client := NewEntClient(pool)
	defer client.Close()
	if err := Migrate(ctx, pool, client); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	if _, err := pool.Exec(ctx, `TRUNCATE organisations CASCADE`); err != nil {
		t.Fatalf("reset evaluation persistence tables: %v", err)
	}

	var firstOrganisationID, secondOrganisationID string
	if err := pool.QueryRow(ctx, `INSERT INTO organisations (name, slug) VALUES ('First', 'first') RETURNING id::text`).Scan(&firstOrganisationID); err != nil {
		t.Fatalf("create first organisation: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO organisations (name, slug) VALUES ('Second', 'second') RETURNING id::text`).Scan(&secondOrganisationID); err != nil {
		t.Fatalf("create second organisation: %v", err)
	}

	var projectID, environmentID, featureFlagID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO projects (organisation_id, name, key)
		VALUES ($1, 'Web application', 'web-app')
		RETURNING id::text
	`, firstOrganisationID).Scan(&projectID); err != nil {
		t.Fatalf("create project: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO environments (organisation_id, project_id, name, key)
		VALUES ($1, $2, 'Production', 'production')
		RETURNING id::text
	`, firstOrganisationID, projectID).Scan(&environmentID); err != nil {
		t.Fatalf("create environment: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO feature_flags (organisation_id, project_id, name, key, kind, default_value, variants)
		VALUES ($1, $2, 'Checkout', 'checkout', 'boolean', 'false'::jsonb, '[{"key":"beta","value":true}]'::jsonb)
		RETURNING id::text
	`, firstOrganisationID, projectID).Scan(&featureFlagID); err != nil {
		t.Fatalf("create feature flag with variants: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO environment_flag_configs (
			organisation_id, project_id, environment_id, feature_flag_id, enabled, policy
		) VALUES (
			$1, $2, $3, $4, true,
			'{"fallthrough":{"rollout":[{"variant":"on","weight":10000},{"variant":"off","weight":90000}]}}'::jsonb
		)
	`, firstOrganisationID, projectID, environmentID, featureFlagID); err != nil {
		t.Fatalf("create environment policy: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO segments (organisation_id, project_id, name, key, conditions)
		VALUES ($1, $2, 'Beta customers', 'beta-customers', '[{"attribute":"plan","operator":"equals","value":"beta"}]'::jsonb)
	`, firstOrganisationID, projectID); err != nil {
		t.Fatalf("create segment: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO scheduled_flag_changes (
			organisation_id, project_id, environment_id, feature_flag_id, execute_at, patch
		) VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP + interval '1 hour', '{"enabled":true}'::jsonb)
	`, firstOrganisationID, projectID, environmentID, featureFlagID); err != nil {
		t.Fatalf("create scheduled flag change: %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO segments (organisation_id, project_id, name, key, conditions)
		VALUES ($1, $2, 'Invalid', 'invalid-segment', '[]'::jsonb)
	`, secondOrganisationID, projectID)
	assertForeignKeyViolation(t, err, "cross-tenant segment")

	_, err = pool.Exec(ctx, `
		INSERT INTO scheduled_flag_changes (
			organisation_id, project_id, environment_id, feature_flag_id, execute_at, patch
		) VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP + interval '1 hour', '{"enabled":false}'::jsonb)
	`, secondOrganisationID, projectID, environmentID, featureFlagID)
	assertForeignKeyViolation(t, err, "cross-tenant scheduled change")
}

func assertForeignKeyViolation(t *testing.T, err error, operation string) {
	t.Helper()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
		t.Fatalf("%s error = %v, want foreign-key violation", operation, err)
	}
}
