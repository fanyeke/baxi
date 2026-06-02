package ontology

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/repository/common"
)

func TestMockQueryByObjectType_UnknownType(t *testing.T) {
	mock := &common.MockQuerier{}
	repo := NewRepository(mock)
	ctx := WithRole(context.Background(), "admin")

	result, err := repo.QueryByObjectType(ctx, "unknown_type", ObjectFilters{Limit: 10})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown object type")
	assert.Nil(t, result)
}

func TestMockQueryByObjectType_AccessDenied(t *testing.T) {
	mock := &common.MockQuerier{}
	repo := NewRepository(mock)
	// analyst role doesn't have access to dwd.order_level
	ctx := WithRole(context.Background(), "analyst")

	result, err := repo.QueryByObjectType(ctx, "order", ObjectFilters{Limit: 10})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not have access")
	assert.Nil(t, result)
}

func TestMockQueryByObjectType_Success(t *testing.T) {
	mock := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRow(1)
		},
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{
				{"ord-1", "delivered", nil, nil, nil, nil, "cust-1", "SP", false, false, nil},
			}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := WithRole(context.Background(), "admin")

	result, err := repo.QueryByObjectType(ctx, "order", ObjectFilters{Limit: 10})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Rows, 1)
	assert.Equal(t, "order", result.Rows[0].ObjectType)
}

func TestMockGetObjectByID(t *testing.T) {
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{
				{"ord-1", "delivered", nil, nil, nil, nil, "cust-1", "SP", false, false, nil},
			}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := WithRole(context.Background(), "admin")

	obj, err := repo.GetObjectByID(ctx, "order", "ord-1")

	require.NoError(t, err)
	require.NotNil(t, obj)
	assert.Equal(t, "ord-1", obj.ID)
	assert.Equal(t, "order", obj.ObjectType)
}

func TestMockGetObjectByID_NotFound(t *testing.T) {
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := WithRole(context.Background(), "admin")

	obj, err := repo.GetObjectByID(ctx, "order", "missing")

	assert.Error(t, err)
	assert.Nil(t, obj)
}

func TestMockSearchObjects(t *testing.T) {
	mock := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRow(1)
		},
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{
				{"ord-search", "delivered", nil, nil, nil, nil, "cust-2", "RJ", false, false, nil},
			}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := WithRole(context.Background(), "admin")

	result, err := repo.SearchObjects(ctx, "order", SearchFilters{Query: "ord-search", Limit: 10})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Rows, 1)
}

func TestMockGetObjectMetrics(t *testing.T) {
	mock := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRow(float64(100), float64(5000.0), float64(4.5))
		},
	}

	repo := NewRepository(mock)
	ctx := WithRole(context.Background(), "admin")

	metrics, err := repo.GetObjectMetrics(ctx, "customer", "cust-1")

	require.NoError(t, err)
	require.NotNil(t, metrics)
	assert.Equal(t, "customer", metrics.ObjectType)
	assert.Equal(t, "cust-1", metrics.ID)
	assert.Contains(t, metrics.Metrics, "total_orders")
	assert.Contains(t, metrics.Metrics, "total_spent")
	assert.Contains(t, metrics.Metrics, "avg_review_score")
}

func TestMockResolveLimit(t *testing.T) {
	n, err := resolveLimit(10)
	require.NoError(t, err)
	assert.Equal(t, 10, n)

	n, err = resolveLimit(0)
	require.NoError(t, err)
	assert.Equal(t, 1000, n)

	_, err = resolveLimit(-1)
	assert.Error(t, err)

	_, err = resolveLimit(10001)
	assert.Error(t, err)
}

func TestMockTableAccessible(t *testing.T) {
	assert.True(t, tableAccessible("admin", "dwd", "order_level"))
	assert.False(t, tableAccessible("analyst", "dwd", "order_level"))
	assert.True(t, tableAccessible("viewer", "ops", "metric_alert"))
	assert.False(t, tableAccessible("unknown_role", "dwd", "order_level"))
}

func TestMockWithRole(t *testing.T) {
	ctx := WithRole(context.Background(), "admin")
	role := resolveRole(ctx)
	assert.Equal(t, "admin", role)

	ctx2 := context.Background()
	role2 := resolveRole(ctx2)
	assert.Equal(t, "analyst", role2) // default
}

func TestMockQueryByObjectType_InvalidLimit(t *testing.T) {
	mock := &common.MockQuerier{}
	repo := NewRepository(mock)
	ctx := WithRole(context.Background(), "admin")

	_, err := repo.QueryByObjectType(ctx, "order", ObjectFilters{Limit: -1})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "limit must be non-negative")

	_, err = repo.QueryByObjectType(ctx, "order", ObjectFilters{Limit: 10001})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum")
}

func TestMockQueryByObjectType_DBError(t *testing.T) {
	mock := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRowError(fmt.Errorf("connection refused"))
		},
	}

	repo := NewRepository(mock)
	ctx := WithRole(context.Background(), "admin")

	_, err := repo.QueryByObjectType(ctx, "order", ObjectFilters{Limit: 10})
	assert.Error(t, err)
}
