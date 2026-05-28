package llm

import (
	"testing"

	"baxi/internal/config"
)

func TestProviderFactory_Disabled(t *testing.T) {
	cfg := &config.Config{
		LLMEnabled:  false,
		LLMProvider: "openai",
		LLMAPIKey:   "sk-test",
	}
	registry, err := NewPromptRegistry()
	if err != nil {
		t.Fatalf("failed to create prompt registry: %v", err)
	}
	factory := NewProviderFactory(cfg, registry)

	provider, err := factory.CreateProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := provider.(*RuleBasedProvider); !ok {
		t.Errorf("expected RuleBasedProvider, got %T", provider)
	}
}

func TestProviderFactory_DisabledProvider(t *testing.T) {
	cfg := &config.Config{
		LLMEnabled:  true,
		LLMProvider: "disabled",
		LLMAPIKey:   "sk-test",
	}
	registry, err := NewPromptRegistry()
	if err != nil {
		t.Fatalf("failed to create prompt registry: %v", err)
	}
	factory := NewProviderFactory(cfg, registry)

	provider, err := factory.CreateProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := provider.(*RuleBasedProvider); !ok {
		t.Errorf("expected RuleBasedProvider, got %T", provider)
	}
}

func TestProviderFactory_EmptyProvider(t *testing.T) {
	cfg := &config.Config{
		LLMEnabled:  true,
		LLMProvider: "",
		LLMAPIKey:   "sk-test",
	}
	registry, err := NewPromptRegistry()
	if err != nil {
		t.Fatalf("failed to create prompt registry: %v", err)
	}
	factory := NewProviderFactory(cfg, registry)

	provider, err := factory.CreateProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := provider.(*RuleBasedProvider); !ok {
		t.Errorf("expected RuleBasedProvider, got %T", provider)
	}
}

func TestProviderFactory_RuleBased(t *testing.T) {
	cfg := &config.Config{
		LLMEnabled:  true,
		LLMProvider: "rule_based",
		LLMAPIKey:   "sk-test",
	}
	registry, err := NewPromptRegistry()
	if err != nil {
		t.Fatalf("failed to create prompt registry: %v", err)
	}
	factory := NewProviderFactory(cfg, registry)

	provider, err := factory.CreateProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := provider.(*RuleBasedProvider); !ok {
		t.Errorf("expected RuleBasedProvider, got %T", provider)
	}
}

func TestProviderFactory_OpenAI(t *testing.T) {
	cfg := &config.Config{
		LLMEnabled:  true,
		LLMProvider: "openai",
		LLMAPIKey:   "sk-test",
		LLMModel:    "gpt-4o",
	}
	registry, err := NewPromptRegistry()
	if err != nil {
		t.Fatalf("failed to create prompt registry: %v", err)
	}
	factory := NewProviderFactory(cfg, registry)

	provider, err := factory.CreateProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := provider.(*OpenAICompatibleProvider); !ok {
		t.Errorf("expected OpenAICompatibleProvider, got %T", provider)
	}
}

func TestProviderFactory_OpenAICompatible(t *testing.T) {
	cfg := &config.Config{
		LLMEnabled:  true,
		LLMProvider: "openai_compatible",
		LLMAPIKey:   "sk-test",
		LLMModel:    "gpt-4o",
	}
	registry, err := NewPromptRegistry()
	if err != nil {
		t.Fatalf("failed to create prompt registry: %v", err)
	}
	factory := NewProviderFactory(cfg, registry)

	provider, err := factory.CreateProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := provider.(*OpenAICompatibleProvider); !ok {
		t.Errorf("expected OpenAICompatibleProvider, got %T", provider)
	}
}

func TestProviderFactory_UnknownProvider(t *testing.T) {
	cfg := &config.Config{
		LLMEnabled:  true,
		LLMProvider: "anthropic",
		LLMAPIKey:   "sk-test",
	}
	registry, err := NewPromptRegistry()
	if err != nil {
		t.Fatalf("failed to create prompt registry: %v", err)
	}
	factory := NewProviderFactory(cfg, registry)

	provider, err := factory.CreateProvider()
	if err == nil {
		t.Fatal("expected error for unknown provider, got nil")
	}
	if provider != nil {
		t.Errorf("expected nil provider, got %T", provider)
	}
	expectedErr := "unknown LLM provider: anthropic"
	if err.Error() != expectedErr {
		t.Errorf("expected error %q, got %q", expectedErr, err.Error())
	}
}
