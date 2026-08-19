package database

import (
	"context"
	"errors"
	"fmt"

	coreproject "github.com/flagstack/flagstack/backend/internal/project"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProjectRepository struct {
	pool *pgxpool.Pool
}

func NewProjectRepository(pool *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{pool: pool}
}

func (r *ProjectRepository) List(ctx context.Context, organisationID string) ([]coreproject.Project, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			p.id::text,
			p.organisation_id::text,
			p.name,
			p.key,
			p.description,
			(SELECT count(*) FROM environments e WHERE e.project_id = p.id),
			(SELECT count(*) FROM feature_flags f WHERE f.project_id = p.id AND f.archived_at IS NULL),
			p.created_at
		FROM projects p
		WHERE p.organisation_id = $1 AND p.archived_at IS NULL
		ORDER BY p.created_at DESC, p.id DESC
	`, organisationID)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	projects := make([]coreproject.Project, 0)
	for rows.Next() {
		var project coreproject.Project
		if err := rows.Scan(
			&project.ID,
			&project.OrganisationID,
			&project.Name,
			&project.Key,
			&project.Description,
			&project.EnvironmentCount,
			&project.FeatureFlagCount,
			&project.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		projects = append(projects, project)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate projects: %w", err)
	}
	return projects, nil
}

func (r *ProjectRepository) Create(ctx context.Context, organisationID string, input coreproject.CreateInput) (coreproject.Project, error) {
	var project coreproject.Project
	err := r.pool.QueryRow(ctx, `
		INSERT INTO projects (organisation_id, name, key, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, organisation_id::text, name, key, description, created_at
	`, organisationID, input.Name, input.Key, input.Description).Scan(
		&project.ID,
		&project.OrganisationID,
		&project.Name,
		&project.Key,
		&project.Description,
		&project.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return coreproject.Project{}, coreproject.ErrKeyConflict
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return coreproject.Project{}, errors.New("project insert returned no row")
		}
		return coreproject.Project{}, fmt.Errorf("create project: %w", err)
	}
	return project, nil
}
