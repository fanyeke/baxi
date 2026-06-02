package outbox

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/repository/common"
)

func TestMockGetDetail(t *testing.T) {
	now := time.Now().UTC()
	mock := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRow(
				"ev1", "alert", "order", "o1",
				"webhook", "pending", []byte(`{"key":"value"}`),
				now, int(0), nil, nil,
			)
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	detail, err := repo.GetDetail(ctx, "ev1")

	require.NoError(t, err)
	require.NotNil(t, detail)
	assert.Equal(t, "ev1", detail.EventID)
	assert.Equal(t, "alert", detail.EventType)
	assert.Equal(t, "pending", detail.Status)
}

func TestMockGetDetail_NotFound(t *testing.T) {
	mock := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRowError(pgx.ErrNoRows)
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	detail, err := repo.GetDetail(ctx, "missing")

	require.NoError(t, err)
	assert.Nil(t, detail)
}

func TestMockGetDetail_DBError(t *testing.T) {
	mock := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRowError(fmt.Errorf("connection refused"))
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	detail, err := repo.GetDetail(ctx, "ev1")

	assert.Error(t, err)
	assert.Nil(t, detail)
}

func TestMockListOutboxEvents(t *testing.T) {
	now := time.Now().UTC()
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{
				{"ev1", "alert", "order", "o1", "webhook", "pending", now, 0, nil, 2},
				{"ev2", "task", "order", "o2", "cli", "dispatched", now, 1, nil, 2},
			}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	rows, total, err := repo.ListOutboxEvents(ctx, OutboxFilters{}, 10, 0)

	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, rows, 2)
	assert.Equal(t, "ev1", rows[0].OutboxID)
	assert.Equal(t, "pending", rows[0].Status)
}

func TestMockListOutboxEvents_Empty(t *testing.T) {
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	rows, total, err := repo.ListOutboxEvents(ctx, OutboxFilters{}, 10, 0)

	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, rows)
}

func TestMockListOutboxEvents_WithStatusFilter(t *testing.T) {
	var capturedSQL string
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			capturedSQL = sql
			return common.NewMockRows([][]interface{}{}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	status := "pending"
	_, _, err := repo.ListOutboxEvents(ctx, OutboxFilters{Status: &status}, 10, 0)

	require.NoError(t, err)
	assert.Contains(t, capturedSQL, "status = $")
}

func TestMockListOutboxEvents_WithChannelFilter(t *testing.T) {
	var capturedSQL string
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			capturedSQL = sql
			return common.NewMockRows([][]interface{}{}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	channel := "webhook"
	_, _, err := repo.ListOutboxEvents(ctx, OutboxFilters{Channel: &channel}, 10, 0)

	require.NoError(t, err)
	assert.Contains(t, capturedSQL, "target_channel = $")
}

func TestMockListOutboxEvents_Pagination(t *testing.T) {
	var capturedArgs []interface{}
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			capturedArgs = args
			return common.NewMockRows([][]interface{}{}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	_, _, err := repo.ListOutboxEvents(ctx, OutboxFilters{}, 5, 10)

	require.NoError(t, err)
	// Last two args should be limit and offset
	require.Len(t, capturedArgs, 2)
	assert.Equal(t, 5, capturedArgs[0])
	assert.Equal(t, 10, capturedArgs[1])
}

func TestMockListOutboxEvents_QueryError(t *testing.T) {
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	rows, total, err := repo.ListOutboxEvents(ctx, OutboxFilters{}, 10, 0)

	assert.Error(t, err)
	assert.Nil(t, rows)
	assert.Equal(t, 0, total)
}
