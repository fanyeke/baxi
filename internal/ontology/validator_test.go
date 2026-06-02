package ontology

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// populatedRegistry creates a fully populated ObjectRegistry for testing.
func populatedRegistry() *ObjectRegistry {
	reg := &ObjectRegistry{
		objects: make(map[string]*ObjectType),
	}

	for _, name := range AllObjectTypes() {
		props := map[string]ObjectProperty{
			"id": {Name: "id", Type: "string", IsPK: true, Sensitivity: "L2"},
		}
		if name == "order" {
			props["amount"] = ObjectProperty{Name: "amount", Type: "float", Sensitivity: "L3"}
			props["status"] = ObjectProperty{Name: "status", Type: "string", Sensitivity: "L0"}
		}
		if name == "customer" {
			props["email"] = ObjectProperty{Name: "email", Type: "string", Sensitivity: "L3"}
		}

		links := []ObjectLink{}
		if name == "order" {
			links = []ObjectLink{
				{Name: "customer", TargetType: "customer", Via: "customer_id"},
				{Name: "seller", TargetType: "seller", Via: "seller_id"},
			}
		}
		if name == "product" {
			links = []ObjectLink{
				{Name: "category", TargetType: "category", Via: "category_id"},
			}
		}

		alertFields := []string{}
		if name == "metric_alert" {
			alertFields = []string{"id"}
		}

		reg.objects[name] = NewObjectType(
			name, "Display "+name, name+"_id", "id",
			props, links, []string{"read"}, defaultLLMAccess(), alertFields,
		)
		reg.objects[name].SourceTables = []string{name + "_table"}
	}
	return reg
}

func TestValidate_AllValid(t *testing.T) {
	reg := populatedRegistry()
	result := reg.Validate()
	assert.True(t, result.Valid)
	assert.Empty(t, result.Issues)
	assert.Contains(t, result.Summary, "valid")
}

func TestValidate_MissingObjectType(t *testing.T) {
	reg := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	reg.objects["order"] = populatedRegistry().objects["order"]

	result := reg.Validate()
	assert.False(t, result.Valid)
	assert.Greater(t, len(result.Issues), 1) // missing other 7 types + link errors
}

func TestValidate_MissingGrain(t *testing.T) {
	reg := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	ot := *populatedRegistry().objects["order"]
	ot.Grain = ""
	ot.SourceTables = []string{"t1"}
	ot.Properties = map[string]ObjectProperty{"id": {Name: "id", IsPK: true, Sensitivity: "L2"}}
	reg.objects["order"] = &ot

	result := reg.Validate()
	assert.False(t, result.Valid)
	assert.Condition(t, func() bool {
		for _, issue := range result.Issues {
			if issue.Message == "grain is empty" {
				return true
			}
		}
		return false
	})
}

func TestValidate_NoProperties(t *testing.T) {
	reg := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	reg.objects["order"] = NewObjectType("order", "Order", "id", "",
		map[string]ObjectProperty{}, nil, []string{"read"}, defaultLLMAccess(), nil)
	reg.objects["order"].SourceTables = []string{"t1"}

	result := reg.Validate()
	assert.False(t, result.Valid)

	// Should find either "no properties defined" or "no primary key"
	found := false
	for _, issue := range result.Issues {
		if issue.Message == "no properties defined" || issue.Message == "no primary key property (is_pk=true) found" {
			found = true
		}
	}
	assert.True(t, found, "should flag missing properties or primary key")
}

func TestValidate_AlertFieldRefersToNonexistentProperty(t *testing.T) {
	reg := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	reg.objects["metric_alert"] = NewObjectType("metric_alert", "Alert", "id", "id",
		map[string]ObjectProperty{"id": {Name: "id", IsPK: true, Sensitivity: "L2"}},
		nil, []string{"read"}, readWriteLLMAccess(), []string{"nonexistent_field"})
	reg.objects["metric_alert"].SourceTables = []string{"alerts"}

	result := reg.Validate()
	assert.False(t, result.Valid)

	found := false
	for _, issue := range result.Issues {
		if issue.ObjectType == "metric_alert" && issue.Severity == "error" {
			found = true
		}
	}
	assert.True(t, found)
}

func TestValidate_UnknownLinkTarget(t *testing.T) {
	reg := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	reg.objects["order"] = NewObjectType("order", "Order", "id", "id",
		map[string]ObjectProperty{"id": {Name: "id", IsPK: true, Sensitivity: "L2"}},
		[]ObjectLink{{Name: "bad_link", TargetType: "nonexistent_type", Via: "x_id"}},
		[]string{"read"}, defaultLLMAccess(), nil)
	reg.objects["order"].SourceTables = []string{"orders"}

	result := reg.Validate()
	assert.False(t, result.Valid)

	found := false
	for _, issue := range result.Issues {
		if issue.ObjectType == "order" && issue.Severity == "error" {
			found = true
		}
	}
	assert.True(t, found)
}

func TestValidate_MultiplePKs_Warning(t *testing.T) {
	reg := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	reg.objects["order"] = NewObjectType("order", "Order", "id", "",
		map[string]ObjectProperty{
			"id":   {Name: "id", IsPK: true, Sensitivity: "L2"},
			"id2":  {Name: "id2", IsPK: true, Sensitivity: "L2"},
			"name": {Name: "name", Sensitivity: "L0"},
		},
		nil, []string{"read"}, defaultLLMAccess(), nil)
	reg.objects["order"].SourceTables = []string{"orders"}

	result := reg.Validate()
	// Should have a warning about multiple PKs
	foundWarning := false
	for _, issue := range result.Issues {
		if issue.Severity == "warning" && issue.Message != "" {
			foundWarning = true
		}
	}
	assert.True(t, foundWarning, "should warn about multiple primary keys")
}

func TestValidationIssue_String(t *testing.T) {
	issue := ValidationIssue{ObjectType: "order", Severity: "error", Message: "test message"}
	assert.Equal(t, "[error] order: test message", issue.String())
}
