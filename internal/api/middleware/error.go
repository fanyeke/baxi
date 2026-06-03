package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime/debug"
)

// contextKey is used for storing request-scoped values in context.
type contextKey string

// RequestIDKey is the context key for the request ID.
const RequestIDKey contextKey = "request_id"

// APIError represents a structured JSON error response.
// Matches the old FastAPI error format exactly.
type APIError struct {
	RequestID       string      `json:"request_id"`
	ErrorCode       string      `json:"error_code"`
	Message         string      `json:"message"`
	Diagnosis       string      `json:"diagnosis"`
	SuggestedAction string      `json:"suggested_action"`
	Details         interface{} `json:"details,omitempty"`
}

// Standard error code constants.
const (
	UNAUTHORIZED        = "UNAUTHORIZED"
	FORBIDDEN           = "FORBIDDEN"
	BAD_REQUEST         = "BAD_REQUEST"
	NOT_FOUND           = "NOT_FOUND"
	DB_QUERY_FAILED     = "DB_QUERY_FAILED"
	INTERNAL_ERROR      = "INTERNAL_ERROR"
	VALIDATION_FAILED   = "VALIDATION_FAILED"
	CONFLICT            = "CONFLICT"
	SERVICE_UNAVAILABLE = "SERVICE_UNAVAILABLE"
)

// WriteError writes a structured JSON error response.
// It extracts request_id from context and marshals a 5-field error payload.
func WriteError(w http.ResponseWriter, r *http.Request, status int, code string, message string, diagnosis string, action string) {
	resp := APIError{
		RequestID:       GetRequestID(r.Context()),
		ErrorCode:       code,
		Message:         message,
		Diagnosis:       diagnosis,
		SuggestedAction: action,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

// WriteErrorWithDetails writes a structured JSON error response with optional details.
// It behaves like WriteError but also includes a details field for additional context
// such as field-level validation errors.
func WriteErrorWithDetails(w http.ResponseWriter, r *http.Request, status int, code string, message string, diagnosis string, action string, details interface{}) {
	resp := APIError{
		RequestID:       GetRequestID(r.Context()),
		ErrorCode:       code,
		Message:         message,
		Diagnosis:       diagnosis,
		SuggestedAction: action,
		Details:         details,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

// RecoveryMiddleware catches panics in upstream handlers, logs them,
// and returns a 500 INTERNAL_ERROR JSON response.
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				rid := GetRequestID(r.Context())
				log.Printf("panic recovered: request_id=%s error=%v stack=%s", rid, rec, string(debug.Stack()))
				WriteError(w, r, http.StatusInternalServerError, INTERNAL_ERROR,
					"An internal server error occurred",
					"panic recovered in request processing",
					"Please retry the request; contact support if the issue persists",
				)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
