package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"
)

// RequestIDMiddleware generates or propagates request IDs.
// It checks for an incoming X-Request-ID header; if present, it uses that value.
// Otherwise, it generates a new request ID in the format req_<timestamp_hex>_<random_hex>.
// The request ID is stored in the request context and set as a response header.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// generateRequestID creates a new request ID in the format req_<timestamp_hex>_<random_hex>.
func generateRequestID() string {
	now := time.Now().UnixNano()
	timestampHex := fmt.Sprintf("%x", now)

	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback: use nanosecond timestamp only if crypto/rand fails
		return fmt.Sprintf("req_%s", timestampHex)
	}

	return fmt.Sprintf("req_%s_%s", timestampHex, hex.EncodeToString(randomBytes))
}

// GetRequestID retrieves the request ID from the context.
// Returns "unknown" if no request ID is set.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return "unknown"
}
