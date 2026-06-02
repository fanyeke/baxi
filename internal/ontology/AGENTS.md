# ontology: AIP Object Schema, V2 Extensions, and Action Binding

## OVERVIEW

The `ontology` package provides the AIP semantic object layer — type definitions, registry, validation, query compilation, link resolution, context recipes, metric definitions, and action binding for the Baxi governance platform.

Migration to v2 introduces richer schema semantics: structured data sources, searchable/filterable properties, multi-cardinality relationships, metric contracts, object-level action binding, and governance policies.

## STRUCTURE

| File | Responsibility |
|------|---------------|
| `schema.go` | V1 `ObjectType`, `ObjectProperty`, `ObjectLink` types + YAML parsing |
| `schema_v2.go` | V2 `ObjectTypeV2`, `ObjectSource`, `ObjectPropertyV2`, `ObjectLinkV2`, `LinkTarget`, `ObjectGovernancePolicy`, `CompiledQuery` types + YAML parsing types |
| `object_type.go` | Type constants (`TypeSeller`, `TypeOrder`, …), `AllObjectTypes()`, display names |
| `registry.go` | `ObjectRegistry` — loads from DB (gov.object_schema) or YAML fallback, concurrently safe, with `objectsV2` field |
| `registry_v2.go` | V2 YAML loader: `ParseObjectSchemaV2`, `LoadObjectSchemaV2`, `ParseMetricDefinitions`, `ParseContextRecipes` (also in separate files) |
| `validator.go` | V1 `Validate()` on ObjectRegistry — checks 8 expected types, grains, PKs, links |
| `validator_v2.go` | V2 `ValidateV2()` standalone — source completeness, PK count, link target existence |
| `metric_definition.go` | `MetricDefinition` types + YAML parsing + `MetricResolver` |
| `context_recipe.go` | `ContextRecipe` types + YAML parsing for LLM-safe context building |
| `query_plan.go` | `QueryCompiler` — compiles v2 schema into safe SQL, methods: `CompileObjectQuery` |
| `compiler.go` | Extended `QueryCompiler` — `CompileGetObject`, `CompileSearchObjects` (with filters/sort/pagination), `CompileObjectMetrics` |
| `link_plan.go` | **(NEW)** `LinkResolver` — resolves v2 relationships with 4 strategies: `direct_key`, `reverse_lookup`, `bridge_table`, `query_ref`. Contains `ObjectRef`, `LinkOptions`, `ObjectInstance`, `LinkedObjectResult`, `CompiledLink` types |
| `action_binding.go` | `ActionProposal` (lifecycle type), `ActionBindingValidator` (pre-execution constraint checks), `ValidatePayload`, `ValidateApproval` |
| `context_builder.go` | `ContextBuilder` — `BuildLLMSafeContext` with role-based redaction |
| `ontology_aware_adapter.go` | Bridge between ontology types and repository layer |

## V2 SCHEMA COMPONENTS

### ObjectTypeV2
- **Source**: `ObjectSource{schema, table, primary_key}` — structured physical source config
- **Properties**: `ObjectPropertyV2` — adds Searchable, Filterable, Expression, MetricRef, LLMReadable flags
- **Links**: `ObjectLinkV2` — adds Cardinality (one_to_one, one_to_many), Strategy (direct_key, reverse_lookup, bridge_table, query_ref), explicit SourceKey, LinkTarget config
- **Metrics**: `[]string` — references to metric definitions
- **AllowedActions**: `[]string` — action types bound to object
- **Governance**: `ObjectGovernancePolicy{DefaultRole, RedactPII}`

### LinkResolver Strategies
1. **direct_key** — source key matches target PK: `SELECT cols FROM target WHERE target.pk = source.id`
2. **reverse_lookup** — target holds source key: `SELECT cols FROM target WHERE target.key = source.id`
3. **bridge_table** — join through intermediate table
4. **query_ref** — use predefined SQL template with `$1` placeholder

### ActionBindingValidator
- `Validate(objectType, actionType, role)` — full validation chain
- `ValidatePayload(schema, payload)` — required field checking
- `ValidateApproval(actionType, isApproved)` — approval constraint check
- `SetActionRegistry(allowedBy, actionEnabled)` — configure from YAML

## CONFIG FILES

| Config | Path | Parsed By |
|--------|------|-----------|
| V2 Object Schema | `config/aip_object_schema_v2.yml` | `ParseObjectSchemaV2` / `LoadObjectSchemaV2` |
| Metric Definitions | `config/metric_definitions.yml` | `ParseMetricDefinitions` / `LoadMetricDefinitions` |
| Context Recipes | `config/context_recipes.yml` | `ParseContextRecipes` / `LoadContextRecipes` |

## KEY PATTERNS

- **Standalone validation**: V2 `ValidateV2` is package-level (no registry needed), returns `[]ValidationIssue`
- **Builder pattern**: `NewLinkResolver(objects)` → `GetLinkedObjects(source, linkName, opts)` returns compiled plan
- **Backward compatible**: V1 types unchanged; V2 added as separate types with same package
- **MCP integration**: Server struct has optional `linkResolver`, `actionBindingValidator`, `objectTypesV2` fields with setter methods

## TEST FILES

| File | Tests |
|------|-------|
| `validator_v2_test.go` | ValidateV2 unit tests (source, PK, properties, link targets) |
| `ontology_v2_integration_test.go` **(NEW)** | End-to-end: load v2 YAML → validate → parse metrics → parse recipes → compile links |
| `registry_test.go` | V1 registry, YAML loading, convertRawObject |
| `schema_test.go` | V1 schema parsing |
| `validator_test.go` | V1 Validate |
| `object_type_test.go` | Type constants, display names |
| `ontology_coverage_test.go` | Coverage edge cases |
| `ontology_query_test.go` | Query compilation scenarios |
| `query_service_test.go` | Query service integration |

## COMMANDS

```bash
go test ./internal/ontology/        # unit + integration tests
go test ./internal/ontology/ -run TestIntegration  # integration tests only
```

## MCP INTEGRATION (Productionized)

The following v2 capabilities are now wired into the MCP server (`cmd/baxi-mcp/main.go`):

| Capability | Status | Entry Point |
|-----------|--------|-------------|
| `build_context` | ✅ Wired | `RecipeContextBuilder` constructed with 7 deps (caseSvc, compiler, metricQuery, linkExec, pool, actionTypes, recipes) |
| `get_linked_objects` | ✅ Wired | `ontologyServiceAdapter` uses v2 `LinkResolver` first, falls back to v1 Via model |
| `propose_action` | ✅ Wired | Creates `ActionProposalRow{ApplyStatus: "proposed"}` in `ai.action_proposal` |
| `execute_action` | ✅ Hardened | Defaults to `dry_run=true`; rejects `dry_run=false` without approved proposal |
| `execute_proposal` | ✅ Existing | Uses `ApplyService.ExecuteProposal` with `action.WithDryRun(true)` default |

### Wiring Details

**build_context** (`internal/decision/context_builder_recipe.go`):
- Constructor: `NewRecipeContextBuilder(caseSvc, compiler, metricQuery, linkExec, pool, actionTypes, recipes)`
- Loads `config/context_recipes.yml` and `config/metric_definitions.yml`
- Returns `LLMSafeContextEnvelope` with `ContextHash`, `ObjectContext`, `AllowedActions`, `Governance`

**get_linked_objects** (`cmd/baxi-mcp/main.go`):
- `ontologyServiceAdapter.linkResolver` is set when v2 objects are available
- `GetLinkedObjects` tries v2 path first (`linkResolver.CompileAllLinks` + SQL execution), falls back to v1
- v2 path executes compiled SQL via `pool.Query()` and maps rows to `ObjectContext`

**Action Safety** (`internal/mcp/tools_action.go`, `tools_ontology.go`):
- `propose_action` tool registered; handler calls `OntologyService.ProposeAction`
- `execute_action` handler defaults `dry_run=true` and rejects non-dry-run without approval
- Proposal creation inserts into `ai.decision_case` + `ai.action_proposal` with `apply_status='proposed'`

## ANTI-PATTERNS

- **Duplicate method names**: `QueryCompiler` methods spread across `query_plan.go` and `compiler.go` — both files define methods on the same type, risking name collisions
- **Test file growth**: `ontology_v2_integration_test.go` tests multiple config types in one file — split if coverage grows
- **V1/V2 parallel hierarchies**: `ObjectType` (v1) and `ObjectTypeV2` (v2) coexist; some fields overlap but are independently maintained
