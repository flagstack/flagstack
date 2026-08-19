package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	coreauth "github.com/flagstack/flagstack/backend/internal/auth"
)

type readinessChecker interface {
	Ping(context.Context) error
}

func NewRouter(logger *slog.Logger, readiness readinessChecker, authService *coreauth.Service, authOptions AuthOptions) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthHandler)
	mux.HandleFunc("GET /readyz", readinessHandler(readiness))
	mux.HandleFunc("GET /api/v1/health", healthHandler)

	if authService != nil {
		authHandlers := newAuthHandlers(logger, authService, authOptions)
		mux.HandleFunc("GET /api/v1/bootstrap", authHandlers.bootstrapStatus)
		mux.HandleFunc("POST /api/v1/bootstrap", authHandlers.bootstrap)
		mux.HandleFunc("POST /api/v1/auth/login", authHandlers.login)
		mux.Handle("GET /api/v1/auth/me", authHandlers.requireAuth(http.HandlerFunc(authHandlers.me)))
		mux.Handle("POST /api/v1/auth/logout", authHandlers.requireAuth(authHandlers.requireCSRF(http.HandlerFunc(authHandlers.logout))))
	}

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
