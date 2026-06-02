package steps

import (
	"encoding/json"
	"testing"
)

func TestBuildEvent_HeuristicStrategy(t *testing.T) {
	tr := taskRow{
		TaskID:           "task-gmv_drop_2018-10-17",
		RecommendationID: "rec-gmv_drop_2018-10-17",
		AlertID:          "gmv_drop_2018-10-17",
		TaskTitle:        "Review gmv anomaly",
		TaskDescription:  "GMV 7日均值较前14天均值下降超过15%",
		TargetObjectType: "global",
		TargetObjectID:   "global",
		TaskSource:       "heuristic_strategy",
		OwnerRole:        "business_ops",
		Priority:         "high",
	}

	got, err := (&CreateOutboxStep{}).buildEvent(tr)
	if err != nil {
		t.Fatalf("buildEvent() error = %v", err)
	}
	if got == nil {
		t.Fatal("buildEvent() returned nil")
	}

	// Check event fields
	if got.EventID != "outbox-task-gmv_drop_2018-10-17" {
		t.Errorf("EventID = %q, want %q", got.EventID, "outbox-task-gmv_drop_2018-10-17")
	}
	if got.EventType != "task_assigned" {
		t.Errorf("EventType = %q, want %q", got.EventType, "task_assigned")
	}
	if got.SourceType != "task" {
		t.Errorf("SourceType = %q, want %q", got.SourceType, "task")
	}
	if got.SourceID != "task-gmv_drop_2018-10-17" {
		t.Errorf("SourceID = %q, want %q", got.SourceID, "task-gmv_drop_2018-10-17")
	}
	if got.Status != "pending" {
		t.Errorf("Status = %q, want %q", got.Status, "pending")
	}
	if got.TargetChannel != "local_cli" {
		t.Errorf("TargetChannel = %q, want %q", got.TargetChannel, "local_cli")
	}
}

func TestBuildEvent_DimensionalRule(t *testing.T) {
	tr := taskRow{
		TaskID:           "dimtask-dim-76085bfcd31d",
		RecommendationID: "dimrec-dim-76085bfcd31d",
		AlertID:          "dim-76085bfcd31d",
		TaskTitle:        "排查区域 SP 延迟配送",
		TaskDescription:  "区域延迟配送率超过20%且样本>=30单",
		TargetObjectType: "region",
		TargetObjectID:   "SP",
		TaskSource:       "dimensional_rule",
		OwnerRole:        "logistics_ops",
		Priority:         "high",
	}

	got, err := (&CreateOutboxStep{}).buildEvent(tr)
	if err != nil {
		t.Fatalf("buildEvent() error = %v", err)
	}

	// Dimensional tasks → feishu_cli
	if got.TargetChannel != "feishu_cli" {
		t.Errorf("TargetChannel = %q, want %q", got.TargetChannel, "feishu_cli")
	}
}

func TestBuildEvent_EventID(t *testing.T) {
	tr := taskRow{
		TaskID: "task-abc-123",
	}

	got, err := (&CreateOutboxStep{}).buildEvent(tr)
	if err != nil {
		t.Fatalf("buildEvent() error = %v", err)
	}

	expectedID := "outbox-task-abc-123"
	if got.EventID != expectedID {
		t.Errorf("EventID = %q, want %q", got.EventID, expectedID)
	}
}

func TestBuildEvent_PayloadJSON(t *testing.T) {
	tr := taskRow{
		TaskID:           "task-test-1",
		RecommendationID: "rec-test-1",
		AlertID:          "alert-test-1",
		TaskTitle:        "Test Title",
		TaskDescription:  "Test Description",
		TargetObjectType: "region",
		TargetObjectID:   "BJ",
		TaskSource:       "heuristic_strategy",
		OwnerRole:        "ops",
		Priority:         "medium",
	}

	got, err := (&CreateOutboxStep{}).buildEvent(tr)
	if err != nil {
		t.Fatalf("buildEvent() error = %v", err)
	}

	// Verify payload can be unmarshalled and contains expected fields
	var payload map[string]interface{}
	if err := json.Unmarshal(got.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	checks := []struct {
		key   string
		value string
	}{
		{"task_id", "task-test-1"},
		{"recommendation_id", "rec-test-1"},
		{"alert_id", "alert-test-1"},
		{"task_title", "Test Title"},
		{"task_source", "heuristic_strategy"},
		{"owner_role", "ops"},
		{"priority", "medium"},
	}

	for _, c := range checks {
		if got, ok := payload[c.key].(string); !ok || got != c.value {
			t.Errorf("payload[%q] = %v, want %q", c.key, payload[c.key], c.value)
		}
	}
}

func TestBuildEvent_NilEventOnError(t *testing.T) {
	// buildEvent should never error on valid inputs; verify it returns an event
	tr := taskRow{
		TaskID:           "task-minimal",
		TaskSource:       "unknown_source",
	}
	got, err := (&CreateOutboxStep{}).buildEvent(tr)
	if err != nil {
		t.Fatalf("buildEvent() with minimal fields error = %v", err)
	}
	if got == nil {
		t.Fatal("buildEvent() with minimal fields returned nil")
	}

	// Unknown task source should default to local_cli
	if got.TargetChannel != "local_cli" {
		t.Errorf("TargetChannel = %q, want %q", got.TargetChannel, "local_cli")
	}
}
