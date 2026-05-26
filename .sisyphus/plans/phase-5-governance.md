# Phase 5: Governance / Ontology Runtime

## TL;DR

> Upgrade AIP governance YAML configs from "static configuration snapshots" to Go runtime services, creating a safe semantic layer for future LLM decision layer.
>
> **Deliverables**:
> - ConfigLoader: Load and validate governance YAML, sync to gov.config_snapshot
> - ObjectRegistry: Load AIP semantic object model from aip_object_schema.yml
> - ObjectQueryService: Query business objects by object_type (GetObject, SearchObjects, GetObjectMetrics)
> - GovernanceService: Classification, lineage, access policy, checkpoint, redaction
> - LLM-safe Context Builder: Generate redacted context for agent_readonly role
> - Governance API: 6 new read-only endpoints
> - Qoder Context Upgrade: Enrich with ontology + governance + redaction + agent_policy
>
> **Estimated Effort**: Large (7 commits, ~20-30 tasks)
> **Parallel Execution**: YES - 5 waves + final verification
> **Critical Path**: ConfigLoader → ObjectRegistry → ObjectQueryService → GovernanceService → Governance API → Qoder Context

---

## Context

### Original Request
Phase 5 of Baxi Go + PostgreSQL migration: transform governance YAML configs into Go runtime capabilities.

### Interview Summary
**Key Decisions**:
- Integration tests: Use testcontainers-go for PostgreSQL (expanding existing usage)
- Object query implementation: Reuse existing repository layer + add ontology_repository.go
- Default role mapping: Existing API Bearer Token maps to `analyst` role
- ConfigLoader: Load at startup only (no hot reload)
- ObjectQueryService: Explicit per-type methods (no dynamic SQL generation)
- Default LIMIT: 1000 rows per query

**Research Findings**:
- testcontainers-go ALREADY used in project (internal/testutil/db.go) but uses deprecated postgres.RunContainer()
- 29 YAML config files in config/ directory
- 7 gov.* tables already created in PostgreSQL via migrations/006_gov_tables.sql
- 8 object types defined in aip_object_schema.yml: customer, order, seller, product, category, region, marketing_lead, metric_alert
- Classification levels: pii, internal, sensitive, public_internal, derived_sensitive
- Access policy roles: admin, analyst, viewer, marketing_ops
- Current governance/status queries gov.config_snapshot
- Current qoder/context aggregates from 4 sources (pipeline_run, metric_alert, task, outbox_event)
- No repository interfaces exist yet (concrete struct dependencies)
- QoderService currently breaks layer boundaries (injects repositories + does direct SQL)

### Metis Review
**Identified Gaps** (addressed):
- YAML load strategy: Startup-only, no hot reload (locked down)
- Memory vs DB source of truth: YAML is canonical, loaded into memory + synced to DB for persistence
- Role resolution: Bearer Token → "analyst" is default; other roles resolved from access_policy.yml
- ObjectQueryService: Explicit typed methods per object type, no generic dynamic SQL
- QoderService anti-pattern: Create ContextRepository to encapsulate data queries
- testcontainers-go: Modernize internal/testutil/db.go from RunContainer() to Run()
- Repository interfaces: Introduce interfaces for all new repositories

---

## Work Objectives

### Core Objective
Create read-only Go runtime services that expose governance YAML configs, AIP object schemas, data classifications, lineage, access policies, and health checks through a unified API layer, with LLM-safe context redaction for agent_readonly role.

### Concrete Deliverables
1. `internal/configloader/` - ConfigLoader with YAML loading, hash computation, DB sync
2. `internal/ontology/` - ObjectRegistry, ObjectQueryService, Context Builder
3. `internal/repository/ontology_repository.go` - Object queries against dwd/mart/ops tables
4. `internal/governance/` - GovernanceService with classification, lineage, access, redaction
5. `internal/api/handler/governance.go` - Expanded with 6 new endpoints
6. `internal/api/handler/qoder.go` - Enriched context response
7. `internal/repository/interfaces.go` - Repository interfaces for testability
8. `migrations/009_gov_indexes.sql` - Performance indexes for gov.* tables
9. Updated `internal/testutil/db.go` - Modernize to postgres.Run()
10. `make governance-load` and `make governance-check` Makefile targets

### Definition of Done
- [ ] All 29 YAML configs loadable via ConfigLoader
- [ ] All 8 object types queryable via ObjectRegistry
- [ ] ObjectQueryService supports GetObject/SearchObjects/GetObjectMetrics for core types
- [ ] GovernanceService provides classification/lineage/access/checkpoint/redaction
- [ ] LLM-safe context redacts PII/sensitive fields for agent_readonly role
- [ ] 6 new governance API endpoints return correct JSON
- [ ] Qoder context includes ontology + governance + redaction + agent_policy
- [ ] All new repository tests use testcontainers-go
- [ ] `make test` passes
- [ ] `make api-compare` passes or accepted WARN
- [ ] `make pipeline-compare` passes or accepted WARN

### Must Have
1. ConfigLoader loads all YAML, computes SHA256 hash, writes to gov.config_snapshot
2. ConfigLoader parses and writes to gov.object_schema, gov.data_classification, gov.data_lineage, gov.access_policy
3. ObjectRegistry supports all 8 object types with typed access
4. ObjectQueryService supports at least: order, seller, category, region, metric_alert
5. GovernanceService provides: GetClassification, GetFieldMarking, CheckAccess, GetLineage, RequiresCheckpoint, RedactObjectContext, GetGovernanceStatus
6. Redaction strips fields with classification: pii, sensitive, derived_sensitive
7. Governance API: /governance/catalog, /classification, /markings, /lineage, /checkpoints, /health
8. Qoder context: ontology.object_types, governance.*, redaction_enabled, agent_policy

### Must NOT Have (Guardrails)
1. No LLM calls or LLM API integration
2. No action execution or outbox dispatch
3. No pipeline logic modification
4. No Python code changes (api/, services/, adapters/ frozen)
5. No React frontend changes
6. No YAML config semantic changes
7. No write-side apply operations (INSERT/UPDATE/DELETE on business tables)
8. No new complex auth system (keep Bearer Token)
9. No WebSocket/SSE or real-time features
10. No graph traversal algorithms for lineage
11. No hot reload of YAML configs at runtime
12. No dynamic SQL generation in ObjectQueryService

---

## Verification Strategy

### Test Decision
- **Infrastructure exists**: YES (testcontainers-go already in use)
- **Automated tests**: YES (Tests after implementation for speed; TDD for critical components)
- **Framework**: Standard Go testing + testcontainers-go v0.30+
- **Build tags**: `//go:build integration` for testcontainers-based tests
- **Testcontainers modernization**: Update internal/testutil/db.go from deprecated `postgres.RunContainer()` to `postgres.Run()`

### QA Policy
Every task MUST include agent-executed QA scenarios. Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **Backend/Repository**: Use Bash (go test) with testcontainers
- **API/Handlers**: Use Bash (curl) - Send requests, assert status + response fields
- **Integration**: Use Bash (make commands) - Run full verification

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Foundation - can start immediately, all independent):
├── Task 1: Modernize testcontainers (internal/testutil/db.go) [quick]
├── Task 2: Add repository interfaces (internal/repository/interfaces.go) [quick]
├── Task 3: Add gov table indexes migration (migrations/009_gov_indexes.sql) [quick]
├── Task 4: Add governance Makefile targets [quick]
└── Task 5: Design and document plan (docs/migration/phase-5-governance-ontology-runtime-plan.md) [quick]

Wave 2 (Core Services - depends on Wave 1):
├── Task 6: ConfigLoader implementation (internal/configloader/) [unspecified-high]
├── Task 7: ObjectRegistry implementation (internal/ontology/registry.go) [unspecified-high]
├── Task 8: Ontology repository (internal/repository/ontology_repository.go) [unspecified-high]
└── Task 9: GovernanceService expansion (internal/governance/service.go) [unspecified-high]

Wave 3 (Query + Redaction - depends on Wave 2):
├── Task 10: ObjectQueryService (internal/ontology/query_service.go) [deep]
├── Task 11: LLM-safe Context Builder (internal/ontology/context_builder.go) [deep]
├── Task 12: Redaction engine (internal/governance/redaction.go) [deep]
└── Task 13: ContextRepository for Qoder (internal/repository/context_repository.go) [unspecified-high]

Wave 4 (API + Integration - depends on Wave 3):
├── Task 14: Expand governance handlers (internal/api/handler/governance.go) [unspecified-high]
├── Task 15: Enrich Qoder context (internal/service/qoder_service.go) [unspecified-high]
├── Task 16: Add new governance DTOs (internal/api/dto/governance.go) [quick]
├── Task 17: Wire routes in server.go [quick]
└── Task 18: CLI commands (cmd/baxi-cli governance load/check) [unspecified-high]

Wave 5 (Validation + Integration):
├── Task 19: Integration test suite (testcontainers for all repos) [unspecified-high]
├── Task 20: API smoke tests (all endpoints) [unspecified-high]
├── Task 21: Run make test, api-compare, pipeline-compare [quick]
└── Task 22: Verify no Python/React/pipeline changes [quick]

Wave FINAL (After ALL tasks - 4 parallel reviews):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Real manual QA (unspecified-high)
└── Task F4: Scope fidelity check (deep)
-> Present results -> Get explicit user okay

Critical Path: Task 1-5 → Task 6-9 → Task 10-13 → Task 14-18 → Task 19-22 → F1-F4 → user okay
Parallel Speedup: ~60% faster than sequential
Max Concurrent: 5 (Waves 2 & 3)
```

### Dependency Matrix

- **1-5**: - - 6-9, 1
- **6-9**: 1-5 - 10-13, 2
- **10-13**: 6-9 - 14-18, 2
- **14-18**: 10-13 - 19-22, 3
- **19-22**: 14-18 - F1-F4, 4
- **F1-F4**: 19-22 - user okay, 4

### Agent Dispatch Summary

- **Wave 1**: **5** - T1-T5 → `quick` × 5
- **Wave 2**: **4** - T6 → `unspecified-high`, T7 → `unspecified-high`, T8 → `unspecified-high`, T9 → `unspecified-high`
- **Wave 3**: **4** - T10 → `deep`, T11 → `deep`, T12 → `deep`, T13 → `unspecified-high`
- **Wave 4**: **5** - T14-T15 → `unspecified-high`, T16-T17 → `quick`, T18 → `unspecified-high`
- **Wave 5**: **4** - T19-T20 → `unspecified-high`, T21-T22 → `quick`
- **FINAL**: **4** - F1 → `oracle`, F2 → `unspecified-high`, F3 → `unspecified-high`, F4 → `deep`

---

## TODOs

- [x] 1. Modernize testcontainers and add repository interfaces

  **What to do**:
  - Update `internal/testutil/db.go` to use `postgres.Run()` instead of deprecated `postgres.RunContainer()`
  - Add `internal/repository/interfaces.go` with repository interfaces for testability
  - Add `migrations/009_gov_indexes.sql` with performance indexes for gov.* tables
  - Add governance Makefile targets: `governance-load`, `governance-check`

  **Must NOT do**:
  - Do NOT change any existing repository implementations (just add interfaces)
  - Do NOT modify existing test files (they'll be updated separately)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - Reason: Infrastructure setup, no complex logic

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2-5)
  - **Blocks**: Tasks 6-22 (all tests depend on modernized testutil)
  - **Blocked By**: None

  **References**:
  - `internal/testutil/db.go` - Current testcontainers setup (needs modernization)
  - `migrations/006_gov_tables.sql` - Existing gov table schema
  - `Makefile` - Add new targets

  **Acceptance Criteria**:
  - [ ] `internal/testutil/db.go` uses `postgres.Run()` API
  - [ ] `internal/repository/interfaces.go` defines interfaces for all new repositories
  - [ ] `migrations/009_gov_indexes.sql` created with indexes on gov.* tables
  - [ ] `make governance-load` target exists
  - [ ] `make governance-check` target exists

  **QA Scenarios**:
  ```
  Scenario: Testcontainers modernized
    Tool: Bash
    Steps:
      1. grep -r "RunContainer" internal/testutil/ || echo "No deprecated API found"
    Expected: No RunContainer references remain
    Evidence: .sisyphus/evidence/task-1-testcontainers-modernized.txt

  Scenario: Makefile targets work
    Tool: Bash
    Steps:
      1. make governance-load --help || true  # verify target exists
      2. make governance-check --help || true  # verify target exists
    Expected: Both targets exist in Makefile
    Evidence: .sisyphus/evidence/task-1-makefile-targets.txt
  ```

  **Commit**: NO (groups with Task 2 in commit 2)

- [x] 2. Add repository interfaces

  **What to do**:
  - Create `internal/repository/interfaces.go` with interfaces:
    - `ConfigSnapshotRepository` - CRUD for gov.config_snapshot
    - `ObjectSchemaRepository` - CRUD for gov.object_schema
    - `DataClassificationRepository` - CRUD for gov.data_classification
    - `DataLineageRepository` - CRUD for gov.data_lineage
    - `AccessPolicyRepository` - CRUD for gov.access_policy
    - `OntologyRepository` - Query dwd/mart/ops tables by object type
    - `ContextRepository` - Encapsulate Qoder data queries

  **Must NOT do**:
  - Do NOT implement these interfaces yet (just define them)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: Tasks 6-13, 18-20
  - **Blocked By**: None

  **References**:
  - `internal/repository/governance_repository.go` - Existing repository pattern
  - `migrations/006_gov_tables.sql` - Table schemas for interface design

  **Acceptance Criteria**:
  - [ ] All interfaces compile without errors
  - [ ] Interfaces match existing repository patterns

  **QA Scenarios**:
  ```
  Scenario: Interfaces compile
    Tool: Bash
    Steps:
      1. go build ./internal/repository/...
    Expected: Build succeeds
    Evidence: .sisyphus/evidence/task-2-interfaces-compile.txt
  ```

  **Commit**: NO (groups with Task 1 in commit 2)

- [x] 3. Add gov table indexes migration

  **What to do**:
  - Create `migrations/009_gov_indexes.sql`
  - Add indexes on frequently queried columns:
    - `gov.config_snapshot(config_key, config_type)`
    - `gov.object_schema(object_type)`
    - `gov.data_classification(field_path, classification_level)`
    - `gov.data_lineage(source_table, target_table)`
    - `gov.access_policy(policy_name, resource_type, action)`

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: Tasks 6-9
  - **Blocked By**: None

  **Acceptance Criteria**:
  - [ ] Migration file created
  - [ ] `make migrate` applies successfully

  **QA Scenarios**:
  ```
  Scenario: Migration applies
    Tool: Bash
    Steps:
      1. make migrate
      2. psql $DATABASE_URL -c "\di gov.*"
    Expected: New indexes visible in gov schema
    Evidence: .sisyphus/evidence/task-3-migration-applied.txt
  ```

  **Commit**: NO (groups with Task 1 in commit 2)

- [x] 4. Add governance Makefile targets

  **What to do**:
  - Add `governance-load` target: `go run ./cmd/baxi-cli governance load --config-dir ./config`
  - Add `governance-check` target: `go run ./cmd/baxi-cli governance check`
  - Add `test-governance` target for governance-specific tests

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: Tasks 18-19
  - **Blocked By**: None

  **Acceptance Criteria**:
  - [ ] `make governance-load` exists and has help text
  - [ ] `make governance-check` exists and has help text

  **QA Scenarios**:
  ```
  Scenario: Makefile targets present
    Tool: Bash
    Steps:
      1. grep -A 2 "governance-load:" Makefile
      2. grep -A 2 "governance-check:" Makefile
    Expected: Both targets found with commands
    Evidence: .sisyphus/evidence/task-4-makefile.txt
  ```

  **Commit**: NO (groups with Task 1 in commit 2)

- [x] 5. Write Phase 5 design document

  **What to do**:
  - Create `docs/migration/phase-5-governance-ontology-runtime-plan.md`
  - Document: YAML→Runtime mapping, ConfigLoader design, ObjectRegistry design, ObjectQueryService design, GovernanceService design, Redaction strategy, Governance API scope, Qoder context upgrade strategy, non-goals, acceptance criteria

  **Recommended Agent Profile**:
  - **Category**: `writing`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: None (documentation)
  - **Blocked By**: None

  **References**:
  - `docs/migration/phase-4-api-migration-plan.md` - Previous phase doc pattern
  - `config/aip_object_schema.yml` - Object schema reference
  - `migrations/006_gov_tables.sql` - DB schema reference

  **Acceptance Criteria**:
  - [ ] Document exists at correct path
  - [ ] Contains all 10 required sections
  - [ ] References current codebase files

  **QA Scenarios**:
  ```
  Scenario: Design document exists
    Tool: Bash
    Steps:
      1. test -f docs/migration/phase-5-governance-ontology-runtime-plan.md
      2. grep -c "##" docs/migration/phase-5-governance-ontology-runtime-plan.md
    Expected: File exists with at least 10 sections
    Evidence: .sisyphus/evidence/task-5-design-doc.txt
  ```

  **Commit**: YES (commit 1: docs)

- [x] 6. Implement ConfigLoader

  **What to do**:
  - Create `internal/configloader/` package:
    - `loader.go` - Main ConfigLoader struct and LoadAll() method
    - `yaml.go` - YAML parsing helpers
    - `hash.go` - SHA256 hash computation for content_hash
    - `validator.go` - Validate required configs exist
    - `snapshot.go` - Sync to gov.config_snapshot table
  - ConfigLoader should:
    1. Scan config/ directory for all .yml files
    2. Load each file into memory
    3. Compute SHA256 content_hash
    4. Write to gov.config_snapshot (config_key, config_type, content_jsonb, content_hash)
    5. Parse and write to gov.object_schema, gov.data_classification, gov.data_lineage, gov.access_policy
    6. Return ConfigRegistry with typed structs
  - Support config types: object_schema, data_classification, access_policy, data_lineage, data_markings, health_checks, checkpoint_rules, alert_rules, metrics

  **Must NOT do**:
  - Do NOT modify YAML file contents or semantics
  - Do NOT implement hot reload (load at startup only)
  - Do NOT write to dwd/mart/ops/raw tables

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
  - Reason: Complex multi-file package with DB interactions

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 7-9)
  - **Blocks**: Tasks 10-13 (ObjectQueryService needs loaded schemas)
  - **Blocked By**: Tasks 1-3 (testutil, interfaces, migration)

  **References**:
  - `config/aip_object_schema.yml` - Object schema YAML structure
  - `config/data_classification.yml` - Classification YAML structure
  - `config/access_policy.yml` - Access policy YAML structure
  - `config/data_lineage.yml` - Lineage YAML structure
  - `migrations/006_gov_tables.sql` - Table schemas for DB writes
  - `internal/repository/governance_repository.go` - Existing gov repo pattern

  **Acceptance Criteria**:
  - [ ] All 29 YAML files load without error
  - [ ] gov.config_snapshot has 29 rows (one per YAML file)
  - [ ] gov.object_schema has 8 rows (one per object type)
  - [ ] gov.data_classification has rows for classified fields
  - [ ] gov.data_lineage has rows for edges
  - [ ] gov.access_policy has rows for roles
  - [ ] content_hash is SHA256 of file content
  - [ ] `make governance-load` executes successfully

  **QA Scenarios**:
  ```
  Scenario: ConfigLoader loads all YAML
    Tool: Bash
    Steps:
      1. make governance-load
      2. psql $DATABASE_URL -c "SELECT COUNT(*) FROM gov.config_snapshot"
      3. psql $DATABASE_URL -c "SELECT COUNT(*) FROM gov.object_schema"
      4. psql $DATABASE_URL -c "SELECT COUNT(*) FROM gov.data_classification"
    Expected: 29 configs, 8 objects, >0 classifications
    Evidence: .sisyphus/evidence/task-6-config-loader.txt

  Scenario: Content hash is correct
    Tool: Bash
    Steps:
      1. HASH=$(sha256sum config/aip_object_schema.yml | awk '{print $1}')
      2. DB_HASH=$(psql $DATABASE_URL -t -c "SELECT content_hash FROM gov.config_snapshot WHERE config_key='aip_object_schema'" | xargs)
      3. test "$HASH" = "$DB_HASH" && echo "MATCH" || echo "MISMATCH"
    Expected: MATCH
    Evidence: .sisyphus/evidence/task-6-hash-verify.txt

  Scenario: Missing config fails
    Tool: Bash
    Steps:
      1. mv config/aip_object_schema.yml config/aip_object_schema.yml.bak
      2. make governance-load 2>&1 | tee /tmp/governance-load-error.txt
      3. mv config/aip_object_schema.yml.bak config/aip_object_schema.yml
    Expected: Load fails with error about missing required config
    Evidence: .sisyphus/evidence/task-6-missing-config.txt
  ```

  **Commit**: YES (commit 3: feat: add governance config loader)

- [x] 7. Implement ObjectRegistry

  **What to do**:
  - Create `internal/ontology/` package:
    - `registry.go` - ObjectRegistry with methods:
      - `GetObjectType(objectType string) (*ObjectType, error)`
      - `ListObjectTypes() []string`
      - `GetProperties(objectType string) ([]ObjectProperty, error)`
      - `GetLinks(objectType string) ([]ObjectLink, error)`
      - `GetAllowedActions(objectType string) []string`
      - `IsLLMReadable(objectType, property string) bool`
      - `GetSourceDataset(objectType string) string`
    - `schema.go` - Typed structs for ObjectType, ObjectProperty, ObjectLink
    - `object_type.go` - Object type constants/enums
    - `validator.go` - Validate object schema completeness
  - Load from gov.object_schema (DB-first) with fallback to aip_object_schema.yml
  - Must support all 8 object types

  **Must NOT do**:
  - Do NOT implement query logic (that's ObjectQueryService)
  - Do NOT modify aip_object_schema.yml

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2
  - **Blocks**: Tasks 10-11 (ObjectQueryService, Context Builder)
  - **Blocked By**: Tasks 1-3, 6

  **References**:
  - `config/aip_object_schema.yml` - Source of truth for object schemas
    - customer (lines 2-18): grain=customer_unique_id, source_tables=[order_level_base], 8 properties
    - order (lines 20-35): grain=order_id, source_tables=[order_level_base], 7 properties
    - seller (lines 37-56): grain=seller_id, source_tables=[item_level_base], 8 properties, 2 relationships
    - product (lines 58-74): grain=product_id, source_tables=[item_level_base], 7 properties
    - category (lines 76-90): grain=product_category_name, source_tables=[item_level_base], 5 properties
    - region (lines 92-106): grain=state, source_tables=[order_level_base, item_level_base], 5 properties
    - marketing_lead (lines 108-121): grain=origin, source_tables=[channel_classification], 5 properties
    - metric_alert (lines 123-136): grain=alert_id, source_tables=[metric_alerts], 7 properties
  - `internal/repository/interfaces.go` - ObjectSchemaRepository interface

  **Acceptance Criteria**:
  - [ ] GetObjectType("seller") returns correct struct with 8 properties
  - [ ] GetObjectType("metric_alert") returns correct struct
  - [ ] ListObjectTypes() returns ["customer", "order", "seller", "product", "category", "region", "marketing_lead", "metric_alert"]
  - [ ] GetProperties("customer") returns 8 properties including types and aggregation
  - [ ] GetLinks("seller") returns 2 relationships (has_items, has_products)
  - [ ] Unknown object type returns error
  - [ ] Unit tests pass

  **QA Scenarios**:
  ```
  Scenario: All 8 object types loaded
    Tool: Bash (go test)
    Steps:
      1. go test ./internal/ontology/ -v -run TestRegistry
    Expected: PASS for all 8 object types
    Evidence: .sisyphus/evidence/task-7-registry-test.txt

  Scenario: Unknown object type error
    Tool: Bash (go test)
    Steps:
      1. go test ./internal/ontology/ -v -run TestRegistryUnknown
    Expected: Error returned for "nonexistent_type"
    Evidence: .sisyphus/evidence/task-7-unknown-type.txt
  ```

  **Commit**: YES (commit 4: feat: add ontology object registry)

- [x] 8. Implement Ontology Repository

  **What to do**:
  - Create `internal/repository/ontology_repository.go`
  - Implement `OntologyRepository` interface:
    - `QueryByObjectType(ctx, objectType string, filters ObjectFilters) (*ObjectQueryResult, error)`
    - `GetObjectByID(ctx, objectType, objectID string) (*ObjectInstance, error)`
    - `GetObjectMetrics(ctx, objectType, objectID string) (*ObjectMetrics, error)`
    - `SearchObjects(ctx, objectType string, filters SearchFilters) (*SearchResult, error)`
  - Map object types to source tables:
    - customer → dwd.order_level (aggregated by customer_unique_id)
    - order → dwd.order_level
    - seller → dwd.item_level (aggregated by seller_id)
    - product → dwd.item_level (aggregated by product_id)
    - category → dwd.item_level or mart.metric_dimension_daily (aggregated by product_category_name)
    - region → dwd.order_level or mart.metric_dimension_daily (aggregated by state)
    - marketing_lead → raw.marketing_qualified_leads / raw.closed_deals
    - metric_alert → ops.metric_alert
  - Apply default LIMIT 1000
  - Apply role-based access control (analyst can read metric_daily, dwd_item_level, metric_dimension_daily, alert_events)

  **Must NOT do**:
  - Do NOT generate dynamic SQL (use prepared statements per object type)
  - Do NOT modify dwd/mart/ops/raw tables
  - Do NOT bypass LIMIT

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2
  - **Blocks**: Tasks 10 (ObjectQueryService)
  - **Blocked By**: Tasks 1-3, 6-7

  **References**:
  - `internal/repository/governance_repository.go` - Repository pattern
    - Uses `*pgxpool.Pool` directly in methods
    - Returns typed structs
  - `migrations/003_dwd_tables.sql` - dwd.order_level, dwd.item_level schemas
  - `migrations/004_mart_tables.sql` - mart.metric_dimension_daily schema
  - `migrations/005_ops_tables.sql` - ops.metric_alert schema
  - `config/access_policy.yml` - Role access restrictions

  **Acceptance Criteria**:
  - [ ] QueryByObjectType("order") returns rows from dwd.order_level
  - [ ] QueryByObjectType("metric_alert") returns rows from ops.metric_alert
  - [ ] SearchObjects("seller", {limit: 10}) returns 10 sellers from dwd.item_level
  - [ ] Default LIMIT 1000 enforced
  - [ ] Role-based denial works (viewer cannot access dwd_order_level)
  - [ ] Unit tests with testcontainers pass

  **QA Scenarios**:
  ```
  Scenario: Query seller objects
    Tool: Bash (go test with testcontainers)
    Steps:
      1. go test ./internal/repository/ -tags integration -v -run TestOntologyRepositorySeller
    Expected: Returns seller rows with correct fields
    Evidence: .sisyphus/evidence/task-8-seller-query.txt

  Scenario: LIMIT enforcement
    Tool: Bash (go test)
    Steps:
      1. go test ./internal/repository/ -tags integration -v -run TestOntologyRepositoryLimit
    Expected: Query without limit returns max 1000 rows
    Evidence: .sisyphus/evidence/task-8-limit-test.txt

  Scenario: Role-based access denial
    Tool: Bash (go test)
    Steps:
      1. go test ./internal/repository/ -tags integration -v -run TestOntologyRepositoryAccessDenied
    Expected: viewer role denied access to dwd tables
    Evidence: .sisyphus/evidence/task-8-access-denied.txt
  ```

  **Commit**: YES (commit 4: feat: add ontology object registry - grouped with Task 7)

- [x] 9. Expand GovernanceService

  **What to do**:
  - Expand `internal/service/governance_service.go` (currently only has GetStatus)
  - Add methods:
    - `GetClassification(ctx, fieldPath string) (*dto.ClassificationResponse, error)`
    - `GetFieldMarking(ctx, objectType, property string) (*dto.FieldMarkingResponse, error)`
    - `CheckAccess(ctx, userRole, objectType, action string) AccessDecision`
    - `GetLineage(ctx, resource string) (*dto.LineageResponse, error)`
    - `RequiresCheckpoint(ctx, action string) bool`
    - `GetHealthChecks(ctx) (*dto.HealthChecksResponse, error)`
    - `GetGovernanceStatus(ctx) (*dto.GovernanceStatusResponse, error)` (enhance existing)
  - Create `internal/governance/` package with:
    - `classification.go` - Classification logic
    - `lineage.go` - Lineage query logic
    - `access_policy.go` - Access policy evaluation
    - `checkpoint.go` - Checkpoint rules
    - `redaction.go` - Redaction engine (Phase 5D prep)
  - Load data from gov.* tables (not YAML directly)

  **Must NOT do**:
  - Do NOT implement RBAC enforcement at HTTP middleware layer
  - Do NOT add graph traversal for lineage (flat queries only)
  - Do NOT modify access_policy.yml semantics

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2
  - **Blocks**: Tasks 12 (Redaction), 14 (Governance API handlers)
  - **Blocked By**: Tasks 1-3, 6

  **References**:
  - `internal/service/governance_service.go` - Existing service (47 lines)
  - `internal/repository/governance_repository.go` - Existing repo
  - `config/data_classification.yml` - Classification levels and field mappings
    - pii, internal, sensitive, public_internal, derived_sensitive
  - `config/access_policy.yml` - Role definitions and allowed actions
    - admin, analyst, viewer, marketing_ops
  - `config/data_lineage.yml` - Nodes and edges
  - `config/health_checks.yml` - Health check definitions
  - `config/checkpoint_rules.yml` - Checkpoint rules

  **Acceptance Criteria**:
  - [ ] GetClassification("dwd_order_level.customer_unique_id") returns pii
  - [ ] GetClassification("metric_daily.gmv") returns derived_sensitive
  - [ ] CheckAccess("analyst", "dwd_item_level", "read") returns ALLOW
  - [ ] CheckAccess("viewer", "dwd_order_level", "read") returns DENY
  - [ ] GetLineage("dwd_order_level") returns upstream [raw_orders_csv, raw_customers_csv]
  - [ ] RequiresCheckpoint("execute_dispatch") returns true
  - [ ] GetHealthChecks() returns all health check definitions
  - [ ] Unit tests pass

  **QA Scenarios**:
  ```
  Scenario: Classification lookup
    Tool: Bash (go test with testcontainers)
    Steps:
      1. go test ./internal/governance/ -tags integration -v -run TestClassification
    Expected: customer_unique_id classified as pii
    Evidence: .sisyphus/evidence/task-9-classification.txt

  Scenario: Access policy evaluation
    Tool: Bash (go test)
    Steps:
      1. go test ./internal/governance/ -tags integration -v -run TestAccessPolicy
    Expected: analyst can read dwd_item_level, viewer cannot
    Evidence: .sisyphus/evidence/task-9-access-policy.txt

  Scenario: Lineage query
    Tool: Bash (go test)
    Steps:
      1. go test ./internal/governance/ -tags integration -v -run TestLineage
    Expected: dwd_order_level upstream includes raw_orders_csv
    Evidence: .sisyphus/evidence/task-9-lineage.txt
  ```

  **Commit**: YES (commit 6: feat: add governance service and redaction - grouped with Task 12)

- [x] 10. Implement ObjectQueryService

  **What to do**:
  - Create `internal/ontology/query_service.go`
  - Implement `ObjectQueryService` with explicit per-type methods:
    - `GetOrder(ctx, orderID string) (*OrderInstance, error)`
    - `GetSeller(ctx, sellerID string) (*SellerInstance, error)`
    - `GetMetricAlert(ctx, alertID string) (*MetricAlertInstance, error)`
    - `SearchSellers(ctx, filters SellerFilters) (*SearchResult, error)`
    - `SearchCategories(ctx, filters CategoryFilters) (*SearchResult, error)`
    - `SearchRegions(ctx, filters RegionFilters) (*SearchResult, error)`
    - `GetSellerMetrics(ctx, sellerID string) (*SellerMetrics, error)`
    - `GetCategoryMetrics(ctx, categoryName string) (*CategoryMetrics, error)`
    - `BuildObjectContext(ctx, objectType, objectID string) (*ObjectContext, error)`
  - Use OntologyRepository for DB queries
  - Apply role-based access control
  - Apply default LIMIT 1000

  **Must NOT do**:
  - Do NOT use dynamic SQL generation
  - Do NOT implement generic Query(ctx, objectType string, filters map[string]interface{})
  - Do NOT bypass LIMIT or access control

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 11-13)
  - **Blocks**: Tasks 11 (Context Builder), 15 (Qoder Context)
  - **Blocked By**: Tasks 6-9

  **References**:
  - `internal/ontology/registry.go` - ObjectRegistry for schema lookups
  - `internal/repository/ontology_repository.go` - OntologyRepository interface
  - `internal/repository/interfaces.go` - Repository interfaces
  - `config/aip_object_schema.yml` - Object-to-table mappings

  **Acceptance Criteria**:
  - [ ] GetOrder returns order from dwd.order_level
  - [ ] GetSeller returns seller from dwd.item_level
  - [ ] SearchSellers returns paginated results with LIMIT
  - [ ] GetMetricAlert returns alert from ops.metric_alert
  - [ ] BuildObjectContext returns structured context for any object type
  - [ ] Role-based access enforced (viewer cannot access sensitive objects)
  - [ ] Unit tests with testcontainers pass

  **QA Scenarios**:
  ```
  Scenario: Get order by ID
    Tool: Bash (go test with testcontainers)
    Steps:
      1. go test ./internal/ontology/ -tags integration -v -run TestQueryServiceOrder
    Expected: Returns order with correct fields
    Evidence: .sisyphus/evidence/task-10-order-query.txt

  Scenario: Search sellers with pagination
    Tool: Bash (go test)
    Steps:
      1. go test ./internal/ontology/ -tags integration -v -run TestQueryServiceSellerSearch
    Expected: Returns sellers with limit/offset
    Evidence: .sisyphus/evidence/task-10-seller-search.txt

  Scenario: Build object context
    Tool: Bash (go test)
    Steps:
      1. go test ./internal/ontology/ -tags integration -v -run TestQueryServiceContext
    Expected: Returns structured context with properties
    Evidence: .sisyphus/evidence/task-10-context.txt
  ```

  **Commit**: YES (commit 5: feat: add object query service and context builder - grouped with Task 11)

- [x] 11. Implement LLM-safe Context Builder

  **What to do**:
  - Create `internal/ontology/context_builder.go`
  - Implement `BuildLLMSafeContext(ctx, input ContextRequest) (*LLMSafeContext, error)`
  - Input: `{object_type, object_id, purpose, role}`
  - Output: `{object_type, object_id, properties, redacted_fields, lineage, allowed_actions, forbidden_actions}`
  - Redaction rules:
    - Strip fields with classification: pii, sensitive, derived_sensitive
    - Strip fields with marking: PII, FINANCIAL_INTERNAL
    - For agent_readonly role: exclude all pii, sensitive, derived_sensitive
    - Include redaction log with field name and reason
  - Use GovernanceService for classification and access policy lookups
  - Use ObjectQueryService for object data

  **Must NOT do**:
  - Do NOT call any LLM APIs
  - Do NOT include raw SQL or internal table names in output
  - Do NOT bypass redaction for any role

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 15 (Qoder Context enrichment)
  - **Blocked By**: Tasks 6-9, 10

  **References**:
  - `config/data_classification.yml` - Classification levels
  - `config/data_markings.yml` - Field markings
  - `config/access_policy.yml` - Role permissions
  - `internal/governance/redaction.go` - Redaction engine

  **Acceptance Criteria**:
  - [ ] PII fields (customer_unique_id) excluded from output
  - [ ] Sensitive fields (payment_value) excluded from output
  - [ ] Redacted_fields list includes field name and reason
  - [ ] agent_readonly role gets most restrictive view
  - [ ] admin role gets full view (no redaction)
  - [ ] Output is deterministic (same input → same output)
  - [ ] Unit tests pass

  **QA Scenarios**:
  ```
  Scenario: PII redaction for agent_readonly
    Tool: Bash (go test)
    Steps:
      1. go test ./internal/ontology/ -v -run TestContextBuilderRedactionPII
    Expected: customer_unique_id redacted with reason
    Evidence: .sisyphus/evidence/task-11-redaction-pii.txt

  Scenario: Admin role no redaction
    Tool: Bash (go test)
    Steps:
      1. go test ./internal/ontology/ -v -run TestContextBuilderAdmin
    Expected: All fields included, redacted_fields empty
    Evidence: .sisyphus/evidence/task-11-admin-no-redaction.txt

  Scenario: Deterministic output
    Tool: Bash (go test)
    Steps:
      1. go test ./internal/ontology/ -v -run TestContextBuilderDeterministic
    Expected: Same input produces identical output
    Evidence: .sisyphus/evidence/task-11-deterministic.txt
  ```

  **Commit**: YES (commit 5: feat: add object query service and context builder)

- [x] 12. Implement Redaction Engine

  **What to do**:
  - Create `internal/governance/redaction.go`
  - Implement `RedactObjectContext(ctx ObjectContext, policy RedactionPolicy) ObjectContext`
  - Classification-based redaction:
    - pii → always redact
    - sensitive → redact for analyst, viewer, agent_readonly
    - derived_sensitive → redact for viewer, agent_readonly
    - internal → redact for viewer
    - public_internal → never redact
  - Marking-based redaction:
    - PII → always redact
    - FINANCIAL_INTERNAL → redact for non-admin
    - OPERATIONAL_INTERNAL → redact for viewer
    - RAW_DATA → redact for non-admin
  - Return redaction log with field, reason, rule triggered

  **Must NOT do**:
  - Do NOT implement tokenization or masking (full exclusion only)
  - Do NOT add configurable redaction rules (hardcoded for Phase 5)

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 11 (Context Builder)
  - **Blocked By**: Tasks 6-9

  **References**:
    - `config/data_classification.yml` - Classification levels
    - `config/data_markings.yml` - Marking definitions

  **Acceptance Criteria**:
  - [ ] All PII fields redacted for all roles
  - [ ] Sensitive fields redacted for non-admin roles
  - [ ] Redaction log includes field, reason, rule
  - [ ] Empty context handled gracefully
  - [ ] Unit tests pass

  **QA Scenarios**:
  ```
  Scenario: Redaction by classification level
    Tool: Bash (go test)
    Steps:
      1. go test ./internal/governance/ -v -run TestRedactionClassification
    Expected: pii fields redacted, public_internal fields kept
    Evidence: .sisyphus/evidence/task-12-redaction-class.txt

  Scenario: Redaction by marking
    Tool: Bash (go test)
    Steps:
      1. go test ./internal/governance/ -v -run TestRedactionMarking
    Expected: FINANCIAL_INTERNAL fields redacted for analyst
    Evidence: .sisyphus/evidence/task-12-redaction-marking.txt
  ```

  **Commit**: YES (commit 6: feat: add governance service and redaction)

- [x] 13. Implement ContextRepository for Qoder

  **What to do**:
  - Create `internal/repository/context_repository.go`
  - Encapsulate all Qoder context data queries:
    - `GetLastPipelineRun(ctx) (*PipelineRunInfo, error)`
    - `GetAlerts(ctx, severity string, limit int) ([]AlertSummary, error)`
    - `GetOpenTasks(ctx, limit int) ([]TaskSummary, error)`
    - `GetPendingOutbox(ctx, limit int) ([]OutboxSummary, error)`
  - Replace direct SQL in QoderService with ContextRepository calls
  - Follow repository interface pattern

  **Must NOT do**:
  - Do NOT modify existing QoderService behavior (just refactor)
  - Do NOT change response format

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 15 (Qoder Context enrichment)
  - **Blocked By**: Tasks 1-3

  **References**:
  - `internal/service/qoder_service.go` - Current QoderService (373 lines)
    - Lines 51-77: GetContext aggregates 4 sources
    - Lines 79-120: queryLastPipelineRun
    - Lines 122-180: queryAlerts
    - Lines 182-240: queryOpenTasks
    - Lines 242-300: queryPendingOutbox
  - `internal/repository/interfaces.go` - ContextRepository interface

  **Acceptance Criteria**:
  - [ ] All 4 query methods implemented in ContextRepository
  - [ ] QoderService uses ContextRepository (not direct SQL)
  - [ ] Existing tests still pass
  - [ ] No behavior changes

  **QA Scenarios**:
  ```
  Scenario: ContextRepository queries
    Tool: Bash (go test with testcontainers)
    Steps:
      1. go test ./internal/repository/ -tags integration -v -run TestContextRepository
    Expected: All 4 query methods return correct data
    Evidence: .sisyphus/evidence/task-13-context-repo.txt
  ```

  **Commit**: YES (commit 7: feat: expand governance read api and enrich qoder context - grouped with Tasks 14-15)

- [x] 14. Expand Governance API Handlers

  **What to do**:
  - Expand `internal/api/handler/governance.go`
  - Add handlers for 6 new endpoints:
    - `GET /api/v1/governance/catalog` - List objects and datasets
    - `GET /api/v1/governance/classification` - List classifications
    - `GET /api/v1/governance/markings` - List field markings
    - `GET /api/v1/governance/lineage` - List lineage (flat, with ?resource and ?object_type filters)
    - `GET /api/v1/governance/checkpoints` - List checkpoint rules
    - `GET /api/v1/governance/health` - List health checks
  - Enhance existing `GET /api/v1/governance/status` with object_schema_count
  - All endpoints require Bearer Token (existing middleware)
  - Return JSON with consistent error format

  **Must NOT do**:
  - Do NOT add write endpoints (POST/PUT/DELETE)
  - Do NOT modify auth middleware
  - Do NOT change existing response field names/types

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4 (with Tasks 15-18)
  - **Blocks**: Task 19 (API smoke tests)
  - **Blocked By**: Tasks 6-9, 12

  **References**:
  - `internal/api/handler/governance.go` - Existing handler (37 lines)
  - `internal/api/handler/qoder.go` - Handler pattern with interfaces
  - `internal/api/dto/governance.go` - Existing DTOs
  - `internal/api/server.go` - Route registration (line 115)
  - `api/routers/governance.py` - Python FastAPI governance router (reference for response shapes)

  **Acceptance Criteria**:
  - [ ] All 6 new endpoints return 200 with correct JSON
  - [ ] /governance/status enhanced with object_schema_count
  - [ ] /governance/lineage supports ?resource and ?object_type query params
  - [ ] All endpoints require Bearer Token
  - [ ] Consistent error format for 404, 500
  - [ ] Handler tests pass

  **QA Scenarios**:
  ```
  Scenario: Governance catalog endpoint
    Tool: Bash (curl)
    Steps:
      1. curl -s -H "Authorization: Bearer $API_BEARER_TOKEN" http://localhost:8080/api/v1/governance/catalog | jq
    Expected: Returns objects array with 8 types and datasets array
    Evidence: .sisyphus/evidence/task-14-catalog.json

  Scenario: Governance classification endpoint
    Tool: Bash (curl)
    Steps:
      1. curl -s -H "Authorization: Bearer $API_BEARER_TOKEN" http://localhost:8080/api/v1/governance/classification | jq
    Expected: Returns levels and resources arrays
    Evidence: .sisyphus/evidence/task-14-classification.json

  Scenario: Governance lineage endpoint with filter
    Tool: Bash (curl)
    Steps:
      1. curl -s -H "Authorization: Bearer $API_BEARER_TOKEN" "http://localhost:8080/api/v1/governance/lineage?resource=dwd_order_level" | jq
    Expected: Returns upstream and downstream arrays
    Evidence: .sisyphus/evidence/task-14-lineage.json

  Scenario: Unauthorized access
    Tool: Bash (curl)
    Steps:
      1. curl -s http://localhost:8080/api/v1/governance/catalog -w "%{http_code}"
    Expected: 401 Unauthorized
    Evidence: .sisyphus/evidence/task-14-unauthorized.txt
  ```

  **Commit**: YES (commit 7: feat: expand governance read api and enrich qoder context)

- [x] 15. Enrich Qoder Context

  **What to do**:
  - Modify `internal/service/qoder_service.go`
  - Enhance ContextResponse with:
    - `ontology.object_types` - Available object types
    - `ontology.objects_available` - Boolean
    - `governance.classification_loaded` - Boolean
    - `governance.lineage_loaded` - Boolean
    - `governance.access_policy_loaded` - Boolean
    - `governance.redaction_enabled` - Boolean
    - `agent_policy.role` - Current role (default: analyst)
    - `agent_policy.can_read_objects` - Boolean
    - `agent_policy.can_execute_actions` - false (Phase 5)
    - `agent_policy.can_write_reports` - false (Phase 5)
  - Use ContextRepository for existing data queries
  - Use ObjectRegistry for ontology metadata
  - Use GovernanceService for governance status
  - Keep existing fields unchanged (backward compatible)

  **Must NOT do**:
  - Do NOT remove existing fields (summary, top_alerts, open_tasks, etc.)
  - Do NOT change existing field types
  - Do NOT enable action execution

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4
  - **Blocks**: Task 19 (API smoke tests)
  - **Blocked By**: Tasks 10-13

  **References**:
  - `internal/service/qoder_service.go` - Current implementation
  - `internal/api/dto/qoder.go` - ContextResponse struct
  - `internal/api/handler/qoder.go` - QoderHandler
  - `migration_baseline/api_responses/qoder_context.json` - Baseline response

  **Acceptance Criteria**:
  - [ ] /qoder/context includes all new fields
  - [ ] Existing fields unchanged
  - [ ] agent_policy.can_execute_actions = false
  - [ ] Response size < 100KB
  - [ ] Tests pass

  **QA Scenarios**:
  ```
  Scenario: Qoder context enriched
    Tool: Bash (curl)
    Steps:
      1. curl -s -H "Authorization: Bearer $API_BEARER_TOKEN" http://localhost:8080/api/v1/qoder/context | jq
    Expected: Contains ontology, governance, agent_policy sections
    Evidence: .sisyphus/evidence/task-15-qoder-context.json

  Scenario: Backward compatibility
    Tool: Bash (curl)
    Steps:
      1. curl -s -H "Authorization: Bearer $API_BEARER_TOKEN" http://localhost:8080/api/v1/qoder/context | jq '.summary, .top_alerts, .open_tasks'
    Expected: All existing fields present and non-null
    Evidence: .sisyphus/evidence/task-15-backward-compat.json
  ```

  **Commit**: YES (commit 7: feat: expand governance read api and enrich qoder context)

- [x] 16. Add Governance API DTOs

  **What to do**:
  - Expand `internal/api/dto/governance.go`
  - Add DTOs for new endpoints:
    - `CatalogResponse` - Objects and datasets catalog
    - `ClassificationResponse` - Classification levels and resources
    - `MarkingsResponse` - Field-level markings
    - `LineageResponse` - Lineage upstream/downstream
    - `CheckpointsResponse` - Checkpoint rules
    - `HealthChecksResponse` - Health check definitions
  - Enhance `GovernanceStatusResponse` with object_schema_count

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4
  - **Blocks**: Task 14
  - **Blocked By**: Tasks 6-9

  **References**:
  - `internal/api/dto/governance.go` - Existing DTOs
  - `api/schemas.py` - Python schemas (reference for response shapes)

  **Acceptance Criteria**:
  - [ ] All DTOs have JSON tags
  - [ ] All DTOs compile
  - [ ] Consistent naming pattern

  **QA Scenarios**:
  ```
  Scenario: DTOs compile
    Tool: Bash
    Steps:
      1. go build ./internal/api/dto/...
    Expected: Build succeeds
    Evidence: .sisyphus/evidence/task-16-dto-compile.txt
  ```

  **Commit**: NO (groups with Task 14 in commit 7)

- [x] 17. Wire Routes in Server

  **What to do**:
  - Update `internal/api/server.go`
  - Add route registrations for 6 new governance endpoints
  - Ensure auth middleware applied to all new endpoints
  - Lazy initialize new handlers following existing pattern

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4
  - **Blocks**: Task 19
  - **Blocked By**: Tasks 14-16

  **References**:
  - `internal/api/server.go` - Route setup (lines 93-123)

  **Acceptance Criteria**:
  - [ ] All 6 new endpoints registered
  - [ ] Auth middleware applied
  - [ ] Server compiles and starts

  **QA Scenarios**:
  ```
  Scenario: Server starts with new routes
    Tool: Bash
    Steps:
      1. go run ./cmd/baxi-api &
      2. sleep 2
      3. curl -s http://localhost:8080/api/v1/health | jq '.status'
    Expected: "ok"
    Evidence: .sisyphus/evidence/task-17-server-start.txt
  ```

  **Commit**: NO (groups with Task 14 in commit 7)

- [x] 18. Implement CLI Commands

  **What to do**:
  - Add to `cmd/baxi-cli/`:
    - `governance load --config-dir ./config` - Load and sync YAML configs
    - `governance check` - Verify all configs loaded correctly
  - Use ConfigLoader for implementation
  - Return exit code 0 on success, 1 on failure
  - Add help text

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4
  - **Blocks**: Task 21
  - **Blocked By**: Tasks 6

  **References**:
  - `cmd/baxi-cli/` - Existing CLI structure
  - `internal/configloader/` - ConfigLoader package

  **Acceptance Criteria**:
  - [ ] `go run ./cmd/baxi-cli governance load --config-dir ./config` works
  - [ ] `go run ./cmd/baxi-cli governance check` works
  - [ ] Exit codes correct
  - [ ] Help text available

  **QA Scenarios**:
  ```
  Scenario: CLI load command
    Tool: Bash
    Steps:
      1. go run ./cmd/baxi-cli governance load --config-dir ./config
      2. echo $?
    Expected: Exit code 0
    Evidence: .sisyphus/evidence/task-18-cli-load.txt

  Scenario: CLI check command
    Tool: Bash
    Steps:
      1. go run ./cmd/baxi-cli governance check
      2. echo $?
    Expected: Exit code 0
    Evidence: .sisyphus/evidence/task-18-cli-check.txt
  ```

  **Commit**: YES (commit 3: feat: add governance config loader - grouped with Task 6)

- [x] 19. Integration Test Suite

  **What to do**:
  - Add integration tests with testcontainers-go for all new repositories
  - Test files:
    - `internal/repository/ontology_repository_test.go`
    - `internal/repository/context_repository_test.go`
    - `internal/governance/*_test.go`
    - `internal/ontology/*_test.go`
    - `internal/configloader/*_test.go`
  - Each test:
    - Starts PostgreSQL container
    - Runs goose migrations
    - Loads test data
    - Runs test queries
    - Verifies results
  - Use `//go:build integration` tag

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 5 (with Tasks 20-22)
  - **Blocks**: None
  - **Blocked By**: Tasks 6-18

  **References**:
  - `internal/testutil/db.go` - Modernized testcontainers setup
  - `migrations/` - Goose migrations
  - `internal/repository/governance_repository_test.go` - Existing repo test pattern

  **Acceptance Criteria**:
  - [ ] All new repositories have integration tests
  - [ ] `go test -tags integration ./internal/repository/...` passes
  - [ ] `go test -tags integration ./internal/governance/...` passes
  - [ ] `go test -tags integration ./internal/ontology/...` passes
  - [ ] Container startup + migrations < 10 seconds

  **QA Scenarios**:
  ```
  Scenario: Integration tests pass
    Tool: Bash
    Steps:
      1. go test -tags integration ./internal/repository/... ./internal/governance/... ./internal/ontology/... -v
    Expected: All tests PASS
    Evidence: .sisyphus/evidence/task-19-integration-tests.txt
  ```

  **Commit**: NO (groups with related feature commits)

- [x] 20. API Smoke Tests

  **What to do**:
  - Test all new API endpoints with real server
  - Test sequence:
    1. Start PostgreSQL (`make up`)
    2. Run migrations (`make migrate`)
    3. Load configs (`make governance-load`)
    4. Start API (`make api`)
    5. Test all endpoints with curl
  - Verify response shapes match expected JSON

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 5
  - **Blocks**: None
  - **Blocked By**: Tasks 14-18

  **Acceptance Criteria**:
  - [ ] All 6 new governance endpoints return 200
  - [ ] /qoder/context returns enriched response
  - [ ] Auth required on all protected endpoints
  - [ ] Response times < 500ms

  **QA Scenarios**:
  ```
  Scenario: All governance endpoints
    Tool: Bash (curl script)
    Steps:
      1. ./scripts/smoke_test_governance.sh
    Expected: All endpoints return 200 with valid JSON
    Evidence: .sisyphus/evidence/task-20-smoke-tests.txt

  Scenario: Qoder context response time
    Tool: Bash (curl with timing)
    Steps:
      1. curl -w "@curl-format.txt" -s -H "Authorization: Bearer $API_BEARER_TOKEN" http://localhost:8080/api/v1/qoder/context
    Expected: Total time < 500ms
    Evidence: .sisyphus/evidence/task-20-response-time.txt
  ```

  **Commit**: NO

- [x] 21. Run Regression Tests

  **What to do**:
  - Run `make test` - All Go tests
  - Run `make api-compare` - Compare API responses against baseline
  - Run `make pipeline-compare` - Compare pipeline output against baseline
  - Fix any failures

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 5
  - **Blocks**: None
  - **Blocked By**: Tasks 19-20

  **Acceptance Criteria**:
  - [ ] `go test ./...` passes
  - [ ] `make api-compare` passes or accepted WARN
  - [ ] `make pipeline-compare` passes or accepted WARN

  **QA Scenarios**:
  ```
  Scenario: All tests pass
    Tool: Bash
    Steps:
      1. make test
    Expected: PASS
    Evidence: .sisyphus/evidence/task-21-tests.txt

  Scenario: API compare
    Tool: Bash
    Steps:
      1. make api-compare
    Expected: PASS or accepted WARN
    Evidence: .sisyphus/evidence/task-21-api-compare.txt
  ```

  **Commit**: NO

- [x] 22. Verify Scope Compliance

  **What to do**:
  - Run `git diff --name-only` and verify:
    - No changes to `api/` (Python)
    - No changes to `frontend/` (React)
    - No changes to `pipeline/` (Python)
    - No changes to `scripts/` (frozen)
    - No changes to `config/*.yml` semantics
  - Verify no LLM API calls added
  - Verify no action execution added
  - Verify no outbox dispatch modifications

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 5
  - **Blocks**: None
  - **Blocked By**: Tasks 14-18

  **Acceptance Criteria**:
  - [ ] No Python files modified
  - [ ] No React files modified
  - [ ] No pipeline logic modified
  - [ ] No YAML semantics modified
  - [ ] No LLM API calls found
  - [ ] No action execution found

  **QA Scenarios**:
  ```
  Scenario: Scope verification
    Tool: Bash
    Steps:
      1. git diff --name-only | grep -E "^(api/|frontend/|pipeline/|scripts/)" || echo "No forbidden changes"
      2. grep -r "llm" internal/ || echo "No LLM references"
      3. grep -r "action.*execute" internal/ || echo "No action execution"
    Expected: No forbidden changes
    Evidence: .sisyphus/evidence/task-22-scope-verify.txt
  ```

  **Commit**: NO

---

## Final Verification Wave

> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, curl endpoint, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in .sisyphus/evidence/. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `tsc --noEmit` + `go vet ./...` + `go test ./...`. Review all changed files for: `as any`, empty catches, `fmt.Println` in prod, commented-out code, unused imports. Check AI slop: excessive comments, over-abstraction, generic names.
  Output: `Build [PASS/FAIL] | Lint [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [ ] F3. **Real Manual QA** — `unspecified-high`
  Start from clean state. Execute EVERY QA scenario from EVERY task — follow exact steps, capture evidence. Test cross-task integration. Test edge cases: empty state, invalid input, rapid actions. Save to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff (git log/diff). Verify 1:1 — everything in spec was built, nothing beyond spec was built. Check "Must NOT do" compliance. Detect cross-task contamination.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

```
Commit 1: docs: add phase 5 governance ontology runtime plan
  Files: docs/migration/phase-5-governance-ontology-runtime-plan.md

Commit 2: chore: modernize testcontainers and add repository interfaces
  Files: internal/testutil/db.go, internal/repository/interfaces.go, migrations/009_gov_indexes.sql

Commit 3: feat: add governance config loader
  Files: internal/configloader/*, cmd/baxi-cli/*, Makefile

Commit 4: feat: add ontology object registry and repository
  Files: internal/ontology/registry.go, internal/ontology/schema.go, internal/repository/ontology_repository.go

Commit 5: feat: add object query service and context builder
  Files: internal/ontology/query_service.go, internal/ontology/context_builder.go

Commit 6: feat: add governance service and redaction
  Files: internal/governance/*, internal/repository/governance_repository.go (expanded)

Commit 7: feat: expand governance read api and enrich qoder context
  Files: internal/api/handler/governance.go, internal/api/handler/qoder.go, internal/service/qoder_service.go, internal/api/server.go
```

---

## Success Criteria

### Verification Commands
```bash
# 1. Database setup
docker compose up -d postgres
export DATABASE_URL="postgres://baxi:baxi_dev@localhost:5432/baxi?sslmode=disable"
export API_BEARER_TOKEN="dev_token_32_chars_minimum_length_here"
make migrate

# 2. Pipeline data generation
make pipeline DATA_DIR=./data/raw

# 3. Config loading
make governance-load

# 4. Config checking
make governance-check

# 5. Start API
make api

# 6. Governance endpoints
curl -s -H "Authorization: Bearer $API_BEARER_TOKEN" http://localhost:8080/api/v1/governance/status | jq
curl -s -H "Authorization: Bearer $API_BEARER_TOKEN" http://localhost:8080/api/v1/governance/catalog | jq
curl -s -H "Authorization: Bearer $API_BEARER_TOKEN" http://localhost:8080/api/v1/governance/classification | jq
curl -s -H "Authorization: Bearer $API_BEARER_TOKEN" http://localhost:8080/api/v1/governance/markings | jq
curl -s -H "Authorization: Bearer $API_BEARER_TOKEN" http://localhost:8080/api/v1/governance/lineage | jq
curl -s -H "Authorization: Bearer $API_BEARER_TOKEN" http://localhost:8080/api/v1/governance/checkpoints | jq
curl -s -H "Authorization: Bearer $API_BEARER_TOKEN" http://localhost:8080/api/v1/governance/health | jq

# 7. Qoder context
curl -s -H "Authorization: Bearer $API_BEARER_TOKEN" http://localhost:8080/api/v1/qoder/context | jq

# 8. Regression tests
make pipeline-compare
make api-compare
go test ./...

# 9. Verify no unintended changes
git diff --name-only | grep -v "^internal/" | grep -v "^cmd/" | grep -v "^migrations/" | grep -v "^docs/" | grep -v "^Makefile"
```

### Final Checklist
- [ ] All "Must Have" present and working
- [ ] All "Must NOT Have" absent (no LLM calls, no action execution, etc.)
- [ ] All 8 object types registered and queryable
- [ ] All 29 YAML configs loadable
- [ ] PII/sensitive fields redacted in LLM context
- [ ] Governance API returns correct JSON for all 6 new endpoints
- [ ] Qoder context enriched with ontology + governance
- [ ] Existing API responses unchanged (backward compatible)
- [ ] All tests pass (including testcontainers integration tests)
- [ ] No Python/React/Pipeline modifications
- [ ] Plan document exists at docs/migration/phase-5-governance-ontology-runtime-plan.md
