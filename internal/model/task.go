// Package model provides domain types shared across layers.
// These types replace the reverse dependency from service → api/dto.
package model

import "time"

// Task represents a task in the domain.
type Task struct {
	TaskID           string
	TaskTitle        string
	TaskDescription  string
	Status           string
	Priority         string
	OwnerRole        string
	OwnerUserID      *string
	DueAt            *time.Time
	CreatedAt        time.Time
	CompletedAt      *time.Time
	Feedback         *string
	RecommendationID *string
	EventID          *string
	TargetObjectType *string
	TargetObjectID   *string
}

// TaskFilters holds optional filter criteria for listing tasks.
type TaskFilters struct {
	Status   *string
	Priority *string
	Owner    *string
}

// TaskListResponse is the result of listing tasks.
type TaskListResponse struct {
	Items []Task
	Total int
}
