package configloader

import (
	"fmt"
	"log/slog"
)

// requiredConfigs lists config keys that must be present.
var requiredConfigs = []string{
	"aip_object_schema",
	"data_classification",
	"access_policy",
	"data_lineage",
}

// ValidateRequired checks that all required configs exist in the registry.
// Returns an error listing all missing required configs.
// Warnings are logged via slog for optional missing configs (not implemented here;
// callers should check individually for optional configs).
func ValidateRequired(registry *ConfigRegistry) error {
	var missing []string
	for _, key := range requiredConfigs {
		if _, ok := registry.RawConfigs[key]; !ok {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required config(s): %v", missing)
	}
	return nil
}

// LogOptionalWarnings logs warnings for known optional configs that are absent.
// This helps with debugging missing non-critical config files.
func LogOptionalWarnings(logger *slog.Logger, registry *ConfigRegistry) {
	optionalConfigs := []string{
		"data_markings",
		"health_checks",
		"checkpoint_rules",
		"alert_rules",
		"metrics",
	}
	for _, key := range optionalConfigs {
		if _, ok := registry.RawConfigs[key]; !ok {
			logger.Warn("optional config not found", "config_key", key)
		}
	}
}
