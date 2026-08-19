package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	coreauth "github.com/flagstack/flagstack/backend/internal/auth"
	coreenvironment "github.com/flagstack/flagstack/backend/internal/environment"
	coreproject "github.com/flagstack/flagstack/backend/internal/project"
)

type readinessChecker interface {
	Ping(context.Context) error
}

type Services struct {
	Auth         *coreauth.Service
	Projects     *coreproject.Service
	Environments *coreenvironment.Service
}

func NewRouter(logger *slog.Logger, readiness readinessChecker, authService *coreauth.Service, authOptions AuthOptions) http.Handler {
	return NewRouterWithServices(logger, readiness, Services{Auth: authService}, authOptions)
}

func NewRouterWithServices(logger *slog.Logger, readiness readinessChecker, services Services, authOptions AuthOptions) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthHandler)
	mux.HandleFunc("GET /readyz", readinessHandler(readiness))
	mux.HandleFunc("GET /api/v1/health", healthHandler)

	if services.Auth != nil {
		authHandlers := newAuthHandlers(logger, services.Auth, authOptions)
		mux.HandleFunc("GET /api/v1/bootstrap", authHandlers.bootstrapStatus)
		mux.HandleFunc("POST /api/v1/bootstrap", authHandlers.bootstrap)
		mux.HandleFunc("POST /api/v1/auth/login", authHandlers.login)
		mux.Handle("GET /api/v1/auth/me", authHandlers.requireAuth(http.HandlerFunc(authHandlers.me)))
		mux.Handle("POST /api/v1/auth/logout", authHandlers.requireAuth(authHandlers.requireCSRF(http.HandlerFunc(authHandlers.logout))))

		if services.Projects != nil {
			projectHandlers := newProjectHandlers(services.Projects)
			projectsPath := "/api/v1/organisations/{organisation}/projects"
			mux.Handle("GET "+projectsPath, authHandlers.requireAuth(http.HandlerFunc(projectHandlers.list)))
			mux.Handle("POST "+projectsPath, authHandlers.requireAuth(authHandlers.requireCSRF(http.HandlerFunc(projectHandlers.create))))
		}

		if services.Environments != nil {
			environmentHandlers := newEnvironmentHandlers(services.Environments)
			environmentsPath := "/api/v1/organisations/{organisation}/projects/{project}/environments"
			mux.Handle("GET "+environmentsPath, authHandlers.requireAuth(http.HandlerFunc(environmentHandlers.list)))
			mux.Handle("POST "+environmentsPath, authHandlers.requireAuth(authHandlers.requireCSRF(http.HandlerFunc(environmentHandlers.create))))
		}
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
