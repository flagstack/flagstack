package database

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestOpenRejectsEmptyDatabaseURL(t *testing.T) {
	if _, err := Open(context.Background(), ""); err == nil {
		t.Fatal("Open() error = nil, want error")
	}
}

func TestOpenIntegration(t *testing.T) {
	databaseURL := os.Getenv("FLAGSTACK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("FLAGSTACK_TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer pool.Close()

	var value int
	if err := pool.QueryRow(ctx, "select 1").Scan(&value); err != nil {
		t.Fatalf("query database: %v", err)
	}
	if value != 1 {
		t.Fatalf("value = %d, want 1", value)
	}
}
