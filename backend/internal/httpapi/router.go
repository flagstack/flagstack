package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	coreauth "github.com/flagstack/flagstack/backend/internal/auth"
	coreenvironment "github.com/flagstack/flagstack/backend/internal/environment"
	corefeatureflag "github.com/flagstack/flagstack/backend/internal/featureflag"
	coreflagconfig "github.com/flagstack/flagstack/backend/internal/flagconfig"
	coreproject "github.com/flagstack/flagstack/backend/internal/project"
	coresdkconfig "github.com/flagstack/flagstack/backend/internal/sdkconfig"
	coretargeting "github.com/flagstack/flagstack/backend/internal/targeting"
)

type readinessChecker interface {
	Ping(context.Context) error
}

type Services struct {
	Auth         *coreauth.Service
	Projects     *coreproject.Service
	Environments *coreenvironment.Service
	FeatureFlags *corefeatureflag.Service
	FlagConfigs  *coreflagconfig.Service
	Targeting    *coretargeting.Service
	SDKConfig    *coresdkconfig.Service
}

func NewRouter(logger *slog.Logger, readiness readinessChecker, authService *coreauth.Service, authOptions AuthOptions) http.Handler {
	return NewRouterWithServices(logger, readiness, Services{Auth: authService}, authOptions)
}

func NewRouterWithServices(logger *slog.Logger, readiness readinessChecker, services Services, authOptions AuthOptions) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthHandler)
	mux.HandleFunc("GET /readyz", readinessHandler(readiness))
	mux.HandleFunc("GET /api/v1/health", healthHandler)

	var sdkHandlers *sdkConfigHandlers
	if services.SDKConfig != nil {
		sdkHandlers = newSDKConfigHandlers(services.SDKConfig)
		mux.Handle("GET /sdk/v1/config", http.HandlerFunc(sdkHandlers.configuration))
	}

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

		if services.FeatureFlags != nil {
			featureFlagHandlers := newFeatureFlagHandlers(services.FeatureFlags)
			featureFlagsPath := "/api/v1/organisations/{organisation}/projects/{project}/feature-flags"
			mux.Handle("GET "+featureFlagsPath, authHandlers.requireAuth(http.HandlerFunc(featureFlagHandlers.list)))
			mux.Handle("POST "+featureFlagsPath, authHandlers.requireAuth(authHandlers.requireCSRF(http.HandlerFunc(featureFlagHandlers.create))))
		}

		if services.FlagConfigs != nil {
			flagConfigHandlers := newFlagConfigHandlers(services.FlagConfigs)
			flagConfigsPath := "/api/v1/organisations/{organisation}/projects/{project}/flag-configs"
			flagConfigPath := "/api/v1/organisations/{organisation}/projects/{project}/environments/{environment}/feature-flags/{featureFlag}"
			mux.Handle("GET "+flagConfigsPath, authHandlers.requireAuth(http.HandlerFunc(flagConfigHandlers.list)))
			mux.Handle("PUT "+flagConfigPath, authHandlers.requireAuth(authHandlers.requireCSRF(http.HandlerFunc(flagConfigHandlers.setEnabled))))
		}

		if services.Targeting != nil {
			targetingHandlers := newTargetingHandlers(services.Targeting)
			projectBase := "/api/v1/organisations/{organisation}/projects/{project}"
			flagBase := projectBase + "/feature-flags/{featureFlag}"
			environmentFlagBase := projectBase + "/environments/{environment}/feature-flags/{featureFlag}"
			segmentsPath := projectBase + "/segments"
			schedulesPath := projectBase + "/scheduled-flag-changes"

			mux.Handle("GET "+flagBase+"/targeting", authHandlers.requireAuth(http.HandlerFunc(targetingHandlers.getFlagTargeting)))
			mux.Handle("PUT "+flagBase+"/variants", authHandlers.requireAuth(authHandlers.requireCSRF(http.HandlerFunc(targetingHandlers.setVariants))))
			mux.Handle("PUT "+environmentFlagBase+"/policy", authHandlers.requireAuth(authHandlers.requireCSRF(http.HandlerFunc(targetingHandlers.setPolicy))))
			mux.Handle("POST "+environmentFlagBase+"/preview", authHandlers.requireAuth(http.HandlerFunc(targetingHandlers.preview)))
			mux.Handle("GET "+segmentsPath, authHandlers.requireAuth(http.HandlerFunc(targetingHandlers.listSegments)))
			mux.Handle("POST "+segmentsPath, authHandlers.requireAuth(authHandlers.requireCSRF(http.HandlerFunc(targetingHandlers.createSegment))))
			mux.Handle("PUT "+segmentsPath+"/{segment}", authHandlers.requireAuth(authHandlers.requireCSRF(http.HandlerFunc(targetingHandlers.updateSegment))))
			mux.Handle("GET "+schedulesPath, authHandlers.requireAuth(http.HandlerFunc(targetingHandlers.listScheduledChanges)))
			mux.Handle("POST "+schedulesPath, authHandlers.requireAuth(authHandlers.requireCSRF(http.HandlerFunc(targetingHandlers.createScheduledChange))))
			mux.Handle("POST "+schedulesPath+"/{schedule}/cancel", authHandlers.requireAuth(authHandlers.requireCSRF(http.HandlerFunc(targetingHandlers.cancelScheduledChange))))
		}

		if sdkHandlers != nil {
			projectBase := "/api/v1/organisations/{organisation}/projects/{project}"
			credentialsPath := projectBase + "/sdk-credentials"
			createCredentialPath := projectBase + "/environments/{environment}/sdk-credentials"
			credentialPath := credentialsPath + "/{credential}"
			clientVisibilityPath := projectBase + "/feature-flags/{featureFlag}/client-visibility"

			mux.Handle("GET "+credentialsPath, authHandlers.requireAuth(http.HandlerFunc(sdkHandlers.listCredentials)))
			mux.Handle("POST "+createCredentialPath, authHandlers.requireAuth(authHandlers.requireCSRF(http.HandlerFunc(sdkHandlers.createCredential))))
			mux.Handle("POST "+credentialPath+"/revoke", authHandlers.requireAuth(authHandlers.requireCSRF(http.HandlerFunc(sdkHandlers.revokeCredential))))
			mux.Handle("PUT "+clientVisibilityPath, authHandlers.requireAuth(authHandlers.requireCSRF(http.HandlerFunc(sdkHandlers.setClientVisible))))
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
