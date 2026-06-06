# Project Research Summary

**Project:** Baxi MCP Server — MCP 信息收束 (Information Containment)
**Domain:** MCP Server security hardening for AI agent access (Go, stdio transport, mcp-go v0.41.1)
**Researched:** 2026-06-06
**Confidence:** HIGH

## Executive Summary

Baxi's MCP Server currently leaks project architecture in three dimensions: **server identity** (name + instructions reveal "Baxi" and "e-commerce governance"), **tool naming** (31 tool names map directly to internal package structure like `describe_ontology`, `run_pipeline`, `get_system_status`), and **output content** (descriptions expose DB schema via `SourceDescriptor`, `get_system_status` returns raw table names with row counts, and `get_object` passes through all properties regardless of `LLMReadable`/`Sensitivity` markers). Pi Agent extensions use REST API for most operations—only `baxi-decision`'s `ctx.sendMessage` hints reference MCP tool names, keeping the blast radius manageable.

**Recommended approach:** A layered, non-invasive containment architecture using 3 new files (`output_filter.go`, `tool_names.go`, `server_identity.go`) and surgical modifications to 14+ existing files. All changes are within `internal/mcp/` — no new dependencies, no mcp-go upgrades, no service layer changes. The core mechanism is handler-level JSON filtering before `NewToolResultJSON()` construction, plus tool name constants replacing hardcoded strings. Backward compatibility with Pi Agent is maintained via dual registration (legacy tool aliases gated by `MCP_ENABLE_LEGACY_TOOLS=true`).

**Key risks and mitigation:** (1) Tool rename breaks 3 test files + 1 Pi extension — fix by doing rename as a single atomic commit and running full test suite. (2) Error messages leak SQL/schema details even after output filtering — create a `sanitizeError()` helper and audit all 31 `NewToolResultError` call sites. (3) Server identity change confuses LLM task planning — use a generic but descriptive identity like "Data Processing Server" and never set instructions to empty.

## Key Findings

### Recommended Stack

All information containment requirements (INT-01 through INT-07) can be implemented **entirely within the existing mcp-go v0.41.1 library** — no new dependencies, no version upgrades. The library provides `ToolHandlerMiddleware` for intercepting tool call outputs and `ToolFilterFunc` for controlling tool listings, both stable since v0.36+.

**Core technologies:**
- `mcp-go v0.41.1` — Existing dependency. `WithToolHandlerMiddleware()` is the primary mechanism for output sanitization (documented in `server/server.go:213-222`)
- `encoding/json` — Standard library for JSON marshal/unmarshal in content filtering. Sufficient for all field-level filtering needs
- `github.com/mark3labs/mcp-go/mcp` — `CallToolResult`, `NewToolResult*`, `TextContent` types for handling tool responses
- Go 1.23 — Standard library (no changes needed)

**Key finding:** `ToolFilterFunc` only affects `tools/list` responses — it does NOT prevent a client from calling a tool by its original name. All filtering must be enforced at the handler/middleware level, not just in listings.

**Critical constraint:** mcp-go v0.41.1 has **no post-processing middleware pipeline for tool call results**. The `ToolHandlerMiddleware` wraps the handler, but the only way to filter content is to modify the `map[string]interface{}` before passing to `NewToolResultJSON()`. This means middleware-based JSON rewriting (as initially proposed in STACK.md) is theoretically possible but ARCHITECTURE.md recommends the simpler approach of per-handler filter function calls.

### Expected Features

Features from FEATURES.md prioritized into 3 tiers for this milestone.

**Must have (MVP — P1, this milestone):**
- **Generic Server Identity** (LOW, ~15 min) — Rename server + blur instructions to eliminate project identification via `server.info`
- **Tool Name Abstraction** (MEDIUM, ~2 hr) — Rename all 31 tools to business-capability names, removing architecture fingerprinting from tool listings
- **Output Trimming for describe_ontology/get_system_status** (LOW, ~1 hr) — Strip `SourceDescriptor` (schema/table/PK), remove `table_counts`, aggregate errors
- **Field-Level Filtering for get_object/get_linked_objects** (MEDIUM, ~3 hr) — Filter properties using existing `LLMReadable` markers already in ontology types but ignored by handlers
- **Input Hardening for run_pipeline/search_objects** (LOW, ~2 hr) — Config allowlist, hardcoded `data_dir`, query length limits, result pagination caps
- **Response Size Bounds** (LOW, ~1 hr) — Per-tool max response size with truncation to prevent context-stuffing attacks
- **Per-Tool Kill-Switch** (LOW, ~1 hr) — Map-based enable/disable for each tool as emergency safety net

**Defer to v1.x (P2 — add after validation):**
- Content Sanitization Pipeline — Only if prompt injection via tool output is observed
- Tool-Level Sensitive-Action Confirmation — Only if read-write chaining exfiltration is observed
- Prompt Injection Detection — Only if injection attempts appear in logs

**Defer to v2+ (P3):**
- Turn-Based Context Leak Prevention — Requires session state tracking, not practical without major state management additions
- Response Body Audit Store — Requires separate storage infrastructure, too heavy for single-tenant demo
- Full RBAC — Architecture overkill for single-agent, single-tenant deployment

### Architecture Approach

The architecture is a minimal, non-invasive layering: 3 new files in `internal/mcp/` plus surgical modifications to handlers. All changes stay within the MCP package — no service layer, no `cmd/baxi-mcp/main.go`, no `internal/ontology` changes.

**Major components:**

1. **`output_filter.go`** (NEW) — Centralized filter functions: `FilterProperties()` strips non-LLMReadable fields, `FilterOntologyDescriptor()` removes Source/Governance fields, `FilterSystemStatus()` removes table_counts, `FilterSearchResult()` caps limits and filters items. All functions operate on `map[string]interface{}` before JSON serialization. Debug-level logging for filtered properties.

2. **`tool_names.go`** (NEW) — Tool name constants (e.g., `ToolDescribeSchema = "describe_schema"`) replacing hardcoded strings across all registration functions. Includes `LegacyToolMap` for old→new name mapping. Dual registration support: if `MCP_ENABLE_LEGACY_TOOLS=true`, both old and new names point to the same handler.

3. **`server_identity.go`** (NEW) — Env-var-driven server identity helpers: `getServerName()`, `getServerInstructions()`, `getServerVersion()` with configurable defaults. Env vars: `MCP_SERVER_NAME`, `MCP_SERVER_VERSION`, `MCP_SERVER_INSTRUCTIONS`.

4. **Modified `server.go`** — Use identity helpers in `NewMCPServer()` constructor. Add `enableLegacyTools` field and `registerLegacyAlias()` helper.

5. **Modified `tools_*.go` (all 12)** — Replace literal tool names with constants from `tool_names.go`. Add legacy alias registration. Add output filter function calls before `NewToolResultJSON()`.

**Key decisions:**
- **Handler-level filtering (not middleware):** mcp-go v0.41.1 has no post-processing pipeline. Filtering must happen at the handler before JSON construction. This is explicit, testable, and requires no framework changes.
- **LLMReadable-only (not Sensitivity):** MVP uses only `LLMReadable` flag. `Sensitivity` (L0-L3) filtering is deferred — not needed for information containment, only for multi-role isolation.
- **Dual registration for backward compatibility:** Pi Agent continues to work via `MCP_ENABLE_LEGACY_TOOLS=true`. Transition: Phase 1 (both names), Phase 2 (verify new names), Phase 3 (disable legacy, clean up).

### Critical Pitfalls

1. **Tool rename breaks tests + Pi extensions** — 3 test files (`server_test.go`, `test/e2e/pi_integration_test.go`, `test/e2e/decision_lifecycle_test.go`) have hardcoded tool names, and `pi-extension/baxi-decision/index.ts` sends stale tool name hints via `ctx.sendMessage`. **Prevention:** Do rename as a single atomic commit. Update all test files and Pi extension simultaneously. Run full test suite before considering done. Create a rename map for audit trail.

2. **Error messages leak SQL/schema/architecture details** — Go's `fmt.Errorf("...: %w", err)` preserves underlying SQL text showing table names (`ops.outbox_event`, `audit.pipeline_run`), column names, and join structures. Even after output filtering, error paths expose everything. **Prevention:** Create `sanitizeError()` helper that redacts `schema.table` patterns and SQL keywords. Use generic error wrappers. Audit all `mcp.NewToolResultError(fmt.Sprintf("...%v", err))` call sites. Log real errors server-side only.

3. **get_object/get_linked_objects bypass LLMReadable filtering** — `DescribeOntology` correctly filters properties by `LLMReadable` flag, but `GetObject` and `GetLinkedObjects` return ALL properties unchanged. The ontology metadata exists but handlers ignore it. **Prevention:** Apply `FilterProperties()` to handler results before JSON construction. Apply same filter to linked objects. Add unit and E2E tests verifying non-readable properties are absent from responses.

4. **Server identity change confuses LLM task planning** — Changing "Baxi MCP Server" / "E-commerce governance and decision platform" to something too generic causes the Pi Agent's LLM to misinterpret capabilities and skip tool calls. **Prevention:** Use "Data Processing Server" — generic but functionally descriptive. Keep instructions focused on what the server does, not what product it is. Never set instructions to empty. Test with Pi Agent after changes.

5. **Pi Agent extension sendMessage hints become stale** — `baxi-decision/index.ts` sends tool name hints as natural language text. If hints reference old tool names, the LLM gets "tool not found" errors. **Prevention:** Update Pi extension in the same commit as tool rename. Change hints to describe intent instead of tool names (e.g., "Creating a decision case..." instead of "call create_decision_case"). The LLM can self-correct via `list_tools` but this adds latency.

## Implications for Roadmap

Based on dependency analysis, the optimal execution order is 3 waves with 7 concrete phases:

### Phase 1: Foundation — Identity, Names, Kill-Switch (INT-01, INT-02, part of testing)
**Rationale:** These three changes are fully independent with no shared code dependencies. They touch different files and can be done in parallel. Server identity is a 2-line change; tool name constants must be in place before any handler can use them; kill-switch is a standalone safety feature.
**Delivers:** Generic server identity (no more "Baxi MCP Server" leak), 31 renamed tools, per-tool kill-switch config, updated test expectations.
**Addresses:** Generic Server Identity (P1), Tool Name Abstraction (P1), Per-Tool Kill-Switch (P1)
**Avoids:** Pitfall 6 (LLM confusion — instructions remain functional), Pitfall 1 (atomic rename with test updates), Pitfall 7 (Pi extension updated same commit), Pitfall 11 (handler function names unchanged)
**Research flag:** Standard patterns — well-documented, purely mechanical changes. No deeper research needed.

### Phase 2: Output Sanitization (INT-03, INT-04, INT-05)
**Rationale:** Builds on Phase 1's name constants. This phase creates `output_filter.go` with all filter functions and integrates them into the 3 handlers that return sensitive structural data. Must be done before input hardening because the filter functions are reused.
**Delivers:** `output_filter.go` with `FilterProperties()`, `FilterOntologyDescriptor()`, `FilterSystemStatus()`, `FilterSearchResult()`. Modified `tools_ontology.go`, `tools_status.go` to call filter functions.
**Addresses:** Output Trimming (P1), Field-Level Filtering (P1)
**Avoids:** Pitfall 3 (unfiltered properties — FilterProperties applied to all 3 handlers), Pitfall 4 (table_counts removed from status), Pitfall 2 (error sanitization helper created)
**Research flag:** Needs moderate research on exact `LLMReadable` field coverage across all ontology types. May need to trace which properties are marked non-readable.

### Phase 3: Input Hardening (INT-06, INT-07)
**Rationale:** Independent of Phase 2's output filtering logic but depends on Phase 1's naming. This phase adds input validation guards to `search_objects` and `run_pipeline` handlers.
**Delivers:** Config allowlist for pipeline run types, hardcoded `data_dir`, query length limits, result pagination caps.
**Addresses:** Input Hardening (P1)
**Avoids:** Pitfall 5 (free-form config → allowlist), Pitfall 8 (incomplete E2E coverage — extend test suite)

### Phase 4: Per-Tool Response Size Bounds
**Rationale:** Lightweight addition that applies uniform size limits across all handlers. Can inform the design of the `safeResult` helper from ARCHITECTURE.md.
**Delivers:** Per-tool max response size enforcement with truncation. Helper function for consistent size checking.
**Addresses:** Response Size Bounds (P1)

### Phase 5: E2E Test Expansion + Error Message Audit
**Rationale:** Cross-cutting validation phase. The E2E test suite currently only covers ~58% of tools. Error message leaks (Pitfall 2) need a systematic audit. This phase fills both gaps.
**Delivers:** Extended `TestPiIntegration_Basic` to cover all 31+ tools. `sanitizeError()` helper. Audit log of all `fmt.Errorf` + `NewToolResultError` call sites.
**Avoids:** Pitfall 8 (incomplete E2E coverage), Pitfall 2 (SQL/schema leak in errors)

### Phase 6: Pi Extension Migration (Post-Deployment Validation)
**Rationale:** Not a code change phase — this is the validation and transition period. Verify Pi Agent works with new tool names. Update `baxi-decision` to use intent-based hints.
**Delivers:** Verified Pi Agent compatibility with new names. `MCP_ENABLE_LEGACY_TOOLS` can be flipped to `false`.
**Avoids:** Pitfall 7 (stale Pi hints), Pitfall 6 (LLM confusion)

### Phase 7: Cleanup — Legacy Alias Removal
**Rationale:** Final cleanup in next milestone after verifying Pi Agent works without legacy names. Remove dual registration code.
**Delivers:** Clean tool registration without legacy aliases. Remove `MCP_ENABLE_LEGACY_TOOLS` config.

### Phase Ordering Rationale

- **Wave 1 (Phases 1-2) can be parallelized** — Identity change and tool name abstraction have zero shared dependencies. Kill-switch is fully independent.
- **Output filtering depends on naming** — Filter functions reference tool name constants, so Phase 1 must precede Phase 2.
- **Input hardening is independent of output filtering** — Phase 3 can run alongside Phase 2.
- **Response size bounds are additive** — Phase 4 has no dependencies on previous phases' logic, just needs tool name constants.
- **Testing and error sanitation are cross-cutting** — Phase 5 should run after all feature phases to catch any regressions.
- **Pi migration and cleanup are post-deployment** — Phases 6-7 happen after the milestone is shipped and verified.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 2 (Output Sanitization):** Need to audit all 31 handlers' output shapes to ensure `LLMReadable` coverage is complete. Some properties may not have the flag set. Trace through `interfaces.go` ontology types.
- **Phase 5 (Error Message Audit):** Need to trace every `fmt.Errorf` and `NewToolResultError` call site across `internal/mcp/` and `cmd/baxi-mcp/main.go` to identify all SQL/schema leak points. This is systematic but need to catalog ~15-20 locations.

Phases with standard patterns (skip research-phase):
- **Phase 1 (Identity + Names + Kill-Switch):** Purely mechanical string replacement and constant extraction. Well-documented in STACK.md and ARCHITECTURE.md.
- **Phase 3 (Input Hardening):** Standard input validation patterns. Allowlist for config, length caps, limit enforcement. No domain unknowns.
- **Phase 4 (Response Size Bounds):** Simple truncation logic. No architectural complexity.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | mcp-go v0.41.1 source code verified directly in `/home/zzz/go/pkg/mod/`. All middleware types, constructor signatures, and tool handler interfaces confirmed. No speculation. |
| Features | HIGH | MSSS v0.1 standard (24 controls, 8 domains) cross-referenced with GitHub Agentic Workflows safe-outputs spec. 7 P1 items map to MSSS L1 controls. Community patterns (mcp-sanitizer, mcpgw) reinforce the same containment approach. |
| Architecture | HIGH | Codebase walk-through confirmed every claim: `HandleGetObject` on `tools_ontology.go:86-93` bypasses `LLMReadable`; `cmd/baxi-mcp/main.go:652-726` already filters in `DescribeOntology`; `server.go:77-82` leaks project identity. All 15 file changes verified against actual source files. |
| Pitfalls | HIGH | All 15 pitfalls derived from codebase inspection (not speculation). Error path analysis traced SQL leaks to adapter-level error wrapping. Pi extension test coverage gaps confirmed by reading test files. Recovery strategies validated against actual file structure. |

**Overall confidence:** HIGH — All four research dimensions confirmed by direct codebase inspection, official library source, and cross-referenced MCP security standards. No significant unknowns remain.

### Gaps to Address

- **LLMReadable flag coverage completeness:** The ontology types have `LLMReadable` markers, but not all properties may be consistently flagged. During Phase 2 implementation, audit all object types to ensure sensitive properties are marked non-readable. If gaps are found, fix in `internal/ontology/schema_v2.go`.
- **Pi Agent test environment availability:** The E2E tests (`test/e2e/`) require a running Postgres instance. Make sure CI or local dev has Postgres available when running Phase 5's expanded E2E test suite.
- **Exact sanitization of error text from adapters:** The adapter layer in `cmd/baxi-mcp/main.go` wraps errors with `fmt.Errorf("...: %w", err)`. Phase 5 must trace the full error path from DB query through adapter to handler to identify all leak points.

## Sources

### Primary (HIGH confidence)
- **mcp-go v0.41.1 source code** — Verified directly: `server/server.go:40-44` (ToolHandlerFunc/Middleware types), `server/server.go:213-222` (WithToolHandlerMiddleware), `server/server.go:334-363` (NewMCPServer constructor), `server/server.go:1099-1106` (ToolFilterFunc), `server/server.go:1132-1190` (middleware chain), `mcp/tools.go:557-574` (Tool struct), `mcp/utils.go:271-298` (NewToolResultJSON/Text)
- **Baxi codebase** — Verified: `internal/mcp/server.go:77-82` (identity leak), `internal/mcp/interfaces.go:105-109` (SourceDescriptor), `internal/ontology/schema_v2.go:41-54` (ObjectPropertyV2 with LLMReadable/Sensitivity), `internal/mcp/tools_ontology.go:86-93` (unfiltered properties), `cmd/baxi-mcp/main.go:652-726` (existing DescribeOntology filtering)
- **MCP Official Security Best Practices** — https://modelcontextprotocol.io/docs/tutorials/security/security_best_practices — Core containment patterns
- **GitHub Agentic Workflows Safe Outputs Specification** — https://github.github.com/gh-aw/specs/safe-outputs-specification/ — Production MCP security architecture

### Secondary (MEDIUM confidence)
- **MSSS v0.1 (MCP Server Security Standard)** — https://github.com/mcp-security-standard/mcp-server-security-standard — 24 controls across 8 domains, L1-L4 compliance levels
- **mcpgw (Go MCP Firewall)** — https://github.com/knorq-ai/mcpgw — Go production patterns for tool-level policy enforcement
- **mcp-sanitizer (npm library)** — https://github.com/starman69/mcp-sanitizer — Community patterns for MCP output sanitization (JS ecosystem, but patterns are language-agnostic)
- **Pi Agent SDK** — Extension model confirmed: `ctx.sendMessage` for MCP hints, REST API for operations. baxi-decision is the only extension affected by MCP changes.

---

*Research completed: 2026-06-06*
*Ready for roadmap: yes*
