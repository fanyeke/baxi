package llm

import (
	"context"
	"fmt"
)

// RuleBasedProvider implements DecisionProvider using severity-to-action rules.
type RuleBasedProvider struct {
	providerType string
}

// NewRuleBasedProvider creates a new RuleBasedProvider.
func NewRuleBasedProvider() *RuleBasedProvider {
	return &RuleBasedProvider{providerType: "rule_based"}
}

// GenerateDecision generates a decision based on severity rules.
func (p *RuleBasedProvider) GenerateDecision(ctx context.Context, input LLMSafeContext) (*DecisionOutput, error) {
	return p.decide(input.Trigger), nil
}

func (p *RuleBasedProvider) decide(trigger TriggerInfo) *DecisionOutput {
	out := &DecisionOutput{
		Severity:            trigger.Severity,
		RequiresHumanReview: true,
	}

	switch trigger.Severity {
	case SeverityCritical:
		out.DecisionType = ActionTypeEscalateToHuman
		out.Confidence = 0.95
		out.Summary = fmt.Sprintf(
			"Alert for %s triggered with %s severity. Current value %v vs baseline %v (delta %v%%).",
			trigger.MetricName, trigger.Severity, trigger.CurrentValue, trigger.BaselineValue, trigger.DeltaPct,
		)
		out.Rationale = []string{
			"The alert severity is critical, requiring immediate attention.",
			fmt.Sprintf("Current metric value (%v) is significantly below baseline (%v).", trigger.CurrentValue, trigger.BaselineValue),
			"Action execution is not allowed in this phase.",
		}
		out.RecommendedActions = []RecommendedAction{
			{ActionType: ActionTypeEscalateToHuman, Priority: SeverityHigh, OwnerRole: "ops"},
			{ActionType: ActionTypeNotifyOwner, Priority: SeverityHigh, OwnerRole: "ops"},
		}

	case SeverityHigh:
		out.DecisionType = ActionTypeEscalateToHuman
		out.Confidence = 0.85
		out.Summary = fmt.Sprintf(
			"Alert for %s triggered with %s severity. Current value %v vs baseline %v (delta %v%%).",
			trigger.MetricName, trigger.Severity, trigger.CurrentValue, trigger.BaselineValue, trigger.DeltaPct,
		)
		out.Rationale = []string{
			"The alert severity is high, requiring immediate attention.",
			fmt.Sprintf("Current metric value (%v) is significantly below baseline (%v).", trigger.CurrentValue, trigger.BaselineValue),
			"Action execution is not allowed in this phase.",
		}
		out.RecommendedActions = []RecommendedAction{
			{ActionType: ActionTypeEscalateToHuman, Priority: SeverityHigh, OwnerRole: "ops"},
			{ActionType: ActionTypeNotifyOwner, Priority: SeverityHigh, OwnerRole: "ops"},
		}

	case SeverityMedium:
		out.DecisionType = DecisionTypeInvestigate
		out.Confidence = 0.72
		out.Summary = fmt.Sprintf("Moderate deviation detected for %s.", trigger.MetricName)
		out.Rationale = []string{
			"Medium severity anomaly identified.",
			"Further investigation is recommended to determine root cause.",
			"Owner notification will ensure timely response.",
		}
		out.RecommendedActions = []RecommendedAction{
			{ActionType: ActionTypeNotifyOwner, Priority: SeverityMedium, OwnerRole: "analyst"},
			{ActionType: ActionTypeCreateFollowupTask, Priority: SeverityMedium, OwnerRole: "analyst"},
		}

	case SeverityLow:
		out.DecisionType = DecisionTypeMonitor
		out.Confidence = 0.60
		out.Summary = fmt.Sprintf("Slight deviation detected for %s. No action required.", trigger.MetricName)
		out.Rationale = []string{
			"Minor anomaly detected within acceptable thresholds.",
			"Monitoring recommended; no immediate intervention needed.",
		}
		out.RecommendedActions = []RecommendedAction{
			{ActionType: ActionTypeCreateFollowupTask, Priority: SeverityLow, OwnerRole: "analyst"},
		}

	default:
		out.DecisionType = DecisionTypeInvestigate
		out.Confidence = 0.50
		out.Summary = fmt.Sprintf("Anomaly detected for %s with unknown severity.", trigger.MetricName)
		out.Rationale = []string{
			"Alert triggered but severity could not be classified.",
			"Investigation recommended to assess impact.",
		}
		out.RecommendedActions = []RecommendedAction{
			{ActionType: ActionTypeNotifyOwner, Priority: SeverityMedium, OwnerRole: "analyst"},
		}
	}

	return out
}
