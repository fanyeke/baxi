// Package dto provides data transfer objects for API request/response payloads.
package dto

import "time"

// TaskItem represents a single task in the API response.
// Matches the old Python FastAPI ActionTask response format exactly.
type TaskItem struct {
	TaskID           string     `json:"task_id"`
	TaskTitle        string     `json:"task_title"`
	TaskDescription  string     `json:"task_description"`
	Status           string     `json:"status"`
	Priority         string     `json:"priority"`
	OwnerRole        string     `json:"owner_role"`
	OwnerUserID      *string    `json:"owner_user_id"`
	DueAt            *time.Time `json:"due_at"`
	CreatedAt        time.Time  `json:"created_at"`
	CompletedAt      *time.Time `json:"completed_at"`
	Feedback         *string    `json:"feedback"`
	RecommendationID *string    `json:"recommendation_id"`
	EventID          *string    `json:"event_id"`
	TargetObjectType *string    `json:"target_object_type"`
	TargetObjectID   *string    `json:"target_object_id"`
}

// TaskFilters holds optional filter criteria for listing tasks.
// All fields are pointers; nil means "no filter" for that field.
type TaskFilters struct {
	Status   *string
	Priority *string
	Owner    *string // maps to owner_role in DB
}

// TaskListResponse is the top-level response for GET /api/v1/tasks.
// Uses the backward-compatible {items, total} format from the old FastAPI.
type TaskListResponse struct {
	Items []TaskItem `json:"items"`
	Total int        `json:"total"`
}
