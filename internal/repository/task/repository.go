// Package task provides repository access for the task domain.
// This is a domain subpackage of the repository layer with pool injection.
package task

import (
	"context"
	"fmt"
	"time"

	"baxi/internal/repository/common"
)

// TaskRow represents a single row from ops.task.
type TaskRow struct {
	TaskID           string
	RecommendationID *string
	AlertID          *string // maps to event_id in API response
	TaskTitle        string
	TaskDescription  *string
	TargetObjectType *string
	TargetObjectID   *string
	OwnerRole        *string
	OwnerUserID      *string
	Priority         string
	DueAt            *time.Time
	Status           string
	Feedback         *string
	CompletedAt      *time.Time
	CreatedAt        time.Time
}

// TaskFilters holds optional WHERE clause filters for listing tasks.
// Only non-nil fields are applied to the query.
type TaskFilters struct {
	Status   *string
	Priority *string
	Owner    *string // maps to owner_role
}

// Repository provides read-only access to ops.task.
type Repository struct {
	*common.PoolProvider
}

// NewRepository creates a new task Repository.
func NewRepository(provider *common.PoolProvider) *Repository {
	return &Repository{PoolProvider: provider}
}

// ListTasks queries ops.task with optional filters and pagination.
// Uses COUNT(*) OVER() to return the total count in a single query.
// Results are ordered by created_at DESC.
func (r *Repository) ListTasks(
	ctx context.Context,
	filters TaskFilters,
	limit, offset int,
) ([]TaskRow, int, error) {
	query := `
		SELECT task_id, recommendation_id, alert_id,
		       task_title, task_description,
		       target_object_type, target_object_id,
		       owner_role, owner_user_id,
		       priority, due_at, status, feedback, completed_at, created_at,
		       COUNT(*) OVER() AS total_count
		FROM ops.task
		WHERE 1=1`

	args := make([]interface{}, 0, 6)
	argIdx := 1

	if filters.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *filters.Status)
		argIdx++
	}
	if filters.Priority != nil {
		query += fmt.Sprintf(" AND priority = $%d", argIdx)
		args = append(args, *filters.Priority)
		argIdx++
	}
	if filters.Owner != nil {
		query += fmt.Sprintf(" AND owner_role = $%d", argIdx)
		args = append(args, *filters.Owner)
		argIdx++
	}

	query += " ORDER BY created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query ops.task: %w", err)
	}
	defer rows.Close()

	var results []TaskRow
	var total int

	for rows.Next() {
		var row TaskRow
		var rowTotal int
		if err := rows.Scan(
			&row.TaskID,
			&row.RecommendationID,
			&row.AlertID,
			&row.TaskTitle,
			&row.TaskDescription,
			&row.TargetObjectType,
			&row.TargetObjectID,
			&row.OwnerRole,
			&row.OwnerUserID,
			&row.Priority,
			&row.DueAt,
			&row.Status,
			&row.Feedback,
			&row.CompletedAt,
			&row.CreatedAt,
			&rowTotal,
		); err != nil {
			return nil, 0, fmt.Errorf("scan task row: %w", err)
		}
		results = append(results, row)
		total = rowTotal
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate task rows: %w", err)
	}

	return results, total, nil
}
