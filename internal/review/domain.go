package review

import (
	"encoding/json"
	"errors"
	"time"
)

// Verdict represents the outcome of a review.
type Verdict string

const (
	VerdictApprove Verdict = "approve"
	VerdictReject  Verdict = "reject"
	VerdictCancel  Verdict = "cancel"
)

// ReviewRecord represents a persisted review outcome for a proposal.
type ReviewRecord struct {
	RecordID   string          `json:"record_id"`
	ProposalID string          `json:"proposal_id"`
	ReviewerID string          `json:"reviewer_id"`
	Verdict    Verdict         `json:"verdict"`
	Feedback   string          `json:"feedback,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	ReviewedAt *time.Time      `json:"reviewed_at,omitempty"`
}

// ReviewRequest is the API/CLI input for performing a review.
type ReviewRequest struct {
	ReviewerID string  `json:"reviewer_id"`
	Verdict    Verdict `json:"verdict"`
	Feedback   string  `json:"feedback,omitempty"`
}

// ErrInvalidVerdict is returned when the verdict is not one of the allowed values.
var ErrInvalidVerdict = errors.New("invalid verdict: must be approve, reject, or cancel")

// IsValid returns true if the verdict is one of the allowed values.
func (v Verdict) IsValid() bool {
	switch v {
	case VerdictApprove, VerdictReject, VerdictCancel:
		return true
	default:
		return false
	}
}

// Validate checks that the review request has all required fields.
func (r *ReviewRequest) Validate() error {
	if r.ReviewerID == "" {
		return errors.New("reviewer_id is required")
	}
	if !r.Verdict.IsValid() {
		return ErrInvalidVerdict
	}
	return nil
}
