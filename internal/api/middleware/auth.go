package middleware

import (
	"context"
	"crypto/subtle"
	"log"
	"net/http"
	"strings"
)

// ActorKey is the context key for storing the authenticated actor name.
const ActorKey contextKey = "actor"

// GetActor retrieves the authenticated actor name from the request context.
// Returns "unknown" if no actor has been authenticated.
func GetActor(ctx context.Context) string {
	if actor, ok := ctx.Value(ActorKey).(string); ok {
		return actor
	}
	return "unknown"
}

// knownWeakTokens is a set of placeholder/example tokens rejected as
// defense-in-depth against accidental deployment with weak credentials.
var knownWeakTokens = map[string]struct{}{
	"test-token":             {},
	"changeme":               {},
	"admin":                  {},
	"password":               {},
	"REPLACE_ME":             {},
	"secret":                 {},
	"your-secret-token-here": {},
	"sk-your-key-here":       {},
}

func isWeakToken(token string) bool {
	_, ok := knownWeakTokens[token]
	return ok
}

// NewAuthMiddleware creates a Bearer Token authentication middleware.
//
// It extracts the Authorization: Bearer <token> header from incoming requests
// and validates it against expectedToken using constant-time comparison via
// crypto/subtle.ConstantTimeCompare.
//
// As defense-in-depth, it also rejects tokens shorter than 32 characters and
// known weak/placeholder tokens before performing the comparison.
//
// publicPaths specifies URL paths that should bypass authentication (e.g.,
// health check endpoints). If no publicPaths are provided, the middleware
// defaults to ["/api/v1/health"].
//
// On successful authentication, the user identity is extracted from the token
// (or defaults to legacy "qoder" for opaque tokens) and stored in context.
// Use GetActor(ctx) for the username string or GetIdentity(ctx) for the full identity.
func NewAuthMiddleware(expectedToken string, publicPaths ...string) func(next http.Handler) http.Handler {
	if isWeakToken(expectedToken) {
		log.Printf("WARNING: API_BEARER_TOKEN is set to a known weak/placeholder value; all requests will be rejected")
	}
	if len(expectedToken) < 32 && expectedToken != "" {
		log.Printf("WARNING: API_BEARER_TOKEN is shorter than 32 characters; all requests will be rejected")
	}

	if len(publicPaths) == 0 {
		publicPaths = []string{"/api/v1/health"}
	}

	publicPathSet := make(map[string]struct{}, len(publicPaths))
	for _, p := range publicPaths {
		publicPathSet[p] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := publicPathSet[r.URL.Path]; ok {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				WriteError(w, r, http.StatusUnauthorized, UNAUTHORIZED,
					"Missing Authorization header",
					"Request must include an Authorization: Bearer <token> header",
					"Provide a valid API bearer token in the Authorization header",
				)
				return
			}

			if !strings.HasPrefix(authHeader, "Bearer ") {
				WriteError(w, r, http.StatusUnauthorized, UNAUTHORIZED,
					"Invalid Authorization header format",
					"Authorization header must use the format: Bearer <token>",
					"Ensure the Authorization header starts with 'Bearer ' followed by the token",
				)
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token == "" {
				WriteError(w, r, http.StatusUnauthorized, UNAUTHORIZED,
					"Missing token",
					"Authorization header is present but no token was provided after 'Bearer '",
					"Provide a token in the format: Bearer <token>",
				)
				return
			}

			if len(token) < 32 {
				WriteError(w, r, http.StatusUnauthorized, UNAUTHORIZED,
					"Invalid token",
					"Token is too short; minimum length is 32 characters",
					"Provide a valid API bearer token with sufficient length",
				)
				return
			}

			if isWeakToken(token) {
				WriteError(w, r, http.StatusUnauthorized, UNAUTHORIZED,
					"Invalid token",
					"Token is a known weak/placeholder value",
					"Provide a valid API bearer token",
				)
				return
			}

			if subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) != 1 {
				WriteError(w, r, http.StatusUnauthorized, UNAUTHORIZED,
					"Invalid token",
					"The provided bearer token is not valid",
					"Provide a valid API bearer token",
				)
				return
			}

			identity := extractIdentity(token)
			ctx := context.WithValue(r.Context(), ActorKey, identity.Username)
			ctx = context.WithValue(ctx, IdentityKey, identity)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
