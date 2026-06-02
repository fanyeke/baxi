package context

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/repository/common"
)

func TestMockGetLastPipelineRun(t *testing.T) {
	mock := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRow("1", "full", "auto", "completed", "2026-01-01T00:00:00Z", nil, int64(10), int64(5), nil)
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	run, err := repo.GetLastPipelineRun(ctx)

	require.NoError(t, err)
	require.NotNil(t, run)
	assert.Equal(t, "completed", run.Status)
	assert.Equal(t, int64(1), run.RunID)
}

func TestMockGetLastPipelineRun_Empty(t *testing.T) {
	mock := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRowError(fmt.Errorf("no rows in result set"))
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	run, err := repo.GetLastPipelineRun(ctx)

	require.NoError(t, err)
	assert.Nil(t, run)
}

func TestMockGetAlerts(t *testing.T) {
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{
				{"a1", "high", "metric1", "new"},
				{"a2", "low", "metric2", "resolved"},
			}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	alerts, err := repo.GetAlerts(ctx, "", 10)

	require.NoError(t, err)
	assert.Len(t, alerts, 2)
	assert.Equal(t, "a1", alerts[0].AlertID)
	assert.Equal(t, "high", alerts[0].Severity)
}

func TestMockGetAlerts_WithSeverityFilter(t *testing.T) {
	var capturedArgs []interface{}
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			capturedArgs = args
			return common.NewMockRows([][]interface{}{
				{"a1", "high", "metric1", "new"},
			}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	alerts, err := repo.GetAlerts(ctx, "high", 10)

	require.NoError(t, err)
	assert.Len(t, alerts, 1)
	require.Len(t, capturedArgs, 2)
	assert.Equal(t, "high", capturedArgs[0])
	assert.Equal(t, 10, capturedArgs[1])
}

func TestMockGetOpenTasks(t *testing.T) {
	ownerRole := "admin"
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{
				{"t1", "Do X", "todo", &ownerRole},
			}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	tasks, err := repo.GetOpenTasks(ctx, 10)

	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "t1", tasks[0].TaskID)
	assert.Equal(t, "admin", tasks[0].OwnerRole)
}

func TestMockGetPendingOutbox(t *testing.T) {
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{
				{"o1", "alert", "pending"},
			}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	events, err := repo.GetPendingOutbox(ctx, 10)

	require.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, "o1", events[0].EventID)
	assert.Equal(t, "pending", events[0].Status)
}
