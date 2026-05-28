// DEPRECATED: Use baxi/internal/repository/alert instead.
// Package repository provides data access for ops schema tables.
package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"baxi/internal/repository/alert"
	"baxi/internal/repository/common"
)

// AlertRow represents a single row from ops.metric_alert.
type AlertRow = alert.AlertRow

// sortMap defines allowed sort fields and their default order.
var sortMap = alert.SortMap

// AlertRepository handles read queries for ops.metric_alert.
// DEPRECATED: Use alert.Repository instead.
type AlertRepository struct {
	inner *alert.Repository
}

// NewAlertRepository creates a new AlertRepository.
// DEPRECATED: Use alert.NewRepository instead.
func NewAlertRepository() *AlertRepository {
	return &AlertRepository{}
}

func (r *AlertRepository) ensureInit(pool *pgxpool.Pool) {
	if r.inner == nil {
		r.inner = alert.NewRepository(common.NewPoolProvider(pool))
	}
}

// ListAlerts queries ops.metric_alert with optional filters, pagination, and sorting.
// Returns the matching rows and total count (unaffected by LIMIT/OFFSET).
func (r *AlertRepository) ListAlerts(
	ctx context.Context,
	pool *pgxpool.Pool,
	severity, status, objectType, ruleID, sort string,
	limit, offset int,
) ([]AlertRow, int, error) {
	r.ensureInit(pool)
	return r.inner.ListAlerts(ctx, severity, status, objectType, ruleID, sort, limit, offset)
}

// GetAlertByID retrieves a single alert by its ID.
func (r *AlertRepository) GetAlertByID(ctx context.Context, pool *pgxpool.Pool, alertID string) (*AlertRow, error) {
	r.ensureInit(pool)
	return r.inner.GetAlertByID(ctx, alertID)
}

// QueryAlerts is a convenience wrapper that accepts pgx.Tx for pipeline contexts.
func (r *AlertRepository) QueryAlerts(
	ctx context.Context,
	tx pgx.Tx,
	severity, status, objectType, ruleID, sort string,
	limit, offset int,
) ([]AlertRow, int, error) {
	return r.inner.QueryAlerts(ctx, tx, severity, status, objectType, ruleID, sort, limit, offset)
}
