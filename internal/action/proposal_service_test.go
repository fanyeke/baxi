package action

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"baxi/internal/llm"
	"baxi/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

// --- Mocks ---

type mockProposalRepo struct {
	createProposalFn      func(ctx context.Context, pool *pgxpool.Pool, row *repository.ActionProposalRow) error
	listProposalsByCaseFn func(ctx context.Context, pool *pgxpool.Pool, caseID string) ([]repository.ActionProposalRow, error)
}

func (m *mockProposalRepo) CreateProposal(ctx context.Context, pool *pgxpool.Pool, row *repository.ActionProposalRow) error {
	return m.createProposalFn(ctx, pool, row)
}

func (m *mockProposalRepo) ListProposalsByCase(ctx context.Context, pool *pgxpool.Pool, caseID string) ([]repository.ActionProposalRow, error) {
	return m.listProposalsByCaseFn(ctx, pool, caseID)
}

type mockCaseUpdater struct {
	updateCaseStatusFn func(ctx context.Context, pool *pgxpool.Pool, caseID string, status string, contextJSON *json.RawMessage, contextHash *string, governanceSnapshot *json.RawMessage) error
}

func (m *mockCaseUpdater) UpdateCaseStatus(ctx context.Context, pool *pgxpool.Pool, caseID string, status string, contextJSON *json.RawMessage, contextHash *string, governanceSnapshot *json.RawMessage) error {
	return m.updateCaseStatusFn(ctx, pool, caseID, status, contextJSON, contextHash, governanceSnapshot)
}

// --- Compile-time interface checks ---

var _ ProposalRepository = (*mockProposalRepo)(nil)
var _ CaseStatusUpdater = (*mockCaseUpdater)(nil)

// --- Test: GenerateProposals ---

func TestGenerateProposals_CreatesProposalsFromDecision(t *testing.T) {
	caseID := "dc_test_123"
	decisionID := "de_test_456"

	var savedProposals []repository.ActionProposalRow

	repo := &mockProposalRepo{
		createProposalFn: func(ctx context.Context, pool *pgxpool.Pool, row *repository.ActionProposalRow) error {
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

	proposals, err := svc.GenerateProposals(context.Background(), caseID, decisionID, decision)

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
		createProposalFn: func(ctx context.Context, pool *pgxpool.Pool, row *repository.ActionProposalRow) error {
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

	proposals, err := svc.GenerateProposals(context.Background(), "case-1", "dec-1", decision)

	assert.NoError(t, err)
	assert.Len(t, proposals, 3)
	for _, p := range proposals {
		assert.True(t, p.RequiresHumanReview, "proposal %s must require human review", p.ProposalID)
	}
}

func TestGenerateProposals_AllProposalsHaveApplyStatusProposed(t *testing.T) {
	repo := &mockProposalRepo{
		createProposalFn: func(ctx context.Context, pool *pgxpool.Pool, row *repository.ActionProposalRow) error {
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

	proposals, err := svc.GenerateProposals(context.Background(), "case-2", "dec-2", decision)

	assert.NoError(t, err)
	assert.Len(t, proposals, 1)
	assert.Equal(t, "proposed", proposals[0].ApplyStatus)
}

func TestGenerateProposals_UpdatesCaseStatus(t *testing.T) {
	caseID := "case-status-update"
	decisionID := "dec-status-update"

	repo := &mockProposalRepo{
		createProposalFn: func(ctx context.Context, pool *pgxpool.Pool, row *repository.ActionProposalRow) error {
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

	_, err := svc.GenerateProposals(context.Background(), caseID, decisionID, decision)

	assert.NoError(t, err)
	assert.True(t, statusUpdated, "case status must be updated to proposal_generated")
}

func TestGenerateProposals_EmptyDecisionReturnsEmptyList(t *testing.T) {
	repo := &mockProposalRepo{
		createProposalFn: func(ctx context.Context, pool *pgxpool.Pool, row *repository.ActionProposalRow) error {
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

	proposals, err := svc.GenerateProposals(context.Background(), "case-empty", "dec-empty", decision)

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

	rows := []repository.ActionProposalRow{
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
		listProposalsByCaseFn: func(ctx context.Context, pool *pgxpool.Pool, cid string) ([]repository.ActionProposalRow, error) {
			assert.Equal(t, caseID, cid)
			return rows, nil
		},
	}

	svc := NewProposalService(repo, &mockCaseUpdater{}, nil)
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
		listProposalsByCaseFn: func(ctx context.Context, pool *pgxpool.Pool, caseID string) ([]repository.ActionProposalRow, error) {
			return []repository.ActionProposalRow{}, nil
		},
	}

	svc := NewProposalService(repo, &mockCaseUpdater{}, nil)
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
		createProposalFn: func(ctx context.Context, pool *pgxpool.Pool, row *repository.ActionProposalRow) error {
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

	proposals, err := svc.GenerateProposals(context.Background(), "case-trunc", "dec-trunc", decision)

	assert.NoError(t, err)
	assert.Len(t, proposals, 1)
	assert.True(t, len(proposals[0].Title) <= 200)
}
