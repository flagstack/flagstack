package httpapi

import (
	"errors"
	"net/http"
	"time"

	coreauth "github.com/flagstack/flagstack/backend/internal/auth"
	coreproject "github.com/flagstack/flagstack/backend/internal/project"
)

type projectHandlers struct {
	service *coreproject.Service
}

type createProjectRequest struct {
	Name        string `json:"name"`
	Key         string `json:"key"`
	Description string `json:"description"`
}

type projectResponse struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Key              string    `json:"key"`
	Description      string    `json:"description"`
	EnvironmentCount int       `json:"environment_count"`
	FeatureFlagCount int       `json:"feature_flag_count"`
	CreatedAt        time.Time `json:"created_at"`
}

type projectListResponse struct {
	Projects []projectResponse `json:"projects"`
}

func newProjectHandlers(service *coreproject.Service) *projectHandlers {
	return &projectHandlers{service: service}
}

func (h *projectHandlers) list(w http.ResponseWriter, r *http.Request) {
	membership, ok := organisationMembership(r, r.PathValue("organisation"))
	if !ok {
		writeAPIError(w, http.StatusNotFound, "organisation_not_found", "Organisation was not found.")
		return
	}

	projects, err := h.service.List(r.Context(), membership.ID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Projects could not be loaded.")
		return
	}

	response := make([]projectResponse, 0, len(projects))
	for _, project := range projects {
		response = append(response, projectFromCore(project))
	}
	writeJSON(w, http.StatusOK, projectListResponse{Projects: response})
}

func (h *projectHandlers) create(w http.ResponseWriter, r *http.Request) {
	membership, ok := organisationMembership(r, r.PathValue("organisation"))
	if !ok {
		writeAPIError(w, http.StatusNotFound, "organisation_not_found", "Organisation was not found.")
		return
	}
	if membership.Role != "owner" && membership.Role != "admin" {
		writeAPIError(w, http.StatusForbidden, "forbidden", "This role cannot create projects.")
		return
	}

	var request createProjectRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "Request body must be valid JSON.")
		return
	}

	project, err := h.service.Create(r.Context(), membership.ID, coreproject.CreateInput{
		Name: request.Name, Key: request.Key, Description: request.Description,
	})
	if err != nil {
		var validationErr *coreproject.ValidationError
		switch {
		case errors.As(err, &validationErr):
			writeAPIError(w, http.StatusBadRequest, "validation_error", validationErr.Error())
		case errors.Is(err, coreproject.ErrKeyConflict):
			writeAPIError(w, http.StatusConflict, "project_key_conflict", "A project with this key already exists.")
		default:
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "Project could not be created.")
		}
		return
	}

	writeJSON(w, http.StatusCreated, projectFromCore(project))
}

func organisationMembership(r *http.Request, slug string) (coreauth.OrganisationMembership, bool) {
	authenticated, ok := authenticatedFromContext(r.Context())
	if !ok {
		return coreauth.OrganisationMembership{}, false
	}
	for _, membership := range authenticated.Session.Principal.Organisations {
		if membership.Slug == slug {
			return membership, true
		}
	}
	return coreauth.OrganisationMembership{}, false
}

func projectFromCore(project coreproject.Project) projectResponse {
	return projectResponse{
		ID:               project.ID,
		Name:             project.Name,
		Key:              project.Key,
		Description:      project.Description,
		EnvironmentCount: project.EnvironmentCount,
		FeatureFlagCount: project.FeatureFlagCount,
		CreatedAt:        project.CreatedAt,
	}
}
