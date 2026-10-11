package config

import (
	"strings"
	"testing"
)

// TEST-001: env lengkap → config terisi dengan benar
func TestLoadEnvLengkap(t *testing.T) {
	t.Setenv("API_KEY", "test-key-123")
	t.Setenv("APP_PORT", "8080")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("REDIS_PORT", "6381")
	t.Setenv("SHORT_CODE_LENGTH", "8")
	t.Setenv("CACHE_TTL_SECONDS", "120")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.APIKey != "test-key-123" {
		t.Errorf("APIKey = %q, want test-key-123", cfg.APIKey)
	}
	if cfg.AppPort != "8080" {
		t.Errorf("AppPort = %q, want 8080", cfg.AppPort)
	}
	if cfg.DBPort != "5433" || cfg.RedisPort != "6381" {
		t.Errorf("port non-default hilang: DB=%s Redis=%s", cfg.DBPort, cfg.RedisPort)
	}
	if cfg.ShortCodeLength != 8 || cfg.CacheTTLSeconds != 120 {
		t.Errorf("ShortCodeLength=%d CacheTTL=%d", cfg.ShortCodeLength, cfg.CacheTTLSeconds)
	}
	if !strings.Contains(cfg.DBDSN(), "port=5433") {
		t.Errorf("DBDSN tidak memakai port non-default: %s", cfg.DBDSN())
	}
}

// TEST-002: API_KEY kosong → error
func TestLoadTanpaAPIKey(t *testing.T) {
	t.Setenv("API_KEY", "")

	cfg, err := Load()
	if err == nil {
		t.Fatal("expected error when API_KEY is empty, got valid config")
	}
	if cfg != nil {
		t.Error("config must be nil on error")
	}
	if !strings.Contains(err.Error(), "API_KEY") {
		t.Errorf("pesan error tidak menyebut API_KEY: %v", err)
	}
}

// TEST-002b: env numerik tidak valid → error, bukan panic/zero-value diam-diam
func TestLoadEnvNumerikInvalid(t *testing.T) {
	t.Setenv("API_KEY", "k")
	t.Setenv("CACHE_TTL_SECONDS", "abc")

	if _, err := Load(); err == nil {
		t.Fatal("expected error saat CACHE_TTL_SECONDS bukan angka")
	}
}
