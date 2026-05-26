package review

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVerdict_IsValid_ValidValues(t *testing.T) {
	assert.True(t, VerdictApprove.IsValid())
	assert.True(t, VerdictReject.IsValid())
	assert.True(t, VerdictCancel.IsValid())
}

func TestVerdict_IsValid_InvalidValues(t *testing.T) {
	assert.False(t, Verdict("").IsValid())
	assert.False(t, Verdict("unknown").IsValid())
	assert.False(t, Verdict("approve ").IsValid())
	assert.False(t, Verdict("APPROVE").IsValid())
}

func TestReviewRequest_Validate_EmptyReviewerID(t *testing.T) {
	req := &ReviewRequest{
		ReviewerID: "",
		Verdict:    VerdictApprove,
	}
	err := req.Validate()
	assert.EqualError(t, err, "reviewer_id is required")
}

func TestReviewRequest_Validate_InvalidVerdict(t *testing.T) {
	req := &ReviewRequest{
		ReviewerID: "user_123",
		Verdict:    Verdict("unknown"),
	}
	err := req.Validate()
	assert.ErrorIs(t, err, ErrInvalidVerdict)
}

func TestReviewRequest_Validate_Valid(t *testing.T) {
	req := &ReviewRequest{
		ReviewerID: "user_123",
		Verdict:    VerdictApprove,
	}
	err := req.Validate()
	assert.NoError(t, err)

	req2 := &ReviewRequest{
		ReviewerID: "user_456",
		Verdict:    VerdictReject,
		Feedback:   "Not needed at this time",
	}
	err2 := req2.Validate()
	assert.NoError(t, err2)
}
