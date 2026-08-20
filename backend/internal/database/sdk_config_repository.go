package database

import (
	"context"
	"fmt"
	"time"

	switchonyourcodeent "github.com/switchonyourcode/switchonyourcode/backend/ent"
	entenvironment "github.com/switchonyourcode/switchonyourcode/backend/ent/environment"
	entenvironmentflagconfig "github.com/switchonyourcode/switchonyourcode/backend/ent/environmentflagconfig"
	entfeatureflag "github.com/switchonyourcode/switchonyourcode/backend/ent/featureflag"
	entproject "github.com/switchonyourcode/switchonyourcode/backend/ent/project"
	entsdkcredential "github.com/switchonyourcode/switchonyourcode/backend/ent/sdkcredential"
	entsegment "github.com/switchonyourcode/switchonyourcode/backend/ent/segment"
	"github.com/switchonyourcode/switchonyourcode/backend/internal/evaluation"
	coresdkconfig "github.com/switchonyourcode/switchonyourcode/backend/internal/sdkconfig"
	"github.com/google/uuid"
)

type SDKConfigRepository struct {
	client *switchonyourcodeent.Client
}

func NewSDKConfigRepository(client *switchonyourcodeent.Client) *SDKConfigRepository {
	return &SDKConfigRepository{client: client}
}

func (r *SDKConfigRepository) ListCredentials(ctx context.Context, organisationID, projectID string) ([]coresdkconfig.Credential, error) {
	organisationUUID, projectUUID, err := parseProjectScope(organisationID, projectID)
	if err != nil {
		return nil, err
	}
	entities, err := r.client.SDKCredential.Query().Where(
		entsdkcredential.OrganisationID(organisationUUID),
		entsdkcredential.ProjectID(projectUUID),
	).Order(switchonyourcodeent.Asc(entsdkcredential.FieldCreatedAt), switchonyourcodeent.Asc(entsdkcredential.FieldID)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list SDK credentials: %w", err)
	}
	credentials := make([]coresdkconfig.Credential, 0, len(entities))
	for _, entity := range entities {
		credentials = append(credentials, sdkCredentialFromEntity(entity))
	}
	return credentials, nil
}

func (r *SDKConfigRepository) CreateCredential(ctx context.Context, record coresdkconfig.CredentialRecord) (coresdkconfig.Credential, error) {
	organisationUUID, projectUUID, err := parseProjectScope(record.OrganisationID, record.ProjectID)
	if err != nil {
		return coresdkconfig.Credential{}, err
	}
	environmentUUID, err := uuid.Parse(record.EnvironmentID)
	if err != nil {
		return coresdkconfig.Credential{}, coresdkconfig.ErrEnvironmentNotFound
	}
	active, err := r.client.Project.Query().Where(
		entproject.ID(projectUUID), entproject.OrganisationID(organisationUUID), entproject.ArchivedAtIsNil(),
	).Exist(ctx)
	if err != nil {
		return coresdkconfig.Credential{}, fmt.Errorf("check SDK credential project: %w", err)
	}
	if !active {
		return coresdkconfig.Credential{}, coresdkconfig.ErrEnvironmentNotFound
	}
	environmentExists, err := r.client.Environment.Query().Where(
		entenvironment.ID(environmentUUID),
		entenvironment.OrganisationID(organisationUUID),
		entenvironment.ProjectID(projectUUID),
	).Exist(ctx)
	if err != nil {
		return coresdkconfig.Credential{}, fmt.Errorf("check SDK credential environment: %w", err)
	}
	if !environmentExists {
		return coresdkconfig.Credential{}, coresdkconfig.ErrEnvironmentNotFound
	}

	builder := r.client.SDKCredential.Create().
		SetID(record.ID).
		SetOrganisationID(organisationUUID).
		SetProjectID(projectUUID).
		SetEnvironmentID(environmentUUID).
		SetName(record.Name).
		SetKind(record.Kind).
		SetClientKey(record.ClientKey)
	if len(record.SecretDigest) > 0 {
		builder.SetSecretDigest(record.SecretDigest)
	}
	entity, err := builder.Save(ctx)
	if err != nil {
		return coresdkconfig.Credential{}, fmt.Errorf("create SDK credential: %w", err)
	}
	return sdkCredentialFromEntity(entity), nil
}

func (r *SDKConfigRepository) RevokeCredential(ctx context.Context, organisationID, projectID, credentialID string, revokedAt time.Time) (coresdkconfig.Credential, error) {
	organisationUUID, projectUUID, err := parseProjectScope(organisationID, projectID)
	if err != nil {
		return coresdkconfig.Credential{}, err
	}
	credentialUUID, err := uuid.Parse(credentialID)
	if err != nil {
		return coresdkconfig.Credential{}, coresdkconfig.ErrCredentialNotFound
	}
	entity, err := r.client.SDKCredential.Query().Where(
		entsdkcredential.ID(credentialUUID),
		entsdkcredential.OrganisationID(organisationUUID),
		entsdkcredential.ProjectID(projectUUID),
	).Only(ctx)
	if switchonyourcodeent.IsNotFound(err) {
		return coresdkconfig.Credential{}, coresdkconfig.ErrCredentialNotFound
	}
	if err != nil {
		return coresdkconfig.Credential{}, fmt.Errorf("load SDK credential: %w", err)
	}
	if entity.RevokedAt != nil {
		return sdkCredentialFromEntity(entity), nil
	}
	updated, err := r.client.SDKCredential.UpdateOne(entity).SetRevokedAt(revokedAt).Save(ctx)
	if err != nil {
		return coresdkconfig.Credential{}, fmt.Errorf("revoke SDK credential: %w", err)
	}
	return sdkCredentialFromEntity(updated), nil
}

func (r *SDKConfigRepository) FindServerCredential(ctx context.Context, credentialID string) (coresdkconfig.StoredServerCredential, error) {
	credentialUUID, err := uuid.Parse(credentialID)
	if err != nil {
		return coresdkconfig.StoredServerCredential{}, coresdkconfig.ErrCredentialNotFound
	}
	entity, err := r.client.SDKCredential.Query().Where(
		entsdkcredential.ID(credentialUUID),
		entsdkcredential.KindEQ(coresdkconfig.KindServer),
		entsdkcredential.RevokedAtIsNil(),
	).Only(ctx)
	if switchonyourcodeent.IsNotFound(err) {
		return coresdkconfig.StoredServerCredential{}, coresdkconfig.ErrCredentialNotFound
	}
	if err != nil {
		return coresdkconfig.StoredServerCredential{}, fmt.Errorf("find server SDK credential: %w", err)
	}
	return coresdkconfig.StoredServerCredential{
		Credential:   sdkCredentialFromEntity(entity),
		SecretDigest: append([]byte(nil), entity.SecretDigest...),
	}, nil
}

func (r *SDKConfigRepository) FindClientCredential(ctx context.Context, clientKey string) (coresdkconfig.Credential, error) {
	entity, err := r.client.SDKCredential.Query().Where(
		entsdkcredential.ClientKeyEQ(clientKey),
		entsdkcredential.KindEQ(coresdkconfig.KindClient),
		entsdkcredential.RevokedAtIsNil(),
	).Only(ctx)
	if switchonyourcodeent.IsNotFound(err) {
		return coresdkconfig.Credential{}, coresdkconfig.ErrCredentialNotFound
	}
	if err != nil {
		return coresdkconfig.Credential{}, fmt.Errorf("find client SDK credential: %w", err)
	}
	return sdkCredentialFromEntity(entity), nil
}

func (r *SDKConfigRepository) SetClientVisible(ctx context.Context, organisationID, projectID, featureFlagID string, visible bool) (bool, error) {
	organisationUUID, projectUUID, err := parseProjectScope(organisationID, projectID)
	if err != nil {
		return false, err
	}
	flagUUID, err := uuid.Parse(featureFlagID)
	if err != nil {
		return false, coresdkconfig.ErrFeatureFlagNotFound
	}
	entity, err := r.client.FeatureFlag.Query().Where(
		entfeatureflag.ID(flagUUID),
		entfeatureflag.OrganisationID(organisationUUID),
		entfeatureflag.ProjectID(projectUUID),
		entfeatureflag.ArchivedAtIsNil(),
	).Only(ctx)
	if switchonyourcodeent.IsNotFound(err) {
		return false, coresdkconfig.ErrFeatureFlagNotFound
	}
	if err != nil {
		return false, fmt.Errorf("load feature flag client visibility: %w", err)
	}
	if entity.ClientVisible == visible {
		return visible, nil
	}
	if _, err := r.client.FeatureFlag.UpdateOne(entity).SetClientVisible(visible).Save(ctx); err != nil {
		return false, fmt.Errorf("update feature flag client visibility: %w", err)
	}
	return visible, nil
}

func (r *SDKConfigRepository) LoadConfiguration(ctx context.Context, credential coresdkconfig.Credential) (coresdkconfig.Configuration, error) {
	organisationUUID, projectUUID, err := parseProjectScope(credential.OrganisationID, credential.ProjectID)
	if err != nil {
		return coresdkconfig.Configuration{}, err
	}
	environmentUUID, err := uuid.Parse(credential.EnvironmentID)
	if err != nil {
		return coresdkconfig.Configuration{}, coresdkconfig.ErrInvalidCredential
	}
	active, err := r.client.Project.Query().Where(
		entproject.ID(projectUUID), entproject.OrganisationID(organisationUUID), entproject.ArchivedAtIsNil(),
	).Exist(ctx)
	if err != nil {
		return coresdkconfig.Configuration{}, fmt.Errorf("check SDK configuration project: %w", err)
	}
	if !active {
		return coresdkconfig.Configuration{}, coresdkconfig.ErrInvalidCredential
	}
	environment, err := r.client.Environment.Query().Where(
		entenvironment.ID(environmentUUID),
		entenvironment.OrganisationID(organisationUUID),
		entenvironment.ProjectID(projectUUID),
	).Only(ctx)
	if switchonyourcodeent.IsNotFound(err) {
		return coresdkconfig.Configuration{}, coresdkconfig.ErrInvalidCredential
	}
	if err != nil {
		return coresdkconfig.Configuration{}, fmt.Errorf("load SDK configuration environment: %w", err)
	}

	flagQuery := r.client.FeatureFlag.Query().Where(
		entfeatureflag.OrganisationID(organisationUUID),
		entfeatureflag.ProjectID(projectUUID),
		entfeatureflag.ArchivedAtIsNil(),
	)
	if credential.Kind == coresdkconfig.KindClient {
		flagQuery = flagQuery.Where(entfeatureflag.ClientVisibleEQ(true))
	}
	flagEntities, err := flagQuery.Order(switchonyourcodeent.Asc(entfeatureflag.FieldKey), switchonyourcodeent.Asc(entfeatureflag.FieldID)).All(ctx)
	if err != nil {
		return coresdkconfig.Configuration{}, fmt.Errorf("load SDK configuration flags: %w", err)
	}
	configEntities, err := r.client.EnvironmentFlagConfig.Query().Where(
		entenvironmentflagconfig.OrganisationID(organisationUUID),
		entenvironmentflagconfig.ProjectID(projectUUID),
		entenvironmentflagconfig.EnvironmentID(environmentUUID),
	).All(ctx)
	if err != nil {
		return coresdkconfig.Configuration{}, fmt.Errorf("load SDK environment flag configuration: %w", err)
	}
	configByFlag := make(map[uuid.UUID]*switchonyourcodeent.EnvironmentFlagConfig, len(configEntities))
	for _, config := range configEntities {
		configByFlag[config.FeatureFlagID] = config
	}

	flags := make([]coresdkconfig.Flag, 0, len(flagEntities))
	for _, flag := range flagEntities {
		resolved := coresdkconfig.Flag{
			ID:           flag.ID.String(),
			Key:          flag.Key,
			Kind:         flag.Kind,
			DefaultValue: flag.DefaultValue,
			Variants:     append([]evaluation.Variant(nil), flag.Variants...),
		}
		if config, ok := configByFlag[flag.ID]; ok {
			resolved.Enabled = config.Enabled
			resolved.Policy = config.Policy
			resolved.Revision = config.Revision
		}
		flags = append(flags, resolved)
	}

	segmentEntities, err := r.client.Segment.Query().Where(
		entsegment.OrganisationID(organisationUUID),
		entsegment.ProjectID(projectUUID),
		entsegment.ArchivedAtIsNil(),
	).Order(switchonyourcodeent.Asc(entsegment.FieldKey), switchonyourcodeent.Asc(entsegment.FieldID)).All(ctx)
	if err != nil {
		return coresdkconfig.Configuration{}, fmt.Errorf("load SDK configuration segments: %w", err)
	}
	segments := make([]evaluation.Segment, 0, len(segmentEntities))
	for _, segment := range segmentEntities {
		segments = append(segments, evaluation.Segment{
			Key:        segment.Key,
			Name:       segment.Name,
			Match:      evaluation.MatchMode(segment.Match),
			Conditions: append([]evaluation.Condition(nil), segment.Conditions...),
		})
	}

	return coresdkconfig.Configuration{
		Environment: coresdkconfig.Environment{ID: environment.ID.String(), Key: environment.Key},
		Flags:       flags,
		Segments:    segments,
	}, nil
}

func sdkCredentialFromEntity(entity *switchonyourcodeent.SDKCredential) coresdkconfig.Credential {
	return coresdkconfig.Credential{
		ID:             entity.ID.String(),
		OrganisationID: entity.OrganisationID.String(),
		ProjectID:      entity.ProjectID.String(),
		EnvironmentID:  entity.EnvironmentID.String(),
		Name:           entity.Name,
		Kind:           entity.Kind,
		ClientKey:      entity.ClientKey,
		RevokedAt:      entity.RevokedAt,
		CreatedAt:      entity.CreatedAt,
	}
}
