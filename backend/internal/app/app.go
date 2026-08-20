package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/flagstack/flagstack/backend/internal/auth"
	"github.com/flagstack/flagstack/backend/internal/config"
	"github.com/flagstack/flagstack/backend/internal/database"
	"github.com/flagstack/flagstack/backend/internal/environment"
	"github.com/flagstack/flagstack/backend/internal/featureflag"
	"github.com/flagstack/flagstack/backend/internal/flagconfig"
	"github.com/flagstack/flagstack/backend/internal/httpapi"
	"github.com/flagstack/flagstack/backend/internal/project"
	"github.com/flagstack/flagstack/backend/internal/sdkconfig"
	"github.com/flagstack/flagstack/backend/internal/targeting"
)

const (
	shutdownTimeout   = 10 * time.Second
	schedulerInterval = 5 * time.Second
)

func Run(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer pool.Close()

	entClient := database.NewEntClient(pool)
	defer entClient.Close()

	authService, err := auth.NewService(database.NewAuthRepository(pool, entClient), cfg.SessionTTL)
	if err != nil {
		return fmt.Errorf("create auth service: %w", err)
	}
	projectService, err := project.NewService(database.NewProjectRepository(entClient))
	if err != nil {
		return fmt.Errorf("create project service: %w", err)
	}
	environmentService, err := environment.NewService(database.NewEnvironmentRepository(entClient))
	if err != nil {
		return fmt.Errorf("create environment service: %w", err)
	}
	featureFlagService, err := featureflag.NewService(database.NewFeatureFlagRepository(entClient))
	if err != nil {
		return fmt.Errorf("create feature flag service: %w", err)
	}
	flagConfigService, err := flagconfig.NewService(database.NewFlagConfigRepository(entClient))
	if err != nil {
		return fmt.Errorf("create flag configuration service: %w", err)
	}
	targetingService, err := targeting.NewService(database.NewTargetingRepository(entClient))
	if err != nil {
		return fmt.Errorf("create targeting service: %w", err)
	}
	sdkConfigService, err := sdkconfig.NewService(database.NewSDKConfigRepository(entClient))
	if err != nil {
		return fmt.Errorf("create SDK configuration service: %w", err)
	}
	sdkInvalidations := sdkconfig.NewInvalidationHub()

	go runScheduler(ctx, logger, targetingService)
	go database.RunSDKInvalidationListener(ctx, pool, sdkInvalidations, logger)

	var handler http.Handler = httpapi.NewRouterWithServices(logger, pool, httpapi.Services{
		Auth:             authService,
		Projects:         projectService,
		Environments:     environmentService,
		FeatureFlags:     featureFlagService,
		FlagConfigs:      flagConfigService,
		Targeting:        targetingService,
		SDKConfig:        sdkConfigService,
		SDKInvalidations: sdkInvalidations,
	}, httpapi.AuthOptions{SecureCookies: cfg.SessionCookieSecure})
	if cfg.StaticDir != "" {
		handler, err = httpapi.NewStaticSPAHandler(cfg.StaticDir, handler)
		if err != nil {
			return fmt.Errorf("configure frontend assets: %w", err)
		}
		logger.Info("serving frontend assets", "directory", cfg.StaticDir)
	}

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("http server listening", "address", cfg.HTTPAddr)
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	logger.Info("shutting down http server")
	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}

	return nil
}

func runScheduler(ctx context.Context, logger *slog.Logger, service *targeting.Service) {
	run := func() {
		completed, err := service.RunDueScheduledChanges(ctx, 50)
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.ErrorContext(ctx, "scheduled flag changes failed", "error", err)
			return
		}
		if completed > 0 {
			logger.InfoContext(ctx, "scheduled flag changes executed", "count", completed)
		}
	}

	run()
	ticker := time.NewTicker(schedulerInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}
