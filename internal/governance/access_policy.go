package governance

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"baxi/internal/api/dto"
	"baxi/internal/repository"
)

// AccessPolicyService provides role-based access policy evaluation.
// Default policy: deny-all unless explicitly granted.
type AccessPolicyService struct {
	pool *pgxpool.Pool
	repo *repository.GovernanceRepository
}

// NewAccessPolicyService creates a new AccessPolicyService.
func NewAccessPolicyService(pool *pgxpool.Pool, repo *repository.GovernanceRepository) *AccessPolicyService {
	return &AccessPolicyService{pool: pool, repo: repo}
}

// CheckAccess evaluates whether a user role can perform an action on an object type.
// Returns ALLOW if an explicit allow policy matches, DENY otherwise (default deny).
func (s *AccessPolicyService) CheckAccess(ctx context.Context, userRole, objectType, action string) dto.AccessDecision {
	policies, err := s.repo.GetAccessPoliciesByRole(ctx, s.pool, userRole)
	if err != nil {
		return dto.AccessDenied
	}

	if len(policies) == 0 {
		// Fallback: load all policies and filter in-memory
		allPolicies, err := s.repo.GetAccessPolicies(ctx, s.pool)
		if err != nil {
			return dto.AccessDenied
		}
		policies = filterByRole(allPolicies, userRole)
	}

	for _, p := range policies {
		if p.Effect != "allow" {
			continue
		}
		if !matchesResource(p, objectType) {
			continue
		}
		if !matchesAction(p, action) {
			continue
		}
		if p.ConditionsJSONB != nil && len(p.ConditionsJSONB) > 4 {
			return dto.AccessConditional
		}
		return dto.AccessAllowed
	}

	return dto.AccessDenied
}

// GetAll returns all access policies from the database.
func (s *AccessPolicyService) GetAll(ctx context.Context) ([]repository.AccessPolicyRow, error) {
	return s.repo.GetAccessPolicies(ctx, s.pool)
}

// filterByRole filters a slice of policies to those matching the given role.
func filterByRole(policies []repository.AccessPolicyRow, role string) []repository.AccessPolicyRow {
	var filtered []repository.AccessPolicyRow
	for _, p := range policies {
		if p.PrincipalPattern == role {
			filtered = append(filtered, p)
		}
	}
	if filtered == nil {
		filtered = []repository.AccessPolicyRow{}
	}
	return filtered
}

// matchesResource checks if a policy's resource pattern matches the object type.
func matchesResource(p repository.AccessPolicyRow, objectType string) bool {
	if p.ResourcePattern == "*" || p.ResourcePattern == "" {
		return true
	}
	if p.ResourcePattern == objectType {
		return true
	}
	// Simple prefix matching for patterns like "dwd_*"
	if strings.HasSuffix(p.ResourcePattern, "*") {
		prefix := strings.TrimSuffix(p.ResourcePattern, "*")
		return strings.HasPrefix(objectType, prefix)
	}
	return false
}

// matchesAction checks if a policy applies to the given action.
func matchesAction(p repository.AccessPolicyRow, action string) bool {
	if p.Action == "*" || p.Action == "" {
		return true
	}
	if p.Action == action {
		return true
	}
	// Wildcard suffix matching for action patterns
	if strings.HasSuffix(p.Action, "*") {
		prefix := strings.TrimSuffix(p.Action, "*")
		return strings.HasPrefix(action, prefix)
	}
	return false
}
