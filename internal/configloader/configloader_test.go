package configloader

import (
	"context"
	"path/filepath"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──── detectConfigType ──────────────────────────────────────────────────────

func TestDetectConfigType_KnownKeys(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"aip_object_schema", "object_schema"},
		{"data_classification", "data_classification"},
		{"access_policy", "access_policy"},
		{"data_lineage", "data_lineage"},
		{"data_markings", "data_markings"},
		{"health_checks", "health_checks"},
		{"checkpoint_rules", "checkpoint_rules"},
		{"alert_rules", "alert_rules"},
		{"metrics", "metrics"},
	}
	for _, tc := range tests {
		t.Run(tc.key, func(t *testing.T) {
			assert.Equal(t, tc.expected, detectConfigType(tc.key))
		})
	}
}

func TestDetectConfigType_UnknownKey(t *testing.T) {
	assert.Equal(t, "custom_type", detectConfigType("custom_type"))
	assert.Equal(t, "some_other_config", detectConfigType("some_other_config"))
}

// ──── computeHash ──────────────────────────────────────────────────────────

func TestComputeHash_EmptyContent(t *testing.T) {
	h := computeHash([]byte{})
	assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", h)
}

func TestComputeHash_KnownContent(t *testing.T) {
	h := computeHash([]byte("hello"))
	assert.Equal(t, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824", h)
}

func TestComputeHash_DifferentInputsHaveDifferentHashes(t *testing.T) {
	h1 := computeHash([]byte("config_a"))
	h2 := computeHash([]byte("config_b"))
	assert.NotEqual(t, h1, h2)
}

// ──── yamlToJSON ───────────────────────────────────────────────────────────

func TestYAMLToJSON_SimpleMap(t *testing.T) {
	yaml := "key: value\nnum: 42\n"
	json, err := yamlToJSON([]byte(yaml))
	require.NoError(t, err)
	assert.Contains(t, string(json), `"key":"value"`)
	assert.Contains(t, string(json), `"num":42`)
}

func TestYAMLToJSON_InvalidYAML(t *testing.T) {
	_, err := yamlToJSON([]byte(": invalid"))
	assert.Error(t, err)
}

func TestYAMLToJSON_EmptyContent(t *testing.T) {
	json, err := yamlToJSON([]byte{})
	require.NoError(t, err)
	assert.Equal(t, "null", string(json))
}

// ──── ValidateRequired ─────────────────────────────────────────────────────

func TestValidateRequired_AllPresent(t *testing.T) {
	reg := &ConfigRegistry{
		RawConfigs: map[string]RawConfig{
			"aip_object_schema":   {},
			"data_classification": {},
			"access_policy":       {},
			"data_lineage":        {},
		},
	}
	assert.NoError(t, ValidateRequired(reg))
}

func TestValidateRequired_MissingSome(t *testing.T) {
	reg := &ConfigRegistry{
		RawConfigs: map[string]RawConfig{
			"aip_object_schema": {},
		},
	}
	err := ValidateRequired(reg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "data_classification")
	assert.Contains(t, err.Error(), "access_policy")
	assert.Contains(t, err.Error(), "data_lineage")
}

func TestValidateRequired_EmptyRegistry(t *testing.T) {
	reg := &ConfigRegistry{
		RawConfigs: map[string]RawConfig{},
	}
	err := ValidateRequired(reg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "aip_object_schema")
}

// ──── sensitivityScore ─────────────────────────────────────────────────────

func TestSensitivityScore(t *testing.T) {
	tests := []struct {
		level    string
		expected float64
	}{
		{"pii", 1.0},
		{"sensitive", 0.8},
		{"derived_sensitive", 0.75},
		{"internal", 0.6},
		{"public_internal", 0.4},
		{"unknown", 0.5},
		{"", 0.5},
	}
	for _, tc := range tests {
		t.Run(tc.level, func(t *testing.T) {
			assert.Equal(t, tc.expected, sensitivityScore(tc.level))
		})
	}
}

// ──── lineageConfidence ────────────────────────────────────────────────────

func TestLineageConfidence(t *testing.T) {
	tests := []struct {
		transformType string
		expected      float64
	}{
		{"batch_load", 1.0},
		{"sql_aggregation", 0.9},
		{"heuristic_rule", 0.7},
		{"template_instantiation", 0.8},
		{"channel_routing", 0.85},
		{"api_sync", 0.95},
		{"unknown", 0.5},
		{"", 0.5},
	}
	for _, tc := range tests {
		t.Run(tc.transformType, func(t *testing.T) {
			assert.Equal(t, tc.expected, lineageConfidence(tc.transformType))
		})
	}
}

// ──── ListConfigKeys ───────────────────────────────────────────────────────

func TestListConfigKeys_Sorted(t *testing.T) {
	reg := &ConfigRegistry{
		RawConfigs: map[string]RawConfig{
			"zeta":  {},
			"alpha": {},
			"gamma": {},
		},
	}
	keys := ListConfigKeys(reg)
	assert.Equal(t, []string{"alpha", "gamma", "zeta"}, keys)
}

func TestListConfigKeys_Empty(t *testing.T) {
	reg := &ConfigRegistry{RawConfigs: map[string]RawConfig{}}
	assert.Empty(t, ListConfigKeys(reg))
}

func TestListConfigKeys_NilMap(t *testing.T) {
	reg := &ConfigRegistry{}
	assert.Empty(t, ListConfigKeys(reg))
}

// ──── parseObjectSchema ────────────────────────────────────────────────────

func TestParseObjectSchema_Valid(t *testing.T) {
	yaml := `objects:
  - object_type_id: customer
    display_name: Customer
    grain: customer_id
    source_tables:
      - raw.customers
    properties:
      id:
        type: string
        is_pk: true
      name:
        type: string
`
	cfg, err := parseObjectSchema([]byte(yaml))
	require.NoError(t, err)
	require.Len(t, cfg.Objects, 1)
	assert.Equal(t, "customer", cfg.Objects[0].ObjectTypeID)
	assert.Equal(t, "customer_id", cfg.Objects[0].Grain)
	assert.True(t, cfg.Objects[0].Properties["id"].IsPK)
}

func TestParseObjectSchema_InvalidYAML(t *testing.T) {
	_, err := parseObjectSchema([]byte(": invalid"))
	assert.Error(t, err)
}

// ──── parseDataClassification ──────────────────────────────────────────────

func TestParseDataClassification_Valid(t *testing.T) {
	yaml := `classifications:
  - asset_ref: "customer.email"
    level: pii
    rationale: "Email is PII"
`
	cfg, err := parseDataClassification([]byte(yaml))
	require.NoError(t, err)
	require.Len(t, cfg.Classifications, 1)
	assert.Equal(t, "customer.email", cfg.Classifications[0].AssetRef)
	assert.Equal(t, "pii", cfg.Classifications[0].Level)
}

// ──── parseAccessPolicy ────────────────────────────────────────────────────

func TestParseAccessPolicy_Valid(t *testing.T) {
	yaml := `access_policy:
  roles:
    - role: admin
      allowed_actions:
        - read
        - write
      data_access:
        - all
  default_policy: deny
`
	cfg, err := parseAccessPolicy([]byte(yaml))
	require.NoError(t, err)
	require.Len(t, cfg.AccessPolicy.Roles, 1)
	assert.Equal(t, "admin", cfg.AccessPolicy.Roles[0].Role)
	assert.Contains(t, cfg.AccessPolicy.Roles[0].AllowedActions, "read")
}

// ──── parseDataLineage ─────────────────────────────────────────────────────

func TestParseDataLineage_Valid(t *testing.T) {
	yaml := `nodes:
  - id: raw.orders
    type: table
    label: Orders
edges:
  - from: raw.orders
    to: dwd.order_level
    transform: sql_aggregate
    transform_type: sql_aggregation
`
	cfg, err := parseDataLineage([]byte(yaml))
	require.NoError(t, err)
	require.Len(t, cfg.Nodes, 1)
	require.Len(t, cfg.Edges, 1)
	assert.Equal(t, "raw.orders", cfg.Edges[0].From)
	assert.Equal(t, "sql_aggregation", cfg.Edges[0].TransformType)
}

// ──── parseDataMarkings ────────────────────────────────────────────────────

func TestParseDataMarkings_Valid(t *testing.T) {
	yaml := `markings:
  PII:
    mandatory_control: true
    access_type: read_only
    policy: restrict
pipeline_stage_markings:
  - stage: ingest
    marking: PII
`
	cfg, err := parseDataMarkings([]byte(yaml))
	require.NoError(t, err)
	assert.Contains(t, cfg.Markings, "PII")
	require.Len(t, cfg.PipelineStageMarkings, 1)
	assert.Equal(t, "ingest", cfg.PipelineStageMarkings[0].Stage)
}

// ──── parseHealthChecks ────────────────────────────────────────────────────

func TestParseHealthChecks_Valid(t *testing.T) {
	yaml := `monitoring_views:
  - id: daily_sales_view
    scope: mart.metric_daily
    check_type: completeness
    rule: count(*) > 0
    severity: high
    alert_channels:
      - feishu
health_checks:
  - id: check_order_data
    resource: dwd.order_level
    description: Orders must have data
    check_type: freshness
    validation: max(created_at) > NOW() - INTERVAL '1 day'
    severity: high
`
	cfg, err := parseHealthChecks([]byte(yaml))
	require.NoError(t, err)
	require.Len(t, cfg.MonitoringViews, 1)
	require.Len(t, cfg.HealthChecks, 1)
	assert.Equal(t, "daily_sales_view", cfg.MonitoringViews[0].ID)
	assert.Equal(t, "check_order_data", cfg.HealthChecks[0].ID)
}

// ──── parseCheckpointRules ─────────────────────────────────────────────────

func TestParseCheckpointRules_Valid(t *testing.T) {
	yaml := `checkpoints:
  execute_dispatch:
    scope: global
    requires_justification: true
checkpoint_audit:
  file: audit.csv
  format: csv
  columns:
    - timestamp
    - action
frequency:
  default: always
  cache_same_rule: true
`
	cfg, err := parseCheckpointRules([]byte(yaml))
	require.NoError(t, err)
	assert.Contains(t, cfg.Checkpoints, "execute_dispatch")
	assert.True(t, cfg.Checkpoints["execute_dispatch"].RequiresJustification)
	assert.Equal(t, "always", cfg.Frequency.Default)
}

// ──── parseAlertRules ──────────────────────────────────────────────────────

func TestParseAlertRules_Valid(t *testing.T) {
	yaml := `rules:
  - rule_id: gmv_drop
    metric: gmv
    condition: "day_over_day < -0.2"
    severity: high
    owner_role: admin
    dimension: seller_id
    min_sample_size: 100
`
	cfg, err := parseAlertRules([]byte(yaml))
	require.NoError(t, err)
	require.Len(t, cfg.Rules, 1)
	assert.Equal(t, "gmv_drop", cfg.Rules[0].RuleID)
	assert.Equal(t, "gmv", cfg.Rules[0].Metric)
}

// ──── parseMetrics ─────────────────────────────────────────────────────────

func TestParseMetrics_Valid(t *testing.T) {
	yaml := `metrics:
  gmv:
    business_definition: "Gross Merchandise Value"
    source_expression: "SUM(amount)"
    grain: daily
    dimensions:
      - seller_id
    owner_role: admin
    window:
      - daily
    unit: USD
`
	cfg, err := parseMetrics([]byte(yaml))
	require.NoError(t, err)
	assert.Contains(t, cfg.Metrics, "gmv")
	assert.Equal(t, "USD", cfg.Metrics["gmv"].Unit)
}

// ──── parseConfigByType ───────────────────────────────────────────────────

func TestParseConfigByType_ObjectSchema(t *testing.T) {
	yaml := `objects:
  - object_type_id: test
    display_name: Test
`
	result, err := parseConfigByType("object_schema", []byte(yaml))
	require.NoError(t, err)
	_, ok := result.(*ObjectSchemaConfig)
	assert.True(t, ok)
}

func TestParseConfigByType_UnknownType(t *testing.T) {
	yaml := "key: value\n"
	result, err := parseConfigByType("unknown_type", []byte(yaml))
	require.NoError(t, err)
	m, ok := result.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "value", m["key"])
}

func TestParseConfigByType_InvalidYAML(t *testing.T) {
	_, err := parseConfigByType("object_schema", []byte(": invalid"))
	assert.Error(t, err)
}

// ──── parseAny ─────────────────────────────────────────────────────────────

func TestParseAny_Valid(t *testing.T) {
	yaml := "key: value\nnested:\n  inner: 42\n"
	result, err := parseAny([]byte(yaml))
	require.NoError(t, err)
	assert.Equal(t, "value", result["key"])
	nested, ok := result["nested"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, 42, nested["inner"])
}

func TestParseAny_InvalidYAML(t *testing.T) {
	_, err := parseAny([]byte(": invalid"))
	assert.Error(t, err)
}

// ──── LogOptionalWarnings ─────────────────────────────────────────────────

func TestLogOptionalWarnings_AllPresent(t *testing.T) {
	reg := &ConfigRegistry{
		RawConfigs: map[string]RawConfig{
			"data_markings":    {},
			"health_checks":    {},
			"checkpoint_rules": {},
			"alert_rules":      {},
			"metrics":          {},
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	// Should not panic or crash; all optional configs present
	LogOptionalWarnings(logger, reg)
}

func TestLogOptionalWarnings_MissingOne(t *testing.T) {
	reg := &ConfigRegistry{
		RawConfigs: map[string]RawConfig{
			"data_markings":    {},
			"health_checks":    {},
			"checkpoint_rules": {},
			"alert_rules":      {},
			// metrics missing
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	LogOptionalWarnings(logger, reg)
}

func TestNewConfigLoader(t *testing.T) {
	cl := NewConfigLoader(nil)
	if cl == nil {
		t.Fatal("expected non-nil ConfigLoader")
	}
}

func TestLoadAll_NonExistentDir(t *testing.T) {
	cl := NewConfigLoader(nil)
	_, err := cl.LoadAll(context.Background(), "/nonexistent/dir")
	if err == nil {
		t.Fatal("expected error for nonexistent dir")
	}
}

func TestLoadAll_EmptyDir(t *testing.T) {
	cl := NewConfigLoader(nil)
	dir := t.TempDir()
	reg, err := cl.LoadAll(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reg == nil {
		t.Fatal("expected non-nil registry")
	}
	if len(reg.RawConfigs) != 0 {
		t.Errorf("expected 0 configs, got %d", len(reg.RawConfigs))
	}
}

func TestLoadAll_LoadsYamlFiles(t *testing.T) {
	cl := NewConfigLoader(nil)
	dir := t.TempDir()

	writeFile(t, dir, "object_schema.yml", `
objects:
  - object_type_id: order
    display_name: "Order"
    source_tables:
      - orders
    grain: order_id
    allow_sync_to_feishu: false
    properties:
      order_id:
        type: string
    alert_fields: []
`)
	writeFile(t, dir, "access_policy.yml", `
policies:
  - name: admin_access
    effect: allow
`)
	writeFile(t, dir, "not_yaml.txt", "this should be ignored")

	reg, err := cl.LoadAll(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reg == nil {
		t.Fatal("expected non-nil registry")
	}
	if len(reg.RawConfigs) != 2 {
		t.Errorf("expected 2 configs (ignoring .txt), got %d", len(reg.RawConfigs))
	}
}

func TestLoadAll_ParsesTypedConfigs(t *testing.T) {
	cl := NewConfigLoader(nil)
	dir := t.TempDir()

	writeFile(t, dir, "object_schema.yml", `
objects:
  - object_type_id: order
    display_name: "Order"
    source_tables:
      - orders
    grain: order_id
    allow_sync_to_feishu: false
    properties:
      order_id:
        type: string
    alert_fields: []
`)
	writeFile(t, dir, "access_policy.yml", `
policies:
  - name: admin_access
    resource_type: "case"
    action: "approve"
    principal_type: "role"
    principal_pattern: "admin"
    effect: "allow"
`)

	reg, err := cl.LoadAll(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reg.ObjectSchema == nil {
		t.Error("expected ObjectSchema to be parsed")
	}
	if reg.AccessPolicy == nil {
		t.Error("expected AccessPolicy to be parsed")
	}
	if reg.ObjectSchema.Objects == nil || len(reg.ObjectSchema.Objects) != 1 {
		t.Errorf("expected 1 object type, got %v", reg.ObjectSchema)
	}
}

func TestLoadAll_SkipsExampleFiles(t *testing.T) {
	cl := NewConfigLoader(nil)
	dir := t.TempDir()

	writeFile(t, dir, "object_schema.yml.example", `
objects:
  - object_type_id: order
`)
	writeFile(t, dir, "real_config.yml", `
objects:
  - object_type_id: seller
`)

	reg, err := cl.LoadAll(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reg.RawConfigs) != 1 {
		t.Errorf("expected 1 config (ignoring .example), got %d", len(reg.RawConfigs))
	}
	if _, ok := reg.RawConfigs["real_config"]; !ok {
		t.Error("expected real_config to be loaded")
	}
}

func TestLoadAll_InvalidYamlDoesNotCrash(t *testing.T) {
	cl := NewConfigLoader(nil)
	dir := t.TempDir()

	writeFile(t, dir, "metrics.yml", `invalid: yaml: [`)
	writeFile(t, dir, "object_schema.yml", `
objects:
  - object_type_id: order
`)

	reg, err := cl.LoadAll(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Invalid YAML should be in RawConfigs but typed parsing should fail silently
	if reg.RawConfigs == nil || len(reg.RawConfigs) != 2 {
		t.Errorf("expected 2 raw configs, got %d", len(reg.RawConfigs))
	}
	// Valid config should still be parsed
	if reg.ObjectSchema == nil {
		t.Error("expected ObjectSchema to be parsed despite invalid other file")
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
