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

	// LLM / AI provider
	LLMAPIKey          string
	LLMAPIBase         string
	LLMModel           string
	LLMTemperature     float64
	LLMMaxTokens       int
	LLMTimeoutSeconds  int
	LLMEnabled         bool
	LLMProvider        string
	LLMFallbackEnabled bool
	LLMStoreRawOutput  bool
	LLMMaxRetries      int

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
	llmTemperature, _ := strconv.ParseFloat(getEnv("LLM_TEMPERATURE", "0.7"), 64)
	llmMaxTokens, _ := strconv.Atoi(getEnv("LLM_MAX_TOKENS", "1024"))
	llmTimeoutSeconds, _ := strconv.Atoi(getEnv("LLM_TIMEOUT_SECONDS", "60"))
	llmEnabled := false
	if v := getEnv("LLM_ENABLED", "false"); v == "true" {
		llmEnabled = true
	}
	llmFallbackEnabled := false
	if v := getEnv("LLM_FALLBACK_ENABLED", "false"); v == "true" {
		llmFallbackEnabled = true
	}
	llmStoreRawOutput := false
	if v := getEnv("LLM_STORE_RAW_OUTPUT", "false"); v == "true" {
		llmStoreRawOutput = true
	}
	llmMaxRetries, _ := strconv.Atoi(getEnv("LLM_MAX_RETRIES", "3"))

	cfg := &Config{
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		APIPort:            getEnv("API_PORT", "8080"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		APIBearerToken:     os.Getenv("API_BEARER_TOKEN"),
		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:3000"),

		// LLM / AI provider
		LLMAPIKey:          os.Getenv("LLM_API_KEY"),
		LLMAPIBase:         os.Getenv("LLM_API_BASE"),
		LLMModel:           getEnv("LLM_MODEL", "gpt-4o-mini"),
		LLMTemperature:     llmTemperature,
		LLMMaxTokens:       llmMaxTokens,
		LLMTimeoutSeconds:  llmTimeoutSeconds,
		LLMEnabled:         llmEnabled,
		LLMProvider:        getEnv("LLM_PROVIDER", "disabled"),
		LLMFallbackEnabled: llmFallbackEnabled,
		LLMStoreRawOutput:  llmStoreRawOutput,
		LLMMaxRetries:      llmMaxRetries,

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
