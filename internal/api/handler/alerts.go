package handler

import (
	"context"
	"net/http"

	"baxi/internal/api/dto"
	"baxi/internal/api/middleware"
	"baxi/internal/httputil"
	"baxi/internal/model"
)

// AlertLister is the interface for listing alerts. Used by AlertHandler so
// tests can substitute a mock without importing the service package.
type AlertLister interface {
	ListAlerts(ctx context.Context, filters model.AlertFilters, sort string, limit, offset int) (*model.AlertListResponse, error)
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
		writeError(w, r, http.StatusBadRequest, middleware.BAD_REQUEST, err.Error())
		return
	}

	q := r.URL.Query()
	filters := model.AlertFilters{
		Severity:   q.Get("severity"),
		Status:     q.Get("status"),
		ObjectType: q.Get("object_type"),
		RuleID:     q.Get("rule_id"),
	}
	sort := q.Get("sort")

	resp, err := h.svc.ListAlerts(r.Context(), filters, sort, pagination.Limit, pagination.Offset)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "internal server error")
		return
	}

	// Convert model to DTO
	dtoResp := dtoFromAlertListResponse(resp)
	httputil.JSON(w, http.StatusOK, dtoResp)
}

// dtoFromAlertListResponse converts model.AlertListResponse to dto.AlertListResponse.
func dtoFromAlertListResponse(m *model.AlertListResponse) *dto.AlertListResponse {
	if m == nil {
		return nil
	}

	items := make([]dto.AlertItem, len(m.Items))
	for i, item := range m.Items {
		items[i] = dto.AlertItem{
			EventID:       item.EventID,
			RuleID:        item.RuleID,
			EventDate:     item.EventDate,
			Severity:      item.Severity,
			MetricName:    item.MetricName,
			ObjectType:    item.ObjectType,
			ObjectID:      item.ObjectID,
			CurrentValue:  item.CurrentValue,
			BaselineValue: item.BaselineValue,
			ChangeRate:    item.ChangeRate,
			OwnerRole:     item.OwnerRole,
			Status:        item.Status,
			ImpactScore:   item.ImpactScore,
		}
	}

	return &dto.AlertListResponse{
		Items: items,
		Total: m.Total,
	}
}
