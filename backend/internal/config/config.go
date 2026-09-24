package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                 int
	DatabasePath         string
	MaxPasteBytes        int64
	RateLimitPerMin      int
	DefaultExpirySeconds int
}

// getEnv returns the config value following the precedence:
// 1. .env file (via godotenv)
// 2. os.Getenv
// 3. empty string (triggers hardcoded default fallback)
func getEnv(key string, fileEnv map[string]string) string {
	if fileEnv != nil {
		if val, ok := fileEnv[key]; ok && strings.TrimSpace(val) != "" {
			return strings.TrimSpace(val)
		}
	}
	return strings.TrimSpace(os.Getenv(key))
}

// Load loads and validates configuration.
// Precedence: .env file (via joho/godotenv) -> os.Getenv -> hardcoded defaults.
func Load() (*Config, error) {
	// ponytail: read .env first; fallback to backend/.env if executed from project root
	fileEnv, err := godotenv.Read(".env")
	if err != nil {
		fileEnv, _ = godotenv.Read("backend/.env")
	}

	cfg := &Config{
		Port:                 8080,
		DatabasePath:         "./data/pastes.db",
		MaxPasteBytes:        524288, // 512KB
		RateLimitPerMin:      30,
		DefaultExpirySeconds: 0,
	}

	if val := getEnv("PORT", fileEnv); val != "" {
		p, err := strconv.Atoi(val)
		if err != nil || p < 1 || p > 65535 {
			return nil, fmt.Errorf("invalid PORT %q: must be between 1 and 65535", val)
		}
		cfg.Port = p
	}

	if val := getEnv("DATABASE_PATH", fileEnv); val != "" {
		cfg.DatabasePath = val
	}
	if cfg.DatabasePath == "" {
		return nil, errors.New("DATABASE_PATH cannot be empty")
	}

	if val := getEnv("MAX_PASTE_BYTES", fileEnv); val != "" {
		b, err := strconv.ParseInt(val, 10, 64)
		if err != nil || b <= 0 {
			return nil, fmt.Errorf("invalid MAX_PASTE_BYTES %q: must be positive integer", val)
		}
		cfg.MaxPasteBytes = b
	}

	if val := getEnv("RATE_LIMIT_PER_MIN", fileEnv); val != "" {
		r, err := strconv.Atoi(val)
		if err != nil || r <= 0 {
			return nil, fmt.Errorf("invalid RATE_LIMIT_PER_MIN %q: must be positive integer", val)
		}
		cfg.RateLimitPerMin = r
	}

	if val := getEnv("DEFAULT_EXPIRY_SECONDS", fileEnv); val != "" {
		e, err := strconv.Atoi(val)
		if err != nil || e < 0 {
			return nil, fmt.Errorf("invalid DEFAULT_EXPIRY_SECONDS %q: must be non-negative integer", val)
		}
		cfg.DefaultExpirySeconds = e
	}

	return cfg, nil
}

func (c *Config) Addr() string {
	return fmt.Sprintf(":%d", c.Port)
}
