# Baxi Access Control Model v0.5.3

> **Generated:** 2026-05-23
> **Purpose:** Define role-based access control for Baxi data and API endpoints
> **Config:** config/access_policy.yml
> **Related:** baxi_data_marking_policy.md, baxi_checkpoint_policy.md, owner_mapping.yml

## Current State (v0.5.3)

Baxi currently uses a **single bearer token** authentication model (HMAC constant-time comparison in api/auth.py). There is no role-based access control — any authenticated user can access all endpoints and data.

This document defines the **target access model** for Baxi, which will be implemented in future versions. In v0.5.3, the model is documented but not enforced.

## Proposed Access Levels

| Access Level | Description | Can Access |
|-------------|-------------|-----------|
| **admin** | Full access to all data, including PII and governance config | All endpoints, all data |
| **analyst** | Business analysis access, no PII, no governance config modification | Most endpoints, non-PII data |
| **viewer** | Read-only access to aggregated metrics and dashboards | Read-only GET endpoints, public data |

## Role Definitions (Based on owner_mapping.yml)

Baxi maps 5 owner roles to 3 owner types:

| owner_role (from owner_mapping.yml) | Business Domain | Recommended Access Level |
|-------------------------------------|-----------------|-------------------------|
| **business_ops** | 业务运营 — customer behavior, revenue, orders | admin |
| **seller_ops** | 卖家运营 — seller performance, quality metrics | analyst |
| **category_ops** | 品类运营 — category trends, product analysis | analyst |
| **logistics_ops** | 物流运营 — delivery, shipping, fulfillment | analyst |
| **marketing_ops** | 营销运营 — channel effectiveness, conversions | analyst |

## Endpoint Access Matrix

| Endpoint | Method | admin | analyst | viewer |
|----------|--------|-------|---------|--------|
| /api/v1/health | GET | ✅ | ✅ | ✅ |
| /api/v1/status | GET | ✅ | ✅ | ✅ |
| /api/v1/alerts | GET | ✅ | ✅ | ✅ |
| /api/v1/tasks | GET | ✅ | ✅ (own domain) | ❌ |
| /api/v1/outbox | GET | ✅ | ✅ | ❌ |
| /api/v1/outbox/dispatch | POST | ✅ | ✅ (with checkpoint) | ❌ |
| /api/v1/logs | GET | ✅ | ✅ | ❌ |
| /api/v1/feishu | GET | ✅ | ✅ (own domain) | ❌ |
| /api/v1/pipeline/run | POST | ✅ | ✅ | ❌ |
| /api/v1/diagnosis | GET | ✅ | ✅ | ❌ |
| /api/v1/governance/catalog | GET | ✅ | ✅ | ❌ |
| /api/v1/governance/lineage | GET | ✅ | ✅ | ✅ |
| /api/v1/governance/health | GET | ✅ | ✅ | ❌ |
| /api/v1/governance/checkpoints | GET | ✅ | ❌ | ❌ |

## Sensitivity-Based Access

Data access is gated by sensitivity level:

| Sensitivity Level | Access Level Required | Description |
|------------------|----------------------|-------------|
| L0 (Public) | viewer | Product metadata, category names |
| L1 (Internal) | analyst | Order status, review scores, timestamps |
| L2 (Confidential) | analyst | GMV, revenue, conversion rates |
| L3 (Restricted) | admin | Seller IDs, customer dates, owner roles |
| L4 (PII) | admin | customer_unique_id, personal identifiers |

### Row-Level Access (Future — v0.5.5+)

Following Palantir's restricted views pattern (参考: 10_restricted_views.md), future versions may implement:

- **Seller operator**: Can only see metrics for sellers in their assigned region
- **Category manager**: Can only see data for categories in their portfolio
- **Marketing lead**: Can only see campaign data for their channel

This would require adding user-to-domain mapping in a new config file and implementing row-level filtering at the API middleware layer.

## Feishu User Access

The feishu_user_mapping.yml file maps Feishu users to internal roles:

| feishu_user_type | Description | Recommended Access Level |
|-----------------|-------------|-------------------------|
| business_admin | 业务管理员 | admin |
| business_ops | 业务运营人员 | analyst |
| seller_ops | 卖家运营人员 | analyst |
| category_ops | 品类运营人员 | analyst |
| logistics_ops | 物流运营人员 | analyst |
| marketing_ops | 营销运营人员 | analyst |
| exec_viewer | 高管查看者 | viewer |

## Access Control Enforcement Plan

### v0.5.3 — Advisory Only
- Access policy is defined in YAML
- API endpoints log access attempts with user identity
- Governance Center UI displays access requirements but doesn't enforce them

### v0.5.4 — Basic Enforcement
- API middleware checks user role against access_policy.yml
- Rejects requests where user role < required level
- Returns 403 Forbidden with error contract format

### v0.5.5 — Full RBAC + Row-Level
- Per-user role assignment (multi-token support)
- Row-level filtering based on user domain assignment
- Restricted view pattern for multi-tenant data isolation

## Configuration Structure

```yaml
access_policy:
  roles:
    admin:
      description: "管理员"
      can_access_sensitivity: [L0, L1, L2, L3, L4]
      can_modify_governance: true
      
    analyst:
      description: "分析师"
      can_access_sensitivity: [L0, L1, L2]
      can_modify_governance: false
      
    viewer:
      description: "查看者"
      can_access_sensitivity: [L0, L1]
      can_modify_governance: false
      
  endpoint_policy:
    - endpoint: /api/v1/governance/catalog
      method: GET
      min_access_level: admin
      
    - endpoint: /api/v1/governance/lineage
      method: GET
      min_access_level: viewer
      
    # ... all endpoints defined above
    
  owner_role_mapping:
    business_ops: admin
    seller_ops: analyst
    category_ops: analyst
    logistics_ops: analyst
    marketing_ops: analyst
```
