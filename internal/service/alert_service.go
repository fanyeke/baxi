package service

import (
	"context"
	"fmt"

	"baxi/internal/model"
	alertRepo "baxi/internal/repository/alert"
)

var allowedSorts = map[string]bool{
	"created_at_desc": true,
	"created_at_asc":  true,
	"severity_desc":   true,
}

type AlertService struct {
	repo *alertRepo.Repository
}

func NewAlertService(repo *alertRepo.Repository) *AlertService {
	return &AlertService{repo: repo}
}

func (s *AlertService) ListAlerts(
	ctx context.Context,
	filters model.AlertFilters,
	sort string,
	limit, offset int,
) (*model.AlertListResponse, error) {
	if sort == "" || !allowedSorts[sort] {
		sort = "created_at_desc"
	}

	rows, total, err := s.repo.ListAlerts(
		ctx,
		filters.Severity, filters.Status, filters.ObjectType, filters.RuleID,
		sort, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list alerts: %w", err)
	}

	items := make([]model.Alert, len(rows))
	for i, row := range rows {
		ownerRole := ""
		if row.OwnerRole != nil {
			ownerRole = *row.OwnerRole
		}
		status := ""
		if row.Status != nil {
			status = *row.Status
		}
		items[i] = model.Alert{
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
			OwnerRole:     ownerRole,
			Status:        status,
			ImpactScore:   row.ImpactScore,
		}
	}

	return &model.AlertListResponse{
		Items: items,
		Total: total,
	}, nil
}
