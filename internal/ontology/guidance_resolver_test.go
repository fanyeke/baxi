package ontology

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveGuidance_High(t *testing.T) {
	recipe := &ContextRecipe{
		Name: "test",
		DecisionGuidance: DecisionGuidance{
			Levels: []GuidanceLevel{
				{Severity: "low", Recommendation: "Monitor", Actions: nil},
				{Severity: "high", Recommendation: "Notify team", Actions: []string{"notify_owner", "create_followup_task"}},
			},
		},
	}
	level, err := ResolveGuidance(recipe, "high")
	require.NoError(t, err)
	require.NotNil(t, level)
	assert.Equal(t, "high", level.Severity)
	assert.Equal(t, "Notify team", level.Recommendation)
	assert.Equal(t, []string{"notify_owner", "create_followup_task"}, level.Actions)
}

func TestResolveGuidance_CaseInsensitive(t *testing.T) {
	recipe := &ContextRecipe{
		Name: "test",
		DecisionGuidance: DecisionGuidance{
			Levels: []GuidanceLevel{
				{Severity: "high", Recommendation: "Notify team", Actions: []string{"notify_owner"}},
			},
		},
	}

	// "HIGH" uppercase
	level, err := ResolveGuidance(recipe, "HIGH")
	require.NoError(t, err)
	require.NotNil(t, level)
	assert.Equal(t, "high", level.Severity)
	assert.Equal(t, "Notify team", level.Recommendation)

	// "High" title case
	level, err = ResolveGuidance(recipe, "High")
	require.NoError(t, err)
	assert.Equal(t, "high", level.Severity)

	// "high" lowercase
	level, err = ResolveGuidance(recipe, "high")
	require.NoError(t, err)
	assert.Equal(t, "high", level.Severity)
}

func TestResolveGuidance_Unknown(t *testing.T) {
	recipe := &ContextRecipe{
		Name: "test",
		DecisionGuidance: DecisionGuidance{
			Levels: []GuidanceLevel{
				{Severity: "low", Recommendation: "Monitor"},
				{Severity: "high", Recommendation: "Notify"},
			},
		},
	}
	level, err := ResolveGuidance(recipe, "unknown_severity")
	require.Error(t, err)
	assert.Nil(t, level)
	assert.Contains(t, err.Error(), "unknown_severity")
}

func TestResolveGuidance_AllLevels(t *testing.T) {
	recipe := &ContextRecipe{
		Name: "test",
		DecisionGuidance: DecisionGuidance{
			Levels: []GuidanceLevel{
				{Severity: "low", Recommendation: "Monitor, no action needed", Actions: nil},
				{Severity: "medium", Recommendation: "Create followup task for review", Actions: []string{"create_followup_task"}},
				{Severity: "high", Recommendation: "Notify team for intervention", Actions: []string{"create_followup_task", "notify_owner"}},
				{Severity: "critical", Recommendation: "Immediate escalation", Actions: []string{"create_followup_task", "notify_owner", "export_report"}},
			},
		},
	}
	expected := []struct {
		severity string
		actions  int
	}{
		{"low", 0},
		{"medium", 1},
		{"high", 2},
		{"critical", 3},
	}
	for _, e := range expected {
		t.Run(e.severity, func(t *testing.T) {
			level, err := ResolveGuidance(recipe, e.severity)
			require.NoError(t, err)
			require.NotNil(t, level)
			assert.Equal(t, e.severity, level.Severity)
			assert.NotEmpty(t, level.Recommendation)
			assert.Len(t, level.Actions, e.actions)
		})
	}
}

func TestGuidanceToPromptFragment(t *testing.T) {
	level := &GuidanceLevel{
		Severity:       "high",
		Recommendation: "Notify seller operations team for intervention",
		Actions:        []string{"create_followup_task", "notify_owner"},
	}
	fragment := GuidanceToPromptFragment(level)
	assert.Contains(t, fragment, level.Recommendation)
	assert.Contains(t, fragment, "create_followup_task")
	assert.Contains(t, fragment, "notify_owner")
	assert.Contains(t, fragment, "Requires human approval: yes.")
}

func TestResolveGuidance_RealRecipe(t *testing.T) {
	path := filepath.Join("..", "..", "config", "context_recipes.yml")
	data, err := os.ReadFile(path)
	require.NoError(t, err, "should read context recipes YAML")

	recipes, err := ParseContextRecipes(data)
	require.NoError(t, err, "should parse context recipes")

	recipe, ok := recipes["seller_late_delivery_alert"]
	require.True(t, ok, "expected seller_late_delivery_alert recipe")

	level, err := ResolveGuidance(recipe, "high")
	require.NoError(t, err)
	require.NotNil(t, level)
	assert.Equal(t, "high", level.Severity)
	assert.Equal(t, "Notify seller operations team for intervention", level.Recommendation)
	assert.Contains(t, level.Actions, "create_followup_task")
	assert.Contains(t, level.Actions, "notify_owner")
}
