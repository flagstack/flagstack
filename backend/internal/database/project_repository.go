package database

import (
	"context"
	"fmt"

	switchonyourcodeent "github.com/switchonyourcode/switchonyourcode/backend/ent"
	entenvironment "github.com/switchonyourcode/switchonyourcode/backend/ent/environment"
	entfeatureflag "github.com/switchonyourcode/switchonyourcode/backend/ent/featureflag"
	entproject "github.com/switchonyourcode/switchonyourcode/backend/ent/project"
	coreproject "github.com/switchonyourcode/switchonyourcode/backend/internal/project"
	"github.com/google/uuid"
)

type ProjectRepository struct {
	client *switchonyourcodeent.Client
}

func NewProjectRepository(client *switchonyourcodeent.Client) *ProjectRepository {
	return &ProjectRepository{client: client}
}

func (r *ProjectRepository) List(ctx context.Context, organisationID string) ([]coreproject.Project, error) {
	organisationUUID, err := uuid.Parse(organisationID)
	if err != nil {
		return nil, fmt.Errorf("parse organisation ID: %w", err)
	}

	entities, err := r.client.Project.Query().
		Where(
			entproject.OrganisationID(organisationUUID),
			entproject.ArchivedAtIsNil(),
		).
		Order(
			switchonyourcodeent.Desc(entproject.FieldCreatedAt),
			switchonyourcodeent.Desc(entproject.FieldID),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}

	projects := make([]coreproject.Project, 0, len(entities))
	for _, entity := range entities {
		environmentCount, err := r.client.Environment.Query().
			Where(
				entenvironment.OrganisationID(organisationUUID),
				entenvironment.ProjectID(entity.ID),
			).
			Count(ctx)
		if err != nil {
			return nil, fmt.Errorf("count project environments: %w", err)
		}

		featureFlagCount, err := r.client.FeatureFlag.Query().
			Where(
				entfeatureflag.OrganisationID(organisationUUID),
				entfeatureflag.ProjectID(entity.ID),
				entfeatureflag.ArchivedAtIsNil(),
			).
			Count(ctx)
		if err != nil {
			return nil, fmt.Errorf("count project feature flags: %w", err)
		}

		projects = append(projects, coreproject.Project{
			ID:               entity.ID.String(),
			OrganisationID:   entity.OrganisationID.String(),
			Name:             entity.Name,
			Key:              entity.Key,
			Description:      entity.Description,
			EnvironmentCount: environmentCount,
			FeatureFlagCount: featureFlagCount,
			CreatedAt:        entity.CreatedAt,
		})
	}

	return projects, nil
}

func (r *ProjectRepository) Create(ctx context.Context, organisationID string, input coreproject.CreateInput) (coreproject.Project, error) {
	organisationUUID, err := uuid.Parse(organisationID)
	if err != nil {
		return coreproject.Project{}, fmt.Errorf("parse organisation ID: %w", err)
	}

	entity, err := r.client.Project.Create().
		SetOrganisationID(organisationUUID).
		SetName(input.Name).
		SetKey(input.Key).
		SetDescription(input.Description).
		Save(ctx)
	if err != nil {
		if switchonyourcodeent.IsConstraintError(err) {
			return coreproject.Project{}, coreproject.ErrKeyConflict
		}
		return coreproject.Project{}, fmt.Errorf("create project: %w", err)
	}

	return coreproject.Project{
		ID:             entity.ID.String(),
		OrganisationID: entity.OrganisationID.String(),
		Name:           entity.Name,
		Key:            entity.Key,
		Description:    entity.Description,
		CreatedAt:      entity.CreatedAt,
	}, nil
}
