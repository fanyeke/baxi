package action

// ActionDefinition describes the structured schema for a single action type.
type ActionDefinition struct {
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	RiskLevel     string                 `json:"risk_level"`
	PayloadSchema map[string]interface{} `json:"payload_schema"`
	AllowedBy     []string               `json:"allowed_by"`
	Adapter       string                 `json:"adapter"`
}

// ActionSchemaCatalog wraps ActionRegistry and exposes list/get methods
// for ActionDefinition, providing a structured schema catalog for each action type.
type ActionSchemaCatalog struct {
	registry *ActionRegistry
}

// NewActionSchemaCatalog creates a new ActionSchemaCatalog wrapping the given registry.
func NewActionSchemaCatalog(registry *ActionRegistry) *ActionSchemaCatalog {
	return &ActionSchemaCatalog{registry: registry}
}

// ListActionSchemas returns all whitelisted action types as ActionDefinitions.
func (c *ActionSchemaCatalog) ListActionSchemas() ([]ActionDefinition, error) {
	types := c.registry.AllowedActions()
	defs := make([]ActionDefinition, 0, len(types))
	for _, t := range types {
		cfg, ok := c.registry.GetActionConfig(t)
		if !ok {
			continue
		}
		def := actionConfigToDefinition(t, cfg)
		defs = append(defs, def)
	}
	return defs, nil
}

// GetActionSchema returns the schema for a single action type.
func (c *ActionSchemaCatalog) GetActionSchema(actionType string) (*ActionDefinition, error) {
	cfg, ok := c.registry.GetActionConfig(actionType)
	if !ok {
		return nil, nil
	}
	def := actionConfigToDefinition(actionType, cfg)
	return &def, nil
}

func actionConfigToDefinition(name string, cfg ActionConfig) ActionDefinition {
	schema := cfg.PayloadSchemaRaw
	if schema == nil {
		schema = make(map[string]interface{})
	}
	return ActionDefinition{
		Name:          name,
		Description:   cfg.Description,
		RiskLevel:     cfg.RiskLevel,
		PayloadSchema: schema,
		AllowedBy:     cfg.AllowedBy,
		Adapter:       cfg.Adapter,
	}
}
