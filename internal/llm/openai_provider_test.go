package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"baxi/internal/config"
)

func TestOpenAIProvider_SatisfiesInterface(t *testing.T) {
	var _ DecisionProvider = (*OpenAICompatibleProvider)(nil)
}

func validDecisionJSON() string {
	return `{"decision_type":"investigate","severity":"medium","summary":"test decision","rationale":["reason 1"],"recommended_actions":[{"action_type":"notify_owner","priority":"medium","owner_role":"analyst","payload":{}}],"confidence":0.72,"requires_human_review":true}`
}

func newMockOpenAIServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, fmt.Sprintf("expected POST, got %s", r.Method), http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/chat/completions" {
			http.Error(w, fmt.Sprintf("expected /chat/completions, got %s", r.URL.Path), http.StatusNotFound)
			return
		}
		if r.Header.Get("Authorization") == "" {
			http.Error(w, "missing Authorization header", http.StatusUnauthorized)
			return
		}
		handler(w, r)
	}))
}

func newTestOpenAIProvider(t *testing.T, mockServerURL string) *OpenAICompatibleProvider {
	t.Helper()
	cfg := &config.Config{
		LLMAPIKey:         "sk-test",
		LLMModel:          "gpt-4o-mini",
		LLMAPIBase:        mockServerURL,
		LLMTemperature:    0.2,
		LLMMaxTokens:      2048,
		LLMTimeoutSeconds: 30,
	}
	registry, err := NewPromptRegistry()
	if err != nil {
		t.Fatalf("failed to create prompt registry: %v", err)
	}
	provider, err := NewOpenAIProvider(cfg, registry)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	return provider
}

func testInput() LLMSafeContext {
	return LLMSafeContext{
		CaseID: "case-001",
		Trigger: TriggerInfo{
			AlertID:      "alert-001",
			RuleID:       "rule-001",
			Severity:     SeverityMedium,
			MetricName:   "test_metric",
			CurrentValue: 50,
			BaselineValue: 100,
			DeltaPct:     -50,
		},
		AllowedActions:   []string{"notify_owner", "create_followup_task"},
		ForbiddenActions: []string{"export_report"},
	}
}

func TestOpenAIProviderValidResponse(t *testing.T) {
	validJSON := validDecisionJSON()
	mockServer := newMockOpenAIServer(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": 1234567890,
			"model":   "gpt-4o-mini",
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"message":       map[string]interface{}{"role": "assistant", "content": validJSON, "refusal": ""},
					"finish_reason": "stop",
					"logprobs":      nil,
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     50,
				"completion_tokens": 100,
				"total_tokens":      150,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("failed to encode mock response: %v", err)
		}
	})
	defer mockServer.Close()

	provider := newTestOpenAIProvider(t, mockServer.URL)
	output, err := provider.GenerateDecision(context.Background(), testInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output == nil {
		t.Fatal("expected non-nil DecisionOutput")
	}
	if output.DecisionType != "investigate" {
		t.Errorf("DecisionType = %q, want %q", output.DecisionType, "investigate")
	}
	if output.Severity != "medium" {
		t.Errorf("Severity = %q, want %q", output.Severity, "medium")
	}
	if output.Summary != "test decision" {
		t.Errorf("Summary = %q, want %q", output.Summary, "test decision")
	}
	if len(output.Rationale) != 1 || output.Rationale[0] != "reason 1" {
		t.Errorf("Rationale = %v, want [reason 1]", output.Rationale)
	}
	if len(output.RecommendedActions) != 1 {
		t.Fatalf("Expected 1 recommended action, got %d", len(output.RecommendedActions))
	}
	if output.RecommendedActions[0].ActionType != "notify_owner" {
		t.Errorf("ActionType = %q, want %q", output.RecommendedActions[0].ActionType, "notify_owner")
	}
	if output.Confidence != 0.72 {
		t.Errorf("Confidence = %v, want %v", output.Confidence, 0.72)
	}
	if !output.RequiresHumanReview {
		t.Error("RequiresHumanReview = false, want true")
	}
}

func TestOpenAIProviderInvalidJSON(t *testing.T) {
	mockServer := newMockOpenAIServer(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": 1234567890,
			"model":   "gpt-4o-mini",
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"message":       map[string]interface{}{"role": "assistant", "content": "this is not json", "refusal": ""},
					"finish_reason": "stop",
					"logprobs":      nil,
				},
			},
			"usage": map[string]interface{}{"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	})
	defer mockServer.Close()

	provider := newTestOpenAIProvider(t, mockServer.URL)
	_, err := provider.GenerateDecision(context.Background(), testInput())
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !containsSubstring(err.Error(), "parse LLM response") {
		t.Errorf("expected error to contain 'parse LLM response', got: %v", err)
	}
}

func TestOpenAIProviderTimeout(t *testing.T) {
	mockServer := newMockOpenAIServer(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{})
	})
	defer mockServer.Close()

	cfg := &config.Config{
		LLMAPIKey:         "sk-test",
		LLMModel:          "gpt-4o-mini",
		LLMAPIBase:        mockServer.URL,
		LLMTemperature:    0.2,
		LLMMaxTokens:      2048,
		LLMTimeoutSeconds: 1,
	}
	registry, err := NewPromptRegistry()
	if err != nil {
		t.Fatalf("failed to create prompt registry: %v", err)
	}
	provider, err := NewOpenAIProvider(cfg, registry)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	ctx := context.Background()
	_, err = provider.GenerateDecision(ctx, testInput())
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	var perr *ProviderError
	if !errors.As(err, &perr) {
		t.Errorf("expected *ProviderError, got %T", err)
	}
}

func TestOpenAIProvider500(t *testing.T) {
	mockServer := newMockOpenAIServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": map[string]interface{}{"message": "internal server error"}})
	})
	defer mockServer.Close()

	provider := newTestOpenAIProvider(t, mockServer.URL)
	_, err := provider.GenerateDecision(context.Background(), testInput())
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
	var perr *ProviderError
	if !errors.As(err, &perr) {
		t.Errorf("expected *ProviderError, got %T", err)
	}
}

func TestOpenAIProviderEmptyChoices(t *testing.T) {
	mockServer := newMockOpenAIServer(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": 1234567890,
			"model":   "gpt-4o-mini",
			"choices": []map[string]interface{}{},
			"usage":   map[string]interface{}{"prompt_tokens": 10, "completion_tokens": 0, "total_tokens": 10},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	})
	defer mockServer.Close()

	provider := newTestOpenAIProvider(t, mockServer.URL)
	_, err := provider.GenerateDecision(context.Background(), testInput())
	if err == nil {
		t.Fatal("expected error for empty choices, got nil")
	}
	if !containsSubstring(err.Error(), "empty choices") {
		t.Errorf("expected error to contain 'empty choices', got: %v", err)
	}
}

func TestOpenAIProviderRefusal(t *testing.T) {
	mockServer := newMockOpenAIServer(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": 1234567890,
			"model":   "gpt-4o-mini",
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"message":       map[string]interface{}{"role": "assistant", "content": "", "refusal": "I cannot comply with this request"},
					"finish_reason": "stop",
					"logprobs":      nil,
				},
			},
			"usage": map[string]interface{}{"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	})
	defer mockServer.Close()

	provider := newTestOpenAIProvider(t, mockServer.URL)
	_, err := provider.GenerateDecision(context.Background(), testInput())
	if err == nil {
		t.Fatal("expected error for refusal, got nil")
	}
	if !containsSubstring(err.Error(), "refusal") {
		t.Errorf("expected error to contain 'refusal', got: %v", err)
	}
}

func TestOpenAIProviderRateLimit(t *testing.T) {
	mockServer := newMockOpenAIServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Rate limit exceeded",
				"type":    "rate_limit_error",
			},
		})
	})
	defer mockServer.Close()

	provider := newTestOpenAIProvider(t, mockServer.URL)
	_, err := provider.GenerateDecision(context.Background(), testInput())
	if err == nil {
		t.Fatal("expected error for 429 response, got nil")
	}
	var perr *ProviderError
	if !errors.As(err, &perr) {
		t.Errorf("expected *ProviderError, got %T", err)
	}
}

func TestOpenAIProvider_MissingAPIKey(t *testing.T) {
	cfg := &config.Config{
		LLMAPIKey: "",
	}
	registry, err := NewPromptRegistry()
	if err != nil {
		t.Fatalf("failed to create prompt registry: %v", err)
	}
	_, err = NewOpenAIProvider(cfg, registry)
	if err == nil {
		t.Fatal("expected error for missing API key, got nil")
	}
	if err.Error() != "LLM_API_KEY is required" {
		t.Errorf("expected error %q, got %q", "LLM_API_KEY is required", err.Error())
	}
}

func TestOpenAIProvider_Name(t *testing.T) {
	mockServer := newMockOpenAIServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": 1234567890,
			"model":   "gpt-4o-mini",
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"message":       map[string]interface{}{"role": "assistant", "content": validDecisionJSON(), "refusal": ""},
					"finish_reason": "stop",
					"logprobs":      nil,
				},
			},
			"usage": map[string]interface{}{"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15},
		})
	})
	defer mockServer.Close()

	provider := newTestOpenAIProvider(t, mockServer.URL)
	if provider.Name() != "openai_compatible" {
		t.Errorf("Name() = %q, want %q", provider.Name(), "openai_compatible")
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && len(substr) > 0 && contains(s, substr))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
