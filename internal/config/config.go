package config

import (
	"errors"
	"os"
	"strconv"
)

// Config holds all configuration for the application.
type Config struct {
	DatabaseURL        string
	APIPort            string
	LogLevel           string
	APIBearerToken     string
	CORSAllowedOrigins string

	// Phase 7: Review / Action / Outbox
	ActionApplyDryRun  bool
	WorkerTickInterval string
	WorkerBatchSize    int
	FeishuWebhookURL   string
	GitHubToken        string
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	workerBatchSize, _ := strconv.Atoi(getEnv("WORKER_BATCH_SIZE", "10"))
	actionApplyDryRun := true
	if v := getEnv("ACTION_APPLY_DRY_RUN", "true"); v == "false" {
		actionApplyDryRun = false
	}

	cfg := &Config{
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		APIPort:            getEnv("API_PORT", "8080"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		APIBearerToken:     os.Getenv("API_BEARER_TOKEN"),
		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:3000"),

		// Phase 7: Review / Action / Outbox
		ActionApplyDryRun:  actionApplyDryRun,
		WorkerTickInterval: getEnv("WORKER_TICK_INTERVAL", "30s"),
		WorkerBatchSize:    workerBatchSize,
		FeishuWebhookURL:   os.Getenv("FEISHU_WEBHOOK_URL"),
		GitHubToken:        os.Getenv("GITHUB_TOKEN"),
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
