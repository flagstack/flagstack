package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	corefeatureflag "github.com/flagstack/flagstack/backend/internal/featureflag"
)

type featureFlagHandlers struct {
	service *corefeatureflag.Service
}

type createFeatureFlagRequest struct {
	Name         string          `json:"name"`
	Key          string          `json:"key"`
	Description  string          `json:"description"`
	Kind         string          `json:"kind"`
	DefaultValue json.RawMessage `json:"default_value"`
}

type featureFlagResponse struct {
	ID            string          `json:"id"`
	ProjectID     string          `json:"project_id"`
	Name          string          `json:"name"`
	Key           string          `json:"key"`
	Description   string          `json:"description"`
	Kind          string          `json:"kind"`
	DefaultValue  json.RawMessage `json:"default_value"`
	ClientVisible bool            `json:"client_visible"`
	CreatedAt     time.Time       `json:"created_at"`
}

type featureFlagListResponse struct {
	FeatureFlags []featureFlagResponse `json:"feature_flags"`
}

func newFeatureFlagHandlers(service *corefeatureflag.Service) *featureFlagHandlers {
	return &featureFlagHandlers{service: service}
}

func (h *featureFlagHandlers) list(w http.ResponseWriter, r *http.Request) {
	membership, ok := organisationMembership(r, r.PathValue("organisation"))
	if !ok {
		writeAPIError(w, http.StatusNotFound, "organisation_not_found", "Organisation was not found.")
		return
	}

	flags, err := h.service.List(r.Context(), membership.ID, r.PathValue("project"))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Feature flags could not be loaded.")
		return
	}

	response := make([]featureFlagResponse, 0, len(flags))
	for _, flag := range flags {
		response = append(response, featureFlagFromCore(flag))
	}
	writeJSON(w, http.StatusOK, featureFlagListResponse{FeatureFlags: response})
}

func (h *featureFlagHandlers) create(w http.ResponseWriter, r *http.Request) {
	membership, ok := organisationMembership(r, r.PathValue("organisation"))
	if !ok {
		writeAPIError(w, http.StatusNotFound, "organisation_not_found", "Organisation was not found.")
		return
	}
	if membership.Role == "viewer" {
		writeAPIError(w, http.StatusForbidden, "forbidden", "This role cannot create feature flags.")
		return
	}

	var request createFeatureFlagRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "Request body must be valid JSON.")
		return
	}

	flag, err := h.service.Create(r.Context(), membership.ID, r.PathValue("project"), corefeatureflag.CreateInput{
		Name:         request.Name,
		Key:          request.Key,
		Description:  request.Description,
		Kind:         request.Kind,
		DefaultValue: request.DefaultValue,
	})
	if err != nil {
		var validationErr *corefeatureflag.ValidationError
		switch {
		case errors.As(err, &validationErr):
			writeAPIError(w, http.StatusBadRequest, "validation_error", validationErr.Error())
		case errors.Is(err, corefeatureflag.ErrKeyConflict):
			writeAPIError(w, http.StatusConflict, "feature_flag_key_conflict", "A feature flag with this key already exists in the project.")
		case errors.Is(err, corefeatureflag.ErrProjectNotFound):
			writeAPIError(w, http.StatusNotFound, "project_not_found", "Project was not found.")
		default:
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "Feature flag could not be created.")
		}
		return
	}

	writeJSON(w, http.StatusCreated, featureFlagFromCore(flag))
}

func featureFlagFromCore(flag corefeatureflag.FeatureFlag) featureFlagResponse {
	return featureFlagResponse{
		ID:            flag.ID,
		ProjectID:     flag.ProjectID,
		Name:          flag.Name,
		Key:           flag.Key,
		Description:   flag.Description,
		Kind:          flag.Kind,
		DefaultValue:  flag.DefaultValue,
		ClientVisible: flag.ClientVisible,
		CreatedAt:     flag.CreatedAt,
	}
}
