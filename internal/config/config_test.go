package config

import (
	"os"
	"testing"
)

func TestLLMConfigDefaults(t *testing.T) {
	// Unset all LLM env vars to test defaults
	for _, key := range llmEnvKeys() {
		os.Unsetenv(key)
	}

	t.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test?sslmode=disable")
	t.Setenv("API_BEARER_TOKEN", "test-token")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.LLMEnabled != false {
		t.Errorf("LLMEnabled default = %v, want false", cfg.LLMEnabled)
	}
	if cfg.LLMProvider != "disabled" {
		t.Errorf("LLMProvider default = %q, want %q", cfg.LLMProvider, "disabled")
	}
	if cfg.LLMAPIKey != "" {
		t.Errorf("LLMAPIKey default = %q, want %q", cfg.LLMAPIKey, "")
	}
	if cfg.LLMModel != "" {
		t.Errorf("LLMModel default = %q, want %q", cfg.LLMModel, "")
	}
	if cfg.LLMAPIBase != "https://api.openai.com/v1" {
		t.Errorf("LLMAPIBase default = %q, want %q", cfg.LLMAPIBase, "https://api.openai.com/v1")
	}
	if cfg.LLMTemperature != 0.2 {
		t.Errorf("LLMTemperature default = %f, want %f", cfg.LLMTemperature, 0.2)
	}
	if cfg.LLMMaxTokens != 2048 {
		t.Errorf("LLMMaxTokens default = %d, want %d", cfg.LLMMaxTokens, 2048)
	}
	if cfg.LLMTimeoutSeconds != 30 {
		t.Errorf("LLMTimeoutSeconds default = %d, want %d", cfg.LLMTimeoutSeconds, 30)
	}
	if cfg.LLMMaxRetries != 2 {
		t.Errorf("LLMMaxRetries default = %d, want %d", cfg.LLMMaxRetries, 2)
	}
	if cfg.LLMFallbackEnabled != true {
		t.Errorf("LLMFallbackEnabled default = %v, want true", cfg.LLMFallbackEnabled)
	}
	if cfg.LLMStoreRawOutput != true {
		t.Errorf("LLMStoreRawOutput default = %v, want true", cfg.LLMStoreRawOutput)
	}
}

func TestLLMConfigWithValues(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test?sslmode=disable")
	t.Setenv("API_BEARER_TOKEN", "test-token")

	t.Setenv("LLM_ENABLED", "true")
	t.Setenv("LLM_PROVIDER", "openai")
	t.Setenv("LLM_API_KEY", "sk-test123")
	t.Setenv("LLM_MODEL", "gpt-4")
	t.Setenv("LLM_API_BASE", "https://custom.api.com/v1")
	t.Setenv("LLM_TEMPERATURE", "0.7")
	t.Setenv("LLM_MAX_TOKENS", "4096")
	t.Setenv("LLM_TIMEOUT_SECONDS", "60")
	t.Setenv("LLM_MAX_RETRIES", "5")
	t.Setenv("LLM_FALLBACK_ENABLED", "false")
	t.Setenv("LLM_STORE_RAW_OUTPUT", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.LLMEnabled != true {
		t.Errorf("LLMEnabled = %v, want true", cfg.LLMEnabled)
	}
	if cfg.LLMProvider != "openai" {
		t.Errorf("LLMProvider = %q, want %q", cfg.LLMProvider, "openai")
	}
	if cfg.LLMAPIKey != "sk-test123" {
		t.Errorf("LLMAPIKey = %q, want %q", cfg.LLMAPIKey, "sk-test123")
	}
	if cfg.LLMModel != "gpt-4" {
		t.Errorf("LLMModel = %q, want %q", cfg.LLMModel, "gpt-4")
	}
	if cfg.LLMAPIBase != "https://custom.api.com/v1" {
		t.Errorf("LLMAPIBase = %q, want %q", cfg.LLMAPIBase, "https://custom.api.com/v1")
	}
	if cfg.LLMTemperature != 0.7 {
		t.Errorf("LLMTemperature = %f, want %f", cfg.LLMTemperature, 0.7)
	}
	if cfg.LLMMaxTokens != 4096 {
		t.Errorf("LLMMaxTokens = %d, want %d", cfg.LLMMaxTokens, 4096)
	}
	if cfg.LLMTimeoutSeconds != 60 {
		t.Errorf("LLMTimeoutSeconds = %d, want %d", cfg.LLMTimeoutSeconds, 60)
	}
	if cfg.LLMMaxRetries != 5 {
		t.Errorf("LLMMaxRetries = %d, want %d", cfg.LLMMaxRetries, 5)
	}
	if cfg.LLMFallbackEnabled != false {
		t.Errorf("LLMFallbackEnabled = %v, want false", cfg.LLMFallbackEnabled)
	}
	if cfg.LLMStoreRawOutput != false {
		t.Errorf("LLMStoreRawOutput = %v, want false", cfg.LLMStoreRawOutput)
	}
}

func TestLLMEnabledWithoutApiKey(t *testing.T) {
	for _, key := range llmEnvKeys() {
		os.Unsetenv(key)
	}

	t.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test?sslmode=disable")
	t.Setenv("API_BEARER_TOKEN", "test-token")
	t.Setenv("LLM_ENABLED", "true")

	_, err := Load()
	if err == nil {
		t.Fatalf("Load() should return error when LLM_ENABLED=true but LLM_API_KEY is not set")
	}
	if !containsString(err.Error(), "LLM_API_KEY") {
		t.Errorf("error message should mention LLM_API_KEY, got: %q", err.Error())
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// llmEnvKeys returns all LLM-related environment variable keys.
func llmEnvKeys() []string {
	return []string{
		"LLM_ENABLED",
		"LLM_PROVIDER",
		"LLM_API_KEY",
		"LLM_MODEL",
		"LLM_API_BASE",
		"LLM_TEMPERATURE",
		"LLM_MAX_TOKENS",
		"LLM_TIMEOUT_SECONDS",
		"LLM_MAX_RETRIES",
		"LLM_FALLBACK_ENABLED",
		"LLM_STORE_RAW_OUTPUT",
	}
}
