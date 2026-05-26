package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TableCount maps a response table name to its row count.
type TableCount struct {
	TableName string
	RowCount  int
}

// PipelineRunRow represents the last pipeline execution from audit.pipeline_run.
type PipelineRunRow struct {
	RunID        string
	RunType      string
	Mode         string
	Status       string
	StartedAt    string
	FinishedAt   *string
	InputCount   int64
	OutputCount  int64
	ErrorMessage *string
}

// StatusRepository handles read queries for system status aggregation.
type StatusRepository struct{}

// NewStatusRepository creates a new StatusRepository.
func NewStatusRepository() *StatusRepository {
	return &StatusRepository{}
}

// GetTableCounts queries row counts from all tracked tables.
// Uses a single UNION ALL query for efficiency.
// Returns old (backward-compatible) table names with new (PostgreSQL) table queries.
func (r *StatusRepository) GetTableCounts(ctx context.Context, pool *pgxpool.Pool) ([]TableCount, error) {
	query := `
		SELECT table_name, row_count FROM (
			SELECT 'alert_events' AS table_name, COUNT(*) AS row_count FROM ops.metric_alert
			UNION ALL
			SELECT 'action_tasks', COUNT(*) FROM ops.task
			UNION ALL
			SELECT 'event_outbox', COUNT(*) FROM ops.outbox_event
			UNION ALL
			SELECT 'dwd_order_level', COUNT(*) FROM dwd.order_level
			UNION ALL
			SELECT 'dwd_item_level', COUNT(*) FROM dwd.item_level
			UNION ALL
			SELECT 'metric_daily', COUNT(*) FROM mart.metric_daily
			UNION ALL
			SELECT 'metric_dimension_daily', COUNT(*) FROM mart.metric_dimension_daily
		) AS counts
		ORDER BY table_name
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query table counts: %w", err)
	}
	defer rows.Close()

	var results []TableCount
	for rows.Next() {
		var tc TableCount
		if err := rows.Scan(&tc.TableName, &tc.RowCount); err != nil {
			return nil, fmt.Errorf("scan table count: %w", err)
		}
		results = append(results, tc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate table counts: %w", err)
	}

	if results == nil {
		results = []TableCount{}
	}

	return results, nil
}

// GetLastPipelineRun queries the most recent pipeline run from audit.pipeline_run.
func (r *StatusRepository) GetLastPipelineRun(ctx context.Context, pool *pgxpool.Pool) (*PipelineRunRow, error) {
	query := `
		SELECT run_id, run_type, mode, status,
		       started_at::TEXT, finished_at::TEXT,
		       COALESCE(input_count, 0), COALESCE(output_count, 0),
		       error_message
		FROM audit.pipeline_run
		ORDER BY started_at DESC
		LIMIT 1
	`

	row := pool.QueryRow(ctx, query)

	var pr PipelineRunRow
	err := row.Scan(
		&pr.RunID, &pr.RunType, &pr.Mode, &pr.Status,
		&pr.StartedAt, &pr.FinishedAt,
		&pr.InputCount, &pr.OutputCount,
		&pr.ErrorMessage,
	)
	if err != nil {
		return nil, fmt.Errorf("query last pipeline run: %w", err)
	}

	return &pr, nil
}
