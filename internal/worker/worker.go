package worker

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Worker represents a background worker process.
type Worker struct {
	logger *zap.Logger
	pool   *pgxpool.Pool
}

// New creates a new Worker instance.
func New(logger *zap.Logger, pool *pgxpool.Pool) *Worker {
	return &Worker{
		logger: logger,
		pool:   pool,
	}
}

// Run starts the worker and blocks until the context is cancelled.
// Phase 1: just verifies DB connectivity and blocks. No business logic.
func (w *Worker) Run(ctx context.Context) error {
	w.logger.Info("baxi-worker started",
		zap.String("service", "baxi-worker"),
	)

	// Verify database connectivity
	if err := w.pool.Ping(ctx); err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}

	w.logger.Info("connected to database")

	// Block until context is cancelled
	<-ctx.Done()

	w.logger.Info("worker shutting down")
	return nil
}
