package eval

import (
	"context"
	"encoding/json"
	"testing"

	"baxi/internal/llm"
)

// mockReplayRepo implements DecisionRepository for testing.
type mockReplayRepo struct {
	data *ReplayData
	err  error
}

func (m *mockReplayRepo) GetLLMDecisionByCaseID(ctx context.Context, caseID string) (*ReplayData, error) {
	return m.data, m.err
}

// mockProvider implements llm.DecisionProvider for testing.
type mockProvider struct {
	output *llm.DecisionOutput
	err    error
	called bool
}

func (m *mockProvider) GenerateDecision(ctx context.Context, input llm.LLMSafeContext) (*llm.DecisionOutput, error) {
	m.called = true
	return m.output, m.err
}

// mockAuditLogger implements llm.LLMAuditLogger for testing.
type mockAuditLogger struct {
	replayedCall string
}

func (m *mockAuditLogger) LogDecisionRequested(ctx context.Context, caseID, provider, model string) {}
func (m *mockAuditLogger) LogDecisionCompleted(ctx context.Context, caseID, provider, model string, latencyMs int64, usage *llm.TokenUsage) {
}
func (m *mockAuditLogger) LogDecisionFailed(ctx context.Context, caseID, provider, model string, err error) {
}
func (m *mockAuditLogger) LogDecisionValidationFailed(ctx context.Context, caseID string, errors []llm.ValidationError) {
}
func (m *mockAuditLogger) LogFallbackUsed(ctx context.Context, caseID string, reason string) {
}
func (m *mockAuditLogger) LogDecisionReplayed(ctx context.Context, caseID, originalDecisionID string) {
	m.replayedCall = caseID
}
func (m *mockAuditLogger) LogEvalCompleted(ctx context.Context, caseID, evalID string) {
}

// replayTestData returns a ReplayData used as baseline for replay tests.
func replayTestData() *ReplayData {
	inputCtx := llm.LLMSafeContext{
		CaseID: "case-replay-1",
		Trigger: llm.TriggerInfo{
			AlertID:    "alert-1",
			Severity:   llm.SeverityMedium,
			MetricName: "revenue",
		},
		ObjectContext: llm.ObjectContext{
			ObjectType: "seller",
			ObjectID:   "seller-123",
		},
		AllowedActions: []string{llm.ActionTypeNotifyOwner},
	}
	inputJSON, _ := json.Marshal(inputCtx)

	return &ReplayData{
		CaseID:             "case-replay-1",
		OriginalDecisionID: "decision-abc-123",
		InputContext:       inputJSON,
		OriginalOutput:     validDecisionOutput(),
		Provider:           "openai",
		Model:              "gpt-4o",
		PromptVersion:      "v2",
		ContextHash:        "hash-abc",
	}
}

func TestReplayDryRun_ReturnsOriginalWithoutProviderCall(t *testing.T) {
	repo := &mockReplayRepo{data: replayTestData()}
	provider := &mockProvider{output: validDecisionOutput()}
	logger := &mockAuditLogger{}
	svc := NewReplayService(repo, provider, logger)

	result, err := svc.Replay(context.Background(), "case-replay-1", ReplayOptions{DryRun: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.DryRun != true {
		t.Error("expected DryRun to be true")
	}

	if result.ReplayedDecision != nil {
		t.Error("expected ReplayedDecision to be nil in dry-run mode")
	}

	if logger.replayedCall != "" {
		t.Error("expected no audit log in dry-run mode")
	}

	if provider.called {
		t.Error("expected provider not to be called in dry-run mode")
	}
}

func TestReplayResult_Fields(t *testing.T) {
	repo := &mockReplayRepo{data: replayTestData()}
	provider := &mockProvider{output: validDecisionOutput()}
	logger := &mockAuditLogger{}
	svc := NewReplayService(repo, provider, logger)

	result, err := svc.Replay(context.Background(), "case-replay-1", ReplayOptions{DryRun: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ContextHash != "hash-abc" {
		t.Errorf("expected ContextHash=hash-abc, got %s", result.ContextHash)
	}

	if result.PromptVersion != "v2" {
		t.Errorf("expected PromptVersion=v2, got %s", result.PromptVersion)
	}

	if result.Model != "gpt-4o" {
		t.Errorf("expected Model=gpt-4o, got %s", result.Model)
	}

	if result.DryRun != false {
		t.Error("expected DryRun to be false")
	}

	if result.OriginalDecision == nil {
		t.Fatal("expected OriginalDecision to be non-nil")
	}

	if result.ReplayedDecision == nil {
		t.Fatal("expected ReplayedDecision to be non-nil")
	}

	if result.OriginalDecision.DecisionType != result.ReplayedDecision.DecisionType {
		t.Error("expected replayed decision type to match original")
	}

	if logger.replayedCall != "case-replay-1" {
		t.Errorf("expected audit log for case-replay-1, got %s", logger.replayedCall)
	}
}

func TestReplayDryRun_ContextHashAndModelPreserved(t *testing.T) {
	repo := &mockReplayRepo{data: replayTestData()}
	provider := &mockProvider{output: validDecisionOutput()}
	logger := &mockAuditLogger{}
	svc := NewReplayService(repo, provider, logger)

	result, err := svc.Replay(context.Background(), "case-replay-1", ReplayOptions{DryRun: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ContextHash != "hash-abc" {
		t.Errorf("expected ContextHash=hash-abc, got %s", result.ContextHash)
	}

	if result.PromptVersion != "v2" {
		t.Errorf("expected PromptVersion=v2, got %s", result.PromptVersion)
	}

	if result.Model != "gpt-4o" {
		t.Errorf("expected Model=gpt-4o, got %s", result.Model)
	}
}

func TestReplay_ProviderError_ReturnsError(t *testing.T) {
	repo := &mockReplayRepo{data: replayTestData()}
	provider := &mockProvider{err: assertAnError{}}
	logger := &mockAuditLogger{}
	svc := NewReplayService(repo, provider, logger)

	_, err := svc.Replay(context.Background(), "case-replay-1", ReplayOptions{DryRun: false})
	if err == nil {
		t.Fatal("expected error when provider fails")
	}
}

type assertAnError struct{}

func (assertAnError) Error() string { return "test error" }

func TestReplay_RepositoryError_ReturnsError(t *testing.T) {
	repo := &mockReplayRepo{err: assertAnError{}}
	provider := &mockProvider{output: validDecisionOutput()}
	logger := &mockAuditLogger{}
	svc := NewReplayService(repo, provider, logger)

	_, err := svc.Replay(context.Background(), "case-replay-nonexistent", ReplayOptions{DryRun: false})
	if err == nil {
		t.Fatal("expected error when repository fails")
	}
}

func TestReplay_NilOriginalOutput(t *testing.T) {
	data := replayTestData()
	data.OriginalOutput = nil
	repo := &mockReplayRepo{data: data}
	provider := &mockProvider{output: validDecisionOutput()}
	logger := &mockAuditLogger{}
	svc := NewReplayService(repo, provider, logger)

	result, err := svc.Replay(context.Background(), "case-replay-1", ReplayOptions{DryRun: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.OriginalDecision != nil {
		t.Error("expected OriginalDecision to be nil when original output is nil")
	}
}

func TestNewReplayService_NonNil(t *testing.T) {
	svc := NewReplayService(nil, nil, nil)
	if svc == nil {
		t.Fatal("expected non-nil ReplayService")
	}
}
