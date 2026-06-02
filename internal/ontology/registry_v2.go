package ontology

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ──── V2 YAML loading ─────────────────────────────────────────────────────────

// LoadObjectSchemaV2 loads v2 object schema YAML from the given file path.
// Returns a map of object type name → ObjectTypeV2.
func LoadObjectSchemaV2(yamlPath string) (map[string]*ObjectTypeV2, error) {
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("read v2 schema YAML: %w", err)
	}
	return ParseObjectSchemaV2(data)
}

// ParseObjectSchemaV2 parses v2 object schema YAML bytes into typed structs.
func ParseObjectSchemaV2(data []byte) (map[string]*ObjectTypeV2, error) {
	var cfg objectSchemaConfigV2
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse v2 schema YAML: %w", err)
	}

	objects := make(map[string]*ObjectTypeV2, len(cfg.Objects))
	for name, raw := range cfg.Objects {
		ot, err := convertRawV2(name, raw)
		if err != nil {
			return nil, fmt.Errorf("convert v2 object %q: %w", name, err)
		}
		objects[name] = ot
	}
	return objects, nil
}

func convertRawV2(name string, raw *rawObjectV2) (*ObjectTypeV2, error) {
	// Basic validation
	if raw.Grain == "" {
		return nil, fmt.Errorf("object %q: grain is required", name)
	}
	if raw.Source.Schema == "" || raw.Source.Table == "" {
		return nil, fmt.Errorf("object %q: source.schema and source.table are required", name)
	}

	// Convert properties
	props := make(map[string]ObjectPropertyV2, len(raw.Properties))
	for pname, rp := range raw.Properties {
		isPK := false
		if rp.IsPK != nil && *rp.IsPK {
			isPK = true
		}

		sensitivity := rp.Sensitivity
		if sensitivity == "" {
			if isPK {
				sensitivity = "L2"
			} else {
				sensitivity = "L0"
			}
		}

		llmReadable := true
		if rp.LLMReadable != nil {
			llmReadable = *rp.LLMReadable
		}

		prop := ObjectPropertyV2{
			Name:        pname,
			Type:        rp.Type,
			SourceField: rp.Source,
			Expression:  rp.Expression,
			MetricRef:   rp.MetricRef,
			Sensitivity: sensitivity,
			Aggregation: rp.Agg,
			LLMReadable: llmReadable,
			Searchable:  rp.Searchable != nil && *rp.Searchable,
			Filterable:  rp.Filterable != nil && *rp.Filterable,
			IsPK:        isPK,
		}
		props[pname] = prop
	}

	// Convert relationships to links
	links := make([]ObjectLinkV2, 0, len(raw.Relationships))
	for lname, rl := range raw.Relationships {
		cardinality := rl.Cardinality
		if cardinality == "" {
			cardinality = "one_to_one"
		}
		links = append(links, ObjectLinkV2{
			Name:        lname,
			DisplayName: rl.DisplayName,
			TargetType:  rl.To,
			Cardinality: cardinality,
			Strategy:    rl.Strategy,
			SourceKey:   rl.SourceKey,
			Target: LinkTarget{
				Schema:        rl.Target.Schema,
				Table:         rl.Target.Table,
				Key:           rl.Target.Key,
				ObjectIDField: rl.Target.ObjectIDField,
			},
			Limit:  rl.Limit,
			Sort:   rl.Sort,
			Fields: rl.Fields,
		})
	}

	// Governance
	gov := ObjectGovernancePolicy{
		DefaultRole: "agent_readonly",
		RedactPII:   false,
	}
	if raw.Governance.DefaultRole != "" {
		gov.DefaultRole = raw.Governance.DefaultRole
	}
	if raw.Governance.RedactPII != nil {
		gov.RedactPII = *raw.Governance.RedactPII
	}

	// LLM access
	llmAccess := defaultLLMAccess()
	if name == TypeMetricAlert {
		llmAccess = readWriteLLMAccess()
	}

	return &ObjectTypeV2{
		Name:           name,
		DisplayName:    raw.DisplayName,
		Description:    raw.Description,
		Grain:          raw.Grain,
		Source:         ObjectSource{Schema: raw.Source.Schema, Table: raw.Source.Table, PrimaryKey: raw.Source.PrimaryKey},
		Properties:     props,
		Metrics:        raw.Metrics,
		Links:          links,
		AllowedActions: raw.Actions,
		LLMAccess:      llmAccess,
		AlertFields:    raw.AlertFields,
		Governance:     gov,
	}, nil
}

// LoadMetricDefinitions loads metric definitions from YAML.
func LoadMetricDefinitions(yamlPath string) (map[string]*MetricDefinition, error) {
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("read metric definitions YAML: %w", err)
	}
	return ParseMetricDefinitions(data)
}

// ParseMetricDefinitions parses metric definitions YAML bytes into typed structs.
func ParseMetricDefinitions(data []byte) (map[string]*MetricDefinition, error) {
	var cfg metricDefsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse metric definitions YAML: %w", err)
	}

	metrics := make(map[string]*MetricDefinition, len(cfg.Metrics))
	for name, raw := range cfg.Metrics {
		severity := raw.Severity
		if severity == nil {
			severity = make(map[string]string)
		}

		metrics[name] = &MetricDefinition{
			Name:           name,
			DisplayName:    raw.DisplayName,
			ObjectType:     raw.ObjectType,
			Grain:          raw.Grain,
			Source:         MetricSource{Schema: raw.Source.Schema, Table: raw.Source.Table},
			Filters:        raw.Filters,
			ValueColumn:    raw.ValueColumn,
			BaselineColumn: raw.BaselineColumn,
			Severity:       severity,
			LLMExplanation: raw.LLMExplanation,
		}
	}
	return metrics, nil
}

// LoadContextRecipes loads context recipes from YAML.
func LoadContextRecipes(yamlPath string) (map[string]*ContextRecipe, error) {
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("read context recipes YAML: %w", err)
	}
	return ParseContextRecipes(data)
}

// ParseContextRecipes parses context recipes YAML bytes into typed structs.
func ParseContextRecipes(data []byte) (map[string]*ContextRecipe, error) {
	var cfg contextRecipesConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse context recipes YAML: %w", err)
	}

	recipes := make(map[string]*ContextRecipe, len(cfg.Recipes))
	for name, raw := range cfg.Recipes {
		links := make(map[string]RecipeLinkInclude, len(raw.Include.Links))
		for lname, rl := range raw.Include.Links {
			links[lname] = RecipeLinkInclude{
				Limit:  rl.Limit,
				Fields: rl.Fields,
			}
		}

		budget := RecipeBudget{
			MaxLinkDepth:  2,
			MaxObjects:    30,
			MaxTokensHint: 4000,
		}
		if raw.Budget.MaxLinkDepth > 0 {
			budget.MaxLinkDepth = raw.Budget.MaxLinkDepth
		}
		if raw.Budget.MaxObjects > 0 {
			budget.MaxObjects = raw.Budget.MaxObjects
		}
		if raw.Budget.MaxTokensHint > 0 {
			budget.MaxTokensHint = raw.Budget.MaxTokensHint
		}

		gov := RecipeGovernance{
			Role:      "agent_readonly",
			RedactPII: true,
		}
		if raw.Governance.Role != "" {
			gov.Role = raw.Governance.Role
		}
		if raw.Governance.RedactPII != nil {
			gov.RedactPII = *raw.Governance.RedactPII
		}

		// Convert evidence rules
		evidenceRules := make([]EvidenceRule, len(raw.EvidenceRules))
		for i, er := range raw.EvidenceRules {
			evidenceRules[i] = EvidenceRule{
				Source:         er.Source,
				Interpretation: er.Interpretation,
			}
		}

		// Convert decision guidance
		var decisionGuidance DecisionGuidance
		if raw.DecisionGuidance != nil {
			levels := make([]GuidanceLevel, len(raw.DecisionGuidance.Levels))
			for j, l := range raw.DecisionGuidance.Levels {
				levels[j] = GuidanceLevel{
					Severity:       l.Severity,
					Recommendation: l.Recommendation,
					Actions:        l.Actions,
				}
			}
			decisionGuidance = DecisionGuidance{Levels: levels}
		}

		recipes[name] = &ContextRecipe{
			Name:        name,
			Description: raw.Description,
			Trigger: RecipeTrigger{
				ObjectType: raw.Trigger.ObjectType,
				RuleID:     raw.Trigger.RuleID,
			},
			RootObject: RecipeRootObject{
				TypeFrom: raw.RootObject.TypeFrom,
				IDFrom:   raw.RootObject.IDFrom,
			},
			Include: RecipeInclude{
				RootProperties: raw.Include.RootProperties,
				Metrics:        raw.Include.Metrics,
				Links:          links,
				Actions:        raw.Include.Actions,
			},
			Budget:           budget,
			Governance:       gov,
			EvidenceRules:    evidenceRules,
			DecisionGuidance: decisionGuidance,
		}
	}
	return recipes, nil
}
