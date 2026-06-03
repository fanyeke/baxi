package action

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"baxi/internal/llm"
	decisionRepo "baxi/internal/repository/decision"
	decisionRepo "baxi/internal/repository/decision"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mocks ---

type mockProposalRepo struct {
	createProposalFn      func(ctx context.Context, pool *pgxpool.Pool, row *decisionRepo.ActionProposalRow) error
	listProposalsByCaseFn func(ctx context.Context, caseID string) ([]decisionRepo.ActionProposalRow, error)
}

func (m *mockProposalRepo) CreateProposal(ctx context.Context, row *decisionRepo.ActionProposalRow) error {
	return m.createProposalFn(ctx, pool, row)
}

func (m *mockProposalRepo) ListProposalsByCase(ctx context.Context, caseID string) ([]decisionRepo.ActionProposalRow, error) {
	return m.listProposalsByCaseFn(ctx, pool, caseID)
}

type mockCaseUpdater struct {
	updateCaseStatusFn func(ctx context.Context, pool *pgxpool.Pool, caseID string, status string, contextJSON *json.RawMessage, contextHash *string, governanceSnapshot *json.RawMessage) error
}

func (m *mockCaseUpdater) UpdateCaseStatus(ctx context.Context, caseID string, status string, contextJSON *json.RawMessage, contextHash *string, governanceSnapshot *json.RawMessage) error {
	return m.updateCaseStatusFn(ctx, pool, caseID, status, contextJSON, contextHash, governanceSnapshot)
}

// --- Compile-time interface checks ---

var _ ProposalRepository = (*mockProposalRepo)(nil)
var _ CaseStatusUpdater = (*mockCaseUpdater)(nil)

// --- Test: GenerateProposals ---

func TestGenerateProposals_CreatesProposalsFromDecision(t *testing.T) {
	caseID := "dc_test_123"
	decisionID := "de_test_456"

	var savedProposals []decisionRepo.ActionProposalRow

	repo := &mockProposalRepo{
		createProposalFn: func(ctx context.Context, row *decisionRepo.ActionProposalRow) error {
			savedProposals = append(savedProposals, *row)
			return nil
		},
	}

	updater := &mockCaseUpdater{
		updateCaseStatusFn: func(ctx context.Context, pool *pgxpool.Pool, cid string, status string, cj *json.RawMessage, ch *string, gs *json.RawMessage) error {
			assert.Equal(t, caseID, cid)
			assert.Equal(t, "proposal_generated", status)
			assert.Nil(t, cj)
			assert.Nil(t, ch)
			assert.Nil(t, gs)
			return nil
		},
	}

	svc := NewProposalService(repo, updater, nil)

	decision := &llm.DecisionOutput{
		DecisionType: "intervention",
		Severity:     "high",
		Summary:      "Sales drop detected for seller-42 in electronics category",
		Rationale:    []string{"Sales dropped 23% MoM", "Rank dropped from top 10 to top 50", "Competitor pricing pressure detected"},
		RecommendedActions: []llm.RecommendedAction{
			{
				ActionType: "notify_owner",
				Priority:   "high",
				OwnerRole:  "category_manager",
				Payload:    map[string]interface{}{"channel": "email", "template": "seller_alert"},
			},
			{
				ActionType: "create_followup_task",
				Priority:   "medium",
				OwnerRole:  "analyst",
				Payload:    map[string]interface{}{"due_in_days": 7, "task_type": "deep_dive"},
			},
		},
		Confidence:          0.85,
		RequiresHumanReview: true,
	}

	proposals, err := svc.GenerateProposals(context.Background(), caseID, decisionID, decision, "")

	assert.NoError(t, err)
	assert.Len(t, proposals, 2)
	assert.Len(t, savedProposals, 2)

	// First proposal: notify_owner
	assert.Equal(t, "notify_owner", proposals[0].ActionType)
	assert.Equal(t, caseID, proposals[0].CaseID)
	assert.Equal(t, decisionID, proposals[0].DecisionID)
	assert.Contains(t, proposals[0].Title, "notify_owner")
	assert.Contains(t, proposals[0].Title, "Sales drop detected")
	assert.NotEmpty(t, proposals[0].Description)
	assert.Equal(t, "high", proposals[0].RiskLevel)
	assert.True(t, proposals[0].RequiresHumanReview)
	assert.Equal(t, "proposed", proposals[0].ApplyStatus)
	assert.NotNil(t, proposals[0].Payload)
	assert.Equal(t, "email", proposals[0].Payload["channel"])

	// Second proposal: create_followup_task
	assert.Equal(t, "create_followup_task", proposals[1].ActionType)
	assert.Equal(t, caseID, proposals[1].CaseID)
	assert.Equal(t, decisionID, proposals[1].DecisionID)
	assert.Contains(t, proposals[1].Title, "create_followup_task")
	assert.True(t, proposals[1].RequiresHumanReview)
	assert.Equal(t, "proposed", proposals[1].ApplyStatus)
	assert.Equal(t, "high", proposals[1].RiskLevel)
	assert.Equal(t, float64(7), proposals[1].Payload["due_in_days"])

	// Verify saved proposals match
	assert.Equal(t, proposals[0].ProposalID, savedProposals[0].ProposalID)
	assert.Equal(t, proposals[1].ProposalID, savedProposals[1].ProposalID)
	assert.True(t, savedProposals[0].RequiresHumanReview)
	assert.True(t, savedProposals[1].RequiresHumanReview)
	assert.Equal(t, "proposed", savedProposals[0].ApplyStatus)
	assert.Equal(t, "proposed", savedProposals[1].ApplyStatus)
	assert.True(t, len(savedProposals[0].ProposalID) > 0)
	assert.True(t, len(savedProposals[1].ProposalID) > 0)
	assert.Contains(t, savedProposals[0].ProposalID, "ap_")
	assert.Contains(t, savedProposals[1].ProposalID, "ap_")
	assert.NotEqual(t, savedProposals[0].ProposalID, savedProposals[1].ProposalID)
}

func TestGenerateProposals_AllProposalsRequireHumanReview(t *testing.T) {
	repo := &mockProposalRepo{
		createProposalFn: func(ctx context.Context, row *decisionRepo.ActionProposalRow) error {
			assert.True(t, row.RequiresHumanReview, "all proposals must require human review")
			return nil
		},
	}

	updater := &mockCaseUpdater{
		updateCaseStatusFn: func(ctx context.Context, pool *pgxpool.Pool, cid string, status string, cj *json.RawMessage, ch *string, gs *json.RawMessage) error {
			return nil
		},
	}

	svc := NewProposalService(repo, updater, nil)

	decision := &llm.DecisionOutput{
		DecisionType: "optimize",
		Severity:     "low",
		Summary:      "Minor optimization opportunity",
		RecommendedActions: []llm.RecommendedAction{
			{ActionType: "export_report", Priority: "low", OwnerRole: "admin"},
			{ActionType: "notify_owner", Priority: "low", OwnerRole: "admin"},
			{ActionType: "create_followup_task", Priority: "low", OwnerRole: "admin"},
		},
		Confidence: 0.95,
	}

	proposals, err := svc.GenerateProposals(context.Background(), "case-1", "dec-1", decision, "")

	assert.NoError(t, err)
	assert.Len(t, proposals, 3)
	for _, p := range proposals {
		assert.True(t, p.RequiresHumanReview, "proposal %s must require human review", p.ProposalID)
	}
}

func TestGenerateProposals_AllProposalsHaveApplyStatusProposed(t *testing.T) {
	repo := &mockProposalRepo{
		createProposalFn: func(ctx context.Context, row *decisionRepo.ActionProposalRow) error {
			assert.Equal(t, "proposed", row.ApplyStatus)
			return nil
		},
	}

	updater := &mockCaseUpdater{
		updateCaseStatusFn: func(ctx context.Context, pool *pgxpool.Pool, cid string, status string, cj *json.RawMessage, ch *string, gs *json.RawMessage) error {
			return nil
		},
	}

	svc := NewProposalService(repo, updater, nil)

	decision := &llm.DecisionOutput{
		DecisionType: "investigate",
		Severity:     "medium",
		Summary:      "Investigate anomaly",
		RecommendedActions: []llm.RecommendedAction{
			{ActionType: "notify_owner", Priority: "high", OwnerRole: "admin"},
		},
		Confidence: 0.7,
	}

	proposals, err := svc.GenerateProposals(context.Background(), "case-2", "dec-2", decision, "")

	assert.NoError(t, err)
	assert.Len(t, proposals, 1)
	assert.Equal(t, "proposed", proposals[0].ApplyStatus)
}

func TestGenerateProposals_UpdatesCaseStatus(t *testing.T) {
	caseID := "case-status-update"
	decisionID := "dec-status-update"

	repo := &mockProposalRepo{
		createProposalFn: func(ctx context.Context, row *decisionRepo.ActionProposalRow) error {
			return nil
		},
	}

	statusUpdated := false
	updater := &mockCaseUpdater{
		updateCaseStatusFn: func(ctx context.Context, pool *pgxpool.Pool, cid string, status string, cj *json.RawMessage, ch *string, gs *json.RawMessage) error {
			statusUpdated = true
			assert.Equal(t, caseID, cid)
			assert.Equal(t, "proposal_generated", status)
			return nil
		},
	}

	svc := NewProposalService(repo, updater, nil)

	decision := &llm.DecisionOutput{
		DecisionType: "monitor_only",
		Severity:     "low",
		Summary:      "Routine check",
		RecommendedActions: []llm.RecommendedAction{
			{ActionType: "export_report", Priority: "low", OwnerRole: "admin"},
		},
		Confidence: 0.9,
	}

	_, err := svc.GenerateProposals(context.Background(), caseID, decisionID, decision, "")

	assert.NoError(t, err)
	assert.True(t, statusUpdated, "case status must be updated to proposal_generated")
}

func TestGenerateProposals_EmptyDecisionReturnsEmptyList(t *testing.T) {
	repo := &mockProposalRepo{
		createProposalFn: func(ctx context.Context, row *decisionRepo.ActionProposalRow) error {
			t.Fatal("CreateProposal should not be called for empty decision")
			return nil
		},
	}

	updater := &mockCaseUpdater{
		updateCaseStatusFn: func(ctx context.Context, pool *pgxpool.Pool, cid string, status string, cj *json.RawMessage, ch *string, gs *json.RawMessage) error {
			return nil
		},
	}

	svc := NewProposalService(repo, updater, nil)

	decision := &llm.DecisionOutput{
		DecisionType:       "monitor_only",
		Severity:           "low",
		Summary:            "No action needed",
		RecommendedActions: []llm.RecommendedAction{},
		Confidence:         1.0,
	}

	proposals, err := svc.GenerateProposals(context.Background(), "case-empty", "dec-empty", decision, "")

	assert.NoError(t, err)
	assert.NotNil(t, proposals)
	assert.Empty(t, proposals)
}

// --- Test: ListProposals ---

func TestListProposals_ReturnsProposalsForCase(t *testing.T) {
	caseID := "dc_list_test"
	now := time.Now()

	decisionID := "de_list_test"
	description := "Investigate sales drop"
	riskLevel := "high"
	payloadRaw := json.RawMessage(`{"channel":"email"}`)

	rows := []decisionRepo.ActionProposalRow{
		{
			ProposalID:          "ap_001",
			CaseID:              caseID,
			DecisionID:          &decisionID,
			ActionType:          "notify_owner",
			Payload:             &payloadRaw,
			ApplyStatus:         "proposed",
			CreatedAt:           now,
			Title:               "notify_owner: Sales drop",
			Description:         &description,
			RiskLevel:           &riskLevel,
			RequiresHumanReview: true,
		},
		{
			ProposalID:          "ap_002",
			CaseID:              caseID,
			DecisionID:          &decisionID,
			ActionType:          "create_followup_task",
			Payload:             nil,
			ApplyStatus:         "proposed",
			CreatedAt:           now.Add(1 * time.Second),
			Title:               "create_followup_task: Sales drop",
			Description:         &description,
			RiskLevel:           &riskLevel,
			RequiresHumanReview: true,
		},
	}

	repo := &mockProposalRepo{
		listProposalsByCaseFn: func(ctx context.Context, pool *pgxpool.Pool, cid string) ([]decisionRepo.ActionProposalRow, error) {
			assert.Equal(t, caseID, cid)
			return rows, nil
		},
	}

	svc := NewProposalService(repo, &mockCaseUpdater{}, nil, nil)
	proposals, err := svc.ListProposals(context.Background(), caseID)

	assert.NoError(t, err)
	assert.Len(t, proposals, 2)

	assert.Equal(t, "ap_001", proposals[0].ProposalID)
	assert.Equal(t, caseID, proposals[0].CaseID)
	assert.Equal(t, decisionID, proposals[0].DecisionID)
	assert.Equal(t, "notify_owner", proposals[0].ActionType)
	assert.Equal(t, "notify_owner: Sales drop", proposals[0].Title)
	assert.Equal(t, "Investigate sales drop", proposals[0].Description)
	assert.Equal(t, "high", proposals[0].RiskLevel)
	assert.True(t, proposals[0].RequiresHumanReview)
	assert.Equal(t, "proposed", proposals[0].ApplyStatus)
	assert.Equal(t, "email", proposals[0].Payload["channel"])

	assert.Equal(t, "ap_002", proposals[1].ProposalID)
	assert.Equal(t, "create_followup_task", proposals[1].ActionType)
	assert.True(t, proposals[1].RequiresHumanReview)
	assert.Equal(t, "proposed", proposals[1].ApplyStatus)
	assert.Nil(t, proposals[1].Payload)
}

func TestListProposals_EmptyCaseReturnsEmptyList(t *testing.T) {
	repo := &mockProposalRepo{
		listProposalsByCaseFn: func(ctx context.Context, caseID string) ([]decisionRepo.ActionProposalRow, error) {
			return []decisionRepo.ActionProposalRow{}, nil
		},
	}

	svc := NewProposalService(repo, &mockCaseUpdater{}, nil, nil)
	proposals, err := svc.ListProposals(context.Background(), "empty-case")

	assert.NoError(t, err)
	assert.NotNil(t, proposals)
	assert.Empty(t, proposals)
}

// --- Risk level mapping ---

func TestMapRiskLevel(t *testing.T) {
	tests := []struct {
		severity string
		want     string
	}{
		{"critical", "high"},
		{"high", "high"},
		{"medium", "medium"},
		{"low", "low"},
		{"unknown", "medium"},
		{"", "medium"},
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			got := mapRiskLevel(tt.severity)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- Title truncation ---

func TestGenerateProposals_TruncatesLongTitle(t *testing.T) {
	longSummary := ""
	for i := 0; i < 300; i++ {
		longSummary += "x"
	}

	repo := &mockProposalRepo{
		createProposalFn: func(ctx context.Context, row *decisionRepo.ActionProposalRow) error {
			assert.True(t, len(row.Title) <= 200, "title must be <= 200 chars, got %d", len(row.Title))
			return nil
		},
	}

	updater := &mockCaseUpdater{
		updateCaseStatusFn: func(ctx context.Context, pool *pgxpool.Pool, cid string, status string, cj *json.RawMessage, ch *string, gs *json.RawMessage) error {
			return nil
		},
	}

	svc := NewProposalService(repo, updater, nil)

	decision := &llm.DecisionOutput{
		DecisionType: "intervention",
		Severity:     "high",
		Summary:      longSummary,
		RecommendedActions: []llm.RecommendedAction{
			{ActionType: "notify_owner", Priority: "high", OwnerRole: "admin"},
		},
		Confidence: 0.8,
	}

	proposals, err := svc.GenerateProposals(context.Background(), "case-trunc", "dec-trunc", decision, "")

	assert.NoError(t, err)
	assert.Len(t, proposals, 1)
	assert.True(t, len(proposals[0].Title) <= 200)
}

// ──── Enhanced Write Path: trace fields ─────────────────────────────────────

// TestProposeAction_WithEvidenceRefs verifies that action proposals can be
// created with the full set of enhanced trace fields (evidence_refs,
// context_hash, recipe_id, decision_id) and that all fields are preserved
// through storage and retrieval.
//
// This tests the enhanced write path — the ability to attach provenance
// metadata (evidence links, context hash, recipe identity) to an action
// proposal so that downstream audit and lineage systems can trace the
// decision back to its source.
//
// TDD RED: This behavior is not yet wired through the service layer.
// The row-level mapping (rowToProposal) is tested here. The full service-
// level flow that accepts evidence_refs and recipe_id from the LLM
// decision output is not yet implemented.
func TestProposeAction_WithEvidenceRefs(t *testing.T) {
	decisionID := "dec-evidence-001"
	contextHash := "ctx-hash-abc123"
	recipeID := "recipe-sales-drop-456"
	evidenceRefsJSON := `["evt-001","evt-002","evt-003"]`

	// Build a row representing what a fully-populated proposal looks like
	// when the enhanced write path is wired through GenerateProposals.
	row := &decisionRepo.ActionProposalRow{
		ProposalID:          "ap-evidence-001",
		CaseID:              "case-evidence-001",
		DecisionID:          &decisionID,
		ActionType:          "notify_owner",
		ApplyStatus:         "proposed",
		CreatedAt:           time.Now(),
		Title:               "notify_owner: Evidence-based alert",
		Description:         strPtr("triggered by evidence refs"),
		RiskLevel:           strPtr("high"),
		RequiresHumanReview: true,
		ContextHash:         &contextHash,
		RecipeID:            &recipeID,
		EvidenceRefs:        &evidenceRefsJSON,
	}

	// Convert to domain model via existing mapping (same path used by ListProposals)
	proposal := rowToProposal(row)

	// Verify every trace field is present and correct
	assert.Equal(t, decisionID, proposal.DecisionID)
	assert.Equal(t, contextHash, proposal.ContextHash)
	assert.Equal(t, recipeID, proposal.RecipeID)
	assert.Equal(t, []string{"evt-001", "evt-002", "evt-003"}, proposal.EvidenceRefs)

	// Verify non-trace fields are intact
	assert.Equal(t, "ap-evidence-001", proposal.ProposalID)
	assert.Equal(t, "case-evidence-001", proposal.CaseID)
	assert.Equal(t, "notify_owner", proposal.ActionType)
	assert.Equal(t, "proposed", proposal.ApplyStatus)
	assert.Equal(t, "high", proposal.RiskLevel)
	assert.True(t, proposal.RequiresHumanReview)

	// Simulate storage → retrieval round-trip via mock repo
	var savedRow *decisionRepo.ActionProposalRow
	repo := &mockProposalRepo{
		createProposalFn: func(ctx context.Context, pool *pgxpool.Pool, r *decisionRepo.ActionProposalRow) error {
			savedRow = r
			return nil
		},
		listProposalsByCaseFn: func(ctx context.Context, caseID string) ([]decisionRepo.ActionProposalRow, error) {
			if savedRow == nil {
				return []decisionRepo.ActionProposalRow{}, nil
			}
			return []decisionRepo.ActionProposalRow{*savedRow}, nil
		},
	}

	updater := &mockCaseUpdater{
		updateCaseStatusFn: func(ctx context.Context, pool *pgxpool.Pool, cid string, status string, cj *json.RawMessage, ch *string, gs *json.RawMessage) error {
			return nil
		},
	}

	svc := NewProposalService(repo, updater, nil)
	listCtx := context.Background()

	// Use GenerateProposals to store a proposal (passing contextHash).
	// Note: evidence_refs and recipe_id are NOT yet passed through
	// the GenerateProposals signature — this is the gap the enhanced
	// write path must fill.
	decision := &llm.DecisionOutput{
		DecisionType: "intervention",
		Severity:     "high",
		Summary:      "Evidence-based alert triggered",
		Rationale:    []string{"Multiple evidence sources triggered"},
		RecommendedActions: []llm.RecommendedAction{
			{
				ActionType: "notify_owner",
				Priority:   "high",
				OwnerRole:  "category_manager",
			},
		},
		Confidence:          0.92,
		RequiresHumanReview: true,
		EvidenceRefs:        []string{"evt-001", "evt-002", "evt-003"},
		RecipeID:            "recipe-sales-drop-456",
	}

	proposals, err := svc.GenerateProposals(listCtx, "case-evidence-001", decisionID, decision, contextHash)
	require.NoError(t, err)
	require.Len(t, proposals, 1)

	// The contextHash is currently wired through and should be set
	assert.Equal(t, contextHash, proposals[0].ContextHash,
		"context_hash must flow through via GenerateProposals contextHash parameter")

	// Retrieve proposals and verify the stored row
	retrieved, err := svc.ListProposals(listCtx, "case-evidence-001")
	require.NoError(t, err)
	require.Len(t, retrieved, 1)

	// These fields ARE preserved through the existing rowToProposal mapping
	assert.Equal(t, decisionID, retrieved[0].DecisionID)
	assert.Equal(t, contextHash, retrieved[0].ContextHash)

	// TDD RED: evidence_refs and recipe_id are NOT yet passed through
	// GenerateProposals. The following assertions document the desired
	// behavior: once the enhanced write path is wired, these fields
	// should be populated from the decision output.
	//
	// These assertions FAIL because the service layer does not yet
	// accept evidence_refs or recipe_id parameters.
	assert.NotEmpty(t, retrieved[0].EvidenceRefs,
		"TDD RED: evidence_refs must be wired through GenerateProposals")
	assert.NotEmpty(t, retrieved[0].RecipeID,
		"TDD RED: recipe_id must be wired through GenerateProposals")
}

// TestProposeAction_WithoutTraceFields verifies backward compatibility:
// creating a proposal WITHOUT the new trace fields works correctly, the
// fields are null/empty, and no error occurs. This ensures existing
// callers that don't supply evidence_refs, context_hash, or recipe_id
// continue to work.
func TestProposeAction_WithoutTraceFields(t *testing.T) {
	decisionID := "dec-legacy-001"

	// Build a row WITHOUT any of the new trace fields (simulating a
	// legacy caller that has not been updated to the enhanced write path)
	row := &decisionRepo.ActionProposalRow{
		ProposalID:          "ap-legacy-001",
		CaseID:              "case-legacy-001",
		DecisionID:          &decisionID,
		ActionType:          "export_report",
		Payload:             nil,
		ApplyStatus:         "proposed",
		CreatedAt:           time.Now(),
		Title:               "export_report: Legacy monthly report",
		Description:         strPtr("Monthly performance report"),
		RiskLevel:           strPtr("low"),
		RequiresHumanReview: true,
		// Trace fields omitted (nil/zero):
		// ContextHash:  nil,
		// RecipeID:     nil,
		// EvidenceRefs: nil,
	}

	proposal := rowToProposal(row)

	// Backward compatibility: all trace fields must be nil/empty
	assert.Empty(t, proposal.ContextHash, "legacy proposals must have empty context_hash")
	assert.Empty(t, proposal.RecipeID, "legacy proposals must have empty recipe_id")
	assert.Nil(t, proposal.EvidenceRefs, "legacy proposals must have nil evidence_refs")

	// Core proposal fields must still be intact
	assert.Equal(t, "ap-legacy-001", proposal.ProposalID)
	assert.Equal(t, "case-legacy-001", proposal.CaseID)
	assert.Equal(t, "export_report", proposal.ActionType)
	assert.Equal(t, "proposed", proposal.ApplyStatus)
	assert.Equal(t, "low", proposal.RiskLevel)
	assert.True(t, proposal.RequiresHumanReview)

	// Simulate a GenerateProposals call without a context hash
	repo := &mockProposalRepo{
		createProposalFn: func(ctx context.Context, pool *pgxpool.Pool, r *decisionRepo.ActionProposalRow) error {
			assert.Nil(t, r.EvidenceRefs, "evidence_refs must be nil when not provided")
			assert.Nil(t, r.RecipeID, "recipe_id must be nil when not provided")
			// legacy callers pass empty context hash
			require.NotNil(t, r.ContextHash)
			assert.Empty(t, *r.ContextHash)
			return nil
		},
		listProposalsByCaseFn: func(ctx context.Context, caseID string) ([]decisionRepo.ActionProposalRow, error) {
			return []decisionRepo.ActionProposalRow{*row}, nil
		},
	}

	updater := &mockCaseUpdater{
		updateCaseStatusFn: func(ctx context.Context, pool *pgxpool.Pool, cid string, status string, cj *json.RawMessage, ch *string, gs *json.RawMessage) error {
			return nil
		},
	}

	svc := NewProposalService(repo, updater, nil)
	ctx := context.Background()

	// Legacy path: empty contextHash (but not nil — Go string is ""
	// which becomes &"" through the &contextHash pointer)
	decision := &llm.DecisionOutput{
		DecisionType: "monitor_only",
		Severity:     "low",
		Summary:      "Legacy monthly report",
		RecommendedActions: []llm.RecommendedAction{
			{
				ActionType: "export_report",
				Priority:   "low",
				OwnerRole:  "admin",
			},
		},
		Confidence: 0.95,
	}

	proposals, err := svc.GenerateProposals(ctx, "case-legacy-001", decisionID, decision, "")
	require.NoError(t, err)
	require.Len(t, proposals, 1)

	// Backward compatible: empty string for context_hash
	assert.Empty(t, proposals[0].ContextHash,
		"legacy GenerateProposals call with empty contextHash must remain empty in the domain model")
}

// ──── Helpers ────────────────────────────────────────────────────────────────

func strPtr(s string) *string { return &s }

// ──── LLMDecision storage and retrieval ──────────────────────────────────────

// TestLLMDecision_CreateAndRetrieve verifies that an LLMDecision can be
// created with full decision JSON and all fields are retrievable by ID.
// This exercises the repository layer for ai.llm_decision.
func TestLLMDecision_CreateAndRetrieve(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupApplyServiceTestDB(t)
	ctx := context.Background()

	// Seed a decision case first (required FK)
	insertTestCase(t, pool, "case-llm-001")

	// Build a full LLM decision row with rich output JSON
	outputJSON := json.RawMessage(`{
		"schema_version": "decision_output.v1",
		"decision_type": "intervention",
		"severity": "high",
		"summary": "Sales drop detected for seller-42",
		"rationale": ["Sales dropped 23% MoM", "Rank dropped from top 10 to top 50"],
		"confidence": 0.85,
		"requires_human_review": true,
		"recommended_actions": [
			{"action_type": "notify_owner", "priority": "high", "owner_role": "category_manager", "payload": {"channel": "email"}},
			{"action_type": "create_followup_task", "priority": "medium", "owner_role": "analyst", "payload": {"due_in_days": 7}}
		]
	}`)
	modelVersion := "gpt-4"
	promptHash := "sha256:abc123def456"
	confidence := 0.85
	status := "completed"
	recipeID := "recipe-sales-001"
	contextHash := "ctx-snapshot-abc"
	severity := "high"

	row := &decisionRepo.LLMDecisionRow{
		DecisionID:   "llm-dec-001",
		CaseID:       "case-llm-001",
		ModelVersion: &modelVersion,
		PromptHash:   &promptHash,
		OutputJSON:   &outputJSON,
		Confidence:   &confidence,
		Status:       &status,
		RecipeID:     &recipeID,
		ContextHash:  &contextHash,
		Severity:     &severity,
	}

	// Use the decision repository to persist the LLM decision
	decisionRepo := repository.NewDecisionRepository()
	decisionRepo.SetPool(pool)

	err := decisionRepo.CreateDecision(ctx, pool, row)
	require.NoError(t, err)

	// Retrieve all LLM decisions for the case via query
	var decisionID string
	var retrievedCaseID string
	var retrievedModelVersion *string
	var retrievedOutputJSON *json.RawMessage
	var retrievedConfidence *float64
	var retrievedStatus *string
	var retrievedRecipeID *string
	var retrievedContextHash *string
	var retrievedSeverity *string

	err = pool.QueryRow(ctx, `
		SELECT decision_id, case_id, model_version, output_json, confidence,
		       status, recipe_id, context_hash, severity
		FROM ai.llm_decision
		WHERE decision_id = $1
	`, "llm-dec-001").Scan(
		&decisionID, &retrievedCaseID, &retrievedModelVersion, &retrievedOutputJSON,
		&retrievedConfidence, &retrievedStatus, &retrievedRecipeID,
		&retrievedContextHash, &retrievedSeverity,
	)
	require.NoError(t, err)

	// Verify all fields match
	assert.Equal(t, "llm-dec-001", decisionID)
	assert.Equal(t, "case-llm-001", retrievedCaseID)
	require.NotNil(t, retrievedModelVersion)
	assert.Equal(t, "gpt-4", *retrievedModelVersion)
	require.NotNil(t, retrievedOutputJSON)

	// Verify the full decision JSON is retrievable
	var decodedOutput map[string]interface{}
	err = json.Unmarshal(*retrievedOutputJSON, &decodedOutput)
	require.NoError(t, err)
	assert.Equal(t, "intervention", decodedOutput["decision_type"])
	assert.Equal(t, "high", decodedOutput["severity"])
	assert.Equal(t, float64(0.85), decodedOutput["confidence"])
	assert.Equal(t, true, decodedOutput["requires_human_review"])

	// Verify trace fields
	require.NotNil(t, retrievedConfidence)
	assert.Equal(t, 0.85, *retrievedConfidence)
	require.NotNil(t, retrievedStatus)
	assert.Equal(t, "completed", *retrievedStatus)
	require.NotNil(t, retrievedRecipeID)
	assert.Equal(t, "recipe-sales-001", *retrievedRecipeID)
	require.NotNil(t, retrievedContextHash)
	assert.Equal(t, "ctx-snapshot-abc", *retrievedContextHash)
	require.NotNil(t, retrievedSeverity)
	assert.Equal(t, "high", *retrievedSeverity)
}
