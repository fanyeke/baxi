package ontology

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests verify that the v2 ontology schema, metric definitions, and
// context recipes can be loaded from their YAML config files and pass
// validation. They serve as integration smoke tests for the v2 config layer.

func configPath(rel string) string {
	return filepath.Join("..", "..", "config", rel)
}

func TestIntegration_LoadV2SchemaAndValidate(t *testing.T) {
	// Path relative to internal/ontology/
	path := configPath("aip_object_schema_v2.yml")
	data, err := os.ReadFile(path)
	require.NoError(t, err, "should read v2 schema YAML")

	objects, err := ParseObjectSchemaV2(data)
	require.NoError(t, err, "should parse v2 schema")
	require.NotEmpty(t, objects, "should have objects")

	// Validate all v2 objects
	issues := ValidateV2(objects)
	for _, iss := range issues {
		t.Logf("validation issue: %s", iss.String())
	}

	hasErrors := false
	for _, iss := range issues {
		if iss.Severity == "error" {
			hasErrors = true
			break
		}
	}
	assert.False(t, hasErrors, "v2 schema validation should have no errors")

	// Verify expected object types exist
	expectedTypes := []string{"seller", "order", "product", "metric_alert", "customer", "category", "region", "global"}
	for _, name := range expectedTypes {
		ot, ok := objects[name]
		assert.True(t, ok, "expected object type %q", name)
		if ok {
			assert.NotEmpty(t, ot.Source.Schema, "%s: source.schema required", name)
			assert.NotEmpty(t, ot.Source.Table, "%s: source.table required", name)
			assert.NotEmpty(t, ot.Source.PrimaryKey, "%s: source.primary_key required", name)
			assert.NotEmpty(t, ot.Properties, "%s: properties required", name)
		}
	}

	// Verify link resolution structure
	seller, ok := objects["seller"]
	if ok {
		assert.Len(t, seller.Links, 2, "seller should have 2 links (recent_orders, products)")
		for _, link := range seller.Links {
			assert.Contains(t, []string{"recent_orders", "products"}, link.Name)
			assert.NotEmpty(t, link.TargetType)
			assert.NotEmpty(t, link.Cardinality)
			assert.NotEmpty(t, link.Strategy)
		}
	}
}

func TestIntegration_LoadMetricDefinitions(t *testing.T) {
	path := configPath("metric_definitions.yml")
	data, err := os.ReadFile(path)
	require.NoError(t, err, "should read metric definitions YAML")

	metrics, err := ParseMetricDefinitions(data)
	require.NoError(t, err, "should parse metric definitions")
	require.NotEmpty(t, metrics, "should have metric definitions")

	// Verify expected metrics exist — TDD red phase for object-level metrics.
	// Seller (expand from 3 to 5), product (4), category (3), region (3), customer (3).
	// Total: 5 + 4 + 3 + 3 + 3 = 18
	expectedMetrics := []string{
		"seller_late_delivery_rate_7d",
		"seller_order_count_7d",
		"seller_gmv_7d",
		"seller_avg_review_score_7d",
		"seller_cancel_rate_7d",
		"product_gmv_7d",
		"product_order_count_7d",
		"product_avg_review_score_7d",
		"product_review_drop_7d",
		"category_gmv_7d",
		"category_order_count_7d",
		"category_avg_review_score_7d",
		"region_order_count_7d",
		"region_late_delivery_rate_7d",
		"region_gmv_7d",
		"customer_order_count_90d",
		"customer_gmv_90d",
		"customer_avg_review_score",
	}
	for _, name := range expectedMetrics {
		m, ok := metrics[name]
		assert.True(t, ok, "expected metric %q", name)
		if ok {
			assert.NotEmpty(t, m.ObjectType, "%s: object_type required", name)
			assert.NotEmpty(t, m.Grain, "%s: grain required", name)
		}
	}
}

func TestIntegration_LoadContextRecipes(t *testing.T) {
	path := configPath("context_recipes.yml")
	data, err := os.ReadFile(path)
	require.NoError(t, err, "should read context recipes YAML")

	recipes, err := ParseContextRecipes(data)
	require.NoError(t, err, "should parse context recipes")
	require.NotEmpty(t, recipes, "should have context recipes")

	// TDD red phase: expecting 5 recipes but only 1 exists in YAML.
	assert.Len(t, recipes, 5, "expected 5 context recipes — TDD red phase")

	// Verify the seller_late_delivery_alert recipe
	recipe, ok := recipes["seller_late_delivery_alert"]
	require.True(t, ok, "expected seller_late_delivery_alert recipe")

	assert.Equal(t, "metric_alert", recipe.Trigger.ObjectType)
	assert.Equal(t, "seller_late_delivery_spike", recipe.Trigger.RuleID)
	assert.Equal(t, "alert.object_type", recipe.RootObject.TypeFrom)
	assert.Equal(t, "alert.object_id", recipe.RootObject.IDFrom)

	// Verify included components
	assert.Contains(t, recipe.Include.Metrics, "seller_late_delivery_rate_7d")
	assert.Contains(t, recipe.Include.Actions, "notify_owner")
	assert.Contains(t, recipe.Include.Actions, "create_followup_task")

	// Verify link includes
	linkConfig, hasLinks := recipe.Include.Links["recent_orders"]
	require.True(t, hasLinks, "should include recent_orders link")
	assert.Equal(t, 10, linkConfig.Limit)
	assert.Contains(t, linkConfig.Fields, "order_id")

	// Verify budget defaults
	assert.Equal(t, 2, recipe.Budget.MaxLinkDepth)
	assert.Equal(t, 30, recipe.Budget.MaxObjects)
}

func TestIntegration_LoadContextRecipes_AllFive(t *testing.T) {
	// Load context recipes using the registry loader.
	path := configPath("context_recipes.yml")
	recipes, err := LoadContextRecipes(path)
	require.NoError(t, err, "should load context recipes")
	require.NotEmpty(t, recipes, "should have context recipes")

	// All 5 context recipes are now defined in config/context_recipes.yml.
	require.Len(t, recipes, 5, "expected 5 context recipes")

	// Expected recipe IDs once all 5 are defined.
	expectedRecipeIDs := []string{
		"seller_late_delivery_alert",
		"order_anomaly_alert",
		"customer_churn_risk_alert",
		"product_performance_drop_alert",
		"inventory_stockout_alert",
	}

	for _, recipeID := range expectedRecipeIDs {
		recipe, ok := recipes[recipeID]
		require.True(t, ok, "expected recipe %q", recipeID)

		// ── Basic metadata ──────────────────────────────────────────────────────
		assert.NotEmpty(t, recipe.Description, "%s: description required", recipeID)

		// ── Trigger ─────────────────────────────────────────────────────────────
		assert.NotEmpty(t, recipe.Trigger.ObjectType, "%s: trigger.object_type required", recipeID)
		assert.NotEmpty(t, recipe.Trigger.RuleID, "%s: trigger.rule_id required", recipeID)

		// ── Root object ─────────────────────────────────────────────────────────
		assert.NotEmpty(t, recipe.RootObject.TypeFrom, "%s: root_object.type_from required", recipeID)
		assert.NotEmpty(t, recipe.RootObject.IDFrom, "%s: root_object.id_from required", recipeID)

		// ── Include ─────────────────────────────────────────────────────────────
		assert.NotEmpty(t, recipe.Include.RootProperties, "%s: include.root_properties must be non-empty", recipeID)
		assert.NotEmpty(t, recipe.Include.Metrics, "%s: include.metrics must be non-empty", recipeID)
		assert.NotEmpty(t, recipe.Include.Links, "%s: include.links must be non-empty", recipeID)
		assert.NotEmpty(t, recipe.Include.Actions, "%s: include.actions must be non-empty", recipeID)

		for linkName, link := range recipe.Include.Links {
			assert.Greater(t, link.Limit, 0, "%s: link %q limit must be positive", recipeID, linkName)
			assert.NotEmpty(t, link.Fields, "%s: link %q fields must be non-empty", recipeID, linkName)
		}

		// TDD red phase: evidence_rules / EvidenceRules field does not exist yet
		// on ContextRecipe. This assertion drives adding the field to the struct.
		require.NotEmpty(t, recipe.EvidenceRules,
			"%s: evidence_rules required — TDD red phase: field does not exist yet", recipeID)
		assert.GreaterOrEqual(t, len(recipe.EvidenceRules), 2,
			"%s: evidence_rules must have at least 2 sources", recipeID)

		// TDD red phase: decision_guidance / DecisionGuidance field does not exist
		// yet on ContextRecipe. This assertion drives adding the field.
		require.NotNil(t, recipe.DecisionGuidance,
			"%s: decision_guidance required — TDD red phase: field does not exist yet", recipeID)
		assert.GreaterOrEqual(t, len(recipe.DecisionGuidance.Levels), 3,
			"%s: decision_guidance must have at least 3 severity levels", recipeID)

		// ── Budget ──────────────────────────────────────────────────────────────
		assert.Greater(t, recipe.Budget.MaxLinkDepth, 0,
			"%s: budget.max_link_depth must be positive", recipeID)
		assert.Greater(t, recipe.Budget.MaxObjects, 0,
			"%s: budget.max_objects must be positive", recipeID)
		assert.Greater(t, recipe.Budget.MaxTokensHint, 0,
			"%s: budget.max_tokens_hint must be positive", recipeID)

		// ── Governance ──────────────────────────────────────────────────────────
		assert.NotEmpty(t, recipe.Governance.Role,
			"%s: governance.role required", recipeID)
	}
}

func TestIntegration_LinkResolverFromV2Schema(t *testing.T) {
	path := configPath("aip_object_schema_v2.yml")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	objects, err := ParseObjectSchemaV2(data)
	require.NoError(t, err)

	resolver := NewLinkResolver(objects)
	require.NotNil(t, resolver)

	// Verify link resolution for seller.recent_orders
	opts := LinkOptions{
		MaxDepth: 1,
		Limit:    10,
	}

	result, err := resolver.GetLinkedObjects(nil, ObjectRef{
		ObjectType: "seller",
		ObjectID:   "test_seller_1",
	}, "recent_orders", opts)
	require.NoError(t, err)
	assert.Equal(t, "seller", result.ObjectType)
	assert.Equal(t, "test_seller_1", result.ObjectID)
	assert.Equal(t, "recent_orders", result.LinkName)
	assert.Equal(t, "order", result.TargetType)
	assert.Equal(t, "one_to_many", result.Cardinality)
}

func TestIntegration_CompileAllLinks(t *testing.T) {
	path := configPath("aip_object_schema_v2.yml")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	objects, err := ParseObjectSchemaV2(data)
	require.NoError(t, err)

	resolver := NewLinkResolver(objects)
	require.NotNil(t, resolver)

	plans, err := resolver.CompileAllLinks(nil, ObjectRef{
		ObjectType: "seller",
		ObjectID:   "test_seller_1",
	}, LinkOptions{Limit: 10})
	require.NoError(t, err)
	assert.Len(t, plans, 2, "seller should have 2 compiled links")

	for _, plan := range plans {
		assert.NotEmpty(t, plan.SQL)
		assert.NotEmpty(t, plan.LinkName)
		assert.NotEmpty(t, plan.TargetType)
	}
}

func TestIntegration_P2ObjectTypes(t *testing.T) {
	path := configPath("aip_object_schema_v2.yml")
	data, err := os.ReadFile(path)
	require.NoError(t, err, "should read v2 schema YAML")

	objects, err := ParseObjectSchemaV2(data)
	require.NoError(t, err, "should parse v2 schema")

	// Verify P2 object types exist in the schema.
	// These are TDD placeholder assertions — they will fail (red phase)
	// until corresponding YAML definitions are added to aip_object_schema_v2.yml.
	expectedTypes := []string{"review", "payment", "shipment", "marketing_lead"}
	for _, name := range expectedTypes {
		ot, ok := objects[name]
		assert.True(t, ok, "P2: expected object type %q (TDD red phase — YAML not yet added)", name)
		if ok {
			assert.NotEmpty(t, ot.Source.Schema, "P2 %s: source.schema required", name)
			assert.NotEmpty(t, ot.Source.Table, "P2 %s: source.table required", name)
			assert.NotEmpty(t, ot.Source.PrimaryKey, "P2 %s: source.primary_key required", name)
			assert.NotEmpty(t, ot.Properties, "P2 %s: properties required", name)
		}
	}
}
