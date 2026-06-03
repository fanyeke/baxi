# Baxi Go + PostgreSQL Migration Plan

## 1. Goal

Migrate Baxi from Python + SQLite + FastAPI to Go + PostgreSQL + Docker Compose.

The migration should preserve the existing decision pipeline:

```
Raw CSV → DWD → Metrics → Alerts → Recommendations → Tasks → Outbox → API / Qoder / LLM Context
```

## 2. Baseline

- **Freeze tag**: `v0.5.3-python-sqlite-freeze`
- **Legacy branch**: `legacy/python-sqlite`
- **Migration branch**: `migration/go-postgres`
- **Baseline directory**: `migration_baseline/` (removed — migration complete; baseline data stored in PostgreSQL)

## 3. Non-Goals for Phase 0

- Do not implement Go services
- Do not introduce PostgreSQL schema
- Do not change the Python pipeline
- Do not change API contracts
- Do not enable LLM execution

## 4. Migration Phases

### Phase 0: Baseline Freeze (COMPLETE)

Captured current Python + SQLite behavior: schema DDL, row counts, pipeline CSV samples, API response snapshots, governance YAML snapshots, migration scripts.

### Phase 1: Docker Compose + PostgreSQL Foundation

- Add Docker Compose configuration
- Set up PostgreSQL container with init scripts
- Add Go project skeleton (module, main package, build scripts)
- Create database migration tool integration

### Phase 2: PostgreSQL Schema

- Create schemas: `raw`, `dwd`, `mart`, `ops`, `gov`, `ai`, `audit`
- Migrate 16-table schema from SQLite to PostgreSQL
- Add proper types (NUMERIC, TIMESTAMPTZ, BOOLEAN) replacing SQLite flexible types
- Add constraints (FK with ON DELETE rules, CHECK, UNIQUE)
- Create indexes matching SQLite baseline

### Phase 3: Pipeline Migration

Port the 9-step Python pipeline to Go:
- CSV ingest → DWD tables
- Daily metric calculation
- Dimension-level metric calculation
- Rule engine (5 alert rules + dimensional rules)
- Strategy recommendation generation
- Action task generation
- Feishu export preparation
- Trigger simulation → event outbox

Verify against Phase 0 baseline CSV samples and row counts.

### Phase 4: Go API Migration

Implement core read APIs first in Go:
- `health` (no auth)
- `status` (system status + table counts)
- `alerts` (query with filters)
- `tasks` (query with filters)
- `outbox` (list + dispatch)
- `governance status` (config aggregation)
- `qoder context` (cross-service context)
- `qoder capabilities` (capability matrix)
- `qoder reports` (recording)

Verify API response format matches Phase 0 JSON snapshots.

### Phase 5: Governance and Ontology Runtime

Implement:
- `ObjectRegistry` — loads `aip_object_schema.yml`, provides typed object access
- `ObjectQueryService` — semantic queries over objects (customer, order, seller, product, category, region, marketing_lead, metric_alert)
- `GovernanceService` — data classification, lineage, access policy, checkpoints, health checks
- LLM-safe context redaction (PII scrubbing based on data_classification.yml)

### Phase 6: Outbox Worker and Adapters

Implement Go worker:
- Poll `event_outbox` for pending events
- Claim events (optimistic locking)
- Dispatch through channel adapters
- Write results + audit logs

Port adapters:
- Feishu adapter (message API)
- GitHub Issue adapter
- Local CLI adapter (CSV audit)
- Manual adapter (review queue)

### Phase 7: LLM Decision Layer

Implement:
- `decision_case` data model
- LLM context builder (aggregates alerts + metrics + strategies + tasks)
- Structured LLM decision output (parsed from LLM response)
- `action_proposal` generation
- Human review workflow
- Decision evaluation (scoring against decision_eval_rules.yml)

## 5. Acceptance Criteria

The Go + PostgreSQL system must pass parity checks against the Phase 0 baseline:

- SQLite table counts ↔ PostgreSQL table counts
- DWD output consistency (dwd_order_level, dwd_item_level)
- Metric output consistency (metric_daily, metric_dimension_daily)
- Alert output consistency (alert_events)
- Recommendation/task output consistency
- Outbox event consistency
- API response format compatibility
- Governance config compatibility

## 6. Branch Strategy

- `main` — stable branch
- `legacy/python-sqlite` — old Python + SQLite implementation (READ-ONLY)
- `migration/go-postgres` — migration integration branch
- `feature/*` — short-lived implementation branches

## 7. Deletion Policy

- Do NOT delete `v0.5.3-python-sqlite-freeze` tag
- Keep `legacy/python-sqlite` until Go version has reached stable production parity
- Keep `migration_baseline/` until all acceptance criteria pass
- Delete only temporary `feature/*` branches after merge
