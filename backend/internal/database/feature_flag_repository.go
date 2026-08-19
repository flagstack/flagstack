package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	corefeatureflag "github.com/flagstack/flagstack/backend/internal/featureflag"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FeatureFlagRepository struct {
	pool *pgxpool.Pool
}

func NewFeatureFlagRepository(pool *pgxpool.Pool) *FeatureFlagRepository {
	return &FeatureFlagRepository{pool: pool}
}

func (r *FeatureFlagRepository) List(ctx context.Context, organisationID, projectID string) ([]corefeatureflag.FeatureFlag, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT f.id::text, f.organisation_id::text, f.project_id::text, f.name, f.key,
		       f.description, f.kind, f.default_value::text, f.created_at
		FROM feature_flags f
		JOIN projects p ON p.id = f.project_id AND p.organisation_id = f.organisation_id
		WHERE f.organisation_id = $1 AND f.project_id = $2
		  AND f.archived_at IS NULL AND p.archived_at IS NULL
		ORDER BY f.created_at, f.id
	`, organisationID, projectID)
	if err != nil {
		return nil, fmt.Errorf("list feature flags: %w", err)
	}
	defer rows.Close()

	flags := make([]corefeatureflag.FeatureFlag, 0)
	for rows.Next() {
		var flag corefeatureflag.FeatureFlag
		var defaultValue string
		if err := rows.Scan(
			&flag.ID,
			&flag.OrganisationID,
			&flag.ProjectID,
			&flag.Name,
			&flag.Key,
			&flag.Description,
			&flag.Kind,
			&defaultValue,
			&flag.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan feature flag: %w", err)
		}
		flag.DefaultValue = json.RawMessage(defaultValue)
		flags = append(flags, flag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate feature flags: %w", err)
	}
	return flags, nil
}

func (r *FeatureFlagRepository) Create(ctx context.Context, organisationID, projectID string, input corefeatureflag.CreateInput) (corefeatureflag.FeatureFlag, error) {
	var flag corefeatureflag.FeatureFlag
	var defaultValue string
	err := r.pool.QueryRow(ctx, `
		WITH target_project AS (
			SELECT id, organisation_id
			FROM projects
			WHERE organisation_id = $1 AND id = $2 AND archived_at IS NULL
		)
		INSERT INTO feature_flags (organisation_id, project_id, name, key, description, kind, default_value)
		SELECT organisation_id, id, $3, $4, $5, $6, $7::jsonb
		FROM target_project
		RETURNING id::text, organisation_id::text, project_id::text, name, key,
		          description, kind, default_value::text, created_at
	`, organisationID, projectID, input.Name, input.Key, input.Description, input.Kind, string(input.DefaultValue)).Scan(
		&flag.ID,
		&flag.OrganisationID,
		&flag.ProjectID,
		&flag.Name,
		&flag.Key,
		&flag.Description,
		&flag.Kind,
		&defaultValue,
		&flag.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return corefeatureflag.FeatureFlag{}, corefeatureflag.ErrKeyConflict
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return corefeatureflag.FeatureFlag{}, corefeatureflag.ErrProjectNotFound
		}
		return corefeatureflag.FeatureFlag{}, fmt.Errorf("create feature flag: %w", err)
	}
	flag.DefaultValue = json.RawMessage(defaultValue)
	return flag, nil
}
