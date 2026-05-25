# Go + PostgreSQL Migration — Phase 0: Baseline Freeze

## TL;DR

> Freeze the current Python + SQLite + FastAPI system as an auditable baseline before migrating to Go + PostgreSQL. Commit all unstaged work, create tag + branches, export DB schema/row counts/pipeline samples/API responses/governance YAML snapshots, and document the 7-phase migration plan.
>
> **Deliverables**:
> - Git tag: `v0.5.3-python-sqlite-freeze`
> - Branches: `legacy/python-sqlite`, `migration/go-postgres`
> - `migration_baseline/` — schema, row counts, CSV samples, API JSON, YAML snapshots
> - `scripts/migration/` — 5 export scripts
> - `docs/migration/go-postgres-migration-plan.md` — 7-phase plan
>
> **Estimated Effort**: Medium
> **Parallel Execution**: YES — 4 export tasks can run in parallel after commit
> **Critical Path**: Commit cleanup → Create tag + branches → Export scripts → Run exports → Write docs → Push

---

## Context

### Original Request
User wants to migrate Baxi from Python + SQLite + FastAPI to Go + PostgreSQL + Docker Compose. Phase 0 is a **baseline freeze** — capture the current system state without modifying existing code or introducing Go/PostgreSQL.

### Interview Summary
**Key Decisions**:
- Uncommitted changes (64 files): **Commit to main first**, then freeze
- Primary DB: **`data/olist_ops.db`** (16 tables, 906,862 rows). `data/system/baxi.db` has 0 tables.
- API service: **Not running**, needs startup. `.env` has placeholder `API_BEARER_TOKEN=REPLACE_ME` — must generate valid token first.
- sqlite3 CLI: **Not available** — must use Python `sqlite3` module.

**Research Findings**:
- API reports version 0.6.0 in `api/main.py` but pyproject.toml says 0.5.3. Using pyproject.toml as authoritative source.
- 17 tables found (not 14 as spec mentions — qoder tables added).
- `metric_dimension_daily` is largest table: 693,602 rows.
- `.sisyphus/` has 22+ untracked files that need classification (commit vs ignore).

### Metis Review
**Identified Gaps** (addressed in plan):
- Version tag name aligned to pyproject.toml (0.5.3)
- API token injection strategy: generate temp token for baseline capture
- `.sisyphus/` files classified: `plans/` committed, `notepads/`/`evidence/`/`drafts/` ignored
- DB backup cleanup: `data/olist_ops.db.pre-fk-backup` noted for cleanup
- TestClient used for API export instead of real uvicorn
- 3-commit structure strictly enforced

---

## Work Objectives

### Core Objective
Create an auditable, reproducible baseline of the current Python + SQLite system before any migration work begins.

### Concrete Deliverables
- [ ] Git annotated tag `v0.5.3-python-sqlite-freeze`
- [ ] Branch `legacy/python-sqlite` (read-only from freeze)
- [ ] Branch `migration/go-postgres` (integration branch for future phases)
- [ ] `migration_baseline/README.md` — index of all captured artifacts
- [ ] `migration_baseline/sqlite_schema.sql` — full DDL from `sqlite_master`
- [ ] `migration_baseline/table_counts.json` — row counts for all 16 tables
- [ ] `migration_baseline/pipeline_outputs/*.csv` — sample exports from core tables
- [ ] `migration_baseline/api_responses/*.json` — API response snapshots
- [ ] `migration_baseline/configs_snapshot/*.yml` — governance YAML copies
- [ ] `scripts/migration/export_schema.py` — schema export script
- [ ] `scripts/migration/export_row_counts.py` — row count export script
- [ ] `scripts/migration/export_pipeline_samples.py` — pipeline CSV export script
- [ ] `scripts/migration/export_api_responses.py` — API snapshot script
- [ ] `scripts/migration/export_config_snapshot.py` — YAML copy script
- [ ] `docs/migration/go-postgres-migration-plan.md` — 7-phase migration plan

### Definition of Done
- [ ] All 16 checklist items in Final Verification Wave are checked
- [ ] `migration_baseline/` contains all expected files
- [ ] Git tag points to correct commit
- [ ] Both branches exist on remote
- [ ] Plan document has 7 phases with acceptance criteria

### Must Have
- Commit all unstaged work before creating tag
- Use annotated git tag (not lightweight)
- Export schema from `sqlite_master` (not `sql/schema.sql`) to capture migration-applied state
- Generate valid `API_BEARER_TOKEN` for API export (inject temp token)
- Copy ALL 29 YAML config files, not just a subset

### Must NOT Have (Guardrails)
- No deletion of existing branches, code, or git history
- No modification of existing business logic, pipeline behavior, or API responses
- No introduction of Go, PostgreSQL, Docker, or any migration-phase code
- No renaming or reorganizing of existing directories
- No fixing of tests, data issues, or bugs during freeze
- No committing of `.db` files (only schema DDL + row counts)
- No committing of `.sisyphus/notepads/`, `.sisyphus/evidence/`, `.sisyphus/drafts/`

---

## Verification Strategy

### Test Decision
- **Infrastructure exists**: YES (pytest, 473+ tests)
- **Automated tests**: NO — this is a non-code freeze task. No new tests needed.
- **Agent-Executed QA**: YES — every task has agent-executable verification

### QA Policy
Every task includes agent-executed QA scenarios using bash/python commands. Evidence saved to `.sisyphus/evidence/`.

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Sequential — Git setup must complete first):
├── T1: Commit cleanup (64 unstaged files + selected untracked)
├── T2: Create annotated tag v0.5.3-python-sqlite-freeze
├── T3: Create legacy/python-sqlite branch
└── T4: Create migration/go-postgres branch

Wave 2 (After Wave 1 — 5 export scripts in parallel):
├── T5: Export SQLite schema + row counts
├── T6: Export pipeline CSV samples
├── T7: Export API response snapshots (inject temp token)
├── T8: Export governance YAML snapshot
└── T9: Generate migration_baseline/README.md

Wave 3 (After Wave 2 — docs + push):
├── T10: Write docs/migration/go-postgres-migration-plan.md
├── T11: Commit 1: scripts/
├── T12: Commit 2: migration_baseline/
├── T13: Commit 3: docs/migration/
└── T14: Push migration/go-postgres branch

Wave FINAL (After ALL tasks — verification):
├── F1: Git compliance audit (tag, branches, commits)
├── F2: Baseline artifact completeness check
├── F3: API response validity check
└── F4: Scope fidelity check (no code modifications)
```

### Dependency Matrix

- **T1**: None → T2, T3, T4
- **T2**: T1 → T3, T4
- **T3**: T2 → None
- **T4**: T2 → T5-T14
- **T5-T9**: T4 → T10-T14
- **T10**: T5-T9 → T11-T14
- **T11**: T10 → T12
- **T12**: T11 → T13
- **T13**: T12 → T14
- **T14**: T13 → F1-F4
- **F1-F4**: T14 → user okay

---

## TODOs

- [ ] 1. Commit cleanup — stage and commit 64 unstaged files + selected untracked

  **What to do**:
  1. Review unstaged changes: `git diff --stat` to understand scope
  2. Classify untracked files:
     - **Commit**: `CHANGELOG.md`, `SECURITY_AUDIT_REPORT.md`, `.github/`, `core/`, `pipeline/`, `api/routers/qoder.py`, `api/schemas_qoder.py`, `config/qoder_capabilities.yml`, `.sisyphus/plans/*.md`
     - **Ignore** (add to `.gitignore`): `.sisyphus/notepads/`, `.sisyphus/evidence/`, `.sisyphus/drafts/`, `frontend/tsconfig.tsbuildinfo`, `.ruff_cache/`, `.pytest_cache/`, `htmlcov/`
     - **Delete** (stale): `data/olist_ops.db.pre-fk-backup` (267MB stale backup)
  3. Stage committed-classified files: `git add [files]`
  4. Commit: `git commit -m "chore: commit pre-freeze changes (64 files)"`

  **Must NOT do**:
  - Do NOT modify any file content during staging
  - Do NOT commit `.db` files, cache directories, or `.sisyphus/` ephemeral files
  - Do NOT amend or squash — preserve individual history

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Reason**: Git operations are straightforward, no coding required

  **Parallelization**:
  - **Can Run In Parallel**: NO — must complete before T2-T4
  - **Blocks**: T2, T3, T4

  **Acceptance Criteria**:
  - [ ] `git status --short` shows only ignored/untracked files (no modified)
  - [ ] `git log --oneline -1` shows cleanup commit

  **QA Scenario**:
  ```
  Tool: Bash
  Steps:
    1. git status --short | grep -c "^ M" → expected: 0
    2. git log --oneline -1 | grep -c "pre-freeze" → expected: 1
  Evidence: .sisyphus/evidence/t1-cleanup-complete.txt
  ```

  **Commit**: YES — standalone commit before any migration work

---

- [ ] 2. Create annotated tag `v0.5.3-python-sqlite-freeze`

  **What to do**:
  1. Verify `pyproject.toml` version is `0.5.3`
  2. Create annotated tag: `git tag -a v0.5.3-python-sqlite-freeze -m "Freeze Python + SQLite baseline before Go + PostgreSQL migration"`
  3. Verify tag points to HEAD: `git rev-parse v0.5.3-python-sqlite-freeze` == `git rev-parse HEAD`

  **Must NOT do**:
  - Do NOT create lightweight tag (always use `git tag -a`)
  - Do NOT overwrite existing tag

  **Recommended Agent Profile**:
  - **Category**: `quick`

  **Parallelization**:
  - **Can Run In Parallel**: NO — sequential after T1
  - **Blocked By**: T1
  - **Blocks**: T3, T4

  **Acceptance Criteria**:
  - [ ] `git tag -l | grep v0.5.3-python-sqlite-freeze` returns the tag
  - [ ] `git cat-file -t v0.5.3-python-sqlite-freeze` returns "tag" (not "commit")

  **QA Scenario**:
  ```
  Tool: Bash
  Steps:
    1. git tag -l | grep -c "v0.5.3-python-sqlite-freeze" → expected: 1
    2. git cat-file -t v0.5.3-python-sqlite-freeze → expected: "tag"
  Evidence: .sisyphus/evidence/t2-tag-created.txt
  ```

  **Commit**: NO — tag is separate from commit

---

- [ ] 3. Create `legacy/python-sqlite` branch

  **What to do**:
  1. Create branch from current HEAD: `git checkout -b legacy/python-sqlite`
  2. Push to remote: `git push origin legacy/python-sqlite`
  3. Verify: `git branch -a | grep legacy/python-sqlite`

  **Must NOT do**:
  - Do NOT modify branch after creation (read-only branch)
  - Do NOT force push

  **Recommended Agent Profile**:
  - **Category**: `quick`

  **Parallelization**:
  - **Can Run In Parallel**: YES — can run in parallel with T4 (both branch from same commit)
  - **Blocked By**: T1, T2
  - **Blocks**: None

  **Acceptance Criteria**:
  - [ ] Branch exists locally and on remote
  - [ ] Branch HEAD matches tag commit

  **QA Scenario**:
  ```
  Tool: Bash
  Steps:
    1. git branch -a | grep -c "legacy/python-sqlite" → expected: 2 (local + remote)
    2. git rev-parse legacy/python-sqlite == git rev-parse v0.5.3-python-sqlite-freeze
  Evidence: .sisyphus/evidence/t3-legacy-branch.txt
  ```

  **Commit**: NO — branch creation only

---

- [ ] 4. Create `migration/go-postgres` branch

  **What to do**:
  1. Checkout main (or stay on current branch)
  2. Create branch: `git checkout -b migration/go-postgres`
  3. Push to remote: `git push origin migration/go-postgres`
  4. Verify branch exists

  **Must NOT do**:
  - Do NOT add Go/PostgreSQL/Docker files to this branch during Phase 0
  - Do NOT force push

  **Recommended Agent Profile**:
  - **Category**: `quick`

  **Parallelization**:
  - **Can Run In Parallel**: YES — with T3
  - **Blocked By**: T1, T2
  - **Blocks**: T5-T14

  **Acceptance Criteria**:
  - [ ] Branch exists locally and on remote
  - [ ] Branch HEAD matches tag commit

  **QA Scenario**:
  ```
  Tool: Bash
  Steps:
    1. git branch -a | grep -c "migration/go-postgres" → expected: 2
    2. git rev-parse migration/go-postgres == git rev-parse v0.5.3-python-sqlite-freeze
  Evidence: .sisyphus/evidence/t4-migration-branch.txt
  ```

  **Commit**: NO — branch creation only

---

- [ ] 5. Export SQLite schema + row counts

  **What to do**:
  1. Create `scripts/migration/export_schema.py`:
     - Connect to `data/olist_ops.db` via Python `sqlite3`
     - Query `sqlite_master` for all tables (type='table', name NOT LIKE 'sqlite_%')
     - Export `.schema` to `migration_baseline/sqlite_schema.sql`
  2. Create `scripts/migration/export_row_counts.py`:
     - Connect to same DB
     - Enumerate all tables from `sqlite_master`
     - For each table: `SELECT COUNT(*) FROM "table"`
     - Output JSON to `migration_baseline/table_counts.json`
  3. Run both scripts

  **Must NOT do**:
  - Do NOT use `sqlite3` CLI (not available in this environment)
  - Do NOT export the `.db` file itself (267MB, must not enter Git)
  - Do NOT hardcode table names — always query `sqlite_master`

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Reason**: Straightforward Python scripting, no external dependencies

  **Parallelization**:
  - **Can Run In Parallel**: YES — with T6, T7, T8
  - **Blocked By**: T4
  - **Blocks**: T9 (README generation), T11-T14

  **References**:
  - Database path: `data/olist_ops.db` (16 tables confirmed)
  - Python sqlite3 docs for `sqlite_master` query pattern

  **Acceptance Criteria**:
  - [ ] `migration_baseline/sqlite_schema.sql` exists and contains 16+ `CREATE TABLE` statements
  - [ ] `migration_baseline/table_counts.json` exists, is valid JSON, has 16 keys
  - [ ] Sum of all row counts == 906,862 (verified)

  **QA Scenario**:
  ```
  Tool: Bash
  Steps:
    1. grep -c "CREATE TABLE" migration_baseline/sqlite_schema.sql → expected: >= 16
    2. python3 -c "import json; d=json.load(open('migration_baseline/table_counts.json')); assert len(d)==16; print(sum(d.values()))" → expected: 906862
  Evidence: .sisyphus/evidence/t5-schema-export.txt
  ```

  **Commit**: NO — part of Commit 2 (baseline artifacts)

---

- [ ] 6. Export pipeline CSV samples

  **What to do**:
  1. Create `scripts/migration/export_pipeline_samples.py`:
     - Connect to `data/olist_ops.db`
     - Export core pipeline tables to CSV (with headers):
       - `metric_daily` → `pipeline_outputs/metric_daily_sample.csv`
       - `metric_dimension_daily` → `pipeline_outputs/metric_dimension_daily_sample.csv` (LIMIT 1000 due to size)
       - `alert_events` → `pipeline_outputs/alert_events_sample.csv`
       - `strategy_recommendations` → `pipeline_outputs/recommendations_sample.csv`
       - `action_tasks` → `pipeline_outputs/tasks_sample.csv`
       - `event_outbox` → `pipeline_outputs/outbox_sample.csv`
       - `pipeline_runs` → `pipeline_outputs/pipeline_runs_sample.csv`
       - `ingestion_batches` → `pipeline_outputs/ingestion_batches_sample.csv`
     - Skip empty tables (governance_checkpoints, governance_health_results, review_retro, qoder_jobs)
  2. Run script

  **Must NOT do**:
  - Do NOT export full `metric_dimension_daily` (693,602 rows) — use LIMIT 1000
  - Do NOT export DB file itself

  **Recommended Agent Profile**:
  - **Category**: `quick`

  **Parallelization**:
  - **Can Run In Parallel**: YES — with T5, T7, T8
  - **Blocked By**: T4

  **Acceptance Criteria**:
  - [ ] All 8 CSV files exist and have valid headers
  - [ ] Each CSV has >0 data rows (except skipped empty tables)
  - [ ] CSV format is valid (no malformed quoting)

  **QA Scenario**:
  ```
  Tool: Bash
  Steps:
    1. ls migration_baseline/pipeline_outputs/*.csv | wc -l → expected: 8
    2. head -1 migration_baseline/pipeline_outputs/metric_daily_sample.csv | grep -c "metric_date" → expected: 1
    3. wc -l migration_baseline/pipeline_outputs/alert_events_sample.csv → expected: > 1
  Evidence: .sisyphus/evidence/t6-pipeline-samples.txt
  ```

  **Commit**: NO — part of Commit 2

---

- [ ] 7. Export API response snapshots

  **What to do**:
  1. Create `scripts/migration/export_api_responses.py`:
     - Generate temporary API_BEARER_TOKEN: `python3 -c "import secrets; print(secrets.token_urlsafe(32))"`
     - Set `os.environ["API_BEARER_TOKEN"] = temp_token`
     - Use `fastapi.testclient.TestClient` (from `api.main:app`) instead of real uvicorn
     - Call endpoints and save JSON responses:
       - `GET /api/v1/health` → `api_responses/health.json`
       - `GET /api/v1/status` (auth) → `api_responses/status.json`
       - `GET /api/v1/alerts` (auth) → `api_responses/alerts.json`
       - `GET /api/v1/tasks` (auth) → `api_responses/tasks.json`
       - `GET /api/v1/outbox` (auth) → `api_responses/outbox.json`
       - `GET /api/v1/governance/status` (auth) → `api_responses/governance_status.json`
       - `GET /api/v1/qoder/context` (auth) → `api_responses/qoder_context.json`
     - For endpoints that fail (e.g., Feishu with missing credentials), record the error response as-is — this IS the baseline state
  2. Run script

  **Must NOT do**:
  - Do NOT start real uvicorn process (use TestClient)
  - Do NOT commit the temporary token anywhere
  - Do NOT "fix" failing endpoints — record failures as baseline

  **Recommended Agent Profile**:
  - **Category**: `quick`

  **Parallelization**:
  - **Can Run In Parallel**: YES — with T5, T6, T8
  - **Blocked By**: T4

  **Acceptance Criteria**:
  - [ ] All 7 JSON files exist
  - [ ] Each file contains valid JSON
  - [ ] `health.json` contains `{"status": "ok"}` or similar

  **QA Scenario**:
  ```
  Tool: Bash
  Steps:
    1. ls migration_baseline/api_responses/*.json | wc -l → expected: 7
    2. python3 -c "import json; d=json.load(open('migration_baseline/api_responses/health.json')); print(d.get('status'))" → expected: "ok"
    3. python3 -c "import json; [json.load(open(f)) for f in glob('migration_baseline/api_responses/*.json')]" → no errors
  Evidence: .sisyphus/evidence/t7-api-responses.txt
  ```

  **Commit**: NO — part of Commit 2

---

- [ ] 8. Export governance YAML snapshot

  **What to do**:
  1. Create `scripts/migration/export_config_snapshot.py`:
     - Discover all YAML files in `config/` directory
     - Copy each to `migration_baseline/configs_snapshot/`
     - Verify file count matches (should be 29 files)
     - Record any missing files in README
  2. Run script

  **Must NOT do**:
  - Do NOT parse/resolve environment variables in YAML
  - Do NOT filter files — copy ALL config YAMLs

  **Recommended Agent Profile**:
  - **Category**: `quick`

  **Parallelization**:
  - **Can Run In Parallel**: YES — with T5, T6, T7
  - **Blocked By**: T4

  **Acceptance Criteria**:
  - [ ] `migration_baseline/configs_snapshot/` contains 29 YAML files
  - [ ] Key files present: `aip_object_schema.yml`, `alert_rules.yml`, `data_classification.yml`, `data_lineage.yml`, `access_policy.yml`, `checkpoint_rules.yml`, `health_checks.yml`, `decision_eval_rules.yml`

  **QA Scenario**:
  ```
  Tool: Bash
  Steps:
    1. ls migration_baseline/configs_snapshot/*.yml | wc -l → expected: 29
    2. ls migration_baseline/configs_snapshot/aip_object_schema.yml → expected: exists
    3. ls migration_baseline/configs_snapshot/alert_rules.yml → expected: exists
  Evidence: .sisyphus/evidence/t8-config-snapshot.txt
  ```

  **Commit**: NO — part of Commit 2

---

- [ ] 9. Generate `migration_baseline/README.md`

  **What to do**:
  1. Create `migration_baseline/README.md` with:
     - Baseline Git Reference (tag, branches)
     - Captured Artifacts list (schema, row counts, pipeline outputs, API responses, configs)
     - Purpose statement (for Go+PostgreSQL parity verification)
     - Known Missing Items section (populate from export script outputs)
     - Notes (do not delete until migration reaches production parity)
  2. Record any missing tables/files/API responses

  **Must NOT do**:
  - Do NOT leave Known Missing Items empty if there are gaps

  **Recommended Agent Profile**:
  - **Category**: `writing`

  **Parallelization**:
  - **Can Run In Parallel**: NO — needs T5-T8 results
  - **Blocked By**: T5, T6, T7, T8
  - **Blocks**: T12

  **Acceptance Criteria**:
  - [ ] README exists and is non-empty
  - [ ] All artifact directories are documented
  - [ ] Known Missing Items accurately reflects any gaps

  **QA Scenario**:
  ```
  Tool: Bash
  Steps:
    1. test -f migration_baseline/README.md → expected: 0 (success)
    2. grep -c "Captured Artifacts" migration_baseline/README.md → expected: 1
    3. grep -c "Known Missing Items" migration_baseline/README.md → expected: 1
  Evidence: .sisyphus/evidence/t9-readme-generated.txt
  ```

  **Commit**: NO — part of Commit 2

---

- [ ] 10. Write `docs/migration/go-postgres-migration-plan.md`

  **What to do**:
  1. Write comprehensive migration plan document with:
     - Goal statement
     - Baseline references (tag, branches, baseline directory)
     - Non-goals for Phase 0 (no Go/PostgreSQL implementation)
     - 7 migration phases:
       - Phase 1: Docker Compose + PostgreSQL foundation
       - Phase 2: PostgreSQL schema (raw, dwd, mart, ops, gov, ai, audit schemas)
       - Phase 3: Pipeline migration (CSV ingest, DWD build, metrics, alerts, recommendations)
       - Phase 4: Go API migration (health, status, alerts, tasks, outbox, governance, qoder)
       - Phase 5: Governance and ontology runtime (ObjectRegistry, ObjectQueryService, GovernanceService)
       - Phase 6: Outbox worker and adapters
       - Phase 7: LLM decision layer
     - Acceptance criteria (parity checks against Phase 0 baseline)
     - Branch strategy
     - Deletion policy

  **Must NOT do**:
  - Do NOT write Go implementation details (those come in later phases)
  - Do NOT include PostgreSQL schema DDL (that comes in Phase 2)

  **Recommended Agent Profile**:
  - **Category**: `writing`

  **Parallelization**:
  - **Can Run In Parallel**: NO — sequential after T9
  - **Blocked By**: T9
  - **Blocks**: T13

  **Acceptance Criteria**:
  - [ ] Document exists and is >100 lines
  - [ ] Contains all 7 phases with descriptions
  - [ ] Contains acceptance criteria section
  - [ ] Contains branch strategy and deletion policy

  **QA Scenario**:
  ```
  Tool: Bash
  Steps:
    1. test -f docs/migration/go-postgres-migration-plan.md → expected: 0
    2. grep -c "Phase 1" docs/migration/go-postgres-migration-plan.md → expected: 1
    3. grep -c "Acceptance Criteria" docs/migration/go-postgres-migration-plan.md → expected: 1
    4. wc -l docs/migration/go-postgres-migration-plan.md → expected: > 100
  Evidence: .sisyphus/evidence/t10-migration-plan.txt
  ```

  **Commit**: NO — part of Commit 3

---

- [ ] 11. Commit 1: Add migration scripts

  **What to do**:
  1. Stage `scripts/migration/` directory (5 export scripts)
  2. Commit: `git add scripts/migration/ && git commit -m "chore: add migration baseline capture scripts"`

  **Must NOT do**:
  - Do NOT stage any other files in this commit

  **Recommended Agent Profile**:
  - **Category**: `quick`

  **Parallelization**:
  - **Can Run In Parallel**: NO — sequential
  - **Blocked By**: T5-T10
  - **Blocks**: T12

  **Acceptance Criteria**:
  - [ ] Commit exists with only `scripts/migration/` changes
  - [ ] Commit message matches convention

  **QA Scenario**:
  ```
  Tool: Bash
  Steps:
    1. git log --oneline -1 | grep -c "migration baseline capture scripts" → expected: 1
    2. git show --name-only --oneline HEAD | grep -c "scripts/migration" → expected: >= 5
  Evidence: .sisyphus/evidence/t11-commit-scripts.txt
  ```

  **Commit**: YES — `chore: add migration baseline capture scripts`

---

- [ ] 12. Commit 2: Capture baseline artifacts

  **What to do**:
  1. Stage `migration_baseline/` directory
  2. Commit: `git add migration_baseline/ && git commit -m "chore: capture python sqlite migration baseline"`

  **Must NOT do**:
  - Do NOT stage any other files in this commit

  **Recommended Agent Profile**:
  - **Category**: `quick`

  **Parallelization**:
  - **Can Run In Parallel**: NO — sequential after T11
  - **Blocked By**: T11
  - **Blocks**: T13

  **Acceptance Criteria**:
  - [ ] Commit exists with only `migration_baseline/` changes
  - [ ] Commit message matches convention

  **QA Scenario**:
  ```
  Tool: Bash
  Steps:
    1. git log --oneline -1 | grep -c "capture python sqlite migration baseline" → expected: 1
    2. git show --name-only --oneline HEAD | grep -c "migration_baseline" → expected: >= 1
  Evidence: .sisyphus/evidence/t12-commit-baseline.txt
  ```

  **Commit**: YES — `chore: capture python sqlite migration baseline`

---

- [ ] 13. Commit 3: Add migration plan document

  **What to do**:
  1. Stage `docs/migration/` directory
  2. Commit: `git add docs/migration/ && git commit -m "docs: add go postgres migration plan"`

  **Must NOT do**:
  - Do NOT stage any other files in this commit

  **Recommended Agent Profile**:
  - **Category**: `quick`

  **Parallelization**:
  - **Can Run In Parallel**: NO — sequential after T12
  - **Blocked By**: T12
  - **Blocks**: T14

  **Acceptance Criteria**:
  - [ ] Commit exists with only `docs/migration/` changes
  - [ ] Commit message matches convention

  **QA Scenario**:
  ```
  Tool: Bash
  Steps:
    1. git log --oneline -1 | grep -c "add go postgres migration plan" → expected: 1
    2. git show --name-only --oneline HEAD | grep -c "docs/migration" → expected: 1
  Evidence: .sisyphus/evidence/t13-commit-docs.txt
  ```

  **Commit**: YES — `docs: add go postgres migration plan`

---

- [ ] 14. Push `migration/go-postgres` branch to remote

  **What to do**:
  1. Ensure on `migration/go-postgres` branch
  2. Push: `git push origin migration/go-postgres`
  3. Verify: `git branch -a | grep migration/go-postgres`

  **Must NOT do**:
  - Do NOT push to `legacy/python-sqlite` (should already be pushed)
  - Do NOT force push

  **Recommended Agent Profile**:
  - **Category**: `quick`

  **Parallelization**:
  - **Can Run In Parallel**: NO — final step
  - **Blocked By**: T13
  - **Blocks**: F1-F4

  **Acceptance Criteria**:
  - [ ] `migration/go-postgres` branch pushed to origin
  - [ ] Remote branch shows all 3 commits + cleanup commit

  **QA Scenario**:
  ```
  Tool: Bash
  Steps:
    1. git branch -a | grep "remotes/origin/migration/go-postgres" → expected: 1 match
    2. git log --oneline origin/migration/go-postgres | wc -l → expected: >= 4
  Evidence: .sisyphus/evidence/t14-branch-pushed.txt
  ```

  **Commit**: NO — push only

---

## Final Verification Wave

- [ ] F1. **Git Compliance Audit**

  **What to verify**:
  - Tag `v0.5.3-python-sqlite-freeze` exists and is annotated
  - Branches `legacy/python-sqlite` and `migration/go-postgres` exist on remote
  - `migration/go-postgres` has 4+ commits (cleanup + 3 migration commits)
  - No modifications to existing application code

  **Agent**: `quick`
  **QA**:
  ```bash
  git tag -l | grep -c "v0.5.3-python-sqlite-freeze"  # expected: 1
  git cat-file -t v0.5.3-python-sqlite-freeze          # expected: "tag"
  git branch -a | grep -c "legacy/python-sqlite"       # expected: 2
  git branch -a | grep -c "migration/go-postgres"      # expected: 2
  git log --oneline origin/migration/go-postgres | wc -l  # expected: >= 4
  ```

- [ ] F2. **Baseline Artifact Completeness**

  **What to verify**:
  - `migration_baseline/README.md` exists
  - `migration_baseline/sqlite_schema.sql` has 16+ CREATE TABLE
  - `migration_baseline/table_counts.json` has 16 keys, valid JSON
  - `migration_baseline/pipeline_outputs/*.csv` — 8 files
  - `migration_baseline/api_responses/*.json` — 7 files
  - `migration_baseline/configs_snapshot/*.yml` — 29 files

  **Agent**: `quick`
  **QA**:
  ```bash
  test -f migration_baseline/README.md
  grep -c "CREATE TABLE" migration_baseline/sqlite_schema.sql
  python3 -c "import json; d=json.load(open('migration_baseline/table_counts.json')); assert len(d)==16"
  ls migration_baseline/pipeline_outputs/*.csv | wc -l   # expected: 8
  ls migration_baseline/api_responses/*.json | wc -l     # expected: 7
  ls migration_baseline/configs_snapshot/*.yml | wc -l   # expected: 29
  ```

- [ ] F3. **API Response Validity**

  **What to verify**:
  - All JSON files are valid
  - `health.json` contains expected fields
  - No syntax errors in any response file

  **Agent**: `quick`
  **QA**:
  ```bash
  python3 -c "import json, glob; [json.load(open(f)) for f in glob('migration_baseline/api_responses/*.json')]"
  python3 -c "import json; d=json.load(open('migration_baseline/api_responses/health.json')); assert 'status' in d"
  ```

- [ ] F4. **Scope Fidelity Check**

  **What to verify**:
  - No changes to `api/`, `services/`, `adapters/`, `core/`, `pipeline/` (except new files in `scripts/migration/`)
  - No `.db` files in git
  - No Go/PostgreSQL/Docker files introduced

  **Agent**: `quick`
  **QA**:
  ```bash
  git diff --name-only HEAD~4..HEAD | grep -E "^(api|services|adapters|core|pipeline)/" | grep -v "scripts/migration"  # expected: empty
  git ls-files | grep "\.db$"  # expected: empty
  git ls-files | grep -E "\.(go|sql|dockerfile|yml)$" | grep -v "migration_baseline" | grep -v "scripts/migration"  # expected: no new Go/Docker files
  ```

---

## Commit Strategy

- **T1**: `chore: commit pre-freeze changes (64 files)`
- **T11**: `chore: add migration baseline capture scripts`
- **T12**: `chore: capture python sqlite migration baseline`
- **T13**: `docs: add go postgres migration plan`

## Success Criteria

### Verification Commands
```bash
# Tag exists and is annotated
git tag -l v0.5.3-python-sqlite-freeze

# Branches exist
git branch -a | grep -E "legacy/python-sqlite|migration/go-postgres"

# Baseline artifacts exist
ls migration_baseline/README.md
ls migration_baseline/sqlite_schema.sql
ls migration_baseline/table_counts.json
ls migration_baseline/pipeline_outputs/*.csv
ls migration_baseline/api_responses/*.json
ls migration_baseline/configs_snapshot/*.yml

# Schema has 16+ tables
grep -c "CREATE TABLE" migration_baseline/sqlite_schema.sql

# Row counts JSON is valid
python3 -c "import json; d=json.load(open('migration_baseline/table_counts.json')); print(len(d), 'tables')"

# API responses are valid JSON
python3 -c "import json; json.load(open('migration_baseline/api_responses/health.json'))"

# Migration plan exists
ls docs/migration/go-postgres-migration-plan.md
```

### Final Checklist
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent
- [ ] Git tag points to correct commit
- [ ] Branches exist on remote
- [ ] No code modifications outside migration scripts
