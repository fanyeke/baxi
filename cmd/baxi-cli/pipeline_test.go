package main

import (
	"testing"
)

func TestAbs64_Positive(t *testing.T) {
	if got := abs64(42); got != 42 {
		t.Errorf("abs64(42) = %d, want 42", got)
	}
}

func TestAbs64_Negative(t *testing.T) {
	if got := abs64(-42); got != 42 {
		t.Errorf("abs64(-42) = %d, want 42", got)
	}
}

func TestAbs64_Zero(t *testing.T) {
	if got := abs64(0); got != 0 {
		t.Errorf("abs64(0) = %d, want 0", got)
	}
}

func TestAllSteps_NotEmpty(t *testing.T) {
	steps := allSteps()
	if len(steps) == 0 {
		t.Fatal("allSteps() returned empty slice")
	}
	for _, s := range steps {
		t.Logf("Step: %s", s.Name())
	}
}

func TestAllSteps_HasExpectedSteps(t *testing.T) {
	steps := allSteps()
	stepNames := make(map[string]bool)
	for _, s := range steps {
		stepNames[s.Name()] = true
	}

	expected := []string{
		"ingest_raw", "build_dwd_order_level", "build_dwd_item_level",
		"build_metric_daily", "build_metric_dimension_daily", "detect_alerts",
		"generate_recommendations", "generate_tasks", "create_outbox_events",
	}
	for _, name := range expected {
		if !stepNames[name] {
			t.Errorf("allSteps() missing expected step: %s", name)
		}
	}
}

func TestAllSteps_Count(t *testing.T) {
	steps := allSteps()
	if len(steps) < 6 {
		t.Errorf("allSteps() returned %d steps, expected at least 6", len(steps))
	}
}
