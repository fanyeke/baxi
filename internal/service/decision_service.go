package service

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"baxi/internal/action"
	"baxi/internal/decision"
	"baxi/internal/llm"
)

// DecisionService composes case, context, engine, and proposal services into a
// single business orchestration layer for the decision workflow.
type DecisionService struct {
	caseSvc     CaseService
	ctxBuilder  ContextBuilder
	engine      DecisionEngine
	proposalSvc ProposalService
	pool        *pgxpool.Pool
}

// CaseService defines the decision case operations needed by DecisionService.
type CaseService interface {
	CreateCaseFromAlert(ctx context.Context, alertID, createdBy string) (*decision.DecisionCase, error)
	GetCase(ctx context.Context, caseID string) (*decision.DecisionCase, error)
	ListCases(ctx context.Context, filter decision.CaseFilter) (*decision.CaseList, error)
}

// ContextBuilder defines the decision context building operation.
type ContextBuilder interface {
	BuildDecisionContext(ctx context.Context, caseID string) (*decision.DecisionContext, error)
}

// DecisionEngine defines the decision generation operation.
type DecisionEngine interface {
	GenerateDecision(ctx context.Context, caseID string, context *decision.DecisionContext) (*llm.DecisionOutput, error)
}

// ProposalService defines the action proposal operations needed by DecisionService.
type ProposalService interface {
	GenerateProposals(ctx context.Context, caseID, decisionID string, dec *llm.DecisionOutput) ([]action.ActionProposal, error)
	ListProposals(ctx context.Context, caseID string) ([]action.ActionProposal, error)
}

var (
	_ CaseService     = (*decision.CaseService)(nil)
	_ ContextBuilder  = (*decision.ContextBuilder)(nil)
	_ DecisionEngine  = (*decision.DecisionEngine)(nil)
	_ ProposalService = (*action.ProposalService)(nil)
)

// NewDecisionService creates a new DecisionService.
func NewDecisionService(
	caseSvc CaseService,
	ctxBuilder ContextBuilder,
	engine DecisionEngine,
	proposalSvc ProposalService,
	pool *pgxpool.Pool,
) *DecisionService {
	return &DecisionService{
		caseSvc:     caseSvc,
		ctxBuilder:  ctxBuilder,
		engine:      engine,
		proposalSvc: proposalSvc,
		pool:        pool,
	}
}

// CreateCaseFromAlert delegates to caseSvc.CreateCaseFromAlert and returns the case.
func (s *DecisionService) CreateCaseFromAlert(ctx context.Context, alertID, createdBy string) (*decision.DecisionCase, error) {
	return s.caseSvc.CreateCaseFromAlert(ctx, alertID, createdBy)
}

// BuildContext delegates to ctxBuilder.BuildDecisionContext and returns the context.
func (s *DecisionService) BuildContext(ctx context.Context, caseID string) (*decision.DecisionContext, error) {
	return s.ctxBuilder.BuildDecisionContext(ctx, caseID)
}

// Decide orchestrates the full decision workflow: get case, build context,
// generate decision, and create action proposals.
func (s *DecisionService) Decide(ctx context.Context, caseID string) (*decision.DecisionContext, *llm.DecisionOutput, []action.ActionProposal, error) {
	_, err := s.caseSvc.GetCase(ctx, caseID)
	if err != nil {
		return nil, nil, nil, err
	}

	decCtx, err := s.ctxBuilder.BuildDecisionContext(ctx, caseID)
	if err != nil {
		return nil, nil, nil, err
	}

	output, err := s.engine.GenerateDecision(ctx, caseID, decCtx)
	if err != nil {
		return nil, nil, nil, err
	}

	decisionID := decision.GenerateDecisionID()
	proposals, err := s.proposalSvc.GenerateProposals(ctx, caseID, decisionID, output)
	if err != nil {
		return nil, nil, nil, err
	}

	return decCtx, output, proposals, nil
}

// GetCase delegates to caseSvc.GetCase.
func (s *DecisionService) GetCase(ctx context.Context, caseID string) (*decision.DecisionCase, error) {
	return s.caseSvc.GetCase(ctx, caseID)
}

// ListCases delegates to caseSvc.ListCases.
func (s *DecisionService) ListCases(ctx context.Context, filter decision.CaseFilter) (*decision.CaseList, error) {
	return s.caseSvc.ListCases(ctx, filter)
}

// ListProposals delegates to proposalSvc.ListProposals.
func (s *DecisionService) ListProposals(ctx context.Context, caseID string) ([]action.ActionProposal, error) {
	return s.proposalSvc.ListProposals(ctx, caseID)
}
