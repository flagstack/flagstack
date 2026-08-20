package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	coresdkconfig "github.com/flagstack/flagstack/backend/internal/sdkconfig"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const sdkInvalidationChannel = "flagstack_sdk_invalidate"

func RunSDKInvalidationListener(ctx context.Context, pool *pgxpool.Pool, hub *coresdkconfig.InvalidationHub, logger *slog.Logger) {
	if pool == nil || hub == nil {
		return
	}
	if logger == nil {
		logger = slog.Default()
	}

	for {
		err := listenForSDKInvalidations(ctx, pool, hub, logger)
		if ctx.Err() != nil {
			return
		}
		logger.ErrorContext(ctx, "SDK invalidation listener disconnected", "error", err)
		timer := time.NewTimer(time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func listenForSDKInvalidations(ctx context.Context, pool *pgxpool.Pool, hub *coresdkconfig.InvalidationHub, logger *slog.Logger) error {
	connection, err := pgx.ConnectConfig(ctx, pool.Config().ConnConfig.Copy())
	if err != nil {
		return fmt.Errorf("connect SDK invalidation listener: %w", err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = connection.Close(closeCtx)
	}()

	if _, err := connection.Exec(ctx, "LISTEN "+sdkInvalidationChannel); err != nil {
		return fmt.Errorf("listen for SDK invalidations: %w", err)
	}
	logger.InfoContext(ctx, "SDK invalidation listener ready")

	for {
		notification, err := connection.WaitForNotification(ctx)
		if err != nil {
			return err
		}
		var invalidation coresdkconfig.Invalidation
		if err := json.Unmarshal([]byte(notification.Payload), &invalidation); err != nil {
			logger.WarnContext(ctx, "ignored malformed SDK invalidation", "error", err)
			continue
		}
		if invalidation.ProjectID == "" {
			logger.WarnContext(ctx, "ignored SDK invalidation without project scope")
			continue
		}
		hub.Publish(invalidation)
	}
}
