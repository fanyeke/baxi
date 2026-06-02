package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ──── parseFieldPath ────────────────────────────────────────────────────────

func TestParseFieldPath_SimpleField(t *testing.T) {
	obj, field := parseFieldPath("customer.name")
	assert.Equal(t, "customer", obj)
	assert.Equal(t, "name", field)
}

func TestParseFieldPath_DeepPath(t *testing.T) {
	obj, field := parseFieldPath("order.items.price")
	assert.Equal(t, "order.items", obj)
	assert.Equal(t, "price", field)
}

func TestParseFieldPath_NoDot(t *testing.T) {
	obj, field := parseFieldPath("customer")
	assert.Equal(t, "customer", obj)
	assert.Equal(t, "*", field)
}

func TestParseFieldPath_Empty(t *testing.T) {
	obj, field := parseFieldPath("")
	assert.Equal(t, "", obj)
	assert.Equal(t, "*", field)
}

// ──── inferSourceDataset ────────────────────────────────────────────────────

func TestInferSourceDataset_KnownTypes(t *testing.T) {
	assert.Equal(t, "olist_customers", inferSourceDataset("customer"))
	assert.Equal(t, "olist_orders", inferSourceDataset("order"))
	assert.Equal(t, "olist_products", inferSourceDataset("product"))
	assert.Equal(t, "olist_sellers", inferSourceDataset("seller"))
	assert.Equal(t, "olist_geolocation", inferSourceDataset("geolocation"))
}

func TestInferSourceDataset_Fallback(t *testing.T) {
	assert.Equal(t, "unknown_type", inferSourceDataset("unknown_type"))
	assert.Equal(t, "order_items", inferSourceDataset("order_items"))
}

// ──── inferPrimaryKey ───────────────────────────────────────────────────────

func TestInferPrimaryKey_KnownTypes(t *testing.T) {
	assert.Equal(t, "customer_id", inferPrimaryKey("customer"))
	assert.Equal(t, "order_id", inferPrimaryKey("order"))
	assert.Equal(t, "product_id", inferPrimaryKey("product"))
	assert.Equal(t, "seller_id", inferPrimaryKey("seller"))
	assert.Equal(t, "zip_code_prefix", inferPrimaryKey("geolocation"))
}

func TestInferPrimaryKey_Fallback(t *testing.T) {
	assert.Equal(t, "id", inferPrimaryKey("unknown_type"))
	assert.Equal(t, "id", inferPrimaryKey("order_items"))
}

// ──── splitDataset ──────────────────────────────────────────────────────────

func TestSplitDataset(t *testing.T) {
	schema, table := splitDataset("olist_orders")
	assert.Equal(t, "public", schema)
	assert.Equal(t, "olist_orders", table)
}

func TestSplitDataset_Empty(t *testing.T) {
	schema, table := splitDataset("")
	assert.Equal(t, "public", schema)
	assert.Equal(t, "", table)
}
