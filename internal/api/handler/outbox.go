package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"baxi/internal/api/dto"
	"baxi/internal/httputil"
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
	List(ctx context.Context, filters dto.OutboxFilters, limit, offset int) (*dto.OutboxListResponse, error)
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

func (h *OutboxHandler) HandleDispatch(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "id")
	if eventID == "" {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "event_id required"})
		return
	}

	err := h.svc.DispatchEvent(r.Context(), eventID)
	if err != nil {
		switch {
		case isNotFound(err):
			httputil.JSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		case isInvalidState(err):
			httputil.JSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		default:
			httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
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
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "event_id required"})
		return
	}

	event, err := h.svc.GetEvent(r.Context(), eventID)
	if err != nil {
		if isNotFound(err) {
			httputil.JSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	if event.Status != "pending" && event.Status != "failed" {
		httputil.JSON(w, http.StatusConflict, map[string]string{
			"error": fmt.Sprintf("event cannot be cancelled in %s state", event.Status),
		})
		return
	}

	err = h.svc.CancelEvent(r.Context(), eventID)
	if err != nil {
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
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
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "event_id required"})
		return
	}

	event, err := h.svc.GetEvent(r.Context(), eventID)
	if err != nil {
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	if event == nil {
		httputil.JSON(w, http.StatusNotFound, map[string]string{"error": "event not found"})
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
