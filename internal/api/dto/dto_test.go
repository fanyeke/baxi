package dto

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestStaticCapabilities_Default(t *testing.T) {
	c := StaticCapabilities()

	if c.Mode != "read_only" {
		t.Errorf("Mode = %q, want %q", c.Mode, "read_only")
	}
	if c.Version != "0.6.0" {
		t.Errorf("Version = %q, want %q", c.Version, "0.6.0")
	}
	if !c.CanReadStatus {
		t.Error("CanReadStatus should be true")
	}
	if !c.CanReadAlerts {
		t.Error("CanReadAlerts should be true")
	}
	if !c.CanReadTasks {
		t.Error("CanReadTasks should be true")
	}
	if c.CanWriteReports {
		t.Error("CanWriteReports should be false")
	}
	if c.CanExecuteActions {
		t.Error("CanExecuteActions should be false")
	}
}

func TestCapabilitiesResponse_AllowedActions_FullReadOnly(t *testing.T) {
	c := StaticCapabilities()
	actions := c.AllowedActions()

	expected := []string{"read_status", "read_alerts", "read_tasks", "read_outbox", "read_governance", "read_logs"}
	if !reflect.DeepEqual(actions, expected) {
		t.Errorf("AllowedActions() = %v, want %v", actions, expected)
	}
}

func TestCapabilitiesResponse_AllowedActions_Partial(t *testing.T) {
	c := CapabilitiesResponse{
		CanReadStatus: true,
		CanReadAlerts: false,
		CanReadTasks:  true,
	}
	actions := c.AllowedActions()

	expected := []string{"read_status", "read_tasks"}
	if !reflect.DeepEqual(actions, expected) {
		t.Errorf("AllowedActions() = %v, want %v", actions, expected)
	}
}

func TestCapabilitiesResponse_AllowedActions_None(t *testing.T) {
	c := CapabilitiesResponse{}
	actions := c.AllowedActions()

	if len(actions) != 0 {
		t.Errorf("AllowedActions() = %v, want empty", actions)
	}
}

func TestCapabilitiesResponse_ForbiddenActions_FullReadOnly(t *testing.T) {
	c := StaticCapabilities()
	actions := c.ForbiddenActions()

	expected := []string{"write_reports", "execute_actions"}
	if !reflect.DeepEqual(actions, expected) {
		t.Errorf("ForbiddenActions() = %v, want %v", actions, expected)
	}
}

func TestCapabilitiesResponse_ForbiddenActions_Partial(t *testing.T) {
	c := CapabilitiesResponse{
		CanWriteReports:   true,
		CanExecuteActions: false,
	}
	actions := c.ForbiddenActions()

	expected := []string{"execute_actions"}
	if !reflect.DeepEqual(actions, expected) {
		t.Errorf("ForbiddenActions() = %v, want %v", actions, expected)
	}
}

func TestCapabilitiesResponse_ForbiddenActions_None(t *testing.T) {
	c := CapabilitiesResponse{
		CanWriteReports:   true,
		CanExecuteActions: true,
	}
	actions := c.ForbiddenActions()

	if len(actions) != 0 {
		t.Errorf("ForbiddenActions() = %v, want empty", actions)
	}
}

func TestCreateCaseRequest_JSONTags(t *testing.T) {
	req := CreateCaseRequest{SourceType: "alert", SourceID: "alert-123"}
	data, _ := json.Marshal(req)

	got := string(data)
	if !contains(got, `"source_type":"alert"`) {
		t.Errorf("expected source_type in JSON, got %s", got)
	}
	if !contains(got, `"source_id":"alert-123"`) {
		t.Errorf("expected source_id in JSON, got %s", got)
	}
}

func TestCreateCaseResponse_JSONTags(t *testing.T) {
	resp := CreateCaseResponse{
		DecisionCaseID: "case-123",
		SourceType:     "alert",
		SourceID:       "alert-456",
		Status:         "open",
	}
	data, _ := json.Marshal(resp)

	got := string(data)
	if !contains(got, `"decision_case_id":"case-123"`) {
		t.Errorf("expected decision_case_id in JSON, got %s", got)
	}
}

func TestDecisionCaseResponse_OmitsEmptyFields(t *testing.T) {
	resp := DecisionCaseResponse{
		DecisionCaseID: "case-123",
		Status:         "open",
	}
	data, _ := json.Marshal(resp)

	got := string(data)
	if contains(got, `"object_type":`) {
		t.Errorf("expected omitempty for object_type, got %s", got)
	}
}

func TestAlertItem_JSONTags(t *testing.T) {
	val := 42.5
	item := AlertItem{
		EventID:     "evt-1",
		RuleID:      "rule-1",
		Severity:    "high",
		ImpactScore: &val,
	}

	data, _ := json.Marshal(item)
	got := string(data)
	if !contains(got, `"event_id":"evt-1"`) {
		t.Errorf("expected event_id in JSON, got %s", got)
	}
	if !contains(got, `"impact_score":42.5`) {
		t.Errorf("expected impact_score in JSON, got %s", got)
	}
}

func TestLogItem_Serialization(t *testing.T) {
	rid := "req-123"
	item := LogItem{
		LogType:   "system",
		Level:     "error",
		Message:   "something failed",
		RequestID: &rid,
		CreatedAt: time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
	}

	data, _ := json.Marshal(item)
	got := string(data)
	if !contains(got, `"log_type":"system"`) {
		t.Errorf("expected log_type in JSON, got %s", got)
	}
	if !contains(got, `"request_id":"req-123"`) {
		t.Errorf("expected request_id in JSON, got %s", got)
	}
}

func TestOutboxListResponse_Items(t *testing.T) {
	resp := OutboxListResponse{
		Items: []OutboxItem{
			{OutboxID: "out-1", EventType: "task_assigned"},
			{OutboxID: "out-2", EventType: "notification"},
		},
		Total: 2,
	}

	data, _ := json.Marshal(resp)
	got := string(data)
	if !contains(got, `"items"`) {
		t.Errorf("expected items field, got %s", got)
	}
}

func TestTaskFilters_DefaultNil(t *testing.T) {
	var f TaskFilters
	if f.Status != nil {
		t.Error("expected Status to be nil by default")
	}
}

func TestOutboxFilters_DefaultNil(t *testing.T) {
	var f OutboxFilters
	if f.Status != nil {
		t.Error("expected Status to be nil by default")
	}
}

func TestAlertFilters_Values(t *testing.T) {
	f := AlertFilters{
		Severity: "high",
		Status:   "open",
		RuleID:   "rule-gmv",
	}
	if f.Severity != "high" {
		t.Errorf("Severity = %q, want %q", f.Severity, "high")
	}
}

func TestPipelineRunResponse_JSON(t *testing.T) {
	resp := PipelineRunResponse{RunID: "run-abc", Status: "running"}
	data, _ := json.Marshal(resp)

	got := string(data)
	if !contains(got, `"run_id":"run-abc"`) {
		t.Errorf("expected run_id in JSON, got %s", got)
	}
}

func TestStatusResponse_JSON(t *testing.T) {
	resp := StatusResponse{
		Version: "0.6.0",
		Database: DatabaseInfo{
			Path:   "postgres://localhost:5432/baxi",
			Exists: true,
			Tables: map[string]int{"alerts": 42},
		},
	}
	data, _ := json.Marshal(resp)

	got := string(data)
	if !contains(got, `"tables"`) {
		t.Errorf("expected tables in JSON, got %s", got)
	}
	if !contains(got, `"version":"0.6.0"`) {
		t.Errorf("expected version in JSON, got %s", got)
	}
}

func TestContextResponse_AllowedActions(t *testing.T) {
	resp := ContextResponse{
		RequestID:       "req-1",
		AllowedActions:  []string{"read_alerts", "read_tasks"},
		ForbiddenActions: []string{"write_reports"},
	}

	data, _ := json.Marshal(resp)
	got := string(data)
	if !contains(got, `"allowed_actions"`) {
		t.Errorf("expected allowed_actions in JSON, got %s", got)
	}
}

func TestSandboxResponse_JSON(t *testing.T) {
	resp := SandboxResponse{
		SandboxID: "sb-1",
		CaseID:    "case-1",
		Data:      map[string]interface{}{"key": "value"},
		Status:    "active",
	}
	data, _ := json.Marshal(resp)

	got := string(data)
	if !contains(got, `"sandbox_id":"sb-1"`) {
		t.Errorf("expected sandbox_id in JSON, got %s", got)
	}
}

func TestCapabilitiesResponse_Version(t *testing.T) {
	c := StaticCapabilities()
	if c.Version == "" {
		t.Error("Version should not be empty")
	}
}
