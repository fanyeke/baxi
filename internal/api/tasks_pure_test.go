package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"baxi/internal/model"
)

// ──── dtoFromTaskListResponse ──────────────────────────────────────────────

func TestDTOFromTaskListResponse_Full(t *testing.T) {
	now := time.Now()
	userID := "user-001"
	m := &model.TaskListResponse{
		Items: []model.Task{
			{
				TaskID:           "task-001",
				TaskTitle:        "Review data quality",
				TaskDescription:  "Check the latest pipeline run for anomalies",
				Status:           "open",
				Priority:         "high",
				OwnerRole:        "analyst",
				OwnerUserID:      &userID,
				DueAt:            &now,
				CreatedAt:        now,
				CompletedAt:      nil,
				Feedback:         nil,
				RecommendationID: nil,
				EventID:          nil,
				TargetObjectType: nil,
				TargetObjectID:   nil,
			},
			{
				TaskID:           "task-002",
				TaskTitle:        "Approve access request",
				TaskDescription:  "",
				Status:           "completed",
				Priority:         "medium",
				OwnerRole:        "admin",
				OwnerUserID:      nil,
				CreatedAt:        now,
				CompletedAt:      &now,
				Feedback:         strPtr("Approved on time"),
			},
		},
		Total: 2,
	}
	d := dtoFromTaskListResponse(m)
	assert.NotNil(t, d)
	assert.Equal(t, 2, d.Total)
	assert.Len(t, d.Items, 2)

	// First item
	assert.Equal(t, "task-001", d.Items[0].TaskID)
	assert.Equal(t, "Review data quality", d.Items[0].TaskTitle)
	assert.Equal(t, "Check the latest pipeline run for anomalies", d.Items[0].TaskDescription)
	assert.Equal(t, "open", d.Items[0].Status)
	assert.Equal(t, "high", d.Items[0].Priority)
	assert.Equal(t, "analyst", d.Items[0].OwnerRole)
	assert.NotNil(t, d.Items[0].OwnerUserID)
	assert.Equal(t, userID, *d.Items[0].OwnerUserID)

	// Second item
	assert.Equal(t, "task-002", d.Items[1].TaskID)
	assert.Equal(t, "completed", d.Items[1].Status)
	assert.Nil(t, d.Items[1].OwnerUserID)
	assert.NotNil(t, d.Items[1].CompletedAt)
	assert.NotNil(t, d.Items[1].Feedback)
	assert.Equal(t, "Approved on time", *d.Items[1].Feedback)
}

func TestDTOFromTaskListResponse_Empty(t *testing.T) {
	m := &model.TaskListResponse{
		Items: []model.Task{},
		Total: 0,
	}
	d := dtoFromTaskListResponse(m)
	assert.NotNil(t, d)
	assert.Empty(t, d.Items)
	assert.Equal(t, 0, d.Total)
}

func TestDTOFromTaskListResponse_Nil(t *testing.T) {
	assert.Nil(t, dtoFromTaskListResponse(nil))
}

func TestDTOFromTaskListResponse_MatchesDTO(t *testing.T) {
	now := time.Now()
	nilStr := (*string)(nil)
	task := model.Task{
		TaskID:           "t1",
		TaskTitle:        "Title",
		TaskDescription:  "Desc",
		Status:           "open",
		Priority:         "low",
		OwnerRole:        "viewer",
		OwnerUserID:      nilStr,
		DueAt:            nil,
		CreatedAt:        now,
		CompletedAt:      nil,
		Feedback:         nilStr,
		RecommendationID: nilStr,
		EventID:          nilStr,
		TargetObjectType: nilStr,
		TargetObjectID:   nilStr,
	}
	m := &model.TaskListResponse{Items: []model.Task{task}, Total: 1}
	d := dtoFromTaskListResponse(m)

	// Verify all nil pointer fields transfer correctly
	var nilDst *string
	assert.Equal(t, nilDst, d.Items[0].OwnerUserID)
	assert.Equal(t, nilDst, d.Items[0].Feedback)
	assert.Nil(t, d.Items[0].DueAt)
	assert.Nil(t, d.Items[0].CompletedAt)
}

func strPtr(s string) *string { return &s }
