package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"baxi/internal/api/dto"
	"baxi/internal/api/middleware"
	"baxi/internal/httputil"
	"baxi/internal/model"
)

type OutboxDetailItem struct {
	EventID          string     `json:"event_id"`
	EventType        string     `json:"event_type"`
	SourceType       string     `json:"source_type"`
	SourceID         string     `json:"source_id"`
	TargetChannel    string     `json:"target_channel"`
	Status           string     `json:"status"`
	Payload          string     `json:"payload_json"`
	CreatedAt        time.Time  `json:"created_at"`
	DispatchAttempts int        `json:"dispatch_attempts"`
	LastDispatchAt   *time.Time `json:"last_dispatch_at"`
	ErrorMessage     *string    `json:"last_error"`
}

type OutboxService interface {
	List(ctx context.Context, filters model.OutboxFilters, limit, offset int) (*model.OutboxListResponse, error)
	GetEvent(ctx context.Context, id string) (*OutboxDetailItem, error)
	DispatchEvent(ctx context.Context, id string) error
	CancelEvent(ctx context.Context, id string) error
}

type ErrEventNotFound struct{}

func (ErrEventNotFound) Error() string { return "event not found" }

type ErrInvalidState struct {
	Status string
}

func (e ErrInvalidState) Error() string {
	return fmt.Sprintf("event cannot be dispatched in %s state", e.Status)
}

type OutboxHandler struct {
	svc OutboxService
}

func NewOutboxHandler(svc OutboxService) *OutboxHandler {
	return &OutboxHandler{svc: svc}
}

func (h *OutboxHandler) HandleListOutbox(w http.ResponseWriter, r *http.Request) {
	pagination, err := httputil.ParsePagination(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, middleware.BAD_REQUEST, err.Error())
		return
	}

	filters := parseOutboxFilters(r)

	resp, err := h.svc.List(r.Context(), filters, pagination.Limit, pagination.Offset)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "internal server error")
		return
	}

	httputil.JSON(w, http.StatusOK, dtoFromOutboxListResponse(resp))
}

func (h *OutboxHandler) HandleDispatch(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "id")
	if eventID == "" {
		writeError(w, r, http.StatusBadRequest, middleware.BAD_REQUEST, "event_id required")
		return
	}

	err := h.svc.DispatchEvent(r.Context(), eventID)
	if err != nil {
		switch {
		case isNotFound(err):
			writeError(w, r, http.StatusNotFound, middleware.NOT_FOUND, err.Error())
		case isInvalidState(err):
			writeError(w, r, http.StatusConflict, middleware.BAD_REQUEST, err.Error())
		default:
			writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "internal server error")
		}
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]string{
		"event_id": eventID,
		"status":   "dispatched",
	})
}

func (h *OutboxHandler) HandleCancel(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "id")
	if eventID == "" {
		writeError(w, r, http.StatusBadRequest, middleware.BAD_REQUEST, "event_id required")
		return
	}

	event, err := h.svc.GetEvent(r.Context(), eventID)
	if err != nil {
		if isNotFound(err) {
			writeError(w, r, http.StatusNotFound, middleware.NOT_FOUND, err.Error())
			return
		}
		writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "internal server error")
		return
	}

	if event.Status != "pending" && event.Status != "failed" {
		writeError(w, r, http.StatusConflict, middleware.BAD_REQUEST, fmt.Sprintf("event cannot be cancelled in %s state", event.Status))
		return
	}

	err = h.svc.CancelEvent(r.Context(), eventID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "internal server error")
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]string{
		"event_id": eventID,
		"status":   "cancelled",
	})
}

func (h *OutboxHandler) HandleGetDetail(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "id")
	if eventID == "" {
		writeError(w, r, http.StatusBadRequest, middleware.BAD_REQUEST, "event_id required")
		return
	}

	event, err := h.svc.GetEvent(r.Context(), eventID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "internal server error")
		return
	}
	if event == nil {
		writeError(w, r, http.StatusNotFound, middleware.NOT_FOUND, "event not found")
		return
	}

	httputil.JSON(w, http.StatusOK, event)
}

func isNotFound(err error) bool {
	_, ok := err.(ErrEventNotFound)
	return ok
}

func isInvalidState(err error) bool {
	_, ok := err.(ErrInvalidState)
	return ok
}

func parseOutboxFilters(r *http.Request) model.OutboxFilters {
	q := r.URL.Query()
	var filters model.OutboxFilters

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

// BatchDispatchResponse holds the result of a batch dispatch operation.
type BatchDispatchResponse struct {
	DryRun     bool     `json:"dry_run"`
	Dispatched int      `json:"dispatched"`
	Failed     int      `json:"failed"`
	EventIDs   []string `json:"event_ids"`
}

// HandleBatchDispatch handles POST /outbox/dispatch.
func (h *OutboxHandler) HandleBatchDispatch(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, http.StatusNotImplemented, middleware.INTERNAL_ERROR, "not implemented")
}

// dtoFromOutboxListResponse converts model.OutboxListResponse to dto.OutboxListResponse.
func dtoFromOutboxListResponse(m *model.OutboxListResponse) *dto.OutboxListResponse {
	if m == nil {
		return nil
	}

	items := make([]dto.OutboxItem, len(m.Items))
	for i, item := range m.Items {
		items[i] = dto.OutboxItem{
			OutboxID:         item.OutboxID,
			EventType:        item.EventType,
			SourceType:       item.SourceType,
			SourceID:         item.SourceID,
			TargetChannel:    item.TargetChannel,
			Status:           item.Status,
			CreatedAt:        item.CreatedAt,
			DispatchAttempts: item.DispatchAttempts,
			LastDispatchAt:   item.LastDispatchAt,
		}
	}

	return &dto.OutboxListResponse{
		Items: items,
		Total: m.Total,
	}
}
