package httpapi

import (
	"context"
	"log/slog"
	"net/http"
)

type readinessChecker interface {
	Ping(context.Context) error
}

func NewRouter(logger *slog.Logger, readiness readinessChecker) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthHandler)
	mux.HandleFunc("GET /readyz", readinessHandler(readiness))
	mux.HandleFunc("GET /api/v1/health", healthHandler)

	return requestLogger(logger, mux)
}

func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.InfoContext(r.Context(), "http request",
			"method", r.Method,
			"path", r.URL.Path,
		)
		next.ServeHTTP(w, r)
	})
}
