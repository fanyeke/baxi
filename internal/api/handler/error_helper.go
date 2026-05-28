package handler

import (
	"errors"
	"net/http"

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

// classifyError maps common error types to appropriate error codes and HTTP statuses.
func classifyError(err error) (int, string) {
	if errors.Is(err, pgx.ErrNoRows) {
		return http.StatusNotFound, middleware.NOT_FOUND
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
	case middleware.INTERNAL_ERROR:
		return "Retry the request; contact support if the issue persists"
	default:
		return "Retry the request or contact support"
	}
}
