package log

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/repository/common"
)

func TestMockListRecentLogs(t *testing.T) {
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{
				{"api_request", "info", "GET /health", nil, "2026-01-01T00:00:00Z", 2},
				{"pipeline_run", "info", "full/auto", nil, "2026-01-01T00:00:00Z", 2},
			}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	logs, total, err := repo.ListRecentLogs(ctx, 10, 0)

	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, logs, 2)
	assert.Equal(t, "api_request", logs[0].LogType)
	assert.Equal(t, "info", logs[0].Level)
}

func TestMockListRecentLogs_Empty(t *testing.T) {
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	logs, total, err := repo.ListRecentLogs(ctx, 10, 0)

	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, logs)
}

func TestMockListRecentLogs_QueryError(t *testing.T) {
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	logs, total, err := repo.ListRecentLogs(ctx, 10, 0)

	assert.Error(t, err)
	assert.Nil(t, logs)
	assert.Equal(t, 0, total)
}

func TestMockListErrorLogs(t *testing.T) {
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{
				{"error_log", "error", "something broke", nil, "2026-01-01T00:00:00Z", 1},
			}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	logs, total, err := repo.ListErrorLogs(ctx, 10, 0)

	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, logs, 1)
	assert.Equal(t, "error_log", logs[0].LogType)
}

func TestMockListAuditLogs(t *testing.T) {
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{
				{"audit_log", "info", "create on case", nil, "2026-01-01T00:00:00Z", 1},
			}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	logs, total, err := repo.ListAuditLogs(ctx, 10, 0)

	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, logs, 1)
	assert.Equal(t, "audit_log", logs[0].LogType)
}

func TestMockListRecentLogs_Pagination(t *testing.T) {
	var capturedArgs []interface{}
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			capturedArgs = args
			return common.NewMockRows([][]interface{}{
				{"api_request", "info", "GET /health", nil, "2026-01-01T00:00:00Z", 10},
			}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	_, _, err := repo.ListRecentLogs(ctx, 5, 10)

	require.NoError(t, err)
	require.Len(t, capturedArgs, 2)
	assert.Equal(t, 5, capturedArgs[0])
	assert.Equal(t, 10, capturedArgs[1])
}
