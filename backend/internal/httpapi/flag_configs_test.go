package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	coreauth "github.com/flagstack/flagstack/backend/internal/auth"
	coreflagconfig "github.com/flagstack/flagstack/backend/internal/flagconfig"
)

type fakeFlagConfigRepository struct {
	configs        []coreflagconfig.Config
	setError       error
	lastSetEnabled bool
}

func (f *fakeFlagConfigRepository) List(context.Context, string, string) ([]coreflagconfig.Config, error) {
	return append([]coreflagconfig.Config(nil), f.configs...), nil
}

func (f *fakeFlagConfigRepository) SetEnabled(_ context.Context, _, _, environmentID, featureFlagID string, enabled bool) (coreflagconfig.Config, error) {
	if f.setError != nil {
		return coreflagconfig.Config{}, f.setError
	}
	f.lastSetEnabled = enabled
	config := coreflagconfig.Config{EnvironmentID: environmentID, FeatureFlagID: featureFlagID, Enabled: enabled, Revision: 1}
	f.configs = []coreflagconfig.Config{config}
	return config, nil
}

func TestFlagConfigHandlersSetAndList(t *testing.T) {
	repository := &fakeFlagConfigRepository{}
	service, err := coreflagconfig.NewService(repository)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	handlers := newFlagConfigHandlers(service)

	setRequest := authenticatedFlagConfigRequest(http.MethodPut, `{"enabled":true}`, "developer")
	setResponse := httptest.NewRecorder()
	handlers.setEnabled(setResponse, setRequest)
	if setResponse.Code != http.StatusOK {
		t.Fatalf("set status = %d, body = %q", setResponse.Code, setResponse.Body.String())
	}
	if !repository.lastSetEnabled || !strings.Contains(setResponse.Body.String(), `"revision":1`) {
		t.Fatalf("set response = %q", setResponse.Body.String())
	}

	listRequest := authenticatedFlagConfigRequest(http.MethodGet, "", "viewer")
	listResponse := httptest.NewRecorder()
	handlers.list(listResponse, listRequest)
	if listResponse.Code != http.StatusOK || !strings.Contains(listResponse.Body.String(), `"enabled":true`) {
		t.Fatalf("list response = %d %q", listResponse.Code, listResponse.Body.String())
	}
}

func TestFlagConfigHandlerRejectsViewerMutation(t *testing.T) {
	repository := &fakeFlagConfigRepository{}
	service, err := coreflagconfig.NewService(repository)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	handlers := newFlagConfigHandlers(service)

	request := authenticatedFlagConfigRequest(http.MethodPut, `{"enabled":true}`, "viewer")
	response := httptest.NewRecorder()
	handlers.setEnabled(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("viewer response = %d %q", response.Code, response.Body.String())
	}
}

func TestFlagConfigHandlerMapsMissingTargets(t *testing.T) {
	for name, repositoryError := range map[string]error{
		"environment":  coreflagconfig.ErrEnvironmentNotFound,
		"feature flag": coreflagconfig.ErrFeatureFlagNotFound,
	} {
		t.Run(name, func(t *testing.T) {
			repository := &fakeFlagConfigRepository{setError: repositoryError}
			service, err := coreflagconfig.NewService(repository)
			if err != nil {
				t.Fatalf("NewService() error = %v", err)
			}
			handlers := newFlagConfigHandlers(service)
			response := httptest.NewRecorder()
			handlers.setEnabled(response, authenticatedFlagConfigRequest(http.MethodPut, `{"enabled":true}`, "admin"))
			if response.Code != http.StatusNotFound {
				t.Fatalf("response = %d %q", response.Code, response.Body.String())
			}
		})
	}
}

func authenticatedFlagConfigRequest(method, body, role string) *http.Request {
	var request *http.Request
	if body == "" {
		request = httptest.NewRequest(method, "/", nil)
	} else {
		request = httptest.NewRequest(method, "/", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
	}
	request.SetPathValue("organisation", "example")
	request.SetPathValue("project", "project-1")
	request.SetPathValue("environment", "environment-1")
	request.SetPathValue("featureFlag", "flag-1")

	principal := coreauth.Principal{
		User: coreauth.User{ID: "user-1", Email: "user@example.com", DisplayName: "User"},
		Organisations: []coreauth.OrganisationMembership{{
			ID: "org-1", Name: "Example", Slug: "example", Role: role,
		}},
	}
	ctx := context.WithValue(request.Context(), authenticatedRequestKey{}, authenticatedRequest{
		Session: coreauth.Session{Principal: principal},
	})
	return request.WithContext(ctx)
}
