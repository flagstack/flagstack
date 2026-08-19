package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/flagstack/flagstack/backend/internal/config"
	"github.com/flagstack/flagstack/backend/internal/httpapi"
)

const shutdownTimeout = 10 * time.Second

func Run(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.NewRouter(logger),
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
