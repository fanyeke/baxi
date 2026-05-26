package llm

import (
	"fmt"
	"strings"
)

// ValidationError represents a single validation failure.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationResult holds the result of validating a DecisionOutput.
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
}

// ValidateDecision checks a DecisionOutput against schema rules.
func ValidateDecision(output *DecisionOutput, allowedActions []string) *ValidationResult {
	result := &ValidationResult{Valid: true, Errors: []ValidationError{}}

	// Guard against nil
	if output == nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{Field: "output", Message: "output is nil"})
		return result
	}

	// 1. decision_type must be valid
	validTypes := map[string]bool{
		DecisionTypeMonitor:      true,
		DecisionTypeInvestigate:  true,
		DecisionTypeOptimize:     true,
		DecisionTypeIntervention: true,
		DecisionTypeExperiment:   true,
	}
	if !validTypes[output.DecisionType] {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "decision_type",
			Message: fmt.Sprintf("invalid decision type: %s", output.DecisionType),
		})
	}

	// 2. severity must be valid
	validSeverity := map[string]bool{
		SeverityLow:      true,
		SeverityMedium:   true,
		SeverityHigh:     true,
		SeverityCritical: true,
	}
	if !validSeverity[output.Severity] {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "severity",
			Message: fmt.Sprintf("invalid severity: %s", output.Severity),
		})
	}

	// 3. confidence must be in [0, 1]
	if output.Confidence < 0 || output.Confidence > 1 {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "confidence",
			Message: fmt.Sprintf("confidence out of range [0,1]: %f", output.Confidence),
		})
	}

	// 4. requires_human_review must be true (Phase 6 guardrail)
	if !output.RequiresHumanReview {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "requires_human_review",
			Message: "must be true in Phase 6",
		})
	}

	// 5. recommended_actions must be subset of allowed_actions
	allowedSet := make(map[string]bool)
	for _, a := range allowedActions {
		allowedSet[a] = true
	}
	for i, action := range output.RecommendedActions {
		if !allowedSet[action.ActionType] {
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("recommended_actions[%d].action_type", i),
				Message: fmt.Sprintf("'%s' is not in allowed_actions", action.ActionType),
			})
		}
		// 6. Each action has valid action_type
		validActions := map[string]bool{
			ActionTypeCreateFollowupTask: true,
			ActionTypeNotifyOwner:        true,
			ActionTypeExportReport:       true,
			ActionTypeEscalateToHuman:    true,
		}
		if !validActions[action.ActionType] {
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("recommended_actions[%d].action_type", i),
				Message: fmt.Sprintf("invalid action_type: %s", action.ActionType),
			})
		}
	}

	if len(result.Errors) > 0 {
		result.Valid = false
	}

	return result
}

// ValidateDecisionErrors returns a human-readable summary of all validation errors.
func ValidateDecisionErrors(output *DecisionOutput, allowedActions []string) string {
	result := ValidateDecision(output, allowedActions)
	if result.Valid {
		return ""
	}
	var sb strings.Builder
	for i, err := range result.Errors {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(err.Error())
	}
	return sb.String()
}
