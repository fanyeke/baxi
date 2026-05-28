// Package service provides business logic between HTTP handlers and data repositories.
package service

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"baxi/internal/api/dto"
	"baxi/internal/repository"
)

// OutboxService handles business logic for outbox event operations.
type OutboxService struct {
	repo *repository.OutboxRepository
	pool *pgxpool.Pool
}

// NewOutboxService creates a new OutboxService.
func NewOutboxService(repo *repository.OutboxRepository, pool *pgxpool.Pool) *OutboxService {
	return &OutboxService{repo: repo, pool: pool}
}

// List returns paginated outbox events matching the given filters.
// Maps repository rows to DTOs and returns a backward-compatible response.
func (s *OutboxService) List(ctx context.Context, filters dto.OutboxFilters, limit, offset int) (*dto.OutboxListResponse, error) {
	repoFilters := repository.OutboxFilters{
		Status:    filters.Status,
		Channel:   filters.Channel,
		EventType: filters.EventType,
	}

	rows, total, err := s.repo.ListOutboxEvents(ctx, s.pool, repoFilters, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list outbox events: %w", err)
	}

	items := make([]dto.OutboxItem, len(rows))
	for i, row := range rows {
		items[i] = dto.OutboxItem{
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

	return &dto.OutboxListResponse{
		Items: items,
		Total: total,
	}, nil
}
