package database

import (
	"context"
	"fmt"

	flagstackent "github.com/flagstack/flagstack/backend/ent"
	flagstackmigrate "github.com/flagstack/flagstack/backend/ent/migrate"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Migrate(ctx context.Context, pool *pgxpool.Pool, client *flagstackent.Client) error {
	if err := prepareLegacyGooseSchema(ctx, pool); err != nil {
		return err
	}

	if err := client.Schema.Create(
		ctx,
		flagstackmigrate.WithDropColumn(false),
		flagstackmigrate.WithDropIndex(false),
		flagstackmigrate.WithForeignKeys(true),
	); err != nil {
		return fmt.Errorf("apply Ent schema migration: %w", err)
	}

	if err := ensureTenantConstraints(ctx, pool); err != nil {
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
