// Package service provides business logic between HTTP handlers and data repositories.
package service

import (
	"context"
	"fmt"

	"baxi/internal/model"
	outboxRepo "baxi/internal/repository/outbox"
)

// OutboxService handles business logic for outbox event operations.
type OutboxService struct {
	repo *outboxRepo.Repository
}

// NewOutboxService creates a new OutboxService.
func NewOutboxService(repo *outboxRepo.Repository) *OutboxService {
	return &OutboxService{repo: repo}
}

// List returns paginated outbox events matching the given filters.
// Maps repository rows to model types and returns a backward-compatible response.
func (s *OutboxService) List(ctx context.Context, filters model.OutboxFilters, limit, offset int) (*model.OutboxListResponse, error) {
	repoFilters := outboxRepo.OutboxFilters{
		Status:    filters.Status,
		Channel:   filters.Channel,
		EventType: filters.EventType,
	}

	rows, total, err := s.repo.ListOutboxEvents(ctx, repoFilters, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list outbox events: %w", err)
	}

	items := make([]model.OutboxEvent, len(rows))
	for i, row := range rows {
		items[i] = model.OutboxEvent{
			OutboxID:         row.OutboxID,
			EventType:        row.EventType,
			SourceType:       row.SourceType,
			SourceID:         row.SourceID,
			TargetChannel:    row.TargetChannel,
			Status:           row.Status,
			CreatedAt:        row.CreatedAt,
			DispatchAttempts: row.DispatchAttempts,
			LastDispatchAt:   row.LastDispatchAt,
		}
	}

	return &model.OutboxListResponse{
		Items: items,
		Total: total,
	}, nil
}
