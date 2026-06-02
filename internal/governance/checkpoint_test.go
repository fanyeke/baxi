package governance

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	governanceRepo "baxi/internal/repository/governance"
)

// mockSnapshotProvider implements ConfigSnapshotProvider for testing.
type mockSnapshotProvider struct {
	rows []governanceRepo.ConfigSnapshotRow
	err  error
}

func (m *mockSnapshotProvider) GetConfigSnapshots(ctx context.Context) ([]governanceRepo.ConfigSnapshotRow, error) {
	if m.err != nil {
		return nil, m.err
	}
	out := make([]governanceRepo.ConfigSnapshotRow, len(m.rows))
	copy(out, m.rows)
	return out, nil
}

func TestCheckpointService_RequiresCheckpoint_SensitiveActions(t *testing.T) {
	tests := []struct {
		name     string
		action   string
		expected bool
	}{
		{"execute_dispatch requires checkpoint", "execute_dispatch", true},
		{"modify_business_policy requires checkpoint", "modify_business_policy", true},
		{"trigger_pipeline requires checkpoint", "trigger_pipeline", true},
		{"view_dashboard does not require checkpoint", "view_dashboard", false},
		{"empty action does not require checkpoint", "", false},
		{"unknown action does not require checkpoint", "unknown_action", false},
	}

	svc := NewCheckpointServiceWithProvider(&mockSnapshotProvider{})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.RequiresCheckpoint(context.Background(), tt.action)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestCheckpointService_RequiresCheckpoint_WithConfigSnapshotRules(t *testing.T) {
	rows := []governanceRepo.ConfigSnapshotRow{
		{
			ConfigKey: "checkpoint_rules",
			Status:    `{"checkpoints":[{"action":"bulk_delete","requires_reason":true,"requires_human_review":true}]}`,
		},
	}
	svc := NewCheckpointServiceWithProvider(&mockSnapshotProvider{rows: rows})

	assert.True(t, svc.RequiresCheckpoint(context.Background(), "bulk_delete"))
	assert.True(t, svc.RequiresCheckpoint(context.Background(), "execute_dispatch"))
	assert.False(t, svc.RequiresCheckpoint(context.Background(), "view_dashboard"))
}

func TestCheckpointService_RequiresCheckpoint_ConfigSnapshotError(t *testing.T) {
	// When provider errors, only built-in sensitive actions return true.
	svc := NewCheckpointServiceWithProvider(&mockSnapshotProvider{err: assert.AnError})

	assert.True(t, svc.RequiresCheckpoint(context.Background(), "execute_dispatch"))
	assert.False(t, svc.RequiresCheckpoint(context.Background(), "bulk_delete"))
	assert.False(t, svc.RequiresCheckpoint(context.Background(), "view_dashboard"))
}

func TestCheckpointService_RequiresCheckpoint_InvalidJSON(t *testing.T) {
	rows := []governanceRepo.ConfigSnapshotRow{
		{
			ConfigKey: "checkpoint_rules",
			Status:    `invalid json`,
		},
	}
	svc := NewCheckpointServiceWithProvider(&mockSnapshotProvider{rows: rows})

	assert.True(t, svc.RequiresCheckpoint(context.Background(), "execute_dispatch"))
	assert.False(t, svc.RequiresCheckpoint(context.Background(), "bulk_delete"))
}

func TestCheckpointService_RequiresCheckpoint_NonCheckpointConfig(t *testing.T) {
	rows := []governanceRepo.ConfigSnapshotRow{
		{ConfigKey: "access_policy", Status: `{}`},
		{ConfigKey: "classification", Status: `{}`},
	}
	svc := NewCheckpointServiceWithProvider(&mockSnapshotProvider{rows: rows})

	assert.True(t, svc.RequiresCheckpoint(context.Background(), "execute_dispatch"))
	assert.False(t, svc.RequiresCheckpoint(context.Background(), "bulk_delete"))
}

func TestCheckpointService_GetRules_WithEmptyProvider(t *testing.T) {
	svc := NewCheckpointServiceWithProvider(&mockSnapshotProvider{})
	rules := svc.GetRules(context.Background())

	assert.Len(t, rules, 3) // only built-in rules
	assert.Equal(t, "execute_dispatch", rules[0].Action)
	assert.Equal(t, "modify_business_policy", rules[1].Action)
	assert.Equal(t, "trigger_pipeline", rules[2].Action)
}

func TestCheckpointService_GetRules_WithConfigSnapshot(t *testing.T) {
	rows := []governanceRepo.ConfigSnapshotRow{
		{
			ConfigKey: "checkpoint_rules.yml",
			Status: `{"checkpoints":[
				{"action":"bulk_delete","requires_reason":true,"requires_human_review":true},
				{"action":"export_data","requires_reason":true,"requires_human_review":false}
			]}`,
		},
	}
	svc := NewCheckpointServiceWithProvider(&mockSnapshotProvider{rows: rows})
	rules := svc.GetRules(context.Background())

	assert.Len(t, rules, 5) // 2 from config + 3 built-in

	// Built-in should be included
	actions := make(map[string]bool)
	for _, r := range rules {
		actions[r.Action] = true
	}
	assert.True(t, actions["execute_dispatch"])
	assert.True(t, actions["modify_business_policy"])
	assert.True(t, actions["trigger_pipeline"])
	assert.True(t, actions["bulk_delete"])
	assert.True(t, actions["export_data"])
}

func TestCheckpointService_GetRules_MergesWithBuiltins(t *testing.T) {
	// When config defines rules that overlap with built-ins, deduplicate.
	rows := []governanceRepo.ConfigSnapshotRow{
		{
			ConfigKey: "checkpoint_rules",
			Status:    `{"checkpoints":[{"action":"execute_dispatch","requires_reason":false,"requires_human_review":false}]}`,
		},
	}
	svc := NewCheckpointServiceWithProvider(&mockSnapshotProvider{rows: rows})
	rules := svc.GetRules(context.Background())

	assert.Len(t, rules, 3) // built-in execute_dispatch dedup'd with config version

	// Config version should override built-in
	for _, r := range rules {
		if r.Action == "execute_dispatch" {
			assert.False(t, r.RequiresReason)
			assert.False(t, r.RequiresHumanReview)
		}
	}
}

func TestCheckpointService_GetRules_ErrorReturnsNil(t *testing.T) {
	svc := NewCheckpointServiceWithProvider(&mockSnapshotProvider{err: assert.AnError})
	rules := svc.GetRules(context.Background())
	assert.Nil(t, rules)
}

func TestCheckpointService_GetRules_InvalidJSONInConfig(t *testing.T) {
	rows := []governanceRepo.ConfigSnapshotRow{
		{ConfigKey: "checkpoint_rules", Status: `bad json`},
	}
	svc := NewCheckpointServiceWithProvider(&mockSnapshotProvider{rows: rows})
	rules := svc.GetRules(context.Background())

	assert.Len(t, rules, 3) // only built-ins
}

func TestCheckpointService_GetRules_AllConfigKeys(t *testing.T) {
	// Test both config key variants: "checkpoint_rules" and "checkpoint_rules.yml"
	t.Run("checkpoint_rules key", func(t *testing.T) {
		rows := []governanceRepo.ConfigSnapshotRow{
			{ConfigKey: "checkpoint_rules", Status: `{"checkpoints":[{"action":"test_action","requires_reason":true}]}`},
		}
		svc := NewCheckpointServiceWithProvider(&mockSnapshotProvider{rows: rows})
		rules := svc.GetRules(context.Background())
		assert.True(t, hasAction(rules, "test_action"))
	})

	t.Run("checkpoint_rules.yml key", func(t *testing.T) {
		rows := []governanceRepo.ConfigSnapshotRow{
			{ConfigKey: "checkpoint_rules.yml", Status: `{"checkpoints":[{"action":"test_action","requires_reason":true}]}`},
		}
		svc := NewCheckpointServiceWithProvider(&mockSnapshotProvider{rows: rows})
		rules := svc.GetRules(context.Background())
		assert.True(t, hasAction(rules, "test_action"))
	})
}

func hasAction(rules []CheckpointRule, action string) bool {
	for _, r := range rules {
		if r.Action == action {
			return true
		}
	}
	return false
}
