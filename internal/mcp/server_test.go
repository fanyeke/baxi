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
	"github.com/mark3labs/mcp-go/mcp"
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
	ApproveProposalFunc   func(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error)
	RejectProposalFunc    func(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error)
	CancelProposalFunc    func(ctx context.Context, proposalID, reviewerID, reason string) error
	GetProposalByIDFunc   func(ctx context.Context, proposalID string) (*action.ActionProposal, error)
	ListReviewRecordsFunc func(ctx context.Context, proposalID string, limit, offset int) ([]review.ReviewRecord, int, error)
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

func (m *MockReviewService) CancelProposal(ctx context.Context, proposalID, reviewerID, reason string) error {
	if m.CancelProposalFunc != nil {
		return m.CancelProposalFunc(ctx, proposalID, reviewerID, reason)
	}
	return nil
}

func (m *MockReviewService) GetProposalByID(ctx context.Context, proposalID string) (*action.ActionProposal, error) {
	if m.GetProposalByIDFunc != nil {
		return m.GetProposalByIDFunc(ctx, proposalID)
	}
	return nil, nil
}

func (m *MockReviewService) ListReviewRecords(ctx context.Context, proposalID string, limit, offset int) ([]review.ReviewRecord, int, error) {
	if m.ListReviewRecordsFunc != nil {
		return m.ListReviewRecordsFunc(ctx, proposalID, limit, offset)
	}
	return []review.ReviewRecord{}, 0, nil
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
	DescribeOntologyFunc func(ctx context.Context) (*OntologyDescriptor, error)
	GetObjectFunc        func(ctx context.Context, objectType, objectID string) (*ObjectContext, error)
	GetObjectMetricsFunc func(ctx context.Context, objectType, objectID string) (map[string]float64, error)
	GetLinkedObjectsFunc func(ctx context.Context, objectType, objectID, linkName string, maxDepth int) (*LinkedObjectsResult, error)
	ExecuteActionFunc    func(ctx context.Context, objectType, objectID, actionType string, params map[string]interface{}) (*ActionResult, error)
	ProposeActionFunc    func(ctx context.Context, objectType, objectID, actionType string, params map[string]interface{}, trace ProposeActionTrace) (*ActionResult, error)
}

type MockBuildContextService struct {
	BuildEnvelopeFunc func(ctx context.Context, caseID, recipeID string) (*llm.LLMSafeContextEnvelope, error)
}

type MockActionSchemaService struct {
	ListActionSchemasFunc func(ctx context.Context) ([]ActionDefinition, error)
	GetActionSchemaFunc   func(ctx context.Context, actionType string) (*ActionDefinition, error)
}

func (m *MockActionSchemaService) ListActionSchemas(ctx context.Context) ([]ActionDefinition, error) {
	if m.ListActionSchemasFunc != nil {
		return m.ListActionSchemasFunc(ctx)
	}
	return []ActionDefinition{}, nil
}

func (m *MockActionSchemaService) GetActionSchema(ctx context.Context, actionType string) (*ActionDefinition, error) {
	if m.GetActionSchemaFunc != nil {
		return m.GetActionSchemaFunc(ctx, actionType)
	}
	return nil, nil
}

type MockSandboxService struct {
	CreateSandboxFunc        func(ctx context.Context, caseID string, data map[string]interface{}) (string, error)
	AddProposalToSandboxFunc func(ctx context.Context, sandboxID, proposalID string) error
	CompareSandboxFunc       func(ctx context.Context, sandboxID1, sandboxID2 string) (*ComparisonResult, error)
	GetSandboxFunc           func(ctx context.Context, sandboxID string) (*Sandbox, error)
}

func (m *MockSandboxService) CreateSandbox(ctx context.Context, caseID string, data map[string]interface{}) (string, error) {
	if m.CreateSandboxFunc != nil {
		return m.CreateSandboxFunc(ctx, caseID, data)
	}
	return "sbx_mock", nil
}

func (m *MockSandboxService) AddProposalToSandbox(ctx context.Context, sandboxID, proposalID string) error {
	if m.AddProposalToSandboxFunc != nil {
		return m.AddProposalToSandboxFunc(ctx, sandboxID, proposalID)
	}
	return nil
}

func (m *MockSandboxService) CompareSandbox(ctx context.Context, sandboxID1, sandboxID2 string) (*ComparisonResult, error) {
	if m.CompareSandboxFunc != nil {
		return m.CompareSandboxFunc(ctx, sandboxID1, sandboxID2)
	}
	return &ComparisonResult{Sandbox1ID: sandboxID1, Sandbox2ID: sandboxID2, Differences: []Difference{}}, nil
}

func (m *MockSandboxService) GetSandbox(ctx context.Context, sandboxID string) (*Sandbox, error) {
	if m.GetSandboxFunc != nil {
		return m.GetSandboxFunc(ctx, sandboxID)
	}
	return nil, nil
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

func (m *MockOntologyService) GetObjectMetrics(ctx context.Context, objectType, objectID string) (map[string]float64, error) {
	if m.GetObjectMetricsFunc != nil {
		return m.GetObjectMetricsFunc(ctx, objectType, objectID)
	}
	return map[string]float64{}, nil
}

func (m *MockOntologyService) ProposeAction(ctx context.Context, objectType, objectID, actionType string, params map[string]interface{}, trace ProposeActionTrace) (*ActionResult, error) {
	if m.ProposeActionFunc != nil {
		return m.ProposeActionFunc(ctx, objectType, objectID, actionType, params, trace)
	}
	return &ActionResult{Success: true, ActionType: actionType, ObjectType: objectType, ObjectID: objectID, Result: map[string]interface{}{"proposal_id": "mock-proposal-123", "status": "proposed"}}, nil
}

func (m *MockBuildContextService) BuildEnvelope(ctx context.Context, caseID, recipeID string) (*llm.LLMSafeContextEnvelope, error) {
	if m.BuildEnvelopeFunc != nil {
		return m.BuildEnvelopeFunc(ctx, caseID, recipeID)
	}
	return &llm.LLMSafeContextEnvelope{CaseID: caseID, ContextHash: "mock-hash"}, nil
}

func TestNewServer(t *testing.T) {
	server, err := NewServer(
		&MockDecisionService{},
		&MockDecisionEngine{},
		&MockContextBuilder{},
		nil, // buildContextSvc
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
		&MockActionSchemaService{},
		&MockSandboxService{},
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
		nil, // buildContextSvc
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
		&MockActionSchemaService{},
		&MockSandboxService{},
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
		"build_context",
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
		"propose_action",
		"resolve_case",
		"cancel_proposal",
		"get_proposal_by_id",
		"list_review_records",
		"list_action_schemas",
		"get_action_schema",
		"create_sandbox",
		"add_to_sandbox",
		"compare_sandboxes",
		"get_sandbox",
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

func TestServerIdentity_DefaultsFromEnv(t *testing.T) {
	// Regression test: Server identity must not be empty (prevents anonymous review/execute).
	server, err := NewServer(
		&MockDecisionService{},
		&MockDecisionEngine{},
		&MockContextBuilder{},
		nil,
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
		&MockActionSchemaService{},
		&MockSandboxService{},
	)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	if server.mcpUserID == "" {
		t.Error("mcpUserID must not be empty")
	}
}

func TestServerIdentity_ApproveRejectUseServerIdentity(t *testing.T) {
	// Regression test: approve_proposal and reject_proposal must use
	// the server's configured mcpUserID, not a caller-supplied value.
	usedReviewerID := ""
	reviewSvc := &MockReviewService{
		ApproveProposalFunc: func(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error) {
			usedReviewerID = reviewerID
			return &review.ReviewRecord{RecordID: "r1", ProposalID: proposalID, Verdict: review.VerdictApprove, Feedback: feedback}, nil
		},
		RejectProposalFunc: func(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error) {
			usedReviewerID = reviewerID
			return &review.ReviewRecord{RecordID: "r2", ProposalID: proposalID, Verdict: review.VerdictReject, Feedback: feedback}, nil
		},
	}

	server, err := NewServer(
		&MockDecisionService{},
		&MockDecisionEngine{},
		&MockContextBuilder{},
		nil,
		&MockProposalService{},
		&MockAlertService{},
		&MockGovernanceService{},
		&MockPipelineRunner{},
		reviewSvc,
		&MockOutboxService{},
		&MockPipelineInfoService{},
		&MockExecuteService{},
		(*pgxpool.Pool)(nil),
		&MockSystemStatusService{},
		&MockObjectSearchService{},
		&MockOntologyService{},
		&MockActionSchemaService{},
		&MockSandboxService{},
	)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	// Verify the server's mcpUserID is non-empty
	if server.mcpUserID == "" {
		t.Fatal("mcpUserID must not be empty")
	}

	// Test approve_proposal handler uses server identity
	approveReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "approve_proposal",
			Arguments: map[string]interface{}{
				"proposal_id": "prop-1",
				"feedback":    "Looks good",
			},
		},
	}
	_, err = server.handleApproveProposal(nil, approveReq)
	if err != nil {
		t.Fatalf("handleApproveProposal failed: %v", err)
	}
	if usedReviewerID != server.mcpUserID {
		t.Errorf("approve_proposal should use server mcpUserID (%q), got %q", server.mcpUserID, usedReviewerID)
	}

	// Test reject_proposal handler uses server identity
	rejectReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "reject_proposal",
			Arguments: map[string]interface{}{
				"proposal_id": "prop-2",
				"feedback":    "Not needed",
			},
		},
	}
	_, err = server.handleRejectProposal(nil, rejectReq)
	if err != nil {
		t.Fatalf("handleRejectProposal failed: %v", err)
	}
	if usedReviewerID != server.mcpUserID {
		t.Errorf("reject_proposal should use server mcpUserID (%q), got %q", server.mcpUserID, usedReviewerID)
	}
}
