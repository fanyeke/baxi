package middleware

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
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
// to the "qoder" service account with admin role for backward compatibility.
func legacyIdentity() *UserIdentity {
	return &UserIdentity{
		UserID:   "qoder",
		Username: "qoder",
		Roles:    []string{"admin"},
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

// parseJWTClaims extracts claims from the JWT payload segment.
// Returns an error if the token is not in JWT format or the payload cannot be decoded.
func parseJWTClaims(token string) (*jwtClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrNotJWT
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	var claims jwtClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}

	return &claims, nil
}

// ErrNotJWT is returned when a token does not have JWT format (3 dot-separated segments).
var ErrNotJWT = errNotJWT{}

type errNotJWT struct{}

func (errNotJWT) Error() string { return "token is not in JWT format" }
