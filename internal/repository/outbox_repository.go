// Package repository provides data access for querying database tables.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OutboxRow represents a single row from ops.outbox_event for read queries.
// Field names match the DB column naming (event_id is mapped to OutboxID for Go convention).
type OutboxRow struct {
	OutboxID         string     `db:"event_id"`
	EventType        string     `db:"event_type"`
	SourceType       string     `db:"source_type"`
	SourceID         string     `db:"source_id"`
	TargetChannel    string     `db:"target_channel"`
	Status           string     `db:"status"`
	CreatedAt        time.Time  `db:"created_at"`
	DispatchAttempts int        `db:"dispatch_attempts"`
	LastDispatchAt   *time.Time `db:"last_dispatch_at"`
}

// OutboxFilters holds optional WHERE clause filters for listing outbox events.
// Only non-nil fields are applied to the query.
type OutboxFilters struct {
	Status    *string
	Channel   *string
	EventType *string
}

// OutboxRepository provides read-only access to ops.outbox_event.
// This is separate from internal/outbox/repository.go (which handles writes) to avoid
// coupling the pipeline's write repository with the API's read repository.
type OutboxRepository struct{}

// NewOutboxRepository creates a new OutboxRepository.
func NewOutboxRepository() *OutboxRepository {
	return &OutboxRepository{}
}

// OutboxDetail represents a full outbox event row for detail/management queries.
type OutboxDetail struct {
	EventID          string         `db:"event_id"`
	EventType        string         `db:"event_type"`
	SourceType       string         `db:"source_type"`
	SourceID         string         `db:"source_id"`
	TargetChannel    string         `db:"target_channel"`
	Status           string         `db:"status"`
	Payload          []byte         `db:"payload_json"`
	CreatedAt        time.Time      `db:"created_at"`
	DispatchAttempts int            `db:"dispatch_attempts"`
	LastDispatchAt   *time.Time     `db:"last_dispatch_at"`
	ErrorMessage     *string        `db:"error_message"`
}

// GetDetail retrieves a full outbox event row by event_id.
// Returns nil, nil if not found.
func (r *OutboxRepository) GetDetail(ctx context.Context, pool *pgxpool.Pool, eventID string) (*OutboxDetail, error) {
	var d OutboxDetail
	err := pool.QueryRow(ctx, `
		SELECT event_id, event_type, source_type, source_id,
		       target_channel, status, payload_json, created_at,
		       dispatch_attempts, last_dispatch_at, error_message
		FROM ops.outbox_event
		WHERE event_id = $1
	`, eventID).Scan(
		&d.EventID, &d.EventType, &d.SourceType, &d.SourceID,
		&d.TargetChannel, &d.Status, &d.Payload, &d.CreatedAt,
		&d.DispatchAttempts, &d.LastDispatchAt, &d.ErrorMessage,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get outbox event detail %s: %w", eventID, err)
	}
	return &d, nil
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
	query := `
		SELECT event_id, event_type, source_type, source_id,
		       target_channel, status, created_at,
		       dispatch_attempts, last_dispatch_at,
		       COUNT(*) OVER() AS total_count
		FROM ops.outbox_event
		WHERE 1=1`

	args := make([]interface{}, 0, 4)
	argIdx := 1

	if filters.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *filters.Status)
		argIdx++
	}
	if filters.Channel != nil {
		query += fmt.Sprintf(" AND target_channel = $%d", argIdx)
		args = append(args, *filters.Channel)
		argIdx++
	}
	if filters.EventType != nil {
		query += fmt.Sprintf(" AND event_type = $%d", argIdx)
		args = append(args, *filters.EventType)
		argIdx++
	}

	query += " ORDER BY created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query outbox events: %w", err)
	}
	defer rows.Close()

	var items []OutboxRow
	var total int

	for rows.Next() {
		var item OutboxRow
		var rowTotal int
		if err := rows.Scan(
			&item.OutboxID,
			&item.EventType,
			&item.SourceType,
			&item.SourceID,
			&item.TargetChannel,
			&item.Status,
			&item.CreatedAt,
			&item.DispatchAttempts,
			&item.LastDispatchAt,
			&rowTotal,
		); err != nil {
			return nil, 0, fmt.Errorf("scan outbox row: %w", err)
		}
		items = append(items, item)
		total = rowTotal // all rows share the same total_count from COUNT(*) OVER()
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate outbox rows: %w", err)
	}

	return items, total, nil
}
