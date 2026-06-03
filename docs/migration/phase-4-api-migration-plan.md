# Phase 4: API Migration Plan (FastAPI → Go API)

## 1. Overview

This document describes how the old Python FastAPI gateway (24 endpoints across 11 router files) maps to the new Go API server. The goal is **response parity**: given the same authorization and request parameters, the Go API must return identical response shapes and near-identical values.

### Guiding Principle

This is a parity migration, not a redesign. The Go API reproduces the FastAPI endpoint behavior faithfully, including response structure, field naming, and query parameter semantics. Any deviation must be documented as an explainable difference.

### Baseline Reference

- **FastAPI freeze tag**: `v0.5.3-python-sqlite-freeze`
- **Migration branch**: `migration/go-postgres`
- **API response snapshots**: `migration_baseline/api_responses/` (removed — migration complete)
- **Blueprint reference**: `docs/API_REFERENCE.md`

---

## 2. Old FastAPI Endpoint Inventory (24 Endpoints)

The existing FastAPI application (`api/main.py`) mounts 11 routers under `/api/v1`. All authenticated endpoints use Bearer token auth. The health endpoint is the sole public endpoint.

### Router Mounting (from `api/main.py`)

```python
app.include_router(health.router, prefix="/api/v1", tags=["Health"])
app.include_router(status.router, prefix="/api/v1", tags=["Status"])
app.include_router(alerts.router, prefix="/api/v1", tags=["Alerts"])
app.include_router(tasks.router, prefix="/api/v1", tags=["Tasks"])
app.include_router(outbox.router, prefix="/api/v1", tags=["Outbox"])
app.include_router(logs.router, prefix="/api/v1", tags=["Logs"])
app.include_router(feishu.router, prefix="/api/v1", tags=["Feishu"])
app.include_router(pipeline.router, prefix="/api/v1", tags=["Pipeline"])
app.include_router(diagnosis.router, prefix="/api/v1", tags=["Diagnosis"])
app.include_router(governance.router, prefix="/api/v1", tags=["Governance"])
app.include_router(qoder.router, prefix="/api/v1", tags=["Qoder"])
```

### Complete Endpoint List

| # | Router File | Method | Path | Auth | Data Source |
|---|---|---|---|---|---|
| 1 | `health.py` | GET | `/api/v1/health` | No | SQLite `db_exists()` |
| 2 | `status.py` | GET | `/api/v1/status` | Yes | SQLite `pipeline_runs`, all table counts |
| 3 | `alerts.py` | GET | `/api/v1/alerts` | Yes | SQLite `alert_events` |
| 4 | `tasks.py` | GET | `/api/v1/tasks` | Yes | SQLite `action_tasks` |
| 5 | `outbox.py` | GET | `/api/v1/outbox` | Yes | SQLite `event_outbox` |
| 6 | `outbox.py` | POST | `/api/v1/outbox/dispatch` | Yes | SQLite `event_outbox` + adapters |
| 7 | `logs.py` | GET | `/api/v1/logs/recent` | Yes | JSONL file `logs/api/api.log` |
| 8 | `logs.py` | GET | `/api/v1/logs/errors` | Yes | JSONL file `logs/api/error.log` |
| 9 | `logs.py` | GET | `/api/v1/logs/audit` | Yes | CSV file `data/system/api_audit_dispatch.csv` |
| 10 | `logs.py` (in `diagnosis.py`) | GET | `/api/v1/logs/diagnosis` | Yes | Aggregated from error.log + audit CSVs |
| 11 | `feishu.py` | POST | `/api/v1/feishu/export` | Yes | SQLite + Feishu API |
| 12 | `feishu.py` | POST | `/api/v1/feishu/sync` | Yes | SQLite + Feishu API |
| 13 | `feishu.py` | POST | `/api/v1/feishu/status/import` | Yes | Feishu API → SQLite |
| 14 | `pipeline.py` | POST | `/api/v1/pipeline/run` | Yes | Pipeline preview config |
| 15 | `governance.py` | GET | `/api/v1/governance/catalog` | Yes | YAML file `config/data_catalog.yml` |
| 16 | `governance.py` | GET | `/api/v1/governance/classification` | Yes | YAML file `config/data_classification.yml` |
| 17 | `governance.py` | GET | `/api/v1/governance/markings` | Yes | YAML file `config/data_markings.yml` |
| 18 | `governance.py` | GET | `/api/v1/governance/lineage` | Yes | YAML file `config/data_lineage.yml` |
| 19 | `governance.py` | GET | `/api/v1/governance/checkpoints` | Yes | YAML file `config/checkpoint_rules.yml` |
| 20 | `governance.py` | GET | `/api/v1/governance/health` | Yes | YAML file `config/health_checks.yml` |
| 21 | `governance.py` | GET | `/api/v1/governance/status` | Yes | All 9 governance YAML files |
| 22 | `qoder.py` | GET | `/api/v1/qoder/capabilities` | Yes | YAML `config/qoder_capabilities.yml` |
| 23 | `qoder.py` | GET | `/api/v1/qoder/context` | Yes | SQLite `alert_events`, `action_tasks`, `event_outbox`, `pipeline_runs` |
| 24 | `qoder.py` | POST | `/api/v1/qoder/reports` | Yes | SQLite `qoder_runs` + `qoder_reports` |

---

## 3. Phase 4 Migration Scope (11 Endpoints IN)

Phase 4 migrates all **read-only GET** endpoints. Write endpoints and governance config-reader endpoints are deferred.

### IN Scope: 11 Read-Only GET Endpoints

| # | Method | Path | Old Router | Go Handler Package |
|---|---|---|---|---|
| P01 | GET | `/api/v1/health` | `health.py` | `internal/api/handler/health.go` |
| P02 | GET | `/api/v1/status` | `status.py` | `internal/api/handler/status.go` |
| P03 | GET | `/api/v1/alerts` | `alerts.py` | `internal/api/handler/alert.go` |
| P04 | GET | `/api/v1/tasks` | `tasks.py` | `internal/api/handler/task.go` |
| P05 | GET | `/api/v1/outbox` | `outbox.py` | `internal/api/handler/outbox.go` |
| P06 | GET | `/api/v1/logs/recent` | `logs.py` | `internal/api/handler/log.go` |
| P07 | GET | `/api/v1/logs/errors` | `logs.py` | `internal/api/handler/log.go` |
| P08 | GET | `/api/v1/logs/audit` | `logs.py` | `internal/api/handler/log.go` |
| P09 | GET | `/api/v1/governance/status` | `governance.py` | `internal/api/handler/governance.go` |
| P10 | GET | `/api/v1/qoder/capabilities` | `qoder.py` | `internal/api/handler/qoder.go` |
| P11 | GET | `/api/v1/qoder/context` | `qoder.py` | `internal/api/handler/qoder.go` |

### Query Parameter Summary

Each migrated endpoint supports the same query parameters as the original FastAPI:

| Endpoint | Parameters | Defaults |
|---|---|---|
| `/alerts` | `status`, `severity`, `object_type`, `object_id`, `limit` | limit=100 |
| `/tasks` | `status`, `priority`, `owner_role`, `limit` | limit=100 |
| `/outbox` | `status`, `channel`, `limit` | limit=100 |
| `/logs/recent` | `limit` | limit=50 |
| `/logs/errors` | `request_id`, `error_code`, `limit` | limit=100 |
| `/logs/audit` | `outbox_id`, `status`, `limit`, `source` | limit=100, source=dispatch |
| `/qoder/context` | `severity`, `limit_alerts`, `limit_tasks`, `limit_outbox`, `include_logs` | limit_alerts=10, limit_tasks=10, limit_outbox=10, include_logs=false |

---

## 4. Staged Migration Scope (13 Endpoints OUT)

These endpoints are structurally complete in the baseline but are **out of scope** for Phase 4:

| # | Method | Path | Reason for Deferral |
|---|---|---|---|
| S01 | POST | `/api/v1/outbox/dispatch` | Write operation; requires Phase 6 Outbox Worker design |
| S02 | POST | `/api/v1/feishu/export` | Write operation; requires Feishu adapter config in Go |
| S03 | POST | `/api/v1/feishu/sync` | Write operation; requires Feishu SDK port |
| S04 | POST | `/api/v1/feishu/status/import` | Write operation; requires Feishu SDK port |
| S05 | POST | `/api/v1/pipeline/run` | Write operation; requires pipeline orchestration |
| S06 | GET | `/api/v1/governance/catalog` | Config file reader; Governance Runtime deferred |
| S07 | GET | `/api/v1/governance/classification` | Config file reader; Governance Runtime deferred |
| S08 | GET | `/api/v1/governance/markings` | Config file reader; Governance Runtime deferred |
| S09 | GET | `/api/v1/governance/lineage` | Config file reader; Governance Runtime deferred |
| S10 | GET | `/api/v1/governance/checkpoints` | Config file reader; Governance Runtime deferred |
| S11 | GET | `/api/v1/governance/health` | Config file reader; Governance Runtime deferred |
| S12 | POST | `/api/v1/qoder/reports` | Write operation; requires LLM integration design |
| S13 | GET | `/api/v1/logs/diagnosis` | Cross-source aggregation; depends on logs migration |

### Summary Table

| Category | IN | OUT | Total |
|---|---|---|---|
| Health & Status | 2 | 0 | 2 |
| Alerts | 1 | 0 | 1 |
| Tasks | 1 | 0 | 1 |
| Outbox | 1 | 1 | 2 |
| Logs | 3 | 1 | 4 |
| Feishu | 0 | 3 | 3 |
| Pipeline | 0 | 1 | 1 |
| Governance | 1 | 6 | 7 |
| Qoder | 2 | 1 | 3 |
| **Total** | **11** | **13** | **24** |

---

## 5. Go API Layered Architecture

The Go API follows a strict layered architecture. Each layer has a single responsibility and depends only on the layer below it.

```
┌─────────────────────────────────────────────────────────────────────┐
│                     cmd/baxi-api/main.go                            │
│                     HTTP server entry point                         │
│                     Router registration, middleware, config         │
└───────────────────────┬─────────────────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────────────────┐
│                 internal/api/router.go                              │
│                     Route → handler binding                         │
│                     Middleware pipeline (auth, CORS, request_id)    │
└───────────────────────┬─────────────────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────────────────┐
│            internal/api/handler/  (HTTP layer)                      │
│                                                                     │
│  health.go   status.go   alert.go   task.go   outbox.go            │
│  log.go      governance.go   qoder.go                               │
│                                                                     │
│  Responsibilities:                                                  │
│  - Parse HTTP request (path params, query params, headers)          │
│  - Call service layer                                               │
│  - Serialize response (JSON)                                        │
│  - Handle HTTP-specific errors (400, 401, 403, 404, 500)            │
│  - NEVER contains business logic or SQL                              │
└───────────────────────┬─────────────────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────────────────┐
│            internal/service/  (Business logic layer)                 │
│                                                                     │
│  alert_service.go    task_service.go    outbox_service.go           │
│  log_service.go      governance_service.go  qoder_service.go       │
│  status_service.go                                                  │
│                                                                     │
│  Responsibilities:                                                  │
│  - Orchestrate multi-repository reads                               │
│  - Apply filtering, sorting, pagination                             │
│  - Map repository DTOs → API response DTOs                         │
│  - Business validation                                              │
│  - NEVER contains SQL                                                │
└───────────────────────┬─────────────────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────────────────┐
│            internal/repository/  (Data access layer)                │
│                                                                     │
│  alert_repo.go      task_repo.go      outbox_repo.go               │
│  log_repo.go        governance_repo.go  qoder_repo.go             │
│  status_repo.go     pipeline_run_repo.go                           │
│                                                                     │
│  Responsibilities:                                                  │
│  - Execute SQL queries against PostgreSQL                           │
│  - Map database rows → internal DTOs                                │
│  - Connection pooling via pgx pool                                  │
|  - NEVER contains business logic                                    │
|  - NEVER leaks *sql.Rows to callers                                 │
└───────────────────────┬─────────────────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────────────────┐
│            internal/api/dto/  (Data Transfer Objects)               │
│                                                                     │
│  health.go   status.go   alert.go   task.go   outbox.go            │
│  log.go      governance.go   qoder.go                               │
│  error.go                                                           │
│                                                                     │
│  Responsibilities:                                                  │
│  - Define JSON-serializable response structs                        │
│  - Define request structs (query params)                           │
│  - JSON tags matching FastAPI field names (snake_case)              │
│  - Response compatibility annotations                               │
└─────────────────────────────────────────────────────────────────────┘
```

### Package Dependency Rule

```
handler → service → repository → PostgreSQL
     ↘         ↘          ↘
      dto       dto        dto (internal)
```

- `handler/` imports `service/` and `dto/`
- `service/` imports `repository/` and `dto/`
- `repository/` imports `dto/` and `pgx` pool
- **No circular dependencies** between service packages

### Request Lifecycle

```
HTTP Request
    │
    ▼
router.go (middleware chain: request_id → auth → CORS → rate_limit)
    │
    ▼
handler/xxx.go (parse query params, validate)
    │
    ▼
service/xxx_service.go (apply business rules, orchestrate)
    │
    ▼
repository/xxx_repo.go (execute SQL, map rows)
    │
    ▼
PostgreSQL
    │
    ▼ (reverse path)
repository → service → handler → JSON Response
```

---

## 6. Endpoint → PostgreSQL Source Table Mapping

Each Phase 4 endpoint reads from specific PostgreSQL schemas and tables. The `ops.*`, `audit.*`, `mart.*`, `dwd.*`, and `gov.*` schemas replace the old SQLite tables.

### Mapping Table

| # | Endpoint | Old SQLite Table(s) | New PostgreSQL Table(s) | Schema |
|---|---|---|---|---|
| P01 | `/health` | SQLite file existence check | PostgreSQL `pg_stat_activity` (connection check) | system |
| P02 | `/status` | `pipeline_runs`, all 16 tables | `audit.pipeline_run`, `audit.pipeline_step_run`, `dwd.*`, `mart.*`, `ops.*` | audit, dwd, mart, ops |
| P03 | `/alerts` | `alert_events` | `ops.metric_alert` | ops |
| P04 | `/tasks` | `action_tasks` | `ops.task`, `ops.recommendation`, `ops.metric_alert` | ops |
| P05 | `/outbox` | `event_outbox` | `ops.outbox_event`, `ops.dispatch_attempt` | ops |
| P06 | `/logs/recent` | JSONL `logs/api/api.log` | `audit.api_request_log` | audit |
| P07 | `/logs/errors` | JSONL `logs/api/error.log` | `audit.error_log` | audit |
| P08 | `/logs/audit` | CSV `data/system/api_audit_dispatch.csv` | `audit.audit_log` | audit |
| P09 | `/governance/status` | 9 YAML config files | `gov.config_snapshot`, `gov.object_schema`, `gov.data_classification`, `gov.data_lineage`, `gov.access_policy`, `gov.health_check_result` | gov |
| P10 | `/qoder/capabilities` | YAML `config/qoder_capabilities.yml` | Static read-only capability matrix (in-code) | — |
| P11 | `/qoder/context` | `alert_events`, `action_tasks`, `event_outbox`, `pipeline_runs` | `ops.metric_alert`, `ops.task`, `ops.outbox_event`, `audit.pipeline_run`, `audit.error_log`, `ai.qoder_runs`, `ai.qoder_reports` | ops, audit, ai |

### Detailed Endpoint Data Sources

#### P02: GET /status

The status endpoint aggregates table row counts and the last pipeline run:

```sql
-- Table counts (aggregated across schemas)
SELECT 'dwd.order_level' AS table_name, COUNT(*) AS row_count FROM dwd.order_level
UNION ALL SELECT 'dwd.item_level', COUNT(*) FROM dwd.item_level
UNION ALL SELECT 'mart.metric_daily', COUNT(*) FROM mart.metric_daily
UNION ALL SELECT 'mart.metric_dimension_daily', COUNT(*) FROM mart.metric_dimension_daily
UNION ALL SELECT 'ops.metric_alert', COUNT(*) FROM ops.metric_alert
UNION ALL SELECT 'ops.recommendation', COUNT(*) FROM ops.recommendation
UNION ALL SELECT 'ops.task', COUNT(*) FROM ops.task
UNION ALL SELECT 'ops.outbox_event', COUNT(*) FROM ops.outbox_event
UNION ALL SELECT 'audit.pipeline_run', COUNT(*) FROM audit.pipeline_run
UNION ALL SELECT 'audit.pipeline_step_run', COUNT(*) FROM audit.pipeline_step_run
UNION ALL SELECT 'audit.ingestion_batch', COUNT(*) FROM audit.ingestion_batch;

-- Last pipeline run
SELECT run_id, run_type, mode, status, started_at, finished_at,
       input_count, output_count, error_message
FROM audit.pipeline_run
ORDER BY started_at DESC LIMIT 1;
```

#### P03: GET /alerts

```sql
SELECT alert_id, rule_id, event_date, severity, metric_name,
       object_type, object_id, current_value, baseline_value,
       change_rate, owner_role, status, impact_score
FROM ops.metric_alert
WHERE (status = $1 OR $1 IS NULL)
  AND (severity = $2 OR $2 IS NULL)
  AND (object_type = $3 OR $3 IS NULL)
  AND (object_id = $4 OR $4 IS NULL)
ORDER BY event_date DESC
LIMIT $5;
```

Note: The old `alert_events.event_id` maps to `ops.metric_alert.alert_id`.

#### P04: GET /tasks

```sql
SELECT task_id, task_title, task_description, status, priority,
       owner_role, owner_user_id, due_at, created_at, completed_at,
       feedback, recommendation_id, alert_id AS event_id,
       target_object_type, target_object_id
FROM ops.task
WHERE (status = $1 OR $1 IS NULL)
  AND (priority = $2 OR $2 IS NULL)
  AND (owner_role = $3 OR $3 IS NULL)
ORDER BY created_at DESC
LIMIT $4;
```

Note: The old `action_tasks.event_id` maps to `ops.task.alert_id`.

#### P05: GET /outbox

```sql
SELECT event_id AS outbox_id, event_type, source_type, source_id,
       target_channel, status, created_at, dispatch_attempts,
       last_dispatch_at
FROM ops.outbox_event
WHERE (status = $1 OR $1 IS NULL)
  AND (target_channel = $2 OR $2 IS NULL)
ORDER BY created_at DESC
LIMIT $3;
```

Note: The old `event_outbox.outbox_id` maps to `ops.outbox_event.event_id`.

#### P06-P08: Logs Endpoints

The logs endpoints transition from JSONL/CSV files to database tables:

| Endpoint | Old Source | New Source | Query Pattern |
|---|---|---|---|
| `/logs/recent` | `logs/api/api.log` (JSONL tail-read) | `audit.api_request_log` | `SELECT ... ORDER BY ts DESC LIMIT $1` |
| `/logs/errors` | `logs/api/error.log` (JSONL tail-read) | `audit.error_log` | `SELECT ... WHERE (request_id=$1 OR $1 IS NULL) AND (error_code=$2 OR $2 IS NULL) ORDER BY ts DESC LIMIT $3` |
| `/logs/audit` | `data/system/api_audit_dispatch.csv` (CSV) | `audit.audit_log` | `SELECT ... WHERE source='dispatch' AND (outbox_id=$1 OR $1 IS NULL) AND (status=$2 OR $2 IS NULL) ORDER BY ts DESC LIMIT $3` |

#### P09: GET /governance/status

The governance status endpoint currently loads 9 YAML config files. In the Go implementation, these configs are managed in the `gov.*` PostgreSQL tables:

```sql
-- Check health status of governance configs
SELECT config_name, status, loaded_at, error_message
FROM gov.config_snapshot
ORDER BY config_name;
```

Config files mapped:
| YAML File | gov Table Reference |
|---|---|
| `data_catalog.yml` | `gov.object_schema` |
| `data_classification.yml` | `gov.data_classification` |
| `data_markings.yml` | `gov.data_classification` (markings sub-type) |
| `data_lineage.yml` | `gov.data_lineage` |
| `checkpoint_rules.yml` | `gov.governance_checkpoint` |
| `retention_policies.yml` | `gov.object_schema` (retention field) |
| `health_checks.yml` | `gov.health_check_result` |
| `decision_eval_rules.yml` | `gov.access_policy` |
| `access_policy.yml` | `gov.access_policy` |

#### P10: GET /qoder/capabilities

This endpoint returns a static read-only capability matrix. Instead of reading a YAML file each time, the Go API embeds the capability configuration:

```go
// internal/api/handler/qoder.go
var defaultCapabilities = dto.CapabilitiesResponse{
    Mode:                  "read_only",
    Version:               "1.0.0",
    CanReadStatus:         true,
    CanReadAlerts:         true,
    CanReadTasks:          true,
    CanReadOutbox:         true,
    CanReadLogs:           true,
    CanDispatch:           false,
    CanApply:              false,
    CanSyncFeishu:         false,
    CanRunPipeline:        false,
    CanModifyRules:        false,
    AllowedEndpoints:      []string{"/api/v1/alerts", "/api/v1/tasks",
                                     "/api/v1/outbox", "/api/v1/status",
                                     "/api/v1/logs/recent", "/api/v1/logs/errors",
                                     "/api/v1/logs/audit", "/api/v1/governance/status",
                                     "/api/v1/qoder/capabilities", "/api/v1/qoder/context"},
    ForbiddenActions:      []string{"direct_sql", "run_shell", "edit_files",
                                     "apply_dispatch", "feishu_sync_apply",
                                     "pipeline_run_apply", "modify_config"},
}
```

#### P11: GET /qoder/context

This endpoint aggregates context from multiple sources:

```sql
-- Last pipeline run (from audit.pipeline_run)
SELECT ... FROM audit.pipeline_run ORDER BY started_at DESC LIMIT 1;

-- Active alerts (from ops.metric_alert)
SELECT ... FROM ops.metric_alert WHERE status = 'new' [AND severity = $1]
ORDER BY event_date DESC LIMIT $2;

-- Open tasks (from ops.task)
SELECT ... FROM ops.task WHERE status IN ('todo', 'in_progress')
ORDER BY created_at DESC LIMIT $3;

-- Pending outbox (from ops.outbox_event)
SELECT ... FROM ops.outbox_event WHERE status = 'pending'
ORDER BY created_at DESC LIMIT $4;

-- Recent errors (from audit.error_log, conditional)
SELECT ... FROM audit.error_log ORDER BY ts DESC LIMIT 5;
```

---

## 7. Response Compatibility Strategy

### Field Name Mapping (Old → New)

The old SQLite tables use different column names from the new PostgreSQL tables. The Go repository layer maps PostgreSQL columns to field names that match the FastAPI response schema.

| FastAPI Response Field | Old SQLite Column | New PostgreSQL Column | Go DTO Field |
|---|---|---|---|
| `event_id` | `alert_events.event_id` | `ops.metric_alert.alert_id` | `AlertItem.EventID` |
| `outbox_id` | `event_outbox.outbox_id` | `ops.outbox_event.event_id` | `OutboxItem.OutboxID` |
| `event_id` (in TaskItem) | `action_tasks.event_id` | `ops.task.alert_id` | `TaskItem.EventID` |
| `dispatch_attempts` | `event_outbox.dispatch_attempts` | `ops.outbox_event.dispatch_attempts` | `OutboxItem.DispatchAttempts` |
| `total` | `COUNT(*)` | `COUNT(*)` | Same |
| `items` | List of rows | List of rows | Same |

### Response Shape Compatibility

Every Go API response must have the same JSON shape as the FastAPI baseline. The `dto/` package explicitly maps to the baseline:

```go
// dto/alert.go
type AlertItem struct {
    EventID       string   `json:"event_id"`
    RuleID        string   `json:"rule_id"`
    EventDate     string   `json:"event_date"`
    Severity      string   `json:"severity"`
    MetricName    string   `json:"metric_name"`
    ObjectType    string   `json:"object_type"`
    ObjectID      string   `json:"object_id"`
    CurrentValue  *float64 `json:"current_value"`
    BaselineValue *float64 `json:"baseline_value"`
    ChangeRate    *float64 `json:"change_rate"`
    OwnerRole     string   `json:"owner_role"`
    Status        string   `json:"status"`
    ImpactScore   *float64 `json:"impact_score"`
}

type AlertListResponse struct {
    Items []AlertItem `json:"items"`
    Total int         `json:"total"`
}
```

### Explainable Differences

| Field | Reason for Difference | Mitigation |
|---|---|---|
| `version` | Go API version string differs from FastAPI `0.5.3` | Update to `1.0.0` in Go response |
| `database.path` | PostgreSQL connection string replaces SQLite file path | Replace with `{"driver": "postgres", "host": "...", "database": "baxi"}` |
| `database.tables` | Table names differ (SQLite vs PostgreSQL schema) | Map table names to equivalent PostgreSQL tables |
| Timestamp formats | SQLite stores ISO strings, PostgreSQL `TIMESTAMPTZ` formats may differ | Ensure `time.RFC3339` serialization |
| `dispatch_attempts` | `int` vs `int64` | No semantic difference; both serialize as JSON number |
| `null` vs omitted fields | Go `*float64` marshals as `null` (matching Python `None`) | Use pointers for nullable fields |
| `created_at` / `updated_at` | Pipeline run timestamps differ between runs | Exclude from comparison or expect delta |

---

## 8. Auth Strategy

### Bearer Token (API_BEARER_TOKEN)

The Go API reuses the same authentication mechanism as FastAPI: Bearer token via the `API_BEARER_TOKEN` environment variable.

**Token Verification (constant-time):**

```go
// internal/api/auth.go
import "crypto/subtle"

func verifyToken(provided string) bool {
    expected := os.Getenv("API_BEARER_TOKEN")
    if expected == "" || len(expected) < 32 {
        log.Warn("API_BEARER_TOKEN not set or too short")
        return false
    }
    return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}
```

**Auth Middleware:**

```go
// internal/api/router.go
func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Skip auth for health endpoint (public)
        if r.URL.Path == "/api/v1/health" {
            next.ServeHTTP(w, r)
            return
        }

        auth := r.Header.Get("Authorization")
        if auth == "" {
            writeError(w, http.StatusUnauthorized, "AUTH_REQUIRED",
                "Authorization header is required", ...)
            return
        }

        if !strings.HasPrefix(auth, "Bearer ") {
            writeError(w, http.StatusUnauthorized, "AUTH_REQUIRED",
                "Invalid Authorization header format", ...)
            return
        }

        token := strings.TrimPrefix(auth, "Bearer ")
        if !verifyToken(token) {
            writeError(w, http.StatusForbidden, "INVALID_TOKEN",
                "Invalid or expired token", ...)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

**Actor name extraction:** The FastAPI `get_current_user()` returns the `DEFAULT_USER` env var (default `"qoder"`). The Go API should do the same:

```go
actor := os.Getenv("DEFAULT_USER")
if actor == "" {
    actor = "api_user"
}
```

### Auth Endpoint List

| Endpoint | Auth Required | Notes |
|---|---|---|
| `GET /api/v1/health` | **No** | Public endpoint for load balancer health checks |
| All other Phase 4 endpoints | **Yes** | Bearer token in `Authorization` header |

---

## 9. Request ID Strategy

### Source

The request ID is extracted from the `X-Request-ID` HTTP header if present. If absent, the Go API generates a new UUID v4.

```go
// internal/api/middleware.go
func requestIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        rid := r.Header.Get("X-Request-ID")
        if rid == "" {
            rid = uuid.New().String()
        }
        ctx := context.WithValue(r.Context(), ctxKeyRequestID, rid)
        w.Header().Set("X-Request-ID", rid)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### Response Inclusion

Every response (success or error) includes the `request_id` field. The request ID is also:

1. Returned in the `X-Request-ID` response header
2. Included in structured log entries for tracing
3. Included in error response bodies for diagnosis

### Error Response Format (includes request_id)

```json
{
  "request_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "error_code": "VALIDATION_ERROR",
  "message": "limit must be between 1 and 1000",
  "diagnosis": "Invalid value for 'limit'",
  "suggested_action": "Check the request parameters against the API schema"
}
```

---

## 10. Unified Error Response Format

All error responses follow a consistent 5-field JSON structure (matching the existing FastAPI `ErrorResponse` model):

```json
{
  "request_id": "<uuid>",
  "error_code": "<machine_readable_code>",
  "message": "<human_readable_summary>",
  "diagnosis": "<technical_cause_explanation>",
  "suggested_action": "<remediation_guidance>"
}
```

### Error Codes

| HTTP Status | error_code | When |
|---|---|---|
| 400 | `VALIDATION_ERROR` | Invalid query parameter values |
| 401 | `AUTH_REQUIRED` | Missing Authorization header |
| 403 | `INVALID_TOKEN` | Token does not match API_BEARER_TOKEN |
| 404 | `NOT_FOUND` | Resource not found |
| 429 | `RATE_LIMITED` | Rate limit exceeded |
| 500 | `INTERNAL_ERROR` | Unexpected server error |
| 503 | `DB_UNAVAILABLE` | Database connection failed |

### Go Implementation

```go
// internal/api/dto/error.go
type ErrorResponse struct {
    RequestID      string `json:"request_id"`
    ErrorCode      string `json:"error_code"`
    Message        string `json:"message"`
    Diagnosis      string `json:"diagnosis"`
    SuggestedAction string `json:"suggested_action"`
}

func writeError(w http.ResponseWriter, status int, code, message, diagnosis, action string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(ErrorResponse{
        RequestID:       getRequestID(r.Context()), // from context
        ErrorCode:       code,
        Message:         message,
        Diagnosis:       diagnosis,
        SuggestedAction: action,
    })
}
```

---

## 11. Pagination, Filtering, and Sorting Strategy

### Pagination (Limit-Offset)

All list endpoints support limit-offset pagination matching the FastAPI behavior:

| Parameter | Type | Default | Min | Max | Description |
|---|---|---|---|---|---|
| `limit` | integer | 100 | 1 | 1000 | Max items to return |
| `offset` | integer | 0 | 0 | — | Number of items to skip |

Exception: Log endpoints (`/logs/recent`, `/logs/errors`, `/logs/audit`) have a max limit of 500 (matching FastAPI).

### Implementation

```go
type PaginationParams struct {
    Limit  int `json:"limit"`  // default: 100, max: 1000
    Offset int `json:"offset"` // default: 0
}

func (p *PaginationParams) Validate() error {
    if p.Limit < 1 {
        p.Limit = 100
    }
    if p.Limit > 1000 {
        return fmt.Errorf("limit must not exceed 1000")
    }
    if p.Offset < 0 {
        p.Offset = 0
    }
    return nil
}
```

### Repository Implementation

```go
// internal/repository/alert_repo.go
func (r *AlertRepository) List(ctx context.Context, filters AlertFilters, pagination PaginationParams) ([]dto.AlertItem, int, error) {
    // Build WHERE clause dynamically from filters (nil-safe)
    // Count query
    countQuery := `SELECT COUNT(*) FROM ops.metric_alert WHERE 1=1` + whereClause
    // Data query
    dataQuery := `SELECT ... FROM ops.metric_alert WHERE 1=1` + whereClause +
                 ` ORDER BY event_date DESC LIMIT $N OFFSET $M`
    
    var total int
    err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
    
    rows, _ := r.pool.Query(ctx, dataQuery, append(args, pagination.Limit, pagination.Offset)...)
    // ... scan rows
    return items, total, nil
}
```

### Filtering

Each endpoint supports the same filter parameters as FastAPI. Filters use `WHERE column = $param` when a value is provided, or skip the condition when the parameter is nil (NULL-safe comparison).

#### Filter: Optional Parameter Pattern

```sql
-- PostgreSQL pattern for optional filters
WHERE (column = $1 OR $1 IS NULL)  -- single value filter
  AND (column IN ($2) OR $2 IS NULL) -- multi-value filter
```

This matches the Python pattern:
```python
conditions = []
if status:
    conditions.append("status = ?")
params.append(status)
```

### Sorting

- Alert endpoints: `ORDER BY event_date DESC` (matches FastAPI)
- Task endpoints: `ORDER BY created_at DESC` (matches FastAPI)
- Outbox endpoints: `ORDER BY created_at DESC` (matches FastAPI)
- Log endpoints: `ORDER BY ts DESC` (newest first, matches tail-read behavior)

Custom sort parameters are **not supported** in Phase 4 (matching FastAPI limitation).

---

## 12. CORS Strategy

### Configuration

The Go API reads CORS settings from the environment, matching the FastAPI `CORS_ORIGINS` behavior:

```go
// internal/api/router.go
corsOriginsRaw := os.Getenv("CORS_ALLOWED_ORIGINS")
if corsOriginsRaw == "" {
    corsOriginsRaw = "http://localhost:5173,http://localhost:3000"
}
origins := strings.Split(corsOriginsRaw, ",")

// Apply CORS middleware
corsHandler := cors.New(cors.Options{
    AllowedOrigins:   origins,
    AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
    AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Request-ID"},
    AllowCredentials: true,
    MaxAge:           86400, // 24 hours
})
```

### Default Origins

When `CORS_ALLOWED_ORIGINS` is not set, the Go API defaults to:

- `http://localhost:5173` (Vite dev server for React frontend)
- `http://localhost:3000` (alternative dev port)

### Security Rules

- **Wildcard origins are rejected** (same as FastAPI security measure)
- **OPTIONS** preflight requests bypass rate limiting
- **Credentials** (Authorization header) are allowed

---

## 13. API Baseline Comparison Strategy

### Python Comparison Script

The baseline comparison is performed by `scripts/migration/compare_api_baseline.py`. This script:

1. Calls the old FastAPI (if still running) or reads baseline JSON files from `migration_baseline/api_responses/`
2. Calls the new Go API
3. Compares responses field by field
4. Reports pass/fail for each endpoint

### Script Design

```python
#!/usr/bin/env python3
"""
compare_api_baseline.py — API response parity verification.

Usage:
    python3 scripts/migration/compare_api_baseline.py \\
        --baseline-dir migration_baseline/api_responses/ \\
        --go-url http://localhost:8766/api/v1 \\
        --token $API_BEARER_TOKEN \\
        --ignore-fields version,database.path,database.tables.created_at

Compares all Phase 4 endpoints between Go API responses and
baseline snapshots. Reports per-endpoint diffs.
"""

import json
import os
import sys
import urllib.request

# Known explainable differences — excluded from comparison
IGNORED_FIELDS = frozenset({
    "version",
    "database.path",
    "last_pipeline_run",
    "run_id",
    "started_at",
    "finished_at",
    "created_at",
    "updated_at",
    "loaded_at",
    "ingested_at",
})

# Endpoints to test
PHASE4_ENDPOINTS = [
    ("GET", "/api/v1/health", {}, None),
    ("GET", "/api/v1/status", {}, None),
    ("GET", "/api/v1/alerts", {"limit": 5}, None),
    ("GET", "/api/v1/tasks", {"limit": 5}, None),
    ("GET", "/api/v1/outbox", {"limit": 5}, None),
    ("GET", "/api/v1/logs/recent", {"limit": 5}, None),
    ("GET", "/api/v1/logs/errors", {"limit": 5}, None),
    ("GET", "/api/v1/logs/audit", {"limit": 5, "source": "dispatch"}, None),
    ("GET", "/api/v1/governance/status", {}, None),
    ("GET", "/api/v1/qoder/capabilities", {}, None),
    ("GET", "/api/v1/qoder/context", {"limit_alerts": 5, "limit_tasks": 5}, None),
]

def compare_values(baseline, actual, path=""):
    """Recursively compare two JSON values, ignoring known fields."""
    if path in IGNORED_FIELDS:
        return []

    diffs = []

    if isinstance(baseline, dict) and isinstance(actual, dict):
        all_keys = set(baseline.keys()) | set(actual.keys())
        for key in all_keys:
            new_path = f"{path}.{key}" if path else key
            if new_path in IGNORED_FIELDS:
                continue
            if key not in baseline:
                diffs.append((new_path, "missing_in_baseline", None, actual.get(key)))
            elif key not in actual:
                diffs.append((new_path, "missing_in_actual", baseline.get(key), None))
            else:
                diffs.extend(compare_values(baseline[key], actual[key], new_path))

    elif isinstance(baseline, list) and isinstance(actual, list):
        max_len = max(len(baseline), len(actual))
        for i in range(max_len):
            new_path = f"{path}[{i}]"
            if i >= len(baseline):
                diffs.append((new_path, "extra_item", None, actual[i]))
            elif i >= len(actual):
                diffs.append((new_path, "missing_item", baseline[i], None))
            else:
                diffs.extend(compare_values(baseline[i], actual[i], new_path))

    else:
        # Numeric comparison with tolerance
        if isinstance(baseline, (int, float)) and isinstance(actual, (int, float)):
            if abs(baseline - actual) > 1e-6:
                diffs.append((path, "value_mismatch", baseline, actual))
        elif baseline != actual:
            diffs.append((path, "value_mismatch", baseline, actual))

    return diffs


def main():
    import argparse
    parser = argparse.ArgumentParser()
    parser.add_argument("--baseline-dir", default="migration_baseline/api_responses/")
    parser.add_argument("--go-url", default="http://localhost:8766/api/v1")
    parser.add_argument("--token", default="")
    args = parser.parse_args()

    summary = {"passed": 0, "failed": 0, "skipped": 0}

    for method, path, params, body in PHASE4_ENDPOINTS:
        endpoint_name = path.replace("/api/v1/", "").replace("/", "_")

        # Load baseline
        baseline_file = os.path.join(args.baseline_dir, f"{endpoint_name}.json")
        if not os.path.exists(baseline_file):
            print(f"[SKIP] {path}: baseline not found")
            summary["skipped"] += 1
            continue

        with open(baseline_file) as f:
            baseline = json.load(f)

        # Call Go API
        url = args.go_url + path
        if params:
            qs = "&".join(f"{k}={v}" for k, v in params.items())
            url += "?" + qs

        req = urllib.request.Request(url)
        if args.token:
            req.add_header("Authorization", f"Bearer {args.token}")

        try:
            with urllib.request.urlopen(req) as resp:
                actual = json.loads(resp.read())
        except Exception as e:
            print(f"[FAIL] {path}: connection error — {e}")
            summary["failed"] += 1
            continue

        # Compare
        diffs = compare_values(baseline, actual)
        if diffs:
            print(f"[FAIL] {path}: {len(diffs)} differences")
            for d in diffs[:10]:  # show first 10 diffs
                print(f"       {d[0]}: {d[1]} (baseline={d[2]}, actual={d[3]})")
            summary["failed"] += 1
        else:
            print(f"[PASS] {path}")
            summary["passed"] += 1

    print(f"\nSummary: {summary['passed']} passed, {summary['failed']} failed, {summary['skipped']} skipped")
    return 0 if summary["failed"] == 0 else 1
```

### Comparison Tiers

| Tier | Method | Scope | Pass Criteria |
|---|---|---|---|
| 1 | Row count | All endpoints | Each endpoint returns same number of items |
| 2 | Field presence | All endpoints | All JSON keys present (excluding explainable differences) |
| 3 | Value comparison | All endpoints | Float values within 1e-6 tolerance; string/int values exact match |
| 4 | Full diff | Selected endpoints | Zero differences (excluding ignore-list) |

### Expected Response Snapshots (for reference)

Baseline files in `migration_baseline/api_responses/`:

| File | Endpoint | Source |
|---|---|---|
| `health.json` | GET /health | FastAPI baseline |
| `status.json` | GET /status | FastAPI baseline |
| `alerts.json` | GET /alerts | FastAPI baseline (limit=100) |
| `tasks.json` | GET /tasks | FastAPI baseline (limit=100) |
| `outbox.json` | GET /outbox | FastAPI baseline (limit=100) |
| `governance_status.json` | GET /governance/status | FastAPI baseline |
| `qoder_context.json` | GET /qoder/context | FastAPI baseline |

New baseline files to add for Phase 4:

| File | Endpoint | Source |
|---|---|---|
| `logs_recent.json` | GET /logs/recent | FastAPI baseline |
| `logs_errors.json` | GET /logs/errors | FastAPI baseline |
| `logs_audit.json` | GET /logs/audit | FastAPI baseline |
| `qoder_capabilities.json` | GET /qoder/capabilities | FastAPI baseline |

---

## 14. Acceptance Criteria

### Tier 1: Response Shape Parity

- [ ] All 11 Phase 4 endpoints respond with HTTP 200 for valid requests
- [ ] All 11 response JSON shapes match the baseline snapshots field-for-field (excluding documented differences)
- [ ] List endpoints return the `{"items": [...], "total": N}` envelope

### Tier 2: Error Handling

- [ ] Missing `Authorization` header returns 401 with `"error_code": "AUTH_REQUIRED"`
- [ ] Invalid Bearer token returns 403 with `"error_code": "INVALID_TOKEN"`
- [ ] Invalid query parameter values return 400 with `"error_code": "VALIDATION_ERROR"`
- [ ] `limit` exceeding 1000 returns 400 for list endpoints
- [ ] Rate limiting returns 429 with `"error_code": "RATE_LIMITED"`
- [ ] Database unavailable returns 503 with `"error_code": "DB_UNAVAILABLE"`
- [ ] All error responses include `request_id`, `error_code`, `message`, `diagnosis`, `suggested_action`

### Tier 3: Request ID

- [ ] `X-Request-ID` header in request is reflected in response header
- [ ] `X-Request-ID` header in request is reflected in response body `request_id` field
- [ ] Request without `X-Request-ID` gets auto-generated UUID in response header and body
- [ ] Error responses include the request_id

### Tier 4: Pagination and Filtering

- [ ] Default limit is 100 for list endpoints
- [ ] Max limit is 1000 for alert/task/outbox; 500 for log endpoints
- [ ] Offset parameter is respected when provided
- [ ] Filter parameters work individually and in combination
- [ ] Unknown filter parameters are silently ignored (not error)

### Tier 5: Auth

- [ ] `GET /health` returns 200 without any auth header
- [ ] All other Phase 4 endpoints return 401 without auth header
- [ ] Weak tokens (length < 32 chars) are rejected
- [ ] Empty `API_BEARER_TOKEN` env var causes all requests to fail with 403

### Tier 6: CORS

- [ ] `OPTIONS` preflight requests from allowed origins return 200 with correct CORS headers
- [ ] `OPTIONS` preflight requests from disallowed origins return 403
- [ ] Default allowed origins include `localhost:5173` and `localhost:3000`
- [ ] Wildcard `*` origin is rejected

### Tier 7: Baseline Comparison Script

- [ ] `scripts/migration/compare_api_baseline.py` runs without errors
- [ ] All 11 Phase 4 endpoints pass comparison with baseline snapshots
- [ ] Float value tolerance of 1e-6 is respected
- [ ] Known difference fields are excluded from comparison

### Tier 8: Performance (Guideline)

- [ ] Each endpoint responds within 500ms (p99, cold start)
- [ ] Each endpoint responds within 100ms (p50, warm)
- [ ] Concurrent requests (10 parallel) do not cause errors

---

## Appendix A: Go Package File Map

```
cmd/
  baxi-api/
    main.go                    — HTTP server entry point, env config loading

internal/
  api/
    router.go                  — Route registration, middleware chain
    middleware.go              — request_id, auth, CORS, rate limiting, logging
    auth.go                    — Bearer token verification (constant-time)

    handler/
      health.go                — GET /api/v1/health
      status.go                — GET /api/v1/status
      alert.go                 — GET /api/v1/alerts
      task.go                  — GET /api/v1/tasks
      outbox.go                — GET /api/v1/outbox
      log.go                   — GET /api/v1/logs/recent, /logs/errors, /logs/audit
      governance.go            — GET /api/v1/governance/status
      qoder.go                 — GET /api/v1/qoder/capabilities, /qoder/context

    service/
      status_service.go        — System status aggregation
      alert_service.go         — Alert filtering and pagination
      task_service.go          — Task filtering and pagination
      outbox_service.go        — Outbox filtering and pagination
      log_service.go           — Log reading and filtering
      governance_service.go    — Governance status aggregation
      qoder_service.go         — Qoder context aggregation

    repository/
      pipeline_run_repo.go     — audit.pipeline_run CRUD
      alert_repo.go            — ops.metric_alert queries
      task_repo.go             — ops.task queries
      outbox_repo.go           — ops.outbox_event queries
      log_repo.go              — audit.api_request_log, audit.error_log, audit.audit_log
      governance_repo.go       — gov.* tables queries
      qoder_repo.go            — ai.qoder_runs, ai.qoder_reports queries

    dto/
      health.go                — HealthResponse
      status.go                — StatusResponse
      alert.go                 — AlertItem, AlertListResponse
      task.go                  — TaskItem, TaskListResponse
      outbox.go                — OutboxItem, OutboxListResponse
      log.go                   — RecentLogEntry, ErrorLogEntry, AuditLogEntry, *ListResponse
      governance.go            — GovernanceConfigResponse, GovernanceStatusResponse
      qoder.go                 — CapabilitiesResponse, ContextResponse
      error.go                 — ErrorResponse
```

## Appendix B: Environment Variables

| Variable | Default | Endpoints | Description |
|---|---|---|---|
| `API_BEARER_TOKEN` | (required) | All (except health) | Bearer token for API authentication |
| `DATABASE_URL` | `postgres://baxi:baxi_dev@localhost:5432/baxi?sslmode=disable` | All data endpoints | PostgreSQL connection string |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:5173,http://localhost:3000` | All | Comma-separated allowed CORS origins |
| `API_PORT` | `8766` | All | HTTP server listen port |
| `DEFAULT_USER` | `api_user` | Status, Qoder context | Default actor name for authenticated requests |
| `DEBUG` | `false` | All | Enable debug mode (verbose error messages) |
| `RATE_LIMIT_PER_MIN` | `300` | All | Default rate limit per endpoint per IP |
| `REQUEST_TIMEOUT` | `30` | All | Request timeout in seconds |

## Appendix C: Migration Baseline Files

### Existing Snapshots (already captured)

```
migration_baseline/api_responses/
├── health.json              — GET /api/v1/health
├── status.json              — GET /api/v1/status
├── alerts.json              — GET /api/v1/alerts
├── tasks.json               — GET /api/v1/tasks
├── outbox.json              — GET /api/v1/outbox
├── governance_status.json   — GET /api/v1/governance/status
└── qoder_context.json       — GET /api/v1/qoder/context
```

### Snapshots to Add

```
migration_baseline/api_responses/
├── logs_recent.json         — GET /api/v1/logs/recent  (to be captured)
├── logs_errors.json         — GET /api/v1/logs/errors  (to be captured)
├── logs_audit.json          — GET /api/v1/logs/audit   (to be captured)
└── qoder_capabilities.json  — GET /api/v1/qoder/capabilities (to be captured)
```

## Appendix D: Rollout Plan

| Phase | Action | Verification |
|---|---|---|
| 4a | Go API scaffolding + router + middleware + health endpoint | health.json baseline pass |
| 4b | Status + alerts + tasks endpoints | status.json, alerts.json, tasks.json baseline pass |
| 4c | Outbox + logs endpoints | outbox.json, logs_recent.json, logs_errors.json, logs_audit.json baseline pass |
| 4d | Governance status + Qoder endpoints | governance_status.json, qoder_capabilities.json, qoder_context.json baseline pass |
| 4e | Full comparison script + full API baseline verification | All 11 endpoints pass compare_api_baseline.py |
