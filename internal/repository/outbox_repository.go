// DEPRECATED: Use baxi/internal/repository/outbox instead.
// Package repository provides data access for querying database tables.
package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"baxi/internal/repository/common"
	"baxi/internal/repository/outbox"
)

// OutboxRow represents a single row from ops.outbox_event for read queries.
// Field names match the DB column naming (event_id is mapped to OutboxID for Go convention).
type OutboxRow = outbox.OutboxRow

// OutboxFilters holds optional WHERE clause filters for listing outbox events.
// Only non-nil fields are applied to the query.
type OutboxFilters = outbox.OutboxFilters

// OutboxDetail represents a full outbox event row for detail/management queries.
type OutboxDetail = outbox.OutboxDetail

// OutboxRepository provides read-only access to ops.outbox_event.
// This is separate from internal/outbox/repository.go (which handles writes) to avoid
// coupling the pipeline's write repository with the API's read repository.
// DEPRECATED: Use outbox.Repository instead.
type OutboxRepository struct {
	inner *outbox.Repository
}

// NewOutboxRepository creates a new OutboxRepository.
// DEPRECATED: Use outbox.NewRepository instead.
func NewOutboxRepository() *OutboxRepository {
	return &OutboxRepository{}
}

func (r *OutboxRepository) ensureInit(pool *pgxpool.Pool) {
	if r.inner == nil {
		r.inner = outbox.NewRepository(common.NewPoolProvider(pool))
	}
}

// GetDetail retrieves a full outbox event row by event_id.
// Returns nil, nil if not found.
func (r *OutboxRepository) GetDetail(ctx context.Context, pool *pgxpool.Pool, eventID string) (*OutboxDetail, error) {
	r.ensureInit(pool)
	return r.inner.GetDetail(ctx, eventID)
}

// ListOutboxEvents queries ops.outbox_event with optional filters and pagination.
// Uses COUNT(*) OVER() to return the total count matching the filters in a single query.
// Results are ordered by created_at DESC.
func (r *OutboxRepository) ListOutboxEvents(
	ctx context.Context,
	pool *pgxpool.Pool,
	filters OutboxFilters,
	limit, offset int,
) ([]OutboxRow, int, error) {
	r.ensureInit(pool)
	return r.inner.ListOutboxEvents(ctx, filters, limit, offset)
}
