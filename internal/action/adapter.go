package action

import "baxi/internal/decision"

// ActionTypeProviderAdapter adapts *ActionRegistry to the decision.ActionTypeProvider
// interface. It lives in the action package to avoid an import cycle (action imports
// decision via proposal_service.go).
type ActionTypeProviderAdapter struct {
	reg *ActionRegistry
}

// NewActionTypeProviderAdapter creates a new adapter wrapping the given registry.
func NewActionTypeProviderAdapter(reg *ActionRegistry) *ActionTypeProviderAdapter {
	return &ActionTypeProviderAdapter{reg: reg}
}

// ListActionTypes delegates to ActionRegistry.ListActionTypes.
func (a *ActionTypeProviderAdapter) ListActionTypes() []string {
	return a.reg.ListActionTypes()
}

// IsActionAllowed delegates to ActionRegistry.IsAllowed.
func (a *ActionTypeProviderAdapter) IsActionAllowed(actionType string) bool {
	return a.reg.IsAllowed(actionType)
}

// GetActionPolicy maps an ActionRegistry config to a decision.ActionPolicy.
func (a *ActionTypeProviderAdapter) GetActionPolicy(actionType string) (decision.ActionPolicy, bool) {
	cfg, ok := a.reg.GetActionConfig(actionType)
	if !ok {
		return decision.ActionPolicy{}, false
	}
	return decision.ActionPolicy{
		RiskLevel:        cfg.RiskLevel,
		RequiresApproval: cfg.RequiresApproval,
		AllowedBy:        cfg.AllowedBy,
	}, true
}
