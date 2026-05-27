package ontology

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/repository"
)

type mockQuerier struct {
	getObj    *repository.ObjectInstance
	getErr    error
	queryResult *repository.ObjectQueryResult
	queryErr    error
}

func (m *mockQuerier) GetObjectByID(_ context.Context, _ *pgxpool.Pool, _, _ string) (*repository.ObjectInstance, error) {
	return m.getObj, m.getErr
}

func (m *mockQuerier) QueryByObjectType(_ context.Context, _ *pgxpool.Pool, _ string, _ repository.ObjectFilters) (*repository.ObjectQueryResult, error) {
	return m.queryResult, m.queryErr
}

type mockRegistry struct {
	types map[string]*ObjectType
}

func (m *mockRegistry) GetObjectType(name string) (*ObjectType, error) {
	ot, ok := m.types[name]
	if !ok {
		return nil, errors.New("unknown object type: " + name)
	}
	return ot, nil
}

func newTestRegistry() *mockRegistry {
	return &mockRegistry{
		types: map[string]*ObjectType{
			"customer": {
				Name:       "customer",
				DisplayName: "Customer",
				Grain:      "customer_unique_id",
				PrimaryKey: "customer_unique_id",
				Properties: map[string]ObjectProperty{
					"customer_unique_id": {Name: "customer_unique_id", Type: "string", IsPK: true},
					"customer_state":     {Name: "customer_state", Type: "string"},
					"payment_value":      {Name: "payment_value", Type: "float"},
				},
			},
			"order": {
				Name:       "order",
				DisplayName: "Order",
				Grain:      "order_id",
				PrimaryKey: "order_id",
				Properties: map[string]ObjectProperty{
					"order_id":     {Name: "order_id", Type: "string", IsPK: true},
					"order_status": {Name: "order_status", Type: "string"},
				},
			},
		},
	}
}

func TestAdapter_GetObjectByID_FiltersProperties(t *testing.T) {
	registry := newTestRegistry()
	querier := &mockQuerier{
		getObj: &repository.ObjectInstance{
			ObjectType: "customer",
			ID:         "cust-1",
			Properties: map[string]any{
				"customer_unique_id": "cust-1",
				"customer_state":     "SP",
				"payment_value":      150.0,
				"unknown_field":      "should-be-removed",
				"another_extra":      42,
			},
		},
	}

	adapter := NewOntologyAwareAdapter(querier, registry)
	obj, err := adapter.GetObjectByID(context.Background(), nil, "customer", "cust-1")

	require.NoError(t, err)
	assert.Equal(t, "cust-1", obj.ID)
	assert.Len(t, obj.Properties, 3)
	assert.Equal(t, "cust-1", obj.Properties["customer_unique_id"])
	assert.Equal(t, "SP", obj.Properties["customer_state"])
	assert.Equal(t, 150.0, obj.Properties["payment_value"])
	assert.NotContains(t, obj.Properties, "unknown_field")
	assert.NotContains(t, obj.Properties, "another_extra")
}

func TestAdapter_GetObjectByID_UnknownType(t *testing.T) {
	registry := newTestRegistry()
	querier := &mockQuerier{}
	adapter := NewOntologyAwareAdapter(querier, registry)

	_, err := adapter.GetObjectByID(context.Background(), nil, "nonexistent", "x")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ontology schema lookup")
}

func TestAdapter_GetObjectByID_QuerierError(t *testing.T) {
	registry := newTestRegistry()
	querier := &mockQuerier{
		getErr: errors.New("db connection failed"),
	}
	adapter := NewOntologyAwareAdapter(querier, registry)

	_, err := adapter.GetObjectByID(context.Background(), nil, "customer", "cust-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db connection failed")
}

func TestAdapter_GetObjectByID_AllPropertiesInSchema(t *testing.T) {
	registry := newTestRegistry()
	querier := &mockQuerier{
		getObj: &repository.ObjectInstance{
			ObjectType: "customer",
			ID:         "cust-1",
			Properties: map[string]any{
				"customer_unique_id": "cust-1",
				"customer_state":     "SP",
				"payment_value":      100.0,
			},
		},
	}
	adapter := NewOntologyAwareAdapter(querier, registry)

	obj, err := adapter.GetObjectByID(context.Background(), nil, "customer", "cust-1")
	require.NoError(t, err)
	assert.Len(t, obj.Properties, 3)
}

func TestAdapter_QueryByObjectType_FiltersProperties(t *testing.T) {
	registry := newTestRegistry()
	querier := &mockQuerier{
		queryResult: &repository.ObjectQueryResult{
			Rows: []repository.ObjectInstance{
				{
					ObjectType: "customer",
					ID:         "cust-1",
					Properties: map[string]any{
						"customer_unique_id": "cust-1",
						"customer_state":     "SP",
						"extra_col":          "removed",
					},
				},
				{
					ObjectType: "customer",
					ID:         "cust-2",
					Properties: map[string]any{
						"customer_unique_id": "cust-2",
						"customer_state":     "RJ",
						"payment_value":      200.0,
						"extra_col":          "removed",
					},
				},
			},
			Total: 2,
		},
	}
	adapter := NewOntologyAwareAdapter(querier, registry)

	result, err := adapter.QueryByObjectType(context.Background(), nil, "customer", repository.ObjectFilters{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, result.Total)
	assert.Len(t, result.Rows, 2)

	for _, row := range result.Rows {
		assert.NotContains(t, row.Properties, "extra_col")
	}
	assert.Equal(t, "SP", result.Rows[0].Properties["customer_state"])
	assert.Equal(t, 200.0, result.Rows[1].Properties["payment_value"])
}

func TestAdapter_QueryByObjectType_UnknownType(t *testing.T) {
	registry := newTestRegistry()
	querier := &mockQuerier{}
	adapter := NewOntologyAwareAdapter(querier, registry)

	_, err := adapter.QueryByObjectType(context.Background(), nil, "nonexistent", repository.ObjectFilters{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ontology schema lookup")
}

func TestAdapter_QueryByObjectType_EmptyResult(t *testing.T) {
	registry := newTestRegistry()
	querier := &mockQuerier{
		queryResult: &repository.ObjectQueryResult{
			Rows:  []repository.ObjectInstance{},
			Total: 0,
		},
	}
	adapter := NewOntologyAwareAdapter(querier, registry)

	result, err := adapter.QueryByObjectType(context.Background(), nil, "customer", repository.ObjectFilters{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 0, result.Total)
	assert.Empty(t, result.Rows)
}

func TestAdapter_QueryByObjectType_QuerierError(t *testing.T) {
	registry := newTestRegistry()
	querier := &mockQuerier{
		queryErr: errors.New("query failed"),
	}
	adapter := NewOntologyAwareAdapter(querier, registry)

	_, err := adapter.QueryByObjectType(context.Background(), nil, "customer", repository.ObjectFilters{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query failed")
}

func TestAdapter_GetObjectTypeSchema_ReturnsCorrectly(t *testing.T) {
	registry := newTestRegistry()
	querier := &mockQuerier{}
	adapter := NewOntologyAwareAdapter(querier, registry)

	schema, err := adapter.GetObjectTypeSchema(context.Background(), "customer")
	require.NoError(t, err)
	assert.Equal(t, "customer", schema.Name)
	assert.Equal(t, "Customer", schema.DisplayName)
	assert.Equal(t, "customer_unique_id", schema.Grain)
	assert.Equal(t, "customer_unique_id", schema.PrimaryKey)
	assert.Len(t, schema.Properties, 3)
	assert.Contains(t, schema.Properties, "customer_unique_id")
	assert.True(t, schema.Properties["customer_unique_id"].IsPK)
	assert.Contains(t, schema.Properties, "customer_state")
	assert.False(t, schema.Properties["customer_state"].IsPK)
}

func TestAdapter_GetObjectTypeSchema_UnknownType(t *testing.T) {
	registry := newTestRegistry()
	querier := &mockQuerier{}
	adapter := NewOntologyAwareAdapter(querier, registry)

	_, err := adapter.GetObjectTypeSchema(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ontology schema")
}

func TestFilterProperties_BasicFiltering(t *testing.T) {
	props := map[string]any{
		"keep":    1,
		"remove":  2,
		"also_ok": "yes",
	}
	allowed := map[string]bool{"keep": true, "also_ok": true}

	filtered := filterProperties(props, allowed)
	assert.Len(t, filtered, 2)
	assert.Equal(t, 1, filtered["keep"])
	assert.Equal(t, "yes", filtered["also_ok"])
	assert.NotContains(t, filtered, "remove")
}

func TestFilterProperties_EmptyAllowed(t *testing.T) {
	props := map[string]any{"a": 1, "b": 2}
	allowed := map[string]bool{}

	filtered := filterProperties(props, allowed)
	assert.Empty(t, filtered)
}

func TestFilterProperties_NilProps(t *testing.T) {
	allowed := map[string]bool{"a": true}
	filtered := filterProperties(nil, allowed)
	assert.Empty(t, filtered)
}
