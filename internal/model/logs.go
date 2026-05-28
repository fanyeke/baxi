package model

import "time"

// LogItem represents a single log entry.
type LogItem struct {
	LogType   string
	Level     string
	Message   string
	RequestID *string
	CreatedAt time.Time
}

// LogListResponse is the paginated response for log listing.
type LogListResponse struct {
	Items []LogItem
	Total int
}
