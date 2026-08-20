package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	coreauth "github.com/switchonyourcode/switchonyourcode/backend/internal/auth"
)

type fakeAuthRepository struct {
	bootstrapRequired bool
	principal         coreauth.Principal
	credential        coreauth.Credential
	sessions          map[[32]byte]coreauth.Session
}

func (f *fakeAuthRepository) BootstrapRequired(context.Context) (bool, error) {
	return f.bootstrapRequired, nil
}

func (f *fakeAuthRepository) Bootstrap(_ context.Context, record coreauth.BootstrapRecord) (coreauth.Principal, error) {
	if !f.bootstrapRequired {
		return coreauth.Principal{}, coreauth.ErrBootstrapComplete
	}
	f.bootstrapRequired = false
	f.sessions[record.SessionTokenHash] = coreauth.Session{
		Principal: f.principal,
		CSRFHash:  record.CSRFHash,
		ExpiresAt: record.SessionExpiresAt,
	}
	return f.principal, nil
}

func (f *fakeAuthRepository) CredentialByEmail(context.Context, string) (coreauth.Credential, error) {
	if f.credential.UserID == "" {
		return coreauth.Credential{}, coreauth.ErrCredentialNotFound
	}
	return f.credential, nil
}

func (f *fakeAuthRepository) CreateSession(_ context.Context, record coreauth.SessionRecord) (coreauth.Principal, error) {
	f.sessions[record.TokenHash] = coreauth.Session{
		Principal: f.principal,
		CSRFHash:  record.CSRFHash,
		ExpiresAt: record.ExpiresAt,
	}
	return f.principal, nil
}

func (f *fakeAuthRepository) SessionByTokenHash(_ context.Context, tokenHash [32]byte) (coreauth.Session, error) {
	session, ok := f.sessions[tokenHash]
	if !ok || !session.ExpiresAt.After(time.Now()) {
		return coreauth.Session{}, coreauth.ErrSessionNotFound
	}
	return session, nil
}

func (f *fakeAuthRepository) DeleteSession(_ context.Context, tokenHash [32]byte) error {
	delete(f.sessions, tokenHash)
	return nil
}

func TestBootstrapSessionAndLogoutFlow(t *testing.T) {
	repository := &fakeAuthRepository{
		bootstrapRequired: true,
		principal: coreauth.Principal{
			User:          coreauth.User{ID: "user-1", Email: "admin@example.com", DisplayName: "Admin"},
			Organisations: []coreauth.OrganisationMembership{{ID: "org-1", Name: "Example", Slug: "example", Role: "owner"}},
		},
		sessions: make(map[[32]byte]coreauth.Session),
	}
	service, err := coreauth.NewService(repository, time.Hour)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewRouter(logger, fakeReadiness{}, service, AuthOptions{})

	statusRequest := httptest.NewRequest(http.MethodGet, "/api/v1/bootstrap", nil)
	statusResponse := httptest.NewRecorder()
	handler.ServeHTTP(statusResponse, statusRequest)
	if statusResponse.Code != http.StatusOK || !strings.Contains(statusResponse.Body.String(), `"required":true`) {
		t.Fatalf("bootstrap status = %d %q", statusResponse.Code, statusResponse.Body.String())
	}

	bootstrapRequest := httptest.NewRequest(http.MethodPost, "/api/v1/bootstrap", strings.NewReader(`{
		"email":"admin@example.com",
		"display_name":"Admin",
		"password":"correct horse battery staple",
		"organisation_name":"Example",
		"organisation_slug":"example"
	}`))
	bootstrapRequest.Header.Set("Content-Type", "application/json")
	bootstrapResponse := httptest.NewRecorder()
	handler.ServeHTTP(bootstrapResponse, bootstrapRequest)
	if bootstrapResponse.Code != http.StatusCreated {
		t.Fatalf("bootstrap status = %d, body = %q", bootstrapResponse.Code, bootstrapResponse.Body.String())
	}

	var sessionCookie, csrfCookie *http.Cookie
	for _, cookie := range bootstrapResponse.Result().Cookies() {
		switch cookie.Name {
		case sessionCookieName:
			sessionCookie = cookie
		case csrfCookieName:
			csrfCookie = cookie
		}
	}
	if sessionCookie == nil || csrfCookie == nil {
		t.Fatal("bootstrap response did not issue both authentication cookies")
	}
	if !sessionCookie.HttpOnly {
		t.Fatal("session cookie must be HttpOnly")
	}
	if csrfCookie.HttpOnly {
		t.Fatal("CSRF cookie must be readable by the frontend")
	}

	meRequest := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	meRequest.AddCookie(sessionCookie)
	meResponse := httptest.NewRecorder()
	handler.ServeHTTP(meResponse, meRequest)
	if meResponse.Code != http.StatusOK || !strings.Contains(meResponse.Body.String(), `"role":"owner"`) {
		t.Fatalf("me response = %d %q", meResponse.Code, meResponse.Body.String())
	}

	logoutWithoutCSRF := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutWithoutCSRF.AddCookie(sessionCookie)
	logoutWithoutCSRFResponse := httptest.NewRecorder()
	handler.ServeHTTP(logoutWithoutCSRFResponse, logoutWithoutCSRF)
	if logoutWithoutCSRFResponse.Code != http.StatusForbidden {
		t.Fatalf("logout without CSRF status = %d, want %d", logoutWithoutCSRFResponse.Code, http.StatusForbidden)
	}

	logoutRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutRequest.AddCookie(sessionCookie)
	logoutRequest.AddCookie(csrfCookie)
	logoutRequest.Header.Set("X-CSRF-Token", csrfCookie.Value)
	logoutResponse := httptest.NewRecorder()
	handler.ServeHTTP(logoutResponse, logoutRequest)
	if logoutResponse.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d, body = %q", logoutResponse.Code, logoutResponse.Body.String())
	}

	meAfterLogout := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	meAfterLogout.AddCookie(sessionCookie)
	meAfterLogoutResponse := httptest.NewRecorder()
	handler.ServeHTTP(meAfterLogoutResponse, meAfterLogout)
	if meAfterLogoutResponse.Code != http.StatusUnauthorized {
		t.Fatalf("me after logout status = %d, want %d", meAfterLogoutResponse.Code, http.StatusUnauthorized)
	}
}

func TestLoginDoesNotRevealUnknownAccounts(t *testing.T) {
	repository := &fakeAuthRepository{sessions: make(map[[32]byte]coreauth.Session)}
	service, err := coreauth.NewService(repository, time.Hour)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewRouter(logger, fakeReadiness{}, service, AuthOptions{})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"unknown@example.com","password":"not-the-right-password"}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("login status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(response.Body.String(), `"code":"invalid_credentials"`) {
		t.Fatalf("login body = %q", response.Body.String())
	}
}
