package ontology

import (
	"context"
	"errors"
	"testing"

	"baxi/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

// --- Mock ObjectQuerier for OntologyAwareAdapter tests ---

type mockObjQuerier struct {
	getObjectByIDFn      func(ctx context.Context, pool *pgxpool.Pool, objectType, objectID string) (*repository.ObjectInstance, error)
	queryByObjTypeFn     func(ctx context.Context, pool *pgxpool.Pool, objectType string, filters repository.ObjectFilters) (*repository.ObjectQueryResult, error)
	getObjectTypeSchemaFn func(ctx context.Context, objectType string) (*repository.ObjectSchema, error)
}

func (m *mockObjQuerier) GetObjectByID(ctx context.Context, pool *pgxpool.Pool, objectType, objectID string) (*repository.ObjectInstance, error) {
	if m.getObjectByIDFn != nil {
		return m.getObjectByIDFn(ctx, pool, objectType, objectID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockObjQuerier) QueryByObjectType(ctx context.Context, pool *pgxpool.Pool, objectType string, filters repository.ObjectFilters) (*repository.ObjectQueryResult, error) {
	if m.queryByObjTypeFn != nil {
		return m.queryByObjTypeFn(ctx, pool, objectType, filters)
	}
	return nil, errors.New("not implemented")
}

func (m *mockObjQuerier) GetObjectTypeSchema(ctx context.Context, objectType string) (*repository.ObjectSchema, error) {
	if m.getObjectTypeSchemaFn != nil {
		return m.getObjectTypeSchemaFn(ctx, objectType)
	}
	return nil, errors.New("not implemented")
}

// --- Tests: OntologyAwareAdapter ---

func TestOntologyAwareAdapter_GetObjectByID_Success(t *testing.T) {
	registry := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	ot := NewObjectType("seller", "Seller", "seller_id", "id",
		map[string]ObjectProperty{"id": {Name: "id", IsPK: true}, "name": {Name: "name"}},
		nil, []string{"read"}, defaultLLMAccess(), nil)
	registry.objects["seller"] = ot

	querier := &mockObjQuerier{
		getObjectByIDFn: func(ctx context.Context, pool *pgxpool.Pool, objectType, objectID string) (*repository.ObjectInstance, error) {
			return &repository.ObjectInstance{
				ObjectType: objectType,
				ID:         objectID,
				Properties: map[string]interface{}{"name": "Test Seller", "secret": "hidden"},
			}, nil
		},
	}

	adapter := NewOntologyAwareAdapter(querier, registry)
	obj, err := adapter.GetObjectByID(context.Background(), nil, "seller", "seller-1")
	assert.NoError(t, err)
	assert.NotNil(t, obj)
	assert.Equal(t, "seller", obj.ObjectType)
	assert.Equal(t, "seller-1", obj.ID)
	assert.Equal(t, "Test Seller", obj.Properties["name"])
	// "secret" should be filtered out since it's not in the ontology
	_, hasSecret := obj.Properties["secret"]
	assert.False(t, hasSecret, "properties not defined in ontology should be filtered")
}

func TestOntologyAwareAdapter_GetObjectByID_Error(t *testing.T) {
	registry := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	ot := NewObjectType("seller", "Seller", "seller_id", "id",
		map[string]ObjectProperty{"id": {Name: "id"}},
		nil, nil, defaultLLMAccess(), nil)
	registry.objects["seller"] = ot

	querier := &mockObjQuerier{
		getObjectByIDFn: func(ctx context.Context, pool *pgxpool.Pool, objectType, objectID string) (*repository.ObjectInstance, error) {
			return nil, errors.New("not found")
		},
	}

	adapter := NewOntologyAwareAdapter(querier, registry)
	obj, err := adapter.GetObjectByID(context.Background(), nil, "seller", "missing")
	assert.Error(t, err)
	assert.Nil(t, obj)
}

func TestOntologyAwareAdapter_GetObjectByID_UnknownType(t *testing.T) {
	registry := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	querier := &mockObjQuerier{}
	adapter := NewOntologyAwareAdapter(querier, registry)

	obj, err := adapter.GetObjectByID(context.Background(), nil, "unknown_type", "id-1")
	assert.Error(t, err)
	assert.Nil(t, obj)
	assert.Contains(t, err.Error(), "ontology schema lookup")
}

func TestOntologyAwareAdapter_GetObjectTypeSchema_Success(t *testing.T) {
	registry := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	ot := NewObjectType("seller", "Seller", "seller_id", "id",
		map[string]ObjectProperty{
			"id":   {Name: "id", Type: "string", IsPK: true},
			"name": {Name: "name", Type: "string"},
		},
		nil, []string{"read"}, defaultLLMAccess(), nil)
	registry.objects["seller"] = ot

	querier := &mockObjQuerier{}
	adapter := NewOntologyAwareAdapter(querier, registry)

	schema, err := adapter.GetObjectTypeSchema(context.Background(), "seller")
	assert.NoError(t, err)
	assert.NotNil(t, schema)
	assert.Equal(t, "seller", schema.Name)
	assert.Equal(t, "Seller", schema.DisplayName)
	assert.Contains(t, schema.Properties, "id")
	assert.Contains(t, schema.Properties, "name")
	assert.True(t, schema.Properties["id"].IsPK)
}

func TestOntologyAwareAdapter_GetObjectTypeSchema_UnknownType(t *testing.T) {
	registry := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	querier := &mockObjQuerier{}
	adapter := NewOntologyAwareAdapter(querier, registry)

	schema, err := adapter.GetObjectTypeSchema(context.Background(), "unknown")
	assert.Error(t, err)
	assert.Nil(t, schema)
	assert.Contains(t, err.Error(), "ontology schema")
}

func TestOntologyAwareAdapter_QueryByObjectType_Success(t *testing.T) {
	registry := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	ot := NewObjectType("seller", "Seller", "seller_id", "id",
		map[string]ObjectProperty{"id": {Name: "id"}, "name": {Name: "name"}},
		nil, nil, defaultLLMAccess(), nil)
	registry.objects["seller"] = ot

	querier := &mockObjQuerier{
		queryByObjTypeFn: func(ctx context.Context, pool *pgxpool.Pool, objectType string, filters repository.ObjectFilters) (*repository.ObjectQueryResult, error) {
			return &repository.ObjectQueryResult{
				Rows: []repository.ObjectInstance{
					{ObjectType: "seller", ID: "s1", Properties: map[string]interface{}{"name": "A", "hidden": "x"}},
				},
				Total: 1,
			}, nil
		},
	}

	adapter := NewOntologyAwareAdapter(querier, registry)
	result, err := adapter.QueryByObjectType(context.Background(), nil, "seller", repository.ObjectFilters{
		ObjectType: "seller",
		Limit:      10,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Rows, 1)
	assert.Equal(t, "A", result.Rows[0].Properties["name"])
	_, hasHidden := result.Rows[0].Properties["hidden"]
	assert.False(t, hasHidden, "hidden should be filtered")
}

func TestOntologyAwareAdapter_QueryByObjectType_Error(t *testing.T) {
	registry := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	ot := NewObjectType("seller", "Seller", "seller_id", "id",
		map[string]ObjectProperty{"id": {Name: "id"}},
		nil, nil, defaultLLMAccess(), nil)
	registry.objects["seller"] = ot

	querier := &mockObjQuerier{
		queryByObjTypeFn: func(ctx context.Context, pool *pgxpool.Pool, objectType string, filters repository.ObjectFilters) (*repository.ObjectQueryResult, error) {
			return nil, errors.New("db error")
		},
	}

	adapter := NewOntologyAwareAdapter(querier, registry)
	result, err := adapter.QueryByObjectType(context.Background(), nil, "seller", repository.ObjectFilters{})
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestOntologyAwareAdapter_QueryByObjectType_UnknownType(t *testing.T) {
	registry := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	querier := &mockObjQuerier{}
	adapter := NewOntologyAwareAdapter(querier, registry)

	result, err := adapter.QueryByObjectType(context.Background(), nil, "unknown", repository.ObjectFilters{})
	assert.Error(t, err)
	assert.Nil(t, result)
}

// --- Tests: ObjectRegistry edge cases ---

func TestObjectRegistry_GetObjectType_NotFound_Extra(t *testing.T) {
	reg := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	_, err := reg.GetObjectType("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestObjectRegistry_GetObjectType_Found_Extra(t *testing.T) {
	reg := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	ot := NewObjectType("seller", "Seller", "seller_id", "id", nil, nil, nil, defaultLLMAccess(), nil)
	reg.objects["seller"] = ot

	result, err := reg.GetObjectType("seller")
	assert.NoError(t, err)
	assert.Equal(t, "seller", result.Name)
}

func TestObjectRegistry_GetProperties_NotFound_Extra(t *testing.T) {
	reg := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	props, err := reg.GetProperties("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, props)
}

func TestObjectRegistry_GetProperties_Found_Extra(t *testing.T) {
	reg := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	ot := NewObjectType("seller", "Seller", "seller_id", "id",
		map[string]ObjectProperty{"id": {Name: "id", IsPK: true}, "name": {Name: "name"}},
		nil, nil, defaultLLMAccess(), nil)
	reg.objects["seller"] = ot

	props, err := reg.GetProperties("seller")
	assert.NoError(t, err)
	assert.Len(t, props, 2)
	assert.Contains(t, props, "id")
	assert.Contains(t, props, "name")
}

func TestObjectRegistry_GetLinks_NotFound_Extra(t *testing.T) {
	reg := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	links, err := reg.GetLinks("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, links)
}

func TestObjectRegistry_GetLinks_Found_Extra(t *testing.T) {
	reg := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	ot := NewObjectType("order", "Order", "order_id", "id",
		map[string]ObjectProperty{"id": {Name: "id"}},
		[]ObjectLink{{Name: "seller", TargetType: "seller", Via: "seller_id"}},
		nil, defaultLLMAccess(), nil)
	reg.objects["order"] = ot

	links, err := reg.GetLinks("order")
	assert.NoError(t, err)
	assert.Len(t, links, 1)
	assert.Equal(t, "seller", links[0].Name)
}

func TestObjectRegistry_IsLLMReadable_NotFound_Extra(t *testing.T) {
	reg := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	assert.False(t, reg.IsLLMReadable("nonexistent", "field"))
}

func TestObjectRegistry_IsLLMReadable_ReadWriteAccess_Extra(t *testing.T) {
	reg := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	ot := NewObjectType("alert", "Alert", "alert_id", "id",
		map[string]ObjectProperty{"status": {Name: "status", LLMReadable: true}},
		nil, nil, readWriteLLMAccess(), nil)
	reg.objects["alert"] = ot

	assert.True(t, reg.IsLLMReadable("alert", "status"))
	assert.False(t, reg.IsLLMReadable("alert", "unknown_field"))
}

func TestObjectRegistry_ListObjectTypes_Empty(t *testing.T) {
	reg := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	types := reg.ListObjectTypes()
	assert.Empty(t, types)
}

func TestObjectRegistry_ListObjectTypes_Multiple(t *testing.T) {
	reg := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	reg.objects["order"] = NewObjectType("order", "Order", "id", "id", nil, nil, nil, defaultLLMAccess(), nil)
	reg.objects["seller"] = NewObjectType("seller", "Seller", "id", "id", nil, nil, nil, defaultLLMAccess(), nil)
	types := reg.ListObjectTypes()
	assert.Len(t, types, 2)
	assert.Contains(t, types, "order")
	assert.Contains(t, types, "seller")
}

func TestObjectRegistry_Validate_EmptyRegistry_Extra(t *testing.T) {
	reg := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	result := reg.Validate()
	assert.False(t, result.Valid)
}

func TestObjectRegistry_Validate_Valid(t *testing.T) {
	reg := &ObjectRegistry{objects: make(map[string]*ObjectType)}
	// Add all required object types to pass validation
	for _, name := range AllObjectTypes() {
		ot := NewObjectType(name, name, "id", "id",
			map[string]ObjectProperty{"id": {Name: "id", IsPK: true, Sensitivity: "L2"}},
			nil, []string{"read"}, defaultLLMAccess(), nil)
		ot.SourceTables = []string{"src_" + name}
		reg.objects[name] = ot
	}

	result := reg.Validate()
	assert.True(t, result.Valid)
}

// --- Tests: ValidationIssue ---

func TestValidationIssue_String_Error_Extra(t *testing.T) {
	issue := ValidationIssue{ObjectType: "order", Severity: "error", Message: "missing pk"}
	assert.Equal(t, "[error] order: missing pk", issue.String())
}

func TestValidationIssue_String_Info_Extra(t *testing.T) {
	issue := ValidationIssue{ObjectType: "order", Severity: "info", Message: "ok"}
	assert.Equal(t, "[info] order: ok", issue.String())
}

// --- Tests: AllObjectTypes ---

func TestAllObjectTypes_Extra(t *testing.T) {
	types := AllObjectTypes()
	assert.NotEmpty(t, types)
	assert.Contains(t, types, TypeOrder)
	assert.Contains(t, types, TypeSeller)
	assert.Contains(t, types, TypeMetricAlert)
}

func TestKnownObjectType_All_Extra(t *testing.T) {
	for _, name := range AllObjectTypes() {
		assert.True(t, KnownObjectType(name), "expected %s to be known", name)
	}
}

func TestKnownObjectType_Unknown_Extra(t *testing.T) {
	assert.False(t, KnownObjectType("unknown_type"))
}

// --- Tests: ObjectTypeDisplayName ---

func TestObjectTypeDisplayName_All_Extra(t *testing.T) {
	for _, name := range AllObjectTypes() {
		display := ObjectTypeDisplayName(name)
		assert.NotEmpty(t, display)
	}
}

func TestObjectTypeDisplayName_Unknown_Extra(t *testing.T) {
	assert.Equal(t, "custom", ObjectTypeDisplayName("custom"))
}
