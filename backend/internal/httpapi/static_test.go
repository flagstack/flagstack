package httpapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStaticSPAHandlerServesAssetsRoutesAndLeavesControlPlaneAlone(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "assets"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("<html>Switch On Your Code</html>"), 0o644); err != nil {
		t.Fatalf("WriteFile(index) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "assets", "app.js"), []byte("console.log('switchonyourcode')"), 0o644); err != nil {
		t.Fatalf("WriteFile(asset) error = %v", err)
	}

	fallback := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	handler, err := NewStaticSPAHandler(root, fallback)
	if err != nil {
		t.Fatalf("NewStaticSPAHandler() error = %v", err)
	}

	t.Run("client route falls back to index", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/projects/example", nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Switch On Your Code") {
			t.Fatalf("response = %d %q", response.Code, response.Body.String())
		}
		if cache := response.Header().Get("Cache-Control"); cache != "no-cache" {
			t.Fatalf("Cache-Control = %q, want no-cache", cache)
		}
	})

	t.Run("hashed asset is immutable", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "switchonyourcode") {
			t.Fatalf("response = %d %q", response.Code, response.Body.String())
		}
		if cache := response.Header().Get("Cache-Control"); cache != "public, max-age=31536000, immutable" {
			t.Fatalf("Cache-Control = %q", cache)
		}
	})

	for _, requestPath := range []string{"/api/v1/missing", "/sdk/v1/missing", "/healthz", "/readyz"} {
		t.Run("control plane "+requestPath, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, requestPath, nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusTeapot {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusTeapot)
			}
		})
	}

	t.Run("non get request stays with control plane", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/projects/example", nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusTeapot {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusTeapot)
		}
	})
}

func TestStaticSPAHandlerRequiresIndex(t *testing.T) {
	_, err := NewStaticSPAHandler(t.TempDir(), http.NotFoundHandler())
	if err == nil {
		t.Fatal("NewStaticSPAHandler() error = nil, want missing index error")
	}
}
