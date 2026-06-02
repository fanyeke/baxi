# Research: Ontology v2 Productionization & E2E

**Date**: 2026-06-02
**Feature**: specs/002-ontology-v2-productionization

## Phase 0 Research Summary

All critical technical unknowns resolved through codebase exploration. The Ontology v2 core code exists and is tested, but MCP wiring is incomplete.

---

### Finding 1: RecipeContextBuilder Dependency Graph

**Decision**: Wire `RecipeContextBuilder` in `cmd/baxi-mcp/main.go` by loading `context_recipes.yml` and `metric_definitions.yml` at startup.

**Rationale**: The constructor requires 7 dependencies, 4 of which are already available in `main.go` (`decisionRepo`, `QueryCompiler`, `pool`, `actionTypes`). The 3 missing ones (`recipes`, `metricQuery`, `linkExec`) can be constructed from YAML files and the pool.

**Constructor** (`internal/decision/context_builder_recipe.go:27`):
```go
func NewRecipeContextBuilder(
    caseSvc     DecisionCaseDataProvider,
    compiler    *ontology.QueryCompiler,
    metricQuery *ontology.MetricQueryResolver,
    linkExec    *ontRepo.LinkExecutor,
    pool        *pgxpool.Pool,
    actionTypes ActionTypeProvider,
    recipes     map[string]*ontology.ContextRecipe,
) *RecipeContextBuilder
```

**Dependency resolution**:
| # | Dependency | Source in main.go |
|---|------------|-------------------|
| 1 | `caseSvc` | `decisionRepo` (implements `DecisionCaseDataProvider`) |
| 2 | `compiler` | `ontology.NewQueryCompiler(v2Objects, 10000)` (already exists at line 156) |
| 3 | `metricQuery` | `ontology.NewMetricQueryResolver(ontology.NewMetricResolver(metricDefs), pool)` |
| 4 | `linkExec` | `ontologyRepo.NewLinkExecutor(common.NewPoolProvider(pool))` |
| 5 | `pool` | `pool.Pool` (line 55) |
| 6 | `actionTypes` | `action.NewActionTypeProviderAdapter(reg)` (line 75) |
| 7 | `recipes` | `ontology.LoadContextRecipes(filepath.Join(configDir, "context_recipes.yml"))` |

**YAML loading functions** (already implemented):
- `ontology.LoadContextRecipes(path)` → `map[string]*ContextRecipe` (`registry_v2.go:187`)
- `ontology.LoadMetricDefinitions(path)` → `map[string]*MetricDefinition` (`metric_definition.go`)

**Alternatives considered**: 
- Delay loading to first `build_context` call → rejected: startup-time validation is preferred for fail-fast behavior
- Inject recipes via environment variables → rejected: YAML is the established config pattern in this project

---

### Finding 2: LinkResolver Wiring Gap

**Decision**: Call `mcpSrv.SetLinkResolver(ontology.NewLinkResolver(v2Objects))` after server creation, then update `handleGetLinkedObjects` to prefer v2 resolver with v1 fallback.

**Rationale**: `LinkResolver` is fully implemented and unit-tested, but never wired into the MCP server. The current `get_linked_objects` handler uses v1 Via model exclusively.

**Key facts**:
- `LinkResolver` struct at `internal/ontology/link_plan.go:59` — compiles SQL plans for v2 relationships
- `Server.SetLinkResolver` at `internal/mcp/server.go:100` — setter exists but is **never called** in `cmd/baxi-mcp/main.go`
- Current handler at `internal/mcp/tools_ontology.go:78-133` — delegates to `s.ontologySvc.GetLinkedObjects` (v1 adapter)
- `ontologyServiceAdapter.GetLinkedObjects` at `cmd/baxi-mcp/main.go:698` — uses v1 Via string lookup
- v2 schema at `config/aip_object_schema_v2.yml` defines `seller → recent_orders` as `cardinality: one_to_many`

**LinkResolver return type**: `LinkedObjectResult` with `QueryPlan` metadata — it compiles but does NOT execute SQL. The handler needs to execute compiled plans via `pool.Query()`.

**Alternatives considered**:
- Make LinkResolver execute queries directly → rejected: separation of compilation and execution is intentional; LinkResolver should remain stateless
- Keep v1 as default → rejected: spec explicitly requires v2 one-to-many support

---

### Finding 3: Action Execution Safety

**Decision**: 
1. Change `execute_action` to default dry-run and create `proposed` (not `approved`) proposals
2. Create new `propose_action` MCP tool for explicit proposal creation
3. Tighten `ExecuteProposal` to require explicit approval when `requires_approval=true`

**Rationale**: Current `ontologyServiceAdapter.ExecuteAction` at `cmd/baxi-mcp/main.go:935` creates a case with status `'closed'`, proposal with status `'approved'`, and executes with `WithDryRun(false)` — violating AIP safety principles.

**Current dangerous flow**:
```
execute_action → create case('closed') → create proposal('approved') → ExecuteProposal(dryRun=false)
```

**Required safe flow**:
```
propose_action → create proposal('proposed') → [review/approve] → execute_proposal(dryRun=true default)
```

**Proposal status enum** (DB constraint from `migrations/011_review_action_outbox.sql`):
```
'proposed' → 'approved' → 'applying' → 'applied'/'failed'
         → 'rejected'
```

**Risk-adaptive backdoor**: `ApplyService.ExecuteProposal` (`apply_service.go:106-116`) auto-executes `'proposed'` proposals if risk is `'low'` and action doesn't require approval. The spec wants this gated more strictly.

**Alternatives considered**:
- Delete `execute_action` entirely → rejected: backward compatibility and tool discoverability
- Keep auto-approval for low-risk actions → rejected: spec requires explicit approval workflow

---

### Finding 4: E2E Test Strategy

**Decision**: Create a Go-based MCP stdio E2E test script that exercises the full seller_late_delivery_alert workflow through actual MCP tool calls.

**Rationale**: The project already has integration tests using testcontainers (`test/integration/`). Extending this pattern to cover MCP stdio transport is the most reliable approach.

**Existing test infrastructure**:
- `test/integration/phase7_test.go` — full pipeline+governance workflow (~485 lines)
- `testutil.StartPostgres()` — testcontainers for PostgreSQL
- Build constraint `//go:build integration` for CI isolation

**Test sequence** (from spec SC-007):
```
describe_ontology
  → get_object(seller, seller_id)
  → get_linked_objects(seller, seller_id, recent_orders)
  → build_context(case_id)
  → propose_action(seller, seller_id, notify_owner)
  → approve_proposal
  → execute_proposal(dry_run=true)
```

**Alternatives considered**:
- Shell-based smoke test with `mcp-cli` → rejected: no mcp-cli dependency in project; Go test is more maintainable
- Manual quickstart verification only → rejected: insufficient for CI gate

---

## Resolved Unknowns

| Unknown | Resolution | Evidence |
|---------|-----------|----------|
| How to construct RecipeContextBuilder | 7-arg constructor, 3 missing deps from YAML | `context_builder_recipe.go:27` |
| Where LinkResolver is defined | `internal/ontology/link_plan.go:59` | Agent search results |
| Why get_linked_objects doesn't use v2 | Handler delegates to v1 adapter, SetLinkResolver never called | `tools_ontology.go:78`, `server.go:100` |
| How to create pending proposals | `ActionProposalRow{ApplyStatus: "proposed"}` + `repo.CreateProposal` | `proposal_service.go:109` |
| Where execute_action bypasses approval | `main.go:935-1043` auto-creates approved + dryRun=false | Agent search results |
| What status values exist | 6-state enum from migration 011 | `011_review_action_outbox.sql` |
| How to test MCP E2E | Extend existing integration test pattern with stdio transport | `test/integration/phase7_test.go` |

## No Remaining NEEDS CLARIFICATION

All technical questions resolved. The implementation plan can proceed without blocking research.
