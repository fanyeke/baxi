# Ontology v1/v2 Coverage Report

**Generated**: 2026-06-02
**Source Config**: `config/aip_object_schema.yml` (v1), `config/aip_object_schema_v2.yml` (v2)
**Go Packages**: `internal/ontology/`, `internal/mcp/`, `internal/action/`

---

## 1. Object Coverage Summary

| Dimension | v1 Objects | v2 Objects | Dual (v2 Exists, v1 Fallback Active) |
|-----------|-----------|-----------|---------------------------------------|
| Total object types | 9 | 11 | 8 |
| Fully defined types | 9 | 7 (P1) + 4 (P2) | See Section 2 |

### v1-Only Objects (no v2 migration yet)

| Object | Notes |
|--------|-------|
| `global` | Exists in v1 schema but has NO v2 definition. Metrics-only aggregate type. Safe to remove v1 once migrated to v2. |

### Objects Present in Both v1 and v2

| Object | v1 Type | v2 Type | Migration Status |
|--------|---------|---------|------------------|
| `seller` | Yes | Yes (P1) | Fully migrated |
| `order` | Yes | Yes (P1) | Fully migrated |
| `product` | Yes | Yes (P1) | Fully migrated |
| `customer` | Yes | Yes (P1) | Fully migrated |
| `category` | Yes | Yes (P1) | Fully migrated |
| `region` | Yes | Yes (P1) | Fully migrated |
| `metric_alert` | Yes | Yes (P1) | Fully migrated |
| `marketing_lead` | Yes | Yes (P2) | Migrated, P2 test validates |

### v2-Only Objects (new in v2, no v1 equivalent)

| Object | Phase | Notes |
|--------|-------|-------|
| `review` | P2 | New semantic entity; no v1 equivalent |
| `payment` | P2 | New semantic entity; no v1 equivalent |
| `shipment` | P2 | New semantic entity; no v1 equivalent |

---

## 2. Per-Object Feature Coverage

### P1 Objects (fully supported)

| Feature | seller | order | product | customer | category | region | metric_alert |
|---------|--------|-------|---------|----------|----------|--------|--------------|
| **source** (schema/table/PK) | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| **properties** | 7 fields | 5 fields | 5 fields | 7 fields | 6 fields | 7 fields | 6 fields |
| **llm_readable** | Yes (5/7) | Yes (4/5) | Yes (4/5) | Yes (5/7) | Yes (5/6) | Yes (6/7) | Yes (5/6) |
| **searchable** | seller_id | order_id | product_id | customer_id | category_id | region_id | alert_id |
| **filterable** | seller_state, seller_city | order_status, delivery_status | product_category_name | customer_city, customer_state | category_name | region_name | rule_id, severity |
| **metrics** | 3 refs | None | None | 3 refs | 3 refs | 3 refs | None |
| **links** | 2 (recent_orders, products) | 6 (customer, seller, product, payment, shipment, reviews) | 3 (category, reviews, orders) | 2 (recent_orders, reviews) | 2 (products, top_sellers) | 2 (sellers, customers) | None |
| **actions** | 3 | 1 | None | 2 | 2 | 2 | None |
| **alert_fields** | 3 | 2 | 1 | 2 | 2 | 2 | None |
| **governance** | agent_readonly, redact_pii: true | agent_readonly, redact_pii: false | agent_readonly, redact_pii: false | agent_readonly, redact_pii: true | agent_readonly, redact_pii: false | agent_readonly, redact_pii: false | agent_readonly, redact_pii: false |
| **LLM access** | read-only | read-only | read-only | read-only | read-only | read-only | read-write |

### P2 Objects (new in v2, validated by tests)

| Feature | review | payment | shipment | marketing_lead |
|---------|--------|---------|----------|----------------|
| **source** (schema/table/PK) | dwd.order_level/order_id | dwd.order_level/order_id | dwd.order_level/order_id | dwd.item_level/seller_id |
| **properties** | 7 fields | 6 fields | 8 fields | 7 fields |
| **llm_readable** | Yes (3/7) | Yes (4/6) | Yes (5/8) | Yes (5/7) |
| **searchable** | review_id, order_id, product_id, customer_id | payment_id, order_id | shipment_id, order_id | lead_id, seller_id |
| **filterable** | review_score | payment_type, payment_status | carrier, delivery_status | campaign_source, conversion_status |
| **metrics** | None | None | None | None |
| **links** | 3 (order, product, customer) | 1 (order) | 2 (order, seller) | 1 (seller) |
| **actions** | 1 | 2 | 2 | 2 |
| **alert_fields** | 1 | 2 | 2 | 2 |
| **governance** | redact_pii: true | redact_pii: false | redact_pii: false | redact_pii: true |
| **LLM access** | read-only | read-only | read-only | read-only |

### v1 Legacy Object with v2 Equivalent Needed

| Feature | global (v1 only) |
|---------|-------------------|
| **source** | metric_daily |
| **properties** | 4 fields (metric_name, current_value, baseline_value, change_pct) |
| **alert_fields** | current_value |
| **v2 migration** | NOT STARTED — no v2 definition exists |

---

## 3. Context Recipe Coverage

| Recipe | Object Type | Trigger Rule | Status |
|--------|-------------|-------------|--------|
| `seller_late_delivery_alert` | seller | seller_late_delivery_spike | v2 recipe loaded and tested |
| `order_anomaly_alert` | region | order_anomaly | v2 recipe loaded and tested |
| `customer_churn_risk_alert` | customer | customer_churn_risk | v2 recipe loaded and tested |
| `product_performance_drop_alert` | product | product_performance_drop | v2 recipe loaded and tested |
| `inventory_stockout_alert` | category | inventory_stockout | v2 recipe loaded and tested |

All 5 context recipes are defined in `config/context_recipes.yml` and validated by `TestIntegration_LoadContextRecipes_AllFive`.

---

## 4. Metric Definition Coverage

| Object Type | v2 Metrics | Count |
|-------------|-----------|-------|
| seller | seller_late_delivery_rate_7d, seller_order_count_7d, seller_gmv_7d, seller_avg_review_score_7d, seller_cancel_rate_7d | 5 |
| product | product_gmv_7d, product_order_count_7d, product_avg_review_score_7d, product_review_drop_7d | 4 |
| category | category_gmv_7d, category_order_count_7d, category_avg_review_score_7d | 3 |
| region | region_order_count_7d, region_late_delivery_rate_7d, region_gmv_7d | 3 |
| customer | customer_order_count_90d, customer_gmv_90d, customer_avg_review_score | 3 |
| **Total** | | **18** |

Note: `order`, `metric_alert`, `review`, `payment`, `shipment`, `marketing_lead` have NO metric definitions in v2.

---

## 5. MCP Tool Coverage (17 registered ontology+action+context tools)

### v2 Only (no v1 equivalent, fully on v2)

| Tool | Handler | Implementation | Runtime Path |
|------|---------|----------------|-------------|
| `build_context` | `handleBuildContext` | `BuildContextService.BuildEnvelope` (RecipeContextBuilder) | `internal/mcp/tools_context.go` |
| `propose_action` | `handleProposeAction` | `ontologyServiceAdapter.ProposeAction` | `internal/mcp/tools_action.go` |
| `execute_proposal` | `handleExecuteProposal` | `ApplyService.ExecuteProposal` (enforces v2 approval) | `internal/mcp/tools_action.go` |

### v1 Primary, v2 Enhanced (dual path)

| Tool | v1 Path | v2 Path | Fallback Logic |
|------|---------|---------|----------------|
| `get_linked_objects` | `getLinkedObjectsV1` (Via model, single-value) | `getLinkedObjectsV2` (LinkResolver, multi-cardinality) | v2 First, fallback to v1 on error or if linkResolver=nil |
| `search_objects` | `ObjectQueryService.SearchObjects` | `V2QueryCompiler.CompileSearchObjects` | v2 compiler set on repo if v2 objects exist |

### v1 Only (no v2 rewrite)

| Tool | Handler | v1 Service | Notes |
|------|---------|-----------|-------|
| `describe_ontology` | `handleDescribeOntology` | `ObjectRegistry.ListObjectTypes` (v1 list only) | **Does NOT enumerate v2 objects** — only returns types from v1 schema. Needs update to merge v2 objects. |
| `get_object` | `handleGetObject` | `ObjectQueryService.BuildObjectContext` | Uses v1 query service; metrics via `ontRepo.GetObjectMetrics` |
| `get_decision_context` | `handleGetDecisionContext` | `ContextBuilder.BuildDecisionContext` | Uses v1/v3 context builder from main.go wiring |

### Action Tools (v2 safety pattern)

| Tool | v1 Behavior (legacy) | v2 Behavior (current) |
|------|---------------------|----------------------|
| `execute_action` | Auto-approved, dry_run=false, immediate execution | **Default dry_run=true. Rejects non-approved execution. Returns "use propose_action" message.** |

### Non-Ontology Tools (not affected by v1/v2 migration)

| Category | Tools |
|----------|-------|
| Decision | `create_decision_case`, `decide`, `list_cases`, `get_case`, `resolve_case` |
| Alert | `list_alerts` |
| Governance | `check_access`, `get_classification` |
| Pipeline | `run_pipeline`, `get_pipeline_status` |
| Review | `approve_proposal`, `reject_proposal`, `cancel_proposal`, `get_proposal_by_id`, `list_review_records` |
| Status | `get_system_status` |
| Outbox | `list_outbox_events` |
| Schema | `list_action_schemas`, `get_action_schema` |
| Sandbox | `create_sandbox`, `add_to_sandbox`, `compare_sandboxes`, `get_sandbox` |

---

## 6. Migration Recommendation: When v1 Can Be Safely Removed

### Preconditions for v1 removal

1. **`describe_ontology` must be updated** to merge v2 object types. Currently it only iterates `registry.ListObjectTypes()` (v1 objects). V2 objects live in `registry.objectsV2` and are never returned.

2. **`get_object` must use v2 query compilation** as primary path, not just a transparent compiler drop-in. Currently the v2 compiler is set on `ontologyRepo` and used transparently by the repository — if all v2 objects compile correctly, v1 is unused here.

3. **`get_object_metrics` must have a v2 equivalent**. Currently uses `ontRepo.GetObjectMetrics` which still follows the v1 metric query pattern. Metrics for v2 objects should go through `MetricQueryResolver`.

4. **`global` object must be defined in v2 schema** or all references to it must be removed.

5. **Context recipes and metric definitions** are already fully on v2 config format.

### Recommended v1 removal timeline

| Phase | Objects/Migration | Condition |
|-------|------------------|-----------|
| **Now** | P1 objects (7): seller, order, product, customer, category, region, metric_alert | v2 fully supported. v1 fallback still active for describe_ontology and get_object. |
| **Phase 1** (immediate) | Add `global` to v2 schema | Blocking: without it, the v1 YAML cannot be removed. |
| **Phase 2** | Update `describe_ontology` to include v2 objects | Currently blocks v1 removal because it only reads v1 objects. |
| **Phase 3** | Remove v1 `aip_object_schema.yml`, `aip_object_schema_v1` loader, `ObjectType` (v1 struct) | All v1-only code paths must be verified unused. |

### Critical Migration Gaps (must fix before v1 removal)

1. **`describe_ontology` does not expose v2 objects** — `ontologyServiceAdapter.DescribeOntology` only iterates `registry.ListObjectTypes()` (v1). V2 objects are never included.
2. **`global` object has no v2 definition** — it is defined in `aip_object_schema.yml` but missing from `aip_object_schema_v2.yml`.
3. **No v2 get_object response includes v2-only fields** (Metrics, Governance, Source) — the v2 QueryCompiler compiles SQL but the response is still the v1 `ObjectContext` struct.

---

## 7. Test Coverage

| Package | Coverage | Test Files | Notes |
|---------|----------|-----------|-------|
| `internal/ontology/` | **44.2%** | 11 test files | Configuration loading, validation, link compilation, YAML parsing. V2 integration tests cover config loading and v2 object validation. |
| `internal/mcp/` | **16.4%** | 1 test file (`server_test.go`) | Server construction, tool registration, identity enforcement. **No handler-level tests** for ontology/action/context tools. No test coverage for `execute_action`, `get_linked_objects`, `build_context`, `propose_action`. |
| `internal/action/` | **72.8%** | 8 test files | ApplyService, proposal service, executors, schema, registry, helpers. Good coverage of core action logic. |

### Specific Test Gaps

| Area | Gap | Risk |
|------|-----|------|
| `get_linked_objects` v2 fallback | No test for v2 resolving a link then falling back to v1 on error | Medium |
| `get_linked_objects` v2 path | No MCP handler test for v2 link resolution with actual compiled SQL | Medium |
| `build_context` | No MCP handler test (only ontology config loading tests exist) | Medium |
| `propose_action` | No handler test for proposal creation path | Medium |
| `execute_action` dry_run behavior | No handler test for the dry_run=true default and rejection of false | Medium |
| `describe_ontology` v2 omission | No test verifying that v2 objects are NOT included (known gap) | Low |
| `ontologyServiceAdapter` | No unit tests for adapter methods (DescribeOntology, GetObject, etc.) | High |
| V2 QueryCompiler integration | Tests exist for ParseObjectSchemaV2 but not for actual DB query compilation | Medium |

### Overall Project Coverage

Per the 2026-05-29 audit: **34.4% overall** (threshold: 80%). The ontology/mcp/action packages alone account for a significant portion of uncovered code.

---

## 8. Key Files

| File | Role |
|------|------|
| `config/aip_object_schema.yml` | v1 object schema (deprecated, kept for backward compat) |
| `config/aip_object_schema_v2.yml` | v2 object schema (11 object types) |
| `config/metric_definitions.yml` | Metric definitions (18 metrics across 5 object types) |
| `config/context_recipes.yml` | Context recipes (5 recipes) |
| `internal/ontology/schema.go` | v1 ObjectType, ObjectProperty, ObjectLink structs |
| `internal/ontology/schema_v2.go` | v2 ObjectTypeV2, ObjectPropertyV2, ObjectLinkV2 structs |
| `internal/ontology/registry.go` | ObjectRegistry (v1+v2 dual registry) |
| `internal/ontology/registry_v2.go` | v2 YAML loading (ParseObjectSchemaV2, ParseContextRecipes, ParseMetricDefinitions) |
| `internal/ontology/context_recipe.go` | ContextRecipe struct and loaders |
| `internal/ontology/link_plan.go` | LinkResolver (v2 link compilation) |
| `internal/ontology/query_plan.go` | QueryCompiler (v2 query compilation) |
| `cmd/baxi-mcp/main.go` | Wiring: v2 builder, buildContextSvc, linkResolver, QueryCompiler |
| `internal/mcp/tools_ontology.go` | Ontology tools (describe_ontology, get_object, get_linked_objects, execute_action) |
| `internal/mcp/tools_action.go` | Action tools (execute_proposal, propose_action, get_decision_context) |
| `internal/mcp/tools_context.go` | Context tool (build_context) |
| `internal/action/apply_service.go` | ApplyService (approval enforcement, dry-run, version gating) |

---

## 9. Summary: v1/v2 Feature Matrix

```
Feature                    v1      v2      Status
──────────────────────────────────────────────────
Object definitions         9       11      v2 has 3 new (review, payment, shipment)
  └ P1 objects             7        7      Fully dual/migrated
  └ P2 objects             0        4      v2 only (review, payment, shipment, marketing_lead)
  └ v1-only objects        1        0      global needs migration
Source schema/table        flat    structured  v2 adds explicit schema.table.PK
Properties                 basic   rich    v2 adds Expression, MetricRef, Searchable, Filterable
Links (relationships)      basic   rich    v2 adds cardinality, strategy, limit, sort, fields
  └ one_to_many            NO      YES     v2 breakthrough feature
Metric definitions         18      18      Same metric YAML, v2 adds property.metric_ref
Context recipes            5       5       Same recipe YAML, loaded by v2 RecipeContextBuilder
Action bindings            flat    per-type v2 binds actions at object level
Governance policy          NO      YES     v2 adds default_role + redact_pii per object
LLM access policy          YES     YES     Same policy model
Query compilation          runtime structured v2 QueryCompiler generates parameterized SQL
Link resolution            flat    multi-  v2 LinkResolver: direct_key, reverse_lookup, bridge_table, query_ref
MCP describe_ontology      v1-only never   describe_ontology does not include v2 objects
MCP get_linked_objects     v1 fallback primary v2-first with v1 fallback
MCP build_context          N/A     YES     v2-only; RecipeContextBuilder
MCP propose_action         N/A     YES     v2-only; creates "proposed" proposals
MCP execute_action         unsafe  safe    v2: dry_run=true default, rejects unapproved
MCP execute_proposal       N/A     YES     v2: requires "approved" status
```
