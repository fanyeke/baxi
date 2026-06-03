package dto

// FieldError represents a single field-level validation error.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// ValidationError wraps a collection of field-level validation errors.
type ValidationError struct {
	Fields []FieldError `json:"fields"`
}
