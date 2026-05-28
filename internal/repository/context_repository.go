// DEPRECATED: Use baxi/internal/repository/context instead.
// This file is a compatibility layer during migration.

package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"baxi/internal/repository/common"
	contextRepo "baxi/internal/repository/context"
)

// ContextRepo provides context data queries (DEPRECATED).
// Use context.Repository instead for new code.
type ContextRepo struct {
	inner *contextRepo.Repository
}

// NewContextRepository creates a new ContextRepo (DEPRECATED).
func NewContextRepository() *ContextRepo {
	return &ContextRepo{}
}

// SetPool initializes the inner repository with a pool provider.
func (r *ContextRepo) SetPool(pool *pgxpool.Pool) {
	r.inner = contextRepo.NewRepository(common.NewPoolProvider(pool))
}

// ensureInitialized lazily initializes the inner repo if needed.
func (r *ContextRepo) ensureInitialized(pool *pgxpool.Pool) *contextRepo.Repository {
	if r.inner == nil {
		r.SetPool(pool)
	}
	return r.inner
}

// GetLastPipelineRun queries the most recent pipeline run (DEPRECATED).
func (r *ContextRepo) GetLastPipelineRun(ctx context.Context, pool *pgxpool.Pool) (*PipelineRunInfo, error) {
	if pool == nil {
		return nil, nil
	}
	info, err := r.ensureInitialized(pool).GetLastPipelineRun(ctx)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, nil
	}
	return &PipelineRunInfo{
		RunID:       info.RunID,
		Status:      info.Status,
		StartedAt:   info.StartedAt,
		CompletedAt: info.CompletedAt,
	}, nil
}

// GetAlerts queries metric alerts with optional severity filter (DEPRECATED).
func (r *ContextRepo) GetAlerts(ctx context.Context, pool *pgxpool.Pool, severity string, limit int) ([]AlertSummary, error) {
	if pool == nil {
		return []AlertSummary{}, nil
	}
	items, err := r.ensureInitialized(pool).GetAlerts(ctx, severity, limit)
	if err != nil {
		return nil, err
	}
	results := make([]AlertSummary, len(items))
	for i, item := range items {
		results[i] = AlertSummary{
			AlertID:  item.AlertID,
			Severity: item.Severity,
			Metric:   item.Metric,
			Status:   item.Status,
		}
	}
	return results, nil
}

// GetOpenTasks queries open (todo/in_progress) tasks (DEPRECATED).
func (r *ContextRepo) GetOpenTasks(ctx context.Context, pool *pgxpool.Pool, limit int) ([]TaskSummary, error) {
	if pool == nil {
		return []TaskSummary{}, nil
	}
	items, err := r.ensureInitialized(pool).GetOpenTasks(ctx, limit)
	if err != nil {
		return nil, err
	}
	results := make([]TaskSummary, len(items))
	for i, item := range items {
		results[i] = TaskSummary{
			TaskID:    item.TaskID,
			Title:     item.Title,
			Status:    item.Status,
			OwnerRole: item.OwnerRole,
		}
	}
	return results, nil
}

// GetPendingOutbox queries pending outbox events (DEPRECATED).
func (r *ContextRepo) GetPendingOutbox(ctx context.Context, pool *pgxpool.Pool, limit int) ([]OutboxSummary, error) {
	if pool == nil {
		return []OutboxSummary{}, nil
	}
	items, err := r.ensureInitialized(pool).GetPendingOutbox(ctx, limit)
	if err != nil {
		return nil, err
	}
	results := make([]OutboxSummary, len(items))
	for i, item := range items {
		results[i] = OutboxSummary{
			EventID:   item.EventID,
			EventType: item.EventType,
			Status:    item.Status,
		}
	}
	return results, nil
}
