// Package alert provides repository access for the alert domain.
// This is a domain subpackage of the repository layer with pool injection.
package alert

import (
	"context"
	"fmt"
	"strings"

	"baxi/internal/repository/common"
	"github.com/jackc/pgx/v5"
)

// Repository provides data access for alert events.
type Repository struct {
	common.Querier
}

// NewRepository creates a new alert repository.
func NewRepository(provider common.Querier) *Repository {
	return &Repository{Querier: provider}
}

// AlertRow represents a single row from ops.metric_alert.
type AlertRow struct {
	AlertID       string
	RuleID        string
	EventDate     string
	Severity      string
	MetricName    string
	ObjectType    string
	ObjectID      string
	CurrentValue  *float64
	BaselineValue *float64
	ChangeRate    *float64
	OwnerRole     string
	Status        string
	ImpactScore   *float64
}

// SortMap defines allowed sort fields and their default order.
// Keys are the API sort parameter values, values are SQL ORDER BY clauses.
var SortMap = map[string]string{
	"created_at_desc": "created_at DESC",
	"created_at_asc":  "created_at ASC",
	"severity_desc":   "severity DESC, created_at DESC",
}

// ListAlerts queries ops.metric_alert with optional filters, pagination, and sorting.
// Returns the matching rows and total count (unaffected by LIMIT/OFFSET).
func (r *Repository) ListAlerts(
	ctx context.Context,
	severity, status, objectType, ruleID, sort string,
	limit, offset int,
) ([]AlertRow, int, error) {
	// Resolve sort clause
	orderClause, ok := SortMap[sort]
	if !ok {
		orderClause = SortMap["created_at_desc"]
	}

	var (
		conditions []string
		args       []any
		argIdx     int
	)

	if severity != "" {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("severity = $%d", argIdx))
		args = append(args, severity)
	}
	if status != "" {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, status)
	}
	if objectType != "" {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("object_type = $%d", argIdx))
		args = append(args, objectType)
	}
	if ruleID != "" {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("rule_id = $%d", argIdx))
		args = append(args, ruleID)
	}

	argIdx++
	args = append(args, limit)
	argIdx++
	args = append(args, offset)

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT alert_id, rule_id, event_date::TEXT, severity, metric_name,
		       object_type, object_id, current_value, baseline_value,
		       change_rate, owner_role, status, impact_score,
		       COUNT(*) OVER() AS total_count
		FROM ops.metric_alert
		%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderClause, argIdx-1, argIdx)

	rows, err := r.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query metric_alert: %w", err)
	}
	defer rows.Close()

	var results []AlertRow
	var totalCount int

	for rows.Next() {
		var row AlertRow
		var total int
		if err := rows.Scan(
			&row.AlertID, &row.RuleID, &row.EventDate, &row.Severity,
			&row.MetricName, &row.ObjectType, &row.ObjectID,
			&row.CurrentValue, &row.BaselineValue, &row.ChangeRate,
			&row.OwnerRole, &row.Status, &row.ImpactScore,
			&total,
		); err != nil {
			return nil, 0, fmt.Errorf("scan alert row: %w", err)
		}
		results = append(results, row)
		totalCount = total
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate alert rows: %w", err)
	}

	if results == nil {
		results = []AlertRow{}
	}

	return results, totalCount, nil
}

// GetAlertByID retrieves a single alert by its ID.
func (r *Repository) GetAlertByID(ctx context.Context, alertID string) (*AlertRow, error) {
	query := `
		SELECT alert_id, rule_id, event_date::TEXT, severity, metric_name,
		       object_type, object_id, current_value, baseline_value,
		       change_rate, owner_role, status, impact_score
		FROM ops.metric_alert
		WHERE alert_id = $1
	`

	var row AlertRow
	err := r.QueryRow(ctx, query, alertID).Scan(
		&row.AlertID, &row.RuleID, &row.EventDate, &row.Severity,
		&row.MetricName, &row.ObjectType, &row.ObjectID,
		&row.CurrentValue, &row.BaselineValue, &row.ChangeRate,
		&row.OwnerRole, &row.Status, &row.ImpactScore,
	)
	if err != nil {
		return nil, fmt.Errorf("query metric_alert by id: %w", err)
	}

	return &row, nil
}

// QueryAlerts is a convenience wrapper that accepts pgx.Tx for pipeline contexts.
func (r *Repository) QueryAlerts(
	ctx context.Context,
	tx pgx.Tx,
	severity, status, objectType, ruleID, sort string,
	limit, offset int,
) ([]AlertRow, int, error) {
	orderClause, ok := SortMap[sort]
	if !ok {
		orderClause = SortMap["created_at_desc"]
	}

	var (
		conditions []string
		args       []any
		argIdx     int
	)

	if severity != "" {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("severity = $%d", argIdx))
		args = append(args, severity)
	}
	if status != "" {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, status)
	}
	if objectType != "" {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("object_type = $%d", argIdx))
		args = append(args, objectType)
	}
	if ruleID != "" {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("rule_id = $%d", argIdx))
		args = append(args, ruleID)
	}

	argIdx++
	args = append(args, limit)
	argIdx++
	args = append(args, offset)

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT alert_id, rule_id, event_date::TEXT, severity, metric_name,
		       object_type, object_id, current_value, baseline_value,
		       change_rate, owner_role, status, impact_score,
		       COUNT(*) OVER() AS total_count
		FROM ops.metric_alert
		%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderClause, argIdx-1, argIdx)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query metric_alert: %w", err)
	}
	defer rows.Close()

	var results []AlertRow
	var totalCount int

	for rows.Next() {
		var row AlertRow
		var total int
		if err := rows.Scan(
			&row.AlertID, &row.RuleID, &row.EventDate, &row.Severity,
			&row.MetricName, &row.ObjectType, &row.ObjectID,
			&row.CurrentValue, &row.BaselineValue, &row.ChangeRate,
			&row.OwnerRole, &row.Status, &row.ImpactScore,
			&total,
		); err != nil {
			return nil, 0, fmt.Errorf("scan alert row: %w", err)
		}
		results = append(results, row)
		totalCount = total
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate alert rows: %w", err)
	}

	if results == nil {
		results = []AlertRow{}
	}

	return results, totalCount, nil
}
