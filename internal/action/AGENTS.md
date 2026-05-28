# action: Action Registry & Execution

**Branch:** main

## OVERVIEW
Action whitelist, proposal lifecycle, and execution dispatch. 13 files.

## WHERE TO LOOK

| Task | File | Notes |
|------|------|-------|
| Add an action type | `registry.go` + `config/action_registry.yml` | Must add to both CanonicalActions and YAML |
| Execute a proposal | `apply_service.go` | Supports dry-run, approval checks, channel dispatch |
| Create a proposal | `proposal_service.go` | Creates ActionProposal with payload |
| Channel routing | `adapter.go` | Maps action types to Feishu/GitHub channels |
| Outbox integration | `outbox_integration.go` | Writes outbox events for delayed dispatch |
| Record outcome | `outcome_service.go` | Persists execution results |
| Action contract | `contract.go` | ActionContract type used by LLM and adapter layers |
| Run executor | `executor.go` | ActionExecutor interface (single Execute method) |

## KEY PATTERNS

- **Whitelist enforcement**: `CanonicalActions` in registry.go is the hard-coded allow list (4 types: create_followup_task, notify_owner, export_report, create_outbox_message)
- **YAML config states**: No file → all canonical allowed; empty `actions:` → none allowed; partial → only those in whitelist
- **Dry-run safety**: Action execution defaults to dry-run=true unless explicitly set to apply
- **Approval gates**: `RequiresApproval` on ActionConfig checked before apply_service executes
- **Status flow**: draft → pending_review → approved → executing → completed
- **LLM visibility**: ActionConfig.LLMVisible controls which actions appear in LLM context (GetLLMVisibleActions)

## ANTI-PATTERNS

- Pool passed as parameter throughout (not yet migrated to PoolProvider)
- apply_service.go imports decision package via interface adapter to avoid circular dependency
- Payload schema validation uses raw YAML map traversal, no reflection-based schema engine
