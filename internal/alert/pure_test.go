package alert

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ──── avg ──────────────────────────────────────────────────────────────────

func TestAvg_Empty(t *testing.T) {
	assert.Equal(t, 0.0, avg(nil))
	assert.Equal(t, 0.0, avg([]float64{}))
}

func TestAvg_SingleValue(t *testing.T) {
	assert.Equal(t, 42.0, avg([]float64{42}))
}

func TestAvg_MultipleValues(t *testing.T) {
	assert.Equal(t, 3.0, avg([]float64{1, 2, 3, 4, 5}))
}

func TestAvg_Fractional(t *testing.T) {
	assert.InDelta(t, 2.5, avg([]float64{1, 2, 3, 4}), 0.0001)
}

func TestAvg_NegativeValues(t *testing.T) {
	assert.Equal(t, 0.0, avg([]float64{-2, -1, 1, 2}))
}

// ──── roundTo ──────────────────────────────────────────────────────────────

func TestRoundTo_ZeroDecimals(t *testing.T) {
	assert.Equal(t, 3.0, roundTo(3.14159, 0))
	assert.Equal(t, 4.0, roundTo(3.999, 0))
}

func TestRoundTo_TwoDecimals(t *testing.T) {
	assert.Equal(t, 3.14, roundTo(3.14159, 2))
	assert.Equal(t, 3.14, roundTo(3.135, 2))
}

func TestRoundTo_FourDecimals(t *testing.T) {
	assert.Equal(t, 3.1416, roundTo(3.14159, 4))
	assert.Equal(t, 0.1235, roundTo(0.12345, 4))
}

func TestRoundTo_NegativeValue(t *testing.T) {
	assert.Equal(t, -3.14, roundTo(-3.14159, 2))
}

func TestRoundTo_ZeroValue(t *testing.T) {
	assert.Equal(t, 0.0, roundTo(0, 2))
}

// ──── absFloat ─────────────────────────────────────────────────────────────

func TestAbsFloat_Positive(t *testing.T) {
	assert.Equal(t, 42.0, absFloat(42))
}

func TestAbsFloat_Negative(t *testing.T) {
	assert.Equal(t, 42.0, absFloat(-42))
}

func TestAbsFloat_Zero(t *testing.T) {
	assert.Equal(t, 0.0, absFloat(0))
}

func TestAbsFloat_Fractional(t *testing.T) {
	assert.InDelta(t, 3.14, absFloat(-3.14), 0.0001)
}

// ──── severityWeights ──────────────────────────────────────────────────────

func TestSeverityWeights_Values(t *testing.T) {
	assert.Equal(t, 3.0, severityWeights["high"])
	assert.Equal(t, 2.0, severityWeights["medium"])
	assert.Equal(t, 1.0, severityWeights["low"])
}

func TestSeverityOrder_Values(t *testing.T) {
	assert.Equal(t, 0, severityOrder["high"])
	assert.Equal(t, 1, severityOrder["medium"])
	assert.Equal(t, 2, severityOrder["low"])
}

// ──── NewEngine ────────────────────────────────────────────────────────────

func TestNewEngine(t *testing.T) {
	e := NewEngine()
	assert.NotNil(t, e)
}

// ──── makeDimAlertID ──────────────────────────────────────────────────────

func TestMakeDimAlertID_HasPrefix(t *testing.T) {
	id := makeDimAlertID("seller_late_delivery_spike", "2018-10-15", "seller", "s-001")
	assert.Contains(t, id, "dim-")
	assert.Len(t, id, 4+12) // "dim-" prefix + 12-char hash
}

func TestMakeDimAlertID_DifferentInputsDiffer(t *testing.T) {
	id1 := makeDimAlertID("r1", "2018-10-15", "seller", "s-001")
	id2 := makeDimAlertID("r1", "2018-10-16", "seller", "s-001")
	assert.NotEqual(t, id1, id2)
}

// ──── CountAlertsByDimension ───────────────────────────────────────────────

func TestCountAlertsByDimension_Nil(t *testing.T) {
	assert.Equal(t, map[string]int{}, CountAlertsByDimension(nil))
	assert.Equal(t, map[string]int{}, CountAlertsByDimension([]DimensionalAlert{}))
}

func TestCountAlertsByDimension_SingleType(t *testing.T) {
	alerts := []DimensionalAlert{
		{ObjectType: "seller", AlertID: "a1"},
		{ObjectType: "seller", AlertID: "a2"},
	}
	counts := CountAlertsByDimension(alerts)
	assert.Equal(t, 2, counts["seller"])
}

func TestCountAlertsByDimension_MultipleTypes(t *testing.T) {
	alerts := []DimensionalAlert{
		{ObjectType: "seller", AlertID: "a1"},
		{ObjectType: "category", AlertID: "a2"},
		{ObjectType: "seller", AlertID: "a3"},
		{ObjectType: "region", AlertID: "a4"},
	}
	counts := CountAlertsByDimension(alerts)
	assert.Equal(t, 2, counts["seller"])
	assert.Equal(t, 1, counts["category"])
	assert.Equal(t, 1, counts["region"])
}

// ──── SuppressAlerts ───────────────────────────────────────────────────────

func TestSuppressAlerts_NoAlerts(t *testing.T) {
	result := SuppressAlerts([]DimensionalAlert{}, 50)
	assert.Empty(t, result.Alerts)
	assert.Equal(t, 0, result.Suppressed)
}

func TestSuppressAlerts_NilInput(t *testing.T) {
	result := SuppressAlerts(nil, 50)
	assert.Empty(t, result.Alerts)
	assert.Equal(t, 0, result.Suppressed)
}

func TestSuppressAlerts_UnderLimit(t *testing.T) {
	alerts := []DimensionalAlert{{AlertID: "a1", Severity: "high"}}
	result := SuppressAlerts(alerts, 50)
	assert.Len(t, result.Alerts, 1)
	assert.Equal(t, 0, result.Suppressed)
}

func TestSuppressAlerts_OverLimit(t *testing.T) {
	alerts := []DimensionalAlert{
		{AlertID: "a1", Severity: "low", ImpactScore: 1, SampleSize: 10},
		{AlertID: "a2", Severity: "high", ImpactScore: 100, SampleSize: 50},
		{AlertID: "a3", Severity: "medium", ImpactScore: 50, SampleSize: 30},
	}
	result := SuppressAlerts(alerts, 2)
	assert.Len(t, result.Alerts, 2)
	assert.Equal(t, 1, result.Suppressed)
	// High severity should be first
	assert.Equal(t, "a2", result.Alerts[0].AlertID)
}

func TestSuppressAlerts_SortsBySeverity(t *testing.T) {
	alerts := []DimensionalAlert{
		{AlertID: "a1", Severity: "low"},
		{AlertID: "a2", Severity: "high"},
		{AlertID: "a3", Severity: "medium"},
	}
	result := SuppressAlerts(alerts, 3)
	assert.Len(t, result.Alerts, 3)
	assert.Equal(t, "a2", result.Alerts[0].AlertID) // high
	assert.Equal(t, "a3", result.Alerts[1].AlertID) // medium
	assert.Equal(t, "a1", result.Alerts[2].AlertID) // low
}

func TestSuppressAlerts_SortsByImpactScore(t *testing.T) {
	alerts := []DimensionalAlert{
		{AlertID: "a1", Severity: "high", ImpactScore: 10, SampleSize: 5},
		{AlertID: "a2", Severity: "high", ImpactScore: 100, SampleSize: 50},
	}
	result := SuppressAlerts(alerts, 2)
	assert.Equal(t, "a2", result.Alerts[0].AlertID)
}

func TestSuppressAlerts_ZeroMaxAlerts(t *testing.T) {
	alerts := []DimensionalAlert{
		{AlertID: "a1", Severity: "high"},
	}
	result := SuppressAlerts(alerts, 0) // should use default 50
	assert.Len(t, result.Alerts, 1)
	assert.Equal(t, 0, result.Suppressed)
}

// ──── DefaultDimensionalRules ──────────────────────────────────────────────

func TestDefaultDimensionalRules_CountSix(t *testing.T) {
	rules := DefaultDimensionalRules()
	assert.Len(t, rules, 6)
}

func TestDefaultDimensionalRules_HasAllTypes(t *testing.T) {
	rules := DefaultDimensionalRules()
	ruleIDs := make(map[string]bool)
	for _, r := range rules {
		ruleIDs[r.RuleID] = true
	}
	assert.True(t, ruleIDs["seller_late_delivery_spike"])
	assert.True(t, ruleIDs["seller_review_score_drop"])
	assert.True(t, ruleIDs["category_gmv_drop"])
	assert.True(t, ruleIDs["category_low_review_cluster"])
	assert.True(t, ruleIDs["region_cancel_rate_spike"])
	assert.True(t, ruleIDs["region_late_delivery_spike"])
}

func TestDefaultDimensionalRules_HasConditions(t *testing.T) {
	for _, r := range DefaultDimensionalRules() {
		assert.NotEmpty(t, r.Condition, "rule %s should have condition", r.RuleID)
	}
}

// ──── GlobalRules ──────────────────────────────────────────────────────────

func TestGlobalRules_HasExpectedRules(t *testing.T) {
	rules := GlobalRules()
	ruleMap := make(map[string]AlertRule)
	for _, r := range rules {
		ruleMap[r.RuleID] = r
	}
	assert.Contains(t, ruleMap, "gmv_drop")
	assert.Contains(t, ruleMap, "late_delivery_spike")
	assert.Contains(t, ruleMap, "cancel_rate_spike")
}

func TestGlobalRules_HasDeadRules(t *testing.T) {
	rules := GlobalRules()
	for _, r := range rules {
		if r.RuleID == "review_score_drop" || r.RuleID == "seller_activation_gap" {
			assert.False(t, r.Enabled, "rule %s should be disabled (dead)", r.RuleID)
		}
	}
}

func TestGlobalRules_ActiveRulesHaveConditions(t *testing.T) {
	for _, r := range GlobalRules() {
		if r.Enabled {
			assert.NotNil(t, r.Condition, "enabled rule %s should have condition", r.RuleID)
		}
	}
}
