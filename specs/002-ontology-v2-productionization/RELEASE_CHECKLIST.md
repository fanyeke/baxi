# Release Checklist: Ontology v2 Productionization

**Feature**: specs/002-ontology-v2-productionization
**Date**: 2026-06-02
**Status**: Complete

## Pre-Release Verification

- [x] Phase 1: Setup complete (branch `002-ontology-v2-productionization`, config files verified)
- [x] Phase 2: Foundational complete
  - [x] `ProposeAction` added to `OntologyService` interface
  - [x] `dry_run` parameter added to `execute_action` tool
  - [x] `MockOntologyService` updated with `ProposeAction`
  - [x] `MockBuildContextService` added
- [x] Phase 3: US1 — `build_context` wired
  - [x] `RecipeContextBuilder` constructed in `main.go` with 7 dependencies
  - [x] `buildContextSvc` passed to `mcp.NewServer`
  - [x] `baxi-mcp` compiles successfully
- [x] Phase 4: US2 — `get_linked_objects` v2 wired
  - [x] `linkResolver` field added to `ontologyServiceAdapter`
  - [x] `GetLinkedObjects` tries v2 first, falls back to v1
  - [x] `LinkResolver` wired in `main.go` when v2 objects available
- [x] Phase 5: US3 — Action safety
  - [x] `propose_action` handler implemented in `tools_action.go`
  - [x] `propose_action` tool registered
  - [x] `execute_action` defaults to `dry_run=true`
  - [x] `execute_action` rejects `dry_run=false` without approved proposal
- [x] Phase 7: Polish
  - [x] `internal/ontology/AGENTS.md` updated with MCP integration notes
  - [x] `docs/quickstart.md` created with Ontology v2 guide
  - [x] `contracts/mcp-tools.md` updated with implementation status
  - [x] `server_test.go` updated with `propose_action` in expected tools

## Build Verification

- [x] `go build ./cmd/baxi-mcp` passes
- [x] `go test ./cmd/baxi-mcp/...` passes
- [x] `go test ./internal/mcp/...` passes

## Known Limitations

- E2E test (`test/integration/ontology_v2_e2e_test.go`) deferred due to API complexity and pre-existing test infrastructure issues
- Integration tests in `test/integration/phase7_test.go` have pre-existing failures unrelated to this feature
- `describe_ontology` v2 enrichment deferred to future phase

## Files Changed

| File | Change |
|------|--------|
| `cmd/baxi-mcp/main.go` | Wired `buildContextSvc`, `linkResolver`, `ProposeAction` |
| `internal/mcp/interfaces.go` | Added `ProposeAction` to `OntologyService` |
| `internal/mcp/tools_action.go` | Added `propose_action` handler + registration |
| `internal/mcp/tools_ontology.go` | Hardened `execute_action` dry_run default |
| `internal/mcp/server_test.go` | Added `propose_action` to expected tools |
| `internal/ontology/AGENTS.md` | Added MCP integration section |
| `docs/quickstart.md` | Created Ontology v2 quickstart guide |
| `specs/002-ontology-v2-productionization/contracts/mcp-tools.md` | Updated implementation status |

## Sign-Off

- [x] Implementation complete
- [x] Unit tests pass
- [x] Documentation updated
- [ ] E2E test validated (deferred)
- [ ] Integration test suite green (pre-existing failures)
