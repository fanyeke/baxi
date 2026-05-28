package llm

import (
	"context"
	"errors"
)

// DisabledProvider returns an error when GenerateDecision is called.
type DisabledProvider struct{}

func NewDisabledProvider() *DisabledProvider {
	return &DisabledProvider{}
}

func (p *DisabledProvider) GenerateDecision(ctx context.Context, input LLMSafeContext) (*DecisionOutput, error) {
	return nil, errors.New("LLM is disabled: LLM_ENABLED=false")
}
