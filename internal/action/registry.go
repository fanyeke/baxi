package action

import (
	"fmt"
	"os"
	"sort"
	"sync"

	"gopkg.in/yaml.v3"
)

// CanonicalActions is the hard-coded whitelist of allowed action types.
// Only these 4 action types are permitted by the system.
var CanonicalActions = []string{
	"create_followup_task",
	"notify_owner",
	"export_report",
	"create_outbox_message",
}

// ActionConfig holds the configuration for a single action type.
type ActionConfig struct {
	Enabled          *bool                  `yaml:"enabled"` // nil = default true
	Version          string                 `yaml:"version"`
	Description      string                 `yaml:"description"`
	RiskLevel        string                 `yaml:"risk_level"`
	RequiresApproval bool                   `yaml:"requires_approval"`
	DryRunDefault    bool                   `yaml:"dry_run_default"`
	Adapter          string                 `yaml:"adapter"`
	AllowedBy        []string               `yaml:"allowed_by"`
	LLMVisible       bool                   `yaml:"llm_visible"`
	LLMDescription   string                 `yaml:"llm_description"`
	PayloadSchemaRaw map[string]interface{} `yaml:"payload_schema"`
}

// ActionRegistryConfig is the top-level YAML structure.
type ActionRegistryConfig struct {
	Actions map[string]ActionConfig `yaml:"actions"`
}

// ActionRegistry provides whitelist-enforced access to action configurations.
// It parses config/action_registry.yml at startup and caches the result.
// Only the 4 canonical action types are allowed:
//   - create_followup_task
//   - notify_owner
//   - export_report
//   - create_outbox_message
type ActionRegistry struct {
	mu           sync.RWMutex
	config       *ActionRegistryConfig
	whitelist    map[string]bool
	path         string
	configLoaded bool // whether YAML file was successfully loaded
}

// NewActionRegistry creates a new registry by parsing the YAML file at the given path.
// If path is empty, it defaults to "config/action_registry.yml".
// If the file does not exist, all canonical actions are allowed with defaults
// (backward-compatible behavior). If the file exists with an explicit empty
// "actions:" block, no actions are allowed.
func NewActionRegistry(path string) (*ActionRegistry, error) {
	if path == "" {
		path = "config/action_registry.yml"
	}

	r := &ActionRegistry{
		path: path,
	}

	if err := r.loadFile(path); err != nil {
		if os.IsNotExist(err) {
			// State 1: No config file → enable all canonical with defaults
			r.configLoaded = false
			r.whitelistActions()
			return r, nil
		}
		return nil, fmt.Errorf("load action registry: %w", err)
	}

	return r, nil
}

func (r *ActionRegistry) loadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err // raw error so caller can os.IsNotExist check
	}

	var cfg ActionRegistryConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parse yaml: %w", err)
	}

	r.config = &cfg
	r.configLoaded = true
	r.whitelistActions()
	return nil
}

func (r *ActionRegistry) whitelistActions() {
	if !r.configLoaded {
		// State 1: No config file — enable all canonical with defaults
		r.whitelist = make(map[string]bool, len(CanonicalActions))
		r.config = &ActionRegistryConfig{
			Actions: make(map[string]ActionConfig, len(CanonicalActions)),
		}
		for _, key := range CanonicalActions {
			r.whitelist[key] = true
			r.config.Actions[key] = ActionConfig{
				Description:      "",
				RiskLevel:        "medium",
				RequiresApproval: true,
				AllowedBy:        []string{},
			}
		}
		return
	}

	// States 2 & 3: Config file was loaded
	if r.config.Actions == nil {
		r.config.Actions = make(map[string]ActionConfig)
	}

	for key := range r.config.Actions {
		if !isCanonical(key) {
			delete(r.config.Actions, key)
		}
	}

	// Rebuild whitelist: only canonical actions explicitly in config (and enabled)
	r.whitelist = make(map[string]bool, len(CanonicalActions))
	for _, key := range CanonicalActions {
		cfg, exists := r.config.Actions[key]
		if !exists {
			continue // State 2: not in config → not whitelisted
		}
		// State 3: check enabled field (nil = default true)
		if cfg.Enabled == nil || *cfg.Enabled {
			r.whitelist[key] = true
		} else {
			delete(r.config.Actions, key)
		}
	}
}

func isCanonical(action string) bool {
	for _, a := range CanonicalActions {
		if a == action {
			return true
		}
	}
	return false
}

// IsAllowed returns true if the action type is in the whitelist.
func (r *ActionRegistry) IsAllowed(actionType string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.whitelist[actionType]
}

// GetActionConfig returns the config for the given action type, or false if not allowed/not found.
func (r *ActionRegistry) GetActionConfig(actionType string) (ActionConfig, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.whitelist[actionType] {
		return ActionConfig{}, false
	}

	cfg, ok := r.config.Actions[actionType]
	if !ok {
		return ActionConfig{}, false
	}

	return cfg, true
}

// AllowedActions returns the list of all whitelisted action types.
func (r *ActionRegistry) AllowedActions() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]string, 0, len(CanonicalActions))
	for _, a := range CanonicalActions {
		if r.whitelist[a] {
			result = append(result, a)
		}
	}
	return result
}

// ListActionTypes returns the sorted list of all configured action types from the YAML config.
// Sorted alphabetically for deterministic ordering.
func (r *ActionRegistry) ListActionTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.config.Actions))
	for t := range r.config.Actions {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}

// GetLLMVisibleActions returns contracts for all actions visible to the LLM.
// These are a subset of AllowedActions filtered to only those with llm_visible=true.
func (r *ActionRegistry) GetLLMVisibleActions() []ActionContract {
	r.mu.RLock()
	defer r.mu.RUnlock()

	contracts := make([]ActionContract, 0, len(CanonicalActions))
	for _, a := range CanonicalActions {
		cfg, ok := r.config.Actions[a]
		if !ok || !r.whitelist[a] {
			continue
		}
		if !cfg.LLMVisible {
			continue
		}
		contracts = append(contracts, buildContract(a, cfg))
	}
	return contracts
}

// GetActionContract returns the LLM-visible contract for a given action type.
func (r *ActionRegistry) GetActionContract(actionType string) (*ActionContract, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.whitelist[actionType] {
		return nil, false
	}
	cfg, ok := r.config.Actions[actionType]
	if !ok {
		return nil, false
	}
	c := buildContract(actionType, cfg)
	return &c, true
}

// ValidatePayload checks that the given payload matches the action's payload_schema.
// Returns nil if no schema is defined or validation passes.
func (r *ActionRegistry) ValidatePayload(actionType string, payload map[string]interface{}) []string {
	r.mu.RLock()
	cfg, ok := r.config.Actions[actionType]
	r.mu.RUnlock()

	if !ok {
		return []string{"action type not found"}
	}

	schema := cfg.PayloadSchemaRaw
	if schema == nil || len(schema) == 0 {
		return nil // no schema to validate against
	}

	var errs []string

	// Check required fields
	if required, ok := schema["required"]; ok {
		if reqList, ok := required.([]interface{}); ok {
			for _, r := range reqList {
				if field, ok := r.(string); ok {
					if _, exists := payload[field]; !exists || payload[field] == nil {
						errs = append(errs, fmt.Sprintf("missing required field: %s", field))
					}
				}
			}
		}
	}

	// Check properties against their constraints
	if props, ok := schema["properties"].(map[string]interface{}); ok {
		for field, value := range payload {
			propDef, ok := props[field].(map[string]interface{})
			if !ok {
				continue
			}

			// Type check
			if expectedType, ok := propDef["type"].(string); ok {
				switch expectedType {
				case "string":
					if _, ok := value.(string); !ok && value != nil {
						errs = append(errs, fmt.Sprintf("field '%s' must be a string", field))
					}
				case "boolean":
					if _, ok := value.(bool); !ok && value != nil {
						errs = append(errs, fmt.Sprintf("field '%s' must be a boolean", field))
					}
				}
			}

			// Enum check
			if enumVals, ok := propDef["enum"].([]interface{}); ok {
				valid := false
				strVal := fmt.Sprintf("%v", value)
				for _, ev := range enumVals {
					if fmt.Sprintf("%v", ev) == strVal {
						valid = true
						break
					}
				}
				if !valid && value != nil {
					errs = append(errs, fmt.Sprintf("field '%s' value '%v' not in allowed values", field, value))
				}
			}

			// String maxLength check
			if maxLen, ok := propDef["maxLength"].(int); ok {
				if strVal, ok := value.(string); ok && len(strVal) > maxLen {
					errs = append(errs, fmt.Sprintf("field '%s' exceeds max length of %d", field, maxLen))
				}
			}
		}
	}

	return errs
}

func buildContract(actionType string, cfg ActionConfig) ActionContract {
	c := ActionContract{
		ActionType:     actionType,
		Description:    cfg.LLMDescription,
		RiskLevel:      cfg.RiskLevel,
		RequiresReview: cfg.RequiresApproval,
		Adapter:        cfg.Adapter,
	}
	if cfg.PayloadSchemaRaw != nil {
		if required, ok := cfg.PayloadSchemaRaw["required"]; ok {
			if reqList, ok := required.([]interface{}); ok {
				requiredFields := make([]string, len(reqList))
				for i, r := range reqList {
					requiredFields[i] = fmt.Sprintf("%v", r)
				}
				c.RequiredPayload = requiredFields
			}
		}
		// Provide a simplified schema for LLM consumption
		simplified := make(map[string]interface{})
		if props, ok := cfg.PayloadSchemaRaw["properties"]; ok {
			simplified["properties"] = props
		}
		c.PayloadSchema = simplified
	}
	return c
}

// NewEmptyRegistry returns a registry that allows no actions.
// Useful as a fallback when the config file cannot be loaded.
func NewEmptyRegistry() *ActionRegistry {
	return &ActionRegistry{
		whitelist: make(map[string]bool),
		config:    &ActionRegistryConfig{Actions: make(map[string]ActionConfig)},
		path:      "",
	}
}

// Reload re-reads the YAML file. Safe for concurrent use.
func (r *ActionRegistry) Reload() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.loadFile(r.path)
}
