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
	expectedTypes := []string{"seller", "order", "product", "metric_alert"}
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

	// Verify expected metrics exist
	expectedMetrics := []string{
		"seller_late_delivery_rate_7d",
		"seller_order_count_7d",
		"seller_gmv_7d",
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
