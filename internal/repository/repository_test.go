package repository

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

func TestOntologyAwareRepo_Interface(t *testing.T) {
	// Verify that the interface can be satisfied
	var _ OntologyAwareRepo = (*mockOntologyAwareRepo)(nil)
}

type mockOntologyAwareRepo struct{}

func (m *mockOntologyAwareRepo) GetObjectByID(ctx context.Context, pool *pgxpool.Pool, objectType, objectID string) (*ObjectInstance, error) {
	return nil, nil
}

func (m *mockOntologyAwareRepo) QueryByObjectType(ctx context.Context, pool *pgxpool.Pool, objectType string, filters ObjectFilters) (*ObjectQueryResult, error) {
	return nil, nil
}

func (m *mockOntologyAwareRepo) GetObjectTypeSchema(ctx context.Context, objectType string) (*ObjectSchema, error) {
	return nil, nil
}

func TestObjectSchema(t *testing.T) {
	schema := ObjectSchema{
		Name:        "order",
		DisplayName: "Order",
		Grain:       "order_id",
		PrimaryKey:  "order_id",
		Properties: map[string]PropertySchema{
			"order_id": {Name: "order_id", Type: "string", IsPK: true},
			"status":   {Name: "status", Type: "string", IsPK: false},
		},
	}

	assert.Equal(t, "order", schema.Name)
	assert.Equal(t, "order_id", schema.PrimaryKey)
	assert.Len(t, schema.Properties, 2)
	assert.True(t, schema.Properties["order_id"].IsPK)
	assert.False(t, schema.Properties["status"].IsPK)
}

func TestObjectInstance(t *testing.T) {
	obj := ObjectInstance{
		ObjectType: "order",
		ID:         "ord-1",
		Properties: map[string]interface{}{
			"status": "delivered",
		},
	}

	assert.Equal(t, "order", obj.ObjectType)
	assert.Equal(t, "ord-1", obj.ID)
	assert.Equal(t, "delivered", obj.Properties["status"])
}

func TestObjectQueryResult(t *testing.T) {
	result := ObjectQueryResult{
		Rows: []ObjectInstance{
			{ObjectType: "order", ID: "ord-1"},
			{ObjectType: "order", ID: "ord-2"},
		},
		Total: 10,
	}

	assert.Len(t, result.Rows, 2)
	assert.Equal(t, 10, result.Total)
}

func TestObjectFilters(t *testing.T) {
	filters := ObjectFilters{
		ObjectType: "order",
		Limit:      100,
		Offset:     0,
		Filters: map[string]interface{}{
			"status": "delivered",
		},
	}

	assert.Equal(t, "order", filters.ObjectType)
	assert.Equal(t, 100, filters.Limit)
	assert.Equal(t, "delivered", filters.Filters["status"])
}

func TestSearchFilters(t *testing.T) {
	filters := SearchFilters{
		ObjectType: "order",
		Query:      "delivered",
		Limit:      50,
		Offset:     0,
	}

	assert.Equal(t, "order", filters.ObjectType)
	assert.Equal(t, "delivered", filters.Query)
	assert.Equal(t, 50, filters.Limit)
}

func TestObjectMetrics(t *testing.T) {
	metrics := ObjectMetrics{
		ObjectType: "order",
		ID:         "ord-1",
		Metrics: map[string]float64{
			"total_spent":    1500.0,
			"review_score":   4.5,
		},
	}

	assert.Equal(t, "order", metrics.ObjectType)
	assert.Equal(t, "ord-1", metrics.ID)
	assert.Equal(t, 1500.0, metrics.Metrics["total_spent"])
	assert.Equal(t, 4.5, metrics.Metrics["review_score"])
}

func TestSearchResult(t *testing.T) {
	result := SearchResult{
		Rows: []ObjectInstance{
			{ObjectType: "order", ID: "ord-1"},
		},
		Total: 5,
	}

	assert.Len(t, result.Rows, 1)
	assert.Equal(t, 5, result.Total)
}

func TestPipelineRunInfo(t *testing.T) {
	info := PipelineRunInfo{
		RunID:       1,
		Status:      "completed",
		StartedAt:   "2026-01-01T00:00:00Z",
		CompletedAt: "2026-01-01T00:01:00Z",
	}

	assert.Equal(t, int64(1), info.RunID)
	assert.Equal(t, "completed", info.Status)
}

func TestAlertSummary(t *testing.T) {
	summary := AlertSummary{
		AlertID:  "a1",
		Severity: "high",
		Metric:   "cpu_usage",
		Status:   "new",
	}

	assert.Equal(t, "a1", summary.AlertID)
	assert.Equal(t, "high", summary.Severity)
}

func TestTaskSummary(t *testing.T) {
	summary := TaskSummary{
		TaskID:    "t1",
		Title:     "Fix alert",
		Status:    "todo",
		OwnerRole: "admin",
	}

	assert.Equal(t, "t1", summary.TaskID)
	assert.Equal(t, "admin", summary.OwnerRole)
}

func TestOutboxSummary(t *testing.T) {
	summary := OutboxSummary{
		EventID:   "e1",
		EventType: "alert",
		Status:    "pending",
	}

	assert.Equal(t, "e1", summary.EventID)
	assert.Equal(t, "pending", summary.Status)
}

func TestUpsertParams(t *testing.T) {
	t.Run("UpsertSnapshotParams", func(t *testing.T) {
		params := UpsertSnapshotParams{
			ConfigKey:    "test_key",
			ConfigType:   "yaml",
			SourcePath:   "/path/to/config",
			ContentJSONB: []byte(`{"key":"value"}`),
			ContentHash:  "abc123",
		}
		assert.Equal(t, "test_key", params.ConfigKey)
		assert.Equal(t, "yaml", params.ConfigType)
	})

	t.Run("ObjectSchemaUpsertParams", func(t *testing.T) {
		params := ObjectSchemaUpsertParams{
			ObjectType:  "order",
			ObjectName:  "Order",
			SchemaJSONB: []byte(`{"type":"object"}`),
			Version:     "v1",
		}
		assert.Equal(t, "order", params.ObjectType)
		assert.Equal(t, "v1", params.Version)
	})

	t.Run("ClassificationUpsertParams", func(t *testing.T) {
		params := ClassificationUpsertParams{
			FieldPath:           "user.email",
			ClassificationLevel: "confidential",
			SensitivityScore:    0.8,
			Description:         "User email address",
		}
		assert.Equal(t, "user.email", params.FieldPath)
		assert.Equal(t, 0.8, params.SensitivityScore)
	})

	t.Run("LineageUpsertParams", func(t *testing.T) {
		params := LineageUpsertParams{
			SourceTable:         "raw.orders",
			SourceColumn:        "order_id",
			TargetTable:         "dwd.orders",
			TargetColumn:        "order_id",
			TransformationLogic: "direct copy",
			Confidence:          1.0,
		}
		assert.Equal(t, "raw.orders", params.SourceTable)
		assert.Equal(t, 1.0, params.Confidence)
	})

	t.Run("AccessPolicyUpsertParams", func(t *testing.T) {
		params := AccessPolicyUpsertParams{
			PolicyName:       "read_orders",
			ResourceType:     "order",
			ResourcePattern:  "*",
			Action:           "read",
			PrincipalType:    "role",
			PrincipalPattern: "analyst",
			Effect:           "allow",
			ConditionsJSONB:  []byte(`{}`),
		}
		assert.Equal(t, "read_orders", params.PolicyName)
		assert.Equal(t, "allow", params.Effect)
	})
}
