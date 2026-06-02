package alert

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/repository/common"
)

// ──── mockTx for QueryAlerts tests ──────────────────────────────────────────

type testMockTx struct {
	queryFunc func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
}

func (m *testMockTx) Begin(ctx context.Context) (pgx.Tx, error) {
	return nil, nil
}

func (m *testMockTx) Commit(ctx context.Context) error {
	return nil
}

func (m *testMockTx) Rollback(ctx context.Context) error {
	return nil
}

func (m *testMockTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}

func (m *testMockTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return nil
}

func (m *testMockTx) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}

func (m *testMockTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, nil
}

func (m *testMockTx) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (m *testMockTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if m.queryFunc != nil {
		return m.queryFunc(ctx, sql, args...)
	}
	return &common.MockRows{}, nil
}

func (m *testMockTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return &common.MockRow{}
}

func (m *testMockTx) Conn() *pgx.Conn {
	return nil
}

// ──── QueryAlerts tests ─────────────────────────────────────────────────────

func TestQueryAlerts_NoFilters(t *testing.T) {
	tx := &testMockTx{
		queryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{
				{"alert-1", "rule-1", "2024-01-01", "high", "cpu_usage", "server", "srv-1",
					95.0, 80.0, 0.18, "ops", "active", 8.5, 1},
			}), nil
		},
	}
	repo := NewRepository(&common.MockQuerier{})

	results, total, err := repo.QueryAlerts(context.Background(), tx, "", "", "", "", "created_at_desc", 10, 0)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, 1, total)
	assert.Equal(t, "alert-1", results[0].AlertID)
}

func TestQueryAlerts_WithFilters(t *testing.T) {
	tx := &testMockTx{
		queryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{
				{"alert-1", "rule-1", "2024-01-01", "high", "cpu_usage", "server", "srv-1",
					95.0, 80.0, 0.18, "ops", "active", 8.5, 1},
			}), nil
		},
	}
	repo := NewRepository(&common.MockQuerier{})

	results, total, err := repo.QueryAlerts(context.Background(), tx, "critical", "active", "server", "rule-1", "severity_desc", 5, 0)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, 1, total)
}

func TestQueryAlerts_Empty(t *testing.T) {
	tx := &testMockTx{
		queryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{}), nil
		},
	}
	repo := NewRepository(&common.MockQuerier{})

	results, total, err := repo.QueryAlerts(context.Background(), tx, "", "", "", "", "created_at_desc", 10, 0)
	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Empty(t, results)
	assert.Equal(t, 0, total)
}

func TestQueryAlerts_QueryError(t *testing.T) {
	tx := &testMockTx{
		queryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return nil, fmt.Errorf("tx error")
		},
	}
	repo := NewRepository(&common.MockQuerier{})

	_, _, err := repo.QueryAlerts(context.Background(), tx, "", "", "", "", "created_at_desc", 10, 0)
	assert.Error(t, err)
}

func TestQueryAlerts_UnknownSort(t *testing.T) {
	tx := &testMockTx{
		queryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{}), nil
		},
	}
	repo := NewRepository(&common.MockQuerier{})

	_, _, err := repo.QueryAlerts(context.Background(), tx, "", "", "", "", "bad_sort", 10, 0)
	require.NoError(t, err)
}

func TestQueryAlerts_PartialFilters(t *testing.T) {
	tx := &testMockTx{
		queryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{}), nil
		},
	}
	repo := NewRepository(&common.MockQuerier{})

	// Only severity filter
	_, _, err := repo.QueryAlerts(context.Background(), tx, "high", "", "", "", "created_at_desc", 10, 0)
	require.NoError(t, err)

	// Only status filter
	_, _, err = repo.QueryAlerts(context.Background(), tx, "", "active", "", "", "created_at_desc", 10, 0)
	require.NoError(t, err)

	// Only objectType filter
	_, _, err = repo.QueryAlerts(context.Background(), tx, "", "", "server", "", "created_at_desc", 10, 0)
	require.NoError(t, err)

	// Only ruleID filter
	_, _, err = repo.QueryAlerts(context.Background(), tx, "", "", "", "rule-1", "created_at_desc", 10, 0)
	require.NoError(t, err)
}

func TestQueryAlerts_SeveritySort(t *testing.T) {
	tx := &testMockTx{
		queryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{}), nil
		},
	}
	repo := NewRepository(&common.MockQuerier{})

	_, _, err := repo.QueryAlerts(context.Background(), tx, "", "", "", "", "severity_desc", 10, 0)
	require.NoError(t, err)
}

func TestQueryAlerts_CreatedAtAscSort(t *testing.T) {
	tx := &testMockTx{
		queryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{}), nil
		},
	}
	repo := NewRepository(&common.MockQuerier{})

	_, _, err := repo.QueryAlerts(context.Background(), tx, "", "", "", "", "created_at_asc", 10, 0)
	require.NoError(t, err)
}

// ──── ListAlerts additional coverage ────────────────────────────────────────

func TestListAlerts_AllFilters(t *testing.T) {
	var capturedArgs []interface{}
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			capturedArgs = args
			return common.NewMockRows([][]interface{}{
				{"a1", "r1", "2024-01-01", "high", "cpu", "server", "s1",
					95.0, 80.0, 0.18, "ops", "active", 8.5, 1},
			}), nil
		},
	}
	repo := NewRepository(mock)

	results, total, err := repo.ListAlerts(context.Background(), "high", "active", "server", "rule-1", "created_at_desc", 10, 0)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, 1, total)

	// Verify all 6 args: severity, status, objectType, ruleID, limit, offset
	require.Len(t, capturedArgs, 6)
	assert.Equal(t, "high", capturedArgs[0])
	assert.Equal(t, "active", capturedArgs[1])
	assert.Equal(t, "server", capturedArgs[2])
	assert.Equal(t, "rule-1", capturedArgs[3])
	assert.Equal(t, 10, capturedArgs[4])
	assert.Equal(t, 0, capturedArgs[5])
}

func TestListAlerts_PartialFilters(t *testing.T) {
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{}), nil
		},
	}
	repo := NewRepository(mock)

	// Only objectType
	_, _, err := repo.ListAlerts(context.Background(), "", "", "server", "", "created_at_desc", 10, 0)
	require.NoError(t, err)

	// Only ruleID
	_, _, err = repo.ListAlerts(context.Background(), "", "", "", "rule-1", "created_at_desc", 10, 0)
	require.NoError(t, err)
}

func TestSortMap_AllKeys(t *testing.T) {
	assert.Equal(t, "created_at DESC", SortMap["created_at_desc"])
	assert.Equal(t, "created_at ASC", SortMap["created_at_asc"])
	assert.Equal(t, "severity DESC, created_at DESC", SortMap["severity_desc"])
	assert.Len(t, SortMap, 3)
}
