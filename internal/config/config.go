package config

import (
	"errors"
	"os"
)

// Config holds all configuration for the application.
type Config struct {
	DatabaseURL        string
	APIPort            string
	LogLevel           string
	APIBearerToken     string
	CORSAllowedOrigins string
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		APIPort:            getEnv("API_PORT", "8080"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		APIBearerToken:     os.Getenv("API_BEARER_TOKEN"),
		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:3000"),
	}

	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required but not set")
	}

	if cfg.APIBearerToken == "" {
		return nil, errors.New("API_BEARER_TOKEN is required but not set")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
