package ontology

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderEvidence_BasicSubstitution(t *testing.T) {
	rule := EvidenceRule{
		Source: "metric:test",
		Interpretation: "Seller {object_id} late delivery rate is {current} (baseline: {baseline}, delta: {delta}), severity: {severity}",
	}
	params := map[string]interface{}{
		"object_id": "SELLER_001",
		"current":   0.31,
		"baseline":  0.08,
		"delta":     0.23,
		"severity":  "high",
	}
	result := RenderEvidence(rule.Interpretation, params)
	expected := "Seller SELLER_001 late delivery rate is 0.31 (baseline: 0.08, delta: 0.23), severity: high"
	assert.Equal(t, expected, result)
}

func TestRenderEvidence_MissingPlaceholders(t *testing.T) {
	// Template uses {object_id} and {current} plus {status} which is missing from params.
	// Missing keys should be replaced with "N/A", no panic.
	template := "Seller {object_id} has {current} orders (status: {status})"
	params := map[string]interface{}{
		"object_id": "SELLER_002",
		"current":   42,
	}
	result := RenderEvidence(template, params)
	expected := "Seller SELLER_002 has 42 orders (status: N/A)"
	assert.Equal(t, expected, result)
}

func TestRenderEvidence_FormatNumbers(t *testing.T) {
	result := RenderEvidence("{current}", map[string]interface{}{"current": 0.3167})
	assert.Equal(t, "0.32", result, "current should format to 2 decimal places")

	result = RenderEvidence("{delta}", map[string]interface{}{"delta": -2.156})
	assert.Equal(t, "-2.16", result, "delta should format to 2 decimal places")

	result = RenderEvidence("{baseline}", map[string]interface{}{"baseline": 5.0})
	assert.Equal(t, "5.00", result, "baseline should format to 2 decimal places")
}

func TestRenderEvidence_NoPlaceholders(t *testing.T) {
	template := "Recent orders show delivery and review patterns"
	params := map[string]interface{}{
		"object_id": "SELLER_004",
	}
	result := RenderEvidence(template, params)
	assert.Equal(t, template, result)
}

func TestRenderEvidence_EmptyTemplate(t *testing.T) {
	result := RenderEvidence("", map[string]interface{}{"object_id": "SELLER_005"})
	assert.Equal(t, "", result)
}

func TestRenderRecipeEvidence_WithSellerLateDelivery(t *testing.T) {
	path := filepath.Join("..", "..", "config", "context_recipes.yml")
	data, err := os.ReadFile(path)
	require.NoError(t, err, "should read context recipes YAML")

	recipes, err := ParseContextRecipes(data)
	require.NoError(t, err, "should parse context recipes")

	recipe, ok := recipes["seller_late_delivery_alert"]
	require.True(t, ok, "expected seller_late_delivery_alert recipe")
	require.NotEmpty(t, recipe.EvidenceRules, "seller_late_delivery_alert should have evidence rules")

	metricResults := map[string]*MetricResult{
		"seller_late_delivery_rate_7d": {Value: 0.31, Baseline: 0.08, Label: "Late Delivery Rate (7d)"},
		"seller_order_count_7d":        {Value: 142.0, Baseline: 98.0, Label: "Order Count (7d)"},
		"seller_gmv_7d":                {Value: 28500.0, Baseline: 22000.0, Label: "GMV (7d)"},
		"seller_avg_review_score_7d":   {Value: 4.2, Baseline: 4.5, Label: "Avg Review Score (7d)"},
	}

	rendered := RenderRecipeEvidence(recipe, "SELLER_001", "", metricResults)
	require.NotNil(t, rendered, "should produce rendered evidence")
	require.Len(t, rendered, len(recipe.EvidenceRules), "should render all evidence rules")

	for _, re := range rendered {
		t.Run(re.Source, func(t *testing.T) {
			assert.NotEmpty(t, re.Source, "source should not be empty")
			assert.NotEmpty(t, re.Rendered, "rendered should not be empty")
			assert.NotContains(t, re.Rendered, "{", "rendered output should not contain raw placeholders")
			assert.NotContains(t, re.Rendered, "}", "rendered output should not contain raw placeholders")
		})
	}
}

func TestRenderRecipeEvidence_NilRecipe(t *testing.T) {
	rendered := RenderRecipeEvidence(nil, "OBJ_001", "", nil)
	assert.Nil(t, rendered, "nil recipe should return nil")
}

func TestRenderRecipeEvidence_EmptyRules(t *testing.T) {
	recipe := &ContextRecipe{
		Name:           "test",
		EvidenceRules:  []EvidenceRule{},
	}
	rendered := RenderRecipeEvidence(recipe, "OBJ_001", "", nil)
	assert.NotNil(t, rendered, "empty rules should return empty slice")
	assert.Empty(t, rendered, "empty rules should return empty slice")
}

func TestRenderRecipeEvidence_MissingMetricResults(t *testing.T) {
	recipe := &ContextRecipe{
		Name: "test",
		EvidenceRules: []EvidenceRule{
			{Source: "metric:missing_metric", Interpretation: "Metric {object_id} value is {current}"},
		},
	}
	rendered := RenderRecipeEvidence(recipe, "OBJ_001", "", nil)
	require.Len(t, rendered, 1)
	assert.Equal(t, "metric:missing_metric", rendered[0].Source)
	// Missing metric defaults to 0.00 for current/baseline/delta
	assert.Contains(t, rendered[0].Rendered, "0.00")
}

func TestRenderRecipeEvidence_LinkRule(t *testing.T) {
	recipe := &ContextRecipe{
		Name: "test",
		EvidenceRules: []EvidenceRule{
			{Source: "link:recent_orders", Interpretation: "Recent orders show delivery patterns"},
		},
	}
	rendered := RenderRecipeEvidence(recipe, "OBJ_001", "", nil)
	require.Len(t, rendered, 1)
	assert.Equal(t, "link:recent_orders", rendered[0].Source)
	assert.Equal(t, "Recent orders show delivery patterns", rendered[0].Rendered)
}

func TestRenderRecipeEvidence_LinkRuleWithObjectID(t *testing.T) {
	recipe := &ContextRecipe{
		Name: "test",
		EvidenceRules: []EvidenceRule{
			{Source: "link:orders", Interpretation: "Orders for {object_id} show patterns"},
		},
	}
	rendered := RenderRecipeEvidence(recipe, "SELLER_042", "", nil)
	require.Len(t, rendered, 1)
	assert.Equal(t, "Orders for SELLER_042 show patterns", rendered[0].Rendered)
}

func TestRenderEvidence_AllEvidenceRules(t *testing.T) {
	path := filepath.Join("..", "..", "config", "context_recipes.yml")
	data, err := os.ReadFile(path)
	require.NoError(t, err, "should read context recipes YAML")

	recipes, err := ParseContextRecipes(data)
	require.NoError(t, err, "should parse context recipes")

	recipe, ok := recipes["seller_late_delivery_alert"]
	require.True(t, ok, "expected seller_late_delivery_alert recipe")
	require.NotEmpty(t, recipe.EvidenceRules, "seller_late_delivery_alert should have evidence rules")

	params := map[string]interface{}{
		"object_id": "SELLER_001",
		"current":   0.31,
		"baseline":  0.08,
		"delta":     0.23,
		"severity":  "high",
	}

	for _, rule := range recipe.EvidenceRules {
		t.Run(rule.Source, func(t *testing.T) {
			result := RenderEvidence(rule.Interpretation, params)
			assert.NotContains(t, result, "{", "rendered output should not contain raw placeholders")
			assert.NotContains(t, result, "}", "rendered output should not contain raw placeholders")
			assert.NotEmpty(t, result, "rendered output should not be empty")
		})
	}
}
