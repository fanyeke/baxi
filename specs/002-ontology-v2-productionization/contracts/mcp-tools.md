# MCP Tool Contract: Ontology v2 Productionization

**Feature**: specs/002-ontology-v2-productionization
**Date**: 2026-06-02
**Transport**: stdio (MCP protocol)
**Server**: baxi-mcp

## Overview

This contract defines the MCP tool interface changes required for the Ontology v2 Productionization phase. All tools use JSON-RPC 2.0 over stdio with the Model Context Protocol.

---

## Tool: `build_context`

**Status**: ✅ Wired
**File**: `internal/mcp/tools_context.go`, `cmd/baxi-mcp/main.go`

### Input

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `case_id` | string | Yes | Decision case identifier |
| `recipe_id` | string | No | Context recipe name; falls back to object default if omitted |

### Output

```json
{
  "context_hash": "sha256:abc123...",
  "evidence": [
    {"type": "metric", "name": "late_delivery_rate", "value": 0.23, "unit": "ratio"},
    {"type": "link", "name": "recent_orders", "count": 5}
  ],
  "object_context": {
    "type": "seller",
    "id": "seller_123",
    "properties": {"name": "Acme Corp", "region": "SP"},
    "metrics": {"late_delivery_rate": 0.23}
  },
  "allowed_actions": [
    {"type": "notify_owner", "description": "Send alert to seller", "params": {"message": "string"}}
  ],
  "governance": {
    "redacted_fields": ["email", "phone"],
    "required_fields_present": ["name", "region", "late_delivery_rate"],
    "policy_version": "v2.1"
  },
  "redaction_summary": ["email: PII policy", "phone: PII policy"]
}
```

### Error Responses

| Code | Message | When |
|------|---------|------|
| `-32602` | `build_context service is not available` | Server started without v2 objects or recipe loading failed |
| `-32602` | `case not found: {case_id}` | Invalid case identifier |
| `-32602` | `recipe not found: {recipe_id}` | Recipe name not in loaded recipes |

---

## Tool: `get_linked_objects`

**Status**: ✅ Wired with v2 LinkResolver + v1 fallback
**File**: `internal/mcp/tools_ontology.go`, `cmd/baxi-mcp/main.go`

### Input

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `object_type` | string | Yes | Source object type (e.g., `seller`) |
| `object_id` | string | Yes | Source object identifier |
| `link_name` | string | Yes | Relationship name (e.g., `recent_orders`) |

### Output (v2 — one_to_many)

```json
{
  "source": {"type": "seller", "id": "seller_123"},
  "link": {"name": "recent_orders", "cardinality": "one_to_many", "count": 5},
  "objects": [
    {"type": "order", "id": "ord_001", "properties": {"status": "delivered", "delay_days": 3}},
    {"type": "order", "id": "ord_002", "properties": {"status": "shipped", "delay_days": 0}}
  ]
}
```

### Output (v1 fallback — one_to_one)

```json
{
  "source": {"type": "seller", "id": "seller_123"},
  "link": {"name": "primary_region", "cardinality": "one_to_one"},
  "object": {"type": "region", "id": "SP", "properties": {"name": "São Paulo"}}
}
```

### Resolution Strategy

1. If `Server.linkResolver` is set and the object type has a v2 link definition for `link_name`:
   - Use v2 `LinkResolver` to compile and execute the link query
   - Return array for `one_to_many`, single object for `one_to_one`
2. Else:
   - Fall back to v1 Via-model lookup via `ontologySvc.GetLinkedObjects`
   - Always returns single object (v1 limitation)

---

## Tool: `propose_action` (NEW)

**Status**: ✅ Implemented
**File**: `internal/mcp/tools_action.go`

### Input

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `object_type` | string | Yes | Target object type |
| `object_id` | string | Yes | Target object identifier |
| `action_type` | string | Yes | Action type from registry (e.g., `notify_owner`) |
| `params` | object | Yes | Action parameters as JSON object |
| `case_id` | string | No | Associate with existing case; creates new case if omitted |
| `dry_run` | boolean | No | Validate only, do not create proposal (default: `false`) |

### Output

```json
{
  "proposal_id": "prop_abc123",
  "status": "proposed",
  "case_id": "case_def456",
  "action_type": "notify_owner",
  "valid": true,
  "validation_messages": []
}
```

### Error Responses

| Code | Message | When |
|------|---------|------|
| `-32602` | `action not bound to object type: {action_type}` | Action not in object's ActionBindings |
| `-32602` | `payload validation failed: {details}` | Params don't match action schema |
| `-32602` | `object not found: {object_type}/{object_id}` | Target object doesn't exist |

### Behavior

1. Validates `action_type` is bound to `object_type` via `ActionBindingValidator`
2. Validates `params` against action JSON schema
3. Creates or retrieves a decision case
4. Creates `ActionProposalRow` with `apply_status = 'proposed'`
5. Does NOT execute the action
6. Returns `proposal_id` for subsequent `approve_proposal` → `execute_proposal` flow

---

## Tool: `execute_action` (MODIFIED)

**Status**: ✅ Hardened (dry_run default + approval gate)
**File**: `internal/mcp/tools_ontology.go`

### Input

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `object_type` | string | Yes | Target object type |
| `object_id` | string | Yes | Target object identifier |
| `action_type` | string | Yes | Action type from registry |
| `params` | object | Yes | Action parameters |
| `dry_run` | boolean | No | **Default: `true`** (changed from previous behavior) |

### Output (dry_run = true)

```json
{
  "dry_run": true,
  "proposal_id": "prop_abc123",
  "status": "proposed",
  "action_type": "notify_owner",
  "would_execute": true,
  "validation_result": {"valid": true}
}
```

### Output (dry_run = false, requires_approval = true, unapproved)

```json
{
  "error": {
    "code": -32602,
    "message": "action requires approval: proposal must be approved before execution"
  }
}
```

### Behavioral Changes

| Before | After |
|--------|-------|
| Creates proposal with `approved` status | Creates proposal with `proposed` status |
| Default `dry_run = false` | Default `dry_run = true` |
| Auto-executes after creation | Returns proposal_id; execution requires separate `execute_proposal` |
| Bypasses approval workflow | Respects `requires_approval` flag |

---

## Tool: `execute_proposal` (UNCHANGED behavior, tightened enforcement)

**Status**: Already registered
**File**: `internal/mcp/tools_action.go`

### Input

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `proposal_id` | string | Yes | Proposal to execute |
| `dry_run` | boolean | No | Default: `true` |

### Enforcement Rules

| Proposal Status | requires_approval | Result |
|-----------------|-------------------|--------|
| `approved` | any | ✅ Execute (dry_run or real) |
| `proposed` | `false` + low risk | ⚠️ Currently auto-executes — **to be removed** |
| `proposed` | `true` | ❌ Error: "proposal must be approved" |
| `rejected` | any | ❌ Error: "proposal was rejected" |

**After productionization**: Only `approved` proposals may be executed. The risk-adaptive auto-execution of `proposed` low-risk proposals is removed.

---

## Tool: `describe_ontology`

**Status**: ✅ Already registered (v1 output; v2 enrichment deferred)
**File**: `internal/mcp/tools_ontology.go`

### Current Output

Returns v1-style descriptor with `Links` array containing `{Name, TargetType, Via}`.

### Required Enhancement

When v2 objects are present, include v2-specific fields:

```json
{
  "version": "v2",
  "object_types": [
    {
      "id": "seller",
      "display_name": "Seller",
      "properties": [...],
      "metrics": [...],
      "links": [
        {"name": "recent_orders", "target_type": "order", "cardinality": "one_to_many", "strategy": "reverse_lookup"}
      ],
      "context_recipes": ["seller_late_delivery_alert"],
      "action_bindings": ["notify_owner", "escalate_to_manager"]
    }
  ]
}
```

---

## Compatibility Notes

- v1 objects continue to work unchanged
- v2 objects require `build_context` service to be wired (non-nil)
- `get_linked_objects` for v2 objects returns arrays; v1 objects return single objects
- `execute_action` defaulting to `dry_run=true` is a **breaking change** for clients that relied on implicit real execution. Documented in migration notes.
- `propose_action` is additive — no breaking changes to existing tools
