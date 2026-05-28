package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"baxi/internal/outbox"
	"baxi/internal/pipeline"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// taskRow holds the fields extracted from a single ops.task row plus its
// associated recommendation context.
type taskRow struct {
	TaskID           string
	RecommendationID string
	AlertID          string
	TaskTitle        string
	TaskDescription  string
	TargetObjectType string
	TargetObjectID   string
	TaskSource       string
	OwnerRole        string
	Priority         string
}

// CreateOutboxStep reads ops.task (and ops.recommendation for payload context)
// and generates ops.outbox_event records with status='pending'.
//
// Each task generates exactly one outbox event. Event IDs follow the pattern
// "outbox-{task_id}" for idempotent inserts. The step uses INSERT … ON CONFLICT
// (event_id) DO NOTHING so re-running does not produce duplicates.
//
// Expected output: 36 outbox events (matching Python baseline).
type CreateOutboxStep struct {
	repo *outbox.OutboxRepository
}

// NewCreateOutboxStep creates a new CreateOutboxStep.
func NewCreateOutboxStep() *CreateOutboxStep {
	return &CreateOutboxStep{
		repo: outbox.NewOutboxRepository(),
	}
}

// Name returns the step name for audit logging.
func (s *CreateOutboxStep) Name() string {
	return "create_outbox_events"
}

// Run reads all tasks from ops.task, builds outbox events, and inserts
// them into ops.outbox_event with status='pending'.
//
// The tx must NOT be committed or rolled back by this step — the Runner
// handles commit/rollback based on the error return.
func (s *CreateOutboxStep) Run(ctx context.Context, tx pgx.Tx, input pipeline.StepInput) (*pipeline.StepOutput, error) {
	// Count source tasks for input
	var inputCount int64
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM ops.task`).Scan(&inputCount); err != nil {
		return nil, fmt.Errorf("count ops.task: %w", err)
	}

	if inputCount == 0 {
		input.Logger.Info("no tasks found; skipping outbox event generation")
		return &pipeline.StepOutput{
			InputCount:  0,
			OutputCount: 0,
		}, nil
	}

	rows, err := tx.Query(ctx, `
		SELECT
			t.task_id,
			COALESCE(t.recommendation_id, '') AS recommendation_id,
			COALESCE(t.alert_id, '') AS alert_id,
			t.task_title,
			COALESCE(t.task_description, '') AS task_description,
			COALESCE(t.target_object_type, '') AS target_object_type,
			COALESCE(t.target_object_id, '') AS target_object_id,
			t.task_source,
			COALESCE(t.owner_role, '') AS owner_role,
			t.priority
		FROM ops.task t
		ORDER BY t.task_id
	`)
	if err != nil {
		return nil, fmt.Errorf("query ops.task: %w", err)
	}
	defer rows.Close()

	var events []outbox.OutboxEvent
	for rows.Next() {
		var tr taskRow
		if err := rows.Scan(
			&tr.TaskID,
			&tr.RecommendationID,
			&tr.AlertID,
			&tr.TaskTitle,
			&tr.TaskDescription,
			&tr.TargetObjectType,
			&tr.TargetObjectID,
			&tr.TaskSource,
			&tr.OwnerRole,
			&tr.Priority,
		); err != nil {
			return nil, fmt.Errorf("scan task row: %w", err)
		}

		event, err := s.buildEvent(tr)
		if err != nil {
			return nil, fmt.Errorf("build event for task %s: %w", tr.TaskID, err)
		}
		events = append(events, *event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task rows: %w", err)
	}

	input.Logger.Info("outbox events prepared",
		zap.Int("count", len(events)),
	)

	if len(events) == 0 {
		return &pipeline.StepOutput{
			InputCount:  inputCount,
			OutputCount: 0,
		}, nil
	}

	outputCount, err := s.repo.CreateEvents(ctx, tx, events)
	if err != nil {
		return nil, fmt.Errorf("create_outbox_events: %w", err)
	}

	input.Logger.Info("outbox events written",
		zap.Int64("events", outputCount),
	)

	return &pipeline.StepOutput{
		InputCount:  inputCount,
		OutputCount: outputCount,
	}, nil
}

// buildEvent constructs an OutboxEvent from a task row.
func (s *CreateOutboxStep) buildEvent(tr taskRow) (*outbox.OutboxEvent, error) {
	// Derive target_channel from task_source:
	//   heuristic_strategy (global tasks) → local_cli
	//   dimensional_rule (dimensional tasks) → feishu_cli
	targetChannel := deriveTargetChannel(tr.TaskSource)

	// Build payload JSON with task and recommendation context
	payload := map[string]interface{}{
		"task_id":            tr.TaskID,
		"recommendation_id":  tr.RecommendationID,
		"alert_id":           tr.AlertID,
		"task_title":         tr.TaskTitle,
		"task_description":   tr.TaskDescription,
		"target_object_type": tr.TargetObjectType,
		"target_object_id":   tr.TargetObjectID,
		"task_source":        tr.TaskSource,
		"owner_role":         tr.OwnerRole,
		"priority":           tr.Priority,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload for task %s: %w", tr.TaskID, err)
	}

	return &outbox.OutboxEvent{
		EventID:       "outbox-" + tr.TaskID,
		EventType:     "task_assigned",
		SourceType:    "task",
		SourceID:      tr.TaskID,
		Status:        "pending",
		Payload:       payloadBytes,
		TargetChannel: targetChannel,
	}, nil
}

// deriveTargetChannel maps task_source to an adapter channel name.
func deriveTargetChannel(taskSource string) string {
	switch taskSource {
	case "dimensional_rule":
		return "feishu_cli"
	case "heuristic_strategy":
		return "local_cli"
	default:
		return "local_cli"
	}
}

// IsDimensionalTask checks whether a task_id corresponds to a dimensional
// (rather than global) task based on its ID prefix.
func IsDimensionalTask(taskID string) bool {
	return strings.HasPrefix(taskID, "dimtask-")
}
