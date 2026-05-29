package mcp

import (
	"context"

	"baxi/internal/action"
	"baxi/internal/decision"
	"baxi/internal/llm"
	"baxi/internal/model"
	"baxi/internal/review"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DecisionService defines the interface for decision case operations.
type DecisionService interface {
	CreateCaseFromAlert(ctx context.Context, alertID, createdBy string) (*decision.DecisionCase, error)
	GetCase(ctx context.Context, caseID string) (*decision.DecisionCase, error)
	ListCases(ctx context.Context, filter decision.CaseFilter) (*decision.CaseList, error)
}

// DecisionEngine defines the interface for generating decisions.
type DecisionEngine interface {
	GenerateDecision(ctx context.Context, caseID string, context *decision.DecisionContext) (*llm.DecisionOutput, error)
}

// ContextBuilder defines the interface for building decision contexts.
type ContextBuilder interface {
	BuildDecisionContext(ctx context.Context, caseID string) (*decision.DecisionContext, error)
}

// ProposalService defines the interface for managing action proposals.
type ProposalService interface {
	ListProposals(ctx context.Context, caseID string) ([]action.ActionProposal, error)
}

// AlertService defines the interface for alert operations.
type AlertService interface {
	ListAlerts(ctx context.Context, filters model.AlertFilters, sort string, limit, offset int) (*model.AlertListResponse, error)
}

// GovernanceService defines the interface for governance operations.
type GovernanceService interface {
	CheckAccess(ctx context.Context, role, objectType, action string) (*model.AccessDecision, error)
	GetClassification(ctx context.Context, fieldPath string) (*model.ClassificationResponse, error)
}

// ReviewService defines the interface for review operations.
type ReviewService interface {
	ApproveProposal(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error)
	RejectProposal(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error)
}

// ExecuteService defines the interface for executing action proposals.
type ExecuteService interface {
	ExecuteProposal(ctx context.Context, pool *pgxpool.Pool, proposalID string, actorID string, opts ...action.ExecuteOption) (*action.ExecutionResult, error)
}

// PipelineRunner defines the interface for running pipelines.
type PipelineRunner interface {
	Run(ctx context.Context, config string) (string, error)
}

// SystemStatusService defines the interface for getting system status.
type SystemStatusService interface {
	GetStatus(ctx context.Context) (*model.SystemStatus, error)
}

// ObjectSearchService defines the interface for searching objects.
type ObjectSearchService interface {
	SearchObjects(ctx context.Context, objectType, query string, limit, offset int) (*model.SearchResult, error)
}

// OutboxService defines the interface for outbox event operations.
type OutboxService interface {
	ListOutboxEvents(ctx context.Context, status string, limit, offset int) ([]model.OutboxEvent, int, error)
}

// PipelineInfoService defines the interface for pipeline status information.
type PipelineInfoService interface {
	GetLastRunStatus(ctx context.Context) (*model.PipelineRun, error)
	ListRuns(ctx context.Context, limit int) ([]model.PipelineRun, error)
}
