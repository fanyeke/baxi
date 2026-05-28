package api

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"baxi/internal/api/dto"
)

// handleListTasks handles GET /api/v1/tasks.
// Query params: status, priority, owner, limit, offset.
func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	pagination, err := ParsePagination(r)
	if err != nil {
		JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	var filters dto.TaskFilters

	if status := r.URL.Query().Get("status"); status != "" {
		filters.Status = &status
	}
	if priority := r.URL.Query().Get("priority"); priority != "" {
		filters.Priority = &priority
	}
	if owner := r.URL.Query().Get("owner"); owner != "" {
		filters.Owner = &owner
	}

	resp, err := s.taskSvc.ListTasks(r.Context(), filters, pagination.Limit, pagination.Offset)
	if err != nil {
		s.logger.Error("failed to list tasks", zap.Error(err))
		JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
