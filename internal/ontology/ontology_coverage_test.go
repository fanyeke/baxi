package ontology

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ──── resolveLimit tests are in query_service_test.go ─────────────────────

// ──── getRole tests are in query_service_test.go ─────────────────────────

// ──── NewObjectType ────────────────────────────────────────────────────────

func TestNewObjectType_NilProperties(t *testing.T) {
	ot := NewObjectType("test", "Test", "grain", "pk", nil, nil, nil, LLMAccessPolicy{}, nil)
	assert.NotNil(t, ot.Properties)
	assert.Empty(t, ot.Properties)
	assert.NotNil(t, ot.Links)
	assert.Empty(t, ot.Links)
	assert.NotNil(t, ot.AllowedActions)
	assert.Empty(t, ot.AllowedActions)
	assert.NotNil(t, ot.AlertFields)
	assert.Empty(t, ot.AlertFields)
}

// ──── loadFromYAML edge cases ──────────────────────────────────────────────

func TestLoadFromYAML_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	yamlPath := dir + "/invalid.yml"
	// Write invalid YAML
	err := writeTestFile(yamlPath, "not: [valid: yaml: content:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = loadFromYAML(yamlPath)
	assert.Error(t, err)
}

func TestLoadFromYAML_EmptyObjects(t *testing.T) {
	dir := t.TempDir()
	yamlPath := dir + "/empty.yml"
	err := writeTestFile(yamlPath, "objects: []\n")
	if err != nil {
		t.Fatal(err)
	}
	objects, err := loadFromYAML(yamlPath)
	assert.NoError(t, err)
	assert.Empty(t, objects)
}

func TestLoadFromYAML_NonexistentFile(t *testing.T) {
	_, err := loadFromYAML("/nonexistent/path/file.yml")
	assert.Error(t, err)
}

func TestLoadFromYAML_WithAlertFields(t *testing.T) {
	dir := t.TempDir()
	yamlPath := dir + "/alert_fields.yml"
	yaml := `
objects:
  - object_type_id: test_alert
    display_name: Test Alert
    grain: id
    source_tables: [alerts]
    properties:
      id:
        type: string
        is_pk: true
      status:
        type: string
    alert_fields:
      - status
`
	err := writeTestFile(yamlPath, yaml)
	if err != nil {
		t.Fatal(err)
	}
	objects, err := loadFromYAML(yamlPath)
	assert.NoError(t, err)
	assert.Len(t, objects, 1)
	ot := objects["test_alert"]
	assert.NotNil(t, ot)
	assert.Equal(t, []string{"status"}, ot.AlertFields)
}

// ──── ObjectRegistry edge cases ────────────────────────────────────────────

func TestNewObjectRegistry_NoSource(t *testing.T) {
	_, err := NewObjectRegistry(context.Background(), nil, nil, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no schema source available")
}

func TestObjectRegistry_CustomObjectType_Extra(t *testing.T) {
	reg := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	reg.objects["order"] = NewObjectType("order", "Order", "order_id", "id",
		map[string]ObjectProperty{"id": {Name: "id", IsPK: true, Sensitivity: "L2"}},
		nil, []string{"read"}, defaultLLMAccess(), nil)

	types := reg.ListObjectTypes()
	assert.Contains(t, types, "order")
	assert.Len(t, types, 1)
}

func TestObjectRegistry_IsLLMReadable_ReadableField(t *testing.T) {
	reg := populatedRegistry()
	// status is not PK, but it's not explicitly marked LLMReadable in populatedRegistry
	// So it should be false by default
	assert.False(t, reg.IsLLMReadable("order", "status"))
	// PK fields should not be LLM-readable
	assert.False(t, reg.IsLLMReadable("order", "id"))
}

func TestObjectRegistry_GetProperties_WithLinks(t *testing.T) {
	reg := populatedRegistry()
	links, err := reg.GetLinks("product")
	assert.NoError(t, err)
	assert.NotEmpty(t, links)
}

func TestObjectRegistry_GetLinks_NoLinks(t *testing.T) {
	reg := populatedRegistry()
	// category has no links
	links, err := reg.GetLinks("category")
	assert.NoError(t, err)
	assert.Empty(t, links)
}

// ──── OntologyAwareAdapter ────────────────────────────────────────────────

func TestNewOntologyAwareAdapter(t *testing.T) {
	reg := populatedRegistry()
	adapter := NewOntologyAwareAdapter(nil, reg)
	assert.NotNil(t, adapter)
}

// ──── Validate edge cases ──────────────────────────────────────────────────

func TestValidate_EmptyRegistry(t *testing.T) {
	reg := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	result := reg.Validate()
	assert.False(t, result.Valid)
	assert.Contains(t, result.Summary, "validation error")
}

func TestValidate_NoSourceTables(t *testing.T) {
	reg := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	ot := NewObjectType("order", "Order", "order_id", "id",
		map[string]ObjectProperty{"id": {Name: "id", IsPK: true, Sensitivity: "L2"}},
		nil, []string{"read"}, defaultLLMAccess(), nil)
	// SourceTables is empty
	reg.objects["order"] = ot

	result := reg.Validate()
	assert.False(t, result.Valid)
	found := false
	for _, issue := range result.Issues {
		if issue.Message == "no source_tables defined" {
			found = true
		}
	}
	assert.True(t, found, "should flag missing source tables")
}

func TestValidate_EmptyDisplayName(t *testing.T) {
	reg := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	ot := NewObjectType("order", "", "order_id", "id",
		map[string]ObjectProperty{"id": {Name: "id", IsPK: true, Sensitivity: "L2"}},
		nil, []string{"read"}, defaultLLMAccess(), nil)
	ot.SourceTables = []string{"orders"}
	reg.objects["order"] = ot

	result := reg.Validate()
	// Should have a warning about empty display_name
	found := false
	for _, issue := range result.Issues {
		if issue.Severity == "warning" && issue.Message == "display_name is empty" {
			found = true
		}
	}
	assert.True(t, found, "should warn about empty display_name")
}

func TestValidationIssue_String_Warning(t *testing.T) {
	issue := ValidationIssue{ObjectType: "order", Severity: "warning", Message: "minor issue"}
	assert.Equal(t, "[warning] order: minor issue", issue.String())
}

// ──── YAML conversion helpers ──────────────────────────────────────────────

func TestDefaultSensitivity_PK(t *testing.T) {
	assert.Equal(t, "L2", defaultSensitivity(true))
}

func TestDefaultSensitivity_NonPK(t *testing.T) {
	assert.Equal(t, "L0", defaultSensitivity(false))
}

func TestAllObjectTypes(t *testing.T) {
	types := AllObjectTypes()
	assert.Len(t, types, 12)
	assert.Contains(t, types, TypeCustomer)
	assert.Contains(t, types, TypeMetricAlert)
}

func TestObjectTypeDisplayName_AllTypes(t *testing.T) {
	assert.Equal(t, "客户", ObjectTypeDisplayName(TypeCustomer))
	assert.Equal(t, "订单", ObjectTypeDisplayName(TypeOrder))
	assert.Equal(t, "卖家", ObjectTypeDisplayName(TypeSeller))
	assert.Equal(t, "产品", ObjectTypeDisplayName(TypeProduct))
	assert.Equal(t, "品类", ObjectTypeDisplayName(TypeCategory))
	assert.Equal(t, "区域", ObjectTypeDisplayName(TypeRegion))
	assert.Equal(t, "营销线索", ObjectTypeDisplayName(TypeMarketingLead))
	assert.Equal(t, "异常事件", ObjectTypeDisplayName(TypeMetricAlert))
	assert.Equal(t, "评价", ObjectTypeDisplayName(TypeReview))
	assert.Equal(t, "支付", ObjectTypeDisplayName(TypePayment))
	assert.Equal(t, "物流", ObjectTypeDisplayName(TypeShipment))
	assert.Equal(t, "平台全局", ObjectTypeDisplayName(TypeGlobal))
}

func TestObjectTypeDisplayName_Unknown(t *testing.T) {
	assert.Equal(t, "custom", ObjectTypeDisplayName("custom"))
}

func TestKnownObjectType_All(t *testing.T) {
	for _, name := range AllObjectTypes() {
		assert.True(t, KnownObjectType(name), "expected %s to be known", name)
	}
}

func TestKnownObjectType_Unknown(t *testing.T) {
	assert.False(t, KnownObjectType("unknown_type"))
}

func writeTestFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
