# Quickstart: Ontology v2 Productionization E2E

**Feature**: specs/002-ontology-v2-productionization
**Date**: 2026-06-02
**Prerequisites**: Docker, Go 1.23, PostgreSQL 16 (via Docker)

## Setup

### 1. Start PostgreSQL

```bash
make up              # docker compose up postgres
make migrate         # goose migrations up
```

### 2. Build and Run MCP Server

```bash
make build           # go build ./cmd/baxi-mcp
make mcp             # or: go run ./cmd/baxi-mcp
```

The server starts in stdio mode and registers all MCP tools.

### 3. Verify Server Startup

Look for these log lines (indicates v2 wiring successful):

```
INFO    Loaded 4 v2 ontology objects
INFO    Loaded 1 context recipes
INFO    Loaded 5 metric definitions
INFO    build_context service wired (non-nil)
INFO    LinkResolver wired with 4 objects
INFO    MCP server ready (31 tools)
```

If you see `WARN build_context service not wired (nil)` or `WARN LinkResolver not wired`, the productionization is incomplete.

---

## E2E Test: seller_late_delivery_alert

This walkthrough exercises the complete v2 pipeline for a seller with late delivery orders.

### Step 1: Describe Ontology

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "describe_ontology",
    "arguments": {}
  }
}
```

**Expected**: See `seller` object with `recent_orders` link (cardinality: one_to_many).

---

### Step 2: Get Seller Object

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "get_object",
    "arguments": {
      "object_type": "seller",
      "object_id": "SELLER_001"
    }
  }
}
```

**Expected**: Seller object with properties like `name`, `region`, `late_delivery_rate`.

---

### Step 3: Get Linked Orders (v2 One-to-Many)

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "get_linked_objects",
    "arguments": {
      "object_type": "seller",
      "object_id": "SELLER_001",
      "link_name": "recent_orders"
    }
  }
}
```

**Expected**: Array of order objects (not a single object). Example:

```json
{
  "objects": [
    {"id": "ORD_001", "status": "delivered", "delay_days": 3},
    {"id": "ORD_002", "status": "delivered", "delay_days": 5}
  ]
}
```

---

### Step 4: Build Context

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/call",
  "params": {
    "name": "build_context",
    "arguments": {
      "case_id": "CASE_001",
      "recipe_id": "seller_late_delivery_alert"
    }
  }
}
```

**Expected**: Complete `LLMSafeContextEnvelope` with:
- `context_hash`
- `evidence` (metrics + links)
- `object_context` (seller + computed metrics)
- `allowed_actions`
- `governance` (redaction rules)
- `redaction_summary`

---

### Step 5: Propose Action

```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "tools/call",
  "params": {
    "name": "propose_action",
    "arguments": {
      "object_type": "seller",
      "object_id": "SELLER_001",
      "action_type": "notify_owner",
      "params": {
        "message": "Your late delivery rate is 23%. Please review shipping process."
      }
    }
  }
}
```

**Expected**:

```json
{
  "proposal_id": "PROP_001",
  "status": "proposed",
  "case_id": "CASE_001"
}
```

**Critical**: Status must be `proposed`, NOT `approved`.

---

### Step 6: Approve Proposal

```json
{
  "jsonrpc": "2.0",
  "id": 6,
  "method": "tools/call",
  "params": {
    "name": "approve_proposal",
    "arguments": {
      "proposal_id": "PROP_001"
    }
  }
}
```

**Expected**:

```json
{
  "proposal_id": "PROP_001",
  "status": "approved"
}
```

---

### Step 7: Execute Proposal (Dry-Run)

```json
{
  "jsonrpc": "2.0",
  "id": 7,
  "method": "tools/call",
  "params": {
    "name": "execute_proposal",
    "arguments": {
      "proposal_id": "PROP_001",
      "dry_run": true
    }
  }
}
```

**Expected**:

```json
{
  "proposal_id": "PROP_001",
  "dry_run": true,
  "executed": true,
  "result": {
    "channel": "feishu",
    "message_id": "msg_sim_001",
    "status": "simulated"
  }
}
```

---

### Step 8: Verify Unapproved Execution is Rejected

Test that safety works by trying to execute without approval:

```json
{
  "jsonrpc": "2.0",
  "id": 8,
  "method": "tools/call",
  "params": {
    "name": "execute_action",
    "arguments": {
      "object_type": "seller",
      "object_id": "SELLER_001",
      "action_type": "notify_owner",
      "params": {"message": "test"},
      "dry_run": false
    }
  }
}
```

**Expected**:

```json
{
  "error": {
    "code": -32602,
    "message": "action requires approval: proposal must be approved before execution"
  }
}
```

---

## Troubleshooting

### `build_context service is not available`

- Check that `config/context_recipes.yml` exists and is valid YAML
- Check that `config/metric_definitions.yml` exists
- Verify v2 objects are loaded: look for `Loaded 4 v2 ontology objects` in logs

### `get_linked_objects` returns single object instead of array

- Verify `LinkResolver` is wired: check for `LinkResolver wired with N objects` in logs
- Check that `seller` object's `recent_orders` link has `cardinality: one_to_many` in `config/aip_object_schema_v2.yml`

### `propose_action` tool not found

- The tool must be registered in `internal/mcp/tools_action.go`
- Verify server was rebuilt after code changes

### `execute_action` still auto-approves

- Check that `cmd/baxi-mcp/main.go` `ontologyServiceAdapter.ExecuteAction` creates proposals with `apply_status = 'proposed'`
- Verify `WithDryRun(false)` has been changed to `WithDryRun(true)` as default

---

## Automated E2E Test

Run the Go integration test:

```bash
go test -tags=integration ./test/integration/... -run TestOntologyV2E2E -v
```

This test runs the full 8-step sequence above against a testcontainers PostgreSQL instance.
