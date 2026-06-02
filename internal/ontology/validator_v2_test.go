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
