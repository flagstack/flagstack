package targeting

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/flagstack/flagstack/backend/internal/evaluation"
)

var (
	ErrProjectNotFound     = errors.New("project not found")
	ErrEnvironmentNotFound = errors.New("environment not found")
	ErrFeatureFlagNotFound = errors.New("feature flag not found")
	ErrSegmentNotFound     = errors.New("segment not found")
	ErrSegmentKeyConflict  = errors.New("segment key already exists")
	ErrScheduleNotFound    = errors.New("scheduled change not found")
	ErrScheduleNotPending  = errors.New("scheduled change is not pending")
)

var keyPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9._-]{0,126}[a-z0-9])?$`)

type EnvironmentState struct {
	EnvironmentID string
	Enabled       bool
	Policy        evaluation.Policy
	Revision      int64
}

type FlagState struct {
	ID           string
	Key          string
	Kind         string
	DefaultValue json.RawMessage
	Variants     []evaluation.Variant
	Environments []EnvironmentState
}

type Segment struct {
	ID             string
	OrganisationID string
	ProjectID      string
	Name           string
	Key            string
	Description    string
	Match          evaluation.MatchMode
	Conditions     []evaluation.Condition
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type SegmentInput struct {
	Name        string
	Key         string
	Description string
	Match       evaluation.MatchMode
	Conditions  []evaluation.Condition
}

type SchedulePatch struct {
	Enabled *bool              `json:"enabled,omitempty"`
	Policy  *evaluation.Policy `json:"policy,omitempty"`
}

type ScheduledChange struct {
	ID              string
	OrganisationID  string
	ProjectID       string
	EnvironmentID   string
	FeatureFlagID   string
	CreatedByUserID string
	ExecuteAt       time.Time
	Patch           SchedulePatch
	Status          string
	ClaimToken      string
	ClaimedAt       *time.Time
	ExecutedAt      *time.Time
	LastError       string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type CreateScheduleInput struct {
	EnvironmentID   string
	FeatureFlagID   string
	CreatedByUserID string
	ExecuteAt       time.Time
	Patch           SchedulePatch
}

type Repository interface {
	GetFlagState(context.Context, string, string, string) (FlagState, error)
	SetVariants(context.Context, string, string, string, []evaluation.Variant) (FlagState, error)
	SetPolicy(context.Context, string, string, string, string, evaluation.Policy) (EnvironmentState, error)
	LoadEvaluation(context.Context, string, string, string, string) (evaluation.Flag, []evaluation.Segment, error)

	ListSegments(context.Context, string, string) ([]Segment, error)
	CreateSegment(context.Context, string, string, SegmentInput) (Segment, error)
	UpdateSegment(context.Context, string, string, string, SegmentInput) (Segment, error)

	ListScheduledChanges(context.Context, string, string) ([]ScheduledChange, error)
	CreateScheduledChange(context.Context, string, string, CreateScheduleInput) (ScheduledChange, error)
	CancelScheduledChange(context.Context, string, string, string) (ScheduledChange, error)
	ClaimDueScheduledChanges(context.Context, time.Time, int) ([]ScheduledChange, error)
	ApplyClaimedScheduledChange(context.Context, ScheduledChange) error
	FailClaimedScheduledChange(context.Context, ScheduledChange, string) error
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) (*Service, error) {
	if repository == nil {
		return nil, errors.New("targeting repository is required")
	}
	return &Service{repository: repository}, nil
}

func (s *Service) SetVariants(ctx context.Context, organisationID, projectID, featureFlagID string, variants []evaluation.Variant) (FlagState, error) {
	state, err := s.repository.GetFlagState(ctx, organisationID, projectID, featureFlagID)
	if err != nil {
		return FlagState{}, err
	}

	for _, environment := range state.Environments {
		if err := evaluation.ValidateDefinition(state.Kind, state.DefaultValue, variants, environment.Policy); err != nil {
			return FlagState{}, &ValidationError{Field: "variants", Message: err.Error()}
		}
	}
	if len(state.Environments) == 0 {
		if err := evaluation.ValidateDefinition(state.Kind, state.DefaultValue, variants, evaluation.Policy{}); err != nil {
			return FlagState{}, &ValidationError{Field: "variants", Message: err.Error()}
		}
	}
	return s.repository.SetVariants(ctx, organisationID, projectID, featureFlagID, variants)
}

func (s *Service) SetPolicy(ctx context.Context, organisationID, projectID, environmentID, featureFlagID string, policy evaluation.Policy) (EnvironmentState, error) {
	state, err := s.repository.GetFlagState(ctx, organisationID, projectID, featureFlagID)
	if err != nil {
		return EnvironmentState{}, err
	}
	if err := evaluation.ValidateDefinition(state.Kind, state.DefaultValue, state.Variants, policy); err != nil {
		return EnvironmentState{}, &ValidationError{Field: "policy", Message: err.Error()}
	}

	segments, err := s.repository.ListSegments(ctx, organisationID, projectID)
	if err != nil {
		return EnvironmentState{}, err
	}
	if err := evaluation.ValidatePolicySegments(policy, evaluationSegments(segments)); err != nil {
		return EnvironmentState{}, &ValidationError{Field: "policy", Message: err.Error()}
	}
	return s.repository.SetPolicy(ctx, organisationID, projectID, environmentID, featureFlagID, policy)
}

func (s *Service) Preview(ctx context.Context, organisationID, projectID, environmentID, featureFlagID string, evaluationContext evaluation.Context) (evaluation.Result, error) {
	flag, segments, err := s.repository.LoadEvaluation(ctx, organisationID, projectID, environmentID, featureFlagID)
	if err != nil {
		return evaluation.Result{}, err
	}
	return evaluation.Evaluate(flag, evaluationContext, segments), nil
}

func (s *Service) ListSegments(ctx context.Context, organisationID, projectID string) ([]Segment, error) {
	return s.repository.ListSegments(ctx, organisationID, projectID)
}

func (s *Service) CreateSegment(ctx context.Context, organisationID, projectID string, input SegmentInput) (Segment, error) {
	normalized, err := normalizeSegmentInput(input, true)
	if err != nil {
		return Segment{}, err
	}
	existing, err := s.repository.ListSegments(ctx, organisationID, projectID)
	if err != nil {
		return Segment{}, err
	}
	candidate := append(evaluationSegments(existing), evaluationSegment(normalized))
	if err := evaluation.ValidateSegmentSet(candidate); err != nil {
		return Segment{}, &ValidationError{Field: "conditions", Message: err.Error()}
	}
	return s.repository.CreateSegment(ctx, organisationID, projectID, normalized)
}

func (s *Service) UpdateSegment(ctx context.Context, organisationID, projectID, segmentID string, input SegmentInput) (Segment, error) {
	normalized, err := normalizeSegmentInput(input, false)
	if err != nil {
		return Segment{}, err
	}
	existing, err := s.repository.ListSegments(ctx, organisationID, projectID)
	if err != nil {
		return Segment{}, err
	}

	found := false
	candidate := make([]evaluation.Segment, 0, len(existing))
	for _, segment := range existing {
		if segment.ID == segmentID {
			found = true
			if normalized.Key == "" {
				normalized.Key = segment.Key
			}
			candidate = append(candidate, evaluationSegment(normalized))
			continue
		}
		candidate = append(candidate, evaluationSegmentFromModel(segment))
	}
	if !found {
		return Segment{}, ErrSegmentNotFound
	}
	if err := evaluation.ValidateSegmentSet(candidate); err != nil {
		return Segment{}, &ValidationError{Field: "conditions", Message: err.Error()}
	}
	return s.repository.UpdateSegment(ctx, organisationID, projectID, segmentID, normalized)
}

func (s *Service) ListScheduledChanges(ctx context.Context, organisationID, projectID string) ([]ScheduledChange, error) {
	return s.repository.ListScheduledChanges(ctx, organisationID, projectID)
}

func (s *Service) CreateScheduledChange(ctx context.Context, organisationID, projectID string, input CreateScheduleInput) (ScheduledChange, error) {
	if input.ExecuteAt.IsZero() {
		return ScheduledChange{}, &ValidationError{Field: "execute_at", Message: "is required"}
	}
	if !input.ExecuteAt.After(time.Now()) {
		return ScheduledChange{}, &ValidationError{Field: "execute_at", Message: "must be in the future"}
	}
	if input.Patch.Enabled == nil && input.Patch.Policy == nil {
		return ScheduledChange{}, &ValidationError{Field: "patch", Message: "must change enabled state or policy"}
	}
	if input.Patch.Policy != nil {
		state, err := s.repository.GetFlagState(ctx, organisationID, projectID, input.FeatureFlagID)
		if err != nil {
			return ScheduledChange{}, err
		}
		if err := evaluation.ValidateDefinition(state.Kind, state.DefaultValue, state.Variants, *input.Patch.Policy); err != nil {
			return ScheduledChange{}, &ValidationError{Field: "patch.policy", Message: err.Error()}
		}
		segments, err := s.repository.ListSegments(ctx, organisationID, projectID)
		if err != nil {
			return ScheduledChange{}, err
		}
		if err := evaluation.ValidatePolicySegments(*input.Patch.Policy, evaluationSegments(segments)); err != nil {
			return ScheduledChange{}, &ValidationError{Field: "patch.policy", Message: err.Error()}
		}
	}
	return s.repository.CreateScheduledChange(ctx, organisationID, projectID, input)
}

func (s *Service) CancelScheduledChange(ctx context.Context, organisationID, projectID, scheduleID string) (ScheduledChange, error) {
	return s.repository.CancelScheduledChange(ctx, organisationID, projectID, scheduleID)
}

func (s *Service) RunDueScheduledChanges(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		limit = 50
	}
	changes, err := s.repository.ClaimDueScheduledChanges(ctx, time.Now(), limit)
	if err != nil {
		return 0, err
	}

	completed := 0
	for _, change := range changes {
		if err := s.validateClaimedChange(ctx, change); err != nil {
			_ = s.repository.FailClaimedScheduledChange(ctx, change, err.Error())
			continue
		}
		if err := s.repository.ApplyClaimedScheduledChange(ctx, change); err != nil {
			_ = s.repository.FailClaimedScheduledChange(ctx, change, err.Error())
			continue
		}
		completed++
	}
	return completed, nil
}

func (s *Service) validateClaimedChange(ctx context.Context, change ScheduledChange) error {
	if change.Patch.Policy == nil {
		return nil
	}
	state, err := s.repository.GetFlagState(ctx, change.OrganisationID, change.ProjectID, change.FeatureFlagID)
	if err != nil {
		return err
	}
	if err := evaluation.ValidateDefinition(state.Kind, state.DefaultValue, state.Variants, *change.Patch.Policy); err != nil {
		return err
	}
	segments, err := s.repository.ListSegments(ctx, change.OrganisationID, change.ProjectID)
	if err != nil {
		return err
	}
	return evaluation.ValidatePolicySegments(*change.Patch.Policy, evaluationSegments(segments))
}

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func normalizeSegmentInput(input SegmentInput, requireKey bool) (SegmentInput, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return SegmentInput{}, &ValidationError{Field: "name", Message: "is required"}
	}
	if len(name) > 160 {
		return SegmentInput{}, &ValidationError{Field: "name", Message: "is too long"}
	}

	key := strings.ToLower(strings.TrimSpace(input.Key))
	if requireKey || key != "" {
		if !keyPattern.MatchString(key) {
			return SegmentInput{}, &ValidationError{Field: "key", Message: "must use lowercase letters, numbers, dots, hyphens or underscores and be at most 128 characters"}
		}
	}

	description := strings.TrimSpace(input.Description)
	if len(description) > 2000 {
		return SegmentInput{}, &ValidationError{Field: "description", Message: "is too long"}
	}
	match := input.Match
	if match == "" {
		match = evaluation.MatchAll
	}
	segment := evaluation.Segment{Key: keyOrValidation(key), Name: name, Match: match, Conditions: input.Conditions}
	if err := evaluation.ValidateSegment(segment); err != nil {
		return SegmentInput{}, &ValidationError{Field: "conditions", Message: err.Error()}
	}
	return SegmentInput{Name: name, Key: key, Description: description, Match: match, Conditions: input.Conditions}, nil
}

func keyOrValidation(key string) string {
	if key == "" {
		return "validation-segment"
	}
	return key
}

func evaluationSegments(segments []Segment) []evaluation.Segment {
	result := make([]evaluation.Segment, 0, len(segments))
	for _, segment := range segments {
		result = append(result, evaluationSegmentFromModel(segment))
	}
	return result
}

func evaluationSegment(input SegmentInput) evaluation.Segment {
	return evaluation.Segment{Key: input.Key, Name: input.Name, Match: input.Match, Conditions: input.Conditions}
}

func evaluationSegmentFromModel(segment Segment) evaluation.Segment {
	return evaluation.Segment{Key: segment.Key, Name: segment.Name, Match: segment.Match, Conditions: segment.Conditions}
}
