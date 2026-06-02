package config

import (
	"os"
	"testing"
)

func TestLoad_MissingDATABASE_URL(t *testing.T) {
	// Ensure DATABASE_URL is not set
	os.Unsetenv("DATABASE_URL")
	t.Setenv("API_BEARER_TOKEN", "test-token")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when DATABASE_URL is not set")
	}
}

func TestLoad_MissingAPIBearerToken(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test?sslmode=disable")
	os.Unsetenv("API_BEARER_TOKEN")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when API_BEARER_TOKEN is not set")
	}
}

func TestLoad_BothMissing(t *testing.T) {
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("API_BEARER_TOKEN")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when both required vars are missing")
	}
}

func TestLoad_EnvVarOverrides(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test?sslmode=disable")
	t.Setenv("API_BEARER_TOKEN", "test-token")

	// Set custom env vars
	t.Setenv("API_PORT", "9090")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:8080")
	t.Setenv("LLM_MODEL", "gpt-4-turbo")
	t.Setenv("LLM_PROVIDER", "openai")
	t.Setenv("LLM_ENABLED", "true")
	t.Setenv("LLM_FALLBACK_ENABLED", "true")
	t.Setenv("LLM_STORE_RAW_OUTPUT", "true")
	t.Setenv("LLM_TEMPERATURE", "0.5")
	t.Setenv("LLM_MAX_TOKENS", "2048")
	t.Setenv("LLM_TIMEOUT_SECONDS", "30")
	t.Setenv("LLM_MAX_RETRIES", "5")
	t.Setenv("WORKER_BATCH_SIZE", "20")
	t.Setenv("WORKER_TICK_INTERVAL", "10s")
	t.Setenv("ACTION_APPLY_DRY_RUN", "false")
	t.Setenv("FEISHU_WEBHOOK_URL", "https://hooks.feishu.cn/test")
	t.Setenv("GITHUB_TOKEN", "ghp_test_token")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.APIPort != "9090" {
		t.Errorf("APIPort = %q, want %q", cfg.APIPort, "9090")
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
	if cfg.CORSAllowedOrigins != "http://localhost:8080" {
		t.Errorf("CORSAllowedOrigins = %q, want %q", cfg.CORSAllowedOrigins, "http://localhost:8080")
	}
	if cfg.LLMModel != "gpt-4-turbo" {
		t.Errorf("LLMModel = %q, want %q", cfg.LLMModel, "gpt-4-turbo")
	}
	if cfg.LLMProvider != "openai" {
		t.Errorf("LLMProvider = %q, want %q", cfg.LLMProvider, "openai")
	}
	if !cfg.LLMEnabled {
		t.Error("LLMEnabled = false, want true")
	}
	if !cfg.LLMFallbackEnabled {
		t.Error("LLMFallbackEnabled = false, want true")
	}
	if !cfg.LLMStoreRawOutput {
		t.Error("LLMStoreRawOutput = false, want true")
	}
	if cfg.LLMTemperature != 0.5 {
		t.Errorf("LLMTemperature = %f, want %f", cfg.LLMTemperature, 0.5)
	}
	if cfg.LLMMaxTokens != 2048 {
		t.Errorf("LLMMaxTokens = %d, want %d", cfg.LLMMaxTokens, 2048)
	}
	if cfg.LLMTimeoutSeconds != 30 {
		t.Errorf("LLMTimeoutSeconds = %d, want %d", cfg.LLMTimeoutSeconds, 30)
	}
	if cfg.LLMMaxRetries != 5 {
		t.Errorf("LLMMaxRetries = %d, want %d", cfg.LLMMaxRetries, 5)
	}
	if cfg.WorkerBatchSize != 20 {
		t.Errorf("WorkerBatchSize = %d, want %d", cfg.WorkerBatchSize, 20)
	}
	if cfg.WorkerTickInterval != "10s" {
		t.Errorf("WorkerTickInterval = %q, want %q", cfg.WorkerTickInterval, "10s")
	}
	if cfg.ActionApplyDryRun {
		t.Error("ActionApplyDryRun = true, want false")
	}
	if cfg.FeishuWebhookURL != "https://hooks.feishu.cn/test" {
		t.Errorf("FeishuWebhookURL = %q, want %q", cfg.FeishuWebhookURL, "https://hooks.feishu.cn/test")
	}
	if cfg.GitHubToken != "ghp_test_token" {
		t.Errorf("GitHubToken = %q, want %q", cfg.GitHubToken, "ghp_test_token")
	}
}

func TestGetEnv_DefaultValue(t *testing.T) {
	os.Unsetenv("TEST_NONEXISTENT_KEY")
	got := getEnv("TEST_NONEXISTENT_KEY", "my_default")
	if got != "my_default" {
		t.Errorf("getEnv() = %q, want %q", got, "my_default")
	}
}

func TestGetEnv_SetValue(t *testing.T) {
	t.Setenv("TEST_EXISTING_KEY", "my_value")
	got := getEnv("TEST_EXISTING_KEY", "default")
	if got != "my_value" {
		t.Errorf("getEnv() = %q, want %q", got, "my_value")
	}
}

func TestGetEnv_EmptyValue_ReturnsDefault(t *testing.T) {
	t.Setenv("TEST_EMPTY_KEY", "")
	got := getEnv("TEST_EMPTY_KEY", "default")
	if got != "default" {
		t.Errorf("getEnv() = %q, want %q", got, "default")
	}
}
