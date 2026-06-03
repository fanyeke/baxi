package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"baxi/internal/action"
	"baxi/internal/api/dto"
	"baxi/internal/api/middleware"
	"baxi/internal/httputil"
)

// ActionService defines the business operations needed by ActionHandler.
// Tests substitute a mock without importing the service package.
type ActionService interface {
	ExecuteProposal(ctx context.Context, pool *pgxpool.Pool, proposalID, actorID string, opts ...action.ExecuteOption) (*action.ExecutionResult, error)
	GetProposalByID(ctx context.Context, pool *pgxpool.Pool, proposalID string) (*action.ActionProposal, error)
}

// ActionHandler handles HTTP requests for action execution endpoints.
type ActionHandler struct {
	svc  ActionService
	pool *pgxpool.Pool
}

// NewActionHandler creates a new ActionHandler.
func NewActionHandler(svc ActionService, pool *pgxpool.Pool) *ActionHandler {
	return &ActionHandler{svc: svc, pool: pool}
}

// executeRequest is the request body for the execute endpoint.
type executeRequest struct {
	DryRun *bool `json:"dry_run,omitempty"`
}

// executeResponse is the response body for the execute endpoint.
type executeResponse struct {
	ProposalID    string `json:"proposal_id"`
	ApplyStatus   string `json:"apply_status"`
	DryRun        bool   `json:"dry_run"`
	OutboxEventID string `json:"outbox_event_id,omitempty"`
}

// statusResponse is the response body for the status endpoint.
type statusResponse struct {
	ProposalID  string `json:"proposal_id"`
	ApplyStatus string `json:"apply_status"`
	ActionType  string `json:"action_type"`
}

// HandleExecute handles POST /api/v1/proposals/{id}/execute.
func (h *ActionHandler) HandleExecute(w http.ResponseWriter, r *http.Request) {
	proposalID := chi.URLParam(r, "id")
	if proposalID == "" {
		writeValidationError(w, r, "validation failed", []dto.FieldError{
			{Field: "id", Message: "proposal ID is required", Code: "required"},
		})
		return
	}

	var req executeRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, middleware.BAD_REQUEST, "invalid request body: "+err.Error())
			return
		}
	}

	dryRun := true
	if req.DryRun != nil {
		dryRun = *req.DryRun
	}

	actorID := getActorFromContext(r.Context())

	result, err := h.svc.ExecuteProposal(r.Context(), h.pool, proposalID, actorID, action.WithDryRun(dryRun))
	if err != nil {
		if errors.Is(err, action.ErrProposalNotFound) {
			writeError(w, r, http.StatusNotFound, middleware.NOT_FOUND, "proposal not found")
			return
		}
		if errors.Is(err, action.ErrNotApproved) || errors.Is(err, action.ErrActionNotAllowed) {
			writeError(w, r, http.StatusForbidden, middleware.FORBIDDEN, err.Error())
			return
		}
		writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "internal server error")
		return
	}

	resp := executeResponse{
		ProposalID:    proposalID,
		DryRun:        result.DryRun,
		OutboxEventID: result.OutboxEventID,
	}
	if !result.Success {
		resp.ApplyStatus = "failed"
	} else if dryRun {
		resp.ApplyStatus = "approved"
	} else {
		resp.ApplyStatus = "applied"
	}

	httputil.JSON(w, http.StatusOK, resp)
}

// HandleStatus handles GET /api/v1/proposals/{id}/status.
func (h *ActionHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	proposalID := chi.URLParam(r, "id")
	if proposalID == "" {
		writeValidationError(w, r, "validation failed", []dto.FieldError{
			{Field: "id", Message: "proposal ID is required", Code: "required"},
		})
		return
	}

	proposal, err := h.svc.GetProposalByID(r.Context(), h.pool, proposalID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, r, http.StatusNotFound, middleware.NOT_FOUND, "proposal not found")
			return
		}
		writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "internal server error")
		return
	}
	if proposal == nil {
		writeError(w, r, http.StatusNotFound, middleware.NOT_FOUND, "proposal not found")
		return
	}

	resp := statusResponse{
		ProposalID:  proposal.ProposalID,
		ApplyStatus: proposal.ApplyStatus,
		ActionType:  proposal.ActionType,
	}
	httputil.JSON(w, http.StatusOK, resp)
}

// getActorFromContext retrieves the authenticated actor from request context.
func getActorFromContext(ctx context.Context) string {
	if actor, ok := ctx.Value(middleware.ActorKey).(string); ok {
		return actor
	}
	return "unknown"
}
