package service

import (
	"context"
	"fmt"

	"baxi/internal/api/dto"
	"baxi/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

var allowedSorts = map[string]bool{
	"created_at_desc": true,
	"created_at_asc":  true,
	"severity_desc":   true,
}

type AlertService struct {
	repo *repository.AlertRepository
	pool *pgxpool.Pool
}

func NewAlertService(repo *repository.AlertRepository, pool *pgxpool.Pool) *AlertService {
	return &AlertService{repo: repo, pool: pool}
}

func (s *AlertService) ListAlerts(
	ctx context.Context,
	filters dto.AlertFilters,
	sort string,
	limit, offset int,
) (*dto.AlertListResponse, error) {
	if sort == "" || !allowedSorts[sort] {
		sort = "created_at_desc"
	}

	rows, total, err := s.repo.ListAlerts(
		ctx, s.pool,
		filters.Severity, filters.Status, filters.ObjectType, filters.RuleID,
		sort, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list alerts: %w", err)
	}

	items := make([]dto.AlertItem, len(rows))
	for i, row := range rows {
		items[i] = dto.AlertItem{
			EventID:       row.AlertID,
			RuleID:        row.RuleID,
			EventDate:     row.EventDate,
			Severity:      row.Severity,
			MetricName:    row.MetricName,
			ObjectType:    row.ObjectType,
			ObjectID:      row.ObjectID,
			CurrentValue:  row.CurrentValue,
			BaselineValue: row.BaselineValue,
			ChangeRate:    row.ChangeRate,
			OwnerRole:     row.OwnerRole,
			Status:        row.Status,
			ImpactScore:   row.ImpactScore,
		}
	}

	return &dto.AlertListResponse{
		Items: items,
		Total: total,
	}, nil
}

