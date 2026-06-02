package status

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/repository/common"
)

func TestMockGetTableCounts(t *testing.T) {
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{
				{"action_tasks", 5},
				{"alert_events", 10},
				{"event_outbox", 3},
				{"dwd_item_level", 100},
				{"dwd_order_level", 50},
				{"metric_daily", 20},
				{"metric_dimension_daily", 15},
			}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	counts, err := repo.GetTableCounts(ctx)

	require.NoError(t, err)
	assert.Len(t, counts, 7)
	m := make(map[string]int)
	for _, c := range counts {
		m[c.TableName] = c.RowCount
	}
	assert.Equal(t, 10, m["alert_events"])
	assert.Equal(t, 5, m["action_tasks"])
	assert.Equal(t, 3, m["event_outbox"])
}

func TestMockGetTableCounts_Empty(t *testing.T) {
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	counts, err := repo.GetTableCounts(ctx)

	require.NoError(t, err)
	assert.Empty(t, counts)
}

func TestMockGetTableCounts_Error(t *testing.T) {
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	counts, err := repo.GetTableCounts(ctx)

	assert.Error(t, err)
	assert.Nil(t, counts)
}

func TestMockGetLastPipelineRun(t *testing.T) {
	mock := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRow(
				"1", "full", "auto", "completed",
				"2026-01-01T00:00:00Z", nil,
				int64(100), int64(50), nil,
			)
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	run, err := repo.GetLastPipelineRun(ctx)

	require.NoError(t, err)
	require.NotNil(t, run)
	assert.Equal(t, "completed", run.Status)
	assert.Equal(t, "1", run.RunID)
}

func TestMockGetLastPipelineRun_Empty(t *testing.T) {
	mock := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRowError(pgx.ErrNoRows)
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	run, err := repo.GetLastPipelineRun(ctx)

	require.NoError(t, err)
	assert.Nil(t, run)
}

func TestMockGetLastPipelineRun_DBError(t *testing.T) {
	mock := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRowError(fmt.Errorf("connection refused"))
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	run, err := repo.GetLastPipelineRun(ctx)

	assert.Error(t, err)
	assert.Nil(t, run)
}
