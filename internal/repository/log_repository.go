// DEPRECATED: Use baxi/internal/repository/log instead.
// Package repository provides data access for audit schema tables.
package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"baxi/internal/repository/common"
	"baxi/internal/repository/log"
)

// LogRow represents a single row from a combined log query.
type LogRow = log.LogRow

// LogRepository provides read-only access to audit log tables.
// DEPRECATED: Use log.Repository instead.
type LogRepository struct {
	inner *log.Repository
}

// NewLogRepository creates a new LogRepository.
// DEPRECATED: Use log.NewRepository instead.
func NewLogRepository() *LogRepository {
	return &LogRepository{}
}

func (r *LogRepository) ensureInit(pool *pgxpool.Pool) {
	if r.inner == nil {
		r.inner = log.NewRepository(common.NewPoolProvider(pool))
	}
}

// ListRecentLogs returns a combined, chronologically-ordered view of API request logs,
// pipeline runs, and pipeline step runs. Results are ordered by created_at DESC.
func (r *LogRepository) ListRecentLogs(
	ctx context.Context,
	pool *pgxpool.Pool,
	limit, offset int,
) ([]LogRow, int, error) {
	r.ensureInit(pool)
	return r.inner.ListRecentLogs(ctx, limit, offset)
}

// ListErrorLogs returns error records from audit.error_log and failed pipeline step runs.
// Results are ordered by created_at DESC.
func (r *LogRepository) ListErrorLogs(
	ctx context.Context,
	pool *pgxpool.Pool,
	limit, offset int,
) ([]LogRow, int, error) {
	r.ensureInit(pool)
	return r.inner.ListErrorLogs(ctx, limit, offset)
}

// ListAuditLogs returns business audit trail entries from audit.audit_log.
// Results are ordered by created_at DESC.
func (r *LogRepository) ListAuditLogs(
	ctx context.Context,
	pool *pgxpool.Pool,
	limit, offset int,
) ([]LogRow, int, error) {
	r.ensureInit(pool)
	return r.inner.ListAuditLogs(ctx, limit, offset)
}
