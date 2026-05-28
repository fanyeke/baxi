// Package dto provides data transfer objects for API responses.
package dto

import "time"

// LogItem represents a single log entry in the API response.
type LogItem struct {
	LogType   string    `json:"log_type"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	RequestID *string   `json:"request_id"`
	CreatedAt time.Time `json:"created_at"`
}

// LogListResponse is the backward-compatible paginated response.
// Uses {"items": [...], "total": N} format matching the Phase 0 baseline.
type LogListResponse struct {
	Items []LogItem `json:"items"`
	Total int       `json:"total"`
}
