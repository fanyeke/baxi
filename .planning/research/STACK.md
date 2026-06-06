# Technology Stack — MCP Information Containment

**Project:** Baxi MCP Server
**Researched:** 2026-06-06
**Mode:** Ecosystem / Feasibility
**Target:** mcp-go v0.41.1 (existing dependency, no upgrades needed)

---

## 1. Executive Summary

All information containment requirements (INT-01 through INT-07) can be implemented **entirely within the existing mcp-go v0.41.1 library** — no new dependencies, no version upgrades. The library provides two first-class mechanisms: `ToolHandlerMiddleware` (for intercepting/filtering tool call outputs) and `ToolFilterFunc` (for controlling which tools appear in listings). Server identity obfuscation is accomplished by changing constructor arguments — no protocol-level hooks needed.

### Key Findings

| INT ID | Requirement | Mechanism | Implementation Target |
|--------|------------|-----------|----------------------|
| INT-01 | Server identity obfuscation | `NewMCPServer()` name/version + `WithInstructions()` | `internal/mcp/server.go` constructor |
| INT-02 | Tool name abstraction | `mcp.NewTool()` name/description strings | Each `tools_*.go` registration function |
| INT-03 | `describe_ontology` output trimming | `ToolHandlerMiddleware` → JSON filter | New `output_filter.go` in `internal/mcp/` |
| INT-04 | `get_system_status` output trimming | `ToolHandlerMiddleware` → JSON filter | Same middleware |
| INT-05 | `get_object` field-level filtering | `ToolHandlerMiddleware` → JSON filter (uses LLMReadable) | Same middleware |
| INT-06 | `search_objects` input/cap enforcement | Input validation in handler OR middleware pre-check | `tools_status.go` handler |
| INT-07 | `run_pipeline` input allowlist | Handler-level input validation | `tools_pipeline.go` handler |

**Critical constraint:** `ToolFilterFunc` only affects `tools/list` responses — it does NOT prevent a client from calling a tool by its original name. All filtering must be enforced at the handler/middleware level, not just in listings.

---

## 2. Recommended Approach: Hybrid Middleware + Per-handler Filter

### Architecture

```
┌─────────────────────────────────────────────────┐
│  MCP Client (Pi Agent)                          │
└──────────────────┬──────────────────────────────┘
                   │ stdio
                   ▼
┌─────────────────────────────────────────────────────┐
│  ServeStdio(s.server)                                │
│  ┌─────────────────────────────────────────────────┐ │
│  │  handleToolCall()                               │ │
│  │  ┌───────────────────────────────────────────┐  │ │
│  │  │  ToolHandlerMiddleware (global chain)     │  │ │
│  │  │  ┌─────────────────────────────────────┐ │  │ │
│  │  │  │  OutputSanitizerMiddleware           │ │  │ │
│  │  │  │  - Intercepts *CallToolResult         │ │  │ │
│  │  │  │  - Dispatches to per-tool filter      │ │  │ │
│  │  │  │  - Rewrites JSON content              │ │  │ │
│  │  │  └─────────────────────────────────────┘ │  │ │
│  │  │  ┌─────────────────────────────────────┐ │  │ │
│  │  │  │  Original tool handler               │ │  │ │
│  │  │  │  (unchanged logic, unchanged         │ │  │ │
│  │  │  │   service calls)                     │ │  │ │
│  │  │  └─────────────────────────────────────┘ │  │ │
│  │  └───────────────────────────────────────────┘  │ │
│  └─────────────────────────────────────────────────┘ │
│                                                       │
│  ┌─────────────────────────────────────────────┐     │
│  │  ToolFilterFunc : tools/list                │     │
│  │  - Filters tool metadata from listing       │     │
│  │  - Renames descriptions in listing          │     │
│  └─────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────┘
```

### Layer 1: Server Identity (INT-01)

**File:** `internal/mcp/server.go` — line 77-82

**Current:**
```go
s := server.NewMCPServer(
    "Baxi MCP Server",           // exposes "Baxi" project identity
    "1.0.0",
    server.WithToolCapabilities(false),
    server.WithInstructions("E-commerce governance and decision platform"),
)
```

**Target:**
```go
s := server.NewMCPServer(
    "Data Agent Interface",          // generic name
    "1.0.0",
    server.WithToolCapabilities(false),
    server.WithInstructions("Query and manage data objects through natural language"),
)
```

**Strategy:** Simple string replacement. The `ServerInfo` field in `InitializeResult` (server.go line 699-702) reads `s.name` and `s.version` directly. No protocol-level hook needed.

**Confidence:** HIGH — Verified in mcp-go source `server/server.go:697-705`.

### Layer 2: Tool Name & Description Abstraction (INT-02)

**File:** Each `tools_*.go` registration function

**Approach:** Rename tools at `mcp.NewTool()` call sites. Group by business capability instead of internal package mapping.

**Current pattern (leaks package structure):**
```go
tool := mcp.NewTool("check_access",
    mcp.WithDescription("Check if a role has access to perform an action on an object type"),
)
```

**Target pattern (generic description):**
```go
tool := mcp.NewTool("access_evaluate",
    mcp.WithDescription("Evaluate whether an operation is permitted"),
)
```

**Tool renaming map:**
| Current Name | Target Name | Rationale |
|---|---|---|
| `check_access` | `access_evaluate` | Remove "check" verb, generify |
| `get_classification` | `data_classify` | Shorter, standard verb |
| `describe_ontology` | `schema_list` | Remove domain term "ontology" |
| `get_object` | `data_get` | Standard CRUD naming |
| `get_linked_objects` | `data_related` | Remove "linked" (implementation detail) |
| `search_objects` | `data_search` | Standard naming |
| `get_system_status` | `system_health` | Remove "status" (implies health) |
| `execute_action` | `action_execute` | Consistent verb-first naming |
| `create_decision_case` | `case_create` | Standard CRUD naming |
| `list_cases` | `case_list` | Standard CRUD naming |
| `get_case` | `case_get` | Standard CRUD naming |
| `get_proposal_by_id` | `proposal_get` | Remove "by_id" (implementation detail) |
| `list_proposals` | `proposal_list` | Standard CRUD naming |
| `run_pipeline` | `pipeline_run` | Verb-first |
| `list_alerts` | `alert_list` | Standard naming |
| `approve_proposal` | `proposal_approve` | Verb-first |
| `reject_proposal` | `proposal_reject` | Verb-first |
| `cancel_proposal` | `proposal_cancel` | Verb-first |
| `list_review_records` | `review_list` | Standard naming |
| `list_outbox_events` | `event_list` | Remove "outbox" (implementation detail) |
| `get_pipeline_status` | `pipeline_status` | Standard naming |
| `create_sandbox` | `sandbox_create` | Verb-first |
| `add_to_sandbox` | `sandbox_add` | Verb-first |
| `compare_sandboxes` | `sandbox_compare` | Verb-first |
| `get_sandbox` | `sandbox_get` | Verb-first |
| `list_action_schemas` | `action_schemas` | Standard naming |
| `get_action_schema` | `action_schema_get` | Verb-first |
| `get_decision_context` | `case_context` | Standard naming |
| `execute_proposal` | `proposal_execute` | Verb-first |
| `decide` | `case_decide` | Add context prefix |
| `resolve_case` | `case_resolve` | Verb-first |
| `build_context` | `case_build_context` | Add context prefix |

**Important:** If `tools/list` is the only channel the client uses to discover tools, and you rename all tools there, the client can never learn the original names. But if any documentation or configuration references original names externally, the client could still call them. This is acceptable because renamed tools use new handler function references — the old names simply no longer exist.

**Confidence:** HIGH — Direct `mcp.NewTool()` API usage, no protocol implications.

### Layer 3: Tool Output Sanitization Middleware (INT-03, INT-04, INT-05)

**Primary Mechanism:** `server.WithToolHandlerMiddleware(middlewareFn)`

**Type signature (from mcp-go source `server/server.go:44`):**
```go
type ToolHandlerMiddleware func(ToolHandlerFunc) ToolHandlerFunc
```

**How it works (from mcp-go source `server/server.go:1169-1189`):**
Middlewares are applied in reverse order, wrapping the handler chain. Each middleware receives the next handler in the chain and returns a new handler.

**Recommended Implementation Pattern:**

Create a new file `internal/mcp/output_filter.go`:

```go
package mcp

import (
    "encoding/json"
    "fmt"

    "github.com/mark3labs/mcp-go/mcp"
)

// ToolOutputFilter defines per-tool output sanitization.
// Implementations inspect and modify tool call results.
type ToolOutputFilter interface {
    // FilterToolResult transforms the result of a tool call.
    // The toolName identifies which tool produced the result.
    // result is the raw *CallToolResult that can be mutated in place.
    // Return the (possibly modified) result.
    FilterToolResult(ctx context.Context, toolName string, result *mcp.CallToolResult) *mcp.CallToolResult
}

// NewOutputSanitizerMiddleware creates a ToolHandlerMiddleware that
// intercepts every tool call result and applies the given filter.
func NewOutputSanitizerMiddleware(filter ToolOutputFilter) server.ToolHandlerMiddleware {
    return func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
        return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
            result, err := next(ctx, req)
            if err != nil {
                return nil, err
            }
            if result == nil {
                return nil, nil
            }
            return filter.FilterToolResult(ctx, req.Params.Name, result), nil
        }
    }
}
```

**Field-Level JSON Content Filtering Strategy:**

Since all existing handlers use `NewToolResultJSON()` which marshals structs/maps to JSON text, the middleware needs to:

1. Check if the `result.Content` contains a JSON text payload
2. Unmarshal the JSON into `map[string]interface{}`
3. Apply field-level redaction/rules
4. Re-marshal into the result

```go
// jsonContentFilter extends ToolOutputFilter to modify JSON text content
// in tool call results.
type jsonContentFilter struct {
    rules map[string]func(map[string]interface{}) map[string]interface{}
}

func (f *jsonContentFilter) FilterToolResult(ctx context.Context, toolName string, result *mcp.CallToolResult) *mcp.CallToolResult {
    // Don't filter error results
    if result.IsError {
        return result
    }
    
    // Find the filter rule for this tool
    rule, ok := f.rules[toolName]
    if !ok {
        return result
    }
    
    // Process each content item
    for i, content := range result.Content {
        if tc, ok := content.(mcp.TextContent); ok {
            var data map[string]interface{}
            if err := json.Unmarshal([]byte(tc.Text), &data); err != nil {
                continue // not JSON, skip
            }
            data = rule(data)
            cleaned, _ := json.Marshal(data)
            result.Content[i] = mcp.TextContent{
                Type: tc.Type,
                Text: string(cleaned),
            }
        }
    }
    
    // Also clean StructuredContent if present
    if result.StructuredContent != nil {
        // Could also sanitize the structured content
        // (this is a `map[string]interface{}` for NewToolResultJSON calls)
    }
    
    return result
}
```

**Filter Rules for Each Target Tool:**

| Tool Name | Filter Rule | What to Remove |
|---|---|---|
| `describe_ontology` (→ `schema_list`) | Remove `source` (schema/table/pk), `governance`, `sensitivity` | Infrastructure schema details |
| `get_system_status` (→ `system_health`) | Remove `table_counts`, truncate `recent_errors` | Database schema, internal error details |
| `get_object` (→ `data_get`) | Apply `LLMReadable` filter on properties | Non-LLM-readable fields |
| `get_linked_objects` (→ `data_related`) | Apply `LLMReadable` filter on linked object properties | Same as get_object |

For INT-05 (`get_object` field-level filtering using `LLMReadable`), the filter rule needs access to the ontology registry to know which properties are allowed. This can be injected via the filter struct:

```go
type ontologyAwareFilter struct {
    registry  *ontology.ObjectRegistry
    baseRules map[string]func(map[string]interface{}) map[string]interface{}
}

func (f *ontologyAwareFilter) FilterToolResult(ctx context.Context, toolName string, result *mcp.CallToolResult) *mcp.CallToolResult {
    if result.IsError {
        return result
    }
    
    // Apply ontology-aware filtering for object-related tools
    switch toolName {
    case "data_get", "data_related":
        return f.filterObjectResult(ctx, result)
    default:
        // Fall through to base rules
        // (same pattern as jsonContentFilter above)
    }
}
```

**Integration into Server Constructor:**

In `internal/mcp/server.go`, the middleware is added via `server.WithToolHandlerMiddleware`:

```go
func NewServer(...) (*Server, error) {
    // Build the filter
    outputFilter := NewOutputSanitizer(&OutputSanitizerConfig{
        Registry: ontologySvc,  // for LLMReadable checks
    })
    
    s := server.NewMCPServer(
        "Data Agent Interface",
        "1.0.0",
        server.WithToolCapabilities(false),
        server.WithInstructions("Query and manage data objects through natural language"),
        server.WithToolHandlerMiddleware(outputFilter),
    )
    // ... rest of setup
}
```

**Important:** The middleware function signature is `func(ToolHandlerFunc) ToolHandlerFunc` and it's added via `server.WithToolHandlerMiddleware()`. The existing project does NOT use this option yet, so adding it is purely additive — no refactoring of existing code needed.

**Confidence:** HIGH — Pattern documented in mcp-go docs (see `server/server.go:213-222` and `server/server.go:1169-1178`). Type signatures verified in source.

### Layer 4: Tool List Filtering (INT-02 supplement)

**Mechanism:** `server.WithToolFilter(filterFn)`

**Type signature (from mcp-go source `server/server.go:50`):**
```go
type ToolFilterFunc func(ctx context.Context, tools []mcp.Tool) []mcp.Tool
```

This adds a filter to `tools/list` responses. It receives the full tool list and returns a filtered subset. **It does NOT change tool names** — it can only include/exclude entire tools and the tool definition (name, description, input schema) is sent as-is.

**Important limitation:** `ToolFilterFunc` operates on the `mcp.Tool` struct directly. You CAN modify tool names here, but the internal handler map keys on the original name — so renaming in the filter doesn't change routing. If you rename in both filter AND registration, the filter must return names matching the registered names.

**Recommendation:** Do NOT use `ToolFilterFunc` for renaming. Do that at registration time. Only use `ToolFilterFunc` if you need conditional tool visibility (e.g., hide certain tools based on connection metadata in the future).

---

## 3. Alternative Approaches Evaluated

### Approach A: Per-Handler Filter (Rejected)

Creating a wrapper function that each handler calls instead of `NewToolResultJSON`:
- Pro: Type-safe, no JSON round-trip
- Con: Requires modifying 31 handlers + their interface methods
- Con: Couples filtering logic into every handler
- **Verdict:** Not worth the churn. Middleware approach is cleaner.

### Approach B: Hook-Based Interception (Rejected)

Using `Hooks.OnAfterCallTool` to observe/modify results:
- Pro: Available in mcp-go
- Con: `OnAfterCallTool` hook type receives `*mcp.CallToolResult` but the hooks are **observation-only** — they fire after the result is already constructed, and mutating the result inside a hook is not documented/supported
- Con: No way to abort or modify through the hook API (result is read-only by contract)
- **Verdict:** Hooks are for observability, not modification. Use middleware.

### Approach C: SetTools at Runtime (Not suitable)

Replacing all tools with renamed copies via `SetTools()`:
- Pro: Can be done after construction
- Con: Lose the original handler references
- Con: Race conditions if tools are called during replacement
- **Verdict:** Overengineered for a static configuration. Registration-time renaming is simpler.

---

## 4. Dependencies

| Dependency | Current | Needed | Status |
|---|---|---|---|
| `github.com/mark3labs/mcp-go` | v0.41.1 | v0.41.1 | ✅ No change needed |
| Go standard library | 1.23 | 1.23 | ✅ No change needed |
| External libs for JSON filtering | None | None | ✅ stdlib `encoding/json` is sufficient |

**No new dependencies.** The entire implementation uses:
- `github.com/mark3labs/mcp-go/mcp` — Tool types, CallToolResult, NewToolResult*
- `github.com/mark3labs/mcp-go/server` — ToolHandlerMiddleware, ToolFilterFunc, WithToolHandlerMiddleware
- `encoding/json` — JSON marshal/unmarshal for content filtering
- Existing `mcp` package interfaces and types

---

## 5. Version Compatibility with mcp-go v0.41.1

| Feature | Available Since | In v0.41.1? |
|---|---|---|
| `ToolHandlerMiddleware` type | v0.36.0+ | ✅ Confirmed (server.go:44) |
| `WithToolHandlerMiddleware` option | v0.36.0+ | ✅ Confirmed (server.go:215) |
| `WithRecovery()` recovery middleware | v0.36.0+ | ✅ Confirmed (server.go:267) |
| `WithToolFilter` | v0.38.0+ | ✅ Confirmed (server.go:256) |
| `ToolFilterFunc` type | v0.38.0+ | ✅ Confirmed (server.go:50) |
| `Hooks` system | v0.39.0+ | ✅ Confirmed (hooks.go:94) |
| `OnAfterCallTool` | v0.39.0+ | ✅ Confirmed (hooks.go:92) |
| `StructuredContent` on CallToolResult | v0.40.0+ | ✅ Confirmed (tools.go:46) |
| `SessionWithTools` interface | v0.39.0+ | ✅ Confirmed (server.go:1143) |

**No breaking changes** between current and needed APIs. The middleware chain system has been stable since v0.36.

---

## 6. Implementation Files

| File | Purpose |
|---|---|
| `internal/mcp/output_filter.go` | **New** — `ToolOutputFilter` interface, `NewOutputSanitizerMiddleware()`, `jsonContentFilter{}`, per-tool filter rules |
| `internal/mcp/server.go` | Modify `NewMCPServer()` args (name, instructions), add `WithToolHandlerMiddleware(outputFilter)` to server options |
| `internal/mcp/tools_*.go` (all 12) | Rename tool names and descriptions in each `register*Tools()` function |
| `internal/mcp/server_test.go` | Update `expectedTools` list to match new names |

---

## 7. Sources

- mcp-go v0.41.1 source code (verified in `/home/zzz/go/pkg/mod/github.com/mark3labs/mcp-go@v0.41.1/`):
  - `server/server.go:40-44` — ToolHandlerFunc and ToolHandlerMiddleware type definitions
  - `server/server.go:213-222` — WithToolHandlerMiddleware option implementation
  - `server/server.go:255-263` — WithToolFilter option implementation
  - `server/server.go:334-363` — NewMCPServer constructor (name, version, opts)
  - `server/server.go:527-530` — AddTool signature
  - `server/server.go:571-594` — AddTools / SetTools signatures
  - `server/server.go:697-705` — InitializeResult with ServerInfo name/version + Instructions
  - `server/server.go:1099-1106` — ToolFilterFunc application in handleListTools
  - `server/server.go:1132-1190` — handleToolCall with middleware chain application
  - `mcp/tools.go:40-51` — CallToolResult struct definition
  - `mcp/tools.go:557-574` — Tool struct definition
  - `mcp/tools.go:679-701` — NewTool() constructor
  - `mcp/utils.go:271-298` — NewToolResultText, NewToolResultJSON
  - `mcp/types.go:952-965` — Content interface, TextContent struct
  - `mcp/types.go:447-463` — InitializeResult with ServerInfo + Instructions
  - `server/hooks.go:88-92` — OnBeforeCallTool / OnAfterCallTool hook types
- Context7 query on mcp-go: `/mark3labs/mcp-go` confirmed ToolHandlerMiddleware and middleware patterns
