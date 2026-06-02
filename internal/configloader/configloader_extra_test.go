package configloader

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──── LoadAll with all config types ─────────────────────────────────────

func TestLoadAll_AllConfigTypes_Extra(t *testing.T) {
	cl := NewConfigLoader(nil)
	dir := t.TempDir()

	writeFile(t, dir, "object_schema.yml", `
objects:
  - object_type_id: order
    display_name: "Order"
    source_tables: [orders]
    grain: order_id
    properties:
      order_id:
        type: string
        is_pk: true
    alert_fields: []
`)
	writeFile(t, dir, "data_classification.yml", `
classifications:
  - asset_ref: "customer.email"
    level: pii
    rationale: "Email is PII"
`)
	writeFile(t, dir, "access_policy.yml", `
access_policy:
  roles:
    - role: admin
      allowed_actions: [read, write]
      data_access: [all]
  default_policy: deny
`)
	writeFile(t, dir, "data_lineage.yml", `
nodes:
  - id: raw.orders
    type: table
    label: Orders
edges:
  - from: raw.orders
    to: dwd.order_level
    transform: sql_aggregate
    transform_type: sql_aggregation
`)
	writeFile(t, dir, "data_markings.yml", `
markings:
  PII:
    mandatory_control: true
    access_type: read_only
    policy: restrict
pipeline_stage_markings:
  - stage: ingest
    marking: PII
`)
	writeFile(t, dir, "health_checks.yml", `
monitoring_views:
  - id: daily_check
    scope: mart.metric_daily
    check_type: completeness
    rule: count(*) > 0
    severity: high
    alert_channels: [feishu]
health_checks:
  - id: order_check
    resource: dwd.order_level
    description: Check orders
    check_type: freshness
    validation: max(created_at) > NOW() - INTERVAL '1 day'
    severity: high
`)
	writeFile(t, dir, "checkpoint_rules.yml", `
checkpoints:
  execute_dispatch:
    scope: global
    requires_justification: true
checkpoint_audit:
  file: audit.csv
  format: csv
  columns: [timestamp, action]
  retention_days: 30
frequency:
  default: always
  cache_same_rule: true
`)
	writeFile(t, dir, "alert_rules.yml", `
rules:
  - rule_id: gmv_drop
    metric: gmv
    condition: "day_over_day < -0.2"
    severity: high
    owner_role: admin
    dimension: seller_id
    min_sample_size: 100
    description: GMV drop alert
`)
	writeFile(t, dir, "metrics.yml", `
metrics:
  gmv:
    business_definition: "Gross Merchandise Value"
    source_expression: "SUM(amount)"
    grain: daily
    dimensions: [seller_id]
    owner_role: admin
    window: [daily]
    unit: USD
`)

	reg, err := cl.LoadAll(context.Background(), dir)
	require.NoError(t, err)
	require.NotNil(t, reg)

	assert.NotNil(t, reg.ObjectSchema, "ObjectSchema should be parsed")
	assert.NotNil(t, reg.DataClassification, "DataClassification should be parsed")
	assert.NotNil(t, reg.AccessPolicy, "AccessPolicy should be parsed")
	assert.NotNil(t, reg.DataLineage, "DataLineage should be parsed")
	assert.NotNil(t, reg.DataMarkings, "DataMarkings should be parsed")
	assert.NotNil(t, reg.HealthChecks, "HealthChecks should be parsed")
	assert.NotNil(t, reg.CheckpointRules, "CheckpointRules should be parsed")
	assert.NotNil(t, reg.AlertRules, "AlertRules should be parsed")
	assert.NotNil(t, reg.Metrics, "Metrics should be parsed")

	assert.Equal(t, 9, len(reg.RawConfigs))
}

func TestLoadAll_SkipsSubdirectories_Extra(t *testing.T) {
	cl := NewConfigLoader(nil)
	dir := t.TempDir()

	writeFile(t, dir, "object_schema.yml", `objects: []`)
	err := os.MkdirAll(filepath.Join(dir, "subdir"), 0755)
	require.NoError(t, err)
	writeFile(t, dir, "subdir/sneaky.yml", `objects: []`)

	reg, err := cl.LoadAll(context.Background(), dir)
	require.NoError(t, err)
	assert.Equal(t, 1, len(reg.RawConfigs))
}

func TestLoadAll_SkipsDoubleExample_Extra(t *testing.T) {
	cl := NewConfigLoader(nil)
	dir := t.TempDir()

	writeFile(t, dir, "test.yml.example", `key: value`)
	writeFile(t, dir, "test.yml.example.bak", `key: value2`)

	reg, err := cl.LoadAll(context.Background(), dir)
	require.NoError(t, err)
	assert.Equal(t, 0, len(reg.RawConfigs))
}

// ──── LogOptionalWarnings edge cases ──────────────────────────────────

func TestLogOptionalWarnings_AllMissing_Extra(t *testing.T) {
	reg := &ConfigRegistry{
		RawConfigs: map[string]RawConfig{},
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	LogOptionalWarnings(logger, reg)
}

func TestLogOptionalWarnings_NilRegistry_Extra(t *testing.T) {
	reg := &ConfigRegistry{
		RawConfigs: nil,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	LogOptionalWarnings(logger, reg)
}

// ──── ValidateRequired edge cases ──────────────────────────────────────

func TestValidateRequired_AllFourPresent_Extra(t *testing.T) {
	reg := &ConfigRegistry{
		RawConfigs: map[string]RawConfig{
			"aip_object_schema":   {},
			"data_classification": {},
			"access_policy":       {},
			"data_lineage":        {},
			"extra_config":        {},
		},
	}
	assert.NoError(t, ValidateRequired(reg))
}

func TestValidateRequired_OnlyOptionalPresent_Extra(t *testing.T) {
	reg := &ConfigRegistry{
		RawConfigs: map[string]RawConfig{
			"data_markings":    {},
			"health_checks":    {},
			"checkpoint_rules": {},
			"alert_rules":      {},
			"metrics":          {},
		},
	}
	err := ValidateRequired(reg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "aip_object_schema")
	assert.Contains(t, err.Error(), "data_classification")
	assert.Contains(t, err.Error(), "access_policy")
	assert.Contains(t, err.Error(), "data_lineage")
}

// ──── parseObjectSchema complex structures ────────────────────────────

func TestParseObjectSchema_ComplexStructure_Extra(t *testing.T) {
	yaml := `objects:
  - object_type_id: order
    display_name: "Order"
    source_tables: [orders, order_items]
    grain: order_id
    allow_sync_to_feishu: true
    properties:
      order_id:
        type: string
        is_pk: true
      total_value:
        type: numeric
        source: payment_value
        agg: sum
    relationships:
      customer:
        to: customer
        grain: customer_id
    alert_fields:
      - total_value
      - status
`
	cfg, err := parseObjectSchema([]byte(yaml))
	require.NoError(t, err)
	require.Len(t, cfg.Objects, 1)

	obj := cfg.Objects[0]
	assert.Equal(t, "order", obj.ObjectTypeID)
	assert.Equal(t, "Order", obj.DisplayName)
	assert.Len(t, obj.SourceTables, 2)
	assert.Equal(t, "order_id", obj.Grain)
	assert.True(t, obj.AllowSyncToFeishu)

	assert.True(t, obj.Properties["order_id"].IsPK)
	assert.Equal(t, "numeric", obj.Properties["total_value"].Type)
	assert.Equal(t, "payment_value", obj.Properties["total_value"].Source)
	assert.Equal(t, "sum", obj.Properties["total_value"].Agg)

	assert.Equal(t, "customer", obj.Relationships["customer"].To)
	assert.Equal(t, "customer_id", obj.Relationships["customer"].Grain)

	assert.Len(t, obj.AlertFields, 2)
}

// ──── parseDataClassification with applies_to_fields ──────────────────

func TestParseDataClassification_WithAppliesToFields_Extra(t *testing.T) {
	yaml := `classifications:
  - asset_ref: "customer"
    level: internal
    rationale: "Customer data"
    applies_to_fields:
      email: pii
      name: internal
      phone: sensitive
`
	cfg, err := parseDataClassification([]byte(yaml))
	require.NoError(t, err)
	require.Len(t, cfg.Classifications, 1)
	assert.Len(t, cfg.Classifications[0].AppliesToFields, 3)
	assert.Equal(t, "pii", cfg.Classifications[0].AppliesToFields["email"])
}

// ──── parseAccessPolicy with multiple roles ──────────────────────────

func TestParseAccessPolicy_MultipleRoles_Extra(t *testing.T) {
	yaml := `access_policy:
  roles:
    - role: admin
      allowed_actions: [read, write, delete]
      data_access: [all]
    - role: analyst
      allowed_actions: [read]
      data_access: [metrics, reports]
    - role: viewer
      allowed_actions: [read]
      data_access: [public_data]
  default_policy: deny
`
	cfg, err := parseAccessPolicy([]byte(yaml))
	require.NoError(t, err)
	require.Len(t, cfg.AccessPolicy.Roles, 3)
	assert.Equal(t, "admin", cfg.AccessPolicy.Roles[0].Role)
	assert.Equal(t, "analyst", cfg.AccessPolicy.Roles[1].Role)
	assert.Equal(t, "viewer", cfg.AccessPolicy.Roles[2].Role)
	assert.Equal(t, "deny", cfg.AccessPolicy.DefaultPolicy)
}

// ──── parseDataLineage with multiple edges ───────────────────────────

func TestParseDataLineage_MultipleEdges_Extra(t *testing.T) {
	yaml := `nodes:
  - id: raw.orders
    type: table
    label: Orders
    status: active
    linked_to: source_system
edges:
  - from: raw.orders
    to: dwd.order_level
    transform: sql_aggregate
    transform_type: sql_aggregation
  - from: dwd.order_level
    to: mart.metric_daily
    transform: daily_agg
    transform_type: batch_load
`
	cfg, err := parseDataLineage([]byte(yaml))
	require.NoError(t, err)
	require.Len(t, cfg.Nodes, 1)
	require.Len(t, cfg.Edges, 2)
	assert.Equal(t, "raw.orders", cfg.Nodes[0].ID)
	assert.Equal(t, "active", cfg.Nodes[0].Status)
	assert.Equal(t, "source_system", cfg.Nodes[0].LinkedTo)
	assert.Equal(t, "batch_load", cfg.Edges[1].TransformType)
}

// ──── parseCheckpointRules complex ──────────────────────────────────

func TestParseCheckpointRules_Complex_Extra(t *testing.T) {
	yaml := `checkpoints:
  execute_dispatch:
    scope: global
    requires_justification: true
    prompt: "Are you sure?"
    record_fields: [action, reason]
    checkpoint_types: [approval, audit]
  data_export:
    scope: dwd.*
    endpoint: /api/export
    requires_justification: false
checkpoint_audit:
  file: audit.csv
  format: csv
  columns: [timestamp, action, actor]
  retention_days: 90
frequency:
  default: daily
  cache_same_rule: false
`
	cfg, err := parseCheckpointRules([]byte(yaml))
	require.NoError(t, err)
	assert.Len(t, cfg.Checkpoints, 2)

	dispatch := cfg.Checkpoints["execute_dispatch"]
	assert.True(t, dispatch.RequiresJustification)
	assert.Equal(t, "Are you sure?", dispatch.Prompt)
	assert.Len(t, dispatch.RecordFields, 2)
	assert.Len(t, dispatch.CheckpointTypes, 2)

	export := cfg.Checkpoints["data_export"]
	assert.False(t, export.RequiresJustification)
	assert.Equal(t, "/api/export", export.Endpoint)

	assert.Equal(t, 90, cfg.CheckpointAudit.RetentionDays)
	assert.Equal(t, "daily", cfg.Frequency.Default)
	assert.False(t, cfg.Frequency.CacheSameRule)
}

// ──── parseAlertRules complex ────────────────────────────────────────

func TestParseAlertRules_MultipleRules_Extra(t *testing.T) {
	yaml := `rules:
  - rule_id: gmv_drop
    metric: gmv
    condition: "day_over_day < -0.2"
    severity: high
    owner_role: business_ops
    dimension: seller_id
    min_sample_size: 100
    description: GMV drop detection
  - rule_id: cancel_spike
    metric: cancel_rate
    condition: "rate > 0.1"
    severity: critical
    owner_role: logistics_ops
    dimension: region
    min_sample_size: 30
    description: Cancel rate spike
`
	cfg, err := parseAlertRules([]byte(yaml))
	require.NoError(t, err)
	require.Len(t, cfg.Rules, 2)
	assert.Equal(t, "gmv_drop", cfg.Rules[0].RuleID)
	assert.Equal(t, "cancel_spike", cfg.Rules[1].RuleID)
	assert.Equal(t, 100, cfg.Rules[0].MinSampleSize)
	assert.Equal(t, 30, cfg.Rules[1].MinSampleSize)
}

// ──── parseMetrics complex ──────────────────────────────────────────

func TestParseMetrics_MultipleMetrics_Extra(t *testing.T) {
	yaml := `metrics:
  gmv:
    business_definition: "Gross Merchandise Value"
    source_expression: "SUM(amount)"
    grain: daily
    dimensions: [seller_id, region]
    owner_role: business_ops
    window: [daily, weekly]
    unit: BRL
  avg_order_value:
    business_definition: "Average Order Value"
    source_expression: "AVG(order_value)"
    grain: daily
    dimensions: [seller_id]
    owner_role: business_ops
    window: [daily]
    unit: BRL
`
	cfg, err := parseMetrics([]byte(yaml))
	require.NoError(t, err)
	assert.Len(t, cfg.Metrics, 2)
	assert.Equal(t, "Gross Merchandise Value", cfg.Metrics["gmv"].BusinessDefinition)
	assert.Equal(t, "BRL", cfg.Metrics["gmv"].Unit)
	assert.Len(t, cfg.Metrics["gmv"].Dimensions, 2)
	assert.Len(t, cfg.Metrics["gmv"].Window, 2)
	assert.Equal(t, "Average Order Value", cfg.Metrics["avg_order_value"].BusinessDefinition)
}

// ──── computeHash additional tests ──────────────────────────────────

func TestComputeHash_LargeContent_Extra(t *testing.T) {
	content := make([]byte, 1024*1024) // 1MB
	for i := range content {
		content[i] = byte(i % 256)
	}
	h := computeHash(content)
	assert.Len(t, h, 64) // SHA256 hex is 64 chars
}

func TestComputeHash_BinaryContent_Extra(t *testing.T) {
	content := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
	h := computeHash(content)
	assert.Len(t, h, 64)
}

// ──── yamlToJSON edge cases ──────────────────────────────────────────

func TestYAMLToJSON_NestedList_Extra(t *testing.T) {
	yaml := "items:\n  - name: a\n    value: 1\n  - name: b\n    value: 2\n"
	json, err := yamlToJSON([]byte(yaml))
	require.NoError(t, err)
	assert.Contains(t, string(json), "items")
	assert.Contains(t, string(json), "name")
}

func TestYAMLToJSON_BooleanValues_Extra(t *testing.T) {
	yaml := "enabled: true\ndisabled: false\n"
	json, err := yamlToJSON([]byte(yaml))
	require.NoError(t, err)
	assert.Contains(t, string(json), "true")
	assert.Contains(t, string(json), "false")
}

// ──── ListConfigKeys ────────────────────────────────────────────────

func TestListConfigKeys_SingleKey_Extra(t *testing.T) {
	reg := &ConfigRegistry{
		RawConfigs: map[string]RawConfig{
			"only_key": {},
		},
	}
	keys := ListConfigKeys(reg)
	assert.Equal(t, []string{"only_key"}, keys)
}

// ──── RawConfig struct ──────────────────────────────────────────────

func TestRawConfig_Fields_Extra(t *testing.T) {
	raw := RawConfig{
		ConfigKey:   "test_key",
		ConfigType:  "test_type",
		SourcePath:  "test.yml",
		Content:     []byte("key: value"),
		ContentHash: "abc123",
	}
	assert.Equal(t, "test_key", raw.ConfigKey)
	assert.Equal(t, "test_type", raw.ConfigType)
	assert.Equal(t, "test.yml", raw.SourcePath)
	assert.Equal(t, []byte("key: value"), raw.Content)
	assert.Equal(t, "abc123", raw.ContentHash)
}

// ──── ObjectSchemaConfig ─────────────────────────────────────────────

func TestObjectSchemaConfig_EmptyProperties_Extra(t *testing.T) {
	yaml := `objects:
  - object_type_id: simple
    display_name: Simple Object
`
	cfg, err := parseObjectSchema([]byte(yaml))
	require.NoError(t, err)
	require.Len(t, cfg.Objects, 1)
	assert.Empty(t, cfg.Objects[0].Properties)
	assert.Empty(t, cfg.Objects[0].SourceTables)
}

// ──── HealthChecksConfig ─────────────────────────────────────────────

func TestHealthChecksConfig_MultipleViews_Extra(t *testing.T) {
	yaml := `monitoring_views:
  - id: view1
    scope: table1
    check_type: completeness
    rule: count(*) > 0
    severity: high
    alert_channels: [feishu]
  - id: view2
    scope: table2
    check_type: freshness
    rule: max(ts) > NOW() - INTERVAL '1 hour'
    severity: critical
    alert_channels: [feishu, email]
health_checks:
  - id: check1
    resource: table1
    description: Check 1
    check_type: row_count
    validation: count(*) > 100
    severity: medium
    alert_channels: [feishu]
`
	cfg, err := parseHealthChecks([]byte(yaml))
	require.NoError(t, err)
	assert.Len(t, cfg.MonitoringViews, 2)
	assert.Len(t, cfg.HealthChecks, 1)
	assert.Contains(t, cfg.MonitoringViews[1].AlertChannels, "email")
	assert.Contains(t, cfg.HealthChecks[0].AlertChannels, "feishu")
}

// ──── parseConfigByType branches ────────────────────────────────────

func TestParseConfigByType_AllBranchesExtra(t *testing.T) {
	tests := []struct {
		configType string
	}{
		{"data_classification"},
		{"access_policy"},
		{"data_lineage"},
		{"data_markings"},
		{"health_checks"},
		{"checkpoint_rules"},
		{"alert_rules"},
		{"metrics"},
	}
	for _, tc := range tests {
		t.Run(tc.configType, func(t *testing.T) {
			result, err := parseConfigByType(tc.configType, []byte("key: value"))
			require.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

// ──── detectConfigType passthrough ──────────────────────────────────

func TestDetectConfigType_PassthroughUnknown_Extra(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"custom_config", "custom_config"},
		{"another_config", "another_config"},
		{"", ""},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, detectConfigType(tc.input))
		})
	}
}

func TestWriteFile(t *testing.T) {
	// verify our test helper works
	dir := t.TempDir()
	writeFile(t, dir, "test.yml", "key: value")
	data, err := os.ReadFile(filepath.Join(dir, "test.yml"))
	require.NoError(t, err)
	assert.Equal(t, "key: value", string(data))
}
