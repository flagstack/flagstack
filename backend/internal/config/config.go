package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

const (
	defaultHTTPAddr = ":8080"
	defaultLogLevel = "info"
)

type Config struct {
	HTTPAddr    string
	DatabaseURL string
	LogLevel    slog.Level
}

func Load() (Config, error) {
	level, err := parseLogLevel(getenv("FLAGSTACK_LOG_LEVEL", defaultLogLevel))
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTPAddr:    getenv("FLAGSTACK_HTTP_ADDR", defaultHTTPAddr),
		DatabaseURL: os.Getenv("FLAGSTACK_DATABASE_URL"),
		LogLevel:    level,
	}, nil
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}

	return fallback
}

func parseLogLevel(value string) (slog.Level, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(strings.ToLower(strings.TrimSpace(value)))); err != nil {
		return 0, fmt.Errorf("invalid FLAGSTACK_LOG_LEVEL %q: %w", value, err)
	}

	return level, nil
}
