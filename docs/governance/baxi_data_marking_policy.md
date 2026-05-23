# Baxi Data Marking Policy v0.5.3

> **Generated:** 2026-05-23
> **Purpose:** Define sensitivity classification and marking propagation for all Baxi data fields
> **Config:** config/data_markings.yml, config/data_classification.yml
> **Related:** baxi_ontology_model.md, baxi_data_lineage_model.md

## Marking System Overview

Baxi uses a 5-level sensitivity marking system inspired by Palantir Foundry's Classification-Based Access Controls (CBAC) (参考: 04_classification_based_access_controls.md, 02_protecting_sensitive_data_markings.md).

### Sensitivity Levels

| Level | Code | Description | Access Requirement | Example |
|-------|------|-------------|-------------------|---------|
| **Public** | `L0` | Non-sensitive, safe for any context | No auth required | product_category_name, product_weight_g |
| **Internal** | `L1` | Business-relevant, authenticated users only | Bearer token auth required | order_status, review_score |
| **Confidential** | `L2` | Business-sensitive, domain owners | Bearer token + domain owner role | gmv_total, conversion_rate |
| **Restricted** | `L3` | Access-controlled, business-critical | Bearer token + specific role assignment | seller_id, customer last_order_date |
| **PII** | `L4` | Personally identifiable, highest protection | Bearer token + admin + checkpoint justification | customer_unique_id |

### Marking Categories

| Category | Description | Values |
|----------|-------------|--------|
| **data_sensitivity** | Primary sensitivity level | L0, L1, L2, L3, L4 |
| **access_scope** | Who can access this data | all, authenticated, owner_role, admin |
| **data_type** | Classification of data content | identifier, metric, temporal, geographic, financial, categorical |
| **pi_indicator** | Whether the field contains PII | yes, no |

---

## SQLite Table Markings (All 12 Tables)

### Table: dwd_order_level (22 columns)

| Column | Sensitivity | Access Scope | Data Type | PII | Rationale |
|--------|------------|--------------|-----------|-----|----------|
| order_id | L1 | authenticated | identifier | no | Internal business key |
| customer_unique_id | **L4** | admin | identifier | **yes** | Personal identifier (customer_email equivalent) |
| order_status | L1 | authenticated | categorical | no | Non-sensitive order state |
| order_purchase_timestamp | L1 | authenticated | temporal | no | Transaction date |
| total_payment_value | L2 | owner_role | financial | no | Revenue data |
| payment_type | L1 | authenticated | categorical | no | Payment method, not personal |
| payment_installments | L1 | authenticated | numeric | no | Installment count |
| payment_value | L2 | owner_role | financial | no | Payment amount |
| price | L1 | authenticated | financial | no | Item price |
| freight_value | L1 | authenticated | financial | no | Shipping cost |
| customer_state | L1 | authenticated | geographic | no | State-level, not PII |
| customer_city | L2 | owner_role | geographic | no | City-level can aid identification |
| product_category_name | L0 | all | categorical | no | Public category |
| product_name_lenght | L0 | all | numeric | no | Product metadata |
| product_description_lenght | L0 | all | numeric | no | Product metadata |
| product_photos_qty | L0 | all | numeric | no | Product metadata |
| product_weight_g | L0 | all | numeric | no | Product metadata |
| product_length_cm | L0 | all | numeric | no | Product metadata |
| product_height_cm | L0 | all | numeric | no | Product metadata |
| product_width_cm | L0 | all | numeric | no | Product metadata |
| review_score | L1 | authenticated | numeric | no | Aggregated review data |
| review_comment_message | L1 | authenticated | categorical | no | Review text, may contain personal info from customers |

### Table: dwd_item_level (18 columns)

| Column | Sensitivity | Access Scope | Data Type | PII |
|--------|------------|--------------|-----------|-----|
| order_id | L1 | authenticated | identifier | no |
| order_item_id | L1 | authenticated | identifier | no |
| product_id | L0 | all | identifier | no |
| seller_id | L3 | owner_role | identifier | no |
| shipping_limit_date | L1 | authenticated | temporal | no |
| price | L1 | authenticated | financial | no |
| freight_value | L1 | authenticated | financial | no |
| customer_city | L2 | owner_role | geographic | no |
| customer_state | L1 | authenticated | geographic | no |
| customer_unique_id | **L4** | admin | identifier | **yes** |
| seller_city | L2 | owner_role | geographic | no |
| seller_state | L1 | authenticated | geographic | no |
| product_category_name | L0 | all | categorical | no |
| product_category_name_english | L0 | all | categorical | no |
| review_score | L1 | authenticated | numeric | no |
| order_purchase_timestamp | L1 | authenticated | temporal | no |
| order_status | L1 | authenticated | categorical | no |
| total_payment_value | L2 | owner_role | financial | no |

### Table: metric_daily

| Column | Sensitivity | Access Scope | Data Type | PII |
|--------|------------|--------------|-----------|-----|
| metric_date | L1 | authenticated | temporal | no |
| metric_name | L1 | authenticated | categorical | no |
| metric_value | L2 | owner_role | financial/metric | no |

**Note**: Aggregated metrics have NO PII fields. The DWD→Metrics transition drops customer_unique_id.

### Table: metric_dimension_daily

| Column | Sensitivity | Access Scope | Data Type | PII |
|--------|------------|--------------|-----------|-----|
| metric_date | L1 | authenticated | temporal | no |
| metric_name | L1 | authenticated | categorical | no |
| dimension_type | L1 | authenticated | categorical | no |
| dimension_value | L1 | authenticated | categorical | no |
| metric_value | L2 | owner_role | financial/metric | no |

### Table: alert_events

| Column | Sensitivity | Access Scope | Data Type | PII |
|--------|------------|--------------|-----------|-----|
| alert_id | L1 | authenticated | identifier | no |
| rule_id | L1 | authenticated | identifier | no |
| metric | L1 | authenticated | categorical | no |
| dimension_type | L1 | authenticated | categorical | no |
| dimension_value | L1 | authenticated | categorical | no |
| alert_date | L1 | authenticated | temporal | no |
| severity | L1 | authenticated | categorical | no |
| current_value | L2 | owner_role | numeric | no |
| baseline_value | L1 | authenticated | numeric | no |
| deviation_pct | L1 | authenticated | numeric | no |
| status | L1 | authenticated | categorical | no |
| owner_role | L3 | owner_role | identifier | no |

### Table: strategy_recommendations

| Column | Sensitivity | Access Scope | Data Type | PII |
|--------|------------|--------------|-----------|-----|
| strategy_id | L1 | authenticated | identifier | no |
| alert_id | L1 | authenticated | foreign key | no |
| strategy_type | L1 | authenticated | categorical | no |
| strategy_text | L1 | authenticated | text | no |
| confidence_score | L1 | authenticated | numeric | no |
| status | L1 | authenticated | categorical | no |
| created_at | L1 | authenticated | temporal | no |

### Table: action_tasks

| Column | Sensitivity | Access Scope | Data Type | PII |
|--------|------------|--------------|-----------|-----|
| task_id | L1 | authenticated | identifier | no |
| strategy_id | L1 | authenticated | foreign key | no |
| action_type | L1 | authenticated | categorical | no |
| risk_level | L3 | owner_role | categorical | no |
| status | L1 | authenticated | categorical | no |
| assigned_role | L3 | owner_role | identifier | no |
| assigned_user | L3 | owner_role | identifier | no |
| target_object_type | L1 | authenticated | categorical | no |
| target_object_id | L1 | authenticated | identifier | no |
| created_at | L1 | authenticated | temporal | no |

### Table: review_retro

| Column | Sensitivity | Access Scope | Data Type | PII |
|--------|------------|--------------|-----------|-----|
| retro_id | L1 | authenticated | identifier | no |
| task_id | L1 | authenticated | foreign key | no |
| outcome | L1 | authenticated | categorical | no |
| status | L1 | authenticated | categorical | no |
| feedback | L1 | authenticated | text | no |

### Table: event_outbox

| Column | Sensitivity | Access Scope | Data Type | PII |
|--------|------------|--------------|-----------|-----|
| outbox_id | L1 | authenticated | identifier | no |
| alert_id | L1 | authenticated | foreign key | no |
| target_channel | L1 | authenticated | categorical | no |
| payload_json | L2 | owner_role | JSON | no |
| status | L1 | authenticated | categorical | no |
| dispatch_attempts | L1 | authenticated | numeric | no |
| last_dispatch_at | L1 | authenticated | temporal | no |
| external_ref | L1 | authenticated | identifier | no |
| adapter_name | L1 | authenticated | categorical | no |
| created_at | L1 | authenticated | temporal | no |

### Table: pipeline_runs

| Column | Sensitivity | Access Scope | Data Type | PII |
|--------|------------|--------------|-----------|-----|
| run_id | L1 | authenticated | identifier | no |
| started_at | L1 | authenticated | temporal | no |
| completed_at | L1 | authenticated | temporal | no |
| status | L1 | authenticated | categorical | no |
| error_message | L1 | authenticated | text | no |

### Table: ingestion_batches

| Column | Sensitivity | Access Scope | Data Type | PII |
|--------|------------|--------------|-----------|-----|
| batch_id | L1 | authenticated | identifier | no |
| started_at | L1 | authenticated | temporal | no |
| completed_at | L1 | authenticated | temporal | no |
| status | L1 | authenticated | categorical | no |
| records_ingested | L1 | authenticated | numeric | no |
| error_message | L1 | authenticated | text | no |

### Table: qoder_jobs

| Column | Sensitivity | Access Scope | Data Type | PII |
|--------|------------|--------------|-----------|-----|
| job_id | L1 | authenticated | identifier | no |
| job_type | L1 | authenticated | categorical | no |
| status | L1 | authenticated | categorical | no |
| payload_json | L2 | owner_role | JSON | no |
| created_at | L1 | authenticated | temporal | no |
| completed_at | L1 | authenticated | temporal | no |

---

## API Endpoint Markings

| Endpoint | Method | Sensitivity | Rationale |
|----------|--------|-------------|-----------|
| /api/v1/health | GET | L0 | No auth required, returns only status |
| /api/v1/status | GET | L1 | Returns system status with metric counts |
| /api/v1/alerts | GET | L1 | Returns alert metadata (no raw values) |
| /api/v1/tasks | GET | L2 | Contains risk levels and owner assignments |
| /api/v1/outbox | GET | L1 | Returns outbox queue status |
| /api/v1/outbox/dispatch | **POST** | **L2** | **Modifies state, requires dry-run/apply** |
| /api/v1/logs | GET | L2 | Contains detailed operational data |
| /api/v1/feishu | GET | L2 | Contains Feishu sync status and data |
| /api/v1/pipeline/run | **POST** | **L3** | **Triggers pipeline execution** |
| /api/v1/diagnosis | GET | L1 | Returns system diagnostics |
| /api/v1/governance/catalog | GET | L2 | Returns sensitivity markings (meta only, not data) |
| /api/v1/governance/lineage | GET | L1 | Returns pipeline structure |
| /api/v1/governance/health | GET | L2 | Returns health check results |
| /api/v1/governance/checkpoints | GET | L2 | Returns checkpoint records (audit data) |

---

## Marking Propagation Rules

Following Palantir Foundry's principle that "markings propagate along data lineage" (参考: 02_protecting_sensitive_data_markings.md):

| Rule | Description | Example |
|------|-------------|---------|
| **Inherit** | Output column inherits sensitivity from all input columns | dwd_order_level.customer_unique_id(L4) inherits from raw customer_id(L4) |
| **Max** | When aggregating, output takes the MAXIMUM sensitivity of all inputs | Aggregating L1 + L4 → L4 (before dropping) |
| **Drop** | When PII columns are dropped, sensitivity drops to L0 | customer_unique_id dropped in metrics stage → L0 |
| **Hash** | When PII is hashed, sensitivity drops to L3 | SHA256(customer_email) → L3 (one-way, not PII) |
| **Aggregate** | When data is aggregated, sensitivity drops one level | L4 → L2 after group-by aggregation |

---

## Markings for Configuration Files

| Config File | Sensitivity | Rationale |
|-------------|------------|-----------|
| aip_object_schema.yml | L1 | Object definitions, business metadata |
| alert_rules.yml | L1 | Rule definitions (no sensitive values) |
| action_registry.yml | L2 | Risk levels and approval requirements |
| status_enums.yml | L0 | Public enum definitions |
| owner_mapping.yml | L3 | Role-to-user assignments |
| channel_routing_rules.yml | L2 | Routing rules with severity mappings |
| adapter_registry.yml | L1 | Adapter configuration |
| feishu_app.yml | **L3** | Feishu app credentials (excluded from git) |
| feishu_field_mapping.yml | L2 | Feishu field mappings |
| data_quality_rules.yml | L1 | Quality rule definitions |
| metrics.yml | L1 | Metric definitions |
| llm_config.yml | **L3** | LLM API keys (excluded from git) |

---

## Configuration Index

The machine-readable version of these markings is stored in `config/data_markings.yml`:

```yaml
# Structure:
tables:
  <table_name>:
    columns:
      <column_name>:
        sensitivity: L0|L1|L2|L3|L4
        access_scope: all|authenticated|owner_role|admin
        data_type: identifier|metric|temporal|geographic|financial|categorical|text|numeric
        is_pii: true|false
api_endpoints:
  <endpoint>:
    sensitivity: L0|L1|L2|L3|L4
    requires_checkpoint: true|false
csv_files:
  <file_name>:
    sensitivity: L0|L1|L2|L3|L4
```
