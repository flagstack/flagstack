package httpapi

import (
	"errors"
	"net/http"
	"time"

	coreenvironment "github.com/switchonyourcode/switchonyourcode/backend/internal/environment"
)

type environmentHandlers struct {
	service *coreenvironment.Service
}

type createEnvironmentRequest struct {
	Name        string `json:"name"`
	Key         string `json:"key"`
	Description string `json:"description"`
}

type environmentResponse struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Name        string    `json:"name"`
	Key         string    `json:"key"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type environmentListResponse struct {
	Environments []environmentResponse `json:"environments"`
}

func newEnvironmentHandlers(service *coreenvironment.Service) *environmentHandlers {
	return &environmentHandlers{service: service}
}

func (h *environmentHandlers) list(w http.ResponseWriter, r *http.Request) {
	membership, ok := organisationMembership(r, r.PathValue("organisation"))
	if !ok {
		writeAPIError(w, http.StatusNotFound, "organisation_not_found", "Organisation was not found.")
		return
	}

	environments, err := h.service.List(r.Context(), membership.ID, r.PathValue("project"))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Environments could not be loaded.")
		return
	}

	response := make([]environmentResponse, 0, len(environments))
	for _, environment := range environments {
		response = append(response, environmentFromCore(environment))
	}
	writeJSON(w, http.StatusOK, environmentListResponse{Environments: response})
}

func (h *environmentHandlers) create(w http.ResponseWriter, r *http.Request) {
	membership, ok := organisationMembership(r, r.PathValue("organisation"))
	if !ok {
		writeAPIError(w, http.StatusNotFound, "organisation_not_found", "Organisation was not found.")
		return
	}
	if membership.Role == "viewer" {
		writeAPIError(w, http.StatusForbidden, "forbidden", "This role cannot create environments.")
		return
	}

	var request createEnvironmentRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "Request body must be valid JSON.")
		return
	}

	environment, err := h.service.Create(r.Context(), membership.ID, r.PathValue("project"), coreenvironment.CreateInput{
		Name: request.Name, Key: request.Key, Description: request.Description,
	})
	if err != nil {
		var validationErr *coreenvironment.ValidationError
		switch {
		case errors.As(err, &validationErr):
			writeAPIError(w, http.StatusBadRequest, "validation_error", validationErr.Error())
		case errors.Is(err, coreenvironment.ErrKeyConflict):
			writeAPIError(w, http.StatusConflict, "environment_key_conflict", "An environment with this key already exists in the project.")
		case errors.Is(err, coreenvironment.ErrProjectNotFound):
			writeAPIError(w, http.StatusNotFound, "project_not_found", "Project was not found.")
		default:
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "Environment could not be created.")
		}
		return
	}

	writeJSON(w, http.StatusCreated, environmentFromCore(environment))
}

func environmentFromCore(environment coreenvironment.Environment) environmentResponse {
	return environmentResponse{
		ID: environment.ID, ProjectID: environment.ProjectID, Name: environment.Name, Key: environment.Key,
		Description: environment.Description, CreatedAt: environment.CreatedAt,
	}
}
