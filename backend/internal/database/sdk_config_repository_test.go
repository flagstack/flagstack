package database

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	coresdkconfig "github.com/switchonyourcode/switchonyourcode/backend/internal/sdkconfig"
)

func TestSDKConfigurationDeliveryIntegration(t *testing.T) {
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
		t.Fatalf("Migrate() error = %v", err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE organisations CASCADE`); err != nil {
		t.Fatalf("reset SDK configuration tables: %v", err)
	}

	var organisationID, secondOrganisationID string
	if err := pool.QueryRow(ctx, `INSERT INTO organisations (name, slug) VALUES ('First', 'first') RETURNING id::text`).Scan(&organisationID); err != nil {
		t.Fatalf("create organisation: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO organisations (name, slug) VALUES ('Second', 'second') RETURNING id::text`).Scan(&secondOrganisationID); err != nil {
		t.Fatalf("create second organisation: %v", err)
	}

	var projectID, environmentID string
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

	var publicFlagID, privateFlagID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO feature_flags (organisation_id, project_id, name, key, kind, default_value)
		VALUES ($1, $2, 'New checkout', 'new-checkout', 'boolean', 'false'::jsonb)
		RETURNING id::text
	`, organisationID, projectID).Scan(&publicFlagID); err != nil {
		t.Fatalf("create public candidate flag: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO feature_flags (organisation_id, project_id, name, key, kind, default_value)
		VALUES ($1, $2, 'Internal billing', 'internal-billing', 'boolean', 'false'::jsonb)
		RETURNING id::text
	`, organisationID, projectID).Scan(&privateFlagID); err != nil {
		t.Fatalf("create server-only flag: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO segments (organisation_id, project_id, name, key, conditions)
		VALUES
			($1, $2, 'Staff', 'staff', '[{"attribute":"email","operator":"ends_with","value":"@example.com"}]'::jsonb),
			($1, $2, 'Unused', 'unused', '[{"attribute":"plan","operator":"equals","value":"enterprise"}]'::jsonb)
	`, organisationID, projectID); err != nil {
		t.Fatalf("create SDK segments: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO environment_flag_configs (
			organisation_id, project_id, environment_id, feature_flag_id, enabled, policy
		) VALUES
			($1, $2, $3, $4, true, '{"rules":[{"id":"staff","match":"all","conditions":[{"operator":"in_segment","value":"staff"}],"outcome":{"variant":"on"}}],"fallthrough":{"variant":"off"}}'::jsonb),
			($1, $2, $3, $5, true, '{"fallthrough":{"variant":"on"}}'::jsonb)
	`, organisationID, projectID, environmentID, publicFlagID, privateFlagID); err != nil {
		t.Fatalf("create environment flag configuration: %v", err)
	}

	repository := NewSDKConfigRepository(client)
	service, err := coresdkconfig.NewService(repository)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	visible, err := service.SetClientVisible(ctx, organisationID, projectID, publicFlagID, true)
	if err != nil || !visible {
		t.Fatalf("SetClientVisible() = %v, %v, want true, nil", visible, err)
	}

	serverCredential, err := service.CreateCredential(ctx, organisationID, projectID, coresdkconfig.CreateInput{
		Name: "Production backend", Kind: coresdkconfig.KindServer, EnvironmentID: environmentID,
	})
	if err != nil {
		t.Fatalf("create server SDK credential: %v", err)
	}
	clientCredential, err := service.CreateCredential(ctx, organisationID, projectID, coresdkconfig.CreateInput{
		Name: "Production browser", Kind: coresdkconfig.KindClient, EnvironmentID: environmentID,
	})
	if err != nil {
		t.Fatalf("create client SDK credential: %v", err)
	}

	listed, err := service.ListCredentials(ctx, organisationID, projectID)
	if err != nil {
		t.Fatalf("ListCredentials() error = %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("credential count = %d, want 2", len(listed))
	}
	for _, credential := range listed {
		if credential.Kind == coresdkconfig.KindServer && credential.ClientKey != "" {
			t.Fatalf("server credential unexpectedly exposes key material: %#v", credential)
		}
	}

	serverConfiguration, err := service.ConfigurationForKey(ctx, serverCredential.Key)
	if err != nil {
		t.Fatalf("server ConfigurationForKey() error = %v", err)
	}
	if serverConfiguration.SchemaVersion != coresdkconfig.SchemaVersion || serverConfiguration.Environment.ID != environmentID {
		t.Fatalf("server configuration metadata = %#v", serverConfiguration)
	}
	if len(serverConfiguration.Flags) != 2 {
		t.Fatalf("server flag count = %d, want 2", len(serverConfiguration.Flags))
	}

	clientConfiguration, err := service.ConfigurationForKey(ctx, clientCredential.Key)
	if err != nil {
		t.Fatalf("client ConfigurationForKey() error = %v", err)
	}
	if len(clientConfiguration.Flags) != 1 || clientConfiguration.Flags[0].ID != publicFlagID {
		t.Fatalf("client flags = %#v, want only client-visible flag", clientConfiguration.Flags)
	}
	if len(clientConfiguration.Segments) != 1 || clientConfiguration.Segments[0].Key != "staff" {
		t.Fatalf("client segments = %#v, want referenced staff segment only", clientConfiguration.Segments)
	}

	if _, err := service.RevokeCredential(ctx, organisationID, projectID, serverCredential.Credential.ID); err != nil {
		t.Fatalf("revoke server credential: %v", err)
	}
	if _, err := service.ConfigurationForKey(ctx, serverCredential.Key); !errors.Is(err, coresdkconfig.ErrInvalidCredential) {
		t.Fatalf("revoked server key error = %v, want ErrInvalidCredential", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO sdk_credentials (
			organisation_id, project_id, environment_id, name, kind, client_key
		) VALUES ($1, $2, $3, 'Cross tenant', 'client', 'syoc_client_cross_tenant')
	`, secondOrganisationID, projectID, environmentID)
	assertForeignKeyViolation(t, err, "cross-tenant SDK credential")
}
