// Package context provides repository access for Qoder context data queries.
// This is a domain subpackage of the repository layer with pool injection.
package context

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"baxi/internal/repository/common"
)

// PipelineRunInfo summarizes a pipeline execution.
type PipelineRunInfo struct {
	RunID       int64
	Status      string
	StartedAt   string
	CompletedAt string
}

// AlertSummary is a compact representation of an alert.
type AlertSummary struct {
	AlertID  string
	Severity string
	Metric   string
	Status   string
}

// TaskSummary is a compact representation of a task.
type TaskSummary struct {
	TaskID    string
	Title     string
	Status    string
	OwnerRole string
}

// OutboxSummary is a compact representation of an outbox event.
type OutboxSummary struct {
	EventID   string
	EventType string
	Status    string
}

// Repository provides context data queries for the Qoder decision engine.
type Repository struct {
	common.Querier
}

// NewRepository creates a new context Repository.
func NewRepository(provider common.Querier) *Repository {
	return &Repository{Querier: provider}
}

type pipelineRunRow struct {
	RunID        string
	RunType      string
	Mode         string
	Status       string
	StartedAt    time.Time
	FinishedAt   *time.Time
	InputCount   int64
	OutputCount  int64
	ErrorMessage *string
}

// GetLastPipelineRun queries the most recent pipeline run from audit.pipeline_run.
func (r *Repository) GetLastPipelineRun(ctx context.Context) (*PipelineRunInfo, error) {
	query := `
		SELECT run_id, run_type, mode, status, started_at,
		       finished_at, input_count, output_count, error_message
		FROM audit.pipeline_run
		ORDER BY started_at DESC
		LIMIT 1
	`

	var row pipelineRunRow
	err := r.QueryRow(ctx, query).Scan(
		&row.RunID,
		&row.RunType,
		&row.Mode,
		&row.Status,
		&row.StartedAt,
		&row.FinishedAt,
		&row.InputCount,
		&row.OutputCount,
		&row.ErrorMessage,
	)
	if err != nil {
		return nil, nil
	}

	var completedAt string
	if row.FinishedAt != nil {
		completedAt = row.FinishedAt.Format(time.RFC3339Nano)
	}
	startedAt := row.StartedAt.Format(time.RFC3339Nano)

	var runID int64
	if parsed, err := strconv.ParseInt(row.RunID, 10, 64); err == nil {
		runID = parsed
	}

	return &PipelineRunInfo{
		RunID:       runID,
		Status:      row.Status,
		StartedAt:   startedAt,
		CompletedAt: completedAt,
	}, nil
}

// GetAlerts queries metric alerts with optional severity filter.
func (r *Repository) GetAlerts(ctx context.Context, severity string, limit int) ([]AlertSummary, error) {
	query := `
		SELECT alert_id, severity, metric_name, status
		FROM ops.metric_alert
	`

	args := []interface{}{}
	argIdx := 1

	if severity != "" {
		query += fmt.Sprintf(" WHERE severity = $%d", argIdx)
		args = append(args, severity)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY severity DESC, event_date DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := r.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query alerts: %w", err)
	}
	defer rows.Close()

	var items []AlertSummary
	for rows.Next() {
		var alertID, sev, metric, status string
		if err := rows.Scan(&alertID, &sev, &metric, &status); err != nil {
			continue
		}
		items = append(items, AlertSummary{
			AlertID:  alertID,
			Severity: sev,
			Metric:   metric,
			Status:   status,
		})
	}

	if items == nil {
		items = []AlertSummary{}
	}

	return items, nil
}

// GetOpenTasks queries open (todo/in_progress) tasks with a limit.
func (r *Repository) GetOpenTasks(ctx context.Context, limit int) ([]TaskSummary, error) {
	openStatuses := []string{"todo", "in_progress"}

	query := `
		SELECT task_id, task_title, status, owner_role
		FROM ops.task
		WHERE status = ANY($1)
		ORDER BY priority DESC, created_at DESC
		LIMIT $2
	`

	rows, err := r.Query(ctx, query, openStatuses, limit)
	if err != nil {
		return nil, fmt.Errorf("query open tasks: %w", err)
	}
	defer rows.Close()

	var items []TaskSummary
	for rows.Next() {
		var taskID, title, status string
		var ownerRole *string
		if err := rows.Scan(&taskID, &title, &status, &ownerRole); err != nil {
			continue
		}
		owner := ""
		if ownerRole != nil {
			owner = *ownerRole
		}
		items = append(items, TaskSummary{
			TaskID:    taskID,
			Title:     title,
			Status:    status,
			OwnerRole: owner,
		})
	}

	if items == nil {
		items = []TaskSummary{}
	}

	return items, nil
}

// GetPendingOutbox queries pending outbox events with a limit.
func (r *Repository) GetPendingOutbox(ctx context.Context, limit int) ([]OutboxSummary, error) {
	pendingStatus := "pending"

	query := `
		SELECT event_id, event_type, status
		FROM ops.outbox_event
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.Query(ctx, query, pendingStatus, limit)
	if err != nil {
		return nil, fmt.Errorf("query pending outbox: %w", err)
	}
	defer rows.Close()

	var items []OutboxSummary
	for rows.Next() {
		var eventID, eventType, status string
		if err := rows.Scan(&eventID, &eventType, &status); err != nil {
			continue
		}
		items = append(items, OutboxSummary{
			EventID:   eventID,
			EventType: eventType,
			Status:    status,
		})
	}

	if items == nil {
		items = []OutboxSummary{}
	}

	return items, nil
}
