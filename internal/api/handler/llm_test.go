package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/config"
	"baxi/internal/eval"
)

// ── Status ───────────────────────────────────────────────────────────

func TestLLMHandler_Status_Enabled(t *testing.T) {
	cfg := &config.Config{
		LLMEnabled:           true,
		LLMProvider:          "openai",
		LLMModel:             "gpt-4",
		LLMFallbackEnabled:   true,
		LLMStoreRawOutput:    true,
	}
	h := NewLLMHandler(cfg, nil)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/llm/status", nil)
	w := httptest.NewRecorder()
	h.Status(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, true, resp["enabled"])
	assert.Equal(t, "openai", resp["provider"])
	assert.Equal(t, "gpt-4", resp["model"])
	assert.Equal(t, true, resp["fallback_enabled"])
	assert.Equal(t, true, resp["raw_output_storage"])
}

func TestLLMHandler_Status_Disabled(t *testing.T) {
	cfg := &config.Config{
		LLMEnabled: false,
	}
	h := NewLLMHandler(cfg, nil)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/llm/status", nil)
	w := httptest.NewRecorder()
	h.Status(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, false, resp["enabled"])
	assert.Equal(t, "", resp["provider"])
	assert.Equal(t, "", resp["model"])
}

func TestLLMHandler_Status_NilConfig(t *testing.T) {
	h := NewLLMHandler(nil, nil)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/llm/status", nil)
	w := httptest.NewRecorder()
	h.Status(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, false, resp["enabled"])
	assert.Equal(t, "", resp["provider"])
}

// ── Metrics ──────────────────────────────────────────────────────────

func TestLLMHandler_Metrics_Empty(t *testing.T) {
	h := NewLLMHandler(nil, nil)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/llm/metrics", nil)
	w := httptest.NewRecorder()
	h.Metrics(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	// MetricsCollector returns empty counts when nothing recorded
	assert.NotNil(t, resp)
}

func TestLLMHandler_Metrics_WithData(t *testing.T) {
	metrics := eval.NewMetricsCollector()
	// Record some metrics
	metrics.RecordDecision("openai", 1500)
	metrics.RecordDecision("openai", 3200)
	metrics.RecordDecision("openai", 800)

	h := NewLLMHandler(nil, metrics)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/llm/metrics", nil)
	w := httptest.NewRecorder()
	h.Metrics(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	total, hasTotal := resp["total_decisions"]
	assert.True(t, hasTotal, "response should contain total_decisions")
	if hasTotal {
		assert.Equal(t, float64(3), total)
	}
}

func TestLLMHandler_NewWithNilMetrics(t *testing.T) {
	// When metrics is nil, NewLLMHandler should create its own MetricsCollector
	cfg := &config.Config{LLMEnabled: true}
	h := NewLLMHandler(cfg, nil)

	// Should handle metrics endpoint without panic
	r := httptest.NewRequest(http.MethodGet, "/api/v1/llm/metrics", nil)
	w := httptest.NewRecorder()
	h.Metrics(w, r)

	require.Equal(t, http.StatusOK, w.Code)
}
