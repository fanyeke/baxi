package middleware

import (
	"net/http"
	"strings"
)

// NewCORSMiddleware creates a CORS middleware that handles preflight requests
// and sets CORS headers for allowed origins.
//
// allowedOrigins is a comma-separated list of origins
// (e.g., "http://localhost:5173,http://localhost:3000").
// Wildcard (*) is not supported.
//
// For preflight OPTIONS requests with an allowed origin, the middleware
// returns 204 No Content with standard CORS headers.
// For actual requests with an allowed origin, it sets Access-Control-Allow-Origin.
// If the origin is not in the allowed list, no CORS headers are set
// and the request passes through to the next handler.
func NewCORSMiddleware(allowedOrigins string) func(next http.Handler) http.Handler {
	origins := parseOrigins(allowedOrigins)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" || !isOriginAllowed(origin, origins) {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", origin)

			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
				w.Header().Set("Access-Control-Max-Age", "86400")
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// parseOrigins parses a comma-separated string of origins into a slice.
// Empty strings are skipped; the function returns nil for empty input.
func parseOrigins(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// isOriginAllowed checks whether the given origin is in the allowed list.
// It performs an exact match against each allowed origin.
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if origin == allowed {
			return true
		}
	}
	return false
}
