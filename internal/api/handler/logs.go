package handler

import (
	"context"
	"net/http"

	"baxi/internal/api/dto"
	"baxi/internal/api/middleware"
	"baxi/internal/httputil"
	"baxi/internal/model"
)

// LogLister is the interface for listing logs. Used by LogHandler so
// tests can substitute a mock without importing the service package.
type LogLister interface {
	ListRecent(ctx context.Context, limit, offset int) (*model.LogListResponse, error)
	ListErrors(ctx context.Context, limit, offset int) (*model.LogListResponse, error)
	ListAudit(ctx context.Context, limit, offset int) (*model.LogListResponse, error)
}

// LogHandler handles HTTP requests for log-related endpoints.
type LogHandler struct {
	svc LogLister
}

// NewLogHandler creates a new LogHandler.
func NewLogHandler(svc LogLister) *LogHandler {
	return &LogHandler{svc: svc}
}

// HandleListRecent handles GET /api/v1/logs/recent.
func (h *LogHandler) HandleListRecent(w http.ResponseWriter, r *http.Request) {
	pagination, err := httputil.ParsePagination(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, middleware.BAD_REQUEST, err.Error())
		return
	}

	resp, err := h.svc.ListRecent(r.Context(), pagination.Limit, pagination.Offset)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "internal server error")
		return
	}

	httputil.JSON(w, http.StatusOK, dtoFromLogListResponse(resp))
}

// HandleListErrors handles GET /api/v1/logs/errors.
func (h *LogHandler) HandleListErrors(w http.ResponseWriter, r *http.Request) {
	pagination, err := httputil.ParsePagination(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, middleware.BAD_REQUEST, err.Error())
		return
	}

	resp, err := h.svc.ListErrors(r.Context(), pagination.Limit, pagination.Offset)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "internal server error")
		return
	}

	httputil.JSON(w, http.StatusOK, dtoFromLogListResponse(resp))
}

// HandleListAudit handles GET /api/v1/logs/audit.
func (h *LogHandler) HandleListAudit(w http.ResponseWriter, r *http.Request) {
	pagination, err := httputil.ParsePagination(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, middleware.BAD_REQUEST, err.Error())
		return
	}

	resp, err := h.svc.ListAudit(r.Context(), pagination.Limit, pagination.Offset)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "internal server error")
		return
	}

	httputil.JSON(w, http.StatusOK, dtoFromLogListResponse(resp))
}

// dtoFromLogListResponse converts model.LogListResponse to dto.LogListResponse.
func dtoFromLogListResponse(m *model.LogListResponse) *dto.LogListResponse {
	if m == nil {
		return nil
	}

	items := make([]dto.LogItem, len(m.Items))
	for i, item := range m.Items {
		items[i] = dto.LogItem{
			LogType:   item.LogType,
			Level:     item.Level,
			Message:   item.Message,
			RequestID: item.RequestID,
			CreatedAt: item.CreatedAt,
		}
	}

	return &dto.LogListResponse{
		Items: items,
		Total: m.Total,
	}
}
