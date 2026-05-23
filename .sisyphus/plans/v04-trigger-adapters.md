# v0.4: Trigger Adapters & Feishu CLI E2E

## TL;DR

> **Quick Summary**: Transform event_outbox from simulation (v0.2) to real external dispatch. Build adapter framework, extend FeishuClient with IM send, wire channel routing rules, and prove E2E: event_outbox → adapter → real Feishu message → status writeback.
> 
> **Deliverables**:
> - Schema migration: 4 new columns on event_outbox (dispatch_attempts, last_dispatch_at, external_ref, adapter_name)
> - Config: adapter_registry.yml
> - Adapter framework: 4 adapters (Feishu, GitHubIssue, LocalCLI, Manual)
> - Dispatcher: db_dispatch_outbox.py (dry-run/apply/channel-filter)
> - FeishuClient extension: send_message() method with dry_run support
> - CLI probe: feishu_cli_probe.py capability detection
> - E2E test + acceptance report
> 
> **Estimated Effort**: Medium (2-3 days)
> **Parallel Execution**: YES - Waves 1-3 parallelizable
> **Critical Path**: Schema → Adapter Base + FeishuClient → Dispatcher → E2E

---

## Context

### Original Request
User wants v0.4 to shift from "数据库后台与规则引擎" to "外部触发与飞书 CLI 端到端验证". Core goal: prove event_outbox can trigger real external actions through an adapter layer, with Feishu as the first real channel.

### Interview Summary
**Key Discussions**:
- Existing `db_trigger_simulator.py` generates mock payloads but makes no real external calls
- `channel_routing_rules.yml` exists but not wired into rule engine (hardcoded `local_cli`)
- Bulk sync via `sync_feishu_bitable.py` works well — don't touch it
- lark-cli is Go binary, not Python-importable → use `lark-oapi` SDK for dispatch
- Oracle review confirmed: extend FeishuClient over CLI subprocess

**Research Findings**:
- `lark-oapi` SDK: auto token refresh, typed models, `im/v1/messages` endpoint supported
- lark-cli capabilities: IM, Base, Docs all available via shortcuts or raw API
- event_outbox payload format: rule_id, event_id, metric_name, current_value, baseline_value, change_rate, severity, owner_role
- 5 alert rules, 6 dimensional routing rules defined

### Metis Review
**Identified Gaps** (addressed):
- Payload format contract: adapters parse event_outbox.payload_json directly, no shared transformer
- Idempotency: only status='pending' AND dispatch_attempts<3 processed; optimistic lock before dispatch
- Partial failure: per-event status writeback, no batch rollback
- Concurrent safety: UPDATE status='dispatching' as optimistic lock before processing
- Error taxonomy: retry on 429/network timeout; terminal on 4xx/permanent failures
- Retry max: dispatch_attempts >= 3 = terminal, mark failed
- Invocation: standalone CLI script, manual/cron trigger (no daemon in v0.4)

---

## Work Objectives

### Core Objective
Build a dispatch system that reads pending events from event_outbox, routes them to the correct adapter, executes real external actions (Feishu IM messages), and writes back status with audit trail.

### Concrete Deliverables
- `sql/migrations/005_dispatch_adapters.sql` — ALTER TABLE event_outbox
- `config/adapter_registry.yml` — 4 adapter definitions
- `scripts/feishu_cli_probe.py` — capability detection
- `scripts/adapters/__init__.py` + `base.py` — ChannelAdapter ABC
- `scripts/adapters/feishu_adapter.py` — Feishu IM dispatch
- `scripts/adapters/github_issue_adapter.py` — payload generation (dry-run only)
- `scripts/adapters/local_cli_adapter.py` — local command suggestion
- `scripts/adapters/manual_adapter.py` — mark for human review
- `scripts/db_dispatch_outbox.py` — dispatch orchestrator
- `scripts/feishu_client.py` — ADD send_message() method (existing file, additive only)
- `requirements.txt` — ADD lark-oapi
- `tests/test_adapter_framework.py` — all adapters
- `tests/test_db_dispatch_outbox.py` — dispatch flow
- `reports/dispatch_e2e_report.md` — E2E acceptance

### Definition of Done
- [ ] Schema migration applied, all 4 columns verified
- [ ] `python3 scripts/db_dispatch_outbox.py --dry-run` exits 0, no status changes
- [ ] `python3 scripts/db_dispatch_outbox.py --channel feishu --apply` sends real Feishu message (test group)
- [ ] event_outbox status updated to dispatched/failed with external_ref
- [ ] All existing 80+ tests still pass
- [ ] dispatch_e2e_report.md with E2E scenario results

### Must Have
- dry-run/apply pattern matching existing conventions
- Max 3 dispatch attempts per event
- Optimistic locking (dispatching status) to prevent concurrent double-dispatch
- NULL/invalid JSON payload handled gracefully (status=failed)
- adapter_registry.yml as single source of truth for adapter routing
- Feishu text-only messages in v0.4 (no cards, no rich text)

### Must NOT Have (Guardrails)
- No changes to sync_feishu_bitable.py or any existing Pandas/CSV pipeline files
- No real GitHub Issue creation (dry-run payload only)
- No Base/Bitable CRUD operations via adapter in v0.4
- No production scheduling/daemon/cron setup
- No LLM decision engine integration
- No Qoder Wake integration
- No complex retry framework (simple 3-attempt max, no circuit breakers)
- No new database tables (dispatch_log) — use event_outbox fields + CSV
- No ORM introduction — raw sqlite3 only
- No changes to existing FeishuClient methods — additive ONLY

---

## Verification Strategy (MANDATORY)

### Test Decision
- **Infrastructure exists**: YES
- **Automated tests**: YES (Tests-after — adapter is new code, insert test data, then verify)
- **Framework**: pytest 9.0
- **If TDD**: Not TDD — insert test data into real SQLite, run dispatch, verify state changes

### QA Policy
Every task MUST include agent-executed QA scenarios.

- **SQL Migration**: Use Bash (python -c) to verify column existence
- **Python Scripts**: Use Bash (python3 scripts/...) to run with --dry-run, assert exit code
- **Feishu Integration**: Use Bash (python3 scripts/...) with dry_run=True first, then --apply with test config
- **Adapter Framework**: Use Bash (python -c) to import and test individual adapters
- **E2E**: Use Bash (python3 scripts/db_dispatch_outbox.py --apply --channel feishu --limit 1) to send one real message

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Foundation — 5 parallel tasks, all independent):
├── T1: SQL migration script [quick]
├── T2: adapter_registry.yml config [quick]
├── T3: FeishuClient.send_message() extension [deep]
├── T4: lark-oapi dependency [quick]
└── T5: feishu_cli_probe.py [quick]

Wave 2 (Adapter Layer — 5 parallel tasks, depend only on Wave 1):
├── T6: ChannelAdapter base class (base.py) [quick] — depends T1 (schema)
├── T7: FeishuAdapter [unspecified-high] — depends T3 (FeishuClient), T6 (base)
├── T8: GitHubIssueAdapter [quick] — depends T6 (base)
├── T9: LocalCLIAdapter [quick] — depends T6 (base)
└── T10: ManualAdapter [quick] — depends T6 (base)

Wave 3 (Dispatcher — 2 sequential, depend on Wave 2):
├── T11: db_dispatch_outbox.py orchestrator [deep] — depends T6-T10 (all adapters)
└── T12: E2E test + report [unspecified-high] — depends T11 (dispatcher)

Wave FINAL (After ALL tasks — 4 parallel reviews):
├── F1: Plan compliance audit (oracle)
├── F2: Code quality review (unspecified-high)
├── F3: Real manual QA (unspecified-high)
└── F4: Scope fidelity check (deep)
-> Present results -> Get explicit user okay

Critical Path: T3 → T7 → T11 → T12 → F1-F4 → user okay
Parallel Speedup: ~60% faster than sequential
Max Concurrent: 5 (Wave 1, Wave 2)
```

### Dependency Matrix

| Task | Blocked By | Blocks |
|------|-----------|--------|
| T1 | None | T6, T11 |
| T2 | None | T11 |
| T3 | None | T7 |
| T4 | None | T7 (import lark_oapi) |
| T5 | None | T12 (optional probe before E2E) |
| T6 | T1 | T7, T8, T9, T10, T11 |
| T7 | T3, T4, T6 | T11 |
| T8 T9 T10 | T6 | T11 |
| T11 | T1-T10 | T12 |
| T12 | T11 | F1-F4 |

### Agent Dispatch Summary

- **Wave 1**: 5 tasks — T1→`quick`, T2→`quick`, T3→`deep`, T4→`quick`, T5→`quick`
- **Wave 2**: 5 tasks — T6→`quick`, T7→`unspecified-high`, T8→`quick`, T9→`quick`, T10→`quick`
- **Wave 3**: 2 tasks — T11→`deep`, T12→`unspecified-high`
- **FINAL**: 4 tasks — F1→`oracle`, F2→`unspecified-high`, F3→`unspecified-high`, F4→`deep`

---

## TODOs

> Implementation + Test = ONE Task. Never separate.
> EVERY task MUST have: Recommended Agent Profile + Parallelization info + QA Scenarios.

- [ ] 1. SQL Migration: 4 new columns on event_outbox

  **What to do**:
  - Create `sql/migrations/005_dispatch_adapters.sql` with ALTER TABLE statements
  - Add columns: `dispatch_attempts INTEGER DEFAULT 0`, `last_dispatch_at TEXT`, `external_ref TEXT`, `adapter_name TEXT`
  - Write migration script following `scripts/db_migrate.py` pattern — check column exists before ALTER (SQLite doesn't support `ADD COLUMN IF NOT EXISTS`)
  - Migration must NOT affect existing rows (all NULL or default values)

  **Must NOT do**:
  - Create new tables (no dispatch_log table)
  - Drop or modify existing columns

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T2-T5)
  - **Blocks**: T6, T11
  - **Blocked By**: None

  **References**:
  - `sql/schema.sql:175-186` — existing event_outbox table definition
  - `sql/migrations/003_dimensional_alerts.sql` — existing migration pattern
  - `scripts/db_migrate.py` — migration runner pattern

  **Acceptance Criteria**:
  - [ ] Migration SQL file created
  - [ ] Migration applied successfully to data/olist_ops.db
  - [ ] All 4 new columns exist in event_outbox
  - [ ] Existing rows NOT modified

  **QA Scenarios**:
  ```
  Scenario: Schema migration applied, columns exist
    Tool: Bash (python -c)
    Steps:
      1. python -c "
         import sqlite3; conn = sqlite3.connect('data/olist_ops.db')
         cols = [r[1] for r in conn.execute('PRAGMA table_info(event_outbox)')]
         for c in ['dispatch_attempts','last_dispatch_at','external_ref','adapter_name']:
             assert c in cols, f'{c} missing'
         print('All 4 columns present')
         "
    Expected Result: All 4 column names found, exit 0
    Evidence: .sisyphus/evidence/task-1-schema-columns.txt

  Scenario: Existing data intact after migration
    Tool: Bash (python -c)
    Steps:
      1. python -c "
         import sqlite3; conn = sqlite3.connect('data/olist_ops.db')
         total = conn.execute('SELECT COUNT(*) FROM event_outbox').fetchone()[0]
         nulls = conn.execute('SELECT COUNT(*) FROM event_outbox WHERE dispatch_attempts IS NULL').fetchone()[0]
         assert nulls == 0
         print(f'total={total}, defaults_ok')
         "
    Expected Result: dispatch_attempts defaults to 0 for all rows (0 NULLs)
    Evidence: .sisyphus/evidence/task-1-existing-data.txt
  ```

  **Commit**: YES (group 1, with T2)
  - Message: `feat(schema): add 4 dispatch columns to event_outbox + adapter_registry.yml`
  - Files: `sql/migrations/005_dispatch_adapters.sql`, `config/adapter_registry.yml`

- [ ] 2. Config: adapter_registry.yml

  **What to do**:
  - Create `config/adapter_registry.yml` with 4 adapter definitions
  - Match existing YAML config patterns (see `config/alert_rules.yml`)
  - Structure with: enabled, module, class, allowed_target_channels, dry_run_only, max_retries
  - Add ADAPTER_REGISTRY_FILE constant to `scripts/config.py`

  **Must NOT do**:
  - Change any existing YAML config files
  - Add more than 4 adapters

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T1, T3-T5)
  - **Blocks**: T11
  - **Blocked By**: None

  **References**:
  - `config/alert_rules.yml` — YAML config style
  - `config/channel_routing_rules.yml` — routing config style
  - `scripts/config.py:11-66` — path constants pattern

  **Acceptance Criteria**:
  - [ ] adapter_registry.yml created with all 4 adapters
  - [ ] config.py has ADAPTER_REGISTRY_FILE constant
  - [ ] YAML parses without errors

  **QA Scenarios**:
  ```
  Scenario: YAML parses with all 4 adapters
    Tool: Bash (python -c)
    Steps:
      1. python -c "
         import yaml
         with open('config/adapter_registry.yml') as f:
             reg = yaml.safe_load(f)
         for a in ['feishu','github_issue','local_cli','manual']:
             assert a in reg['adapters'], f'{a} missing'
             assert 'module' in reg['adapters'][a]
             assert 'class' in reg['adapters'][a]
         print('All 4 adapters registered correctly')
         "
    Expected Result: All 4 adapters have module+class fields, exit 0
    Evidence: .sisyphus/evidence/task-2-registry-yaml.txt
  ```

  **Commit**: YES (group 1, with T1)

- [ ] 3. FeishuClient extension: send_message() method

  **What to do**:
  - READ `scripts/feishu_client.py` fully to understand: constructor, get_tenant_access_token(), _request(), retry pattern
  - Add `send_message(self, chat_id, content, msg_type="text", dry_run=None) -> Optional[str]`
  - Uses POST /open-apis/im/v1/messages?receive_id_type=chat_id
  - Body: {"receive_id": chat_id, "msg_type": msg_type, "content": json.dumps({"text": content})}
  - dry_run=True: log what would be sent, return mock message_id
  - dry_run=None: falls back to self.dry_run from constructor
  - Error handling: catch network/HTTP errors, return None on failure
  - Must be purely ADDITIVE — zero changes to existing methods

  **Must NOT do**:
  - Modify any existing FeishuClient methods
  - Add card/interactive message support (text only)
  - Change constructor signature

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T1-T2, T4-T5)
  - **Blocks**: T7
  - **Blocked By**: None

  **References**:
  - `scripts/feishu_client.py` — FULL READ required
  - `config/feishu_app.yml` — app ID/secret config
  - `.env.example` — FEISHU_APP_ID, FEISHU_APP_SECRET

  **Acceptance Criteria**:
  - [ ] send_message() method exists
  - [ ] Signature: chat_id, content, msg_type="text", dry_run=None
  - [ ] dry_run=True returns mock message_id without API call
  - [ ] Existing tests still pass
  - [ ] git diff shows ONLY additive changes

  **QA Scenarios**:
  ```
  Scenario: send_message dry_run returns without API call
    Tool: Bash (python -c)
    Steps:
      1. python -c "
         import sys; sys.path.insert(0, 'scripts')
         from feishu_client import FeishuClient
         fc = FeishuClient(app_id='test', app_secret='test', dry_run=True)
         result = fc.send_message('test_chat', 'Test alert', dry_run=True)
         assert result is not None
         print(f'dry_run result: {result}')
         "
    Expected Result: Returns mock message_id, exit 0
    Evidence: .sisyphus/evidence/task-3-send-message-dryrun.txt

  Scenario: send_message signature correct
    Tool: Bash (python -c)
    Steps:
      1. python -c "
         import sys, inspect; sys.path.insert(0, 'scripts')
         from feishu_client import FeishuClient
         sig = inspect.signature(FeishuClient.send_message)
         params = list(sig.parameters.keys())
         for p in ['self','chat_id','content','msg_type','dry_run']:
             assert p in params
         assert sig.parameters['msg_type'].default == 'text'
         print('send_message signature correct')
         "
    Expected Result: All params present, msg_type defaults to 'text'
    Evidence: .sisyphus/evidence/task-3-signature.txt
  ```

  **Commit**: YES (group 2)
  - Message: `feat(feishu): add send_message() to FeishuClient`
  - Files: `scripts/feishu_client.py`

- [ ] 4. Dependency: Add lark-oapi to requirements.txt

  **What to do**:
  - Add `lark-oapi>=2.1.0` to requirements.txt
  - Verify `pip install --dry-run lark-oapi` succeeds

  **Must NOT do**:
  - Change other dependency versions

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: T7
  - **Blocked By**: None

  **Acceptance Criteria**:
  - [ ] lark-oapi in requirements.txt
  - [ ] pip install --dry-run succeeds

  **QA Scenarios**:
  ```
  Scenario: lark-oapi installable
    Tool: Bash
    Steps:
      1. grep lark-oapi requirements.txt
      2. pip install --dry-run lark-oapi 2>&1
    Expected Result: grep finds lark-oapi, pip dry-run exits 0
    Evidence: .sisyphus/evidence/task-4-pip-dryrun.txt
  ```

  **Commit**: YES (group 2, with T3)

- [ ] 5. Feishu CLI Capability Probe

  **What to do**:
  - Create `scripts/feishu_cli_probe.py`
  - Check 3 things: (1) Feishu credentials in config/feishu_app.yml or .env, (2) lark-cli binary on PATH, (3) lark-oapi import success
  - Output capability matrix table to stdout AND write to `reports/feishu_cli_capability_matrix.md`
  - No real API calls

  **Must NOT do**:
  - Send real messages, create docs, touch Base
  - Install any software

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: T12 (optional probe before E2E)
  - **Blocked By**: None

  **References**:
  - `config/feishu_app.yml`, `.env.example`

  **Acceptance Criteria**:
  - [ ] Script runs and exits 0
  - [ ] Reports credential/lark-cli/lark-oapi status
  - [ ] Creates reports/feishu_cli_capability_matrix.md

  **QA Scenarios**:
  ```
  Scenario: Probe runs and produces matrix
    Tool: Bash
    Steps:
      1. python3 scripts/feishu_cli_probe.py
      2. assert exit code == 0
      3. assert reports/feishu_cli_capability_matrix.md exists and contains "Capability" or "Status"
    Expected Result: Exits 0, creates markdown report
    Evidence: .sisyphus/evidence/task-5-probe.md
  ```

  **Commit**: YES (group 4)
  - Message: `feat(probe): add feishu_cli_probe.py capability detection`
  - Files: `scripts/feishu_cli_probe.py`, `reports/feishu_cli_capability_matrix.md`

- [ ] 6. ChannelAdapter abstract base class

  **What to do**:
  - Create `scripts/adapters/` directory with `__init__.py`
  - Create `scripts/adapters/base.py` with ChannelAdapter ABC
  - Required interface:
    ```python
    class ChannelAdapter(ABC):
        @abstractmethod
        def dry_run(self, event: dict) -> dict:
            """Preview what would be dispatched. No side effects."""

        @abstractmethod
        def dispatch(self, event: dict) -> dict:
            """Execute real external action. Returns {status, external_ref, error}.
            status: 'dispatched' | 'failed' | 'skipped'
            external_ref: target system's reference ID (message_id, issue_number, etc.)
            error: error message string or None
            """
    ```
  - Create helper: `def load_adapter_registry() -> dict` that reads config/adapter_registry.yml
  - Create helper: `def resolve_adapter(channel: str) -> ChannelAdapter` that instantiates the correct adapter class
  - Handle adapter instantiation errors gracefully (missing module, missing class)

  **Must NOT do**:
  - Implement any real adapter logic (just ABC + registry loader)
  - Import FeishuClient or lark-oapi in base.py

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO (depends on T1 schema migration — adapter needs to know what fields exist)
  - **Parallel Group**: Wave 2 start
  - **Blocks**: T7, T8, T9, T10, T11
  - **Blocked By**: T1

  **References**:
  - `scripts/config.py` — config path pattern
  - `config/adapter_registry.yml` — (created by T2) registry format
  - `scripts/db_trigger_simulator.py` — existing event processing pattern

  **Acceptance Criteria**:
  - [ ] scripts/adapters/__init__.py exists
  - [ ] scripts/adapters/base.py defines ChannelAdapter ABC with dry_run() and dispatch()
  - [ ] load_adapter_registry() reads YAML successfully
  - [ ] resolve_adapter('feishu') returns FeishuAdapter instance (when adapter exists)

  **QA Scenarios**:
  ```
  Scenario: ABC importable with required methods
    Tool: Bash (python -c)
    Steps:
      1. python -c "
         import sys; sys.path.insert(0, 'scripts')
         from adapters.base import ChannelAdapter
         assert hasattr(ChannelAdapter, 'dry_run')
         assert hasattr(ChannelAdapter, 'dispatch')
         import inspect
         assert inspect.isabstract(ChannelAdapter)
         print('ChannelAdapter ABC valid')
         "
    Expected Result: ABC is abstract, has both methods, exit 0
    Evidence: .sisyphus/evidence/task-6-abc-import.txt

  Scenario: Registry loader reads YAML
    Tool: Bash (python -c)
    Steps:
      1. python -c "
         import sys; sys.path.insert(0, 'scripts')
         from adapters.base import load_adapter_registry
         reg = load_adapter_registry()
         assert 'adapters' in reg
         print(f'Registry loaded: {list(reg["adapters"].keys())}')
         "
    Expected Result: Registry dict loaded with 4 adapter keys
    Evidence: .sisyphus/evidence/task-6-registry-load.txt
  ```

  **Commit**: YES (group 5, Wave 2 start)
  - Message: `feat(adapter): ChannelAdapter ABC and registry loader`
  - Files: `scripts/adapters/__init__.py`, `scripts/adapters/base.py`

- [ ] 7. FeishuAdapter: Feishu IM dispatch

  **What to do**:
  - Create `scripts/adapters/feishu_adapter.py`
  - FeishuAdapter(ChannelAdapter) implementation
  - constructor: reads Feishu app credentials from config, creates FeishuClient instance
  - dry_run(): parse payload_json, format message text, return preview dict without sending
  - dispatch(): call FeishuClient.send_message() with real chat_id, return {status, external_ref, error}
  - Payload parsing: extract rule_id, metric_name, current_value, baseline_value, severity, owner_role from event_outbox.payload_json
  - Message format: `[Alert] {rule_id}: {metric_name}
Current: {current_value}
Baseline: {baseline_value}
Severity: {severity}
Owner: {owner_role}`
  - Handle NULL payload_json → status='failed', error='payload_json is NULL'
  - Handle invalid JSON → status='failed', error='invalid JSON in payload'
  - Handle FeishuClient errors (token failure, network error) → status='failed', error=...
  - chat_id lookup: read from config/feishu_app.yml or fall back to FEISHU_CHAT_ID env var

  **Must NOT do**:
  - Send card/interactive messages (text only)
  - Batch operations
  - Modify FeishuClient (just consume it)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with T8-T10)
  - **Parallel Group**: Wave 2 (with T8-T10)
  - **Blocks**: T11
  - **Blocked By**: T3, T4, T6

  **References**:
  - `scripts/feishu_client.py` — send_message() method (from T3)
  - `scripts/db_trigger_simulator.py:65-77` — existing feishu payload format
  - `config/feishu_app.yml` — chat_id config
  - `config/alert_rules.yml` — alert format for reference

  **Acceptance Criteria**:
  - [ ] FeishuAdapter extends ChannelAdapter
  - [ ] dry_run() returns formatted message without API call
  - [ ] dispatch() calls FeishuClient.send_message() and returns status/external_ref
  - [ ] NULL payload → status='failed'
  - [ ] Invalid JSON → status='failed'
  - [ ] dry_run mode: no external API call made

  **QA Scenarios**:
  ```
  Scenario: FeishuAdapter dry_run formats message
    Tool: Bash (python -c)
    Steps:
      1. python -c "
         import sys, json; sys.path.insert(0, 'scripts')
         from adapters.feishu_adapter import FeishuAdapter
         adapter = FeishuAdapter(dry_run=True)
         event = {'payload_json': json.dumps({'rule_id':'gmv_drop','metric_name':'gmv','current_value':1000,'baseline_value':1500,'severity':'high','owner_role':'business_ops'})}
         result = adapter.dry_run(event)
         assert result['status'] == 'preview'
         assert 'gmv_drop' in result['message']
         print(f'Dry run OK: {result}')
         "
    Expected Result: Returns preview dict with formatted alert message
    Evidence: .sisyphus/evidence/task-7-dryrun.txt

  Scenario: FeishuAdapter handles NULL payload
    Tool: Bash (python -c)
    Steps:
      1. python -c "
         import sys; sys.path.insert(0, 'scripts')
         from adapters.feishu_adapter import FeishuAdapter
         adapter = FeishuAdapter(dry_run=True)
         result = adapter.dispatch({'payload_json': None})
         assert result['status'] == 'failed'
         assert result['error'] is not None
         print(f'NULL payload handled: {result}')
         "
    Expected Result: status='failed', meaningful error message
    Evidence: .sisyphus/evidence/task-7-null-payload.txt

  Scenario: FeishuAdapter handles invalid JSON
    Tool: Bash (python -c)
    Steps:
      1. python -c "
         import sys; sys.path.insert(0, 'scripts')
         from adapters.feishu_adapter import FeishuAdapter
         adapter = FeishuAdapter(dry_run=True)
         result = adapter.dispatch({'payload_json': 'not valid json'})
         assert result['status'] == 'failed'
         print(f'Invalid JSON handled: {result}')
         "
    Expected Result: status='failed', error mentions JSON parse failure
    Evidence: .sisyphus/evidence/task-7-invalid-json.txt
  ```

  **Commit**: YES (group 5, Wave 2)
  - Message: `feat(adapter): FeishuAdapter for IM message dispatch`
  - Files: `scripts/adapters/feishu_adapter.py`

- [ ] 8. GitHubIssueAdapter: payload generation only (dry-run)

  **What to do**:
  - Create `scripts/adapters/github_issue_adapter.py`
  - GitHubIssueAdapter(ChannelAdapter)
  - dry_run(): parse payload_json, generate GitHub Issue payload JSON
  - dispatch(): ALWAYS raise NotImplementedError or return status='skipped' with message 'GitHub adapter is dry-run only in v0.4'
  - Issue payload format: {"title": "[Alert] {rule_id}: {metric_name} anomaly", "body": markdown with metric details, "labels": ["alert", severity, owner_role]}
  - Write dry-run payload to `data/system/github_dispatch_dryrun/` as JSON files
  - Handle NULL/invalid JSON gracefully

  **Must NOT do**:
  - Call GitHub API
  - Create real issues
  - Accept any parameter that bypasses the dry-run restriction

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with T7, T9-T10)
  - **Blocks**: T11
  - **Blocked By**: T6

  **References**:
  - `scripts/db_trigger_simulator.py:45-62` — existing GitHub payload format
  - `scripts/adapters/base.py` — ChannelAdapter ABC (from T6)

  **Acceptance Criteria**:
  - [ ] GitHubIssueAdapter extends ChannelAdapter
  - [ ] dry_run() generates valid GitHub Issue payload
  - [ ] dispatch() raises NotImplementedError (cannot create real issues)
  - [ ] Writes dry-run payload to data/system/github_dispatch_dryrun/
  - [ ] NULL/invalid payload handled gracefully

  **QA Scenarios**:
  ```
  Scenario: GitHub adapter generates valid issue payload
    Tool: Bash (python -c)
    Steps:
      1. python -c "
         import sys, json; sys.path.insert(0, 'scripts')
         from adapters.github_issue_adapter import GitHubIssueAdapter
         adapter = GitHubIssueAdapter()
         event = {'payload_json': json.dumps({'rule_id':'gmv_drop','metric_name':'gmv','current_value':1000,'baseline_value':1500,'severity':'high','owner_role':'business_ops'})}
         result = adapter.dry_run(event)
         assert 'title' in result['payload']
         assert 'body' in result['payload']
         assert 'gmv_drop' in result['payload']['title']
         print(f'GitHub payload: {result}')
         "
    Expected Result: Valid GitHub payload with title/body, exit 0
    Evidence: .sisyphus/evidence/task-8-github-payload.txt

  Scenario: GitHub adapter dispatch raises NotImplementedError
    Tool: Bash (python -c)
    Steps:
      1. python -c "
         import sys, json; sys.path.insert(0, 'scripts')
         from adapters.github_issue_adapter import GitHubIssueAdapter
         adapter = GitHubIssueAdapter()
         event = {'payload_json': json.dumps({})}
         try:
             result = adapter.dispatch(event)
             # If it doesn't raise, check status
             assert result['status'] in ['skipped', 'failed'], 'should not dispatch'
         except NotImplementedError:
             print('NotImplementedError raised as expected')
         "
    Expected Result: Either NotImplementedError or status='skipped'/'failed'
    Evidence: .sisyphus/evidence/task-8-dispatch-blocked.txt
  ```

  **Commit**: YES (group 5, Wave 2)
  - Message: `feat(adapter): GitHubIssueAdapter dry-run only`
  - Files: `scripts/adapters/github_issue_adapter.py`

- [ ] 9. LocalCLIAdapter

  **What to do**:
  - Create `scripts/adapters/local_cli_adapter.py`
  - LocalCLIAdapter(ChannelAdapter)
  - dry_run(): parse payload_json, generate local command suggestion
  - dispatch(): write command suggestion to CSV log, return status='dispatched' with external_ref=log_file_path
  - Command format: `python3 scripts/run_alert_detection.py --rule {rule_id} --investigate`
  - Follow existing db_trigger_simulator local_cli pattern
  - Handle NULL/invalid JSON

  **Must NOT do**:
  - Execute any shell commands
  - Write to any production output directory

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with T7-T8, T10)
  - **Blocks**: T11
  - **Blocked By**: T6

  **References**:
  - `scripts/db_trigger_simulator.py:80-88` — existing local_cli payload format
  - `scripts/adapters/base.py` — ChannelAdapter ABC

  **Acceptance Criteria**:
  - [ ] LocalCLIAdapter extends ChannelAdapter
  - [ ] dry_run() generates command suggestion
  - [ ] dispatch() writes to CSV and returns status='dispatched'
  - [ ] NULL/invalid payload handled

  **QA Scenarios**:
  ```
  Scenario: LocalCLI adapter dispatch writes CSV
    Tool: Bash (python -c)
    Steps:
      1. python -c "
         import sys, json, os; sys.path.insert(0, 'scripts')
         from adapters.local_cli_adapter import LocalCLIAdapter
         adapter = LocalCLIAdapter()
         event = {'outbox_id':'outbox-test','payload_json':json.dumps({'rule_id':'gmv_drop','metric_name':'gmv','severity':'high','owner_role':'business_ops'})}
         result = adapter.dispatch(event)
         assert result['status'] == 'dispatched'
         assert 'external_ref' in result
         assert os.path.exists(result['external_ref'])
         print(f'Local CLI dispatched: {result}')
         "
    Expected Result: status='dispatched', external_ref is CSV file path that exists
    Evidence: .sisyphus/evidence/task-9-local-dispatch.txt
  ```

  **Commit**: YES (group 5, Wave 2)
  - Message: `feat(adapter): LocalCLIAdapter for local command suggestions`
  - Files: `scripts/adapters/local_cli_adapter.py`

- [ ] 10. ManualAdapter

  **What to do**:
  - Create `scripts/adapters/manual_adapter.py`
  - ManualAdapter(ChannelAdapter)
  - dry_run(): return preview indicating event will be queued for human review
  - dispatch(): update status='skipped', message='queued for manual review', write log entry
  - Minimal implementation — just marks events for human handling
  - Handle NULL/invalid JSON

  **Must NOT do**:
  - Send notifications
  - Create tickets
  - Any external action

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with T7-T9)
  - **Blocks**: T11
  - **Blocked By**: T6

  **References**:
  - `scripts/adapters/base.py` — ChannelAdapter ABC

  **Acceptance Criteria**:
  - [ ] ManualAdapter extends ChannelAdapter
  - [ ] dispatch() returns status='skipped' with manual review message
  - [ ] NULL/invalid payload handled

  **QA Scenarios**:
  ```
  Scenario: Manual adapter marks for review
    Tool: Bash (python -c)
    Steps:
      1. python -c "
         import sys, json; sys.path.insert(0, 'scripts')
         from adapters.manual_adapter import ManualAdapter
         adapter = ManualAdapter()
         event = {'outbox_id':'outbox-test','payload_json':json.dumps({'rule_id':'gmv_drop'})}
         result = adapter.dispatch(event)
         assert result['status'] == 'skipped'
         assert 'manual' in result.get('message','').lower()
         print(f'Manual: {result}')
         "
    Expected Result: status='skipped', message mentions manual review
    Evidence: .sisyphus/evidence/task-10-manual.txt
  ```

  **Commit**: YES (group 5, Wave 2)
  - Message: `feat(adapter): ManualAdapter for human review queue`
  - Files: `scripts/adapters/manual_adapter.py`

- [ ] 11. Dispatch orchestrator: db_dispatch_outbox.py

  **What to do**:
  - Create `scripts/db_dispatch_outbox.py`
  - Main dispatch script, follows existing db_* pattern:
    - get_db() wrapper with PRAGMA settings
    - argparse with --dry-run, --apply, --channel, --limit, --db flags
    - Reads event_outbox WHERE status='pending' AND dispatch_attempts < 3
    - For each event: loads adapter from registry by target_channel
    - Optimistic locking: UPDATE event_outbox SET status='dispatching' WHERE outbox_id=X AND status='pending'
    - If update affects 0 rows: skip (another process already claimed it)
    - Call adapter.dry_run() or adapter.dispatch() based on --dry-run flag
    - Write back: status='dispatched'/'failed', external_ref, adapter_name, dispatch_attempts+1, last_dispatch_at
    - Log results to stdout
    - CSV audit log: write dispatch results to data/system/dispatch_archive.csv

  **Command line**:
  ```
  python3 scripts/db_dispatch_outbox.py --dry-run          # Preview all
  python3 scripts/db_dispatch_outbox.py --apply             # Dispatch all pending
  python3 scripts/db_dispatch_outbox.py --channel feishu --apply   # Only feishu channel
  python3 scripts/db_dispatch_outbox.py --limit 10 --apply  # Max 10 events
  ```

  **Status transition**:
  ```
  pending → dispatching (optimistic lock)
  dispatching → dispatched (success)
  dispatching → failed (error)
  dispatching → pending (retry, if dispatch_attempts < 3)
  ```

  **Must NOT do**:
  - Modify sync_feishu_bitable.py
  - Run as daemon/service
  - Process events with status != 'pending' AND dispatch_attempts >= 3
  - Process more than --limit events

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO (depends on all prior tasks)
  - **Parallel Group**: Wave 3 start
  - **Blocks**: T12
  - **Blocked By**: T1-T10

  **References**:
  - `scripts/db_trigger_simulator.py` — existing dispatch simulation pattern
  - `scripts/db_rule_engine.py` — event writing pattern
  - `scripts/config.py` — path constants, DB_PATH
  - `config/adapter_registry.yml` — adapter resolution
  - `scripts/adapters/base.py` — ChannelAdapter ABC, load_adapter_registry(), resolve_adapter()

  **Acceptance Criteria**:
  - [ ] --dry-run exits 0, no status changes in DB
  - [ ] --apply dispatches pending events through correct adapters
  - [ ] --channel feishu filters to feishu channel only
  - [ ] --limit 10 processes max 10 events
  - [ ] Optimistic locking prevents double-dispatch
  - [ ] dispatch_attempts incremented per attempt
  - [ ] CSV audit log written to data/system/dispatch_archive.csv
  - [ ] NULL/invalid payload events marked as failed
  - [ ] Events with dispatch_attempts >= 3 are skipped

  **QA Scenarios**:
  ```
  Scenario: dry-run processes events without side effects
    Tool: Bash (python3 scripts/...)
    Steps:
      1. python -c "
         import sqlite3; conn = sqlite3.connect('data/olist_ops.db')
         before = conn.execute('SELECT COUNT(*) FROM event_outbox WHERE status="pending"').fetchone()[0]
         "
      2. python3 scripts/db_dispatch_outbox.py --dry-run 2>&1
      3. python -c "
         import sqlite3; conn = sqlite3.connect('data/olist_ops.db')
         after = conn.execute('SELECT COUNT(*) FROM event_outbox WHERE status="pending"').fetchone()[0]
         dispatched = conn.execute('SELECT COUNT(*) FROM event_outbox WHERE status="dispatched" AND adapter_name IS NOT NULL').fetchone()[0]
         assert dispatched == 0, 'dry-run should not dispatch any events'
         print(f'pending: {after}, dispatched_with_adapter: {dispatched}')
         "
    Expected Result: Pending count unchanged, no events dispatched with adapter_name, exit 0
    Evidence: .sisyphus/evidence/task-11-dryrun.txt

  Scenario: channel filter works
    Tool: Bash (python3 scripts/...)
    Steps:
      1. Insert test events with target_channel='feishu' and target_channel='manual'
      2. python3 scripts/db_dispatch_outbox.py --channel feishu --dry-run 2>&1
      3. Verify only feishu-channel events appear in output
    Expected Result: Output mentions only feishu events, manual events not processed
    Evidence: .sisyphus/evidence/task-11-channel-filter.txt

  Scenario: Optimistic lock prevents double-dispatch
    Tool: Bash (python -c)
    Steps:
      1. Insert test event with status='dispatching' (simulating concurrent claim)
      2. Run db_dispatch_outbox.py --apply --limit 1
      3. Verify the 'dispatching' event was NOT double-processed
    Expected Result: Event with status='dispatching' skipped, no error
    Evidence: .sisyphus/evidence/task-11-optimistic-lock.txt
  ```

  **Commit**: YES (group 6)
  - Message: `feat(dispatch): db_dispatch_outbox.py orchestrator with dry-run/apply`
  - Files: `scripts/db_dispatch_outbox.py`
  - Pre-commit: python3 scripts/db_dispatch_outbox.py --dry-run

- [ ] 12. E2E test + acceptance report

  **What to do**:
  - Create end-to-end test: `python3 scripts/e2e_dispatch_test.py` or integrate into existing test suite
  - Full flow: insert test event → run dispatch with dry_run=True → verify status → run with real dispatch (if credentials available)
  - Test all adapters through dispatcher:
    1. Insert event with target_channel='feishu' → dispatch (dry_run) → verify dry_run path
    2. Insert event with target_channel='github_issue' → dispatch → verify payload generated, NOT dispatched
    3. Insert event with target_channel='local_cli' → dispatch → verify CSV written
    4. Insert event with target_channel='manual' → dispatch → verify skipped
  - NULL payload test: insert event with NULL payload_json → dispatch → verify failed status
  - dispatch_attempts test: insert event with dispatch_attempts=3 → dispatch → verify skipped
  - After all tests, write `reports/dispatch_e2e_report.md` with results

  **Must NOT do**:
  - Send real messages to production groups (use test group only)
  - Create real GitHub issues

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO (depends on T11)
  - **Parallel Group**: Wave 3
  - **Blocks**: F1-F4
  - **Blocked By**: T11

  **References**:
  - `tests/conftest.py` — pytest fixtures pattern
  - `scripts/db_dispatch_outbox.py` — dispatcher
  - `scripts/feishu_cli_probe.py` — prerequisite probe
  - `tests/test_db_event_outbox_routing.py` — existing outbox test patterns

  **Acceptance Criteria**:
  - [ ] All 4 adapters E2E through dispatcher (dry_run)
  - [ ] NULL payload handled
  - [ ] dispatch_attempts >= 3 skipped
  - [ ] Feishu real dispatch if credentials valid
  - [ ] reports/dispatch_e2e_report.md written

  **QA Scenarios**:
  ```
  Scenario: Full E2E flow — insert, dispatch, verify
    Tool: Bash (python3 scripts/e2e_dispatch_test.py)
    Steps:
      1. python -c "
         import sqlite3, json, datetime, uuid
         conn = sqlite3.connect('data/olist_ops.db')
         eid = f'outbox-e2e-test-{uuid.uuid4().hex[:8]}'
         now = datetime.datetime.now().isoformat()
         conn.execute('INSERT INTO event_outbox (outbox_id, event_type, source_type, source_id, payload_json, target_channel, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)',
             (eid, 'alert', 'e2e_test', 'test-1', json.dumps({'rule_id':'gmv_drop','metric_name':'gmv','current_value':1000,'baseline_value':1500,'severity':'high','owner_role':'business_ops'}), 'feishu', 'pending', now))
         conn.commit()
         print(f'Inserted test event: {eid}')
         "
      2. python3 scripts/db_dispatch_outbox.py --channel feishu --dry-run 2>&1
      3. python -c "
         import sqlite3; conn = sqlite3.connect('data/olist_ops.db')
         row = conn.execute('SELECT status, adapter_name, dispatch_attempts FROM event_outbox WHERE outbox_id = ?', (eid,)).fetchone()
         # dry-run should NOT change status to dispatched
         assert row[0] != 'dispatched' or row[1] is None, 'dry-run should not fully dispatch'
         print(f'dry_run E2E: status={row[0]}, adapter={row[1]}, attempts={row[2]}')
         "
    Expected Result: Event inserted, dry-run processed, status NOT changed to dispatched
    Evidence: .sisyphus/evidence/task-12-e2e-flow.txt
  ```

  **Commit**: YES (group 7)
  - Message: `test(e2e): dispatch E2E test and acceptance report`
  - Files: `reports/dispatch_e2e_report.md`

---

## Final Verification Wave (MANDATORY — after ALL implementation tasks)

> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.
>
> **Do NOT auto-proceed after verification. Wait for user's explicit approval before marking work complete.**
> **Never mark F1-F4 as checked before getting user's okay.**

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check all evidence files exist in .sisyphus/evidence/. Verify: send_message() is purely additive, no sync_feishu_bitable.py changes, no dispatch_log table, no GitHub real issues.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `pytest -v` with ALL existing tests (must pass). Run `python3 scripts/db_dispatch_outbox.py --dry-run`. Review all new files for: duplicate imports, missing error handling, unused variables. Check: adapters follow ChannelAdapter interface, dispatch_outbox.py follows db_* conventions. Verify no `as any`, `@ts-ignore` equivalents, no console.log in prod (no print() in non-debug paths).
  Output: `Build [PASS/FAIL] | Lint [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [ ] F3. **Real Manual QA** — `unspecified-high`
  Start from clean state. Run the full chain: `python3 scripts/feishu_cli_probe.py` → `python3 scripts/db_dispatch_outbox.py --dry-run` → insert test event → `python3 scripts/db_dispatch_outbox.py --channel feishu --dry-run` → verify all 4 adapters work through dispatcher. Verify CSV audit log written. Verify status transitions correct. Test edge cases: NULL payload, invalid JSON, dispatch_attempts=3, concurrent dispatching status.
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff (git log/diff). Verify 1:1 — everything in spec was built (no missing), nothing beyond spec was built (no creep). Check "Must NOT do" compliance for each task. Detect cross-task contamination: Task N touching Task M's files. Verify sync_feishu_bitable.py unchanged, existing FeishuClient methods unchanged.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

- **Group 1 (Wave 1A - Schema + Config)**: T1 + T2
  - Message: `feat(v0.4): add dispatch columns to event_outbox + adapter_registry.yml`
  - Files: `sql/migrations/005_dispatch_adapters.sql`, `config/adapter_registry.yml`, `scripts/config.py` (added ADAPTER_REGISTRY_FILE)
  - Pre-commit: python -c "import sqlite3; conn=sqlite3.connect('data/olist_ops.db'); cols=[r[1] for r in conn.execute('PRAGMA table_info(event_outbox)')]; assert 'dispatch_attempts' in cols"

- **Group 2 (Wave 1B - FeishuClient)**: T3 + T4
  - Message: `feat(feishu): add send_message() + lark-oapi dependency`
  - Files: `scripts/feishu_client.py`, `requirements.txt`
  - Pre-commit: python -c "import sys; sys.path.insert(0,'scripts'); from feishu_client import FeishuClient; assert hasattr(FeishuClient,'send_message')"

- **Group 3 (Wave 1C - Probe)**: T5
  - Message: `feat(probe): add feishu_cli_probe.py capability detection`
  - Files: `scripts/feishu_cli_probe.py`, `reports/feishu_cli_capability_matrix.md`
  - Pre-commit: python3 scripts/feishu_cli_probe.py

- **Group 4 (Wave 2 - Adapters)**: T6-T10
  - Message: `feat(adapter): ChannelAdapter base class + 4 adapter implementations`
  - Files: `scripts/adapters/__init__.py`, `scripts/adapters/base.py`, `scripts/adapters/feishu_adapter.py`, `scripts/adapters/github_issue_adapter.py`, `scripts/adapters/local_cli_adapter.py`, `scripts/adapters/manual_adapter.py`
  - Pre-commit: python -c "import sys; sys.path.insert(0,'scripts'); from adapters.base import ChannelAdapter; assert __import__('inspect').isabstract(ChannelAdapter)"

- **Group 5 (Wave 3 - Dispatcher)**: T11
  - Message: `feat(dispatch): db_dispatch_outbox.py orchestrator with dry-run/apply`
  - Files: `scripts/db_dispatch_outbox.py`
  - Pre-commit: python3 scripts/db_dispatch_outbox.py --dry-run

- **Group 6 (Wave 3 - E2E)**: T12
  - Message: `test(e2e): dispatch E2E test and acceptance report`
  - Files: `reports/dispatch_e2e_report.md`
  - Pre-commit: pytest -v

---

## Success Criteria

### Verification Commands
```bash
# Schema migration applied
python -c "import sqlite3; conn=sqlite3.connect('data/olist_ops.db'); cols=[r[1] for r in conn.execute('PRAGMA table_info(event_outbox)')]; [print(c) for c in ['dispatch_attempts','last_dispatch_at','external_ref','adapter_name'] if c in cols]"  # Output: all 4 columns

# All existing tests pass
pytest -v  # All 80+ existing tests pass, no failures

# Dispatcher dry-run works
python3 scripts/db_dispatch_outbox.py --dry-run  # Exit 0, no status changes

# Capability probe works
python3 scripts/feishu_cli_probe.py  # Exit 0, capability matrix generated

# FeishuClient send_message importable
python -c "from scripts.feishu_client import FeishuClient; fc=FeishuClient('x','x',dry_run=True); r=fc.send_message('test','hello',dry_run=True); print(r)"  # Returns mock message_id
```

### Final Checklist
- [ ] All "Must Have" items present and verified
- [ ] All "Must NOT Have" items absent from codebase
- [ ] All 12 tasks completed with QA scenarios passed
- [ ] All 4 F-review tasks APPROVE
- [ ] No existing files modified except: feishu_client.py (additive only), config.py (additive constant)
- [ ] Existing tests pass unchanged
- [ ] dispatch_e2e_report.md written with results