package service

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"baxi/internal/model"
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
// mapping repository rows to model types.
func (s *TaskService) ListTasks(
	ctx context.Context,
	filters model.TaskFilters,
	limit, offset int,
) (*model.TaskListResponse, error) {
	repoFilters := repository.TaskFilters{
		Status:   filters.Status,
		Priority: filters.Priority,
		Owner:    filters.Owner,
	}

	rows, total, err := s.repo.ListTasks(ctx, s.pool, repoFilters, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	items := make([]model.Task, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapRowToTask(row))
	}

	return &model.TaskListResponse{
		Items: items,
		Total: total,
	}, nil
}

// mapRowToTask converts a repository row to a model Task.
// alert_id in PostgreSQL maps to event_id in the model.
func mapRowToTask(row repository.TaskRow) model.Task {
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

	return model.Task{
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
