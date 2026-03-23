package config_test

import (
	"log/slog"
	"testing"

	"github.com/lobo235/cloudflare-gateway/internal/config"
)

// setRequired sets all required env vars; individual tests may blank one out.
func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("CF_API_TOKEN", "test-cf-token")
	t.Setenv("CF_ZONE_ID", "")
	t.Setenv("GATEWAY_API_KEY", "key123")
	t.Setenv("PORT", "")
	t.Setenv("LOG_LEVEL", "")
}

func TestLoad_Defaults(t *testing.T) {
	setRequired(t)
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", cfg.LogLevel)
	}
	if cfg.CFZoneID != "" {
		t.Errorf("CFZoneID = %q, want empty", cfg.CFZoneID)
	}
}

func TestLoad_AllSet(t *testing.T) {
	setRequired(t)
	t.Setenv("CF_ZONE_ID", "zone-abc")
	t.Setenv("PORT", "9090")
	t.Setenv("LOG_LEVEL", "debug")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.CFAPIToken != "test-cf-token" {
		t.Errorf("CFAPIToken = %q", cfg.CFAPIToken)
	}
	if cfg.CFZoneID != "zone-abc" {
		t.Errorf("CFZoneID = %q, want zone-abc", cfg.CFZoneID)
	}
	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want 9090", cfg.Port)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want debug", cfg.LogLevel)
	}
}

func TestLoad_MissingCFAPIToken(t *testing.T) {
	setRequired(t)
	t.Setenv("CF_API_TOKEN", "")
	if _, err := config.Load(); err == nil {
		t.Fatal("expected error for missing CF_API_TOKEN")
	}
}

func TestLoad_MissingGatewayAPIKey(t *testing.T) {
	setRequired(t)
	t.Setenv("GATEWAY_API_KEY", "")
	if _, err := config.Load(); err == nil {
		t.Fatal("expected error for missing GATEWAY_API_KEY")
	}
}

func TestLoad_LogLevels(t *testing.T) {
	for _, level := range []string{"debug", "info", "warn", "warning", "error"} {
		t.Run(level, func(t *testing.T) {
			setRequired(t)
			t.Setenv("LOG_LEVEL", level)
			if _, err := config.Load(); err != nil {
				t.Errorf("LOG_LEVEL=%q should be valid, got: %v", level, err)
			}
		})
	}
}

func TestLoad_InvalidLogLevel(t *testing.T) {
	setRequired(t)
	t.Setenv("LOG_LEVEL", "verbose")
	if _, err := config.Load(); err == nil {
		t.Fatal("expected error for invalid LOG_LEVEL")
	}
}

func TestSlogLevel(t *testing.T) {
	cases := []struct {
		level string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"INFO", slog.LevelInfo}, // case-insensitive
	}
	for _, tc := range cases {
		setRequired(t)
		t.Setenv("LOG_LEVEL", tc.level)
		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("LOG_LEVEL=%q: unexpected error: %v", tc.level, err)
		}
		if got := cfg.SlogLevel(); got != tc.want {
			t.Errorf("LOG_LEVEL=%q: SlogLevel() = %v, want %v", tc.level, got, tc.want)
		}
	}
}
