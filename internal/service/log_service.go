package service

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"baxi/internal/api/dto"
	"baxi/internal/repository"
)

// LogService handles business logic for log-related operations.
type LogService struct {
	repo *repository.LogRepository
	pool *pgxpool.Pool
}

// NewLogService creates a new LogService.
func NewLogService(repo *repository.LogRepository, pool *pgxpool.Pool) *LogService {
	return &LogService{repo: repo, pool: pool}
}

// ListRecent retrieves a combined view of recent logs from multiple tables.
func (s *LogService) ListRecent(ctx context.Context, limit, offset int) (*dto.LogListResponse, error) {
	rows, total, err := s.repo.ListRecentLogs(ctx, s.pool, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list recent logs: %w", err)
	}

	items := make([]dto.LogItem, len(rows))
	for i, row := range rows {
		items[i] = dto.LogItem{
			LogType:   row.LogType,
			Level:     row.Level,
			Message:   row.Message,
			RequestID: row.RequestID,
			CreatedAt: row.CreatedAt,
		}
	}

	return &dto.LogListResponse{Items: items, Total: total}, nil
}

// ListErrors retrieves error logs from error_log and failed pipeline step runs.
func (s *LogService) ListErrors(ctx context.Context, limit, offset int) (*dto.LogListResponse, error) {
	rows, total, err := s.repo.ListErrorLogs(ctx, s.pool, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list error logs: %w", err)
	}

	items := make([]dto.LogItem, len(rows))
	for i, row := range rows {
		items[i] = dto.LogItem{
			LogType:   row.LogType,
			Level:     row.Level,
			Message:   row.Message,
			RequestID: row.RequestID,
			CreatedAt: row.CreatedAt,
		}
	}

	return &dto.LogListResponse{Items: items, Total: total}, nil
}

// ListAudit retrieves business audit trail entries.
func (s *LogService) ListAudit(ctx context.Context, limit, offset int) (*dto.LogListResponse, error) {
	rows, total, err := s.repo.ListAuditLogs(ctx, s.pool, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}

	items := make([]dto.LogItem, len(rows))
	for i, row := range rows {
		items[i] = dto.LogItem{
			LogType:   row.LogType,
			Level:     row.Level,
			Message:   row.Message,
			RequestID: row.RequestID,
			CreatedAt: row.CreatedAt,
		}
	}

	return &dto.LogListResponse{Items: items, Total: total}, nil
}
