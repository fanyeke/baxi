package mcp

import (
	"context"
	"testing"

	"baxi/internal/action"
	"baxi/internal/decision"
	"baxi/internal/llm"
	"baxi/internal/model"
)

// MockDecisionService is a mock implementation of DecisionService for testing.
type MockDecisionService struct {
	CreateCaseFromAlertFunc func(ctx context.Context, alertID, createdBy string) (*decision.DecisionCase, error)
	GetCaseFunc             func(ctx context.Context, caseID string) (*decision.DecisionCase, error)
	ListCasesFunc           func(ctx context.Context, filter decision.CaseFilter) (*decision.CaseList, error)
}

func (m *MockDecisionService) CreateCaseFromAlert(ctx context.Context, alertID, createdBy string) (*decision.DecisionCase, error) {
	if m.CreateCaseFromAlertFunc != nil {
		return m.CreateCaseFromAlertFunc(ctx, alertID, createdBy)
	}
	return nil, nil
}

func (m *MockDecisionService) GetCase(ctx context.Context, caseID string) (*decision.DecisionCase, error) {
	if m.GetCaseFunc != nil {
		return m.GetCaseFunc(ctx, caseID)
	}
	return nil, nil
}

func (m *MockDecisionService) ListCases(ctx context.Context, filter decision.CaseFilter) (*decision.CaseList, error) {
	if m.ListCasesFunc != nil {
		return m.ListCasesFunc(ctx, filter)
	}
	return &decision.CaseList{Cases: []decision.DecisionCase{}, Total: 0}, nil
}

// MockDecisionEngine is a mock implementation of DecisionEngine for testing.
type MockDecisionEngine struct {
	GenerateDecisionFunc func(ctx context.Context, caseID string, context *decision.DecisionContext) (*llm.DecisionOutput, error)
}

func (m *MockDecisionEngine) GenerateDecision(ctx context.Context, caseID string, context *decision.DecisionContext) (*llm.DecisionOutput, error) {
	if m.GenerateDecisionFunc != nil {
		return m.GenerateDecisionFunc(ctx, caseID, context)
	}
	return &llm.DecisionOutput{}, nil
}

// MockContextBuilder is a mock implementation of ContextBuilder for testing.
type MockContextBuilder struct {
	BuildDecisionContextFunc func(ctx context.Context, caseID string) (*decision.DecisionContext, error)
}

func (m *MockContextBuilder) BuildDecisionContext(ctx context.Context, caseID string) (*decision.DecisionContext, error) {
	if m.BuildDecisionContextFunc != nil {
		return m.BuildDecisionContextFunc(ctx, caseID)
	}
	return &decision.DecisionContext{}, nil
}

// MockProposalService is a mock implementation of ProposalService for testing.
type MockProposalService struct {
	ListProposalsFunc func(ctx context.Context, caseID string) ([]action.ActionProposal, error)
}

func (m *MockProposalService) ListProposals(ctx context.Context, caseID string) ([]action.ActionProposal, error) {
	if m.ListProposalsFunc != nil {
		return m.ListProposalsFunc(ctx, caseID)
	}
	return []action.ActionProposal{}, nil
}

// MockAlertService is a mock implementation of AlertService for testing.
type MockAlertService struct {
	ListAlertsFunc func(ctx context.Context, filters model.AlertFilters, sort string, limit, offset int) (*model.AlertListResponse, error)
}

func (m *MockAlertService) ListAlerts(ctx context.Context, filters model.AlertFilters, sort string, limit, offset int) (*model.AlertListResponse, error) {
	if m.ListAlertsFunc != nil {
		return m.ListAlertsFunc(ctx, filters, sort, limit, offset)
	}
	return &model.AlertListResponse{Items: []model.Alert{}, Total: 0}, nil
}

// MockGovernanceService is a mock implementation of GovernanceService for testing.
type MockGovernanceService struct {
	CheckAccessFunc       func(ctx context.Context, role, objectType, action string) (*model.AccessDecision, error)
	GetClassificationFunc func(ctx context.Context, fieldPath string) (*model.ClassificationResponse, error)
}

func (m *MockGovernanceService) CheckAccess(ctx context.Context, role, objectType, action string) (*model.AccessDecision, error) {
	if m.CheckAccessFunc != nil {
		return m.CheckAccessFunc(ctx, role, objectType, action)
	}
	allowed := model.AccessAllowed
	return &allowed, nil
}

func (m *MockGovernanceService) GetClassification(ctx context.Context, fieldPath string) (*model.ClassificationResponse, error) {
	if m.GetClassificationFunc != nil {
		return m.GetClassificationFunc(ctx, fieldPath)
	}
	return &model.ClassificationResponse{}, nil
}

// MockPipelineRunner is a mock implementation of PipelineRunner for testing.
type MockPipelineRunner struct {
	RunFunc func(ctx context.Context, config string) (string, error)
}

func (m *MockPipelineRunner) Run(ctx context.Context, config string) (string, error) {
	if m.RunFunc != nil {
		return m.RunFunc(ctx, config)
	}
	return "", nil
}

func TestNewServer(t *testing.T) {
	mockDecisionSvc := &MockDecisionService{}
	mockDecisionEngine := &MockDecisionEngine{}
	mockContextBuilder := &MockContextBuilder{}
	mockProposalSvc := &MockProposalService{}
	mockAlertSvc := &MockAlertService{}
	mockGovSvc := &MockGovernanceService{}
	mockPipelineRunner := &MockPipelineRunner{}

	server, err := NewServer(
		mockDecisionSvc,
		mockDecisionEngine,
		mockContextBuilder,
		mockProposalSvc,
		mockAlertSvc,
		mockGovSvc,
		mockPipelineRunner,
	)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	if server == nil {
		t.Fatal("Server is nil")
	}

	if server.server == nil {
		t.Fatal("MCP server is nil")
	}

	if server.decisionSvc != mockDecisionSvc {
		t.Error("Decision service not set correctly")
	}

	if server.decisionEngine != mockDecisionEngine {
		t.Error("Decision engine not set correctly")
	}

	if server.contextBuilder != mockContextBuilder {
		t.Error("Context builder not set correctly")
	}

	if server.proposalSvc != mockProposalSvc {
		t.Error("Proposal service not set correctly")
	}

	if server.alertSvc != mockAlertSvc {
		t.Error("Alert service not set correctly")
	}

	if server.govSvc != mockGovSvc {
		t.Error("Governance service not set correctly")
	}

	if server.pipelineRunner != mockPipelineRunner {
		t.Error("Pipeline runner not set correctly")
	}
}

func TestServerToolRegistration(t *testing.T) {
	mockDecisionSvc := &MockDecisionService{}
	mockDecisionEngine := &MockDecisionEngine{}
	mockContextBuilder := &MockContextBuilder{}
	mockProposalSvc := &MockProposalService{}
	mockAlertSvc := &MockAlertService{}
	mockGovSvc := &MockGovernanceService{}
	mockPipelineRunner := &MockPipelineRunner{}

	server, err := NewServer(
		mockDecisionSvc,
		mockDecisionEngine,
		mockContextBuilder,
		mockProposalSvc,
		mockAlertSvc,
		mockGovSvc,
		mockPipelineRunner,
	)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	// Get the list of tools from the underlying server
	tools := server.server.ListTools()
	if tools == nil {
		t.Fatal("ListTools returned nil")
	}

	// Expected tools
	expectedTools := []string{
		"create_decision_case",
		"decide",
		"list_cases",
		"get_case",
		"list_proposals",
		"list_alerts",
		"check_access",
		"get_classification",
		"run_pipeline",
	}

	// Check that all expected tools are registered
	toolNames := make(map[string]bool)
	for name := range tools {
		toolNames[name] = true
	}

	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Errorf("Expected tool '%s' not found in registered tools", expected)
		}
	}

	// Verify we have the expected number of tools
	if len(tools) != len(expectedTools) {
		t.Errorf("Expected %d tools, got %d", len(expectedTools), len(tools))
	}
}
