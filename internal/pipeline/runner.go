package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Runner orchestrates a sequence of pipeline Steps with audit logging.
// Each step runs inside a fresh transaction — on success the transaction is
// committed and the step is marked completed; on failure the transaction is
// rolled back and the entire run is marked failed.
type Runner struct {
	DB    *pgxpool.Pool
	Steps []Step
	Log   *zap.Logger
}

// RunInput configures a single pipeline execution.
type RunInput struct {
	// RunType identifies the type of run: "full" or "partial".
	RunType string
	// Mode identifies how the run was triggered: "manual", etc.
	Mode string
	// DataDir is the directory containing data files (CSV, etc.).
	DataDir string
}

// Run executes all registered steps sequentially.
//
// For each step:
//  1. An audit.pipeline_step_run record is created with status "running".
//  2. A database transaction is begun.
//  3. step.Run is called with the transaction.
//  4. On success the transaction is committed and the step is marked "completed".
//  5. On failure the transaction is rolled back, the step is marked "failed",
//     the run is marked "failed", and the error is returned immediately.
func (r *Runner) Run(ctx context.Context, input RunInput) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	recorder := &AuditRecorder{
		Pool:   r.DB,
		Logger: r.Log,
	}

	runID, err := recorder.CreateRun(ctx, input.RunType, input.Mode)
	if err != nil {
		return fmt.Errorf("create run: %w", err)
	}

	for i, step := range r.Steps {
		if err := r.runStep(ctx, recorder, runID, step, i, input); err != nil {
			_ = recorder.CompleteRun(ctx, runID, "failed", err.Error())
			return err
		}
	}

	if err := recorder.CompleteRun(ctx, runID, "completed", ""); err != nil {
		return fmt.Errorf("complete run: %w", err)
	}
	return nil
}

func (r *Runner) runStep(ctx context.Context, recorder *AuditRecorder, runID string, step Step, order int, input RunInput) error {
	stepRunID, err := recorder.CreateStepRun(ctx, runID, step.Name(), order, "running")
	if err != nil {
		return fmt.Errorf("create step run for %s: %w", step.Name(), err)
	}

	tx, err := r.DB.Begin(ctx)
	if err != nil {
		_ = recorder.CompleteStepRun(ctx, stepRunID, "failed", 0, 0, err.Error())
		return fmt.Errorf("begin tx for %s: %w", step.Name(), err)
	}

	stepInput := StepInput{
		RunID:   runID,
		Logger:  r.Log.With(zap.String("step", step.Name()), zap.Int("order", order)),
		DataDir: input.DataDir,
	}

	output, runErr := step.Run(ctx, tx, stepInput)
	if runErr != nil {
		_ = tx.Rollback(ctx)
		_ = recorder.CompleteStepRun(ctx, stepRunID, "failed", 0, 0, runErr.Error())
		return fmt.Errorf("step %s: %w", step.Name(), runErr)
	}

	if err := tx.Commit(ctx); err != nil {
		_ = recorder.CompleteStepRun(ctx, stepRunID, "failed", 0, 0, err.Error())
		return fmt.Errorf("commit tx for %s: %w", step.Name(), err)
	}

	inputCount := int64(0)
	outputCount := int64(0)
	if output != nil {
		inputCount = output.InputCount
		outputCount = output.OutputCount
	}

	if err := recorder.CompleteStepRun(ctx, stepRunID, "completed", inputCount, outputCount, ""); err != nil {
		return fmt.Errorf("complete step run for %s: %w", step.Name(), err)
	}

	r.Log.Info("step completed",
		zap.String("step", step.Name()),
		zap.Int("order", order),
		zap.Int64("input_count", inputCount),
		zap.Int64("output_count", outputCount),
	)
	return nil
}
