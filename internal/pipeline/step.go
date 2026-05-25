package pipeline

import (
	"context"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// StepInput contains context for a step execution.
type StepInput struct {
	// RunID is the audit pipeline_run identifier.
	RunID string
	// Logger is a step-scoped logger.
	Logger *zap.Logger
	// DataDir is the directory containing data files (CSV, etc.).
	DataDir string
}

// StepOutput contains the result of a step execution.
type StepOutput struct {
	InputCount  int64
	OutputCount int64
}

// Step defines the interface for a pipeline step.
// Each step runs inside a database transaction and reports audit counts.
type Step interface {
	// Name returns the human-readable step name for audit logging.
	Name() string

	// Run executes the step logic within the given transaction.
	// The tx must NOT be committed or rolled back by the step — the Runner
	// handles commit/rollback based on the error return.
	Run(ctx context.Context, tx pgx.Tx, input StepInput) (*StepOutput, error)
}
