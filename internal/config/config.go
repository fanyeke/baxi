package config

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

// Load reads configuration from environment variables and optional .env file.
func Load() (*Config, error) {
	return loadInternal(true)
}

// LoadLocal reads configuration for commands that only need local DB access
// (pipeline, e2e, governance). Does not require API_BEARER_TOKEN.
func LoadLocal() (*Config, error) {
	return loadInternal(false)
}

func loadInternal(requireAPIToken bool) (*Config, error) {
	// Try to load .env file if present
	_ = loadEnvFile(".env")

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

	if requireAPIToken && cfg.APIBearerToken == "" {
		return nil, errors.New("API_BEARER_TOKEN is required but not set")
	}

	return cfg, nil
}

// loadEnvFile loads environment variables from a .env file.
// Does not override existing environment variables.
func loadEnvFile(path string) error {
	// Try relative to current directory, then try to find project root
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Try to find .env in parent directories (up to 5 levels)
		for i := 0; i < 5; i++ {
			path = filepath.Join("..", path)
			if _, err := os.Stat(path); err == nil {
				break
			}
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil // .env not found, skip silently
		}
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		// Remove surrounding quotes if present
		value = strings.Trim(value, `"'`)
		// Only set if not already set in environment
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
	return scanner.Err()
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
