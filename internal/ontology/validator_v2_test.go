package ontology

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateV2_ValidObject(t *testing.T) {
	objects := map[string]*ObjectTypeV2{
		"seller": {
			Name:        "seller",
			DisplayName: "卖家",
			Grain:       "seller_id",
			Source: ObjectSource{
				Schema:     "dwd",
				Table:      "item_level",
				PrimaryKey: "seller_id",
			},
			Properties: map[string]ObjectPropertyV2{
				"seller_id": {Name: "seller_id", Type: "string", IsPK: true},
				"seller_state": {Name: "seller_state", Type: "string", Filterable: true},
			},
		},
	}

	issues := ValidateV2(objects)
	assert.Len(t, issues, 0, "valid v2 object should have no issues")
}

func TestValidateV2_MissingSource(t *testing.T) {
	objects := map[string]*ObjectTypeV2{
		"bad": {
			Name:  "bad",
			Grain: "id",
			Source: ObjectSource{
				Schema: "",
				Table:  "",
			},
			Properties: map[string]ObjectPropertyV2{
				"id": {Name: "id", Type: "string", IsPK: true},
			},
		},
	}

	issues := ValidateV2(objects)
	assert.Len(t, issues, 3, "should have errors for missing schema, table, primary_key")
}

func TestValidateV2_MissingPK(t *testing.T) {
	objects := map[string]*ObjectTypeV2{
		"nopk": {
			Name:  "nopk",
			Grain: "id",
			Source: ObjectSource{
				Schema: "dwd", Table: "test", PrimaryKey: "id",
			},
			Properties: map[string]ObjectPropertyV2{
				"id":   {Name: "id", Type: "string"},
				"name": {Name: "name", Type: "string"},
			},
		},
	}

	issues := ValidateV2(objects)
	assert.Len(t, issues, 1, "should have error for missing PK")
	assert.Contains(t, issues[0].Message, "no primary key")
}

func TestValidateV2_MultiplePK(t *testing.T) {
	objects := map[string]*ObjectTypeV2{
		"multi": {
			Name:  "multi",
			Grain: "id",
			Source: ObjectSource{
				Schema: "dwd", Table: "test", PrimaryKey: "id",
			},
			Properties: map[string]ObjectPropertyV2{
				"id1":  {Name: "id1", Type: "string", IsPK: true},
				"id2": {Name: "id2", Type: "string", IsPK: true},
			},
		},
	}

	issues := ValidateV2(objects)
	assert.Len(t, issues, 1, "should have error for multiple PKs")
	assert.Contains(t, issues[0].Message, "multiple primary key")
}

func TestValidateV2_NoProperties(t *testing.T) {
	objects := map[string]*ObjectTypeV2{
		"empty": {
			Name:  "empty",
			Grain: "id",
			Source: ObjectSource{
				Schema: "dwd", Table: "test", PrimaryKey: "id",
			},
			Properties: map[string]ObjectPropertyV2{},
		},
	}

	issues := ValidateV2(objects)
	assert.Len(t, issues, 2, "should have errors for missing PK and no properties")
}

func TestValidateV2_LinkTargetNotFound(t *testing.T) {
	objects := map[string]*ObjectTypeV2{
		"source": {
			Name:  "source",
			Grain: "id",
			Source: ObjectSource{
				Schema: "dwd", Table: "test", PrimaryKey: "id",
			},
			Properties: map[string]ObjectPropertyV2{
				"id": {Name: "id", Type: "string", IsPK: true},
			},
			Links: []ObjectLinkV2{
				{Name: "bad_link", TargetType: "nonexistent"},
			},
		},
	}

	issues := ValidateV2(objects)
	found := false
	for _, iss := range issues {
		if iss.Message == `link "bad_link" targets unknown object type "nonexistent"` {
			found = true
			break
		}
	}
	assert.True(t, found, "should have error for bad link target")
}

func TestValidateV2_RealFieldMissingSourceAndExpression(t *testing.T) {
	objects := map[string]*ObjectTypeV2{
		"item": {
			Name:     "item",
			Grain:    "id",
			Maturity: "stable",
			Source: ObjectSource{
				Schema: "dwd", Table: "test", PrimaryKey: "id",
			},
			Properties: map[string]ObjectPropertyV2{
				"id":   {Name: "id", Type: "string", IsPK: true, Availability: "real", SourceField: "id"},
				"bad":  {Name: "bad", Type: "string", Availability: "real"}, // no source_field or expression
			},
		},
	}

	issues := ValidateV2(objects)
	found := false
	for _, iss := range issues {
		if iss.Severity == "error" && iss.Message == `real property "bad" must have source_field or expression` {
			found = true
			break
		}
	}
	assert.True(t, found, "should have error for real field without source_field or expression")
}

func TestValidateV2_VirtualFieldNoMetricRefOrExpression(t *testing.T) {
	objects := map[string]*ObjectTypeV2{
		"item": {
			Name:     "item",
			Grain:    "id",
			Maturity: "stable",
			Source: ObjectSource{
				Schema: "dwd", Table: "test", PrimaryKey: "id",
			},
			Properties: map[string]ObjectPropertyV2{
				"id":     {Name: "id", Type: "string", IsPK: true, Availability: "real", SourceField: "id"},
				"vfield": {Name: "vfield", Type: "int", Availability: "virtual"}, // no metric_ref or expression
			},
		},
	}

	issues := ValidateV2(objects)
	found := false
	for _, iss := range issues {
		if iss.Severity == "warning" && iss.Message == `virtual property "vfield" has no metric_ref or expression` {
			found = true
			break
		}
	}
	assert.True(t, found, "should warn for virtual field without metric_ref or expression")
}

func TestValidateV2_PlannedFieldLLMReadable(t *testing.T) {
	objects := map[string]*ObjectTypeV2{
		"item": {
			Name:     "item",
			Grain:    "id",
			Maturity: "planned",
			Source: ObjectSource{
				Schema: "dwd", Table: "test", PrimaryKey: "id",
			},
			Properties: map[string]ObjectPropertyV2{
				"id":     {Name: "id", Type: "string", IsPK: true, Availability: "real", SourceField: "id"},
				"secret": {Name: "secret", Type: "string", Availability: "planned", LLMReadable: true},
			},
		},
	}

	issues := ValidateV2(objects)
	found := false
	for _, iss := range issues {
		if iss.Severity == "error" && iss.Message == `planned property "secret" must not have llm_readable=true` {
			found = true
			break
		}
	}
	assert.True(t, found, "should error for planned field with llm_readable=true")
}

func TestValidateV2_StableObjectNoRealFields(t *testing.T) {
	objects := map[string]*ObjectTypeV2{
		"item": {
			Name:     "item",
			Grain:    "id",
			Maturity: "stable",
			Source: ObjectSource{
				Schema: "dwd", Table: "test", PrimaryKey: "id",
			},
			Properties: map[string]ObjectPropertyV2{
				"id": {Name: "id", Type: "string", IsPK: true, Availability: "virtual", MetricRef: "count"},
			},
		},
	}

	issues := ValidateV2(objects)
	found := false
	for _, iss := range issues {
		if iss.Severity == "warning" && iss.Message == "stable object has no real fields (availability=real)" {
			found = true
			break
		}
	}
	assert.True(t, found, "should warn for stable object with no real fields")
}

func TestValidateV2FieldConsistency_ValidRealVirtualPlanned(t *testing.T) {
	objects := map[string]*ObjectTypeV2{
		"item": {
			Name:     "item",
			Grain:    "id",
			Maturity: "stable",
			Source: ObjectSource{
				Schema: "dwd", Table: "test", PrimaryKey: "id",
			},
			Properties: map[string]ObjectPropertyV2{
				"id":       {Name: "id", Type: "string", IsPK: true, Availability: "real", SourceField: "id"},
				"expr":     {Name: "expr", Type: "float", Availability: "real", Expression: "avg(score)"},
				"vmetric":  {Name: "vmetric", Type: "int", Availability: "virtual", MetricRef: "total_sales"},
				"vexpr":    {Name: "vexpr", Type: "int", Availability: "virtual", Expression: "sum(amt)"},
				"planning": {Name: "planning", Type: "string", Availability: "planned"},
			},
		},
	}

	issues := ValidateV2(objects)
	assert.Len(t, issues, 0, "valid mix of real/virtual/planned fields should have no issues")
}

func TestValidateV2FieldConsistency_RealFieldWithExpressionValid(t *testing.T) {
	// Real field with only Expression (no SourceField) is valid.
	objects := map[string]*ObjectTypeV2{
		"item": {
			Name:     "item",
			Grain:    "id",
			Maturity: "stable",
			Source: ObjectSource{
				Schema: "dwd", Table: "test", PrimaryKey: "id",
			},
			Properties: map[string]ObjectPropertyV2{
				"id":  {Name: "id", Type: "string", IsPK: true, Availability: "real", SourceField: "id"},
				"avg": {Name: "avg", Type: "float", Availability: "real", Expression: "AVG(score)"},
			},
		},
	}

	issues := ValidateV2(objects)
	for _, iss := range issues {
		assert.NotContains(t, iss.Message, "must have source_field or expression",
			"real field with expression should be valid")
	}
}

func TestValidateV2_PlannedFieldWithoutLLMReadableIsValid(t *testing.T) {
	objects := map[string]*ObjectTypeV2{
		"item": {
			Name:     "item",
			Grain:    "id",
			Maturity: "planned",
			Source: ObjectSource{
				Schema: "dwd", Table: "test", PrimaryKey: "id",
			},
			Properties: map[string]ObjectPropertyV2{
				"id":   {Name: "id", Type: "string", IsPK: true, Availability: "real", SourceField: "id"},
				"plan": {Name: "plan", Type: "string", Availability: "planned"}, // LLMReadable=false by default
			},
		},
	}

	issues := ValidateV2(objects)
	for _, iss := range issues {
		assert.NotContains(t, iss.Message, "must not have llm_readable=true",
			"planned field with llm_readable=false should be valid")
	}
}
