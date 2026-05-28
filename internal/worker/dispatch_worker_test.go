package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"baxi/internal/action"
	"baxi/internal/outbox"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

type mockOutboxRepository struct {
	GetPendingEventsFunc  func(ctx context.Context, pool *pgxpool.Pool, limit int) ([]outbox.OutboxEvent, error)
	MarkDispatchedFunc    func(ctx context.Context, tx pgx.Tx, eventID string) error
	MarkFailedFunc        func(ctx context.Context, tx pgx.Tx, eventID string, errMsg string) error
	UpdateMaxAttemptsFunc func(ctx context.Context, tx pgx.Tx, eventID string) error
	SetNextRetryAtFunc    func(ctx context.Context, tx pgx.Tx, eventID string, nextRetryAt time.Time) error
}

func (m *mockOutboxRepository) GetPendingEvents(ctx context.Context, pool *pgxpool.Pool, limit int) ([]outbox.OutboxEvent, error) {
	if m.GetPendingEventsFunc != nil {
		return m.GetPendingEventsFunc(ctx, pool, limit)
	}
	return nil, nil
}

func (m *mockOutboxRepository) MarkDispatched(ctx context.Context, tx pgx.Tx, eventID string) error {
	if m.MarkDispatchedFunc != nil {
		return m.MarkDispatchedFunc(ctx, tx, eventID)
	}
	return nil
}

func (m *mockOutboxRepository) MarkFailed(ctx context.Context, tx pgx.Tx, eventID string, errMsg string) error {
	if m.MarkFailedFunc != nil {
		return m.MarkFailedFunc(ctx, tx, eventID, errMsg)
	}
	return nil
}

func (m *mockOutboxRepository) UpdateMaxAttempts(ctx context.Context, tx pgx.Tx, eventID string) error {
	if m.UpdateMaxAttemptsFunc != nil {
		return m.UpdateMaxAttemptsFunc(ctx, tx, eventID)
	}
	return nil
}

func (m *mockOutboxRepository) SetNextRetryAt(ctx context.Context, tx pgx.Tx, eventID string, nextRetryAt time.Time) error {
	if m.SetNextRetryAtFunc != nil {
		return m.SetNextRetryAtFunc(ctx, tx, eventID, nextRetryAt)
	}
	return nil
}

// mockTx satisfies pgx.Tx with no-op implementations.
type mockTx struct{}

func (tx *mockTx) Begin(ctx context.Context) (pgx.Tx, error) { return tx, nil }

func (tx *mockTx) Commit(ctx context.Context) error { return nil }

func (tx *mockTx) Rollback(ctx context.Context) error { return nil }

func (tx *mockTx) Conn() *pgx.Conn { return nil }

func (tx *mockTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}

func (tx *mockTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (tx *mockTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, nil
}

func (tx *mockTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}

func (tx *mockTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return nil
}

func (tx *mockTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return nil
}

func (tx *mockTx) LargeObjects() pgx.LargeObjects { return pgx.LargeObjects{} }

// mockTxBeginner returns a mock tx on Begin.
type mockTxBeginner struct {
	beginFunc func(ctx context.Context) (pgx.Tx, error)
}

func (m *mockTxBeginner) Begin(ctx context.Context) (pgx.Tx, error) {
	if m.beginFunc != nil {
		return m.beginFunc(ctx)
	}
	return &mockTx{}, nil
}

// mockActionExecutor records calls and returns configured results.
type mockActionExecutor struct {
	executeFunc func(ctx context.Context, proposal action.ActionProposal, dryRun bool) (action.ExecutionResult, error)
}

func (m *mockActionExecutor) Execute(ctx context.Context, proposal action.ActionProposal, dryRun bool) (action.ExecutionResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, proposal, dryRun)
	}
	return action.ExecutionResult{Success: true}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// testEvent builds a minimal OutboxEvent with valid JSON payload.
func testEvent(overrides *outbox.OutboxEvent) outbox.OutboxEvent {
	payload, _ := json.Marshal(action.ActionProposal{
		ProposalID: "prop-test",
		ActionType: "notify_owner",
		Title:      "test proposal",
	})
	e := outbox.OutboxEvent{
		EventID:          "evt-test",
		EventType:        "notify_owner",
		SourceType:       "alert",
		SourceID:         "alert-1",
		Status:           "pending",
		Payload:          payload,
		TargetChannel:    "feishu",
		CreatedAt:        time.Now(),
		DispatchAttempts: 0,
	}
	if overrides != nil {
		if overrides.EventID != "" {
			e.EventID = overrides.EventID
		}
		if overrides.TargetChannel != "" {
			e.TargetChannel = overrides.TargetChannel
		}
		if overrides.DispatchAttempts != 0 {
			e.DispatchAttempts = overrides.DispatchAttempts
		}
		if overrides.Payload != nil {
			e.Payload = overrides.Payload
		}
	}
	return e
}

// newTestWorker creates a DispatchWorker with mocks for testing.
// The pool is nil (mock repo ignores it).
func newTestWorker(
	repo outboxRepository,
	txBegin txBeginner,
	executors map[string]action.ActionExecutor,
	config DispatchConfig,
) *DispatchWorker {
	if executors == nil {
		executors = make(map[string]action.ActionExecutor)
	}
	return &DispatchWorker{
		repo:      repo,
		pool:      nil,
		txBegin:   txBegin,
		executors: executors,
		config:    config,
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestDefaultDispatchConfig(t *testing.T) {
	cfg := DefaultDispatchConfig()
	if cfg.PollInterval != 30*time.Second {
		t.Errorf("PollInterval = %v, want %v", cfg.PollInterval, 30*time.Second)
	}
	if cfg.BatchSize != 10 {
		t.Errorf("BatchSize = %d, want 10", cfg.BatchSize)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3 (matches Python MAX_ATTEMPTS)", cfg.MaxRetries)
	}
	if cfg.DryRun != false {
		t.Errorf("DryRun = %v, want false", cfg.DryRun)
	}
	if !strings.HasSuffix(cfg.AuditLogPath, "dispatch_archive.csv") {
		t.Errorf("AuditLogPath = %q, want path ending in dispatch_archive.csv", cfg.AuditLogPath)
	}
}

func TestBackoffDuration(t *testing.T) {
	tests := []struct {
		attempts int64
		want     time.Duration
	}{
		{0, 0},
		{1, 1 * time.Minute},
		{2, 2 * time.Minute},
		{3, 4 * time.Minute},
		{4, 8 * time.Minute},
		{5, 16 * time.Minute},
		{6, 30 * time.Minute},
		{7, 30 * time.Minute},
		{10, 30 * time.Minute},
	}
	for _, tt := range tests {
		got := backoffDuration(tt.attempts)
		if got != tt.want {
			t.Errorf("backoffDuration(%d) = %v, want %v", tt.attempts, got, tt.want)
		}
	}
}

func TestDispatchWorker_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	mockRepo := &mockOutboxRepository{}
	mockTxB := &mockTxBeginner{}

	w := newTestWorker(mockRepo, mockTxB, nil, DispatchConfig{
		PollInterval: 10 * time.Millisecond,
		BatchSize:    10,
		MaxRetries:   3,
		DryRun:       true,
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := w.Run(ctx); err != nil {
			t.Errorf("Run returned error: %v", err)
		}
	}()

	// Give the ticker a chance to fire once
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Worker exited cleanly
	case <-time.After(time.Second):
		t.Fatal("worker did not stop within 1s after context cancel")
	}
}

func TestDispatchWorker_EmptyBatch(t *testing.T) {
	ctx := context.Background()
	mockTxB := &mockTxBeginner{}

	mockRepo := &mockOutboxRepository{
		GetPendingEventsFunc: func(_ context.Context, _ *pgxpool.Pool, limit int) ([]outbox.OutboxEvent, error) {
			return nil, nil
		},
	}

	w := newTestWorker(mockRepo, mockTxB, nil, DefaultDispatchConfig())
	// processBatch should not panic or error with empty results
	w.processBatch(ctx)
}

func TestDispatchWorker_SuccessfulDispatch(t *testing.T) {
	var mu sync.Mutex
	markedDispatched := false

	ctx := context.Background()
	mockTxB := &mockTxBeginner{}

	mockRepo := &mockOutboxRepository{
		GetPendingEventsFunc: func(_ context.Context, _ *pgxpool.Pool, limit int) ([]outbox.OutboxEvent, error) {
			return []outbox.OutboxEvent{testEvent(nil)}, nil
		},
		MarkDispatchedFunc: func(_ context.Context, _ pgx.Tx, eventID string) error {
			mu.Lock()
			defer mu.Unlock()
			if eventID != "evt-test" {
				t.Errorf("MarkDispatched got eventID=%q, want %q", eventID, "evt-test")
			}
			markedDispatched = true
			return nil
		},
	}

	executors := map[string]action.ActionExecutor{
		"feishu": &mockActionExecutor{
			executeFunc: func(_ context.Context, _ action.ActionProposal, dryRun bool) (action.ExecutionResult, error) {
				return action.ExecutionResult{Success: true, DryRun: dryRun}, nil
			},
		},
	}

	// Test with DryRun=false so we verify actual dispatch
	w := newTestWorker(mockRepo, mockTxB, executors, DispatchConfig{
		PollInterval: 10 * time.Millisecond,
		BatchSize:    10,
		MaxRetries:   3,
		DryRun:       false,
	})
	w.processBatch(ctx)

	mu.Lock()
	if !markedDispatched {
		t.Error("expected MarkDispatched to be called")
	}
	mu.Unlock()
}

func TestDispatchWorker_MaxRetries(t *testing.T) {
	var mu sync.Mutex
	updatedMaxAttempts := false

	ctx := context.Background()
	mockTxB := &mockTxBeginner{}

	mockRepo := &mockOutboxRepository{
		GetPendingEventsFunc: func(_ context.Context, _ *pgxpool.Pool, limit int) ([]outbox.OutboxEvent, error) {
			evt := testEvent(nil)
			evt.DispatchAttempts = 10 // >= MaxRetries
			return []outbox.OutboxEvent{evt}, nil
		},
		UpdateMaxAttemptsFunc: func(_ context.Context, _ pgx.Tx, eventID string) error {
			mu.Lock()
			defer mu.Unlock()
			if eventID != "evt-test" {
				t.Errorf("UpdateMaxAttempts got eventID=%q, want %q", eventID, "evt-test")
			}
			updatedMaxAttempts = true
			return nil
		},
	}

	w := newTestWorker(mockRepo, mockTxB, nil, DispatchConfig{
		PollInterval: 10 * time.Millisecond,
		BatchSize:    10,
		MaxRetries:   3,
		DryRun:       true,
	})
	w.processBatch(ctx)

	mu.Lock()
	if !updatedMaxAttempts {
		t.Error("expected UpdateMaxAttempts to be called")
	}
	mu.Unlock()
}

func TestDispatchWorker_NoExecutor(t *testing.T) {
	var mu sync.Mutex
	markedFailed := false

	ctx := context.Background()
	mockTxB := &mockTxBeginner{}

	mockRepo := &mockOutboxRepository{
		GetPendingEventsFunc: func(_ context.Context, _ *pgxpool.Pool, limit int) ([]outbox.OutboxEvent, error) {
			evt := testEvent(&outbox.OutboxEvent{EventID: "no-exec", TargetChannel: "unknown_channel"})
			return []outbox.OutboxEvent{evt}, nil
		},
		MarkFailedFunc: func(_ context.Context, _ pgx.Tx, eventID, errMsg string) error {
			mu.Lock()
			defer mu.Unlock()
			if eventID != "no-exec" {
				t.Errorf("MarkFailed got eventID=%q, want %q", eventID, "no-exec")
			}
			if errMsg != "no executor for channel: unknown_channel" {
				t.Errorf("MarkFailed got errMsg=%q, want %q", errMsg, "no executor for channel: unknown_channel")
			}
			markedFailed = true
			return nil
		},
	}

	w := newTestWorker(mockRepo, mockTxB, nil, DefaultDispatchConfig())
	w.processBatch(ctx)

	mu.Lock()
	if !markedFailed {
		t.Error("expected MarkFailed to be called")
	}
	mu.Unlock()
}

func TestDispatchWorker_FailedExecuteError(t *testing.T) {
	var mu sync.Mutex
	markedFailed := false
	setRetry := false

	ctx := context.Background()
	mockTxB := &mockTxBeginner{}

	mockRepo := &mockOutboxRepository{
		GetPendingEventsFunc: func(_ context.Context, _ *pgxpool.Pool, limit int) ([]outbox.OutboxEvent, error) {
			return []outbox.OutboxEvent{testEvent(nil)}, nil
		},
		MarkFailedFunc: func(_ context.Context, _ pgx.Tx, eventID, errMsg string) error {
			mu.Lock()
			defer mu.Unlock()
			markedFailed = true
			return nil
		},
		SetNextRetryAtFunc: func(_ context.Context, _ pgx.Tx, eventID string, _ time.Time) error {
			mu.Lock()
			defer mu.Unlock()
			setRetry = true
			return nil
		},
	}

	executors := map[string]action.ActionExecutor{
		"feishu": &mockActionExecutor{
			executeFunc: func(_ context.Context, _ action.ActionProposal, dryRun bool) (action.ExecutionResult, error) {
				return action.ExecutionResult{}, fmt.Errorf("executor exploded")
			},
		},
	}

	w := newTestWorker(mockRepo, mockTxB, executors, DefaultDispatchConfig())
	w.processBatch(ctx)

	mu.Lock()
	if !markedFailed {
		t.Error("expected MarkFailed to be called on executor error")
	}
	if !setRetry {
		t.Error("expected SetNextRetryAt to be called after failure")
	}
	mu.Unlock()
}

func TestDispatchWorker_FailedExecuteResult(t *testing.T) {
	var mu sync.Mutex
	markedFailed := false

	ctx := context.Background()
	mockTxB := &mockTxBeginner{}

	mockRepo := &mockOutboxRepository{
		GetPendingEventsFunc: func(_ context.Context, _ *pgxpool.Pool, limit int) ([]outbox.OutboxEvent, error) {
			return []outbox.OutboxEvent{testEvent(nil)}, nil
		},
		MarkFailedFunc: func(_ context.Context, _ pgx.Tx, eventID, errMsg string) error {
			mu.Lock()
			defer mu.Unlock()
			markedFailed = true
			return nil
		},
		SetNextRetryAtFunc: func(_ context.Context, _ pgx.Tx, eventID string, _ time.Time) error {
			return nil
		},
	}

	executors := map[string]action.ActionExecutor{
		"feishu": &mockActionExecutor{
			executeFunc: func(_ context.Context, _ action.ActionProposal, dryRun bool) (action.ExecutionResult, error) {
				return action.ExecutionResult{Success: false, Error: "nope"}, nil
			},
		},
	}

	w := newTestWorker(mockRepo, mockTxB, executors, DefaultDispatchConfig())
	w.processBatch(ctx)

	mu.Lock()
	if !markedFailed {
		t.Error("expected MarkFailed to be called when executor returns !Success")
	}
	mu.Unlock()
}

func TestDispatchWorker_InvalidPayload(t *testing.T) {
	var mu sync.Mutex
	markedFailed := false

	ctx := context.Background()
	mockTxB := &mockTxBeginner{}

	mockRepo := &mockOutboxRepository{
		GetPendingEventsFunc: func(_ context.Context, _ *pgxpool.Pool, limit int) ([]outbox.OutboxEvent, error) {
			evt := testEvent(nil)
			evt.Payload = json.RawMessage(`{invalid json!!!}`)
			return []outbox.OutboxEvent{evt}, nil
		},
		MarkFailedFunc: func(_ context.Context, _ pgx.Tx, eventID, errMsg string) error {
			mu.Lock()
			defer mu.Unlock()
			markedFailed = true
			return nil
		},
		SetNextRetryAtFunc: func(_ context.Context, _ pgx.Tx, eventID string, _ time.Time) error {
			return nil
		},
	}

	executors := map[string]action.ActionExecutor{
		"feishu": &mockActionExecutor{},
	}

	w := newTestWorker(mockRepo, mockTxB, executors, DefaultDispatchConfig())
	w.processBatch(ctx)

	mu.Lock()
	if !markedFailed {
		t.Error("expected MarkFailed to be called on invalid payload")
	}
	mu.Unlock()
}

func TestDispatchWorker_GetPendingEventsError(t *testing.T) {
	ctx := context.Background()
	mockTxB := &mockTxBeginner{}

	mockRepo := &mockOutboxRepository{
		GetPendingEventsFunc: func(_ context.Context, _ *pgxpool.Pool, limit int) ([]outbox.OutboxEvent, error) {
			return nil, fmt.Errorf("db error")
		},
	}

	w := newTestWorker(mockRepo, mockTxB, nil, DefaultDispatchConfig())
	// Should not panic; just log the error
	w.processBatch(ctx)
}

func TestDispatchWorker_DryRunDispatches(t *testing.T) {
	var mu sync.Mutex
	markedDispatched := false

	ctx := context.Background()
	mockTxB := &mockTxBeginner{}

	mockRepo := &mockOutboxRepository{
		GetPendingEventsFunc: func(_ context.Context, _ *pgxpool.Pool, limit int) ([]outbox.OutboxEvent, error) {
			return []outbox.OutboxEvent{testEvent(nil)}, nil
		},
		MarkDispatchedFunc: func(_ context.Context, _ pgx.Tx, eventID string) error {
			mu.Lock()
			defer mu.Unlock()
			markedDispatched = true
			return nil
		},
	}

	executors := map[string]action.ActionExecutor{
		"feishu": &mockActionExecutor{
			executeFunc: func(_ context.Context, _ action.ActionProposal, dryRun bool) (action.ExecutionResult, error) {
				if !dryRun {
					t.Error("expected dryRun=true in DryRun mode")
				}
				return action.ExecutionResult{Success: true, DryRun: true}, nil
			},
		},
	}

	// DryRun: true — status should stay pending (Python behaviour)
	w := newTestWorker(mockRepo, mockTxB, executors, DispatchConfig{
		PollInterval: 10 * time.Millisecond,
		BatchSize:    10,
		MaxRetries:   3,
		DryRun:       true,
	})
	w.processBatch(ctx)

	mu.Lock()
	if markedDispatched {
		t.Error("expected MarkDispatched to NOT be called in dry-run mode (status stays pending)")
	}
	mu.Unlock()
}

func TestDispatchWorker_Stop(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockOutboxRepository{}
	mockTxB := &mockTxBeginner{}

	w := newTestWorker(mockRepo, mockTxB, nil, DispatchConfig{
		PollInterval: 10 * time.Millisecond,
		BatchSize:    10,
		MaxRetries:   3,
		DryRun:       true,
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := w.Run(ctx); err != nil {
			t.Errorf("Run returned error: %v", err)
		}
	}()

	time.Sleep(50 * time.Millisecond)
	w.Stop()

	select {
	case <-done:
		// Worker exited via Stop
	case <-time.After(time.Second):
		t.Fatal("worker did not stop within 1s after Stop()")
	}
}

// ---------------------------------------------------------------------------
// New tests for added functionality
// ---------------------------------------------------------------------------

func TestDispatchOne_Success(t *testing.T) {
	var mu sync.Mutex
	markedDispatched := false

	ctx := context.Background()
	mockTxB := &mockTxBeginner{}

	mockRepo := &mockOutboxRepository{
		MarkDispatchedFunc: func(_ context.Context, _ pgx.Tx, eventID string) error {
			mu.Lock()
			defer mu.Unlock()
			markedDispatched = true
			return nil
		},
	}

	executors := map[string]action.ActionExecutor{
		"feishu": &mockActionExecutor{
			executeFunc: func(_ context.Context, _ action.ActionProposal, dryRun bool) (action.ExecutionResult, error) {
				return action.ExecutionResult{Success: true}, nil
			},
		},
	}

	w := newTestWorker(mockRepo, mockTxB, executors, DefaultDispatchConfig())
	evt := testEvent(nil)
	result := w.DispatchOne(ctx, &evt)

	if result.Status != "dispatched" {
		t.Errorf("DispatchOne status = %q, want dispatched", result.Status)
	}
	if result.AdapterName == "" {
		t.Error("DispatchOne AdapterName should not be empty")
	}
	if result.DryRun {
		t.Error("DispatchOne DryRun should be false")
	}

	mu.Lock()
	if !markedDispatched {
		t.Error("expected MarkDispatched to be called")
	}
	mu.Unlock()
}

func TestDispatchOne_DryRun(t *testing.T) {
	var mu sync.Mutex
	markedDispatched := false

	ctx := context.Background()
	mockTxB := &mockTxBeginner{}

	mockRepo := &mockOutboxRepository{
		MarkDispatchedFunc: func(_ context.Context, _ pgx.Tx, eventID string) error {
			mu.Lock()
			defer mu.Unlock()
			markedDispatched = true
			return nil
		},
	}

	executors := map[string]action.ActionExecutor{
		"feishu": &mockActionExecutor{
			executeFunc: func(_ context.Context, _ action.ActionProposal, dryRun bool) (action.ExecutionResult, error) {
				return action.ExecutionResult{Success: true}, nil
			},
		},
	}

	w := newTestWorker(mockRepo, mockTxB, executors, DispatchConfig{
		MaxRetries: 3,
		DryRun:     true,
	})
	evt := testEvent(nil)
	result := w.DispatchOne(ctx, &evt)

	if result.Status != "dispatched" {
		t.Errorf("DispatchOne status = %q, want dispatched", result.Status)
	}
	if !result.DryRun {
		t.Error("DispatchOne DryRun should be true")
	}

	mu.Lock()
	if markedDispatched {
		t.Error("expected MarkDispatched to NOT be called in dry-run mode")
	}
	mu.Unlock()
}

func TestDispatchOne_MaxRetries(t *testing.T) {
	var mu sync.Mutex
	updatedMaxAttempts := false

	ctx := context.Background()
	mockTxB := &mockTxBeginner{}

	mockRepo := &mockOutboxRepository{
		UpdateMaxAttemptsFunc: func(_ context.Context, _ pgx.Tx, eventID string) error {
			mu.Lock()
			defer mu.Unlock()
			updatedMaxAttempts = true
			return nil
		},
	}

	w := newTestWorker(mockRepo, mockTxB, nil, DefaultDispatchConfig())
	evt := testEvent(&outbox.OutboxEvent{DispatchAttempts: 10})
	result := w.DispatchOne(ctx, &evt)

	if result.Status != "failed" {
		t.Errorf("DispatchOne status = %q, want failed", result.Status)
	}
	if !strings.Contains(result.Error, "max retry attempts reached") {
		t.Errorf("DispatchOne error = %q, want max retry attempts error", result.Error)
	}

	mu.Lock()
	if !updatedMaxAttempts {
		t.Error("expected UpdateMaxAttempts to be called")
	}
	mu.Unlock()
}

func TestDispatchOne_NoExecutor(t *testing.T) {
	var mu sync.Mutex
	markedFailed := false

	ctx := context.Background()
	mockTxB := &mockTxBeginner{}

	mockRepo := &mockOutboxRepository{
		MarkFailedFunc: func(_ context.Context, _ pgx.Tx, eventID, errMsg string) error {
			mu.Lock()
			defer mu.Unlock()
			markedFailed = true
			return nil
		},
	}

	w := newTestWorker(mockRepo, mockTxB, nil, DefaultDispatchConfig())
	evt := testEvent(&outbox.OutboxEvent{TargetChannel: "unknown"})
	result := w.DispatchOne(ctx, &evt)

	if result.Status != "failed" {
		t.Errorf("DispatchOne status = %q, want failed", result.Status)
	}
	if !strings.Contains(result.Error, "no executor for channel") {
		t.Errorf("DispatchOne error = %q, want no executor error", result.Error)
	}

	mu.Lock()
	if !markedFailed {
		t.Error("expected MarkFailed to be called")
	}
	mu.Unlock()
}

func TestFetchPending(t *testing.T) {
	ctx := context.Background()
	mockTxB := &mockTxBeginner{}

	expectedEvents := []outbox.OutboxEvent{testEvent(nil)}
	mockRepo := &mockOutboxRepository{
		GetPendingEventsFunc: func(_ context.Context, _ *pgxpool.Pool, limit int) ([]outbox.OutboxEvent, error) {
			if limit != 5 {
				t.Errorf("FetchPending limit = %d, want 5", limit)
			}
			return expectedEvents, nil
		},
	}

	w := newTestWorker(mockRepo, mockTxB, nil, DispatchConfig{BatchSize: 5})
	events, err := w.FetchPending(ctx, 5)
	if err != nil {
		t.Fatalf("FetchPending error: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("FetchPending returned %d events, want 1", len(events))
	}
}

func TestFetchPending_DefaultLimit(t *testing.T) {
	ctx := context.Background()
	mockTxB := &mockTxBeginner{}

	mockRepo := &mockOutboxRepository{
		GetPendingEventsFunc: func(_ context.Context, _ *pgxpool.Pool, limit int) ([]outbox.OutboxEvent, error) {
			if limit != 7 {
				t.Errorf("FetchPending limit = %d, want 7", limit)
			}
			return nil, nil
		},
	}

	w := newTestWorker(mockRepo, mockTxB, nil, DispatchConfig{BatchSize: 7})
	_, _ = w.FetchPending(ctx, 0) // 0 should fall back to BatchSize
}

func TestWriteAuditLog(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.csv")

	w := newTestWorker(&mockOutboxRepository{}, &mockTxBeginner{}, nil,
		DispatchConfig{AuditLogPath: logPath})

	entries := []auditLogEntry{
		{
			Timestamp:     time.Now().Format(time.RFC3339),
			EventID:       "evt-1",
			TargetChannel: "feishu",
			AdapterName:   "MockAdapter",
			Mode:          "live",
			Status:        "dispatched",
			ExternalRef:   "ref-1",
			Error:         "",
		},
	}
	w.writeAuditLog(entries)

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading audit log: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "timestamp") {
		t.Error("audit log missing header")
	}
	if !strings.Contains(content, "evt-1") {
		t.Error("audit log missing event ID")
	}
	if !strings.Contains(content, "MockAdapter") {
		t.Error("audit log missing adapter name")
	}
}

func TestWriteAuditLog_Append(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.csv")

	w := newTestWorker(&mockOutboxRepository{}, &mockTxBeginner{}, nil,
		DispatchConfig{AuditLogPath: logPath})

	w.writeAuditLog([]auditLogEntry{{EventID: "evt-1"}})
	w.writeAuditLog([]auditLogEntry{{EventID: "evt-2"}})

	data, _ := os.ReadFile(logPath)
	content := string(data)
	// Header should appear only once
	if strings.Count(content, "timestamp") != 1 {
		t.Error("audit log header should appear exactly once")
	}
	if !strings.Contains(content, "evt-1") || !strings.Contains(content, "evt-2") {
		t.Error("audit log missing appended entries")
	}
}

func TestWriteAuditLog_NoPath(t *testing.T) {
	w := newTestWorker(&mockOutboxRepository{}, &mockTxBeginner{}, nil,
		DispatchConfig{AuditLogPath: ""})

	// Should not panic
	w.writeAuditLog([]auditLogEntry{{EventID: "evt-1"}})
}

func TestWriteAuditLog_EmptyEntries(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.csv")

	w := newTestWorker(&mockOutboxRepository{}, &mockTxBeginner{}, nil,
		DispatchConfig{AuditLogPath: logPath})

	// Should not create file when entries are empty
	w.writeAuditLog([]auditLogEntry{})
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Error("audit log file should not be created for empty entries")
	}
}

func TestExecutorName(t *testing.T) {
	if got := executorName(nil); got != "" {
		t.Errorf("executorName(nil) = %q, want empty", got)
	}
	if got := executorName(action.NewNoOpExecutor()); got != "NoOpExecutor" {
		t.Errorf("executorName(NoOpExecutor) = %q, want NoOpExecutor", got)
	}
	if got := executorName(&mockActionExecutor{}); got != "*worker.mockActionExecutor" {
		t.Errorf("executorName(mockActionExecutor) = %q, want *worker.mockActionExecutor", got)
	}
}

func TestModeString(t *testing.T) {
	wDry := newTestWorker(nil, nil, nil, DispatchConfig{DryRun: true})
	if got := wDry.modeString(); got != "dry_run" {
		t.Errorf("modeString() = %q, want dry_run", got)
	}

	wLive := newTestWorker(nil, nil, nil, DispatchConfig{DryRun: false})
	if got := wLive.modeString(); got != "live" {
		t.Errorf("modeString() = %q, want live", got)
	}
}

// ---------------------------------------------------------------------------
// Error-path tests for transaction handling
// ---------------------------------------------------------------------------

func TestMarkDispatchedInTx_BeginError(t *testing.T) {
	ctx := context.Background()
	mockTxB := &mockTxBeginner{
		beginFunc: func(_ context.Context) (pgx.Tx, error) {
			return nil, fmt.Errorf("tx begin failed")
		},
	}

	w := newTestWorker(&mockOutboxRepository{}, mockTxB, nil, DefaultDispatchConfig())
	// Should not panic
	w.markDispatchedInTx(ctx, &outbox.OutboxEvent{EventID: "evt-test"})
}

func TestMarkDispatchedInTx_MarkError(t *testing.T) {
	ctx := context.Background()
	mockTxB := &mockTxBeginner{}

	mockRepo := &mockOutboxRepository{
		MarkDispatchedFunc: func(_ context.Context, _ pgx.Tx, eventID string) error {
			return fmt.Errorf("mark failed")
		},
	}

	w := newTestWorker(mockRepo, mockTxB, nil, DefaultDispatchConfig())
	// Should not panic
	w.markDispatchedInTx(ctx, &outbox.OutboxEvent{EventID: "evt-test"})
}

func TestMarkDispatchedInTx_CommitError(t *testing.T) {
	ctx := context.Background()
	mockTxB := &mockTxBeginner{
		beginFunc: func(_ context.Context) (pgx.Tx, error) {
			return &mockTxCommitError{}, nil
		},
	}

	mockRepo := &mockOutboxRepository{
		MarkDispatchedFunc: func(_ context.Context, _ pgx.Tx, eventID string) error {
			return nil
		},
	}

	w := newTestWorker(mockRepo, mockTxB, nil, DefaultDispatchConfig())
	// Should not panic
	w.markDispatchedInTx(ctx, &outbox.OutboxEvent{EventID: "evt-test"})
}

func TestHandleMaxAttempts_BeginError(t *testing.T) {
	ctx := context.Background()
	mockTxB := &mockTxBeginner{
		beginFunc: func(_ context.Context) (pgx.Tx, error) {
			return nil, fmt.Errorf("tx begin failed")
		},
	}

	w := newTestWorker(&mockOutboxRepository{}, mockTxB, nil, DefaultDispatchConfig())
	w.handleMaxAttempts(ctx, &outbox.OutboxEvent{EventID: "evt-test"})
}

func TestHandleMaxAttempts_UpdateError(t *testing.T) {
	ctx := context.Background()
	mockTxB := &mockTxBeginner{}

	mockRepo := &mockOutboxRepository{
		UpdateMaxAttemptsFunc: func(_ context.Context, _ pgx.Tx, eventID string) error {
			return fmt.Errorf("update failed")
		},
	}

	w := newTestWorker(mockRepo, mockTxB, nil, DefaultDispatchConfig())
	w.handleMaxAttempts(ctx, &outbox.OutboxEvent{EventID: "evt-test"})
}

func TestHandleMaxAttempts_CommitError(t *testing.T) {
	ctx := context.Background()
	mockTxB := &mockTxBeginner{
		beginFunc: func(_ context.Context) (pgx.Tx, error) {
			return &mockTxCommitError{}, nil
		},
	}

	mockRepo := &mockOutboxRepository{
		UpdateMaxAttemptsFunc: func(_ context.Context, _ pgx.Tx, eventID string) error {
			return nil
		},
	}

	w := newTestWorker(mockRepo, mockTxB, nil, DefaultDispatchConfig())
	w.handleMaxAttempts(ctx, &outbox.OutboxEvent{EventID: "evt-test"})
}

func TestHandleNoExecutor_BeginError(t *testing.T) {
	ctx := context.Background()
	mockTxB := &mockTxBeginner{
		beginFunc: func(_ context.Context) (pgx.Tx, error) {
			return nil, fmt.Errorf("tx begin failed")
		},
	}

	w := newTestWorker(&mockOutboxRepository{}, mockTxB, nil, DefaultDispatchConfig())
	w.handleNoExecutor(ctx, &outbox.OutboxEvent{EventID: "evt-test", TargetChannel: "unknown"})
}

func TestHandleNoExecutor_MarkError(t *testing.T) {
	ctx := context.Background()
	mockTxB := &mockTxBeginner{}

	mockRepo := &mockOutboxRepository{
		MarkFailedFunc: func(_ context.Context, _ pgx.Tx, eventID, errMsg string) error {
			return fmt.Errorf("mark failed")
		},
	}

	w := newTestWorker(mockRepo, mockTxB, nil, DefaultDispatchConfig())
	w.handleNoExecutor(ctx, &outbox.OutboxEvent{EventID: "evt-test", TargetChannel: "unknown"})
}

func TestHandleNoExecutor_CommitError(t *testing.T) {
	ctx := context.Background()
	mockTxB := &mockTxBeginner{
		beginFunc: func(_ context.Context) (pgx.Tx, error) {
			return &mockTxCommitError{}, nil
		},
	}

	mockRepo := &mockOutboxRepository{
		MarkFailedFunc: func(_ context.Context, _ pgx.Tx, eventID, errMsg string) error {
			return nil
		},
	}

	w := newTestWorker(mockRepo, mockTxB, nil, DefaultDispatchConfig())
	w.handleNoExecutor(ctx, &outbox.OutboxEvent{EventID: "evt-test", TargetChannel: "unknown"})
}

func TestHandleFailed_BeginError(t *testing.T) {
	ctx := context.Background()
	mockTxB := &mockTxBeginner{
		beginFunc: func(_ context.Context) (pgx.Tx, error) {
			return nil, fmt.Errorf("tx begin failed")
		},
	}

	w := newTestWorker(&mockOutboxRepository{}, mockTxB, nil, DefaultDispatchConfig())
	w.handleFailed(ctx, &outbox.OutboxEvent{EventID: "evt-test"}, "some error")
}

func TestHandleFailed_MarkError(t *testing.T) {
	ctx := context.Background()
	mockTxB := &mockTxBeginner{}

	mockRepo := &mockOutboxRepository{
		MarkFailedFunc: func(_ context.Context, _ pgx.Tx, eventID, errMsg string) error {
			return fmt.Errorf("mark failed")
		},
	}

	w := newTestWorker(mockRepo, mockTxB, nil, DefaultDispatchConfig())
	w.handleFailed(ctx, &outbox.OutboxEvent{EventID: "evt-test"}, "some error")
}

func TestHandleFailed_SetNextRetryError(t *testing.T) {
	ctx := context.Background()
	mockTxB := &mockTxBeginner{}

	mockRepo := &mockOutboxRepository{
		MarkFailedFunc: func(_ context.Context, _ pgx.Tx, eventID, errMsg string) error {
			return nil
		},
		SetNextRetryAtFunc: func(_ context.Context, _ pgx.Tx, eventID string, _ time.Time) error {
			return fmt.Errorf("set retry failed")
		},
	}

	w := newTestWorker(mockRepo, mockTxB, nil, DefaultDispatchConfig())
	w.handleFailed(ctx, &outbox.OutboxEvent{EventID: "evt-test"}, "some error")
}

func TestHandleFailed_CommitError(t *testing.T) {
	ctx := context.Background()
	mockTxB := &mockTxBeginner{
		beginFunc: func(_ context.Context) (pgx.Tx, error) {
			return &mockTxCommitError{}, nil
		},
	}

	mockRepo := &mockOutboxRepository{
		MarkFailedFunc: func(_ context.Context, _ pgx.Tx, eventID, errMsg string) error {
			return nil
		},
		SetNextRetryAtFunc: func(_ context.Context, _ pgx.Tx, eventID string, _ time.Time) error {
			return nil
		},
	}

	w := newTestWorker(mockRepo, mockTxB, nil, DefaultDispatchConfig())
	w.handleFailed(ctx, &outbox.OutboxEvent{EventID: "evt-test"}, "some error")
}

// mockTxCommitError is a mockTx that fails on Commit.
type mockTxCommitError struct {
	mockTx
}

func (tx *mockTxCommitError) Commit(ctx context.Context) error {
	return fmt.Errorf("commit failed")
}

func (tx *mockTxCommitError) Rollback(ctx context.Context) error {
	return nil
}
