package database

import (
	"context"
	"errors"
	"fmt"

	coreenvironment "github.com/flagstack/flagstack/backend/internal/environment"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EnvironmentRepository struct {
	pool *pgxpool.Pool
}

func NewEnvironmentRepository(pool *pgxpool.Pool) *EnvironmentRepository {
	return &EnvironmentRepository{pool: pool}
}

func (r *EnvironmentRepository) List(ctx context.Context, organisationID, projectID string) ([]coreenvironment.Environment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT e.id::text, e.organisation_id::text, e.project_id::text, e.name, e.key, e.description, e.created_at
		FROM environments e
		JOIN projects p ON p.id = e.project_id AND p.organisation_id = e.organisation_id
		WHERE e.organisation_id = $1 AND e.project_id = $2 AND p.archived_at IS NULL
		ORDER BY e.created_at, e.id
	`, organisationID, projectID)
	if err != nil {
		return nil, fmt.Errorf("list environments: %w", err)
	}
	defer rows.Close()

	environments := make([]coreenvironment.Environment, 0)
	for rows.Next() {
		var environment coreenvironment.Environment
		if err := rows.Scan(
			&environment.ID,
			&environment.OrganisationID,
			&environment.ProjectID,
			&environment.Name,
			&environment.Key,
			&environment.Description,
			&environment.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan environment: %w", err)
		}
		environments = append(environments, environment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate environments: %w", err)
	}
	return environments, nil
}

func (r *EnvironmentRepository) Create(ctx context.Context, organisationID, projectID string, input coreenvironment.CreateInput) (coreenvironment.Environment, error) {
	var environment coreenvironment.Environment
	err := r.pool.QueryRow(ctx, `
		WITH target_project AS (
			SELECT id, organisation_id
			FROM projects
			WHERE organisation_id = $1 AND id = $2 AND archived_at IS NULL
		)
		INSERT INTO environments (organisation_id, project_id, name, key, description)
		SELECT organisation_id, id, $3, $4, $5
		FROM target_project
		RETURNING id::text, organisation_id::text, project_id::text, name, key, description, created_at
	`, organisationID, projectID, input.Name, input.Key, input.Description).Scan(
		&environment.ID,
		&environment.OrganisationID,
		&environment.ProjectID,
		&environment.Name,
		&environment.Key,
		&environment.Description,
		&environment.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return coreenvironment.Environment{}, coreenvironment.ErrKeyConflict
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return coreenvironment.Environment{}, coreenvironment.ErrProjectNotFound
		}
		return coreenvironment.Environment{}, fmt.Errorf("create environment: %w", err)
	}
	return environment, nil
}
