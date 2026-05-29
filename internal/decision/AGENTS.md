# decision: Decision Engine & Case Management

**Branch:** main

## OVERVIEW
LLM-driven decision engine with rule-based fallback, case lifecycle, and lineage tracking. 15 files.

Exposed via MCP tools: `create_decision_case`, `decide`, `list_cases`, `get_case`, `list_proposals`. See `internal/mcp/tools_decision.go` for handler implementations.

## WHERE TO LOOK

| Task | File | Notes |
|------|------|-------|
| Decision generation | `engine.go` | Primary LLM → validate → repair → fallback chain |
| Case CRUD | `case_service.go` | Create/get/list decision cases |
| Context assembly | `context_builder_v2.go` | Builds LLMSafeContext from alerts, governance, ontology |
| Context assembly (legacy) | `context_builder.go` | Original v1 builder, may be deprecated |
| Context switch | `switchable_context_builder.go` | Switches between v1 and v2 builders |
| Lineage tracking | `lineage_service.go` | Records all decision events for audit |
| Lineage adapter | `lineage_adapter.go` | Bridges lineage events to outbox dispatch |
| Snapshots | `snapshot_recorder.go` | Persists LLM input/output/validation for replay |
| ID generation | `idgen.go` | Generates decision IDs |

## KEY PATTERNS

- **Provider chain**: OpenAI LLM → validation → repair retry → rule-based fallback (4-phase)
- **Snapshot types**: LLMSafeContext, LLMRawOutput, LLMParsedOutput, LLMValidation — all persisted per decision
- **Lineage events**: Every state transition recorded (requested, generated, validated, failed, fallback, repair_attempted, repair_succeeded, repair_failed)
- **Local interfaces**: CaseService, ContextBuilder, DecisionEngine defined as local interfaces in service/decision_service.go
- **Repair mechanism**: On validation failure, single repair retry with same provider before falling back

## ANTI-PATTERNS

- Pool passed as parameter throughout all services
- DecisionHandler has 5 stubs returning 501 Not Implemented (DecideLLM, Compare, Replay, ListLLMDecisions, ListEvals)
- V1/V2 context builders coexist — v1 may be dead code
