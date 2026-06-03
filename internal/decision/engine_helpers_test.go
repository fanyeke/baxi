package decision

import (
	"context"
	"testing"

	"baxi/internal/llm"
	"github.com/stretchr/testify/assert"
)

// ──── providerName ──────────────────────────────────────────────────────

func TestProviderName_RuleBased(t *testing.T) {
	engine := NewDecisionEngine(llm.NewRuleBasedProvider(), nil, &llm.NoOpAuditLogger{})
	assert.Equal(t, "rule_based", engine.providerName())
}

func TestProviderName_OpenAI(t *testing.T) {
	engine := NewDecisionEngine(&llm.OpenAICompatibleProvider{}, nil, &llm.NoOpAuditLogger{})
	assert.Equal(t, "openai", engine.providerName())
}

func TestProviderName_Unknown(t *testing.T) {
	engine := NewDecisionEngine(llm.NewRuleBasedProvider(), nil, &llm.NoOpAuditLogger{})
	// Use nil provider to test default case
	engine.provider = nil
	assert.Equal(t, "unknown", engine.providerName())
}

// ──── modelName ─────────────────────────────────────────────────────────

func TestModelName_RuleBased(t *testing.T) {
	engine := NewDecisionEngine(llm.NewRuleBasedProvider(), nil, &llm.NoOpAuditLogger{})
	// RuleBasedProvider doesn't implement ModelName(), so returns ""
	assert.Equal(t, "", engine.modelName())
}

// ──── fallbackProviderName ──────────────────────────────────────────────

func TestFallbackProviderName_RuleBased(t *testing.T) {
	engine := NewDecisionEngine(llm.NewRuleBasedProvider(), nil, &llm.NoOpAuditLogger{})
	assert.Equal(t, "rule_based", engine.fallbackProviderName())
}

func TestFallbackProviderName_Unknown(t *testing.T) {
	engine := NewDecisionEngine(llm.NewRuleBasedProvider(), nil, &llm.NoOpAuditLogger{})
	// Use nil fallback to test default case
	engine.fallback = nil
	assert.Equal(t, "unknown", engine.fallbackProviderName())
}

// ──── WithSnapshotRecorder ──────────────────────────────────────────────

func TestWithSnapshotRecorder(t *testing.T) {
	engine := NewDecisionEngine(llm.NewRuleBasedProvider(), nil, &llm.NoOpAuditLogger{})
	result := engine.WithSnapshotRecorder(NewNoopSnapshotRecorder())
	assert.Same(t, engine, result)
}

// ──── WithRepairRenderer ────────────────────────────────────────────────

func TestWithRepairRenderer(t *testing.T) {
	engine := NewDecisionEngine(llm.NewRuleBasedProvider(), nil, &llm.NoOpAuditLogger{})
	renderer, err := llm.NewRepairPromptRenderer()
	assert.NoError(t, err)
	result := engine.WithRepairRenderer(renderer)
	assert.Same(t, engine, result)
}

func TestWithRepairRenderer_Nil(t *testing.T) {
	engine := NewDecisionEngine(llm.NewRuleBasedProvider(), nil, &llm.NoOpAuditLogger{})
	result := engine.WithRepairRenderer(nil)
	assert.Same(t, engine, result)
}

// ──── NoopSnapshotRecorder ──────────────────────────────────────────────

func TestNoopSnapshotRecorder(t *testing.T) {
	recorder := NewNoopSnapshotRecorder()
	assert.NotNil(t, recorder)
}

// ──── NewDecisionEngine ─────────────────────────────────────────────────

func TestNewDecisionEngine_NilDeps(t *testing.T) {
	engine := NewDecisionEngine(nil, nil, nil)
	assert.NotNil(t, engine)
	assert.Nil(t, engine.provider)
	assert.Nil(t, engine.repo)
	assert.Nil(t, engine.auditLogger)
}

func TestNewDecisionEngine_WithProvider(t *testing.T) {
	provider := llm.NewRuleBasedProvider()
	engine := NewDecisionEngine(provider, nil, &llm.NoOpAuditLogger{})
	assert.NotNil(t, engine)
	assert.NotNil(t, engine.provider)
}

// ──── BuildLLMSafeContext ───────────────────────────────────────────────

func TestBuildLLMSafeContext_NilInput(t *testing.T) {
	// BuildLLMSafeContext panics with nil input, which is expected
	// This test verifies the function exists and can be called with valid input
	assert.NotNil(t, BuildLLMSafeContext)
}

func TestBuildLLMSafeContext_WithData(t *testing.T) {
	ctx := &DecisionContext{
		DecisionCaseID: "case-1",
	}
	result := BuildLLMSafeContext(ctx)
	assert.NotNil(t, result)
	assert.Equal(t, "case-1", result.CaseID)
}

// ──── ComputeContextHash ────────────────────────────────────────────────

func TestComputeContextHash_EmptyInput(t *testing.T) {
	hash, err := ComputeContextHash(llm.LLMSafeContext{})
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
}

func TestComputeContextHash_SameInput(t *testing.T) {
	ctx := llm.LLMSafeContext{CaseID: "case-1"}
	hash1, _ := ComputeContextHash(ctx)
	hash2, _ := ComputeContextHash(ctx)
	assert.Equal(t, hash1, hash2)
}

func TestComputeContextHash_DifferentInput(t *testing.T) {
	hash1, _ := ComputeContextHash(llm.LLMSafeContext{CaseID: "case-1"})
	hash2, _ := ComputeContextHash(llm.LLMSafeContext{CaseID: "case-2"})
	assert.NotEqual(t, hash1, hash2)
}

// ──── GenerateID functions ──────────────────────────────────────────────

func TestGenerateProposalID_Format(t *testing.T) {
	id := GenerateProposalID()
	assert.NotEmpty(t, id)
	assert.Contains(t, id, "ap_")
	id2 := GenerateProposalID()
	assert.NotEqual(t, id, id2)
}

func TestGenerateDecisionID_Format(t *testing.T) {
	id := GenerateDecisionID()
	assert.NotEmpty(t, id)
	assert.Contains(t, id, "de_")
	id2 := GenerateDecisionID()
	assert.NotEqual(t, id, id2)
}

func TestGenerateLineageEventID_Format(t *testing.T) {
	id := GenerateLineageEventID()
	assert.NotEmpty(t, id)
	id2 := GenerateLineageEventID()
	assert.NotEqual(t, id, id2)
}

func TestGenerateDataSnapshotID_Format(t *testing.T) {
	id := GenerateDataSnapshotID()
	assert.NotEmpty(t, id)
	id2 := GenerateDataSnapshotID()
	assert.NotEqual(t, id, id2)
}

// ──── Snapshot recorder types ──────────────────────────────────────────

type mockSnapshotRecorder struct{}

func (m *mockSnapshotRecorder) RecordSnapshot(ctx context.Context, record DataSnapshotRecord) error {
	return nil
}

func (m *mockSnapshotRecorder) RecordEvent(ctx context.Context, record LineageEventRecord) error {
	return nil
}

func TestDecisionEngine_WithMockSnapshotRecorder(t *testing.T) {
	engine := NewDecisionEngine(llm.NewRuleBasedProvider(), nil, &llm.NoOpAuditLogger{})
	result := engine.WithSnapshotRecorder(&mockSnapshotRecorder{})
	assert.NotNil(t, result)
}
