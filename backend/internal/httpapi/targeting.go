package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/flagstack/flagstack/backend/internal/evaluation"
	coretargeting "github.com/flagstack/flagstack/backend/internal/targeting"
)

type targetingHandlers struct {
	service *coretargeting.Service
}

type variantsRequest struct {
	Variants []evaluation.Variant `json:"variants"`
}

type policyRequest struct {
	Policy evaluation.Policy `json:"policy"`
}

type previewRequest struct {
	Context map[string]any `json:"context"`
}

type segmentRequest struct {
	Name        string                 `json:"name"`
	Key         string                 `json:"key"`
	Description string                 `json:"description"`
	Match       evaluation.MatchMode   `json:"match"`
	Conditions  []evaluation.Condition `json:"conditions"`
}

type scheduleRequest struct {
	EnvironmentID string                    `json:"environment_id"`
	FeatureFlagID string                    `json:"feature_flag_id"`
	ExecuteAt     time.Time                 `json:"execute_at"`
	Patch         coretargeting.SchedulePatch `json:"patch"`
}

type environmentStateResponse struct {
	EnvironmentID string            `json:"environment_id"`
	Enabled       bool              `json:"enabled"`
	Policy        evaluation.Policy `json:"policy"`
	Revision      int64             `json:"revision"`
}

type flagTargetingResponse struct {
	ID           string                     `json:"id"`
	Key          string                     `json:"key"`
	Kind         string                     `json:"kind"`
	DefaultValue any                        `json:"default_value"`
	Variants     []evaluation.Variant       `json:"variants"`
	Environments []environmentStateResponse `json:"environments"`
}

type segmentResponse struct {
	ID          string                 `json:"id"`
	ProjectID   string                 `json:"project_id"`
	Name        string                 `json:"name"`
	Key         string                 `json:"key"`
	Description string                 `json:"description"`
	Match       evaluation.MatchMode   `json:"match"`
	Conditions  []evaluation.Condition `json:"conditions"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type segmentListResponse struct {
	Segments []segmentResponse `json:"segments"`
}

type scheduledChangeResponse struct {
	ID              string                      `json:"id"`
	ProjectID       string                      `json:"project_id"`
	EnvironmentID   string                      `json:"environment_id"`
	FeatureFlagID   string                      `json:"feature_flag_id"`
	CreatedByUserID string                      `json:"created_by_user_id,omitempty"`
	ExecuteAt       time.Time                   `json:"execute_at"`
	Patch           coretargeting.SchedulePatch `json:"patch"`
	Status          string                      `json:"status"`
	ClaimedAt       *time.Time                  `json:"claimed_at,omitempty"`
	ExecutedAt      *time.Time                  `json:"executed_at,omitempty"`
	LastError       string                      `json:"last_error,omitempty"`
	CreatedAt       time.Time                   `json:"created_at"`
	UpdatedAt       time.Time                   `json:"updated_at"`
}

type scheduledChangeListResponse struct {
	ScheduledChanges []scheduledChangeResponse `json:"scheduled_changes"`
}

func newTargetingHandlers(service *coretargeting.Service) *targetingHandlers {
	return &targetingHandlers{service: service}
}

func (h *targetingHandlers) setVariants(w http.ResponseWriter, r *http.Request) {
	membership, ok := targetingMembership(w, r, true)
	if !ok {
		return
	}
	var request variantsRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "Request body must be valid JSON.")
		return
	}
	state, err := h.service.SetVariants(r.Context(), membership.ID, r.PathValue("project"), r.PathValue("featureFlag"), request.Variants)
	if err != nil {
		writeTargetingError(w, err, "Feature flag variants could not be updated.")
		return
	}
	writeJSON(w, http.StatusOK, flagTargetingFromCore(state))
}

func (h *targetingHandlers) setPolicy(w http.ResponseWriter, r *http.Request) {
	membership, ok := targetingMembership(w, r, true)
	if !ok {
		return
	}
	var request policyRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "Request body must be valid JSON.")
		return
	}
	state, err := h.service.SetPolicy(
		r.Context(), membership.ID, r.PathValue("project"), r.PathValue("environment"), r.PathValue("featureFlag"), request.Policy,
	)
	if err != nil {
		writeTargetingError(w, err, "Feature flag policy could not be updated.")
		return
	}
	writeJSON(w, http.StatusOK, environmentStateFromCore(state))
}

func (h *targetingHandlers) preview(w http.ResponseWriter, r *http.Request) {
	membership, ok := targetingMembership(w, r, false)
	if !ok {
		return
	}
	var request previewRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "Request body must be valid JSON.")
		return
	}
	context := evaluationContext(request.Context)
	result, err := h.service.Preview(
		r.Context(), membership.ID, r.PathValue("project"), r.PathValue("environment"), r.PathValue("featureFlag"), context,
	)
	if err != nil {
		writeTargetingError(w, err, "Feature flag evaluation could not be previewed.")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *targetingHandlers) listSegments(w http.ResponseWriter, r *http.Request) {
	membership, ok := targetingMembership(w, r, false)
	if !ok {
		return
	}
	segments, err := h.service.ListSegments(r.Context(), membership.ID, r.PathValue("project"))
	if err != nil {
		writeTargetingError(w, err, "Segments could not be loaded.")
		return
	}
	response := make([]segmentResponse, 0, len(segments))
	for _, segment := range segments {
		response = append(response, segmentFromCore(segment))
	}
	writeJSON(w, http.StatusOK, segmentListResponse{Segments: response})
}

func (h *targetingHandlers) createSegment(w http.ResponseWriter, r *http.Request) {
	membership, ok := targetingMembership(w, r, true)
	if !ok {
		return
	}
	var request segmentRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "Request body must be valid JSON.")
		return
	}
	segment, err := h.service.CreateSegment(r.Context(), membership.ID, r.PathValue("project"), coretargeting.SegmentInput{
		Name: request.Name, Key: request.Key, Description: request.Description, Match: request.Match, Conditions: request.Conditions,
	})
	if err != nil {
		writeTargetingError(w, err, "Segment could not be created.")
		return
	}
	writeJSON(w, http.StatusCreated, segmentFromCore(segment))
}

func (h *targetingHandlers) updateSegment(w http.ResponseWriter, r *http.Request) {
	membership, ok := targetingMembership(w, r, true)
	if !ok {
		return
	}
	var request segmentRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "Request body must be valid JSON.")
		return
	}
	segment, err := h.service.UpdateSegment(r.Context(), membership.ID, r.PathValue("project"), r.PathValue("segment"), coretargeting.SegmentInput{
		Name: request.Name, Key: request.Key, Description: request.Description, Match: request.Match, Conditions: request.Conditions,
	})
	if err != nil {
		writeTargetingError(w, err, "Segment could not be updated.")
		return
	}
	writeJSON(w, http.StatusOK, segmentFromCore(segment))
}

func (h *targetingHandlers) listScheduledChanges(w http.ResponseWriter, r *http.Request) {
	membership, ok := targetingMembership(w, r, false)
	if !ok {
		return
	}
	changes, err := h.service.ListScheduledChanges(r.Context(), membership.ID, r.PathValue("project"))
	if err != nil {
		writeTargetingError(w, err, "Scheduled changes could not be loaded.")
		return
	}
	response := make([]scheduledChangeResponse, 0, len(changes))
	for _, change := range changes {
		response = append(response, scheduledChangeFromCore(change))
	}
	writeJSON(w, http.StatusOK, scheduledChangeListResponse{ScheduledChanges: response})
}

func (h *targetingHandlers) createScheduledChange(w http.ResponseWriter, r *http.Request) {
	membership, ok := targetingMembership(w, r, true)
	if !ok {
		return
	}
	var request scheduleRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "Request body must be valid JSON.")
		return
	}
	authenticated, _ := authenticatedFromContext(r.Context())
	change, err := h.service.CreateScheduledChange(r.Context(), membership.ID, r.PathValue("project"), coretargeting.CreateScheduleInput{
		EnvironmentID: request.EnvironmentID,
		FeatureFlagID: request.FeatureFlagID,
		CreatedByUserID: authenticated.Session.Principal.User.ID,
		ExecuteAt: request.ExecuteAt,
		Patch: request.Patch,
	})
	if err != nil {
		writeTargetingError(w, err, "Scheduled change could not be created.")
		return
	}
	writeJSON(w, http.StatusCreated, scheduledChangeFromCore(change))
}

func (h *targetingHandlers) cancelScheduledChange(w http.ResponseWriter, r *http.Request) {
	membership, ok := targetingMembership(w, r, true)
	if !ok {
		return
	}
	change, err := h.service.CancelScheduledChange(r.Context(), membership.ID, r.PathValue("project"), r.PathValue("schedule"))
	if err != nil {
		writeTargetingError(w, err, "Scheduled change could not be cancelled.")
		return
	}
	writeJSON(w, http.StatusOK, scheduledChangeFromCore(change))
}

func targetingMembership(w http.ResponseWriter, r *http.Request, mutate bool) (membershipResponse, bool) {
	membership, ok := organisationMembership(r, r.PathValue("organisation"))
	if !ok {
		writeAPIError(w, http.StatusNotFound, "organisation_not_found", "Organisation was not found.")
		return membershipResponse{}, false
	}
	if mutate && membership.Role == "viewer" {
		writeAPIError(w, http.StatusForbidden, "forbidden", "This role cannot change feature targeting.")
		return membershipResponse{}, false
	}
	return membershipResponse{ID: membership.ID, Role: membership.Role}, true
}

type membershipResponse struct {
	ID   string
	Role string
}

func evaluationContext(values map[string]any) evaluation.Context {
	attributes := make(map[string]any, len(values))
	var targetingKey string
	for key, value := range values {
		if key == "targetingKey" {
			if candidate, ok := value.(string); ok {
				targetingKey = candidate
			}
			continue
		}
		attributes[key] = value
	}
	return evaluation.Context{TargetingKey: targetingKey, Attributes: attributes}
}

func flagTargetingFromCore(state coretargeting.FlagState) flagTargetingResponse {
	environments := make([]environmentStateResponse, 0, len(state.Environments))
	for _, environment := range state.Environments {
		environments = append(environments, environmentStateFromCore(environment))
	}
	return flagTargetingResponse{
		ID: state.ID, Key: state.Key, Kind: state.Kind, DefaultValue: state.DefaultValue,
		Variants: state.Variants, Environments: environments,
	}
}

func environmentStateFromCore(state coretargeting.EnvironmentState) environmentStateResponse {
	return environmentStateResponse{
		EnvironmentID: state.EnvironmentID, Enabled: state.Enabled, Policy: state.Policy, Revision: state.Revision,
	}
}

func segmentFromCore(segment coretargeting.Segment) segmentResponse {
	return segmentResponse{
		ID: segment.ID, ProjectID: segment.ProjectID, Name: segment.Name, Key: segment.Key,
		Description: segment.Description, Match: segment.Match, Conditions: segment.Conditions,
		CreatedAt: segment.CreatedAt, UpdatedAt: segment.UpdatedAt,
	}
}

func scheduledChangeFromCore(change coretargeting.ScheduledChange) scheduledChangeResponse {
	return scheduledChangeResponse{
		ID: change.ID, ProjectID: change.ProjectID, EnvironmentID: change.EnvironmentID, FeatureFlagID: change.FeatureFlagID,
		CreatedByUserID: change.CreatedByUserID, ExecuteAt: change.ExecuteAt, Patch: change.Patch, Status: change.Status,
		ClaimedAt: change.ClaimedAt, ExecutedAt: change.ExecutedAt, LastError: change.LastError,
		CreatedAt: change.CreatedAt, UpdatedAt: change.UpdatedAt,
	}
}

func writeTargetingError(w http.ResponseWriter, err error, fallback string) {
	var validationErr *coretargeting.ValidationError
	switch {
	case errors.As(err, &validationErr):
		writeAPIError(w, http.StatusBadRequest, "validation_error", validationErr.Error())
	case errors.Is(err, coretargeting.ErrProjectNotFound):
		writeAPIError(w, http.StatusNotFound, "project_not_found", "Project was not found.")
	case errors.Is(err, coretargeting.ErrEnvironmentNotFound):
		writeAPIError(w, http.StatusNotFound, "environment_not_found", "Environment was not found.")
	case errors.Is(err, coretargeting.ErrFeatureFlagNotFound):
		writeAPIError(w, http.StatusNotFound, "feature_flag_not_found", "Feature flag was not found.")
	case errors.Is(err, coretargeting.ErrSegmentNotFound):
		writeAPIError(w, http.StatusNotFound, "segment_not_found", "Segment was not found.")
	case errors.Is(err, coretargeting.ErrSegmentKeyConflict):
		writeAPIError(w, http.StatusConflict, "segment_key_conflict", "A segment with this key already exists in the project.")
	case errors.Is(err, coretargeting.ErrScheduleNotFound):
		writeAPIError(w, http.StatusNotFound, "schedule_not_found", "Scheduled change was not found.")
	case errors.Is(err, coretargeting.ErrScheduleNotPending):
		writeAPIError(w, http.StatusConflict, "schedule_not_pending", "Only pending scheduled changes can be modified.")
	default:
		writeAPIError(w, http.StatusInternalServerError, "internal_error", fallback)
	}
}
