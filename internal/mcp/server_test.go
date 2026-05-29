package mcp

import (
	"context"
	"testing"

	"baxi/internal/action"
	"baxi/internal/decision"
	"baxi/internal/llm"
	"baxi/internal/model"
	"baxi/internal/review"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MockDecisionService struct {
	CreateCaseFromAlertFunc func(ctx context.Context, alertID, createdBy string) (*decision.DecisionCase, error)
	GetCaseFunc             func(ctx context.Context, caseID string) (*decision.DecisionCase, error)
	ListCasesFunc           func(ctx context.Context, filter decision.CaseFilter) (*decision.CaseList, error)
	DecideFunc              func(ctx context.Context, caseID string) ([]action.ActionProposal, error)
	ResolveCaseFunc         func(ctx context.Context, caseID, resolution string, comment string) error
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

func (m *MockDecisionService) Decide(ctx context.Context, caseID string) ([]action.ActionProposal, error) {
	if m.DecideFunc != nil {
		return m.DecideFunc(ctx, caseID)
	}
	return []action.ActionProposal{}, nil
}

func (m *MockDecisionService) ResolveCase(ctx context.Context, caseID, resolution string, comment string) error {
	if m.ResolveCaseFunc != nil {
		return m.ResolveCaseFunc(ctx, caseID, resolution, comment)
	}
	return nil
}

type MockDecisionEngine struct {
	GenerateDecisionFunc func(ctx context.Context, caseID string, context *decision.DecisionContext) (*llm.DecisionOutput, error)
}

func (m *MockDecisionEngine) GenerateDecision(ctx context.Context, caseID string, context *decision.DecisionContext) (*llm.DecisionOutput, error) {
	if m.GenerateDecisionFunc != nil {
		return m.GenerateDecisionFunc(ctx, caseID, context)
	}
	return &llm.DecisionOutput{}, nil
}

type MockContextBuilder struct {
	BuildDecisionContextFunc func(ctx context.Context, caseID string) (*decision.DecisionContext, error)
}

func (m *MockContextBuilder) BuildDecisionContext(ctx context.Context, caseID string) (*decision.DecisionContext, error) {
	if m.BuildDecisionContextFunc != nil {
		return m.BuildDecisionContextFunc(ctx, caseID)
	}
	return &decision.DecisionContext{}, nil
}

type MockProposalService struct {
	ListProposalsFunc func(ctx context.Context, caseID string) ([]action.ActionProposal, error)
}

func (m *MockProposalService) ListProposals(ctx context.Context, caseID string) ([]action.ActionProposal, error) {
	if m.ListProposalsFunc != nil {
		return m.ListProposalsFunc(ctx, caseID)
	}
	return []action.ActionProposal{}, nil
}

type MockAlertService struct {
	ListAlertsFunc func(ctx context.Context, filters model.AlertFilters, sort string, limit, offset int) (*model.AlertListResponse, error)
}

func (m *MockAlertService) ListAlerts(ctx context.Context, filters model.AlertFilters, sort string, limit, offset int) (*model.AlertListResponse, error) {
	if m.ListAlertsFunc != nil {
		return m.ListAlertsFunc(ctx, filters, sort, limit, offset)
	}
	return &model.AlertListResponse{Items: []model.Alert{}, Total: 0}, nil
}

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

type MockPipelineRunner struct {
	RunFunc func(ctx context.Context, config string) (string, error)
}

func (m *MockPipelineRunner) Run(ctx context.Context, config string) (string, error) {
	if m.RunFunc != nil {
		return m.RunFunc(ctx, config)
	}
	return "", nil
}

type MockReviewService struct {
	ApproveProposalFunc func(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error)
	RejectProposalFunc  func(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error)
	CancelProposalFunc  func(ctx context.Context, proposalID, reason string) error
	GetProposalByIDFunc func(ctx context.Context, proposalID string) (*action.ActionProposal, error)
}

func (m *MockReviewService) ApproveProposal(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error) {
	if m.ApproveProposalFunc != nil {
		return m.ApproveProposalFunc(ctx, proposalID, reviewerID, feedback)
	}
	return nil, nil
}

func (m *MockReviewService) RejectProposal(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error) {
	if m.RejectProposalFunc != nil {
		return m.RejectProposalFunc(ctx, proposalID, reviewerID, feedback)
	}
	return nil, nil
}

func (m *MockReviewService) CancelProposal(ctx context.Context, proposalID, reason string) error {
	if m.CancelProposalFunc != nil {
		return m.CancelProposalFunc(ctx, proposalID, reason)
	}
	return nil
}

func (m *MockReviewService) GetProposalByID(ctx context.Context, proposalID string) (*action.ActionProposal, error) {
	if m.GetProposalByIDFunc != nil {
		return m.GetProposalByIDFunc(ctx, proposalID)
	}
	return nil, nil
}

type MockOutboxService struct {
	ListOutboxEventsFunc func(ctx context.Context, status string, limit, offset int) ([]model.OutboxEvent, int, error)
}

func (m *MockOutboxService) ListOutboxEvents(ctx context.Context, status string, limit, offset int) ([]model.OutboxEvent, int, error) {
	if m.ListOutboxEventsFunc != nil {
		return m.ListOutboxEventsFunc(ctx, status, limit, offset)
	}
	return []model.OutboxEvent{}, 0, nil
}

type MockPipelineInfoService struct {
	GetLastRunStatusFunc func(ctx context.Context) (*model.PipelineRun, error)
	ListRunsFunc         func(ctx context.Context, limit int) ([]model.PipelineRun, error)
}

func (m *MockPipelineInfoService) GetLastRunStatus(ctx context.Context) (*model.PipelineRun, error) {
	if m.GetLastRunStatusFunc != nil {
		return m.GetLastRunStatusFunc(ctx)
	}
	return nil, nil
}

func (m *MockPipelineInfoService) ListRuns(ctx context.Context, limit int) ([]model.PipelineRun, error) {
	if m.ListRunsFunc != nil {
		return m.ListRunsFunc(ctx, limit)
	}
	return []model.PipelineRun{}, nil
}

type MockSystemStatusService struct {
	GetStatusFunc func(ctx context.Context) (*model.SystemStatus, error)
}

func (m *MockSystemStatusService) GetStatus(ctx context.Context) (*model.SystemStatus, error) {
	if m.GetStatusFunc != nil {
		return m.GetStatusFunc(ctx)
	}
	return &model.SystemStatus{}, nil
}

type MockObjectSearchService struct {
	SearchObjectsFunc func(ctx context.Context, objectType, query string, limit, offset int) (*model.SearchResult, error)
}

func (m *MockObjectSearchService) SearchObjects(ctx context.Context, objectType, query string, limit, offset int) (*model.SearchResult, error) {
	if m.SearchObjectsFunc != nil {
		return m.SearchObjectsFunc(ctx, objectType, query, limit, offset)
	}
	return &model.SearchResult{Items: []map[string]interface{}{}, Total: 0}, nil
}

type MockExecuteService struct {
	ExecuteProposalFunc func(ctx context.Context, pool *pgxpool.Pool, proposalID string, actorID string, opts ...action.ExecuteOption) (*action.ExecutionResult, error)
}

func (m *MockExecuteService) ExecuteProposal(ctx context.Context, pool *pgxpool.Pool, proposalID string, actorID string, opts ...action.ExecuteOption) (*action.ExecutionResult, error) {
	if m.ExecuteProposalFunc != nil {
		return m.ExecuteProposalFunc(ctx, pool, proposalID, actorID, opts...)
	}
	return &action.ExecutionResult{Success: true, DryRun: true}, nil
}

type MockOntologyService struct {
	DescribeOntologyFunc    func(ctx context.Context) (*OntologyDescriptor, error)
	GetObjectFunc           func(ctx context.Context, objectType, objectID string) (*ObjectContext, error)
	GetLinkedObjectsFunc    func(ctx context.Context, objectType, objectID, linkName string, maxDepth int) (*LinkedObjectsResult, error)
	ExecuteActionFunc       func(ctx context.Context, objectType, objectID, actionType string, params map[string]interface{}) (*ActionResult, error)
}

func (m *MockOntologyService) DescribeOntology(ctx context.Context) (*OntologyDescriptor, error) {
	if m.DescribeOntologyFunc != nil {
		return m.DescribeOntologyFunc(ctx)
	}
	return &OntologyDescriptor{ObjectTypes: []ObjectTypeDescriptor{}}, nil
}

func (m *MockOntologyService) GetObject(ctx context.Context, objectType, objectID string) (*ObjectContext, error) {
	if m.GetObjectFunc != nil {
		return m.GetObjectFunc(ctx, objectType, objectID)
	}
	return &ObjectContext{ObjectType: objectType, ObjectID: objectID, Properties: map[string]interface{}{}}, nil
}

func (m *MockOntologyService) GetLinkedObjects(ctx context.Context, objectType, objectID, linkName string, maxDepth int) (*LinkedObjectsResult, error) {
	if m.GetLinkedObjectsFunc != nil {
		return m.GetLinkedObjectsFunc(ctx, objectType, objectID, linkName, maxDepth)
	}
	return &LinkedObjectsResult{ObjectType: objectType, ObjectID: objectID, Links: []LinkResult{}}, nil
}

func (m *MockOntologyService) ExecuteAction(ctx context.Context, objectType, objectID, actionType string, params map[string]interface{}) (*ActionResult, error) {
	if m.ExecuteActionFunc != nil {
		return m.ExecuteActionFunc(ctx, objectType, objectID, actionType, params)
	}
	return &ActionResult{Success: true, ActionType: actionType, ObjectType: objectType, ObjectID: objectID, Result: map[string]interface{}{}}, nil
}

func TestNewServer(t *testing.T) {
	server, err := NewServer(
		&MockDecisionService{},
		&MockDecisionEngine{},
		&MockContextBuilder{},
		&MockProposalService{},
		&MockAlertService{},
		&MockGovernanceService{},
		&MockPipelineRunner{},
		&MockReviewService{},
		&MockOutboxService{},
		&MockPipelineInfoService{},
		&MockExecuteService{},
		(*pgxpool.Pool)(nil),
		&MockSystemStatusService{},
		&MockObjectSearchService{},
		&MockOntologyService{},
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
}

func TestServerToolRegistration(t *testing.T) {
	server, err := NewServer(
		&MockDecisionService{},
		&MockDecisionEngine{},
		&MockContextBuilder{},
		&MockProposalService{},
		&MockAlertService{},
		&MockGovernanceService{},
		&MockPipelineRunner{},
		&MockReviewService{},
		&MockOutboxService{},
		&MockPipelineInfoService{},
		&MockExecuteService{},
		(*pgxpool.Pool)(nil),
		&MockSystemStatusService{},
		&MockObjectSearchService{},
		&MockOntologyService{},
	)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	tools := server.server.ListTools()
	if tools == nil {
		t.Fatal("ListTools returned nil")
	}

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
		"approve_proposal",
		"reject_proposal",
		"execute_proposal",
		"get_decision_context",
		"get_system_status",
		"search_objects",
		"list_outbox_events",
		"get_pipeline_status",
		"describe_ontology",
		"get_object",
		"get_linked_objects",
		"execute_action",
		"resolve_case",
		"cancel_proposal",
		"get_proposal_by_id",
	}

	toolNames := make(map[string]bool)
	for name := range tools {
		toolNames[name] = true
	}

	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Errorf("Expected tool '%s' not found in registered tools", expected)
		}
	}

	if len(tools) != len(expectedTools) {
		t.Errorf("Expected %d tools, got %d", len(expectedTools), len(tools))
	}
}
