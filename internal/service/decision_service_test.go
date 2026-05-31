package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/action"
	"baxi/internal/decision"
	"baxi/internal/eval"
	"baxi/internal/llm"
)

// ──── mocks ────────────────────────────────────────────────────────────

type mockCaseService struct {
	createCaseFromAlertFn func(ctx context.Context, alertID, createdBy string) (*decision.DecisionCase, error)
	getCaseFn             func(ctx context.Context, caseID string) (*decision.DecisionCase, error)
	listCasesFn           func(ctx context.Context, filter decision.CaseFilter) (*decision.CaseList, error)
}

func (m *mockCaseService) CreateCaseFromAlert(ctx context.Context, alertID, createdBy string) (*decision.DecisionCase, error) {
	return m.createCaseFromAlertFn(ctx, alertID, createdBy)
}

func (m *mockCaseService) GetCase(ctx context.Context, caseID string) (*decision.DecisionCase, error) {
	return m.getCaseFn(ctx, caseID)
}

func (m *mockCaseService) ListCases(ctx context.Context, filter decision.CaseFilter) (*decision.CaseList, error) {
	return m.listCasesFn(ctx, filter)
}

type mockContextBuilder struct {
	buildDecisionContextFn func(ctx context.Context, caseID string) (*decision.DecisionContext, error)
}

func (m *mockContextBuilder) BuildDecisionContext(ctx context.Context, caseID string) (*decision.DecisionContext, error) {
	return m.buildDecisionContextFn(ctx, caseID)
}

type mockDecisionEngine struct {
	generateDecisionFn func(ctx context.Context, caseID string, context *decision.DecisionContext) (*llm.DecisionOutput, error)
}

func (m *mockDecisionEngine) GenerateDecision(ctx context.Context, caseID string, context *decision.DecisionContext) (*llm.DecisionOutput, error) {
	return m.generateDecisionFn(ctx, caseID, context)
}

type mockProposalService struct {
	generateProposalsFn func(ctx context.Context, caseID, decisionID string, dec *llm.DecisionOutput, contextHash string) ([]action.ActionProposal, error)
	listProposalsFn     func(ctx context.Context, caseID string) ([]action.ActionProposal, error)
}

func (m *mockProposalService) GenerateProposals(ctx context.Context, caseID, decisionID string, dec *llm.DecisionOutput, contextHash string) ([]action.ActionProposal, error) {
	return m.generateProposalsFn(ctx, caseID, decisionID, dec, contextHash)
}

func (m *mockProposalService) ListProposals(ctx context.Context, caseID string) ([]action.ActionProposal, error) {
	return m.listProposalsFn(ctx, caseID)
}

// ──── helpers ──────────────────────────────────────────────────────────

func newTestDecisionService(
	caseSvc CaseService,
	ctxBuilder ContextBuilder,
	engine DecisionEngine,
	proposalSvc ProposalService,
) *DecisionService {
	return NewDecisionService(caseSvc, ctxBuilder, engine, proposalSvc, nil)
}

func testDecisionCase() *decision.DecisionCase {
	return &decision.DecisionCase{
		CaseID:     "dc_1000000_testabc",
		AlertID:    strPtr("alert-1"),
		SourceType: strPtr("alert"),
		SourceID:   strPtr("alert-1"),
		Status:     "created",
		CreatedAt:  time.Now(),
		CreatedBy:  "test-user",
	}
}

func testDecisionContext() *decision.DecisionContext {
	return &decision.DecisionContext{
		DecisionCaseID: "dc_1000000_testabc",
		SourceType:     strPtr("alert"),
		SourceID:       strPtr("alert-1"),
		Trigger: decision.TriggerInfo{
			AlertID:  "alert-1",
			Severity: "high",
		},
		AllowedActions: []string{"notify_owner", "escalate_to_human"},
		Governance: decision.GovernanceData{
			Role: "agent_readonly",
		},
	}
}

func testDecisionOutput() *llm.DecisionOutput {
	return &llm.DecisionOutput{
		DecisionType: llm.DecisionTypeInvestigate,
		Severity:     llm.SeverityHigh,
		Summary:      "Investigate anomaly in seller performance",
		Rationale:    []string{"Metric dropped by 20%"},
		RecommendedActions: []llm.RecommendedAction{
			{ActionType: "notify_owner", Priority: "high", OwnerRole: "admin"},
		},
		Confidence:          0.85,
		RequiresHumanReview: true,
	}
}

func testActionProposals() []action.ActionProposal {
	return []action.ActionProposal{
		{
			ProposalID:          "ap_1000000_propabc",
			CaseID:              "dc_1000000_testabc",
			ActionType:          "notify_owner",
			Title:               "Notify owner about anomaly",
			RiskLevel:           "high",
			RequiresHumanReview: true,
			CreatedAt:           time.Now(),
		},
	}
}

func testCaseList() *decision.CaseList {
	return &decision.CaseList{
		Cases: []decision.DecisionCase{*testDecisionCase()},
		Total: 1,
	}
}

// ──── tests ────────────────────────────────────────────────────────────

func TestDecisionService_CreateCaseFromAlert(t *testing.T) {
	expected := testDecisionCase()
	caseSvc := &mockCaseService{
		createCaseFromAlertFn: func(_ context.Context, alertID, createdBy string) (*decision.DecisionCase, error) {
			assert.Equal(t, "alert-1", alertID)
			assert.Equal(t, "tester", createdBy)
			return expected, nil
		},
	}

	svc := newTestDecisionService(caseSvc, nil, nil, nil)
	got, err := svc.CreateCaseFromAlert(context.Background(), "alert-1", "tester")

	require.NoError(t, err)
	assert.Same(t, expected, got)
}

func TestDecisionService_BuildContext(t *testing.T) {
	expected := testDecisionContext()
	ctxBuilder := &mockContextBuilder{
		buildDecisionContextFn: func(_ context.Context, caseID string) (*decision.DecisionContext, error) {
			assert.Equal(t, "dc_1000000_testabc", caseID)
			return expected, nil
		},
	}

	svc := newTestDecisionService(nil, ctxBuilder, nil, nil)
	got, err := svc.BuildContext(context.Background(), "dc_1000000_testabc")

	require.NoError(t, err)
	assert.Same(t, expected, got)
}

func TestDecisionService_Decide(t *testing.T) {
	expectedCase := testDecisionCase()
	expectedCtx := testDecisionContext()
	expectedOutput := testDecisionOutput()
	expectedProps := testActionProposals()

	var recordedGenerateProposalsCaseID string
	var recordedGenerateProposalsDecisionID string
	var recordedGenerateProposalsOutput *llm.DecisionOutput

	caseSvc := &mockCaseService{
		getCaseFn: func(_ context.Context, caseID string) (*decision.DecisionCase, error) {
			assert.Equal(t, "dc_1000000_testabc", caseID)
			return expectedCase, nil
		},
	}
	ctxBuilder := &mockContextBuilder{
		buildDecisionContextFn: func(_ context.Context, caseID string) (*decision.DecisionContext, error) {
			assert.Equal(t, "dc_1000000_testabc", caseID)
			return expectedCtx, nil
		},
	}
	engine := &mockDecisionEngine{
		generateDecisionFn: func(_ context.Context, caseID string, ctx *decision.DecisionContext) (*llm.DecisionOutput, error) {
			assert.Equal(t, "dc_1000000_testabc", caseID)
			assert.Same(t, expectedCtx, ctx)
			return expectedOutput, nil
		},
	}
	proposalSvc := &mockProposalService{
		generateProposalsFn: func(_ context.Context, caseID, decisionID string, dec *llm.DecisionOutput, _ string) ([]action.ActionProposal, error) {
			recordedGenerateProposalsCaseID = caseID
			recordedGenerateProposalsDecisionID = decisionID
			recordedGenerateProposalsOutput = dec
			return expectedProps, nil
		},
	}

	svc := newTestDecisionService(caseSvc, ctxBuilder, engine, proposalSvc)
	ctx, output, proposals, err := svc.Decide(context.Background(), "dc_1000000_testabc")

	require.NoError(t, err)
	assert.Same(t, expectedCtx, ctx)
	assert.Same(t, expectedOutput, output)
	assert.Equal(t, expectedProps, proposals)

	assert.Equal(t, "dc_1000000_testabc", recordedGenerateProposalsCaseID)
	assert.NotEmpty(t, recordedGenerateProposalsDecisionID, "should generate a decision ID")
	assert.Contains(t, recordedGenerateProposalsDecisionID, "de_")
	assert.Same(t, expectedOutput, recordedGenerateProposalsOutput)
}

func TestDecisionService_Decide_GetCaseError(t *testing.T) {
	caseSvc := &mockCaseService{
		getCaseFn: func(_ context.Context, caseID string) (*decision.DecisionCase, error) {
			return nil, assert.AnError
		},
	}
	svc := newTestDecisionService(caseSvc, nil, nil, nil)
	ctx, output, proposals, err := svc.Decide(context.Background(), "dc_unknown")
	assert.Error(t, err)
	assert.Nil(t, ctx)
	assert.Nil(t, output)
	assert.Nil(t, proposals)
}

func TestDecisionService_Decide_BuildContextError(t *testing.T) {
	caseSvc := &mockCaseService{
		getCaseFn: func(_ context.Context, caseID string) (*decision.DecisionCase, error) {
			return testDecisionCase(), nil
		},
	}
	ctxBuilder := &mockContextBuilder{
		buildDecisionContextFn: func(_ context.Context, caseID string) (*decision.DecisionContext, error) {
			return nil, assert.AnError
		},
	}
	svc := newTestDecisionService(caseSvc, ctxBuilder, nil, nil)
	ctx, output, proposals, err := svc.Decide(context.Background(), "dc_1000000_testabc")
	assert.Error(t, err)
	assert.Nil(t, ctx)
	assert.Nil(t, output)
	assert.Nil(t, proposals)
}

func TestDecisionService_Decide_GenerateDecisionError(t *testing.T) {
	caseSvc := &mockCaseService{
		getCaseFn: func(_ context.Context, caseID string) (*decision.DecisionCase, error) {
			return testDecisionCase(), nil
		},
	}
	ctxBuilder := &mockContextBuilder{
		buildDecisionContextFn: func(_ context.Context, caseID string) (*decision.DecisionContext, error) {
			return testDecisionContext(), nil
		},
	}
	engine := &mockDecisionEngine{
		generateDecisionFn: func(_ context.Context, caseID string, ctx *decision.DecisionContext) (*llm.DecisionOutput, error) {
			return nil, assert.AnError
		},
	}
	svc := newTestDecisionService(caseSvc, ctxBuilder, engine, nil)
	ctx, output, proposals, err := svc.Decide(context.Background(), "dc_1000000_testabc")
	assert.Error(t, err)
	assert.Nil(t, ctx)
	assert.Nil(t, output)
	assert.Nil(t, proposals)
}

func TestDecisionService_Decide_GenerateProposalsError(t *testing.T) {
	caseSvc := &mockCaseService{
		getCaseFn: func(_ context.Context, caseID string) (*decision.DecisionCase, error) {
			return testDecisionCase(), nil
		},
	}
	ctxBuilder := &mockContextBuilder{
		buildDecisionContextFn: func(_ context.Context, caseID string) (*decision.DecisionContext, error) {
			return testDecisionContext(), nil
		},
	}
	engine := &mockDecisionEngine{
		generateDecisionFn: func(_ context.Context, caseID string, ctx *decision.DecisionContext) (*llm.DecisionOutput, error) {
			return testDecisionOutput(), nil
		},
	}
	proposalSvc := &mockProposalService{
		generateProposalsFn: func(_ context.Context, caseID, decisionID string, dec *llm.DecisionOutput, _ string) ([]action.ActionProposal, error) {
			return nil, assert.AnError
		},
	}
	svc := newTestDecisionService(caseSvc, ctxBuilder, engine, proposalSvc)
	ctx, output, proposals, err := svc.Decide(context.Background(), "dc_1000000_testabc")
	assert.Error(t, err)
	assert.Nil(t, ctx)
	assert.Nil(t, output)
	assert.Nil(t, proposals)
}

func TestDecisionService_GetCase(t *testing.T) {
	expected := testDecisionCase()
	caseSvc := &mockCaseService{
		getCaseFn: func(_ context.Context, caseID string) (*decision.DecisionCase, error) {
			assert.Equal(t, "dc_1000000_testabc", caseID)
			return expected, nil
		},
	}

	svc := newTestDecisionService(caseSvc, nil, nil, nil)
	got, err := svc.GetCase(context.Background(), "dc_1000000_testabc")

	require.NoError(t, err)
	assert.Same(t, expected, got)
}

func TestDecisionService_ListCases(t *testing.T) {
	expected := testCaseList()
	filter := decision.CaseFilter{Status: strPtr("created"), Limit: 10, Offset: 0}

	caseSvc := &mockCaseService{
		listCasesFn: func(_ context.Context, f decision.CaseFilter) (*decision.CaseList, error) {
			assert.Equal(t, "created", *f.Status)
			assert.Equal(t, 10, f.Limit)
			assert.Equal(t, 0, f.Offset)
			return expected, nil
		},
	}

	svc := newTestDecisionService(caseSvc, nil, nil, nil)
	got, err := svc.ListCases(context.Background(), filter)

	require.NoError(t, err)
	assert.Same(t, expected, got)
}

func TestDecisionService_ListProposals(t *testing.T) {
	expected := testActionProposals()
	proposalSvc := &mockProposalService{
		listProposalsFn: func(_ context.Context, caseID string) ([]action.ActionProposal, error) {
			assert.Equal(t, "dc_1000000_testabc", caseID)
			return expected, nil
		},
	}

	svc := newTestDecisionService(nil, nil, nil, proposalSvc)
	got, err := svc.ListProposals(context.Background(), "dc_1000000_testabc")

	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

// ──── mockLlmProvider ──────────────────────────────────────────────────────

type mockLlmProvider struct {
	generateDecisionFn func(ctx context.Context, input llm.LLMSafeContext) (*llm.DecisionOutput, error)
}

func (m *mockLlmProvider) GenerateDecision(ctx context.Context, input llm.LLMSafeContext) (*llm.DecisionOutput, error) {
	return m.generateDecisionFn(ctx, input)
}

// ──── DecideLLM tests ──────────────────────────────────────────────────────

func TestDecisionService_DecideLLM(t *testing.T) {
	expectedCase := testDecisionCase()
	expectedCtx := testDecisionContext()
	expectedOutput := testDecisionOutput()
	expectedProps := testActionProposals()

	caseSvc := &mockCaseService{
		getCaseFn: func(_ context.Context, caseID string) (*decision.DecisionCase, error) {
			return expectedCase, nil
		},
	}
	ctxBuilder := &mockContextBuilder{
		buildDecisionContextFn: func(_ context.Context, caseID string) (*decision.DecisionContext, error) {
			return expectedCtx, nil
		},
	}
	engine := &mockDecisionEngine{
		generateDecisionFn: func(_ context.Context, caseID string, ctx *decision.DecisionContext) (*llm.DecisionOutput, error) {
			return expectedOutput, nil
		},
	}
	proposalSvc := &mockProposalService{
		generateProposalsFn: func(_ context.Context, caseID, decisionID string, dec *llm.DecisionOutput, _ string) ([]action.ActionProposal, error) {
			return expectedProps, nil
		},
	}

	svc := NewDecisionService(caseSvc, ctxBuilder, engine, proposalSvc, nil)
	ctx, output, proposals, err := svc.DecideLLM(context.Background(), "dc_1000000_testabc")

	require.NoError(t, err)
	assert.Same(t, expectedCtx, ctx)
	assert.Same(t, expectedOutput, output)
	assert.Equal(t, expectedProps, proposals)
}

func TestDecisionService_DecideLLM_WithMetrics(t *testing.T) {
	expectedCase := testDecisionCase()
	expectedCtx := testDecisionContext()
	expectedOutput := testDecisionOutput()
	expectedProps := testActionProposals()

	caseSvc := &mockCaseService{
		getCaseFn: func(_ context.Context, caseID string) (*decision.DecisionCase, error) {
			return expectedCase, nil
		},
	}
	ctxBuilder := &mockContextBuilder{
		buildDecisionContextFn: func(_ context.Context, caseID string) (*decision.DecisionContext, error) {
			return expectedCtx, nil
		},
	}
	engine := &mockDecisionEngine{
		generateDecisionFn: func(_ context.Context, caseID string, ctx *decision.DecisionContext) (*llm.DecisionOutput, error) {
			return expectedOutput, nil
		},
	}
	proposalSvc := &mockProposalService{
		generateProposalsFn: func(_ context.Context, caseID, decisionID string, dec *llm.DecisionOutput, _ string) ([]action.ActionProposal, error) {
			return expectedProps, nil
		},
	}

	metrics := eval.NewMetricsCollector()
	svc := NewDecisionService(caseSvc, ctxBuilder, engine, proposalSvc, nil)
	svc.WithMetrics(metrics)

	ctx, output, proposals, err := svc.DecideLLM(context.Background(), "dc_1000000_testabc")
	require.NoError(t, err)
	assert.Same(t, expectedCtx, ctx)
	assert.Same(t, expectedOutput, output)
	assert.Equal(t, expectedProps, proposals)

	// Verify metrics were recorded
	m := metrics.GetMetrics()
	assert.Equal(t, 1, m.TotalDecisions)
	assert.Equal(t, 1, m.ProviderDecisionCount["llm"])
}

// ──── Compare tests ─────────────────────────────────────────────────────────

func TestDecisionService_Compare(t *testing.T) {
	caseSvc := &mockCaseService{
		getCaseFn: func(_ context.Context, caseID string) (*decision.DecisionCase, error) {
			return testDecisionCase(), nil
		},
	}
	ctxBuilder := &mockContextBuilder{
		buildDecisionContextFn: func(_ context.Context, caseID string) (*decision.DecisionContext, error) {
			return testDecisionContext(), nil
		},
	}
	engine := &mockDecisionEngine{
		generateDecisionFn: func(_ context.Context, caseID string, ctx *decision.DecisionContext) (*llm.DecisionOutput, error) {
			return testDecisionOutput(), nil
		},
	}
	ruleProvider := &mockLlmProvider{
		generateDecisionFn: func(_ context.Context, input llm.LLMSafeContext) (*llm.DecisionOutput, error) {
			return &llm.DecisionOutput{
				DecisionType: "monitor_only",
				Severity:     "low",
				Summary:      "rule-based decision",
				Confidence:   0.5,
			}, nil
		},
	}

	svc := newTestDecisionService(caseSvc, ctxBuilder, engine, nil)
	svc.WithRuleProvider(ruleProvider)

	comparison, err := svc.Compare(context.Background(), "dc_1000000_testabc")
	require.NoError(t, err)
	require.NotNil(t, comparison)
	assert.Equal(t, "dc_1000000_testabc", comparison.DecisionCaseID)
	assert.False(t, comparison.DecisionTypeMatch) // investigate vs monitor_only
	assert.False(t, comparison.SeverityMatch)      // high vs low
	assert.True(t, comparison.LLMValid)
}

func TestDecisionService_Compare_NoRuleProvider(t *testing.T) {
	caseSvc := &mockCaseService{
		getCaseFn: func(_ context.Context, caseID string) (*decision.DecisionCase, error) {
			return testDecisionCase(), nil
		},
	}
	ctxBuilder := &mockContextBuilder{
		buildDecisionContextFn: func(_ context.Context, caseID string) (*decision.DecisionContext, error) {
			return testDecisionContext(), nil
		},
	}
	engine := &mockDecisionEngine{
		generateDecisionFn: func(_ context.Context, caseID string, ctx *decision.DecisionContext) (*llm.DecisionOutput, error) {
			return testDecisionOutput(), nil
		},
	}

	// No WithRuleProvider → should use fallback monitor_only decision
	svc := newTestDecisionService(caseSvc, ctxBuilder, engine, nil)
	comparison, err := svc.Compare(context.Background(), "dc_1000000_testabc")
	require.NoError(t, err)
	require.NotNil(t, comparison)
	assert.Equal(t, "dc_1000000_testabc", comparison.DecisionCaseID)
	assert.Equal(t, "monitor_only", comparison.RuleDecisionType)
	assert.False(t, comparison.DecisionTypeMatch) // investigate vs monitor_only
}

func TestDecisionService_Compare_GetCaseError(t *testing.T) {
	caseSvc := &mockCaseService{
		getCaseFn: func(_ context.Context, caseID string) (*decision.DecisionCase, error) {
			return nil, assert.AnError
		},
	}

	svc := newTestDecisionService(caseSvc, nil, nil, nil)
	comparison, err := svc.Compare(context.Background(), "dc_unknown")

	assert.Error(t, err)
	assert.Nil(t, comparison)
}

func TestDecisionService_Compare_BuildContextError(t *testing.T) {
	caseSvc := &mockCaseService{
		getCaseFn: func(_ context.Context, caseID string) (*decision.DecisionCase, error) {
			return testDecisionCase(), nil
		},
	}
	ctxBuilder := &mockContextBuilder{
		buildDecisionContextFn: func(_ context.Context, caseID string) (*decision.DecisionContext, error) {
			return nil, assert.AnError
		},
	}

	svc := newTestDecisionService(caseSvc, ctxBuilder, nil, nil)
	comparison, err := svc.Compare(context.Background(), "dc_1000000_testabc")

	assert.Error(t, err)
	assert.Nil(t, comparison)
}

// ──── mockDecisionRepository for Replay tests ──────────────────────────────

type mockDecisionRepository struct {
	getLLMDecisionByCaseIDFn func(ctx context.Context, caseID string) (*eval.ReplayData, error)
}

func (m *mockDecisionRepository) GetLLMDecisionByCaseID(ctx context.Context, caseID string) (*eval.ReplayData, error) {
	return m.getLLMDecisionByCaseIDFn(ctx, caseID)
}

// ──── Replay tests ─────────────────────────────────────────────────────────

func TestDecisionService_Replay_NotConfigured(t *testing.T) {
	svc := newTestDecisionService(nil, nil, nil, nil)
	result, err := svc.Replay(context.Background(), "dc_1000000_testabc", true)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "replay service not configured")
}

func TestDecisionService_Replay_Success(t *testing.T) {
	repo := &mockDecisionRepository{
		getLLMDecisionByCaseIDFn: func(_ context.Context, caseID string) (*eval.ReplayData, error) {
			return &eval.ReplayData{
				CaseID:             caseID,
				OriginalDecisionID: "dec-1",
				InputContext:       []byte(`{"case_id":"` + caseID + `"}`),
				OriginalOutput:     testDecisionOutput(),
				Provider:           "openai",
				Model:              "gpt-4",
				PromptVersion:      "v1",
				ContextHash:        "hash-1",
			}, nil
		},
	}
	provider := &mockLlmProvider{
		generateDecisionFn: func(_ context.Context, _ llm.LLMSafeContext) (*llm.DecisionOutput, error) {
			return testDecisionOutput(), nil
		},
	}
	replaySvc := eval.NewReplayService(repo, provider, &llm.NoOpAuditLogger{})
	svc := newTestDecisionService(nil, nil, nil, nil)
	svc.WithReplayService(replaySvc)

	result, err := svc.Replay(context.Background(), "dc_1000000_testabc", true)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.DryRun)
	assert.NotNil(t, result.OriginalDecision)
}

// ──── DecideLLM error path ────────────────────────────────────────────────

func TestDecisionService_DecideLLM_Error(t *testing.T) {
	caseSvc := &mockCaseService{
		getCaseFn: func(_ context.Context, caseID string) (*decision.DecisionCase, error) {
			return nil, assert.AnError
		},
	}
	svc := newTestDecisionService(caseSvc, nil, nil, nil)
	ctx, output, proposals, err := svc.DecideLLM(context.Background(), "dc_unknown")

	assert.Error(t, err)
	assert.Nil(t, ctx)
	assert.Nil(t, output)
	assert.Nil(t, proposals)
}

// ──── ListLLMDecisions panic with nil pool ────────────────────────────────

func TestDecisionService_ListLLMDecisions_NilPoolPanics(t *testing.T) {
	svc := NewDecisionService(nil, nil, nil, nil, nil)
	assert.Panics(t, func() {
		svc.ListLLMDecisions(context.Background(), "dc_1000000_testabc")
	})
}

// ──── ListEvals panic with nil pool ───────────────────────────────────────

func TestDecisionService_ListEvals_NilPoolPanics(t *testing.T) {
	svc := NewDecisionService(nil, nil, nil, nil, nil)
	assert.Panics(t, func() {
		svc.ListEvals(context.Background(), "dc_1000000_testabc")
	})
}
