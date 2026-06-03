package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"baxi/internal/api/dto"
	"baxi/internal/api/middleware"
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
		writeError(w, r, http.StatusBadRequest, middleware.BAD_REQUEST, "invalid request body")
		return
	}

	if req.ReviewerID == "" {
		writeValidationError(w, r, "validation failed", []dto.FieldError{
			{Field: "reviewer_id", Message: "reviewer_id is required", Code: "required"},
		})
		return
	}

	record, err := action(r.Context(), proposalID, req.ReviewerID, req.Feedback)
	if err != nil {
		if errors.Is(err, review.ErrProposalNotFound) {
			writeError(w, r, http.StatusNotFound, middleware.NOT_FOUND, "proposal not found")
			return
		}
		if errors.Is(err, review.ErrInvalidState) {
			writeError(w, r, http.StatusConflict, middleware.CONFLICT, err.Error())
			return
		}
		writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "internal server error")
		return
	}

	httputil.JSON(w, http.StatusOK, recordToResponse(record))
}

// HandleGetReview handles GET /api/v1/proposals/{id}/review.
func (h *ReviewHandler) HandleGetReview(w http.ResponseWriter, r *http.Request) {
	proposalID := chi.URLParam(r, "id")

	record, err := h.svc.GetReviewByProposal(r.Context(), proposalID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "internal server error")
		return
	}
	if record == nil {
		writeError(w, r, http.StatusNotFound, middleware.NOT_FOUND, "review not found")
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
