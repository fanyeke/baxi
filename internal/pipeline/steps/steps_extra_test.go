package steps

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeJSON_AllBranches(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"valid_json", `{"key":"value"}`, `{"key":"value"}`},
		{"empty", "", "{}"},
		{"invalid_json", "not json", "{}"},
		{"numeric", "42", "42"},
		{"boolean_true", "true", "true"},
		{"boolean_false", "false", "false"},
		{"null", "null", "null"},
		{"array", `[1,2,3]`, `[1,2,3]`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeJSON(tc.input)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestDeriveOwnerRole_ExtraBranches(t *testing.T) {
	tests := []struct {
		ruleID   string
		expected string
	}{
		{"cancel_rate_spike", "logistics_ops"},
		{"review_score_drop", "category_ops"},
		{"seller_activation_gap", "seller_ops"},
		{"unknown_rule", "unassigned"},
		{"", "unassigned"},
	}
	for _, tc := range tests {
		t.Run(tc.ruleID, func(t *testing.T) {
			got := deriveOwnerRole(tc.ruleID)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestIsDimensionalTask_ExtraCases(t *testing.T) {
	tests := []struct {
		taskID   string
		expected bool
	}{
		{"dimtask-abc", true},
		{"dimtask-", true},
		{"dimtask", false},
		{"rec-abc", false},
		{"", false},
		{"task-123", false},
	}
	for _, tc := range tests {
		t.Run(tc.taskID, func(t *testing.T) {
			got := IsDimensionalTask(tc.taskID)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestDeriveTargetChannel_AllBranches(t *testing.T) {
	tests := []struct {
		source   string
		expected string
	}{
		{"dimensional_rule", "feishu_cli"},
		{"heuristic_strategy", "local_cli"},
		{"unknown_source", "local_cli"},
		{"", "local_cli"},
	}
	for _, tc := range tests {
		t.Run(tc.source, func(t *testing.T) {
			got := deriveTargetChannel(tc.source)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestNewGenerateTasksStep_Name(t *testing.T) {
	step := NewGenerateTasksStep()
	assert.NotNil(t, step)
	assert.Equal(t, "generate_tasks", step.Name())
}

func TestNewGenerateRecommendationsStep_Name(t *testing.T) {
	step := NewGenerateRecommendationsStep()
	assert.NotNil(t, step)
	assert.Equal(t, "generate_recommendations", step.Name())
}

func TestNewBuildDWDSOrderLevelStep_Name(t *testing.T) {
	step := NewBuildDWDSOrderLevelStep()
	assert.NotNil(t, step)
	assert.Equal(t, "build_dwd_order_level", step.Name())
}

func TestNewBuildDWDItemLevelStep_Name(t *testing.T) {
	step := NewBuildDWDItemLevelStep()
	assert.NotNil(t, step)
	assert.Equal(t, "build_dwd_item_level", step.Name())
}

func TestNewBuildMetricDailyStep_Name(t *testing.T) {
	step := NewBuildMetricDailyStep()
	assert.NotNil(t, step)
	assert.Equal(t, "build_metric_daily", step.Name())
}

func TestNewBuildMetricDimensionDailyStep_Name(t *testing.T) {
	step := NewBuildMetricDimensionDailyStep()
	assert.NotNil(t, step)
	assert.Equal(t, "build_metric_dimension_daily", step.Name())
}

func TestNewDetectAlertsStep_Name(t *testing.T) {
	step := NewDetectAlertsStep()
	assert.NotNil(t, step)
	assert.Equal(t, "detect_alerts", step.Name())
}

func TestNewCreateOutboxStep_Name(t *testing.T) {
	step := NewCreateOutboxStep()
	assert.NotNil(t, step)
	assert.Equal(t, "create_outbox_events", step.Name())
}

func TestNewIngestRawStep_Name(t *testing.T) {
	step := NewIngestRawStep()
	assert.NotNil(t, step)
	assert.Equal(t, "ingest_raw", step.Name())
}

func TestBuildEvent_HeuristicStrategy_Extra(t *testing.T) {
	step := NewCreateOutboxStep()
	tr := taskRow{
		TaskID:           "task-001",
		RecommendationID: "rec-001",
		AlertID:          "alert-001",
		TaskTitle:        "Test task",
		TaskDescription:  "Test description",
		TargetObjectType: "order",
		TargetObjectID:   "order-001",
		TaskSource:       "heuristic_strategy",
		OwnerRole:        "business_ops",
		Priority:         "high",
	}
	event, err := step.buildEvent(tr)
	assert.NoError(t, err)
	assert.Equal(t, "outbox-task-001", event.EventID)
	assert.Equal(t, "task_assigned", event.EventType)
	assert.Equal(t, "task", event.SourceType)
	assert.Equal(t, "task-001", event.SourceID)
	assert.Equal(t, "pending", event.Status)
	assert.Equal(t, "local_cli", event.TargetChannel)
}

func TestBuildEvent_DimensionalRule_Extra(t *testing.T) {
	step := NewCreateOutboxStep()
	tr := taskRow{
		TaskID:           "dimtask-001",
		RecommendationID: "dimrec-001",
		TaskTitle:        "Dimensional task",
		TaskSource:       "dimensional_rule",
	}
	event, err := step.buildEvent(tr)
	assert.NoError(t, err)
	assert.Equal(t, "outbox-dimtask-001", event.EventID)
	assert.Equal(t, "feishu_cli", event.TargetChannel)
	assert.NotEmpty(t, event.Payload)
}

func TestMaxDimAlertsPerRun(t *testing.T) {
	assert.Equal(t, 50, maxDimAlertsPerRun)
}
