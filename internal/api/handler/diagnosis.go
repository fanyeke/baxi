package handler

import (
	"net/http"

	"baxi/internal/api/dto"
	"baxi/internal/api/middleware"
	"baxi/internal/httputil"
	"baxi/internal/model"
)

// Diagnoser is the interface for cross-source request tracing.
type Diagnoser interface {
	DiagnoseByRequestID(requestID string) (*model.DiagnosisResponse, error)
}

// diagnosisResponseFromModel converts a model.DiagnosisResponse to a dto.DiagnosisResponse
// for JSON serialization with proper JSON tags.
func diagnosisResponseFromModel(m *model.DiagnosisResponse) *dto.DiagnosisResponse {
	if m == nil {
		return nil
	}
	logs := make([]dto.DiagnosisLogEntry, len(m.RelatedLogs))
	for i, l := range m.RelatedLogs {
		logs[i] = dto.DiagnosisLogEntry{
			Source:    l.Source,
			Ts:        l.Ts,
			Timestamp: l.Timestamp,
			ErrorCode: l.ErrorCode,
			Message:   l.Message,
			Diagnosis: l.Diagnosis,
			OutboxID:  l.OutboxID,
			Status:    l.Status,
			Error:     l.Error,
			Action:    l.Action,
		}
	}
	return &dto.DiagnosisResponse{
		RequestID:       m.RequestID,
		Summary:         m.Summary,
		ErrorCode:       m.ErrorCode,
		Diagnosis:       m.Diagnosis,
		SuggestedAction: m.SuggestedAction,
		RelatedLogs:     logs,
	}
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

	httputil.JSON(w, http.StatusOK, diagnosisResponseFromModel(result))
}
