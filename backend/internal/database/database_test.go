package database

import (
	"context"
	"os"
	"testing"
	"time"

	coreauth "github.com/flagstack/flagstack/backend/internal/auth"
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

func TestAuthRepositoryIntegration(t *testing.T) {
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

	if _, err := pool.Exec(ctx, `TRUNCATE user_sessions, local_credentials, organisation_memberships, organisations, users CASCADE`); err != nil {
		t.Fatalf("reset authentication tables: %v", err)
	}

	repository := NewAuthRepository(pool)
	required, err := repository.BootstrapRequired(ctx)
	if err != nil {
		t.Fatalf("BootstrapRequired() error = %v", err)
	}
	if !required {
		t.Fatal("BootstrapRequired() = false, want true")
	}

	var tokenHash, csrfHash [32]byte
	for i := range tokenHash {
		tokenHash[i] = byte(i + 1)
		csrfHash[i] = byte(255 - i)
	}
	expiresAt := time.Now().UTC().Add(time.Hour)
	principal, err := repository.Bootstrap(ctx, coreauth.BootstrapRecord{
		Email:            "admin@example.com",
		DisplayName:      "Admin",
		PasswordHash:     "$test$hash",
		OrganisationName: "Example",
		OrganisationSlug: "example",
		SessionTokenHash: tokenHash,
		CSRFHash:         csrfHash,
		SessionExpiresAt: expiresAt,
	})
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	if principal.User.Email != "admin@example.com" || len(principal.Organisations) != 1 || principal.Organisations[0].Role != "owner" {
		t.Fatalf("Bootstrap() principal = %#v", principal)
	}

	required, err = repository.BootstrapRequired(ctx)
	if err != nil {
		t.Fatalf("BootstrapRequired() after bootstrap error = %v", err)
	}
	if required {
		t.Fatal("BootstrapRequired() after bootstrap = true, want false")
	}

	if _, err := repository.Bootstrap(ctx, coreauth.BootstrapRecord{}); err != coreauth.ErrBootstrapComplete {
		t.Fatalf("second Bootstrap() error = %v, want ErrBootstrapComplete", err)
	}

	session, err := repository.SessionByTokenHash(ctx, tokenHash)
	if err != nil {
		t.Fatalf("SessionByTokenHash() error = %v", err)
	}
	if session.Principal.User.ID != principal.User.ID || session.CSRFHash != csrfHash {
		t.Fatalf("SessionByTokenHash() session = %#v", session)
	}

	if err := repository.DeleteSession(ctx, tokenHash); err != nil {
		t.Fatalf("DeleteSession() error = %v", err)
	}
	if _, err := repository.SessionByTokenHash(ctx, tokenHash); err != coreauth.ErrSessionNotFound {
		t.Fatalf("SessionByTokenHash() after delete error = %v, want ErrSessionNotFound", err)
	}
}
