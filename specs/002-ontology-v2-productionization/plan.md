# Implementation Plan: Ontology v2 Productionization & E2E

**Branch**: `002-ontology-v2-productionization` | **Date**: 2026-06-02 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/002-ontology-v2-productionization/spec.md`

## Summary

Wire the Ontology v2 executable semantic layer into the MCP server's active runtime path. The v2 core code (schema, parser, compiler, registry, LinkResolver, RecipeContextBuilder) is already implemented and tested, but the MCP server boots with `buildContextSvc=nil`, `linkResolver` unset, and `execute_action` bypasses the approval workflow. This plan covers: (1) wiring `RecipeContextBuilder` into MCP startup, (2) connecting v2 `LinkResolver` to `get_linked_objects`, (3) adding `propose_action` and hardening `execute_action` safety, and (4) end-to-end validation via MCP stdio tests for the `seller_late_delivery_alert` scenario.

## Technical Context

**Language/Version**: Go 1.23  
**Primary Dependencies**: chi (HTTP router), pgx (PostgreSQL driver), mark3labs/mcp-go (MCP protocol), goose (migrations), testify (tests)  
**Storage**: PostgreSQL 16 (via Docker Compose)  
**Testing**: Go `testing` + testcontainers for PostgreSQL; build constraint `//go:build integration` for E2E tests  
**Target Platform**: Linux server (Docker), stdio MCP transport for Pi Agent  
**Project Type**: Web service + CLI + MCP server  
**Performance Goals**: `build_context` response under 2 seconds; `get_linked_objects` under 1 second for typical one-to-many (≤50 records)  
**Constraints**: v1/v2 coexistence required; no breaking changes to existing v1 object queries; must not auto-execute actions without explicit approval  
**Scale/Scope**: 4 v2 objects (`seller`, `order`, `product`, `customer`) for this phase; 5+ additional objects in future phases

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

The project's `.specify/memory/constitution.md` is a template and has not been ratified with project-specific principles. Based on the codebase conventions observed during research:

| Principle | Status | Notes |
|-----------|--------|-------|
| Test-First | ⚠️ Partial | Integration tests exist (`test/integration/`) but MCP E2E coverage is missing. This plan adds it. |
| Integration Testing | ✅ Pass | `testcontainers` pattern already established; this plan extends it to MCP stdio. |
| Observability | ✅ Pass | Structured logging with zap already in use. |
| Simplicity/YAGNI | ✅ Pass | No new external dependencies required; all wiring uses existing code. |

**Post-design re-check**: All changes are wiring and safety fixes — no new frameworks, no new storage engines, no new deployment artifacts. Complexity is justified by the existing untested v2 code paths.

## Project Structure

### Documentation (this feature)

```text
specs/002-ontology-v2-productionization/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0: technical research and dependency resolution
├── data-model.md        # Phase 1: entities, state machines, validation rules
├── quickstart.md        # Phase 1: E2E walkthrough and troubleshooting
├── contracts/
│   └── mcp-tools.md     # Phase 1: MCP tool input/output contracts
└── tasks.md             # Phase 2: actionable tasks (/speckit-tasks command)
```

### Source Code (repository root)

```text
cmd/baxi-mcp/main.go              # Wiring changes (buildContextSvc, LinkResolver, action safety)
internal/mcp/server.go            # Server struct (already has fields, no changes needed)
internal/mcp/interfaces.go        # Add ProposeAction to OntologyService interface
internal/mcp/tools_context.go     # build_context handler (already exists)
internal/mcp/tools_ontology.go    # get_linked_objects + describe_ontology handlers
internal/mcp/tools_action.go      # Add propose_action; modify execute_action
internal/decision/context_builder_recipe.go  # RecipeContextBuilder (no changes)
internal/ontology/link_plan.go    # LinkResolver (no changes)
internal/action/apply_service.go  # Tighten approval enforcement
internal/repository/decision/repository.go  # CreateProposal (no changes)
test/integration/                 # Add MCP E2E test
config/
├── context_recipes.yml           # Already exists
├── metric_definitions.yml        # Already exists
└── aip_object_schema_v2.yml      # Already exists
```

**Structure Decision**: Single Go project with modular internal packages. Changes are concentrated in `cmd/baxi-mcp/main.go` (wiring) and `internal/mcp/*.go` (tool handlers). No new packages or subprojects.

## Complexity Tracking

> No constitution violations requiring justification. All changes are minimal wiring and safety fixes within existing package boundaries.

## Phase 0: Research

Completed. All technical unknowns resolved:

1. **RecipeContextBuilder constructor** — 7 dependencies, 3 need YAML loading (`recipes`, `metricQuery`, `linkExec`). See `research.md` for full dependency graph.
2. **LinkResolver wiring gap** — `SetLinkResolver` exists but is never called; `handleGetLinkedObjects` uses v1 adapter. See `research.md` for architecture comparison.
3. **Action execution safety** — Current flow auto-creates `approved` proposals with `dryRun=false`. Proposed flow creates `proposed` proposals with `dryRun=true` default. See `research.md` for status enum and lifecycle.
4. **E2E test strategy** — Extend existing `test/integration/` pattern with MCP stdio transport. See `research.md`.

**Output**: [research.md](research.md) — all NEEDS CLARIFICATION resolved, no blockers.

## Phase 1: Design

### Data Model

**Output**: [data-model.md](data-model.md)

Key entities: `ContextRecipe`, `ActionProposal`, `ObjectTypeV2`, `ObjectLinkV2`, `LLMSafeContextEnvelope`.

State machines:
- **ActionProposal**: `proposed` → `approved` → `applying` → `applied`/`failed` (or `rejected` from any state)
- **build_context availability**: startup wiring → non-nil service (or nil fallback with error)

### Interface Contracts

**Output**: [contracts/mcp-tools.md](contracts/mcp-tools.md)

Covers 5 tools:
1. `build_context` — already registered, needs service wiring
2. `get_linked_objects` — v2 resolver integration with v1 fallback
3. `propose_action` — **NEW** tool
4. `execute_action` — modified default behavior (dry-run=true, proposed status)
5. `execute_proposal` — unchanged interface, tightened enforcement

### Quickstart

**Output**: [quickstart.md](quickstart.md)

8-step E2E walkthrough for `seller_late_delivery_alert`:
```
describe_ontology → get_object → get_linked_objects → build_context
  → propose_action → approve_proposal → execute_proposal(dry_run=true)
  → verify unapproved execution rejected
```

### Agent Context Update

The plan reference in `CLAUDE.md` between `<!-- SPECKIT START -->` and `<!-- SPECKIT END -->` markers should point to:
`specs/002-ontology-v2-productionization/plan.md`

## Implementation Approach

### Phase 1: Wiring Fixes

**T001-T004**: In `cmd/baxi-mcp/main.go`:
1. After `v2Builder` construction (~line 95), load `context_recipes.yml` and `metric_definitions.yml`
2. Construct `MetricQueryResolver`, `LinkExecutor`, and `RecipeContextBuilder`
3. Pass `buildContextSvc` (the `RecipeContextBuilder`) to `mcp.NewServer(...)` instead of `nil`

### Phase 2: LinkResolver Integration

**T005-T009**: 
1. After `mcpSrv` creation (~line 191), call `mcpSrv.SetLinkResolver(ontology.NewLinkResolver(v2Objects))`
2. Update `handleGetLinkedObjects` in `tools_ontology.go`:
   - Check `s.linkResolver` first
   - If v2 link exists, compile + execute SQL via pool, return array for one_to_many
   - If not, fall back to `s.ontologySvc.GetLinkedObjects` (v1)

### Phase 3: Action Safety

**T010-T015**:
1. Add `propose_action` handler in `tools_action.go` — validate binding, create `ActionProposalRow{ApplyStatus: "proposed"}`, return proposal_id
2. Fix `ontologyServiceAdapter.ExecuteAction` in `main.go`:
   - Change `WithDryRun(false)` → `WithDryRun(true)`
   - Change proposal status `'approved'` → `'proposed'`
   - Return proposal_id instead of auto-executing
3. Add `dry_run` parameter to `execute_action` tool definition in `tools_ontology.go`
4. Remove or gate risk-adaptive auto-execution in `apply_service.go:106-116`

### Phase 4: E2E Tests

**T016-T020**:
1. Create `test/integration/ontology_v2_e2e_test.go` with `//go:build integration`
2. Use `testutil.StartPostgres()` for DB, run MCP server in-process
3. Execute the 8-step sequence and assert each step
4. Update `quickstart.md` with verified commands

### Phase 5: Documentation

**T021-T025**:
1. Update `internal/ontology/AGENTS.md` with v2 wiring notes
2. Update README quickstart section
3. Add release checklist

## Success Criteria Mapping

| SC | Verification Method |
|----|---------------------|
| SC-001 | Log assertion: `buildContextSvc != nil` at startup |
| SC-002 | E2E test: `build_context` returns envelope with 6 fields in <2s |
| SC-003 | E2E test: `get_linked_objects(seller, id, recent_orders)` returns `[]` with ≥1 item |
| SC-004 | E2E test: `propose_action` response has `status: "proposed"` |
| SC-005 | E2E test: `execute_action` without `dry_run=false` does not modify state |
| SC-006 | E2E test: `execute_proposal` on unapproved proposal returns auth error |
| SC-007 | E2E test: full 8-step sequence passes |
| SC-008 | Manual: new user follows quickstart in <15 minutes |

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| `RecipeContextBuilder` constructor signature changes | High | All deps are in same repo; compilation will catch it |
| v1 fallback breaks for existing objects | Medium | v1 Via model code is untouched; v2 only activates when `linkResolver` is set |
| `execute_action` behavior change breaks Pi Agent | High | `dry_run` default change is breaking; document in release notes; Pi extension uses `execute_proposal` already |
| YAML loading fails at startup | Low | Fail-open: warn in logs, set `buildContextSvc=nil`, tool returns "not available" |
| E2E test flaky with testcontainers | Medium | Use existing `testutil.StartPostgres()` pattern; add retry logic |
