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
	"github.com/flagstack/flagstack/backend/internal/httpapi"
	"github.com/flagstack/flagstack/backend/internal/project"
)

const shutdownTimeout = 10 * time.Second

func Run(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer pool.Close()

	authService, err := auth.NewService(database.NewAuthRepository(pool), cfg.SessionTTL)
	if err != nil {
		return fmt.Errorf("create auth service: %w", err)
	}
	projectService, err := project.NewService(database.NewProjectRepository(pool))
	if err != nil {
		return fmt.Errorf("create project service: %w", err)
	}

	server := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: httpapi.NewRouterWithProjects(logger, pool, authService, projectService, httpapi.AuthOptions{
			SecureCookies: cfg.SessionCookieSecure,
		}),
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
