package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"baxi/internal/action"
	"baxi/internal/api/dto"
	"baxi/internal/api/middleware"
	"baxi/internal/decision"
	"baxi/internal/eval"
	"baxi/internal/httputil"
	"baxi/internal/llm"
)

// DecisionService defines the business operations needed by DecisionHandler.
// Tests substitute a mock without importing the service package.
type DecisionService interface {
	CreateCaseFromAlert(ctx context.Context, alertID, createdBy string) (*decision.DecisionCase, error)
	GetCase(ctx context.Context, caseID string) (*decision.DecisionCase, error)
	ListCases(ctx context.Context, filter decision.CaseFilter) (*decision.CaseList, error)
	BuildContext(ctx context.Context, caseID string) (*decision.DecisionContext, error)
	Decide(ctx context.Context, caseID string) (*decision.DecisionContext, *llm.DecisionOutput, []action.ActionProposal, error)
	ListProposals(ctx context.Context, caseID string) ([]action.ActionProposal, error)
	DecideLLM(ctx context.Context, caseID string) (*decision.DecisionContext, *llm.DecisionOutput, []action.ActionProposal, error)
	ListLLMDecisions(ctx context.Context, caseID string) (interface{}, error)
	ListEvals(ctx context.Context, caseID string) (interface{}, error)
	Compare(ctx context.Context, caseID string) (*eval.DecisionComparison, error)
	Replay(ctx context.Context, caseID string, dryRun bool) (*eval.ReplayResult, error)
}

// DecisionHandler handles HTTP requests for decision case endpoints.
type DecisionHandler struct {
	svc DecisionService
}

// NewDecisionHandler creates a new DecisionHandler.
func NewDecisionHandler(svc DecisionService) *DecisionHandler {
	return &DecisionHandler{svc: svc}
}

// CreateCase handles POST /api/v1/decisions/cases.
func (h *DecisionHandler) CreateCase(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, middleware.BAD_REQUEST, "invalid request body")
		return
	}

	if req.SourceType == "" || req.SourceID == "" {
		writeError(w, r, http.StatusBadRequest, middleware.BAD_REQUEST, "source_type and source_id are required")
		return
	}

	c, err := h.svc.CreateCaseFromAlert(r.Context(), req.SourceID, "api_user")
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "internal server error")
		return
	}

	sourceType := ""
	sourceID := ""
	if c.SourceType != nil {
		sourceType = *c.SourceType
	}
	if c.SourceID != nil {
		sourceID = *c.SourceID
	}
	resp := dto.CreateCaseResponse{
		DecisionCaseID: c.CaseID,
		SourceType:     sourceType,
		SourceID:       sourceID,
		Status:         c.Status,
	}

	httputil.JSON(w, http.StatusCreated, resp)
}

// GetCase handles GET /api/v1/decisions/cases/{case_id}.
func (h *DecisionHandler) GetCase(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "case_id")

	c, err := h.svc.GetCase(r.Context(), caseID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, r, http.StatusNotFound, middleware.NOT_FOUND, "case not found")
			return
		}
		writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "internal server error")
		return
	}

	httputil.JSON(w, http.StatusOK, caseToResponse(c))
}

// ListCases handles GET /api/v1/decisions/cases.
func (h *DecisionHandler) ListCases(w http.ResponseWriter, r *http.Request) {
	pagination, err := httputil.ParsePagination(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, middleware.BAD_REQUEST, err.Error())
		return
	}

	q := r.URL.Query()
	filter := decision.CaseFilter{
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
	}
	if v := q.Get("source_type"); v != "" {
		filter.SourceType = &v
	}
	if v := q.Get("status"); v != "" {
		filter.Status = &v
	}
	if v := q.Get("severity"); v != "" {
		filter.Severity = &v
	}

	result, err := h.svc.ListCases(r.Context(), filter)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "internal server error")
		return
	}

	items := make([]dto.DecisionCaseResponse, len(result.Cases))
	for i := range result.Cases {
		items[i] = caseToResponse(&result.Cases[i])
	}

	resp := dto.CaseListResponse{
		Items: items,
		Total: result.Total,
	}
	httputil.JSON(w, http.StatusOK, resp)
}

// BuildContext handles POST /api/v1/decisions/cases/{case_id}/context.
func (h *DecisionHandler) BuildContext(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "case_id")

	// Check existence and get current status.
	c, err := h.svc.GetCase(r.Context(), caseID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, r, http.StatusNotFound, middleware.NOT_FOUND, "case not found")
			return
		}
		writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "internal server error")
		return
	}

	decCtx, err := h.svc.BuildContext(r.Context(), caseID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "internal server error")
		return
	}

	resp := dto.DecisionContextResponse{
		DecisionCaseID: decCtx.DecisionCaseID,
		Status:         c.Status,
		Trigger:        structToMap(decCtx.Trigger),
		ObjectContext:  structToMap(decCtx.ObjectContext),
		Governance:     structToMap(decCtx.Governance),
		AllowedActions: decCtx.AllowedActions,
	}
	httputil.JSON(w, http.StatusOK, resp)
}

// Decide handles POST /api/v1/decisions/cases/{case_id}/decide.
func (h *DecisionHandler) Decide(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "case_id")

	decCtx, output, proposals, err := h.svc.Decide(r.Context(), caseID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, r, http.StatusNotFound, middleware.NOT_FOUND, "case not found")
			return
		}
		writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "internal server error")
		return
	}

	resp := dto.DecisionResponse{
		DecisionCaseID: decCtx.DecisionCaseID,
		Status:         "decision_generated",
		Decision:       structToMap(output),
		Proposals:      proposalsToDTO(proposals),
	}
	httputil.JSON(w, http.StatusOK, resp)
}

// ListProposals handles GET /api/v1/decisions/cases/{case_id}/proposals.
func (h *DecisionHandler) ListProposals(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "case_id")

	proposals, err := h.svc.ListProposals(r.Context(), caseID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "internal server error")
		return
	}

	resp := dto.ProposalListResponse{
		Items: proposalsToDTO(proposals),
	}
	httputil.JSON(w, http.StatusOK, resp)
}

// --- DTO mapping helpers ---

func caseToResponse(c *decision.DecisionCase) dto.DecisionCaseResponse {
	sourceType := ""
	sourceID := ""
	if c.SourceType != nil {
		sourceType = *c.SourceType
	}
	if c.SourceID != nil {
		sourceID = *c.SourceID
	}
	resp := dto.DecisionCaseResponse{
		DecisionCaseID: c.CaseID,
		SourceType:     sourceType,
		SourceID:       sourceID,
		ObjectType:     c.ObjectType,
		ObjectID:       c.ObjectID,
		Severity:       c.Severity,
		Status:         c.Status,
		ContextHash:    c.ContextHash,
		CreatedAt:      c.CreatedAt.Format(time.RFC3339),
	}
	if c.UpdatedAt != nil {
		resp.UpdatedAt = c.UpdatedAt.Format(time.RFC3339)
	}
	return resp
}

func proposalsToDTO(proposals []action.ActionProposal) []dto.ProposalItem {
	items := make([]dto.ProposalItem, len(proposals))
	for i := range proposals {
		p := &proposals[i]
		items[i] = dto.ProposalItem{
			ProposalID:          p.ProposalID,
			ActionType:          p.ActionType,
			Title:               p.Title,
			RiskLevel:           p.RiskLevel,
			RequiresHumanReview: p.RequiresHumanReview,
			ApplyStatus:         p.ApplyStatus,
			CreatedAt:           p.CreatedAt.Format(time.RFC3339),
		}
	}
	return items
}

func structToMap(v interface{}) map[string]interface{} {
	data, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	return m
}

// DecideLLM handles POST /decisions/cases/{case_id}/decide/llm.
func (h *DecisionHandler) DecideLLM(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "case_id")

	decCtx, output, proposals, err := h.svc.DecideLLM(r.Context(), caseID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, r, http.StatusNotFound, middleware.NOT_FOUND, "case not found")
			return
		}
		writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "internal server error")
		return
	}

	resp := dto.DecisionResponse{
		DecisionCaseID: decCtx.DecisionCaseID,
		Status:         "decision_generated",
		Decision:       structToMap(output),
		Proposals:      proposalsToDTO(proposals),
	}
	httputil.JSON(w, http.StatusOK, resp)
}

// Compare handles POST /decisions/cases/{case_id}/compare.
func (h *DecisionHandler) Compare(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, http.StatusNotImplemented, middleware.INTERNAL_ERROR, "not implemented")
}

// Replay handles POST /decisions/cases/{case_id}/replay.
func (h *DecisionHandler) Replay(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, http.StatusNotImplemented, middleware.INTERNAL_ERROR, "not implemented")
}

// ListLLMDecisions handles GET /decisions/cases/{case_id}/llm-decisions.
func (h *DecisionHandler) ListLLMDecisions(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "case_id")

	result, err := h.svc.ListLLMDecisions(r.Context(), caseID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "internal server error")
		return
	}

	httputil.JSON(w, http.StatusOK, result)
}

// ListEvals handles GET /decisions/cases/{case_id}/evals.
func (h *DecisionHandler) ListEvals(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "case_id")

	result, err := h.svc.ListEvals(r.Context(), caseID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "internal server error")
		return
	}

	httputil.JSON(w, http.StatusOK, result)
}
