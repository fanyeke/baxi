package model

// LogEntry represents a single related log entry from a diagnosis source.
type LogEntry struct {
	Source    string
	Ts        string
	Timestamp string
	ErrorCode string
	Message   string
	Diagnosis string
	OutboxID  string
	Status    string
	Error     string
	Action    string
}

// DiagnosisResponse is the structured diagnosis result for a request_id lookup.
type DiagnosisResponse struct {
	RequestID       string
	Summary         string
	ErrorCode       string
	Diagnosis       string
	SuggestedAction string
	RelatedLogs     []LogEntry
}
