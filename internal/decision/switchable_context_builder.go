package decision

import (
	"context"

	"baxi/internal/feature"
)

// Builder version constants for SwitchableContextBuilder.
const (
	BuilderV2 = "v2"
	BuilderV3 = "v3"
)

// SwitchableContextBuilder delegates to ContextBuilder, ContextBuilderV2,
// or ContextBuilderV3 based on the USE_NEW_CONTEXT_BUILDER feature flag
// (default: old builder) or an explicit version set via SwitchTo.
type SwitchableContextBuilder struct {
	oldBuilder *ContextBuilder
	newBuilder *ContextBuilderV2
	v3Builder  *ContextBuilderV3
	flags      *feature.FeatureFlags
	version    string // explicit version set via SwitchTo, takes priority over feature flag
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

// WithV3Builder attaches a ContextBuilderV3 to the switchable builder.
// The v3 builder wraps the v2 builder and enriches contexts with
// ontology link traversal.
func (s *SwitchableContextBuilder) WithV3Builder(v3 *ContextBuilderV3) *SwitchableContextBuilder {
	s.v3Builder = v3
	return s
}

// SwitchTo explicitly selects which builder version to use.
// Valid values: "v2", "v3". An empty string or any other value
// falls back to the feature flag behavior (v1 vs v2).
func (s *SwitchableContextBuilder) SwitchTo(version string) {
	s.version = version
}

func (s *SwitchableContextBuilder) BuildDecisionContext(ctx context.Context, caseID string) (*DecisionContext, error) {
	switch s.version {
	case BuilderV3:
		if s.v3Builder != nil {
			return s.v3Builder.BuildDecisionContext(ctx, caseID)
		}
	case BuilderV2:
		if s.newBuilder != nil {
			return s.newBuilder.BuildDecisionContext(ctx, caseID)
		}
	}
	if s.flags != nil && s.flags.IsEnabled(feature.FlagNewContextBuilder) && s.newBuilder != nil {
		return s.newBuilder.BuildDecisionContext(ctx, caseID)
	}
	return s.oldBuilder.BuildDecisionContext(ctx, caseID)
}
