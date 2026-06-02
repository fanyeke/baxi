package alert

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/repository/common"
)

func TestMockListAlerts_NoFilters(t *testing.T) {
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{
				{"a1", "r1", "2026-01-01", "high", "metric1", "order", "o1", nil, nil, nil, "owner", "new", nil, 2},
				{"a2", "r2", "2026-01-02", "low", "metric2", "order", "o2", nil, nil, nil, "owner", "new", nil, 2},
			}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	rows, total, err := repo.ListAlerts(ctx, "", "", "", "", "", 10, 0)

	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, rows, 2)
	assert.Equal(t, "a1", rows[0].AlertID)
	assert.Equal(t, "high", rows[0].Severity)
	assert.Equal(t, "a2", rows[1].AlertID)
}

func TestMockListAlerts_WithSeverityFilter(t *testing.T) {
	var capturedArgs []interface{}
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			capturedArgs = args
			return common.NewMockRows([][]interface{}{
				{"a1", "r1", "2026-01-01", "high", "metric1", "order", "o1", nil, nil, nil, "owner", "new", nil, 1},
			}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	rows, total, err := repo.ListAlerts(ctx, "high", "", "", "", "", 10, 0)

	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, rows, 1)
	// First arg should be severity="high", second should be limit, third offset
	require.Len(t, capturedArgs, 3)
	assert.Equal(t, "high", capturedArgs[0])
	assert.Equal(t, 10, capturedArgs[1])
	assert.Equal(t, 0, capturedArgs[2])
}

func TestMockListAlerts_EmptyResult(t *testing.T) {
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	rows, total, err := repo.ListAlerts(ctx, "", "", "", "", "", 10, 0)

	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, rows)
}

func TestMockListAlerts_QueryError(t *testing.T) {
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	rows, total, err := repo.ListAlerts(ctx, "", "", "", "", "", 10, 0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
	assert.Nil(t, rows)
	assert.Equal(t, 0, total)
}

func TestMockListAlerts_SortBySeverity(t *testing.T) {
	var capturedSQL string
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			capturedSQL = sql
			return common.NewMockRows([][]interface{}{
				{"a1", "r1", "2026-01-01", "low", "metric1", "order", "o1", nil, nil, nil, "owner", "new", nil, 2},
				{"a2", "r2", "2026-01-02", "high", "metric2", "order", "o2", nil, nil, nil, "owner", "new", nil, 2},
			}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	_, _, err := repo.ListAlerts(ctx, "", "", "", "", "severity_desc", 10, 0)

	require.NoError(t, err)
	assert.Contains(t, capturedSQL, "severity DESC")
}

func TestMockGetAlertByID(t *testing.T) {
	mock := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRow("a1", "r1", "2026-01-01", "high", "metric1", "order", "o1", nil, nil, nil, "owner", "new", nil)
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	row, err := repo.GetAlertByID(ctx, "a1")

	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "a1", row.AlertID)
	assert.Equal(t, "high", row.Severity)
}

func TestMockGetAlertByID_NotFound(t *testing.T) {
	mock := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRowError(fmt.Errorf("no rows in result set"))
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	row, err := repo.GetAlertByID(ctx, "missing")

	assert.Error(t, err)
	assert.Nil(t, row)
}

func TestMockListAlerts_DefaultSort(t *testing.T) {
	var capturedSQL string
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			capturedSQL = sql
			return common.NewMockRows([][]interface{}{}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	_, _, err := repo.ListAlerts(ctx, "", "", "", "", "invalid_sort", 10, 0)

	require.NoError(t, err)
	// Default sort should be created_at DESC
	assert.Contains(t, capturedSQL, "created_at DESC")
}
