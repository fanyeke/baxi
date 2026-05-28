# Baxi Migration Baseline

This directory captures the Python + SQLite baseline before migration to Go + PostgreSQL.

## Baseline Git Reference

- **Freeze tag**: `v0.5.3-python-sqlite-freeze`
- **Legacy branch**: `legacy/python-sqlite`
- **Migration branch**: `migration/go-postgres`
- **Freeze commit**: `8a0f57e`

## Captured Artifacts

| Artifact | Path | Description |
|----------|------|-------------|
| SQLite Schema DDL | `sqlite_schema.sql` | 16 tables from `data/olist_ops.db` schema |
| Table Row Counts | `table_counts.json` | 16 tables, 906,526 total rows |
| Pipeline CSV Samples | `pipeline_outputs/` | 8 CSV exports from core pipeline tables |
| API Response Snapshots | `api_responses/` | 7 JSON files from FastAPI endpoints |
| Governance YAML Snapshot | `configs_snapshot/` | 28 YAML config files from `config/` |

## Purpose

This baseline is used to verify that the Go + PostgreSQL migration preserves core behavior:

- Raw and DWD row counts
- Metric calculation results
- Alert generation results
- Recommendation and task generation
- Outbox behavior
- API response compatibility
- Governance configuration compatibility

## Known Missing Items

- **No remote `origin` configured**: Branches exist locally only. Push requires adding `git remote add origin <url>`.
- **Empty tables**: governance_checkpoints (0), governance_health_results (0), review_retro (0), qoder_jobs (0) — these tables have schema but no data.
- **Feishu endpoints not captured**: Feishu export/sync/import API responses were intentionally excluded (require real Feishu credentials).
- **Pipeline logs not captured**: Pipeline execution logs in `logs/` are excluded (runtime artifacts).
- **feishu_table_ids.yml.example**: Excluded from snapshot (contains real token placeholders); `feishu_table_ids.yml` itself IS captured.

## Notes

- Do not delete this directory until the Go + PostgreSQL migration has reached production parity.
- The SQLite database file (`data/olist_ops.db`, 268MB) is NOT included in this baseline — only schema DDL and row counts.
- All export scripts are in `scripts/migration/` and can be re-run to refresh the baseline.
