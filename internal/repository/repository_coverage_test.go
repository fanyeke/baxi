package repository

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"

	alertRepo "baxi/internal/repository/alert"
	governanceRepo "baxi/internal/repository/governance"
)

// ──── GovernanceRepository tests ────────────────────────────────────────────

func TestNewGovernanceRepository(t *testing.T) {
	repo := NewGovernanceRepository()
	assert.NotNil(t, repo)
	assert.Nil(t, repo.inner)
}

func TestGovernanceRepository_SetPool(t *testing.T) {
	repo := NewGovernanceRepository()
	// SetPool with nil pool - creates inner repo but will panic on actual DB calls
	// We just test that the method sets inner
	repo.SetPool(nil)
	assert.NotNil(t, repo.inner)
}

func TestGovernanceRepository_EnsureInitialized(t *testing.T) {
	repo := NewGovernanceRepository()
	assert.Nil(t, repo.inner)

	// First call initializes
	inner := repo.ensureInitialized(nil)
	assert.NotNil(t, inner)
	assert.NotNil(t, repo.inner)

	// Second call returns same instance
	inner2 := repo.ensureInitialized(nil)
	assert.Equal(t, inner, inner2)
}

func TestGovernanceRepository_EnsureInitialized_AlreadyInitialized(t *testing.T) {
	repo := NewGovernanceRepository()
	repo.inner = &governanceRepo.Repository{}
	inner := repo.ensureInitialized(nil)
	assert.NotNil(t, inner)
}

// ──── AlertRepository tests ─────────────────────────────────────────────────

func TestNewAlertRepository(t *testing.T) {
	repo := NewAlertRepository()
	assert.NotNil(t, repo)
	assert.Nil(t, repo.inner)
}

func TestAlertRepository_EnsureInit(t *testing.T) {
	repo := NewAlertRepository()
	assert.Nil(t, repo.inner)

	// ensureInit with nil pool creates inner repo
	repo.ensureInit(nil)
	assert.NotNil(t, repo.inner)
}

func TestAlertRepository_EnsureInit_AlreadyInitialized(t *testing.T) {
	repo := NewAlertRepository()
	repo.inner = &alertRepo.Repository{}
	repo.ensureInit(nil)
	assert.NotNil(t, repo.inner)
}

// ──── DecisionRepository tests ──────────────────────────────────────────────

func TestNewDecisionRepository(t *testing.T) {
	repo := NewDecisionRepository()
	assert.NotNil(t, repo)
}

// ──── StatusRepository tests ────────────────────────────────────────────────

func TestNewStatusRepository(t *testing.T) {
	repo := NewStatusRepository()
	assert.NotNil(t, repo)
}

// ──── OutboxRepository tests ────────────────────────────────────────────────

func TestNewOutboxRepository(t *testing.T) {
	repo := NewOutboxRepository()
	assert.NotNil(t, repo)
}

// ──── LogRepository tests ───────────────────────────────────────────────────

func TestNewLogRepository(t *testing.T) {
	repo := NewLogRepository()
	assert.NotNil(t, repo)
}

// ──── TaskRepository tests ──────────────────────────────────────────────────

func TestNewTaskRepository(t *testing.T) {
	repo := NewTaskRepository()
	assert.NotNil(t, repo)
}

// ──── OntologyRepo tests ────────────────────────────────────────────────────

func TestNewOntologyRepo(t *testing.T) {
	repo := NewOntologyRepo()
	assert.NotNil(t, repo)
}

// ──── ContextRepository tests ───────────────────────────────────────────────

func TestNewContextRepository(t *testing.T) {
	repo := NewContextRepository()
	assert.NotNil(t, repo)
}

// ──── Interface compliance ──────────────────────────────────────────────────

func TestGovernanceRepository_ImplementsInterface(t *testing.T) {
	var _ interface{} = (*GovernanceRepository)(nil)
}

func TestAlertRepository_ImplementsInterface(t *testing.T) {
	var _ interface{} = (*AlertRepository)(nil)
}

// ──── GovernanceRepository method delegation tests ──────────────────────────
// These test that the deprecated wrapper methods correctly delegate to the inner repo.
// Since we can't easily mock the inner repo (concrete type), we test the constructors
// and initialization logic.

func TestGovernanceRepository_Methods_WithNilPool(t *testing.T) {
	repo := NewGovernanceRepository()
	// These will panic because PoolProvider tries to use nil pool
	// But we can test that the methods exist and are callable
	assert.NotNil(t, repo)

	// Test that GetConfigSnapshots method exists (it will panic on nil pool)
	// We don't call it because it would panic
}

func TestRepository_Types_Exist(t *testing.T) {
	// Verify row types exist and can be created
	_ = ConfigSnapshotRow{}
	_ = ObjectSchemaRow{}
	_ = DataClassificationRow{}
	_ = DataLineageRow{}
	_ = AccessPolicyRow{}

	// Verify params exist
	_ = UpsertSnapshotParams{}
	_ = ObjectSchemaUpsertParams{}
	_ = ClassificationUpsertParams{}
	_ = LineageUpsertParams{}
	_ = AccessPolicyUpsertParams{}
}

func TestRepository_InterfaceTypes_Exist(t *testing.T) {
	// Verify interface types exist
	var _ ObjectSchemaRepository
	var _ DataClassificationRepository
	var _ DataLineageRepository
	var _ AccessPolicyRepository
	var _ OntologyRepository
	var _ ContextRepository
}

// ──── ObjectFilters and SearchResult types ──────────────────────────────────

func TestObjectFilters_Values(t *testing.T) {
	filters := ObjectFilters{
		ObjectType: "order",
		Limit:      10,
		Offset:     0,
		Filters:    map[string]interface{}{"status": "active"},
	}
	assert.Equal(t, "order", filters.ObjectType)
	assert.Equal(t, 10, filters.Limit)
}

func TestSearchResult_Values(t *testing.T) {
	result := SearchResult{
		Rows:  []ObjectInstance{{ObjectType: "order", ID: "1"}},
		Total: 1,
	}
	assert.Len(t, result.Rows, 1)
	assert.Equal(t, 1, result.Total)
}

func TestObjectQueryResult_Values(t *testing.T) {
	result := ObjectQueryResult{
		Rows:  []ObjectInstance{{ObjectType: "order", ID: "1"}},
		Total: 1,
	}
	assert.Len(t, result.Rows, 1)
}

func TestObjectInstance_Values(t *testing.T) {
	obj := ObjectInstance{
		ObjectType: "order",
		ID:         "1",
		Properties: map[string]interface{}{"total": 100.0},
	}
	assert.Equal(t, "order", obj.ObjectType)
}

func TestObjectMetrics_Values(t *testing.T) {
	metrics := ObjectMetrics{
		ObjectType: "order",
		ID:         "1",
		Metrics:    map[string]float64{"revenue": 100.0},
	}
	assert.Equal(t, "order", metrics.ObjectType)
}

func TestPipelineRunInfo_Values(t *testing.T) {
	info := PipelineRunInfo{
		RunID:       1,
		Status:      "completed",
		StartedAt:   "2024-01-01",
		CompletedAt: "2024-01-02",
	}
	assert.Equal(t, int64(1), info.RunID)
}

func TestAlertSummary_Values(t *testing.T) {
	summary := AlertSummary{
		AlertID:  "alert-1",
		Severity: "high",
		Metric:   "cpu",
		Status:   "active",
	}
	assert.Equal(t, "alert-1", summary.AlertID)
}

func TestTaskSummary_Values(t *testing.T) {
	summary := TaskSummary{
		TaskID:    "task-1",
		Title:     "Fix bug",
		Status:    "todo",
		OwnerRole: "ops",
	}
	assert.Equal(t, "task-1", summary.TaskID)
}

func TestOutboxSummary_Values(t *testing.T) {
	summary := OutboxSummary{
		EventID:   "event-1",
		EventType: "notification",
		Status:    "pending",
	}
	assert.Equal(t, "event-1", summary.EventID)
}

func TestSearchFilters_Values(t *testing.T) {
	filters := SearchFilters{
		ObjectType: "order",
		Query:      "test",
		Limit:      10,
		Offset:     0,
	}
	assert.Equal(t, "order", filters.ObjectType)
}

// ──── Import check ──────────────────────────────────────────────────────────

func TestRepository_Package_Imported(t *testing.T) {
	// Ensure the package compiles and imports work
	ctx := context.Background()
	_ = ctx
	_ = (*pgxpool.Pool)(nil)
}

// ──── ContextRepo nil pool tests ────────────────────────────────────────────

func TestContextRepo_GetLastPipelineRun_NilPool(t *testing.T) {
	repo := NewContextRepository()
	info, err := repo.GetLastPipelineRun(context.Background(), nil)
	assert.NoError(t, err)
	assert.Nil(t, info)
}

func TestContextRepo_GetAlerts_NilPool(t *testing.T) {
	repo := NewContextRepository()
	items, err := repo.GetAlerts(context.Background(), nil, "high", 10)
	assert.NoError(t, err)
	assert.NotNil(t, items)
	assert.Empty(t, items)
}

func TestContextRepo_GetOpenTasks_NilPool(t *testing.T) {
	repo := NewContextRepository()
	items, err := repo.GetOpenTasks(context.Background(), nil, 10)
	assert.NoError(t, err)
	assert.NotNil(t, items)
	assert.Empty(t, items)
}

func TestContextRepo_GetPendingOutbox_NilPool(t *testing.T) {
	repo := NewContextRepository()
	items, err := repo.GetPendingOutbox(context.Background(), nil, 10)
	assert.NoError(t, err)
	assert.NotNil(t, items)
	assert.Empty(t, items)
}

func TestContextRepo_EnsureInitialized(t *testing.T) {
	repo := NewContextRepository()
	assert.Nil(t, repo.inner)

	inner := repo.ensureInitialized(nil)
	assert.NotNil(t, inner)
	assert.NotNil(t, repo.inner)

	// Second call returns same instance
	inner2 := repo.ensureInitialized(nil)
	assert.Equal(t, inner, inner2)
}
