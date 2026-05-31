# SERVICE: Business Orchestration Layer

**Generated:** 2026-05-28 15:45
**Commit:** d908f6d
**Branch:** main

## OVERVIEW

Business orchestration layer between HTTP handlers and repositories. 11 files, 11 orchestrator services, flat package.

Services exposed via 31 MCP tools across 11 domains. See `internal/mcp/` for the full tool-to-service mapping (tools_decision.go: 6 tools, tools_review.go: 5, tools_alert.go: 1, tools_governance.go: 2, tools_pipeline.go: 1, tools_action.go: 2, tools_outbox.go: 2, tools_status.go: 2, tools_ontology.go: 4, tools_sandbox.go: 4, tools_schema.go: 2).

## WHERE TO LOOK

| Service | File | Orchestrates |
|---------|------|-------------|
| Agent Log | `agent_log_service.go` | Agent execution log queries, MCP audit trail |
| Feishu sync | `feishu_service.go` | REST calls to Feishu Open API, CSV export, YAML config sync |
| Decision | `decision_service.go` | Case engine, context builder, proposals, LLM provider |
| Pipeline | `pipeline_service.go` | Pipeline runner, step orchestration, outbox creation |
| Log | `log_service.go` | Activity log queries, aggregation, DTO projection |
| Diagnosis | `diagnosis_service.go` | Root cause analysis, metric drill-down |
| Alert | `alert_service.go` | Alert rule engine, dimensional alert checks |
| Governance | `governance_service.go` | Policy enforcement, classification, lineage queries |
| Task/Outbox/Status/Qoder | corresponding files | Background task dispatch, outbox polling, status checks, Qoder integration |

Each service typically has a `*_test.go` counterpart using the local interface pattern.

## CONVENTIONS

- **Local interfaces**: Each service defines narrow dependency interfaces (e.g. `CaseService`, `DecisionEngine`) at the top of the file, not shared across services.
- **Struct composition**: `DecisionService` composes sub-services (`caseSvc`, `ctxBuilder`, `engine`, `proposalSvc`) rather than inheriting them.
- **Feishu service** is the exception: a standalone HTTP client with its own retry logic, no repository wiring.

## ANTI-PATTERNS

- **Feishu service is 621 lines**: The largest file in `internal/`. Mixes HTTP client logic, CSV parsing, data sync, and YAML config reading. Should be split into client + sync + export sub-packages.
- **Only `qoder_service.go` lacks a dedicated `_test.go`**: 10 of 11 production files have test coverage; qoder_service.go (385 lines) has no unit tests despite substantial logic.

### ✅ Resolved Anti-Patterns

- **`api/dto` dependency in services**: Previously 9 of 17 files imported `api/dto`, creating reverse dependency from business layer to API types. Now all services use `internal/model` package — handlers import service types correctly.
