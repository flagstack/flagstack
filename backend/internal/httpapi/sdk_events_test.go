package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	coresdkconfig "github.com/flagstack/flagstack/backend/internal/sdkconfig"
)

type flushRecorder struct {
	*httptest.ResponseRecorder
	flushed chan struct{}
}

func newFlushRecorder() *flushRecorder {
	return &flushRecorder{ResponseRecorder: httptest.NewRecorder(), flushed: make(chan struct{}, 8)}
}

func (r *flushRecorder) Flush() {
	r.ResponseRecorder.Flush()
	select {
	case r.flushed <- struct{}{}:
	default:
	}
}

func TestSDKEventsStreamsScopedInvalidationsAndRevocation(t *testing.T) {
	repository := &fakeSDKConfigRepository{
		credential: coresdkconfig.Credential{
			ID: "credential-1", OrganisationID: "org-1", ProjectID: "project-1", EnvironmentID: "env-1",
			Name: "Browser", Kind: coresdkconfig.KindClient, ClientKey: "fs_client_test",
		},
	}
	service, err := coresdkconfig.NewService(repository)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	hub := coresdkconfig.NewInvalidationHub()
	handler := newSDKConfigHandlers(service, hub)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request := httptest.NewRequest(http.MethodGet, "/sdk/v1/events", nil).WithContext(ctx)
	request.Header.Set("Authorization", "Bearer fs_client_test")
	response := newFlushRecorder()
	done := make(chan struct{})
	go func() {
		defer close(done)
		handler.events(response, request)
	}()

	waitForFlush(t, response.flushed)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", response.Code, response.Body.String())
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "text/event-stream" {
		t.Fatalf("Content-Type = %q", contentType)
	}
	if body := response.Body.String(); !strings.Contains(body, "event: ready") || !strings.Contains(body, `"environment_id":"env-1"`) {
		t.Fatalf("ready event body = %q", body)
	}

	hub.Publish(coresdkconfig.Invalidation{ProjectID: "project-2"})
	select {
	case <-response.flushed:
		t.Fatal("unrelated project unexpectedly flushed an SDK event")
	case <-time.After(20 * time.Millisecond):
	}

	hub.Publish(coresdkconfig.Invalidation{ProjectID: "project-1", EnvironmentID: "env-1"})
	waitForFlush(t, response.flushed)
	if body := response.Body.String(); !strings.Contains(body, "event: configuration_changed") {
		t.Fatalf("configuration event body = %q", body)
	}

	hub.Publish(coresdkconfig.Invalidation{ProjectID: "project-1", EnvironmentID: "env-1", CredentialID: "credential-1"})
	waitForFlush(t, response.flushed)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("SDK stream did not close after credential revocation")
	}
	if body := response.Body.String(); !strings.Contains(body, "event: credential_revoked") {
		t.Fatalf("revocation event body = %q", body)
	}
}

func TestSDKEventsRejectsInvalidCredentialAndSupportsPreflight(t *testing.T) {
	service, err := coresdkconfig.NewService(&fakeSDKConfigRepository{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	handler := newSDKConfigHandlers(service)

	request := httptest.NewRequest(http.MethodGet, "/sdk/v1/events", nil)
	request.Header.Set("Authorization", "Bearer fs_client_invalid")
	response := httptest.NewRecorder()
	handler.events(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("invalid credential response = %d %q", response.Code, response.Body.String())
	}

	preflight := httptest.NewRequest(http.MethodOptions, "/sdk/v1/events", nil)
	preflightResponse := httptest.NewRecorder()
	handler.events(preflightResponse, preflight)
	if preflightResponse.Code != http.StatusNoContent || preflightResponse.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("preflight = %d %#v", preflightResponse.Code, preflightResponse.Header())
	}
}

func waitForFlush(t *testing.T, flushed <-chan struct{}) {
	t.Helper()
	select {
	case <-flushed:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for streamed SDK event")
	}
}
