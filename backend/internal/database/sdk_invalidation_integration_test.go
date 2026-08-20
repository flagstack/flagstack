package database

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	coresdkconfig "github.com/switchonyourcode/switchonyourcode/backend/internal/sdkconfig"
)

func TestSDKInvalidationTriggersIntegration(t *testing.T) {
	databaseURL := os.Getenv("SWITCHONYOURCODE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("SWITCHONYOURCODE_TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer pool.Close()
	client := NewEntClient(pool)
	defer client.Close()
	if err := Migrate(ctx, pool, client); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE organisations CASCADE`); err != nil {
		t.Fatalf("reset database: %v", err)
	}

	listener, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("pgx.Connect() error = %v", err)
	}
	defer func() { _ = listener.Close(context.Background()) }()
	if _, err := listener.Exec(ctx, "LISTEN "+sdkInvalidationChannel); err != nil {
		t.Fatalf("LISTEN error = %v", err)
	}

	var organisationID, projectID string
	if err := pool.QueryRow(ctx, `INSERT INTO organisations (name, slug) VALUES ('SDK events', 'sdk-events') RETURNING id::text`).Scan(&organisationID); err != nil {
		t.Fatalf("create organisation: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO projects (organisation_id, name, key) VALUES ($1, 'SDK events', 'sdk-events') RETURNING id::text`, organisationID).Scan(&projectID); err != nil {
		t.Fatalf("create project: %v", err)
	}

	var environmentID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO environments (organisation_id, project_id, name, key)
		VALUES ($1, $2, 'Production', 'production')
		RETURNING id::text
	`, organisationID, projectID).Scan(&environmentID); err != nil {
		t.Fatalf("create environment: %v", err)
	}

	notification, err := listener.WaitForNotification(ctx)
	if err != nil {
		t.Fatalf("WaitForNotification() error = %v", err)
	}
	if notification.Channel != sdkInvalidationChannel {
		t.Fatalf("channel = %q, want %q", notification.Channel, sdkInvalidationChannel)
	}
	var invalidation coresdkconfig.Invalidation
	if err := json.Unmarshal([]byte(notification.Payload), &invalidation); err != nil {
		t.Fatalf("decode notification: %v", err)
	}
	if invalidation.ProjectID != projectID || invalidation.EnvironmentID != environmentID || invalidation.CredentialID != "" {
		t.Fatalf("invalidation = %#v", invalidation)
	}

	var triggerCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM pg_trigger
		WHERE tgname LIKE 'switchonyourcode_sdk_invalidate_%' AND NOT tgisinternal
	`).Scan(&triggerCount); err != nil {
		t.Fatalf("count invalidation triggers: %v", err)
	}
	if triggerCount != 6 {
		t.Fatalf("SDK invalidation trigger count = %d, want 6", triggerCount)
	}
}
