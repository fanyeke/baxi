package decision

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"baxi/internal/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DecisionCase is the domain model for ai.decision_case.
type DecisionCase struct {
	CaseID                 string
	AlertID                *string
	CaseType               string
	Status                 string
	ContextJSON            json.RawMessage
	CreatedAt              time.Time
	ResolvedAt             *time.Time
	SourceType             *string
	SourceID               *string
	ObjectType             string
	ObjectID               string
	Severity               string
	ContextHash            string
	GovernanceSnapshotJSON json.RawMessage
	CreatedBy              string
	ErrorMessage           string
	UpdatedAt              *time.Time
	AlertRulesVersion      string
	AlertRulesHash         string
	ActionRegistryVersion  string
	ActionRegistryHash     string
	ContextSnapshotJSON    json.RawMessage
	DataSnapshotJSON       json.RawMessage
}

// CaseFilter holds optional filter criteria for listing decision cases.
type CaseFilter struct {
	SourceType *string
	SourceID   *string
	Status     *string
	Severity   *string
	Limit      int
	Offset     int
}

// CaseList holds a paginated list of cases.
type CaseList struct {
	Cases []DecisionCase
	Total int
}

// CaseRepository defines the interface for decision case storage operations.
type CaseRepository interface {
	CreateCase(ctx context.Context, pool *pgxpool.Pool, row *repository.DecisionCaseRow) error
	GetCaseByID(ctx context.Context, pool *pgxpool.Pool, caseID string) (*repository.DecisionCaseRow, error)
	GetCaseBySource(ctx context.Context, pool *pgxpool.Pool, sourceType, sourceID *string) (*repository.DecisionCaseRow, error)
	UpdateCaseStatus(ctx context.Context, pool *pgxpool.Pool, caseID string, status string, contextJSON *json.RawMessage, contextHash *string, governanceSnapshot *json.RawMessage) error
	ListCases(ctx context.Context, pool *pgxpool.Pool, filter repository.CaseFilter) ([]repository.DecisionCaseRow, int, error)
}

// AlertRepository defines the interface for reading alert data.
type AlertRepository interface {
	GetAlertByID(ctx context.Context, pool *pgxpool.Pool, alertID string) (*repository.AlertRow, error)
}

// CaseService handles decision case lifecycle operations.
type CaseService struct {
	caseRepo  CaseRepository
	alertRepo AlertRepository
	pool      *pgxpool.Pool
}

// NewCaseService creates a new CaseService.
func NewCaseService(caseRepo CaseRepository, alertRepo AlertRepository, pool *pgxpool.Pool) *CaseService {
	return &CaseService{
		caseRepo:  caseRepo,
		alertRepo: alertRepo,
		pool:      pool,
	}
}

// CreateCaseFromAlert creates a new decision case from an alert.
// Implements idempotency: if an active case already exists for the alert,
// returns the existing case instead of creating a duplicate.
func (s *CaseService) CreateCaseFromAlert(ctx context.Context, alertID, createdBy string) (*DecisionCase, error) {
	alert, err := s.alertRepo.GetAlertByID(ctx, s.pool, alertID)
	if err != nil {
		return nil, fmt.Errorf("get alert %s: %w", alertID, err)
	}

	sourceType := "alert"
	existing, err := s.caseRepo.GetCaseBySource(ctx, s.pool, &sourceType, &alertID)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("check existing case for alert %s: %w", alertID, err)
		}
	} else if existing != nil && existing.Status != "closed" && existing.Status != "failed" {
		return rowToCase(existing), nil
	}

	now := time.Now()
	caseID := GenerateCaseID()
	sourceTypeVal := "alert"
	row := &repository.DecisionCaseRow{
		CaseID:     caseID,
		AlertID:    &alertID,
		SourceType: &sourceTypeVal,
		SourceID:   &alertID,
		ObjectType: &alert.ObjectType,
		ObjectID:   &alert.ObjectID,
		Severity:   &alert.Severity,
		Status:     "created",
		CreatedBy:  &createdBy,
		CreatedAt:  now,
	}

	if err := s.caseRepo.CreateCase(ctx, s.pool, row); err != nil {
		return nil, fmt.Errorf("create case for alert %s: %w", alertID, err)
	}

	return rowToCase(row), nil
}

// GetCase retrieves a single decision case by its ID.
func (s *CaseService) GetCase(ctx context.Context, caseID string) (*DecisionCase, error) {
	row, err := s.caseRepo.GetCaseByID(ctx, s.pool, caseID)
	if err != nil {
		return nil, err
	}
	return rowToCase(row), nil
}

// ListCases returns a paginated list of decision cases matching the given filter.
func (s *CaseService) ListCases(ctx context.Context, filter CaseFilter) (*CaseList, error) {
	repoFilter := repository.CaseFilter{
		SourceType: filter.SourceType,
		SourceID:   filter.SourceID,
		Status:     filter.Status,
		Severity:   filter.Severity,
		Limit:      filter.Limit,
		Offset:     filter.Offset,
	}

	rows, total, err := s.caseRepo.ListCases(ctx, s.pool, repoFilter)
	if err != nil {
		return nil, err
	}

	cases := make([]DecisionCase, len(rows))
	for i := range rows {
		cases[i] = *rowToCase(&rows[i])
	}

	return &CaseList{Cases: cases, Total: total}, nil
}

// UpdateCaseStatus updates the status of a decision case.
func (s *CaseService) UpdateCaseStatus(ctx context.Context, caseID, status string) error {
	return s.caseRepo.UpdateCaseStatus(ctx, s.pool, caseID, status, nil, nil, nil)
}

func rowToCase(row *repository.DecisionCaseRow) *DecisionCase {
	c := &DecisionCase{
		CaseID:     row.CaseID,
		AlertID:    row.AlertID,
		Status:     row.Status,
		CreatedAt:  row.CreatedAt,
		ResolvedAt: row.ResolvedAt,
		SourceType: row.SourceType,
		SourceID:   row.SourceID,
		UpdatedAt:  row.UpdatedAt,
	}

	if row.CaseType != nil {
		c.CaseType = *row.CaseType
	}
	if row.ContextJSON != nil {
		c.ContextJSON = *row.ContextJSON
	}
	if row.ObjectType != nil {
		c.ObjectType = *row.ObjectType
	}
	if row.ObjectID != nil {
		c.ObjectID = *row.ObjectID
	}
	if row.Severity != nil {
		c.Severity = *row.Severity
	}
	if row.ContextHash != nil {
		c.ContextHash = *row.ContextHash
	}
	if row.GovernanceSnapshotJSON != nil {
		c.GovernanceSnapshotJSON = *row.GovernanceSnapshotJSON
	}
	if row.CreatedBy != nil {
		c.CreatedBy = *row.CreatedBy
	}
	if row.ErrorMessage != nil {
		c.ErrorMessage = *row.ErrorMessage
	}
	if row.AlertRulesVersion != nil {
		c.AlertRulesVersion = *row.AlertRulesVersion
	}
	if row.AlertRulesHash != nil {
		c.AlertRulesHash = *row.AlertRulesHash
	}
	if row.ActionRegistryVersion != nil {
		c.ActionRegistryVersion = *row.ActionRegistryVersion
	}
	if row.ActionRegistryHash != nil {
		c.ActionRegistryHash = *row.ActionRegistryHash
	}
	if row.ContextSnapshotJSON != nil {
		c.ContextSnapshotJSON = *row.ContextSnapshotJSON
	}
	if row.DataSnapshotJSON != nil {
		c.DataSnapshotJSON = *row.DataSnapshotJSON
	}

	return c
}
