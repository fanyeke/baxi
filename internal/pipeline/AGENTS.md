# PIPELINE

**Generated:** 2026-05-28 15:45
**Commit:** d908f6d
**Branch:** main

## OVERVIEW
Go data pipeline: 7 sequential steps, Step interface, per-step pgx transactions, audit logging.

Exposed via MCP tools: `run_pipeline`, `get_pipeline_status`. See `internal/mcp/tools_pipeline.go` and `internal/mcp/tools_outbox.go` for handler implementations.

## STEPS (execution order)
1. `ingest_raw` — CSV → raw tables
2. `build_dwd` — DWD order-level + item-level
3. `build_metrics` — Daily + dimension-daily metric tables
4. `detect_alerts` — Anomaly detection
5. `generate_recommendations`
6. `generate_tasks`
7. `create_outbox` — Outbox event dispatch

## KEY PATTERNS
- **Step interface**: `Name() string` + `Run(ctx, tx, input) (*StepOutput, error)` in `step.go`
- **Per-step transactions**: Runner wraps each step in own `pgx.Tx`; commit on success, rollback on failure
- **Audit**: `AuditRecorder` logs input/output row counts to `audit.pipeline_step_run`
- **Makefile targets**: `pipeline-ingest`, `pipeline-dwd`, etc. for step-level execution
- **Step ordering**: hardcoded in `Steps` slice in `runner.go`

## ANTI-PATTERNS
- No step-level idempotency keys — re-runs may duplicate rows
- Step ordering is a flat list, not a DAG — no conditional branching or parallel execution
