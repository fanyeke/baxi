# Architecture Research

**Domain:** MCP Server Information Containment (Go, stdio transport, mcp-go v0.41.1)
**Researched:** 2026-06-06
**Confidence:** HIGH

## System Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│   MCP Client (Pi Agent / Claude Desktop / Any MCP Client)           │
│   ┌───────────────────────────────────────────────────────────────┐ │
│   │  Sees: generic server name, business-oriented tool names,      │ │
│   │  filtered output (no non-LLMReadable props, no table_counts)   │ │
│   └───────────────────────────────────────────────────────────────┘ │
└──────────────────────────┬──────────────────────────────────────────┘
                           │ stdio
                           ▼
┌──────────────────────────────────────────────────────────────────────┐
│  server.ServeStdio(s.server)  ← mcp-go stdio transport layer         │
│                                                                       │
│  Dispatch by tool name:                                               │
│    ├─ New name (business-oriented) → handler                         │
│    └─ Old name (legacy alias, if enabled) → same handler             │
└──────────────────────────┬───────────────────────────────────────────┘
                           ▼
┌──────────────────────────────────────────────────────────────────────┐
│  MCP Tool Handlers  (12 registerXxxTools() groups)                   │
│                                                                       │
│  1. Parse arguments from req.GetArguments()                           │
│  2. Call service interface (unchanged business logic)                 │
│  3. Build result as map[string]interface{}                            │
│  4. ★ Apply output_filter.go functions                               │
│     ├── FilterProperties()       — strips !LLMReadable props         │
│     ├── FilterOntologyDescriptor — strips Source + Governance fields │
│     ├── FilterSystemStatus()     — strips table_counts               │
│     └── FilterSearchResult()     — caps limit + filters items        │
│  5. mcp.NewToolResultJSON(filteredMap) → return                      │
└──────────────────────────┬───────────────────────────────────────────┘
                           │
                           ▼
┌──────────────────────────────────────────────────────────────────────┐
│  Service Layer  (cmd/baxi-mcp/main.go adapters → internal/services)  │
│                                                                       │
│  Unchanged — all filtering happens at the handler layer               │
└──────────────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | File(s) |
|-----------|---------------|---------|
| `output_filter.go` | Centralized property/status/ontology filtering functions | **NEW** |
| `tool_names.go` | Tool name constants + old→new mapping table | **NEW** |
| `server_identity.go` | Server name/instructions from env vars with generic defaults | **NEW** |
| `Server` (modified) | Accept `enableLegacyTools` field, use identity helpers | `server.go` |
| `register*Tools()` (modified) | Use name constants + register legacy aliases | `tools_*.go` |
| `server_test.go` (modified) | Updated tool name list, filter unit tests | `server_test.go` |
| `cmd/baxi-mcp/main.go` | Unchanged — no interface changes needed | `cmd/baxi-mcp/main.go` |

## Executive Summary

The Baxi MCP server currently leaks project architecture information in three dimensions: **server identity** (name + instructions reveal the product), **tool naming** (names map directly to internal package structure), and **output content** (descriptions, properties, and status data expose schema internals and sensitive fields). The existing `LLMReadable` and `Sensitivity` markers in the v2 ontology schema are already loaded into `Server.objectTypesV2` but are **not used** by the `GetObject`/`GetLinkedObjects` handlers.

This architecture proposes a minimal, non-invasive layering:

1. **output_filter.go** — centralized filter functions handlers call before building JSON responses
2. **tool_names.go** — constants for new abstracted tool names + old→new mapping table
3. **server_identity.go** — env-var-driven server name/instructions with generic defaults
4. **Per-handler modifications** — minimal surgical changes at JSON-construction time

No new external dependencies. No changes to mcp-go framework. All changes contained within `internal/mcp/`.

---

## Question 1: Where Should the Output Filtering Layer Sit?

### Options Considered

| Option | Mechanism | Pros | Cons |
|--------|-----------|------|------|
| **A: Per-handler inline filtering** | Each handler manually checks LLMReadable | Simple to understand | Duplicated logic, easy to miss |
| **B: Centralized middleware** | Intercept `CallToolResult` before returning | Clean separation | **Impossible** — mcp-go v0.41.1 has no post-processing pipeline for tool results |
| **C: Centralized filter functions** + handler calls them | Standalone `FilterProperties()`, `FilterSystemStatus()` etc. in one file, called by each handler | Single source of truth, testable, minimal handler changes | Each handler must remember to call the filter |
| **D: At the service adapter layer** | Filter in `ontologyServiceAdapter.GetObject()` etc. | Catches all callers | Wrong layer — adapters are in `cmd/baxi-mcp/main.go`, not `internal/mcp/`. Would apply filtering to HTTP API too. |

### Recommendation: **Option C** — Centralized filter functions

**Why:**
- mcp-go `server.AddTool()` takes `(Tool, ToolHandlerFunc)` with no chain/wrapper mechanism — middleware impossible
- A handler-wrapper pattern (`withResultFilter`) is possible but cannot cleanly post-process `mcp.CallToolResult.Content` after JSON construction
- The simplest correct approach: filter the `map[string]interface{}` **before** passing to `mcp.NewToolResultJSON()`
- All filter logic lives in one file → one place to audit, one place to test

**What goes in output_filter.go:**
```go
// output_filter.go — package mcp

// FilterProperties removes properties where LLMReadable=false.
// Uses s.objectTypesV2 to look up property metadata.
func (s *Server) FilterProperties(objectType string, props map[string]interface{}) {
    ot, ok := s.objectTypesV2[objectType]
    if !ok {
        return // no v2 schema — pass through (or strip all for safety?)
    }
    for key := range props {
        prop, ok := ot.Properties[key]
        if ok && !prop.LLMReadable {
            delete(props, key)
        }
    }
}

// FilterOntologyDescriptor removes internal schema details from ontology output.
// Strips: SourceDescriptor, Governance policy, internal field names.
func FilterOntologyDescriptor(desc *OntologyDescriptor) {
    for i := range desc.ObjectTypes {
        // Source reveals physical schema layout
        desc.ObjectTypes[i].Source = nil
        // Governance reveals internal policy structure
        desc.ObjectTypes[i].Governance = nil
        // Optional: strip metrics list if it reveals internal metric names
        desc.ObjectTypes[i].Metrics = nil
    }
}

// FilterSystemStatus removes table_counts and masks internal identifiers.
func FilterSystemStatus(result map[string]interface{}) {
    // Remove table_counts (reveals schema.table layout)
    delete(result, "table_counts")
    // Mask pipeline_run internal IDs
    if pr, ok := result["pipeline_run"].(map[string]interface{}); ok {
        delete(pr, "run_id")
        delete(pr, "run_type")
        delete(pr, "mode")
    }
    // Keep: alert_count, top-level status summary only
}

// FilterSearchResult caps result size and strips internal fields.
func (s *Server) FilterSearchResult(objectType string, result map[string]interface{}) {
    // Enforce max limit
    // Filter properties per object type
    if items, ok := result["items"].([]map[string]interface{}); ok {
        for _, item := range items {
            s.FilterProperties(objectType, item)
        }
    }
}
```

### Handler Integration Pattern

Every handler follows this pattern:
```go
func (s *Server) handleGetObject(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // ... existing argument parsing ...
    
    obj, err := s.ontologySvc.GetObject(ctx, objectType, objectID)
    // ... error handling ...
    
    result := map[string]interface{}{
        "object_type": obj.ObjectType,
        "object_id":   obj.ObjectID,
        "properties":  obj.Properties,
    }
    
    // ★ NEW: Filter non-LLMReadable properties
    s.FilterProperties(objectType, result["properties"].(map[string]interface{}))
    
    return mcp.NewToolResultJSON(result)
}
```

---

## Question 2: Handler-Level vs Centralized Middleware

### Verdict: Handler-level (by necessity)

**The constraint:** `mcp-go` v0.41.1 does not expose middleware hooks for tool call results. The server's `AddTool()` signature is:
```go
func (s *MCPServer) AddTool(tool Tool, handler ToolHandlerFunc)
```

`ToolHandlerFunc` is `func(ctx context.Context, req CallToolRequest) (*CallToolResult, error)`. There is no chain/wrapper mechanism on the server side. The response is returned directly to the transport layer (stdio).

**What doesn't work:**
- A wrapper that post-processes `CallToolResult` — impossible because `CallToolResult.Content` is `[]Content` which is an interface, and `mcp.NewToolResultJSON()` constructs concrete content before returning
- A transport-level interceptor — stdio is synchronous, no hook for response modification

**What works (and is already the pattern):**
- Handlers construct `map[string]interface{}` → filter it → pass to `mcp.NewToolResultJSON()`
- This is explicit, testable, and requires no framework changes

**Q: Can we abstract the pattern to reduce repetition?**
Yes — but only the filter-function call, not a wrapper:

```go
// In output_filter.go:

// safeResult applies standard output filtering and wraps in CallToolResult.
// Use this instead of mcp.NewToolResultJSON for ontology-related responses.
func (s *Server) safeResult(data map[string]interface{}) (*mcp.CallToolResult, error) {
    return mcp.NewToolResultJSON(data)
}
```

This is intentionally thin. The actual filtering is explicit in each handler because different handlers need different filters. A generic wrapper would either over-filter (strip things some handlers need) or under-filter (miss handler-specific concerns).

**Rule of thumb:** If 3+ handlers need the same filter, extract a helper function. If only 1-2 handlers need it, keep it explicit.

---

## Question 3: Leveraging Existing LLMReadable/Sensitivity Markers

### Current State

The v2 ontology schema (`ObjectPropertyV2`) has:
```go
type ObjectPropertyV2 struct {
    // ...
    Sensitivity string // L0 (public), L1 (internal), L2 (confidential), L3 (restricted)
    LLMReadable bool   // whether LLM may read this in context
    // ...
    Availability string // real, virtual, planned
}
```

The `Server` struct already has:
```go
objectTypesV2 map[string]*ontology.ObjectTypeV2  // populated via SetObjectTypesV2()
```

The `DescribeOntology` adapter **already filters** by `!prop.LLMReadable`:
```go
// In ontologyServiceAdapter.DescribeOntology():
for _, prop := range ot.Properties {
    if !prop.LLMReadable {
        continue  // ← Already done!
    }
    otDesc.Properties = append(otDesc.Properties, mcp.PropertyDescriptor{...})
}
```

**But:** `GetObject`, `GetLinkedObjects`, and `search_objects` handlers **do NOT filter** — they pass through all properties.

### Integration Points

| Handler | Current Behavior | Fix | Marker Used |
|---------|-----------------|-----|-------------|
| `handleGetObject` | Returns all object properties | Call `FilterProperties(objectType, props)` before JSON | `LLMReadable` |
| `handleGetLinkedObjects` | Returns all properties on linked objects | Apply filter per link result entry | `LLMReadable` |
| `handleSearchObjects` | Returns all search result items unfiltered | Apply filter per item | `LLMReadable` |
| `handleDescribeOntology` | Already filtered ✅ | No change needed | N/A |
| `handleBuildContext` | Returns envelope with object_context | Filter per the recipe's spec (handled by ContextRecipe engine) | N/A |

### Sensitivity-based Filtering (Future)

For now, use only `LLMReadable`. `Sensitivity` (L0–L3) can be layered in later for role-based filtering:

```go
// In output_filter.go (future extension):
func (s *Server) FilterPropertiesByRole(objectType string, props map[string]interface{}, role string) {
    // Map role to max sensitivity level
    maxLevel := sensitivityForRole(role) // admin→L3, operator→L2, viewer→L1
    ot := s.objectTypesV2[objectType]
    for key := range props {
        prop := ot.Properties[key]
        if !prop.LLMReadable || sensitivityLevel(prop.Sensitivity) > maxLevel {
            delete(props, key)
        }
    }
}
```

**MVP scope:** LLMReadable filtering only. Sensitivity filtering is deferred — not needed for information containment, only for multi-role isolation which is out of scope.

---

## Question 4: Tool Name Abstraction

### Options Considered

| Option | Pros | Cons |
|--------|------|------|
| **A: Constants per file** (`tool_names.go`) | Single source of truth, easy to audit, testable mapping | Need to update 12 registration functions |
| **B: Inline in each registration function** | No new file | Names scattered, hard to audit for leaks |
| **C: Runtime name mapping table** | Can switch at runtime | Indirection, harder to trace in code |

### Recommendation: **Option A** — Constants file + registration use

### File: `tool_names.go`

```go
// ──── MCP Tool Name Constants ─────────────────────────────────────────────
// These replace internal-package-derived names with business-oriented names.
// The Pi Agent integration will continue to work via legacy tool aliases
// (controlled by MCP_ENABLE_LEGACY_TOOLS).

package mcp

// Decision & Assessment tools
const (
    ToolAssessAlert      = "assess_alert"       // was: create_decision_case
    ToolEvaluate         = "evaluate"            // was: decide
    ToolResolve          = "resolve"             // was: resolve_case
    ToolListAssessments  = "list_assessments"    // was: list_cases
    ToolGetAssessment    = "get_assessment"      // was: get_case
    ToolListProposals    = "list_proposals"      // was: list_proposals — unchanged, generic enough
)

// Monitoring tools
const (
    ToolListAlerts       = "list_alerts"         // was: list_alerts — unchanged
    ToolListEvents       = "list_events"         // was: list_outbox_events
)

// Compliance tools
const (
    ToolCheckAccess      = "check_access"        // was: check_access — unchanged
    ToolGetClassification = "get_classification"  // was: get_classification — unchanged
)

// Analysis tools
const (
    ToolRunAnalysis      = "run_analysis"        // was: run_pipeline
    ToolAnalysisStatus   = "analysis_status"     // was: get_pipeline_status
)

// Review & Approval tools
const (
    ToolApproveProposal  = "approve_proposal"    // was: approve_proposal — unchanged
    ToolRejectProposal   = "reject_proposal"     // was: reject_proposal — unchanged
    ToolCancelProposal   = "cancel_proposal"     // was: cancel_proposal — unchanged
    ToolGetProposal      = "get_proposal"        // was: get_proposal_by_id
    ToolGetContext       = "get_context"         // was: get_decision_context
    ToolListApprovals    = "list_approvals"      // was: list_review_records
    ToolExecuteProposal  = "execute_proposal"    // was: execute_proposal — unchanged
    ToolProposeAction    = "propose_action"      // was: propose_action — unchanged
)

// Data tools
const (
    ToolDescribeSchema   = "describe_schema"     // was: describe_ontology
    ToolGetRecord        = "get_record"          // was: get_object
    ToolGetRelated       = "get_related"         // was: get_linked_objects
    ToolExecuteAction    = "execute_action"      // was: execute_action — unchanged
    ToolSearch           = "search"              // was: search_objects
)

// Workspace tools
const (
    ToolCreateWorkspace  = "create_workspace"    // was: create_sandbox
    ToolAddToWorkspace   = "add_to_workspace"    // was: add_to_sandbox
    ToolCompareWorkspaces = "compare_workspaces"  // was: compare_sandboxes
    ToolGetWorkspace     = "get_workspace"       // was: get_sandbox
)

// Schema tools
const (
    ToolListActionTypes  = "list_action_types"   // was: list_action_schemas
    ToolGetActionType    = "get_action_type"     // was: get_action_schema
)

// Context tools
const (
    ToolBuildContext     = "build_context"       // was: build_context — unchanged
)

// System tools
const (
    ToolSystemInfo       = "system_info"         // was: get_system_status
)
```

### Backward Compat Mapping

```go
// LegacyToolMap maps old tool names to new constants (for alias registration).
var LegacyToolMap = map[string]string{
    "create_decision_case": ToolAssessAlert,
    "decide":              ToolEvaluate,
    "resolve_case":        ToolResolve,
    "list_cases":          ToolListAssessments,
    "get_case":            ToolGetAssessment,
    "run_pipeline":        ToolRunAnalysis,
    "get_pipeline_status": ToolAnalysisStatus,
    "get_decision_context": ToolGetContext,
    "get_proposal_by_id":  ToolGetProposal,
    "list_review_records": ToolListApprovals,
    "describe_ontology":   ToolDescribeSchema,
    "get_object":          ToolGetRecord,
    "get_linked_objects":  ToolGetRelated,
    "search_objects":      ToolSearch,
    "create_sandbox":      ToolCreateWorkspace,
    "add_to_sandbox":      ToolAddToWorkspace,
    "compare_sandboxes":   ToolCompareWorkspaces,
    "get_sandbox":         ToolGetWorkspace,
    "list_action_schemas": ToolListActionTypes,
    "get_action_schema":   ToolGetActionType,
    "list_outbox_events":  ToolListEvents,
    "get_system_status":   ToolSystemInfo,
}
```

### Registration Approach

Each `registerXxxTools()` function uses the constants:

```go
func (s *Server) registerOntologyTools() {
    // Primary: new name
    tool := mcp.NewTool(ToolDescribeSchema, ...)
    s.server.AddTool(tool, s.handleDescribeOntology)
    
    // Alias: old name (if legacy mode enabled)
    if s.enableLegacyTools {
        oldTool := mcp.NewTool("describe_ontology", ...)
        s.server.AddTool(oldTool, s.handleDescribeOntology)
    }
}
```

This avoids code duplication: the same handler function runs for both names.

---

## Question 5: Server Identity Configuration

### Recommendation: Env Var Driven with Generic Defaults

**Current (leaking):**
```go
s := server.NewMCPServer(
    "Baxi MCP Server",                                         // ← Leaks project name
    "1.0.0",
    server.WithToolCapabilities(false),
    server.WithInstructions("E-commerce governance and decision platform"), // ← Leaks domain
)
```

**Proposed (generic):**
```go
s := server.NewMCPServer(
    getServerName(),     // env: MCP_SERVER_NAME,  default: "Data Processing Server"
    "1.0.0",
    server.WithToolCapabilities(false),
    server.WithInstructions(getServerInstructions()), // env: MCP_SERVER_INSTRUCTIONS, default: "Platform for data processing and decision management"
)
```

### Config helper functions

```go
// server_identity.go — package mcp

func getServerName() string {
    if v := os.Getenv("MCP_SERVER_NAME"); v != "" {
        return v
    }
    return "Data Processing Server"
}

func getServerVersion() string {
    if v := os.Getenv("MCP_SERVER_VERSION"); v != "" {
        return v
    }
    return "1.0.0"
}

func getServerInstructions() string {
    if v := os.Getenv("MCP_SERVER_INSTRUCTIONS"); v != "" {
        return v
    }
    return "Platform for data processing and decision management"
}
```

### Env Vars Added

| Env Var | Purpose | Default |
|---------|---------|---------|
| `MCP_SERVER_NAME` | Server identity announced to MCP client | `Data Processing Server` |
| `MCP_SERVER_VERSION` | Server version string | `1.0.0` |
| `MCP_SERVER_INSTRUCTIONS` | Instructions sent to MCP client | `Platform for data processing and decision management` |
| `MCP_ENABLE_LEGACY_TOOLS` | If `true`, register old tool names as aliases | `true` (safe default during transition) |

These go in `internal/config/config.go` alongside existing env vars (or simply use `os.Getenv` in the MCP package — they're MCP-specific, not general config).

---

## Question 6: Backward Compatibility for Pi Agent

### Strategy: Dual Registration with Runtime Control

**The constraint:** Pi Agent references specific tool names. If we change names, Pi Agent integration breaks.

**Solution:** Register tools under both old and new names, controlled by `MCP_ENABLE_LEGACY_TOOLS=true` (default).

```
MCP_ENABLE_LEGACY_TOOLS=true  → Both names registered (current behavior preserved)
MCP_ENABLE_LEGACY_TOOLS=false → Only new names (clean break, future state)
```

### How Dual Registration Works

```go
// In registerXxxTools() functions:
func (s *Server) registerDecisionTools() {
    // 1. Define the handler and new-name tool
    handler := s.handleAssessAlert
    newTool := mcp.NewTool(ToolAssessAlert, /* params */)
    s.server.AddTool(newTool, handler)
    
    // 2. Alias old name if legacy mode is on
    s.registerLegacyAlias("create_decision_case", ToolAssessAlert, handler)
}

func (s *Server) registerLegacyAlias(oldName, newName string, handler toolHandler) {
    if !s.enableLegacyTools {
        return
    }
    // Create a tool with the old name pointing to same handler
    oldTool := mcp.NewTool(oldName, /* need descriptions */)
    // Problem: tool parameter descriptions differ between old and new...
}
```

**Wait — there's a complication.** Tool definitions include parameter names and descriptions. If the old `create_decision_case` had `alert_id` as a parameter, the alias needs the same parameters for the MCP client to understand them.

**Resolution:** The alias tool **keeps the same parameter names** as the old tool. The handler internally maps parameters if needed:

```go
func (s *Server) registerLegacyAlias(oldName, newName string, makeTool func() mcp.Tool, handler toolHandler) {
    if !s.enableLegacyTools {
        return
    }
    // Registers under old name with old-style params
    s.server.AddTool(makeTool(), handler)
}
```

This means each `registerXxxTools()` creates two tools when legacy mode is on:
1. New-name tool with new parameter schema (obscured names)
2. Old-name tool with old parameter schema (original names, same handler)

### Pi Agent Compatibility Guarantee

| Aspect | Behavior | Risk |
|--------|----------|------|
| Tool names | Both old and new names work | None (dual registration) |
| Tool parameters | Old names → old schema, new names → new schema | Low (some parameters may be removed/consolidated) |
| Return format | Same JSON structure (filtered) | **Medium** — `table_counts` removed from `system_info` |
| Server identity | Generic (no longer "Baxi MCP Server") | Low — Pi Agent likely doesn't check server metadata |

**Deployment transition:**
1. Phase 1: Ship with `MCP_ENABLE_LEGACY_TOOLS=true` (default). Pi Agent continues working unmodified.
2. Phase 2: Verify Pi Agent works with new tool names. Update Pi Agent config to use new names.
3. Phase 3: Set `MCP_ENABLE_LEGACY_TOOLS=false` in deployment. Remove legacy registration code in next milestone.

---

## File Change Summary

### New Files (in `internal/mcp/`)

| File | Responsibility | Approx Lines |
|------|---------------|-------------|
| `tool_names.go` | Tool name constants + `LegacyToolMap` + tool group metadata | 120 |
| `output_filter.go` | `FilterProperties()`, `FilterOntologyDescriptor()`, `FilterSystemStatus()`, `FilterSearchResult()` | 150 |
| `server_identity.go` | `getServerName()`, `getServerInstructions()`, env var helpers | 60 |

### Modified Files

| File | Change | Rationale |
|------|--------|-----------|
| `server.go` | Use identity helpers in `NewServer()`. Add `enableLegacyTools` field. Add `registerLegacyAlias()` helper. | Core wiring change |
| `tools_decision.go` | Replace literal tool names with constants + legacy alias registration | Abstract tool names |
| `tools_alert.go` | Same pattern | Abstract tool names |
| `tools_governance.go` | Same pattern | Abstract tool names |
| `tools_pipeline.go` | Same pattern + parameter hardening (config allowlist) | Abstract + security |
| `tools_review.go` | Same pattern | Abstract tool names |
| `tools_action.go` | Same pattern | Abstract tool names |
| `tools_outbox.go` | Same pattern | Abstract tool names |
| `tools_context.go` | Same pattern | Abstract tool names |
| `tools_status.go` | Same pattern + `FilterSystemStatus()` + `FilterSearchResult()` call | Abstract + output filtering |
| `tools_ontology.go` | Same pattern + `FilterProperties()` + `FilterOntologyDescriptor()` call | Abstract + output filtering |
| `tools_sandbox.go` | Same pattern | Abstract tool names |
| `tools_schema.go` | Same pattern | Abstract tool names |
| `server_test.go` | Update `expectedTools` slice to new names. Add test for legacy aliases. Add test for `FilterProperties()`. | Tests must match reality |
| `interfaces.go` | Remove `SourceDescriptor` from `ObjectTypeDescriptor` (or add JSON `omit:"always"` | Prevent accidental re-introduction |

### Files NOT Modified

| File | Reason |
|------|--------|
| `cmd/baxi-mcp/main.go` | No interface changes. Adapters continue to work unchanged. |
| `internal/ontology/*` | Ontology types stay as-is. Filtering uses existing markers. |
| `go.mod` / `go.sum` | No new dependencies. |

---

## Data Flow (After Changes)

```
MCP Client (Pi Agent)
  │
  ▼
server.ServeStdio(s.server)        ← mcp-go transport layer
  │
  ├─ Tool dispatch by name
  │   ├─ New name → handler
  │   └─ Old name (legacy alias) → same handler
  │
  ▼
Handler function
  │
  ├─ Parse arguments
  ├─ Call service (unchanged business logic)
  ├─ Build result map
  ├─ ★ Call output_filter.go function(s)
  │   ├─ FilterProperties() — strips !LLMReadable fields
  │   ├─ FilterOntologyDescriptor() — strips Source/Gov
  │   ├─ FilterSystemStatus() — strips table_counts
  │   └─ FilterSearchResult() — caps limit + filters items
  └─ mcp.NewToolResultJSON(filteredMap)
       │
       ▼
  stdio transport → MCP Client
```

---

## Anti-Patterns to Avoid

### 1. Duplicating property metadata lookup
**Bad:** Each handler re-implements the LLMReadable check.
**Good:** One `FilterProperties()` method on Server, uses `s.objectTypesV2` which is already loaded.

### 2. Silent filtering without logging
**Bad:** Properties silently disappear — debugging becomes confusing.
**Good:** Log at debug level when properties are filtered:
```go
if ok && !prop.LLMReadable {
    s.log.Debug("filtered non-LLMReadable property",
        zap.String("object_type", objectType),
        zap.String("property", key))
    delete(props, key)
}
```

### 3. Mixing tool name strings and constants
**Bad:** Some registration functions use constants, others use literal strings.
**Good:** All registration functions use constants from `tool_names.go`. No exceptions.

### 4. Parameter name leakage in legacy aliases
**Bad:** Legacy tool uses `object_type` parameter (reveals internal terminology).
**Acceptable for now:** Legacy aliases keep old parameter names to maintain Pi Agent compatibility. Only new-name tools get renamed parameters.

### 5. Over-filtering
**Bad:** Stripping data that the Pi Agent genuinely needs to function (e.g., the `case_id` in responses).
**Good:** Filter only data that reveals internal architecture or is marked non-LLMReadable. Test with real Pi Agent interactions.

---

## Sources

- [Context7: mcp-go v0.41.1 Server API](https://github.com/mark3labs/mcp-go) — confirmed `AddTool(tool, handler)` has no middleware chain (HIGH confidence, code inspection)
- Codebase: `internal/mcp/server.go:77-82` — current server identity strings reveal project name and domain (HIGH confidence)
- Codebase: `internal/mcp/interfaces.go:105-109` — `SourceDescriptor` struct exists but never populated (HIGH confidence)
- Codebase: `internal/ontology/schema_v2.go:41-54` — `ObjectPropertyV2` has `LLMReadable` and `Sensitivity` markers (HIGH confidence)
- Codebase: `cmd/baxi-mcp/main.go:652-726` — `DescribeOntology` already filters by `!prop.LLMReadable` (HIGH confidence)
- Codebase: `internal/mcp/tools_ontology.go:66-96` — `handleGetObject` passes through all properties without filtering (HIGH confidence)
- Codebase: `internal/mcp/server_test.go:420-454` — hardcoded tool name list must be updated (HIGH confidence)
