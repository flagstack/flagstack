package database

import (
	"context"
	"fmt"

	switchonyourcodeent "github.com/switchonyourcode/switchonyourcode/backend/ent"
	entenvironment "github.com/switchonyourcode/switchonyourcode/backend/ent/environment"
	entproject "github.com/switchonyourcode/switchonyourcode/backend/ent/project"
	coreenvironment "github.com/switchonyourcode/switchonyourcode/backend/internal/environment"
	"github.com/google/uuid"
)

type EnvironmentRepository struct {
	client *switchonyourcodeent.Client
}

func NewEnvironmentRepository(client *switchonyourcodeent.Client) *EnvironmentRepository {
	return &EnvironmentRepository{client: client}
}

func (r *EnvironmentRepository) List(ctx context.Context, organisationID, projectID string) ([]coreenvironment.Environment, error) {
	organisationUUID, projectUUID, err := parseProjectScope(organisationID, projectID)
	if err != nil {
		return nil, err
	}

	active, err := r.client.Project.Query().
		Where(
			entproject.ID(projectUUID),
			entproject.OrganisationID(organisationUUID),
			entproject.ArchivedAtIsNil(),
		).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("check environment project: %w", err)
	}
	if !active {
		return []coreenvironment.Environment{}, nil
	}

	entities, err := r.client.Environment.Query().
		Where(
			entenvironment.OrganisationID(organisationUUID),
			entenvironment.ProjectID(projectUUID),
		).
		Order(
			switchonyourcodeent.Asc(entenvironment.FieldCreatedAt),
			switchonyourcodeent.Asc(entenvironment.FieldID),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list environments: %w", err)
	}

	environments := make([]coreenvironment.Environment, 0, len(entities))
	for _, entity := range entities {
		environments = append(environments, coreenvironment.Environment{
			ID:             entity.ID.String(),
			OrganisationID: entity.OrganisationID.String(),
			ProjectID:      entity.ProjectID.String(),
			Name:           entity.Name,
			Key:            entity.Key,
			Description:    entity.Description,
			CreatedAt:      entity.CreatedAt,
		})
	}
	return environments, nil
}

func (r *EnvironmentRepository) Create(ctx context.Context, organisationID, projectID string, input coreenvironment.CreateInput) (coreenvironment.Environment, error) {
	organisationUUID, projectUUID, err := parseProjectScope(organisationID, projectID)
	if err != nil {
		return coreenvironment.Environment{}, err
	}

	active, err := r.client.Project.Query().
		Where(
			entproject.ID(projectUUID),
			entproject.OrganisationID(organisationUUID),
			entproject.ArchivedAtIsNil(),
		).
		Exist(ctx)
	if err != nil {
		return coreenvironment.Environment{}, fmt.Errorf("check environment project: %w", err)
	}
	if !active {
		return coreenvironment.Environment{}, coreenvironment.ErrProjectNotFound
	}

	entity, err := r.client.Environment.Create().
		SetOrganisationID(organisationUUID).
		SetProjectID(projectUUID).
		SetName(input.Name).
		SetKey(input.Key).
		SetDescription(input.Description).
		Save(ctx)
	if err != nil {
		if switchonyourcodeent.IsConstraintError(err) {
			return coreenvironment.Environment{}, coreenvironment.ErrKeyConflict
		}
		return coreenvironment.Environment{}, fmt.Errorf("create environment: %w", err)
	}

	return coreenvironment.Environment{
		ID:             entity.ID.String(),
		OrganisationID: entity.OrganisationID.String(),
		ProjectID:      entity.ProjectID.String(),
		Name:           entity.Name,
		Key:            entity.Key,
		Description:    entity.Description,
		CreatedAt:      entity.CreatedAt,
	}, nil
}

func parseProjectScope(organisationID, projectID string) (uuid.UUID, uuid.UUID, error) {
	organisationUUID, err := uuid.Parse(organisationID)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("parse organisation ID: %w", err)
	}
	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("parse project ID: %w", err)
	}
	return organisationUUID, projectUUID, nil
}
