package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port                 int
	DatabasePath         string
	MaxPasteBytes        int64
	RateLimitPerMin      int
	DefaultExpirySeconds int
}

// Load loads and validates configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		Port:                 8080,
		DatabasePath:         "./data/pastes.db",
		MaxPasteBytes:        524288, // 512KB
		RateLimitPerMin:      30,
		DefaultExpirySeconds: 0,
	}

	if val := os.Getenv("PORT"); val != "" {
		p, err := strconv.Atoi(val)
		if err != nil || p < 1 || p > 65535 {
			return nil, fmt.Errorf("invalid PORT %q: must be between 1 and 65535", val)
		}
		cfg.Port = p
	}

	if val := os.Getenv("DATABASE_PATH"); val != "" {
		cfg.DatabasePath = val
	}
	if cfg.DatabasePath == "" {
		return nil, errors.New("DATABASE_PATH cannot be empty")
	}

	if val := os.Getenv("MAX_PASTE_BYTES"); val != "" {
		b, err := strconv.ParseInt(val, 10, 64)
		if err != nil || b <= 0 {
			return nil, fmt.Errorf("invalid MAX_PASTE_BYTES %q: must be positive integer", val)
		}
		cfg.MaxPasteBytes = b
	}

	if val := os.Getenv("RATE_LIMIT_PER_MIN"); val != "" {
		r, err := strconv.Atoi(val)
		if err != nil || r <= 0 {
			return nil, fmt.Errorf("invalid RATE_LIMIT_PER_MIN %q: must be positive integer", val)
		}
		cfg.RateLimitPerMin = r
	}

	if val := os.Getenv("DEFAULT_EXPIRY_SECONDS"); val != "" {
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
