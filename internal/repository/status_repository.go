// DEPRECATED: Use baxi/internal/repository/status instead.
// This file is a compatibility layer during migration.

package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"baxi/internal/repository/common"
	statusRepo "baxi/internal/repository/status"
)

// TableCount maps a response table name to its row count.
// DEPRECATED: Use status.TableCount instead.
type TableCount = statusRepo.TableCount

// PipelineRunRow represents the last pipeline execution from audit.pipeline_run.
// DEPRECATED: Use status.PipelineRunRow instead.
type PipelineRunRow = statusRepo.PipelineRunRow

// StatusRepository handles read queries for system status aggregation (DEPRECATED).
// Use status.Repository instead for new code.
type StatusRepository struct {
	inner *statusRepo.Repository
}

// NewStatusRepository creates a new StatusRepository (DEPRECATED).
func NewStatusRepository() *StatusRepository {
	return &StatusRepository{}
}

// SetPool initializes the inner repository with a pool provider.
func (r *StatusRepository) SetPool(pool *pgxpool.Pool) {
	r.inner = statusRepo.NewRepository(common.NewPoolProvider(pool))
}

// ensureInitialized lazily initializes the inner repo if needed.
func (r *StatusRepository) ensureInitialized(pool *pgxpool.Pool) *statusRepo.Repository {
	if r.inner == nil {
		r.SetPool(pool)
	}
	return r.inner
}

// GetTableCounts queries row counts from all tracked tables (DEPRECATED).
func (r *StatusRepository) GetTableCounts(ctx context.Context, pool *pgxpool.Pool) ([]TableCount, error) {
	return r.ensureInitialized(pool).GetTableCounts(ctx)
}

// GetLastPipelineRun queries the most recent pipeline run from audit.pipeline_run (DEPRECATED).
func (r *StatusRepository) GetLastPipelineRun(ctx context.Context, pool *pgxpool.Pool) (*PipelineRunRow, error) {
	return r.ensureInitialized(pool).GetLastPipelineRun(ctx)
}
