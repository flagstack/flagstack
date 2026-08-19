package config

import (
	"log/slog"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("FLAGSTACK_HTTP_ADDR", "")
	t.Setenv("FLAGSTACK_DATABASE_URL", "")
	t.Setenv("FLAGSTACK_LOG_LEVEL", "")
	t.Setenv("FLAGSTACK_SESSION_TTL", "")
	t.Setenv("FLAGSTACK_SESSION_COOKIE_SECURE", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTPAddr != defaultHTTPAddr {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, defaultHTTPAddr)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Fatalf("LogLevel = %v, want %v", cfg.LogLevel, slog.LevelInfo)
	}
	if cfg.SessionTTL != defaultSessionTTL {
		t.Fatalf("SessionTTL = %v, want %v", cfg.SessionTTL, defaultSessionTTL)
	}
	if cfg.SessionCookieSecure != defaultSessionCookieSecure {
		t.Fatalf("SessionCookieSecure = %v, want %v", cfg.SessionCookieSecure, defaultSessionCookieSecure)
	}
}

func TestLoadRejectsInvalidLogLevel(t *testing.T) {
	t.Setenv("FLAGSTACK_LOG_LEVEL", "loud")
	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}

func TestLoadRejectsInvalidSessionTTL(t *testing.T) {
	t.Setenv("FLAGSTACK_SESSION_TTL", "forever")
	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}

func TestLoadRejectsInvalidCookieSecureValue(t *testing.T) {
	t.Setenv("FLAGSTACK_SESSION_COOKIE_SECURE", "sometimes")
	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}
