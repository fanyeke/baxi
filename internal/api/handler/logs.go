package handler

import (
	"context"
	"net/http"

	"baxi/internal/api/dto"
	"baxi/internal/httputil"
)

// LogLister is the interface for listing logs. Used by LogHandler so
// tests can substitute a mock without importing the service package.
type LogLister interface {
	ListRecent(ctx context.Context, limit, offset int) (*dto.LogListResponse, error)
	ListErrors(ctx context.Context, limit, offset int) (*dto.LogListResponse, error)
	ListAudit(ctx context.Context, limit, offset int) (*dto.LogListResponse, error)
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
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	resp, err := h.svc.ListRecent(r.Context(), pagination.Limit, pagination.Offset)
	if err != nil {
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	httputil.JSON(w, http.StatusOK, resp)
}

// HandleListErrors handles GET /api/v1/logs/errors.
func (h *LogHandler) HandleListErrors(w http.ResponseWriter, r *http.Request) {
	pagination, err := httputil.ParsePagination(r)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	resp, err := h.svc.ListErrors(r.Context(), pagination.Limit, pagination.Offset)
	if err != nil {
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	httputil.JSON(w, http.StatusOK, resp)
}

// HandleListAudit handles GET /api/v1/logs/audit.
func (h *LogHandler) HandleListAudit(w http.ResponseWriter, r *http.Request) {
	pagination, err := httputil.ParsePagination(r)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	resp, err := h.svc.ListAudit(r.Context(), pagination.Limit, pagination.Offset)
	if err != nil {
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	httputil.JSON(w, http.StatusOK, resp)
}
