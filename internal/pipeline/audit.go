package pipeline

import (
	"context"
	"crypto/rand"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// AuditRecorder writes pipeline execution audit records to the database.
// It manages lifecycle for pipeline_run and pipeline_step_run rows.
type AuditRecorder struct {
	Pool   *pgxpool.Pool
	Logger *zap.Logger
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// CreateRun inserts a new pipeline_run record with status='running' and returns the run_id.
func (a *AuditRecorder) CreateRun(ctx context.Context, runType, mode string) (string, error) {
	runID := newUUID()
	_, err := a.Pool.Exec(ctx, `
		INSERT INTO audit.pipeline_run (run_id, run_type, mode, status, started_at)
		VALUES ($1, $2, $3, 'running', NOW())
	`, runID, runType, mode)
	if err != nil {
		a.Logger.Error("failed to create pipeline run",
			zap.Error(err),
		)
		return "", fmt.Errorf("create pipeline run: %w", err)
	}
	a.Logger.Info("pipeline run created",
		zap.String("run_id", runID),
		zap.String("run_type", runType),
		zap.String("mode", mode),
	)
	return runID, nil
}

// CompleteRun updates a pipeline_run with final status, finished_at, and optional error message.
func (a *AuditRecorder) CompleteRun(ctx context.Context, runID, status, errMsg string) error {
	_, err := a.Pool.Exec(ctx, `
		UPDATE audit.pipeline_run
		SET status = $1, finished_at = NOW(), error_message = NULLIF($2, '')
		WHERE run_id = $3
	`, status, errMsg, runID)
	if err != nil {
		a.Logger.Error("failed to complete pipeline run",
			zap.Error(err),
			zap.String("run_id", runID),
			zap.String("status", status),
		)
		return fmt.Errorf("complete pipeline run: %w", err)
	}
	a.Logger.Info("pipeline run completed",
		zap.String("run_id", runID),
		zap.String("status", status),
	)
	return nil
}

// CreateStepRun inserts a new pipeline_step_run record and returns the step_run_id.
func (a *AuditRecorder) CreateStepRun(ctx context.Context, runID, stepName string, order int, status string) (string, error) {
	stepRunID := newUUID()
	_, err := a.Pool.Exec(ctx, `
		INSERT INTO audit.pipeline_step_run (step_run_id, pipeline_run_id, step_name, step_order, status, started_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`, stepRunID, runID, stepName, order, status)
	if err != nil {
		a.Logger.Error("failed to create step run",
			zap.Error(err),
			zap.String("step", stepName),
			zap.String("run_id", runID),
		)
		return "", fmt.Errorf("create step run for %s: %w", stepName, err)
	}
	return stepRunID, nil
}

// CompleteStepRun updates a pipeline_step_run with final status, counts, and optional error message.
func (a *AuditRecorder) CompleteStepRun(ctx context.Context, stepRunID, status string, inputCount, outputCount int64, errMsg string) error {
	_, err := a.Pool.Exec(ctx, `
		UPDATE audit.pipeline_step_run
		SET status = $1, finished_at = NOW(), input_count = $2, output_count = $3, error_message = NULLIF($4, '')
		WHERE step_run_id = $5
	`, status, inputCount, outputCount, errMsg, stepRunID)
	if err != nil {
		a.Logger.Error("failed to complete step run",
			zap.Error(err),
			zap.String("step_run_id", stepRunID),
			zap.String("status", status),
		)
		return fmt.Errorf("complete step run: %w", err)
	}
	return nil
}
