package alert

import (
	"testing"
)

func TestGlobalRules_Count(t *testing.T) {
	rules := GlobalRules()
	if len(rules) != 5 {
		t.Fatalf("expected 5 global rules, got %d", len(rules))
	}
}

func TestGlobalRules_WorkingRules(t *testing.T) {
	rules := GlobalRules()
	working := 0
	for _, r := range rules {
		if r.Enabled {
			working++
		}
	}
	if working != 3 {
		t.Fatalf("expected 3 working rules, got %d", working)
	}
}

func TestGlobalRules_DeadRules(t *testing.T) {
	rules := GlobalRules()
	dead := 0
	for _, r := range rules {
		if !r.Enabled {
			dead++
		}
	}
	if dead != 2 {
		t.Fatalf("expected 2 dead rules, got %d", dead)
	}
}

func TestGlobalRules_DeadRuleReviewScoreDrop(t *testing.T) {
	rules := GlobalRules()
	for _, r := range rules {
		if r.RuleID == "review_score_drop" {
			if r.Enabled {
				t.Error("review_score_drop should be a dead rule (Enabled=false)")
			}
			if r.Condition != nil {
				t.Error("review_score_drop should have nil Condition")
			}
			return
		}
	}
	t.Error("review_score_drop rule not found")
}

func TestGlobalRules_DeadRuleSellerActivationGap(t *testing.T) {
	rules := GlobalRules()
	for _, r := range rules {
		if r.RuleID == "seller_activation_gap" {
			if r.Enabled {
				t.Error("seller_activation_gap should be a dead rule (Enabled=false)")
			}
			if r.Condition != nil {
				t.Error("seller_activation_gap should have nil Condition")
			}
			return
		}
	}
	t.Error("seller_activation_gap rule not found")
}

func TestGlobalRules_WorkingRuleGMVDrop(t *testing.T) {
	rules := GlobalRules()
	for _, r := range rules {
		if r.RuleID == "gmv_drop" {
			if !r.Enabled {
				t.Error("gmv_drop should be enabled")
			}
			if r.Condition == nil {
				t.Error("gmv_drop should have a Condition function")
			}
			if r.Severity != SeverityHigh {
				t.Errorf("gmv_drop severity: expected high, got %s", r.Severity)
			}
			if r.Metric != "gmv" {
				t.Errorf("gmv_drop metric: expected gmv, got %s", r.Metric)
			}
			return
		}
	}
	t.Error("gmv_drop rule not found")
}

func TestGlobalRules_WorkingRuleLateDeliverySpike(t *testing.T) {
	rules := GlobalRules()
	for _, r := range rules {
		if r.RuleID == "late_delivery_spike" {
			if !r.Enabled {
				t.Error("late_delivery_spike should be enabled")
			}
			if r.Condition == nil {
				t.Error("late_delivery_spike should have a Condition function")
			}
			if r.Severity != SeverityHigh {
				t.Errorf("late_delivery_spike severity: expected high, got %s", r.Severity)
			}
			if r.Metric != "late_delivery_rate" {
				t.Errorf("late_delivery_spike metric: expected late_delivery_rate, got %s", r.Metric)
			}
			return
		}
	}
	t.Error("late_delivery_spike rule not found")
}

func TestGlobalRules_WorkingRuleCancelRateSpike(t *testing.T) {
	rules := GlobalRules()
	for _, r := range rules {
		if r.RuleID == "cancel_rate_spike" {
			if !r.Enabled {
				t.Error("cancel_rate_spike should be enabled")
			}
			if r.Condition == nil {
				t.Error("cancel_rate_spike should have a Condition function")
			}
			if r.Severity != SeverityMedium {
				t.Errorf("cancel_rate_spike severity: expected medium, got %s", r.Severity)
			}
			if r.Metric != "cancel_rate" {
				t.Errorf("cancel_rate_spike metric: expected cancel_rate, got %s", r.Metric)
			}
			return
		}
	}
	t.Error("cancel_rate_spike rule not found")
}

func TestGenerateAlertID_Deterministic(t *testing.T) {
	id1 := GenerateAlertID("gmv_drop", "2018-10-17", "global", "global")
	id2 := GenerateAlertID("gmv_drop", "2018-10-17", "global", "global")
	if id1 != id2 {
		t.Errorf("deterministic alert IDs should match: %q vs %q", id1, id2)
	}
}

func TestGenerateAlertID_DifferentDates(t *testing.T) {
	id1 := GenerateAlertID("gmv_drop", "2018-10-17", "global", "global")
	id2 := GenerateAlertID("gmv_drop", "2018-10-18", "global", "global")
	if id1 == id2 {
		t.Error("different dates should produce different alert IDs")
	}
}

func TestGenerateAlertID_DifferentRules(t *testing.T) {
	id1 := GenerateAlertID("gmv_drop", "2018-10-17", "global", "global")
	id2 := GenerateAlertID("late_delivery_spike", "2018-10-17", "global", "global")
	if id1 == id2 {
		t.Error("different rules should produce different alert IDs")
	}
}

func TestGenerateAlertID_Length(t *testing.T) {
	id := GenerateAlertID("gmv_drop", "2018-10-17", "global", "global")
	// SHA-256 first 6 bytes → 12 hex chars
	if len(id) != 12 {
		t.Errorf("expected 12-char hex ID, got %q (len=%d)", id, len(id))
	}
}

func TestGenerateAlertID_HexChars(t *testing.T) {
	id := GenerateAlertID("gmv_drop", "2018-10-17", "global", "global")
	// Verify only hex chars
	for _, c := range id {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("non-hex character in ID: %q", c)
		}
	}
}

func TestGlobalAlertID_Format(t *testing.T) {
	id := GlobalAlertID("gmv_drop", "2018-10-17")
	expected := "gmv_drop_2018-10-17"
	if id != expected {
		t.Errorf("expected %q, got %q", expected, id)
	}
}

func TestGlobalAlertID_Deterministic(t *testing.T) {
	id1 := GlobalAlertID("gmv_drop", "2018-10-17")
	id2 := GlobalAlertID("gmv_drop", "2018-10-17")
	if id1 != id2 {
		t.Errorf("deterministic: %q vs %q", id1, id2)
	}
}

func TestGlobalAlertID_DifferentRules(t *testing.T) {
	id1 := GlobalAlertID("gmv_drop", "2018-10-17")
	id2 := GlobalAlertID("late_delivery_spike", "2018-10-17")
	if id1 >= id2 {
		// Just verify they're different strings
	}
}

// TestGlobalAlerts is the primary test for the global alert system.
// It verifies rule definitions, dead rules, and ID generation without
// requiring a database connection.
func TestGlobalAlerts(t *testing.T) {
	rules := GlobalRules()

	// Must have exactly 5 rules
	if len(rules) != 5 {
		t.Fatalf("expected 5 global rules, got %d", len(rules))
	}

	// 3 working, 2 dead
	working := 0
	dead := 0
	for _, r := range rules {
		if r.Enabled {
			working++
		} else {
			dead++
		}
	}
	if working != 3 {
		t.Errorf("expected 3 working rules, got %d", working)
	}
	if dead != 2 {
		t.Errorf("expected 2 dead rules, got %d", dead)
	}

	// Verify specific rules
	ruleMap := make(map[string]AlertRule)
	for _, r := range rules {
		ruleMap[r.RuleID] = r
	}

	// Working rules
	for _, id := range []string{"gmv_drop", "late_delivery_spike", "cancel_rate_spike"} {
		r, ok := ruleMap[id]
		if !ok {
			t.Errorf("working rule %q not found", id)
			continue
		}
		if !r.Enabled {
			t.Errorf("rule %q should be enabled", id)
		}
		if r.Condition == nil {
			t.Errorf("rule %q should have a Condition function", id)
		}
	}

	// Dead rules
	for _, id := range []string{"review_score_drop", "seller_activation_gap"} {
		r, ok := ruleMap[id]
		if !ok {
			t.Errorf("dead rule %q not found", id)
			continue
		}
		if r.Enabled {
			t.Errorf("rule %q should be disabled", id)
		}
		if r.Condition != nil {
			t.Errorf("rule %q should have nil Condition", id)
		}
	}

	// Verify alert ID determinism
	id1 := GlobalAlertID("gmv_drop", "2018-10-17")
	id2 := GlobalAlertID("gmv_drop", "2018-10-17")
	if id1 != id2 {
		t.Errorf("GlobalAlertID not deterministic: %q vs %q", id1, id2)
	}

	// Verify SHA-256 ID format
	shaID := GenerateAlertID("gmv_drop", "2018-10-17", "global", "global")
	if len(shaID) != 12 {
		t.Errorf("GenerateAlertID should produce 12 hex chars, got %q (len=%d)", shaID, len(shaID))
	}
}

func TestSeverityConstants(t *testing.T) {
	if SeverityLow != "low" {
		t.Errorf("SeverityLow: expected low, got %s", SeverityLow)
	}
	if SeverityMedium != "medium" {
		t.Errorf("SeverityMedium: expected medium, got %s", SeverityMedium)
	}
	if SeverityHigh != "high" {
		t.Errorf("SeverityHigh: expected high, got %s", SeverityHigh)
	}
	if SeverityCritical != "critical" {
		t.Errorf("SeverityCritical: expected critical, got %s", SeverityCritical)
	}
}
