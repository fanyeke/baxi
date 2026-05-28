package handler

import (
	"net/http"

	"baxi/internal/config"
	"baxi/internal/eval"
	"baxi/internal/httputil"
)

// LLMHandler serves LLM status, configuration, and metrics endpoints.
type LLMHandler struct {
	cfg     *config.Config
	metrics *eval.MetricsCollector
}

// NewLLMHandler creates a new LLMHandler.
// If metrics is nil, a new MetricsCollector is created.
func NewLLMHandler(cfg *config.Config, metrics *eval.MetricsCollector) *LLMHandler {
	if metrics == nil {
		metrics = eval.NewMetricsCollector()
	}
	return &LLMHandler{cfg: cfg, metrics: metrics}
}

// Status handles GET /api/v1/llm/status.
func (h *LLMHandler) Status(w http.ResponseWriter, r *http.Request) {
	enabled := false
	provider := ""
	model := ""
	fallbackEnabled := false
	rawOutputStorage := false
	if h.cfg != nil {
		enabled = h.cfg.LLMEnabled
		provider = h.cfg.LLMProvider
		model = h.cfg.LLMModel
		fallbackEnabled = h.cfg.LLMFallbackEnabled
		rawOutputStorage = h.cfg.LLMStoreRawOutput
	}
	httputil.JSON(w, http.StatusOK, map[string]interface{}{
		"enabled":            enabled,
		"provider":           provider,
		"model":              model,
		"fallback_enabled":   fallbackEnabled,
		"raw_output_storage": rawOutputStorage,
	})
}

// Metrics handles GET /api/v1/llm/metrics.
func (h *LLMHandler) Metrics(w http.ResponseWriter, r *http.Request) {
	metrics := h.metrics.GetMetrics()
	httputil.JSON(w, http.StatusOK, metrics)
}
