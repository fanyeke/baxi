package handler

import (
	"net/http"

	"baxi/internal/api/dto"
	"baxi/internal/httputil"
	"baxi/internal/service"
)

type OutboxHandler struct {
	svc *service.OutboxService
}

func NewOutboxHandler(svc *service.OutboxService) *OutboxHandler {
	return &OutboxHandler{svc: svc}
}

func (h *OutboxHandler) HandleListOutbox(w http.ResponseWriter, r *http.Request) {
	pagination, err := httputil.ParsePagination(r)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	filters := parseOutboxFilters(r)

	resp, err := h.svc.List(r.Context(), filters, pagination.Limit, pagination.Offset)
	if err != nil {
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	httputil.JSON(w, http.StatusOK, resp)
}

func parseOutboxFilters(r *http.Request) dto.OutboxFilters {
	q := r.URL.Query()
	var filters dto.OutboxFilters

	if s := q.Get("status"); s != "" {
		filters.Status = &s
	}
	if c := q.Get("channel"); c != "" {
		filters.Channel = &c
	}
	if e := q.Get("event_type"); e != "" {
		filters.EventType = &e
	}

	return filters
}
