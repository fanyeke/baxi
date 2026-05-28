package model

import "time"

// OutboxEvent represents an outbox event.
type OutboxEvent struct {
	OutboxID         string
	EventType        string
	SourceType       string
	SourceID         string
	TargetChannel    string
	Status           string
	DispatchAttempts int
	CreatedAt        time.Time
	LastDispatchAt   *time.Time
}

// OutboxFilters holds filter criteria for listing outbox events.
type OutboxFilters struct {
	Status    *string
	Channel   *string
	EventType *string
}

// OutboxListResponse is the paginated response for outbox listing.
type OutboxListResponse struct {
	Items []OutboxEvent
	Total int
}
