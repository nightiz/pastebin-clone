package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetEnv_Precedence(t *testing.T) {
	fileEnv := map[string]string{
		"PORT": "9000",
	}

	t.Setenv("PORT", "7000")
	t.Setenv("DATABASE_PATH", "./custom/pastes.db")

	// 1. .env takes precedence over os.Getenv
	if val := getEnv("PORT", fileEnv); val != "9000" {
		t.Errorf("expected PORT=9000 from fileEnv, got %q", val)
	}

	// 2. os.Getenv used when not in fileEnv
	if val := getEnv("DATABASE_PATH", fileEnv); val != "./custom/pastes.db" {
		t.Errorf("expected DATABASE_PATH from os.Getenv, got %q", val)
	}

	// 3. Fallback when neither has it
	if val := getEnv("NON_EXISTENT", fileEnv); val != "" {
		t.Errorf("expected empty string for non-existent key, got %q", val)
	}
}

func TestLoad_Defaults(t *testing.T) {
	// Isolate test from real .env
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error loading defaults: %v", err)
	}

	if cfg.Port != 8080 {
		t.Errorf("expected default Port 8080, got %d", cfg.Port)
	}
	if cfg.DatabasePath != "./data/pastes.db" {
		t.Errorf("expected default DatabasePath './data/pastes.db', got %q", cfg.DatabasePath)
	}
	if cfg.MaxPasteBytes != 524288 {
		t.Errorf("expected default MaxPasteBytes 524288, got %d", cfg.MaxPasteBytes)
	}
	if cfg.RateLimitPerMin != 30 {
		t.Errorf("expected default RateLimitPerMin 30, got %d", cfg.RateLimitPerMin)
	}
	if cfg.DefaultExpirySeconds != 0 {
		t.Errorf("expected default DefaultExpirySeconds 0, got %d", cfg.DefaultExpirySeconds)
	}
}

func TestLoad_DotEnvTakesPrecedence(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	envContent := "PORT=9999\nDATABASE_PATH=./test.db\n"
	if err := os.WriteFile(filepath.Join(tmpDir, ".env"), []byte(envContent), 0644); err != nil {
		t.Fatalf("failed to write test .env: %v", err)
	}

	t.Setenv("PORT", "5555") // OS env has 5555, but .env has 9999

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.Port != 9999 {
		t.Errorf("expected Port 9999 from .env, got %d", cfg.Port)
	}
	if cfg.DatabasePath != "./test.db" {
		t.Errorf("expected DatabasePath './test.db' from .env, got %q", cfg.DatabasePath)
	}
}
