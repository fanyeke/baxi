# Baxi Data Retention Policy v0.5.3

> **Generated:** 2026-05-23
> **Purpose:** Define retention periods and deletion workflow for all Baxi data assets
> **Config:** config/retention_policies.yml
> **Related:** baxi_data_lineage_model.md, baxi_data_governance_policy.md

## Retention Policy Overview

Baxi's data retention policy follows the FIPPs principle that sensitive data should be deleted as soon as the processing purpose is fulfilled, balanced against legal/compliance requirements for long-term data retention (参考: 09_data_lifetime_retention.md).

### Guiding Principles

1. **Determine requirements early** — Retention periods are defined before data is ingested
2. **Lineage-aware deletion** — When source data is deleted, downstream derived data should also be reviewed
3. **Grace period** — Soft-delete first (7 days), then hard-delete for reversibility
4. **Audit everything** — Every deletion event is logged to the audit trail
5. **Irreversible after grace period** — Once grace period expires, deletion cannot be undone

---

## Retention Periods by Asset

### SQLite Tables

| Table | Retention Period | Rationale | Sensitivity | Deletion Method |
|-------|-----------------|-----------|-------------|----------------|
| **dwd_order_level** | 825 days (2.25 years) | Tax/legal compliance requires 2+ years of transaction data | L1/L4 | Soft-delete → 7-day grace → hard-delete |
| **dwd_item_level** | 825 days (2.25 years) | Same as order data (linked to orders) | L1/L4 | Soft-delete → 7-day grace → hard-delete |
| **metric_daily** | 730 days (2 years) | Analytical metrics needed for year-over-year comparison | L1/L2 | Soft-delete → 7-day grace → hard-delete |
| **metric_dimension_daily** | 730 days (2 years) | Same as metric_daily | L1/L2 | Soft-delete → 7-day grace → hard-delete |
| **alert_events** | 365 days (1 year) | Alerts are operational, not legally required long-term | L1/L2/L3 | Soft-delete → 7-day grace → hard-delete |
| **strategy_recommendations** | 365 days (1 year) | AI strategies are time-sensitive decision outputs | L1 | Soft-delete → 7-day grace → hard-delete |
| **action_tasks** | 365 days (1 year) | Tasks expire after execution | L1/L3 | Soft-delete → 7-day grace → hard-delete |
| **review_retro** | 365 days (1 year) | Retro analysis tied to action tasks | L1 | Soft-delete → 7-day grace → hard-delete |
| **event_outbox** | 90 days | Outbox items are dispatched quickly; old items are stale | L1/L2 | Hard-delete after 90 days (no grace needed for queue data) |
| **qoder_jobs** | 180 days | External job tracking, medium-term retention | L1/L2 | Soft-delete → 7-day grace → hard-delete |
| **pipeline_runs** | 365 days | Pipeline history for debugging and monitoring | L1 | Soft-delete → 7-day grace → hard-delete |
| **ingestion_batches** | 365 days | Ingestion history for data quality tracking | L1 | Soft-delete → 7-day grace → hard-delete |

### CSV Audit Files

| File | Retention Period | Rationale | Deletion Method |
|------|-----------------|-----------|-----------------|
| **api_audit_dispatch.csv** | 2555 days (7 years) | Compliance: audit logs required for legal discovery | Archive to cold storage → delete original |
| **dispatch_archive.csv** | 2555 days (7 years) | Same as above | Archive to cold storage → delete original |
| **governance_checkpoints.csv** | 2555 days (7 years) | Same as above | Archive to cold storage → delete original |
| **API log files** | 365 days (1 year) | Operational logs, not legal requirement | Hard-delete (rotation) |

### Raw CSV Source Files

| File | Retention Period | Rationale |
|------|-----------------|-----------|
| Raw Kaggle CSV files | 90 days after DWD ingestion | DWD tables contain cleansed data; raw files only needed for re-ingestion |
| ZIP downloads | Delete immediately after extraction | Archive format, no analytical value after extraction |

---

## Deletion Workflow

### Step 1: Eligibility Check

For each asset, check if `created_at < NOW() - retention_period`.

### Step 2: Soft-Delete

Mark the record as deleted without removing data:

```sql
-- SQLite pattern (add column if not exists)
ALTER TABLE <table> ADD COLUMN deleted_at TEXT DEFAULT NULL;

-- Soft-delete: mark but don't remove
UPDATE <table> SET deleted_at = CURRENT_TIMESTAMP WHERE created_at < <cutoff_date>;
```

### Step 3: Grace Period (7 days)

Soft-deleted records are excluded from queries but still exist in the database:

```sql
-- Application queries must filter out soft-deleted records
SELECT * FROM dwd_order_level WHERE deleted_at IS NULL;
```

During grace period:
- Records are NOT returned by queries
- Records CAN be restored (set deleted_at back to NULL)
- A "pending deletion" report is available at GET /api/v1/governance/retention/pending

### Step 4: Hard-Delete

After 7 days, physically remove the records:

```sql
DELETE FROM <table> WHERE deleted_at IS NOT NULL 
  AND deleted_at < DATETIME('now', '-7 days');
```

### Step 5: Audit Log

Log every deletion:

```csv
deletion_id,timestamp,table_name,records_deleted,deletion_type,initiated_by
uuid-1,2026-05-30T00:00:00Z,dwd_order_level,15000,soft-delete,scheduled_policy
uuid-2,2026-06-06T00:00:00Z,dwd_order_level,15000,hard-delete,scheduled_policy
```

---

## Lineage-Aware Deletion

Following Palantir Foundry's "Data Lifetime" pattern (参考: 09_data_lifetime_retention.md), deletion should cascade:

```
When dwd_order_level is deleted:
  → Review metric_daily (aggregated from dwd)
  → Review metric_dimension_daily (aggregated from dwd)
  → Review alert_events (derived from metrics)
  → Review strategy_recommendations (derived from alerts)
  → Review action_tasks (derived from strategies)
  → Review event_outbox (dispatched from alerts)

When metric_daily is deleted:
  → Review alert_events (derived from metrics)
  → Review metric_dimension_daily (same stage, different grain)
```

**Key insight**: Since Baxi data flows unidirectionally (CSV→DWD→Metrics→Alerts→Strategies→Tasks→Outbox), deleting upstream data does NOT automatically invalidate downstream data. Metrics are already aggregated; deleting DWD won't corrupt metrics. However, lineage documentation should record that historical context is lost.

### Deletion Cascade Rules

| Upstream Deleted | Downstream Impact | Action Required |
|-----------------|-------------------|-----------------|
| Raw CSV | DWD tables still valid | No action (DWD is the source of truth after ingestion) |
| dwd_order_level | Metrics still valid (already aggregated) | Document lineage break |
| metric_daily | Alerts still valid (already triggered) | No action |
| alert_events | Strategies become orphaned | Review and potentially soft-delete strategies |
| strategy_recommendations | Tasks become orphaned | Review and potentially mark tasks as "strategy_revoked" |
| action_tasks | Retro analysis becomes orphaned | No action (retro is post-execution) |
| event_outbox | External systems already notified | No action |

---

## Retention Policy Configuration

Machine-readable version in `config/retention_policies.yml`:

```yaml
retention_policies:
  - policy_id: dwd_transaction_retention
    name: "DWD交易数据保留"
    target_tables: [dwd_order_level, dwd_item_level]
    retention_days: 825
    grace_period_days: 7
    description: "交易相关数据保留2.25年（税务合规要求）"
    
  - policy_id: metric_retention
    name: "指标数据保留"
    target_tables: [metric_daily, metric_dimension_daily]
    retention_days: 730
    grace_period_days: 7
    description: "聚合指标数据保留2年（年度对比分析需求）"
    
  - policy_id: decision_chain_retention
    name: "决策链数据保留"
    target_tables: [alert_events, strategy_recommendations, action_tasks, review_retro, pipeline_runs, ingestion_batches]
    retention_days: 365
    grace_period_days: 7
    description: "决策相关数据保留1年（运营分析需求）"
    
  - policy_id: outbox_cleanup
    name: "Outbox清理"
    target_tables: [event_outbox]
    retention_days: 90
    grace_period_days: 0
    description: "分发队列保留90天（已分发的数据不需要保留）"
    
  - policy_id: audit_log_retention
    name: "审计日志保留"
    target_files: [api_audit_dispatch.csv, dispatch_archive.csv, governance_checkpoints.csv]
    retention_days: 2555
    archive_before_delete: true
    description: "审计日志保留7年（法律合规要求）"
    
  - policy_id: raw_csv_cleanup
    name: "原始CSV清理"
    target_files: ["data/raw/*.csv"]
    retention_days: 90
    description: "DWD导入后保留原始CSV文件90天"
```
