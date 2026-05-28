package eval

import (
	"sync"
	"testing"
)

func TestRecordDecision(t *testing.T) {
	m := NewMetricsCollector()

	m.RecordDecision("openai", 150)
	m.RecordDecision("openai", 200)
	m.RecordDecision("rule_based", 5)

	metrics := m.GetMetrics()
	if metrics.TotalDecisions != 3 {
		t.Errorf("expected TotalDecisions=3, got %d", metrics.TotalDecisions)
	}
	if metrics.ProviderDecisionCount["openai"] != 2 {
		t.Errorf("expected openai count=2, got %d", metrics.ProviderDecisionCount["openai"])
	}
	if metrics.ProviderDecisionCount["rule_based"] != 1 {
		t.Errorf("expected rule_based count=1, got %d", metrics.ProviderDecisionCount["rule_based"])
	}
}

func TestRecordDecision_LatencyTracking(t *testing.T) {
	m := NewMetricsCollector()

	m.RecordDecision("openai", 100)
	m.RecordDecision("openai", 200)

	metrics := m.GetMetrics()
	if metrics.AverageLatencyMs != 150 {
		t.Errorf("expected AverageLatencyMs=150, got %.2f", metrics.AverageLatencyMs)
	}
}

func TestRecordFallback(t *testing.T) {
	m := NewMetricsCollector()

	m.RecordDecision("openai", 100)
	m.RecordDecision("openai", 100)
	m.RecordDecision("openai", 100)
	m.RecordDecision("openai", 100)
	m.RecordFallback("rate_limit")
	m.RecordFallback("validation_error")

	metrics := m.GetMetrics()
	if metrics.FallbackCount != 2 {
		t.Errorf("expected FallbackCount=2, got %d", metrics.FallbackCount)
	}
	if metrics.FallbackRate != 0.5 {
		t.Errorf("expected FallbackRate=0.5, got %.4f", metrics.FallbackRate)
	}
	if metrics.FallbackReasonCount["rate_limit"] != 1 {
		t.Errorf("expected rate_limit count=1, got %d", metrics.FallbackReasonCount["rate_limit"])
	}
	if metrics.FallbackReasonCount["validation_error"] != 1 {
		t.Errorf("expected validation_error count=1, got %d", metrics.FallbackReasonCount["validation_error"])
	}
}

func TestRecordFallback_WithReasons(t *testing.T) {
	m := NewMetricsCollector()

	m.RecordDecision("openai", 50)
	m.RecordFallback("timeout")
	m.RecordFallback("timeout")
	m.RecordFallback("auth_error")

	metrics := m.GetMetrics()
	if metrics.FallbackReasonCount["timeout"] != 2 {
		t.Errorf("expected timeout count=2, got %d", metrics.FallbackReasonCount["timeout"])
	}
	if metrics.FallbackReasonCount["auth_error"] != 1 {
		t.Errorf("expected auth_error count=1, got %d", metrics.FallbackReasonCount["auth_error"])
	}
}

func TestRecordValidationFailure(t *testing.T) {
	m := NewMetricsCollector()

	m.RecordDecision("openai", 100)
	m.RecordDecision("openai", 100)
	m.RecordDecision("openai", 100)
	m.RecordDecision("openai", 100)
	m.RecordValidationFailure()
	m.RecordValidationFailure()

	metrics := m.GetMetrics()
	if metrics.ValidationFailures != 2 {
		t.Errorf("expected ValidationFailures=2, got %d", metrics.ValidationFailures)
	}
	if metrics.ValidationFailureRate != 0.5 {
		t.Errorf("expected ValidationFailureRate=0.5, got %.4f", metrics.ValidationFailureRate)
	}
}

func TestRecordApproval(t *testing.T) {
	m := NewMetricsCollector()

	m.RecordApproval(true)
	m.RecordApproval(true)
	m.RecordApproval(true)
	m.RecordApproval(false)

	metrics := m.GetMetrics()
	if metrics.Approvals != 3 {
		t.Errorf("expected Approvals=3, got %d", metrics.Approvals)
	}
	if metrics.Rejections != 1 {
		t.Errorf("expected Rejections=1, got %d", metrics.Rejections)
	}
	if metrics.ApprovalRate != 0.75 {
		t.Errorf("expected ApprovalRate=0.75, got %.4f", metrics.ApprovalRate)
	}
}

func TestRecordApproval_AllRejected(t *testing.T) {
	m := NewMetricsCollector()

	m.RecordApproval(false)
	m.RecordApproval(false)

	metrics := m.GetMetrics()
	if metrics.Approvals != 0 {
		t.Errorf("expected Approvals=0, got %d", metrics.Approvals)
	}
	if metrics.Rejections != 2 {
		t.Errorf("expected Rejections=2, got %d", metrics.Rejections)
	}
	if metrics.ApprovalRate != 0 {
		t.Errorf("expected ApprovalRate=0, got %.4f", metrics.ApprovalRate)
	}
}

func TestGetMetrics_Empty(t *testing.T) {
	m := NewMetricsCollector()

	metrics := m.GetMetrics()

	if metrics.TotalDecisions != 0 {
		t.Errorf("expected TotalDecisions=0, got %d", metrics.TotalDecisions)
	}
	if metrics.FallbackCount != 0 {
		t.Errorf("expected FallbackCount=0, got %d", metrics.FallbackCount)
	}
	if metrics.FallbackRate != 0 {
		t.Errorf("expected FallbackRate=0, got %f", metrics.FallbackRate)
	}
	if metrics.ValidationFailures != 0 {
		t.Errorf("expected ValidationFailures=0, got %d", metrics.ValidationFailures)
	}
	if metrics.ValidationFailureRate != 0 {
		t.Errorf("expected ValidationFailureRate=0, got %f", metrics.ValidationFailureRate)
	}
	if metrics.Approvals != 0 {
		t.Errorf("expected Approvals=0, got %d", metrics.Approvals)
	}
	if metrics.Rejections != 0 {
		t.Errorf("expected Rejections=0, got %d", metrics.Rejections)
	}
	if metrics.ApprovalRate != 0 {
		t.Errorf("expected ApprovalRate=0, got %f", metrics.ApprovalRate)
	}
	if metrics.AverageLatencyMs != 0 {
		t.Errorf("expected AverageLatencyMs=0, got %f", metrics.AverageLatencyMs)
	}
	if len(metrics.ProviderDecisionCount) != 0 {
		t.Errorf("expected empty ProviderDecisionCount, got %v", metrics.ProviderDecisionCount)
	}
	if len(metrics.FallbackReasonCount) != 0 {
		t.Errorf("expected empty FallbackReasonCount, got %v", metrics.FallbackReasonCount)
	}
}

func TestGetMetrics_Snapshot(t *testing.T) {
	m := NewMetricsCollector()

	m.RecordDecision("openai", 200)
	m.RecordDecision("rule_based", 10)
	m.RecordFallback("timeout")
	m.RecordValidationFailure()
	m.RecordApproval(true)

	metrics := m.GetMetrics()

	if metrics.TotalDecisions != 2 {
		t.Errorf("expected TotalDecisions=2, got %d", metrics.TotalDecisions)
	}
	if metrics.FallbackCount != 1 {
		t.Errorf("expected FallbackCount=1, got %d", metrics.FallbackCount)
	}
	if metrics.FallbackRate != 0.5 {
		t.Errorf("expected FallbackRate=0.5, got %.4f", metrics.FallbackRate)
	}
	if metrics.ValidationFailures != 1 {
		t.Errorf("expected ValidationFailures=1, got %d", metrics.ValidationFailures)
	}
	if metrics.ValidationFailureRate != 0.5 {
		t.Errorf("expected ValidationFailureRate=0.5, got %.4f", metrics.ValidationFailureRate)
	}
	if metrics.Approvals != 1 {
		t.Errorf("expected Approvals=1, got %d", metrics.Approvals)
	}
	if metrics.Rejections != 0 {
		t.Errorf("expected Rejections=0, got %d", metrics.Rejections)
	}
	if metrics.ApprovalRate != 1.0 {
		t.Errorf("expected ApprovalRate=1.0, got %.4f", metrics.ApprovalRate)
	}
	if metrics.AverageLatencyMs != 105 {
		t.Errorf("expected AverageLatencyMs=105, got %.2f", metrics.AverageLatencyMs)
	}
	if metrics.ProviderDecisionCount["openai"] != 1 {
		t.Errorf("expected openai count=1, got %d", metrics.ProviderDecisionCount["openai"])
	}
	if metrics.ProviderDecisionCount["rule_based"] != 1 {
		t.Errorf("expected rule_based count=1, got %d", metrics.ProviderDecisionCount["rule_based"])
	}
	if metrics.FallbackReasonCount["timeout"] != 1 {
		t.Errorf("expected timeout count=1, got %d", metrics.FallbackReasonCount["timeout"])
	}
}

func TestGetMetrics_SnapshotIsolation(t *testing.T) {
	m := NewMetricsCollector()

	m.RecordDecision("openai", 100)
	metrics1 := m.GetMetrics()

	m.RecordDecision("rule_based", 50)
	metrics2 := m.GetMetrics()

	if metrics1.TotalDecisions != 1 {
		t.Errorf("expected metrics1 TotalDecisions=1 (isolated), got %d", metrics1.TotalDecisions)
	}
	if metrics2.TotalDecisions != 2 {
		t.Errorf("expected metrics2 TotalDecisions=2, got %d", metrics2.TotalDecisions)
	}
}

func TestConcurrentSafety(t *testing.T) {
	m := NewMetricsCollector()
	var wg sync.WaitGroup
	n := 50

	wg.Add(4 * n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			m.RecordDecision("openai", 100)
		}()
		go func() {
			defer wg.Done()
			m.RecordFallback("timeout")
		}()
		go func() {
			defer wg.Done()
			m.RecordValidationFailure()
		}()
		go func() {
			defer wg.Done()
			m.RecordApproval(true)
		}()
	}
	wg.Wait()

	metrics := m.GetMetrics()
	if metrics.TotalDecisions != n {
		t.Errorf("expected TotalDecisions=%d, got %d", n, metrics.TotalDecisions)
	}
	if metrics.FallbackCount != n {
		t.Errorf("expected FallbackCount=%d, got %d", n, metrics.FallbackCount)
	}
	if metrics.ValidationFailures != n {
		t.Errorf("expected ValidationFailures=%d, got %d", n, metrics.ValidationFailures)
	}
	if metrics.Approvals != n {
		t.Errorf("expected Approvals=%d, got %d", n, metrics.Approvals)
	}
}
