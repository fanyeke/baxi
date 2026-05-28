package handler

import (
	"context"
	"net/http"

	"baxi/internal/api/dto"
	"baxi/internal/httputil"
)

// AlertLister is the interface for listing alerts. Used by AlertHandler so
// tests can substitute a mock without importing the service package.
type AlertLister interface {
	ListAlerts(ctx context.Context, filters dto.AlertFilters, sort string, limit, offset int) (*dto.AlertListResponse, error)
}

type AlertHandler struct {
	svc AlertLister
}

func NewAlertHandler(svc AlertLister) *AlertHandler {
	return &AlertHandler{svc: svc}
}

func (h *AlertHandler) HandleListAlerts(w http.ResponseWriter, r *http.Request) {
	pagination, err := httputil.ParsePagination(r)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	q := r.URL.Query()
	filters := dto.AlertFilters{
		Severity:   q.Get("severity"),
		Status:     q.Get("status"),
		ObjectType: q.Get("object_type"),
		RuleID:     q.Get("rule_id"),
	}
	sort := q.Get("sort")

	resp, err := h.svc.ListAlerts(r.Context(), filters, sort, pagination.Limit, pagination.Offset)
	if err != nil {
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	httputil.JSON(w, http.StatusOK, resp)
}

