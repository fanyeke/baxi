package action

import (
	"fmt"
	"os"
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
	Description      string   `yaml:"description"`
	RiskLevel        string   `yaml:"risk_level"`
	RequiresApproval bool     `yaml:"requires_approval"`
	AllowedBy        []string `yaml:"allowed_by"`
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
	mu        sync.RWMutex
	config    *ActionRegistryConfig
	whitelist map[string]bool
	path      string
}

// NewActionRegistry creates a new registry by parsing the YAML file at the given path.
// If path is empty, it defaults to "config/action_registry.yml".
func NewActionRegistry(path string) (*ActionRegistry, error) {
	if path == "" {
		path = "config/action_registry.yml"
	}

	r := &ActionRegistry{
		whitelist: make(map[string]bool, len(CanonicalActions)),
		path:      path,
	}

	for _, a := range CanonicalActions {
		r.whitelist[a] = true
	}

	if err := r.loadFile(path); err != nil {
		return nil, fmt.Errorf("load action registry: %w", err)
	}

	return r, nil
}

func (r *ActionRegistry) loadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	var cfg ActionRegistryConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parse yaml: %w", err)
	}

	r.config = &cfg
	r.whitelistActions()
	return nil
}

func (r *ActionRegistry) whitelistActions() {
	if r.config.Actions == nil {
		r.config.Actions = make(map[string]ActionConfig)
	}

	for key := range r.config.Actions {
		if !r.whitelist[key] {
			delete(r.config.Actions, key)
		}
	}

	for _, key := range CanonicalActions {
		if _, exists := r.config.Actions[key]; !exists {
			r.config.Actions[key] = ActionConfig{
				Description:      "",
				RiskLevel:        "medium",
				RequiresApproval: true,
				AllowedBy:        []string{},
			}
		}
	}
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

	result := make([]string, len(CanonicalActions))
	copy(result, CanonicalActions)
	return result
}

// Reload re-reads the YAML file. Safe for concurrent use.
func (r *ActionRegistry) Reload() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.loadFile(r.path)
}
