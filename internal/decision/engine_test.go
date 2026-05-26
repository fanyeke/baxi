package decision

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"baxi/internal/llm"
	"baxi/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

type mockDecisionProvider struct {
	generateDecisionFn func(ctx context.Context, input llm.LLMSafeContext) (*llm.DecisionOutput, error)
}

func (m *mockDecisionProvider) GenerateDecision(ctx context.Context, input llm.LLMSafeContext) (*llm.DecisionOutput, error) {
	return m.generateDecisionFn(ctx, input)
}

type mockDecisionEngineRepository struct {
	createDecisionFn   func(ctx context.Context, pool *pgxpool.Pool, row *repository.LLMDecisionRow) error
	updateCaseStatusFn func(ctx context.Context, pool *pgxpool.Pool, caseID string, status string, contextJSON *json.RawMessage, contextHash *string, governanceSnapshot *json.RawMessage) error
	getCaseByIDFn      func(ctx context.Context, pool *pgxpool.Pool, caseID string) (*repository.DecisionCaseRow, error)
}

func (m *mockDecisionEngineRepository) CreateDecision(ctx context.Context, pool *pgxpool.Pool, row *repository.LLMDecisionRow) error {
	return m.createDecisionFn(ctx, pool, row)
}

func (m *mockDecisionEngineRepository) UpdateCaseStatus(ctx context.Context, pool *pgxpool.Pool, caseID string, status string, contextJSON *json.RawMessage, contextHash *string, governanceSnapshot *json.RawMessage) error {
	return m.updateCaseStatusFn(ctx, pool, caseID, status, contextJSON, contextHash, governanceSnapshot)
}

func (m *mockDecisionEngineRepository) GetCaseByID(ctx context.Context, pool *pgxpool.Pool, caseID string) (*repository.DecisionCaseRow, error) {
	return m.getCaseByIDFn(ctx, pool, caseID)
}

func validDecisionContext() *DecisionContext {
	return &DecisionContext{
		DecisionCaseID: "dc-1",
		SourceType:     "alert",
		SourceID:       "alert-1",
		Trigger: TriggerInfo{
			AlertID:       "alert-1",
			RuleID:        "rule-1",
			Severity:      "high",
			MetricName:    "gmv_drop",
			CurrentValue:  100.0,
			BaselineValue: 150.0,
			DeltaPct:      -33.3,
		},
		ObjectContext: ObjectContextData{
			ObjectType: "seller",
			ObjectID:   "seller-42",
			Properties: map[string]interface{}{
				"name": "Test Seller",
			},
		},
		Governance: GovernanceData{
			Classification:   "L2",
			RedactionApplied: false,
			RedactedFields:   []string{},
			Role:             "agent_readonly",
		},
		AllowedActions: []string{
			llm.ActionTypeCreateFollowupTask,
			llm.ActionTypeNotifyOwner,
			llm.ActionTypeExportReport,
			llm.ActionTypeEscalateToHuman,
		},
		ForbiddenActions: []string{"execute_dispatch"},
	}
}

func validDecisionOutput() *llm.DecisionOutput {
	return &llm.DecisionOutput{
		DecisionType:        llm.DecisionTypeInvestigate,
		Severity:            llm.SeverityHigh,
		Summary:             "Test summary",
		Rationale:           []string{"reason 1"},
		RecommendedActions:  []llm.RecommendedAction{
			{ActionType: llm.ActionTypeNotifyOwner, Priority: llm.SeverityHigh, OwnerRole: "ops"},
		},
		Confidence:          0.85,
		RequiresHumanReview: true,
	}
}

func invalidDecisionOutput() *llm.DecisionOutput {
	return &llm.DecisionOutput{
		DecisionType:        "invalid_type",
		Severity:            llm.SeverityHigh,
		Summary:             "Invalid decision",
		Rationale:           []string{"bad"},
		RecommendedActions:  []llm.RecommendedAction{},
		Confidence:          0.85,
		RequiresHumanReview: true,
	}
}

func TestEngine_ValidPath(t *testing.T) {
	ctx := context.Background()
	dc := validDecisionContext()
	expectedOutput := validDecisionOutput()

	var savedRow *repository.LLMDecisionRow
	var updatedStatus string

	provider := &mockDecisionProvider{
		generateDecisionFn: func(ctx context.Context, input llm.LLMSafeContext) (*llm.DecisionOutput, error) {
			return expectedOutput, nil
		},
	}

	repo := &mockDecisionEngineRepository{
		createDecisionFn: func(ctx context.Context, pool *pgxpool.Pool, row *repository.LLMDecisionRow) error {
			savedRow = row
			return nil
		},
		updateCaseStatusFn: func(ctx context.Context, pool *pgxpool.Pool, caseID string, status string, contextJSON *json.RawMessage, contextHash *string, governanceSnapshot *json.RawMessage) error {
			updatedStatus = status
			return nil
		},
	}

	engine := NewDecisionEngine(provider, repo, nil)
	output, err := engine.GenerateDecision(ctx, "dc-1", dc)

	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, expectedOutput.DecisionType, output.DecisionType)
	assert.Equal(t, expectedOutput.Severity, output.Severity)
	assert.Equal(t, expectedOutput.Confidence, output.Confidence)

	assert.NotNil(t, savedRow)
	assert.Equal(t, "dc-1", savedRow.CaseID)
	assert.Equal(t, "valid", *savedRow.Status)
	assert.NotNil(t, savedRow.OutputJSON)
	assert.Nil(t, savedRow.ValidationErrors)
	assert.Nil(t, savedRow.FallbackReason)
	assert.Equal(t, "decision_generated", updatedStatus)
}

func TestEngine_FallbackOnInvalid(t *testing.T) {
	ctx := context.Background()
	dc := validDecisionContext()

	var savedRow *repository.LLMDecisionRow
	var updatedStatus string

	provider := &mockDecisionProvider{
		generateDecisionFn: func(ctx context.Context, input llm.LLMSafeContext) (*llm.DecisionOutput, error) {
			return invalidDecisionOutput(), nil
		},
	}

	repo := &mockDecisionEngineRepository{
		createDecisionFn: func(ctx context.Context, pool *pgxpool.Pool, row *repository.LLMDecisionRow) error {
			savedRow = row
			return nil
		},
		updateCaseStatusFn: func(ctx context.Context, pool *pgxpool.Pool, caseID string, status string, contextJSON *json.RawMessage, contextHash *string, governanceSnapshot *json.RawMessage) error {
			updatedStatus = status
			return nil
		},
	}

	engine := NewDecisionEngine(provider, repo, nil)
	output, err := engine.GenerateDecision(ctx, "dc-1", dc)

	assert.NoError(t, err)
	assert.NotNil(t, output)

	assert.NotNil(t, savedRow)
	assert.Equal(t, "dc-1", savedRow.CaseID)
	assert.Equal(t, "fallback", *savedRow.Status)
	assert.NotNil(t, savedRow.OutputJSON)
	assert.NotNil(t, savedRow.ValidationErrors)
	assert.NotNil(t, savedRow.FallbackReason)
	assert.Equal(t, "decision_generated", updatedStatus)

	var validationErrors []llm.ValidationError
	unmarshalErr := json.Unmarshal(*savedRow.ValidationErrors, &validationErrors)
	assert.NoError(t, unmarshalErr)
	assert.Greater(t, len(validationErrors), 0)
}

func TestEngine_BothFail(t *testing.T) {
	ctx := context.Background()
	dc := validDecisionContext()

	var savedRow *repository.LLMDecisionRow
	var updatedStatus string

	provider := &mockDecisionProvider{
		generateDecisionFn: func(ctx context.Context, input llm.LLMSafeContext) (*llm.DecisionOutput, error) {
			return nil, errors.New("provider connection refused")
		},
	}

	repo := &mockDecisionEngineRepository{
		createDecisionFn: func(ctx context.Context, pool *pgxpool.Pool, row *repository.LLMDecisionRow) error {
			savedRow = row
			return nil
		},
		updateCaseStatusFn: func(ctx context.Context, pool *pgxpool.Pool, caseID string, status string, contextJSON *json.RawMessage, contextHash *string, governanceSnapshot *json.RawMessage) error {
			updatedStatus = status
			return nil
		},
	}

	engine := NewDecisionEngine(provider, repo, nil)
	engine.fallback = &mockDecisionProvider{
		generateDecisionFn: func(ctx context.Context, input llm.LLMSafeContext) (*llm.DecisionOutput, error) {
			return nil, errors.New("fallback also failed")
		},
	}

	output, err := engine.GenerateDecision(ctx, "dc-1", dc)

	assert.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "provider error")
	assert.Contains(t, err.Error(), "fallback error")

	assert.NotNil(t, savedRow)
	assert.Equal(t, "dc-1", savedRow.CaseID)
	assert.Equal(t, "failed", *savedRow.Status)
	assert.NotNil(t, savedRow.FallbackReason)
	assert.Contains(t, *savedRow.FallbackReason, "provider error")
	assert.Contains(t, *savedRow.FallbackReason, "fallback error")
	assert.Equal(t, "failed", updatedStatus)
}

func TestEngine_ProviderError(t *testing.T) {
	ctx := context.Background()
	dc := validDecisionContext()

	var savedRow *repository.LLMDecisionRow
	var updatedStatus string

	provider := &mockDecisionProvider{
		generateDecisionFn: func(ctx context.Context, input llm.LLMSafeContext) (*llm.DecisionOutput, error) {
			return nil, errors.New("provider rate limited")
		},
	}

	repo := &mockDecisionEngineRepository{
		createDecisionFn: func(ctx context.Context, pool *pgxpool.Pool, row *repository.LLMDecisionRow) error {
			savedRow = row
			return nil
		},
		updateCaseStatusFn: func(ctx context.Context, pool *pgxpool.Pool, caseID string, status string, contextJSON *json.RawMessage, contextHash *string, governanceSnapshot *json.RawMessage) error {
			updatedStatus = status
			return nil
		},
	}

	engine := NewDecisionEngine(provider, repo, nil)
	output, err := engine.GenerateDecision(ctx, "dc-1", dc)

	assert.NoError(t, err)
	assert.NotNil(t, output)

	assert.NotNil(t, savedRow)
	assert.Equal(t, "dc-1", savedRow.CaseID)
	assert.Equal(t, "fallback", *savedRow.Status)
	assert.NotNil(t, savedRow.FallbackReason)
	assert.Contains(t, *savedRow.FallbackReason, "provider error")
	assert.Equal(t, "decision_generated", updatedStatus)
}

func TestEngine_ContextBuilding(t *testing.T) {
	ctx := context.Background()
	dc := validDecisionContext()

	var capturedInput llm.LLMSafeContext

	provider := &mockDecisionProvider{
		generateDecisionFn: func(ctx context.Context, input llm.LLMSafeContext) (*llm.DecisionOutput, error) {
			capturedInput = input
			return validDecisionOutput(), nil
		},
	}

	repo := &mockDecisionEngineRepository{
		createDecisionFn: func(ctx context.Context, pool *pgxpool.Pool, row *repository.LLMDecisionRow) error {
			return nil
		},
		updateCaseStatusFn: func(ctx context.Context, pool *pgxpool.Pool, caseID string, status string, contextJSON *json.RawMessage, contextHash *string, governanceSnapshot *json.RawMessage) error {
			return nil
		},
	}

	engine := NewDecisionEngine(provider, repo, nil)
	_, err := engine.GenerateDecision(ctx, "dc-1", dc)

	assert.NoError(t, err)
	assert.Equal(t, dc.DecisionCaseID, capturedInput.CaseID)
	assert.Equal(t, dc.Trigger.AlertID, capturedInput.Trigger.AlertID)
	assert.Equal(t, dc.Trigger.Severity, capturedInput.Trigger.Severity)
	assert.Equal(t, dc.Trigger.MetricName, capturedInput.Trigger.MetricName)
	assert.Equal(t, dc.Trigger.CurrentValue, capturedInput.Trigger.CurrentValue)
	assert.Equal(t, dc.Trigger.BaselineValue, capturedInput.Trigger.BaselineValue)
	assert.Equal(t, dc.Trigger.DeltaPct, capturedInput.Trigger.DeltaPct)
	assert.Equal(t, dc.ObjectContext.ObjectType, capturedInput.ObjectContext.ObjectType)
	assert.Equal(t, dc.ObjectContext.ObjectID, capturedInput.ObjectContext.ObjectID)
	assert.Equal(t, dc.Governance.Classification, capturedInput.GovernanceInfo.Classification)
	assert.Equal(t, dc.Governance.RedactionApplied, capturedInput.GovernanceInfo.RedactionApplied)
	assert.Equal(t, dc.Governance.Role, capturedInput.GovernanceInfo.Role)
	assert.Equal(t, dc.AllowedActions, capturedInput.AllowedActions)
	assert.Equal(t, dc.ForbiddenActions, capturedInput.ForbiddenActions)
}
