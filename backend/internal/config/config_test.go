package config

import (
	"log/slog"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("FLAGSTACK_HTTP_ADDR", "")
	t.Setenv("FLAGSTACK_DATABASE_URL", "")
	t.Setenv("FLAGSTACK_LOG_LEVEL", "")

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
}

func TestLoadRejectsInvalidLogLevel(t *testing.T) {
	t.Setenv("FLAGSTACK_LOG_LEVEL", "loud")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}
