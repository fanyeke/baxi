package ontology

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewObjectType_Basic(t *testing.T) {
	props := map[string]ObjectProperty{
		"id":   {Name: "id", Type: "string", IsPK: true, Sensitivity: "L2"},
		"name": {Name: "name", Type: "string", Sensitivity: "L0"},
	}
	links := []ObjectLink{
		{Name: "seller", TargetType: "seller", Via: "seller_id"},
	}
	ot := NewObjectType("order", "订单", "order_id", "id", props, links, []string{"read"}, defaultLLMAccess(), []string{})

	assert.Equal(t, "order", ot.Name)
	assert.Equal(t, "订单", ot.DisplayName)
	assert.Equal(t, "order_id", ot.Grain)
	assert.Equal(t, "id", ot.PrimaryKey)
	assert.Equal(t, props, ot.Properties)
	assert.Equal(t, links, ot.Links)
	assert.Equal(t, []string{"read"}, ot.AllowedActions)
	assert.True(t, ot.LLMAccess.CanRead)
	assert.False(t, ot.LLMAccess.CanWrite)
	assert.True(t, ot.LLMAccess.ReadOnly)
}

func TestNewObjectType_NilSlices(t *testing.T) {
	ot := NewObjectType("test", "", "", "", nil, nil, nil, LLMAccessPolicy{}, nil)
	assert.NotNil(t, ot.Properties)
	assert.NotNil(t, ot.Links)
	assert.NotNil(t, ot.AllowedActions)
	assert.NotNil(t, ot.AlertFields)
	assert.Empty(t, ot.Properties)
	assert.Empty(t, ot.Links)
	assert.Empty(t, ot.AllowedActions)
	assert.Empty(t, ot.AlertFields)
}

func TestNewObjectType_ReadWriteAccess(t *testing.T) {
	ot := NewObjectType("metric_alert", "异常事件", "alert_id", "id",
		map[string]ObjectProperty{"id": {Name: "id", IsPK: true}},
		nil, nil, readWriteLLMAccess(), nil)

	assert.True(t, ot.LLMAccess.CanRead)
	assert.True(t, ot.LLMAccess.CanWrite)
	assert.False(t, ot.LLMAccess.ReadOnly)
}

func TestDefaultLLMAccess(t *testing.T) {
	policy := defaultLLMAccess()
	assert.True(t, policy.CanRead)
	assert.False(t, policy.CanWrite)
	assert.True(t, policy.ReadOnly)
}

func TestReadWriteLLMAccess(t *testing.T) {
	policy := readWriteLLMAccess()
	assert.True(t, policy.CanRead)
	assert.True(t, policy.CanWrite)
	assert.False(t, policy.ReadOnly)
}

func TestDefaultSensitivity(t *testing.T) {
	assert.Equal(t, "L2", defaultSensitivity(true))
	assert.Equal(t, "L0", defaultSensitivity(false))
}
