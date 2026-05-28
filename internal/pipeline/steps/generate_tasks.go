package steps

import (
	"context"
	"fmt"

	"baxi/internal/pipeline"
	"baxi/internal/recommendation"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// GenerateTasksStep reads ops.recommendation and generates ops.task records.
// Each recommendation maps to exactly one task with status='pending'.
//
// The step uses INSERT ... ON CONFLICT (task_id) DO NOTHING for idempotency.
type GenerateTasksStep struct {
	generator *recommendation.TaskGenerator
}

// NewGenerateTasksStep creates a new GenerateTasksStep.
func NewGenerateTasksStep() *GenerateTasksStep {
	return &GenerateTasksStep{
		generator: recommendation.NewTaskGenerator(),
	}
}

// Name returns the step name for audit logging.
func (s *GenerateTasksStep) Name() string {
	return "generate_tasks"
}

// Run reads all recommendations from ops.recommendation, generates task records,
// and inserts them into ops.task. Each task gets status='pending'.
func (s *GenerateTasksStep) Run(ctx context.Context, tx pgx.Tx, input pipeline.StepInput) (*pipeline.StepOutput, error) {
	tasks, err := s.generator.GenerateTasks(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("generate_tasks: %w", err)
	}

	input.Logger.Info("task generation complete",
		zap.Int("generated", len(tasks)),
	)

	// Count input recommendations
	var inputCount int64
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM ops.recommendation`).Scan(&inputCount); err != nil {
		return nil, fmt.Errorf("count recommendations: %w", err)
	}

	if len(tasks) == 0 {
		return &pipeline.StepOutput{
			InputCount:  inputCount,
			OutputCount: 0,
		}, nil
	}

	var outputCount int64
	for _, t := range tasks {
		_, err := tx.Exec(ctx, `
			INSERT INTO ops.task (
				task_id, recommendation_id, alert_id,
				task_title, task_description,
				target_object_type, target_object_id,
				task_source, owner_role, priority, status,
				created_at
			) VALUES (
				$1, $2, NULLIF($3, ''),
				$4, NULLIF($5, ''),
				NULLIF($6, ''), NULLIF($7, ''),
				$8, NULLIF($9, ''), $10, $11,
				NOW()
			)
			ON CONFLICT (task_id) DO NOTHING
		`,
			t.TaskID,
			t.RecommendationID,
			t.AlertID,
			t.TaskTitle,
			t.TaskDescription,
			t.TargetObjectType,
			t.TargetObjectID,
			t.TaskSource,
			t.OwnerRole,
			t.Priority,
			t.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("insert task %s: %w", t.TaskID, err)
		}
		outputCount++
	}

	input.Logger.Info("tasks written",
		zap.Int64("tasks", outputCount),
	)

	return &pipeline.StepOutput{
		InputCount:  inputCount,
		OutputCount: outputCount,
	}, nil
}
