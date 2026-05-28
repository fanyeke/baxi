package decision

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"baxi/internal/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

// --- Mocks ---

type mockCaseRepo struct {
	createCaseFn       func(ctx context.Context, pool *pgxpool.Pool, row *repository.DecisionCaseRow) error
	getCaseByIDFn      func(ctx context.Context, pool *pgxpool.Pool, caseID string) (*repository.DecisionCaseRow, error)
	getCaseBySourceFn  func(ctx context.Context, pool *pgxpool.Pool, sourceType, sourceID string) (*repository.DecisionCaseRow, error)
	updateCaseStatusFn func(ctx context.Context, pool *pgxpool.Pool, caseID string, status string, contextJSON *json.RawMessage, contextHash *string, governanceSnapshot *json.RawMessage) error
	listCasesFn        func(ctx context.Context, pool *pgxpool.Pool, filter repository.CaseFilter) ([]repository.DecisionCaseRow, int, error)
}

func (m *mockCaseRepo) CreateCase(ctx context.Context, pool *pgxpool.Pool, row *repository.DecisionCaseRow) error {
	return m.createCaseFn(ctx, pool, row)
}

func (m *mockCaseRepo) GetCaseByID(ctx context.Context, pool *pgxpool.Pool, caseID string) (*repository.DecisionCaseRow, error) {
	return m.getCaseByIDFn(ctx, pool, caseID)
}

func (m *mockCaseRepo) GetCaseBySource(ctx context.Context, pool *pgxpool.Pool, sourceType, sourceID string) (*repository.DecisionCaseRow, error) {
	return m.getCaseBySourceFn(ctx, pool, sourceType, sourceID)
}

func (m *mockCaseRepo) UpdateCaseStatus(ctx context.Context, pool *pgxpool.Pool, caseID string, status string, contextJSON *json.RawMessage, contextHash *string, governanceSnapshot *json.RawMessage) error {
	return m.updateCaseStatusFn(ctx, pool, caseID, status, contextJSON, contextHash, governanceSnapshot)
}

func (m *mockCaseRepo) ListCases(ctx context.Context, pool *pgxpool.Pool, filter repository.CaseFilter) ([]repository.DecisionCaseRow, int, error) {
	return m.listCasesFn(ctx, pool, filter)
}

type mockAlertRepo struct {
	getAlertByIDFn func(ctx context.Context, pool *pgxpool.Pool, alertID string) (*repository.AlertRow, error)
}

func (m *mockAlertRepo) GetAlertByID(ctx context.Context, pool *pgxpool.Pool, alertID string) (*repository.AlertRow, error) {
	return m.getAlertByIDFn(ctx, pool, alertID)
}

// --- Compile-time interface checks ---

var _ CaseRepository = (*repository.DecisionRepository)(nil)
var _ AlertRepository = (*repository.AlertRepository)(nil)

// --- Tests: CreateCaseFromAlert ---

func TestCreateCaseFromAlert_CreatesNewCase(t *testing.T) {
	alertID := "alert-1"
	createdBy := "system"

	caseRepo := &mockCaseRepo{
		getCaseBySourceFn: func(ctx context.Context, pool *pgxpool.Pool, sourceType, sourceID string) (*repository.DecisionCaseRow, error) {
			return nil, pgx.ErrNoRows
		},
		createCaseFn: func(ctx context.Context, pool *pgxpool.Pool, row *repository.DecisionCaseRow) error {
			assert.Equal(t, "alert", row.SourceType)
			assert.Equal(t, alertID, row.SourceID)
			assert.Equal(t, "high", *row.Severity)
			assert.Equal(t, "seller", *row.ObjectType)
			assert.Equal(t, "seller-42", *row.ObjectID)
			assert.Equal(t, "created", row.Status)
			assert.Equal(t, createdBy, *row.CreatedBy)
			return nil
		},
	}

	alertRepo := &mockAlertRepo{
		getAlertByIDFn: func(ctx context.Context, pool *pgxpool.Pool, aid string) (*repository.AlertRow, error) {
			assert.Equal(t, alertID, aid)
			return &repository.AlertRow{
				AlertID:    alertID,
				Severity:   "high",
				ObjectType: "seller",
				ObjectID:   "seller-42",
			}, nil
		},
	}

	svc := NewCaseService(caseRepo, alertRepo, nil)
	result, err := svc.CreateCaseFromAlert(context.Background(), alertID, createdBy)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "alert", result.SourceType)
	assert.Equal(t, alertID, result.SourceID)
	assert.Equal(t, "high", result.Severity)
	assert.Equal(t, "seller", result.ObjectType)
	assert.Equal(t, "seller-42", result.ObjectID)
	assert.Equal(t, "created", result.Status)
	assert.Contains(t, result.CaseID, "dc_")
}

func TestCreateCaseFromAlert_ReturnsExistingActiveCase(t *testing.T) {
	alertID := "alert-2"
	existingCaseID := "dc_existing"

	caseRepo := &mockCaseRepo{
		getCaseBySourceFn: func(ctx context.Context, pool *pgxpool.Pool, sourceType, sourceID string) (*repository.DecisionCaseRow, error) {
			return &repository.DecisionCaseRow{
				CaseID:     existingCaseID,
				Status:     "open",
				SourceType: "alert",
				SourceID:   alertID,
			}, nil
		},
		createCaseFn: func(ctx context.Context, pool *pgxpool.Pool, row *repository.DecisionCaseRow) error {
			t.Fatal("CreateCase should not be called when active case exists")
			return nil
		},
	}

	alertRepo := &mockAlertRepo{
		getAlertByIDFn: func(ctx context.Context, pool *pgxpool.Pool, aid string) (*repository.AlertRow, error) {
			return &repository.AlertRow{
				AlertID:    alertID,
				Severity:   "medium",
				ObjectType: "product",
				ObjectID:   "prod-1",
			}, nil
		},
	}

	svc := NewCaseService(caseRepo, alertRepo, nil)
	result, err := svc.CreateCaseFromAlert(context.Background(), alertID, "system")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, existingCaseID, result.CaseID)
	assert.Equal(t, "open", result.Status)
}

func TestCreateCaseFromAlert_ReturnsExistingForActiveStatuses(t *testing.T) {
	statuses := []string{"created", "open", "context_built", "proposal_generated"}

	for _, status := range statuses {
		t.Run(status, func(t *testing.T) {
			alertID := "alert-" + status
			existingCaseID := "dc_" + status

			caseRepo := &mockCaseRepo{
				getCaseBySourceFn: func(ctx context.Context, pool *pgxpool.Pool, sourceType, sourceID string) (*repository.DecisionCaseRow, error) {
					return &repository.DecisionCaseRow{
						CaseID:     existingCaseID,
						Status:     status,
						SourceType: "alert",
						SourceID:   alertID,
					}, nil
				},
				createCaseFn: func(ctx context.Context, pool *pgxpool.Pool, row *repository.DecisionCaseRow) error {
					t.Fatal("CreateCase should not be called")
					return nil
				},
			}

			alertRepo := &mockAlertRepo{
				getAlertByIDFn: func(ctx context.Context, pool *pgxpool.Pool, aid string) (*repository.AlertRow, error) {
					return &repository.AlertRow{
						AlertID:    alertID,
						Severity:   "low",
						ObjectType: "category",
						ObjectID:   "cat-1",
					}, nil
				},
			}

			svc := NewCaseService(caseRepo, alertRepo, nil)
			result, err := svc.CreateCaseFromAlert(context.Background(), alertID, "system")

			assert.NoError(t, err)
			assert.Equal(t, existingCaseID, result.CaseID)
		})
	}
}

func TestCreateCaseFromAlert_CreatesNewWhenExistingClosedOrFailed(t *testing.T) {
	statuses := []string{"closed", "failed"}

	for _, status := range statuses {
		t.Run(status, func(t *testing.T) {
			alertID := "alert-closed-1"

			caseRepo := &mockCaseRepo{
				getCaseBySourceFn: func(ctx context.Context, pool *pgxpool.Pool, sourceType, sourceID string) (*repository.DecisionCaseRow, error) {
					return &repository.DecisionCaseRow{
						CaseID:     "dc_closed",
						Status:     status,
						SourceType: "alert",
						SourceID:   alertID,
					}, nil
				},
				createCaseFn: func(ctx context.Context, pool *pgxpool.Pool, row *repository.DecisionCaseRow) error {
					return nil
				},
			}

			alertRepo := &mockAlertRepo{
				getAlertByIDFn: func(ctx context.Context, pool *pgxpool.Pool, aid string) (*repository.AlertRow, error) {
					return &repository.AlertRow{
						AlertID:    alertID,
						Severity:   "critical",
						ObjectType: "seller",
						ObjectID:   "seller-99",
					}, nil
				},
			}

			svc := NewCaseService(caseRepo, alertRepo, nil)
			result, err := svc.CreateCaseFromAlert(context.Background(), alertID, "system")

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.NotEqual(t, "dc_closed", result.CaseID)
			assert.Equal(t, "created", result.Status)
		})
	}
}

func TestCreateCaseFromAlert_AlertNotFound(t *testing.T) {
	alertID := "nonexistent"

	caseRepo := &mockCaseRepo{
		getCaseBySourceFn: func(ctx context.Context, pool *pgxpool.Pool, sourceType, sourceID string) (*repository.DecisionCaseRow, error) {
			return nil, pgx.ErrNoRows
		},
		createCaseFn: func(ctx context.Context, pool *pgxpool.Pool, row *repository.DecisionCaseRow) error {
			t.Fatal("CreateCase should not be called when alert not found")
			return nil
		},
	}

	alertRepo := &mockAlertRepo{
		getAlertByIDFn: func(ctx context.Context, pool *pgxpool.Pool, aid string) (*repository.AlertRow, error) {
			return nil, errors.New("alert not found")
		},
	}

	svc := NewCaseService(caseRepo, alertRepo, nil)
	result, err := svc.CreateCaseFromAlert(context.Background(), alertID, "system")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "get alert")
}

func TestCreateCaseFromAlert_GetCaseBySourceError(t *testing.T) {
	alertID := "alert-error"

	caseRepo := &mockCaseRepo{
		getCaseBySourceFn: func(ctx context.Context, pool *pgxpool.Pool, sourceType, sourceID string) (*repository.DecisionCaseRow, error) {
			return nil, errors.New("database connection lost")
		},
	}

	alertRepo := &mockAlertRepo{
		getAlertByIDFn: func(ctx context.Context, pool *pgxpool.Pool, aid string) (*repository.AlertRow, error) {
			return &repository.AlertRow{
				AlertID:    alertID,
				Severity:   "high",
				ObjectType: "seller",
				ObjectID:   "seller-1",
			}, nil
		},
	}

	svc := NewCaseService(caseRepo, alertRepo, nil)
	result, err := svc.CreateCaseFromAlert(context.Background(), alertID, "system")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "check existing case")
}

// --- Tests: GetCase ---

func TestGetCase(t *testing.T) {
	caseID := "dc_test"
	now := time.Now()

	caseRepo := &mockCaseRepo{
		getCaseByIDFn: func(ctx context.Context, pool *pgxpool.Pool, cid string) (*repository.DecisionCaseRow, error) {
			assert.Equal(t, caseID, cid)
			return &repository.DecisionCaseRow{
				CaseID:     caseID,
				Status:     "created",
				SourceType: "alert",
				SourceID:   "alert-1",
				CreatedAt:  now,
			}, nil
		},
	}

	svc := NewCaseService(caseRepo, &mockAlertRepo{}, nil)
	result, err := svc.GetCase(context.Background(), caseID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, caseID, result.CaseID)
	assert.Equal(t, "created", result.Status)
}

func TestGetCase_NotFound(t *testing.T) {
	caseRepo := &mockCaseRepo{
		getCaseByIDFn: func(ctx context.Context, pool *pgxpool.Pool, cid string) (*repository.DecisionCaseRow, error) {
			return nil, pgx.ErrNoRows
		},
	}

	svc := NewCaseService(caseRepo, &mockAlertRepo{}, nil)
	result, err := svc.GetCase(context.Background(), "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, result)
}

// --- Tests: ListCases ---

func TestListCases(t *testing.T) {
	now := time.Now()
	rows := []repository.DecisionCaseRow{
		{CaseID: "dc-1", Status: "created", SourceType: "alert", SourceID: "a1", CreatedAt: now, Severity: strPtr("high")},
		{CaseID: "dc-2", Status: "open", SourceType: "alert", SourceID: "a2", CreatedAt: now.Add(-1 * time.Hour), Severity: strPtr("medium")},
	}

	caseRepo := &mockCaseRepo{
		listCasesFn: func(ctx context.Context, pool *pgxpool.Pool, filter repository.CaseFilter) ([]repository.DecisionCaseRow, int, error) {
			assert.Equal(t, 10, filter.Limit)
			assert.Equal(t, 0, filter.Offset)
			return rows, 2, nil
		},
	}

	svc := NewCaseService(caseRepo, &mockAlertRepo{}, nil)
	result, err := svc.ListCases(context.Background(), CaseFilter{Limit: 10, Offset: 0})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.Total)
	assert.Len(t, result.Cases, 2)
	assert.Equal(t, "dc-1", result.Cases[0].CaseID)
	assert.Equal(t, "dc-2", result.Cases[1].CaseID)
	assert.Equal(t, "high", result.Cases[0].Severity)
	assert.Equal(t, "medium", result.Cases[1].Severity)
}

func TestListCases_WithFilters(t *testing.T) {
	severity := "critical"
	status := "created"

	caseRepo := &mockCaseRepo{
		listCasesFn: func(ctx context.Context, pool *pgxpool.Pool, filter repository.CaseFilter) ([]repository.DecisionCaseRow, int, error) {
			assert.NotNil(t, filter.Severity)
			assert.Equal(t, severity, *filter.Severity)
			assert.NotNil(t, filter.Status)
			assert.Equal(t, status, *filter.Status)
			return []repository.DecisionCaseRow{}, 0, nil
		},
	}

	svc := NewCaseService(caseRepo, &mockAlertRepo{}, nil)
	result, err := svc.ListCases(context.Background(), CaseFilter{
		Severity: &severity,
		Status:   &status,
		Limit:    10,
		Offset:   0,
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, result.Total)
	assert.Empty(t, result.Cases)
}

func TestListCases_Empty(t *testing.T) {
	caseRepo := &mockCaseRepo{
		listCasesFn: func(ctx context.Context, pool *pgxpool.Pool, filter repository.CaseFilter) ([]repository.DecisionCaseRow, int, error) {
			return []repository.DecisionCaseRow{}, 0, nil
		},
	}

	svc := NewCaseService(caseRepo, &mockAlertRepo{}, nil)
	result, err := svc.ListCases(context.Background(), CaseFilter{Limit: 10, Offset: 0})

	assert.NoError(t, err)
	assert.Equal(t, 0, result.Total)
	assert.Empty(t, result.Cases)
}

// --- Tests: UpdateCaseStatus ---

func TestUpdateCaseStatus(t *testing.T) {
	caseID := "dc-update"
	status := "context_built"

	caseRepo := &mockCaseRepo{
		updateCaseStatusFn: func(ctx context.Context, pool *pgxpool.Pool, cid string, st string, cj *json.RawMessage, ch *string, gs *json.RawMessage) error {
			assert.Equal(t, caseID, cid)
			assert.Equal(t, status, st)
			assert.Nil(t, cj)
			assert.Nil(t, ch)
			assert.Nil(t, gs)
			return nil
		},
	}

	svc := NewCaseService(caseRepo, &mockAlertRepo{}, nil)
	err := svc.UpdateCaseStatus(context.Background(), caseID, status)

	assert.NoError(t, err)
}

func TestUpdateCaseStatus_NotFound(t *testing.T) {
	caseRepo := &mockCaseRepo{
		updateCaseStatusFn: func(ctx context.Context, pool *pgxpool.Pool, cid string, st string, cj *json.RawMessage, ch *string, gs *json.RawMessage) error {
			return errors.New("decision case nonexistent not found")
		},
	}

	svc := NewCaseService(caseRepo, &mockAlertRepo{}, nil)
	err := svc.UpdateCaseStatus(context.Background(), "nonexistent", "closed")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUpdateCaseStatus_EmptyStatus(t *testing.T) {
	caseID := "dc-empty"

	caseRepo := &mockCaseRepo{
		updateCaseStatusFn: func(ctx context.Context, pool *pgxpool.Pool, cid string, st string, cj *json.RawMessage, ch *string, gs *json.RawMessage) error {
			assert.Equal(t, caseID, cid)
			assert.Empty(t, st)
			return nil
		},
	}

	svc := NewCaseService(caseRepo, &mockAlertRepo{}, nil)
	err := svc.UpdateCaseStatus(context.Background(), caseID, "")

	assert.NoError(t, err)
}

// --- Helper: strPtr for tests in this package ---

func strPtr(s string) *string {
	return &s
}
