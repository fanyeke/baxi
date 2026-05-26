package configloader

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// ObjectSchemaConfig mirrors config/aip_object_schema.yml.
type ObjectSchemaConfig struct {
	Objects []ObjectSchema `yaml:"objects"`
}

// ObjectSchema defines a single object type.
type ObjectSchema struct {
	ObjectTypeID      string                 `yaml:"object_type_id"`
	DisplayName       string                 `yaml:"display_name"`
	SourceTables      []string               `yaml:"source_tables"`
	Grain             string                 `yaml:"grain"`
	AllowSyncToFeishu bool                   `yaml:"allow_sync_to_feishu"`
	Properties        map[string]PropertyDef `yaml:"properties"`
	Relationships     map[string]RelationDef `yaml:"relationships,omitempty"`
	AlertFields       []string               `yaml:"alert_fields"`
}

// PropertyDef defines a single property on an object.
type PropertyDef struct {
	Type   string `yaml:"type"`
	IsPK   bool   `yaml:"is_pk,omitempty"`
	Source string `yaml:"source,omitempty"`
	Agg    string `yaml:"agg,omitempty"`
}

// RelationDef defines a relationship to another object.
type RelationDef struct {
	To    string `yaml:"to"`
	Grain string `yaml:"grain"`
}

// DataClassificationConfig mirrors config/data_classification.yml.
type DataClassificationConfig struct {
	Classifications []Classification `yaml:"classifications"`
}

// Classification defines a single data classification entry.
type Classification struct {
	AssetRef        string            `yaml:"asset_ref"`
	Level           string            `yaml:"level"`
	Rationale       string            `yaml:"rationale"`
	AppliesToFields map[string]string `yaml:"applies_to_fields,omitempty"`
}

// AccessPolicyConfig mirrors config/access_policy.yml.
type AccessPolicyConfig struct {
	AccessPolicy struct {
		Roles         []Role `yaml:"roles"`
		DefaultPolicy string `yaml:"default_policy"`
	} `yaml:"access_policy"`
}

// Role defines a single role with allowed actions and data access.
type Role struct {
	Role           string   `yaml:"role"`
	AllowedActions []string `yaml:"allowed_actions"`
	DataAccess     []string `yaml:"data_access"`
}

// DataLineageConfig mirrors config/data_lineage.yml.
type DataLineageConfig struct {
	Nodes []LineageNode `yaml:"nodes"`
	Edges []LineageEdge `yaml:"edges"`
}

// LineageNode defines a node in the data lineage graph.
type LineageNode struct {
	ID       string `yaml:"id"`
	Type     string `yaml:"type"`
	Label    string `yaml:"label"`
	Status   string `yaml:"status"`
	LinkedTo string `yaml:"linked_to,omitempty"`
}

// LineageEdge defines an edge connecting two lineage nodes.
type LineageEdge struct {
	From           string `yaml:"from"`
	To             string `yaml:"to"`
	Transform      string `yaml:"transform"`
	TransformType  string `yaml:"transform_type"`
}

// DataMarkingsConfig mirrors config/data_markings.yml.
type DataMarkingsConfig struct {
	Markings             map[string]Marking       `yaml:"markings"`
	PipelineStageMarkings []PipelineStageMarking  `yaml:"pipeline_stage_markings"`
}

// Marking defines a mandatory access control marking.
type Marking struct {
	MandatoryControl      bool     `yaml:"mandatory_control"`
	AccessType            string   `yaml:"access_type"`
	Conjunctive           bool     `yaml:"conjunctive"`
	Inheritance           []string `yaml:"inheritance"`
	AppliesTo             []string `yaml:"applies_to"`
	Policy                string   `yaml:"policy"`
	ExpandAccessPermission string  `yaml:"expand_access_permission"`
}

// PipelineStageMarking maps a pipeline stage to a marking.
type PipelineStageMarking struct {
	Stage   string `yaml:"stage"`
	Marking string `yaml:"marking"`
}

// HealthChecksConfig mirrors config/health_checks.yml.
type HealthChecksConfig struct {
	MonitoringViews []MonitoringView `yaml:"monitoring_views"`
	HealthChecks    []HealthCheck    `yaml:"health_checks"`
}

// MonitoringView defines a scope-based monitoring rule.
type MonitoringView struct {
	ID           string   `yaml:"id"`
	Scope        string   `yaml:"scope"`
	CheckType    string   `yaml:"check_type"`
	Rule         string   `yaml:"rule"`
	Severity     string   `yaml:"severity"`
	AlertChannels []string `yaml:"alert_channels"`
}

// HealthCheck defines a per-resource health validation.
type HealthCheck struct {
	ID           string   `yaml:"id"`
	Resource     string   `yaml:"resource"`
	Description  string   `yaml:"description"`
	CheckType    string   `yaml:"check_type"`
	Validation   string   `yaml:"validation"`
	Severity     string   `yaml:"severity"`
	AlertChannels []string `yaml:"alert_channels,omitempty"`
}

// CheckpointRulesConfig mirrors config/checkpoint_rules.yml.
type CheckpointRulesConfig struct {
	Checkpoints       map[string]Checkpoint `yaml:"checkpoints"`
	CheckpointAudit   CheckpointAuditConfig `yaml:"checkpoint_audit"`
	Frequency         FrequencyConfig       `yaml:"frequency"`
}

// Checkpoint defines a single checkpoint rule.
type Checkpoint struct {
	Scope               string   `yaml:"scope"`
	Endpoint            string   `yaml:"endpoint,omitempty"`
	RequiresJustification bool   `yaml:"requires_justification"`
	Prompt              string   `yaml:"prompt,omitempty"`
	RecordFields        []string `yaml:"record_fields,omitempty"`
	CheckpointTypes     []string `yaml:"checkpoint_types,omitempty"`
}

// CheckpointAuditConfig configures the checkpoint audit file.
type CheckpointAuditConfig struct {
	File         string   `yaml:"file"`
	Format       string   `yaml:"format"`
	Columns      []string `yaml:"columns"`
	RetentionDays int     `yaml:"retention_days"`
}

// FrequencyConfig configures checkpoint prompting frequency.
type FrequencyConfig struct {
	Default         string `yaml:"default"`
	CacheSameRule   bool   `yaml:"cache_same_rule"`
}

// AlertRulesConfig mirrors config/alert_rules.yml.
type AlertRulesConfig struct {
	Rules []AlertRule `yaml:"rules"`
}

// AlertRule defines a single alert rule.
type AlertRule struct {
	RuleID         string `yaml:"rule_id"`
	Metric         string `yaml:"metric"`
	Condition      string `yaml:"condition"`
	Severity       string `yaml:"severity"`
	OwnerRole      string `yaml:"owner_role"`
	Dimension      string `yaml:"dimension"`
	MinSampleSize  int    `yaml:"min_sample_size"`
	Description    string `yaml:"description"`
}

// MetricsConfig mirrors config/metrics.yml.
type MetricsConfig struct {
	Metrics map[string]Metric `yaml:"metrics"`
}

// Metric defines a single business metric.
type Metric struct {
	BusinessDefinition string   `yaml:"business_definition"`
	SourceExpression   string   `yaml:"source_expression"`
	Grain              string   `yaml:"grain"`
	Dimensions         []string `yaml:"dimensions"`
	OwnerRole          string   `yaml:"owner_role"`
	Window             []string `yaml:"window"`
	Unit               string   `yaml:"unit"`
}

// configTypeKey maps config key (filename without .yml) to a normalized type.
var configTypeKey = map[string]string{
	"aip_object_schema":   "object_schema",
	"data_classification":  "data_classification",
	"access_policy":        "access_policy",
	"data_lineage":         "data_lineage",
	"data_markings":        "data_markings",
	"health_checks":        "health_checks",
	"checkpoint_rules":     "checkpoint_rules",
	"alert_rules":          "alert_rules",
	"metrics":              "metrics",
}

func detectConfigType(configKey string) string {
	if t, ok := configTypeKey[configKey]; ok {
		return t
	}
	return configKey
}

func parseObjectSchema(content []byte) (*ObjectSchemaConfig, error) {
	var cfg ObjectSchemaConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("parse object schema: %w", err)
	}
	return &cfg, nil
}

func parseDataClassification(content []byte) (*DataClassificationConfig, error) {
	var cfg DataClassificationConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("parse data classification: %w", err)
	}
	return &cfg, nil
}

func parseAccessPolicy(content []byte) (*AccessPolicyConfig, error) {
	var cfg AccessPolicyConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("parse access policy: %w", err)
	}
	return &cfg, nil
}

func parseDataLineage(content []byte) (*DataLineageConfig, error) {
	var cfg DataLineageConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("parse data lineage: %w", err)
	}
	return &cfg, nil
}

func parseDataMarkings(content []byte) (*DataMarkingsConfig, error) {
	var cfg DataMarkingsConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("parse data markings: %w", err)
	}
	return &cfg, nil
}

func parseHealthChecks(content []byte) (*HealthChecksConfig, error) {
	var cfg HealthChecksConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("parse health checks: %w", err)
	}
	return &cfg, nil
}

func parseCheckpointRules(content []byte) (*CheckpointRulesConfig, error) {
	var cfg CheckpointRulesConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("parse checkpoint rules: %w", err)
	}
	return &cfg, nil
}

func parseAlertRules(content []byte) (*AlertRulesConfig, error) {
	var cfg AlertRulesConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("parse alert rules: %w", err)
	}
	return &cfg, nil
}

func parseMetrics(content []byte) (*MetricsConfig, error) {
	var cfg MetricsConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("parse metrics: %w", err)
	}
	return &cfg, nil
}

func parseAny(content []byte) (map[string]interface{}, error) {
	var data map[string]interface{}
	if err := yaml.Unmarshal(content, &data); err != nil {
		return nil, fmt.Errorf("parse generic yaml: %w", err)
	}
	return data, nil
}

func yamlToJSON(content []byte) ([]byte, error) {
	var data interface{}
	if err := yaml.Unmarshal(content, &data); err != nil {
		return nil, fmt.Errorf("yaml to json: %w", err)
	}
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal json: %w", err)
	}
	return jsonBytes, nil
}

func parseConfigByType(configType string, content []byte) (interface{}, error) {
	switch configType {
	case "object_schema":
		return parseObjectSchema(content)
	case "data_classification":
		return parseDataClassification(content)
	case "access_policy":
		return parseAccessPolicy(content)
	case "data_lineage":
		return parseDataLineage(content)
	case "data_markings":
		return parseDataMarkings(content)
	case "health_checks":
		return parseHealthChecks(content)
	case "checkpoint_rules":
		return parseCheckpointRules(content)
	case "alert_rules":
		return parseAlertRules(content)
	case "metrics":
		return parseMetrics(content)
	default:
		return parseAny(content)
	}
}
