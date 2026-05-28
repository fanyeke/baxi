package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"baxi/internal/httputil"
	"baxi/internal/review"
)

// ReviewService defines the business operations needed by ReviewHandler.
// Tests substitute a mock without importing the service package.
type ReviewService interface {
	ApproveProposal(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error)
	RejectProposal(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error)
	CancelProposal(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error)
	GetReviewByProposal(ctx context.Context, proposalID string) (*review.ReviewRecord, error)
}

// ReviewHandler handles HTTP requests for proposal review endpoints.
type ReviewHandler struct {
	svc ReviewService
}

// NewReviewHandler creates a new ReviewHandler.
func NewReviewHandler(svc ReviewService) *ReviewHandler {
	return &ReviewHandler{svc: svc}
}

// approveRequest is the request body for approve/reject/cancel endpoints.
type approveRequest struct {
	ReviewerID string `json:"reviewer_id"`
	Feedback   string `json:"feedback"`
}

// reviewResponse is the response body for review operations.
type reviewResponse struct {
	RecordID   string         `json:"record_id"`
	ProposalID string         `json:"proposal_id"`
	Verdict    review.Verdict `json:"verdict"`
	ReviewerID string         `json:"reviewer_id"`
	Feedback   string         `json:"feedback,omitempty"`
	CreatedAt  string         `json:"created_at"`
}

// HandleApprove handles POST /api/v1/proposals/{id}/approve.
func (h *ReviewHandler) HandleApprove(w http.ResponseWriter, r *http.Request) {
	proposalID := chi.URLParam(r, "id")
	h.handleReviewAction(w, r, proposalID, h.svc.ApproveProposal)
}

// HandleReject handles POST /api/v1/proposals/{id}/reject.
func (h *ReviewHandler) HandleReject(w http.ResponseWriter, r *http.Request) {
	proposalID := chi.URLParam(r, "id")
	h.handleReviewAction(w, r, proposalID, h.svc.RejectProposal)
}

// HandleCancel handles POST /api/v1/proposals/{id}/cancel.
func (h *ReviewHandler) HandleCancel(w http.ResponseWriter, r *http.Request) {
	proposalID := chi.URLParam(r, "id")
	h.handleReviewAction(w, r, proposalID, h.svc.CancelProposal)
}

// reviewActionFunc is the common signature for ApproveProposal / RejectProposal / CancelProposal.
type reviewActionFunc func(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error)

// handleReviewAction is a shared helper for approve/reject/cancel endpoints.
func (h *ReviewHandler) handleReviewAction(w http.ResponseWriter, r *http.Request, proposalID string, action reviewActionFunc) {
	var req approveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.ReviewerID == "" {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "reviewer_id is required"})
		return
	}

	record, err := action(r.Context(), proposalID, req.ReviewerID, req.Feedback)
	if err != nil {
		if errors.Is(err, review.ErrProposalNotFound) {
			httputil.JSON(w, http.StatusNotFound, map[string]string{"error": "proposal not found"})
			return
		}
		if errors.Is(err, review.ErrInvalidState) {
			httputil.JSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	httputil.JSON(w, http.StatusOK, recordToResponse(record))
}

// HandleGetReview handles GET /api/v1/proposals/{id}/review.
func (h *ReviewHandler) HandleGetReview(w http.ResponseWriter, r *http.Request) {
	proposalID := chi.URLParam(r, "id")

	record, err := h.svc.GetReviewByProposal(r.Context(), proposalID)
	if err != nil {
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	if record == nil {
		httputil.JSON(w, http.StatusNotFound, map[string]string{"error": "review not found"})
		return
	}

	httputil.JSON(w, http.StatusOK, recordToResponse(record))
}

// recordToResponse converts a ReviewRecord to the response DTO.
func recordToResponse(r *review.ReviewRecord) reviewResponse {
	return reviewResponse{
		RecordID:   r.RecordID,
		ProposalID: r.ProposalID,
		Verdict:    r.Verdict,
		ReviewerID: r.ReviewerID,
		Feedback:   r.Feedback,
		CreatedAt:  r.CreatedAt.Format(time.RFC3339),
	}
}
