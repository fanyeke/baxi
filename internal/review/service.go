package review

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrProposalNotFound is returned when the referenced proposal does not exist.
	ErrProposalNotFound = errors.New("proposal not found")
	// ErrInvalidState is returned when the proposal is not in 'proposed' state.
	ErrInvalidState = errors.New("invalid proposal state for operation")
)

// LineageRecorder defines a minimal interface for recording lineage events in the review flow.
// This avoids importing the decision package to prevent circular dependencies.
type LineageRecorder interface {
	RecordLineageEvent(ctx context.Context, tx pgx.Tx, caseID, eventType, actor string, eventData map[string]interface{}) error
}

// ReviewServiceInterface defines the contract for review operations.
type ReviewServiceInterface interface {
	ApproveProposal(ctx context.Context, proposalID, reviewerID, feedback string) (*ReviewRecord, error)
	RejectProposal(ctx context.Context, proposalID, reviewerID, feedback string) (*ReviewRecord, error)
	CancelProposal(ctx context.Context, proposalID, reviewerID, feedback string) (*ReviewRecord, error)
	GetReviewRecord(ctx context.Context, reviewID string) (*ReviewRecord, error)
	GetProposalByID(ctx context.Context, proposalID string) (*ActionProposalRow, error)
}

// ReviewService handles the review/approval lifecycle for action proposals.
type ReviewService struct {
	repo    *ReviewRepository
	pool    *pgxpool.Pool
	lineage LineageRecorder
}

// NewReviewService creates a new ReviewService.
func NewReviewService(repo *ReviewRepository, pool *pgxpool.Pool) *ReviewService {
	return &ReviewService{
		repo: repo,
		pool: pool,
	}
}

// WithLineageRecorder attaches a LineageRecorder for automatic lineage events.
func (s *ReviewService) WithLineageRecorder(l LineageRecorder) *ReviewService {
	s.lineage = l
	return s
}

// ApproveProposal approves an action proposal with transaction-safe state transitions.
func (s *ReviewService) ApproveProposal(ctx context.Context, proposalID, reviewerID, feedback string) (*ReviewRecord, error) {
	return s.transitionProposal(ctx, proposalID, reviewerID, feedback, VerdictApprove, "approved", "proposal_approved")
}

// RejectProposal rejects an action proposal with transaction-safe state transitions.
func (s *ReviewService) RejectProposal(ctx context.Context, proposalID, reviewerID, feedback string) (*ReviewRecord, error) {
	return s.transitionProposal(ctx, proposalID, reviewerID, feedback, VerdictReject, "rejected", "proposal_rejected")
}

// CancelProposal cancels an action proposal, mapping cancel to rejected status.
func (s *ReviewService) CancelProposal(ctx context.Context, proposalID, reviewerID, feedback string) (*ReviewRecord, error) {
	return s.transitionProposal(ctx, proposalID, reviewerID, feedback, VerdictCancel, "rejected", "proposal_cancelled")
}

// GetReviewRecord retrieves a single review record by its ID.
func (s *ReviewService) GetReviewRecord(ctx context.Context, reviewID string) (*ReviewRecord, error) {
	return s.repo.GetReviewByID(ctx, s.pool, reviewID)
}

// GetProposalByID retrieves a single action proposal by its ID.
func (s *ReviewService) GetProposalByID(ctx context.Context, proposalID string) (*ActionProposalRow, error) {
	return s.repo.GetProposalByID(ctx, s.pool, proposalID)
}

// transitionProposal executes the common transaction pattern for approve/reject/cancel.
func (s *ReviewService) transitionProposal(ctx context.Context, proposalID, reviewerID, feedback string, verdict Verdict, newStatus, auditAction string) (*ReviewRecord, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. SELECT proposal FOR UPDATE (prevent race)
	proposal, err := s.selectProposalForUpdate(ctx, tx, proposalID)
	if err != nil {
		return nil, err
	}
	if proposal == nil {
		return nil, ErrProposalNotFound
	}

	// 2. Verify proposal exists and apply_status='proposed'
	if proposal.ApplyStatus != "proposed" {
		return nil, fmt.Errorf("%w: expected apply_status='proposed', got %q", ErrInvalidState, proposal.ApplyStatus)
	}

	// 3. UPDATE proposal SET apply_status=...
	if err := s.repo.UpdateProposalApplyStatus(ctx, tx, proposalID, newStatus); err != nil {
		return nil, fmt.Errorf("update proposal status: %w", err)
	}

	// 4. INSERT review_record
	record := &ReviewRecord{
		RecordID:   generateReviewID(),
		ProposalID: proposalID,
		ReviewerID: reviewerID,
		Verdict:    verdict,
		Feedback:   feedback,
		CreatedAt:  time.Now().UTC(),
	}
	if err := s.repo.InsertReview(ctx, tx, record); err != nil {
		return nil, fmt.Errorf("insert review record: %w", err)
	}

	// 5. Record lineage event (Phase 5: automatic lineage)
	if s.lineage != nil {
		caseID := ""
		if proposal != nil {
			caseID = proposal.CaseID
		}
		_ = s.lineage.RecordLineageEvent(ctx, tx, caseID, auditAction, reviewerID, map[string]interface{}{
			"proposal_id":  proposalID,
			"action_type":  proposal.ActionType,
			"verdict":      string(verdict),
			"feedback":     feedback,
		})
	}

	// 6. UPDATE case status if 'proposal_generated' -> 'review_required'
	if err := s.updateCaseStatusIfNeeded(ctx, tx, proposal.CaseID); err != nil {
		return nil, err
	}

	// 7. INSERT audit_log
	metadata := map[string]interface{}{
		"verdict":  string(verdict),
		"feedback": feedback,
	}
	if err := s.insertAuditLog(ctx, tx, "review", auditAction, reviewerID, "action_proposal", proposalID, metadata); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return record, nil
}

// selectProposalForUpdate retrieves a proposal row with SELECT ... FOR UPDATE.
func (s *ReviewService) selectProposalForUpdate(ctx context.Context, tx pgx.Tx, proposalID string) (*ActionProposalRow, error) {
	query := `
		SELECT proposal_id, case_id, decision_id, action_type,
		       payload, apply_status, created_at,
		       applied_at, applied_by,
		       title, description, risk_level, requires_human_review
		FROM ai.action_proposal
		WHERE proposal_id = $1
		FOR UPDATE
	`

	var row ActionProposalRow
	err := tx.QueryRow(ctx, query, proposalID).Scan(
		&row.ProposalID,
		&row.CaseID,
		&row.DecisionID,
		&row.ActionType,
		&row.Payload,
		&row.ApplyStatus,
		&row.CreatedAt,
		&row.AppliedAt,
		&row.AppliedBy,
		&row.Title,
		&row.Description,
		&row.RiskLevel,
		&row.RequiresHumanReview,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("select proposal for update: %w", err)
	}
	return &row, nil
}

// updateCaseStatusIfNeeded updates the case status from 'proposal_generated' to 'review_required'.
func (s *ReviewService) updateCaseStatusIfNeeded(ctx context.Context, tx pgx.Tx, caseID string) error {
	if caseID == "" {
		return nil
	}

	var status string
	err := tx.QueryRow(ctx, `SELECT status FROM ai.decision_case WHERE case_id = $1`, caseID).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("select case status: %w", err)
	}
	if status == "proposal_generated" {
		if err := s.repo.UpdateCaseStatus(ctx, tx, caseID, "review_required"); err != nil {
			return fmt.Errorf("update case status: %w", err)
		}
	}
	return nil
}

// insertAuditLog inserts a row into audit.audit_log.
func (s *ReviewService) insertAuditLog(ctx context.Context, tx pgx.Tx, category, action, actor, resourceType, resourceID string, metadata map[string]interface{}) error {
	query := `
		INSERT INTO audit.audit_log (category, action, actor, resource_type, resource_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal audit metadata: %w", err)
	}

	_, err = tx.Exec(ctx, query, category, action, actor, resourceType, resourceID, metadataJSON)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

// generateReviewID generates a unique review record identifier.
func generateReviewID() string {
	return "rev_" + uuid.NewString()
}
