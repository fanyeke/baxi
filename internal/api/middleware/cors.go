package middleware

import (
	"log"
	"net"
	"net/http"
	"net/url"
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

// parseOrigins parses a comma-separated string of origins into a slice of
// parsed URL values. Each origin is parsed with url.Parse and its host is
// normalized to include an explicit port number. Invalid entries are skipped
// and logged. The function returns nil for empty input.
func parseOrigins(s string) []url.URL {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]url.URL, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		u, err := url.Parse(p)
		if err != nil {
			log.Printf("invalid allowed origin %q: %v", p, err)
			continue
		}
		normalizeHostPort(u)
		result = append(result, *u)
	}
	return result
}

// isOriginAllowed checks whether the given origin is in the allowed list.
// It parses the origin URL and compares scheme and normalized host against
// each allowed origin. If the origin URL cannot be parsed, the request is
// rejected (fail closed).
func isOriginAllowed(origin string, allowedOrigins []url.URL) bool {
	u, err := url.Parse(origin)
	if err != nil {
		log.Printf("failed to parse request origin %q: %v", origin, err)
		return false
	}
	if u.Scheme == "" || u.Host == "" {
		return false
	}
	normalizeHostPort(u)
	for _, allowed := range allowedOrigins {
		if u.Scheme == allowed.Scheme && u.Host == allowed.Host {
			return true
		}
	}
	return false
}

// normalizeHostPort ensures the URL.Host includes an explicit port number.
// If the port is missing, it assigns the default port for the scheme
// (80 for http, 443 for https).
func normalizeHostPort(u *url.URL) {
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		switch u.Scheme {
		case "https":
			port = "443"
		default:
			port = "80"
		}
	}
	u.Host = net.JoinHostPort(host, port)
}
