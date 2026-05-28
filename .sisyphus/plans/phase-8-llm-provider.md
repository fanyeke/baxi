# Phase 8: Real LLM Provider Activation & Evaluation

## TL;DR

> **Goal**: Safely enable OpenAI-compatible LLM provider on the existing Phase 6/7 governance/decision pipeline. LLM can suggest decisions and generate action proposals, but CANNOT approve, apply, dispatch, query SQL, or write DB.
>
> **Deliverables**:
> - `internal/llm/openai_provider.go` — OpenAI-compatible provider implementation
> - `internal/llm/prompt_registry.go` — versioned prompt loading with hash tracking
> - `internal/eval/` — decision evaluation, comparison, and metrics
> - `internal/llm/audit.go` — LLM operation audit logging
> - `prompts/decision_system_v1.md` + `prompts/decision_user_v1.md`
> - `migrations/012_llm_activation_eval.sql` — eval schema + LLM audit columns
> - Extended API: `/llm/status`, `/compare`, `/replay`, `/evals`
> - Extended CLI: `llm status`, `decision compare`, `decision replay`, `decision evals`
>
> **Estimated Effort**: Large (~15 tasks across 5 waves)
> **Parallel Execution**: YES — 5 waves with max 7 concurrent tasks
> **Critical Path**: Config drift fix → Config wiring → Prompt registry → OpenAI provider → Provider selection → API/CLI → Eval module → Integration tests

---

## Context

### Original Request
Phase 8 of Baxi Go + PostgreSQL migration: enable real LLM provider on top of existing governance/decision/action pipeline with strict safety constraints.

### Interview Summary
**Key Discussions**:
- Prompt storage: filesystem (prompts/*.md), loaded at startup, hash computed from content
- OpenAI SDK: official `github.com/openai/openai-go/v3`
- Git: commit Phase 6/7 first, then Phase 8 on clean tree
- Eval storage: dedicated `ai.decision_eval_result` table

**Research Findings**:
- Phase 6/7 core architecture (DecisionProvider interface, DecisionEngine, validation, fallback) is 100% production-ready
- `config/llm_config.yml` exists but is NOT parsed by any Go code
- `internal/config/config.go` missing LLM fields entirely
- `server.go:131` and `cmd/baxi-cli/decision.go:118` hardcoded `NewRuleBasedProvider()`
- `prompts/` directory does not exist
- `ai.llm_decision` has `model_version` and `prompt_hash` columns but currently NULL
- `escalate_to_human` constant in `provider.go:68` vs `create_outbox_message` in DB CHECK constraint — DRIFT BUG
- `context_builder.go:163-168` hardcodes AllowedActions separately from ActionRegistry

### Metis Review
**Identified Gaps** (addressed in plan):
- **Critical**: `escalate_to_human` → `create_outbox_message` drift must be fixed before any Phase 8 code
- **Critical**: Config doesn't parse LLM settings — need to add fields + wiring
- **Critical**: ActionRegistry must be injected into ContextBuilder
- **Guardrails**: Never approve, never apply, never dispatch, fallback on ALL errors
- **Scope locks**: NO streaming, NO multi-turn, NO A/B testing, NO real execution, NO hot-reload
- **Edge cases**: 429 rate limit, empty choices, refusal, context length, 401 auth, missing prompt file, concurrent decisions
- **Assumptions validated**: DecisionProvider interface IS sufficient, schema validator covers structural safety, manual validation pipeline stays

---

## Work Objectives

### Core Objective
Implement OpenAI-compatible LLM provider that plugs into existing `DecisionProvider` interface, with prompt versioning, structured output validation, automatic fallback, decision evaluation, audit logging, and replay capability.

### Concrete Deliverables
- `internal/llm/openai_provider.go` + test
- `internal/llm/prompt_registry.go` + test
- `internal/llm/audit.go` + test
- `internal/eval/decision_eval.go` + test
- `internal/eval/comparison.go` + test
- `internal/eval/metrics.go` + test
- `prompts/decision_system_v1.md`
- `prompts/decision_user_v1.md`
- `migrations/012_llm_activation_eval.sql`
- Extended API handlers (`llm.go`, `decision.go` additions)
- Extended CLI commands (`decision.go` additions)
- `internal/config/config.go` LLM fields
- Provider factory/selector

### Definition of Done
- `go test ./...` passes
- `go vet ./...` passes
- All API endpoints respond correctly (verified via curl)
- LLM_ENABLED=false → rule-based fallback with audit trail
- Mock OpenAI tests cover: valid JSON, invalid JSON, forbidden action, timeout, 500
- No proposal has requires_human_review=false
- No Python/React/Pipeline code modified

### Must Have
- OpenAI-compatible provider implementing DecisionProvider
- Prompt registry with version + SHA-256 hash
- Schema validation of LLM output (existing + new rules)
- Automatic fallback to RuleBasedProvider on ANY error
- Audit logging of all LLM operations
- Eval result table and comparison logic
- API: /llm/status, /compare, /replay, /evals
- CLI: llm status, decision compare, decision replay, decision evals

### Must NOT Have (Guardrails)
- LLM approving proposals
- LLM applying proposals
- LLM dispatching outbox
- LLM querying SQL or writing DB directly
- Streaming responses from LLM
- Multi-turn conversations
- A/B testing of prompts
- Real action execution (NoOpExecutor stays)
- Prompt hot-reload
- Input guardrails / prompt injection defense
- Additional LLM providers (Anthropic, Gemini, etc.)
- Modifications to Python business logic
- Modifications to React frontend
- Modifications to Pipeline computation logic
- Modifications to YAML governance semantics

---

## Verification Strategy

### Test Decision
- **Infrastructure exists**: YES — `go test`, testify, testcontainers in testutil
- **Automated tests**: YES (TDD-style) — each task includes tests
- **Framework**: `go test` + testify + httptest for mock OpenAI server
- **Mock strategy**: httptest server simulating OpenAI API responses

### QA Policy
Every task MUST include agent-executed QA scenarios. Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **API/Backend**: Use Bash (curl) — Send requests, assert status + response fields
- **Library/Module**: Use Bash (go test) — Run tests, assert PASS
- **Database**: Use Bash (psql) — Query tables, assert row contents

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Foundation — cleanup + config, can start immediately):
├── T1: Fix action type drift (escalate_to_human → create_outbox_message)
├── T2: Add LLM config fields to internal/config/config.go
├── T3: Create prompt infrastructure (prompts/*.md + registry)
├── T4: Inject ActionRegistry into ContextBuilder
└── T5: Create migration 012_llm_activation_eval.sql

Wave 2 (Core Provider — after Wave 1):
├── T6: Implement OpenAICompatibleProvider
├── T7: Implement provider factory/selector
├── T8: Wire provider selection into server.go + CLI
└── T9: Add audit logging for LLM operations

Wave 3 (Eval + Comparison — after Wave 2):
├── T10: Implement decision evaluation module
├── T11: Implement decision comparison logic
├── T12: Implement metrics collection
└── T13: Add replay capability

Wave 4 (API + CLI — after Wave 2):
├── T14: Extend API handlers (llm status, compare, replay, evals)
└── T15: Extend CLI commands (llm status, decision compare/replay/evals)

Wave FINAL (After ALL tasks — 4 parallel reviews, then user okay):
├── F1: Plan compliance audit (oracle)
├── F2: Code quality review (unspecified-high)
├── F3: Real manual QA (unspecified-high)
└── F4: Scope fidelity check (deep)
-> Present results -> Get explicit user okay
```

### Dependency Matrix

| Task | Depends On | Blocks |
|------|-----------|--------|
| T1 | - | T4, T6 |
| T2 | - | T7, T8 |
| T3 | - | T6, T9 |
| T4 | T1 | T6 |
| T5 | - | T10, T11 |
| T6 | T1, T3, T4 | T7, T8, T9 |
| T7 | T2, T6 | T8 |
| T8 | T2, T6, T7 | F1-F4 |
| T9 | T3, T6 | F1-F4 |
| T10 | T5, T6 | T11, T12 |
| T11 | T5, T6, T10 | T12 |
| T12 | T10, T11 | F1-F4 |
| T13 | T6, T9 | F1-F4 |
| T14 | T6, T7, T8, T9, T10, T13 | F1-F4 |
| T15 | T6, T7, T8, T9, T10, T13 | F1-F4 |

### Agent Dispatch Summary

| Wave | Tasks | Agents |
|------|-------|--------|
| 1 | T1-T5 | quick (T1, T2, T5), unspecified-high (T3, T4) |
| 2 | T6-T9 | deep (T6, T7), unspecified-high (T8), quick (T9) |
| 3 | T10-T13 | unspecified-high (T10, T11, T12), deep (T13) |
| 4 | T14-T15 | unspecified-high (T14, T15) |
| FINAL | F1-F4 | oracle, unspecified-high, unspecified-high, deep |

---

## TODOs

- [x] T1. Fix action type drift: `escalate_to_human` → `create_outbox_message`

  **What to do**:
  - In `internal/llm/provider.go:68`, change constant `ActionTypeEscalateToHuman = "escalate_to_human"` to `ActionTypeCreateOutboxMessage = "create_outbox_message"`
  - In `internal/llm/rule_provider.go`, update all references from `ActionTypeEscalateToHuman` to `ActionTypeCreateOutboxMessage`
  - Verify `config/action_registry.yml` already has `create_outbox_message` (it does)
  - Run existing tests to confirm no breakage

  **Must NOT do**:
  - Do NOT modify DB CHECK constraints (migration 011 already did this)
  - Do NOT change any other action types

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - Reason: Simple string replacement across 2 files

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T2, T3, T4, T5)
  - **Blocks**: T4, T6
  - **Blocked By**: None

  **References**:
  - `internal/llm/provider.go:64-69` — ActionType constants
  - `internal/llm/rule_provider.go:42-44` — Usage in critical/high severity branches
  - `migrations/011_review_action_outbox.sql:24-27` — DB CHECK constraint (already correct)
  - `config/action_registry.yml` — Registry whitelist (already correct)

  **Acceptance Criteria**:
  - [ ] `grep -r "escalate_to_human" internal/` returns zero matches
  - [ ] `grep -r "create_outbox_message" internal/llm/` returns matches in provider.go and rule_provider.go
  - [ ] `go test ./internal/llm/...` passes

  **QA Scenarios**:
  ```
  Scenario: Action type constant updated
    Tool: Bash (grep + go test)
    Preconditions: Working tree has uncommitted changes
    Steps:
      1. grep -r "escalate_to_human" internal/llm/
      2. grep -r "create_outbox_message" internal/llm/
      3. go test ./internal/llm/...
    Expected Result: Zero matches for old string, matches for new string, all tests PASS
    Evidence: .sisyphus/evidence/t1-action-drift-fix.txt
  ```

  **Commit**: YES (Commit 1)
  - Message: `fix: action type drift escalate_to_human → create_outbox_message`
  - Files: `internal/llm/provider.go`, `internal/llm/rule_provider.go`
  - Pre-commit: `go test ./internal/llm/...`

- [x] T2. Add LLM config fields to `internal/config/config.go`

  **What to do**:
  - Add to `Config` struct:
    ```go
    LLMEnabled         bool
    LLMProvider        string
    LLMAPIKey          string
    LLMModel           string
    LLMAPIBase         string
    LLMTemperature     float64
    LLMMaxTokens       int
    LLMTimeoutSeconds  int
    LLMMaxRetries      int
    LLMFallbackEnabled bool
    LLMStoreRawOutput  bool
    ```
  - Parse from environment variables with sensible defaults:
    - `LLM_ENABLED` → default `false`
    - `LLM_PROVIDER` → default `"disabled"`
    - `LLM_API_KEY` → default `""`
    - `LLM_MODEL` → default `""`
    - `LLM_API_BASE` → default `"https://api.openai.com/v1"`
    - `LLM_TEMPERATURE` → default `0.2`
    - `LLM_MAX_TOKENS` → default `2048`
    - `LLM_TIMEOUT_SECONDS` → default `30`
    - `LLM_MAX_RETRIES` → default `2`
    - `LLM_FALLBACK_ENABLED` → default `true`
    - `LLM_STORE_RAW_OUTPUT` → default `true`
  - Update `.env.example` to match (it already has most of these, verify completeness)
  - Add validation: if `LLMEnabled=true` but `LLMAPIKey==""`, log WARNING but don't fail (graceful degradation to fallback)

  **Must NOT do**:
  - Do NOT parse `config/llm_config.yml` in Phase 8 (it remains dead code; env vars are the source of truth)
  - Do NOT make LLM_ENABLED default to true
  - Do NOT require LLM_API_KEY when LLM_ENABLED=false

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - Reason: Simple struct + env parsing

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T1, T3, T4, T5)
  - **Blocks**: T7, T8
  - **Blocked By**: None

  **References**:
  - `internal/config/config.go:1-64` — Existing config structure
  - `.env.example` — Existing LLM env vars
  - `config/llm_config.yml` — YAML config (DO NOT parse; keep as reference only)

  **Acceptance Criteria**:
  - [ ] `go test ./internal/config/...` passes
  - [ ] New test: `TestLLMConfigDefaults` — verify defaults when no env vars set
  - [ ] New test: `TestLLMConfigWithValues` — verify parsing with env vars set
  - [ ] `LLMEnabled` defaults to `false`
  - [ ] `LLMProvider` defaults to `"disabled"`

  **QA Scenarios**:
  ```
  Scenario: Default config (no LLM env vars)
    Tool: Bash (go test)
    Preconditions: Clean environment
    Steps:
      1. unset LLM_ENABLED LLM_PROVIDER LLM_API_KEY
      2. go test ./internal/config/ -run TestLLMConfigDefaults -v
    Expected Result: Test PASS, LLMEnabled=false, LLMProvider="disabled"
    Evidence: .sisyphus/evidence/t2-config-defaults.txt

  Scenario: Config with LLM enabled
    Tool: Bash (go test)
    Preconditions: Clean environment
    Steps:
      1. LLM_ENABLED=true LLM_PROVIDER=openai LLM_API_KEY=sk-test LLM_MODEL=gpt-4o-mini go test ./internal/config/ -run TestLLMConfigWithValues -v
    Expected Result: Test PASS, LLMEnabled=true, LLMProvider="openai", LLMModel="gpt-4o-mini"
    Evidence: .sisyphus/evidence/t2-config-enabled.txt
  ```

  **Commit**: YES (Commit 2)
  - Message: `feat: add LLM config fields and parser`
  - Files: `internal/config/config.go`, `.env.example`
  - Pre-commit: `go test ./internal/config/...`

- [x] T3. Create prompt infrastructure: `prompts/*.md` + `prompt_registry.go`

  **What to do**:
  - Create `prompts/decision_system_v1.md`:
    ```markdown
    # Decision System Prompt v1

    You are an e-commerce operations decision assistant. You can only generate structured JSON decisions based on the provided LLM-safe context.

    RULES:
    - You CANNOT request additional database access
    - You CANNOT generate actions not listed in allowed_actions
    - You CANNOT approve, execute, or dispatch actions
    - All action_proposals MUST have requires_human_review=true
    - Output MUST be valid JSON only

    OUTPUT SCHEMA:
    {
      "decision_type": "monitor_only|investigate|optimize|intervention|experiment",
      "severity": "low|medium|high|critical",
      "summary": "string (non-empty)",
      "rationale": ["string"],
      "recommended_actions": [
        {
          "action_type": "must be in allowed_actions",
          "priority": "low|medium|high|critical",
          "owner_role": "string",
          "payload": {}
        }
      ],
      "confidence": 0.0-1.0,
      "requires_human_review": true
    }
    ```
  - Create `prompts/decision_user_v1.md`:
    ```markdown
    # Decision User Prompt Template v1

    CONTEXT:
    {{.ContextJSON}}

    ALLOWED ACTIONS:
    {{range .AllowedActions}}- {{.}}
{{end}}

    FORBIDDEN ACTIONS:
    {{range .ForbiddenActions}}- {{.}}
{{end}}

    Generate a structured JSON decision following the system instructions.
    ```
  - Create `internal/llm/prompt_registry.go`:
    - `PromptRegistry` struct with methods:
      - `Load(promptID string) (*PromptTemplate, error)`
      - `Hash(promptID string) (string, error)` — SHA-256 of file content
      - `List() []string`
    - `PromptTemplate` struct: `ID`, `Version`, `SystemPrompt`, `UserTemplate`, `Hash`
    - Use `embed` package to embed prompt files into binary
    - Load all prompts at package init or first use
    - Compute SHA-256 hash on load

  **Must NOT do**:
  - Do NOT implement hot-reload (prompts load once at startup)
  - Do NOT store prompts in database
  - Do NOT hardcode prompt text in Go source

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
  - Reason: File I/O + template rendering + hash computation

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T1, T2, T4, T5)
  - **Blocks**: T6, T9
  - **Blocked By**: None

  **References**:
  - `internal/llm/provider.go:71-88` — DecisionOutput + RecommendedAction structs (output schema must match)
  - `internal/llm/schema_validator.go` — Validation rules that LLM output must pass
  - Go `embed` package docs — For embedding files into binary
  - `text/template` package — For user prompt template rendering

  **Acceptance Criteria**:
  - [ ] `go test ./internal/llm/...` passes
  - [ ] New test: `TestPromptRegistryLoad` — loads system + user prompts, verifies content
  - [ ] New test: `TestPromptHash` — computes consistent SHA-256 hash
  - [ ] `prompts/decision_system_v1.md` exists and contains JSON schema
  - [ ] `prompts/decision_user_v1.md` exists and contains Go template syntax

  **QA Scenarios**:
  ```
  Scenario: Prompt registry loads and hashes correctly
    Tool: Bash (go test)
    Preconditions: prompts/*.md files exist
    Steps:
      1. go test ./internal/llm/ -run TestPromptRegistryLoad -v
      2. go test ./internal/llm/ -run TestPromptHash -v
    Expected Result: Both tests PASS, hash is 64-char hex string
    Evidence: .sisyphus/evidence/t3-prompt-registry.txt

  Scenario: Prompt template renders with context
    Tool: Bash (go test)
    Preconditions: Registry initialized
    Steps:
      1. go test ./internal/llm/ -run TestPromptRender -v
    Expected Result: Rendered prompt contains context JSON and allowed actions list
    Evidence: .sisyphus/evidence/t3-prompt-render.txt
  ```

  **Commit**: YES (Commit 3)
  - Message: `feat: add versioned prompt templates and registry`
  - Files: `prompts/decision_system_v1.md`, `prompts/decision_user_v1.md`, `internal/llm/prompt_registry.go`
  - Pre-commit: `go test ./internal/llm/...`

- [x] T4. Inject ActionRegistry into ContextBuilder

  **What to do**:
  - Modify `internal/decision/context_builder.go`:
    - Add `ActionRegistry *action.Registry` field to `ContextBuilder` struct
    - In `BuildDecisionContext()`, replace hardcoded `AllowedActions` slice with dynamic lookup from registry:
      ```go
      // Instead of:
      // AllowedActions: []string{"create_followup_task", "notify_owner", "export_report", "create_outbox_message"},
      // Use:
      AllowedActions: b.registry.ListActionTypes(),
      ```
    - Update `NewContextBuilder()` constructor to accept `*action.Registry`
  - Update all call sites that construct `ContextBuilder`:
    - `internal/api/server.go` — pass registry instance
    - `cmd/baxi-cli/main.go` — pass registry instance
    - Any test files that construct ContextBuilder directly
  - Ensure `action.Registry` has `ListActionTypes() []string` method (add if missing)

  **Must NOT do**:
  - Do NOT change the action types themselves
  - Do NOT modify ActionRegistry loading logic
  - Do NOT break existing tests

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
  - Reason: Cross-package refactoring with multiple call sites

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Wave 1 tasks)
  - **Parallel Group**: Wave 1
  - **Blocks**: T6
  - **Blocked By**: T1 (action type names must be consistent first)

  **References**:
  - `internal/decision/context_builder.go:160-170` — Hardcoded AllowedActions
  - `internal/action/registry.go` — ActionRegistry implementation
  - `internal/api/server.go` — Where ContextBuilder is constructed
  - `cmd/baxi-cli/main.go` — CLI ContextBuilder construction

  **Acceptance Criteria**:
  - [ ] `go test ./internal/decision/...` passes
  - [ ] `go test ./internal/action/...` passes
  - [ ] `go test ./internal/api/...` passes
  - [ ] `grep -n "create_followup_task" internal/decision/context_builder.go` shows only registry usage, not hardcoded list
  - [ ] New test: `TestContextBuilderUsesRegistry` — verify allowed actions come from registry

  **QA Scenarios**:
  ```
  Scenario: ContextBuilder uses registry for allowed actions
    Tool: Bash (go test)
    Steps:
      1. go test ./internal/decision/ -run TestContextBuilderUsesRegistry -v
    Expected Result: Test PASS, AllowedActions matches registry entries
    Evidence: .sisyphus/evidence/t4-registry-injection.txt
  ```

  **Commit**: YES (Commit 4)
  - Message: `refactor: inject ActionRegistry into ContextBuilder`
  - Files: `internal/decision/context_builder.go`, `internal/api/server.go`, `cmd/baxi-cli/main.go`
  - Pre-commit: `go test ./internal/decision/... ./internal/action/...`

- [x] T5. Create migration `012_llm_activation_eval.sql`

  **What to do**:
  - Create `migrations/012_llm_activation_eval.sql` with:
    ```sql
    -- +goose Up
    -- +goose StatementBegin

    -- Extend ai.llm_decision with LLM audit fields
    ALTER TABLE ai.llm_decision
      ADD COLUMN IF NOT EXISTS provider TEXT,
      ADD COLUMN IF NOT EXISTS model TEXT,
      ADD COLUMN IF NOT EXISTS prompt_id TEXT,
      ADD COLUMN IF NOT EXISTS prompt_version TEXT,
      ADD COLUMN IF NOT EXISTS prompt_hash TEXT,
      ADD COLUMN IF NOT EXISTS context_hash TEXT,
      ADD COLUMN IF NOT EXISTS input_json JSONB,
      ADD COLUMN IF NOT EXISTS raw_output TEXT,
      ADD COLUMN IF NOT EXISTS parsed_output_json JSONB,
      ADD COLUMN IF NOT EXISTS validation_status TEXT,
      ADD COLUMN IF NOT EXISTS fallback_used BOOLEAN DEFAULT FALSE,
      ADD COLUMN IF NOT EXISTS fallback_reason TEXT,
      ADD COLUMN IF NOT EXISTS token_prompt INT,
      ADD COLUMN IF NOT EXISTS token_completion INT,
      ADD COLUMN IF NOT EXISTS latency_ms INT,
      ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ DEFAULT NOW();

    -- Decision eval result table
    CREATE TABLE IF NOT EXISTS ai.decision_eval_result (
      eval_id TEXT PRIMARY KEY,
      decision_case_id TEXT NOT NULL,
      llm_decision_id TEXT,
      eval_rule_id TEXT,
      eval_status TEXT NOT NULL,
      score NUMERIC(4,2),
      details_json JSONB,
      created_at TIMESTAMPTZ DEFAULT NOW()
    );

    -- Indexes
    CREATE INDEX IF NOT EXISTS idx_llm_decision_provider ON ai.llm_decision(provider, created_at);
    CREATE INDEX IF NOT EXISTS idx_llm_decision_fallback ON ai.llm_decision(fallback_used, created_at);
    CREATE INDEX IF NOT EXISTS idx_eval_result_case ON ai.decision_eval_result(decision_case_id);
    CREATE INDEX IF NOT EXISTS idx_eval_result_decision ON ai.decision_eval_result(llm_decision_id);

    -- +goose StatementEnd

    -- +goose Down
    -- +goose StatementBegin
    DROP INDEX IF EXISTS ai.idx_eval_result_decision;
    DROP INDEX IF EXISTS ai.idx_eval_result_case;
    DROP INDEX IF EXISTS ai.idx_llm_decision_fallback;
    DROP INDEX IF EXISTS ai.idx_llm_decision_provider;
    DROP TABLE IF EXISTS ai.decision_eval_result;
    ALTER TABLE ai.llm_decision
      DROP COLUMN IF EXISTS provider,
      DROP COLUMN IF EXISTS model,
      DROP COLUMN IF EXISTS prompt_id,
      DROP COLUMN IF EXISTS prompt_version,
      DROP COLUMN IF EXISTS prompt_hash,
      DROP COLUMN IF EXISTS context_hash,
      DROP COLUMN IF EXISTS input_json,
      DROP COLUMN IF EXISTS raw_output,
      DROP COLUMN IF EXISTS parsed_output_json,
      DROP COLUMN IF EXISTS validation_status,
      DROP COLUMN IF EXISTS fallback_used,
      DROP COLUMN IF EXISTS fallback_reason,
      DROP COLUMN IF EXISTS token_prompt,
      DROP COLUMN IF EXISTS token_completion,
      DROP COLUMN IF EXISTS latency_ms;
    -- +goose StatementEnd
    ```
  - Run `make migrate` to verify migration applies cleanly
  - Run `make migrate` again to verify idempotency (goose should report "no change")
  - Test rollback: apply UP, then apply DOWN, then UP again

  **Must NOT do**:
  - Do NOT modify existing column semantics (only ADD new columns)
  - Do NOT drop existing data
  - Do NOT change primary keys or foreign keys

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - Reason: SQL DDL only

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Wave 1 tasks)
  - **Parallel Group**: Wave 1
  - **Blocks**: T10, T11
  - **Blocked By**: None

  **References**:
  - `migrations/010_ai_tables_enhance.sql` — Pattern for adding columns to ai.llm_decision
  - `migrations/011_review_action_outbox.sql` — Pattern for CHECK constraints and indexes
  - Goose documentation for Up/Down migration format

  **Acceptance Criteria**:
  - [ ] `make migrate` succeeds without errors
  - [ ] `psql $DATABASE_URL -c "\d ai.llm_decision"` shows new columns
  - [ ] `psql $DATABASE_URL -c "\d ai.decision_eval_result"` shows new table
  - [ ] Migration is idempotent (running twice is safe)
  - [ ] Rollback works: UP → DOWN → UP succeeds

  **QA Scenarios**:
  ```
  Scenario: Migration applies cleanly
    Tool: Bash (make migrate + psql)
    Preconditions: Postgres running, previous migrations applied
    Steps:
      1. make migrate
      2. psql $DATABASE_URL -c "\d ai.llm_decision" | grep provider
      3. psql $DATABASE_URL -c "\d ai.decision_eval_result"
    Expected Result: Migration succeeds, new columns present, new table exists
    Evidence: .sisyphus/evidence/t5-migration-apply.txt

  Scenario: Migration is idempotent
    Tool: Bash (make migrate)
    Steps:
      1. make migrate
      2. make migrate
    Expected Result: Second run reports "no change" or equivalent
    Evidence: .sisyphus/evidence/t5-migration-idempotent.txt
  ```

  **Commit**: YES (Commit 5)
  - Message: `feat: add LLM activation and eval schema migration`
  - Files: `migrations/012_llm_activation_eval.sql`
  - Pre-commit: `make migrate`

- [x] T6. Implement `OpenAICompatibleProvider`

  **What to do**:
  - Create `internal/llm/openai_provider.go`:
    - `OpenAICompatibleProvider` struct implementing `DecisionProvider`
    - Fields: `client *openai.Client`, `model string`, `temperature float64`, `maxTokens int`, `timeout time.Duration`, `maxRetries int`, `promptRegistry *PromptRegistry`
    - `NewOpenAIProvider(cfg *config.Config, registry *PromptRegistry) (*OpenAICompatibleProvider, error)`
    - `GenerateDecision(ctx, LLMSafeContext) (*DecisionOutput, error)`:
      1. Load prompt templates from registry
      2. Render user prompt with context data
      3. Call OpenAI chat completions API via SDK
      4. Parse JSON from response content
      5. Unmarshal into `DecisionOutput`
      6. Return output + error
    - Handle errors: network, timeout, 401, 429, 500, empty choices, invalid JSON
    - Track token usage: `prompt_tokens`, `completion_tokens`
    - Track latency: measure time before/after API call
    - Use `context.WithTimeout` for timeout control
    - Set `Seed` parameter for reproducibility
  - Create `internal/llm/openai_provider_test.go`:
    - Use `httptest` server to mock OpenAI API
    - Test valid JSON response → correct DecisionOutput
    - Test invalid JSON response → error
    - Test 500 response → error
    - Test timeout → error
    - Test empty choices → error
    - Test refusal → error (detect "I'm sorry" or similar patterns)
    - Test rate limit (429) → error
  - Only THIS file imports `github.com/openai/openai-go/v3` (2-file rule)

  **Must NOT do**:
  - Do NOT import database packages (pgx, sql)
  - Do NOT implement streaming
  - Do NOT implement multi-turn
  - Do NOT call real OpenAI in tests
  - Do NOT hardcode API key in source

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []
  - Reason: Complex integration with external SDK, error handling, JSON parsing

  **Parallelization**:
  - **Can Run In Parallel**: NO (depends on T1, T3, T4)
  - **Parallel Group**: Wave 2
  - **Blocks**: T7, T8, T9
  - **Blocked By**: T1, T3, T4

  **References**:
  - `internal/llm/provider.go:6-8` — DecisionProvider interface
  - `internal/llm/provider.go:71-88` — DecisionOutput struct
  - `internal/llm/rule_provider.go` — Reference implementation pattern
  - `github.com/openai/openai-go/v3` — Official SDK
  - `internal/config/config.go` — Config struct (after T2)
  - `internal/llm/prompt_registry.go` — PromptRegistry (after T3)

  **Acceptance Criteria**:
  - [ ] `go test ./internal/llm/...` passes
  - [ ] Mock server test: valid JSON → DecisionOutput with correct fields
  - [ ] Mock server test: invalid JSON → error
  - [ ] Mock server test: timeout → error
  - [ ] Mock server test: 500 → error
  - [ ] Mock server test: empty choices → error
  - [ ] `go vet ./internal/llm/...` clean
  - [ ] Only `openai_provider.go` imports OpenAI SDK

  **QA Scenarios**:
  ```
  Scenario: Valid LLM response parsed correctly
    Tool: Bash (go test)
    Steps:
      1. go test ./internal/llm/ -run TestOpenAIProviderValidResponse -v
    Expected Result: Test PASS, DecisionOutput has decision_type, severity, confidence in [0,1], requires_human_review=true
    Evidence: .sisyphus/evidence/t6-valid-response.txt

  Scenario: Invalid JSON response returns error
    Tool: Bash (go test)
    Steps:
      1. go test ./internal/llm/ -run TestOpenAIProviderInvalidJSON -v
    Expected Result: Test PASS, error returned, no panic
    Evidence: .sisyphus/evidence/t6-invalid-json.txt

  Scenario: Timeout returns error
    Tool: Bash (go test)
    Steps:
      1. go test ./internal/llm/ -run TestOpenAIProviderTimeout -v
    Expected Result: Test PASS, context deadline exceeded error
    Evidence: .sisyphus/evidence/t6-timeout.txt
  ```

  **Commit**: YES (Commit 6)
  - Message: `feat: add OpenAI-compatible provider`
  - Files: `internal/llm/openai_provider.go`, `internal/llm/openai_provider_test.go`
  - Pre-commit: `go test ./internal/llm/...`

- [x] T7. Implement provider factory/selector

  **What to do**:
  - Create `internal/llm/provider_factory.go`:
    - `ProviderFactory` struct with config + dependencies
    - `CreateProvider() (DecisionProvider, error)` method:
      ```go
      switch cfg.LLMProvider {
      case "disabled", "":
        return NewDisabledProvider(), nil
      case "rule_based":
        return NewRuleBasedProvider(), nil
      case "openai", "openai_compatible":
        if cfg.LLMAPIKey == "" {
          return NewRuleBasedProvider(), nil // graceful degradation
        }
        return NewOpenAIProvider(cfg, registry), nil
      default:
        return nil, fmt.Errorf("unknown LLM provider: %s", cfg.LLMProvider)
      }
      ```
    - If `LLMEnabled=false`, always return `DisabledProvider` or `RuleBasedProvider` (regardless of LLM_PROVIDER value)
    - If `LLMEnabled=true` but `LLMAPIKey==""`, log warning and return `RuleBasedProvider`
  - Add test coverage for all provider selection paths
  - Update `internal/llm/provider.go` to export factory if needed

  **Must NOT do**:
  - Do NOT add Anthropic, Gemini, or other providers
  - Do NOT implement provider hot-swapping at runtime

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []
  - Reason: Config-driven factory with multiple paths

  **Parallelization**:
  - **Can Run In Parallel**: NO (depends on T2, T6)
  - **Parallel Group**: Wave 2
  - **Blocks**: T8
  - **Blocked By**: T2, T6

  **References**:
  - `internal/llm/provider.go` — Existing providers (Disabled, RuleBased)
  - `internal/llm/openai_provider.go` — OpenAI provider (after T6)
  - `internal/config/config.go` — Config struct (after T2)

  **Acceptance Criteria**:
  - [ ] `go test ./internal/llm/...` passes
  - [ ] Test: `LLM_ENABLED=false` → returns RuleBasedProvider regardless of LLM_PROVIDER
  - [ ] Test: `LLM_ENABLED=true, LLM_PROVIDER=openai, LLM_API_KEY=set` → returns OpenAIProvider
  - [ ] Test: `LLM_ENABLED=true, LLM_PROVIDER=openai, LLM_API_KEY=""` → returns RuleBasedProvider with warning
  - [ ] Test: `LLM_PROVIDER=unknown` → returns error

  **QA Scenarios**:
  ```
  Scenario: Factory returns correct provider based on config
    Tool: Bash (go test)
    Steps:
      1. go test ./internal/llm/ -run TestProviderFactory -v
    Expected Result: All subtests PASS for each config combination
    Evidence: .sisyphus/evidence/t7-provider-factory.txt
  ```

  **Commit**: YES (Commit 7)
  - Message: `feat: add provider factory and selector`
  - Files: `internal/llm/provider_factory.go`
  - Pre-commit: `go test ./internal/llm/...`

- [x] T8. Wire provider selection into `server.go` + CLI

  **What to do**:
  - Update `internal/api/server.go`:
    - Replace `llm.NewRuleBasedProvider()` with factory-based selection
    - Wire `ProviderFactory` into `DecisionEngine` construction
    - Pass `PromptRegistry` instance through the chain
  - Update `cmd/baxi-cli/decision.go`:
    - Replace `llm.NewRuleBasedProvider()` with factory-based selection
    - Support `--provider` flag for explicit override (optional)
  - Update `cmd/baxi-cli/main.go`:
    - Initialize `PromptRegistry` at startup
    - Pass registry to provider factory
  - Ensure both server and CLI use the same provider selection logic
  - Add `internal/api/handler/llm.go` for `/api/v1/llm/status` endpoint:
    ```go
    func (h *LLMHandler) Status(w http.ResponseWriter, r *http.Request) {
      httputil.JSON(w, http.StatusOK, map[string]interface{}{
        "enabled": cfg.LLMEnabled,
        "provider": cfg.LLMProvider,
        "model": cfg.LLMModel,
        "fallback_enabled": cfg.LLMFallbackEnabled,
        "raw_output_storage": cfg.LLMStoreRawOutput,
      })
    }
    ```

  **Must NOT do**:
  - Do NOT change existing API route registration pattern
  - Do NOT break existing handler tests

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
  - Reason: Cross-package wiring, handler updates

  **Parallelization**:
  - **Can Run In Parallel**: NO (depends on T2, T6, T7)
  - **Parallel Group**: Wave 2
  - **Blocks**: F1-F4
  - **Blocked By**: T2, T6, T7

  **References**:
  - `internal/api/server.go:131` — Hardcoded NewRuleBasedProvider
  - `cmd/baxi-cli/decision.go:118` — Hardcoded NewRuleBasedProvider
  - `internal/api/server.go` — Route registration pattern
  - `internal/api/handler/decision.go` — Existing handler pattern

  **Acceptance Criteria**:
  - [ ] `go test ./internal/api/...` passes
  - [ ] `go test ./cmd/baxi-cli/...` passes (if tests exist)
  - [ ] `go build ./cmd/baxi-api` succeeds
  - [ ] `go build ./cmd/baxi-cli` succeeds
  - [ ] API: `GET /api/v1/llm/status` returns correct config

  **QA Scenarios**:
  ```
  Scenario: API returns LLM status
    Tool: Bash (curl)
    Preconditions: Server running with LLM_ENABLED=false
    Steps:
      1. curl -s http://localhost:8080/api/v1/llm/status | jq .
    Expected Result: {"enabled":false,"provider":"disabled","fallback_enabled":true}
    Evidence: .sisyphus/evidence/t8-llm-status.json

  Scenario: CLI uses correct provider
    Tool: Bash (go run)
    Steps:
      1. LLM_ENABLED=false go run ./cmd/baxi-cli decision decide --case-id=test-case-1
    Expected Result: Uses RuleBasedProvider, decision generated
    Evidence: .sisyphus/evidence/t8-cli-provider.txt
  ```

  **Commit**: YES (Commit 7)
  - Message: `feat: wire provider selection into server and CLI`
  - Files: `internal/api/server.go`, `cmd/baxi-cli/decision.go`, `cmd/baxi-cli/main.go`, `internal/api/handler/llm.go`
  - Pre-commit: `go test ./internal/api/... && go build ./cmd/baxi-api && go build ./cmd/baxi-cli`

- [x] T9. Add audit logging for LLM operations

  **What to do**:
  - Create `internal/llm/audit.go`:
    - `LLMAuditLogger` interface:
      ```go
      type LLMAuditLogger interface {
        LogDecisionRequested(ctx context.Context, caseID, provider, model string)
        LogDecisionCompleted(ctx context.Context, caseID, provider, model string, latencyMs int64, tokenUsage TokenUsage)
        LogDecisionFailed(ctx context.Context, caseID, provider, model string, err error)
        LogDecisionValidationFailed(ctx context.Context, caseID string, errors []ValidationError)
        LogFallbackUsed(ctx context.Context, caseID string, reason string)
        LogDecisionReplayed(ctx context.Context, caseID, originalDecisionID string)
        LogEvalCompleted(ctx context.Context, caseID, evalID string)
      }
      ```
    - `TokenUsage` struct: `PromptTokens`, `CompletionTokens`, `TotalTokens`
    - `DBAuditLogger` implementation that writes to `audit.audit_log` table
    - `NoOpAuditLogger` for tests
  - Integrate audit logging into `DecisionEngine`:
    - Log `llm_decision_requested` before provider call
    - Log `llm_decision_completed` after successful provider call
    - Log `llm_decision_failed` on provider error
    - Log `llm_decision_validation_failed` on validation failure
    - Log `llm_fallback_used` when fallback triggers
  - Update `internal/decision/engine.go` to accept `LLMAuditLogger`
  - Audit metadata must include: case_id, provider, model, prompt_version, context_hash, fallback_used, validation_status, latency_ms, token_usage

  **Must NOT do**:
  - Do NOT write audit logs synchronously if it blocks decision flow (use goroutine or buffered channel)
  - Do NOT log sensitive data (API keys, full prompts with PII)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - Reason: Interface + DB insert, straightforward

  **Parallelization**:
  - **Can Run In Parallel**: NO (depends on T3, T6)
  - **Parallel Group**: Wave 2
  - **Blocks**: F1-F4
  - **Blocked By**: T3, T6

  **References**:
  - `internal/audit/integration.go` — Existing audit integration pattern
  - `internal/decision/engine.go` — Where to inject audit calls
  - `internal/repository/decision_repository.go` — DB access patterns

  **Acceptance Criteria**:
  - [ ] `go test ./internal/llm/...` passes
  - [ ] `go test ./internal/decision/...` passes
  - [ ] Test: `TestAuditLogDecisionRequested` — verify request event logged
  - [ ] Test: `TestAuditLogFallbackUsed` — verify fallback event logged

  **QA Scenarios**:
  ```
  Scenario: Audit log records LLM decision flow
    Tool: Bash (go test)
    Steps:
      1. go test ./internal/llm/ -run TestAuditLogDecisionRequested -v
      2. go test ./internal/llm/ -run TestAuditLogFallbackUsed -v
    Expected Result: Both tests PASS, audit events contain correct metadata
    Evidence: .sisyphus/evidence/t9-audit-logging.txt
  ```

  **Commit**: YES (Commit 8)
  - Message: `feat: add LLM audit logging`
  - Files: `internal/llm/audit.go`, `internal/decision/engine.go`
  - Pre-commit: `go test ./internal/llm/... ./internal/decision/...`

- [x] T10. Implement decision evaluation module

  **What to do**:
  - Create `internal/eval/decision_eval.go`:
    - `DecisionEvaluator` struct with eval rule config
    - `Evaluate(decisionCaseID, llmDecisionID string, output *llm.DecisionOutput) (*EvalResult, error)`
    - Eval dimensions (from `config/decision_eval_rules.yml`):
      - `schema_validity` — did output pass schema validation?
      - `governance_compliance` — are actions in allowed_actions?
      - `action_safety` — are forbidden_actions absent?
      - `human_review_required` — is requires_human_review=true?
      - `context_grounding` — does summary reference context?
      - `rationale_completeness` — is rationale non-empty and relevant?
    - `EvalResult` struct:
      ```go
      type EvalResult struct {
        EvalID         string
        DecisionCaseID string
        LLMDecisionID  string
        EvalRuleID     string
        EvalStatus     string // pass / fail / partial
        Score          float64
        DetailsJSON    json.RawMessage
        CreatedAt      time.Time
      }
      ```
    - `SaveResult(ctx, *EvalResult) error` — persist to `ai.decision_eval_result`
  - Create `internal/eval/decision_eval_test.go`:
    - Test valid decision → all eval dimensions pass
    - Test invalid action type → governance_compliance fails
    - Test requires_human_review=false → human_review_required fails
    - Test empty rationale → rationale_completeness fails

  **Must NOT do**:
  - Do NOT use LLM-as-judge for eval (deterministic metrics only)
  - Do NOT implement complex NLP for rationale analysis (simple heuristics only)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
  - Reason: Multi-dimensional scoring logic + DB persistence

  **Parallelization**:
  - **Can Run In Parallel**: NO (depends on T5, T6)
  - **Parallel Group**: Wave 3
  - **Blocks**: T11, T12
  - **Blocked By**: T5, T6

  **References**:
  - `config/decision_eval_rules.yml` — Eval dimensions and weights
  - `internal/llm/schema_validator.go` — Validation rules to reference
  - `internal/llm/provider.go:71-88` — DecisionOutput structure
    - `migrations/012_llm_activation_eval.sql` — ai.decision_eval_result schema

  **Acceptance Criteria**:
  - [ ] `go test ./internal/eval/...` passes
  - [ ] Test: valid decision → score >= pass_threshold
  - [ ] Test: forbidden action → governance_compliance fails
  - [ ] Test: requires_human_review=false → human_review_required fails
  - [ ] Eval results persist to ai.decision_eval_result

  **QA Scenarios**:
  ```
  Scenario: Valid decision passes all eval dimensions
    Tool: Bash (go test)
    Steps:
      1. go test ./internal/eval/ -run TestEvalValidDecision -v
    Expected Result: Test PASS, score >= 6.0, status="pass"
    Evidence: .sisyphus/evidence/t10-eval-valid.txt

  Scenario: Invalid decision fails eval
    Tool: Bash (go test)
    Steps:
      1. go test ./internal/eval/ -run TestEvalInvalidDecision -v
    Expected Result: Test PASS, score < 6.0, status="fail"
    Evidence: .sisyphus/evidence/t10-eval-invalid.txt
  ```

  **Commit**: YES (Commit 9)
  - Message: `feat: add decision evaluation and comparison`
  - Files: `internal/eval/decision_eval.go`, `internal/eval/decision_eval_test.go`
  - Pre-commit: `go test ./internal/eval/...`

- [x] T11. Implement decision comparison logic

  **What to do**:
  - Create `internal/eval/comparison.go`:
    - `DecisionComparison` struct:
      ```go
      type DecisionComparison struct {
        DecisionCaseID       string
        LLMDecisionType      string
        RuleDecisionType     string
        DecisionTypeMatch    bool
        SeverityMatch        bool
        ActionOverlap        float64 // Jaccard index of action types
        LLMValid             bool
        RuleValid            bool
        ConfidenceDiff       float64
        LLMRequiresReview    bool
        RuleRequiresReview   bool
        ComparisonJSON       json.RawMessage
        CreatedAt            time.Time
      }
      ```
    - `Compare(ctx, caseID string, llmDecision, ruleDecision *llm.DecisionOutput) (*DecisionComparison, error)`
    - Comparison dimensions:
      - decision_type consistency
      - severity consistency
      - action overlap (Jaccard index)
      - confidence difference
      - requires_human_review consistency
    - Save comparison result (can reuse ai.decision_eval_result with eval_rule_id="comparison")
  - Create `internal/eval/comparison_test.go`:
    - Test identical decisions → all match
    - Test different decision types → mismatch flagged
    - Test different actions → overlap < 1.0

  **Must NOT do**:
  - Do NOT implement semantic comparison of rationale text
  - Do NOT implement complex NLP similarity

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
  - Reason: Structured comparison logic

  **Parallelization**:
  - **Can Run In Parallel**: NO (depends on T5, T6, T10)
  - **Parallel Group**: Wave 3
  - **Blocks**: T12
  - **Blocked By**: T5, T6, T10

  **References**:
  - `internal/llm/provider.go:71-88` — DecisionOutput fields to compare
  - `internal/llm/rule_provider.go` — Rule-based output structure
  - `internal/eval/decision_eval.go` — Eval patterns to follow

  **Acceptance Criteria**:
  - [ ] `go test ./internal/eval/...` passes
  - [ ] Test: identical decisions → 100% match
  - [ ] Test: different types → type mismatch flagged
  - [ ] Test: different actions → overlap < 1.0

  **QA Scenarios**:
  ```
  Scenario: Compare identical decisions
    Tool: Bash (go test)
    Steps:
      1. go test ./internal/eval/ -run TestCompareIdentical -v
    Expected Result: Test PASS, all match=true, overlap=1.0
    Evidence: .sisyphus/evidence/t11-compare-identical.txt

  Scenario: Compare different decisions
    Tool: Bash (go test)
    Steps:
      1. go test ./internal/eval/ -run TestCompareDifferent -v
    Expected Result: Test PASS, type_match=false, overlap < 1.0
    Evidence: .sisyphus/evidence/t11-compare-different.txt
  ```

  **Commit**: YES (Commit 9 — grouped with T10)
  - Message: `feat: add decision evaluation and comparison`
  - Files: `internal/eval/comparison.go`, `internal/eval/comparison_test.go`
  - Pre-commit: `go test ./internal/eval/...`

- [x] T12. Implement metrics collection

  **What to do**:
  - Create `internal/eval/metrics.go`:
    - `MetricsCollector` struct
    - Methods:
      - `RecordDecision(provider string, latencyMs int64, tokenUsage TokenUsage)`
      - `RecordFallback(reason string)`
      - `RecordValidationFailure(errors []ValidationError)`
      - `RecordApproval(caseID string, approved bool)`
      - `GetFallbackRate() float64`
      - `GetValidationFailureRate() float64`
      - `GetAverageLatency() float64`
      - `GetApprovalRate() float64`
    - Store metrics in memory (simple counters) for Phase 8
    - Expose via API endpoint `/api/v1/llm/metrics`
  - Create `internal/eval/metrics_test.go`:
    - Test recording and retrieval
    - Test rate calculations

  **Must NOT do**:
  - Do NOT implement persistent metrics storage (in-memory only for Phase 8)
  - Do NOT implement time-series aggregation

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
  - Reason: Simple counter logic

  **Parallelization**:
  - **Can Run In Parallel**: NO (depends on T10, T11)
  - **Parallel Group**: Wave 3
  - **Blocks**: F1-F4
  - **Blocked By**: T10, T11

  **References**:
  - `internal/eval/decision_eval.go` — Eval patterns
  - `internal/eval/comparison.go` — Comparison patterns

  **Acceptance Criteria**:
  - [ ] `go test ./internal/eval/...` passes
    - [ ] API: `GET /api/v1/llm/metrics` returns current metrics

  **QA Scenarios**:
  ```
  Scenario: Metrics endpoint returns data
    Tool: Bash (curl)
    Preconditions: Server running, some decisions made
    Steps:
      1. curl -s http://localhost:8080/api/v1/llm/metrics | jq .
    Expected Result: JSON with fallback_rate, validation_failure_rate, avg_latency, approval_rate
    Evidence: .sisyphus/evidence/t12-metrics.json
  ```

  **Commit**: YES (Commit 10)
  - Message: `feat: add decision metrics and replay`
  - Files: `internal/eval/metrics.go`, `internal/eval/metrics_test.go`
  - Pre-commit: `go test ./internal/eval/...`

- [x] T13. Add replay capability

  **What to do**:
  - Create `internal/eval/replay.go`:
    - `ReplayService` struct with repository dependency
    - `Replay(ctx, caseID string, dryRun bool) (*ReplayResult, error)`:
      1. Fetch original decision from `ai.llm_decision` by case_id
      2. Extract: context_hash, prompt_version, model, input_json
      3. Rebuild `LLMSafeContext` from stored input
      4. If `dryRun=true`: return what WOULD be generated without calling provider
      5. If `dryRun=false`: call provider with same context + same prompt version
      6. Compare new output with original
      7. Return `ReplayResult` with both decisions + diff
    - `ReplayResult` struct:
      ```go
      type ReplayResult struct {
        OriginalDecision  *llm.DecisionOutput
        ReplayedDecision  *llm.DecisionOutput
        ContextHash       string
        PromptVersion     string
        Model             string
        DryRun            bool
        DiffJSON          json.RawMessage
      }
      ```
    - Log replay event to audit log
    - If `dryRun=false`, still require `requires_human_review=true` for any new proposals
  - Create `internal/eval/replay_test.go`:
    - Test dry-run replay → no provider call
    - Test actual replay → provider called with same context
    - Test missing original decision → error

  **Must NOT do**:
  - Do NOT auto-approve replayed decisions
  - Do NOT auto-apply replayed proposals
  - Do NOT skip validation on replayed output

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []
  - Reason: State reconstruction + conditional execution

  **Parallelization**:
  - **Can Run In Parallel**: NO (depends on T6, T9)
  - **Parallel Group**: Wave 3
  - **Blocks**: F1-F4
  - **Blocked By**: T6, T9

  **References**:
  - `internal/decision/engine.go` — Decision flow to replicate
  - `internal/llm/audit.go` — Audit logging (after T9)
  - `internal/repository/decision_repository.go` — DB access for original decisions

  **Acceptance Criteria**:
  - [ ] `go test ./internal/eval/...` passes
  - [ ] Test: dry-run replay → no provider call, returns original data
  - [ ] Test: actual replay → provider called, output compared
  - [ ] Test: missing original → error

  **QA Scenarios**:
  ```
  Scenario: Dry-run replay returns original decision
    Tool: Bash (go test)
    Steps:
      1. go test ./internal/eval/ -run TestReplayDryRun -v
    Expected Result: Test PASS, no provider call, original decision returned
    Evidence: .sisyphus/evidence/t13-replay-dryrun.txt

  Scenario: Actual replay calls provider
    Tool: Bash (go test)
    Steps:
      1. go test ./internal/eval/ -run TestReplayActual -v
    Expected Result: Test PASS, provider called with matching context
    Evidence: .sisyphus/evidence/t13-replay-actual.txt
  ```

  **Commit**: YES (Commit 10 — grouped with T12)
  - Message: `feat: add decision metrics and replay`
  - Files: `internal/eval/replay.go`, `internal/eval/replay_test.go`
  - Pre-commit: `go test ./internal/eval/...`

- [x] T14. Extend API handlers (llm status, compare, replay, evals)

  **What to do**:
  - Create/extend `internal/api/handler/llm.go`:
    - `GET /api/v1/llm/status` → returns LLM config status
    - `GET /api/v1/llm/metrics` → returns metrics from MetricsCollector
  - Extend `internal/api/handler/decision.go`:
    - `POST /api/v1/decisions/cases/{case_id}/decide/llm` → explicit LLM decision (same as /decide but documents LLM path)
    - `POST /api/v1/decisions/cases/{case_id}/compare` → compare LLM vs rule-based decisions
    - `POST /api/v1/decisions/cases/{case_id}/replay` → replay decision with dry-run default
    - `GET /api/v1/decisions/cases/{case_id}/llm-decisions` → list LLM decisions for case
    - `GET /api/v1/decisions/cases/{case_id}/evals` → list eval results for case
  - Add DTOs to `internal/api/dto/decision.go` for new endpoints
  - Register new routes in `internal/api/server.go`
  - Add tests for new handlers

  **Must NOT do**:
  - Do NOT modify existing handler signatures (only add new methods)
  - Do NOT break existing route tests

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
  - Reason: Handler implementation + route registration + DTOs + tests

  **Parallelization**:
  - **Can Run In Parallel**: NO (depends on T6, T7, T8, T9, T10, T13)
  - **Parallel Group**: Wave 4
  - **Blocks**: F1-F4
  - **Blocked By**: T6, T7, T8, T9, T10, T13

  **References**:
  - `internal/api/handler/decision.go` — Existing handler pattern
  - `internal/api/dto/decision.go` — Existing DTOs
  - `internal/api/server.go` — Route registration
  - `internal/api/handler/decision_test.go` — Test patterns

  **Acceptance Criteria**:
  - [ ] `go test ./internal/api/...` passes
  - [ ] All new endpoints respond with correct status codes
  - [ ] `GET /api/v1/llm/status` returns config
  - [ ] `POST /api/v1/decisions/cases/{id}/compare` returns comparison result
  - [ ] `POST /api/v1/decisions/cases/{id}/replay` returns replay result

  **QA Scenarios**:
  ```
  Scenario: LLM status endpoint
    Tool: Bash (curl)
    Preconditions: Server running
    Steps:
      1. curl -s http://localhost:8080/api/v1/llm/status | jq .
    Expected Result: 200 OK, JSON with enabled, provider, model fields
    Evidence: .sisyphus/evidence/t14-llm-status.json

  Scenario: Compare endpoint
    Tool: Bash (curl)
    Preconditions: Case exists with both LLM and rule-based decisions
    Steps:
      1. curl -s -X POST http://localhost:8080/api/v1/decisions/cases/{id}/compare | jq .
    Expected Result: 200 OK, JSON with decision_type_match, severity_match, action_overlap
    Evidence: .sisyphus/evidence/t14-compare.json
  ```

  **Commit**: YES (Commit 11)
  - Message: `feat: add LLM status and decision API endpoints`
  - Files: `internal/api/handler/llm.go`, `internal/api/handler/decision.go`, `internal/api/dto/decision.go`, `internal/api/server.go`
  - Pre-commit: `go test ./internal/api/...`

- [x] T15. Extend CLI commands (llm status, decision compare/replay/evals)

  **What to do**:
  - Extend `cmd/baxi-cli/decision.go`:
    - Add `decision compare --case-id=X` command → calls compare API
    - Add `decision replay --case-id=X [--dry-run=true]` command → calls replay API
    - Add `decision evals --case-id=X` command → lists eval results
  - Create `cmd/baxi-cli/llm.go`:
    - Add `llm status` command → calls status API
    - Add `llm metrics` command → calls metrics API
  - Update `cmd/baxi-cli/main.go` to register new commands
  - Update `Makefile` with new targets:
    ```makefile
    llm-status:
      go run ./cmd/baxi-cli llm status

    decision-llm:
      go run ./cmd/baxi-cli decision llm --case-id $(CASE_ID)

    decision-compare:
      go run ./cmd/baxi-cli decision compare --case-id $(CASE_ID)

    decision-replay:
      go run ./cmd/baxi-cli decision replay --case-id $(CASE_ID) --dry-run=true

    decision-evals:
      go run ./cmd/baxi-cli decision evals --case-id $(CASE_ID)
    ```

  **Must NOT do**:
  - Do NOT modify existing CLI commands
  - Do NOT require new dependencies beyond existing HTTP client

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
  - Reason: CLI command implementation + Makefile updates

  **Parallelization**:
  - **Can Run In Parallel**: NO (depends on T6, T7, T8, T9, T10, T13)
  - **Parallel Group**: Wave 4
  - **Blocks**: F1-F4
  - **Blocked By**: T6, T7, T8, T9, T10, T13

  **References**:
  - `cmd/baxi-cli/decision.go` — Existing CLI command pattern
  - `cmd/baxi-cli/main.go` — Command registration
  - `Makefile` — Existing targets

  **Acceptance Criteria**:
  - [ ] `go build ./cmd/baxi-cli` succeeds
  - [ ] `go run ./cmd/baxi-cli llm status` works
  - [ ] `go run ./cmd/baxi-cli decision compare --case-id=test` works
  - [ ] `go run ./cmd/baxi-cli decision replay --case-id=test --dry-run=true` works
  - [ ] New Makefile targets work

  **QA Scenarios**:
  ```
  Scenario: CLI llm status
    Tool: Bash (go run)
    Steps:
      1. go run ./cmd/baxi-cli llm status
    Expected Result: Prints JSON status (enabled, provider, model)
    Evidence: .sisyphus/evidence/t15-cli-llm-status.txt

  Scenario: CLI decision compare
    Tool: Bash (go run)
    Steps:
      1. go run ./cmd/baxi-cli decision compare --case-id=test-case-1
    Expected Result: Prints comparison result JSON
    Evidence: .sisyphus/evidence/t15-cli-compare.txt
  ```

  **Commit**: YES (Commit 12)
  - Message: `feat: add LLM and eval CLI commands`
  - Files: `cmd/baxi-cli/llm.go`, `cmd/baxi-cli/decision.go`, `cmd/baxi-cli/main.go`, `Makefile`
  - Pre-commit: `go build ./cmd/baxi-cli`

---

## Final Verification Wave

### F1. Plan Compliance Audit — `oracle`
Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, curl endpoint, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in `.sisyphus/evidence/`. Compare deliverables against plan.

### F2. Code Quality Review — `unspecified-high`
Run `go test ./...`, `go vet ./...`. Review all changed files for: `as any`/`@ts-ignore`, empty catches, `console.log` in prod, commented-out code, unused imports. Check AI slop: excessive comments, over-abstraction, generic names.

### F3. Real Manual QA — `unspecified-high`
Start from clean state. Execute EVERY QA scenario from EVERY task — follow exact steps, capture evidence. Test cross-task integration. Test edge cases: empty state, invalid input, rapid actions. Save to `.sisyphus/evidence/final-qa/`.

### F4. Scope Fidelity Check — `deep`
For each task: read "What to do", read actual diff (git log/diff). Verify 1:1 — everything in spec was built, nothing beyond spec was built. Check "Must NOT do" compliance. Detect cross-task contamination.

---

## Commit Strategy

| # | Commit | Files | Pre-commit |
|---|--------|-------|------------|
| 1 | `fix: action type drift escalate_to_human → create_outbox_message` | provider.go, rule_provider.go | `go test ./internal/llm/...` |
| 2 | `feat: add LLM config fields and parser` | config.go, .env.example | `go test ./internal/config/...` |
| 3 | `feat: add versioned prompt templates and registry` | prompts/*.md, prompt_registry.go | `go test ./internal/llm/...` |
| 4 | `refactor: inject ActionRegistry into ContextBuilder` | context_builder.go | `go test ./internal/decision/...` |
| 5 | `feat: add LLM activation and eval schema migration` | 012_llm_activation_eval.sql | `make migrate` |
| 6 | `feat: add OpenAI-compatible provider` | openai_provider.go | `go test ./internal/llm/...` |
| 7 | `feat: add provider factory and selector` | provider.go additions, server.go, CLI | `go test ./...` |
| 8 | `feat: add LLM audit logging` | audit.go | `go test ./internal/llm/...` |
| 9 | `feat: add decision evaluation and comparison` | eval/*.go | `go test ./internal/eval/...` |
| 10 | `feat: add decision metrics and replay` | metrics.go, replay.go | `go test ./internal/eval/...` |
| 11 | `feat: add LLM status and decision API endpoints` | api/handler/*.go | `go test ./internal/api/...` |
| 12 | `feat: add LLM and eval CLI commands` | cmd/baxi-cli/*.go | `go build ./cmd/baxi-cli` |
| 13 | `test: add integration tests for LLM decision flow` | test/*_test.go | `go test ./test/...` |
| 14 | `docs: add phase 8 runbook and API docs` | docs/*.md | - |

---

## Success Criteria

### Verification Commands
```bash
# 1. Database migration
docker compose up -d postgres
export DATABASE_URL="postgres://baxi:baxi_dev@localhost:5432/baxi?sslmode=disable"
make migrate

# 2. LLM disabled (default)
export LLM_ENABLED=false
make api
curl -s http://localhost:8080/api/v1/llm/status | jq .
# Expected: {"enabled":false,"provider":"disabled","fallback_enabled":true}

# 3. Decision with LLM disabled → fallback
curl -s -X POST http://localhost:8080/api/v1/decisions/cases/{case_id}/decide/llm \
  -H "Authorization: Bearer $API_BEARER_TOKEN" | jq .
# Expected: fallback_used=true, provider="rule_based"

# 4. Run all tests
go test ./...
go vet ./...

# 5. Pipeline regression
make pipeline DATA_DIR=./data/raw
make pipeline-compare
make api-compare
make governance-check

# 6. Verify no Python/React/Pipeline changes
git diff --name-only | grep -E "^(api/|services/|adapters/|frontend/|pipeline/)" || echo "OK: no Python/React/Pipeline changes"
```

### Final Checklist
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent
- [ ] All tests pass
- [ ] `go vet ./...` clean
- [ ] No Python/React/Pipeline code modified
- [ ] All API endpoints respond correctly
- [ ] LLM_ENABLED=false → safe fallback
- [ ] Audit trail complete for all LLM operations
- [ ] Eval table populated and queryable
- [ ] Prompt hash tracked in ai.llm_decision
