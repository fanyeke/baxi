package decision

import (
	"regexp"
	"testing"
)

// idPattern matches the expected ID format: prefix_timestamp_6chars
var idPattern = regexp.MustCompile(`^(dc|ap|de)_\d+_[A-Za-z0-9]{6}$`)

func TestIDGen_Format(t *testing.T) {
	tests := []struct {
		name   string
		actual string
	}{
		{"GenerateCaseID", GenerateCaseID()},
		{"GenerateProposalID", GenerateProposalID()},
		{"GenerateDecisionID", GenerateDecisionID()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !idPattern.MatchString(tt.actual) {
				t.Errorf("ID %q does not match pattern %s", tt.actual, idPattern)
			}
		})
	}
}

func TestIDGen_Uniqueness(t *testing.T) {
	const n = 1000
	seen := make(map[string]bool, n)
	for i := 0; i < n; i++ {
		id := GenerateCaseID()
		if seen[id] {
			t.Fatalf("duplicate ID generated after %d iterations: %s", i, id)
		}
		seen[id] = true
	}
}

func TestIDGen_PrefixDistinction(t *testing.T) {
	caseID := GenerateCaseID()
	proposalID := GenerateProposalID()
	decisionID := GenerateDecisionID()

	if caseID[:2] != "dc" {
		t.Errorf("expected caseID prefix 'dc', got %q", caseID[:2])
	}
	if proposalID[:2] != "ap" {
		t.Errorf("expected proposalID prefix 'ap', got %q", proposalID[:2])
	}
	if decisionID[:2] != "de" {
		t.Errorf("expected decisionID prefix 'de', got %q", decisionID[:2])
	}
}
