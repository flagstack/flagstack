package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	coreauth "github.com/flagstack/flagstack/backend/internal/auth"
	coresdkconfig "github.com/flagstack/flagstack/backend/internal/sdkconfig"
)

type fakeSDKConfigRepository struct {
	credential    coresdkconfig.Credential
	configuration coresdkconfig.Configuration
	visible       bool
}

func (f *fakeSDKConfigRepository) ListCredentials(context.Context, string, string) ([]coresdkconfig.Credential, error) {
	if f.credential.ID == "" {
		return []coresdkconfig.Credential{}, nil
	}
	return []coresdkconfig.Credential{f.credential}, nil
}

func (f *fakeSDKConfigRepository) CreateCredential(_ context.Context, record coresdkconfig.CredentialRecord) (coresdkconfig.Credential, error) {
	f.credential = coresdkconfig.Credential{
		ID:             record.ID.String(),
		OrganisationID: record.OrganisationID,
		ProjectID:      record.ProjectID,
		EnvironmentID:  record.EnvironmentID,
		Name:           record.Name,
		Kind:           record.Kind,
		ClientKey:      record.ClientKey,
		CreatedAt:      time.Now().UTC(),
	}
	return f.credential, nil
}

func (f *fakeSDKConfigRepository) RevokeCredential(_ context.Context, _, _, credentialID string, revokedAt time.Time) (coresdkconfig.Credential, error) {
	if f.credential.ID != credentialID {
		return coresdkconfig.Credential{}, coresdkconfig.ErrCredentialNotFound
	}
	f.credential.RevokedAt = &revokedAt
	return f.credential, nil
}

func (f *fakeSDKConfigRepository) FindServerCredential(context.Context, string) (coresdkconfig.StoredServerCredential, error) {
	return coresdkconfig.StoredServerCredential{}, coresdkconfig.ErrCredentialNotFound
}

func (f *fakeSDKConfigRepository) FindClientCredential(_ context.Context, clientKey string) (coresdkconfig.Credential, error) {
	if f.credential.ClientKey != clientKey || f.credential.RevokedAt != nil {
		return coresdkconfig.Credential{}, coresdkconfig.ErrCredentialNotFound
	}
	return f.credential, nil
}

func (f *fakeSDKConfigRepository) SetClientVisible(context.Context, string, string, string, bool) (bool, error) {
	return f.visible, nil
}

func (f *fakeSDKConfigRepository) LoadConfiguration(context.Context, coresdkconfig.Credential) (coresdkconfig.Configuration, error) {
	return f.configuration, nil
}

func TestSDKConfigurationHandlerSupportsETagAndCORS(t *testing.T) {
	repository := &fakeSDKConfigRepository{
		credential: coresdkconfig.Credential{
			ID: "credential-1", OrganisationID: "org-1", ProjectID: "project-1", EnvironmentID: "env-1",
			Name: "Browser", Kind: coresdkconfig.KindClient, ClientKey: "fs_client_test",
		},
		configuration: coresdkconfig.Configuration{
			Environment: coresdkconfig.Environment{ID: "env-1", Key: "production"},
		},
	}
	service, err := coresdkconfig.NewService(repository)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	handler := newSDKConfigHandlers(service)

	request := httptest.NewRequest(http.MethodGet, "/sdk/v1/config", nil)
	request.Header.Set("Authorization", "Bearer fs_client_test")
	response := httptest.NewRecorder()
	handler.configuration(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", response.Code, response.Body.String())
	}
	etag := response.Header().Get("ETag")
	if etag == "" || !strings.Contains(response.Body.String(), `"schema_version":1`) {
		t.Fatalf("etag = %q, body = %q", etag, response.Body.String())
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "*" || response.Header().Get("Access-Control-Expose-Headers") != "ETag" {
		t.Fatalf("CORS headers = %#v", response.Header())
	}

	revalidate := httptest.NewRequest(http.MethodGet, "/sdk/v1/config", nil)
	revalidate.Header.Set("Authorization", "Bearer fs_client_test")
	revalidate.Header.Set("If-None-Match", etag)
	revalidationResponse := httptest.NewRecorder()
	handler.configuration(revalidationResponse, revalidate)
	if revalidationResponse.Code != http.StatusNotModified || revalidationResponse.Body.Len() != 0 {
		t.Fatalf("revalidation = %d %q", revalidationResponse.Code, revalidationResponse.Body.String())
	}

	preflight := httptest.NewRequest(http.MethodOptions, "/sdk/v1/config", nil)
	preflightResponse := httptest.NewRecorder()
	handler.configuration(preflightResponse, preflight)
	if preflightResponse.Code != http.StatusNoContent || !strings.Contains(preflightResponse.Header().Get("Access-Control-Allow-Headers"), "Authorization") {
		t.Fatalf("preflight = %d %#v", preflightResponse.Code, preflightResponse.Header())
	}
}

func TestSDKConfigurationHandlerRejectsInvalidCredential(t *testing.T) {
	service, err := coresdkconfig.NewService(&fakeSDKConfigRepository{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	handler := newSDKConfigHandlers(service)
	request := httptest.NewRequest(http.MethodGet, "/sdk/v1/config", nil)
	request.Header.Set("Authorization", "Bearer fs_client_invalid")
	response := httptest.NewRecorder()
	handler.configuration(response, request)
	if response.Code != http.StatusUnauthorized || response.Header().Get("WWW-Authenticate") == "" {
		t.Fatalf("response = %d %q", response.Code, response.Body.String())
	}
}

func TestSDKManagementRejectsDeveloper(t *testing.T) {
	service, err := coresdkconfig.NewService(&fakeSDKConfigRepository{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	handler := newSDKConfigHandlers(service)

	request := authenticatedSDKRequest(http.MethodPost, `{"name":"Backend","kind":"server"}`, "developer")
	response := httptest.NewRecorder()
	handler.createCredential(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("developer create response = %d %q", response.Code, response.Body.String())
	}

	visibilityRequest := authenticatedSDKRequest(http.MethodPut, `{"client_visible":true}`, "developer")
	visibilityResponse := httptest.NewRecorder()
	handler.setClientVisible(visibilityResponse, visibilityRequest)
	if visibilityResponse.Code != http.StatusForbidden {
		t.Fatalf("developer visibility response = %d %q", visibilityResponse.Code, visibilityResponse.Body.String())
	}
}

func authenticatedSDKRequest(method, body, role string) *http.Request {
	request := httptest.NewRequest(method, "/", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.SetPathValue("organisation", "example")
	request.SetPathValue("project", "project-1")
	request.SetPathValue("environment", "environment-1")
	request.SetPathValue("featureFlag", "flag-1")
	request.SetPathValue("credential", "credential-1")

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
