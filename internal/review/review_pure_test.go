package review

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewReviewRepository(t *testing.T) {
	repo := NewReviewRepository()
	assert.NotNil(t, repo)
}

func TestNewReviewService(t *testing.T) {
	svc := NewReviewService(nil, nil)
	assert.NotNil(t, svc)
}

func TestReviewService_WithLineageRecorder(t *testing.T) {
	svc := NewReviewService(nil, nil)
	result := svc.WithLineageRecorder(nil)
	assert.NotNil(t, result)
	assert.Equal(t, svc, result)
}

func TestGenerateReviewID_Prefix(t *testing.T) {
	id := generateReviewID()
	assert.Regexp(t, `^rev_`, id)
}

func TestGenerateReviewID_Uniqueness(t *testing.T) {
	ids := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		id := generateReviewID()
		assert.False(t, ids[id], "duplicate review ID: %s", id)
		ids[id] = true
	}
}

func TestVerdict_Constants(t *testing.T) {
	assert.Equal(t, Verdict("approve"), VerdictApprove)
	assert.Equal(t, Verdict("reject"), VerdictReject)
	assert.Equal(t, Verdict("cancel"), VerdictCancel)
}

func TestReviewRecord_Structure(t *testing.T) {
	record := &ReviewRecord{
		RecordID:   "rev-001",
		ProposalID: "prop-001",
		ReviewerID: "user-001",
		Verdict:    VerdictApprove,
		Feedback:   "Looks good",
	}
	assert.Equal(t, "rev-001", record.RecordID)
	assert.Equal(t, "prop-001", record.ProposalID)
	assert.Equal(t, "user-001", record.ReviewerID)
	assert.Equal(t, VerdictApprove, record.Verdict)
	assert.Equal(t, "Looks good", record.Feedback)
}

func TestReviewRequest_AllValidVerdicts(t *testing.T) {
	tests := []struct {
		name    string
		verdict Verdict
	}{
		{"approve", VerdictApprove},
		{"reject", VerdictReject},
		{"cancel", VerdictCancel},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &ReviewRequest{
				ReviewerID: "user-001",
				Verdict:    tc.verdict,
			}
			assert.NoError(t, req.Validate())
		})
	}
}

func TestReviewRequest_EmptyReviewerAndInvalidVerdict(t *testing.T) {
	req := &ReviewRequest{}
	err := req.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reviewer_id is required")
}

func TestActionProposalRow_Structure(t *testing.T) {
	row := &ActionProposalRow{
		ProposalID:  "prop-001",
		CaseID:      "case-001",
		ActionType:  "notify_owner",
		ApplyStatus: "proposed",
		Title:       "Test Proposal",
	}
	assert.Equal(t, "prop-001", row.ProposalID)
	assert.Equal(t, "case-001", row.CaseID)
	assert.Equal(t, "notify_owner", row.ActionType)
	assert.Equal(t, "proposed", row.ApplyStatus)
	assert.Equal(t, "Test Proposal", row.Title)
}

func TestSandbox_Structure(t *testing.T) {
	sb := &Sandbox{
		SandboxID:  "sbx-001",
		CaseID:     "case-001",
		Status:     "draft",
		ComparedWith: []string{},
	}
	assert.Equal(t, "sbx-001", sb.SandboxID)
	assert.Equal(t, "case-001", sb.CaseID)
	assert.Equal(t, "draft", sb.Status)
	assert.Empty(t, sb.ComparedWith)
}

func TestComparisonResult_Structure(t *testing.T) {
	cr := &ComparisonResult{
		Sandbox1ID:  "sbx-001",
		Sandbox2ID:  "sbx-002",
		Differences: []Difference{{Field: "score", Value1: 90, Value2: 85}},
	}
	assert.Equal(t, "sbx-001", cr.Sandbox1ID)
	assert.Equal(t, "sbx-002", cr.Sandbox2ID)
	assert.Len(t, cr.Differences, 1)
	assert.Equal(t, "score", cr.Differences[0].Field)
}

func TestReviewServiceInterface(t *testing.T) {
	// Verify that ReviewService implements ReviewServiceInterface
	var svc ReviewServiceInterface = &ReviewService{}
	_ = svc // compile-time check
}

func TestErrProposalNotFound_Error(t *testing.T) {
	assert.Equal(t, "proposal not found", ErrProposalNotFound.Error())
}

func TestErrInvalidState_Error(t *testing.T) {
	assert.Contains(t, ErrInvalidState.Error(), "invalid proposal state")
}

func TestErrInvalidVerdict_Error(t *testing.T) {
	assert.Contains(t, ErrInvalidVerdict.Error(), "invalid verdict")
}
