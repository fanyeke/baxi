package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	"baxi/internal/action"
	"baxi/internal/outbox"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DispatchConfig configures the DispatchWorker behaviour.
type DispatchConfig struct {
	PollInterval time.Duration
	BatchSize    int
	MaxRetries   int64
	DryRun       bool
}

// DefaultDispatchConfig returns a DispatchConfig with sensible defaults.
func DefaultDispatchConfig() DispatchConfig {
	return DispatchConfig{
		PollInterval: 30 * time.Second,
		BatchSize:    10,
		MaxRetries:   10,
		DryRun:       true,
	}
}

// outboxRepository defines the subset of OutboxRepository methods needed
// by the DispatchWorker. Using an interface keeps the worker testable.
type outboxRepository interface {
	GetPendingEvents(ctx context.Context, pool *pgxpool.Pool, limit int) ([]outbox.OutboxEvent, error)
	MarkDispatched(ctx context.Context, tx pgx.Tx, eventID string) error
	MarkFailed(ctx context.Context, tx pgx.Tx, eventID string, errMsg string) error
	UpdateMaxAttempts(ctx context.Context, tx pgx.Tx, eventID string) error
	SetNextRetryAt(ctx context.Context, tx pgx.Tx, eventID string, nextRetryAt time.Time) error
}

// txBeginner abstracts transaction creation so tests can inject a mock.
type txBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// DispatchWorker polls pending outbox events and dispatches them through
// registered ActionExecutor adapters.
type DispatchWorker struct {
	repo      outboxRepository
	pool      *pgxpool.Pool // passed to repo.GetPendingEvents; nil in tests
	txBegin   txBeginner    // used for tx creation; *pgxpool.Pool in prod
	executors map[string]action.ActionExecutor
	config    DispatchConfig
	cancel    context.CancelFunc
}

// NewDispatchWorker creates a DispatchWorker.
// Pass a *pgxpool.Pool for both internal pool and txBegin in production.
func NewDispatchWorker(
	repo outboxRepository,
	pool *pgxpool.Pool,
	executors map[string]action.ActionExecutor,
	config DispatchConfig,
) *DispatchWorker {
	if executors == nil {
		executors = make(map[string]action.ActionExecutor)
	}
	return &DispatchWorker{
		repo:      repo,
		pool:      pool,
		txBegin:   pool,
		executors: executors,
		config:    config,
	}
}

// Run starts the dispatch loop and blocks until the context is cancelled.
func (w *DispatchWorker) Run(ctx context.Context) error {
	ctx, w.cancel = context.WithCancel(ctx)
	defer w.cancel()

	log.Printf("[DispatchWorker] started (poll_interval=%s, batch_size=%d, dry_run=%v)",
		w.config.PollInterval, w.config.BatchSize, w.config.DryRun)

	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[DispatchWorker] shutting down")
			return nil
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

// Stop cancels the worker context, triggering Run() to exit cleanly.
func (w *DispatchWorker) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
}

// processBatch fetches the next batch of pending events and dispatches each one.
func (w *DispatchWorker) processBatch(ctx context.Context) {
	events, err := w.repo.GetPendingEvents(ctx, w.pool, w.config.BatchSize)
	if err != nil {
		log.Printf("[DispatchWorker] error fetching pending events: %v", err)
		return
	}
	if len(events) == 0 {
		return
	}
	log.Printf("[DispatchWorker] processing %d events", len(events))
	for i := range events {
		w.dispatchEvent(ctx, &events[i])
	}
}

// dispatchEvent handles a single outbox event through the dispatch lifecycle.
func (w *DispatchWorker) dispatchEvent(ctx context.Context, event *outbox.OutboxEvent) {
	log.Printf("[DispatchWorker] dispatching event %s (channel=%s, type=%s, attempts=%d)",
		event.EventID, event.TargetChannel, event.EventType, event.DispatchAttempts)

	if event.DispatchAttempts >= w.config.MaxRetries {
		w.handleMaxAttempts(ctx, event)
		return
	}

	executor, ok := w.executors[event.TargetChannel]
	if !ok {
		w.handleNoExecutor(ctx, event)
		return
	}

	var proposal action.ActionProposal
	if err := json.Unmarshal(event.Payload, &proposal); err != nil {
		w.handleFailed(ctx, event, fmt.Sprintf("invalid payload: %v", err))
		return
	}

	result, err := executor.Execute(ctx, proposal, w.config.DryRun)
	if err != nil {
		w.handleFailed(ctx, event, err.Error())
		return
	}
	if !result.Success {
		w.handleFailed(ctx, event, result.Error)
		return
	}

	log.Printf("[DispatchWorker] event %s successfully dispatched via %s", event.EventID, event.TargetChannel)
	w.markDispatchedInTx(ctx, event)
}

// markDispatchedInTx marks the event as dispatched within a transaction.
func (w *DispatchWorker) markDispatchedInTx(ctx context.Context, event *outbox.OutboxEvent) {
	tx, err := w.txBegin.Begin(ctx)
	if err != nil {
		log.Printf("[DispatchWorker] error beginning tx for dispatched: %v", err)
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := w.repo.MarkDispatched(ctx, tx, event.EventID); err != nil {
		log.Printf("[DispatchWorker] error marking dispatched: %v", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		log.Printf("[DispatchWorker] error committing dispatched: %v", err)
	}
}

// handleMaxAttempts permanently fails the event via UpdateMaxAttempts.
func (w *DispatchWorker) handleMaxAttempts(ctx context.Context, event *outbox.OutboxEvent) {
	log.Printf("[DispatchWorker] event %s reached max retry attempts (%d), permanently failing",
		event.EventID, w.config.MaxRetries)

	tx, err := w.txBegin.Begin(ctx)
	if err != nil {
		log.Printf("[DispatchWorker] error beginning tx for max attempts: %v", err)
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := w.repo.UpdateMaxAttempts(ctx, tx, event.EventID); err != nil {
		log.Printf("[DispatchWorker] error updating max attempts for %s: %v", event.EventID, err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		log.Printf("[DispatchWorker] error committing max attempts for %s: %v", event.EventID, err)
	}
}

// handleNoExecutor marks the event as failed when no adapter is registered
// for the target channel.
func (w *DispatchWorker) handleNoExecutor(ctx context.Context, event *outbox.OutboxEvent) {
	errMsg := fmt.Sprintf("no executor for channel: %s", event.TargetChannel)
	log.Printf("[DispatchWorker] event %s: %s", event.EventID, errMsg)

	tx, err := w.txBegin.Begin(ctx)
	if err != nil {
		log.Printf("[DispatchWorker] error beginning tx for no executor: %v", err)
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := w.repo.MarkFailed(ctx, tx, event.EventID, errMsg); err != nil {
		log.Printf("[DispatchWorker] error marking failed for %s: %v", event.EventID, err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		log.Printf("[DispatchWorker] error committing failure for %s: %v", event.EventID, err)
	}
}

// handleFailed marks the event as failed and sets next_retry_at for
// exponential backoff.
func (w *DispatchWorker) handleFailed(ctx context.Context, event *outbox.OutboxEvent, errMsg string) {
	log.Printf("[DispatchWorker] event %s failed: %s", event.EventID, errMsg)

	tx, err := w.txBegin.Begin(ctx)
	if err != nil {
		log.Printf("[DispatchWorker] error beginning tx for failure: %v", err)
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := w.repo.MarkFailed(ctx, tx, event.EventID, errMsg); err != nil {
		log.Printf("[DispatchWorker] error marking failed for %s: %v", event.EventID, err)
		return
	}

	nextRetryAt := time.Now().Add(backoffDuration(event.DispatchAttempts + 1))
	if err := w.repo.SetNextRetryAt(ctx, tx, event.EventID, nextRetryAt); err != nil {
		log.Printf("[DispatchWorker] error setting next_retry_at for %s: %v", event.EventID, err)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		log.Printf("[DispatchWorker] error committing failure for %s: %v", event.EventID, err)
	}
}

// backoffDuration computes exponential backoff for the given attempt.
// Sequence: 1m, 2m, 4m, 8m, 16m, 30m (capped).
func backoffDuration(attempts int64) time.Duration {
	if attempts <= 0 {
		return 0
	}
	d := time.Duration(math.Pow(2, float64(attempts-1))) * time.Minute
	if d > 30*time.Minute {
		d = 30 * time.Minute
	}
	return d
}

// Compile-time assertions.
var _ txBeginner = (*pgxpool.Pool)(nil)
var _ outboxRepository = (*outbox.OutboxRepository)(nil)
