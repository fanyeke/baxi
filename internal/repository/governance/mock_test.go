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

func TestMockGetConfigSnapshots(t *testing.T) {
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{
				{"object_schema"},
				{"access_policy"},
			}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	snapshots, err := repo.GetConfigSnapshots(ctx)

	require.NoError(t, err)
	assert.Len(t, snapshots, 2)
	assert.Equal(t, "object_schema", snapshots[0].ConfigKey)
	assert.Equal(t, "loaded", snapshots[0].Status)
}

func TestMockGetConfigSnapshots_Empty(t *testing.T) {
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	snapshots, err := repo.GetConfigSnapshots(ctx)

	require.NoError(t, err)
	assert.Empty(t, snapshots)
}

func TestMockCountTableRows(t *testing.T) {
	mock := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRow(42)
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	count := repo.CountTableRows(ctx, "gov", "config_snapshot")

	assert.Equal(t, 42, count)
}

func TestMockCountTableRows_Error(t *testing.T) {
	mock := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRowError(fmt.Errorf("relation does not exist"))
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	count := repo.CountTableRows(ctx, "gov", "nonexistent")

	assert.Equal(t, 0, count)
}

func TestMockGetObjectSchemas(t *testing.T) {
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{
				{"order", "Order", []byte(`{"type":"object"}`), "v1"},
				{"product", "Product", []byte(`{"type":"object"}`), "v1"},
			}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	schemas, err := repo.GetObjectSchemas(ctx)

	require.NoError(t, err)
	assert.Len(t, schemas, 2)
	assert.Equal(t, "order", schemas[0].ObjectType)
	assert.Equal(t, "v1", schemas[0].Version)
}

func TestMockCountObjectSchemas(t *testing.T) {
	mock := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRow(5)
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	count := repo.CountObjectSchemas(ctx)

	assert.Equal(t, 5, count)
}

func TestMockGetDataClassifications(t *testing.T) {
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{
				{"user.email", "confidential", 0.8, "Email address"},
			}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	rows, err := repo.GetDataClassifications(ctx)

	require.NoError(t, err)
	assert.Len(t, rows, 1)
	assert.Equal(t, "user.email", rows[0].FieldPath)
	assert.Equal(t, "confidential", rows[0].ClassificationLevel)
}

func TestMockGetByFieldPath(t *testing.T) {
	mock := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRow("user.name", "public", 0.1, "User display name")
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	row, err := repo.GetByFieldPath(ctx, "user.name")

	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "public", row.ClassificationLevel)
}

func TestMockGetByFieldPath_NotFound(t *testing.T) {
	mock := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRowError(fmt.Errorf("no rows in result set"))
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	row, err := repo.GetByFieldPath(ctx, "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, row)
}
