package httpapi

import (
	"errors"
	"net/http"

	coreflagconfig "github.com/switchonyourcode/switchonyourcode/backend/internal/flagconfig"
)

type flagConfigHandlers struct {
	service *coreflagconfig.Service
}

type setFlagEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

type flagConfigResponse struct {
	EnvironmentID string `json:"environment_id"`
	FeatureFlagID string `json:"feature_flag_id"`
	Enabled       bool   `json:"enabled"`
	Revision      int64  `json:"revision"`
}

type flagConfigListResponse struct {
	Configs []flagConfigResponse `json:"configs"`
}

func newFlagConfigHandlers(service *coreflagconfig.Service) *flagConfigHandlers {
	return &flagConfigHandlers{service: service}
}

func (h *flagConfigHandlers) list(w http.ResponseWriter, r *http.Request) {
	membership, ok := organisationMembership(r, r.PathValue("organisation"))
	if !ok {
		writeAPIError(w, http.StatusNotFound, "organisation_not_found", "Organisation was not found.")
		return
	}

	configs, err := h.service.List(r.Context(), membership.ID, r.PathValue("project"))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Feature flag configuration could not be loaded.")
		return
	}

	response := make([]flagConfigResponse, 0, len(configs))
	for _, config := range configs {
		response = append(response, flagConfigFromCore(config))
	}
	writeJSON(w, http.StatusOK, flagConfigListResponse{Configs: response})
}

func (h *flagConfigHandlers) setEnabled(w http.ResponseWriter, r *http.Request) {
	membership, ok := organisationMembership(r, r.PathValue("organisation"))
	if !ok {
		writeAPIError(w, http.StatusNotFound, "organisation_not_found", "Organisation was not found.")
		return
	}
	if membership.Role == "viewer" {
		writeAPIError(w, http.StatusForbidden, "forbidden", "This role cannot change feature flag configuration.")
		return
	}

	var request setFlagEnabledRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "Request body must be valid JSON.")
		return
	}

	config, err := h.service.SetEnabled(
		r.Context(),
		membership.ID,
		r.PathValue("project"),
		r.PathValue("environment"),
		r.PathValue("featureFlag"),
		request.Enabled,
	)
	if err != nil {
		switch {
		case errors.Is(err, coreflagconfig.ErrEnvironmentNotFound):
			writeAPIError(w, http.StatusNotFound, "environment_not_found", "Environment was not found.")
		case errors.Is(err, coreflagconfig.ErrFeatureFlagNotFound):
			writeAPIError(w, http.StatusNotFound, "feature_flag_not_found", "Feature flag was not found.")
		default:
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "Feature flag configuration could not be updated.")
		}
		return
	}

	writeJSON(w, http.StatusOK, flagConfigFromCore(config))
}

func flagConfigFromCore(config coreflagconfig.Config) flagConfigResponse {
	return flagConfigResponse{
		EnvironmentID: config.EnvironmentID,
		FeatureFlagID: config.FeatureFlagID,
		Enabled:       config.Enabled,
		Revision:      config.Revision,
	}
}
