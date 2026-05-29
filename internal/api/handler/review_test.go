package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/review"
)

type mockReviewService struct {
	approveFn   func(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error)
	rejectFn    func(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error)
	cancelFn    func(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error)
	getReviewFn func(ctx context.Context, proposalID string) (*review.ReviewRecord, error)
}

func (m *mockReviewService) ApproveProposal(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error) {
	if m.approveFn != nil {
		return m.approveFn(ctx, proposalID, reviewerID, feedback)
	}
	return nil, errors.New("unexpected call to ApproveProposal")
}

func (m *mockReviewService) RejectProposal(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error) {
	if m.rejectFn != nil {
		return m.rejectFn(ctx, proposalID, reviewerID, feedback)
	}
	return nil, errors.New("unexpected call to RejectProposal")
}

func (m *mockReviewService) CancelProposal(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error) {
	if m.cancelFn != nil {
		return m.cancelFn(ctx, proposalID, reviewerID, feedback)
	}
	return nil, errors.New("unexpected call to CancelProposal")
}

func (m *mockReviewService) GetReviewByProposal(ctx context.Context, proposalID string) (*review.ReviewRecord, error) {
	if m.getReviewFn != nil {
		return m.getReviewFn(ctx, proposalID)
	}
	return nil, errors.New("unexpected call to GetReviewByProposal")
}

func nowPtr() *time.Time {
	t := time.Now().UTC()
	return &t
}

func TestReviewHandler_HandleApprove_200(t *testing.T) {
	now := time.Now().UTC()
	svc := &mockReviewService{
		approveFn: func(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error) {
			assert.Equal(t, "prop-1", proposalID)
			assert.Equal(t, "reviewer-42", reviewerID)
			assert.Equal(t, "Looks good", feedback)
			return &review.ReviewRecord{
				RecordID:   "rev_abc",
				ProposalID: "prop-1",
				ReviewerID: "reviewer-42",
				Verdict:    review.VerdictApprove,
				Feedback:   "Looks good",
				CreatedAt:  now,
			}, nil
		},
	}
	h := NewReviewHandler(svc)

	body := `{"reviewer_id":"reviewer-42","feedback":"Looks good"}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/proposals/prop-1/approve", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("id", "prop-1")
	w := httptest.NewRecorder()
	h.HandleApprove(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp reviewResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "rev_abc", resp.RecordID)
	assert.Equal(t, "prop-1", resp.ProposalID)
	assert.Equal(t, review.VerdictApprove, resp.Verdict)
	assert.Equal(t, "reviewer-42", resp.ReviewerID)
	assert.Equal(t, "Looks good", resp.Feedback)
	assert.Equal(t, now.Format(time.RFC3339), resp.CreatedAt)
}

func TestReviewHandler_HandleApprove_400_MissingReviewerID(t *testing.T) {
	svc := &mockReviewService{}
	h := NewReviewHandler(svc)

	body := `{"feedback":"Looks good"}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/proposals/prop-1/approve", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("id", "prop-1")
	w := httptest.NewRecorder()
	h.HandleApprove(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "reviewer_id is required", resp["message"])
}

func TestReviewHandler_HandleApprove_400_InvalidJSON(t *testing.T) {
	svc := &mockReviewService{}
	h := NewReviewHandler(svc)

	r := httptest.NewRequest(http.MethodPost, "/api/v1/proposals/prop-1/approve", strings.NewReader("not json"))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("id", "prop-1")
	w := httptest.NewRecorder()
	h.HandleApprove(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["message"].(string), "invalid request body")
}

func TestReviewHandler_HandleApprove_404_ProposalNotFound(t *testing.T) {
	svc := &mockReviewService{
		approveFn: func(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error) {
			return nil, fmt.Errorf("%w: proposal prop-999 not found", review.ErrProposalNotFound)
		},
	}
	h := NewReviewHandler(svc)

	body := `{"reviewer_id":"reviewer-42"}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/proposals/prop-999/approve", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("id", "prop-999")
	w := httptest.NewRecorder()
	h.HandleApprove(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["message"].(string), "proposal not found")
}

func TestReviewHandler_HandleApprove_409_InvalidState(t *testing.T) {
	svc := &mockReviewService{
		approveFn: func(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error) {
			return nil, fmt.Errorf("%w: expected apply_status='proposed', got 'approved'", review.ErrInvalidState)
		},
	}
	h := NewReviewHandler(svc)

	body := `{"reviewer_id":"reviewer-42"}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/proposals/prop-1/approve", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("id", "prop-1")
	w := httptest.NewRecorder()
	h.HandleApprove(w, r)

	require.Equal(t, http.StatusConflict, w.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["message"].(string), "invalid proposal state for operation")
}

func TestReviewHandler_HandleReject_200(t *testing.T) {
	now := time.Now().UTC()
	svc := &mockReviewService{
		rejectFn: func(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error) {
			assert.Equal(t, "prop-1", proposalID)
			assert.Equal(t, "reviewer-42", reviewerID)
			assert.Equal(t, "Not acceptable", feedback)
			return &review.ReviewRecord{
				RecordID:   "rev_def",
				ProposalID: "prop-1",
				ReviewerID: "reviewer-42",
				Verdict:    review.VerdictReject,
				Feedback:   "Not acceptable",
				CreatedAt:  now,
			}, nil
		},
	}
	h := NewReviewHandler(svc)

	body := `{"reviewer_id":"reviewer-42","feedback":"Not acceptable"}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/proposals/prop-1/reject", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("id", "prop-1")
	w := httptest.NewRecorder()
	h.HandleReject(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp reviewResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "rev_def", resp.RecordID)
	assert.Equal(t, "prop-1", resp.ProposalID)
	assert.Equal(t, review.VerdictReject, resp.Verdict)
	assert.Equal(t, "Not acceptable", resp.Feedback)
}

func TestReviewHandler_HandleCancel_200(t *testing.T) {
	now := time.Now().UTC()
	svc := &mockReviewService{
		cancelFn: func(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error) {
			assert.Equal(t, "prop-1", proposalID)
			assert.Equal(t, "reviewer-42", reviewerID)
			return &review.ReviewRecord{
				RecordID:   "rev_ghi",
				ProposalID: "prop-1",
				ReviewerID: "reviewer-42",
				Verdict:    review.VerdictCancel,
				CreatedAt:  now,
			}, nil
		},
	}
	h := NewReviewHandler(svc)

	body := `{"reviewer_id":"reviewer-42"}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/proposals/prop-1/cancel", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("id", "prop-1")
	w := httptest.NewRecorder()
	h.HandleCancel(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp reviewResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "rev_ghi", resp.RecordID)
	assert.Equal(t, "prop-1", resp.ProposalID)
	assert.Equal(t, review.VerdictCancel, resp.Verdict)
	assert.Equal(t, "reviewer-42", resp.ReviewerID)
}

func TestReviewHandler_HandleGetReview_200(t *testing.T) {
	now := time.Now().UTC()
	svc := &mockReviewService{
		getReviewFn: func(ctx context.Context, proposalID string) (*review.ReviewRecord, error) {
			assert.Equal(t, "prop-1", proposalID)
			return &review.ReviewRecord{
				RecordID:   "rev_abc",
				ProposalID: "prop-1",
				ReviewerID: "reviewer-42",
				Verdict:    review.VerdictApprove,
				Feedback:   "Approved",
				CreatedAt:  now,
			}, nil
		},
	}
	h := NewReviewHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/proposals/prop-1/review", nil)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("id", "prop-1")
	w := httptest.NewRecorder()
	h.HandleGetReview(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp reviewResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "rev_abc", resp.RecordID)
	assert.Equal(t, "prop-1", resp.ProposalID)
	assert.Equal(t, review.VerdictApprove, resp.Verdict)
	assert.Equal(t, "reviewer-42", resp.ReviewerID)
	assert.Equal(t, "Approved", resp.Feedback)
	assert.Equal(t, now.Format(time.RFC3339), resp.CreatedAt)
}

func TestReviewHandler_HandleGetReview_404(t *testing.T) {
	svc := &mockReviewService{
		getReviewFn: func(ctx context.Context, proposalID string) (*review.ReviewRecord, error) {
			return nil, nil
		},
	}
	h := NewReviewHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/proposals/prop-999/review", nil)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("id", "prop-999")
	w := httptest.NewRecorder()
	h.HandleGetReview(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["message"].(string), "review not found")
}

func TestReviewHandler_HandleApprove_500_InternalError(t *testing.T) {
	svc := &mockReviewService{
		approveFn: func(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error) {
			return nil, errors.New("unexpected database error")
		},
	}
	h := NewReviewHandler(svc)

	body := `{"reviewer_id":"reviewer-42"}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/proposals/prop-1/approve", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("id", "prop-1")
	w := httptest.NewRecorder()
	h.HandleApprove(w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["message"].(string), "internal server error")
}

func TestReviewHandler_HandleGetReview_500_InternalError(t *testing.T) {
	svc := &mockReviewService{
		getReviewFn: func(ctx context.Context, proposalID string) (*review.ReviewRecord, error) {
			return nil, errors.New("unexpected database error")
		},
	}
	h := NewReviewHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/proposals/prop-1/review", nil)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("id", "prop-1")
	w := httptest.NewRecorder()
	h.HandleGetReview(w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["message"].(string), "internal server error")
}
