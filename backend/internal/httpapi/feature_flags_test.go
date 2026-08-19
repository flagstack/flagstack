package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	coreauth "github.com/flagstack/flagstack/backend/internal/auth"
	corefeatureflag "github.com/flagstack/flagstack/backend/internal/featureflag"
)

type fakeFeatureFlagRepository struct {
	flags []corefeatureflag.FeatureFlag
}

func (f *fakeFeatureFlagRepository) List(context.Context, string, string) ([]corefeatureflag.FeatureFlag, error) {
	return append([]corefeatureflag.FeatureFlag(nil), f.flags...), nil
}

func (f *fakeFeatureFlagRepository) Create(_ context.Context, organisationID, projectID string, input corefeatureflag.CreateInput) (corefeatureflag.FeatureFlag, error) {
	for _, flag := range f.flags {
		if flag.ProjectID == projectID && flag.Key == input.Key {
			return corefeatureflag.FeatureFlag{}, corefeatureflag.ErrKeyConflict
		}
	}
	flag := corefeatureflag.FeatureFlag{
		ID:             "flag-1",
		OrganisationID: organisationID,
		ProjectID:      projectID,
		Name:           input.Name,
		Key:            input.Key,
		Description:    input.Description,
		Kind:           input.Kind,
		DefaultValue:   input.DefaultValue,
		CreatedAt:      time.Unix(1, 0).UTC(),
	}
	f.flags = append(f.flags, flag)
	return flag, nil
}

func TestFeatureFlagHandlersCreateAndList(t *testing.T) {
	repository := &fakeFeatureFlagRepository{}
	service, err := corefeatureflag.NewService(repository)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	handlers := newFeatureFlagHandlers(service)

	createRequest := authenticatedFeatureFlagRequest(http.MethodPost, `{
		"name":"Checkout flow",
		"key":"checkout.new-flow",
		"description":"New checkout",
		"kind":"boolean",
		"default_value":false
	}`, "owner")
	createResponse := httptest.NewRecorder()
	handlers.create(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %q", createResponse.Code, createResponse.Body.String())
	}
	if !strings.Contains(createResponse.Body.String(), `"key":"checkout.new-flow"`) || !strings.Contains(createResponse.Body.String(), `"default_value":false`) {
		t.Fatalf("create body = %q", createResponse.Body.String())
	}

	listRequest := authenticatedFeatureFlagRequest(http.MethodGet, "", "developer")
	listResponse := httptest.NewRecorder()
	handlers.list(listResponse, listRequest)
	if listResponse.Code != http.StatusOK || !strings.Contains(listResponse.Body.String(), `"checkout.new-flow"`) {
		t.Fatalf("list response = %d %q", listResponse.Code, listResponse.Body.String())
	}
}

func TestFeatureFlagHandlerRejectsInvalidDefaultValueAndViewer(t *testing.T) {
	repository := &fakeFeatureFlagRepository{}
	service, err := corefeatureflag.NewService(repository)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	handlers := newFeatureFlagHandlers(service)

	invalidRequest := authenticatedFeatureFlagRequest(http.MethodPost, `{
		"name":"Checkout flow",
		"key":"checkout.new-flow",
		"kind":"boolean",
		"default_value":"false"
	}`, "developer")
	invalidResponse := httptest.NewRecorder()
	handlers.create(invalidResponse, invalidRequest)
	if invalidResponse.Code != http.StatusBadRequest || !strings.Contains(invalidResponse.Body.String(), `"code":"validation_error"`) {
		t.Fatalf("invalid response = %d %q", invalidResponse.Code, invalidResponse.Body.String())
	}

	viewerRequest := authenticatedFeatureFlagRequest(http.MethodPost, `{
		"name":"Viewer flag",
		"key":"viewer-flag",
		"kind":"boolean",
		"default_value":false
	}`, "viewer")
	viewerResponse := httptest.NewRecorder()
	handlers.create(viewerResponse, viewerRequest)
	if viewerResponse.Code != http.StatusForbidden {
		t.Fatalf("viewer response = %d %q", viewerResponse.Code, viewerResponse.Body.String())
	}
}

func TestFeatureFlagHandlerMapsDuplicateKeyConflict(t *testing.T) {
	repository := &fakeFeatureFlagRepository{}
	service, err := corefeatureflag.NewService(repository)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	handlers := newFeatureFlagHandlers(service)
	body := `{"name":"Flag","key":"same-key","kind":"number","default_value":1}`

	first := httptest.NewRecorder()
	handlers.create(first, authenticatedFeatureFlagRequest(http.MethodPost, body, "admin"))
	if first.Code != http.StatusCreated {
		t.Fatalf("first create status = %d", first.Code)
	}

	second := httptest.NewRecorder()
	handlers.create(second, authenticatedFeatureFlagRequest(http.MethodPost, body, "admin"))
	if second.Code != http.StatusConflict || !strings.Contains(second.Body.String(), `"code":"feature_flag_key_conflict"`) {
		t.Fatalf("duplicate response = %d %q", second.Code, second.Body.String())
	}
}

func authenticatedFeatureFlagRequest(method, body, role string) *http.Request {
	var request *http.Request
	if body == "" {
		request = httptest.NewRequest(method, "/", nil)
	} else {
		request = httptest.NewRequest(method, "/", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
	}
	request.SetPathValue("organisation", "example")
	request.SetPathValue("project", "project-1")

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
