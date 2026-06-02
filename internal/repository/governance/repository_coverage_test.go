package governance

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/repository/common"
)

// ──── helper to create mock querier with specific row data ──────────────────

func newMockQuerierWithRows(rows [][]interface{}, err error) *common.MockQuerier {
	return &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			if err != nil {
				return nil, err
			}
			return common.NewMockRows(rows), nil
		},
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			if err != nil {
				return common.NewMockRowError(err)
			}
			if len(rows) > 0 {
				return common.NewMockRow(rows[0]...)
			}
			return common.NewMockRow()
		},
	}
}

func TestRepository_NewRepository(t *testing.T) {
	q := &common.MockQuerier{}
	repo := NewRepository(q)
	assert.NotNil(t, repo)
}

// ──── GetConfigSnapshots ────────────────────────────────────────────────────

func TestRepository_GetConfigSnapshots_Success(t *testing.T) {
	q := newMockQuerierWithRows([][]interface{}{
		{"classification_rules"},
		{"access_policies"},
	}, nil)
	repo := NewRepository(q)

	results, err := repo.GetConfigSnapshots(context.Background())
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "classification_rules", results[0].ConfigKey)
	assert.Equal(t, "loaded", results[0].Status)
	assert.Equal(t, "access_policies", results[1].ConfigKey)
}

func TestRepository_GetConfigSnapshots_Empty(t *testing.T) {
	q := newMockQuerierWithRows([][]interface{}{}, nil)
	repo := NewRepository(q)

	results, err := repo.GetConfigSnapshots(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Empty(t, results)
}

func TestRepository_GetConfigSnapshots_QueryError(t *testing.T) {
	q := newMockQuerierWithRows(nil, fmt.Errorf("connection refused"))
	repo := NewRepository(q)

	_, err := repo.GetConfigSnapshots(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
}

func TestRepository_GetConfigSnapshots_NilRows(t *testing.T) {
	q := newMockQuerierWithRows(nil, nil)
	// When rows is nil, NewMockRows returns empty MockRows with no data
	q.QueryFunc = func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
		return &common.MockRows{}, nil
	}
	repo := NewRepository(q)

	results, err := repo.GetConfigSnapshots(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Empty(t, results)
}

// ──── CountTableRows ────────────────────────────────────────────────────────

func TestRepository_CountTableRows_Success(t *testing.T) {
	q := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRow(42)
		},
	}
	repo := NewRepository(q)

	count := repo.CountTableRows(context.Background(), "gov", "config_snapshot")
	assert.Equal(t, 42, count)
}

func TestRepository_CountTableRows_Error(t *testing.T) {
	q := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRowError(fmt.Errorf("table not found"))
		},
	}
	repo := NewRepository(q)

	count := repo.CountTableRows(context.Background(), "gov", "nonexistent")
	assert.Equal(t, 0, count)
}

// ──── GetObjectSchemas ──────────────────────────────────────────────────────

func TestRepository_GetObjectSchemas_Success(t *testing.T) {
	q := newMockQuerierWithRows([][]interface{}{
		{"order", "Order", []byte(`{"type":"object"}`), "1"},
		{"customer", "Customer", []byte(`{"type":"object"}`), "1"},
	}, nil)
	repo := NewRepository(q)

	results, err := repo.GetObjectSchemas(context.Background())
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "order", results[0].ObjectType)
}

func TestRepository_GetObjectSchemas_Empty(t *testing.T) {
	q := newMockQuerierWithRows([][]interface{}{}, nil)
	repo := NewRepository(q)

	results, err := repo.GetObjectSchemas(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Empty(t, results)
}

func TestRepository_GetObjectSchemas_QueryError(t *testing.T) {
	q := newMockQuerierWithRows(nil, fmt.Errorf("db error"))
	repo := NewRepository(q)

	_, err := repo.GetObjectSchemas(context.Background())
	assert.Error(t, err)
}

func TestRepository_GetObjectSchemas_NilRows(t *testing.T) {
	q := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return &common.MockRows{}, nil
		},
	}
	repo := NewRepository(q)

	results, err := repo.GetObjectSchemas(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Empty(t, results)
}

// ──── CountObjectSchemas ────────────────────────────────────────────────────

func TestRepository_CountObjectSchemas_Success(t *testing.T) {
	q := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRow(5)
		},
	}
	repo := NewRepository(q)

	count := repo.CountObjectSchemas(context.Background())
	assert.Equal(t, 5, count)
}

func TestRepository_CountObjectSchemas_Error(t *testing.T) {
	q := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRowError(fmt.Errorf("table not found"))
		},
	}
	repo := NewRepository(q)

	count := repo.CountObjectSchemas(context.Background())
	assert.Equal(t, 0, count)
}

// ──── GetDataClassifications ────────────────────────────────────────────────

func TestRepository_GetDataClassifications_Success(t *testing.T) {
	q := newMockQuerierWithRows([][]interface{}{
		{"customer.email", "pii", 0.9, "Email address"},
		{"order.total", "internal", 0.3, "Order total"},
	}, nil)
	repo := NewRepository(q)

	results, err := repo.GetDataClassifications(context.Background())
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "customer.email", results[0].FieldPath)
	assert.Equal(t, "pii", results[0].ClassificationLevel)
	assert.InDelta(t, 0.9, results[0].SensitivityScore, 0.01)
}

func TestRepository_GetDataClassifications_Empty(t *testing.T) {
	q := newMockQuerierWithRows([][]interface{}{}, nil)
	repo := NewRepository(q)

	results, err := repo.GetDataClassifications(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Empty(t, results)
}

func TestRepository_GetDataClassifications_QueryError(t *testing.T) {
	q := newMockQuerierWithRows(nil, fmt.Errorf("db error"))
	repo := NewRepository(q)

	_, err := repo.GetDataClassifications(context.Background())
	assert.Error(t, err)
}

func TestRepository_GetDataClassifications_NilRows(t *testing.T) {
	q := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return &common.MockRows{}, nil
		},
	}
	repo := NewRepository(q)

	results, err := repo.GetDataClassifications(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Empty(t, results)
}

// ──── GetByFieldPath ────────────────────────────────────────────────────────

func TestRepository_GetByFieldPath_Success(t *testing.T) {
	q := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRow("customer.email", "pii", 0.9, "Email address")
		},
	}
	repo := NewRepository(q)

	result, err := repo.GetByFieldPath(context.Background(), "customer.email")
	require.NoError(t, err)
	assert.Equal(t, "customer.email", result.FieldPath)
	assert.Equal(t, "pii", result.ClassificationLevel)
}

func TestRepository_GetByFieldPath_NotFound(t *testing.T) {
	q := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRowError(fmt.Errorf("no rows"))
		},
	}
	repo := NewRepository(q)

	_, err := repo.GetByFieldPath(context.Background(), "unknown.field")
	assert.Error(t, err)
}

// ──── GetDataLineage ────────────────────────────────────────────────────────

func TestRepository_GetDataLineage_Success(t *testing.T) {
	q := newMockQuerierWithRows([][]interface{}{
		{"dwd_orders", "order_id", "mart_orders", "order_id", "direct", 0.95},
	}, nil)
	repo := NewRepository(q)

	results, err := repo.GetDataLineage(context.Background())
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "dwd_orders", results[0].SourceTable)
}

func TestRepository_GetDataLineage_Empty(t *testing.T) {
	q := newMockQuerierWithRows([][]interface{}{}, nil)
	repo := NewRepository(q)

	results, err := repo.GetDataLineage(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Empty(t, results)
}

func TestRepository_GetDataLineage_QueryError(t *testing.T) {
	q := newMockQuerierWithRows(nil, fmt.Errorf("db error"))
	repo := NewRepository(q)

	_, err := repo.GetDataLineage(context.Background())
	assert.Error(t, err)
}

func TestRepository_GetDataLineage_NilRows(t *testing.T) {
	q := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return &common.MockRows{}, nil
		},
	}
	repo := NewRepository(q)

	results, err := repo.GetDataLineage(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Empty(t, results)
}

// ──── GetLineageBySource ────────────────────────────────────────────────────

func TestRepository_GetLineageBySource_Success(t *testing.T) {
	q := newMockQuerierWithRows([][]interface{}{
		{"dwd_orders", "order_id", "mart_orders", "order_id", "direct", 0.95},
	}, nil)
	repo := NewRepository(q)

	results, err := repo.GetLineageBySource(context.Background(), "dwd_orders")
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestRepository_GetLineageBySource_Empty(t *testing.T) {
	q := newMockQuerierWithRows([][]interface{}{}, nil)
	repo := NewRepository(q)

	results, err := repo.GetLineageBySource(context.Background(), "dwd_orders")
	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Empty(t, results)
}

func TestRepository_GetLineageBySource_QueryError(t *testing.T) {
	q := newMockQuerierWithRows(nil, fmt.Errorf("db error"))
	repo := NewRepository(q)

	_, err := repo.GetLineageBySource(context.Background(), "dwd_orders")
	assert.Error(t, err)
}

func TestRepository_GetLineageBySource_NilRows(t *testing.T) {
	q := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return &common.MockRows{}, nil
		},
	}
	repo := NewRepository(q)

	results, err := repo.GetLineageBySource(context.Background(), "dwd_orders")
	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Empty(t, results)
}

// ──── GetLineageByTarget ────────────────────────────────────────────────────

func TestRepository_GetLineageByTarget_Success(t *testing.T) {
	q := newMockQuerierWithRows([][]interface{}{
		{"dwd_orders", "order_id", "mart_orders", "order_id", "direct", 0.95},
	}, nil)
	repo := NewRepository(q)

	results, err := repo.GetLineageByTarget(context.Background(), "mart_orders")
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestRepository_GetLineageByTarget_Empty(t *testing.T) {
	q := newMockQuerierWithRows([][]interface{}{}, nil)
	repo := NewRepository(q)

	results, err := repo.GetLineageByTarget(context.Background(), "mart_orders")
	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Empty(t, results)
}

func TestRepository_GetLineageByTarget_QueryError(t *testing.T) {
	q := newMockQuerierWithRows(nil, fmt.Errorf("db error"))
	repo := NewRepository(q)

	_, err := repo.GetLineageByTarget(context.Background(), "mart_orders")
	assert.Error(t, err)
}

func TestRepository_GetLineageByTarget_NilRows(t *testing.T) {
	q := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return &common.MockRows{}, nil
		},
	}
	repo := NewRepository(q)

	results, err := repo.GetLineageByTarget(context.Background(), "mart_orders")
	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Empty(t, results)
}

// ──── GetAccessPolicies ─────────────────────────────────────────────────────

func TestRepository_GetAccessPolicies_Success(t *testing.T) {
	q := newMockQuerierWithRows([][]interface{}{
		{"read_policy", "data", "order", "read", "role", "admin", "allow", []byte(`{}`)},
	}, nil)
	repo := NewRepository(q)

	results, err := repo.GetAccessPolicies(context.Background())
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "read_policy", results[0].PolicyName)
}

func TestRepository_GetAccessPolicies_Empty(t *testing.T) {
	q := newMockQuerierWithRows([][]interface{}{}, nil)
	repo := NewRepository(q)

	results, err := repo.GetAccessPolicies(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Empty(t, results)
}

func TestRepository_GetAccessPolicies_QueryError(t *testing.T) {
	q := newMockQuerierWithRows(nil, fmt.Errorf("db error"))
	repo := NewRepository(q)

	_, err := repo.GetAccessPolicies(context.Background())
	assert.Error(t, err)
}

func TestRepository_GetAccessPolicies_NilRows(t *testing.T) {
	q := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return &common.MockRows{}, nil
		},
	}
	repo := NewRepository(q)

	results, err := repo.GetAccessPolicies(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Empty(t, results)
}

// ──── GetAccessPoliciesByRole ───────────────────────────────────────────────

func TestRepository_GetAccessPoliciesByRole_Success(t *testing.T) {
	q := newMockQuerierWithRows([][]interface{}{
		{"read_policy", "data", "order", "read", "role", "admin", "allow", []byte(`{}`)},
	}, nil)
	repo := NewRepository(q)

	results, err := repo.GetAccessPoliciesByRole(context.Background(), "admin")
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "admin", results[0].PrincipalPattern)
}

func TestRepository_GetAccessPoliciesByRole_Empty(t *testing.T) {
	q := newMockQuerierWithRows([][]interface{}{}, nil)
	repo := NewRepository(q)

	results, err := repo.GetAccessPoliciesByRole(context.Background(), "viewer")
	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Empty(t, results)
}

func TestRepository_GetAccessPoliciesByRole_QueryError(t *testing.T) {
	q := newMockQuerierWithRows(nil, fmt.Errorf("db error"))
	repo := NewRepository(q)

	_, err := repo.GetAccessPoliciesByRole(context.Background(), "admin")
	assert.Error(t, err)
}

func TestRepository_GetAccessPoliciesByRole_NilRows(t *testing.T) {
	q := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return &common.MockRows{}, nil
		},
	}
	repo := NewRepository(q)

	results, err := repo.GetAccessPoliciesByRole(context.Background(), "admin")
	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Empty(t, results)
}
