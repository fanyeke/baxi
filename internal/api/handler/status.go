package handler

import (
	"context"
	"net/http"

	"baxi/internal/api/dto"
	"baxi/internal/api/middleware"
	"baxi/internal/httputil"
	"baxi/internal/model"
)

// StatusGetter is the interface for getting system status. Used by StatusHandler so
// tests can substitute a mock without importing the service package.
type StatusGetter interface {
	GetStatus(ctx context.Context) (*model.StatusResponse, error)
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
		writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "internal server error")
		return
	}

	// Convert model to DTO
	dtoResp := dtoFromStatusResponse(resp)
	httputil.JSON(w, http.StatusOK, dtoResp)
}

// dtoFromStatusResponse converts model.StatusResponse to dto.StatusResponse.
func dtoFromStatusResponse(m *model.StatusResponse) *dto.StatusResponse {
	if m == nil {
		return nil
	}

	database := dto.DatabaseInfo{
		Path:   m.Database.Path,
		Exists: m.Database.Exists,
		Tables: m.Database.Tables,
	}

	var pipelineRun *dto.PipelineRun
	if m.LastPipelineRun != nil {
		pipelineRun = &dto.PipelineRun{
			RunID:        m.LastPipelineRun.RunID,
			RunType:      m.LastPipelineRun.RunType,
			Mode:         m.LastPipelineRun.Mode,
			Status:       m.LastPipelineRun.Status,
			StartedAt:    m.LastPipelineRun.StartedAt,
			FinishedAt:   m.LastPipelineRun.FinishedAt,
			InputCount:   m.LastPipelineRun.InputCount,
			OutputCount:  m.LastPipelineRun.OutputCount,
			ErrorMessage: m.LastPipelineRun.ErrorMessage,
		}
	}

	return &dto.StatusResponse{
		Database:        database,
		LastPipelineRun: pipelineRun,
		Version:         m.Version,
	}
}
