package worker

import (
	"context"
	"encoding/json"
	"fmt"
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
		MaxRetries:   10,
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
		MaxRetries:   10,
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
		MaxRetries:   10,
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

	// DryRun: true
	w := newTestWorker(mockRepo, mockTxB, executors, DispatchConfig{
		PollInterval: 10 * time.Millisecond,
		BatchSize:    10,
		MaxRetries:   10,
		DryRun:       true,
	})
	w.processBatch(ctx)

	mu.Lock()
	if !markedDispatched {
		t.Error("expected MarkDispatched to be called in dry-run mode")
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
		MaxRetries:   10,
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
