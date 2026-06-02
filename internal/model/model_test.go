package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ──── Capabilities ──────────────────────────────────────────────────────────

func strPtr(s string) *string { return &s }

func TestStaticCapabilities(t *testing.T) {
	caps := StaticCapabilities()
	assert.Equal(t, "read_only", caps.Mode)
	assert.Equal(t, "0.6.0", caps.Version)
	assert.True(t, caps.CanReadStatus)
	assert.True(t, caps.CanReadAlerts)
	assert.True(t, caps.CanReadTasks)
	assert.True(t, caps.CanReadOutbox)
	assert.True(t, caps.CanReadGovernance)
	assert.True(t, caps.CanReadLogs)
	assert.False(t, caps.CanWriteReports)
	assert.False(t, caps.CanExecuteActions)
}

func TestAllowedActions_AllReadOnly(t *testing.T) {
	caps := StaticCapabilities()
	actions := caps.AllowedActions()
	assert.Contains(t, actions, "read_status")
	assert.Contains(t, actions, "read_alerts")
	assert.Contains(t, actions, "read_tasks")
	assert.Contains(t, actions, "read_outbox")
	assert.Contains(t, actions, "read_governance")
	assert.Contains(t, actions, "read_logs")
	assert.NotContains(t, actions, "write_reports")
	assert.NotContains(t, actions, "execute_actions")
}

func TestAllowedActions_Partial(t *testing.T) {
	caps := CapabilitiesResponse{
		CanReadStatus: true,
		CanReadAlerts: false,
		CanReadTasks:  true,
	}
	actions := caps.AllowedActions()
	assert.Contains(t, actions, "read_status")
	assert.Contains(t, actions, "read_tasks")
	assert.NotContains(t, actions, "read_alerts")
	assert.NotContains(t, actions, "read_logs")
}

func TestForbiddenActions_AllPermitted(t *testing.T) {
	caps := CapabilitiesResponse{}
	forbidden := caps.ForbiddenActions()
	assert.Contains(t, forbidden, "write_reports")
	assert.Contains(t, forbidden, "execute_actions")
}

func TestForbiddenActions_NoneForbidden(t *testing.T) {
	caps := CapabilitiesResponse{
		CanWriteReports:   true,
		CanExecuteActions: true,
	}
	forbidden := caps.ForbiddenActions()
	assert.Empty(t, forbidden)
}

// ──── Struct Construction ───────────────────────────────────────────────────

func TestAlertCreation(t *testing.T) {
	a := Alert{
		EventID:    "evt_001",
		RuleID:     "rule_001",
		Severity:   "high",
		MetricName: "fraud_score",
		ObjectType: "order",
		ObjectID:   "ord_42",
		Status: "new",
	}
	assert.Equal(t, "evt_001", a.EventID)
	assert.Equal(t, "high", a.Severity)
	assert.Equal(t, "new", a.Status)
}

func TestTaskDefaults(t *testing.T) {
	task := Task{
		TaskID:    "task_001",
		TaskTitle: "Review flagged order",
		Status:    StatusTodo,
		Priority:  PriorityHigh,
		CreatedAt: time.Now(),
	}
	assert.Equal(t, StatusTodo, task.Status)
	assert.Equal(t, PriorityHigh, task.Priority)
	assert.NotNil(t, task.CreatedAt)
}

func TestOutboxEventCreation(t *testing.T) {
	now := time.Now()
	e := OutboxEvent{
		OutboxID:      "ob_001",
		EventType:     "order.flagged",
		SourceType:    "alert",
		SourceID:      "alert_001",
		TargetChannel: "feishu",
		Status: "pending",
		CreatedAt:     now,
	}
	assert.Equal(t, "ob_001", e.OutboxID)
	assert.Equal(t, "pending", e.Status)
}

func TestDiagnosisResponse(t *testing.T) {
	r := DiagnosisResponse{
		RequestID:       "req_001",
		Summary:         "Order flagged for review",
		ErrorCode:       "ERR_001",
		Diagnosis:       "Suspicious transaction pattern",
		SuggestedAction: "Manual review required",
		RelatedLogs:     []LogEntry{{Source: "pipeline", Message: "error processing order"}},
	}
	assert.Equal(t, "req_001", r.RequestID)
	assert.Len(t, r.RelatedLogs, 1)
	assert.Equal(t, "pipeline", r.RelatedLogs[0].Source)
}

func TestPipelineInfo(t *testing.T) {
	preview := PipelinePreview{
		Command:      "run.sh",
		PipelineType: "governance",
		Warnings:     []string{"deprecated"},
	}
	assert.Equal(t, "run.sh", preview.Command)
	assert.Contains(t, preview.Warnings[0], "deprecated")
}

func TestGovernanceStatusResponse(t *testing.T) {
	gs := GovernanceStatusResponse{
		GovernanceLayer:   "data",
		Configs:           map[string]string{"version": "1.0"},
		ObjectSchemaCount: 8,
	}
	assert.Equal(t, "data", gs.GovernanceLayer)
	assert.Equal(t, 8, gs.ObjectSchemaCount)
	assert.Equal(t, "1.0", gs.Configs["version"])
}

func TestContextResponse(t *testing.T) {
	ctx := ContextResponse{
		RequestID: "req_001",
		System: SystemInfo{
			LastPipelineRun: &PipelineRunInfo{
				RunID:   "run_001",
				Status: "completed",
				InputCount:  100,
				OutputCount: 95,
			},
		},
		Summary: ContextSummary{
			TotalAlerts:    5,
			TotalOpenTasks: 3,
		},
		AllowedActions:   []string{"read_status", "read_alerts"},
		ForbiddenActions: []string{"write_reports"},
	}
	assert.Equal(t, "req_001", ctx.RequestID)
	assert.Equal(t, 5, ctx.Summary.TotalAlerts)
	assert.Equal(t, 3, ctx.Summary.TotalOpenTasks)
	assert.Equal(t, "run_001", ctx.System.LastPipelineRun.RunID)
}

func TestOntologyInfo(t *testing.T) {
	oi := OntologyInfo{
		ObjectTypes:      []string{"order", "seller", "product"},
		ObjectsAvailable: true,
	}
	assert.Len(t, oi.ObjectTypes, 3)
	assert.True(t, oi.ObjectsAvailable)
}

func TestAlertFilters(t *testing.T) {
	f := AlertFilters{
		Severity:   "high",
		Status: "new",
		ObjectType: "order",
		RuleID:     "rule_001",
	}
	assert.Equal(t, "high", f.Severity)
	assert.Equal(t, "order", f.ObjectType)
}

func TestTaskFilters(t *testing.T) {
	priority := PriorityHigh
	status := StatusTodo
	f := TaskFilters{
		Status:   &status,
		Priority: &priority,
	}
	assert.Equal(t, PriorityHigh, *f.Priority)
	assert.Equal(t, StatusTodo, *f.Status)
}

func TestCatalogResponse(t *testing.T) {
	cat := CatalogResponse{
		Objects: []CatalogObject{
			{ObjectType: "order", PropertiesCount: 5},
			{ObjectType: "seller", PropertiesCount: 3},
		},
		Datasets: []CatalogDataset{
			{Dataset: "orders", Schema: "public", Table: "orders"},
		},
	}
	assert.Len(t, cat.Objects, 2)
	assert.Len(t, cat.Datasets, 1)
}
