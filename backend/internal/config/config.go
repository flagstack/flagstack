package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultHTTPAddr            = ":8080"
	defaultLogLevel            = "info"
	defaultSessionTTL          = 7 * 24 * time.Hour
	defaultSessionCookieSecure = true
)

type Config struct {
	HTTPAddr            string
	DatabaseURL         string
	LogLevel            slog.Level
	SessionTTL          time.Duration
	SessionCookieSecure bool
	StaticDir           string
}

func Load() (Config, error) {
	level, err := parseLogLevel(getenv("SWITCHONYOURCODE_LOG_LEVEL", defaultLogLevel))
	if err != nil {
		return Config{}, err
	}

	sessionTTL, err := parseDuration("SWITCHONYOURCODE_SESSION_TTL", getenv("SWITCHONYOURCODE_SESSION_TTL", defaultSessionTTL.String()))
	if err != nil {
		return Config{}, err
	}
	secureCookie, err := parseBool("SWITCHONYOURCODE_SESSION_COOKIE_SECURE", getenv("SWITCHONYOURCODE_SESSION_COOKIE_SECURE", strconv.FormatBool(defaultSessionCookieSecure)))
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTPAddr:            getenv("SWITCHONYOURCODE_HTTP_ADDR", defaultHTTPAddr),
		DatabaseURL:         os.Getenv("SWITCHONYOURCODE_DATABASE_URL"),
		LogLevel:            level,
		SessionTTL:          sessionTTL,
		SessionCookieSecure: secureCookie,
		StaticDir:           strings.TrimSpace(os.Getenv("SWITCHONYOURCODE_STATIC_DIR")),
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
		return 0, fmt.Errorf("invalid SWITCHONYOURCODE_LOG_LEVEL %q: %w", value, err)
	}
	return level, nil
}

func parseDuration(key, value string) (time.Duration, error) {
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("invalid %s %q", key, value)
	}
	return duration, nil
}

func parseBool(key, value string) (bool, error) {
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return false, fmt.Errorf("invalid %s %q: %w", key, value, err)
	}
	return parsed, nil
}
