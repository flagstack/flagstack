package httpapi

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeReadiness struct {
	err error
}

func (f fakeReadiness) Ping(context.Context) error {
	return f.err
}

func TestHealthEndpoint(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httptest.NewServer(NewRouter(logger, fakeReadiness{}, nil, AuthOptions{}))
	defer server.Close()

	response, err := http.Get(server.URL + "/api/v1/health")
	if err != nil {
		t.Fatalf("GET health endpoint: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	if strings.TrimSpace(string(body)) != `{"status":"ok"}` {
		t.Fatalf("body = %q", body)
	}
}

func TestReadinessEndpoint(t *testing.T) {
	tests := []struct {
		name       string
		readiness  readinessChecker
		wantStatus int
		wantBody   string
	}{
		{name: "ready", readiness: fakeReadiness{}, wantStatus: http.StatusOK, wantBody: `{"status":"ok"}`},
		{name: "database unavailable", readiness: fakeReadiness{err: errors.New("database unavailable")}, wantStatus: http.StatusServiceUnavailable, wantBody: `{"status":"unavailable"}`},
		{name: "missing checker", readiness: nil, wantStatus: http.StatusServiceUnavailable, wantBody: `{"status":"unavailable"}`},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
			response := httptest.NewRecorder()

			NewRouter(logger, tt.readiness, nil, AuthOptions{}).ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			if strings.TrimSpace(response.Body.String()) != tt.wantBody {
				t.Fatalf("body = %q, want %q", response.Body.String(), tt.wantBody)
			}
		})
	}
}
