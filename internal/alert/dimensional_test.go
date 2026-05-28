package alert

import (
	"math"
	"testing"
)

func TestEvaluateCondition_ValueGt(t *testing.T) {
	tests := []struct {
		name      string
		condition string
		curVal    float64
		want      bool
	}{
		{"above threshold", "value_gt: 0.25", 0.30, true},
		{"below threshold", "value_gt: 0.25", 0.20, false},
		{"at threshold", "value_gt: 0.25", 0.25, false},
		{"zero threshold", "value_gt: 0.0", 0.01, true},
		{"negative threshold above", "value_gt: -0.5", -0.3, true},
		{"negative threshold below", "value_gt: -0.5", -0.6, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evaluateCondition(tt.condition, tt.curVal, nil, 0)
			if got != tt.want {
				t.Errorf("evaluateCondition(%q, %f) = %v, want %v", tt.condition, tt.curVal, got, tt.want)
			}
		})
	}
}

func TestEvaluateCondition_ValueLt(t *testing.T) {
	tests := []struct {
		name      string
		condition string
		curVal    float64
		want      bool
	}{
		{"below threshold", "value_lt: 3.5", 3.0, true},
		{"above threshold", "value_lt: 3.5", 4.0, false},
		{"at threshold", "value_lt: 3.5", 3.5, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evaluateCondition(tt.condition, tt.curVal, nil, 0)
			if got != tt.want {
				t.Errorf("evaluateCondition(%q, %f) = %v, want %v", tt.condition, tt.curVal, got, tt.want)
			}
		})
	}
}

func TestEvaluateCondition_ChangeRateLt(t *testing.T) {
	baseline := float64Ptr(100.0)
	tests := []struct {
		name       string
		condition  string
		curVal     float64
		changeRate float64
		want       bool
	}{
		{"drop exceeds threshold", "change_rate_lt: -0.20", 50, -0.5, true},
		{"drop within threshold", "change_rate_lt: -0.20", 90, -0.1, false},
		{"positive change", "change_rate_lt: -0.20", 120, 0.2, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evaluateCondition(tt.condition, tt.curVal, baseline, tt.changeRate)
			if got != tt.want {
				t.Errorf("evaluateCondition(%q, %f) = %v, want %v", tt.condition, tt.curVal, got, tt.want)
			}
		})
	}
}

func TestEvaluateCondition_ChangeRateGt(t *testing.T) {
	baseline := float64Ptr(100.0)
	tests := []struct {
		name       string
		condition  string
		curVal     float64
		changeRate float64
		want       bool
	}{
		{"increase exceeds threshold", "change_rate_gt: 0.5", 200, 1.0, true},
		{"increase within threshold", "change_rate_gt: 0.5", 140, 0.4, false},
		{"negative change", "change_rate_gt: 0.5", 80, -0.2, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evaluateCondition(tt.condition, tt.curVal, baseline, tt.changeRate)
			if got != tt.want {
				t.Errorf("evaluateCondition(%q, %f) = %v, want %v", tt.condition, tt.curVal, got, tt.want)
			}
		})
	}
}

func TestEvaluateCondition_Unknown(t *testing.T) {
	if evaluateCondition("unknown_condition", 100, nil, 0) {
		t.Error("unknown condition should return false")
	}
}

func TestEvaluateCondition_ZeroBaseline(t *testing.T) {
	bv := float64Ptr(0)
	if evaluateCondition("change_rate_lt: -0.1", 50, bv, 0) {
		t.Error("change_rate with zero baseline should return false")
	}
}

func TestMakeDimAlertID_Format(t *testing.T) {
	id := makeDimAlertID("region_late_delivery_spike", "2018-08-06", "region", "SP")
	if len(id) != 16 {
		t.Errorf("expected length 16 (dim- + 12 hex), got %d (%q)", len(id), id)
	}
	if id[:4] != "dim-" {
		t.Errorf("expected dim- prefix, got %q", id[:4])
	}
	for _, c := range id[4:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("non-hex character in alert ID: %c", c)
		}
	}
}

func TestMakeDimAlertID_Deterministic(t *testing.T) {
	id1 := makeDimAlertID("region_late_delivery_spike", "2018-08-06", "region", "SP")
	id2 := makeDimAlertID("region_late_delivery_spike", "2018-08-06", "region", "SP")
	if id1 != id2 {
		t.Errorf("deterministic IDs should match: %q vs %q", id1, id2)
	}
}

func TestMakeDimAlertID_DifferentValues(t *testing.T) {
	id1 := makeDimAlertID("region_late_delivery_spike", "2018-08-06", "region", "SP")
	id2 := makeDimAlertID("region_late_delivery_spike", "2018-08-06", "region", "RJ")
	if id1 == id2 {
		t.Error("different dimension values should produce different alert IDs")
	}
}

func TestImpactScoring_High(t *testing.T) {
	score := severityWeights["high"] * 100
	if math.Abs(score-300.0) > 0.001 {
		t.Errorf("high impact: expected 300, got %f", score)
	}
}

func TestImpactScoring_Medium(t *testing.T) {
	score := severityWeights["medium"] * 75
	if math.Abs(score-150.0) > 0.001 {
		t.Errorf("medium impact: expected 150, got %f", score)
	}
}

func TestImpactScoring_Low(t *testing.T) {
	score := severityWeights["low"] * 50
	if math.Abs(score-50.0) > 0.001 {
		t.Errorf("low impact: expected 50, got %f", score)
	}
}

func TestImpactScoring_ZeroSample(t *testing.T) {
	score := severityWeights["high"] * 0
	if score != 0 {
		t.Errorf("zero sample: expected 0, got %f", score)
	}
}

func makeTestAlert(id string, severity string, score float64, sampleSize int64) DimensionalAlert {
	return DimensionalAlert{
		AlertID:     id,
		Severity:    severity,
		ImpactScore: score,
		SampleSize:  sampleSize,
	}
}

func TestSuppressAlerts_NoSuppression(t *testing.T) {
	alerts := []DimensionalAlert{
		makeTestAlert("a1", "high", 100, 50),
		makeTestAlert("a2", "medium", 80, 40),
		makeTestAlert("a3", "low", 60, 30),
	}
	result := SuppressAlerts(alerts, 10)
	if len(result.Alerts) != 3 {
		t.Errorf("expected 3 alerts, got %d", len(result.Alerts))
	}
	if result.Suppressed != 0 {
		t.Errorf("expected 0 suppressed, got %d", result.Suppressed)
	}
}

func TestSuppressAlerts_GlobalCap(t *testing.T) {
	alerts := []DimensionalAlert{
		makeTestAlert("a1", "high", 100, 50),
		makeTestAlert("a2", "high", 90, 45),
		makeTestAlert("a3", "high", 80, 40),
	}
	result := SuppressAlerts(alerts, 2)
	if len(result.Alerts) != 2 {
		t.Errorf("expected 2 alerts, got %d", len(result.Alerts))
	}
	if result.Suppressed != 1 {
		t.Errorf("expected 1 suppressed, got %d", result.Suppressed)
	}
}

func TestSuppressAlerts_SortBySeverity(t *testing.T) {
	alerts := []DimensionalAlert{
		makeTestAlert("low_score", "low", 200, 100),
		makeTestAlert("high_score", "high", 50, 25),
		makeTestAlert("med_score", "medium", 100, 50),
	}
	result := SuppressAlerts(alerts, 3)
	expected := []string{"high_score", "med_score", "low_score"}
	for i, a := range result.Alerts {
		if a.AlertID != expected[i] {
			t.Errorf("position %d: expected %q, got %q", i, expected[i], a.AlertID)
		}
	}
}

func TestSuppressAlerts_SortByImpactWithinSeverity(t *testing.T) {
	alerts := []DimensionalAlert{
		makeTestAlert("med_high", "medium", 200, 100),
		makeTestAlert("med_low", "medium", 100, 50),
		makeTestAlert("med_mid", "medium", 150, 75),
	}
	result := SuppressAlerts(alerts, 3)
	expected := []string{"med_high", "med_mid", "med_low"}
	for i, a := range result.Alerts {
		if a.AlertID != expected[i] {
			t.Errorf("position %d: expected %q, got %q", i, expected[i], a.AlertID)
		}
	}
}

func TestSuppressAlerts_Empty(t *testing.T) {
	result := SuppressAlerts(nil, 50)
	if result.Alerts != nil {
		t.Errorf("expected nil alerts, got %d", len(result.Alerts))
	}
	if result.Suppressed != 0 {
		t.Errorf("expected 0 suppressed, got %d", result.Suppressed)
	}
}

func TestDefaultDimensionalRules_Count(t *testing.T) {
	rules := DefaultDimensionalRules()
	if len(rules) != 6 {
		t.Errorf("expected 6 dimensional rules, got %d", len(rules))
	}
}

func TestDefaultDimensionalRules_Dimensions(t *testing.T) {
	rules := DefaultDimensionalRules()
	dims := map[string]int{}
	for _, r := range rules {
		dims[r.DimensionType]++
	}
	if dims["seller"] != 2 || dims["category"] != 2 || dims["region"] != 2 {
		t.Errorf("expected 2 per dimension, got seller=%d category=%d region=%d", dims["seller"], dims["category"], dims["region"])
	}
}

func TestDefaultDimensionalRules_Severities(t *testing.T) {
	rules := DefaultDimensionalRules()
	sevs := map[string]int{}
	for _, r := range rules {
		sevs[r.Severity]++
	}
	if sevs["high"] != 2 {
		t.Errorf("expected 2 high, got %d", sevs["high"])
	}
	if sevs["medium"] != 4 {
		t.Errorf("expected 4 medium, got %d", sevs["medium"])
	}
}

func TestCountAlertsByDimension(t *testing.T) {
	alerts := []DimensionalAlert{
		{AlertID: "a1", ObjectType: "seller"},
		{AlertID: "a2", ObjectType: "seller"},
		{AlertID: "a3", ObjectType: "category"},
		{AlertID: "a4", ObjectType: "region"},
	}
	counts := CountAlertsByDimension(alerts)
	if counts["seller"] != 2 || counts["category"] != 1 || counts["region"] != 1 {
		t.Errorf("unexpected counts: %v", counts)
	}
}

func TestCountAlertsByDimension_Empty(t *testing.T) {
	counts := CountAlertsByDimension(nil)
	if len(counts) != 0 {
		t.Errorf("expected empty, got %d entries", len(counts))
	}
}

func float64Ptr(v float64) *float64 {
	return &v
}
