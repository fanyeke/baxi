package eval

import (
	"sync"
)

// MetricsCollector tracks LLM decision metrics in memory.
// All methods are thread-safe via sync.RWMutex.
// This is an in-memory collector for Phase 8; Phase 9+ can add DB persistence.
type MetricsCollector struct {
	mu                 sync.RWMutex
	totalDecisions     int
	fallbackCount      int
	validationFailures int
	approvals          int
	rejections         int
	totalLatencyMs     int64
	latencySampleCount int
	providerDecisions  map[string]int
	fallbackReasons    map[string]int
}

// DecisionMetrics is a snapshot of the current metrics state.
type DecisionMetrics struct {
	TotalDecisions        int            `json:"total_decisions"`
	FallbackCount         int            `json:"fallback_count"`
	FallbackRate          float64        `json:"fallback_rate"`
	ValidationFailures    int            `json:"validation_failures"`
	ValidationFailureRate float64        `json:"validation_failure_rate"`
	Approvals             int            `json:"approvals"`
	Rejections            int            `json:"rejections"`
	ApprovalRate          float64        `json:"approval_rate"`
	AverageLatencyMs      float64        `json:"avg_latency_ms"`
	ProviderDecisionCount map[string]int `json:"provider_decision_count"`
	FallbackReasonCount   map[string]int `json:"fallback_reason_count"`
}

// NewMetricsCollector creates a new MetricsCollector with zeroed counters.
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		providerDecisions: make(map[string]int),
		fallbackReasons:   make(map[string]int),
	}
}

// RecordDecision records an LLM decision made by the given provider with the
// observed latency in milliseconds.
func (m *MetricsCollector) RecordDecision(provider string, latencyMs int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalDecisions++
	m.providerDecisions[provider]++
	m.totalLatencyMs += latencyMs
	m.latencySampleCount++
}

// RecordFallback records that a fallback occurred for the given reason.
func (m *MetricsCollector) RecordFallback(reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.fallbackCount++
	m.fallbackReasons[reason]++
}

// RecordValidationFailure records that a decision failed validation.
func (m *MetricsCollector) RecordValidationFailure() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.validationFailures++
}

// RecordApproval records the result of a human review.
func (m *MetricsCollector) RecordApproval(approved bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if approved {
		m.approvals++
	} else {
		m.rejections++
	}
}

// GetMetrics returns a snapshot of the current metrics.
func (m *MetricsCollector) GetMetrics() DecisionMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var fallbackRate, validationFailureRate, approvalRate float64
	var avgLatencyMs float64

	if m.totalDecisions > 0 {
		fallbackRate = float64(m.fallbackCount) / float64(m.totalDecisions)
		validationFailureRate = float64(m.validationFailures) / float64(m.totalDecisions)
	}

	totalApprovalDecisions := m.approvals + m.rejections
	if totalApprovalDecisions > 0 {
		approvalRate = float64(m.approvals) / float64(totalApprovalDecisions)
	}

	if m.latencySampleCount > 0 {
		avgLatencyMs = float64(m.totalLatencyMs) / float64(m.latencySampleCount)
	}

	// Copy maps to avoid caller mutation of internal state.
	providerCopy := make(map[string]int, len(m.providerDecisions))
	for k, v := range m.providerDecisions {
		providerCopy[k] = v
	}
	reasonCopy := make(map[string]int, len(m.fallbackReasons))
	for k, v := range m.fallbackReasons {
		reasonCopy[k] = v
	}

	return DecisionMetrics{
		TotalDecisions:        m.totalDecisions,
		FallbackCount:         m.fallbackCount,
		FallbackRate:          fallbackRate,
		ValidationFailures:    m.validationFailures,
		ValidationFailureRate: validationFailureRate,
		Approvals:             m.approvals,
		Rejections:            m.rejections,
		ApprovalRate:          approvalRate,
		AverageLatencyMs:      avgLatencyMs,
		ProviderDecisionCount: providerCopy,
		FallbackReasonCount:   reasonCopy,
	}
}
