# Baxi Data Health Checks v0.5.3

> **Generated:** 2026-05-23
> **Purpose:** Define data health monitoring rules and thresholds for all Baxi tables
> **Config:** config/health_checks.yml
> **Related:** baxi_data_lineage_model.md (health expectations per stage), baxi_data_governance_policy.md

## Overview

Baxi's data health monitoring follows Palantir Foundry's Data Health pattern (参考: 14_data_health_checks.md), adapted for a solo developer's operational needs. Health checks cover 5 categories: status, time, size, content, and schema.

### Alert Routing

| Severity | Routing | Response Time |
|----------|---------|---------------|
| **critical** | Feishu immediate notification + email | Within 1 hour |
| **warning** | Feishu daily digest | Within 24 hours |
| **info** | Governance Center UI only | No action required |

---

## Health Checks by Table

### Table: dwd_order_level

| Check ID | Category | Type | Description | Threshold | Severity |
|----------|----------|------|-------------|-----------|----------|
| DWD001 | Status | Build status | DWD order table build succeeded | Status = success | critical |
| DWD002 | Size | Row count | Orders ingested | row_count > 50000 | warning |
| DWD003 | Content | Primary key | Order IDs are unique | duplicate_count = 0 | critical |
| DWD004 | Content | Null percentage | Required columns not null | order_id null% < 0.1% | critical |
| DWD005 | Content | Null percentage | Payment value not null | total_payment_value null% < 5% | warning |
| DWD006 | Schema | Column count | Table has all expected columns | column_count = 22 | critical |
| DWD007 | Time | Data freshness | DWD data is up-to-date | Last update < 24 hours ago | warning |

### Table: dwd_item_level

| Check ID | Category | Type | Description | Threshold | Severity |
|----------|----------|------|-------------|-----------|----------|
| DWI001 | Status | Build status | DWD item table build succeeded | Status = success | critical |
| DWI002 | Size | Row count | Items ingested | row_count > 100000 | warning |
| DWI003 | Content | Primary key | (order_id, product_id) composite unique | duplicate_count = 0 | critical |
| DWI004 | Schema | Column count | Table has all expected columns | column_count = 18 | critical |
| DWI005 | Time | Data freshness | DWD item data is up-to-date | Last update < 24 hours ago | warning |

### Table: metric_daily

| Check ID | Category | Type | Description | Threshold | Severity |
|----------|----------|------|-------------|-----------|----------|
| MTR001 | Status | Build status | Daily metrics build succeeded | Status = success | critical |
| MTR002 | Size | Row count | At least one metric per day | row_count > 0 | critical |
| MTR003 | Time | Data freshness | Daily metrics updated within 24h | Last update < 24 hours ago | warning |
| MTR004 | Content | Date range | No gaps > 1 day in metric_date | max_gap ≤ 1 day | warning |
| MTR005 | Content | Numeric range | GMV is non-negative | gmv ≥ 0 for all rows | warning |
| MTR006 | Content | Numeric range | Order count is non-negative | order_count ≥ 0 for all rows | critical |

### Table: metric_dimension_daily

| Check ID | Category | Type | Description | Threshold | Severity |
|----------|----------|------|-------------|-----------|----------|
| MTD001 | Status | Build status | Dimension metrics build succeeded | Status = success | critical |
| MTD002 | Schema | Column count | Table has all expected columns | column_count matches definition | critical |
| MTD003 | Content | Null percentage | dimension_value not null | null% < 1% | warning |
| MTD004 | Time | Data freshness | Last update < 24 hours ago | Last update < 24 hours ago | warning |

### Table: alert_events

| Check ID | Category | Type | Description | Threshold | Severity |
|----------|----------|------|-------------|-----------|----------|
| ALE001 | Content | Null percentage | Required fields populated | metric, severity, status null% < 1% | warning |
| ALE002 | Content | Allowed values | Severity is valid | severity ∈ {critical, warning, info} | critical |
| ALE003 | Content | Foreign key | rule_id references alert_rules.yml | All rule_ids valid | warning |
| ALE004 | Content | Foreign key | owner_role references owner_mapping.yml | All owner_roles valid | warning |
| ALE005 | Size | Row count | Alerts generated within expected range | daily_alerts < 1000 | info |
| ALE006 | Time | Data freshness | New alerts in last 24h (if rules trigger) | Last alert < 48h or 0 alerts | warning |

### Table: strategy_recommendations

| Check ID | Category | Type | Description | Threshold | Severity |
|----------|----------|------|-------------|-----------|----------|
| STR001 | Content | Required fields | All required fields present | strategy_type, confidence not null | warning |
| STR002 | Content | Numeric range | Confidence score valid | 0.0 ≤ confidence_score ≤ 1.0 | critical |
| STR003 | Content | Foreign key | alert_id references alert_events | All alert_ids valid | warning |
| STR004 | Content | Null percentage | strategy_text not empty | text_length > 0 | warning |

### Table: action_tasks

| Check ID | Category | Type | Description | Threshold | Severity |
|----------|----------|------|-------------|-----------|----------|
| TSK001 | Content | Allowed values | risk_level is valid | risk_level ∈ {low, medium, high} | critical |
| TSK002 | Content | Allowed values | status is valid per status_enums.yml | status ∈ {pending, in_progress, completed, failed, cancelled} | critical |
| TSK003 | Content | Foreign key | strategy_id references strategy_recommendations | All strategy_ids valid | warning |
| TSK004 | Content | Foreign key | assigned_role references owner_mapping.yml | All assigned_roles valid | warning |

### Table: review_retro

| Check ID | Category | Type | Description | Threshold | Severity |
|----------|----------|------|-------------|-----------|----------|
| RTR001 | Content | Foreign key | task_id references action_tasks | All task_ids valid | warning |
| RTR002 | Content | Allowed values | status is valid per status_enums.yml | status ∈ valid enum | critical |

### Table: event_outbox

| Check ID | Category | Type | Description | Threshold | Severity |
|----------|----------|------|-------------|-----------|----------|
| OUT001 | Content | Allowed values | status is valid | status ∈ {pending, dispatched, failed, retry_pending} | critical |
| OUT002 | Content | Numeric range | dispatch_attempts within limit | dispatch_attempts ≤ max_retries (from adapter_registry.yml) | warning |
| OUT003 | Time | Data freshness | No pending items older than 1h | pending_age < 1 hour | critical |
| OUT004 | Size | Row count | Pending items manageable | pending_count < 100 | warning |

### Table: pipeline_runs

| Check ID | Category | Type | Description | Threshold | Severity |
|----------|----------|------|-------------|-----------|----------|
| PIP001 | Status | Schedule status | Last pipeline run succeeded | Status = success | critical |
| PIP002 | Time | Build duration | Pipeline completes within time limit | duration < 60 minutes | warning |
| PIP003 | Time | Schedule freshness | Pipeline runs on schedule | Last run > 0 (if scheduled) | critical |

### Table: ingestion_batches

| Check ID | Category | Type | Description | Threshold | Severity |
|----------|----------|------|-------------|-----------|----------|
| ING001 | Status | Sync status | Last ingestion succeeded | Status = success | critical |
| ING002 | Size | Row count | Records ingested | records_ingested > 0 | warning |
| ING003 | Time | Sync duration | Ingestion completes within time limit | duration < 30 minutes | warning |

---

## Health Check Configuration

```yaml
health_checks:
  checks:
    - check_id: DWD001
      name: "DWD订单表构建状态"
      target_table: dwd_order_level
      category: status
      check_type: build_status
      threshold: "status = 'success'"
      severity: critical
      
    - check_id: MTR003
      name: "日指标数据新鲜度"
      target_table: metric_daily
      category: time
      check_type: data_freshness
      threshold: "last_update < 24h"
      severity: warning
    
    # ... all checks defined above
    
  execution_schedule:
    - schedule_id: post_pipeline
      description: "Run after each pipeline completion"
      checks: [DWD001, DWD002, DWI001, MTR001, MTD001, PIP001]
      
    - schedule_id: daily_morning
      description: "Daily health check at 8:00 AM"
      checks: [MTR003, MTD004, OUT003, ALE006]
      trigger: cron("0 8 * * *")
      
    - schedule_id: weekly_review
      description: "Weekly comprehensive health check"
      checks: [ALL]
      trigger: cron("0 9 * * 1")  # Monday 9:00 AM
      
  alert_routing:
    critical:
      channels: [feishu_immediate, email]
      response_time: "1h"
    warning:
      channels: [feishu_daily_digest]
      response_time: "24h"
    info:
      channels: [governance_center_ui]
      response_time: "no_action"
```

---

## Stale Data Detection

A table is considered **stale** if its last update exceeds the freshness threshold:

| Table | Stale Threshold | Current Check |
|-------|----------------|---------------|
| dwd_order_level | 24 hours | DWD007 |
| dwd_item_level | 24 hours | DWI005 |
| metric_daily | 24 hours | MTR003 |
| metric_dimension_daily | 24 hours | MTD004 |
| alert_events | 48 hours (may not trigger daily) | ALE006 |
| event_outbox | 1 hour (for pending items) | OUT003 |

Stale detection is surfaced in the Governance Center UI as a "Health Status" indicator with color coding:
- 🟢 Healthy: All checks passed
- 🟡 Warning: Some checks in warning state
- 🔴 Critical: At least one critical check failed
- ⚪ Unknown: No recent health check data
