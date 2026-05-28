# model: Shared Domain Types

**Branch:** main

## OVERVIEW
Shared domain types used by Service layer (NOT by API DTOs). Eliminates the reverse dependency from service → api/dto. 10 files, 549 lines.

## WHERE TO LOOK

| File | Types | Used By |
|------|-------|---------|
| `task.go` | Task, TaskFilters, TaskListResponse | task_service, alert_service |
| `alert.go` | Alert, AlertFilters, AlertListResponse | alert_service |
| `governance.go` | GovernanceStatusResponse, ClassificationResponse, FieldMarking | governance_service |
| `outbox.go` | OutboxEvent, OutboxFilters | outbox_service |
| `logs.go` | LogItem, LogListResponse | log_service |
| `status.go` | StatusResponse | status_service |
| `qoder.go` | CapabilitiesResponse, ContextResponse | qoder_service |
| `diagnosis.go` | DiagnosisResponse | diagnosis_service |
| `pipeline.go` | PipelinePreview, PipelineInfo | pipeline_service |
| `constants.go` | Priority*, Status*, Severity*, AlertStatus* constants | All services |

## KEY PATTERNS

- **No JSON tags**: Model types are for Go code, not serialization. JSON tags are in api/dto
- **Handler conversion**: Each handler converts model → dto before JSON serialization
- **Zero dependencies**: Model package imports only stdlib (time)
- **Service boundary**: Services return model types; handlers convert to dto types

## ANTI-PATTERNS

- qoder.go is 226 lines (40% of model package) — consider splitting into qoder_types.go + qoder_params.go
- constants.go could be extended — currently only covers priorities/statuses/severities
