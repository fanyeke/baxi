package decision

import (
	"testing"
)

func TestBuildEvidenceItems_EmptyContext(t *testing.T) {
	items := buildEvidenceItems(&DecisionContext{})
	if len(items) != 0 {
		t.Errorf("buildEvidenceItems(empty) = %d items, want 0", len(items))
	}
}

func TestBuildEvidenceItems_WithAlert(t *testing.T) {
	dc := &DecisionContext{
		Trigger: TriggerInfo{
			AlertID: "alert-123",
			RuleID:  "rule-456",
		},
	}
	items := buildEvidenceItems(dc)
	if len(items) < 2 {
		t.Fatalf("buildEvidenceItems(alert) = %d items, want >= 2", len(items))
	}
	if items[0].Type != "alert" || items[0].Key != "alert_id" || items[0].Value != "alert-123" {
		t.Errorf("first evidence item = %+v, want alert/alert_id/alert-123", items[0])
	}
}

func TestBuildEvidenceItems_WithMetric(t *testing.T) {
	dc := &DecisionContext{
		Trigger: TriggerInfo{
			MetricName:   "gmv",
			CurrentValue: 1000,
			BaselineValue: 900,
			DeltaPct:     11.1,
		},
	}
	items := buildEvidenceItems(dc)
	if len(items) < 4 {
		t.Fatalf("buildEvidenceItems(metric) = %d items, want >= 4", len(items))
	}
	if items[0].Type != "metric" || items[0].Key != "metric_name" {
		t.Errorf("first metric item = %+v, want metric/metric_name", items[0])
	}
}

func TestBuildEvidenceItems_FullContext(t *testing.T) {
	dc := &DecisionContext{
		Trigger: TriggerInfo{
			AlertID:      "alert-001",
			RuleID:       "rule-001",
			MetricName:   "gmv",
			CurrentValue: 5000,
			BaselineValue: 4500,
			DeltaPct:     11.1,
		},
	}
	items := buildEvidenceItems(dc)
	if len(items) < 6 {
		t.Fatalf("buildEvidenceItems(full) = %d items, want >= 6 (full alert + metric)", len(items))
	}
	hasAlert := false
	hasMetric := false
	for _, item := range items {
		if item.Type == "alert" {
			hasAlert = true
		}
		if item.Type == "metric" {
			hasMetric = true
		}
	}
	if !hasAlert {
		t.Error("buildEvidenceItems(full) missing alert evidence items")
	}
	if !hasMetric {
		t.Error("buildEvidenceItems(full) missing metric evidence items")
	}
}
