package ontology

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllObjectTypes_ContainsExpectedTypes(t *testing.T) {
	types := AllObjectTypes()
	assert.Len(t, types, 12)
	assert.Contains(t, types, "customer")
	assert.Contains(t, types, "order")
	assert.Contains(t, types, "seller")
	assert.Contains(t, types, "product")
	assert.Contains(t, types, "category")
	assert.Contains(t, types, "region")
	assert.Contains(t, types, "marketing_lead")
	assert.Contains(t, types, "metric_alert")
	assert.Contains(t, types, "review")
	assert.Contains(t, types, "payment")
	assert.Contains(t, types, "shipment")
	assert.Contains(t, types, "global")
}

func TestObjectTypeDisplayName_KnownTypes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"customer", "客户"},
		{"order", "订单"},
		{"seller", "卖家"},
		{"product", "产品"},
		{"category", "品类"},
		{"region", "区域"},
		{"marketing_lead", "营销线索"},
		{"metric_alert", "异常事件"},
		{"review", "评价"},
		{"payment", "支付"},
		{"shipment", "物流"},
		{"global", "平台全局"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, ObjectTypeDisplayName(tt.input))
		})
	}
}

func TestObjectTypeDisplayName_UnknownType(t *testing.T) {
	assert.Equal(t, "unknown_type", ObjectTypeDisplayName("unknown_type"))
	assert.Equal(t, "", ObjectTypeDisplayName(""))
}

func TestKnownObjectType_ValidTypes(t *testing.T) {
	for _, name := range AllObjectTypes() {
		assert.True(t, KnownObjectType(name), "expected %q to be a known type", name)
	}
}

func TestKnownObjectType_InvalidTypes(t *testing.T) {
	assert.False(t, KnownObjectType(""))
	assert.False(t, KnownObjectType("user"))
	assert.False(t, KnownObjectType("Customer")) // case-sensitive
	assert.False(t, KnownObjectType("order "))   // trailing space
}
