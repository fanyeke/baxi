// DEPRECATED: Use baxi/internal/repository/decision instead.
// This file is a compatibility layer during migration.

package repository

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"

	"baxi/internal/repository/common"
	decisionRepo "baxi/internal/repository/decision"
)

// RawMessage is an alias for json.RawMessage for backward compatibility.
type RawMessage = json.RawMessage

// DecisionCaseRow represents a single row from ai.decision_case.
// DEPRECATED: Use decision.DecisionCaseRow instead.
type DecisionCaseRow = decisionRepo.DecisionCaseRow

// LLMDecisionRow represents a single row from ai.llm_decision.
// DEPRECATED: Use decision.LLMDecisionRow instead.
type LLMDecisionRow = decisionRepo.LLMDecisionRow

// ActionProposalRow represents a single row from ai.action_proposal.
// DEPRECATED: Use decision.ActionProposalRow instead.
type ActionProposalRow = decisionRepo.ActionProposalRow

// CaseFilter holds optional WHERE clause filters for listing decision cases.
// DEPRECATED: Use decision.CaseFilter instead.
type CaseFilter = decisionRepo.CaseFilter

// DecisionRepository provides data access for decision domain (DEPRECATED).
// Use decision.Repository instead for new code.
type DecisionRepository struct {
	inner *decisionRepo.Repository
}

// NewDecisionRepository creates a new DecisionRepository (DEPRECATED).
func NewDecisionRepository() *DecisionRepository {
	return &DecisionRepository{}
}

// SetPool initializes the inner repository with a pool provider.
func (r *DecisionRepository) SetPool(pool *pgxpool.Pool) {
	r.inner = decisionRepo.NewRepository(common.NewPoolProvider(pool))
}

// ensureInitialized lazily initializes the inner repo if needed.
func (r *DecisionRepository) ensureInitialized(pool *pgxpool.Pool) *decisionRepo.Repository {
	if r.inner == nil {
		r.SetPool(pool)
	}
	return r.inner
}

// CreateCase inserts a new row into ai.decision_case (DEPRECATED).
func (r *DecisionRepository) CreateCase(ctx context.Context, pool *pgxpool.Pool, row *DecisionCaseRow) error {
	return r.ensureInitialized(pool).CreateCase(ctx, row)
}

// GetCaseByID retrieves a single decision case by its case_id (DEPRECATED).
func (r *DecisionRepository) GetCaseByID(ctx context.Context, pool *pgxpool.Pool, caseID string) (*DecisionCaseRow, error) {
	return r.ensureInitialized(pool).GetCaseByID(ctx, caseID)
}

// GetCaseBySource retrieves a single decision case by source_type and source_id (DEPRECATED).
// If sourceType or sourceID is nil, matches NULL in the database.
func (r *DecisionRepository) GetCaseBySource(ctx context.Context, pool *pgxpool.Pool, sourceType, sourceID string) (*DecisionCaseRow, error) {
	return r.ensureInitialized(pool).GetCaseBySource(ctx, sourceType, sourceID)
}

// UpdateCaseStatus updates the status and related metadata for a decision case (DEPRECATED).
func (r *DecisionRepository) UpdateCaseStatus(
	ctx context.Context,
	pool *pgxpool.Pool,
	caseID string,
	status string,
	contextJSON *RawMessage,
	contextHash *string,
	governanceSnapshot *RawMessage,
) error {
	return r.ensureInitialized(pool).UpdateCaseStatus(ctx, caseID, status, contextJSON, contextHash, governanceSnapshot)
}

// ListCases queries ai.decision_case with optional filters and pagination (DEPRECATED).
// Uses COUNT(*) OVER() to return the total count in a single query.
// Results are ordered by created_at DESC.
func (r *DecisionRepository) ListCases(
	ctx context.Context,
	pool *pgxpool.Pool,
	filter CaseFilter,
) ([]DecisionCaseRow, int, error) {
	return r.ensureInitialized(pool).ListCases(ctx, filter)
}

// CreateDecision inserts a new row into ai.llm_decision (DEPRECATED).
func (r *DecisionRepository) CreateDecision(ctx context.Context, pool *pgxpool.Pool, row *LLMDecisionRow) error {
	return r.ensureInitialized(pool).CreateDecision(ctx, row)
}

// CreateProposal inserts a new row into ai.action_proposal (DEPRECATED).
func (r *DecisionRepository) CreateProposal(ctx context.Context, pool *pgxpool.Pool, row *ActionProposalRow) error {
	return r.ensureInitialized(pool).CreateProposal(ctx, row)
}

// ListProposalsByCase retrieves all action proposals for a given case (DEPRECATED).
// Results are ordered by created_at ASC.
func (r *DecisionRepository) ListProposalsByCase(ctx context.Context, pool *pgxpool.Pool, caseID string) ([]ActionProposalRow, error) {
	return r.ensureInitialized(pool).ListProposalsByCase(ctx, caseID)
}
