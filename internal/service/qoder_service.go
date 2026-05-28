package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"baxi/internal/api/dto"
	"baxi/internal/repository"
)

// pipelineRunRow represents the last pipeline run from audit.pipeline_run.
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

// QoderService handles business logic for Qoder AI decision engine endpoints.
type QoderService struct {
	contextRepo repository.ContextRepository
	pool        *pgxpool.Pool
}

// NewQoderService creates a new QoderService.
func NewQoderService(contextRepo repository.ContextRepository, pool *pgxpool.Pool) *QoderService {
	return &QoderService{
		contextRepo: contextRepo,
		pool:        pool,
	}
}

// GetContext aggregates system context from multiple sources.
func (s *QoderService) GetContext(ctx context.Context, requestID string, params dto.ContextQueryParams) (*dto.ContextResponse, error) {
	pipelineRun := s.queryLastPipelineRun(ctx)
	totalAlerts, topAlerts := s.queryAlerts(ctx, params.Severity, params.LimitAlerts)
	totalTasks, openTasks := s.queryOpenTasks(ctx, params.LimitTasks)
	totalOutbox, pendingOutbox := s.queryPendingOutbox(ctx, params.LimitOutbox)
	caps := dto.StaticCapabilities()

	resp := &dto.ContextResponse{
		RequestID: requestID,
		System: dto.SystemInfo{
			LastPipelineRun: pipelineRun,
		},
		Summary: dto.ContextSummary{
			TotalAlerts:        totalAlerts,
			TotalOpenTasks:     totalTasks,
			TotalPendingOutbox: totalOutbox,
		},
		TopAlerts:       topAlerts,
		OpenTasks:       openTasks,
		PendingOutbox:   pendingOutbox,
		RecentDiagnosis: []interface{}{},
		AllowedActions:  caps.AllowedActions(),
		ForbiddenActions: caps.ForbiddenActions(),
		// Enrichment: ontology, governance, agent_policy
		// These are static until configloader services are wired.
		Ontology: dto.OntologyInfo{
			ObjectTypes: []string{
				"customer", "order", "seller", "product",
				"category", "region", "marketing_lead", "metric_alert",
			},
			ObjectsAvailable: true,
		},
		Governance: dto.GovernanceInfo{
			ClassificationLoaded: true,
			LineageLoaded:        true,
			AccessPolicyLoaded:   true,
			RedactionEnabled:     true,
		},
		AgentPolicy: dto.AgentPolicyInfo{
			Role:              "analyst",
			CanReadObjects:    true,
			CanExecuteActions: false,
			CanWriteReports:   false,
		},
	}

	return resp, nil
}

func (s *QoderService) queryLastPipelineRun(ctx context.Context) *dto.PipelineRunInfo {
	if s.pool == nil {
		return nil
	}
	query := `
		SELECT run_id, run_type, mode, status, started_at,
		       finished_at, input_count, output_count, error_message
		FROM audit.pipeline_run
		ORDER BY started_at DESC
		LIMIT 1
	`

	var row pipelineRunRow
	err := s.pool.QueryRow(ctx, query).Scan(
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
		return nil
	}

	var finishedAt *string
	if row.FinishedAt != nil {
		s := row.FinishedAt.Format(time.RFC3339Nano)
		finishedAt = &s
	}
	startedAt := row.StartedAt.Format(time.RFC3339Nano)

	return &dto.PipelineRunInfo{
		RunID:        row.RunID,
		RunType:      row.RunType,
		Mode:         row.Mode,
		Status:       row.Status,
		StartedAt:    startedAt,
		FinishedAt:   finishedAt,
		InputCount:   row.InputCount,
		OutputCount:  row.OutputCount,
		ErrorMessage: row.ErrorMessage,
	}
}

func (s *QoderService) queryAlerts(ctx context.Context, severity string, limit int) (int, []dto.AlertItem) {
	if s.pool == nil {
		return 0, []dto.AlertItem{}
	}
	countQuery := `SELECT COUNT(*) FROM ops.metric_alert`
	countArgs := []interface{}{}

	if severity != "" {
		countQuery += " WHERE severity = $1"
		countArgs = append(countArgs, severity)
	}

	var total int
	err := s.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return 0, []dto.AlertItem{}
	}

	itemsQuery := `
		SELECT alert_id, rule_id, event_date::TEXT, severity, metric_name,
		       object_type, object_id, current_value, baseline_value,
		       change_rate, owner_role, status, impact_score
		FROM ops.metric_alert
	`
	args := []interface{}{}
	argIdx := 1

	if severity != "" {
		itemsQuery += fmt.Sprintf(" WHERE severity = $%d", argIdx)
		args = append(args, severity)
		argIdx++
	}

	itemsQuery += fmt.Sprintf(" ORDER BY severity DESC, event_date DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := s.pool.Query(ctx, itemsQuery, args...)
	if err != nil {
		return total, []dto.AlertItem{}
	}
	defer rows.Close()

	var items []dto.AlertItem
	for rows.Next() {
		var repoRow repository.AlertRow
		if err := rows.Scan(
			&repoRow.AlertID, &repoRow.RuleID, &repoRow.EventDate,
			&repoRow.Severity, &repoRow.MetricName,
			&repoRow.ObjectType, &repoRow.ObjectID,
			&repoRow.CurrentValue, &repoRow.BaselineValue,
			&repoRow.ChangeRate, &repoRow.OwnerRole,
			&repoRow.Status, &repoRow.ImpactScore,
		); err != nil {
			continue
		}
		items = append(items, dto.AlertItem{
			EventID:       repoRow.AlertID,
			RuleID:        repoRow.RuleID,
			EventDate:     repoRow.EventDate,
			Severity:      repoRow.Severity,
			MetricName:    repoRow.MetricName,
			ObjectType:    repoRow.ObjectType,
			ObjectID:      repoRow.ObjectID,
			CurrentValue:  repoRow.CurrentValue,
			BaselineValue: repoRow.BaselineValue,
			ChangeRate:    repoRow.ChangeRate,
			OwnerRole:     repoRow.OwnerRole,
			Status:        repoRow.Status,
			ImpactScore:   repoRow.ImpactScore,
		})
	}

	if items == nil {
		items = []dto.AlertItem{}
	}

	return total, items
}

func (s *QoderService) queryOpenTasks(ctx context.Context, limit int) (int, []dto.TaskItem) {
	if s.pool == nil {
		return 0, []dto.TaskItem{}
	}
	openStatuses := []string{"todo", "in_progress"}

	countQuery := `SELECT COUNT(*) FROM ops.task WHERE status = ANY($1)`
	var total int
	err := s.pool.QueryRow(ctx, countQuery, openStatuses).Scan(&total)
	if err != nil {
		return 0, []dto.TaskItem{}
	}

	itemsQuery := `
		SELECT task_id, recommendation_id, alert_id,
		       task_title, task_description,
		       target_object_type, target_object_id,
		       owner_role, owner_user_id,
		       priority, due_at, status, feedback, completed_at, created_at
		FROM ops.task
		WHERE status = ANY($1)
		ORDER BY priority DESC, created_at DESC
		LIMIT $2
	`

	rows, err := s.pool.Query(ctx, itemsQuery, openStatuses, limit)
	if err != nil {
		return total, []dto.TaskItem{}
	}
	defer rows.Close()

	var items []dto.TaskItem
	for rows.Next() {
		var row repository.TaskRow
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
		); err != nil {
			continue
		}
		items = append(items, mapRowToTaskItem(row))
	}

	if items == nil {
		items = []dto.TaskItem{}
	}

	return total, items
}

func (s *QoderService) queryPendingOutbox(ctx context.Context, limit int) (int, []dto.OutboxItem) {
	if s.pool == nil {
		return 0, []dto.OutboxItem{}
	}
	pendingStatus := "pending"

	countQuery := `SELECT COUNT(*) FROM ops.outbox_event WHERE status = $1`
	var total int
	err := s.pool.QueryRow(ctx, countQuery, pendingStatus).Scan(&total)
	if err != nil {
		return 0, []dto.OutboxItem{}
	}

	itemsQuery := `
		SELECT event_id, event_type, source_type, source_id,
		       target_channel, status, created_at,
		       dispatch_attempts, last_dispatch_at
		FROM ops.outbox_event
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := s.pool.Query(ctx, itemsQuery, pendingStatus, limit)
	if err != nil {
		return total, []dto.OutboxItem{}
	}
	defer rows.Close()

	var items []dto.OutboxItem
	for rows.Next() {
		var row repository.OutboxRow
		if err := rows.Scan(
			&row.OutboxID,
			&row.EventType,
			&row.SourceType,
			&row.SourceID,
			&row.TargetChannel,
			&row.Status,
			&row.CreatedAt,
			&row.DispatchAttempts,
			&row.LastDispatchAt,
		); err != nil {
			continue
		}
		items = append(items, dto.OutboxItem{
			OutboxID:         row.OutboxID,
			EventType:        row.EventType,
			SourceType:       row.SourceType,
			SourceID:         row.SourceID,
			TargetChannel:    row.TargetChannel,
			Status:           row.Status,
			CreatedAt:        row.CreatedAt,
			DispatchAttempts: row.DispatchAttempts,
			LastDispatchAt:   row.LastDispatchAt,
		})
	}

	if items == nil {
		items = []dto.OutboxItem{}
	}

	return total, items
}

func mapRowToTaskItem(row repository.TaskRow) dto.TaskItem {
	desc := ""
	if row.TaskDescription != nil {
		desc = *row.TaskDescription
	}
	ownerRole := ""
	if row.OwnerRole != nil {
		ownerRole = *row.OwnerRole
	}
	priority := row.Priority
	if priority == "" {
		priority = "medium"
	}
	status := row.Status
	if status == "" {
		status = "todo"
	}
	var eventID *string
	if row.AlertID != nil {
		e := *row.AlertID
		eventID = &e
	}
	return dto.TaskItem{
		TaskID:           row.TaskID,
		TaskTitle:        row.TaskTitle,
		TaskDescription:  desc,
		Status:           status,
		Priority:         priority,
		OwnerRole:        ownerRole,
		OwnerUserID:      row.OwnerUserID,
		DueAt:            row.DueAt,
		CreatedAt:        row.CreatedAt,
		CompletedAt:      row.CompletedAt,
		Feedback:         row.Feedback,
		RecommendationID: row.RecommendationID,
		EventID:          eventID,
		TargetObjectType: row.TargetObjectType,
		TargetObjectID:   row.TargetObjectID,
	}
}
