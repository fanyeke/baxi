package steps

import (
	"testing"

	"baxi/internal/alert"

	"github.com/stretchr/testify/assert"
)

// ──── deriveTargetChannel ──────────────────────────────────────────────────

func TestDeriveTargetChannel_DimensionalRule(t *testing.T) {
	assert.Equal(t, "feishu_cli", deriveTargetChannel("dimensional_rule"))
}

func TestDeriveTargetChannel_HeuristicStrategy(t *testing.T) {
	assert.Equal(t, "local_cli", deriveTargetChannel("heuristic_strategy"))
}

func TestDeriveTargetChannel_Unknown(t *testing.T) {
	assert.Equal(t, "local_cli", deriveTargetChannel("unknown_source"))
}

func TestDeriveTargetChannel_Empty(t *testing.T) {
	assert.Equal(t, "local_cli", deriveTargetChannel(""))
}

// ──── IsDimensionalTask ────────────────────────────────────────────────────

func TestIsDimensionalTask_DimTaskPrefix(t *testing.T) {
	assert.True(t, IsDimensionalTask("dimtask-001"))
	assert.True(t, IsDimensionalTask("dimtask-fraud-42"))
}

func TestIsDimensionalTask_OtherPrefix(t *testing.T) {
	assert.False(t, IsDimensionalTask("global-001"))
	assert.False(t, IsDimensionalTask("task-abc"))
}

func TestIsDimensionalTask_EmptyID(t *testing.T) {
	assert.False(t, IsDimensionalTask(""))
}

// ──── countEnabledGlobalRules ─────────────────────────────────────────────

func TestCountEnabledGlobalRules_AllEnabled(t *testing.T) {
	rules := []alert.AlertRule{
		{Name: "r1", Enabled: true},
		{Name: "r2", Enabled: true},
		{Name: "r3", Enabled: true},
	}
	assert.Equal(t, 3, countEnabledGlobalRules(rules))
}

func TestCountEnabledGlobalRules_SomeEnabled(t *testing.T) {
	rules := []alert.AlertRule{
		{Name: "r1", Enabled: true},
		{Name: "r2", Enabled: false},
		{Name: "r3", Enabled: true},
	}
	assert.Equal(t, 2, countEnabledGlobalRules(rules))
}

func TestCountEnabledGlobalRules_NoneEnabled(t *testing.T) {
	rules := []alert.AlertRule{
		{Name: "r1", Enabled: false},
		{Name: "r2", Enabled: false},
	}
	assert.Equal(t, 0, countEnabledGlobalRules(rules))
}

func TestCountEnabledGlobalRules_Empty(t *testing.T) {
	assert.Equal(t, 0, countEnabledGlobalRules(nil))
	assert.Equal(t, 0, countEnabledGlobalRules([]alert.AlertRule{}))
}

// ──── sanitizeJSON ─────────────────────────────────────────────────────────

func TestSanitizeJSON_ValidJSON(t *testing.T) {
	result := sanitizeJSON(`{"key": "value"}`)
	assert.Equal(t, `{"key": "value"}`, result)
}

func TestSanitizeJSON_EmptyString(t *testing.T) {
	result := sanitizeJSON("")
	assert.Equal(t, "{}", result)
}

func TestSanitizeJSON_InvalidJSON(t *testing.T) {
	result := sanitizeJSON("not-json")
	assert.Equal(t, "{}", result)
}

func TestSanitizeJSON_NestedJSON(t *testing.T) {
	result := sanitizeJSON(`{"nested": {"inner": 42}}`)
	assert.Equal(t, `{"nested": {"inner": 42}}`, result)
}

func TestSanitizeJSON_ArrayJSON(t *testing.T) {
	result := sanitizeJSON(`[1, 2, 3]`)
	assert.Equal(t, `[1, 2, 3]`, result)
}

// ──── deriveOwnerRole ──────────────────────────────────────────────────────

func TestDeriveOwnerRole_KnownRules(t *testing.T) {
	assert.Equal(t, "business_ops", deriveOwnerRole("gmv_drop"))
	assert.Equal(t, "logistics_ops", deriveOwnerRole("late_delivery_spike"))
	assert.Equal(t, "logistics_ops", deriveOwnerRole("cancel_rate_spike"))
	assert.Equal(t, "category_ops", deriveOwnerRole("review_score_drop"))
	assert.Equal(t, "seller_ops", deriveOwnerRole("seller_activation_gap"))
}

func TestDeriveOwnerRole_UnknownRule(t *testing.T) {
	assert.Equal(t, "unassigned", deriveOwnerRole("unknown_rule"))
}

func TestDeriveOwnerRole_EmptyRule(t *testing.T) {
	assert.Equal(t, "unassigned", deriveOwnerRole(""))
}
