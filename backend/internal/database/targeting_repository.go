package database

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	flagstackent "github.com/flagstack/flagstack/backend/ent"
	entenvironment "github.com/flagstack/flagstack/backend/ent/environment"
	entenvironmentflagconfig "github.com/flagstack/flagstack/backend/ent/environmentflagconfig"
	entfeatureflag "github.com/flagstack/flagstack/backend/ent/featureflag"
	entproject "github.com/flagstack/flagstack/backend/ent/project"
	entscheduledflagchange "github.com/flagstack/flagstack/backend/ent/scheduledflagchange"
	entsegment "github.com/flagstack/flagstack/backend/ent/segment"
	"github.com/flagstack/flagstack/backend/internal/evaluation"
	coretargeting "github.com/flagstack/flagstack/backend/internal/targeting"
	"github.com/google/uuid"
)

const scheduleClaimLease = 2 * time.Minute

type TargetingRepository struct {
	client *flagstackent.Client
}

func NewTargetingRepository(client *flagstackent.Client) *TargetingRepository {
	return &TargetingRepository{client: client}
}

func (r *TargetingRepository) GetFlagState(ctx context.Context, organisationID, projectID, featureFlagID string) (coretargeting.FlagState, error) {
	organisationUUID, projectUUID, err := parseProjectScope(organisationID, projectID)
	if err != nil {
		return coretargeting.FlagState{}, err
	}
	flagUUID, err := uuid.Parse(featureFlagID)
	if err != nil {
		return coretargeting.FlagState{}, fmt.Errorf("parse feature flag ID: %w", err)
	}

	flag, err := r.client.FeatureFlag.Query().Where(
		entfeatureflag.ID(flagUUID),
		entfeatureflag.OrganisationID(organisationUUID),
		entfeatureflag.ProjectID(projectUUID),
		entfeatureflag.ArchivedAtIsNil(),
	).Only(ctx)
	if flagstackent.IsNotFound(err) {
		return coretargeting.FlagState{}, coretargeting.ErrFeatureFlagNotFound
	}
	if err != nil {
		return coretargeting.FlagState{}, fmt.Errorf("load targeting flag: %w", err)
	}

	configs, err := r.client.EnvironmentFlagConfig.Query().Where(
		entenvironmentflagconfig.OrganisationID(organisationUUID),
		entenvironmentflagconfig.ProjectID(projectUUID),
		entenvironmentflagconfig.FeatureFlagID(flagUUID),
	).All(ctx)
	if err != nil {
		return coretargeting.FlagState{}, fmt.Errorf("load flag environment policies: %w", err)
	}

	environments := make([]coretargeting.EnvironmentState, 0, len(configs))
	for _, config := range configs {
		environments = append(environments, environmentStateFromEntity(config))
	}
	return coretargeting.FlagState{
		ID:           flag.ID.String(),
		Key:          flag.Key,
		Kind:         flag.Kind,
		DefaultValue: cloneJSON(flag.DefaultValue),
		Variants:     append([]evaluation.Variant(nil), flag.Variants...),
		Environments: environments,
	}, nil
}

func (r *TargetingRepository) SetVariants(ctx context.Context, organisationID, projectID, featureFlagID string, variants []evaluation.Variant) (coretargeting.FlagState, error) {
	organisationUUID, projectUUID, flagUUID, err := parseFlagScope(organisationID, projectID, featureFlagID)
	if err != nil {
		return coretargeting.FlagState{}, err
	}
	entity, err := r.client.FeatureFlag.Query().Where(
		entfeatureflag.ID(flagUUID),
		entfeatureflag.OrganisationID(organisationUUID),
		entfeatureflag.ProjectID(projectUUID),
		entfeatureflag.ArchivedAtIsNil(),
	).Only(ctx)
	if flagstackent.IsNotFound(err) {
		return coretargeting.FlagState{}, coretargeting.ErrFeatureFlagNotFound
	}
	if err != nil {
		return coretargeting.FlagState{}, fmt.Errorf("load feature flag for variants: %w", err)
	}
	if reflect.DeepEqual(entity.Variants, variants) {
		return r.GetFlagState(ctx, organisationID, projectID, featureFlagID)
	}
	if _, err := r.client.FeatureFlag.UpdateOne(entity).SetVariants(variants).Save(ctx); err != nil {
		return coretargeting.FlagState{}, fmt.Errorf("update feature flag variants: %w", err)
	}
	return r.GetFlagState(ctx, organisationID, projectID, featureFlagID)
}

func (r *TargetingRepository) SetPolicy(ctx context.Context, organisationID, projectID, environmentID, featureFlagID string, policy evaluation.Policy) (coretargeting.EnvironmentState, error) {
	organisationUUID, projectUUID, environmentUUID, flagUUID, err := parseEnvironmentFlagScope(organisationID, projectID, environmentID, featureFlagID)
	if err != nil {
		return coretargeting.EnvironmentState{}, err
	}
	if err := ensureEvaluationTargets(ctx, r.client, organisationUUID, projectUUID, environmentUUID, flagUUID); err != nil {
		return coretargeting.EnvironmentState{}, err
	}

	entity, err := r.client.EnvironmentFlagConfig.Query().Where(
		entenvironmentflagconfig.OrganisationID(organisationUUID),
		entenvironmentflagconfig.ProjectID(projectUUID),
		entenvironmentflagconfig.EnvironmentID(environmentUUID),
		entenvironmentflagconfig.FeatureFlagID(flagUUID),
	).Only(ctx)
	if flagstackent.IsNotFound(err) {
		created, createErr := r.client.EnvironmentFlagConfig.Create().
			SetOrganisationID(organisationUUID).
			SetProjectID(projectUUID).
			SetEnvironmentID(environmentUUID).
			SetFeatureFlagID(flagUUID).
			SetEnabled(false).
			SetPolicy(policy).
			Save(ctx)
		if createErr != nil {
			return coretargeting.EnvironmentState{}, fmt.Errorf("create environment targeting policy: %w", createErr)
		}
		return environmentStateFromEntity(created), nil
	}
	if err != nil {
		return coretargeting.EnvironmentState{}, fmt.Errorf("load environment targeting policy: %w", err)
	}
	if reflect.DeepEqual(entity.Policy, policy) {
		return environmentStateFromEntity(entity), nil
	}
	updated, err := r.client.EnvironmentFlagConfig.UpdateOne(entity).
		SetPolicy(policy).
		AddRevision(1).
		Save(ctx)
	if err != nil {
		return coretargeting.EnvironmentState{}, fmt.Errorf("update environment targeting policy: %w", err)
	}
	return environmentStateFromEntity(updated), nil
}

func (r *TargetingRepository) LoadEvaluation(ctx context.Context, organisationID, projectID, environmentID, featureFlagID string) (evaluation.Flag, []evaluation.Segment, error) {
	organisationUUID, projectUUID, environmentUUID, flagUUID, err := parseEnvironmentFlagScope(organisationID, projectID, environmentID, featureFlagID)
	if err != nil {
		return evaluation.Flag{}, nil, err
	}
	if err := ensureEvaluationTargets(ctx, r.client, organisationUUID, projectUUID, environmentUUID, flagUUID); err != nil {
		return evaluation.Flag{}, nil, err
	}

	flag, err := r.client.FeatureFlag.Query().Where(entfeatureflag.ID(flagUUID)).Only(ctx)
	if err != nil {
		return evaluation.Flag{}, nil, fmt.Errorf("load evaluation flag: %w", err)
	}

	config, err := r.client.EnvironmentFlagConfig.Query().Where(
		entenvironmentflagconfig.OrganisationID(organisationUUID),
		entenvironmentflagconfig.ProjectID(projectUUID),
		entenvironmentflagconfig.EnvironmentID(environmentUUID),
		entenvironmentflagconfig.FeatureFlagID(flagUUID),
	).Only(ctx)
	state := coretargeting.EnvironmentState{EnvironmentID: environmentID}
	if err == nil {
		state = environmentStateFromEntity(config)
	} else if !flagstackent.IsNotFound(err) {
		return evaluation.Flag{}, nil, fmt.Errorf("load evaluation config: %w", err)
	}

	segments, err := r.ListSegments(ctx, organisationID, projectID)
	if err != nil {
		return evaluation.Flag{}, nil, err
	}
	evaluationSegments := make([]evaluation.Segment, 0, len(segments))
	for _, segment := range segments {
		evaluationSegments = append(evaluationSegments, evaluation.Segment{
			Key: segment.Key, Name: segment.Name, Match: segment.Match, Conditions: segment.Conditions,
		})
	}

	return evaluation.Flag{
		ID:            flag.ID.String(),
		EnvironmentID: environmentID,
		Key:           flag.Key,
		Kind:          flag.Kind,
		DefaultValue:  cloneJSON(flag.DefaultValue),
		Enabled:       state.Enabled,
		Variants:      append([]evaluation.Variant(nil), flag.Variants...),
		Policy:        state.Policy,
	}, evaluationSegments, nil
}

func (r *TargetingRepository) ListSegments(ctx context.Context, organisationID, projectID string) ([]coretargeting.Segment, error) {
	organisationUUID, projectUUID, err := parseProjectScope(organisationID, projectID)
	if err != nil {
		return nil, err
	}
	entities, err := r.client.Segment.Query().Where(
		entsegment.OrganisationID(organisationUUID),
		entsegment.ProjectID(projectUUID),
		entsegment.ArchivedAtIsNil(),
	).Order(flagstackent.Asc(entsegment.FieldCreatedAt), flagstackent.Asc(entsegment.FieldID)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list targeting segments: %w", err)
	}
	segments := make([]coretargeting.Segment, 0, len(entities))
	for _, entity := range entities {
		segments = append(segments, segmentFromEntity(entity))
	}
	return segments, nil
}

func (r *TargetingRepository) CreateSegment(ctx context.Context, organisationID, projectID string, input coretargeting.SegmentInput) (coretargeting.Segment, error) {
	organisationUUID, projectUUID, err := parseProjectScope(organisationID, projectID)
	if err != nil {
		return coretargeting.Segment{}, err
	}
	active, err := r.client.Project.Query().Where(
		entproject.ID(projectUUID), entproject.OrganisationID(organisationUUID), entproject.ArchivedAtIsNil(),
	).Exist(ctx)
	if err != nil {
		return coretargeting.Segment{}, fmt.Errorf("check segment project: %w", err)
	}
	if !active {
		return coretargeting.Segment{}, coretargeting.ErrProjectNotFound
	}
	entity, err := r.client.Segment.Create().
		SetOrganisationID(organisationUUID).
		SetProjectID(projectUUID).
		SetName(input.Name).
		SetKey(input.Key).
		SetDescription(input.Description).
		SetMatch(string(input.Match)).
		SetConditions(input.Conditions).
		Save(ctx)
	if err != nil {
		if flagstackent.IsConstraintError(err) {
			return coretargeting.Segment{}, coretargeting.ErrSegmentKeyConflict
		}
		return coretargeting.Segment{}, fmt.Errorf("create targeting segment: %w", err)
	}
	return segmentFromEntity(entity), nil
}

func (r *TargetingRepository) UpdateSegment(ctx context.Context, organisationID, projectID, segmentID string, input coretargeting.SegmentInput) (coretargeting.Segment, error) {
	organisationUUID, projectUUID, err := parseProjectScope(organisationID, projectID)
	if err != nil {
		return coretargeting.Segment{}, err
	}
	segmentUUID, err := uuid.Parse(segmentID)
	if err != nil {
		return coretargeting.Segment{}, fmt.Errorf("parse segment ID: %w", err)
	}
	entity, err := r.client.Segment.Query().Where(
		entsegment.ID(segmentUUID), entsegment.OrganisationID(organisationUUID), entsegment.ProjectID(projectUUID), entsegment.ArchivedAtIsNil(),
	).Only(ctx)
	if flagstackent.IsNotFound(err) {
		return coretargeting.Segment{}, coretargeting.ErrSegmentNotFound
	}
	if err != nil {
		return coretargeting.Segment{}, fmt.Errorf("load targeting segment: %w", err)
	}
	updated, err := r.client.Segment.UpdateOne(entity).
		SetName(input.Name).
		SetDescription(input.Description).
		SetMatch(string(input.Match)).
		SetConditions(input.Conditions).
		Save(ctx)
	if err != nil {
		return coretargeting.Segment{}, fmt.Errorf("update targeting segment: %w", err)
	}
	return segmentFromEntity(updated), nil
}

func (r *TargetingRepository) ListScheduledChanges(ctx context.Context, organisationID, projectID string) ([]coretargeting.ScheduledChange, error) {
	organisationUUID, projectUUID, err := parseProjectScope(organisationID, projectID)
	if err != nil {
		return nil, err
	}
	entities, err := r.client.ScheduledFlagChange.Query().Where(
		entscheduledflagchange.OrganisationID(organisationUUID),
		entscheduledflagchange.ProjectID(projectUUID),
	).Order(flagstackent.Asc(entscheduledflagchange.FieldExecuteAt), flagstackent.Asc(entscheduledflagchange.FieldID)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list scheduled flag changes: %w", err)
	}
	changes := make([]coretargeting.ScheduledChange, 0, len(entities))
	for _, entity := range entities {
		change, err := scheduledChangeFromEntity(entity)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}
	return changes, nil
}

func (r *TargetingRepository) CreateScheduledChange(ctx context.Context, organisationID, projectID string, input coretargeting.CreateScheduleInput) (coretargeting.ScheduledChange, error) {
	organisationUUID, projectUUID, environmentUUID, flagUUID, err := parseEnvironmentFlagScope(organisationID, projectID, input.EnvironmentID, input.FeatureFlagID)
	if err != nil {
		return coretargeting.ScheduledChange{}, err
	}
	if err := ensureEvaluationTargets(ctx, r.client, organisationUUID, projectUUID, environmentUUID, flagUUID); err != nil {
		return coretargeting.ScheduledChange{}, err
	}
	patch, err := json.Marshal(input.Patch)
	if err != nil {
		return coretargeting.ScheduledChange{}, fmt.Errorf("encode scheduled flag patch: %w", err)
	}

	builder := r.client.ScheduledFlagChange.Create().
		SetOrganisationID(organisationUUID).
		SetProjectID(projectUUID).
		SetEnvironmentID(environmentUUID).
		SetFeatureFlagID(flagUUID).
		SetExecuteAt(input.ExecuteAt.UTC()).
		SetPatch(patch)
	if input.CreatedByUserID != "" {
		creatorUUID, parseErr := uuid.Parse(input.CreatedByUserID)
		if parseErr != nil {
			return coretargeting.ScheduledChange{}, fmt.Errorf("parse schedule creator ID: %w", parseErr)
		}
		builder.SetCreatedByUserID(creatorUUID)
	}
	entity, err := builder.Save(ctx)
	if err != nil {
		return coretargeting.ScheduledChange{}, fmt.Errorf("create scheduled flag change: %w", err)
	}
	return scheduledChangeFromEntity(entity)
}

func (r *TargetingRepository) CancelScheduledChange(ctx context.Context, organisationID, projectID, scheduleID string) (coretargeting.ScheduledChange, error) {
	organisationUUID, projectUUID, scheduleUUID, err := parseScheduleScope(organisationID, projectID, scheduleID)
	if err != nil {
		return coretargeting.ScheduledChange{}, err
	}
	entity, err := r.client.ScheduledFlagChange.Query().Where(
		entscheduledflagchange.ID(scheduleUUID),
		entscheduledflagchange.OrganisationID(organisationUUID),
		entscheduledflagchange.ProjectID(projectUUID),
	).Only(ctx)
	if flagstackent.IsNotFound(err) {
		return coretargeting.ScheduledChange{}, coretargeting.ErrScheduleNotFound
	}
	if err != nil {
		return coretargeting.ScheduledChange{}, fmt.Errorf("load scheduled flag change: %w", err)
	}
	if entity.Status != "pending" {
		return coretargeting.ScheduledChange{}, coretargeting.ErrScheduleNotPending
	}
	updated, err := r.client.ScheduledFlagChange.UpdateOne(entity).SetStatus("cancelled").Save(ctx)
	if err != nil {
		return coretargeting.ScheduledChange{}, fmt.Errorf("cancel scheduled flag change: %w", err)
	}
	return scheduledChangeFromEntity(updated)
}

func (r *TargetingRepository) ClaimDueScheduledChanges(ctx context.Context, now time.Time, limit int) ([]coretargeting.ScheduledChange, error) {
	staleBefore := now.Add(-scheduleClaimLease)
	eligible := entscheduledflagchange.Or(
		entscheduledflagchange.And(
			entscheduledflagchange.StatusEQ("pending"),
			entscheduledflagchange.ExecuteAtLTE(now),
		),
		entscheduledflagchange.And(
			entscheduledflagchange.StatusEQ("running"),
			entscheduledflagchange.ClaimedAtNotNil(),
			entscheduledflagchange.ClaimedAtLTE(staleBefore),
		),
	)
	entities, err := r.client.ScheduledFlagChange.Query().Where(eligible).
		Order(flagstackent.Asc(entscheduledflagchange.FieldExecuteAt), flagstackent.Asc(entscheduledflagchange.FieldID)).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list due scheduled flag changes: %w", err)
	}

	claimed := make([]coretargeting.ScheduledChange, 0, len(entities))
	for _, entity := range entities {
		token := uuid.New()
		count, err := r.client.ScheduledFlagChange.Update().Where(
			entscheduledflagchange.ID(entity.ID),
			entscheduledflagchange.Or(
				entscheduledflagchange.And(
					entscheduledflagchange.StatusEQ("pending"),
					entscheduledflagchange.ExecuteAtLTE(now),
				),
				entscheduledflagchange.And(
					entscheduledflagchange.StatusEQ("running"),
					entscheduledflagchange.ClaimedAtNotNil(),
					entscheduledflagchange.ClaimedAtLTE(staleBefore),
				),
			),
		).SetStatus("running").SetClaimToken(token).SetClaimedAt(now).Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("claim scheduled flag change: %w", err)
		}
		if count != 1 {
			continue
		}
		change, err := scheduledChangeFromEntity(entity)
		if err != nil {
			return nil, err
		}
		change.Status = "running"
		change.ClaimToken = token.String()
		claimedAt := now
		change.ClaimedAt = &claimedAt
		claimed = append(claimed, change)
	}
	return claimed, nil
}

func (r *TargetingRepository) ApplyClaimedScheduledChange(ctx context.Context, change coretargeting.ScheduledChange) error {
	organisationUUID, projectUUID, environmentUUID, flagUUID, err := parseEnvironmentFlagScope(change.OrganisationID, change.ProjectID, change.EnvironmentID, change.FeatureFlagID)
	if err != nil {
		return err
	}
	scheduleUUID, err := uuid.Parse(change.ID)
	if err != nil {
		return fmt.Errorf("parse scheduled change ID: %w", err)
	}
	claimUUID, err := uuid.Parse(change.ClaimToken)
	if err != nil {
		return fmt.Errorf("parse scheduled change claim token: %w", err)
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin scheduled change transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	schedule, err := tx.ScheduledFlagChange.Query().Where(
		entscheduledflagchange.ID(scheduleUUID),
		entscheduledflagchange.StatusEQ("running"),
		entscheduledflagchange.ClaimTokenEQ(claimUUID),
	).Only(ctx)
	if flagstackent.IsNotFound(err) {
		return coretargeting.ErrScheduleNotPending
	}
	if err != nil {
		return fmt.Errorf("load claimed scheduled change: %w", err)
	}

	config, err := tx.EnvironmentFlagConfig.Query().Where(
		entenvironmentflagconfig.OrganisationID(organisationUUID),
		entenvironmentflagconfig.ProjectID(projectUUID),
		entenvironmentflagconfig.EnvironmentID(environmentUUID),
		entenvironmentflagconfig.FeatureFlagID(flagUUID),
	).Only(ctx)
	if flagstackent.IsNotFound(err) {
		builder := tx.EnvironmentFlagConfig.Create().
			SetOrganisationID(organisationUUID).
			SetProjectID(projectUUID).
			SetEnvironmentID(environmentUUID).
			SetFeatureFlagID(flagUUID).
			SetEnabled(false)
		if change.Patch.Enabled != nil {
			builder.SetEnabled(*change.Patch.Enabled)
		}
		if change.Patch.Policy != nil {
			builder.SetPolicy(*change.Patch.Policy)
		}
		if _, err := builder.Save(ctx); err != nil {
			return fmt.Errorf("create scheduled environment config: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("load scheduled environment config: %w", err)
	} else {
		update := tx.EnvironmentFlagConfig.UpdateOne(config)
		changed := false
		if change.Patch.Enabled != nil && config.Enabled != *change.Patch.Enabled {
			update.SetEnabled(*change.Patch.Enabled)
			changed = true
		}
		if change.Patch.Policy != nil && !reflect.DeepEqual(config.Policy, *change.Patch.Policy) {
			update.SetPolicy(*change.Patch.Policy)
			changed = true
		}
		if changed {
			if _, err := update.AddRevision(1).Save(ctx); err != nil {
				return fmt.Errorf("apply scheduled environment config: %w", err)
			}
		}
	}

	now := time.Now().UTC()
	if _, err := tx.ScheduledFlagChange.UpdateOne(schedule).
		SetStatus("executed").
		SetExecutedAt(now).
		SetLastError("").
		Save(ctx); err != nil {
		return fmt.Errorf("complete scheduled flag change: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit scheduled flag change: %w", err)
	}
	return nil
}

func (r *TargetingRepository) FailClaimedScheduledChange(ctx context.Context, change coretargeting.ScheduledChange, message string) error {
	scheduleUUID, err := uuid.Parse(change.ID)
	if err != nil {
		return fmt.Errorf("parse scheduled change ID: %w", err)
	}
	claimUUID, err := uuid.Parse(change.ClaimToken)
	if err != nil {
		return fmt.Errorf("parse scheduled change claim token: %w", err)
	}
	if len(message) > 4000 {
		message = message[:4000]
	}
	count, err := r.client.ScheduledFlagChange.Update().Where(
		entscheduledflagchange.ID(scheduleUUID),
		entscheduledflagchange.StatusEQ("running"),
		entscheduledflagchange.ClaimTokenEQ(claimUUID),
	).SetStatus("failed").SetLastError(message).Save(ctx)
	if err != nil {
		return fmt.Errorf("fail scheduled flag change: %w", err)
	}
	if count != 1 {
		return coretargeting.ErrScheduleNotPending
	}
	return nil
}

func ensureEvaluationTargets(ctx context.Context, client *flagstackent.Client, organisationID, projectID, environmentID, featureFlagID uuid.UUID) error {
	environmentExists, err := client.Environment.Query().Where(
		entenvironment.ID(environmentID),
		entenvironment.OrganisationID(organisationID),
		entenvironment.ProjectID(projectID),
	).Exist(ctx)
	if err != nil {
		return fmt.Errorf("check targeting environment: %w", err)
	}
	if !environmentExists {
		return coretargeting.ErrEnvironmentNotFound
	}
	flagExists, err := client.FeatureFlag.Query().Where(
		entfeatureflag.ID(featureFlagID),
		entfeatureflag.OrganisationID(organisationID),
		entfeatureflag.ProjectID(projectID),
		entfeatureflag.ArchivedAtIsNil(),
	).Exist(ctx)
	if err != nil {
		return fmt.Errorf("check targeting feature flag: %w", err)
	}
	if !flagExists {
		return coretargeting.ErrFeatureFlagNotFound
	}
	return nil
}

func environmentStateFromEntity(entity *flagstackent.EnvironmentFlagConfig) coretargeting.EnvironmentState {
	return coretargeting.EnvironmentState{
		EnvironmentID: entity.EnvironmentID.String(),
		Enabled:       entity.Enabled,
		Policy:        entity.Policy,
		Revision:      entity.Revision,
	}
}

func segmentFromEntity(entity *flagstackent.Segment) coretargeting.Segment {
	return coretargeting.Segment{
		ID:             entity.ID.String(),
		OrganisationID: entity.OrganisationID.String(),
		ProjectID:      entity.ProjectID.String(),
		Name:           entity.Name,
		Key:            entity.Key,
		Description:    entity.Description,
		Match:          evaluation.MatchMode(entity.Match),
		Conditions:     append([]evaluation.Condition(nil), entity.Conditions...),
		CreatedAt:      entity.CreatedAt,
		UpdatedAt:      entity.UpdatedAt,
	}
}

func scheduledChangeFromEntity(entity *flagstackent.ScheduledFlagChange) (coretargeting.ScheduledChange, error) {
	var patch coretargeting.SchedulePatch
	if err := decodeJSONStrict(entity.Patch, &patch); err != nil {
		return coretargeting.ScheduledChange{}, fmt.Errorf("decode scheduled flag patch: %w", err)
	}
	change := coretargeting.ScheduledChange{
		ID:             entity.ID.String(),
		OrganisationID: entity.OrganisationID.String(),
		ProjectID:      entity.ProjectID.String(),
		EnvironmentID:  entity.EnvironmentID.String(),
		FeatureFlagID:  entity.FeatureFlagID.String(),
		ExecuteAt:      entity.ExecuteAt,
		Patch:          patch,
		Status:         entity.Status,
		ClaimedAt:      entity.ClaimedAt,
		ExecutedAt:     entity.ExecutedAt,
		LastError:      entity.LastError,
		CreatedAt:      entity.CreatedAt,
		UpdatedAt:      entity.UpdatedAt,
	}
	if entity.CreatedByUserID != nil {
		change.CreatedByUserID = entity.CreatedByUserID.String()
	}
	if entity.ClaimToken != nil {
		change.ClaimToken = entity.ClaimToken.String()
	}
	return change, nil
}

func parseFlagScope(organisationID, projectID, featureFlagID string) (uuid.UUID, uuid.UUID, uuid.UUID, error) {
	organisationUUID, projectUUID, err := parseProjectScope(organisationID, projectID)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, err
	}
	flagUUID, err := uuid.Parse(featureFlagID)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, fmt.Errorf("parse feature flag ID: %w", err)
	}
	return organisationUUID, projectUUID, flagUUID, nil
}

func parseEnvironmentFlagScope(organisationID, projectID, environmentID, featureFlagID string) (uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, error) {
	organisationUUID, projectUUID, flagUUID, err := parseFlagScope(organisationID, projectID, featureFlagID)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil, err
	}
	environmentUUID, err := uuid.Parse(environmentID)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil, fmt.Errorf("parse environment ID: %w", err)
	}
	return organisationUUID, projectUUID, environmentUUID, flagUUID, nil
}

func parseScheduleScope(organisationID, projectID, scheduleID string) (uuid.UUID, uuid.UUID, uuid.UUID, error) {
	organisationUUID, projectUUID, err := parseProjectScope(organisationID, projectID)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, err
	}
	scheduleUUID, err := uuid.Parse(scheduleID)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, fmt.Errorf("parse scheduled change ID: %w", err)
	}
	return organisationUUID, projectUUID, scheduleUUID, nil
}

func cloneJSON(value json.RawMessage) json.RawMessage {
	return append(json.RawMessage(nil), value...)
}

func decodeJSONStrict(raw json.RawMessage, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return fmt.Errorf("multiple JSON values are not allowed")
	}
	return nil
}
