package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"baxi/internal/action"
	"baxi/internal/adapter"
	"baxi/internal/api/middleware"
	"baxi/internal/decision"
	"baxi/internal/httputil"
)

// ─── DTOs ─────────────────────────────────────────────────────────────

// feishuRequest is the shared request body for all Feishu endpoints.
// Matches Python FeishuExportRequest / FeishuSyncRequest / FeishuStatusImportRequest.
type feishuRequest struct {
	Tables []string `json:"tables"`
	Apply  bool     `json:"apply"`
}

// FeishuTableResult represents per-table result in the response.
// Matches Python FeishuTableResult.
type FeishuTableResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Rows   int    `json:"rows"`
}

// feishuResponse is the shared response body for all Feishu endpoints.
// Matches Python FeishuExportResponse / FeishuSyncResponse / FeishuStatusImportResponse.
type feishuResponse struct {
	Status  string             `json:"status"`
	Message string             `json:"message"`
	Tables  []FeishuTableResult `json:"tables"`
}

// ─── Service Interface ───────────────────────────────────────────────

// FeishuService defines the business operations needed by FeishuHandler.
// Tests substitute a mock without importing the service package.
type FeishuService interface {
	Export(ctx context.Context, tables []string, apply bool) (*feishuResponse, error)
	Sync(ctx context.Context, tables []string, apply bool) (*feishuResponse, error)
	StatusImport(ctx context.Context, tables []string, apply bool) (*feishuResponse, error)
}

// ─── Service Implementation ──────────────────────────────────────────

// feishuService implements FeishuService by wrapping adapter.FeishuAdapter.
type feishuService struct {
	adapter *adapter.FeishuAdapter
}

// NewFeishuService creates a new FeishuService backed by the given adapter.
func NewFeishuService(feishuAdapter *adapter.FeishuAdapter) FeishuService {
	return &feishuService{adapter: feishuAdapter}
}

func (s *feishuService) Export(ctx context.Context, tables []string, apply bool) (*feishuResponse, error) {
	return s.dispatch(ctx, "export_report", "feishu_export", tables, apply)
}

func (s *feishuService) Sync(ctx context.Context, tables []string, apply bool) (*feishuResponse, error) {
	return s.dispatch(ctx, "sync_data", "feishu_sync", tables, apply)
}

func (s *feishuService) StatusImport(ctx context.Context, tables []string, apply bool) (*feishuResponse, error) {
	return s.dispatch(ctx, "import_status", "feishu_status_import", tables, apply)
}

// dispatch creates an ActionProposal and dispatches it via the FeishuAdapter.
func (s *feishuService) dispatch(ctx context.Context, actionType, title string, tables []string, apply bool) (*feishuResponse, error) {
	dryRun := !apply

	payload := map[string]interface{}{
		"tables": tables,
	}

	proposal := action.ActionProposal{
		ProposalID: decision.GenerateProposalID(),
		ActionType: actionType,
		Title:      title,
		Payload:    payload,
	}

	result, err := s.adapter.Execute(ctx, proposal, dryRun)
	if err != nil {
		tableResults := make([]FeishuTableResult, len(tables))
		for i, tbl := range tables {
			tableResults[i] = FeishuTableResult{
				Name:   tbl,
				Status: "failed",
				Rows:   0,
			}
		}
		return &feishuResponse{
			Status:  "failed",
			Message: err.Error(),
			Tables:  tableResults,
		}, nil
	}

	tableResults := make([]FeishuTableResult, len(tables))
	for i, tbl := range tables {
		status := "ok"
		if result.DryRun {
			status = "dry_run"
		}
		tableResults[i] = FeishuTableResult{
			Name:   tbl,
			Status: status,
			Rows:   0,
		}
	}

	status := "ok"
	message := ""
	if result.DryRun {
		status = "dry_run"
		message = "dry-run mode, no data was pushed to Feishu"
	}

	return &feishuResponse{
		Status:  status,
		Message: message,
		Tables:  tableResults,
	}, nil
}

// ─── Handler ─────────────────────────────────────────────────────────

// FeishuHandler handles HTTP requests for Feishu integration endpoints.
type FeishuHandler struct {
	svc FeishuService
}

// NewFeishuHandler creates a new FeishuHandler.
func NewFeishuHandler(svc FeishuService) *FeishuHandler {
	return &FeishuHandler{svc: svc}
}

// HandleExport handles POST /api/v1/feishu/export.
func (h *FeishuHandler) HandleExport(w http.ResponseWriter, r *http.Request) {
	var req feishuRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, middleware.BAD_REQUEST, "invalid request body")
		return
	}

	resp, err := h.svc.Export(r.Context(), req.Tables, req.Apply)
	if err != nil {
		writeServiceError(w, r, err, "failed to export to Feishu")
		return
	}

	statusCode := http.StatusOK
	if resp.Status == "failed" {
		statusCode = http.StatusInternalServerError
	}
	httputil.JSON(w, statusCode, resp)
}

// HandleSync handles POST /api/v1/feishu/sync.
func (h *FeishuHandler) HandleSync(w http.ResponseWriter, r *http.Request) {
	var req feishuRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, middleware.BAD_REQUEST, "invalid request body")
		return
	}

	resp, err := h.svc.Sync(r.Context(), req.Tables, req.Apply)
	if err != nil {
		writeServiceError(w, r, err, "failed to sync with Feishu")
		return
	}

	statusCode := http.StatusOK
	if resp.Status == "failed" {
		statusCode = http.StatusInternalServerError
	}
	httputil.JSON(w, statusCode, resp)
}

// HandleStatusImport handles POST /api/v1/feishu/status/import.
func (h *FeishuHandler) HandleStatusImport(w http.ResponseWriter, r *http.Request) {
	var req feishuRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, middleware.BAD_REQUEST, "invalid request body")
		return
	}

	resp, err := h.svc.StatusImport(r.Context(), req.Tables, req.Apply)
	if err != nil {
		writeServiceError(w, r, err, "failed to import status from Feishu")
		return
	}

	statusCode := http.StatusOK
	if resp.Status == "failed" {
		statusCode = http.StatusInternalServerError
	}
	httputil.JSON(w, statusCode, resp)
}

// Compile-time assertion that *feishuService satisfies FeishuService.
var _ FeishuService = (*feishuService)(nil)
