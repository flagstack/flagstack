package httpapi

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func NewStaticSPAHandler(root string, next http.Handler) (http.Handler, error) {
	if next == nil {
		return nil, fmt.Errorf("fallback handler is required")
	}

	absoluteRoot, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil {
		return nil, fmt.Errorf("resolve static directory: %w", err)
	}
	indexPath := filepath.Join(absoluteRoot, "index.html")
	indexInfo, err := os.Stat(indexPath)
	if err != nil {
		return nil, fmt.Errorf("frontend index %q: %w", indexPath, err)
	}
	if indexInfo.IsDir() {
		return nil, fmt.Errorf("frontend index %q is a directory", indexPath)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if (r.Method != http.MethodGet && r.Method != http.MethodHead) || controlPlanePath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		relative := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if relative != "" && relative != "." {
			candidate := filepath.Join(absoluteRoot, filepath.FromSlash(relative))
			if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
				if strings.HasPrefix(relative, "assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				} else {
					w.Header().Set("Cache-Control", "no-cache")
				}
				http.ServeFile(w, r, candidate)
				return
			}
		}

		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, indexPath)
	}), nil
}

func controlPlanePath(requestPath string) bool {
	return requestPath == "/healthz" || requestPath == "/readyz" ||
		requestPath == "/api" || strings.HasPrefix(requestPath, "/api/") ||
		requestPath == "/sdk" || strings.HasPrefix(requestPath, "/sdk/")
}
