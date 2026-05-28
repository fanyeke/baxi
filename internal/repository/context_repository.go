package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ctxRepo struct{}

var _ ContextRepository = (*ctxRepo)(nil)

func NewContextRepository() *ctxRepo {
	return &ctxRepo{}
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

func (r *ctxRepo) GetLastPipelineRun(ctx context.Context, pool *pgxpool.Pool) (*PipelineRunInfo, error) {
	if pool == nil {
		return nil, nil
	}

	query := `
		SELECT run_id, run_type, mode, status, started_at,
		       finished_at, input_count, output_count, error_message
		FROM audit.pipeline_run
		ORDER BY started_at DESC
		LIMIT 1
	`

	var row pipelineRunRow
	err := pool.QueryRow(ctx, query).Scan(
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

func (r *ctxRepo) GetAlerts(ctx context.Context, pool *pgxpool.Pool, severity string, limit int) ([]AlertSummary, error) {
	if pool == nil {
		return []AlertSummary{}, nil
	}

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

	rows, err := pool.Query(ctx, query, args...)
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

func (r *ctxRepo) GetOpenTasks(ctx context.Context, pool *pgxpool.Pool, limit int) ([]TaskSummary, error) {
	if pool == nil {
		return []TaskSummary{}, nil
	}

	openStatuses := []string{"todo", "in_progress"}

	query := `
		SELECT task_id, task_title, status, owner_role
		FROM ops.task
		WHERE status = ANY($1)
		ORDER BY priority DESC, created_at DESC
		LIMIT $2
	`

	rows, err := pool.Query(ctx, query, openStatuses, limit)
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

func (r *ctxRepo) GetPendingOutbox(ctx context.Context, pool *pgxpool.Pool, limit int) ([]OutboxSummary, error) {
	if pool == nil {
		return []OutboxSummary{}, nil
	}

	pendingStatus := "pending"

	query := `
		SELECT event_id, event_type, status
		FROM ops.outbox_event
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := pool.Query(ctx, query, pendingStatus, limit)
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
