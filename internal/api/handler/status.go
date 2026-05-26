package handler

import (
	"context"
	"net/http"

	"baxi/internal/api/dto"
	"baxi/internal/httputil"
)

// StatusGetter is the interface for getting system status. Used by StatusHandler so
// tests can substitute a mock without importing the service package.
type StatusGetter interface {
	GetStatus(ctx context.Context) (*dto.StatusResponse, error)
}

type StatusHandler struct {
	svc StatusGetter
}

func NewStatusHandler(svc StatusGetter) *StatusHandler {
	return &StatusHandler{svc: svc}
}

func (h *StatusHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.GetStatus(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	httputil.JSON(w, http.StatusOK, resp)
}
