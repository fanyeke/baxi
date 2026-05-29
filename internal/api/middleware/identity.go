package middleware

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// IdentityKey is the context key for storing the authenticated user identity.
const IdentityKey contextKey = "identity"

// UserIdentity represents the authenticated user's identity extracted from a token.
type UserIdentity struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	Email    string   `json:"email,omitempty"`
}

// legacyIdentity returns the default identity for legacy (non-JWT) bearer tokens.
// Legacy tokens are opaque strings that don't contain JWT claims, so we map them
// to the "qoder" service account with viewer role.
func legacyIdentity() *UserIdentity {
	return &UserIdentity{
		UserID:   "qoder",
		Username: "qoder",
		Roles:    []string{"viewer"},
	}
}

// GetIdentity retrieves the UserIdentity from the request context.
// Returns a legacy "qoder" identity if no identity has been set (backward compatible).
func GetIdentity(ctx context.Context) *UserIdentity {
	if identity, ok := ctx.Value(IdentityKey).(*UserIdentity); ok && identity != nil {
		return identity
	}
	return legacyIdentity()
}

// jwtClaims represents the minimal JWT claims we extract from the payload.
type jwtClaims struct {
	Sub   string   `json:"sub"`
	Name  string   `json:"name"`
	Roles []string `json:"roles"`
	Email string   `json:"email"`
}

// extractIdentity attempts to parse the token as a JWT (base64-encoded JSON claims
// in the payload segment). If the token is not a valid JWT or claims are missing,
// it falls back to the legacy "qoder" identity.
//
// JWT format: header.payload.signature (each segment is base64url-encoded).
// We only decode the payload segment to extract identity claims.
func extractIdentity(token string) *UserIdentity {
	claims, err := parseJWTClaims(token)
	if err != nil || claims == nil {
		return legacyIdentity()
	}

	identity := &UserIdentity{
		UserID:   claims.Sub,
		Username: claims.Name,
		Roles:    claims.Roles,
		Email:    claims.Email,
	}

	// Fill in defaults for missing fields.
	if identity.UserID == "" {
		if identity.Username != "" {
			identity.UserID = identity.Username
		} else {
			return legacyIdentity()
		}
	}
	if identity.Username == "" {
		identity.Username = identity.UserID
	}
	if len(identity.Roles) == 0 {
		identity.Roles = []string{"viewer"}
	}

	return identity
}

// jwtSecret is the HMAC key used for JWT signature verification.
// Set via SetJWTSecret during server initialization.
var jwtSecret []byte

// SetJWTSecret configures the HMAC key used for JWT verification.
// Must be called during server initialization before processing requests.
func SetJWTSecret(secret []byte) {
	jwtSecret = secret
}

// parseJWTClaims extracts claims from the JWT payload segment.
// Returns an error if the token is not in JWT format or the payload cannot be decoded.
func parseJWTClaims(token string) (*jwtClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrNotJWT
	}

	if len(jwtSecret) == 0 {
		return nil, fmt.Errorf("JWT secret not configured")
	}

	parser := jwt.NewParser(jwt.WithValidMethods([]string{"HS256"}))
	parsed, err := parser.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok || !parsed.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	result := &jwtClaims{}
	if sub, ok := claims["sub"].(string); ok {
		result.Sub = sub
	}
	if name, ok := claims["name"].(string); ok {
		result.Name = name
	}
	if email, ok := claims["email"].(string); ok {
		result.Email = email
	}
	if roles, ok := claims["roles"].([]interface{}); ok {
		for _, r := range roles {
			if s, ok := r.(string); ok {
				result.Roles = append(result.Roles, s)
			}
		}
	}

	return result, nil
}

// ErrNotJWT is returned when a token does not have JWT format (3 dot-separated segments).
var ErrNotJWT = errNotJWT{}

type errNotJWT struct{}

func (errNotJWT) Error() string { return "token is not in JWT format" }
