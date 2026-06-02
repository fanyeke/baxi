package handler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"baxi/internal/model"
)

// ──── dtoFromOutboxListResponse ────────────────────────────────────────────

func TestDTOFromOutboxListResponse_Full(t *testing.T) {
	now := time.Now()
	m := &model.OutboxListResponse{
		Items: []model.OutboxEvent{
			{
				OutboxID:         "evt-001",
				EventType:        "pipeline.complete",
				SourceType:       "pipeline",
				SourceID:         "run-001",
				TargetChannel:    "feishu",
				Status:           "pending",
				DispatchAttempts: 2,
				CreatedAt:        now,
				LastDispatchAt:   &now,
			},
			{
				OutboxID:         "evt-002",
				EventType:        "alert.triggered",
				SourceType:       "alert",
				SourceID:         "alert-001",
				TargetChannel:    "github",
				Status:           "dispatched",
				DispatchAttempts: 1,
				CreatedAt:        now,
			},
		},
		Total: 2,
	}
	d := dtoFromOutboxListResponse(m)
	assert.NotNil(t, d)
	assert.Equal(t, 2, d.Total)
	assert.Len(t, d.Items, 2)
	assert.Equal(t, "evt-001", d.Items[0].OutboxID)
	assert.Equal(t, "pipeline.complete", d.Items[0].EventType)
	assert.Equal(t, "pending", d.Items[0].Status)
	assert.Equal(t, 2, d.Items[0].DispatchAttempts)
	assert.NotNil(t, d.Items[0].LastDispatchAt)
	assert.Equal(t, "evt-002", d.Items[1].OutboxID)
	assert.Nil(t, d.Items[1].LastDispatchAt)
}

func TestDTOFromOutboxListResponse_Empty(t *testing.T) {
	m := &model.OutboxListResponse{
		Items: []model.OutboxEvent{},
		Total: 0,
	}
	d := dtoFromOutboxListResponse(m)
	assert.NotNil(t, d)
	assert.Empty(t, d.Items)
	assert.Equal(t, 0, d.Total)
}

func TestDTOFromOutboxListResponse_Nil(t *testing.T) {
	assert.Nil(t, dtoFromOutboxListResponse(nil))
}

// ──── dtoFromPipelinePreview ───────────────────────────────────────────────

func TestDTOFromPipelinePreview_Full(t *testing.T) {
	m := &model.PipelinePreview{
		Command:           "baxi pipeline run ingest_raw",
		PipelineType:      "ingest_raw",
		EstimatedDuration: "2m30s",
		RequiredEnvVars:   []string{"API_KEY", "DB_URL"},
		Warnings:          []string{"Large dataset may take longer", "Ensure API access"},
		Description:       "Ingests raw data from source",
	}
	d := dtoFromPipelinePreview(m)
	assert.NotNil(t, d)
	assert.Equal(t, "baxi pipeline run ingest_raw", d.Command)
	assert.Equal(t, "ingest_raw", d.PipelineType)
	assert.Equal(t, "2m30s", d.EstimatedDuration)
	assert.Equal(t, []string{"API_KEY", "DB_URL"}, d.RequiredEnvVars)
	assert.Equal(t, []string{"Large dataset may take longer", "Ensure API access"}, d.Warnings)
	assert.Equal(t, "Ingests raw data from source", d.Description)
}

func TestDTOFromPipelinePreview_Empty(t *testing.T) {
	m := &model.PipelinePreview{
		RequiredEnvVars: []string{},
		Warnings:        []string{},
	}
	d := dtoFromPipelinePreview(m)
	assert.NotNil(t, d)
	assert.Empty(t, d.RequiredEnvVars)
	assert.Empty(t, d.Warnings)
}

func TestDTOFromPipelinePreview_Nil(t *testing.T) {
	assert.Nil(t, dtoFromPipelinePreview(nil))
}

// ──── dtoFromCapabilities ──────────────────────────────────────────────────

func TestDTOFromCapabilities_Full(t *testing.T) {
	m := model.CapabilitiesResponse{
		Mode:              "read_only",
		Version:           "0.6.0",
		CanReadStatus:     true,
		CanReadAlerts:     true,
		CanReadTasks:      true,
		CanReadOutbox:     true,
		CanReadGovernance: true,
		CanReadLogs:       true,
		CanWriteReports:   false,
		CanExecuteActions: false,
	}
	d := dtoFromCapabilities(m)
	assert.Equal(t, "read_only", d.Mode)
	assert.Equal(t, "0.6.0", d.Version)
	assert.True(t, d.CanReadStatus)
	assert.True(t, d.CanReadGovernance)
	assert.False(t, d.CanWriteReports)
	assert.False(t, d.CanExecuteActions)
}

func TestDTOFromCapabilities_Mixed(t *testing.T) {
	m := model.CapabilitiesResponse{
		Mode:              "interactive",
		Version:           "1.0.0",
		CanReadStatus:     false,
		CanReadAlerts:     false,
		CanReadTasks:      true,
		CanReadOutbox:     false,
		CanReadGovernance: true,
		CanReadLogs:       false,
		CanWriteReports:   true,
		CanExecuteActions: true,
	}
	d := dtoFromCapabilities(m)
	assert.Equal(t, "interactive", d.Mode)
	assert.True(t, d.CanReadTasks)
	assert.True(t, d.CanReadGovernance)
	assert.False(t, d.CanReadStatus)
	assert.False(t, d.CanReadAlerts)
	assert.True(t, d.CanWriteReports)
	assert.True(t, d.CanExecuteActions)
}
