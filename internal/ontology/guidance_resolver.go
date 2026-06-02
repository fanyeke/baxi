package ontology

import (
	"fmt"
	"strings"
)

// ResolveGuidance matches severity against recipe.DecisionGuidance.Levels
// with case-insensitive comparison. Returns the matching GuidanceLevel or
// an error if no level matches or the recipe is nil.
func ResolveGuidance(recipe *ContextRecipe, severity string) (*GuidanceLevel, error) {
	if recipe == nil {
		return nil, fmt.Errorf("guidance: recipe is nil")
	}
	lower := strings.ToLower(severity)
	for i := range recipe.DecisionGuidance.Levels {
		if strings.ToLower(recipe.DecisionGuidance.Levels[i].Severity) == lower {
			return &recipe.DecisionGuidance.Levels[i], nil
		}
	}
	return nil, fmt.Errorf("guidance: severity %q not found in recipe %q", severity, recipe.Name)
}

// GuidanceToPromptFragment formats a GuidanceLevel as a concise string suitable
// for injecting into an LLM system prompt.
//
// Format: "Recommendation: {level.Recommendation}. Suggested actions: {actions}. Requires human approval: yes."
func GuidanceToPromptFragment(level *GuidanceLevel) string {
	if level == nil {
		return ""
	}
	actions := strings.Join(level.Actions, ", ")
	return fmt.Sprintf("Recommendation: %s. Suggested actions: %s. Requires human approval: yes.",
		level.Recommendation, actions)
}
