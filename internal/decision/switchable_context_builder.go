package decision

import (
	"context"

	"baxi/internal/feature"
)

// SwitchableContextBuilder delegates to ContextBuilder or ContextBuilderV2
// based on the USE_NEW_CONTEXT_BUILDER feature flag (default: old builder).
type SwitchableContextBuilder struct {
	oldBuilder *ContextBuilder
	newBuilder *ContextBuilderV2
	flags      *feature.FeatureFlags
}

// NewSwitchableContextBuilder creates a feature-flag-aware context builder.
func NewSwitchableContextBuilder(
	oldBuilder *ContextBuilder,
	newBuilder *ContextBuilderV2,
	flags *feature.FeatureFlags,
) *SwitchableContextBuilder {
	return &SwitchableContextBuilder{
		oldBuilder: oldBuilder,
		newBuilder: newBuilder,
		flags:      flags,
	}
}

func (s *SwitchableContextBuilder) BuildDecisionContext(ctx context.Context, caseID string) (*DecisionContext, error) {
	if s.flags != nil && s.flags.IsEnabled(feature.FlagNewContextBuilder) && s.newBuilder != nil {
		return s.newBuilder.BuildDecisionContext(ctx, caseID)
	}
	return s.oldBuilder.BuildDecisionContext(ctx, caseID)
}
