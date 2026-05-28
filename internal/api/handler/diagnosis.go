package handler

import (
	"net/http"

	"baxi/internal/api/dto"
	"baxi/internal/api/middleware"
	"baxi/internal/httputil"
)

// Diagnoser is the interface for cross-source request tracing.
type Diagnoser interface {
	DiagnoseByRequestID(requestID string) (*dto.DiagnosisResponse, error)
}

// DiagnosisHandler handles HTTP requests for the diagnosis endpoint.
type DiagnosisHandler struct {
	svc Diagnoser
}

// NewDiagnosisHandler creates a new DiagnosisHandler.
func NewDiagnosisHandler(svc Diagnoser) *DiagnosisHandler {
	return &DiagnosisHandler{svc: svc}
}

// HandleDiagnosis handles GET /api/v1/logs/diagnosis.
func (h *DiagnosisHandler) HandleDiagnosis(w http.ResponseWriter, r *http.Request) {
	requestID := r.URL.Query().Get("request_id")
	if requestID == "" {
		writeError(w, r, http.StatusBadRequest, middleware.BAD_REQUEST, "request_id is required")
		return
	}

	result, err := h.svc.DiagnoseByRequestID(requestID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "failed to diagnose request")
		return
	}

	if result == nil {
		writeErrorWithDetails(w, r, http.StatusNotFound, middleware.NOT_FOUND,
			"No logs found for request_id: "+requestID,
			"The request_id was not found in error.log, audit CSV, or Feishu audit CSV.",
			"Verify the request_id is correct. Logs may have been rotated or the request may not have generated an error.",
		)
		return
	}

	httputil.JSON(w, http.StatusOK, result)
}
