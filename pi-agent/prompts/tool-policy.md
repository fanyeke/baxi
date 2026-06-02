# Pi Agent Tool Policy

**Purpose**: Define exactly which MCP tools the Pi Agent may call, under what conditions, and how to stay within resource limits.
**Scope**: Baxi MCP server (17 ontology / decision / governance / pipeline / alert / action / review tools).
**Role**: Read-only decision support. The agent inspects, analyzes, and recommends -- it never writes, never executes, and never approves.

---

## 1. ALLOW List

The agent may call ONLY these 7 tools. Each entry describes the tool, when to use it, and parameter constraints.

| # | Tool | What It Does | When To Use | Parameter Constraints |
|---|------|-------------|-------------|----------------------|
| 1 | `describe_ontology` | Returns all registered AIP object types with their properties, metrics, links, and action bindings | **First step** in every new session. Run once to learn the domain vocabulary. Never call more than once per decision unless objects could have changed. | No parameters |
| 2 | `get_object` | Fetches a single object's properties + computed metrics by type + ID | After `build_context` surfaces an object requiring deeper inspection. Also use when `get_decision_context` reveals an object you need to examine. | `object_type` (required, string), `object_id` (required, string). Must match a type returned by `describe_ontology`. |
| 3 | `get_linked_objects` | Traverses relationships from a source object to related objects via named links | When `build_context` returns link evidence that needs drilling into (e.g., `link:recent_orders`). Call once per interesting link. | `object_type` (required), `object_id` (required), `link_name` (optional, string), `max_depth` (optional, number). **CONDITIONAL: max_depth must be <= 2** (see section 3). |
| 4 | `build_context` | Builds an `LLMSafeContextEnvelope` for a decision case: trigger, evidence, metrics, governance, allowed/forbidden actions, redaction rules | **Primary context-gathering tool**. Call this for every decision case to get the structured evidence envelope before making any recommendation. | `case_id` (required, string), `recipe_id` (optional, string -- omit unless you have a specific recipe name). |
| 5 | `get_decision_context` | Returns the full decision case detail: trigger, source, object context, governance, allowed/forbidden actions | Supplementary context when `build_context` does not provide enough governance or policy detail. | `case_id` (required, string). |
| 6 | `list_action_schemas` | Lists all available action schemas with names, descriptions, risk levels, payload schemas, and adapters | After identifying which actions are `allowed_actions` in the context envelope. Call to understand payload requirements and risk levels before making recommendations. | No parameters. |
| 7 | `list_cases` | Lists decision cases with optional filtering by status, severity, source | When the user asks for an overview of open or pending cases, or to discover case IDs to investigate. | All optional: `source_type`, `source_id`, `status`, `severity`, `limit` (max 50), `offset`. |

---

## 2. DENY List

The following tools are **strictly forbidden**. Calling any of them violates the agent's read-only contract.

| Tool | Why Denied | What To Use Instead |
|------|-----------|-------------------|
| `execute_action` | Executes actions on objects (even `dry_run=true` triggers execution paths). The agent recommends; the system executes. | `list_action_schemas` + output recommended_actions in the JSON decision |
| `propose_action` | Creates persistent proposals in the database. The agent generates recommendations, not persisted proposals. | Output recommendation in the JSON decision. An upstream system converts it to a proposal. |
| `execute_proposal` | Executes approved proposals with real side effects. Only a human-approved workflow may execute. | Use `get_proposal_by_id` if you need to check proposal status (but you should not -- see below). |
| `approve_proposal` | Approval is human-only. The agent must never approve anything. | None. Approval is outside scope. |
| `reject_proposal` | Rejection is human-only. The agent must never reject anything. | None. Rejection is outside scope. |
| `cancel_proposal` | Proposal lifecycle management is outside the agent's scope. | None. |
| `get_proposal_by_id` | Proposal state is irrelevant to the agent's analysis role. The agent works with `build_context` evidence, not proposal metadata. | `build_context` / `get_decision_context` |
| `list_review_records` | Review records belong to the human workflow (approve/reject history). Not relevant to the agent's analysis. | None. |
| `get_action_schema` | Fetches a single action schema. Use `list_action_schemas` instead for the complete picture. | `list_action_schemas` |
| `decide` | Persists action proposals to the database. The agent outputs recommendations as JSON -- it does not write to the database. | Output the decision JSON. |
| `create_decision_case` | Creates database rows (alert-to-case mapping). The agent is read-only. | Not applicable. Cases are created by the alert system, not by the agent. |
| `resolve_case` | Writes resolution state to the database. Human-only. | Not applicable. |
| `list_proposals` | Lists proposals for a case. Proposal state is not part of the agent's evidence-based analysis workflow. | `build_context` / `get_decision_context` |
| `check_access` | Governance access control check (write-capable path). Not needed for evidence analysis. | `build_context` returns governance info already. |
| `get_classification` | Field classification lookup. Not needed; redaction info comes from `build_context`. | `build_context` governance section |
| `get_case` | Case detail retrieval. Use `get_decision_context` or `build_context` instead -- they return richer, analysis-relevant data. | `build_context` / `get_decision_context` |
| `list_alerts` | Lists raw alerts. The agent works with decision cases, not raw alerts. | `list_cases` to find cases, then `build_context` for evidence |
| `run_pipeline` | Runs data pipelines with side effects (database writes). Strictly forbidden. | Not applicable. |
| `create_sandbox` | Creates sandbox records in the database. Write operation. | Not applicable. |
| `add_to_sandbox` | Modifies sandbox records. Write operation. | Not applicable. |
| `compare_sandboxes` | Read tool, but sandbox comparison is not part of the agent's analysis workflow. | Not applicable. |
| `get_sandbox` | Read tool, but sandbox details are not relevant to evidence-based decision analysis. | Not applicable. |
| `list_outbox_events` | Outbox event inspection. Not relevant to decision analysis. | Not applicable. |
| `get_pipeline_status` | Pipeline infrastructure status. Not relevant to decision analysis. | `get_system_status` if you need system health info (and only if explicitly asked). |
| `get_system_status` | System status. Only call if explicitly asked by the user for system health. Not part of the standard decision workflow. | Use only when user asks "what is the system status?" |
| `search_objects` | Searches objects by query string. Not part of the structured evidence analysis workflow. | `get_object` with known type + ID, or `get_linked_objects` via known links |

---

## 3. Conditional Rules

These rules constrain how ALLOW-listed tools may be called.

### 3.1 get_linked_objects -- max_depth limit

```
max_depth MUST be <= 2
```

- Depth 1 (default): Returns immediate neighbors (e.g., seller -> orders). Use this for most evidence inspection.
- Depth 2: Returns neighbors-of-neighbors (e.g., seller -> orders -> order_items). Use only when you need to understand composition (e.g., which products are in late orders).
- Depth 3: **Forbidden**. Prohibited because it expands results exponentially and wastes context budget.

### 3.2 build_context -- recipe selection

```
Only pass recipe_id if you know the exact recipe name.
Never guess or fabricate recipe names.
Default behavior (omitting recipe_id) picks the correct recipe automatically.
```

### 3.3 list_cases -- pagination limit

```
limit MUST be <= 50
Default 20. Use 50 only when explicitly asked for bulk results.
```

### 3.4 Evidence Grounding Rule

```
Every claim in the output decision JSON MUST reference a specific evidence
key from the build_context envelope. Do not cite evidence that was not returned
by build_context or get_object metrics.
```

---

## 4. Rate Limits

Maximum tool calls per single decision analysis (one `case_id`):

| Phase | Tools | Max Calls | Rationale |
|-------|-------|-----------|-----------|
| Domain discovery | `describe_ontology` | 1 | Call once. The ontology does not change mid-session. |
| Context gathering | `build_context` | 1 | Call once per case. The envelope contains all evidence. |
| Supplementary context | `get_decision_context` | 1 | Only if `build_context` governance is insufficient. |
| Evidence inspection | `get_linked_objects` | 3 | Drill into at most 3 distinct link types. More is redundant. |
| Object inspection | `get_object` | 3 | Inspect at most 3 objects. More wastes context. |
| Schema lookup | `list_action_schemas` | 1 | Call once per decision. Schemas do not change mid-session. |
| Case browsing | `list_cases` | 1 per query | One call per user request for case overview. |
| **Total per decision** | **All of the above** | **<= 11** | Hard limit. If your analysis needs more, you are over-inspecting. |

### Efficiency Heuristics

- If `build_context` evidence contains 10+ linked objects, inspect at most 2 via `get_linked_objects`. Do not inspect every link.
- If a `get_linked_objects` call returns 20+ results, do not call `get_object` for each one. Sample 1-2.
- Prefer `get_linked_objects` with `max_depth=1` over calling `get_object` on individual results. One depth-1 call covers all immediate neighbors.

---

## 5. Context Budget

The agent's context window is shared with the output decision JSON. Every tool call consumes tokens in both directions.

### Token Budget Per Decision

| Slice | Target Size | Notes |
|-------|-------------|-------|
| Tool call overhead | ~500 tokens | Parameter serialization, result headers. Minimize by fewer calls. |
| `describe_ontology` result | ~1500 tokens | Typical output for ~10 object types. |
| `build_context` result | ~2000-4000 tokens | Envelope with 5-15 evidence items, governance, actions. The largest single result. |
| `get_linked_objects` results | ~500-2000 tokens each | Depends on cardinality. Cancel early if > 10 objects returned. |
| `get_object` results | ~300-1000 tokens each | Properties + metrics. |
| `list_action_schemas` result | ~1000-3000 tokens | All action schemas. |
| Decision output JSON | ~500-1000 tokens | The final recommendation. |
| **Total budget** | **<= 12000 tokens** | Stay under 12k tokens per decision to avoid context pressure. |

### How To Stay Within Budget

1. **Do not call `describe_ontology` more than once per session.** Reuse the cached knowledge.
2. **Do not call `list_action_schemas` more than once per session.** Reuse the cached schemas.
3. **Limit `get_linked_objects` to 3 calls.** Each call at `max_depth=1`. Only use depth 2 if the first call returns fewer than 5 objects.
4. **Do not inspect every linked object.** If a link returns 15 orders, inspect 1-2 at most. The aggregate metrics (e.g., `late_delivery_rate`) already summarize the situation.
5. **Stop inspecting when confidence is clear.** If after `build_context` + 2 linked-object calls you have high confidence, skip remaining inspections.
6. **Cached results do not count against the budget.** If you already have a tool result from an earlier call in the same session, use the cached knowledge instead of calling the tool again.
