package ontology

// ──── Context Recipe types ────────────────────────────────────────────────────
// Loaded from config/context_recipes.yml. A ContextRecipe defines which object
// properties, metrics, links, and actions to include when building an LLM-safe
// context for a given alert scenario.

// ContextRecipe defines the recipe for building an LLM-safe context for a
// specific alert scenario. It specifies what root object properties, metrics,
// linked objects, and actions to include, along with budget and governance.
type ContextRecipe struct {
	Name        string
	Description string
	Trigger     RecipeTrigger
	RootObject  RecipeRootObject
	Include     RecipeInclude
	Budget      RecipeBudget
	Governance  RecipeGovernance
}

// RecipeTrigger defines which alert object type and rule ID this recipe matches.
type RecipeTrigger struct {
	ObjectType string // e.g. "metric_alert"
	RuleID     string // e.g. "seller_late_delivery_spike"
}

// RecipeRootObject defines how to resolve the root object from the alert.
type RecipeRootObject struct {
	TypeFrom string // e.g. "alert.object_type"
	IDFrom   string // e.g. "alert.object_id"
}

// RecipeInclude specifies what to include in the context.
type RecipeInclude struct {
	RootProperties []string                    // property names to include
	Metrics        []string                    // metric names to include
	Links          map[string]RecipeLinkInclude // link name → config
	Actions        []string                    // action types to include
}

// RecipeLinkInclude configures how a linked object set is fetched.
type RecipeLinkInclude struct {
	Limit  int      // max results
	Fields []string // fields to include
}

// RecipeBudget caps context size.
type RecipeBudget struct {
	MaxLinkDepth  int // maximum link traversal depth
	MaxObjects    int // maximum total objects in context
	MaxTokensHint int // estimated token budget hint
}

// RecipeGovernance defines governance rules for the context.
type RecipeGovernance struct {
	Role      string
	RedactPII bool
}

// ──── YAML parsing types ─────────────────────────────────────────────────────

type contextRecipesConfig struct {
	Version string                  `yaml:"version"`
	Recipes map[string]*rawRecipeV1 `yaml:"recipes"`
}

type rawRecipeV1 struct {
	Description string             `yaml:"description"`
	Trigger     rawTriggerV1       `yaml:"trigger"`
	RootObject  rawRootObjectV1    `yaml:"root_object"`
	Include     rawIncludeV1       `yaml:"include"`
	Budget      rawBudgetV1        `yaml:"budget,omitempty"`
	Governance  rawRecipeGovV1     `yaml:"governance,omitempty"`
}

type rawTriggerV1 struct {
	ObjectType string `yaml:"object_type"`
	RuleID     string `yaml:"rule_id"`
}

type rawRootObjectV1 struct {
	TypeFrom string `yaml:"type_from"`
	IDFrom   string `yaml:"id_from"`
}

type rawIncludeV1 struct {
	RootProperties []string                   `yaml:"root_properties,omitempty"`
	Metrics        []string                   `yaml:"metrics,omitempty"`
	Links          map[string]rawLinkIncludeV1 `yaml:"links,omitempty"`
	Actions        []string                   `yaml:"actions,omitempty"`
}

type rawLinkIncludeV1 struct {
	Limit  int      `yaml:"limit"`
	Fields []string `yaml:"fields,omitempty"`
}

type rawBudgetV1 struct {
	MaxLinkDepth  int `yaml:"max_link_depth"`
	MaxObjects    int `yaml:"max_objects"`
	MaxTokensHint int `yaml:"max_tokens_hint"`
}

type rawRecipeGovV1 struct {
	Role      string `yaml:"role"`
	RedactPII *bool  `yaml:"redact_pii,omitempty"`
}
