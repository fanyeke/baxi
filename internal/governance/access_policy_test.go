package governance

import (
	"testing"

	"github.com/stretchr/testify/assert"

	governanceRepo "baxi/internal/repository/governance"
)

// ──── filterByRole ──────────────────────────────────────────────────────────

func TestFilterByRole_NilInputReturnsEmpty(t *testing.T) {
	result := filterByRole(nil, "admin")
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestFilterByRole_EmptyInputReturnsEmpty(t *testing.T) {
	result := filterByRole([]governanceRepo.AccessPolicyRow{}, "admin")
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestFilterByRole_ExactMatch(t *testing.T) {
	policies := []governanceRepo.AccessPolicyRow{
		{PolicyName: "p1", PrincipalPattern: "admin", Effect: "allow"},
		{PolicyName: "p2", PrincipalPattern: "analyst", Effect: "deny"},
	}
	result := filterByRole(policies, "admin")
	assert.Len(t, result, 1)
	assert.Equal(t, "p1", result[0].PolicyName)
}

func TestFilterByRole_NoMatch(t *testing.T) {
	policies := []governanceRepo.AccessPolicyRow{
		{PolicyName: "p1", PrincipalPattern: "admin", Effect: "allow"},
	}
	result := filterByRole(policies, "viewer")
	assert.Empty(t, result)
}

func TestFilterByRole_MultipleMatches(t *testing.T) {
	policies := []governanceRepo.AccessPolicyRow{
		{PolicyName: "p1", PrincipalPattern: "admin", Effect: "allow"},
		{PolicyName: "p2", PrincipalPattern: "admin", Effect: "deny"},
		{PolicyName: "p3", PrincipalPattern: "analyst", Effect: "allow"},
	}
	result := filterByRole(policies, "admin")
	assert.Len(t, result, 2)
}

// ──── matchesResource ───────────────────────────────────────────────────────

func TestMatchesResource_Wildcard(t *testing.T) {
	p := governanceRepo.AccessPolicyRow{ResourcePattern: "*"}
	assert.True(t, matchesResource(p, "any_type"))
}

func TestMatchesResource_EmptyPattern(t *testing.T) {
	p := governanceRepo.AccessPolicyRow{ResourcePattern: ""}
	assert.True(t, matchesResource(p, "any_type"))
}

func TestMatchesResource_ExactMatch(t *testing.T) {
	p := governanceRepo.AccessPolicyRow{ResourcePattern: "order"}
	assert.True(t, matchesResource(p, "order"))
}

func TestMatchesResource_PrefixWildcard(t *testing.T) {
	p := governanceRepo.AccessPolicyRow{ResourcePattern: "dwd_*"}
	assert.True(t, matchesResource(p, "dwd_order_level"))
	assert.True(t, matchesResource(p, "dwd_customer"))
	assert.False(t, matchesResource(p, "raw_orders"))
}

func TestMatchesResource_NoMatch(t *testing.T) {
	p := governanceRepo.AccessPolicyRow{ResourcePattern: "order"}
	assert.False(t, matchesResource(p, "customer"))
}

// ──── matchesAction ─────────────────────────────────────────────────────────

func TestMatchesAction_Wildcard(t *testing.T) {
	p := governanceRepo.AccessPolicyRow{Action: "*"}
	assert.True(t, matchesAction(p, "read"))
	assert.True(t, matchesAction(p, "write"))
	assert.True(t, matchesAction(p, "delete"))
}

func TestMatchesAction_EmptyAction(t *testing.T) {
	p := governanceRepo.AccessPolicyRow{Action: ""}
	assert.True(t, matchesAction(p, "read"))
}

func TestMatchesAction_ExactMatch(t *testing.T) {
	p := governanceRepo.AccessPolicyRow{Action: "read"}
	assert.True(t, matchesAction(p, "read"))
	assert.False(t, matchesAction(p, "write"))
}

func TestMatchesAction_PrefixWildcard(t *testing.T) {
	p := governanceRepo.AccessPolicyRow{Action: "read*"}
	assert.True(t, matchesAction(p, "read"))
	assert.True(t, matchesAction(p, "readonly"))
	assert.False(t, matchesAction(p, "write"))
}

func TestMatchesAction_NoMatch(t *testing.T) {
	p := governanceRepo.AccessPolicyRow{Action: "delete"}
	assert.False(t, matchesAction(p, "read"))
}
