package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	coresdkconfig "github.com/flagstack/flagstack/backend/internal/sdkconfig"
)

type sdkConfigHandlers struct {
	service *coresdkconfig.Service
}

type createSDKCredentialRequest struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type clientVisibilityRequest struct {
	ClientVisible bool `json:"client_visible"`
}

type clientVisibilityResponse struct {
	ClientVisible bool `json:"client_visible"`
}

type sdkCredentialListResponse struct {
	Credentials []coresdkconfig.Credential `json:"credentials"`
}

func newSDKConfigHandlers(service *coresdkconfig.Service) *sdkConfigHandlers {
	return &sdkConfigHandlers{service: service}
}

func (h *sdkConfigHandlers) configuration(w http.ResponseWriter, r *http.Request) {
	setSDKCORSHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET, OPTIONS")
		writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET and OPTIONS are supported.")
		return
	}

	key, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		writeSDKUnauthorized(w)
		return
	}
	configuration, err := h.service.ConfigurationForKey(r.Context(), key)
	if err != nil {
		if errors.Is(err, coresdkconfig.ErrInvalidCredential) || errors.Is(err, coresdkconfig.ErrCredentialNotFound) {
			writeSDKUnauthorized(w)
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "SDK configuration could not be loaded.")
		return
	}
	body, err := json.Marshal(configuration)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "SDK configuration could not be encoded.")
		return
	}
	digest := sha256.Sum256(body)
	etag := `"sha256-` + hex.EncodeToString(digest[:]) + `"`
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "private, max-age=0, must-revalidate")
	w.Header().Set("Vary", "Authorization")
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func (h *sdkConfigHandlers) listCredentials(w http.ResponseWriter, r *http.Request) {
	membership, ok := organisationMembership(r, r.PathValue("organisation"))
	if !ok {
		writeAPIError(w, http.StatusNotFound, "organisation_not_found", "Organisation was not found.")
		return
	}
	if !canManageSDKCredentials(membership.Role) {
		writeAPIError(w, http.StatusForbidden, "forbidden", "This role cannot manage SDK credentials.")
		return
	}
	credentials, err := h.service.ListCredentials(r.Context(), membership.ID, r.PathValue("project"))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "SDK credentials could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, sdkCredentialListResponse{Credentials: credentials})
}

func (h *sdkConfigHandlers) createCredential(w http.ResponseWriter, r *http.Request) {
	membership, ok := organisationMembership(r, r.PathValue("organisation"))
	if !ok {
		writeAPIError(w, http.StatusNotFound, "organisation_not_found", "Organisation was not found.")
		return
	}
	if !canManageSDKCredentials(membership.Role) {
		writeAPIError(w, http.StatusForbidden, "forbidden", "This role cannot manage SDK credentials.")
		return
	}
	var request createSDKCredentialRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "Request body must be valid JSON.")
		return
	}
	created, err := h.service.CreateCredential(r.Context(), membership.ID, r.PathValue("project"), coresdkconfig.CreateInput{
		Name:          request.Name,
		Kind:          request.Kind,
		EnvironmentID: r.PathValue("environment"),
	})
	if err != nil {
		var validationErr *coresdkconfig.ValidationError
		switch {
		case errors.As(err, &validationErr):
			writeAPIError(w, http.StatusBadRequest, "validation_error", validationErr.Error())
		case errors.Is(err, coresdkconfig.ErrEnvironmentNotFound):
			writeAPIError(w, http.StatusNotFound, "environment_not_found", "Environment was not found.")
		default:
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "SDK credential could not be created.")
		}
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *sdkConfigHandlers) revokeCredential(w http.ResponseWriter, r *http.Request) {
	membership, ok := organisationMembership(r, r.PathValue("organisation"))
	if !ok {
		writeAPIError(w, http.StatusNotFound, "organisation_not_found", "Organisation was not found.")
		return
	}
	if !canManageSDKCredentials(membership.Role) {
		writeAPIError(w, http.StatusForbidden, "forbidden", "This role cannot manage SDK credentials.")
		return
	}
	credential, err := h.service.RevokeCredential(r.Context(), membership.ID, r.PathValue("project"), r.PathValue("credential"))
	if err != nil {
		if errors.Is(err, coresdkconfig.ErrCredentialNotFound) {
			writeAPIError(w, http.StatusNotFound, "sdk_credential_not_found", "SDK credential was not found.")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "SDK credential could not be revoked.")
		return
	}
	writeJSON(w, http.StatusOK, credential)
}

func (h *sdkConfigHandlers) setClientVisible(w http.ResponseWriter, r *http.Request) {
	membership, ok := organisationMembership(r, r.PathValue("organisation"))
	if !ok {
		writeAPIError(w, http.StatusNotFound, "organisation_not_found", "Organisation was not found.")
		return
	}
	if !canManageSDKCredentials(membership.Role) {
		writeAPIError(w, http.StatusForbidden, "forbidden", "This role cannot expose flags to client SDKs.")
		return
	}
	var request clientVisibilityRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "Request body must be valid JSON.")
		return
	}
	visible, err := h.service.SetClientVisible(r.Context(), membership.ID, r.PathValue("project"), r.PathValue("featureFlag"), request.ClientVisible)
	if err != nil {
		if errors.Is(err, coresdkconfig.ErrFeatureFlagNotFound) {
			writeAPIError(w, http.StatusNotFound, "feature_flag_not_found", "Feature flag was not found.")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Client visibility could not be updated.")
		return
	}
	writeJSON(w, http.StatusOK, clientVisibilityResponse{ClientVisible: visible})
}

func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return "", false
	}
	return parts[1], true
}

func writeSDKUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="FlagStack SDK"`)
	writeAPIError(w, http.StatusUnauthorized, "invalid_sdk_credential", "A valid SDK credential is required.")
}

func setSDKCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, If-None-Match")
	w.Header().Set("Access-Control-Expose-Headers", "ETag")
}

func canManageSDKCredentials(role string) bool {
	return role == "owner" || role == "admin"
}
