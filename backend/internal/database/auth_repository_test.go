package database

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/flagstack/flagstack/backend/internal/auth"
)

func TestAuthRepositoryIntegration(t *testing.T) {
	databaseURL := os.Getenv("FLAGSTACK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("FLAGSTACK_TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer pool.Close()

	client := NewEntClient(pool)
	defer client.Close()

	if _, err := pool.Exec(ctx, `TRUNCATE users, organisations CASCADE`); err != nil {
		t.Fatalf("reset authentication tables: %v", err)
	}

	repository := NewAuthRepository(pool, client)
	required, err := repository.BootstrapRequired(ctx)
	if err != nil {
		t.Fatalf("BootstrapRequired() error = %v", err)
	}
	if !required {
		t.Fatal("BootstrapRequired() = false, want true")
	}

	tokenHash := sha256.Sum256([]byte("bootstrap-session"))
	csrfHash := sha256.Sum256([]byte("bootstrap-csrf"))
	record := auth.BootstrapRecord{
		Email:            "owner@example.com",
		DisplayName:      "FlagStack Owner",
		PasswordHash:     "$argon2id$example",
		OrganisationName: "Example Organisation",
		OrganisationSlug: "example-organisation",
		SessionTokenHash: tokenHash,
		CSRFHash:         csrfHash,
		SessionExpiresAt: time.Now().Add(time.Hour),
	}

	principal, err := repository.Bootstrap(ctx, record)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	if principal.User.Email != record.Email || len(principal.Organisations) != 1 || principal.Organisations[0].Role != "owner" {
		t.Fatalf("Bootstrap() principal = %#v", principal)
	}

	required, err = repository.BootstrapRequired(ctx)
	if err != nil {
		t.Fatalf("BootstrapRequired() after bootstrap error = %v", err)
	}
	if required {
		t.Fatal("BootstrapRequired() after bootstrap = true, want false")
	}

	if _, err := repository.Bootstrap(ctx, record); !errors.Is(err, auth.ErrBootstrapComplete) {
		t.Fatalf("second Bootstrap() error = %v, want ErrBootstrapComplete", err)
	}

	credential, err := repository.CredentialByEmail(ctx, "OWNER@EXAMPLE.COM")
	if err != nil {
		t.Fatalf("CredentialByEmail() error = %v", err)
	}
	if credential.UserID != principal.User.ID || credential.PasswordHash != record.PasswordHash {
		t.Fatalf("CredentialByEmail() = %#v", credential)
	}

	bootstrapSession, err := repository.SessionByTokenHash(ctx, tokenHash)
	if err != nil {
		t.Fatalf("SessionByTokenHash() bootstrap error = %v", err)
	}
	if bootstrapSession.Principal.User.ID != principal.User.ID || bootstrapSession.CSRFHash != csrfHash {
		t.Fatalf("SessionByTokenHash() bootstrap session = %#v", bootstrapSession)
	}

	secondTokenHash := sha256.Sum256([]byte("second-session"))
	secondCSRFHash := sha256.Sum256([]byte("second-csrf"))
	secondPrincipal, err := repository.CreateSession(ctx, auth.SessionRecord{
		UserID:    principal.User.ID,
		TokenHash: secondTokenHash,
		CSRFHash:  secondCSRFHash,
		ExpiresAt: time.Now().Add(2 * time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if secondPrincipal.User.ID != principal.User.ID || len(secondPrincipal.Organisations) != 1 {
		t.Fatalf("CreateSession() principal = %#v", secondPrincipal)
	}

	if err := repository.DeleteSession(ctx, secondTokenHash); err != nil {
		t.Fatalf("DeleteSession() error = %v", err)
	}
	if _, err := repository.SessionByTokenHash(ctx, secondTokenHash); !errors.Is(err, auth.ErrSessionNotFound) {
		t.Fatalf("deleted SessionByTokenHash() error = %v, want ErrSessionNotFound", err)
	}
}
