package action

import (
	"context"
	"encoding/json"
	"testing"

	"baxi/internal/llm"
	"baxi/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──── actionChannel ─────────────────────────────────────────────────────

func TestActionChannel_FeishuActions(t *testing.T) {
	tests := []struct {
		actionType string
		expected   string
	}{
		{"export_report", "feishu"},
		{"notify_owner", "feishu"},
		{"create_outbox_message", "feishu"},
	}
	for _, tt := range tests {
		t.Run(tt.actionType, func(t *testing.T) {
			assert.Equal(t, tt.expected, actionChannel(tt.actionType))
		})
	}
}

func TestActionChannel_GithubActions(t *testing.T) {
	assert.Equal(t, "github", actionChannel("create_followup_task"))
}

func TestActionChannel_Unknown(t *testing.T) {
	assert.Equal(t, "unknown", actionChannel("unknown_action"))
	assert.Equal(t, "unknown", actionChannel(""))
	assert.Equal(t, "unknown", actionChannel("hack_database"))
}

// ──── generateTraceID ───────────────────────────────────────────────────

func TestGenerateTraceID_Format(t *testing.T) {
	id := generateTraceID()
	assert.Contains(t, id, "trace-")
	assert.True(t, len(id) > 6)
}

// ──── NoOpExecutor ──────────────────────────────────────────────────────

func TestNoOpExecutor_Execute(t *testing.T) {
	executor := NewNoOpExecutor()
	proposal := ActionProposal{
		ProposalID: "prop-1",
		ActionType: "notify_owner",
		CaseID:     "case-1",
	}

	result, err := executor.Execute(context.Background(), proposal, true)
	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.True(t, result.DryRun)
	assert.Equal(t, "notify_owner", result.DispatchPayload["action_type"])
	assert.Equal(t, "prop-1", result.DispatchPayload["proposal_id"])
	assert.Equal(t, "case-1", result.DispatchPayload["case_id"])
	assert.Equal(t, true, result.DispatchPayload["dry_run"])
}

func TestNoOpExecutor_Execute_NotDryRun(t *testing.T) {
	executor := NewNoOpExecutor()
	proposal := ActionProposal{ProposalID: "prop-2"}

	result, err := executor.Execute(context.Background(), proposal, false)
	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.False(t, result.DryRun)
}

// ──── ApplyService: proposed + low risk path ────────────────────────────

func TestApplyService_ProposedLowRisk_NotApproved(t *testing.T) {
	reg := setupTestRegistry(t)
	proposal := newTestProposal("proposed", "notify_owner")
	proposal.RiskLevel = "low"

	// Action config has requires_approval: false for notify_owner in test YAML
	loader := &mockProposalLoader{proposal: proposal}
	svc := NewApplyService(reg, nil, loader, nil, nil, nil)

	ctx := context.Background()
	_, err := svc.ExecuteProposal(ctx, nil, "prop-test-001", "actor-1")

	require.Error(t, err)
	assert.True(t, err == ErrNotApproved)
}

func TestApplyService_ProposedHighRisk_NotApproved(t *testing.T) {
	reg := setupTestRegistry(t)
	proposal := newTestProposal("proposed", "notify_owner")
	proposal.RiskLevel = "high"

	loader := &mockProposalLoader{proposal: proposal}
	svc := NewApplyService(reg, nil, loader, nil, nil, nil)

	ctx := context.Background()
	_, err := svc.ExecuteProposal(ctx, nil, "prop-test-001", "actor-1")

	require.Error(t, err)
	assert.True(t, err == ErrNotApproved)
}

func TestApplyService_ProposedLowRisk_RequiresApproval(t *testing.T) {
	reg := setupTestRegistry(t)
	proposal := newTestProposal("proposed", "create_followup_task")
	proposal.RiskLevel = "low"

	// create_followup_task has requires_approval: true in test YAML
	loader := &mockProposalLoader{proposal: proposal}
	svc := NewApplyService(reg, nil, loader, nil, nil, nil)

	ctx := context.Background()
	_, err := svc.ExecuteProposal(ctx, nil, "prop-test-001", "actor-1")

	require.Error(t, err)
	assert.True(t, err == ErrNotApproved)
}

// ──── ApplyService: lineage verification ────────────────────────────────

type mockLineageVerifier struct {
	ok   bool
	err  error
}

func (m *mockLineageVerifier) HasCompleteLineage(ctx context.Context, caseID string) (bool, error) {
	return m.ok, m.err
}

func TestApplyService_LineageIncomplete(t *testing.T) {
	reg := setupTestRegistry(t)
	loader := &mockProposalLoader{proposal: newTestProposal("approved", "notify_owner")}
	verifier := &mockLineageVerifier{ok: false, err: nil}
	svc := NewApplyService(reg, nil, loader, verifier, nil, nil)

	ctx := context.Background()
	_, err := svc.ExecuteProposal(ctx, nil, "prop-test-001", "actor-1", WithDryRun(false))

	require.Error(t, err)
	assert.True(t, err == ErrLineageIncomplete)
}

func TestApplyService_LineageVerifierError(t *testing.T) {
	reg := setupTestRegistry(t)
	loader := &mockProposalLoader{proposal: newTestProposal("approved", "notify_owner")}
	verifier := &mockLineageVerifier{ok: false, err: assert.AnError}
	svc := NewApplyService(reg, nil, loader, verifier, nil, nil)

	ctx := context.Background()
	_, err := svc.ExecuteProposal(ctx, nil, "prop-test-001", "actor-1", WithDryRun(false))

	require.Error(t, err)
	assert.True(t, err == ErrLineageIncomplete)
}

func TestApplyService_LineageComplete(t *testing.T) {
	reg := setupTestRegistry(t)
	loader := &mockProposalLoader{proposal: newTestProposal("approved", "notify_owner")}
	verifier := &mockLineageVerifier{ok: true, err: nil}
	exec := &mockExecutor{result: ExecutionResult{Success: true, DryRun: false}}
	svc := NewApplyService(reg, map[string]ActionExecutor{"feishu": exec}, loader, verifier, nil, nil)

	// With dry-run=true, lineage verification is skipped and NoOpExecutor is used
	ctx := context.Background()
	result, err := svc.ExecuteProposal(ctx, nil, "prop-test-001", "actor-1")

	require.NoError(t, err)
	assert.True(t, result.Success)
}

// ──── ApplyService: nil executors defaults to empty map ─────────────────

func TestApplyService_NilExecutors(t *testing.T) {
	reg := setupTestRegistry(t)
	loader := &mockProposalLoader{proposal: newTestProposal("approved", "notify_owner")}
	svc := NewApplyService(reg, nil, loader, nil, nil, nil)

	// nil executors -> dry-run should still work
	ctx := context.Background()
	result, err := svc.ExecuteProposal(ctx, nil, "prop-test-001", "actor-1")
	require.NoError(t, err)
	assert.True(t, result.Success)
}

// ──── ExecuteOption: WithDryRun ─────────────────────────────────────────

func TestWithDryRun_True(t *testing.T) {
	opts := &ExecuteOptions{DryRun: false}
	WithDryRun(true)(opts)
	assert.True(t, opts.DryRun)
}

func TestWithDryRun_False(t *testing.T) {
	opts := &ExecuteOptions{DryRun: true}
	WithDryRun(false)(opts)
	assert.False(t, opts.DryRun)
}

// ──── Registry: GetLLMVisibleActions ────────────────────────────────────

func TestGetLLMVisibleActions_Filtered(t *testing.T) {
	dir := t.TempDir()
	path := writeTestRegistryYAML(t, dir, `
actions:
  notify_owner:
    description: "Notify"
    risk_level: low
    llm_visible: true
    llm_description: "Notify the owner"
    adapter: feishu
  export_report:
    description: "Export"
    risk_level: low
    llm_visible: false
  create_followup_task:
    description: "Task"
    risk_level: medium
    llm_visible: true
    llm_description: "Create a task"
`)
	reg, err := NewActionRegistry(path)
	require.NoError(t, err)

	contracts := reg.GetLLMVisibleActions()
	require.Len(t, contracts, 2)

	types := make(map[string]bool)
	for _, c := range contracts {
		types[c.ActionType] = true
	}
	assert.True(t, types["notify_owner"])
	assert.True(t, types["create_followup_task"])
	assert.False(t, types["export_report"])
}

func TestGetLLMVisibleActions_EmptyRegistry(t *testing.T) {
	reg := NewEmptyRegistry()
	contracts := reg.GetLLMVisibleActions()
	assert.Empty(t, contracts)
}

// ──── Registry: ListActionTypes ─────────────────────────────────────────

func TestListActionTypes_Sorted(t *testing.T) {
	dir := t.TempDir()
	path := writeTestRegistryYAML(t, dir, `
actions:
  notify_owner:
    description: "Notify"
  export_report:
    description: "Export"
  create_followup_task:
    description: "Task"
  create_outbox_message:
    description: "Message"
`)
	reg, err := NewActionRegistry(path)
	require.NoError(t, err)

	types := reg.ListActionTypes()
	require.Len(t, types, 4)
	assert.Equal(t, []string{
		"create_followup_task",
		"create_outbox_message",
		"export_report",
		"notify_owner",
	}, types)
}

// ──── Registry: Enabled field handling ──────────────────────────────────

func TestRegistry_EnabledFalse(t *testing.T) {
	dir := t.TempDir()
	path := writeTestRegistryYAML(t, dir, `
actions:
  notify_owner:
    enabled: false
    description: "Notify"
`)
	reg, err := NewActionRegistry(path)
	require.NoError(t, err)

	assert.False(t, reg.IsAllowed("notify_owner"))
	assert.Empty(t, reg.AllowedActions())
}

func TestRegistry_EnabledTrue(t *testing.T) {
	dir := t.TempDir()
	path := writeTestRegistryYAML(t, dir, `
actions:
  notify_owner:
    enabled: true
    description: "Notify"
`)
	reg, err := NewActionRegistry(path)
	require.NoError(t, err)

	assert.True(t, reg.IsAllowed("notify_owner"))
}

func TestRegistry_EnabledNil_DefaultsTrue(t *testing.T) {
	dir := t.TempDir()
	path := writeTestRegistryYAML(t, dir, `
actions:
  notify_owner:
    description: "Notify"
`)
	reg, err := NewActionRegistry(path)
	require.NoError(t, err)

	assert.True(t, reg.IsAllowed("notify_owner"))
}

// ──── Registry: Non-canonical actions filtered ──────────────────────────

func TestRegistry_NonCanonicalActionsFiltered(t *testing.T) {
	dir := t.TempDir()
	path := writeTestRegistryYAML(t, dir, `
actions:
  hack_database:
    description: "Hack"
    risk_level: critical
`)
	reg, err := NewActionRegistry(path)
	require.NoError(t, err)

	// Non-canonical action should be filtered out
	assert.False(t, reg.IsAllowed("hack_database"))
	assert.Empty(t, reg.ListActionTypes())
}

// ──── Registry: Nil actions block ───────────────────────────────────────

func TestRegistry_NilActionsBlock(t *testing.T) {
	dir := t.TempDir()
	path := writeTestRegistryYAML(t, dir, "actions: ~\n")
	reg, err := NewActionRegistry(path)
	require.NoError(t, err)

	// Nil actions block should result in no whitelisted actions
	assert.Empty(t, reg.AllowedActions())
}

// ──── OutcomeService constructor ────────────────────────────────────────

func TestProposalService_ListProposals_Error(t *testing.T) {
	repo := &mockProposalRepo{
		listProposalsByCaseFn: func(ctx context.Context, pool *pgxpool.Pool, caseID string) ([]repository.ActionProposalRow, error) {
			return nil, assert.AnError
		},
	}

	svc := NewProposalService(repo, &mockCaseUpdater{}, nil, nil)
	_, err := svc.ListProposals(context.Background(), "case-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "list proposals")
}

func TestProposalService_GenerateProposals_CaseUpdateError(t *testing.T) {
	repo := &mockProposalRepo{
		createProposalFn: func(ctx context.Context, pool *pgxpool.Pool, row *repository.ActionProposalRow) error {
			return nil
		},
	}

	updater := &mockCaseUpdater{
		updateCaseStatusFn: func(ctx context.Context, pool *pgxpool.Pool, caseID string, status string, cj *json.RawMessage, ch *string, gs *json.RawMessage) error {
			return assert.AnError
		},
	}

	svc := NewProposalService(repo, updater, nil, nil)
	decision := &llm.DecisionOutput{
		DecisionType: "intervention",
		Severity:     "high",
		Summary:      "Test",
		RecommendedActions: []llm.RecommendedAction{
			{ActionType: "notify_owner", Priority: "high"},
		},
	}

	_, err := svc.GenerateProposals(context.Background(), "case-1", "dec-1", decision, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update case status")
}

func TestProposalService_GenerateProposals_RepoCreateError(t *testing.T) {
	repo := &mockProposalRepo{
		createProposalFn: func(ctx context.Context, pool *pgxpool.Pool, row *repository.ActionProposalRow) error {
			return assert.AnError
		},
	}

	updater := &mockCaseUpdater{
		updateCaseStatusFn: func(ctx context.Context, pool *pgxpool.Pool, caseID string, status string, cj *json.RawMessage, ch *string, gs *json.RawMessage) error {
			return nil
		},
	}

	svc := NewProposalService(repo, updater, nil, nil)
	decision := &llm.DecisionOutput{
		DecisionType: "intervention",
		Severity:     "high",
		Summary:      "Test",
		RecommendedActions: []llm.RecommendedAction{
			{ActionType: "notify_owner", Priority: "high"},
		},
	}

	_, err := svc.GenerateProposals(context.Background(), "case-1", "dec-1", decision, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create proposal")
}

// ──── ProposalService: registry with validation skips ───────────────────

func TestGenerateProposals_RegistrySkipsInvalidPayload(t *testing.T) {
	reg := &ActionRegistry{
		whitelist: map[string]bool{"notify_owner": true},
		config: &ActionRegistryConfig{
			Actions: map[string]ActionConfig{
				"notify_owner": {
					PayloadSchemaRaw: map[string]interface{}{
						"required": []interface{}{"channel"},
					},
				},
			},
		},
	}

	var savedProposals []repository.ActionProposalRow
	repo := &mockProposalRepo{
		createProposalFn: func(ctx context.Context, pool *pgxpool.Pool, row *repository.ActionProposalRow) error {
			savedProposals = append(savedProposals, *row)
			return nil
		},
	}
	updater := &mockCaseUpdater{
		updateCaseStatusFn: func(ctx context.Context, pool *pgxpool.Pool, cid string, status string, cj *json.RawMessage, ch *string, gs *json.RawMessage) error {
			return nil
		},
	}

	svc := NewProposalService(repo, updater, reg, nil)
	decision := &llm.DecisionOutput{
		DecisionType: "intervention",
		Severity:     "high",
		Summary:      "Test",
		RecommendedActions: []llm.RecommendedAction{
			{
				ActionType: "notify_owner",
				Priority:   "high",
				Payload:    map[string]interface{}{}, // missing required "channel"
			},
			{
				ActionType: "notify_owner",
				Priority:   "high",
				Payload:    map[string]interface{}{"channel": "email"}, // valid
			},
		},
	}

	proposals, err := svc.GenerateProposals(context.Background(), "case-1", "dec-1", decision, "")
	assert.NoError(t, err)
	assert.Len(t, proposals, 1) // only the valid one
	assert.Len(t, savedProposals, 1)
}

func TestGenerateProposals_RegistrySchemaVersion(t *testing.T) {
	reg := &ActionRegistry{
		whitelist: map[string]bool{"notify_owner": true},
		config: &ActionRegistryConfig{
			Actions: map[string]ActionConfig{
				"notify_owner": {
					Version: "v2",
				},
			},
		},
	}

	var savedRow repository.ActionProposalRow
	repo := &mockProposalRepo{
		createProposalFn: func(ctx context.Context, pool *pgxpool.Pool, row *repository.ActionProposalRow) error {
			savedRow = *row
			return nil
		},
	}
	updater := &mockCaseUpdater{
		updateCaseStatusFn: func(ctx context.Context, pool *pgxpool.Pool, cid string, status string, cj *json.RawMessage, ch *string, gs *json.RawMessage) error {
			return nil
		},
	}

	svc := NewProposalService(repo, updater, reg, nil)
	decision := &llm.DecisionOutput{
		DecisionType: "intervention",
		Severity:     "high",
		Summary:      "Test",
		RecommendedActions: []llm.RecommendedAction{
			{ActionType: "notify_owner", Priority: "high"},
		},
	}

	_, err := svc.GenerateProposals(context.Background(), "case-1", "dec-1", decision, "")
	assert.NoError(t, err)
	require.NotNil(t, savedRow.ActionSchemaVersion)
	assert.Equal(t, "v2", *savedRow.ActionSchemaVersion)
}

// ──── ProposalService: payload nil/marshal error ────────────────────────

func TestGenerateProposals_NilPayload(t *testing.T) {
	repo := &mockProposalRepo{
		createProposalFn: func(ctx context.Context, pool *pgxpool.Pool, row *repository.ActionProposalRow) error {
			return nil
		},
	}
	updater := &mockCaseUpdater{
		updateCaseStatusFn: func(ctx context.Context, pool *pgxpool.Pool, cid string, status string, cj *json.RawMessage, ch *string, gs *json.RawMessage) error {
			return nil
		},
	}

	svc := NewProposalService(repo, updater, nil, nil)
	decision := &llm.DecisionOutput{
		DecisionType: "intervention",
		Severity:     "high",
		Summary:      "Test",
		RecommendedActions: []llm.RecommendedAction{
			{ActionType: "notify_owner", Priority: "high", Payload: nil},
		},
	}

	proposals, err := svc.GenerateProposals(context.Background(), "case-1", "dec-1", decision, "")
	assert.NoError(t, err)
	require.Len(t, proposals, 1)
	assert.Nil(t, proposals[0].Payload)
}


