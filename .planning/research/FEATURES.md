# Feature Research: MCP 信息收束 (Information Containment)

**Domain:** MCP Server Information Containment for AI Agent Access
**Researched:** 2026-06-06
**Confidence:** HIGH

## Feature Landscape

### Table Stakes (Minimum Viable Containment)

The MCP Server Security Standard (MSSS) L1 defines these as essential. An MCP server that exposes tool capabilities to an AI agent **must** implement these for any reasonable level of information containment. Missing any = architectural information leaks.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **Generic Server Identity** | Agent must not be able to identify the project from `server.info` — server name + instructions leak project name, domain, tech stack | LOW | Change `NewMCPServer("Baxi MCP Server", "1.0.0", ...)` to generic name like `"Data Platform Server"`, instructions to `"Data analysis and operations platform"` |
| **Tool Name Abstraction** | Tool names like `describe_ontology`, `run_pipeline`, `get_system_status` directly expose internal package naming, domain boundaries, and project architecture | MEDIUM | Rename to business-capability names: `describe_ontology` → `list_data_models`, `run_pipeline` → `execute_data_processing`, `get_system_status` → `view_platform_overview`. Must update `server_test.go` expected tool list |
| **Output Trimming for Structural Tools** | `describe_ontology` returns full schema (source table, properties, links, PKs) — reveals DB schema. `get_system_status` returns `table_counts` with raw table names | LOW | Strip `SourceDescriptor` (schema/table/pk) from ontology descriptor output. Remove `table_counts` from status. Keep aggregate alert count, pipeline run status |
| **Field-Level Filtering** | `get_object` / `get_linked_objects` return all properties regardless of sensitivity — ontology already marks `sensitivity` and `llm_readable` but handlers ignore them | MEDIUM | Filter `obj.Properties` to only include fields where `llm_readable == true` and `sensitivity != "high"`. Apply same filtering in `get_linked_objects` and `search_objects` |
| **Input Hardening** | `run_pipeline` accepts free-form `config` string + arbitrary `data_dir` — agent can specify any pipeline config. `search_objects` has no query-length or result-size limits | LOW | `config` → allowlist of predefined pipeline names. `data_dir` → remove param, hardcode to `./data/raw`. `search_objects`: cap `limit` at 50, reject queries > 200 chars |
| **Response Size Bounds** | No response size limits in any handler → agent can consume unbounded context from a single tool call, enabling prompt-stuffing attacks via tool output | LOW | Set `max_response_size` per tool (e.g., 10KB for searches, 50KB for get_object). Truncate or paginate beyond the limit |

### Differentiators (Beyond Basic Containment)

Features that provide defense-in-depth and significantly raise the bar for adversarial agents. The MSSS standard maps these to L2+ controls.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Tool-Level Sensitive-Action Confirmation** | State-mutating tools (`run_pipeline`, `execute_proposal`) require explicit confirmation when invoked within a turn where a read tool was also called — prevents confused-deputy agents | MEDIUM | Requires turn-tracking state in the MCP server. Check: was a "read" tool (get_object, search_objects) called before this mutation? If yes, return a confirmation-required response |
| **Content Sanitization Pipeline** | Tool outputs from external sources are wrapped in untrusted-content delimiters, preventing indirect prompt injection through tool response data | HIGH | Add post-processing step in each handler that fences output with `<tool_output>...</tool_output>` markers. For text fields, strip embedded role markers (`SYSTEM`, `ASSISTANT`, `<\|im_start\|>`) |
| **Per-Tool Output Budgeting** | Different tools have different information value. High-value tools (describe_ontology with its schema map) need stricter trimming than simple status tools | MEDIUM | Classify tools into tiers: Tier 1 (structural) = heavily trimmed, Tier 2 (data retrieval) = field-filtered, Tier 3 (action) = minimal output. Apply tier-specific output policies |
| **Response Body Audit Store** | Separates full tool response bodies from operational logs — prevents log store from becoming a secondary PII repository | MEDIUM | Store full responses keyed by correlation ID in a separate access-controlled bucket with shorter retention. Return summary or truncated content in the operational log |
| **Prompt Injection Detection Heuristics** | Catches common injection patterns in tool input parameters — `query` strings in `search_objects` that contain "ignore previous instructions" patterns | HIGH | Add regex-based detection for role markers, instruction overrides, and exfiltration commands. Can use Go regex patterns matching known injection signatures |
| **Per-Tool Kill-Switch Feature Flags** | Ability to disable a single tool independently of deploy cycle — incident response in minutes instead of hours | LOW | Add a `map[string]bool` config checked at the top of each handler. When a tool is disabled, handler returns `ToolResultError("tool disabled")` without executing |
| **Turn-Based Context Leak Prevention** | Tracks which tools were invoked in the current "turn" (sequence between user messages). Blocks suspicious patterns like: read sensitive data → call write tool with data in args | HIGH | Requires session state tracking. Compare tool call patterns against suspicious sequences. Reject calls that match exfiltration patterns |
| **Sensitivity-Adaptive Pagination** | For `search_objects`, dynamically reduce `limit` based on result sensitivity. High-sensitivity object types return fewer results per page | MEDIUM | Add a sensitivity-tier → max-results mapping. High-sensitivity: max 10. Medium: max 25. Low: max 50. Check ontology object-type sensitivity |
| **Untrusted-Content Fencing** | All content from external sources (webhook payloads, fetched documents, user-submitted data) is wrapped in clear delimiters telling the agent "this is data, not instructions" | MEDIUM | Not high priority for Baxi since most tools access internal database, not external content. Useful for channels that receive external data |

### Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| **Full RBAC for MCP** | "Let's control exactly which tools each agent can access" | Single-tenant deployment, single agent (Pi Agent). RBAC overhead doesn't pay for itself. Complexity far outweighs benefit | Use env-var based tool blacklist instead. `MCP_DISABLED_TOOLS=describe_ontology,run_pipeline` |
| **Complete Tool Output Encryption** | "Encrypt all tool responses so even if agent is compromised, data stays safe" | MCP stdio transport is local process communication. Encryption adds overhead with no real benefit in single-tenant, local deployment | Focus on access control and output filtering instead |
| **AI-Only Log Analysis** | "Let the LLM analyze audit logs to detect attacks" | Hallucination risk, circular reasoning (the compromised agent evaluates its own behavior) | Use structured logging with deterministic alert rules. Human review for suspicious patterns |
| **Blind Output Truncation (all tools, same limit)** | "Simplest approach — just truncate all tool outputs to N chars" | Blind truncation breaks tools that need to return complete data (list_cases, search_results) and doesn't protect against targeted leaks | Per-tool, tiered output policies based on tool function and data sensitivity |
| **Removing Tools Entirely** | "If we remove the tool, it can't leak" | Breaks Pi Agent functionality. The goal is controlled access, not no access | Keep tools, apply appropriate containment for each. Removal is a last resort |

## Feature Dependencies

```
Generic Server Identity
    └──requires──> No other features

Tool Name Abstraction
    └──requires──> server_test.go tool list update

Output Trimming (describe_ontology / get_system_status)
    └──requires──> Understanding of what each output field exposes
                   (already researched: SourceDescriptor exposes DB schema)

Field-Level Filtering
    └──requires──> Ontology PropertyDescriptor metadata (already exists:
                   LLMReadable, Sensitivity markers exist but unused)
    └──enhances──> search_objects hardening (reuses the same filter logic)

Input Hardening (search_objects / run_pipeline)
    └──requires──> Allowlist definition for pipeline configs
    └──enhances──> Field-level filtering (can be applied in same handler pass)

Per-Tool Kill-Switch
    └──requires──> Handler config map (independent feature)
    └──enhances──> All other features (provides emergency stop)

Content Sanitization Pipeline
    └──requires──> Tool output format understanding (independent)
    └──enhances──> Field-Level Filtering (combined sanitization pass)

Prompt Injection Detection
    └──requires──> Defined injection signatures
    └──conflicts──> Overly broad detection that blocks legitimate queries
```

### Dependency Notes

- **Field-Level Filtering depends on existing ontology metadata:** The `PropertyDescriptor.Sensitivity` and `PropertyDescriptor.LLMReadable` fields are already defined in `interfaces.go` but never checked in the tool handlers. No new data model work needed.
- **Generic Server Identity is fully independent:** It's a single change to `server.go:77-82` in `NewMCPServer()`. No other code depends on it.
- **Tool Name Abstraction requires test updates:** `server_test.go` maintains an `expectedTools` slice that must be updated to match the new names. Tests will fail until the list is synced.
- **Input Hardening for run_pipeline needs business input:** The allowlist of permitted pipeline configs must be defined based on what's safe to expose. Default to empty (block all) and add known-safe configs.
- **Per-Tool Kill-Switch enhances everything:** It's an independent safety net. Add early in the milestone so other features can rely on it.

## MVP Definition

### Launch With (v1 — This Milestone)

The minimum viable set for meaningful information containment. These 7 items directly address the stated goal: "Agent 接触 MCP 时无法拼凑项目架构，也无法获取不该拿的业务数据".

- [x] **Generic Server Identity** — Rename server + blur instructions. Eliminates the most obvious signal about what the system is. **Complexity: LOW, Effort: ~15 min**
- [x] **Tool Name Abstraction** — Rename all 31 tools to business-capability names. Removes the primary mechanism for inferring internal architecture from tool listings. **Complexity: MEDIUM, Effort: ~2 hours** (31 renames + test updates)
- [x] **Output Trimming (describe_ontology / get_system_status)** — Strip schema details and table names. Removes direct DB schema exposure. **Complexity: LOW, Effort: ~1 hour**
- [x] **Field-Level Filtering (get_object / get_linked_objects)** — Filter properties using existing LLMReadable/sensitivity markers. Blocks access to non-public fields. **Complexity: MEDIUM, Effort: ~3 hours** (filter function + handler integration + tests)
- [x] **Input Hardening (run_pipeline / search_objects)** — Config allowlist, data_dir fixed, query length limits, result pagination caps. Prevents abuse of tool parameters to access unauthorized resources. **Complexity: LOW, Effort: ~2 hours**
- [x] **Response Size Bounds** — Per-tool maximum response size with truncation. Prevents context-stuffing through overly large responses. **Complexity: LOW, Effort: ~1 hour**
- [x] **Per-Tool Kill-Switch** — Map-based enable/disable for each tool. Quick containment for any tool that turns out to leak more than expected. **Complexity: LOW, Effort: ~1 hour**

### Add After Validation (v1.x)

Features to layer on once core containment is in production and weaknesses are observed.

- [ ] **Content Sanitization** — If agents are observed following instructions embedded in tool output data, add fencing and role-marker stripping. Trigger: observed prompt injection through tool response.
- [ ] **Tool-Level Sensitive-Action Confirmation** — If agents are observed chaining read + write tools to exfiltrate data (e.g., read sensitive object → pass data to a write tool parameter), add turn-tracking confirmation. Trigger: audit log shows read-write chaining.
- [ ] **Prompt Injection Detection** — If input parameters are used to inject instructions (especially `search_objects.query`), add regex-based injection signature detection. Trigger: observed injection attempts in logs.
- [ ] **Sensitivity-Adaptive Pagination** — If high-sensitivity object searches return too many results, add per-tier pagination limits. Trigger: business feedback that search returns too much sensitive data.

### Future Consideration (v2+)

Features that require significant architectural changes or whose value is uncertain for the current deployment model.

- [ ] **Turn-Based Context Leak Prevention** — Requires session state tracking across tool calls. Not practical without significant state management additions. Defer until multi-turn patterns are better understood.
- [ ] **Response Body Audit Store** — Requires separate storage infrastructure. Overhead doesn't justify for single-tenant demo deployment. Add if production deployment with compliance requirements.
- [ ] **Full RBAC** — Single agent, single tenant. RBAC is architecture overkill. If multi-agent or multi-tenant MCP is needed, revisit.

## Feature Prioritization Matrix

| Feature | User Value (Containment Impact) | Implementation Cost | Priority |
|---------|--------------------------------|---------------------|----------|
| Generic Server Identity | MEDIUM — prevents easy identification but doesn't stop determined probing | LOW — 2 lines changed | **P1 (MVP)** |
| Tool Name Abstraction | HIGH — eliminates primary architecture fingerprinting vector | MEDIUM — 31 renames + test sync | **P1 (MVP)** |
| Output Trimming (describe_ontology / get_system_status) | HIGH — schema and table names are the most sensitive structural leaks | LOW — modify 2 handlers | **P1 (MVP)** |
| Field-Level Filtering | CRITICAL — directly prevents business data leaks via sensitivity markers | MEDIUM — filter function + 3 handler integrations | **P1 (MVP)** |
| Input Hardening (run_pipeline / search_objects) | HIGH — prevents agent from running arbitrary pipelines or unbounded searches | LOW — allowlist + input validation in 2 handlers | **P1 (MVP)** |
| Response Size Bounds | MEDIUM — prevents context-stuffing, less critical than field filtering | LOW — per-tool size check at response construction | **P1 (MVP)** |
| Per-Tool Kill-Switch | MEDIUM — safety net, not a primary containment mechanism | LOW — config map + handler guard | **P1 (MVP)** |
| Content Sanitization | MEDIUM — important defense-in-depth but no observed injection vectors yet | MEDIUM — content scanning per tool output | **P2 (v1.x)** |
| Tool-Level Sensitive-Action Confirmation | MEDIUM — prevents chained read-write exfiltration | MEDIUM — turn tracking state | **P2 (v1.x)** |
| Prompt Injection Detection | LOW — inputs are developer-controlled, not user-controlled yet | HIGH — regex signature library + maintenance | **P2 (v1.x)** |
| Sensitivity-Adaptive Pagination | LOW — adequate pagination exists, this is optimization | MEDIUM — per-type limit config | **P3 (v2+)** |
| Turn-Based Context Leak Prevention | LOW — no evidence of this attack pattern in current deployment | HIGH — session state tracking | **P3 (v2+)** |
| Response Body Audit Store | LOW — single-tenant, short retention needs are minimal | HIGH — separate storage infra | **P3 (v2+)** |
| Full RBAC | LOW — single agent, single tenant | VERY HIGH — new auth infrastructure | **Out of Scope** |

**Priority key:**
- **P1:** Required for MVP — deliver in this milestone
- **P2:** Add after validation — observed need triggers implementation
- **P3:** Future consideration — significant architecture or low current value
- **Out of Scope:** Not aligned with deployment model

## Implementation Ordering Within Milestone

Based on dependency analysis, the optimal execution order is:

```
Wave 1 (Independent, parallel):
  1. Generic Server Identity       (server.go, no deps)
  2. Tool Name Abstraction         (all tools_*.go + server_test.go, no deps)
  3. Per-Tool Kill-Switch          (handler config, no deps)

Wave 2 (Builds on Wave 1):
  4. Output Trimming               (tools_ontology.go, tools_status.go)
  5. Input Hardening               (tools_pipeline.go, tools_status.go)
  6. Response Size Bounds          (response helper in all handlers)

Wave 3 (Builds on Wave 2):
  7. Field-Level Filtering         (tools_ontology.go, tools_status.go — reuses
                                     ontology metadata already in interfaces.go)
```

## Sources

- **MCP Official Security Best Practices** — https://modelcontextprotocol.io/docs/tutorials/security/security_best_practices — HIGH confidence (official protocol docs)
- **MCP Server Security Standard (MSSS) v0.1** — https://github.com/mcp-security-standard/mcp-server-security-standard — HIGH confidence (24 controls across 8 domains, L1-L4 compliance levels)
- **mcp-sanitizer (npm library)** — https://github.com/starman69/mcp-sanitizer — MEDIUM confidence (community project, demonstrates input sanitization patterns)
- **mcpgw (Go MCP Firewall)** — https://github.com/knorq-ai/mcpgw — MEDIUM confidence (Go production pattern, policy engine approach)
- **MCP Server Security Best Practices 2026 Engineering Guide** — https://www.digitalapplied.com/blog/mcp-server-security-best-practices-2026-engineering-guide — MEDIUM confidence (industry guide, 8 practice areas)
- **Zealynx: MCP Server Hardening Guide** — https://www.zealynx.io/research/adversarial-security/mcp-server-hardening — MEDIUM confidence (security vendor, overlaps with industry consensus)
- **GitHub Agentic Workflows: Safe Outputs MCP Gateway Specification** — https://github.github.com/gh-aw/specs/safe-outputs-specification/ — HIGH confidence (GitHub's production MCP security architecture)
- **Baxi Codebase (existing ontology metadata)** — `internal/mcp/interfaces.go` lines 118-124 — HIGH confidence (Field-level filtering infrastructure already exists but unused)

---
*Feature research for: MCP Information Containment (Baxi MCP Server v1.1)*
*Researched: 2026-06-06*
