package review

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ActionProposalRow represents a single row from ai.action_proposal.
// Defined locally to avoid circular dependency with internal/repository.
type ActionProposalRow struct {
	ProposalID          string
	CaseID              string
	DecisionID          *string
	ActionType          string
	Payload             *json.RawMessage
	ApplyStatus         string
	CreatedAt           time.Time
	AppliedAt           *time.Time
	AppliedBy           *string
	Title               string
	Description         *string
	RiskLevel           *string
	RequiresHumanReview bool
}

// ReviewRepository provides data access for ai.review_record,
// ai.action_proposal, and ai.decision_case tables.
type ReviewRepository struct{}

// NewReviewRepository creates a new ReviewRepository.
func NewReviewRepository() *ReviewRepository {
	return &ReviewRepository{}
}

// InsertReview inserts a new row into ai.review_record.
func (r *ReviewRepository) InsertReview(ctx context.Context, tx pgx.Tx, record *ReviewRecord) error {
	query := `
		INSERT INTO ai.review_record (
			review_id, proposal_id, reviewer_id, verdict, feedback, reviewed_at
		) VALUES (
			$1, $2, $3, $4, $5, $6
		)
	`

	_, err := tx.Exec(ctx, query,
		record.RecordID,
		record.ProposalID,
		record.ReviewerID,
		record.Verdict,
		record.Feedback,
		record.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert ai.review_record: %w", err)
	}
	return nil
}

// GetReviewByID retrieves a single review record by its review_id.
func (r *ReviewRepository) GetReviewByID(ctx context.Context, pool *pgxpool.Pool, reviewID string) (*ReviewRecord, error) {
	query := `
		SELECT review_id, proposal_id, reviewer_id, verdict, feedback, reviewed_at
		FROM ai.review_record
		WHERE review_id = $1
	`

	var record ReviewRecord
	err := pool.QueryRow(ctx, query, reviewID).Scan(
		&record.RecordID,
		&record.ProposalID,
		&record.ReviewerID,
		&record.Verdict,
		&record.Feedback,
		&record.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query ai.review_record by id: %w", err)
	}
	return &record, nil
}

// GetReviewsByProposal retrieves all review records for a given proposal_id.
// Results are ordered by reviewed_at DESC (most recent first).
func (r *ReviewRepository) GetReviewsByProposal(ctx context.Context, pool *pgxpool.Pool, proposalID string) ([]ReviewRecord, error) {
	query := `
		SELECT review_id, proposal_id, reviewer_id, verdict, feedback, reviewed_at
		FROM ai.review_record
		WHERE proposal_id = $1
		ORDER BY reviewed_at DESC
	`

	rows, err := pool.Query(ctx, query, proposalID)
	if err != nil {
		return nil, fmt.Errorf("query ai.review_record by proposal: %w", err)
	}
	defer rows.Close()

	var results []ReviewRecord
	for rows.Next() {
		var record ReviewRecord
		if err := rows.Scan(
			&record.RecordID,
			&record.ProposalID,
			&record.ReviewerID,
			&record.Verdict,
			&record.Feedback,
			&record.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan ai.review_record row: %w", err)
		}
		results = append(results, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ai.review_record rows: %w", err)
	}

	if results == nil {
		results = []ReviewRecord{}
	}

	return results, nil
}

// ListReviewRecords retrieves review records for a given proposal_id with pagination.
func (r *ReviewRepository) ListReviewRecords(ctx context.Context, pool *pgxpool.Pool, proposalID string, limit, offset int) ([]ReviewRecord, int, error) {
	// Count query
	var total int
	err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM ai.review_record WHERE proposal_id = $1`, proposalID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count ai.review_record: %w", err)
	}

	// Data query with LIMIT/OFFSET
	query := `
		SELECT review_id, proposal_id, reviewer_id, verdict, feedback, reviewed_at
		FROM ai.review_record
		WHERE proposal_id = $1
		ORDER BY reviewed_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := pool.Query(ctx, query, proposalID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query ai.review_record: %w", err)
	}
	defer rows.Close()

	var results []ReviewRecord
	for rows.Next() {
		var record ReviewRecord
		if err := rows.Scan(
			&record.RecordID,
			&record.ProposalID,
			&record.ReviewerID,
			&record.Verdict,
			&record.Feedback,
			&record.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan ai.review_record row: %w", err)
		}
		results = append(results, record)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate ai.review_record rows: %w", err)
	}

	if results == nil {
		results = []ReviewRecord{}
	}

	return results, total, nil
}

// UpdateProposalApplyStatus updates the apply_status of an action proposal.
func (r *ReviewRepository) UpdateProposalApplyStatus(ctx context.Context, tx pgx.Tx, proposalID string, status string) error {
	query := `
		UPDATE ai.action_proposal
		SET apply_status = $1
		WHERE proposal_id = $2
	`

	res, err := tx.Exec(ctx, query, status, proposalID)
	if err != nil {
		return fmt.Errorf("update ai.action_proposal status: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("action proposal %s not found", proposalID)
	}
	return nil
}

// GetProposalByID retrieves a single action proposal by its proposal_id.
func (r *ReviewRepository) GetProposalByID(ctx context.Context, pool *pgxpool.Pool, proposalID string) (*ActionProposalRow, error) {
	query := `
		SELECT proposal_id, case_id, decision_id, action_type,
		       payload, apply_status, created_at,
		       applied_at, applied_by,
		       title, description, risk_level, requires_human_review
		FROM ai.action_proposal
		WHERE proposal_id = $1
	`

	var row ActionProposalRow
	err := pool.QueryRow(ctx, query, proposalID).Scan(
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
		return nil, fmt.Errorf("query ai.action_proposal by id: %w", err)
	}
	return &row, nil
}

// UpdateCaseStatus updates the status of a decision case with updated_at = NOW().
func (r *ReviewRepository) UpdateCaseStatus(ctx context.Context, tx pgx.Tx, caseID string, status string) error {
	query := `
		UPDATE ai.decision_case
		SET status = $1,
		    updated_at = NOW()
		WHERE case_id = $2
	`

	res, err := tx.Exec(ctx, query, status, caseID)
	if err != nil {
		return fmt.Errorf("update ai.decision_case status: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("decision case %s not found", caseID)
	}
	return nil
}
