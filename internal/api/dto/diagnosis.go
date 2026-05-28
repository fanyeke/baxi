// Package dto provides data transfer objects for API responses.
package dto

// DiagnosisLogEntry represents a single related log entry from a diagnosis source.
type DiagnosisLogEntry struct {
	Source    string `json:"source"`
	Ts        string `json:"ts,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	ErrorCode string `json:"error_code,omitempty"`
	Message   string `json:"message,omitempty"`
	Diagnosis string `json:"diagnosis,omitempty"`
	OutboxID  string `json:"outbox_id,omitempty"`
	Status    string `json:"status,omitempty"`
	Error     string `json:"error,omitempty"`
	Action    string `json:"action,omitempty"`
}

// DiagnosisResponse is the structured diagnosis result for a request_id lookup.
type DiagnosisResponse struct {
	RequestID       string                `json:"request_id"`
	Summary         string                `json:"summary"`
	ErrorCode       string                `json:"error_code"`
	Diagnosis       string                `json:"diagnosis"`
	SuggestedAction string                `json:"suggested_action"`
	RelatedLogs     []DiagnosisLogEntry   `json:"related_logs"`
}
