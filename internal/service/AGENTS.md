# SERVICE: Business Orchestration Layer

**Generated:** 2026-05-28 15:45
**Commit:** d908f6d
**Branch:** main

## OVERVIEW

Business orchestration layer between HTTP handlers and repositories. 17 files, 8 orchestrator services, flat package.

## WHERE TO LOOK

| Service | File | Orchestrates |
|---------|------|-------------|
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

- **9 of 17 files import `api/dto`**: The business layer depends on API types (pipeline_service, status_service, diagnosis_service, qoder_service, outbox_service, alert_service, log_service, task_service, governance_service). This creates a reverse dependency — handlers should import service types, not the other way around. Fix: define request/response types in the service layer or use a shared model package.
- **Feishu service is 967 lines**: The largest file in `internal/`. Mixes HTTP client logic, CSV parsing, data sync, and YAML config reading. Should be split into client + sync + export sub-packages.
- **No `_test.go` for alert, governance, qoder, status, task services**: 5 of 13 production files lack test coverage.
