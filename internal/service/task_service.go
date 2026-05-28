package service

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"baxi/internal/api/dto"
	"baxi/internal/repository"
)

// TaskService handles business logic for task-related operations.
type TaskService struct {
	repo *repository.TaskRepository
	pool *pgxpool.Pool
}

// NewTaskService creates a new TaskService.
func NewTaskService(repo *repository.TaskRepository, pool *pgxpool.Pool) *TaskService {
	return &TaskService{repo: repo, pool: pool}
}

// ListTasks retrieves tasks with optional filters and pagination,
// mapping repository rows to API DTOs.
func (s *TaskService) ListTasks(
	ctx context.Context,
	filters dto.TaskFilters,
	limit, offset int,
) (*dto.TaskListResponse, error) {
	repoFilters := repository.TaskFilters{
		Status:   filters.Status,
		Priority: filters.Priority,
		Owner:    filters.Owner,
	}

	rows, total, err := s.repo.ListTasks(ctx, s.pool, repoFilters, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	items := make([]dto.TaskItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapRowToItem(row))
	}

	return &dto.TaskListResponse{
		Items: items,
		Total: total,
	}, nil
}

// mapRowToItem converts a repository row to an API DTO.
// alert_id in PostgreSQL maps to event_id in the API response.
func mapRowToItem(row repository.TaskRow) dto.TaskItem {
	desc := ""
	if row.TaskDescription != nil {
		desc = *row.TaskDescription
	}

	ownerRole := ""
	if row.OwnerRole != nil {
		ownerRole = *row.OwnerRole
	}

	priority := row.Priority
	if priority == "" {
		priority = "medium"
	}

	status := row.Status
	if status == "" {
		status = "todo"
	}

	var eventID *string
	if row.AlertID != nil {
		e := *row.AlertID
		eventID = &e
	}

	return dto.TaskItem{
		TaskID:           row.TaskID,
		TaskTitle:        row.TaskTitle,
		TaskDescription:  desc,
		Status:           status,
		Priority:         priority,
		OwnerRole:        ownerRole,
		OwnerUserID:      row.OwnerUserID,
		DueAt:            row.DueAt,
		CreatedAt:        row.CreatedAt,
		CompletedAt:      row.CompletedAt,
		Feedback:         row.Feedback,
		RecommendationID: row.RecommendationID,
		EventID:          eventID,
		TargetObjectType: row.TargetObjectType,
		TargetObjectID:   row.TargetObjectID,
	}
}
