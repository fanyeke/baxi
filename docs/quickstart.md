# Baxi Ontology v2 Quickstart

This guide covers the Ontology v2 productionized features available via the MCP server.

## Prerequisites

```bash
make up      # Start PostgreSQL
make migrate # Run goose migrations
make api     # Start API server (optional)
```

## MCP Server

Start the MCP server:

```bash
make mcp     # go run ./cmd/baxi-mcp
```

The server communicates via stdio using the Model Context Protocol.

## Available Tools

### 1. build_context

Build an LLM-safe context envelope for a decision case:

```json
{
  "tool": "build_context",
  "arguments": {
    "case_id": "CASE_001",
    "recipe_id": "seller_late_delivery_alert"
  }
}
```

**Response**:

```json
{
  "context_hash": "sha256:abc123...",
  "evidence": [
    {"type": "metric", "name": "late_delivery_rate", "value": 0.23}
  ],
  "object_context": {
    "type": "seller",
    "id": "SELLER_001",
    "properties": {"city": "Sao Paulo", "state": "SP"}
  },
  "allowed_actions": [{"type": "notify_owner", "params": {"message": "string"}}],
  "governance": {"redacted_fields": ["email", "phone"]},
  "redaction_summary": ["email: PII policy"]
}
```

### 2. get_linked_objects

Query linked objects with v2 relationship resolution:

```json
{
  "tool": "get_linked_objects",
  "arguments": {
    "object_type": "seller",
    "object_id": "SELLER_001",
    "link_name": "recent_orders"
  }
}
```

**Response** (v2 one_to_many):

```json
{
  "object_type": "seller",
  "object_id": "SELLER_001",
  "links": [
    {
      "link_name": "recent_orders",
      "target_type": "order",
      "objects": [
        {"object_type": "order", "object_id": "ORDER_001", "properties": {"status": "delivered"}},
        {"object_type": "order", "object_id": "ORDER_002", "properties": {"status": "shipped"}}
      ]
    }
  ]
}
```

The tool tries v2 `LinkResolver` first (SQL compilation + execution), falling back to v1 Via-model if v2 schema is unavailable.

### 3. propose_action

Create a proposal for approval (does not execute):

```json
{
  "tool": "propose_action",
  "arguments": {
    "object_type": "seller",
    "object_id": "SELLER_001",
    "action_type": "notify_owner",
    "params": "{\"message\": \"Late delivery alert\"}"
  }
}
```

**Response**:

```json
{
  "success": true,
  "proposal_id": "mcp-proposal-123456",
  "status": "proposed",
  "message": "Action \"notify_owner\" proposed on seller SELLER_001"
}
```

### 4. execute_action

Execute an action (defaults to dry_run):

```json
{
  "tool": "execute_action",
  "arguments": {
    "object_type": "seller",
    "object_id": "SELLER_001",
    "action_type": "notify_owner",
    "params": "{\"message\": \"Test\"}"
  }
}
```

**Response** (dry_run default):

```json
{
  "success": true,
  "action_type": "notify_owner",
  "dry_run": true,
  "result": {}
}
```

**Note**: `dry_run` defaults to `true`. To execute for real, first `propose_action`, then `approve_proposal`, then `execute_proposal`.

### 5. execute_proposal

Execute an approved proposal:

```json
{
  "tool": "execute_proposal",
  "arguments": {
    "proposal_id": "mcp-proposal-123456",
    "dry_run": true
  }
}
```

**Response**:

```json
{
  "proposal_id": "mcp-proposal-123456",
  "success": true,
  "dry_run": true
}
```

## Safety Model

| Operation | Requires Approval | Default dry_run |
|-----------|-------------------|-----------------|
| `propose_action` | N/A (creates proposal) | N/A |
| `execute_action` | Yes (implicit) | `true` |
| `execute_proposal` | Yes (must be approved) | `true` |

## Workflow Example

1. `build_context(CASE_001, seller_late_delivery_alert)` → Get context
2. `get_linked_objects(seller, SELLER_001, recent_orders)` → Get evidence
3. `propose_action(seller, SELLER_001, notify_owner)` → Create proposal
4. `approve_proposal(prop_id, reviewer_id)` → Approve (human or automated)
5. `execute_proposal(prop_id, dry_run=true)` → Simulate
6. `execute_proposal(prop_id, dry_run=false)` → Execute

## Configuration

V2 schema files (loaded at startup):

- `config/aip_object_schema_v2.yml` — V2 object types, properties, links
- `config/metric_definitions.yml` — Metric definitions
- `config/context_recipes.yml` — Context recipes for build_context

## Troubleshooting

| Issue | Cause | Fix |
|-------|-------|-----|
| `build_context unavailable` | Recipe/metric loading failed | Check config YAML syntax and paths |
| `link not found` | V2 schema missing link definition | Update `aip_object_schema_v2.yml` |
| `action not allowed` | Action not bound to object type | Check `AllowedActions` in schema or action registry |
| `proposal must be approved` | Tried to execute without approval | Use `approve_proposal` first |
