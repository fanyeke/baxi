# Phase 5: Governance / Ontology Runtime Plan

## 1. Overview

Phase 5 builds a read-only runtime layer over the 29 YAML governance configuration files and the 8 AIP object types. This runtime makes governance metadata (classification, lineage, access policy, health checks, checkpoints) and ontology data (object schemas, property definitions, relationships) available through well-typed Go services, a PostgreSQL-backed config registry, and a set of Governance API endpoints.

The Phase 4 Go API migration deferred all governance config reader endpoints (S06-S11 in the Phase 4 plan). Phase 5 implements them properly: instead of reading YAML files from disk on every request, YAML content is loaded into PostgreSQL `gov.*` tables at startup and served through defined Go structs and query services.

### Guiding Principle

Phase 5 is a **read-only runtime**. It does not write to business data tables (ops.*, dwd.*, mart.*, audit.*). It does not execute actions, make LLM calls, or modify YAML files. All governance state is loaded at startup and refreshed through explicit sync operations only.

### Baseline Reference

- **Code freeze tag**: `v0.5.3-python-sqlite-freeze`
- **Migration branch**: `migration/go-postgres`
- **YAML config directory**: `config/` (29 YAML files)
- **PostgreSQL gov schema**: `gov.*` (7 tables)

---

## 2. YAML to Runtime Mapping

Each YAML configuration file maps to a Go runtime module and a `gov.*` PostgreSQL table. The mapping is one-to-one for config files that have dedicated tables; secondary config files (rules, policies without dedicated tables) share the `gov.config_snapshot` catch-all table.

### Mapping Table

| YAML File | Go Package / Module | PostgreSQL Table | Config Key | Loaded By |
|---|---|---|---|---|
| `aip_object_schema.yml` | `internal/ontology/registry.go` (ObjectRegistry) | `gov.object_schema` | `object_schema` | ConfigLoader |
| `data_catalog.yml` | `internal/ontology/registry.go` (ObjectRegistry) | `gov.object_schema` | `data_catalog` | ConfigLoader |
| `data_classification.yml` | `internal/service/governance_service.go` (GovernanceService) | `gov.data_classification` | `data_classification` | ConfigLoader |
| `data_markings.yml` | `internal/service/governance_service.go` (GovernanceService) | `gov.config_snapshot` | `data_markings` | ConfigLoader |
| `data_lineage.yml` | `internal/service/governance_service.go` (GovernanceService) | `gov.data_lineage` | `data_lineage` | ConfigLoader |
| `access_policy.yml` | `internal/service/governance_service.go` (GovernanceService) | `gov.access_policy` | `access_policy` | ConfigLoader |
| `checkpoint_rules.yml` | `internal/service/governance_service.go` (GovernanceService) | `gov.config_snapshot` | `checkpoint_rules` | ConfigLoader |
| `health_checks.yml` | `internal/service/governance_service.go` (GovernanceService) | `gov.config_snapshot` | `health_checks` | ConfigLoader |
| `retention_policies.yml` | `internal/service/governance_service.go` (GovernanceService) | `gov.config_snapshot` | `retention_policies` | ConfigLoader |
| `decision_eval_rules.yml` | Phase 7 (decision engine) | `gov.config_snapshot` | `decision_eval_rules` | ConfigLoader |
| `alert_rules.yml` | `internal/service/governance_service.go` (via AlertRuleRegistry) | `gov.config_snapshot` | `alert_rules` | ConfigLoader |
| `adapter_registry.yml` | Phase 6 (dispatch) | `gov.config_snapshot` | `adapter_registry` | ConfigLoader |
| `channel_routing_rules.yml` | Phase 6 (dispatch) | `gov.config_snapshot` | `channel_routing_rules` | ConfigLoader |
| `action_registry.yml` | Phase 6 (dispatch) | `gov.config_snapshot` | `action_registry` | ConfigLoader |
| `action_templates.yml` | Phase 7 (decision engine) | `gov.config_snapshot` | `action_templates` | ConfigLoader |
| `dimensional_alert_rules.yml` | Phase 7 (decision engine) | `gov.config_snapshot` | `dimensional_alert_rules` | ConfigLoader |
| `data_quality_rules.yml` | Phase 7 (decision engine) | `gov.config_snapshot` | `data_quality_rules` | ConfigLoader |
| `metrics.yml` | Phase 7 (decision engine) | `gov.config_snapshot` | `metrics` | ConfigLoader |
| `status_enums.yml` | `internal/service/governance_service.go` | `gov.config_snapshot` | `status_enums` | ConfigLoader |
| `owner_mapping.yml` | `internal/service/governance_service.go` | `gov.config_snapshot` | `owner_mapping` | ConfigLoader |
| `llm_config.yml` | Phase 7 (decision engine) | `gov.config_snapshot` | `llm_config` | ConfigLoader |
| `qoder_capabilities.yml` | Static in `internal/api/dto/qoder.go` | (embedded in code) | `qoder_capabilities` | Not loaded (static) |
| `feishu_app.yml` | Phase 8 (Feishu integration) | `gov.config_snapshot` | `feishu_app` | ConfigLoader |
| `feishu_base_schema.yml` | Phase 8 (Feishu integration) | `gov.config_snapshot` | `feishu_base_schema` | ConfigLoader |
| `feishu_field_mapping.yml` | Phase 8 (Feishu integration) | `gov.config_snapshot` | `feishu_field_mapping` | ConfigLoader |
| `feishu_table_ids.yml` | Phase 8 (Feishu integration) | `gov.config_snapshot` | `feishu_table_ids` | ConfigLoader |
| `feishu_user_mapping.yml` | Phase 8 (Feishu integration) | `gov.config_snapshot` | `feishu_user_mapping` | ConfigLoader |
| `wake_io_contract.yml` | Phase 6 (dispatch) | `gov.config_snapshot` | `wake_io_contract` | ConfigLoader |

### YAML Files for Phase 5 Governance API Endpoints

| Endpoint | Primary YAML | Secondary YAMLs |
|---|---|---|
| `GET /api/v1/governance/catalog` | `aip_object_schema.yml` | `data_catalog.yml`, `retention_policies.yml` |
| `GET /api/v1/governance/classification` | `data_classification.yml` | — |
| `GET /api/v1/governance/markings` | `data_markings.yml` | — |
| `GET /api/v1/governance/lineage` | `data_lineage.yml` | — |
| `GET /api/v1/governance/checkpoints` | `checkpoint_rules.yml` | `governance_checkpoints` (sqlite table) |
| `GET /api/v1/governance/health` | `health_checks.yml` | `governance_health_results` (sqlite table) |

---

## 3. ConfigLoader Design

The ConfigLoader is the bridge between YAML files on disk and PostgreSQL `gov.*` tables. It runs at application startup (no hot reload in Phase 5) and can be triggered manually via the `SyncSnapshots` method.

### ConfigRegistry Struct

```go
// Package configloader manages loading YAML governance configs into PostgreSQL gov.* tables.
package configloader

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "os"
    "path/filepath"

    "github.com/jackc/pgx/v5/pgxpool"
    "gopkg.in/yaml.v3"
)

// ConfigEntry represents a loaded YAML file for upsert into gov.config_snapshot.
type ConfigEntry struct {
    ConfigKey    string `json:"config_key"`      // e.g. "data_classification"
    ContentHash  string `json:"content_hash"`    // SHA256 of raw YAML content
    ContentJSON  string `json:"content_json"`    // YAML converted to JSON for storage
    FilePath     string `json:"file_path"`       // Relative path to YAML file
    Status       string `json:"status"`          // "loaded" or "error"
    ErrorMessage string `json:"error_message,omitempty"`
}

// ConfigRegistry holds all loaded governance configuration entries.
type ConfigRegistry struct {
    entries map[string]*ConfigEntry  // key: config_key
}

func NewConfigRegistry() *ConfigRegistry {
    return &ConfigRegistry{
        entries: make(map[string]*ConfigEntry),
    }
}

// LoadAll scans config/ directory, parses YAML, computes SHA256 hashes,
// and populates the in-memory registry.
func (r *ConfigRegistry) LoadAll(configDir string) error { ... }

// Get returns a ConfigEntry by key. Returns nil if not found.
func (r *ConfigRegistry) Get(key string) *ConfigEntry { ... }

// Entries returns all loaded config entries.
func (r *ConfigRegistry) Entries() []*ConfigEntry { ... }

// SyncSnapshots upserts all ConfigEntry values into gov.config_snapshot.
// It does NOT delete entries that exist in the DB but not in the registry.
func (r *ConfigRegistry) SyncSnapshots(ctx context.Context, pool *pgxpool.Pool) error { ... }
```

### LoadAll Method

```
LoadAll(configDir)
  │
  ├── List all *.yml files in configDir
  │   (excluding *.yml.example files)
  │
  ├── For each file:
  │     ├── Read raw content
  │     ├── Compute SHA256(content) → content_hash
  │     ├── Unmarshal YAML → validate structure
  │     ├── Marshal to JSON → content_json
  │     ├── Derive config_key from filename:
  │     │   aip_object_schema.yml → "object_schema"
  │     │   data_classification.yml → "data_classification"
  │     │   (strip .yml, strip config/ prefix)
  │     └── Store in ConfigRegistry.entries[key]
  │
  └── Return error count (non-fatal per file)
```

### SyncSnapshots SQL

```sql
-- Upsert into gov.config_snapshot
INSERT INTO gov.config_snapshot (config_key, content_hash, content_json, status, loaded_at)
VALUES ($1, $2, $3, 'loaded', NOW())
ON CONFLICT (config_key) DO UPDATE SET
    content_hash = EXCLUDED.content_hash,
    content_json = EXCLUDED.content_json,
    status = 'loaded',
    loaded_at = NOW();
```

### gov.config_snapshot Schema

```sql
CREATE TABLE gov.config_snapshot (
    config_key   TEXT PRIMARY KEY,
    content_hash TEXT NOT NULL,           -- SHA256 hex digest
    content_json JSONB NOT NULL,          -- YAML converted to JSON
    status       TEXT NOT NULL DEFAULT 'loaded',  -- loaded | error | stale
    error_msg    TEXT,
    loaded_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### SHA256 Hash Strategy

- **Input**: Raw bytes of the YAML file on disk.
- **Output**: 64-character lowercase hex string.
- **Purpose**: Detect content changes between sync runs (if a config file has not changed, skip the upsert).
- **Edge case**: YAML comments and whitespace affect the hash. This is intentional: if a developer edits comments, the hash changes, which triggers a re-sync. For production, strip comments before hashing if needed (deferred to Phase 6+).

### Initialization Sequence

```
main.go startup
  │
  ├── 1. Connect to PostgreSQL
  ├── 2. Create ConfigRegistry
  ├── 3. registry.LoadAll("config/")
  │       └── Log warnings for parse errors (non-fatal)
  ├── 4. registry.SyncSnapshots(ctx, pool)
  │       └── Upsert each entry into gov.config_snapshot
  ├── 5. Initialize ObjectRegistry from gov.object_schema / config data
  ├── 6. Initialize GovernanceService from gov.* tables
  ├── 7. Initialize GovernanceHandler
  └── 8. Start HTTP server
```

---

## 4. ObjectRegistry Design

The ObjectRegistry provides typed access to the 8 AIP object types defined in `aip_object_schema.yml` and `data_catalog.yml`. It is the single source of truth for object metadata at runtime.

### ObjectType Struct

```go
package ontology

// ObjectType represents one AIP object type from the YAML schema.
type ObjectType struct {
    ObjectTypeID     string                  // e.g. "customer", "order"
    DisplayName      string                  // e.g. "客户", "订单"
    Grain            string                  // e.g. "customer_unique_id"
    SourceTables     []string                // e.g. ["order_level_base"]
    Properties       map[string]PropertyDef  // key: property name
    Links            []ObjectLink            // Relationships to other objects
    AllowedActions   []string                // From access_policy for this role/object
    LLMAccess        LLMAccessConfig         // What LLM can see/do with this object
}

// PropertyDef defines a single property of an object type.
type PropertyDef struct {
    Type      string   `yaml:"type"`       // string, int, float, datetime, bool
    IsPK      bool     `yaml:"is_pk,omitempty"`
    Source    string   `yaml:"source,omitempty"`   // Source column name
    Agg       string   `yaml:"agg,omitempty"`       // Aggregation function
    Sensitive bool     `yaml:"sensitive,omitempty"` // Is this PII/sensitive?
    Marking   string   `yaml:"marking,omitempty"`   // Data marking label
}

// ObjectLink represents a relationship to another object type.
type ObjectLink struct {
    Name      string // e.g. "has_items"
    ToType    string // e.g. "order"
    Grain     string // e.g. "order_id"
    ViaColumn string // e.g. "order_id" (join column in current object)
}

// LLMAccessConfig defines what the LLM (Qoder) can access on this object.
type LLMAccessConfig struct {
    CanQuery       bool     // Can LLM query this object type?
    AllowedFields  []string // Fields the LLM can see (empty = all)
    MaxReturnRows  int      // Maximum rows returned (default 1000)
    RequiresMarking []string // Markings required for LLM access
}

// ObjectRegistry provides lookup of object types by ID.
type ObjectRegistry struct {
    objects map[string]*ObjectType  // key: object_type_id
}

func NewObjectRegistry() *ObjectRegistry { ... }

// Register adds an ObjectType to the registry.
func (r *ObjectRegistry) Register(ot *ObjectType) { ... }

// Get returns an ObjectType by ID. Returns nil if not found.
func (r *ObjectRegistry) Get(objectTypeID string) *ObjectType { ... }

// List returns all registered object types.
func (r *ObjectRegistry) List() []*ObjectType { ... }

// ResolveLink follows a relationship from one object to another.
func (r *ObjectRegistry) ResolveLink(fromType, linkName string) *ObjectLink { ... }
```

### Eight Object Types

| # | ObjectTypeID | Grain | Source Tables | Properties | Relationships |
|---|---|---|---|---|---|
| 1 | `customer` | `customer_unique_id` | `order_level_base` | 8 properties (customer_unique_id, customer_state, customer_city, order_count, gmv_total, avg_review_score, first_order_date, last_order_date) | — |
| 2 | `order` | `order_id` | `order_level_base` | 7 properties (order_id, order_status, order_purchase_timestamp, total_payment_value, payment_type, review_score, delivery_status) | — |
| 3 | `seller` | `seller_id` | `item_level_base` | 7 properties + 2 relationships | has_items → order, has_products → product |
| 4 | `product` | `product_id` | `item_level_base` | 8 properties | — |
| 5 | `category` | `product_category_name` | `item_level_base` | 6 properties | — |
| 6 | `region` | `state` | `order_level_base`, `item_level_base` | 6 properties | — |
| 7 | `marketing_lead` | `origin` | `channel_classification` | 6 properties | — |
| 8 | `metric_alert` | `alert_id` | `metric_alerts` | 8 properties | — |

### ObjectRegistry Initialization

The ObjectRegistry is populated from `gov.object_schema` (which holds the parsed contents of `aip_object_schema.yml` and `data_catalog.yml`):

```
ObjectRegistry.LoadFromDB(ctx, pool)
  │
  ├── Query: SELECT object_type_id, schema_json FROM gov.object_schema
  ├── For each row:
  │     ├── Unmarshal schema_json into ObjectType
  │     ├── Enrich with data_catalog info (sensitivity, owner_role, retention)
  │     ├── Enrich with access_policy allowed_actions (role-dependent runtime)
  │     ├── Set LLMAccess defaults (can_query: true, max_return_rows: 1000)
  │     └── Register in registry
  └── Return registry
```

### gov.object_schema Schema

```sql
CREATE TABLE gov.object_schema (
    object_type_id TEXT PRIMARY KEY,
    schema_json    JSONB NOT NULL,         -- Full ObjectType definition
    sensitivity    TEXT,                    -- L0-L4 from data_catalog
    owner_role     TEXT,                    -- e.g. "business_ops"
    retention_days INTEGER,
    loaded_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## 5. ObjectQueryService Design

The ObjectQueryService provides read-only access to AIP object data from the `dwd.*`, `mart.*`, and `ops.*` tables. It is the service layer between the Governance API handlers and the PostgreSQL data tables.

### Design Principle: Explicit Per-Type Methods

ObjectQueryService does NOT use generic SQL builders or dynamic query generation. Each AIP object type has its own explicit query method. This prevents SQL injection, enables compile-time type safety, and makes each query independently optimizable.

```go
package service

// ObjectQueryService provides read-only queries for AIP object types.
// Each AIP object type has its own explicit method.
type ObjectQueryService struct {
    pool *pgxpool.Pool
}

func NewObjectQueryService(pool *pgxpool.Pool) *ObjectQueryService { ... }

// ListCustomers returns customer objects. Used by ObjectQueryService handler.
func (s *ObjectQueryService) ListCustomers(ctx context.Context, role string, limit int, offset int) ([]dto.CustomerObject, int, error) {
    // Read from dwd_order_level with aggregation
    // Apply role-based filtering (analyst cannot see PII fields)
    // Default LIMIT 1000
    ...
}

// ListOrders returns order objects.
func (s *ObjectQueryService) ListOrders(ctx context.Context, role string, limit int, offset int) ([]dto.OrderObject, int, error) { ... }

// ListSellers returns seller objects.
func (s *ObjectQueryService) ListSellers(ctx context.Context, role string, limit int, offset int) ([]dto.SellerObject, int, error) { ... }

// ListProducts returns product objects.
func (s *ObjectQueryService) ListProducts(ctx context.Context, role string, limit int, offset int, categoryFilter string) ([]dto.ProductObject, int, error) { ... }

// ListCategories returns category objects.
func (s *ObjectQueryService) ListCategories(ctx context.Context, role string, limit int, offset int) ([]dto.CategoryObject, int, error) { ... }

// ListRegions returns region objects.
func (s *ObjectQueryService) ListRegions(ctx context.Context, role string, limit int, offset int) ([]dto.RegionObject, int, error) { ... }

// ListMarketingLeads returns marketing lead objects.
func (s *ObjectQueryService) ListMarketingLeads(ctx context.Context, role string, limit int, offset int) ([]dto.MarketingLeadObject, int, error) { ... }

// ListMetricAlerts returns metric alert objects.
func (s *ObjectQueryService) ListMetricAlerts(ctx context.Context, role string, limit int, offset int) ([]dto.MetricAlertObject, int, error) { ... }

// GetByID retrieves a single object by its grain/primary key value.
func (s *ObjectQueryService) GetByID(ctx context.Context, objectTypeID string, id string, role string) (interface{}, error) {
    // Route to typed method based on objectTypeID
    // Return 404 if not found
    // Apply role-based redaction
    ...
}
```

### OntologyRepository Mapping

```go
package repository

// OntologyRepository provides data access for ontology object queries.
type OntologyRepository struct{}

// ObjectType → SQL Source Mapping:
//
// customer       → SELECT ... FROM mart.customer_summary
// order          → SELECT ... FROM mart.order_summary
// seller         → SELECT ... FROM mart.seller_summary
// product        → SELECT ... FROM mart.product_summary
// category       → SELECT ... FROM mart.category_summary
// region         → SELECT ... FROM mart.region_summary
// marketing_lead → SELECT ... FROM mart.marketing_lead_summary
// metric_alert   → SELECT ... FROM ops.metric_alert
```

### Default LIMIT 1000

All list methods enforce a default limit of 1000 rows:

```go
const DefaultObjectQueryLimit = 1000
const MaxObjectQueryLimit = 10000
```

### Role-Based Access in ObjectQueryService

Each query method accepts a `role` parameter and applies field-level filtering based on the role's `data_access` from `access_policy.yml`:

| Role | Can Access |
|---|---|
| `admin` | All fields on all objects |
| `analyst` | dwd_item_level, metric_daily, metric_dimension_daily, alert_events (no PII) |
| `viewer` | metric_daily only |
| `marketing_ops` | metric_daily, dwd_order_level (no PII) |

---

## 6. GovernanceService Design

The GovernanceService manages read-only access to governance metadata. It extends the existing `GovernanceService` (which currently only serves the `/governance/status` endpoint) with methods for each governance domain.

### Existing GovernanceService (Phase 4)

```go
// internal/service/governance_service.go
type GovernanceService struct {
    repo *repository.GovernanceRepository
    pool *pgxpool.Pool
}

func (s *GovernanceService) GetStatus(ctx context.Context) (*dto.GovernanceStatusResponse, error) { ... }
```

### Phase 5 Extended GovernanceService

```go
package service

// GovernanceService handles business logic for governance operations.
// Phase 5 adds read-only query methods for all governance domains.
type GovernanceService struct {
    repo        *repository.GovernanceRepository
    pool        *pgxpool.Pool
    configReg   *configloader.ConfigRegistry
    objectReg   *ontology.ObjectRegistry
}

func NewGovernanceService(
    repo *repository.GovernanceRepository,
    pool *pgxpool.Pool,
    configReg *configloader.ConfigRegistry,
    objectReg *ontology.ObjectRegistry,
) *GovernanceService { ... }

// GetConfigSnapshot returns the raw JSON content for a given config key.
func (s *GovernanceService) GetConfigSnapshot(ctx context.Context, configKey string) (*dto.ConfigSnapshotResponse, error) { ... }

// GetClassification returns the data classification entries.
// Reads from gov.data_classification or falls back to ConfigRegistry.
func (s *GovernanceService) GetClassification(ctx context.Context) (*dto.ClassificationResponse, error) { ... }

// GetLineage returns the data lineage graph (nodes + edges).
// Reads from gov.data_lineage.
func (s *GovernanceService) GetLineage(ctx context.Context) (*dto.LineageResponse, error) { ... }

// GetAccessPolicy returns the role-based access policy.
// Reads from gov.access_policy.
func (s *GovernanceService) GetAccessPolicy(ctx context.Context) (*dto.AccessPolicyResponse, error) { ... }

// GetCheckpoints returns checkpoint history.
// Reads from governance_checkpoints table.
func (s *GovernanceService) GetCheckpoints(ctx context.Context, limit, offset int) (*dto.CheckpointListResponse, error) { ... }

// GetHealthChecks returns health check results.
// Reads from governance_health_results table.
func (s *GovernanceService) GetHealthChecks(ctx context.Context) (*dto.HealthCheckListResponse, error) { ... }

// GetCatalog returns the full data catalog (objects + tables + endpoints).
// Merges data from object_schema, data_catalog, and retention_policies.
func (s *GovernanceService) GetCatalog(ctx context.Context) (*dto.CatalogResponse, error) { ... }

// GetMarkings returns the data marking definitions.
// Reads from gov.config_snapshot where config_key = 'data_markings'.
func (s *GovernanceService) GetMarkings(ctx context.Context) (*dto.MarkingsResponse, error) { ... }
```

### GovernanceRepository Extensions (Phase 5)

```go
package repository

// GovernanceRepository provides read-only access to gov.* tables.
// Phase 5 adds query methods for each governance domain.

// GetClassificationRows queries gov.data_classification.
func (r *GovernanceRepository) GetClassificationRows(ctx context.Context, pool *pgxpool.Pool) ([]ClassificationRow, error) { ... }

// GetLineageNodes queries gov.data_lineage for lineage nodes.
func (r *GovernanceRepository) GetLineageNodes(ctx context.Context, pool *pgxpool.Pool) ([]LineageNode, error) { ... }

// GetLineageEdges queries gov.data_lineage for lineage edges.
func (r *GovernanceRepository) GetLineageEdges(ctx context.Context, pool *pgxpool.Pool) ([]LineageEdge, error) { ... }

// GetAccessPolicyRows queries gov.access_policy.
func (r *GovernanceRepository) GetAccessPolicyRows(ctx context.Context, pool *pgxpool.Pool) ([]AccessPolicyRow, error) { ... }

// GetObjectSchema queries gov.object_schema.
func (r *GovernanceRepository) GetObjectSchema(ctx context.Context, pool *pgxpool.Pool) ([]ObjectSchemaRow, error) { ... }

// GetConfigSnapshotByKey queries gov.config_snapshot by config_key.
func (r *GovernanceRepository) GetConfigSnapshotByKey(ctx context.Context, pool *pgxpool.Pool, configKey string) (*ConfigSnapshotRow, error) { ... }
```

### gov.* Table Schema Summary

```sql
-- gov.config_snapshot: stores all YAML configs as JSONB
CREATE TABLE gov.config_snapshot (
    config_key   TEXT PRIMARY KEY,
    content_hash TEXT NOT NULL,
    content_json JSONB NOT NULL,
    status       TEXT NOT NULL DEFAULT 'loaded',
    error_msg    TEXT,
    loaded_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- gov.object_schema: parsed object type definitions
CREATE TABLE gov.object_schema (
    object_type_id TEXT PRIMARY KEY,
    schema_json    JSONB NOT NULL,
    sensitivity    TEXT,
    owner_role     TEXT,
    retention_days INTEGER,
    loaded_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- gov.data_classification: parsed classification entries
CREATE TABLE gov.data_classification (
    asset_ref    TEXT PRIMARY KEY,
    level        TEXT NOT NULL,
    rationale    TEXT,
    field_levels JSONB,        -- {"field_name": "level", ...}
    loaded_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- gov.data_lineage: parsed lineage graph
CREATE TABLE gov.data_lineage (
    node_id   TEXT PRIMARY KEY,
    node_type TEXT NOT NULL,   -- source, dataset, object_type, sync
    label     TEXT,
    status    TEXT,
    loaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE gov.data_lineage_edge (
    edge_id        SERIAL PRIMARY KEY,
    from_node_id   TEXT NOT NULL REFERENCES gov.data_lineage(node_id),
    to_node_id     TEXT NOT NULL REFERENCES gov.data_lineage(node_id),
    transform      TEXT,
    transform_type TEXT,
    loaded_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- gov.access_policy: parsed role definitions
CREATE TABLE gov.access_policy (
    role           TEXT PRIMARY KEY,
    allowed_actions TEXT[] NOT NULL,
    data_access    TEXT[] NOT NULL,
    loaded_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## 7. LLM-Safe Redaction Strategy

The LLM-safe redaction strategy ensures that sensitive data is stripped from responses before they reach the Qoder AI engine, the Governance API, or any user-facing endpoint.

### Classification-Based Stripping

Data classification levels (`data_classification.yml`) define which fields are sensitive:

| Classification Level | Redaction Behavior |
|---|---|
| `pii` | **Always stripped**. Never returned in API responses. |
| `sensitive` | Stripped unless requester role is `admin`. |
| `internal` | Stripped unless requester role is `admin` or `analyst`. |
| `derived_sensitive` | Stripped unless requester role is `admin`. |
| `public_internal` | Always visible. No redaction. |

### Marking-Based Stripping

Data markings (`data_markings.yml`) define mandatory controls:

| Marking | Redaction Behavior |
|---|---|
| `PII` | Always stripped. Applies to raw_customers.customer_unique_id, raw_orders.customer_id, dwd_order_level.customer_unique_id. |
| `FINANCIAL_INTERNAL` | Stripped unless admin. Applies to payment_value, price, freight_value. |
| `OPERATIONAL_INTERNAL` | Stripped unless admin or analyst. Applies to alert_events, action_tasks, event_outbox. |
| `RAW_DATA` | Stripped unless admin. Applies to all raw_* tables. |

### Role-Based Access in Redaction

The redaction function takes a `role` parameter and applies both classification and marking rules:

```go
package redaction

// RedactionRule defines what to redact based on role and classification.
type RedactionRule struct {
    ClassificationLevel string   // pii, sensitive, internal, derived_sensitive, public_internal
    RequiredRoles       []string // roles that can see this data
    Marking             string   // PII, FINANCIAL_INTERNAL, etc.
}

// RedactionLogEntry records a redaction event for audit.
type RedactionLogEntry struct {
    Timestamp    string
    RequestID    string
    ObjectType   string
    Field        string
    Reason       string   // "classification:pii" or "marking:FINANCIAL_INTERNAL"
    ActorRole    string
}

// FieldRedactor strips sensitive fields from a response map.
// Returns the redacted map and a list of redaction log entries.
func FieldRedactor(role string) func(data map[string]interface{}) (map[string]interface{}, []RedactionLogEntry) { ... }
```

### Redaction Log

Every redaction event is logged to `gov.redaction_log`:

```sql
CREATE TABLE gov.redaction_log (
    log_id      SERIAL PRIMARY KEY,
    request_id  TEXT NOT NULL,
    timestamp   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    object_type TEXT,
    field_name  TEXT NOT NULL,
    reason      TEXT NOT NULL,     -- classification:pii, marking:FINANCIAL_INTERNAL, role:viewer
    actor_role  TEXT NOT NULL
);
```

### Redaction in Practice

```
API Request → handler.go
  │
  ├── Extract role from auth context (default: "analyst")
  │
  ├── Call service layer → Get response data
  │
  ├── Apply FieldRedactor(role) to response:
  │     ├── Check each field against classification levels
  │     ├── Check each field against data markings
  │     ├── Strip sensitive fields (set to null in JSON)
  │     └── Log redaction events to gov.redaction_log
  │
  └── Return redacted response to caller
```

Example: An `analyst` requests customer data. The redactor strips `customer_unique_id` (PII) and `payment_value` (FINANCIAL_INTERNAL) from the response, while keeping `customer_state`, `order_count`, and `gmv_total` visible.

---

## 8. Governance API Scope

Phase 5 defines 6 new Governance API endpoints. These endpoints are read-only and serve governance metadata from the `gov.*` tables.

### Endpoint Summary

| # | Method | Path | Source | Handler Method |
|---|---|---|---|---|
| G01 | GET | `/api/v1/governance/catalog` | `gov.object_schema` + `gov.config_snapshot` | `HandleCatalog` |
| G02 | GET | `/api/v1/governance/classification` | `gov.data_classification` | `HandleClassification` |
| G03 | GET | `/api/v1/governance/markings` | `gov.config_snapshot` (key: data_markings) | `HandleMarkings` |
| G04 | GET | `/api/v1/governance/lineage` | `gov.data_lineage` + `gov.data_lineage_edge` | `HandleLineage` |
| G05 | GET | `/api/v1/governance/checkpoints` | `governance_checkpoints` table | `HandleCheckpoints` |
| G06 | GET | `/api/v1/governance/health` | `governance_health_results` table | `HandleHealth` |

### Endpoint Details

#### G01: GET /api/v1/governance/catalog

Returns the full data catalog including object types, tables, API endpoints, and retention policies.

```json
{
  "objects": [
    {
      "object_type_id": "customer",
      "display_name": "客户",
      "sensitivity": "L4",
      "owner_role": "business_ops",
      "grain": "customer_unique_id",
      "property_count": 8,
      "pii_field_count": 1,
      "retention_days": 730
    }
  ],
  "tables": [
    {
      "table_name": "dwd_order_level",
      "sensitivity": "L4",
      "column_count": 22,
      "grain": "order_id"
    }
  ],
  "endpoints": [
    {
      "endpoint": "/api/v1/alerts",
      "method": "GET",
      "sensitivity": "L1"
    }
  ]
}
```

#### G02: GET /api/v1/governance/classification

Returns the data classification definitions.

```json
{
  "classifications": [
    {
      "asset_ref": "raw_customers",
      "level": "pii",
      "rationale": "Contains customer_unique_id, a direct PII identifier",
      "applies_to_fields": {
        "customer_unique_id": "pii"
      }
    }
  ]
}
```

#### G03: GET /api/v1/governance/markings

Returns the data marking definitions (mandatory access controls).

```json
{
  "markings": {
    "PII": {
      "mandatory_control": true,
      "access_type": "binary",
      "conjunctive": true,
      "applies_to": ["raw_customers.customer_unique_id", ...],
      "policy": "Do not expose in frontend or Feishu export"
    }
  },
  "pipeline_stage_markings": [...]
}
```

#### G04: GET /api/v1/governance/lineage

Returns the full data lineage graph.

```json
{
  "nodes": [
    {"id": "raw_orders_csv", "type": "source", "label": "olist_orders_dataset.csv", "status": "active"},
    {"id": "dwd_order_level", "type": "dataset", "label": "dwd_order_level (1r/order, 22c)", "status": "active"}
  ],
  "edges": [
    {"from": "raw_orders_csv", "to": "dwd_order_level", "transform": "CSV ingestion: orders + payments + reviews", "transform_type": "batch_load"}
  ]
}
```

#### G05: GET /api/v1/governance/checkpoints

Returns checkpoint history.

```json
{
  "items": [
    {
      "checkpoint_id": 1,
      "action_type": "outbox_dispatch_apply",
      "endpoint": "POST /api/v1/outbox/dispatch",
      "actor": "qoder",
      "status": "recorded",
      "created_at": "2026-05-25T10:30:00Z"
    }
  ],
  "total": 42
}
```

#### G06: GET /api/v1/governance/health

Returns health check results.

```json
{
  "items": [
    {
      "check_id": "classification_coverage",
      "check_type": "governance",
      "status": "pass",
      "detail": "All 28 SQLite tables have a classification entry",
      "checked_at": "2026-05-25T10:00:00Z"
    }
  ]
}
```

### GovernanceHandler Extension

```go
package handler

// GovernanceHandler handles HTTP requests for governance-related endpoints.
// Phase 5 adds 6 new handler methods.
type GovernanceHandler struct {
    svc GovernanceProvider
}

// GovernanceProvider is the interface for governance operations.
type GovernanceProvider interface {
    GetStatus(ctx context.Context) (*dto.GovernanceStatusResponse, error)
    GetCatalog(ctx context.Context) (*dto.CatalogResponse, error)
    GetClassification(ctx context.Context) (*dto.ClassificationResponse, error)
    GetMarkings(ctx context.Context) (*dto.MarkingsResponse, error)
    GetLineage(ctx context.Context) (*dto.LineageResponse, error)
    GetCheckpoints(ctx context.Context, limit, offset int) (*dto.CheckpointListResponse, error)
    GetHealthChecks(ctx context.Context) (*dto.HealthCheckListResponse, error)
}

func NewGovernanceHandler(svc GovernanceProvider) *GovernanceHandler { ... }
func (h *GovernanceHandler) HandleGovernanceStatus(w http.ResponseWriter, r *http.Request) { ... }
func (h *GovernanceHandler) HandleCatalog(w http.ResponseWriter, r *http.Request) { ... }
func (h *GovernanceHandler) HandleClassification(w http.ResponseWriter, r *http.Request) { ... }
func (h *GovernanceHandler) HandleMarkings(w http.ResponseWriter, r *http.Request) { ... }
func (h *GovernanceHandler) HandleLineage(w http.ResponseWriter, r *http.Request) { ... }
func (h *GovernanceHandler) HandleCheckpoints(w http.ResponseWriter, r *http.Request) { ... }
func (h *GovernanceHandler) HandleHealth(w http.ResponseWriter, r *http.Request) { ... }
```

### Route Registration

```go
// internal/api/router.go
r.Route("/api/v1/governance", func(r chi.Router) {
    r.Get("/catalog", govHandler.HandleCatalog)
    r.Get("/classification", govHandler.HandleClassification)
    r.Get("/markings", govHandler.HandleMarkings)
    r.Get("/lineage", govHandler.HandleLineage)
    r.Get("/checkpoints", govHandler.HandleCheckpoints)
    r.Get("/health", govHandler.HandleHealth)
    r.Get("/status", govHandler.HandleGovernanceStatus)
})
```

---

## 9. Qoder Context Upgrade

The Qoder Context endpoint (`GET /api/v1/qoder/context`) provides AI decision context to the Qoder agent. Phase 5 upgrades this context with ontology information, governance metadata, and agent policy data.

### New Context Fields

```go
// Phase 5 additions to ContextResponse
type ContextResponse struct {
    // Phase 4 fields (existing):
    RequestID        string          `json:"request_id"`
    System           SystemInfo      `json:"system"`
    Summary          ContextSummary  `json:"summary"`
    TopAlerts        []AlertItem     `json:"top_alerts"`
    OpenTasks        []TaskItem      `json:"open_tasks"`
    PendingOutbox    []OutboxItem    `json:"pending_outbox"`
    RecentDiagnosis  []interface{}   `json:"recent_diagnosis"`
    AllowedActions   []string        `json:"allowed_actions"`
    ForbiddenActions []string        `json:"forbidden_actions"`

    // Phase 5 additions:
    Ontology         OntologyContext     `json:"ontology,omitempty"`
    Governance       GovernanceContext   `json:"governance,omitempty"`
    AgentPolicy      AgentPolicyContext  `json:"agent_policy,omitempty"`
}

// OntologyContext provides object type metadata to the Qoder.
type OntologyContext struct {
    ObjectTypes []OntologySummary `json:"object_types"`
}

type OntologySummary struct {
    ObjectTypeID    string   `json:"object_type_id"`
    DisplayName     string   `json:"display_name"`
    Grain           string   `json:"grain"`
    PropertyNames   []string `json:"property_names"`
    RelationshipSummary string `json:"relationship_summary,omitempty"`
}

// GovernanceContext provides governance rules to the Qoder.
type GovernanceContext struct {
    ClassificationCount int      `json:"classification_count"`
    LineageNodeCount    int      `json:"lineage_node_count"`
    AccessPolicyRoles   []string `json:"access_policy_roles"`
    ActiveHealthChecks  int      `json:"active_health_checks"`
}

// AgentPolicyContext tells the Qoder what it can and cannot do.
type AgentPolicyContext struct {
    DefaultRole       string   `json:"default_role"`        // "analyst"
    EffectiveRole     string   `json:"effective_role"`      // actual role after auth
    MaxQueryLimit     int      `json:"max_query_limit"`      // 1000
    AllowedObjects    []string `json:"allowed_objects"`      // object types Qoder can query
    RedactedFields    []string `json:"redacted_fields,omitempty"` // fields stripped from Qoder
    ForbiddenMarkings []string `json:"forbidden_markings,omitempty"` // markings blocking access
}
```

### Backward Compatibility

The Phase 5 ContextResponse is **backward compatible** with Phase 4 consumers:

- New fields (`ontology`, `governance`, `agent_policy`) use `omitempty` JSON tags, so they are absent when the Qoder context is served from an environment without Phase 5 governance loaded.
- Existing fields (`top_alerts`, `open_tasks`, etc.) remain in the same position with the same types.
- The `AllowedActions` list adds `read_governance` if governance is active (matching the static capability matrix update in Phase 4).

### Query Parameter Additions

The context endpoint adds optional query parameters for Phase 5:

| Parameter | Type | Default | Description |
|---|---|---|---|
| `include_ontology` | bool | `false` | Include ontology context in response |
| `include_governance` | bool | `false` | Include governance metadata in response |
| `include_policy` | bool | `false` | Include agent policy in response |
| `ontology_limit` | int | `8` | Max object types to include in ontology |

---

## 10. Non-Goals

The following are explicitly NOT in scope for Phase 5:

### No LLM Calls
Phase 5 does not make any calls to LLM services. All responses are deterministic, rule-based reads from PostgreSQL. LLM integration (Qoder report generation) remains in Phase 7.

### No Action Execution
Phase 5 does not dispatch outbox events, trigger pipelines, send Feishu messages, or modify any operational data. It is strictly read-only at the business data layer.

### No Python/React Changes
Phase 5 does not modify any Python files (api/*, services/*, scripts/*) or React frontend files (frontend/*). The Phase 5 services are Go-only.

### No YAML Changes
Phase 5 does not add, remove, or modify any YAML configuration files. All 29 existing YAML files are consumed as-is. YAML evolution is a separate concern.

### No RBAC Enforcement at HTTP Layer
Phase 5 applies role-based access at the service/redaction layer, not through HTTP middleware. HTTP-level RBAC is a future concern for Phase 7+ when write operations are introduced.

### No Hot Reload
Config changes require a server restart. There is no filesystem watcher or runtime config reload. Hot reload is deferred to a future phase.

### No Graph Traversal
The lineage service returns the raw lineage graph nodes and edges without performing graph traversal or dependency resolution. Advanced lineage queries (impact analysis, root cause tracing) are deferred.

### No Write-Back to gov.* Tables
Phase 5 does not write governance checkpoint records or health check results. Writing to `gov.config_snapshot` happens only through the ConfigLoader at startup. The `governance_checkpoints` and `governance_health_results` tables are populated by other services (dispatch, pipeline).

### No Caching Layer
Phase 5 does not introduce an in-memory cache or Redis layer. All reads go directly to PostgreSQL. Caching is deferred to a future phase if performance requirements demand it.

---

## 11. Acceptance Criteria

### AC1: ConfigLoader

- [ ] `ConfigRegistry.LoadAll("config/")` loads all 29 YAML files without error
- [ ] Each loaded entry has a non-empty SHA256 `content_hash`
- [ ] `SyncSnapshots` upserts all entries into `gov.config_snapshot` successfully
- [ ] Duplicate calls to `SyncSnapshots` are idempotent (same hash = no update)
- [ ] Invalid YAML files are logged as warnings, not fatal errors
- [ ] ConfigLoader completes within 500ms for 29 files
- [ ] `ConfigRegistry.Get("data_classification")` returns the correct entry

### AC2: ObjectRegistry

- [ ] All 8 AIP object types are registered after loading
- [ ] `ObjectRegistry.Get("customer")` returns a fully populated `ObjectType` with all 8 properties
- [ ] `ObjectRegistry.Get("seller")` includes relationships (has_items, has_products)
- [ ] `ObjectRegistry.Get("nonexistent")` returns nil
- [ ] Property type mappings are correct (string, int, float, datetime)
- [ ] LLMAccess config defaults to `can_query: true` and `max_return_rows: 1000`

### AC3: GovernanceService

- [ ] `GetClassification` returns all 24 classification entries from `data_classification.yml`
- [ ] `GetLineage` returns nodes and edges matching `data_lineage.yml`
- [ ] `GetAccessPolicy` returns the 4 roles (admin, analyst, viewer, marketing_ops)
- [ ] `GetCatalog` returns 8 objects, 12 tables, 14 API endpoints, 4 CSV files from `data_catalog.yml`
- [ ] `GetCheckpoints` paginates correctly (default limit 100)
- [ ] `GetHealthChecks` returns all 5 health check definitions from `health_checks.yml`
- [ ] All methods return empty response (not error) when gov.* tables are empty

### AC4: Redaction

- [ ] `customer_unique_id` is redacted for `analyst` role
- [ ] `payment_value` is redacted for `viewer` role
- [ ] `review_comment_message` is redacted for `viewer` role (sensitive field)
- [ ] `gmv` is visible for all roles (derived_sensitive, visible to admin/analyst)
- [ ] All fields are visible for `admin` role
- [ ] Redaction events are logged to `gov.redaction_log` with correct reason
- [ ] Redaction does not modify the original data, only the response

### AC5: Governance API (6 Endpoints)

- [ ] `GET /api/v1/governance/catalog` returns HTTP 200 with full catalog
- [ ] `GET /api/v1/governance/classification` returns HTTP 200 with classifications
- [ ] `GET /api/v1/governance/markings` returns HTTP 200 with markings
- [ ] `GET /api/v1/governance/lineage` returns HTTP 200 with lineage graph
- [ ] `GET /api/v1/governance/checkpoints` returns HTTP 200 with checkpoints (supports `?limit=5&offset=0`)
- [ ] `GET /api/v1/governance/health` returns HTTP 200 with health checks
- [ ] All endpoints return proper error codes (400, 401, 403, 500) under error conditions
- [ ] All endpoints respond within 200ms (p50, warm)

### AC6: Qoder Context Upgrade

- [ ] `GET /api/v1/qoder/context?include_ontology=true` includes ontology context
- [ ] `GET /api/v1/qoder/context?include_governance=true` includes governance metadata
- [ ] `GET /api/v1/qoder/context?include_policy=true` includes agent policy
- [ ] `GET /api/v1/qoder/context` (no params) is backward compatible with Phase 4
- [ ] `AgentPolicyContext.default_role` is `"analyst"`
- [ ] `AgentPolicyContext.max_query_limit` is `1000`

### AC7: Regression

- [ ] All Phase 4 endpoints continue to work unchanged
- [ ] Existing `GET /api/v1/governance/status` still returns governance_layer + configs
- [ ] All existing unit tests pass (no regressions)
- [ ] No existing API response shapes changed

---

## 12. Implementation Plan

### Wave 1: Schema + ConfigLoader (Commit 1)

**Files to create/modify:**

```
sql/migrations/012_gov_tables.sql          — CREATE TABLE gov.* (config_snapshot, object_schema,
                                              data_classification, data_lineage,
                                              data_lineage_edge, access_policy, redaction_log)

internal/configloader/config_loader.go      — ConfigRegistry struct, LoadAll, Get, Entries
internal/configloader/config_loader_test.go — Unit tests for LoadAll, SHA256, duplicate handling
```

**Commit message:** `feat(phase5): add gov.* schema and ConfigLoader`

**Acceptance:** `ConfigRegistry.LoadAll("config/")` succeeds, all 29 files parsed, SHA256 hashes verified.

### Wave 2: ObjectRegistry + gov Schema Data Loader (Commit 2)

**Files to create/modify:**

```
internal/ontology/registry.go          — ObjectType struct, ObjectRegistry, Register, Get, List
internal/ontology/registry_test.go     — Unit tests for all 8 object types
internal/configloader/schema_loader.go — gov.object_schema loader (reads from gov.*, populates ObjectRegistry)
internal/repository/governance_repository.go — Add GetObjectSchema, GetClassificationRows, GetLineageNodes, etc.
```

**Commit message:** `feat(phase5): add ObjectRegistry with 8 AIP object types`

**Acceptance:** ObjectRegistry populated, all 8 types accessible, property types correct, relationships resolved.

### Wave 3: GovernanceService + Redaction (Commit 3)

**Files to create/modify:**

```
internal/service/governance_service.go  — Extend with GetClassification, GetLineage, GetCatalog,
                                           GetMarkings, GetAccessPolicy, GetCheckpoints, GetHealthChecks
internal/redaction/redactor.go          — FieldRedactor, classification-based + marking-based stripping
internal/redaction/redactor_test.go     — Redaction tests for each role
internal/api/dto/governance.go          — Add all response DTOs (CatalogResponse, ClassificationResponse,
                                           LineageResponse, MarkingsResponse, CheckpointListResponse,
                                           HealthCheckListResponse)
```

**Commit message:** `feat(phase5): add GovernanceService and LLM-safe redaction`

**Acceptance:** All 7 governance query methods work, redaction strips PII/sensitive fields per role, redaction log populated.

### Wave 4: Governance API + Route Registration (Commit 4)

**Files to create/modify:**

```
internal/api/handler/governance.go     — Add HandleCatalog, HandleClassification, HandleMarkings,
                                          HandleLineage, HandleCheckpoints, HandleHealth
internal/api/handler/governance_test.go — Integration tests for all 6 endpoints
internal/api/dto/governance.go          — Add any missing DTOs for endpoint responses
internal/api/router.go                  — Add /api/v1/governance/ route group (6 new routes)
```

**Commit message:** `feat(phase5): add 6 Governance API endpoints`

**Acceptance:** All 6 endpoints return correct responses, pagination works, error handling is correct, Phase 4 endpoints not affected.

### Wave 5: Qoder Context Upgrade + Integration Tests (Commits 5-7)

**Commit 5 — Qoder Context Upgrade:**

```
internal/service/qoder_service.go   — Add ontology/governance/agent_policy to GetContext
internal/api/dto/qoder.go           — Add OntologyContext, GovernanceContext, AgentPolicyContext
internal/api/handler/qoder.go       — Add new query param parsing (include_ontology, etc.)
internal/api/dto/qoder_test.go      — Test backward compatibility
```

**Commit message:** `feat(phase5): upgrade Qoder context with ontology, governance, agent policy`

**Commit 6 — ConfigSync Command:**

```
cmd/baxi-cli/config_sync.go         — CLI command to manually trigger ConfigLoader.SyncSnapshots
```

**Commit message:** `feat(phase5): add config-sync CLI command`

**Commit 7 — Integration Tests:**

```
internal/api/handler/governance_integration_test.go  — Full integration test with PostgreSQL
internal/configloader/config_loader_integration_test.go  — ConfigLoader + gov.* round trip
tests/phase5_regression_test.go     — Verify Phase 4 endpoints still pass
```

**Commit message:** `test(phase5): add integration and regression tests`

**Acceptance:** All integration tests pass, Phase 4 regression tests pass, config-sync command works manually.

### File Dependency Graph

```
migrations/012_gov_tables.sql          (Wave 1, no deps)
    │
    ▼
internal/configloader/config_loader.go  (Wave 1, reads YAML files)
    │
    ▼
internal/ontology/registry.go           (Wave 2, consumes config data)
    │
    ▼
internal/repository/governance_repository.go  (Wave 2, gov.* table queries)
    │
    ▼
internal/redaction/redactor.go          (Wave 3, classification + marking policy)
    │
    ▼
internal/service/governance_service.go  (Wave 3, orchestrates repo + redaction)
    │
    ▼
internal/api/dto/governance.go          (Wave 3, response types)
    │
    ▼
internal/api/handler/governance.go      (Wave 4, HTTP handlers)
    │
    ▼
internal/api/router.go                  (Wave 4, route registration)
    │
    ▼
internal/service/qoder_service.go       (Wave 5, ontology context enrichment)
    │
    ▼
cmd/baxi-cli/config_sync.go             (Wave 6, CLI command)
```

---

## Appendix A: Complete File List

### New Files

| File | Wave | Purpose |
|---|---|---|
| `sql/migrations/012_gov_tables.sql` | 1 | Gov schema tables |
| `internal/configloader/config_loader.go` | 1 | ConfigRegistry + LoadAll |
| `internal/configloader/config_loader_test.go` | 1 | Unit tests |
| `internal/ontology/registry.go` | 2 | ObjectRegistry + ObjectType |
| `internal/ontology/registry_test.go` | 2 | Unit tests |
| `internal/configloader/schema_loader.go` | 2 | Object schema loader |
| `internal/redaction/redactor.go` | 3 | Field redactor |
| `internal/redaction/redactor_test.go` | 3 | Redaction tests |
| `cmd/baxi-cli/config_sync.go` | 6 | CLI config sync command |
| `internal/api/handler/governance_integration_test.go` | 7 | Integration tests |
| `internal/configloader/config_loader_integration_test.go` | 7 | Integration tests |
| `tests/phase5_regression_test.go` | 7 | Regression tests |

### Modified Files

| File | Wave | Changes |
|---|---|---|
| `internal/repository/governance_repository.go` | 2 | Add query methods for all gov.* tables |
| `internal/service/governance_service.go` | 3 | Add 7 new methods, remove old GetStatus-only impl |
| `internal/api/dto/governance.go` | 3 | Add all Phase 5 response DTOs |
| `internal/api/handler/governance.go` | 4 | Add 6 handlers, update GovernanceProvider interface |
| `internal/api/router.go` | 4 | Add governance route group |
| `internal/service/qoder_service.go` | 5 | Enrich context with ontology/governance/policy |
| `internal/api/dto/qoder.go` | 5 | Add ontology context DTOs |
| `internal/api/handler/qoder.go` | 5 | Add new query parameters |
| `go.mod` | 1 | Add `gopkg.in/yaml.v3` dependency |

---

## Appendix B: gov.* SQL Schema (Full DDL)

```sql
-- 012_gov_tables.sql : Governance Runtime tables for Phase 5

-- 1. Config snapshot: generic YAML-to-JSONB storage
CREATE TABLE IF NOT EXISTS gov.config_snapshot (
    config_key   TEXT PRIMARY KEY,
    content_hash TEXT NOT NULL,
    content_json JSONB NOT NULL,
    status       TEXT NOT NULL DEFAULT 'loaded',
    error_msg    TEXT,
    loaded_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Object schema: parsed AIP object type definitions
CREATE TABLE IF NOT EXISTS gov.object_schema (
    object_type_id TEXT PRIMARY KEY,
    schema_json    JSONB NOT NULL,
    sensitivity    TEXT,
    owner_role     TEXT,
    retention_days INTEGER,
    loaded_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. Data classification: asset-level + field-level sensitivity
CREATE TABLE IF NOT EXISTS gov.data_classification (
    asset_ref    TEXT PRIMARY KEY,
    level        TEXT NOT NULL,
    rationale    TEXT,
    field_levels JSONB,
    loaded_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 4. Data lineage: nodes
CREATE TABLE IF NOT EXISTS gov.data_lineage (
    node_id   TEXT PRIMARY KEY,
    node_type TEXT NOT NULL,
    label     TEXT,
    status    TEXT,
    loaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 5. Data lineage: edges
CREATE TABLE IF NOT EXISTS gov.data_lineage_edge (
    edge_id        SERIAL PRIMARY KEY,
    from_node_id   TEXT NOT NULL REFERENCES gov.data_lineage(node_id),
    to_node_id     TEXT NOT NULL REFERENCES gov.data_lineage(node_id),
    transform      TEXT,
    transform_type TEXT,
    loaded_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 6. Access policy: role definitions
CREATE TABLE IF NOT EXISTS gov.access_policy (
    role            TEXT PRIMARY KEY,
    allowed_actions TEXT[] NOT NULL,
    data_access     TEXT[] NOT NULL,
    loaded_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 7. Redaction log: audit trail for redacted fields
CREATE TABLE IF NOT EXISTS gov.redaction_log (
    log_id      SERIAL PRIMARY KEY,
    request_id  TEXT NOT NULL,
    timestamp   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    object_type TEXT,
    field_name  TEXT NOT NULL,
    reason      TEXT NOT NULL,
    actor_role  TEXT NOT NULL
);
```

---

## Appendix C: Existing gov.* Tables (from migrations/008_governance.sql)

The following tables already exist from the SQLite migration and are reused in Phase 5:

```sql
-- governance_checkpoints: checkpoint audit trail (SQLite compatible)
CREATE TABLE IF NOT EXISTS governance_checkpoints (
    checkpoint_id INTEGER PRIMARY KEY AUTOINCREMENT,
    action_type TEXT NOT NULL,
    endpoint TEXT NOT NULL,
    actor TEXT NOT NULL,
    request_id TEXT,
    justification TEXT,
    mode TEXT NOT NULL DEFAULT 'dry_run',
    status TEXT NOT NULL DEFAULT 'recorded',
    metadata_json TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- governance_health_results: health check execution results (SQLite compatible)
CREATE TABLE IF NOT EXISTS governance_health_results (
    result_id INTEGER PRIMARY KEY AUTOINCREMENT,
    check_id TEXT NOT NULL,
    check_type TEXT NOT NULL,
    status TEXT NOT NULL,
    detail TEXT,
    checked_at TEXT NOT NULL DEFAULT (datetime('now'))
);
```

In the Go/PostgreSQL migration, these will be ported to `gov.governance_checkpoint` and `gov.health_check_result` respectively (if not already done in Phase 4 migrations).

---

## Appendix D: DTO Summary

```go
package dto

// Phase 5 Response DTOs

// CatalogResponse — GET /api/v1/governance/catalog
type CatalogResponse struct {
    Objects   []CatalogObject   `json:"objects"`
    Tables    []CatalogTable    `json:"tables"`
    Endpoints []CatalogEndpoint `json:"endpoints"`
}

type CatalogObject struct {
    ObjectTypeID  string `json:"object_type_id"`
    DisplayName   string `json:"display_name"`
    Sensitivity   string `json:"sensitivity"`
    OwnerRole     string `json:"owner_role"`
    Grain         string `json:"grain"`
    PropertyCount int    `json:"property_count"`
    PIIFieldCount int    `json:"pii_field_count"`
    RetentionDays int    `json:"retention_days"`
}

type CatalogTable struct {
    TableName   string `json:"table_name"`
    Sensitivity string `json:"sensitivity"`
    ColumnCount int    `json:"column_count"`
    Grain       string `json:"grain"`
}

type CatalogEndpoint struct {
    Endpoint    string `json:"endpoint"`
    Method      string `json:"method"`
    Sensitivity string `json:"sensitivity"`
}

// ClassificationResponse — GET /api/v1/governance/classification
type ClassificationResponse struct {
    Classifications []ClassificationEntry `json:"classifications"`
}

type ClassificationEntry struct {
    AssetRef       string            `json:"asset_ref"`
    Level          string            `json:"level"`
    Rationale      string            `json:"rationale"`
    AppliesToFields map[string]string `json:"applies_to_fields,omitempty"`
}

// MarkingsResponse — GET /api/v1/governance/markings
type MarkingsResponse struct {
    Markings            map[string]MarkingDef   `json:"markings"`
    PipelineStageMarkings []PipelineStageMarking `json:"pipeline_stage_markings"`
}

type MarkingDef struct {
    MandatoryControl       bool     `json:"mandatory_control"`
    AccessType             string   `json:"access_type"`
    Conjunctive            bool     `json:"conjunctive"`
    AppliesTo              []string `json:"applies_to"`
    Policy                 string   `json:"policy"`
    ExpandAccessPermission string   `json:"expand_access_permission"`
}

// LineageResponse — GET /api/v1/governance/lineage
type LineageResponse struct {
    Nodes []LineageNode `json:"nodes"`
    Edges []LineageEdge `json:"edges"`
}

type LineageNode struct {
    ID     string `json:"id"`
    Type   string `json:"type"`
    Label  string `json:"label"`
    Status string `json:"status"`
}

type LineageEdge struct {
    From          string `json:"from"`
    To            string `json:"to"`
    Transform     string `json:"transform"`
    TransformType string `json:"transform_type"`
}

// CheckpointListResponse — GET /api/v1/governance/checkpoints
type CheckpointListResponse struct {
    Items []CheckpointItem `json:"items"`
    Total int              `json:"total"`
}

type CheckpointItem struct {
    CheckpointID  int    `json:"checkpoint_id"`
    ActionType    string `json:"action_type"`
    Endpoint      string `json:"endpoint"`
    Actor         string `json:"actor"`
    Status        string `json:"status"`
    CreatedAt     string `json:"created_at"`
}

// HealthCheckListResponse — GET /api/v1/governance/health
type HealthCheckListResponse struct {
    Items []HealthCheckItem `json:"items"`
}

type HealthCheckItem struct {
    CheckID   string `json:"check_id"`
    CheckType string `json:"check_type"`
    Status    string `json:"status"`
    Detail    string `json:"detail"`
    CheckedAt string `json:"checked_at"`
}

// ConfigSnapshotResponse — GET /api/v1/governance/config/{key} (internal)
type ConfigSnapshotResponse struct {
    ConfigKey   string `json:"config_key"`
    ContentHash string `json:"content_hash"`
    Status      string `json:"status"`
    LoadedAt    string `json:"loaded_at"`
}

// AccessPolicyResponse — GET /api/v1/governance/access-policy
type AccessPolicyResponse struct {
    Roles         []PolicyRole `json:"roles"`
    DefaultPolicy string       `json:"default_policy"`
}

type PolicyRole struct {
    Role           string   `json:"role"`
    AllowedActions []string `json:"allowed_actions"`
    DataAccess     []string `json:"data_access"`
}
```

---

## Appendix E: Environment Variables

| Variable | Default | Endpoints | Description |
|---|---|---|---|
| `API_BEARER_TOKEN` | (required) | All governance endpoints | Bearer token for API authentication |
| `DATABASE_URL` | `postgres://baxi:baxi_dev@localhost:5432/baxi?sslmode=disable` | All governance endpoints | PostgreSQL connection string |
| `GOV_CONFIG_DIR` | `config/` | ConfigLoader | Directory containing YAML config files |
| `GOV_DEFAULT_ROLE` | `analyst` | Governance API, Qoder Context | Default role for requests without explicit role |
| `GOV_MAX_QUERY_LIMIT` | `1000` | Object queries | Default max rows returned by object queries |
| `GOV_LOG_REDACTIONS` | `true` | Redaction | Enable/disable redaction audit logging |
