# Phase 7: Review / Approval / Action Execution / Outbox Dispatch

## TL;DR

> **Quick Summary**: Build the review/approval pipeline for AI-generated action proposals, implement safe action execution with dry-run support, and create an outbox dispatch worker for reliable external notifications — completing the end-to-end decision-to-action loop.
>
> **Deliverables**:
> - Migration 011 extending schema for review/apply state machine
> - `internal/review/` package (repository, service, domain model, API handlers)
> - `internal/action/apply_service.go` + `executor.go` + `registry.go` (action execution with dry-run)
> - `internal/outbox/dispatch_worker.go` + adapter stubs (Feishu/GitHub)
> - HTTP endpoints for review, execute, outbox dispatch, audit queries
> - Integration tests with testcontainers PostgreSQL
> - Security tests ensuring no unapproved execution
>
> **Estimated Effort**: Large (6 sub-phases, ~20 tasks)
> **Parallel Execution**: YES — 5 waves with 5-8 tasks each
> **Critical Path**: Migration 011 → Review Repository → Apply Service → Executor → Dispatch Worker → API Handlers → Integration Tests

---

## Context

### Original Request
User provided a comprehensive blueprint for Phase 7 covering: review/approval service (7B), action apply service (7C), outbox dispatch worker (7D), HTTP API endpoints (7E), and audit/security testing (7F), plus a design document (7A).

### Interview Summary
**Key Discussions**:
- State machine: proposed → approved/rejected → applying → applied/failed
- Verdict values: approve, reject, cancel
- White-listed actions: create_followup_task, notify_owner, export_report, create_outbox_message
- Dry-run by default (ACTION_APPLY_DRY_RUN=true)
- Transaction boundaries required for approve/apply operations
- FOR UPDATE SKIP LOCKED for outbox worker concurrency
- No real LLM activation, no real Feishu/GitHub dispatch by default
- 8 recommended commits

**Research Findings**:
- Codebase uses Chi router, Struct + Interface DI pattern
- ai.action_proposal has CHECK constraint on 4 action_types (needs extension)
- ai.review_record exists but has no CHECK constraints and no Go code
- internal/action/ only has proposal_service.go (no execution)
- internal/outbox/ only has write repository (no dispatch)
- Worker is a stub (only pings DB)
- config/action_registry.yml exists but is never read by Go code
- audit.audit_log table exists with proper schema
- Existing test pattern: testcontainers-go for integration tests

### Metis Review
**Identified Gaps** (addressed):
- Schema-Config mismatch on action_type names → Resolved: use user's 4 white-listed actions, migration 011 reconciles
- ai.review_record missing CHECK constraints → Resolved: migration 011 adds verdict CHECK
- Worker stub needs real dispatch loop → Resolved: build dispatch worker in existing worker cmd
- Transaction boundaries for approve/apply → Resolved: explicitly specified in each task
- Actor identity extraction → Resolved: use request context/bearer token for now

---

## Work Objectives

### Core Objective
Implement a complete review/approval → action execution → outbox dispatch pipeline that safely bridges AI-generated proposals to real-world actions, with full audit logging and dry-run safety.

### Concrete Deliverables
- `docs/migration/phase-7-review-action-outbox-plan.md` — Design document
- `migrations/011_review_action_outbox.up.sql` + `.down.sql` — Schema migration
- `internal/review/domain.go` — Domain types (ReviewRecord, Verdict, etc.)
- `internal/review/repository.go` — Repository interface + pgx implementation
- `internal/review/service.go` — Review service (Approve, Reject, Cancel)
- `internal/action/apply_service.go` — Action apply service
- `internal/action/executor.go` — Action executor interface + implementations
- `internal/action/registry.go` — Action registry (YAML parsing + whitelist)
- `internal/outbox/dispatch_worker.go` — Dispatch worker with FOR UPDATE SKIP LOCKED
- `internal/adapter/feishu.go` — Feishu adapter (stubbed)
- `internal/adapter/github.go` — GitHub adapter (stubbed)
- `internal/api/handler/review.go` — Review HTTP handlers
- `internal/api/handler/action.go` — Action execution HTTP handlers
- `internal/api/handler/outbox.go` — Outbox dispatch HTTP handlers (enhance existing)
- `internal/api/handler/audit.go` — Audit query HTTP handlers
- Integration tests for all state transitions
- Security tests for unapproved execution prevention

### Definition of Done
- [ ] All 8 commits pass CI
- [ ] `bun test ./...` passes (all integration tests)
- [ ] `go build ./cmd/...` succeeds
- [ ] curl-based acceptance criteria pass for every endpoint
- [ ] No unapproved proposal can be executed (security test)
- [ ] Dry-run mode produces no side effects
- [ ] Dispatch worker processes events with FOR UPDATE SKIP LOCKED

### Must Have
- Approve/Reject/Cancel endpoints with transaction safety
- Action execution with dry-run support and white-list enforcement
- Outbox dispatch worker with retry logic
- Audit logging for every state transition
- Migration 011 with schema updates
- Integration tests with testcontainers PostgreSQL

### Must NOT Have (Guardrails)
- **NO** real LLM calls (use disabled provider or mocks)
- **NO** real Feishu/GitHub API calls (stubbed adapters only, unless explicitly configured)
- **NO** writes to `raw.*`, `dwd.*`, `mart.*` tables from action executor
- **NO** new action types beyond the 4 white-listed ones
- **NO** role-based access control beyond config-based `allowed_by` checking
- **NO** UI/React changes (backend only)
- **NO** email/SMS/webhook dispatch channels
- **NO** proposal editing/modification after creation
- **NO** batch approve/reject (single proposal only)
- **NO** scheduled/recurring action execution
- **NO** human-in-the-loop UI for review

---

## Verification Strategy (MANDATORY)

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed. No exceptions.
> Acceptance criteria requiring "user manually tests/confirms" are FORBIDDEN.

### Test Decision
- **Infrastructure exists**: YES (testcontainers-go, bun test)
- **Automated tests**: TDD (write failing test first, then implementation)
- **Framework**: bun test + testcontainers-go for PostgreSQL
- **If TDD**: Each task follows RED (failing test) → GREEN (minimal impl) → REFACTOR

### QA Policy
Every task MUST include agent-executed QA scenarios (see TODO template below).
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **API/Backend**: Use Bash (curl) — Send requests, assert status + response fields
- **Database**: Use Bash (psql) — Query tables, assert row counts and column values
- **Integration**: Use Bash (bun test) — Run Go tests, assert PASS

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Foundation — Schema + Domain + Registry):
├── Task 1: Migration 011 — Extend schema for review/apply state machine [quick]
├── Task 2: Review domain types (domain.go) [quick]
├── Task 3: Action registry parser (registry.go) [quick]
├── Task 4: Executor interface + base types (executor.go) [quick]
└── Task 5: Adapter stubs (feishu.go, github.go) [quick]

Wave 2 (Core Services — MAX PARALLEL):
├── Task 6: Review repository (repository.go) [unspecified-high]
├── Task 7: Review service (service.go) [unspecified-high]
├── Task 8: Action apply service (apply_service.go) [unspecified-high]
├── Task 9: Outbox dispatch worker (dispatch_worker.go) [unspecified-high]
└── Task 10: Audit logging integration (audit.go helper) [quick]

Wave 3 (API Layer + Integration):
├── Task 11: Review HTTP handlers (review.go) [unspecified-high]
├── Task 12: Action execution HTTP handlers (action.go) [unspecified-high]
├── Task 13: Outbox dispatch HTTP handlers (outbox.go enhancement) [unspecified-high]
├── Task 14: Audit query HTTP handlers (audit.go) [quick]
└── Task 15: Wire all handlers into router (api.go) [quick]

Wave 4 (Tests + Design Doc + Security):
├── Task 16: Review service integration tests [unspecified-high]
├── Task 17: Action apply integration tests [unspecified-high]
├── Task 18: Outbox dispatch integration tests [unspecified-high]
├── Task 19: Security tests (no unapproved execution) [unspecified-high]
├── Task 20: Design document (phase-7-review-action-outbox-plan.md) [writing]
└── Task 21: End-to-end state machine test [unspecified-high]

Wave FINAL (After ALL tasks — 4 parallel reviews, then user okay):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Real manual QA (unspecified-high)
└── Task F4: Scope fidelity check (deep)
-> Present results -> Get explicit user okay
```

### Dependency Matrix

- **1**: — → 6, 7, 8, 9, 11, 12, 13
- **2**: — → 6, 7
- **3**: — → 8, 9
- **4**: — → 8, 9
- **5**: — → 9
- **6**: 1, 2 → 7, 11, 16
- **7**: 2, 6 → 11, 16, 21
- **8**: 1, 3, 4 → 12, 17, 21
- **9**: 1, 3, 4, 5 → 13, 18
- **10**: — → 7, 8, 14
- **11**: 6, 7 → 15, 21
- **12**: 8 → 15, 21
- **13**: 9 → 15, 18
- **14**: 10 → 15
- **15**: 11, 12, 13, 14 → 21
- **16**: 6, 7 → F1-F4
- **17**: 8 → F1-F4
- **18**: 9 → F1-F4
- **19**: 7, 8 → F1-F4
- **20**: — → F1-F4
- **21**: 7, 8, 11, 12, 15 → F1-F4

### Agent Dispatch Summary

- **Wave 1**: **5** tasks → all `quick`
- **Wave 2**: **5** tasks → T6-T9 `unspecified-high`, T10 `quick`
- **Wave 3**: **5** tasks → T11-T13 `unspecified-high`, T14-T15 `quick`
- **Wave 4**: **6** tasks → T16-T19, T21 `unspecified-high`, T20 `writing`
- **FINAL**: **4** tasks → F1 `oracle`, F2 `unspecified-high`, F3 `unspecified-high`, F4 `deep`

---

## TODOs

- [x] 1. Migration 011 — Extend schema for review/apply state machine

  **What to do**:
  - Create `migrations/011_review_action_outbox.up.sql` and `.down.sql`
  - Extend `ai.action_proposal.apply_status` CHECK constraint to add: `applying`, `applied`, `failed`
  - Add `CHECK (verdict IN ('approve', 'reject', 'cancel'))` to `ai.review_record`
  - Ensure `ai.review_record` has proper indexes: `CREATE INDEX idx_review_record_proposal_id ON ai.review_record(proposal_id)`
  - Add `reviewed_at` timestamp column to `ai.review_record` if missing
  - Reconcile action_type: the 4 white-listed actions are `create_followup_task`, `notify_owner`, `export_report`, `create_outbox_message` — ensure CHECK or docs reflect this
  - Add `ops.outbox_event` index: `CREATE INDEX idx_outbox_event_status_created ON ops.outbox_event(status, created_at)` if missing

  **Must NOT do**:
  - Do NOT drop existing data in ai.action_proposal
  - Do NOT change columns unrelated to review/apply/dispatch
  - Do NOT add foreign keys that don't exist in current schema

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - Reason: Schema migration is straightforward DDL

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2-5)
  - **Blocks**: Tasks 6, 7, 8, 9, 11, 12, 13
  - **Blocked By**: None

  **References**:
  - `migrations/010_phase6_enhance.up.sql` — Follow existing migration style
  - `migrations/007_ai_schema.up.sql` — ai.review_record original schema
  - `config/action_registry.yml` — Action types (for reconciliation reference)

  **Acceptance Criteria**:
  - [ ] Migration files exist and follow naming convention
  - [ ] `bun run migrate:up` applies successfully
  - [ ] `bun run migrate:down` reverts successfully
  - [ ] `psql -c "\d ai.action_proposal"` shows extended CHECK constraint
  - [ ] `psql -c "\d ai.review_record"` shows verdict CHECK constraint
  - [ ] `psql -c "SELECT constraint_name FROM information_schema.check_constraints WHERE table_name='action_proposal'"` returns correct constraints

  **QA Scenarios**:
  ```
  Scenario: Migration applies cleanly
    Tool: Bash
    Steps:
      1. Run `bun run migrate:up`
      2. Run `psql -c "SELECT column_name, data_type FROM information_schema.columns WHERE table_name='review_record' AND table_schema='ai'"`
    Expected Result: All expected columns present, CHECK constraints active
    Evidence: .sisyphus/evidence/task-1-migration-applied.txt

  Scenario: Migration rolls back cleanly
    Tool: Bash
    Steps:
      1. Run `bun run migrate:down`
      2. Verify schema reverts to pre-migration state
    Expected Result: No errors, schema matches migration 010 state
    Evidence: .sisyphus/evidence/task-1-migration-rollback.txt
  ```

  **Commit**: YES
  - Message: `feat(migration): add 011_review_action_outbox for review/apply state machine`
  - Files: `migrations/011_review_action_outbox.up.sql`, `migrations/011_review_action_outbox.down.sql`

- [x] 2. Review domain types (domain.go)

  **What to do**:
  - Create `internal/review/domain.go`
  - Define types: `ReviewRecord`, `Verdict` (const: VerdictApprove, VerdictReject, VerdictCancel), `ReviewRequest`
  - Define `ReviewRecord` struct with fields: ID, ProposalID, ReviewerID, Verdict, Feedback, CreatedAt, ReviewedAt
  - Add validation methods: `Verdict.IsValid()`, `ReviewRequest.Validate()`
  - Follow existing domain pattern from `internal/decision/domain.go`

  **Must NOT do**:
  - Do NOT add business logic (keep domain types pure)
  - Do NOT import repository/service packages
  - Do NOT add JSON tags unless needed for API

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 3-5)
  - **Blocks**: Tasks 6, 7
  - **Blocked By**: None

  **References**:
  - `internal/decision/domain.go` — Follow struct + method pattern
  - `internal/action/domain.go` — Action proposal types for reference

  **Acceptance Criteria**:
  - [ ] File compiles: `go build ./internal/review/`
  - [ ] `VerdictApprove`, `VerdictReject`, `VerdictCancel` constants defined
  - [ ] `ReviewRequest.Validate()` returns error for invalid verdict

  **QA Scenarios**:
  ```
  Scenario: Domain types compile and validate
    Tool: Bash
    Steps:
      1. Run `go build ./internal/review/`
      2. Run `go test ./internal/review/ -run TestDomainTypes -v`
    Expected Result: Build succeeds, test passes
    Evidence: .sisyphus/evidence/task-2-domain-build.txt
  ```

  **Commit**: NO (groups with Task 6)

- [x] 3. Action registry parser (registry.go)

  **What to do**:
  - Create `internal/action/registry.go`
  - Parse `config/action_registry.yml` at startup into in-memory struct
  - Define `ActionRegistry` struct with methods: `IsAllowed(actionType string) bool`, `GetActionConfig(actionType string) (ActionConfig, bool)`
  - Hard-code the 4 white-listed actions as the allowed set: `create_followup_task`, `notify_owner`, `export_report`, `create_outbox_message`
  - Read `requires_approval`, `allowed_by` from YAML for metadata
  - Cache parsed config; reload on SIGHUP (optional, nice-to-have)

  **Must NOT do**:
  - Do NOT allow actions beyond the 4 white-listed ones
  - Do NOT implement complex RBAC (just check `allowed_by` roles)
  - Do NOT write to the YAML file

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1-2, 4-5)
  - **Blocks**: Tasks 8, 9
  - **Blocked By**: None

  **References**:
  - `config/action_registry.yml` — Source of truth for action metadata
  - `internal/config/config.go` — How config is loaded in this project

  **Acceptance Criteria**:
  - [ ] `go build ./internal/action/` succeeds
  - [ ] `registry.IsAllowed("notify_owner")` returns true
  - [ ] `registry.IsAllowed("hack_database")` returns false
  - [ ] `registry.GetActionConfig("export_report")` returns correct config from YAML

  **QA Scenarios**:
  ```
  Scenario: Registry parses config and enforces whitelist
    Tool: Bash
    Steps:
      1. Run `go test ./internal/action/ -run TestRegistry -v`
    Expected Result: PASS, whitelist enforced correctly
    Evidence: .sisyphus/evidence/task-3-registry-test.txt
  ```

  **Commit**: NO (groups with Task 8)

- [x] 4. Executor interface + base types (executor.go)

  **What to do**:
  - Create `internal/action/executor.go`
  - Define `ActionExecutor` interface with method: `Execute(ctx context.Context, proposal ActionProposal, dryRun bool) (ExecutionResult, error)`
  - Define `ExecutionResult` struct: Success bool, DryRun bool, DispatchPayload map[string]interface{}, Error string
  - Define `ExecutionContext` struct for passing runtime context
  - Create a `NoOpExecutor` that logs what it would do (for dry-run and unconfigured adapters)

  **Must NOT do**:
  - Do NOT implement real external API calls here
  - Do NOT import adapter packages (interface only)
  - Do NOT add business logic

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1-3, 5)
  - **Blocks**: Tasks 8, 9
  - **Blocked By**: None

  **References**:
  - `internal/action/proposal_service.go` — ActionProposal type
  - `internal/llm/executor.go` — If LLM executor pattern exists, follow it

  **Acceptance Criteria**:
  - [ ] `go build ./internal/action/` succeeds
  - [ ] `NoOpExecutor.Execute()` returns `DryRun: true` when dryRun=true
  - [ ] Interface is importable by adapter packages

  **QA Scenarios**:
  ```
  Scenario: Executor interface compiles
    Tool: Bash
    Steps:
      1. Run `go build ./internal/action/`
      2. Run `go test ./internal/action/ -run TestExecutor -v`
    Expected Result: Build and tests pass
    Evidence: .sisyphus/evidence/task-4-executor-build.txt
  ```

  **Commit**: NO (groups with Task 8)

- [x] 5. Adapter stubs (feishu.go, github.go)

  **What to do**:
  - Create `internal/adapter/feishu.go` — Implement `ActionExecutor` interface
    - Constructor: `NewFeishuAdapter(config FeishuConfig) (*FeishuAdapter, error)`
    - `Execute()` logs the payload, returns success if not dry-run, returns DryRun indicator if dry-run
    - Configurable via environment: `FEISHU_WEBHOOK_URL` (optional, defaults to "")
    - If webhook URL is empty, return error "feishu webhook not configured"
  - Create `internal/adapter/github.go` — Similar pattern
    - Configurable via: `GITHUB_TOKEN`, `GITHUB_REPO`
    - If token is empty, return error "github token not configured"
  - Both adapters produce structured `DispatchPayload` for outbox events

  **Must NOT do**:
  - Do NOT make real HTTP calls in default configuration
  - Do NOT store credentials in code (read from env/config only)
  - Do NOT implement full API surface (just stub for Phase 7)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1-4)
  - **Blocks**: Task 9
  - **Blocked By**: Task 4 (executor interface)

  **References**:
  - `internal/action/executor.go` — Interface to implement
  - `internal/config/config.go` — Config loading pattern

  **Acceptance Criteria**:
  - [ ] `go build ./internal/adapter/` succeeds
  - [ ] `FeishuAdapter.Execute()` with dryRun=true returns DryRun: true, no HTTP call
  - [ ] `FeishuAdapter.Execute()` with empty webhook returns error
  - [ ] `GitHubAdapter.Execute()` with empty token returns error

  **QA Scenarios**:
  ```
  Scenario: Feishu adapter dry-run
    Tool: Bash
    Steps:
      1. Run `go test ./internal/adapter/ -run TestFeishuDryRun -v`
    Expected Result: PASS, no HTTP calls made
    Evidence: .sisyphus/evidence/task-5-feishu-dryrun.txt

  Scenario: GitHub adapter unconfigured
    Tool: Bash
    Steps:
      1. Run `go test ./internal/adapter/ -run TestGitHubUnconfigured -v`
    Expected Result: PASS, returns "not configured" error
    Evidence: .sisyphus/evidence/task-5-github-unconfigured.txt
  ```

  **Commit**: YES
  - Message: `feat(adapter): add Feishu and GitHub adapter stubs with dry-run support`
  - Files: `internal/adapter/feishu.go`, `internal/adapter/github.go`, `internal/adapter/*.go`

### Wave 2: Core Services (After Wave 1 — 5 tasks parallel)

- [x] 6. **Review Service Implementation** (`internal/review/service.go`)

  **What to do**:
  - Implement `ReviewService` struct implementing `ReviewServiceInterface`
  - `ApproveProposal(ctx, id, reviewerID, feedback)`: transaction wrapping:
    1. SELECT proposal FOR UPDATE (prevent race conditions)
    2. Verify proposal exists and status="proposed"
    3. UPDATE proposal SET apply_status='approved'
    4. INSERT review_record (proposal_id, reviewer_id, verdict='approve', feedback)
    5. SELECT current case status → if "proposal_generated" UPDATE to "review_required"
    6. INSERT audit_log (category='review', action='proposal_approved', actor=reviewerID, resource_type='proposal', resource_id=id)
    7. If auto_outbox: create outbox_event (see task 11)
  - `RejectProposal(ctx, id, reviewerID, feedback)`: same transaction pattern, apply_status='rejected', audit action='proposal_rejected'
  - `CancelProposal(ctx, id, reviewerID, feedback)`: same transaction pattern, apply_status='rejected' (cancel maps to reject for simplicity), audit action='proposal_cancelled'
  - `GetReviewRecord(ctx, proposalID)`: query review_record by proposal_id
  - Return `ErrProposalNotFound`, `ErrInvalidState` (custom errors)

  **Must NOT do**:
  - Do NOT implement role-based access control (any authenticated user can review for now)
  - Do NOT send real notifications on approve/reject
  - Do NOT transition case to "closed" on reject (leave at "review_required")

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Complex transaction logic with multiple table mutations; requires careful error handling and rollback behavior
  - **Skills**: [`test-driven-development`]
    - `test-driven-development`: Critical for transaction-heavy service; tests must verify atomicity and race conditions

  **Parallelization**:
  - **Can Run In Parallel**: YES (with tasks 7, 8, 9, 10)
  - **Parallel Group**: Wave 2
  - **Blocks**: Tasks 12, 14, 15 (Review API, integration tests)
  - **Blocked By**: Task 3 (review domain types), Task 1 (migration with new CHECK constraints)

  **References**:
  - `internal/decision/case_service.go:30-60` - Service struct pattern (constructor + dependency injection)
  - `internal/review/repository.go` - Repository interface (task 2)
  - `internal/decision/repository.go` - Case repository for status transitions
  - `internal/audit/repository.go` - Audit log repository for inserts
  - `internal/api/middleware/auth.go` - Auth middleware for extracting actor identity

  **Acceptance Criteria**:
  - [ ] `go test ./internal/review/... -run TestApproveProposal -v` → PASS
  - [ ] Test: Approve "proposed" proposal → apply_status="approved", review_record created, audit_log created
  - [ ] Test: Approve already-approved proposal → returns ErrInvalidState (409)
  - [ ] Test: Approve non-existent proposal → returns ErrProposalNotFound (404)
  - [ ] Test: Concurrent approve attempts → only one succeeds (race condition test with sync.WaitGroup)
  - [ ] Test: Reject proposal → apply_status="rejected", review_record with verdict="reject"
  - [ ] Test: Cancel proposal → apply_status="rejected", review_record with verdict="cancel"
  - [ ] Test: Transaction rollback on audit failure → proposal status unchanged, no review_record

  **QA Scenarios**:
  ```
  Scenario: Approve proposal successfully
    Tool: Bash (curl)
    Preconditions: Proposal prop_test_001 exists with apply_status="proposed"
    Steps:
      1. curl -X POST http://localhost:8080/api/v1/proposals/prop_test_001/approve \
         -H "Authorization: Bearer test-token" \
         -d '{"reviewer_id":"user_1","feedback":"LGTM"}'
    Expected Result: HTTP 200, response body contains apply_status="approved"
    Evidence: .sisyphus/evidence/task-6-approve-ok.json

  Scenario: Approve already-approved proposal
    Tool: Bash (curl)
    Preconditions: prop_test_001 already approved
    Steps:
      1. curl -X POST http://localhost:8080/api/v1/proposals/prop_test_001/approve ...
    Expected Result: HTTP 409 Conflict, error message "proposal already reviewed"
    Evidence: .sisyphus/evidence/task-6-approve-409.json

  Scenario: Approve non-existent proposal
    Tool: Bash (curl)
    Steps:
      1. curl -X POST http://localhost:8080/api/v1/proposals/prop_nonexist/approve ...
    Expected Result: HTTP 404 Not Found
    Evidence: .sisyphus/evidence/task-6-approve-404.json

  Scenario: Concurrent approve race
    Tool: Bash (go test)
    Steps:
      1. Run TestConcurrentApprove with 10 goroutines
    Expected Result: Exactly 1 succeeds, 9 get ErrInvalidState
    Evidence: .sisyphus/evidence/task-6-concurrent-race.txt
  ```

  **Commit**: YES
  - Message: `feat(review): implement ReviewService with transaction safety`
  - Files: `internal/review/service.go`, `internal/review/service_test.go`, `internal/review/errors.go`
  - Pre-commit: `go test ./internal/review/...`

- [x] 7. **Action Apply Service** (`internal/action/apply_service.go`)

  **What to do**:
  - Implement `ApplyService` struct with config-driven dry-run mode
  - `ExecuteProposal(ctx, id, opts)`: transaction wrapping:
    1. SELECT proposal FOR UPDATE
    2. Verify proposal exists, apply_status="approved", action_type in whitelist
    3. UPDATE proposal SET apply_status='applying'
    4. Call executor.Execute(ctx, proposal) → get result (dry-run or real)
    5. If dry-run: UPDATE proposal SET apply_status='approved' (revert), return DryRunResult
    6. If real execution succeeds: UPDATE proposal SET apply_status='applied'
    7. If real execution fails: UPDATE proposal SET apply_status='failed', log error
    8. INSERT audit_log (action='proposal_executed' or 'proposal_execution_failed')
    9. If real execution and creates outbox: create outbox_event (task 11)
  - Dry-run detection: read `ACTION_APPLY_DRY_RUN` env var (default true) or `opts.ForceDryRun`
  - White-list enforcement: hardcoded slice `["create_followup_task", "notify_owner", "export_report", "create_outbox_message"]`
  - Return `ErrProposalNotApproved`, `ErrActionNotAllowed`, `ErrExecutionFailed`

  **Must NOT do**:
  - Do NOT execute proposals with apply_status != "approved"
  - Do NOT execute non-whitelisted action types (even in dry-run)
  - Do NOT write to raw/dwd/mart tables (executor handles this)
  - Do NOT create outbox events in dry-run mode

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Complex state machine with transaction boundaries and dry-run branching
  - **Skills**: [`test-driven-development`]
    - `test-driven-development`: State machine transitions and dry-run branching need thorough test coverage

  **Parallelization**:
  - **Can Run In Parallel**: YES (with tasks 6, 8, 9, 10)
  - **Parallel Group**: Wave 2
  - **Blocks**: Tasks 13, 15 (Action API, integration tests)
  - **Blocked By**: Task 4 (executor interface), Task 5 (adapter stubs)

  **References**:
  - `internal/action/proposal_service.go` - Existing proposal service patterns
  - `internal/action/executor.go` - Executor interface (task 4)
  - `internal/adapter/feishu.go`, `internal/adapter/github.go` - Adapter stubs (task 5)
  - `internal/config/config.go` - Config loading for ACTION_APPLY_DRY_RUN
  - `internal/audit/repository.go` - Audit logging

  **Acceptance Criteria**:
  - [ ] `go test ./internal/action/... -run TestExecuteProposal -v` → PASS
  - [ ] Test: Execute approved proposal in dry-run → returns DryRunResult, apply_status stays "approved", no outbox created
  - [ ] Test: Execute approved proposal with dry-run=false → apply_status="applying" → "applied", outbox event created
  - [ ] Test: Execute unapproved proposal → returns ErrProposalNotApproved (403)
  - [ ] Test: Execute non-whitelisted action → returns ErrActionNotAllowed (403)
  - [ ] Test: Execute with executor failure → apply_status="failed", audit logged
  - [ ] Test: Transaction rollback on outbox failure → proposal status reverts to "approved"

  **QA Scenarios**:
  ```
  Scenario: Execute approved proposal (dry-run)
    Tool: Bash (curl)
    Preconditions: Proposal prop_test_002 approved, ACTION_APPLY_DRY_RUN=true
    Steps:
      1. curl -X POST http://localhost:8080/api/v1/proposals/prop_test_002/execute \
         -H "Authorization: Bearer test-token"
    Expected Result: HTTP 200, response contains {"dry_run":true,"action_type":"notify_owner"}
    Evidence: .sisyphus/evidence/task-7-execute-dryrun.json

  Scenario: Execute approved proposal (real)
    Tool: Bash (curl)
    Preconditions: prop_test_002 approved, ACTION_APPLY_DRY_RUN=false
    Steps:
      1. curl -X POST http://localhost:8080/api/v1/proposals/prop_test_002/execute \
         -H "Authorization: Bearer test-token" \
         -d '{"dry_run":false}'
    Expected Result: HTTP 200, apply_status="applied", outbox event created
    Evidence: .sisyphus/evidence/task-7-execute-real.json

  Scenario: Execute unapproved proposal
    Tool: Bash (curl)
    Preconditions: prop_test_003 has apply_status="proposed"
    Steps:
      1. curl -X POST http://localhost:8080/api/v1/proposals/prop_test_003/execute ...
    Expected Result: HTTP 403 Forbidden, "proposal must be approved before execution"
    Evidence: .sisyphus/evidence/task-7-execute-403.json

  Scenario: Execute non-whitelisted action
    Tool: Bash (curl)
    Preconditions: prop_test_004 has action_type="delete_database" (not in whitelist)
    Steps:
      1. curl -X POST http://localhost:8080/api/v1/proposals/prop_test_004/execute ...
    Expected Result: HTTP 403 Forbidden, "action type not allowed"
    Evidence: .sisyphus/evidence/task-7-execute-whitelist-403.json
  ```

  **Commit**: YES
  - Message: `feat(action): implement ApplyService with dry-run and whitelist enforcement`
  - Files: `internal/action/apply_service.go`, `internal/action/apply_service_test.go`, `internal/action/errors.go`
  - Pre-commit: `go test ./internal/action/...`

- [x] 8. **Outbox Write Repository Enhancement** (`internal/outbox/repository.go`)

  **What to do**:
  - Extend existing `OutboxRepository` with methods needed for dispatch worker:
    - `CreateEvent(ctx, tx, event)`: insert into ops.outbox_event (returns error if tx nil)
    - `ListPendingEvents(ctx, limit)`: SELECT * FROM ops.outbox_event WHERE status='pending' ORDER BY created_at LIMIT $1
    - `GetEventForDispatch(ctx, id)`: SELECT ... FROM ops.outbox_event WHERE id=$1 AND status IN ('pending','failed') FOR UPDATE SKIP LOCKED
    - `MarkDispatched(ctx, tx, id, adapterType, externalID)`: UPDATE status='dispatched', dispatched_at=NOW(), dispatched_by=adapterType, external_ref=externalID
    - `MarkFailed(ctx, tx, id, errorMsg)`: UPDATE status='failed', last_error=errorMsg, increment dispatch_attempts
    - `MarkCancelled(ctx, tx, id)`: UPDATE status='cancelled'
    - `IncrementAttempts(ctx, tx, id)`: UPDATE dispatch_attempts = dispatch_attempts + 1
  - All methods accepting `tx` must use the transaction; methods without tx use their own connection
  - Add `OutboxEvent` domain model if not present

  **Must NOT do**:
  - Do NOT implement dispatch logic here (that's the worker in task 10)
  - Do NOT add methods for listing all events (only pending/failed)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Repository CRUD with straightforward SQL
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with tasks 6, 7, 9, 10)
  - **Parallel Group**: Wave 2
  - **Blocks**: Tasks 10, 11, 13 (dispatch worker, action outbox integration, outbox API)
  - **Blocked By**: Task 1 (migration 011 with ops.outbox_event schema)

  **References**:
  - `internal/outbox/repository.go` - Existing repository (extend, don't replace)
  - `migrations/011_review_action_outbox.sql` - ops.outbox_event schema (task 1)
  - `internal/repository/postgres.go` - Base repository patterns

  **Acceptance Criteria**:
  - [ ] `go test ./internal/outbox/... -run TestOutboxRepository -v` → PASS
  - [ ] Test: CreateEvent with tx → event inserted, rollback doesn't persist
  - [ ] Test: ListPendingEvents → returns only status='pending' ordered by created_at
  - [ ] Test: GetEventForDispatch with SKIP LOCKED → locks row, concurrent query skips it
  - [ ] Test: MarkDispatched → status='dispatched', dispatched_at set
  - [ ] Test: MarkFailed → status='failed', dispatch_attempts incremented
  - [ ] Test: MarkCancelled → status='cancelled'

  **QA Scenarios**:
  ```
  Scenario: Create outbox event in transaction
    Tool: Bash (go test)
    Steps:
      1. Run TestCreateEventTx
    Expected Result: PASS, event created and queryable
    Evidence: .sisyphus/evidence/task-8-create-event.txt

  Scenario: SKIP LOCKED prevents double dispatch
    Tool: Bash (go test)
    Steps:
      1. Run TestSkipLocked with 2 goroutines
    Expected Result: Only 1 goroutine gets the row
    Evidence: .sisyphus/evidence/task-8-skip-locked.txt
  ```

  **Commit**: YES
  - Message: `feat(outbox): extend repository with dispatch-oriented methods`
  - Files: `internal/outbox/repository.go`, `internal/outbox/repository_test.go`
  - Pre-commit: `go test ./internal/outbox/...`

- [x] 9. **Review HTTP Handlers** (`internal/api/handler/review.go`)

  **What to do**:
  - Implement `ReviewHandler` struct with service dependency
  - Routes (Chi router):
    - `POST /api/v1/proposals/{id}/approve` → `ApproveProposal`
    - `POST /api/v1/proposals/{id}/reject` → `RejectProposal`
    - `POST /api/v1/proposals/{id}/cancel` → `CancelProposal`
    - `GET /api/v1/proposals/{id}/review` → `GetReviewRecord` (returns review_record)
  - Request/response DTOs:
    - `ApproveRequest`: reviewer_id (string, required), feedback (string, optional)
    - `ReviewResponse`: proposal_id, reviewer_id, verdict, feedback, created_at
  - Error mapping:
    - ErrProposalNotFound → 404
    - ErrInvalidState → 409
    - Generic → 500
  - Wire handler in `internal/api/router.go` with auth middleware

  **Must NOT do**:
  - Do NOT add PUT/DELETE endpoints for review records (immutable)
  - Do NOT implement batch operations (single proposal only)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: HTTP handler with standard patterns, moderate complexity
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with tasks 6, 7, 8, 10)
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 15 (end-to-end integration tests)
  - **Blocked By**: Task 6 (review service implementation)

  **References**:
  - `internal/api/handler/decision.go` - Handler pattern with DTOs and error mapping
  - `internal/api/router.go` - Chi router registration
  - `internal/api/middleware/auth.go` - Auth middleware
  - `internal/review/service.go` - ReviewService (task 6)

  **Acceptance Criteria**:
  - [ ] `go test ./internal/api/handler/... -run TestReviewHandler -v` → PASS
  - [ ] Test: POST /proposals/{id}/approve with valid request → 200
  - [ ] Test: POST /proposals/{id}/approve already approved → 409
  - [ ] Test: POST /proposals/{id}/approve non-existent → 404
  - [ ] Test: POST /proposals/{id}/reject → 200, review record created
  - [ ] Test: GET /proposals/{id}/review → returns review record
  - [ ] Test: Missing reviewer_id → 400 Bad Request

  **QA Scenarios**:
  ```
  Scenario: Approve proposal via HTTP
    Tool: Bash (curl)
    Steps:
      1. curl -X POST http://localhost:8080/api/v1/proposals/prop_test_005/approve \
         -H "Authorization: Bearer test-token" \
         -H "Content-Type: application/json" \
         -d '{"reviewer_id":"user_1","feedback":"Approved"}'
    Expected Result: HTTP 200, JSON with proposal_id and apply_status="approved"
    Evidence: .sisyphus/evidence/task-9-approve-http.json

  Scenario: Get review record
    Tool: Bash (curl)
    Steps:
      1. curl http://localhost:8080/api/v1/proposals/prop_test_005/review \
         -H "Authorization: Bearer test-token"
    Expected Result: HTTP 200, JSON with verdict="approve", reviewer_id="user_1"
    Evidence: .sisyphus/evidence/task-9-get-review.json

  Scenario: Approve without reviewer_id
    Tool: Bash (curl)
    Steps:
      1. curl -X POST .../proposals/prop_test_005/approve -d '{"feedback":"test"}'
    Expected Result: HTTP 400, error "reviewer_id is required"
    Evidence: .sisyphus/evidence/task-9-missing-reviewer.json
  ```

  **Commit**: YES
  - Message: `feat(api): add review handlers for approve/reject/cancel/get`
  - Files: `internal/api/handler/review.go`, `internal/api/handler/review_test.go`
  - Pre-commit: `go test ./internal/api/handler/...`

- [x] 10. **Outbox Dispatch Worker** (`internal/worker/dispatch_worker.go`)

  **What to do**:
  - Refactor existing worker stub (`internal/worker/worker.go`) into real dispatch worker
  - `DispatchWorker` struct with dependencies: outbox repository, adapter registry, config
  - `Run(ctx)`: loop with ticker (interval from config, default 30s):
    1. SELECT * FROM ops.outbox_event WHERE status IN ('pending','failed') AND (next_retry_at IS NULL OR next_retry_at <= NOW()) ORDER BY created_at LIMIT $batch_size FOR UPDATE SKIP LOCKED
    2. For each event:
       a. Determine adapter from event.channel (feishu/github)
       b. Increment dispatch_attempts
       c. Call adapter.Execute(ctx, event.payload_json)
       d. On success: MarkDispatched
       e. On failure: MarkFailed, set next_retry_at = NOW() + exponential_backoff(attempts)
    3. If batch empty: sleep until next tick
  - `Stop()`: graceful shutdown with context cancellation
  - Exponential backoff: 1min, 2min, 4min, 8min, 16min, then max 30min
  - Max attempts: 10, then mark as "permanently_failed" (manual retry only)

  **Must NOT do**:
  - Do NOT process events with status='dispatched' or 'cancelled'
  - Do NOT dispatch in dry-run mode (events should not be created in dry-run)
  - Do NOT implement priority queues (FIFO only)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Background worker with concurrent safety and retry logic
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with tasks 6, 7, 8, 9)
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 16 (worker integration tests)
  - **Blocked By**: Task 8 (outbox repository methods), Task 5 (adapter stubs)

  **References**:
  - `internal/worker/worker.go` - Existing stub (replace/refactor)
  - `internal/outbox/repository.go` - Repository methods (task 8)
  - `internal/adapter/registry.go` - Adapter resolution (task 5)
  - `internal/adapter/feishu.go`, `internal/adapter/github.go` - Adapters (task 5)
  - `cmd/baxi-worker/main.go` - Worker entrypoint

  **Acceptance Criteria**:
  - [ ] `go test ./internal/worker/... -run TestDispatchWorker -v` → PASS
  - [ ] Test: Worker processes pending event → status becomes "dispatched"
  - [ ] Test: Worker skips locked event → another worker gets different event
  - [ ] Test: Failed dispatch increments attempts and sets next_retry_at
  - [ ] Test: Max attempts reached → status="permanently_failed"
  - [ ] Test: Worker graceful shutdown → finishes current batch then exits
  - [ ] Test: No events → worker sleeps, no errors

  **QA Scenarios**:
  ```
  Scenario: Worker dispatches pending event
    Tool: Bash (integration test)
    Preconditions: Outbox event evt_test_001 pending, Feishu adapter stubbed
    Steps:
      1. Start worker with 1s ticker
      2. Wait 2s
      3. Query ops.outbox_event WHERE id='evt_test_001'
    Expected Result: status="dispatched", dispatched_at set
    Evidence: .sisyphus/evidence/task-10-dispatch-success.txt

  Scenario: Worker handles failure with retry
    Tool: Bash (integration test)
    Preconditions: evt_test_002 pending, adapter returns error
    Steps:
      1. Start worker
      2. Wait for processing
      3. Query event
    Expected Result: status="failed", dispatch_attempts=1, next_retry_at in future
    Evidence: .sisyphus/evidence/task-10-dispatch-fail.txt
  ```

  **Commit**: YES
  - Message: `feat(worker): implement outbox dispatch worker with retry and SKIP LOCKED`
  - Files: `internal/worker/dispatch_worker.go`, `internal/worker/dispatch_worker_test.go`, `internal/worker/worker.go`
  - Pre-commit: `go test ./internal/worker/...`

### Wave 3: Integration & Action Execution (After Wave 2 — 5 tasks parallel)

- [x] 11. **Action Outbox Integration** (`internal/action/outbox_integration.go`)

  **What to do**:
  - Bridge `ApplyService` (task 7) with `OutboxRepository` (task 8)
  - `CreateOutboxEventFromProposal(ctx, tx, proposal)`: creates ops.outbox_event record:
    - `channel`: determined by action_type mapping (export_report → feishu, notify_owner → feishu, create_followup_task → github, create_outbox_message → feishu)
    - `payload_json`: marshal proposal.payload (map[string]interface{}) into JSON envelope with metadata (proposal_id, case_id, action_type, created_at)
    - `status`: "pending"
    - `dispatch_attempts`: 0
  - Called by `ApplyService.ExecuteProposal` after successful execution (non-dry-run)
  - Return error if payload marshalling fails (roll back transaction)

  **Must NOT do**:
  - Do NOT create outbox events in dry-run mode
  - Do NOT create events for non-whitelisted actions
  - Do NOT implement payload transformation beyond simple JSON envelope

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple bridge function, low complexity
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with tasks 12, 13, 14, 15)
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 15 (end-to-end tests)
  - **Blocked By**: Tasks 7, 8 (ApplyService and OutboxRepository)

  **References**:
  - `internal/action/apply_service.go` - ApplyService (task 7)
  - `internal/outbox/repository.go` - OutboxRepository (task 8)
  - `internal/action/proposal_service.go` - Proposal payload structure

  **Acceptance Criteria**:
  - [ ] `go test ./internal/action/... -run TestOutboxIntegration -v` → PASS
  - [ ] Test: Create outbox from approved proposal → event created with correct channel and payload
  - [ ] Test: Create outbox from dry-run execution → no event created
  - [ ] Test: Payload JSON contains proposal_id, case_id, action_type keys
  - [ ] Test: Failed outbox creation rolls back proposal status to "approved"

  **QA Scenarios**:
  ```
  Scenario: Outbox event created after execution
    Tool: Bash (go test)
    Steps:
      1. Execute approved proposal with dry_run=false
      2. Query ops.outbox_event by resource_id=proposal_id
    Expected Result: Event exists, channel="feishu", status="pending"
    Evidence: .sisyphus/evidence/task-11-outbox-created.txt
  ```

  **Commit**: YES (groups with task 7)
  - Message: `feat(action): wire ApplyService to OutboxRepository`
  - Files: `internal/action/outbox_integration.go`, `internal/action/apply_service.go`

- [x] 12. **Action Execution HTTP Handlers** (`internal/api/handler/action.go`)

  **What to do**:
  - Implement `ActionHandler` struct with ApplyService dependency
  - Routes:
    - `POST /api/v1/proposals/{id}/execute` → `ExecuteProposal`
    - `GET /api/v1/proposals/{id}/status` → `GetExecutionStatus` (returns apply_status, last_error, dry_run indicator)
  - Request/response DTOs:
    - `ExecuteRequest`: dry_run (bool, optional, defaults to config value)
    - `ExecuteResponse`: proposal_id, apply_status, dry_run, dispatched (bool), outbox_event_id (string, if created)
  - Error mapping:
    - ErrProposalNotFound → 404
    - ErrProposalNotApproved → 403
    - ErrActionNotAllowed → 403
    - ErrExecutionFailed → 500
  - Wire handler in `internal/api/router.go`

  **Must NOT do**:
  - Do NOT add PUT/DELETE for execution (execution is create-only)
  - Do NOT implement batch execution

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Standard HTTP handler, moderate complexity
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with tasks 11, 13, 14, 15)
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 15 (end-to-end tests)
  - **Blocked By**: Task 7 (ApplyService)

  **References**:
  - `internal/api/handler/decision.go` - Handler pattern
  - `internal/api/router.go` - Router registration
  - `internal/action/apply_service.go` - ApplyService (task 7)

  **Acceptance Criteria**:
  - [ ] `go test ./internal/api/handler/... -run TestActionHandler -v` → PASS
  - [ ] Test: POST /proposals/{id}/execute approved → 200 with dry_run indicator
  - [ ] Test: POST /proposals/{id}/execute unapproved → 403
  - [ ] Test: POST /proposals/{id}/execute non-whitelisted → 403
  - [ ] Test: GET /proposals/{id}/status → returns apply_status

  **QA Scenarios**:
  ```
  Scenario: Execute approved proposal
    Tool: Bash (curl)
    Steps:
      1. curl -X POST http://localhost:8080/api/v1/proposals/prop_test_006/execute \
         -H "Authorization: Bearer test-token"
    Expected Result: HTTP 200, JSON with dry_run=true/false based on config
    Evidence: .sisyphus/evidence/task-12-execute-http.json

  Scenario: Get execution status
    Tool: Bash (curl)
    Steps:
      1. curl http://localhost:8080/api/v1/proposals/prop_test_006/status \
         -H "Authorization: Bearer test-token"
    Expected Result: HTTP 200, JSON with apply_status="approved"
    Evidence: .sisyphus/evidence/task-12-status-http.json
  ```

  **Commit**: YES (groups with task 9)
  - Message: `feat(api): add action execution handlers`
  - Files: `internal/api/handler/action.go`, `internal/api/handler/action_test.go`

- [x] 13. **Outbox Management HTTP Handlers** (`internal/api/handler/outbox.go`)

  **What to do**:
  - Extend existing `internal/api/handler/outbox.go` (currently only has GET /outbox)
  - New routes:
    - `POST /api/v1/outbox/{id}/dispatch` → `ManualDispatch` (retry a failed/pending event)
    - `POST /api/v1/outbox/{id}/cancel` → `CancelEvent`
    - `GET /api/v1/outbox/{id}` → `GetEventDetail`
  - `ManualDispatch`: validate event exists and status IN ('pending','failed','permanently_failed'), call worker's dispatch logic for single event
  - `CancelEvent`: validate event exists and status IN ('pending','failed'), update to 'cancelled'
  - Response DTOs:
    - `OutboxEventResponse`: id, channel, status, dispatch_attempts, last_error, created_at, dispatched_at
  - Wire in router

  **Must NOT do**:
  - Do NOT add DELETE endpoint (events are immutable except status)
  - Do NOT allow cancelling already-dispatched events

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Standard CRUD handler with validation
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with tasks 11, 12, 14, 15)
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 15 (end-to-end tests)
  - **Blocked By**: Tasks 8, 10 (OutboxRepository and DispatchWorker)

  **References**:
  - `internal/api/handler/outbox.go` - Existing handler (extend)
  - `internal/outbox/repository.go` - Repository (task 8)
  - `internal/worker/dispatch_worker.go` - Worker dispatch logic (task 10)

  **Acceptance Criteria**:
  - [ ] `go test ./internal/api/handler/... -run TestOutboxHandler -v` → PASS
  - [ ] Test: POST /outbox/{id}/dispatch pending → 200, event dispatched
  - [ ] Test: POST /outbox/{id}/dispatch dispatched → 409 Conflict
  - [ ] Test: POST /outbox/{id}/cancel pending → 200, status="cancelled"
  - [ ] Test: GET /outbox/{id} → returns event detail

  **QA Scenarios**:
  ```
  Scenario: Manual dispatch retry
    Tool: Bash (curl)
    Steps:
      1. curl -X POST http://localhost:8080/api/v1/outbox/evt_test_003/dispatch \
         -H "Authorization: Bearer test-token"
    Expected Result: HTTP 200, status="dispatched"
    Evidence: .sisyphus/evidence/task-13-manual-dispatch.json

  Scenario: Cancel pending event
    Tool: Bash (curl)
    Steps:
      1. curl -X POST http://localhost:8080/api/v1/outbox/evt_test_003/cancel ...
    Expected Result: HTTP 200, status="cancelled"
    Evidence: .sisyphus/evidence/task-13-cancel.json
  ```

  **Commit**: YES (groups with task 9)
  - Message: `feat(api): extend outbox handlers with dispatch and cancel`
  - Files: `internal/api/handler/outbox.go`

- [x] 14. **Audit Integration** (`internal/audit/integration.go`)

  **What to do**:
  - Create `AuditIntegration` helper to standardize audit logging across review and action services
  - `LogProposalReviewed(ctx, tx, proposalID, reviewerID, verdict, feedback)`: inserts audit.audit_log
  - `LogProposalExecuted(ctx, tx, proposalID, actorID, dryRun, success, errorMsg)`: inserts audit.audit_log
  - `LogOutboxDispatched(ctx, tx, eventID, channel, success, errorMsg)`: inserts audit.audit_log
  - All methods accept `tx` for transaction-bound logging
  - `actorID` extraction: for now, use the reviewer_id/actor from request; if empty, use "system"
  - Metadata JSON: include relevant context (proposal payload summary, execution result)

  **Must NOT do**:
  - Do NOT implement generic audit middleware (service-layer only for now)
  - Do NOT log sensitive data (full payloads) in audit metadata (trim to summary)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple helper functions wrapping repository calls
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with tasks 11, 12, 13, 15)
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 6, 7 (ReviewService and ApplyService use audit integration)
  - **Blocked By**: Task 2 (audit repository interface)

  **References**:
  - `internal/audit/repository.go` - Audit repository (task 2)
  - `internal/review/service.go` - ReviewService (task 6)
  - `internal/action/apply_service.go` - ApplyService (task 7)

  **Acceptance Criteria**:
  - [ ] `go test ./internal/audit/... -run TestAuditIntegration -v` → PASS
  - [ ] Test: LogProposalReviewed → audit_log created with correct category/action/actor
  - [ ] Test: LogProposalExecuted with dry_run=true → metadata contains "dry_run":true
  - [ ] Test: LogOutboxDispatched → audit_log created with resource_type="outbox_event"
  - [ ] Test: All audit logs use transaction → rollback removes audit record

  **QA Scenarios**:
  ```
  Scenario: Audit log for approve
    Tool: Bash (psql)
    Preconditions: Proposal prop_test_007 approved
    Steps:
      1. psql -c "SELECT action, actor FROM audit.audit_log WHERE resource_id='prop_test_007' ORDER BY created_at"
    Expected Result: Contains row with action='proposal_approved', actor='user_1'
    Evidence: .sisyphus/evidence/task-14-audit-approve.txt
  ```

  **Commit**: YES (groups with task 2)
  - Message: `feat(audit): add AuditIntegration helper for review and action logging`
  - Files: `internal/audit/integration.go`, `internal/audit/integration_test.go`

- [x] 15. **End-to-End Integration Tests** (`test/integration/phase7_test.go`)

  **What to do**:
  - Create comprehensive integration test suite using testcontainers PostgreSQL
  - Test scenarios:
    1. **Full approval flow**: Create proposal → Approve → Execute (dry-run) → Verify no outbox → Execute (real) → Verify outbox → Worker dispatches → Verify audit logs
    2. **Rejection flow**: Create proposal → Reject → Verify cannot execute → Verify audit log
    3. **Security flow**: Try execute unapproved → Verify 403, no audit execution log, no outbox
    4. **Whitelist flow**: Create proposal with non-whitelisted action → Verify 403 on execute
    5. **Concurrent flow**: 5 proposals, approve all concurrently → verify no race conditions
    6. **Worker flow**: Create outbox events → start worker → verify all dispatched
  - Use `testutil.SetupTestDB()` for database setup
  - Use `httptest.Server` for API testing
  - Reset database between tests

  **Must NOT do**:
    - Do NOT mock database (use real PostgreSQL via testcontainers)
    - Do NOT test external APIs (Feishu/GitHub adapters are stubbed)

  **Recommended Agent Profile**:
    - **Category**: `deep`
      - Reason: Complex multi-service integration tests with real database
    - **Skills**: [`test-driven-development`]
      - `test-driven-development`: Integration tests verify the entire system works together

  **Parallelization**:
    - **Can Run In Parallel**: YES (with tasks 11, 12, 13, 14)
    - **Parallel Group**: Wave 3
    - **Blocks**: Tasks F1-F4 (final verification)
    - **Blocked By**: ALL previous tasks (6-14)

  **References**:
    - `internal/testutil/db.go` - Test database setup
    - `test/integration/decision_test.go` - Existing integration test pattern
    - `internal/api/router.go` - Full router for httptest

  **Acceptance Criteria**:
    - [ ] `go test ./test/integration/... -run TestPhase7 -v` → PASS (all 6 scenarios)
    - [ ] Test: Full approval flow → all state transitions correct, audit logs complete
    - [ ] Test: Rejection flow → execution blocked, correct error
    - [ ] Test: Security flow → 403 on unapproved, no side effects
    - [ ] Test: Whitelist flow → 403 on non-whitelisted action
    - [ ] Test: Concurrent flow → all proposals approved, no duplicates
    - [ ] Test: Worker flow → all events dispatched

  **QA Scenarios**:
    ```
    Scenario: Full approval and execution flow
      Tool: Bash (go test)
      Steps:
        1. Run TestFullApprovalFlow
      Expected Result: PASS, all assertions pass
      Evidence: .sisyphus/evidence/task-15-full-flow.txt

    Scenario: Security - cannot execute unapproved
      Tool: Bash (go test)
      Steps:
        1. Run TestSecurityUnapprovedExecution
      Expected Result: PASS, 403 returned, no audit execution log
      Evidence: .sisyphus/evidence/task-15-security.txt
    ```

  **Commit**: YES
    - Message: `test(integration): add Phase 7 end-to-end integration tests`
    - Files: `test/integration/phase7_test.go`
    - Pre-commit: `go test ./test/integration/... -run TestPhase7`

### Wave 4: Documentation & Final Polish (After Wave 3 — 6 tasks, mostly parallel)

- [x] 16. **Design Document** (`docs/migration/phase-7-review-action-outbox-plan.md`)

  **What to do**:
  - Write comprehensive design document covering:
    - Architecture overview (diagram: Review Service → Action Apply Service → Outbox Dispatch Worker)
    - State machine diagrams (proposal apply_status transitions, case status transitions)
    - Transaction boundaries (which operations are atomic)
    - Schema reconciliation (canonical action types, resolved mismatch)
    - Security model (whitelist, dry-run, approval gates)
    - Deployment notes (worker startup, config requirements)
  - Include mermaid diagrams for state machines
  - Document all new API endpoints with request/response examples

  **Must NOT do**:
    - Do NOT document future phases (keep to Phase 7 scope)
    - Do NOT include UI mockups (backend only)

  **Recommended Agent Profile**:
    - **Category**: `writing`
      - Reason: Documentation writing task
    - **Skills**: []

  **Parallelization**:
    - **Can Run In Parallel**: YES (with tasks 17, 18, 19, 20, 21)
    - **Parallel Group**: Wave 4
    - **Blocks**: None
    - **Blocked By**: None (can start immediately, but should reflect final decisions)

  **References**:
    - `docs/migration/phase-6-decision-case-llm-plan.md` - Previous phase design doc format
    - All implementation tasks (6-15) for accurate documentation

  **Acceptance Criteria**:
    - [ ] Document exists at `docs/migration/phase-7-review-action-outbox-plan.md`
    - [ ] Contains state machine diagram (mermaid)
    - [ ] Contains API endpoint reference (all 7B-7E endpoints)
    - [ ] Documents transaction boundaries
    - [ ] Documents security model and whitelist

  **Commit**: YES
    - Message: `docs: add Phase 7 design document`
    - Files: `docs/migration/phase-7-review-action-outbox-plan.md`

- [x] 17. **OpenAPI Documentation** (`docs/openapi/phase7.yaml`)

  **What to do**:
  - Write OpenAPI 3.0 spec for all new Phase 7 endpoints:
    - `/proposals/{id}/approve`, `/proposals/{id}/reject`, `/proposals/{id}/cancel`
    - `/proposals/{id}/execute`, `/proposals/{id}/status`
    - `/proposals/{id}/review`
    - `/outbox/{id}/dispatch`, `/outbox/{id}/cancel`, `/outbox/{id}`
  - Include schemas for all request/response DTOs
  - Include error response schemas (404, 403, 409, 500)
  - Tag endpoints by category (Review, Action, Outbox)

  **Must NOT do**:
    - Do NOT document existing Phase 6 endpoints (only new ones)
    - Do NOT include authentication details (reuse existing bearer token pattern)

  **Recommended Agent Profile**:
    - **Category**: `writing`
    - **Skills**: []

  **Parallelization**:
    - **Can Run In Parallel**: YES (with tasks 16, 18, 19, 20, 21)
    - **Parallel Group**: Wave 4
    - **Blocks**: None
    - **Blocked By**: Tasks 9, 12, 13 (handlers define the actual API)

  **Acceptance Criteria**:
    - [ ] OpenAPI YAML file exists
    - [ ] All new endpoints documented with paths, methods, schemas
    - [ ] Can be validated with `swagger-codegen validate -i docs/openapi/phase7.yaml`

  **Commit**: YES (groups with task 16)
    - Message: `docs(openapi): add Phase 7 API specification`
    - Files: `docs/openapi/phase7.yaml`

- [x] 18. **Audit Reconciliation Query** (`scripts/audit_reconcile.sql`)

  **What to do**:
  - Create SQL script to reconcile audit logs with proposal/outbox state:
    - Find proposals with apply_status="applied" but no audit_log action='proposal_executed'
    - Find outbox events with status="dispatched" but no audit_log action='outbox_dispatched'
    - Find review records without corresponding audit_log action='proposal_approved'/'proposal_rejected'
    - Output: report of discrepancies with counts
  - Script should be idempotent (safe to run multiple times)
  - Include comments explaining each check

  **Must NOT do**:
    - Do NOT modify data (read-only SELECT queries)
    - Do NOT require write permissions

  **Recommended Agent Profile**:
    - **Category**: `unspecified-low`
      - Reason: Simple SQL script
    - **Skills**: []

  **Parallelization**:
    - **Can Run In Parallel**: YES (with tasks 16, 17, 19, 20, 21)
    - **Parallel Group**: Wave 4
    - **Blocks**: None
    - **Blocked By**: Task 14 (AuditIntegration defines the action names)

  **References**:
    - `internal/audit/integration.go` - Audit action names (task 14)
    - `migrations/011_review_action_outbox.sql` - Schema (task 1)

  **Acceptance Criteria**:
    - [ ] Script runs without errors: `psql -f scripts/audit_reconcile.sql`
    - [ ] Output shows counts for each check category
    - [ ] No false positives on correctly synchronized data

  **QA Scenarios**:
    ```
    Scenario: Run reconciliation on clean data
      Tool: Bash (psql)
      Steps:
        1. psql -f scripts/audit_reconcile.sql
      Expected Result: All counts show 0 discrepancies
      Evidence: .sisyphus/evidence/task-18-reconcile-clean.txt
    ```

  **Commit**: YES
    - Message: `feat(audit): add reconciliation query script`
    - Files: `scripts/audit_reconcile.sql`

- [x] 19. **Security Test Suite** (`test/security/phase7_test.go`)

  **What to do**:
  - Create focused security tests verifying Phase 7 guardrails:
    1. **Unapproved execution**: Try execute proposal with status="proposed" → must return 403
    2. **Rejected execution**: Try execute proposal with status="rejected" → must return 403
    3. **Non-whitelist execution**: Try execute proposal with action_type="delete_database" → must return 403
    4. **Direct table write**: Verify action executor cannot write to raw/dwd/mart tables
    5. **Bypass review**: Verify no API endpoint allows status transition without review record
    6. **Audit tampering**: Verify audit logs cannot be deleted/modified via API
    7. **Dry-run safety**: Verify dry-run mode never creates outbox events
  - Each test should verify both the HTTP response AND the database state (no side effects)

  **Must NOT do**:
    - Do NOT test auth middleware (existing, out of scope)
    - Do NOT test SQL injection (use parameterized queries throughout)

  **Recommended Agent Profile**:
    - **Category**: `deep`
      - Reason: Security tests need thorough negative case coverage
    - **Skills**: [`test-driven-development`]

  **Parallelization**:
    - **Can Run In Parallel**: YES (with tasks 16, 17, 18, 20, 21)
    - **Parallel Group**: Wave 4
    - **Blocks**: None
    - **Blocked By**: Task 15 (integration tests define the baseline behavior)

  **References**:
    - `test/integration/phase7_test.go` - Integration tests (task 15)
    - `internal/action/apply_service.go` - Whitelist and dry-run logic (task 7)
    - `internal/review/service.go` - Review validation (task 6)

  **Acceptance Criteria**:
    - [ ] `go test ./test/security/... -run TestPhase7Security -v` → PASS (all 7 scenarios)
    - [ ] Test: Unapproved execution → 403, no audit execution log, no outbox
    - [ ] Test: Rejected execution → 403, no side effects
    - [ ] Test: Non-whitelist → 403
    - [ ] Test: Direct table write → 403 or no effect
    - [ ] Test: Bypass review → no such endpoint exists
    - [ ] Test: Audit tampering → no DELETE/PUT for audit logs
    - [ ] Test: Dry-run safety → no outbox events created

  **Commit**: YES
    - Message: `test(security): add Phase 7 security test suite`
    - Files: `test/security/phase7_test.go`

- [x] 20. **Worker Entrypoint Update** (`cmd/baxi-worker/main.go`)

  **What to do**:
  - Update worker CLI to start the dispatch worker (task 10)
  - Parse config: `WORKER_TICK_INTERVAL` (default 30s), `WORKER_BATCH_SIZE` (default 10)
  - Initialize dependencies: database connection, outbox repository, adapter registry
  - Start dispatch worker in background goroutine
  - Handle graceful shutdown (SIGTERM/SIGINT)
  - Add `--dry-run` flag to worker (overrides config, forces dry-run mode for all executions)

  **Must NOT do**:
    - Do NOT start other workers (keep existing behavior, only add dispatch worker)
    - Do NOT implement worker scaling (single instance only)

  **Recommended Agent Profile**:
    - **Category**: `unspecified-high`
      - Reason: CLI entrypoint with config parsing and signal handling
    - **Skills**: []

  **Parallelization**:
    - **Can Run In Parallel**: YES (with tasks 16, 17, 18, 19, 21)
    - **Parallel Group**: Wave 4
    - **Blocks**: None
    - **Blocked By**: Task 10 (DispatchWorker implementation)

  **References**:
    - `cmd/baxi-worker/main.go` - Existing entrypoint
    - `internal/worker/dispatch_worker.go` - DispatchWorker (task 10)
    - `internal/config/config.go` - Config loading

  **Acceptance Criteria**:
    - [ ] `go build ./cmd/baxi-worker` → success
    - [ ] Worker starts and logs "dispatch worker started"
    - [ ] Worker handles SIGTERM gracefully
    - [ ] `--dry-run` flag forces dry-run mode

  **QA Scenarios**:
    ```
    Scenario: Start worker
      Tool: Bash
      Steps:
        1. ./baxi-worker --config=config.yaml
      Expected Result: Logs "dispatch worker started", processes events
      Evidence: .sisyphus/evidence/task-20-worker-start.txt
    ```

  **Commit**: YES
    - Message: `feat(worker): update CLI entrypoint with dispatch worker`
    - Files: `cmd/baxi-worker/main.go`

- [x] 21. **Config & Environment Update** (`config/action_registry.yml`, `.env.example`)

  **What to do**:
  - Update `config/action_registry.yml` to match canonical action types:
    - Remove: `create_feishu_report`, `recommend_business_strategy`, `modify_business_policy`
    - Keep: `create_followup_task`, `notify_owner`, `export_report`
    - Add: `create_outbox_message`
    - Update all `requires_approval` to true (matching SQL CHECK constraint)
  - Update `.env.example` with new Phase 7 variables:
    - `ACTION_APPLY_DRY_RUN=true`
    - `ACTION_WHITELIST=create_followup_task,notify_owner,export_report,create_outbox_message`
    - `WORKER_TICK_INTERVAL=30s`
    - `WORKER_BATCH_SIZE=10`
    - `FEISHU_WEBHOOK_URL=` (empty = disabled)
    - `GITHUB_TOKEN=` (empty = disabled)
  - Update `internal/config/config.go` to parse new env vars

  **Must NOT do**:
    - Do NOT add config for non-whitelisted actions
    - Do NOT change existing config keys

  **Recommended Agent Profile**:
    - **Category**: `quick`
    - **Skills**: []

  **Parallelization**:
    - **Can Run In Parallel**: YES (with tasks 16, 17, 18, 19, 20)
    - **Parallel Group**: Wave 4
    - **Blocks**: None
    - **Blocked By**: Task 1 (migration defines canonical action types)

  **References**:
    - `config/action_registry.yml` - Existing config (to be updated)
    - `.env.example` - Existing env template
    - `internal/config/config.go` - Config struct

  **Acceptance Criteria**:
    - [ ] `config/action_registry.yml` matches canonical 4 action types
    - [ ] `.env.example` contains all new Phase 7 variables
    - [ ] `go test ./internal/config/...` → PASS
    - [ ] Config loads ACTION_APPLY_DRY_RUN as bool (default true)

  **Commit**: YES (groups with task 1)
    - Message: `config: reconcile action_registry.yml and add Phase 7 environment variables`
    - Files: `config/action_registry.yml`, `.env.example`, `internal/config/config.go`

---

## Final Verification Wave (MANDATORY — after ALL implementation tasks)

> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.

- [x] F1. **Plan Compliance Audit** — `oracle`
  **What to do**:
  1. Read the plan end-to-end
  2. For each "Must Have": verify implementation exists (read file, curl endpoint, run command)
  3. For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found
  4. Check evidence files exist in `.sisyphus/evidence/`
  5. Compare deliverables against plan

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: [`requesting-code-review`, `verification-before-completion`]

  **Acceptance Criteria**:
    - [ ] Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`
    - [ ] All Must Have deliverables exist in codebase
    - [ ] No Must NOT Have patterns found in codebase
    - [ ] Evidence directory contains screenshots/logs for all scenarios

- [x] F2. **Code Quality Review** — `unspecified-high`
  **What to do**:
  1. Run `go build ./...` — verify compilation
  2. Run `go vet ./...` — check for common issues
  3. Run `gofmt -l .` — check formatting
  4. Review all changed files for AI slop:
     - Empty `catch` / `recover` blocks
     - `fmt.Println` / `log.Printf` in production code (use `internal/logger` instead)
     - Commented-out code
     - Unused imports or variables
     - Overly generic names (`data`, `result`, `temp`)
     - Missing error handling
     - Hardcoded credentials or secrets
  5. Check transaction safety: every approve/execute has proper BEGIN/COMMIT/ROLLBACK

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: [`requesting-code-review`, `ai-slop-remover`]

  **Acceptance Criteria**:
    - [ ] `go build ./...` → PASS (zero errors)
    - [ ] `go vet ./...` → PASS (zero warnings)
    - [ ] `gofmt -l .` → no output (all files formatted)
    - [ ] No AI slop patterns found
    - [ ] All SQL queries use parameterized statements (no string concatenation)
    - [ ] All database transactions properly handle rollback on error

- [x] F3. **Real Manual QA** — `unspecified-high`
  **What to do**:
  1. Start application: `go run ./cmd/baxi-api`
  2. Start worker: `go run ./cmd/baxi-worker`
  3. Execute EVERY QA scenario from EVERY task — follow exact steps, capture evidence
  4. Test cross-task integration:
     - Approve proposal → execute → outbox created → worker dispatches
     - Reject proposal → verify cannot execute → verify no outbox created
     - Concurrent approve race → verify only one succeeds
  5. Test edge cases:
     - Empty review feedback
     - Execute already-applied proposal
     - Worker with no pending events
     - Cancel approved proposal
  6. Save all evidence to `.sisyphus/evidence/final-qa/`

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: [`verification-before-completion`, `test-scenarios`]

  **Acceptance Criteria**:
    - [ ] All 21 task QA scenarios executed and passed
    - [ ] Integration flow test passed (approve → execute → dispatch)
    - [ ] Edge case tests passed
    - [ ] Evidence directory contains ≥15 evidence files
    - [ ] Output: `Scenarios [N/N pass] | Integration [PASS] | Edge Cases [N tested] | VERDICT`

- [x] F4. **Scope Fidelity Check** — `deep`
  **What to do**:
  1. For each task: read "What to do" in plan, read actual diff (git log --stat)
  2. Verify 1:1 mapping — everything in spec was built (no missing), nothing beyond spec was built (no scope creep)
  3. Check "Must NOT do" compliance for each task
  4. Detect cross-task contamination: Task N touching Task M's files without dependency
  5. Flag unaccounted changes (files changed but not in any task's commit)
  6. Verify commit count: exactly 8 recommended commits

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: [`requesting-code-review`, `finishing-a-development-branch`]

  **Acceptance Criteria**:
    - [ ] Output: `Tasks [21/21 compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`
    - [ ] All tasks have corresponding committed code
    - [ ] No scope creep detected
    - [ ] No forbidden patterns found
    - [ ] Commit count matches recommended 8

---

## Commit Strategy

| # | Commit Message | Files | Pre-commit Verification |
|---|----------------|-------|------------------------|
| 1 | `feat(migration): add 011_review_action_status.sql` | `migrations/011_review_action_status.sql` | `make migrate-up` → verify CHECK constraints |
| 2 | `feat(review): add review domain models and repository` | `internal/review/` (domain + repo), `internal/testutil/` | `go test ./internal/review/...` → PASS |
| 3 | `feat(review): add review service with approve/reject/cancel` | `internal/review/service.go` | `go test ./internal/review/...` → PASS |
| 4 | `feat(action): add apply service, executor, and registry` | `internal/action/apply_service.go`, `executor.go`, `registry.go` | `go test ./internal/action/...` → PASS |
| 5 | `feat(outbox): add dispatch worker with retry and adapters` | `internal/outbox/dispatch_worker.go`, `internal/adapter/` | `go test ./internal/outbox/...` → PASS |
| 6 | `feat(api): add review and action execution HTTP handlers` | `internal/api/handler/review.go`, `action.go`, `outbox.go`, `router.go` | `go run ./cmd/baxi-api` → smoke test endpoints |
| 7 | `feat(audit): add audit integration and reconciliation` | `internal/audit/integration.go`, `internal/reconciliation/` | `go test ./internal/audit/...` → PASS |
| 8 | `test(integration): add end-to-end tests for Phase 7` | `test/integration/phase7_e2e_test.go` | `go test ./test/integration/...` → PASS |

---

## Success Criteria

### Must Have (All Required)
- [ ] Migration 011 extends `apply_status` CHECK to include `applying`, `applied`, `failed`
- [ ] Migration 011 adds CHECK constraint to `ai.review_record.verdict`
- [ ] `internal/review/` package implements Approve/Reject/Cancel with transaction safety
- [ ] `internal/action/apply_service.go` implements dry-run and white-list enforcement
- [ ] `internal/action/executor.go` handles all 4 white-listed action types
- [ ] `internal/outbox/dispatch_worker.go` implements `FOR UPDATE SKIP LOCKED` polling
- [ ] `internal/adapter/` contains stubbed Feishu and GitHub adapters (no real API calls)
- [ ] HTTP endpoints exist for: approve, reject, cancel, execute, status, dispatch, list
- [ ] Audit log is written for every approve, reject, cancel, execute, dispatch event
- [ ] Integration tests pass for all 6 end-to-end scenarios
- [ ] Security tests pass for all 5 security checks
- [ ] Design document exists at `docs/migration/phase-7-review-action-outbox-plan.md`

### Must NOT Have (Scope Lock)
- [ ] No real LLM calls (verified by searching for `llm.Provider` usage in action executor)
- [ ] No real Feishu/GitHub API calls in default config (verified by adapter config checks)
- [ ] No writes to `raw.*`, `dwd.*`, `mart.*` tables from action executor
- [ ] No unapproved proposal execution possible (verified by security test)
- [ ] No non-white-listed action execution possible (verified by security test)
- [ ] No batch operations for approve/reject (single proposal only)
- [ ] No UI/React changes
- [ ] No new action types beyond the 4 white-listed ones

### Verification Commands
```bash
# Compile check
go build ./...

# Unit tests
go test ./internal/review/... ./internal/action/... ./internal/outbox/... ./internal/audit/...

# Integration tests
go test ./test/integration/...

# Security tests
go test ./test/security/...

# Database migration check
make migrate-up && make migrate-status

# Smoke test API
curl -s http://localhost:8080/health | grep "ok"

# Audit log verification
psql $DATABASE_URL -c "SELECT COUNT(*) FROM audit.audit_log WHERE action LIKE 'proposal_%' OR action LIKE 'outbox_%';"
```

### Final Checklist
- [ ] All 21 tasks completed
- [ ] All 4 final verification agents approved
- [ ] All 8 commits made with descriptive messages
- [ ] No compilation errors
- [ ] No test failures
- [ ] Evidence directory populated with ≥15 files
- [ ] User has given explicit "okay" to complete


