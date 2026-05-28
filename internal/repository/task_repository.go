// DEPRECATED: Use baxi/internal/repository/task instead.
// This file is a compatibility layer during migration.

package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"baxi/internal/repository/common"
	taskRepo "baxi/internal/repository/task"
)

// TaskRow represents a single row from ops.task.
// DEPRECATED: Use task.TaskRow instead.
type TaskRow = taskRepo.TaskRow

// TaskFilters holds optional WHERE clause filters for listing tasks.
// DEPRECATED: Use task.TaskFilters instead.
type TaskFilters = taskRepo.TaskFilters

// TaskRepository provides read-only access to ops.task (DEPRECATED).
// Use task.Repository instead for new code.
type TaskRepository struct {
	inner *taskRepo.Repository
}

// NewTaskRepository creates a new TaskRepository (DEPRECATED).
func NewTaskRepository() *TaskRepository {
	return &TaskRepository{}
}

// SetPool initializes the inner repository with a pool provider.
func (r *TaskRepository) SetPool(pool *pgxpool.Pool) {
	r.inner = taskRepo.NewRepository(common.NewPoolProvider(pool))
}

// ensureInitialized lazily initializes the inner repo if needed.
func (r *TaskRepository) ensureInitialized(pool *pgxpool.Pool) *taskRepo.Repository {
	if r.inner == nil {
		r.SetPool(pool)
	}
	return r.inner
}

// ListTasks queries ops.task with optional filters and pagination (DEPRECATED).
// Uses COUNT(*) OVER() to return the total count in a single query.
// Results are ordered by created_at DESC.
func (r *TaskRepository) ListTasks(
	ctx context.Context,
	pool *pgxpool.Pool,
	filters TaskFilters,
	limit, offset int,
) ([]TaskRow, int, error) {
	return r.ensureInitialized(pool).ListTasks(ctx, filters, limit, offset)
}
