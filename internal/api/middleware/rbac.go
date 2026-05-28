package middleware

import (
	"net/http"
)

// NewRBACMiddleware creates a role-based access control middleware.
// It checks that the authenticated user has at least one of the requiredRoles.
// Must be placed after NewAuthMiddleware in the middleware chain (requires identity in context).
//
// If no requiredRoles are provided, all authenticated users are allowed.
// Users with no roles are denied access (except when requiredRoles is empty).
func NewRBACMiddleware(requiredRoles ...string) func(next http.Handler) http.Handler {
	roleSet := make(map[string]struct{}, len(requiredRoles))
	for _, role := range requiredRoles {
		roleSet[role] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(roleSet) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			identity := GetIdentity(r.Context())

			if hasAnyRole(identity.Roles, roleSet) {
				next.ServeHTTP(w, r)
				return
			}

			WriteError(w, r, http.StatusForbidden, FORBIDDEN,
				"Insufficient permissions",
				"User does not have the required role(s) to access this resource",
				"Contact an administrator to request the necessary role(s)",
			)
		})
	}
}

func hasAnyRole(userRoles []string, required map[string]struct{}) bool {
	for _, role := range userRoles {
		if _, ok := required[role]; ok {
			return true
		}
	}
	return false
}
