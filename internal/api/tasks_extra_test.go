package api

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"baxi/internal/model"
)

func TestDtoFromTaskListResponse_Nil(t *testing.T) {
	result := dtoFromTaskListResponse(nil)
	assert.Nil(t, result)
}

func TestDtoFromTaskListResponse_Empty(t *testing.T) {
	input := &model.TaskListResponse{
		Items: []model.Task{},
		Total: 0,
	}
	result := dtoFromTaskListResponse(input)
	assert.NotNil(t, result)
	assert.Equal(t, 0, result.Total)
	assert.Empty(t, result.Items)
}

func TestDtoFromTaskListResponse_WithItems(t *testing.T) {
	input := &model.TaskListResponse{
		Items: []model.Task{
			{
				TaskID:    "task-1",
				TaskTitle: "Test task",
				Status:    "todo",
				Priority:  "high",
			},
			{
				TaskID:    "task-2",
				TaskTitle: "Another task",
				Status:    "done",
				Priority:  "low",
			},
		},
		Total: 2,
	}
	result := dtoFromTaskListResponse(input)
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.Total)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, "task-1", result.Items[0].TaskID)
	assert.Equal(t, "Test task", result.Items[0].TaskTitle)
	assert.Equal(t, "task-2", result.Items[1].TaskID)
}
