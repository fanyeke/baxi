package handler

import (
	"errors"
	"net/http"
	"strings"

	"baxi/internal/api/dto"
	"baxi/internal/api/middleware"

	"github.com/jackc/pgx/v5"
)

// writeError is a convenience wrapper around middleware.WriteError.
// It provides sensible defaults for diagnosis and suggested_action based on the error code.
func writeError(w http.ResponseWriter, r *http.Request, status int, code string, message string) {
	diagnosis := defaultDiagnosis(code)
	action := defaultAction(code)
	middleware.WriteError(w, r, status, code, message, diagnosis, action)
}

// writeErrorWithDetails allows overriding diagnosis and suggested_action.
func writeErrorWithDetails(w http.ResponseWriter, r *http.Request, status int, code string, message string, diagnosis string, action string) {
	middleware.WriteError(w, r, status, code, message, diagnosis, action)
}

// isDatabaseConnectionError checks if an error is related to database connection failure.
func isDatabaseConnectionError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "connect: connection refused") ||
		strings.Contains(msg, "pool closed") ||
		strings.Contains(msg, "failed to connect")
}

// classifyError maps common error types to appropriate error codes and HTTP statuses.
func classifyError(err error) (int, string) {
	if errors.Is(err, pgx.ErrNoRows) {
		return http.StatusNotFound, middleware.NOT_FOUND
	}
	if isDatabaseConnectionError(err) {
		return http.StatusServiceUnavailable, middleware.SERVICE_UNAVAILABLE
	}
	return http.StatusInternalServerError, middleware.INTERNAL_ERROR
}

// writeServiceError handles service-layer errors by classifying them and writing the appropriate response.
func writeServiceError(w http.ResponseWriter, r *http.Request, err error, fallbackMessage string) {
	status, code := classifyError(err)
	message := fallbackMessage
	if code == middleware.NOT_FOUND {
		message = "The requested resource was not found"
	}
	writeError(w, r, status, code, message)
}

// writeValidationError writes a 400 Bad Request response with field-level validation details.
func writeValidationError(w http.ResponseWriter, r *http.Request, message string, fields []dto.FieldError) {
	details := map[string]interface{}{
		"fields": fields,
	}
	middleware.WriteErrorWithDetails(w, r, http.StatusBadRequest, middleware.VALIDATION_FAILED,
		message, defaultDiagnosis(middleware.VALIDATION_FAILED),
		defaultAction(middleware.VALIDATION_FAILED), details)
}

// writeDatabaseError writes an appropriate error response based on the database error type.
// For connection errors, it sets Retry-After header and returns 503 Service Unavailable.
// For other database errors, it returns 500 Internal Server Error.
func writeDatabaseError(w http.ResponseWriter, r *http.Request, err error, message string) {
	if isDatabaseConnectionError(err) {
		w.Header().Set("Retry-After", "5")
		middleware.WriteError(w, r, http.StatusServiceUnavailable, middleware.SERVICE_UNAVAILABLE,
			message, "Database connection failed. The service may be temporarily unavailable.",
			"Wait a few seconds and retry the request")
		return
	}
	writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, message)
}

// defaultDiagnosis returns a human-readable diagnosis based on the error code.
func defaultDiagnosis(code string) string {
	switch code {
	case middleware.UNAUTHORIZED:
		return "The request lacks valid authentication credentials"
	case middleware.FORBIDDEN:
		return "The server understood the request but refuses to authorize it"
	case middleware.BAD_REQUEST:
		return "The request contains invalid parameters or malformed data"
	case middleware.NOT_FOUND:
		return "The requested resource does not exist at the specified path"
	case middleware.DB_QUERY_FAILED:
		return "A database query failed to execute successfully"
	case middleware.VALIDATION_FAILED:
		return "The request data failed validation checks"
	case middleware.CONFLICT:
		return "The request conflicts with the current state of the resource"
	case middleware.SERVICE_UNAVAILABLE:
		return "The service is temporarily unavailable"
	case middleware.INTERNAL_ERROR:
		return "An unexpected error occurred while processing the request"
	default:
		return "An error occurred while processing the request"
	}
}

// defaultAction returns a suggested action based on the error code.
func defaultAction(code string) string {
	switch code {
	case middleware.UNAUTHORIZED:
		return "Include a valid Authorization header with a Bearer token"
	case middleware.FORBIDDEN:
		return "Check that you have permission to access this resource"
	case middleware.BAD_REQUEST:
		return "Review the request parameters and try again"
	case middleware.NOT_FOUND:
		return "Verify the resource ID and try again"
	case middleware.DB_QUERY_FAILED:
		return "Retry the request; contact support if the issue persists"
	case middleware.VALIDATION_FAILED:
		return "Fix the validation errors and retry"
	case middleware.CONFLICT:
		return "Review the resource state and retry with updated data"
	case middleware.SERVICE_UNAVAILABLE:
		return "Wait a moment and retry the request"
	case middleware.INTERNAL_ERROR:
		return "Retry the request; contact support if the issue persists"
	default:
		return "Retry the request or contact support"
	}
}
