package config

import (
	"errors"
	"os"
)

// Config holds all configuration for the application.
type Config struct {
	DatabaseURL string
	APIPort     string
	LogLevel    string
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		APIPort:     getEnv("API_PORT", "8080"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
	}

	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required but not set")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
