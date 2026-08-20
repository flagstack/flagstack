package database

import (
	"context"
	"fmt"

	switchonyourcodeent "github.com/switchonyourcode/switchonyourcode/backend/ent"
	switchonyourcodemigrate "github.com/switchonyourcode/switchonyourcode/backend/ent/migrate"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Migrate(ctx context.Context, pool *pgxpool.Pool, client *switchonyourcodeent.Client) error {
	if err := prepareLegacyGooseSchema(ctx, pool); err != nil {
		return err
	}

	if err := client.Schema.Create(
		ctx,
		switchonyourcodemigrate.WithDropColumn(false),
		switchonyourcodemigrate.WithDropIndex(false),
		switchonyourcodemigrate.WithForeignKeys(true),
	); err != nil {
		return fmt.Errorf("apply Ent schema migration: %w", err)
	}

	if err := ensureTenantConstraints(ctx, pool); err != nil {
		return err
	}
	if err := ensureSDKInvalidationTriggers(ctx, pool); err != nil {
		return err
	}

	if _, err := pool.Exec(ctx, `DROP TABLE IF EXISTS goose_db_version`); err != nil {
		return fmt.Errorf("remove legacy Goose migration metadata: %w", err)
	}
	return nil
}

func prepareLegacyGooseSchema(ctx context.Context, pool *pgxpool.Pool) error {
	const statement = `
DO $$
BEGIN
    IF to_regclass('public.organisation_memberships') IS NOT NULL
       AND NOT EXISTS (
           SELECT 1 FROM information_schema.columns
           WHERE table_schema = 'public' AND table_name = 'organisation_memberships' AND column_name = 'id'
       ) THEN
        ALTER TABLE organisation_memberships ADD COLUMN id uuid DEFAULT uuidv7();
        ALTER TABLE organisation_memberships ALTER COLUMN id SET NOT NULL;
        ALTER TABLE organisation_memberships DROP CONSTRAINT IF EXISTS organisation_memberships_pkey;
        ALTER TABLE organisation_memberships
            ADD CONSTRAINT organisation_memberships_organisation_id_user_id_key UNIQUE (organisation_id, user_id);
        ALTER TABLE organisation_memberships
            ADD CONSTRAINT organisation_memberships_pkey PRIMARY KEY (id);
    END IF;

    IF to_regclass('public.local_credentials') IS NOT NULL
       AND NOT EXISTS (
           SELECT 1 FROM information_schema.columns
           WHERE table_schema = 'public' AND table_name = 'local_credentials' AND column_name = 'id'
       ) THEN
        ALTER TABLE local_credentials ADD COLUMN id uuid DEFAULT uuidv7();
        ALTER TABLE local_credentials ALTER COLUMN id SET NOT NULL;
        ALTER TABLE local_credentials DROP CONSTRAINT IF EXISTS local_credentials_pkey;
        ALTER TABLE local_credentials
            ADD CONSTRAINT local_credentials_user_id_key UNIQUE (user_id);
        ALTER TABLE local_credentials
            ADD CONSTRAINT local_credentials_pkey PRIMARY KEY (id);
    END IF;

    IF to_regclass('public.environment_flag_configs') IS NOT NULL
       AND NOT EXISTS (
           SELECT 1 FROM information_schema.columns
           WHERE table_schema = 'public' AND table_name = 'environment_flag_configs' AND column_name = 'id'
       ) THEN
        ALTER TABLE environment_flag_configs ADD COLUMN id uuid DEFAULT uuidv7();
        ALTER TABLE environment_flag_configs ALTER COLUMN id SET NOT NULL;
        ALTER TABLE environment_flag_configs DROP CONSTRAINT IF EXISTS environment_flag_configs_pkey;
        ALTER TABLE environment_flag_configs
            ADD CONSTRAINT environment_flag_configs_environment_id_feature_flag_id_key UNIQUE (environment_id, feature_flag_id);
        ALTER TABLE environment_flag_configs
            ADD CONSTRAINT environment_flag_configs_pkey PRIMARY KEY (id);
    END IF;
END
$$;`

	if _, err := pool.Exec(ctx, statement); err != nil {
		return fmt.Errorf("prepare legacy Goose schema for Ent: %w", err)
	}
	return nil
}

func ensureTenantConstraints(ctx context.Context, pool *pgxpool.Pool) error {
	statements := []struct {
		name string
		sql  string
	}{
		{
			name: "environments_project_tenant_fkey",
			sql: `ALTER TABLE environments
				ADD CONSTRAINT environments_project_tenant_fkey
				FOREIGN KEY (organisation_id, project_id)
				REFERENCES projects (organisation_id, id) ON DELETE CASCADE`,
		},
		{
			name: "feature_flags_project_tenant_fkey",
			sql: `ALTER TABLE feature_flags
				ADD CONSTRAINT feature_flags_project_tenant_fkey
				FOREIGN KEY (organisation_id, project_id)
				REFERENCES projects (organisation_id, id) ON DELETE CASCADE`,
		},
		{
			name: "environment_flag_configs_environment_tenant_fkey",
			sql: `ALTER TABLE environment_flag_configs
				ADD CONSTRAINT environment_flag_configs_environment_tenant_fkey
				FOREIGN KEY (organisation_id, project_id, environment_id)
				REFERENCES environments (organisation_id, project_id, id) ON DELETE CASCADE`,
		},
		{
			name: "environment_flag_configs_feature_flag_tenant_fkey",
			sql: `ALTER TABLE environment_flag_configs
				ADD CONSTRAINT environment_flag_configs_feature_flag_tenant_fkey
				FOREIGN KEY (organisation_id, project_id, feature_flag_id)
				REFERENCES feature_flags (organisation_id, project_id, id) ON DELETE CASCADE`,
		},
		{
			name: "segments_project_tenant_fkey",
			sql: `ALTER TABLE segments
				ADD CONSTRAINT segments_project_tenant_fkey
				FOREIGN KEY (organisation_id, project_id)
				REFERENCES projects (organisation_id, id) ON DELETE CASCADE`,
		},
		{
			name: "scheduled_flag_changes_environment_tenant_fkey",
			sql: `ALTER TABLE scheduled_flag_changes
				ADD CONSTRAINT scheduled_flag_changes_environment_tenant_fkey
				FOREIGN KEY (organisation_id, project_id, environment_id)
				REFERENCES environments (organisation_id, project_id, id) ON DELETE CASCADE`,
		},
		{
			name: "scheduled_flag_changes_feature_flag_tenant_fkey",
			sql: `ALTER TABLE scheduled_flag_changes
				ADD CONSTRAINT scheduled_flag_changes_feature_flag_tenant_fkey
				FOREIGN KEY (organisation_id, project_id, feature_flag_id)
				REFERENCES feature_flags (organisation_id, project_id, id) ON DELETE CASCADE`,
		},
		{
			name: "sdk_credentials_environment_tenant_fkey",
			sql: `ALTER TABLE sdk_credentials
				ADD CONSTRAINT sdk_credentials_environment_tenant_fkey
				FOREIGN KEY (organisation_id, project_id, environment_id)
				REFERENCES environments (organisation_id, project_id, id) ON DELETE CASCADE`,
		},
	}

	for _, statement := range statements {
		var exists bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM pg_constraint
				WHERE conname = $1
			)
		`, statement.name).Scan(&exists); err != nil {
			return fmt.Errorf("check tenant constraint %s: %w", statement.name, err)
		}
		if exists {
			continue
		}
		if _, err := pool.Exec(ctx, statement.sql); err != nil {
			return fmt.Errorf("create tenant constraint %s: %w", statement.name, err)
		}
	}
	return nil
}

func ensureSDKInvalidationTriggers(ctx context.Context, pool *pgxpool.Pool) error {
	const statement = `
CREATE OR REPLACE FUNCTION switchonyourcode_notify_sdk_invalidation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    project_value uuid;
    environment_value uuid;
    credential_value uuid;
    payload jsonb;
BEGIN
    IF TG_TABLE_NAME = 'projects' THEN
        IF TG_OP = 'DELETE' THEN
            project_value := OLD.id;
        ELSE
            project_value := NEW.id;
        END IF;
    ELSIF TG_TABLE_NAME = 'environments' THEN
        IF TG_OP = 'DELETE' THEN
            project_value := OLD.project_id;
            environment_value := OLD.id;
        ELSE
            project_value := NEW.project_id;
            environment_value := NEW.id;
        END IF;
    ELSIF TG_TABLE_NAME = 'environment_flag_configs' THEN
        IF TG_OP = 'DELETE' THEN
            project_value := OLD.project_id;
            environment_value := OLD.environment_id;
        ELSE
            project_value := NEW.project_id;
            environment_value := NEW.environment_id;
        END IF;
    ELSIF TG_TABLE_NAME = 'sdk_credentials' THEN
        project_value := NEW.project_id;
        environment_value := NEW.environment_id;
        credential_value := NEW.id;
    ELSE
        IF TG_OP = 'DELETE' THEN
            project_value := OLD.project_id;
        ELSE
            project_value := NEW.project_id;
        END IF;
    END IF;

    payload := jsonb_build_object('project_id', project_value::text);
    IF environment_value IS NOT NULL THEN
        payload := payload || jsonb_build_object('environment_id', environment_value::text);
    END IF;
    IF credential_value IS NOT NULL THEN
        payload := payload || jsonb_build_object('credential_id', credential_value::text);
    END IF;

    PERFORM pg_notify('switchonyourcode_sdk_invalidate', payload::text);
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS switchonyourcode_sdk_invalidate_flag_configs ON environment_flag_configs;
CREATE TRIGGER switchonyourcode_sdk_invalidate_flag_configs
AFTER INSERT OR UPDATE OR DELETE ON environment_flag_configs
FOR EACH ROW EXECUTE FUNCTION switchonyourcode_notify_sdk_invalidation();

DROP TRIGGER IF EXISTS switchonyourcode_sdk_invalidate_feature_flags ON feature_flags;
CREATE TRIGGER switchonyourcode_sdk_invalidate_feature_flags
AFTER INSERT OR UPDATE OR DELETE ON feature_flags
FOR EACH ROW EXECUTE FUNCTION switchonyourcode_notify_sdk_invalidation();

DROP TRIGGER IF EXISTS switchonyourcode_sdk_invalidate_segments ON segments;
CREATE TRIGGER switchonyourcode_sdk_invalidate_segments
AFTER INSERT OR UPDATE OR DELETE ON segments
FOR EACH ROW EXECUTE FUNCTION switchonyourcode_notify_sdk_invalidation();

DROP TRIGGER IF EXISTS switchonyourcode_sdk_invalidate_environments ON environments;
CREATE TRIGGER switchonyourcode_sdk_invalidate_environments
AFTER INSERT OR UPDATE OR DELETE ON environments
FOR EACH ROW EXECUTE FUNCTION switchonyourcode_notify_sdk_invalidation();

DROP TRIGGER IF EXISTS switchonyourcode_sdk_invalidate_projects ON projects;
CREATE TRIGGER switchonyourcode_sdk_invalidate_projects
AFTER UPDATE OR DELETE ON projects
FOR EACH ROW EXECUTE FUNCTION switchonyourcode_notify_sdk_invalidation();

DROP TRIGGER IF EXISTS switchonyourcode_sdk_invalidate_credentials ON sdk_credentials;
CREATE TRIGGER switchonyourcode_sdk_invalidate_credentials
AFTER UPDATE OF revoked_at ON sdk_credentials
FOR EACH ROW
WHEN (OLD.revoked_at IS DISTINCT FROM NEW.revoked_at)
EXECUTE FUNCTION switchonyourcode_notify_sdk_invalidation();
`
	if _, err := pool.Exec(ctx, statement); err != nil {
		return fmt.Errorf("install SDK invalidation triggers: %w", err)
	}
	return nil
}
