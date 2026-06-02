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
		Source:         "metric:test",
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
		Name:          "test",
		EvidenceRules: []EvidenceRule{},
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

func TestUnifyEvidenceFormat(t *testing.T) {
	t.Run("metric evidence", func(t *testing.T) {
		re := RenderedEvidence{
			Source:   "metric:seller_late_delivery_rate_7d",
			Rendered: "Seller SELLER_001 late delivery rate is 0.31 (baseline: 0.08, delta: 0.23)",
		}
		vals := EvidenceValues{
			ObjectID: "SELLER_001",
			Current:  0.31,
			Baseline: 0.08,
			Delta:    0.23,
			Severity: "high",
		}
		ue := UnifyEvidenceFormat(re, vals, "seller")

		assert.Equal(t, "metric:SELLER_001:seller_late_delivery_rate_7d", ue.EvidenceID)
		assert.Equal(t, "metric", ue.Type)
		assert.Equal(t, "seller", ue.ObjectType)
		assert.Equal(t, "SELLER_001", ue.ObjectID)
		assert.Equal(t, 0.31, ue.Current)
		assert.Equal(t, 0.08, ue.Baseline)
		assert.Equal(t, 0.23, ue.Delta)
		assert.Equal(t, "high", ue.Severity)
		assert.Equal(t, re.Rendered, ue.Interpretation)
	})

	t.Run("link evidence", func(t *testing.T) {
		re := RenderedEvidence{
			Source:   "link:recent_orders",
			Rendered: "Recent orders for SELLER_042 show delivery patterns",
		}
		vals := EvidenceValues{
			ObjectID: "SELLER_042",
			Severity: "medium",
		}
		ue := UnifyEvidenceFormat(re, vals, "seller")

		assert.Equal(t, "link:SELLER_042:recent_orders", ue.EvidenceID)
		assert.Equal(t, "link", ue.Type)
		assert.Equal(t, "seller", ue.ObjectType)
		assert.Equal(t, "SELLER_042", ue.ObjectID)
		assert.Equal(t, 0.0, ue.Current)
		assert.Equal(t, 0.0, ue.Baseline)
		assert.Equal(t, 0.0, ue.Delta)
		assert.Equal(t, "medium", ue.Severity)
		assert.Equal(t, re.Rendered, ue.Interpretation)
	})

	t.Run("unknown source type", func(t *testing.T) {
		re := RenderedEvidence{
			Source:   "log:some_check",
			Rendered: "Check passed for OBJ_099",
		}
		vals := EvidenceValues{
			ObjectID: "OBJ_099",
			Severity: "low",
		}
		ue := UnifyEvidenceFormat(re, vals, "task")

		assert.Equal(t, "unknown", ue.Type)
		assert.Equal(t, "unknown:OBJ_099:log:some_check", ue.EvidenceID)
	})

	t.Run("zero values", func(t *testing.T) {
		re := RenderedEvidence{
			Source:   "metric:no_data_metric",
			Rendered: "No data for OBJ_000",
		}
		vals := EvidenceValues{
			ObjectID: "OBJ_000",
			Severity: "low",
		}
		ue := UnifyEvidenceFormat(re, vals, "order")

		assert.Equal(t, "metric", ue.Type)
		assert.Equal(t, 0.0, ue.Current)
		assert.Equal(t, 0.0, ue.Baseline)
		assert.Equal(t, 0.0, ue.Delta)
	})
}

func TestRenderRecipeEvidenceUnified(t *testing.T) {
	t.Run("with metric results", func(t *testing.T) {
		recipe := &ContextRecipe{
			Name: "test_recipe",
			EvidenceRules: []EvidenceRule{
				{Source: "metric:late_rate", Interpretation: "Late rate is {current} (baseline: {baseline})"},
				{Source: "link:delivery", Interpretation: "Delivery check for {object_id}"},
			},
		}
		metricResults := map[string]*MetricResult{
			"late_rate": {Value: 0.31, Baseline: 0.08},
		}
		unified := RenderRecipeEvidenceUnified(recipe, "SELLER_001", "seller", "high", metricResults)

		require.Len(t, unified, 2)

		// Metric evidence
		assert.Equal(t, "metric:SELLER_001:late_rate", unified[0].EvidenceID)
		assert.Equal(t, "metric", unified[0].Type)
		assert.Equal(t, "seller", unified[0].ObjectType)
		assert.Equal(t, "SELLER_001", unified[0].ObjectID)
		assert.Equal(t, 0.31, unified[0].Current)
		assert.Equal(t, 0.08, unified[0].Baseline)
		assert.InDelta(t, 0.23, unified[0].Delta, 0.001)
		assert.Equal(t, "high", unified[0].Severity)
		assert.Contains(t, unified[0].Interpretation, "0.31")

		// Link evidence
		assert.Equal(t, "link:SELLER_001:delivery", unified[1].EvidenceID)
		assert.Equal(t, "link", unified[1].Type)
		assert.Equal(t, "seller", unified[1].ObjectType)
		assert.Contains(t, unified[1].Interpretation, "SELLER_001")
	})

	t.Run("nil recipe", func(t *testing.T) {
		unified := RenderRecipeEvidenceUnified(nil, "OBJ_001", "seller", "", nil)
		assert.Nil(t, unified)
	})

	t.Run("empty rules", func(t *testing.T) {
		recipe := &ContextRecipe{
			Name:          "test",
			EvidenceRules: []EvidenceRule{},
		}
		unified := RenderRecipeEvidenceUnified(recipe, "OBJ_001", "seller", "", nil)
		assert.NotNil(t, unified)
		assert.Empty(t, unified)
	})

	t.Run("missing metric results", func(t *testing.T) {
		recipe := &ContextRecipe{
			Name: "test",
			EvidenceRules: []EvidenceRule{
				{Source: "metric:missing_metric", Interpretation: "Value is {current}"},
			},
		}
		unified := RenderRecipeEvidenceUnified(recipe, "OBJ_001", "seller", "low", nil)

		require.Len(t, unified, 1)
		assert.Equal(t, "metric:OBJ_001:missing_metric", unified[0].EvidenceID)
		assert.Equal(t, "metric", unified[0].Type)
		assert.Equal(t, 0.0, unified[0].Current)
		assert.Equal(t, 0.0, unified[0].Baseline)
		assert.Equal(t, 0.0, unified[0].Delta)
		assert.Equal(t, "low", unified[0].Severity)
		assert.Contains(t, unified[0].Interpretation, "0.00")
	})
}
