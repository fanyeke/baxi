# Phase 6: Decision Case / LLM Context Builder / Action Proposal

## TL;DR

> **Core Objective**: Build a controlled decision pipeline that converts `ops.metric_alert` into `ai.decision_case`, constructs a governance-redacted LLM-safe context, generates structured decisions via rule-based fallback (LLM disabled by default), and produces `ai.action_proposal` with mandatory human review.
>
> **Deliverables**:
> - `migrations/010_ai_tables_enhance.sql` — Schema enhancements for decision_case, action_proposal, llm_decision
> - `internal/decision/` — Case lifecycle, context builder, decision engine
> - `internal/llm/` — DecisionProvider interface, RuleBasedProvider, DisabledProvider
> - `internal/repository/decision_repository.go` — Data access layer
> - `internal/service/decision_service.go` — Business orchestration
> - `internal/api/dto/decision.go` + `internal/api/handler/decision.go` — HTTP API
> - `cmd/baxi-cli/decision.go` — CLI commands
> - `docs/migration/phase-6-decision-case-llm-context-plan.md` — Design document
>
> **Estimated Effort**: Large (7-8 commits, ~15 tasks)
> **Parallel Execution**: YES — 4 waves with 4-5 tasks each
> **Critical Path**: Schema → Repository → Domain Services → API → CLI → Final Verification

---

## Context

### Original Request
Implement Phase 6 of the Baxi Go + PostgreSQL migration: Decision Case lifecycle, LLM-safe Context Builder, Decision Engine with rule-based fallback, and Action Proposal generation. This phase bridges Phase 5 (Governance/Ontology) and Phase 7 (Review/Execution).

### Interview Summary
**Key Discussions**:
- Action types: Fixed enum (create_followup_task, notify_owner, export_report, escalate_to_human)
- Migration strategy: Audit schema first, create enhancement migration (008) — never modify 007
- Default provider: RuleBasedProvider with severity-to-action mapping
- LLM: Disabled by default (LLM_ENABLED=false), interface defined but no real calls

**Research Findings**:
- Phase 5 complete: `internal/ontology/` (6 files), `internal/governance/` (6 files)
- `migrations/007_ai_tables.sql` exists but MISSING fields required by spec
- `internal/decision/`, `internal/action/`, `internal/llm/` do NOT exist
- Existing patterns: Repository (alert_repository.go), Handler (alerts.go, governance.go), Service (governance_service.go), CLI (governance.go)
- `internal/ontology/context_builder.go` has data fetch gap — Phase 6 must compose with `ObjectQueryService.BuildObjectContext`

### Metis Review
**Identified Gaps** (addressed):
- **Schema drift risk**: Resolved — commit 1 is isolated migration `008_ai_tables_enhance.sql`, never modify 007
- **Idempotency race condition**: Resolved — partial unique index at DB level + application-level check
- **Context build timeout**: Resolved — add context timeout in service layer
- **Role confusion**: Resolved — hardcode `agent_readonly` role for decision contexts
- **Active case definition**: Resolved — `status NOT IN ('closed', 'failed')`
- **Rule-based logic**: Resolved — default severity→action mapping (see Task 9)
- **Governance snapshot**: Resolved — store classifications/markings for relevant object only
- **Context hash**: Resolved — SHA256 of final redacted context JSON

---

## Work Objectives

### Core Objective
Build a controlled decision pipeline: alert → decision_case → governed context → rule-based decision → action_proposal, with all proposals requiring human review and no automatic execution.

### Concrete Deliverables
- `migrations/010_ai_tables_enhance.sql`
- `internal/decision/case_service.go`, `repository.go`, `idgen.go`, `context_builder.go`, `engine.go`, `rule_based.go`, `schema.go`
- `internal/llm/provider.go`, `disabled_provider.go`, `rule_provider.go`, `schema_validator.go`
- `internal/repository/decision_repository.go`
- `internal/service/decision_service.go`
- `internal/api/dto/decision.go`
- `internal/api/handler/decision.go`
- `cmd/baxi-cli/decision.go`
- `docs/migration/phase-6-decision-case-llm-context-plan.md`
- Updated `.env.example` with LLM config
- Updated `Makefile` with decision targets

### Definition of Done
- [ ] `make migrate` applies 008 migration successfully
- [ ] `POST /api/v1/decisions/cases` creates case from alert
- [ ] Duplicate alert creation returns existing case (idempotent)
- [ ] `POST /context` builds LLM-safe context with governance/redaction
- [ ] `POST /decide` generates rule-based decision with valid schema
- [ ] `GET /proposals` lists proposals with `requires_human_review=true`
- [ ] No real LLM API calls made (LLM_ENABLED=false)
- [ ] No actions executed
- [ ] No outbox events generated
- [ ] All regression tests pass

### Must Have
- Idempotent case creation from alert
- LLM-safe context with governance snapshot and redaction
- Rule-based decision provider as default
- Schema-validated decision output
- Action proposals with requires_human_review=true
- Decision API endpoints with Bearer auth
- CLI commands for decision workflow
- Design document

### Must NOT Have (Guardrails)
- Real LLM API calls
- Action execution
- Outbox event generation
- Pipeline calculation logic changes
- Python code modifications
- React frontend modifications
- LLM direct SQL access
- LLM direct database writes
- Automatic action application

---

## Verification Strategy

### Test Decision
- **Infrastructure exists**: YES — `go test`, testcontainers integration tests in Phase 5
- **Automated tests**: YES (Tests after) — Repository integration tests, service unit tests, handler tests
- **Framework**: Go standard testing + testify + testcontainers
- **Agent-Executed QA**: MANDATORY for all tasks

### QA Policy
Every task MUST include agent-executed QA scenarios. Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **API/Backend**: Use Bash (curl) — Send requests, assert status + response fields
- **Service/Domain**: Use Bash (go test) — Run tests, verify pass/fail
- **CLI**: Use Bash (go run) — Run commands, validate output
- **Database**: Use Bash (psql) — Query tables, verify state

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Foundation — Schema + Repository + Design Doc):
├── Task 1: Schema enhancement migration (008_ai_tables_enhance.sql)
├── Task 2: Decision repository layer (decision_repository.go + tests)
├── Task 3: Design document (phase-6-decision-case-llm-context-plan.md)
└── Task 4: ID generator + utility functions (idgen.go)

Wave 2 (Domain Layer — MAX PARALLEL):
├── Task 5: Case service (case lifecycle, CreateCaseFromAlert)
├── Task 6: Decision context builder (compose ontology + governance + redaction)
├── Task 7: LLM provider interface + disabled provider
└── Task 8: Decision schema + validator

Wave 3 (Core Logic):
├── Task 9: Rule-based decision provider (severity → action mapping)
├── Task 10: Decision engine (orchestrate provider + validator + fallback)
└── Task 11: Action proposal service (generate proposals from decisions)

Wave 4 (API + CLI + Integration):
├── Task 12: Decision service (business orchestration layer)
├── Task 13: API DTOs + handlers (decision.go)
├── Task 14: Server wiring (route registration)
├── Task 15: CLI commands (cmd/baxi-cli/decision.go)
└── Task 16: Config updates (.env.example, Makefile)

Wave FINAL (After ALL tasks — 4 parallel reviews):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Integration QA (unspecified-high)
└── Task F4: Scope fidelity check (deep)
-> Present results -> Get explicit user okay

Critical Path: T1 → T2 → T5 → T6 → T9 → T10 → T11 → T12 → T13 → T14 → F1-F4 → user okay
Parallel Speedup: ~60% faster than sequential
Max Concurrent: 4 (Wave 1 & 2)
```

### Dependency Matrix

| Task | Depends On | Blocks |
|------|-----------|--------|
| 1 (Schema) | — | 2, 5, 8 |
| 2 (Repository) | 1 | 5, 6, 9, 10, 11 |
| 3 (Design Doc) | — | — |
| 4 (ID Gen) | — | 5 |
| 5 (Case Service) | 1, 2, 4 | 6, 9, 10, 11 |
| 6 (Context Builder) | 2, 5 | 9, 10 |
| 7 (Provider Interface) | — | 9, 10 |
| 8 (Schema Validator) | — | 9, 10 |
| 9 (Rule Provider) | 7, 8 | 10 |
| 10 (Decision Engine) | 5, 6, 7, 8, 9 | 11, 12 |
| 11 (Proposal Service) | 2, 10 | 12 |
| 12 (Decision Service) | 5, 6, 10, 11 | 13 |
| 13 (API Handlers) | 12 | 14 |
| 14 (Server Wiring) | 13 | — |
| 15 (CLI) | 12 | — |
| 16 (Config) | — | — |

### Agent Dispatch Summary

- **Wave 1**: 4 tasks — T1(quick), T2(unspecified-high), T3(writing), T4(quick)
- **Wave 2**: 4 tasks — T5(unspecified-high), T6(deep), T7(quick), T8(quick)
- **Wave 3**: 3 tasks — T9(unspecified-high), T10(deep), T11(unspecified-high)
- **Wave 4**: 5 tasks — T12(unspecified-high), T13(unspecified-high), T14(quick), T15(unspecified-high), T16(quick)
- **FINAL**: 4 tasks — F1(oracle), F2(unspecified-high), F3(unspecified-high), F4(deep)

---

## TODOs

> Implementation + Test = ONE Task. Never separate.
> EVERY task MUST have: Recommended Agent Profile + Parallelization info + QA Scenarios.
> **A task WITHOUT QA Scenarios is INCOMPLETE. No exceptions.**

- [x] 1. **Schema Enhancement Migration** (`migrations/010_ai_tables_enhance.sql`)

  **What to do**:
  - Create `migrations/010_ai_tables_enhance.sql` with `ALTER TABLE` statements
  - Add to `ai.decision_case`: `source_type TEXT NOT NULL`, `source_id TEXT NOT NULL`, `object_type TEXT`, `object_id TEXT`, `severity TEXT`, `context_hash TEXT`, `governance_snapshot_json JSONB`, `created_by TEXT`, `error_message TEXT`, `updated_at TIMESTAMPTZ`
  - Add to `ai.action_proposal`: `title TEXT NOT NULL`, `description TEXT`, `risk_level TEXT`, `requires_human_review BOOLEAN DEFAULT TRUE`
  - Add to `ai.llm_decision`: `status TEXT`, `fallback_reason TEXT`, `validation_errors JSONB`
  - Add `CHECK` constraints: decision_case status enum, action_proposal apply_status enum (proposed/approved/rejected), action_type enum
  - Add partial unique index: `CREATE UNIQUE INDEX idx_ai_decision_case_active_source ON ai.decision_case(source_type, source_id) WHERE status NOT IN ('closed', 'failed')`
  - Add standard indexes for new columns
  - Include `+goose Up` and `+goose Down` sections

  **Must NOT do**:
  - Modify `007_ai_tables.sql` (already applied to databases)
  - Add columns not in spec
  - Remove existing columns from 007

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: SQL DDL changes are straightforward
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2, 3, 4)
  - **Blocks**: Tasks 2, 5, 8
  - **Blocked By**: None

  **References**:
  - `migrations/007_ai_tables.sql` — Base schema to enhance
  - `migrations/006_gov_tables.sql` — Pattern for CHECK constraints and indexes
  - Goose migration format: up/down sections

  **Acceptance Criteria**:
  - [ ] Migration file exists at `migrations/010_ai_tables_enhance.sql`
  - [ ] `make migrate` applies successfully (no errors)
  - [ ] `make migrate-status` shows 008 as applied
  - [ ] `make migrate-down` reverts 008 successfully
  - [ ] psql query shows new columns exist: `\d ai.decision_case`

  **QA Scenarios**:

  ```
  Scenario: Migration applies cleanly
    Tool: Bash (psql)
    Preconditions: postgres running, 001-007 applied
    Steps:
      1. Run: make migrate
      2. Query: psql $DATABASE_URL -c "\d ai.decision_case"
    Expected Result: Columns source_type, source_id, object_type, object_id, severity, context_hash, governance_snapshot_json, created_by, error_message, updated_at visible
    Evidence: .sisyphus/evidence/task-1-migration-applies.txt

  Scenario: Idempotency index exists
    Tool: Bash (psql)
    Preconditions: 008 applied
    Steps:
      1. Query: psql $DATABASE_URL -c "\di ai.idx_ai_decision_case_active_source"
    Expected Result: Index found with WHERE condition
    Evidence: .sisyphus/evidence/task-1-index-exists.txt
  ```

  **Commit**: YES (Commit 1)
  - Message: `feat(migration): add 008_ai_tables_enhance.sql for decision case schema`
  - Files: `migrations/010_ai_tables_enhance.sql`
  - Pre-commit: `make migrate && make migrate-status`

- [x] 2. **Decision Repository Layer** (`internal/repository/decision_repository.go`)

  **What to do**:
  - Create `internal/repository/decision_repository.go`
  - Define `DecisionRepository` struct with `pool *pgxpool.Pool`
  - Implement methods:
    - `CreateCase(ctx, case *DecisionCase) error`
    - `GetCaseByID(ctx, caseID string) (*DecisionCase, error)`
    - `GetCaseBySource(ctx, sourceType, sourceID string) (*DecisionCase, error)`
    - `UpdateCaseStatus(ctx, caseID, status string, contextJSON, contextHash, governanceSnapshot *string) error`
    - `ListCases(ctx, filter CaseFilter) ([]DecisionCase, int, error)`
    - `CreateDecision(ctx, decision *LLMDecision) error`
    - `CreateProposal(ctx, proposal *ActionProposal) error`
    - `ListProposalsByCase(ctx, caseID string) ([]ActionProposal, error)`
  - Follow existing repository pattern from `alert_repository.go`
  - Use `COUNT(*) OVER()` for pagination
  - Create `internal/repository/decision_repository_test.go` with integration tests using `testutil.StartPostgres()`
  - Define repository row types: `DecisionCaseRow`, `LLMDecisionRow`, `ActionProposalRow`

  **Must NOT do**:
  - Use transactions (follow existing pattern of direct pool queries)
  - Add business logic (keep it data access only)
  - Skip integration tests

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Repository layer with multiple methods and integration tests
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 3, 4)
  - **Blocks**: Tasks 5, 6, 9, 10, 11
  - **Blocked By**: Task 1 (schema must exist)

  **References**:
  - `internal/repository/alert_repository.go` — Repository pattern to follow
  - `internal/repository/alert_repository_test.go` — Integration test pattern
  - `internal/repository/testutil/` — Postgres test container helper
  - `migrations/010_ai_tables_enhance.sql` — Table schema

  **Acceptance Criteria**:
  - [ ] `go test ./internal/repository/...` passes with integration tests
  - [ ] `CreateCase` inserts row into `ai.decision_case`
  - [ ] `GetCaseBySource` returns case by source_type + source_id
  - [ ] `ListCases` returns paginated results with total count
  - [ ] `CreateProposal` inserts row with `requires_human_review=true`

  **QA Scenarios**:

  ```
  Scenario: Create and retrieve case
    Tool: Bash (go test)
    Preconditions: postgres testcontainer running
    Steps:
      1. Run: go test ./internal/repository/... -run TestDecisionRepository_CreateAndGet -v
    Expected Result: PASS, case created and retrieved with matching fields
    Evidence: .sisyphus/evidence/task-2-repo-create-get.txt

  Scenario: Idempotency query by source
    Tool: Bash (go test)
    Preconditions: testcontainer running
    Steps:
      1. Run: go test ./internal/repository/... -run TestDecisionRepository_GetBySource -v
    Expected Result: PASS, case found by source_type + source_id
    Evidence: .sisyphus/evidence/task-2-repo-source-query.txt

  Scenario: Proposal creation with flags
    Tool: Bash (go test)
    Preconditions: testcontainer running
    Steps:
      1. Run: go test ./internal/repository/... -run TestDecisionRepository_CreateProposal -v
    Expected Result: PASS, proposal created with requires_human_review=true
    Evidence: .sisyphus/evidence/task-2-repo-proposal.txt
  ```

  **Commit**: YES (Commit 2)
  - Message: `feat(decision): add decision repository with integration tests`
  - Files: `internal/repository/decision_repository.go`, `internal/repository/decision_repository_test.go`
  - Pre-commit: `go test ./internal/repository/...`

- [x] 3. **Design Document**

  **What to do**:
  - Create `docs/migration/phase-6-decision-case-llm-context-plan.md`
  - Document:
    1. decision_case lifecycle (created → context_built → decision_generated → proposal_generated → review_required → closed → failed)
    2. alert → decision_case mapping
    3. decision context schema (trigger, object_context, metric_context, governance, allowed_actions, forbidden_actions)
    4. LLM-safe context schema
    5. governance/redaction usage strategy
    6. rule_based fallback strategy
    7. LLM feature flag strategy
    8. decision output validation rules
    9. action_proposal generation strategy
    10. human review boundaries
    11. API endpoint reference
    12. non-goals (Phase 6 scope exclusions)
    13. acceptance criteria
  - Include JSON examples for context, decision output, and proposal

  **Must NOT do**:
  - Document Phase 7 features (execution, review UI)
  - Include implementation code
  - Make promises about LLM performance

  **Recommended Agent Profile**:
  - **Category**: `writing`
    - Reason: Documentation task
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2, 4)
  - **Blocks**: None
  - **Blocked By**: None

  **References**:
  - `docs/migration/phase-5-governance-ontology-runtime-plan.md` — Document format to follow
  - User's Phase 6 spec — Source content

  **Acceptance Criteria**:
  - [ ] Document exists at `docs/migration/phase-6-decision-case-llm-context-plan.md`
  - [ ] All 13 required sections present
  - [ ] Contains JSON examples for context, decision, proposal
  - [ ] Non-goals clearly state no action execution, no LLM calls, no outbox

  **QA Scenarios**:

  ```
  Scenario: Document completeness
    Tool: Bash (grep)
    Preconditions: file exists
    Steps:
      1. grep -c "## " docs/migration/phase-6-decision-case-llm-context-plan.md
    Expected Result: Count >= 13 (all sections present)
    Evidence: .sisyphus/evidence/task-3-doc-sections.txt
  ```

  **Commit**: YES (Commit 9 — can be done early)
  - Message: `docs: add phase 6 decision case design document`
  - Files: `docs/migration/phase-6-decision-case-llm-context-plan.md`

- [x] 4. **ID Generator + Utility Functions** (`internal/decision/idgen.go`)

  **What to do**:
  - Create `internal/decision/idgen.go`
  - Implement `GenerateCaseID() string` — format: `dc_<timestamp>_<6char_hash>`
  - Implement `GenerateProposalID() string` — format: `ap_<timestamp>_<6char_hash>`
  - Implement `GenerateDecisionID() string` — format: `de_<timestamp>_<6char_hash>`
  - Use `time.Now().Unix()` for timestamp
  - Use `crypto/rand` + base64 for hash portion
  - Add unit tests in `internal/decision/idgen_test.go`

  **Must NOT do**:
  - Use UUID library (follow Phase 2 TEXT convention)
  - Make IDs predictable (use random component)
  - Add external dependencies

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple utility functions
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2, 3)
  - **Blocks**: Task 5
  - **Blocked By**: None

  **References**:
  - Existing ID patterns in codebase (e.g., alert_id, task_id formats)

  **Acceptance Criteria**:
  - [ ] `go test ./internal/decision/... -run TestIDGen` passes
  - [ ] Generated IDs match pattern: `dc_\d+_[A-Za-z0-9]{6}`
  - [ ] IDs are unique across 1000 generations

  **QA Scenarios**:

  ```
  Scenario: ID format validation
    Tool: Bash (go test)
    Preconditions: code exists
    Steps:
      1. Run: go test ./internal/decision/... -run TestGenerateCaseID -v
    Expected Result: PASS, IDs match regex ^dc_\d+_[A-Za-z0-9]{6}$
    Evidence: .sisyphus/evidence/task-4-id-format.txt
  ```

  **Commit**: Bundled with Task 3 (Commit 3)
  - Message: `feat(decision): add case service, context builder, id generator`
  - Files: `internal/decision/idgen.go`, `internal/decision/idgen_test.go`

- [x] 5. **Case Service** (`internal/decision/case_service.go`)

  **What to do**:
  - Create `internal/decision/case_service.go`
  - Define `CaseService` struct with repository dependency
  - Implement `CreateCaseFromAlert(ctx, alertID, createdBy string) (*DecisionCase, error)`:
    1. Query `ops.metric_alert` by alert_id to get rule_id, severity, object_type, object_id, metric_name, current_value, baseline_value, change_rate
    2. Check idempotency: query `ai.decision_case` for source_type='alert', source_id=alert_id with status NOT IN ('closed', 'failed')
    3. If active case exists, return existing case (200-equivalent behavior)
    4. If not, generate case_id using idgen, create case with status='created'
    5. Return created case
  - Implement `GetCase(ctx, caseID string) (*DecisionCase, error)`
  - Implement `ListCases(ctx, filter CaseFilter) (*CaseList, error)`
  - Implement `UpdateCaseStatus(ctx, caseID, status string) error`
  - Define `DecisionCase` domain struct matching enhanced schema
  - Add unit tests with mock repository: `internal/decision/case_service_test.go`

  **Must NOT do**:
  - Skip idempotency check
  - Allow duplicate active cases for same alert
  - Add context building logic (that goes in Task 6)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Core domain logic with idempotency and alert integration
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Wave 1)
  - **Parallel Group**: Wave 2 (with Tasks 6, 7, 8)
  - **Blocks**: Tasks 6, 9, 10, 11, 12
  - **Blocked By**: Tasks 1, 2, 4

  **References**:
  - `internal/decision/idgen.go` — ID generation
  - `internal/repository/decision_repository.go` — Data access
  - `internal/service/qoder_service.go` — Alert query pattern (lines 139-216)
  - `migrations/005_ops_tables.sql` — metric_alert schema

  **Acceptance Criteria**:
  - [ ] `go test ./internal/decision/... -run TestCaseService` passes
  - [ ] CreateCaseFromAlert creates case when no active case exists
  - [ ] CreateCaseFromAlert returns existing case on duplicate
  - [ ] Case status starts as 'created'
  - [ ] Case has correct source_type, source_id, object_type, object_id from alert

  **QA Scenarios**:

  ```
  Scenario: Create case from alert
    Tool: Bash (go test)
    Preconditions: mock repository setup
    Steps:
      1. Run: go test ./internal/decision/... -run TestCaseService_CreateFromAlert -v
    Expected Result: PASS, case created with status=created
    Evidence: .sisyphus/evidence/task-5-create-case.txt

  Scenario: Idempotent creation
    Tool: Bash (go test)
    Preconditions: mock repository with existing case
    Steps:
      1. Run: go test ./internal/decision/... -run TestCaseService_Idempotent -v
    Expected Result: PASS, returns existing case without creating new one
    Evidence: .sisyphus/evidence/task-5-idempotent.txt
  ```

  **Commit**: Bundled with Tasks 4, 6 (Commit 3)

- [x] 6. **Decision Context Builder** (`internal/decision/context_builder.go`)

  **What to do**:
  - Create `internal/decision/context_builder.go`
  - Define `ContextBuilder` struct with dependencies:
    - `ObjectQueryService` (from `internal/ontology/`)
    - `GovernanceService` (from `internal/governance/`)
    - `RedactionEngine` (from `internal/governance/`)
  - Implement `BuildContext(ctx, caseID string) (*DecisionContext, error)`:
    1. Fetch case from repository to get object_type, object_id, source_id
    2. Fetch alert data from `ops.metric_alert` for trigger info
    3. Call `ObjectQueryService.BuildObjectContext(ctx, objectType, objectID)` for object data
    4. Call `GovernanceService.GetClassification(ctx, objectType, objectID)` for L1-L4 classification
    5. Call `GovernanceService.GetLineage(ctx, objectType, objectID)` for lineage
    6. Call `RedactionEngine.Redact(ctx, objectContext, "agent_readonly")` for redacted fields
    7. Build `DecisionContext` with trigger, object_context, metric_context, governance, allowed_actions, forbidden_actions
    8. Compute SHA256 hash of final redacted context JSON
    9. Return context
  - Implement `BuildLLMSafeContext(ctx, decisionContext *DecisionContext) (*LLMSafeContext, error)`
  - Define `DecisionContext` and `LLMSafeContext` structs
  - Add tests: `internal/decision/context_builder_test.go`

  **Must NOT do**:
  - Allow direct SQL to be passed to context
  - Include raw/dwd detail data in context
  - Skip redaction step
  - Skip governance snapshot

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Complex orchestration of multiple Phase 5 services
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Wave 1)
  - **Parallel Group**: Wave 2 (with Tasks 5, 7, 8)
  - **Blocks**: Tasks 9, 10
  - **Blocked By**: Tasks 2, 5

  **References**:
  - `internal/ontology/query_service.go` — ObjectQueryService.BuildObjectContext
  - `internal/ontology/context_builder.go` — LLM-safe context pattern (but fix data fetch gap)
  - `internal/governance/redaction.go` — RedactionEngine
  - `internal/governance/classification.go` — GovernanceService.GetClassification
  - `internal/governance/lineage.go` — GovernanceService.GetLineage
  - `internal/service/governance_service.go` — Service composition pattern

  **Acceptance Criteria**:
  - [ ] `go test ./internal/decision/... -run TestContextBuilder` passes
  - [ ] Context includes trigger with alert data
  - [ ] Context includes object_context from ObjectQueryService
  - [ ] Context includes governance.classification
  - [ ] Context includes redacted_fields list
  - [ ] Context hash is SHA256 of final JSON
  - [ ] LLM-safe context excludes L3/L4 sensitive fields

  **QA Scenarios**:

  ```
  Scenario: Build complete context
    Tool: Bash (go test)
    Preconditions: mock ontology/governance services
    Steps:
      1. Run: go test ./internal/decision/... -run TestContextBuilder_Build -v
    Expected Result: PASS, context has trigger, object_context, governance, allowed_actions, hash
    Evidence: .sisyphus/evidence/task-6-build-context.txt

  Scenario: Redaction applied
    Tool: Bash (go test)
    Preconditions: mock with L3/L4 fields
    Steps:
      1. Run: go test ./internal/decision/... -run TestContextBuilder_Redaction -v
    Expected Result: PASS, redacted_fields contains L3/L4 field names
    Evidence: .sisyphus/evidence/task-6-redaction.txt
  ```

  **Commit**: Bundled with Tasks 4, 5 (Commit 3)

- [x] 7. **LLM Provider Interface + Disabled Provider** (`internal/llm/provider.go`)

  **What to do**:
  - Create `internal/llm/provider.go`
  - Define `DecisionProvider` interface:
    ```go
    type DecisionProvider interface {
        GenerateDecision(ctx context.Context, input LLMSafeContext) (*DecisionOutput, error)
    }
    ```
  - Create `internal/llm/disabled_provider.go`
  - Implement `DisabledProvider` that always returns error: "LLM is disabled"
  - Define `LLMSafeContext` struct (input to provider)
  - Define `DecisionOutput` struct (output from provider):
    - decision_type (monitor_only, investigate, optimize, intervention, experiment)
    - severity (low, medium, high, critical)
    - summary (string)
    - rationale ([]string)
    - recommended_actions ([]RecommendedAction)
    - confidence (float64, [0,1])
    - requires_human_review (bool, always true in Phase 6)
  - Define `RecommendedAction` struct:
    - action_type (create_followup_task, notify_owner, export_report, escalate_to_human)
    - priority (low, medium, high)
    - owner_role (string)
    - payload (map[string]interface{})
  - Add tests: `internal/llm/provider_test.go`

  **Must NOT do**:
  - Implement real HTTP LLM calls
  - Skip interface abstraction (must be swappable for future LLM providers)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Interface definitions and simple stub
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Wave 1)
  - **Parallel Group**: Wave 2 (with Tasks 5, 6, 8)
  - **Blocks**: Tasks 9, 10
  - **Blocked By**: None

  **References**:
  - User spec — Decision output schema
  - `internal/ontology/context_builder.go` — Context struct patterns

  **Acceptance Criteria**:
  - [ ] `go test ./internal/llm/...` passes
  - [ ] DecisionProvider interface defined
  - [ ] DisabledProvider returns error on GenerateDecision
  - [ ] DecisionOutput has all required fields

  **QA Scenarios**:

  ```
  Scenario: Disabled provider rejects
    Tool: Bash (go test)
    Preconditions: code exists
    Steps:
      1. Run: go test ./internal/llm/... -run TestDisabledProvider -v
    Expected Result: PASS, error contains "disabled"
    Evidence: .sisyphus/evidence/task-7-disabled.txt
  ```

  **Commit**: Bundled with Task 9 (Commit 4)

- [x] 8. **Decision Schema Validator** (`internal/llm/schema_validator.go`)

  **What to do**:
  - Create `internal/llm/schema_validator.go`
  - Implement `ValidateDecision(output *DecisionOutput, allowedActions []string) error`
  - Validation rules:
    1. decision_type ∈ {monitor_only, investigate, optimize, intervention, experiment}
    2. severity ∈ {low, medium, high, critical}
    3. confidence ∈ [0.0, 1.0]
    4. recommended_actions is subset of allowed_actions
    5. requires_human_review == true (Phase 6 guardrail)
    6. Each action has valid action_type ∈ enum
    7. payload is valid JSON (map[string]interface{})
  - Return detailed validation errors list
  - Add tests: `internal/llm/schema_validator_test.go`

  **Must NOT do**:
  - Allow requires_human_review=false
  - Allow actions outside allowed_actions
  - Skip confidence bounds check

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Validation logic with clear rules
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Wave 1)
  - **Parallel Group**: Wave 2 (with Tasks 5, 6, 7)
  - **Blocks**: Tasks 9, 10
  - **Blocked By**: Task 7 (DecisionOutput struct)

  **References**:
  - User spec — Validation rules section

  **Acceptance Criteria**:
  - [ ] `go test ./internal/llm/... -run TestValidator` passes
  - [ ] Invalid decision_type returns error
  - [ ] Confidence > 1.0 returns error
  - [ ] Action not in allowed_actions returns error
  - [ ] requires_human_review=false returns error

  **QA Scenarios**:

  ```
  Scenario: Valid decision passes
    Tool: Bash (go test)
    Preconditions: code exists
    Steps:
      1. Run: go test ./internal/llm/... -run TestValidator_Valid -v
    Expected Result: PASS, no error
    Evidence: .sisyphus/evidence/task-8-valid.txt

  Scenario: Invalid action rejected
    Tool: Bash (go test)
    Preconditions: code exists
    Steps:
      1. Run: go test ./internal/llm/... -run TestValidator_InvalidAction -v
    Expected Result: PASS, error mentions disallowed action
    Evidence: .sisyphus/evidence/task-8-invalid-action.txt
  ```

  **Commit**: Bundled with Task 9 (Commit 4)

- [x] 9. **Rule-Based Decision Provider** (`internal/llm/rule_provider.go`)

  **What to do**:
  - Create `internal/llm/rule_provider.go`
  - Implement `RuleBasedProvider` implementing `DecisionProvider`
  - Logic (severity → actions mapping):
    ```
    severity == "critical" || severity == "high":
      decision_type = "escalate_to_human"
      recommended_actions = [
        {action_type: "escalate_to_human", priority: "high", owner_role: "ops"},
        {action_type: "notify_owner", priority: "high", owner_role: "ops"}
      ]
    
    severity == "medium":
      decision_type = "investigate"
      recommended_actions = [
        {action_type: "notify_owner", priority: "medium", owner_role: "analyst"},
        {action_type: "create_followup_task", priority: "medium", owner_role: "analyst"}
      ]
    
    severity == "low":
      decision_type = "monitor_only"
      recommended_actions = [
        {action_type: "create_followup_task", priority: "low", owner_role: "analyst"}
      ]
    
    default:
      decision_type = "investigate"
      recommended_actions = [
        {action_type: "notify_owner", priority: "medium", owner_role: "analyst"}
      ]
    ```
  - Set confidence based on severity: critical=0.95, high=0.85, medium=0.72, low=0.60
  - Set requires_human_review=true always
  - Add summary: "{Severity} severity alert for {metric_name}. {rationale}."
  - Add rationale explaining the rule mapping
  - Add tests: `internal/llm/rule_provider_test.go`
  - Table-driven tests for all severity levels

  **Must NOT do**:
  - Make confidence > 1.0 or < 0.0
  - Set requires_human_review=false
  - Skip rationale generation

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Core business logic with rule mapping
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Wave 2)
  - **Parallel Group**: Wave 3 (with Tasks 10, 11)
  - **Blocks**: Task 10
  - **Blocked By**: Tasks 7, 8

  **References**:
  - User spec — Rule-based fallback section
  - `internal/llm/provider.go` — DecisionProvider interface
  - `internal/llm/schema_validator.go` — Validation rules

  **Acceptance Criteria**:
  - [ ] `go test ./internal/llm/... -run TestRuleProvider` passes
  - [ ] Critical severity produces escalate_to_human action
  - [ ] High severity produces escalate_to_human action
  - [ ] Medium severity produces investigate + notify + task
  - [ ] Low severity produces monitor_only + task
  - [ ] All outputs have requires_human_review=true
  - [ ] Confidence values within [0,1]

  **QA Scenarios**:

  ```
  Scenario: Critical alert decision
    Tool: Bash (go test)
    Preconditions: code exists
    Steps:
      1. Run: go test ./internal/llm/... -run TestRuleProvider_Critical -v
    Expected Result: PASS, decision_type=escalate_to_human, actions include escalate_to_human + notify_owner
    Evidence: .sisyphus/evidence/task-9-critical.txt

  Scenario: Low alert decision
    Tool: Bash (go test)
    Preconditions: code exists
    Steps:
      1. Run: go test ./internal/llm/... -run TestRuleProvider_Low -v
    Expected Result: PASS, decision_type=monitor_only, actions include create_followup_task
    Evidence: .sisyphus/evidence/task-9-low.txt
  ```

  **Commit**: YES (Commit 4)
  - Message: `feat(llm): add decision provider interface and rule-based fallback`
  - Files: `internal/llm/*.go`, `internal/llm/*_test.go`
  - Pre-commit: `go test ./internal/llm/...`

- [x] 10. **Decision Engine** (`internal/decision/engine.go`)

  **What to do**:
  - Create `internal/decision/engine.go`
  - Define `DecisionEngine` struct with:
    - `provider llm.DecisionProvider`
    - `validator *llm.SchemaValidator`
  - Implement `GenerateDecision(ctx, caseID string, context *DecisionContext) (*llm.DecisionOutput, error)`:
    1. Build LLM-safe context from DecisionContext
    2. Call `provider.GenerateDecision(ctx, llmSafeContext)`
    3. Validate output against allowed_actions from context
    4. If validation fails:
       - Log fallback_reason
       - Call `RuleBasedProvider` as fallback
       - Mark decision status as "fallback"
       - Record validation_errors
    5. Save decision to repository with status
    6. Update case status to "decision_generated"
    7. Return decision
  - Add tests: `internal/decision/engine_test.go`
  - Mock provider and validator for unit tests

  **Must NOT do**:
  - Skip validation
  - Skip fallback on validation failure
  - Allow invalid decisions to proceed

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Complex orchestration with fallback logic
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Wave 2)
  - **Parallel Group**: Wave 3 (with Tasks 9, 11)
  - **Blocks**: Tasks 11, 12
  - **Blocked By**: Tasks 5, 6, 7, 8, 9

  **References**:
  - `internal/llm/provider.go` — DecisionProvider interface
  - `internal/llm/schema_validator.go` — Validation logic
  - `internal/llm/rule_provider.go` — Fallback provider
  - `internal/decision/context_builder.go` — Context building

  **Acceptance Criteria**:
  - [ ] `go test ./internal/decision/... -run TestEngine` passes
  - [ ] Valid decision passes through without fallback
  - [ ] Invalid decision triggers fallback to rule-based
  - [ ] Case status updated to decision_generated
  - [ ] Decision saved to ai.llm_decision with correct status

  **QA Scenarios**:

  ```
  Scenario: Valid decision path
    Tool: Bash (go test)
    Preconditions: mock provider returns valid decision
    Steps:
      1. Run: go test ./internal/decision/... -run TestEngine_Valid -v
    Expected Result: PASS, no fallback, case status=decision_generated
    Evidence: .sisyphus/evidence/task-10-valid.txt

  Scenario: Fallback on invalid decision
    Tool: Bash (go test)
    Preconditions: mock provider returns invalid decision
    Steps:
      1. Run: go test ./internal/decision/... -run TestEngine_Fallback -v
    Expected Result: PASS, fallback triggered, decision status=fallback
    Evidence: .sisyphus/evidence/task-10-fallback.txt
  ```

  **Commit**: Bundled with Task 11 (Commit 5)

- [x] 11. **Action Proposal Service** (`internal/action/proposal_service.go`)

  **What to do**:
  - Create `internal/action/proposal_service.go`
  - Define `ProposalService` struct with repository dependency
  - Implement `GenerateProposalsFromDecision(ctx, caseID string, decision *llm.DecisionOutput) ([]ActionProposal, error)`:
    1. For each recommended_action in decision:
       - Generate proposal_id using idgen
       - Set title: "{action_type}: {decision.summary}"
       - Set description from decision.rationale
       - Set action_type, priority, payload
       - Set risk_level based on decision.severity (critical/high=high, medium=medium, low=low)
       - Set requires_human_review=true (Phase 6 guardrail)
       - Set apply_status='proposed'
       - Save to ai.action_proposal
    2. Update case status to "proposal_generated" or "review_required"
    3. Return list of proposals
  - Implement `ListProposalsByCase(ctx, caseID string) ([]ActionProposal, error)`
  - Define `ActionProposal` domain struct
  - Add tests: `internal/action/proposal_service_test.go`

  **Must NOT do**:
  - Set requires_human_review=false
  - Set apply_status to 'applied' or 'executed'
  - Generate outbox events
  - Execute actions

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Business logic connecting decisions to proposals
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Wave 2)
  - **Parallel Group**: Wave 3 (with Tasks 9, 10)
  - **Blocks**: Task 12
  - **Blocked By**: Tasks 2, 10

  **References**:
  - `internal/decision/idgen.go` — ID generation
  - `internal/repository/decision_repository.go` — Proposal persistence
  - User spec — Action Proposal section

  **Acceptance Criteria**:
  - [ ] `go test ./internal/action/...` passes
  - [ ] Proposals generated for each recommended action
  - [ ] All proposals have requires_human_review=true
  - [ ] All proposals have apply_status=proposed
  - [ ] Case status updated to proposal_generated
  - [ ] No outbox events created

  **QA Scenarios**:

  ```
  Scenario: Generate proposals from decision
    Tool: Bash (go test)
    Preconditions: mock repository
    Steps:
      1. Run: go test ./internal/action/... -run TestProposalService_Generate -v
    Expected Result: PASS, proposals created with correct flags
    Evidence: .sisyphus/evidence/task-11-generate.txt

  Scenario: Proposal flags validation
    Tool: Bash (go test)
    Preconditions: generated proposals
    Steps:
      1. Run: go test ./internal/action/... -run TestProposalService_Flags -v
    Expected Result: PASS, all proposals have requires_human_review=true, apply_status=proposed
    Evidence: .sisyphus/evidence/task-11-flags.txt
  ```

  **Commit**: YES (Commit 5)
  - Message: `feat(decision): add decision engine and action proposal service`
  - Files: `internal/decision/engine.go`, `internal/action/*.go`, `*_test.go`
  - Pre-commit: `go test ./internal/decision/... ./internal/action/...`

- [x] 12. **Decision Business Service** (`internal/service/decision_service.go`)

  **What to do**:
  - Create `internal/service/decision_service.go`
  - Define `DecisionService` struct composing:
    - `*decision.CaseService`
    - `*decision.ContextBuilder`
    - `*decision.DecisionEngine`
    - `*action.ProposalService`
    - `*repository.DecisionRepository`
    - `*pgxpool.Pool`
  - Implement orchestration methods:
    - `CreateCaseFromAlert(ctx, alertID, createdBy string) (*dto.DecisionCaseResponse, error)`
    - `BuildContext(ctx, caseID string) (*dto.DecisionContextResponse, error)`
    - `Decide(ctx, caseID string) (*dto.DecisionResponse, error)`
    - `ListCases(ctx, filter dto.CaseFilter) (*dto.CaseListResponse, error)`
    - `GetCase(ctx, caseID string) (*dto.DecisionCaseResponse, error)`
    - `ListProposals(ctx, caseID string) (*dto.ProposalListResponse, error)`
  - Map domain structs to DTOs
  - Add tests: `internal/service/decision_service_test.go`

  **Must NOT do**:
  - Add transaction logic (follow existing pool-direct pattern)
  - Skip DTO mapping
  - Mix HTTP concerns

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Business orchestration layer
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Wave 3)
  - **Parallel Group**: Wave 4 (with Tasks 13, 14, 15, 16)
  - **Blocks**: Task 13
  - **Blocked By**: Tasks 5, 6, 10, 11

  **References**:
  - `internal/service/governance_service.go` — Service composition pattern
  - `internal/decision/case_service.go` — Case lifecycle
  - `internal/decision/engine.go` — Decision generation
  - `internal/action/proposal_service.go` — Proposal generation

  **Acceptance Criteria**:
  - [ ] `go test ./internal/service/... -run TestDecisionService` passes
  - [ ] Orchestration methods call correct domain services
  - [ ] DTOs map correctly from domain structs

  **QA Scenarios**:

  ```
  Scenario: Full orchestration flow
    Tool: Bash (go test)
    Preconditions: mocks for all dependencies
    Steps:
      1. Run: go test ./internal/service/... -run TestDecisionService_Flow -v
    Expected Result: PASS, case → context → decision → proposals flow works
    Evidence: .sisyphus/evidence/task-12-flow.txt
  ```

  **Commit**: Bundled with Tasks 13, 14 (Commit 6)

- [x] 13. **Decision API Handlers** (`internal/api/handler/decision.go` + `internal/api/dto/decision.go`)

  **What to do**:
  - Create `internal/api/dto/decision.go` with request/response DTOs:
    - `CreateCaseRequest`, `CreateCaseResponse`
    - `DecisionCaseResponse`, `CaseListResponse`
    - `DecisionContextResponse`
    - `DecisionResponse`
    - `ProposalResponse`, `ProposalListResponse`
    - `CaseFilter` (query params)
  - Create `internal/api/handler/decision.go`
  - Define narrow handler interfaces:
    ```go
    type CaseCreator interface { CreateCaseFromAlert(...) }
    type CaseLister interface { ListCases(...) }
    type ContextBuilder interface { BuildContext(...) }
    type DecisionGenerator interface { Decide(...) }
    type ProposalLister interface { ListProposals(...) }
    ```
  - Implement HTTP handlers:
    - `POST /api/v1/decisions/cases` — Create case from alert
    - `GET /api/v1/decisions/cases` — List cases with filtering
    - `GET /api/v1/decisions/cases/{case_id}` — Get case by ID
    - `POST /api/v1/decisions/cases/{case_id}/context` — Build context
    - `POST /api/v1/decisions/cases/{case_id}/decide` — Generate decision
    - `GET /api/v1/decisions/cases/{case_id}/proposals` — List proposals
  - All endpoints require Bearer Token auth
  - Use `httputil.JSON(w, status, body)` for responses
  - Add tests: `internal/api/handler/decision_test.go`
  - Mock service interfaces for handler tests

  **Must NOT do**:
  - Implement Phase 7 endpoints (approve, apply, execute)
  - Skip auth middleware
  - Add non-JSON responses

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: HTTP handlers with multiple endpoints and tests
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Wave 3)
  - **Parallel Group**: Wave 4 (with Tasks 12, 14, 15, 16)
  - **Blocks**: Task 14
  - **Blocked By**: Task 12

  **References**:
  - `internal/api/handler/alerts.go` — Handler pattern
  - `internal/api/handler/alerts_test.go` — Handler test pattern
  - `internal/api/dto/alert.go` — DTO pattern
  - `internal/api/httputil/` — JSON response helper

  **Acceptance Criteria**:
  - [ ] `go test ./internal/api/handler/... -run TestDecision` passes
  - [ ] All 6 endpoints have handler tests
  - [ ] POST /cases returns 201 with case data
  - [ ] GET /cases/{id} returns 200 for existing case
  - [ ] POST /context returns 200 with context
  - [ ] POST /decide returns 200 with decision + proposals
  - [ ] GET /proposals returns 200 with proposal list
  - [ ] Missing auth returns 401

  **QA Scenarios**:

  ```
  Scenario: Create case endpoint
    Tool: Bash (go test)
    Preconditions: mock service
    Steps:
      1. Run: go test ./internal/api/handler/... -run TestDecisionHandler_CreateCase -v
    Expected Result: PASS, status 201, response contains case_id
    Evidence: .sisyphus/evidence/task-13-create.txt

  Scenario: Decide endpoint
    Tool: Bash (go test)
    Preconditions: mock service with full flow
    Steps:
      1. Run: go test ./internal/api/handler/... -run TestDecisionHandler_Decide -v
    Expected Result: PASS, status 200, response contains decision + proposals
    Evidence: .sisyphus/evidence/task-13-decide.txt

  Scenario: Auth required
    Tool: Bash (go test)
    Preconditions: handler setup
    Steps:
      1. Run: go test ./internal/api/handler/... -run TestDecisionHandler_Auth -v
    Expected Result: PASS, missing auth returns 401
    Evidence: .sisyphus/evidence/task-13-auth.txt
  ```

  **Commit**: Bundled with Tasks 12, 14 (Commit 6)

- [x] 14. **Server Wiring** (`internal/api/server.go`)

  **What to do**:
  - Update `internal/api/server.go`
  - Add lazy-initialized decision handler:
    ```go
    func (s *Server) decisionHandler() *handler.DecisionHandler {
        if s.decisionHandler == nil {
            svc := service.NewDecisionService(...)
            s.decisionHandler = handler.NewDecisionHandler(svc)
        }
        return s.decisionHandler
    }
    ```
  - Register routes in `setupRoutes()`:
    ```go
    r.Route("/api/v1/decisions", func(r chi.Router) {
        r.Use(authMiddleware)
        r.Post("/cases", s.decisionHandler().CreateCase)
        r.Get("/cases", s.decisionHandler().ListCases)
        r.Get("/cases/{case_id}", s.decisionHandler().GetCase)
        r.Post("/cases/{case_id}/context", s.decisionHandler().BuildContext)
        r.Post("/cases/{case_id}/decide", s.decisionHandler().Decide)
        r.Get("/cases/{case_id}/proposals", s.decisionHandler().ListProposals)
    })
    ```
  - Wire dependencies: pool, repository, ontology service, governance service
  - Ensure auth middleware applied

  **Must NOT do**:
  - Register Phase 7 routes (approve, apply)
  - Skip auth middleware
  - Create new router instance

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Route registration following existing pattern
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Wave 3)
  - **Parallel Group**: Wave 4 (with Tasks 12, 13, 15, 16)
  - **Blocks**: None
  - **Blocked By**: Task 13

  **References**:
  - `internal/api/server.go` — Existing route registration pattern
  - `internal/api/handler/decision.go` — Handler to wire

  **Acceptance Criteria**:
  - [ ] `go build ./cmd/baxi-api` succeeds
  - [ ] Server starts without errors
  - [ ] Decision routes accessible (test with curl)

  **QA Scenarios**:

  ```
  Scenario: Server builds and starts
    Tool: Bash (go build + curl)
    Preconditions: code compiles
    Steps:
      1. Run: go build ./cmd/baxi-api
      2. Start server: ./baxi-api &
      3. Curl: curl -H "Authorization: Bearer dev_token_32_chars_minimum_length_here" http://localhost:8080/api/v1/decisions/cases
    Expected Result: Build succeeds, server responds 200 (empty list) or 401 if no DB
    Evidence: .sisyphus/evidence/task-14-server.txt
  ```

  **Commit**: Bundled with Tasks 12, 13 (Commit 6)
  - Message: `feat(api): add decision DTOs, handlers, and server wiring`
  - Files: `internal/api/dto/decision.go`, `internal/api/handler/decision.go`, `internal/api/handler/decision_test.go`, `internal/api/server.go`, `internal/service/decision_service.go`, `internal/service/decision_service_test.go`
  - Pre-commit: `go test ./internal/api/... ./internal/service/...`

- [x] 15. **CLI Commands** (`cmd/baxi-cli/decision.go`)

  **What to do**:
  - Create `cmd/baxi-cli/decision.go`
  - Implement CLI commands:
    - `decision create --alert-id=ALERT_ID` — Create case from alert
    - `decision context --case-id=CASE_ID` — Build context
    - `decision decide --case-id=CASE_ID` — Generate decision
    - `decision list` — List cases
  - Use existing flag pattern (plain `flag` package, no Cobra)
  - Follow `cmd/baxi-cli/governance.go` structure
  - Output pretty-printed JSON to stdout
  - Add to main.go switch dispatch

  **Must NOT do**:
  - Add approve/apply commands (Phase 7)
  - Use external CLI framework
  - Skip error handling

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: CLI with multiple subcommands and DB interaction
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Wave 3)
  - **Parallel Group**: Wave 4 (with Tasks 12, 13, 14, 16)
  - **Blocks**: None
  - **Blocked By**: Task 12

  **References**:
  - `cmd/baxi-cli/governance.go` — CLI command pattern
  - `cmd/baxi-cli/main.go` — Switch dispatch pattern

  **Acceptance Criteria**:
  - [ ] `go build ./cmd/baxi-cli` succeeds
  - [ ] `baxi-cli decision create --alert-id=test` compiles
  - [ ] `baxi-cli decision list` compiles
  - [ ] Commands have --help output

  **QA Scenarios**:

  ```
  Scenario: CLI build
    Tool: Bash (go build)
    Preconditions: code exists
    Steps:
      1. Run: go build -o baxi-cli ./cmd/baxi-cli
      2. Run: ./baxi-cli decision --help
    Expected Result: Build succeeds, help text shows available commands
    Evidence: .sisyphus/evidence/task-15-cli-build.txt
  ```

  **Commit**: YES (Commit 7)
  - Message: `feat(cli): add decision commands to baxi-cli`
  - Files: `cmd/baxi-cli/decision.go`, `cmd/baxi-cli/main.go`
  - Pre-commit: `go build ./cmd/baxi-cli`

- [x] 16. **Config Updates** (`.env.example`, `Makefile`)

  **What to do**:
  - Update `.env.example`:
    ```
    # LLM Configuration (Phase 6)
    LLM_ENABLED=false
    LLM_PROVIDER=disabled
    LLM_MODEL=
    LLM_TIMEOUT_SECONDS=30
    DECISION_DEFAULT_ROLE=agent_readonly
    DECISION_REQUIRE_HUMAN_REVIEW=true
    ```
  - Update `Makefile`:
    ```makefile
    decision-create:
        go run ./cmd/baxi-cli decision create --alert-id $(ALERT_ID)
    
    decision-context:
        go run ./cmd/baxi-cli decision context --case-id $(CASE_ID)
    
    decision-decide:
        go run ./cmd/baxi-cli decision decide --case-id $(CASE_ID)
    
    decision-list:
        go run ./cmd/baxi-cli decision list
    ```

  **Must NOT do**:
  - Add LLM_API_KEY or real provider config
  - Enable LLM by default
  - Add Phase 7 commands

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Config additions
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4 (with Tasks 12, 13, 14, 15)
  - **Blocks**: None
  - **Blocked By**: None

  **References**:
  - `.env.example` — Existing format
  - `Makefile` — Existing target pattern

  **Acceptance Criteria**:
  - [ ] `.env.example` contains all 6 new LLM/decision variables
  - [ ] `Makefile` has 4 new decision targets
  - [ ] `make decision-list` works (builds and runs)

  **QA Scenarios**:

  ```
  Scenario: Makefile targets
    Tool: Bash (make)
    Preconditions: Makefile updated
    Steps:
      1. Run: make decision-list 2>&1 | head -5
    Expected Result: Command runs without "No rule" error
    Evidence: .sisyphus/evidence/task-16-makefile.txt
  ```

  **Commit**: YES (Commit 8)
  - Message: `chore(config): add LLM config to .env.example and decision Makefile targets`
  - Files: `.env.example`, `Makefile`

---

## Final Verification Wave

> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.

- [x] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, curl endpoint, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in .sisyphus/evidence/. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [x] F2. **Code Quality Review** — `unspecified-high`
  Run `tsc --noEmit` + linter + `go test ./...`. Review all changed files for: `as any`/`@ts-ignore`, empty catches, `console.log` in prod, commented-out code, unused imports. Check AI slop: excessive comments, over-abstraction, generic names (data/result/item/temp).
  Output: `Build [PASS/FAIL] | Lint [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [x] F3. **Integration QA** — `unspecified-high`
  Start from clean state. Execute EVERY QA scenario from EVERY task — follow exact steps, capture evidence. Test cross-task integration: create case → build context → decide → list proposals. Verify no outbox events created. Save to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [x] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff (git log/diff). Verify 1:1 — everything in spec was built (no missing), nothing beyond spec was built (no creep). Check "Must NOT do" compliance. Detect cross-task contamination. Flag unaccounted changes.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

| Commit | Message | Files | Pre-commit |
|--------|---------|-------|------------|
| 1 | `feat(migration): add 008_ai_tables_enhance.sql for decision case schema` | `migrations/010_ai_tables_enhance.sql` | `make migrate` |
| 2 | `feat(decision): add decision repository with integration tests` | `internal/repository/decision_repository.go`, `*_test.go` | `go test ./internal/repository/...` |
| 3 | `feat(decision): add case service, context builder, id generator` | `internal/decision/*.go`, `*_test.go` | `go test ./internal/decision/...` |
| 4 | `feat(llm): add decision provider interface and rule-based fallback` | `internal/llm/*.go`, `*_test.go` | `go test ./internal/llm/...` |
| 5 | `feat(decision): add decision engine and action proposal service` | `internal/decision/engine.go`, `internal/action/*.go` | `go test ./internal/decision/... ./internal/action/...` |
| 6 | `feat(api): add decision DTOs, handlers, and server wiring` | `internal/api/dto/decision.go`, `internal/api/handler/decision.go`, `internal/api/server.go`, `*_test.go` | `go test ./internal/api/...` |
| 7 | `feat(cli): add decision commands to baxi-cli` | `cmd/baxi-cli/decision.go` | `go build ./cmd/baxi-cli` |
| 8 | `chore(config): add LLM config to .env.example and decision Makefile targets` | `.env.example`, `Makefile` | — |
| 9 | `docs: add phase 6 decision case design document` | `docs/migration/phase-6-decision-case-llm-context-plan.md` | — |

---

## Success Criteria

### Verification Commands
```bash
# Schema
make migrate

# Case creation
curl -X POST http://localhost:8080/api/v1/decisions/cases \
  -H "Authorization: Bearer dev_token_32_chars_minimum_length_here" \
  -H "Content-Type: application/json" \
  -d '{"source_type":"alert","source_id":"alert_xxx"}'

# Context build
curl -X POST http://localhost:8080/api/v1/decisions/cases/{case_id}/context \
  -H "Authorization: Bearer dev_token_32_chars_minimum_length_here"

# Decision generation
curl -X POST http://localhost:8080/api/v1/decisions/cases/{case_id}/decide \
  -H "Authorization: Bearer dev_token_32_chars_minimum_length_here"

# List proposals
curl http://localhost:8080/api/v1/decisions/cases/{case_id}/proposals \
  -H "Authorization: Bearer dev_token_32_chars_minimum_length_here"

# Regression
make pipeline-compare
make api-compare
make governance-check
go test ./...
```

### Final Checklist
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent
- [ ] All tests pass
- [ ] No real LLM calls
- [ ] No action execution
- [ ] No outbox events generated
