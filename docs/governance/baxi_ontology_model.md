# Baxi Ontology 模型 v0.5.3

> **Generated:** 2026-05-23
> **Purpose:** Map all AIP objects with governance annotations (sensitivity, ownership, lineage, retention)
> **Base Config:** config/aip_object_schema.yml (extended, not modified)
> **Related:** baxi_data_marking_policy.md, baxi_data_lineage_model.md, baxi_ontology_alignment.md

## Overview

This document extends Baxi's 9 AIP object types defined in `aip_object_schema.yml` with governance metadata following Palantir Foundry's ontology model (参考: 08_ontology_overview.md). Each object type includes:

- **Governing metadata**: sensitivity, owner role, retention period, purpose
- **Property-level markings**: per-field sensitivity classification
- **Lineage tracking**: source table/column for each property
- **Relationship governance**: security implications of object links

## Sensitivity Levels

| Level | Description | Example |
|-------|-------------|---------|
| **public** | Non-sensitive, aggregatable data | product weight, category name |
| **internal** | Business-relevant but not sensitive | order status, review score |
| **confidential** | Business-sensitive data | sales performance, seller GMV |
| **restricted** | Access-controlled, business-critical | customer purchasing patterns |
| **PII** | Personally identifiable information | customer email, phone, address |

---

## Object 1: customer (客户)

| Attribute | Value |
|-----------|-------|
| **object_type_id** | customer |
| **display_name** | 客户 |
| **grain** | customer_unique_id |
| **owner_role** | business_ops |
| **retention_period** | 730 days (2 years) |
| **allow_sync_to_feishu** | false (PII restriction) |
| **purpose** | Customer behavior analysis, LTV calculation, geographic segmentation |

### Property Markings

| Property | Type | Sensitivity | Source Table | Source Column | Lineage Description |
|----------|------|-------------|-------------|---------------|-------------------|
| customer_unique_id | string (PK) | **PII** | dwd_order_level | customer_unique_id | Direct from raw: customer_id is personal identifier |
| customer_state | string | **internal** | dwd_order_level | customer_state | Geographic region, not PII at state level |
| customer_city | string | **confidential** | dwd_order_level | customer_city | City-level location, can combine with other data to identify |
| order_count | int | **internal** | dwd_order_level | order_id (agg:nunique) | Aggregated, no PII |
| gmv_total | float | **confidential** | dwd_order_level | total_payment_value (agg:sum) | Revenue data, business-sensitive |
| avg_review_score | float | **internal** | dwd_order_level | review_score (agg:avg) | Aggregated rating data |
| first_order_date | datetime | **restricted** | dwd_order_level | order_purchase_timestamp (agg:min) | Temporal pattern data, useful for customer profiling |
| last_order_date | datetime | **restricted** | dwd_order_level | order_purchase_timestamp (agg:max) | Recency data for customer segmentation |

### Lineage
```
Raw CSV: olist_orders_dataset.customer_id → DWD: customer_unique_id → Ontology: customer
Raw CSV: olist_customers_dataset.customer_state → DWD: customer_state → Ontology: customer_state
```

### Relationships
- customer → order (1:N): A customer has many orders. Link inherits PII marking.
- customer → region (N:1): A customer belongs to a region. Link downgrades to internal (region is not PII).

---

## Object 2: order (订单)

| Attribute | Value |
|-----------|-------|
| **object_type_id** | order |
| **display_name** | 订单 |
| **grain** | order_id |
| **owner_role** | business_ops |
| **retention_period** | 825 days (2.25 years for tax/legal compliance) |
| **allow_sync_to_feishu** | false |
| **purpose** | Order tracking, fulfillment monitoring, revenue analysis |

### Property Markings

| Property | Type | Sensitivity | Source Table | Source Column |
|----------|------|-------------|-------------|---------------|
| order_id | string (PK) | **internal** | dwd_order_level | order_id |
| order_status | string | **internal** | dwd_order_level | order_status |
| order_purchase_timestamp | datetime | **internal** | dwd_order_level | order_purchase_timestamp |
| total_payment_value | float | **confidential** | dwd_order_level | total_payment_value |
| payment_type | string | **internal** | dwd_order_level | payment_type |
| review_score | float | **internal** | dwd_order_level | review_score |
| delivery_status | string | **internal** | dwd_order_level | derived (status mapping) |

### Lineage
```
Raw CSV: olist_orders_dataset.order_id → DWD: dwd_order_level.order_id → Ontology: order
```

---

## Object 3: seller (卖家)

| Attribute | Value |
|-----------|-------|
| **object_type_id** | seller |
| **display_name** | 卖家 |
| **grain** | seller_id |
| **owner_role** | seller_ops |
| **retention_period** | 730 days |
| **allow_sync_to_feishu** | false |
| **purpose** | Seller performance analysis, quality monitoring, geographic distribution |

### Property Markings

| Property | Type | Sensitivity | Source Table | Source Column |
|----------|------|-------------|-------------|---------------|
| seller_id | string (PK) | **restricted** | dwd_item_level | seller_id |
| seller_state | string | **internal** | dwd_item_level | seller_state |
| seller_city | string | **confidential** | dwd_item_level | seller_city |
| gmv | float | **confidential** | dwd_item_level | price (agg:sum) |
| order_count | int | **restricted** | dwd_item_level | order_id (agg:nunique) |
| avg_review_score | float | **internal** | dwd_item_level | review_score (agg:avg) |
| late_delivery_rate | float | **confidential** | dwd_item_level | derived |

### Relationships
- seller → order (N:M): Seller fulfills orders through products
- seller → product (M:N): Seller stocks products

---

## Object 4: product (产品)

| Attribute | Value |
|-----------|-------|
| **object_type_id** | product |
| **display_name** | 产品 |
| **grain** | product_id |
| **owner_role** | category_ops |
| **retention_period** | 365 days |
| **allow_sync_to_feishu** | true (non-PII product data) |
| **purpose** | Product catalog analysis, pricing insights, category distribution |

### Property Markings

| Property | Type | Sensitivity | Source Table | Source Column |
|----------|------|-------------|-------------|---------------|
| product_id | string (PK) | **public** | dwd_item_level | product_id |
| product_category_name | string | **public** | dwd_item_level | product_category_name |
| product_category_name_english | string | **public** | dwd_item_level | product_category_name_english |
| price | float | **internal** | dwd_item_level | price |
| freight_value | float | **internal** | dwd_item_level | freight_value |
| product_weight_g | float | **public** | dwd_item_level | product_weight_g |
| sales_count | int | **internal** | dwd_item_level | order_id (agg:count) |
| avg_review_score | float | **public** | dwd_item_level | review_score (agg:avg) |

---

## Object 5: category (品类)

| Attribute | Value |
|-----------|-------|
| **object_type_id** | category |
| **display_name** | 品类 |
| **grain** | product_category_name |
| **owner_role** | category_ops |
| **retention_period** | 365 days |
| **purpose** | Category-level performance analysis, market trends |

### Property Markings

| Property | Type | Sensitivity | Source Table |
|----------|------|-------------|-------------|
| product_category_name | string (PK) | public | dwd_item_level |
| product_category_name_english | string | public | dwd_item_level |
| gmv | float | confidential | dwd_item_level (agg:sum) |
| order_count | int | internal | dwd_item_level (agg:nunique) |
| avg_review_score | float | public | dwd_item_level (agg:avg) |
| late_delivery_rate | float | confidential | dwd_item_level (derived) |

---

## Object 6: region (区域)

| Attribute | Value |
|-----------|-------|
| **object_type_id** | region |
| **display_name** | 区域 |
| **grain** | state |
| **owner_role** | business_ops |
| **retention_period** | 365 days |
| **purpose** | Regional market analysis, logistics optimization |

### Property Markings

| Property | Type | Sensitivity | Source Tables |
|----------|------|-------------|---------------|
| state | string (PK) | public | dwd_order_level, dwd_item_level |
| customer_count | int | internal | dwd_order_level (derived) |
| seller_count | int | internal | dwd_item_level (derived) |
| gmv | float | confidential | both tables (sum) |
| avg_review_score | float | public | both tables (avg) |
| avg_delivery_days | float | internal | dwd_order_level (derived) |

---

## Object 7: marketing_lead (营销线索)

| Attribute | Value |
|-----------|-------|
| **object_type_id** | marketing_lead |
| **display_name** | 营销线索 |
| **grain** | origin |
| **owner_role** | marketing_ops |
| **retention_period** | 365 days |
| **purpose** | Marketing channel effectiveness, conversion analysis |

### Property Markings

| Property | Type | Sensitivity | Source Table |
|----------|------|-------------|-------------|
| origin | string (PK) | public | channel_classification |
| mql_count | int | internal | channel_classification (derived) |
| conversion_count | int | internal | channel_classification (derived) |
| conversion_rate | float | confidential | channel_classification (derived) |
| gmv_per_seller | float | confidential | channel_classification (derived) |
| category | string | public | channel_classification |

---

## Object 8: metric_alert (异常事件)

| Attribute | Value |
|-----------|-------|
| **object_type_id** | metric_alert |
| **display_name** | 异常事件 |
| **grain** | alert_id |
| **owner_role** | owner_role (from alert_rules.yml) |
| **retention_period** | 365 days |
| **allow_sync_to_feishu** | true (alerts are Feishu targets) |
| **purpose** | Anomaly detection, operational alerts, decision triggers |

### Property Markings

| Property | Type | Sensitivity | Source Table |
|----------|------|-------------|-------------|
| alert_id | string (PK) | internal | alert_events |
| rule_id | string | internal | alert_events |
| metric | string | internal | alert_events |
| severity | string | internal | alert_events |
| current_value | float | internal | alert_events |
| baseline_value | float | internal | alert_events |
| owner_role | string | restricted | alert_events |
| status | string | internal | alert_events |

---

## Object 9: channel (渠道)

| Attribute | Value |
|-----------|-------|
| **object_type_id** | channel |
| **display_name** | 渠道 |
| **grain** | channel_name |
| **owner_role** | business_ops |
| **retention_period** | 365 days |
| **purpose** | Channel routing rules, dispatch configuration |

### Property Markings

| Property | Type | Sensitivity | Source |
|----------|------|-------------|--------|
| channel_name | string (PK) | internal | channel_routing_rules.yml |
| channel_type | string | internal | channel_routing_rules.yml |
| severity_routing | object | restricted | channel_routing_rules.yml |
| enabled | boolean | internal | adapter_registry.yml |

---

## Ontology Summary

| Object | Properties | PII Fields | Restricted Fields | Secret Fields | Owner |
|--------|-----------|------------|-------------------|---------------|-------|
| customer | 8 | 1 | 2 | 0 | business_ops |
| order | 7 | 0 | 0 | 1 | business_ops |
| seller | 7 | 0 | 2 | 1 | seller_ops |
| product | 8 | 0 | 0 | 0 | category_ops |
| category | 6 | 0 | 0 | 0 | category_ops |
| region | 6 | 0 | 0 | 0 | business_ops |
| marketing_lead | 6 | 0 | 0 | 0 | marketing_ops |
| metric_alert | 8 | 0 | 1 | 0 | (varies by owner_role) |
| channel | 4 | 0 | 1 | 0 | business_ops |

**Total**: 60 properties across 9 objects. PII fields: 1 (customer_unique_id). Restricted fields: 7. Confidential fields: 14.

This ontology model is the primary reference for all governance configurations. The `config/data_markings.yml` file contains the machine-readable version of these markings.
