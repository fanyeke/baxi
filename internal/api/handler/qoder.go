package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5/middleware"

	"baxi/internal/api/dto"
	"baxi/internal/httputil"
)

// ContextFetcher is the interface for fetching the Qoder context response.
// Tests can substitute a mock without importing the service package.
type ContextFetcher interface {
	GetContext(ctx context.Context, requestID string, params dto.ContextQueryParams) (*dto.ContextResponse, error)
}

// QoderHandler handles Qoder AI decision engine endpoints.
type QoderHandler struct {
	ctxFetcher ContextFetcher
}

// NewQoderHandler creates a new QoderHandler.
// When called with no arguments, the handler works in a static mode for capabilities only.
// When called with a ContextFetcher, it enables the context endpoint.
func NewQoderHandler(ctxFetcher ...ContextFetcher) *QoderHandler {
	h := &QoderHandler{}
	if len(ctxFetcher) > 0 {
		h.ctxFetcher = ctxFetcher[0]
	}
	return h
}

// HandleCapabilities returns the static Qoder capability matrix.
func (h *QoderHandler) HandleCapabilities(w http.ResponseWriter, r *http.Request) {
	httputil.JSON(w, http.StatusOK, dto.StaticCapabilities())
}

// HandleContext returns the aggregated Qoder context.
// Supports query params: severity, limit_alerts (1-100, default 10),
// limit_tasks (1-100, default 10), limit_outbox (1-100, default 10),
// include_logs (default false).
func (h *QoderHandler) HandleContext(w http.ResponseWriter, r *http.Request) {
	if h.ctxFetcher == nil {
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "context fetcher not available"})
		return
	}

	params := parseContextParams(r)
	requestID := middleware.GetReqID(r.Context())
	if requestID == "" {
		requestID = "unknown"
	}

	resp, err := h.ctxFetcher.GetContext(r.Context(), requestID, params)
	if err != nil {
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	httputil.JSON(w, http.StatusOK, resp)
}

func parseContextParams(r *http.Request) dto.ContextQueryParams {
	q := r.URL.Query()

	params := dto.ContextQueryParams{
		Severity:    q.Get("severity"),
		LimitAlerts: 10,
		LimitTasks:  10,
		LimitOutbox: 10,
	}

	if v := q.Get("limit_alerts"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			params.LimitAlerts = n
			if params.LimitAlerts > 100 {
				params.LimitAlerts = 100
			}
		}
	}
	if v := q.Get("limit_tasks"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			params.LimitTasks = n
			if params.LimitTasks > 100 {
				params.LimitTasks = 100
			}
		}
	}
	if v := q.Get("limit_outbox"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			params.LimitOutbox = n
			if params.LimitOutbox > 100 {
				params.LimitOutbox = 100
			}
		}
	}
	if v := q.Get("include_logs"); v == "true" || v == "1" {
		params.IncludeLogs = true
	}

	return params
}
