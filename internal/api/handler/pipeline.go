package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"baxi/internal/api/dto"
	"baxi/internal/api/middleware"
	"baxi/internal/httputil"
)

// PipelineRunner defines the interface for running pipelines.
// Tests substitute a mock without importing the pipeline package.
type PipelineRunner interface {
	Run(ctx context.Context, config string) (string, error)
}

// PipelineHandler handles HTTP requests for pipeline endpoints.
type PipelineHandler struct {
	svc PipelineRunner
}

// NewPipelineHandler creates a new PipelineHandler.
func NewPipelineHandler(svc PipelineRunner) *PipelineHandler {
	return &PipelineHandler{svc: svc}
}

// HandleRun handles POST /api/v1/pipeline/run.
func (h *PipelineHandler) HandleRun(w http.ResponseWriter, r *http.Request) {
	var req dto.PipelineRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, middleware.BAD_REQUEST, "invalid request body")
		return
	}

	if req.Config == "" {
		writeError(w, r, http.StatusBadRequest, middleware.BAD_REQUEST, "config is required")
		return
	}

	runID, err := h.svc.Run(r.Context(), req.Config)
	if err != nil {
		writeServiceError(w, r, err, "internal server error")
		return
	}

	resp := dto.PipelineRunResponse{
		RunID:  runID,
		Status: "started",
	}
	httputil.JSON(w, http.StatusOK, resp)
}
