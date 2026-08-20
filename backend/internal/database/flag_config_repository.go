package database

import (
	"context"
	"fmt"

	switchonyourcodeent "github.com/switchonyourcode/switchonyourcode/backend/ent"
	entenvironment "github.com/switchonyourcode/switchonyourcode/backend/ent/environment"
	entenvironmentflagconfig "github.com/switchonyourcode/switchonyourcode/backend/ent/environmentflagconfig"
	entfeatureflag "github.com/switchonyourcode/switchonyourcode/backend/ent/featureflag"
	coreflagconfig "github.com/switchonyourcode/switchonyourcode/backend/internal/flagconfig"
	"github.com/google/uuid"
)

type FlagConfigRepository struct {
	client *switchonyourcodeent.Client
}

func NewFlagConfigRepository(client *switchonyourcodeent.Client) *FlagConfigRepository {
	return &FlagConfigRepository{client: client}
}

func (r *FlagConfigRepository) List(ctx context.Context, organisationID, projectID string) ([]coreflagconfig.Config, error) {
	organisationUUID, projectUUID, err := parseProjectScope(organisationID, projectID)
	if err != nil {
		return nil, err
	}

	entities, err := r.client.EnvironmentFlagConfig.Query().
		Where(
			entenvironmentflagconfig.OrganisationID(organisationUUID),
			entenvironmentflagconfig.ProjectID(projectUUID),
		).
		Order(
			switchonyourcodeent.Asc(entenvironmentflagconfig.FieldEnvironmentID),
			switchonyourcodeent.Asc(entenvironmentflagconfig.FieldFeatureFlagID),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list environment flag configs: %w", err)
	}

	configs := make([]coreflagconfig.Config, 0, len(entities))
	for _, entity := range entities {
		configs = append(configs, flagConfigFromEntity(entity))
	}
	return configs, nil
}

func (r *FlagConfigRepository) SetEnabled(ctx context.Context, organisationID, projectID, environmentID, featureFlagID string, enabled bool) (coreflagconfig.Config, error) {
	organisationUUID, projectUUID, err := parseProjectScope(organisationID, projectID)
	if err != nil {
		return coreflagconfig.Config{}, err
	}
	environmentUUID, err := uuid.Parse(environmentID)
	if err != nil {
		return coreflagconfig.Config{}, fmt.Errorf("parse environment ID: %w", err)
	}
	featureFlagUUID, err := uuid.Parse(featureFlagID)
	if err != nil {
		return coreflagconfig.Config{}, fmt.Errorf("parse feature flag ID: %w", err)
	}

	if err := r.validateTargets(ctx, organisationUUID, projectUUID, environmentUUID, featureFlagUUID); err != nil {
		return coreflagconfig.Config{}, err
	}

	entity, err := r.find(ctx, organisationUUID, projectUUID, environmentUUID, featureFlagUUID)
	if err != nil && !switchonyourcodeent.IsNotFound(err) {
		return coreflagconfig.Config{}, fmt.Errorf("find environment flag config: %w", err)
	}
	if switchonyourcodeent.IsNotFound(err) {
		if !enabled {
			return coreflagconfig.Config{
				EnvironmentID: environmentID,
				FeatureFlagID: featureFlagID,
				Enabled:       false,
				Revision:      0,
			}, nil
		}

		created, createErr := r.client.EnvironmentFlagConfig.Create().
			SetOrganisationID(organisationUUID).
			SetProjectID(projectUUID).
			SetEnvironmentID(environmentUUID).
			SetFeatureFlagID(featureFlagUUID).
			SetEnabled(true).
			Save(ctx)
		if createErr == nil {
			return flagConfigFromEntity(created), nil
		}
		if !switchonyourcodeent.IsConstraintError(createErr) {
			return coreflagconfig.Config{}, fmt.Errorf("create environment flag config: %w", createErr)
		}

		entity, err = r.find(ctx, organisationUUID, projectUUID, environmentUUID, featureFlagUUID)
		if err != nil {
			return coreflagconfig.Config{}, fmt.Errorf("reload environment flag config after concurrent create: %w", err)
		}
	}

	if entity.Enabled == enabled {
		return flagConfigFromEntity(entity), nil
	}

	updated, err := r.client.EnvironmentFlagConfig.UpdateOne(entity).
		SetEnabled(enabled).
		SetRevision(entity.Revision + 1).
		Save(ctx)
	if err != nil {
		return coreflagconfig.Config{}, fmt.Errorf("update environment flag config: %w", err)
	}
	return flagConfigFromEntity(updated), nil
}

func (r *FlagConfigRepository) validateTargets(ctx context.Context, organisationID, projectID, environmentID, featureFlagID uuid.UUID) error {
	environmentExists, err := r.client.Environment.Query().
		Where(
			entenvironment.ID(environmentID),
			entenvironment.OrganisationID(organisationID),
			entenvironment.ProjectID(projectID),
		).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("check flag config environment: %w", err)
	}
	if !environmentExists {
		return coreflagconfig.ErrEnvironmentNotFound
	}

	featureFlagExists, err := r.client.FeatureFlag.Query().
		Where(
			entfeatureflag.ID(featureFlagID),
			entfeatureflag.OrganisationID(organisationID),
			entfeatureflag.ProjectID(projectID),
			entfeatureflag.ArchivedAtIsNil(),
		).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("check flag config feature flag: %w", err)
	}
	if !featureFlagExists {
		return coreflagconfig.ErrFeatureFlagNotFound
	}
	return nil
}

func (r *FlagConfigRepository) find(ctx context.Context, organisationID, projectID, environmentID, featureFlagID uuid.UUID) (*switchonyourcodeent.EnvironmentFlagConfig, error) {
	return r.client.EnvironmentFlagConfig.Query().
		Where(
			entenvironmentflagconfig.OrganisationID(organisationID),
			entenvironmentflagconfig.ProjectID(projectID),
			entenvironmentflagconfig.EnvironmentID(environmentID),
			entenvironmentflagconfig.FeatureFlagID(featureFlagID),
		).
		Only(ctx)
}

func flagConfigFromEntity(entity *switchonyourcodeent.EnvironmentFlagConfig) coreflagconfig.Config {
	return coreflagconfig.Config{
		EnvironmentID: entity.EnvironmentID.String(),
		FeatureFlagID: entity.FeatureFlagID.String(),
		Enabled:       entity.Enabled,
		Revision:      entity.Revision,
	}
}
