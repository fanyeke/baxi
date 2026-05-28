package worker

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
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
	AuditLogPath string
}

// DefaultDispatchConfig returns a DispatchConfig with sensible defaults.
func DefaultDispatchConfig() DispatchConfig {
	return DispatchConfig{
		PollInterval: 30 * time.Second,
		BatchSize:    10,
		MaxRetries:   3,
		DryRun:       false,
		AuditLogPath: "./data/system/dispatch_archive.csv",
	}
}

// DispatchResult represents the outcome of dispatching a single event,
// equivalent to the dict returned by Python's dispatch_one().
type DispatchResult struct {
	Status      string
	AdapterName string
	Error       string
	Message     string
	ExternalRef string
	DryRun      bool
}

// auditLogEntry represents a single row in the dispatch audit CSV.
type auditLogEntry struct {
	Timestamp     string
	EventID       string
	TargetChannel string
	AdapterName   string
	Mode          string
	Status        string
	ExternalRef   string
	Error         string
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

	log.Printf("[DispatchWorker] started (poll_interval=%s, batch_size=%d, max_retries=%d, dry_run=%v)",
		w.config.PollInterval, w.config.BatchSize, w.config.MaxRetries, w.config.DryRun)

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

// FetchPending fetches pending events from the outbox, optionally filtered
// by channel. Equivalent to Python's fetch_pending().
func (w *DispatchWorker) FetchPending(ctx context.Context, limit int) ([]outbox.OutboxEvent, error) {
	if limit <= 0 {
		limit = w.config.BatchSize
	}
	return w.repo.GetPendingEvents(ctx, w.pool, limit)
}

// DispatchOne dispatches a single outbox event through its resolved adapter.
// Equivalent to Python's dispatch_one().
func (w *DispatchWorker) DispatchOne(ctx context.Context, event *outbox.OutboxEvent) DispatchResult {
	return w.dispatchEvent(ctx, event)
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
// Returns a DispatchResult describing the outcome.
func (w *DispatchWorker) dispatchEvent(ctx context.Context, event *outbox.OutboxEvent) DispatchResult {
	log.Printf("[DispatchWorker] dispatching event %s (channel=%s, type=%s, attempts=%d)",
		event.EventID, event.TargetChannel, event.EventType, event.DispatchAttempts)

	if event.DispatchAttempts >= w.config.MaxRetries {
		w.handleMaxAttempts(ctx, event)
		entry := auditLogEntry{
			Timestamp:     time.Now().Format(time.RFC3339),
			EventID:       event.EventID,
			TargetChannel: event.TargetChannel,
			AdapterName:   "",
			Mode:          w.modeString(),
			Status:        "max_attempts",
			ExternalRef:   "",
			Error:         fmt.Sprintf("max retry attempts reached (%d)", w.config.MaxRetries),
		}
		w.writeAuditLog([]auditLogEntry{entry})
		return DispatchResult{
			Status:      "failed",
			AdapterName: "",
			Error:       fmt.Sprintf("max retry attempts reached (%d)", w.config.MaxRetries),
		}
	}

	executor, ok := w.executors[event.TargetChannel]
	if !ok {
		errMsg := fmt.Sprintf("no executor for channel: %s", event.TargetChannel)
		w.handleNoExecutor(ctx, event)
		entry := auditLogEntry{
			Timestamp:     time.Now().Format(time.RFC3339),
			EventID:       event.EventID,
			TargetChannel: event.TargetChannel,
			AdapterName:   "",
			Mode:          w.modeString(),
			Status:        "failed",
			ExternalRef:   "",
			Error:         errMsg,
		}
		w.writeAuditLog([]auditLogEntry{entry})
		return DispatchResult{
			Status:      "failed",
			AdapterName: "",
			Error:       errMsg,
		}
	}

	var proposal action.ActionProposal
	if err := json.Unmarshal(event.Payload, &proposal); err != nil {
		errMsg := fmt.Sprintf("invalid payload: %v", err)
		w.handleFailed(ctx, event, errMsg)
		entry := auditLogEntry{
			Timestamp:     time.Now().Format(time.RFC3339),
			EventID:       event.EventID,
			TargetChannel: event.TargetChannel,
			AdapterName:   executorName(executor),
			Mode:          w.modeString(),
			Status:        "failed",
			ExternalRef:   "",
			Error:         errMsg,
		}
		w.writeAuditLog([]auditLogEntry{entry})
		return DispatchResult{
			Status:      "failed",
			AdapterName: executorName(executor),
			Error:       errMsg,
		}
	}

	execResult, err := executor.Execute(ctx, proposal, w.config.DryRun)
	if err != nil {
		w.handleFailed(ctx, event, err.Error())
		entry := auditLogEntry{
			Timestamp:     time.Now().Format(time.RFC3339),
			EventID:       event.EventID,
			TargetChannel: event.TargetChannel,
			AdapterName:   executorName(executor),
			Mode:          w.modeString(),
			Status:        "failed",
			ExternalRef:   "",
			Error:         err.Error(),
		}
		w.writeAuditLog([]auditLogEntry{entry})
		return DispatchResult{
			Status:      "failed",
			AdapterName: executorName(executor),
			Error:       err.Error(),
		}
	}
	if !execResult.Success {
		w.handleFailed(ctx, event, execResult.Error)
		entry := auditLogEntry{
			Timestamp:     time.Now().Format(time.RFC3339),
			EventID:       event.EventID,
			TargetChannel: event.TargetChannel,
			AdapterName:   executorName(executor),
			Mode:          w.modeString(),
			Status:        "failed",
			ExternalRef:   execResult.OutboxEventID,
			Error:         execResult.Error,
		}
		w.writeAuditLog([]auditLogEntry{entry})
		return DispatchResult{
			Status:      "failed",
			AdapterName: executorName(executor),
			Error:       execResult.Error,
		}
	}

	// In dry-run mode, do NOT mark as dispatched — status stays pending,
	// matching Python's write_result() behaviour.
	if w.config.DryRun {
		log.Printf("[DispatchWorker] event %s dry-run dispatch via %s", event.EventID, event.TargetChannel)
		entry := auditLogEntry{
			Timestamp:     time.Now().Format(time.RFC3339),
			EventID:       event.EventID,
			TargetChannel: event.TargetChannel,
			AdapterName:   executorName(executor),
			Mode:          "dry_run",
			Status:        "dispatched",
			ExternalRef:   execResult.OutboxEventID,
			Error:         "",
		}
		w.writeAuditLog([]auditLogEntry{entry})
		return DispatchResult{
			Status:      "dispatched",
			AdapterName: executorName(executor),
			DryRun:      true,
		}
	}

	log.Printf("[DispatchWorker] event %s successfully dispatched via %s", event.EventID, event.TargetChannel)
	w.markDispatchedInTx(ctx, event)
	entry := auditLogEntry{
		Timestamp:     time.Now().Format(time.RFC3339),
		EventID:       event.EventID,
		TargetChannel: event.TargetChannel,
		AdapterName:   executorName(executor),
		Mode:          "live",
		Status:        "dispatched",
		ExternalRef:   execResult.OutboxEventID,
		Error:         "",
	}
	w.writeAuditLog([]auditLogEntry{entry})
	return DispatchResult{
		Status:      "dispatched",
		AdapterName: executorName(executor),
		DryRun:      false,
	}
}

// modeString returns "dry_run" or "live" based on config.
func (w *DispatchWorker) modeString() string {
	if w.config.DryRun {
		return "dry_run"
	}
	return "live"
}

// executorName returns the type name of the executor for audit logging.
func executorName(exec action.ActionExecutor) string {
	if exec == nil {
		return ""
	}
	// Try to get a meaningful name; fall back to generic type info.
	switch e := exec.(type) {
	case *action.NoOpExecutor:
		return "NoOpExecutor"
	default:
		_ = e
		return fmt.Sprintf("%T", exec)
	}
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

// writeAuditLog appends dispatch audit entries to the archive CSV.
// Equivalent to Python's write_audit_log().
func (w *DispatchWorker) writeAuditLog(entries []auditLogEntry) {
	if w.config.AuditLogPath == "" || len(entries) == 0 {
		return
	}

	if err := os.MkdirAll(filepath.Dir(w.config.AuditLogPath), 0o755); err != nil {
		log.Printf("[DispatchWorker] error creating audit log directory: %v", err)
		return
	}

	writeHeader := false
	if _, err := os.Stat(w.config.AuditLogPath); os.IsNotExist(err) {
		writeHeader = true
	}

	f, err := os.OpenFile(w.config.AuditLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Printf("[DispatchWorker] error opening audit log: %v", err)
		return
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	if writeHeader {
		if err := writer.Write([]string{"timestamp", "event_id", "target_channel", "adapter_name",
			"mode", "status", "external_ref", "error"}); err != nil {
			log.Printf("[DispatchWorker] error writing audit log header: %v", err)
			return
		}
	}

	for _, e := range entries {
		if err := writer.Write([]string{
			e.Timestamp,
			e.EventID,
			e.TargetChannel,
			e.AdapterName,
			e.Mode,
			e.Status,
			e.ExternalRef,
			e.Error,
		}); err != nil {
			log.Printf("[DispatchWorker] error writing audit log entry: %v", err)
			return
		}
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
