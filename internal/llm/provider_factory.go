package llm

import (
	"fmt"
	"log"

	"baxi/internal/config"
)

// ProviderFactory creates DecisionProvider instances based on config.
type ProviderFactory struct {
	cfg      *config.Config
	registry *PromptRegistry
}

// NewProviderFactory creates a new ProviderFactory.
func NewProviderFactory(cfg *config.Config, registry *PromptRegistry) *ProviderFactory {
	return &ProviderFactory{cfg: cfg, registry: registry}
}

// CreateProvider returns the appropriate DecisionProvider based on config.
func (f *ProviderFactory) CreateProvider() (DecisionProvider, error) {
	// If LLM is disabled, always use rule-based
	if !f.cfg.LLMEnabled {
		log.Printf("LLM disabled (LLM_ENABLED=false), using rule-based provider")
		return NewRuleBasedProvider(), nil
	}

	switch f.cfg.LLMProvider {
	case "disabled", "":
		return NewRuleBasedProvider(), nil

	case "rule_based":
		return NewRuleBasedProvider(), nil

	case "openai", "openai_compatible":
		if f.registry == nil {
			log.Printf("WARNING: LLM enabled but prompt registry is nil, falling back to rule-based provider")
			return NewRuleBasedProvider(), nil
		}
		log.Printf("LLM enabled: provider=openai model=%s", f.cfg.LLMModel)
		return NewOpenAIProvider(f.cfg, f.registry)

	default:
		return nil, fmt.Errorf("unknown LLM provider: %s", f.cfg.LLMProvider)
	}
}
