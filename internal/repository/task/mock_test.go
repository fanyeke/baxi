package task

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

func TestMockListTasks(t *testing.T) {
	now := time.Now().UTC()
	ownerRole := "admin"
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{
				{"t1", nil, nil, "Do X", nil, nil, nil, &ownerRole, nil, "high", nil, "todo", nil, nil, now, 3},
			}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	rows, total, err := repo.ListTasks(ctx, TaskFilters{}, 10, 0)

	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, rows, 1)
	assert.Equal(t, "t1", rows[0].TaskID)
	require.NotNil(t, rows[0].OwnerRole)
	assert.Equal(t, "admin", *rows[0].OwnerRole)
}

func TestMockListTasks_Empty(t *testing.T) {
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	rows, total, err := repo.ListTasks(ctx, TaskFilters{}, 10, 0)

	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, rows)
}

func TestMockListTasks_WithStatusFilter(t *testing.T) {
	var capturedSQL string
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			capturedSQL = sql
			return common.NewMockRows([][]interface{}{}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	status := "todo"
	_, _, err := repo.ListTasks(ctx, TaskFilters{Status: &status}, 10, 0)

	require.NoError(t, err)
	assert.Contains(t, capturedSQL, "status = $")
}

func TestMockListTasks_WithPriorityFilter(t *testing.T) {
	var capturedSQL string
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			capturedSQL = sql
			return common.NewMockRows([][]interface{}{}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	priority := "high"
	_, _, err := repo.ListTasks(ctx, TaskFilters{Priority: &priority}, 10, 0)

	require.NoError(t, err)
	assert.Contains(t, capturedSQL, "priority = $")
}

func TestMockListTasks_WithOwnerFilter(t *testing.T) {
	var capturedSQL string
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			capturedSQL = sql
			return common.NewMockRows([][]interface{}{}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	owner := "admin"
	_, _, err := repo.ListTasks(ctx, TaskFilters{Owner: &owner}, 10, 0)

	require.NoError(t, err)
	assert.Contains(t, capturedSQL, "owner_role = $")
}

func TestMockListTasks_Pagination(t *testing.T) {
	var capturedArgs []interface{}
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			capturedArgs = args
			return common.NewMockRows([][]interface{}{}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	_, _, err := repo.ListTasks(ctx, TaskFilters{}, 5, 10)

	require.NoError(t, err)
	require.Len(t, capturedArgs, 2)
	assert.Equal(t, 5, capturedArgs[0])
	assert.Equal(t, 10, capturedArgs[1])
}

func TestMockListTasks_QueryError(t *testing.T) {
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	rows, total, err := repo.ListTasks(ctx, TaskFilters{}, 10, 0)

	assert.Error(t, err)
	assert.Nil(t, rows)
	assert.Equal(t, 0, total)
}

func TestMockListTasks_DefaultPagination(t *testing.T) {
	var capturedArgs []interface{}
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			capturedArgs = args
			return common.NewMockRows([][]interface{}{}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	// limit=0 should use default
	_, _, err := repo.ListTasks(ctx, TaskFilters{}, 0, 0)

	require.NoError(t, err)
	require.Len(t, capturedArgs, 2)
	assert.Equal(t, 0, capturedArgs[0])
}
