// Package repository provides data access for audit schema tables.
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// LogRow represents a single row from a combined log query.
type LogRow struct {
	LogType   string    `db:"log_type"`
	Level     string    `db:"level"`
	Message   string    `db:"message"`
	RequestID *string   `db:"request_id"`
	CreatedAt time.Time `db:"created_at"`
}

// LogRepository provides read-only access to audit log tables.
type LogRepository struct{}

// NewLogRepository creates a new LogRepository.
func NewLogRepository() *LogRepository {
	return &LogRepository{}
}

// ListRecentLogs returns a combined, chronologically-ordered view of API request logs,
// pipeline runs, and pipeline step runs. Results are ordered by created_at DESC.
func (r *LogRepository) ListRecentLogs(
	ctx context.Context,
	pool *pgxpool.Pool,
	limit, offset int,
) ([]LogRow, int, error) {
	query := `
		SELECT log_type, level, message, request_id, created_at, total_count
		FROM (
			SELECT
				'api_request' AS log_type,
				CASE
					WHEN status_code >= 200 AND status_code < 300 THEN 'info'
					WHEN status_code >= 300 AND status_code < 400 THEN 'warn'
					ELSE 'error'
				END AS level,
				method || ' ' || path AS message,
				request_id,
				created_at,
				COUNT(*) OVER() AS total_count
			FROM audit.api_request_log
			UNION ALL
			SELECT
				'pipeline_run' AS log_type,
				CASE
					WHEN status = 'completed' THEN 'info'
					WHEN status = 'failed' THEN 'error'
					ELSE 'warn'
				END AS level,
				run_type || '/' || mode AS message,
				NULL::TEXT AS request_id,
				started_at AS created_at,
				COUNT(*) OVER() AS total_count
			FROM audit.pipeline_run
			UNION ALL
			SELECT
				'pipeline_step' AS log_type,
				CASE
					WHEN status = 'completed' THEN 'info'
					WHEN status = 'failed' THEN 'error'
					ELSE 'warn'
				END AS level,
				COALESCE(step_name, 'unknown') AS message,
				NULL::TEXT AS request_id,
				started_at AS created_at,
				COUNT(*) OVER() AS total_count
			FROM audit.pipeline_step_run
		) AS combined
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	return r.queryLogs(ctx, pool, query, limit, offset)
}

// ListErrorLogs returns error records from audit.error_log and failed pipeline step runs.
// Results are ordered by created_at DESC.
func (r *LogRepository) ListErrorLogs(
	ctx context.Context,
	pool *pgxpool.Pool,
	limit, offset int,
) ([]LogRow, int, error) {
	query := `
		SELECT log_type, level, message, request_id, created_at, total_count
		FROM (
			SELECT
				'error_log' AS log_type,
				'error' AS level,
				COALESCE(error_message, '') AS message,
				request_id,
				created_at,
				COUNT(*) OVER() AS total_count
			FROM audit.error_log
			UNION ALL
			SELECT
				'pipeline_step' AS log_type,
				'error' AS level,
				COALESCE(step_name, 'unknown') ||
					CASE WHEN COALESCE(error_message, '') = '' THEN ''
					ELSE ': ' || error_message
					END AS message,
				NULL::TEXT AS request_id,
				COALESCE(finished_at, started_at) AS created_at,
				COUNT(*) OVER() AS total_count
			FROM audit.pipeline_step_run
			WHERE error_message IS NOT NULL AND error_message != ''
		) AS combined
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	return r.queryLogs(ctx, pool, query, limit, offset)
}

// ListAuditLogs returns business audit trail entries from audit.audit_log.
// Results are ordered by created_at DESC.
func (r *LogRepository) ListAuditLogs(
	ctx context.Context,
	pool *pgxpool.Pool,
	limit, offset int,
) ([]LogRow, int, error) {
	query := `
		SELECT
			'audit_log' AS log_type,
			'info' AS level,
			COALESCE(action, '') || ' on ' || COALESCE(resource_type, 'unknown') AS message,
			NULL::TEXT AS request_id,
			created_at,
			COUNT(*) OVER() AS total_count
		FROM audit.audit_log
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	return r.queryLogs(ctx, pool, query, limit, offset)
}

// queryLogs executes a log query and scans rows into LogRow results.
func (r *LogRepository) queryLogs(
	ctx context.Context,
	pool *pgxpool.Pool,
	query string,
	limit, offset int,
) ([]LogRow, int, error) {
	rows, err := pool.Query(ctx, query, limit, offset)
	if err != nil {
		return []LogRow{}, 0, nil
	}
	defer rows.Close()

	var results []LogRow
	var totalCount int

	for rows.Next() {
		var row LogRow
		var total int
		if err := rows.Scan(
			&row.LogType,
			&row.Level,
			&row.Message,
			&row.RequestID,
			&row.CreatedAt,
			&total,
		); err != nil {
			return nil, 0, fmt.Errorf("scan log row: %w", err)
		}
		results = append(results, row)
		totalCount = total
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate log rows: %w", err)
	}

	if results == nil {
		results = []LogRow{}
	}

	return results, totalCount, nil
}
