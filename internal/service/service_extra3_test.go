package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/model"
	outboxRepo "baxi/internal/repository/outbox"
	statusRepo "baxi/internal/repository/status"
	taskRepo "baxi/internal/repository/task"
)

// ============================================================
// Constructor tests (covers New* functions at 0%)
// ============================================================

func TestNewTaskService_NilPool(t *testing.T) {
	repo := taskRepo.NewRepository(nil)
	svc := NewTaskService(repo)
	assert.NotNil(t, svc)
	assert.NotNil(t, svc.repo)
}

func TestNewOutboxService_NilPool(t *testing.T) {
	repo := outboxRepo.NewRepository(nil)
	svc := NewOutboxService(repo)
	assert.NotNil(t, svc)
	assert.NotNil(t, svc.repo)
}

func TestNewStatusService_NilPool(t *testing.T) {
	repo := statusRepo.NewRepository(nil)
	svc := NewStatusService(repo, "test-db-url")
	assert.NotNil(t, svc)
	assert.Equal(t, "test-db-url", svc.dbURL)
}

func TestNewQoderService_NilPool(t *testing.T) {
	svc := NewQoderService(nil)
	assert.NotNil(t, svc)
	assert.Nil(t, svc.pool)
}

func TestNewLogService_NilPool(t *testing.T) {
	svc := NewLogService(nil)
	assert.NotNil(t, svc)
}

func TestNewGovernanceService_NilPool(t *testing.T) {
	svc := NewGovernanceService(nil, nil)
	assert.NotNil(t, svc)
	assert.Nil(t, svc.pool)
}

func TestNewPipelineService_Empty(t *testing.T) {
	svc := NewPipelineService("")
	assert.NotNil(t, svc)
}

func TestNewDecisionService_NilDeps(t *testing.T) {
	svc := NewDecisionService(nil, nil, nil, nil, nil)
	assert.NotNil(t, svc)
}

func TestNewAlertService_NilPool(t *testing.T) {
	svc := NewAlertService(nil)
	assert.NotNil(t, svc)
}

// ============================================================
// mapRowToTask - pure function, various edge cases
// ============================================================

func TestMapRowToTask_FullRow(t *testing.T) {
	now := time.Now().UTC()
	completedAt := now.Add(1 * time.Hour)

	row := taskRepo.TaskRow{
		TaskID:           "task-1",
		TaskTitle:        "Review anomaly",
		TaskDescription:  strPtr("Full task description"),
		Status:           "in_progress",
		Priority:         "high",
		OwnerRole:        strPtr("analyst"),
		OwnerUserID:      strPtr("user-1"),
		DueAt:            &now,
		CreatedAt:        now,
		CompletedAt:      &completedAt,
		Feedback:         strPtr("Looks good"),
		RecommendationID: strPtr("rec-1"),
		AlertID:          strPtr("alert-123"),
		TargetObjectType: strPtr("order"),
		TargetObjectID:   strPtr("ord-456"),
	}

	task := mapRowToTask(row)

	assert.Equal(t, "task-1", task.TaskID)
	assert.Equal(t, "Review anomaly", task.TaskTitle)
	assert.Equal(t, "Full task description", task.TaskDescription)
	assert.Equal(t, "in_progress", task.Status)
	assert.Equal(t, "high", task.Priority)
	assert.Equal(t, "analyst", task.OwnerRole)
	require.NotNil(t, task.OwnerUserID)
	assert.Equal(t, "user-1", *task.OwnerUserID)
	assert.NotNil(t, task.DueAt)
	assert.NotNil(t, task.CompletedAt)
	require.NotNil(t, task.Feedback)
	assert.Equal(t, "Looks good", *task.Feedback)
	require.NotNil(t, task.RecommendationID)
	assert.Equal(t, "rec-1", *task.RecommendationID)
	require.NotNil(t, task.EventID)
	assert.Equal(t, "alert-123", *task.EventID)
	require.NotNil(t, task.TargetObjectType)
	assert.Equal(t, "order", *task.TargetObjectType)
	require.NotNil(t, task.TargetObjectID)
	assert.Equal(t, "ord-456", *task.TargetObjectID)
}

func TestMapRowToTask_NullFields(t *testing.T) {
	row := taskRepo.TaskRow{
		TaskID:    "task-2",
		TaskTitle: "Minimal task",
		CreatedAt: time.Now().UTC(),
	}

	task := mapRowToTask(row)

	assert.Equal(t, "task-2", task.TaskID)
	assert.Equal(t, "", task.TaskDescription)
	assert.Equal(t, "", task.OwnerRole)
	assert.Equal(t, "medium", task.Priority)
	assert.Equal(t, "todo", task.Status)
	assert.Nil(t, task.EventID)
	assert.Nil(t, task.DueAt)
	assert.Nil(t, task.CompletedAt)
	assert.Nil(t, task.Feedback)
	assert.Nil(t, task.OwnerUserID)
	assert.Nil(t, task.RecommendationID)
	assert.Nil(t, task.TargetObjectType)
	assert.Nil(t, task.TargetObjectID)
}

func TestMapRowToTask_EmptyPriorityDefault(t *testing.T) {
	row := taskRepo.TaskRow{
		TaskID:    "task-3",
		TaskTitle: "Empty priority",
		Priority:  "",
		Status:    "done",
		CreatedAt: time.Now().UTC(),
	}

	task := mapRowToTask(row)
	assert.Equal(t, "medium", task.Priority, "empty priority should default to medium")
	assert.Equal(t, "done", task.Status, "non-empty status should be kept")
}

func TestMapRowToTask_EmptyStatusDefault(t *testing.T) {
	row := taskRepo.TaskRow{
		TaskID:    "task-4",
		TaskTitle: "Empty status",
		Status:    "",
		CreatedAt: time.Now().UTC(),
	}

	task := mapRowToTask(row)
	assert.Equal(t, "todo", task.Status, "empty status should default to todo")
}

// ============================================================
// mapRowToTaskItem - pure function, various edge cases
// ============================================================

func TestMapRowToTaskItem_FullRow(t *testing.T) {
	now := time.Now().UTC()

	row := taskRepo.TaskRow{
		TaskID:           "ti-1",
		TaskTitle:        "Fix pipeline",
		TaskDescription:  strPtr("Task item description"),
		Status:           "in_progress",
		Priority:         "high",
		OwnerRole:        strPtr("ops"),
		OwnerUserID:      strPtr("user-2"),
		DueAt:            &now,
		CreatedAt:        now,
		Feedback:         strPtr("In progress"),
		RecommendationID: strPtr("rec-2"),
		AlertID:          strPtr("alert-456"),
		TargetObjectType: strPtr("order"),
		TargetObjectID:   strPtr("ord-789"),
	}

	item := mapRowToTaskItem(row)

	assert.Equal(t, "ti-1", item.TaskID)
	assert.Equal(t, "Fix pipeline", item.TaskTitle)
	assert.Equal(t, "Task item description", item.TaskDescription)
	assert.Equal(t, "in_progress", item.Status)
	assert.Equal(t, "high", item.Priority)
	assert.Equal(t, "ops", item.OwnerRole)
	require.NotNil(t, item.OwnerUserID)
	assert.Equal(t, "user-2", *item.OwnerUserID)
	require.NotNil(t, item.EventID)
	assert.Equal(t, "alert-456", *item.EventID)
	require.NotNil(t, item.TargetObjectType)
	assert.Equal(t, "order", *item.TargetObjectType)
}

func TestMapRowToTaskItem_NullFields(t *testing.T) {
	row := taskRepo.TaskRow{
		TaskID:    "ti-2",
		TaskTitle: "Minimal item",
		CreatedAt: time.Now().UTC(),
	}

	item := mapRowToTaskItem(row)

	assert.Equal(t, "ti-2", item.TaskID)
	assert.Equal(t, "", item.TaskDescription)
	assert.Equal(t, "", item.OwnerRole)
	assert.Equal(t, "medium", item.Priority)
	assert.Equal(t, "todo", item.Status)
	assert.Nil(t, item.EventID)
	assert.Nil(t, item.OwnerUserID)
	assert.Nil(t, item.TargetObjectType)
}

// ============================================================
// QoderService with nil pool (early returns)
// ============================================================

func TestQoderService_GetContext_NilPool(t *testing.T) {
	svc := NewQoderService(nil)
	ctx := context.Background()
	params := model.ContextQueryParams{
		Severity:    "high",
		LimitAlerts: 10,
		LimitTasks:  20,
		LimitOutbox: 30,
	}

	resp, err := svc.GetContext(ctx, "req-test-1", params)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "req-test-1", resp.RequestID)

	assert.Nil(t, resp.System.LastPipelineRun)
	assert.Equal(t, 0, resp.Summary.TotalAlerts)
	assert.Equal(t, 0, resp.Summary.TotalOpenTasks)
	assert.Equal(t, 0, resp.Summary.TotalPendingOutbox)

	assert.NotNil(t, resp.TopAlerts)
	assert.Empty(t, resp.TopAlerts)
	assert.NotNil(t, resp.OpenTasks)
	assert.Empty(t, resp.OpenTasks)
	assert.NotNil(t, resp.PendingOutbox)
	assert.Empty(t, resp.PendingOutbox)

	assert.NotNil(t, resp.RecentDiagnosis)
	assert.NotNil(t, resp.AllowedActions)
	assert.NotNil(t, resp.ForbiddenActions)
	assert.True(t, resp.Ontology.ObjectsAvailable)
	assert.True(t, resp.Governance.ClassificationLoaded)
	assert.Equal(t, "analyst", resp.AgentPolicy.Role)
}

func TestQoderService_QueryLastPipelineRun_NilPool(t *testing.T) {
	svc := NewQoderService(nil)
	result := svc.queryLastPipelineRun(context.Background())
	assert.Nil(t, result)
}

func TestQoderService_QueryAlerts_NilPool(t *testing.T) {
	svc := NewQoderService(nil)
	total, items := svc.queryAlerts(context.Background(), "high", 10)
	assert.Equal(t, 0, total)
	assert.NotNil(t, items)
	assert.Empty(t, items)
}

func TestQoderService_QueryOpenTasks_NilPool(t *testing.T) {
	svc := NewQoderService(nil)
	total, items := svc.queryOpenTasks(context.Background(), 20)
	assert.Equal(t, 0, total)
	assert.NotNil(t, items)
	assert.Empty(t, items)
}

func TestQoderService_QueryPendingOutbox_NilPool(t *testing.T) {
	svc := NewQoderService(nil)
	total, items := svc.queryPendingOutbox(context.Background(), 30)
	assert.Equal(t, 0, total)
	assert.NotNil(t, items)
	assert.Empty(t, items)
}

// ============================================================
// healthStatus with nil pool
// ============================================================

func TestHealthStatus_NilPool(t *testing.T) {
	status := healthStatus(context.Background(), nil, "gov", "config_snapshot")
	assert.Equal(t, "unknown", status)
}

// ============================================================
// GovernanceService with nil pool (nil pool branches)
// ============================================================

func TestGovernanceService_GetHealthChecks_NilPool(t *testing.T) {
	svc := NewGovernanceService(nil, nil)
	resp, err := svc.GetHealthChecks(context.Background())

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "healthy", resp.Status)
	assert.Len(t, resp.Checks, 5)

	for _, check := range resp.Checks {
		assert.Equal(t, "unknown", check.Status,
			"health check %s should be 'unknown' with nil pool", check.Name)
	}
}

func TestGovernanceService_RequiresCheckpoint_SensitiveActions(t *testing.T) {
	svc := NewGovernanceService(nil, nil)

	// Built-in sensitive actions return true immediately (no repo access)
	assert.True(t, svc.RequiresCheckpoint(context.Background(), "execute_dispatch"))
	assert.True(t, svc.RequiresCheckpoint(context.Background(), "modify_business_policy"))
	assert.True(t, svc.RequiresCheckpoint(context.Background(), "trigger_pipeline"))
}

// ============================================================
// DecisionService with nil deps
// ============================================================

func TestDecisionService_NewDecisionService(t *testing.T) {
	svc := NewDecisionService(nil, nil, nil, nil, nil)
	assert.NotNil(t, svc)
}

func TestDecisionService_WithReplayService(t *testing.T) {
	svc := NewDecisionService(nil, nil, nil, nil, nil)
	svc2 := svc.WithReplayService(nil)
	assert.NotNil(t, svc2)
	assert.Equal(t, svc, svc2)
}

func TestDecisionService_WithMetrics(t *testing.T) {
	svc := NewDecisionService(nil, nil, nil, nil, nil)
	svc2 := svc.WithMetrics(nil)
	assert.NotNil(t, svc2)
	assert.Equal(t, svc, svc2)
}

func TestDecisionService_WithRuleProvider(t *testing.T) {
	svc := NewDecisionService(nil, nil, nil, nil, nil)
	svc2 := svc.WithRuleProvider(nil)
	assert.NotNil(t, svc2)
	assert.Equal(t, svc, svc2)
}

// Note: Service methods that call repositories with nil pool cause panics
// because repositories don't handle nil pools gracefully. These are covered
// by integration tests with real databases. The QoderService is an exception
// because its query methods check for nil pool before calling the database.
