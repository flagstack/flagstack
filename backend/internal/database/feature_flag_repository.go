package database

import (
	"context"
	"fmt"

	flagstackent "github.com/flagstack/flagstack/backend/ent"
	entfeatureflag "github.com/flagstack/flagstack/backend/ent/featureflag"
	entproject "github.com/flagstack/flagstack/backend/ent/project"
	corefeatureflag "github.com/flagstack/flagstack/backend/internal/featureflag"
)

type FeatureFlagRepository struct {
	client *flagstackent.Client
}

func NewFeatureFlagRepository(client *flagstackent.Client) *FeatureFlagRepository {
	return &FeatureFlagRepository{client: client}
}

func (r *FeatureFlagRepository) List(ctx context.Context, organisationID, projectID string) ([]corefeatureflag.FeatureFlag, error) {
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
		return nil, fmt.Errorf("check feature flag project: %w", err)
	}
	if !active {
		return []corefeatureflag.FeatureFlag{}, nil
	}

	entities, err := r.client.FeatureFlag.Query().
		Where(
			entfeatureflag.OrganisationID(organisationUUID),
			entfeatureflag.ProjectID(projectUUID),
			entfeatureflag.ArchivedAtIsNil(),
		).
		Order(
			flagstackent.Asc(entfeatureflag.FieldCreatedAt),
			flagstackent.Asc(entfeatureflag.FieldID),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list feature flags: %w", err)
	}

	flags := make([]corefeatureflag.FeatureFlag, 0, len(entities))
	for _, entity := range entities {
		flags = append(flags, featureFlagFromEntity(entity))
	}
	return flags, nil
}

func (r *FeatureFlagRepository) Create(ctx context.Context, organisationID, projectID string, input corefeatureflag.CreateInput) (corefeatureflag.FeatureFlag, error) {
	organisationUUID, projectUUID, err := parseProjectScope(organisationID, projectID)
	if err != nil {
		return corefeatureflag.FeatureFlag{}, err
	}

	active, err := r.client.Project.Query().
		Where(
			entproject.ID(projectUUID),
			entproject.OrganisationID(organisationUUID),
			entproject.ArchivedAtIsNil(),
		).
		Exist(ctx)
	if err != nil {
		return corefeatureflag.FeatureFlag{}, fmt.Errorf("check feature flag project: %w", err)
	}
	if !active {
		return corefeatureflag.FeatureFlag{}, corefeatureflag.ErrProjectNotFound
	}

	entity, err := r.client.FeatureFlag.Create().
		SetOrganisationID(organisationUUID).
		SetProjectID(projectUUID).
		SetName(input.Name).
		SetKey(input.Key).
		SetDescription(input.Description).
		SetKind(input.Kind).
		SetDefaultValue(input.DefaultValue).
		Save(ctx)
	if err != nil {
		if flagstackent.IsConstraintError(err) {
			return corefeatureflag.FeatureFlag{}, corefeatureflag.ErrKeyConflict
		}
		return corefeatureflag.FeatureFlag{}, fmt.Errorf("create feature flag: %w", err)
	}

	return featureFlagFromEntity(entity), nil
}

func featureFlagFromEntity(entity *flagstackent.FeatureFlag) corefeatureflag.FeatureFlag {
	return corefeatureflag.FeatureFlag{
		ID:             entity.ID.String(),
		OrganisationID: entity.OrganisationID.String(),
		ProjectID:      entity.ProjectID.String(),
		Name:           entity.Name,
		Key:            entity.Key,
		Description:    entity.Description,
		Kind:           entity.Kind,
		DefaultValue:   entity.DefaultValue,
		ClientVisible:  entity.ClientVisible,
		CreatedAt:      entity.CreatedAt,
	}
}
