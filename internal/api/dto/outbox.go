// Package dto provides data transfer objects for API responses.
package dto

import "time"

// OutboxItem represents a single outbox event in the API response.
type OutboxItem struct {
	OutboxID         string     `json:"outbox_id"`
	EventType        string     `json:"event_type"`
	SourceType       string     `json:"source_type"`
	SourceID         string     `json:"source_id"`
	TargetChannel    string     `json:"target_channel"`
	Status           string     `json:"status"`
	CreatedAt        time.Time  `json:"created_at"`
	DispatchAttempts int        `json:"dispatch_attempts"`
	LastDispatchAt   *time.Time `json:"last_dispatch_at"`
}

// OutboxListResponse is the top-level response for GET /outbox.
// Uses the backward-compatible {items, total} format matching the old Python API.
type OutboxListResponse struct {
	Items []OutboxItem `json:"items"`
	Total int          `json:"total"`
}

// OutboxFilters holds optional query filters for listing outbox events.
type OutboxFilters struct {
	Status    *string
	Channel   *string
	EventType *string
}
