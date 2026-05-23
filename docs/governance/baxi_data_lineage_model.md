# Baxi Data Lineage Model v0.5.3

> **Generated:** 2026-05-23
> **Purpose:** Document complete data lineage from raw CSV through decision pipeline to external dispatch
> **Config:** config/data_lineage.yml
> **Related:** baxi_data_marking_policy.md (sensitivity propagation), baxi_data_health_checks.md (health expectations)

## Overview

Baxi's data flows through 8 distinct pipeline stages. This document maps each stage with its inputs, outputs, transformation type, and sensitivity propagation rules (参考: 03_data_lineage_overview.md, 09_data_lifetime_retention.md).

```
Stage 0: Raw CSV (Kaggle)
  │
  ├─ olist_orders_dataset.csv ──────────────┐
  ├─ olist_order_items_dataset.csv ─────────┤
  ├─ olist_customers_dataset.csv ───────────┤
  ├─ olist_sellers_dataset.csv ─────────────┤
  ├─ olist_products_dataset.csv ────────────┤
  ├─ olist_product_category_name_dataset ───┤
  └─ olist_geolocation_dataset.csv ─────────┤
                                            │
                                            ▼
Stage 1: Raw Ingestion → DWD Tables
  ├─ dwd_order_level (22 cols, PK: order_id)
  └─ dwd_item_level  (18 cols, PK: order_id+product_id)
                                            │
                                            ▼
Stage 2: DWD Transformation → Aggregated Metrics
  ├─ metric_daily              (daily aggregated metrics)
  └─ metric_dimension_daily    (dimension-level daily metrics)
                                            │
                                            ▼
Stage 3: Metrics → Alert Detection
  └─ alert_events                (triggered anomalies)
                                            │
                                            ▼
Stage 4: Alert → Strategy Decision
  └─ strategy_recommendations    (AI-generated strategies)
                                            │
                                            ▼
Stage 5: Strategy → Action Task
  └─ action_tasks                (execution work items)
                                            │
                                            ▼
Stage 6: Audit → Retro Analysis
  └─ review_retro                (post-execution analysis)
                                            │
                                            ▼
Stage 7: Event Dispatch → External Systems
  ├─ event_outbox                (dispatch queue)
  └─ Feishu                      (外部系统通知)
                                            │
                                            ▼
[Cross-cutting: Audit Logging]
  ├─ api_audit_dispatch.csv      (API-level audit)
  ├─ dispatch_archive.csv        (dispatch history)
  └─ API log files               (general logging)
```

---

## Stage 0: Raw CSV Files

| Attribute | Value |
|-----------|-------|
| **stage_id** | raw_csv |
| **name** | 原始CSV数据 |
| **description** | Kaggle Olist dataset, downloaded once, source of truth |

### Inputs
- External: Kaggle Olist E-Commerce dataset (9 CSV files + 1 ZIP)

### Outputs
- dwd_order_level (via ingestion script)
- dwd_item_level (via ingestion script)

### Sensitive Fields at Source
| CSV File | Sensitive Columns | Sensitivity |
|----------|-----------------|-------------|
| olist_customers_dataset.csv | customer_unique_id (PII), customer_zip_code_prefix, customer_city | PII, confidential |
| olist_orders_dataset.csv | customer_id (PII reference) | PII |
| olist_order_items_dataset.csv | seller_id, product_id | restricted, public |
| olist_sellers_dataset.csv | seller_zip_code_prefix, seller_city | confidential |

### Health Expectations
- File integrity: ZIP must extract successfully
- Row counts must match expected values
- No missing required columns

---

## Stage 1: DWD Tables

| Attribute | Value |
|-----------|-------|
| **stage_id** | dwd_creation |
| **name** | DWD明细层构建 |
| **description** | Transform raw CSV into normalized fact tables with cleaned data |
| **transformation_type** | aggregation, deduplication, join |

### Inputs
- raw CSV files (all 9 datasets)

### Outputs
- dwd_order_level (22 columns): one row per order, enriched with customer + order info
- dwd_item_level (18 columns): one row per order-item, enriched with seller + product info

### Sensitivity Propagation
```
customer_unique_id(PII) → dwd_order_level.customer_unique_id(PII) [propagated]
customer_city(confidential) → dwd_order_level.customer_city(confidential) [propagated]
order_id(internal) → dwd_order_level.order_id(internal) [propagated]
price(internal) → dwd_item_level.price(internal) [propagated]
seller_id(restricted) → dwd_item_level.seller_id(restricted) [propagated]
```

**Key rule**: PII and confidential fields from raw CSV propagate unchanged to DWD. They must be cleaned (dropped, hashed, or aggregated) before reaching the metrics stage.

### Health Expectations
- dwd_order_level.row_count > 0
- dwd_order_level has order_id UNIQUE constraint
- dwd_item_level has (order_id, product_id) composite PK
- No null values in required columns (order_id, order_status, total_payment_value)

---

## Stage 2: Aggregated Metrics

| Attribute | Value |
|-----------|-------|
| **stage_id** | metric_aggregation |
| **name** | 指标聚合层 |
| **description** | Aggregate DWD data into daily and dimension-level metrics |
| **transformation_type** | aggregation, grouping, temporal summarization |

### Inputs
- dwd_order_level
- dwd_item_level

### Outputs
- metric_daily: aggregated daily metrics (GMV, order count, revenue by date)
- metric_dimension_daily: dimension-level daily metrics (by category, region, etc.)

### Sensitivity Propagation
```
customer_unique_id(PII) → [DROPPED at aggregation - no PII in metrics]
customer_city(confidential) → metric_dimension_daily.dimension_value(internal) [downgraded - city is aggregated]
order_purchase_timestamp(internal) → metric_daily.metric_date(internal) [coarsened to date]
total_payment_value(confidential) → metric_daily.gmv(confidential) [aggregated, sensitivity preserved]
```

**Key transformation**: PII fields from DWD are dropped during aggregation. The metrics layer has NO PII fields. This is the point where sensitive raw data becomes safe analytical data.

### Health Expectations
- metric_daily.row_count > 0 for each day in data range
- metric_daily.date column is contiguous (no gaps > 1 day)
- metric_daily.gmv > 0 for each day
- metric_dimension_daily.grain matches defined dimensions from metrics.yml

---

## Stage 3: Alert Detection

| Attribute | Value |
|-----------|-------|
| **stage_id** | alert_detection |
| **name** | 异常检测 |
| **description** | Apply alert rules from config against metrics to detect anomalies |
| **transformation_type** | rule evaluation, threshold comparison |

### Inputs
- metric_daily
- metric_dimension_daily
- config/alert_rules.yml
- config/dimensional_alert_rules.yml

### Outputs
- alert_events: detected anomalies with severity, owner_role, metric info

### Sensitivity Propagation
```
metric_daily(internal) → alert_events.metric(internal) [propagated]
metric_daily.gmv(confidential) → alert_events.current_value(confidential) [propagated]
alert_rules.yml → alert_events.owner_role(restricted) [new field from config]
alert_rules.yml → alert_events.rule_id(internal) [new field from config]
```

Alert events contain metric values (confidential) but NO PII. Owner role mapping is restricted (ties to owner_mapping.yml).

### Health Expectations
- Alert events must have valid rule_id (referencing alert_rules.yml)
- Alert events must have valid owner_role (referencing owner_mapping.yml)
- No orphaned alerts (referencing non-existent metrics)

---

## Stage 4: Strategy Decision

| Attribute | Value |
|-----------|-------|
| **stage_id** | strategy_generation |
| **name** | 策略生成 |
| **description** | AI generates strategy recommendations from alert events |
| **transformation_type** | LLM inference, decision generation |

### Inputs
- alert_events
- config/llm_config.yml
- config/wake_io_contract.yml (I/O contract)

### Outputs
- strategy_recommendations: AI-generated strategies with confidence scores

### Sensitivity Propagation
```
alert_events(internal) → strategy_recommendations.alert_id(internal) [propagated]
alert_events.current_value(confidential) → strategy_recommendations.context(confidential) [propagated]
LLM output → strategy_recommendations.strategy_text(internal) [generated]
LLM output → strategy_recommendations.confidence_score(internal) [generated]
```

### Health Expectations
- Each strategy must reference an existing alert_id
- confidence_score ∈ [0.0, 1.0]
- strategy_text must not be empty

---

## Stage 5: Action Task

| Attribute | Value |
|-----------|-------|
| **stage_id** | task_creation |
| **name** | 任务创建 |
| **description** | Convert strategy recommendations into executable action tasks |
| **transformation_type** | strategy-to-task mapping, role-based assignment |

### Inputs
- strategy_recommendations
- config/action_registry.yml
- config/action_templates.yml
- config/owner_mapping.yml

### Outputs
- action_tasks: execution work items with risk level, assigned owner

### Sensitivity Propagation
```
strategy_recommendations(internal) → action_tasks.strategy_id(internal) [propagated]
action_registry.yml → action_tasks.risk_level(restricted) [from config]
owner_mapping.yml → action_tasks.assigned_role(restricted) [from config]
```

### Health Expectations
- Each task must reference a valid strategy_id
- risk_level must be in {low, medium, high}
- assigned_role must map to a valid owner_mapping.yml entry

---

## Stage 6: Retro Analysis

| Attribute | Value |
|-----------|-------|
| **stage_id** | retro_analysis |
| **name** | 回顾分析 |
| **description** | Post-execution analysis of action task outcomes |
| **transformation_type** | outcome tracking, status feedback |

### Inputs
- action_tasks
- status_enums.yml (state machine)

### Outputs
- review_retro: post-execution analysis records with status and feedback

### Health Expectations
- Each retro must reference existing action_tasks
- status must be valid per status_enums.yml

---

## Stage 7: External Dispatch

| Attribute | Value |
|-----------|-------|
| **stage_id** | external_dispatch |
| **name** | 外部分发 |
| **description** | Dispatch events to Feishu and external systems via outbox pattern |
| **transformation_type** | formatting, routing, API call |

### Inputs
- event_outbox (dispatch queue)
- config/channel_routing_rules.yml
- config/adapter_registry.yml
- config/feishu_field_mapping.yml
- config/feishu_app.yml

### Outputs
- Feishu message notifications
- External API callbacks (if configured)

### Sensitivity Propagation
```
event_outbox(internal) → Feishu message body(internal) [via field mapping]
alert_events.current_value(confidential) → Feishu field mapping checks feishu_field_mapping.yml [restricted fields filtered]
checkpoint_rules.yml → checkpoint justification required for PII exports [governance enforcement]
```

**Governance rule**: Before dispatching any data with PII markings, checkpoint justification must be recorded. This is enforced via the dry-run/apply pattern: dry-run shows what would be sent, apply executes only after confirmation.

### Health Expectations
- event_outbox.status transitions correctly per status_enums.yml
- dispatch_attempts increases correctly per adapter_registry.yml max_retries
- Feishu API returns 200 for successful sends

---

## Cross-Cutting: Audit Logging

All stages produce audit records:

| Audit Trail | Format | Retention | Source |
|-------------|--------|-----------|--------|
| api_audit_dispatch.csv | CSV | 2555 days (7 years) | api/routers/outbox.py |
| dispatch_archive.csv | CSV | 2555 days (7 years) | services/dispatch_service.py |
| API log files | Text/log | 365 days | FastAPI logging_config.py |
| checkpoint audit (future) | CSV | 2555 days (7 years) | governance layer |

### Audit Schema
```csv
request_id,timestamp,outbox_id,target_channel,adapter_name,mode,status,external_ref,error
```

---

## Sensitivity Propagation Summary

```
PII Fields:
  Raw CSV → DWD (propagated) → Metrics (DROPPED) → Alerts (no PII) → Strategies (no PII)

Confidential Fields:
  Raw CSV → DWD → Metrics (preserved, aggregated) → Alerts (preserved) → Strategies (preserved)

Internal Fields:
  All stages (full propagation)

Restricted Fields:
  Source config files → Alert events, Action tasks (owner roles, risk levels)
```

**Key Insight**: The DWD → Metrics transition is the "security boundary." Before Metrics, data contains PII and confidential fields. After Metrics, only aggregated, non-PII data exists. The Governance Center UI should only display metrics-level and above data to non-admin users.
