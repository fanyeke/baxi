# CONFIG: YAML Governance & Business Configs

**Generated:** 2026-05-28
**Commit:** d908f6d
**Branch:** main

## OVERVIEW

28 flat YAML files governing data policies, alert rules, metrics, actions, adapters, and Feishu integrations. Parsed at startup by `internal/configloader/` with type-specific deserialization and whitelist enforcement via `internal/action/`.

MCP configuration: `action_registry.yml` controls the MCP tool whitelist — only actions explicitly listed and enabled in the YAML are surfaced as MCP tools. The MCP server (`cmd/baxi-mcp/`) reads these same configs alongside the API server for consistent enforcement.

## WHERE TO LOOK

| Task | File | Notes |
|------|------|-------|
| Governance rules | `access_policy.yml`, `data_classification.yml`, `data_lineage.yml`, `data_markings.yml` | Object-level ACLs, sensitivity tags, lineage config |
| Alert configs | `alert_rules.yml`, `dimensional_alert_rules.yml` | Thresholds, dimensions, notification routing |
| Metrics | `metrics.yml` | KPI definitions, aggregation specs |
| Action types | `action_registry.yml`, `action_templates.yml` | Registry with payload schema, risk level, required approval |
| Feishu sync | `feishu_app.yml`, `feishu_base_schema.yml`, `feishu_field_mapping.yml`, `feishu_table_ids.yml` | Base schema, field mappings, table IDs |
| Channel routing | `channel_routing_rules.yml`, `adapter_registry.yml` | Dispatch to Feishu/GitHub/CLI/Manual adapters |
| Data quality | `data_quality_rules.yml`, `data_catalog.yml`, `data_lineage.yml` | Quality checks, catalog metadata |
| Decision engine | `decision_eval_rules.yml` | Case eval rules for the decision engine |
| Operations | `health_checks.yml`, `checkpoint_rules.yml`, `retention_policies.yml`, `owner_mapping.yml` | Pipeline health, snapshots, retention, ownership |
| LLM | `llm_config.yml`, `qoder_capabilities.yml` | LLM provider config, capability definitions |
| Enums & contracts | `status_enums.yml`, `wake_io_contract.yml`, `aip_object_schema.yml` | Shared enums, integration contracts |

## CONVENTIONS

- **Whitelist enforcement**: `action_registry.yml` intersected with hard-coded `CanonicalActions` in `internal/action/registry.go`. Only explicitly listed + enabled action types are allowed at runtime.
- **Type-specific parsing**: `internal/configloader/yaml.go` dispatches by config type (`access_policy`, `data_classification`, `health_checks`, etc.) to typed Go structs. Unknown types fall through to `parseAny`.
- **Required configs**: `aip_object_schema`, `data_classification`, `access_policy`, `data_lineage` must exist. Validated by `ValidateRequired` in `configloader/validator.go`.
- **`.example` files skipped**: `feishu_table_ids.yml.example` is ignored by the loader.
- **Flat hierarchy**: All 28 files sit at `config/*.yml`. No subdirectories.

## ANTI-PATTERNS

- **No schema validation at edit time**: YAML files are only validated at Go startup. Invalid entries produce slog warnings, not hard failures (except required configs).
- **Mixed domains**: Single flat directory mixes governance (access_policy), operations (health_checks), and integration (feishu_*) configs with no subdirectory separation.
- **`.example` files in-tree**: `feishu_table_ids.yml.example` ships alongside live configs with no load-time isolation beyond the skip logic.
