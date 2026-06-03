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

// SandboxService defines the business operations needed by SandboxHandler.
type SandboxService interface {
	CreateSandbox(ctx context.Context, caseID string, data map[string]interface{}) (string, error)
	GetSandbox(ctx context.Context, sandboxID string) (*review.Sandbox, error)
	ListSandboxes(ctx context.Context) ([]review.Sandbox, error)
	AddProposalToSandbox(ctx context.Context, sandboxID, proposalID string) error
	CompareSandbox(ctx context.Context, sandboxID1, sandboxID2 string) (*review.ComparisonResult, error)
}

// SandboxHandler handles HTTP requests for sandbox endpoints.
type SandboxHandler struct {
	svc SandboxService
}

// NewSandboxHandler creates a new SandboxHandler.
func NewSandboxHandler(svc SandboxService) *SandboxHandler {
	return &SandboxHandler{svc: svc}
}

// HandleCreate handles POST /api/v1/sandboxes.
func (h *SandboxHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateSandboxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, middleware.BAD_REQUEST, "invalid request body")
		return
	}

	if req.CaseID == "" {
		writeError(w, r, http.StatusBadRequest, middleware.BAD_REQUEST, "case_id is required")
		return
	}

	if req.Data == nil {
		req.Data = make(map[string]interface{})
	}

	sandboxID, err := h.svc.CreateSandbox(r.Context(), req.CaseID, req.Data)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "failed to create sandbox")
		return
	}

	httputil.JSON(w, http.StatusCreated, map[string]string{"sandbox_id": sandboxID})
}

// HandleGet handles GET /api/v1/sandboxes/{id}.
func (h *SandboxHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	sandboxID := chi.URLParam(r, "id")

	sb, err := h.svc.GetSandbox(r.Context(), sandboxID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "failed to get sandbox")
		return
	}
	if sb == nil {
		writeError(w, r, http.StatusNotFound, middleware.NOT_FOUND, "sandbox not found")
		return
	}

	httputil.JSON(w, http.StatusOK, sandboxToResponse(sb))
}

// HandleList handles GET /api/v1/sandboxes.
func (h *SandboxHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	sandboxes, err := h.svc.ListSandboxes(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "failed to list sandboxes")
		return
	}

	items := make([]dto.SandboxResponse, 0, len(sandboxes))
	for _, sb := range sandboxes {
		items = append(items, sandboxToResponse(&sb))
	}

	httputil.JSON(w, http.StatusOK, dto.SandboxListResponse{Items: items})
}

// HandleAddProposal handles POST /api/v1/sandboxes/{id}/proposals.
func (h *SandboxHandler) HandleAddProposal(w http.ResponseWriter, r *http.Request) {
	sandboxID := chi.URLParam(r, "id")

	var req dto.AddProposalToSandboxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, middleware.BAD_REQUEST, "invalid request body")
		return
	}

	if req.ProposalID == "" {
		writeError(w, r, http.StatusBadRequest, middleware.BAD_REQUEST, "proposal_id is required")
		return
	}

	if err := h.svc.AddProposalToSandbox(r.Context(), sandboxID, req.ProposalID); err != nil {
		if errors.Is(err, review.ErrSandboxNotFound) {
			writeError(w, r, http.StatusNotFound, middleware.NOT_FOUND, "sandbox not found")
			return
		}
		writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "failed to add proposal to sandbox")
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// HandleCompare handles GET /api/v1/sandboxes/compare?...
func (h *SandboxHandler) HandleCompare(w http.ResponseWriter, r *http.Request) {
	sandboxID1 := r.URL.Query().Get("sandbox1_id")
	sandboxID2 := r.URL.Query().Get("sandbox2_id")

	if sandboxID1 == "" || sandboxID2 == "" {
		writeError(w, r, http.StatusBadRequest, middleware.BAD_REQUEST, "both sandbox1_id and sandbox2_id query parameters are required")
		return
	}

	result, err := h.svc.CompareSandbox(r.Context(), sandboxID1, sandboxID2)
	if err != nil {
		if errors.Is(err, review.ErrSandboxNotFound) {
			writeError(w, r, http.StatusNotFound, middleware.NOT_FOUND, "sandbox not found")
			return
		}
		writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "failed to compare sandboxes")
		return
	}

	httputil.JSON(w, http.StatusOK, comparisonToResponse(result))
}

// sandboxToResponse converts a domain Sandbox to the response DTO.
func sandboxToResponse(sb *review.Sandbox) dto.SandboxResponse {
	resp := dto.SandboxResponse{
		SandboxID:    sb.SandboxID,
		CaseID:       sb.CaseID,
		ProposalID:   sb.ProposalID,
		Data:         sb.SandboxData,
		Status:       sb.Status,
		ComparedWith: sb.ComparedWith,
		CreatedAt:    sb.CreatedAt.Format(time.RFC3339),
	}
	if sb.UpdatedAt != nil {
		formatted := sb.UpdatedAt.Format(time.RFC3339)
		resp.UpdatedAt = &formatted
	}
	return resp
}

// comparisonToResponse converts a domain ComparisonResult to the response DTO.
func comparisonToResponse(cr *review.ComparisonResult) dto.ComparisonResponse {
	diffs := make([]dto.SandboxDiffItem, 0, len(cr.Differences))
	for _, d := range cr.Differences {
		diffs = append(diffs, dto.SandboxDiffItem{
			Field:  d.Field,
			Value1: d.Value1,
			Value2: d.Value2,
		})
	}
	return dto.ComparisonResponse{
		Sandbox1ID:  cr.Sandbox1ID,
		Sandbox2ID:  cr.Sandbox2ID,
		Differences: diffs,
	}
}
