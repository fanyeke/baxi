# Domain Pitfalls: MCP 信息收束 (Information Containment)

**Domain:** MCP Server information hiding for an existing Go MCP server with Pi Agent integration
**Researched:** 2026-06-06
**Overall confidence:** HIGH

## Summary of Findings

The MCP information containment feature has 4 categories of risk: (1) tool rename breaks hardcoded references in 3 test files + 1 Pi Agent extension, (2) output filtering is incomplete at the handler layer—error messages leak SQL and schema details, (3) server identity changes confuse the LLM's task planning, and (4) over-engineering the naming/grouping scheme wastes time without proportional security benefit. Recovery is straightforward: the Pi Agent extensions use REST API (not MCP) for most operations, so only the decision extension's `sendMessage` hints need updating. The mcp-go library itself handles rename transparently—it's just a string change.

**Key insight:** The Pi Agent extensions (`baxi-operations`, `baxi-sandbox`, `baxi-logger`) call the REST API directly via `fetch()`, not through MCP tools. Only `baxi-decision` uses `ctx.sendMessage` to hint MCP tool names. This means MCP changes impact a much smaller surface area than it first appears.

---

## Critical Pitfalls

Mistakes that cause runtime failures, security leaks, or integration breaks.

### Pitfall 1: Tool Rename Breaks Hardcoded References in Tests and Pi Extensions

**What goes wrong:** Renaming a tool causes `TestServerToolRegistration` (unit test), `TestPiIntegration_Basic` (E2E test), and `TestDecisionLifecycle` (E2E test) to fail. The Pi Agent decision extension sends stale tool name hints to the LLM.

**Why it happens:** Tool names are hardcoded strings in:
- `internal/mcp/server_test.go` line 420-454 — `expectedTools` list (31+ names)
- `test/e2e/pi_integration_test.go` line 153-176 — `expectedTools` list (17 names)
- `test/e2e/pi_integration_test.go` line 228-290 — `Name` field in `CallToolParams` (5 tools)
- `test/e2e/decision_lifecycle_test.go` lines 20, 33, 46, 53, 60, 81, 94, 109, 128, 147, 159, 166, 174, 185, 199, 208 — 16 hardcoded tool name strings
- `pi-extension/baxi-decision/index.ts` lines 15, 31 — `ctx.sendMessage` strings with tool names

**Consequences:**
- Unit test failure: `go test ./internal/mcp/` fails immediately
- E2E test failure: `go test -tags integration ./test/e2e/` fails
- Pi Agent hint confusion: LLM sees old name in instruction text but new name in tool list
- No silent failures: every break is loud (test fails) or self-correcting (LLM re-discovers tools)

**Prevention:**
1. **Do tool rename as a single atomic change** — rename in `mcp.NewTool()`, update all test files, and update Pi extension in the same commit. Use `go test ./...` and verify Pi extension tests pass before considering done.
2. **Create a rename map** — document old→new name mapping in `SPEC.md` or commit message for audit trail.
3. **Run E2E tests after every rename** — `test/e2e/` tests are integration tests with real Postgres. They catch what unit tests miss (the actual tool call path).

**Detection:**
- `go test ./internal/mcp/...` — catches `server_test.go` mismatch
- `go test -tags integration ./test/e2e/...` — catches all tool call reference mismatches
- `grep -r '"old_tool_name"' .` — finds stale string references (only useful if you search for each renamed tool)

**Example rename map (for reference):**
```
create_decision_case → case_create
decide              → case_decide
list_cases          → case_list
get_case            → case_get
resolve_case        → case_resolve
list_proposals      → proposal_list
...
```

---

### Pitfall 2: Error Messages Leak Schema, SQL, and Architecture Details

**What goes wrong:** After filtering a tool's normal output, error messages still reveal internal table names, column names, SQL queries, and schema structure.

**Why it happens:** Go's error wrapping pattern (`fmt.Errorf("...: %w", err)`) preserves the underlying error text, which often contains SQL or database details. When the handler converts this to `mcp.NewToolResultError(fmt.Sprintf("...: %v", err))`, the full error chain is returned to the MCP client.

**Consequences:** An attacker (or curious LLM) can trigger errors to discover:
- Table names: `ops.outbox_event`, `audit.pipeline_run`, `ai.llm_decision` (revealed by `outboxServiceAdapter.ListOutboxEvents` and `pipelineInfoAdapter`)
- Column names: `event_id`, `dispatch_attempts`, `context_hash` (embedded in error context)
- Schema structure: Union queries in `statusServiceAdapter.GetStatus` reveal 9 table names and their join structure
- File paths: `data_dir` default reveals `./data/raw` project layout
- Internal architecture: Adapter names in error messages (`v2 link resolution failed`) reveal internal versioning

**Specific leak points:**

| Handler | File:Line | What Leaks | Severity |
|---------|-----------|------------|----------|
| `statusServiceAdapter.GetStatus` | `cmd/baxi-mcp/main.go:566` | SQL COUNT queries, 9 table names with schema | HIGH |
| `outboxServiceAdapter.ListOutboxEvents` | `cmd/baxi-mcp/main.go:408-409` | SQL query text with table/column names | HIGH |
| `pipelineInfoAdapter.GetLastRunStatus` | `cmd/baxi-mcp/main.go:450-454` | SQL query with `audit.pipeline_run` columns | HIGH |
| `pipelineRunService.Run` | `cmd/baxi-mcp/main.go:382` | Accepts arbitrary `data_dir`, exposes file paths | HIGH |
| `ontologyServiceAdapter.GetObject` | `cmd/baxi-mcp/main.go:735` | Wraps underlying error which may contain SQL | MEDIUM |
| `ontologyServiceAdapter.getLinkedObjectsV2` | `cmd/baxi-mcp/main.go:804` | Executes raw SQL, error may contain query | MEDIUM |
| `handleProposeAction` | `tools_action.go:72-76` | `params` JSON parse error reveals expected format | LOW |
| All `fmt.Errorf("...: %w")` patterns | All `tools_*.go` files | Database errors propagated to MCP response | MEDIUM |

**Prevention:**
1. **Strip SQL from error messages** — Create a helper function that sanitizes error strings before returning:
   ```go
   // sanitizeError removes internal details from error messages
   func sanitizeError(msg string) string {
       // Replace known patterns: schema.table, SQL keywords, file paths
       re := regexp.MustCompile(`[a-z_]+\.([a-z_]+)`)
       return re.ReplaceAllString(msg, "[redacted]")
   }
   ```
2. **Use generic error wrappers** — Replace `fmt.Sprintf("Failed to get system status: %v", err)` with `fmt.Sprintf("Operation failed: try again later")`. Log the real error server-side.
3. **Never return raw database errors** — Wrap all `pool.Query`, `rows.Scan` errors with sanitized messages:
   ```go
   if err != nil {
       zapLog.Error("database error", zap.Error(err))
       return mcp.NewToolResultError("internal operation failed"), nil
   }
   ```
4. **Audit all error returns** — Search for `mcp.NewToolResultError(fmt.Sprintf` and `mcp.NewToolResultError(fmt.Errorf` — these are all potential leak points.

**Detection:**
- Code review: search for `fmt.Sprintf(".*err)` in `mcp.NewToolResultError` calls
- Testing: call each tool with invalid arguments and assert error messages don't contain SQL, table names, or file paths
- `grep -rn 'mcp.NewToolResultError.*%v' internal/mcp/ cmd/baxi-mcp/` — finds all error wrappers

---

### Pitfall 3: get_object / get_linked_objects Return Unfiltered Properties

**What goes wrong:** Even when `describe_ontology` correctly filters properties by `LLMReadable` flag, `get_object` and `get_linked_objects` return ALL properties without field-level filtering.

**Why it happens:** The `handleGetObject` handler (`tools_ontology.go:86-93`) does:
```go
result := map[string]interface{}{
    "object_type": obj.ObjectType,
    "object_id":   obj.ObjectID,
    "properties":  obj.Properties,  // <-- ALL properties, no LLMReadable filter
}
```
The `obj.Properties` comes from `ontologyServiceAdapter.GetObject` → `querySvc.BuildObjectContext`, which returns properties from the database directly. The `LLMReadable` flag is only checked in `DescribeOntology` for the schema listing, NOT in `GetObject` for data retrieval.

Similarly, `get_linked_objects` (line 129-133) returns `result.Links` where each link's objects contain unfiltered properties.

**Consequences:** Sensitive fields (PII, pricing, internal notes) are returned even though `describe_ontology` marks them as `llm_readable: false`. The schema says "don't read this" but the data endpoints ignore the schema.

**Prevention:**
1. **Apply LLMReadable filtering at the handler layer** — Filter `obj.Properties` before returning:
   ```go
   // After getting the object, filter properties
   filteredProps := make(map[string]interface{})
   for key, value := range obj.Properties {
       if s.ontologySvc.IsPropertyReadable(obj.ObjectType, key) {
           filteredProps[key] = value
       }
   }
   result["properties"] = filteredProps
   ```
2. **Add field-level filtering to `GetObject` in the adapter** — The ontology adapter should enforce `LLMReadable` at the query level.
3. **Apply same filter to `get_linked_objects`** — Each linked object's properties must be filtered too.
4. **Add sensitivity-based redaction** — For properties with `sensitivity: "high"`, either omit or replace with a masked value.

**Detection:**
- Unit test: Set up a mock ontology with a non-readable property, call `get_object`, assert property is absent from response.
- E2E test: Load data with known sensitive properties, call `get_object`, assert sensitive values not returned.

---

### Pitfall 4: get_system_status / get_decision_context Over-Expose Internal State

**What goes wrong:** These tools return more information than needed, including architecture-revealing details.

**Why it happens:** `get_system_status` (currently planned for INT-04) returns:
- `table_counts` — 9 table names with schema prefixes and exact row counts. This exposes the full database schema layout.
- `error_message` — Pipeline error messages may contain SQL or file paths.
- `recent_errors` — Unfiltered error strings from `audit.audit_log` (line 617-631).

`get_decision_context` (tools_action.go:322-331) returns:
- `policy` — Governance policy details (line 338-340)
- `rendered_evidence` — Full evidence context
- `governance.classification` — Internal classification levels
- Source type/ID details

**Consequences:**
- `get_system_status` with `table_counts` reveals the entire database schema (3 schemas: `raw`, `dwd`, `metric`, `ops`, `ai`, `audit`) — this is what INT-04 aims to fix.
- `recent_errors` in `get_system_status` directly exposes database error messages to the MCP caller.
- `get_decision_context` reveals internal governance classification levels and policy structure.

**Prevention:**
1. **Remove `table_counts` entirely** — Replace with generic health indicators: `"storage": "operational"` instead of `"table_counts": [{"table_name": "raw.orders", "row_count": 15000}]`.
2. **Remove `recent_errors` from status** — Error details should be server-side only. Replace with `"has_errors": false`.
3. **Truncate/redact error messages in pipeline status** — Replace `error_message` with `"error_occurred": true`.
4. **Filter `get_decision_context` output** — Remove policy internal details, sanitize governance levels.

**Detection:**
- E2E test: Call `get_system_status`, assert no key contains `table_` or `schema`. Assert no SQL keywords appear.
- Code review: Check `get_decision_context` output map for internal-only fields.

---

### Pitfall 5: run_pipeline Accepts Free-form Config and data_dir

**What goes wrong:** The `run_pipeline` tool accepts a free-form `config` string that becomes `RunType`, and a `data_dir` parameter that defaults to an internal project path.

**Why it happens:** The current handler (tools_pipeline.go:23-28) accepts any string as config and any string as data_dir. The adapter (cmd/baxi-mcp/main.go:381-398) allows `config` to override `RunType` entirely and `data_dir` to reference any directory.

**Consequences:**
- Attackers can trigger pipeline runs with arbitrary `RunType` values, potentially causing unexpected behavior.
- `data_dir` can be set to `/etc/passwd` or other sensitive paths; while pipeline steps may not read them, the error messages might reveal file existence.
- Free-form `config` bypasses the governance layer's validation.

**Prevention (INT-07):**
1. **Implement an allowlist for config values** — Only permit known RunType values:
   ```go
   allowedConfigs := map[string]bool{"full": true, "incremental": true}
   if !allowedConfigs[config] {
       return mcp.NewToolResultError("invalid config: must be one of [full, incremental]"), nil
   }
   ```
2. **Hardcode data_dir** — Remove the `data_dir` parameter entirely or restrict it to an allowlist of paths.
3. **Validate input length** — Cap config length to prevent abuse.

**Detection:**
- Negative test: Call `run_pipeline` with an arbitrary config string, assert it returns an error.

---

### Pitfall 6: Server Identity Change Confuses LLM Task Planning

**What goes wrong:** Changing the MCP server's name and instructions causes the LLM (Pi Agent) to misinterpret what the server can do, leading to incorrect tool selection or refusal to use tools.

**Why it happens:** The server identity is set in `internal/mcp/server.go:77-82`:
```go
s := server.NewMCPServer(
    "Baxi MCP Server",
    "1.0.0",
    server.WithToolCapabilities(false),
    server.WithInstructions("E-commerce governance and decision platform"),
)
```
The name "Baxi MCP Server" and instructions "E-commerce governance and decision platform" are sent to every MCP client during initialization. If these are changed to something too generic (e.g., "Data Processing Server"), the Pi Agent's LLM may not recognize the server's capabilities and may not attempt to use its tools.

**Consequences:**
- Pi Agent may decide the server is irrelevant to the user's request and skip MCP tool calls
- LLM may misinterpret tool purposes leading to wrong tool selection
- The Pi Agent extension's session_start handler (`baxi-decision/index.ts:73-76`) notifies "Baxi decision monitor started" — if the server name changes, this still works (it's a Pi extension, not MCP)

**Prevention:**
1. **Use a generic but descriptive name** — Something like "Data Operations Server" that describes the function without exposing the project identity.
2. **Keep instructions functional** — "Analyze business data, enforce governance rules, and execute approved actions" instead of anything product-specific.
3. **Never change instructions to empty** — Empty instructions deprive the LLM of critical context.
4. **Test with Pi Agent** — After changing identity, verify Pi Agent still discovers and uses MCP tools correctly.

**Detection:**
- `TestPiIntegration_Basic` logs the server info (line 88-89) — check that the new name/version don't break the LLM's understanding.
- Manual validation: Connect Pi Agent, verify it still suggests relevant MCP tools.

---

### Pitfall 7: Pi Agent Extension sendMessage Hints Become Stale

**What goes wrong:** After tool rename, the Pi Agent's decision extension sends text hints with old tool names, confusing the LLM.

**Why it happens:** `pi-extension/baxi-decision/index.ts` uses `ctx.sendMessage` to tell the LLM what MCP tools to call:
```typescript
ctx.sendMessage(`Please use the mcp tool to call baxi_mcp with: create_decision_case with alert_id=${params.alert_id}`);
```
This is natural language text sent to the conversation, not an API call. The LLM sees this message and decides whether to follow it.

**Consequences:**
- If the hint says `create_decision_case` but the tool is now `case_create`, the LLM will try to call the old name and get a "tool not found" error from the MCP protocol.
- The LLM can self-correct by calling `list_tools` to discover the new name, but this adds latency and may confuse the user.
- In the worst case, if the LLM doesn't self-correct, the decision workflow silently fails.

**Prevention:**
1. **Update Pi extension alongside MCP tool rename** — Edit `pi-extension/baxi-decision/index.ts` in the same commit.
2. **Use generic instructions instead of tool names** — Change hints to describe intent, not tool names:
   ```typescript
   ctx.sendMessage(`Creating a decision case for alert ${params.alert_id}...`);
   ```
   The LLM will naturally discover the correct tool name from `list_tools`.
3. **Add a migration test** — Verify the Pi extension tests pass after renaming (the decision test checks tool names indirectly via `createMockExtensionAPI`).

**Detection:**
- `pi-extension/baxi-decision/index.test.ts` line 48-60 checks for tool name existence in `registeredTools` — but these are Pi extension tool names (`baxi_create_decision_case`), NOT MCP tool names. The test won't catch MCP rename issues.
- Manual search: `grep -rn 'create_decision_case\|decide\|list_cases\|get_case\|list_proposals\|resolve_case' pi-extension/`

---

### Pitfall 8: Incomplete E2E Test Coverage After Changes

**What goes wrong:** The E2E test suite (`test/e2e/`) only tests a subset of tools, leaving renamed or sanitized tools untested.

**Why it happens:** `TestPiIntegration_Basic` only checks 17 of 31+ tools in the `expectedTools` list. The other 14+ tools (sandbox, schema, ontology tools besides search) are not verified by any E2E test for their name or basic function.

**Current E2E coverage:**
| Test | Tools Tested | Coverage |
|------|-------------|----------|
| `TestPiIntegration_Basic` | 17 tool names (list only) | ~55% of tools |
| `TestPiIntegration_ParameterValidation` | 5 tool calls | ~16% of tools |
| `TestDecisionLifecycle` | 11 tool calls | ~35% of tools |
| `TestDecisionSandboxFlow` | 4 tool calls | ~13% of tools |
| `TestDecisionAlternateFlows` | 3 tool calls | ~10% of tools |
| **Total unique tools tested** | ~18 tools | **~58%** |

**Untested by E2E:** `cancel_proposal`, `get_proposal_by_id`, `list_review_records`, `propose_action`, `describe_ontology`, `execute_action`, `list_action_schemas`, `get_action_schema`, `add_to_sandbox`, `compare_sandboxes`, `build_context`, `get_classification`, `check_access`

**Consequences:** Renaming any of the 14 untested tools would not be caught by CI unless the unit test (`TestServerToolRegistration`) is also running. If the unit test is skipped (e.g., during build), the change goes unnoticed until Pi Agent integration fails.

**Prevention:**
1. **Keep `TestServerToolRegistration` as a mandatory CI check** — This test validates all tool names.
2. **Add a smoke test that enumerates all tools** — `TestPiIntegration_Basic` should check all 31+ tools, not just 17.
3. **Run unit tests before E2E** — Unit tests (`go test ./internal/mcp/...`) catch rename errors in <1 second vs. E2E tests in ~30 seconds.

**Detection:**
- Code review: Check that `expectedTools` in both test files matches `mcp.NewTool("...")` calls
- Automation: `make test` must include `go test ./internal/mcp/...`

---

## Moderate Pitfalls

### Pitfall 9: Changing Tool Descriptions Breaks LLM Behavior

**What goes wrong:** When renaming tools, descriptions often get rewritten too. The LLM uses descriptions to decide which tool to call. If descriptions are too generic or imprecise, the LLM picks wrong tools.

**Why it happens:** Each tool's `mcp.WithDescription(...)` string is the LLM's primary signal for tool selection. Going from `"List alerts with optional filtering and sorting"` to `"Query alert records"` loses critical context about filtering capabilities.

**Consequences:** The LLM may pass incorrect parameters, or skip filtering entirely and filter client-side, causing excessive data transfer and worse decisions.

**Prevention:**
1. **Keep descriptions functionally accurate** — Describe what the tool does, not what it's named.
2. **Include parameter intent** — Mention key parameters in the description: `"Query alerts (filterable by severity, status, object_type)"`.
3. **Test with Pi Agent after description changes** — Verify the LLM still selects appropriate tools.

---

### Pitfall 10: Over-Grouping Tools into Too Few Categories

**What goes wrong:** In an effort to "abstract by business capability", tools get grouped into 3-4 broad categories, making the tool list harder to navigate and the names too generic.

**Why it happens:** The current 11 groups (decision, alert, governance, pipeline, outbox, review, action, status, ontology, sandbox, schema) are already domain-aligned. Trying to merge these into fewer groups produces awkward combinations like `data_maintenance_get_system_status` and `data_maintenance_run_pipeline`.

**Symptoms of over-grouping:**
- Group names become meaningless (e.g., "misc_operations" catches everything)
- Tool names need extra qualifiers: `alert_query` vs `decision_query` vs `object_query`
- Losing the clear domain separation that helps the LLM understand tool purpose

**Prevention:**
1. **Aim for 5-8 groups** — Current 11 groups can be collapsed to ~6 without losing clarity:
   - `case_*` (decision case lifecycle) — 6 tools
   - `alert_*` (monitoring and alerts) — 1 tool
   - `governance_*` (policy and classification) — 2 tools
   - `pipeline_*` (data pipeline) — 2 tools
   - `review_*` (approval workflow) — 5 tools
   - `object_*` (ontology queries) — 4 tools
   - `sandbox_*` (testing/simulation) — 4 tools
   - `system_*` (status and setup) — 2+ tools
2. **Don't merge unrelated tools** — Pipeline tools and ontology tools serve different purposes; merging them confuses the LLM.
3. **Validate naming with a colleague** — If another developer can't guess what a tool does from its name alone, it's too abstract.

---

### Pitfall 11: Renaming Handler Functions Creates Merge Conflicts

**What goes wrong:** Renaming both the tool name AND the handler Go function (`handleCreateDecisionCase` → `handleCaseCreate`) creates unnecessary code churn and merge conflicts.

**Why it happens:** The handler function names are internal Go symbols that don't affect MCP behavior. Renaming them is pure code churn with no security benefit.

**Prevention:**
1. **ONLY rename the MCP tool name string** — Keep handler function names unchanged:
   ```go
   // BEFORE:
   mcp.NewTool("create_decision_case", ...)
   s.server.AddTool(createCaseTool, s.handleCreateDecisionCase)
   
   // AFTER:
   mcp.NewTool("case_create", ...)  // <-- ONLY change this
   s.server.AddTool(createCaseTool, s.handleCreateDecisionCase)  // <-- keep as-is
   ```
2. **Use rename refactoring tools** — If renaming Go symbols, use IDE refactoring (not manual find-replace) to avoid missed callers.

---

## Minor Pitfalls

### Pitfall 12: Version String in Server Identity

Bumping the server version string (`"1.0.0"` → `"1.1.0"`) when doing information containment is unnecessary. Version strings help with debugging and don't significantly leak architecture. Keep the version aligned with project versioning.

### Pitfall 13: Tool Parameter Names Leak Domain Concepts

Parameter names like `object_type`, `rule_id`, `grain`, `source_type` hint at the underlying domain model. Replacing these with generic names (`entity_type` for `object_type`, `source` for `source_type`) adds minimal security benefit but breaks the LLM's understanding of what values to pass. Not worth changing.

### Pitfall 14: Removing Optional Parameters Breaks Backward Compatibility

Removing optional parameters from existing tools (e.g., removing `link_name` from `get_linked_objects`) is a breaking change for any client that passes that parameter. The mcp-go library silently ignores unknown parameters, so this won't cause errors—but it's technically a protocol-level change. Better to keep optional parameters and just ignore them if sanitizing output.

### Pitfall 15: Renaming Instructions String to Empty

Setting `server.WithInstructions("")` removes all context for the LLM. The instructions string describes the server's domain and helps the LLM decide when to use its tools. Keep a functional description even if it's generic.

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| **INT-01**: Server identity | Pitfall 6 — LLM confusion after rename | Keep functional description in instructions |
| **INT-02**: Tool rename | Pitfall 1, 7, 9, 10 — test failures, stale Pi hints, over-grouping | Rename as atomic commit, update tests+Pi, keep 5-8 groups |
| **INT-03**: describe_ontology output | Pitfall 3 — LLMReadable filtering only in describe, not in data endpoints | Ensure filtering is consistent across all ontology handlers |
| **INT-04**: get_system_status | Pitfall 4 — table_counts leak DB schema | Remove table_counts, aggregate errors |
| **INT-05**: get_object/get_linked_objects | Pitfall 3 — unfiltered properties | Add handler-level field filtering |
| **INT-06**: search_objects | Pitfall 3 — unfiltered search results | Apply same field filter as get_object |
| **INT-07**: run_pipeline input | Pitfall 5 — free-form config | Implement allowlist, hardcode data_dir |
| **Cross-cutting**: Error messages | Pitfall 2 — SQL/schema in errors | Add error sanitization helper, audit all error paths |
| **Testing** | Pitfall 8 — incomplete E2E coverage | Extend `TestPiIntegration_Basic` to verify all 31+ tools |

---

## Recovery Strategies for Pi Agent Integration Break

### Immediate Rollback (Fastest, < 1 min)
```bash
# Rebuild the OLD binary and restart MCP
git checkout HEAD~1 -- cmd/baxi-mcp/ internal/mcp/
go build -o /tmp/baxi-mcp ./cmd/baxi-mcp
# The Pi Agent will reconnect on next MCP call (stdio transport)
```

### Staged Migration (Lowest Risk)
1. **Phase 1** — Deploy only output filtering changes (INT-03, INT-04, INT-05). No rename.
2. **Test** — Verify Pi Agent works with filtered outputs.
3. **Phase 2** — Deploy rename (INT-01, INT-02) with Pi extension updates.
4. **Test** — Full integration test.
5. **Phase 3** — Deploy input hardening (INT-06, INT-07).

### Pi Extension Fallback (If Decision Workflow Breaks)
The `baxi-operations` and `baxi-sandbox` extensions use REST API and WILL CONTINUE WORKING regardless of MCP changes. Only the `baxi-decision` extension's `ctx.sendMessage` hints are affected.

To fix `baxi-decision`:
```typescript
// BEFORE (line 15):
ctx.sendMessage(`Please use the mcp tool to call baxi_mcp with: create_decision_case with alert_id=${params.alert_id}`);

// AFTER — generic instruction, let LLM discover tool names:
ctx.sendMessage(`Creating a decision case from alert ${params.alert_id}...`);
```

### Testing the Recovery
1. After any MCP change, run: `cd pi-extension && npm test` (verifies Pi extensions compile and register correctly)
2. Run: `go test -tags integration ./test/e2e/...` (verifies MCP protocol integration)
3. Run: `go test ./internal/mcp/...` (verifies tool registration)

---

## Sources

- Codebase analysis: `internal/mcp/` (14 files), `cmd/baxi-mcp/main.go` (1294 lines), `test/e2e/` (2 test files), `pi-extension/` (4 extension dirs)
- mcp-go library v0.41.1 — tool name is an opaque string, protocol-level rename is safe
- Pi Agent SDK (`@earendil-works/pi-coding-agent`) — extension model uses `ctx.sendMessage` for MCP hints, REST API for operations
