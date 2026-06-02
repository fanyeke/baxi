package ontology

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetObjectType_Known(t *testing.T) {
	reg := populatedRegistry()
	ot, err := reg.GetObjectType("order")
	assert.NoError(t, err)
	assert.NotNil(t, ot)
	assert.Equal(t, "order", ot.Name)
}

func TestGetObjectType_Unknown(t *testing.T) {
	reg := populatedRegistry()
	ot, err := reg.GetObjectType("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, ot)
}

func TestListObjectTypes_ReturnsRegistered(t *testing.T) {
	reg := populatedRegistry()
	types := reg.ListObjectTypes()
	assert.Len(t, types, 12)
	assert.Equal(t, "customer", types[0])
	assert.Equal(t, "global", types[11])
}

func TestListObjectTypes_Partial(t *testing.T) {
	reg := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	reg.objects["order"] = populatedRegistry().objects["order"]
	reg.objects["customer"] = populatedRegistry().objects["customer"]
	types := reg.ListObjectTypes()
	assert.Len(t, types, 2)
	assert.Contains(t, types, "order")
	assert.Contains(t, types, "customer")
}

func TestGetProperties_Known(t *testing.T) {
	reg := populatedRegistry()
	props, err := reg.GetProperties("order")
	assert.NoError(t, err)
	assert.NotNil(t, props)
	assert.Contains(t, props, "id")
	assert.Contains(t, props, "amount")
	assert.Contains(t, props, "status")
}

func TestGetProperties_Unknown(t *testing.T) {
	reg := populatedRegistry()
	props, err := reg.GetProperties("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, props)
}

func TestGetLinks_Known(t *testing.T) {
	reg := populatedRegistry()
	links, err := reg.GetLinks("order")
	assert.NoError(t, err)
	assert.Len(t, links, 2)
}

func TestGetLinks_Unknown(t *testing.T) {
	reg := populatedRegistry()
	links, err := reg.GetLinks("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, links)
}

func TestGetAllowedActions_Known(t *testing.T) {
	reg := populatedRegistry()
	actions := reg.GetAllowedActions("order")
	assert.ElementsMatch(t, []string{"read"}, actions)
}

func TestGetAllowedActions_Unknown(t *testing.T) {
	reg := populatedRegistry()
	actions := reg.GetAllowedActions("nonexistent")
	assert.Nil(t, actions)
}

func TestIsLLMReadable_PKFieldNotReadable(t *testing.T) {
	reg := populatedRegistry()
	assert.False(t, reg.IsLLMReadable("order", "id")) // PK, not LLM-readable
}

func TestIsLLMReadable_UnknownObjectType(t *testing.T) {
	reg := populatedRegistry()
	assert.False(t, reg.IsLLMReadable("nonexistent", "id"))
}

func TestIsLLMReadable_UnknownProperty(t *testing.T) {
	reg := populatedRegistry()
	assert.False(t, reg.IsLLMReadable("order", "nonexistent"))
}

func TestGetSourceDataset_Known(t *testing.T) {
	reg := populatedRegistry()
	dataset := reg.GetSourceDataset("order")
	assert.Equal(t, "order_table", dataset)
}

func TestGetSourceDataset_Unknown(t *testing.T) {
	reg := populatedRegistry()
	dataset := reg.GetSourceDataset("nonexistent")
	assert.Equal(t, "", dataset)
}

func TestGetSourceDataset_NoTables(t *testing.T) {
	reg := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	reg.objects["empty"] = NewObjectType("empty", "", "", "", nil, nil, nil, LLMAccessPolicy{}, nil)
	dataset := reg.GetSourceDataset("empty")
	assert.Equal(t, "", dataset)
}

func TestConvertRawObject_BasicOrder(t *testing.T) {
	raw := rawObjectType{
		ObjectTypeID: "test_type",
		DisplayName:  "Test Type",
		SourceTables: []string{"test_table"},
		Grain:        "test_id",
		Properties: map[string]rawObjectProperty{
			"id":   {Type: "string", IsPK: boolPtr(true)},
			"name": {Type: "string"},
		},
		Relationships: map[string]rawObjectRelationship{
			"parent": {To: "parent_type", Grain: "parent_id"},
		},
	}

	ot, err := convertRawObject(raw)
	assert.NoError(t, err)
	assert.Equal(t, "test_type", ot.Name)
	assert.Equal(t, "Test Type", ot.DisplayName)
	assert.Equal(t, "test_id", ot.Grain)
	assert.Equal(t, "id", ot.PrimaryKey)
	assert.Equal(t, []string{"test_table"}, ot.SourceTables)
	assert.Len(t, ot.Links, 1)
	assert.Equal(t, "parent_type", ot.Links[0].TargetType)
	assert.True(t, ot.Links[0].Name == "parent")
	assert.Contains(t, ot.Properties, "id")
	assert.Contains(t, ot.Properties, "name")
	assert.True(t, ot.Properties["id"].IsPK)
	assert.Equal(t, "L2", ot.Properties["id"].Sensitivity) // PK gets L2
	assert.Equal(t, "L0", ot.Properties["name"].Sensitivity)
	assert.ElementsMatch(t, []string{"read"}, ot.AllowedActions)
}

func TestConvertRawObject_MetricAlertHasReadWriteAccess(t *testing.T) {
	raw := rawObjectType{
		ObjectTypeID: "metric_alert",
		DisplayName:  "Alert",
		Properties:   map[string]rawObjectProperty{"id": {Type: "string", IsPK: boolPtr(true)}},
	}

	ot, err := convertRawObject(raw)
	assert.NoError(t, err)
	assert.True(t, ot.LLMAccess.CanWrite)
	assert.False(t, ot.LLMAccess.ReadOnly)
}

func TestNewObjectRegistry_WithValidYAML(t *testing.T) {
	reg, err := NewObjectRegistry(context.Background(), nil, nil, "testdata/object_schema.yml")
	assert.NoError(t, err)
	assert.NotNil(t, reg)

	types := reg.ListObjectTypes()
	assert.Len(t, types, 1)

	order, err := reg.GetObjectType("order")
	assert.NoError(t, err)
	assert.NotNil(t, order)
	assert.NotEmpty(t, order.Properties)
}

func TestNewObjectRegistry_InvalidYAMLPath(t *testing.T) {
	reg, err := NewObjectRegistry(context.Background(), nil, nil, "/nonexistent/path.yml")
	assert.Error(t, err)
	assert.Nil(t, reg)
}

func boolPtr(b bool) *bool {
	return &b
}
