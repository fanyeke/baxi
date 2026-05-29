package api

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"baxi/internal/api/dto"
	"baxi/internal/model"
)

// handleListTasks handles GET /api/v1/tasks.
// Query params: status, priority, owner, limit, offset.
func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	pagination, err := ParsePagination(r)
	if err != nil {
		JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	var filters model.TaskFilters

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

	// Convert model to DTO
	dtoResp := dtoFromTaskListResponse(resp)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(dtoResp)
}

// dtoFromTaskListResponse converts model.TaskListResponse to dto.TaskListResponse.
func dtoFromTaskListResponse(m *model.TaskListResponse) *dto.TaskListResponse {
	if m == nil {
		return nil
	}

	items := make([]dto.TaskItem, len(m.Items))
	for i, item := range m.Items {
		items[i] = dto.TaskItem{
			TaskID:           item.TaskID,
			TaskTitle:        item.TaskTitle,
			TaskDescription:  item.TaskDescription,
			Status:           item.Status,
			Priority:         item.Priority,
			OwnerRole:        item.OwnerRole,
			OwnerUserID:      item.OwnerUserID,
			DueAt:            item.DueAt,
			CreatedAt:        item.CreatedAt,
			CompletedAt:      item.CompletedAt,
			Feedback:         item.Feedback,
			RecommendationID: item.RecommendationID,
			EventID:          item.EventID,
			TargetObjectType: item.TargetObjectType,
			TargetObjectID:   item.TargetObjectID,
		}
	}

	return &dto.TaskListResponse{
		Items: items,
		Total: m.Total,
	}
}
