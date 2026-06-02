# Tasks: Ontology v2 Productionization & E2E

**Feature**: Ontology v2 Productionization & E2E  
**Branch**: `002-ontology-v2-productionization`  
**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)  
**Generated**: 2026-06-02

---

## Phase 1: Setup

**Goal**: Verify current codebase state and confirm no blocking issues.

- [ ] T001 Verify current branch is `002-ontology-v2-productionization` and working tree is clean
- [ ] T002 Run `go test ./...` to establish baseline; document any pre-existing failures
- [ ] T003 Verify `config/context_recipes.yml` and `config/metric_definitions.yml` exist and parse correctly

---

## Phase 2: Foundational

**Goal**: Prepare interfaces and shared structures used by multiple user stories.

- [ ] T004 [P] Add `ProposeAction` method to `OntologyService` interface in `internal/mcp/interfaces.go`
- [ ] T005 [P] Add `dry_run` parameter to `execute_action` tool definition in `internal/mcp/tools_ontology.go`
- [ ] T006 Add `MockBuildContextService` implementation for `server_test.go` in `internal/mcp/server_test.go`

---

## Phase 3: User Story 1 — Recipe-Driven Context Building

**Goal**: Wire `RecipeContextBuilder` so `build_context` returns a complete `LLMSafeContextEnvelope`.

**Independent Test**: Start baxi-mcp, call `build_context(CASE_001, seller_late_delivery_alert)`, verify response contains `context_hash`, `evidence`, `object_context`, `allowed_actions`, `governance`, `redaction_summary`.

- [ ] T007 Load `config/context_recipes.yml` at startup in `cmd/baxi-mcp/main.go` using `ontology.LoadContextRecipes(path)`
- [ ] T008 Load `config/metric_definitions.yml` at startup in `cmd/baxi-mcp/main.go` using `ontology.LoadMetricDefinitions(path)`
- [ ] T009 Construct `MetricQueryResolver` from loaded metric definitions in `cmd/baxi-mcp/main.go`
- [ ] T010 Construct `LinkExecutor` via `common.NewPoolProvider(pool)` in `cmd/baxi-mcp/main.go`
- [ ] T011 Construct `RecipeContextBuilder` with all 7 dependencies in `cmd/baxi-mcp/main.go`
- [ ] T012 Wire `buildContextSvc` into `mcp.NewServer(...)` call (replace `nil` with constructed builder)
- [ ] T013 Add startup log verification: `buildContextSvc` is non-nil when v2 objects present
- [ ] T014 [US1] Test `build_context` returns complete envelope with all 6 fields via manual MCP stdio call

---

## Phase 4: User Story 2 — One-to-Many Object Relationship Queries

**Goal**: Connect v2 `LinkResolver` to `get_linked_objects` so `seller → recent_orders` returns an array.

**Independent Test**: Start baxi-mcp, call `get_linked_objects(seller, SELLER_001, recent_orders)`, verify `objects` array with ≥1 order record.

- [ ] T015 [P] Create `LinkResolver` from `v2Objects` in `cmd/baxi-mcp/main.go` using `ontology.NewLinkResolver(v2Objects)`
- [ ] T016 Call `mcpSrv.SetLinkResolver(linkResolver)` after server creation in `cmd/baxi-mcp/main.go`
- [ ] T017 Update `handleGetLinkedObjects` in `internal/mcp/tools_ontology.go` to check `s.linkResolver` first
- [ ] T018 Implement v2 link query execution: compile SQL plan via `LinkResolver`, execute via `pool.Query()`, handle `one_to_many` cardinality
- [ ] T019 Add v1 Via-model fallback in `handleGetLinkedObjects` when v2 link is not configured
- [ ] T020 Add startup log verification: `LinkResolver` wired with N objects
- [ ] T021 [US2] Test `get_linked_objects(seller, SELLER_001, recent_orders)` returns array via manual MCP stdio call
- [ ] T022 [US2] Test v1 fallback works for object types without v2 links

---

## Phase 5: User Story 3 — Safe Action Execution with Approval Workflow

**Goal**: Add `propose_action` tool and harden `execute_action` so no action bypasses approval.

**Independent Test**: Call `propose_action` → verify `status: proposed`; call `execute_action` without approval → verify rejection; call `execute_proposal` on approved proposal with `dry_run=true` → verify simulated execution.

- [ ] T023 Implement `propose_action` handler in `internal/mcp/tools_action.go`
- [ ] T024 Add `propose_action` tool registration in `internal/mcp/tools_action.go`
- [ ] T025 Implement validation in `propose_action`: check action binding via `ActionBindingValidator`, validate payload schema
- [ ] T026 Implement proposal creation in `propose_action`: build `ActionProposalRow{ApplyStatus: "proposed"}`, call `repo.CreateProposal`, return `proposal_id`
- [ ] T027 Modify `ontologyServiceAdapter.ExecuteAction` in `cmd/baxi-mcp/main.go`: change default `WithDryRun(false)` to `WithDryRun(true)`
- [ ] T028 Modify `ontologyServiceAdapter.ExecuteAction`: change proposal `apply_status` from `"approved"` to `"proposed"`
- [ ] T029 Modify `ontologyServiceAdapter.ExecuteAction`: remove auto-execution after proposal creation; return proposal_id only
- [ ] T030 Tighten `ExecuteProposal` in `internal/action/apply_service.go`: remove or gate risk-adaptive auto-execution of `proposed` proposals
- [ ] T031 Ensure `execute_proposal` rejects unapproved proposals with clear authorization error
- [ ] T032 [US3] Test `propose_action` creates proposal with `status: proposed`
- [ ] T033 [US3] Test `execute_action` defaults to dry-run and does not modify state
- [ ] T034 [US3] Test `execute_proposal` on unapproved proposal returns error

---

## Phase 6: User Story 4 — End-to-End MCP Validation

**Goal**: Validate the complete seller_late_delivery_alert workflow via automated E2E test.

**Independent Test**: Run `go test -tags=integration ./test/integration/... -run TestOntologyV2E2E` and verify all 8 steps pass.

- [ ] T035 Create `test/integration/ontology_v2_e2e_test.go` with `//go:build integration`
- [ ] T036 Add `testutil.StartPostgres()` setup and MCP server in-process bootstrap
- [ ] T037 Add fixture data: insert seller, orders, decision case into test DB
- [ ] T038 Implement E2E step 1: `describe_ontology` returns v2 object types
- [ ] T039 Implement E2E step 2: `get_object(seller, SELLER_001)` returns valid object
- [ ] T040 Implement E2E step 3: `get_linked_objects(seller, SELLER_001, recent_orders)` returns array
- [ ] T041 Implement E2E step 4: `build_context(CASE_001, seller_late_delivery_alert)` returns complete envelope
- [ ] T042 Implement E2E step 5: `propose_action` returns `proposal_id` with `status: proposed`
- [ ] T043 Implement E2E step 6: `approve_proposal` transitions status to `approved`
- [ ] T044 Implement E2E step 7: `execute_proposal(dry_run=true)` returns simulated result
- [ ] T045 Implement E2E step 8: `execute_action` without approval is rejected
- [ ] T046 [US4] Run E2E test and verify all assertions pass
- [ ] T047 [US4] Update `quickstart.md` with verified commands and actual output samples

---

## Phase 7: Polish & Cross-Cutting Concerns

**Goal**: Documentation, release readiness, and code quality.

- [ ] T048 [P] Update `internal/ontology/AGENTS.md` with v2 wiring notes and productionization status
- [ ] T049 [P] Update `docs/quickstart.md` (project root) with Ontology v2 E2E section
- [ ] T050 [P] Update `specs/002-ontology-v2-productionization/contracts/mcp-tools.md` if any implementation deviations
- [ ] T051 Add release checklist to `specs/002-ontology-v2-productionization/RELEASE_CHECKLIST.md`
- [ ] T052 Run full test suite: `go test ./...` and `go test -tags=integration ./test/...`
- [ ] T053 Run linter: `golangci-lint run` (or `make lint`)
- [ ] T054 Verify all SC-001 through SC-008 are met; update spec.md status to "Complete"

---

## Dependency Graph

```
Phase 1 (Setup)
    │
    ↓
Phase 2 (Foundational)
    │
    ├──→ Phase 3 (US1: build_context) ──┐
    │                                      │
    ├──→ Phase 4 (US2: get_linked_objects)─┤→ Phase 6 (US4: E2E)
    │                                      │
    └──→ Phase 5 (US3: action safety) ────┘
                                              │
                                              ↓
                                        Phase 7 (Polish)
```

**Notes**:
- US1, US2, and US3 can be developed in parallel after Phase 2
- US4 (E2E) depends on all three P1 stories being functionally complete
- Phase 7 is parallel-friendly (documentation tasks are independent)

## Parallel Execution Opportunities

| Group | Tasks | Why Parallel |
|-------|-------|-------------|
| A | T007–T012 (US1 wiring) | All in `cmd/baxi-mcp/main.go`; sequential within group |
| B | T015–T020 (US2 wiring) | Independent of Group A; same file but different sections |
| C | T023–T031 (US3 safety) | Independent of Groups A and B; touches different files |
| D | T048–T050 (Docs) | Pure documentation; can run anytime after implementation |

## Implementation Strategy

**MVP Scope**: Complete US1 (build_context wiring) first. This is the highest-value, highest-risk item because it requires loading YAML configs and constructing the most complex dependency graph. Once US1 works, US2 and US3 can proceed in parallel.

**Incremental Delivery**:
1. **Sprint 1**: Phase 1–3 (Setup + Foundational + US1) — deliver working `build_context`
2. **Sprint 2**: Phase 4–5 (US2 + US3) — deliver `get_linked_objects` v2 + action safety
3. **Sprint 3**: Phase 6–7 (US4 E2E + Polish) — deliver automated E2E + documentation

## Task Summary

| Phase | Story | Task Count | Key Deliverable |
|-------|-------|-----------|-----------------|
| 1 | — | 3 | Baseline verified |
| 2 | — | 3 | Interfaces prepared |
| 3 | US1 | 8 | `build_context` returns full envelope |
| 4 | US2 | 8 | `get_linked_objects` uses v2 LinkResolver |
| 5 | US3 | 12 | `propose_action` exists; `execute_action` is safe |
| 6 | US4 | 13 | E2E test passes for seller_late_delivery_alert |
| 7 | — | 7 | Documentation + release ready |
| **Total** | | **54** | |

## Success Criteria Verification Map

| SC | Verifying Task(s) |
|----|-------------------|
| SC-001 | T013 (log assertion) |
| SC-002 | T014, T041 (E2E envelope validation) |
| SC-003 | T021, T040 (E2E linked objects array) |
| SC-004 | T032 (propose_action status) |
| SC-005 | T033 (execute_action dry-run default) |
| SC-006 | T034, T045 (unapproved rejection) |
| SC-007 | T046 (full E2E pass) |
| SC-008 | T047 (quickstart verified) |
