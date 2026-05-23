# Phase I-Local: Full Data + Local AI Decision Closed-Loop

## TL;DR

> **Quick Summary**: Add `--mode full` parameter across the existing pipeline to process ALL 634 days of Olist data, generate 8+ expanded alert types, power an LLM-based AI Decision Engine with structured outputs, and populate all 5 Feishu sandbox tables — including a previously empty review_retro table — to validate the end-to-end "data → AI decision → Feishu workbench → feedback loop" workflow.
>
> **Deliverables**:
> - `--mode full` on 4 existing scripts (calculate_daily_metrics, run_alert_detection, build_aip_context_bundle, generate_feishu_sandbox)
> - New `scripts/run_ai_decision_engine.py` with OpenAI-compatible LLM structured output
> - Expanded alert rules: 5→11 in config, 4→8 active handlers
> - 5-10 deterministic review_retro sample records
> - `config/llm_config.yml` for LLM endpoint/model/auth configuration
> - All outputs use `_full` suffix to avoid daily mode pollution
>
> **Estimated Effort**: Large (~14 tasks, 5 waves)
> **Parallel Execution**: YES — 4 waves with max 5 parallel tasks per wave
> **Critical Path**: T1→T2→T3→T6→T7→T8→T13→T14

---

## Context

### Original Request
User provided detailed proposal for "Phase I-Local: Full Data + Local AI Decision Closed-Loop Verification." The core goal: use full historical Olist data (2016-2018, 634 days) to test whether Feishu can serve as an AI decision workbench — generating structured decisions, assigning tasks, enabling human feedback, and closing the loop via review/retro analysis.

### Interview Summary
**Key Discussions**:
- **AI Engine**: LLM API call (OpenAI/Claude compatible), not rule-based templates
- **Script Strategy**: Add `--mode full` to existing scripts (not parallel `_full` copies)
- **Data Processing**: Full mode bypasses `ingestion_state.json`, processes ALL data directly from intermediate tables
- **Test Strategy**: No automated tests (pytest) — agent-executed QA scenarios only
- **Feishu Sync**: Phase I = LOCAL CSV ONLY (no cloud API calls). Phase H already handles cloud sync.

**Research Findings**:
- `scripts/run_daily_pipeline.py` orchestrates 8 sequential steps with subprocess isolation
- `scripts/calculate_daily_metrics.py` reads `ingestion_state.json` for `as_of_date` cutoff — this is the primary blocker for full mode
- `scripts/run_alert_detection.py` has 4 active rules + 1 disabled; uses a `RULE_HANDLERS` dict pattern
- `scripts/run_wake_agent.py` is entirely rule-based — full mode replaces this with `run_ai_decision_engine.py`
- `generate_feishu_sandbox.py` generates 5 CSVs but `review_retro` is always empty (returns `make_empty_csv`)
- `config/feishu_base_schema.yml` defines 5 tables with 54 total fields — schema complete
- `data/system/ingestion_state.json` is read by 3 scripts; full mode must NOT write to it

### Metis Review
**Identified Gaps** (addressed):
- **State file collision**: Full mode is READ-ONLY on `ingestion_state.json` — no separate state file. Only `_full` output files are written.
- **3 dimension-level alert rules** (`category_decline`, `region_anomaly`, `marketing_seller_gap`): Deferred to Phase II. Require per-dimension aggregations not available in `daily_metrics.csv`. v1 implements 8 of 11 rules.
- **Dashboard scope creep**: Explicitly EXCLUDED from Phase I. No dashboard code exists in the project.
- **Time window mismatch**: 7d/14d windows make no sense for 634-day full dataset. Full mode uses **monthly snapshots** + **full-range aggregates** instead.
- **Review retro generation**: Deterministic templates (not LLM-generated) for consistency.
- **Wake agent replacement**: Full mode completely replaces `run_wake_agent.py` — AI Decision Engine generates all 4 outputs.
- **alert_count field**: Added as metric #13 in daily_metrics_full.csv.

---

## Work Objectives

### Core Objective
Implement `--mode full` that uses complete 2016-2018 Olist data to generate enriched metrics, expanded alerts, LLM-structured AI decisions, and fully populated Feishu sandbox tables — validating the complete closed-loop decision workflow.

### Concrete Deliverables
- `data/ads/daily_metrics_full.csv` — 634 rows × 14 columns (13 metrics + alert_count)
- `data/ads/metric_alerts_full.csv` — 50-200 alerts across 8+ anomaly types
- `data/aip/aip_context_bundle_full.json` — monthly snapshots + full-range aggregates
- `outputs/ai/decision_report.md` — LLM-generated narrative report
- `outputs/ai/strategy_recommendations.json` — Top-20 structured strategies
- `outputs/ai/action_tasks.json` — Top-40 tasks derived from strategies
- `outputs/ai/review_retro_draft.json` — 5-10 deterministic review samples
- `data/feishu/daily_metrics_for_feishu_full.csv` — Full series for Feishu table 1
- `data/feishu/alert_events_for_feishu_full.csv` — Feishu table 2
- `data/feishu/strategy_recommendations_for_feishu_full.csv` — Feishu table 3
- `data/feishu/action_tasks_for_feishu_full.csv` — Feishu table 4
- `data/feishu/execution_reviews_for_feishu_full.csv` — Feishu table 5 (no longer empty)
- `config/llm_config.yml` — LLM endpoint/model/auth configuration

### Definition of Done
- [ ] `daily_metrics_full.csv` has 600-650 rows, 14 columns, all metrics non-null
- [ ] `metric_alerts_full.csv` has ≥10 alerts across ≥7 rule types
- [ ] AI Decision Engine generates ≥15 valid strategies with all 8 quality criteria met
- [ ] All 5 Feishu CSVs match field count from `feishu_base_schema.yml`
- [ ] `execution_reviews_for_feishu_full.csv` has 5-10 non-empty rows
- [ ] Daily mode still produces identical output after full mode changes (`daily_metrics.csv` unchanged)
- [ ] `ingestion_state.json` is never modified by full mode

### Must Have
- `--mode full` on `calculate_daily_metrics.py`, `run_alert_detection.py`, `build_aip_context_bundle.py`, `generate_feishu_sandbox.py`
- New `run_ai_decision_engine.py` with configurable OpenAI-compatible LLM endpoint
- 8+ alert types operational (up from 4)
- All `_full` output files use separate paths from daily mode
- Structured decision format: [问题][证据][判断][建议动作][预期收益][风险][验收指标]
- Top-N filtering with `impact_score` for alert prioritization
- Full mode is READ-ONLY on `ingestion_state.json`

### Must NOT Have (Guardrails)
- **G1**: NO Feishu cloud API calls — Phase I = local CSV only (Phase H handles cloud sync)
- **G2**: NO modification to `ingestion_state.json` in full mode
- **G3**: NO changes to daily mode output paths or behavior
- **G4**: NO pytest, unittest, or automated test infrastructure
- **G5**: NO dashboard code or UI changes
- **G6**: NO `simulate_daily_ingestion.py` or `run_daily_pipeline.py` modification
- **G7**: NO parallel `_full` script copies — use `--mode` parameter on existing scripts
- **G8**: NO dimension-level alerts in v1 (category_decline, region_anomaly, marketing_seller_gap deferred)
- **G9**: LLM call failure → fallback to heuristic rules, log warning, do NOT halt pipeline

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed.

### Test Decision
- **Infrastructure exists**: NO (no pytest/test framework; data analysis project)
- **Automated tests**: None — agent-executed QA scenarios only
- **Framework**: N/A

### QA Policy
Every task MUST include agent-executed QA scenarios. Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **Script verification**: Bash — run Python scripts, capture stdout, verify exit code + output files
- **CSV validation**: Bash (python3 -c or pandas) — verify row counts, column counts, null checks
- **JSON validation**: Bash (python3 -c + jsonschema) — verify schema compliance
- **Schema compliance**: Bash — compare CSV column names to `feishu_base_schema.yml` field_ids

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Start Immediately — config + plumbing, MAX PARALLEL):
├── Task 1:  Expand alert_rules.yml (5→11 rules) + add owner_mapping entries [quick]
├── Task 2:  Create config/llm_config.yml + .env template [quick]
├── Task 3:  Add `--mode full` to calculate_daily_metrics.py [quick]
└── Task 4:  Define Pydantic AI decision output schemas [quick]

Wave 2 (After Wave 1 — core pipeline, MAX PARALLEL):
├── Task 5:  Add 7 new alert handler functions to run_alert_detection.py [deep]
├── Task 6:  Add `--mode full` to run_alert_detection.py + Top-N impact_score [deep]
├── Task 7:  Add `--mode full` to build_aip_context_bundle.py (monthly windows) [deep]
├── Task 8:  Create run_ai_decision_engine.py with LLM structured output [deep]
└── Task 9:  Add `--mode full` to generate_feishu_sandbox.py (read _full inputs) [quick]

Wave 3 (After Wave 2 — review retro + integration, MAX PARALLEL):
├── Task 10: Generate 5-10 deterministic review_retro sample records [quick]
├── Task 11: Pipeline orchestration: run_full_decision_pipeline.py wrapper [quick]
└── Task 12: End-to-end integration QA — run ALL steps sequentially [deep]

Wave FINAL (After ALL tasks — 4 parallel reviews, then user okay):
├── Task F1: Plan Compliance Audit (oracle)
├── Task F2: Code Quality Review (unspecified-high)
├── Task F3: Real Manual QA — run ALL QA scenarios (unspecified-high)
└── Task F4: Scope Fidelity Check (deep)
→ Present results → Get explicit user okay
```

### Dependency Matrix

| Task | Depends On | Blocks | Wave |
|------|-----------|--------|------|
| 1 | — | 5, 6 | 1 |
| 2 | — | 8 | 1 |
| 3 | — | 5, 6, 7 | 1 |
| 4 | — | 8 | 1 |
| 5 | 1, 3 | 6, 9 | 2 |
| 6 | 1, 3, 5 | 9, 11 | 2 |
| 7 | 3 | 11 | 2 |
| 8 | 2, 4, 7 | 9, 10, 11 | 2 |
| 9 | 5, 6, 8 | 10, 11 | 2 |
| 10 | 8, 9 | 11 | 3 |
| 11 | 6, 7, 8, 9, 10 | 12 | 3 |
| 12 | 11 | F1-F4 | 3 |

### Agent Dispatch Summary

- **Wave 1**: 4 tasks — T1-T4 → `quick`
- **Wave 2**: 5 tasks — T5-T7 → `deep`, T8 → `deep`, T9 → `quick`
- **Wave 3**: 3 tasks — T10 → `quick`, T11 → `quick`, T12 → `deep`
- **FINAL**: 4 tasks — F1 → `oracle`, F2 → `unspecified-high`, F3 → `unspecified-high`, F4 → `deep`

---

## TODOs

- [ ] 1. Expand alert_rules.yml (5→11 rules) + add owner_mapping entries

  **What to do**:
  - Add 7 new alert rules to `config/alert_rules.yml`: `gmv_spike`, `order_drop`, `low_review_cluster`, `seller_risk`, `category_decline`, `region_anomaly`, `marketing_seller_gap`
  - For each new rule, specify: `rule_id`, `metric`, `condition` (natural language for Phase I, Python handler in Task 5), `severity`, `owner_role`, `dimension`, `min_sample_size`, `description`
  - Note: `category_decline`, `region_anomaly`, `marketing_seller_gap` are defined in config but DEFERRED — their handlers return empty `[]` in Task 5 (needs dimension-level data not in daily_metrics.csv)
  - Add corresponding `owner_role` entries to `config/owner_mapping.yml` for new roles: `category_ops`, `logistics_ops`, `seller_ops`, `marketing_ops`
  - Keep all existing 5 rules unchanged — additive only

  **Must NOT do**:
  - Do NOT remove or rename any existing rule
  - Do NOT modify `run_alert_detection.py` in this task (config-only)

  **Recommended Agent Profile**:
  > Select category + skills based on task domain. Justify each choice.
  - **Category**: `quick`
    - Reason: Config file editing — straightforward YAML additions
  - **Skills**: [`writing-plans`]
    - `writing-plans`: Follow structured config patterns from existing rules

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2, 3, 4)
  - **Blocks**: Tasks 5, 6
  - **Blocked By**: None (can start immediately)

  **References** (CRITICAL):
  - `config/alert_rules.yml` — Existing 5 rule definitions to follow as template pattern
  - `config/owner_mapping.yml` — Existing owner_role definitions to extend
  - `config/feishu_base_schema.yml:55-106` — alert_events table field definitions for severity/owner_role options

  **Acceptance Criteria**:
  - [ ] `config/alert_rules.yml` has 11 rules (5 existing + 6 new active + 3 deferred)
  - [ ] `config/owner_mapping.yml` has new roles: category_ops, logistics_ops, seller_ops, marketing_ops
  - [ ] All existing rules unchanged (line-level diff confirms additive only)

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Config files are valid YAML with correct rule count
    Tool: Bash
    Preconditions: Working directory is project root
    Steps:
      1. python3 -c "import yaml; r=yaml.safe_load(open('config/alert_rules.yml')); rules=r['rules']; assert len(rules)>=11; print(f'OK: {len(rules)} rules')"
      2. python3 -c "import yaml; o=yaml.safe_load(open('config/owner_mapping.yml')); owners=o.get('owners',[]); assert len(owners)>=4; print(f'OK: {len(owners)} owner entries')"
    Expected Result: ≥11 rules in alert_rules.yml, ≥4 owner entries in owner_mapping.yml
    Failure Indicators: YAML parse error, rule count < 11
    Evidence: .sisyphus/evidence/task-1-config-valid.txt

  Scenario: Existing rules remain untouched
    Tool: Bash
    Preconditions: Git repo with pre-change state
    Steps:
      1. git diff config/alert_rules.yml | grep "^-" | head -20
      2. Assert: No lines REMOVED from existing 5 rules
    Expected Result: Only additions (+ lines), no deletions (- lines) in existing rule blocks
    Failure Indicators: Any existing rule_id removed or renamed
    Evidence: .sisyphus/evidence/task-1-no-removals.txt
  ```

  **Commit**: YES
  - Message: `feat(config): expand alert_rules 5→11 with owner mappings`
  - Files: `config/alert_rules.yml`, `config/owner_mapping.yml`

- [ ] 2. Create config/llm_config.yml + .env template

  **What to do**:
  - Create `config/llm_config.yml` with:
    ```yaml
    provider: openai  # or anthropic
    model: gpt-4o
    api_base: https://api.openai.com/v1
    temperature: 0.3
    max_tokens: 2000
    timeout_seconds: 60
    max_retries: 2
    fallback: rule_based  # on API failure
    ```
  - Include comments explaining each field
  - The actual API key is read from environment variable `LLM_API_KEY` (NOT stored in config)
  - Add `.env.example` template to project root:
    ```
    # LLM API Key for AI Decision Engine
    LLM_API_KEY=sk-your-key-here
    ```
  - Ensure `.env` is already in `.gitignore` (verify, add if missing)

  **Must NOT do**:
  - Do NOT store real API keys in config files
  - Do NOT hardcode the key in Python scripts

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple config file creation
  - **Skills**: [`writing-plans`]

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 3, 4)
  - **Blocks**: Task 8
  - **Blocked By**: None

  **References**:
  - `config/feishu_app.yml` — Existing config template pattern for sensitive credentials
  - `.gitignore` — Verify `.env` exclusion exists

  **Acceptance Criteria**:
  - [ ] `config/llm_config.yml` exists with all fields: provider, model, api_base, temperature, max_tokens, timeout_seconds, max_retries, fallback
  - [ ] `.env.example` exists with LLM_API_KEY placeholder
  - [ ] `.gitignore` includes `.env` entry

  **QA Scenarios**:

  ```
  Scenario: llm_config.yml is valid YAML with all required fields
    Tool: Bash
    Steps:
      1. python3 -c "import yaml; c=yaml.safe_load(open('config/llm_config.yml')); assert c['model']; assert c['api_base']; assert c['fallback']; print('OK: all fields present')"
    Expected Result: "OK: all fields present"
    Failure Indicators: YAML parse error, missing required field
    Evidence: .sisyphus/evidence/task-2-config-valid.txt

  Scenario: .env.example exists and .gitignore protects .env
    Tool: Bash
    Steps:
      1. test -f .env.example && echo "OK: .env.example exists" || echo "FAIL"
      2. grep -q "^\.env$" .gitignore && echo "OK: .env in gitignore" || echo "FAIL"
    Expected Result: Both checks pass
    Failure Indicators: .env.example missing, .env not in gitignore
    Evidence: .sisyphus/evidence/task-2-env-check.txt
  ```

  **Commit**: YES
  - Message: `feat(config): add llm_config.yml for AI decision engine`
  - Files: `config/llm_config.yml`, `.env.example`, `.gitignore` (if modified)

- [ ] 3. Add `--mode full` to calculate_daily_metrics.py

  **What to do**:
  - Add `import argparse` to `scripts/calculate_daily_metrics.py`
  - Add `--mode` CLI argument with choices `['daily', 'full']`, default `'daily'`
  - In `main()`: when `--mode full`:
    - Skip `get_as_of_date()` call entirely (do NOT read `ingestion_state.json`)
    - Skip `filter_to_cutoff()` — keep ALL rows from `calculate_metrics()`
    - Write output to `DAILY_METRICS_FULL_FILE` instead of `DAILY_METRICS_FILE`
    - Add `alert_count` field: for each date, count alerts from `metric_alerts_full.csv` if exists, else 0
    - Print mode in output: `[FULL MODE]` prefix
  - When `--mode daily`: preserve EXACT existing behavior (no changes)
  - Add `DAILY_METRICS_FULL_FILE` path to `scripts/config.py`:
    ```python
    DAILY_METRICS_FULL_FILE = os.path.join(ADS_DIR, 'daily_metrics_full.csv')
    ```

  **Must NOT do**:
  - Do NOT change daily mode output path or behavior
  - Do NOT modify `ingestion_state.json` in full mode
  - Do NOT change the signature of `calculate_metrics()`, `prepare_timestamps()`, or `load_data()`

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Single-script refactoring with clear branching logic
  - **Skills**: [`writing-plans`]

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2, 4)
  - **Blocks**: Tasks 5, 6, 7
  - **Blocked By**: None

  **References**:
  - `scripts/calculate_daily_metrics.py:117-142` — Current `get_as_of_date()` and `filter_to_cutoff()` logic to branch
  - `scripts/calculate_daily_metrics.py:165-198` — `main()` function to modify
  - `scripts/config.py:74` — `DAILY_METRICS_FILE` for reference path pattern
  - `scripts/run_daily_pipeline.py:54-76` — How daily pipeline calls this script (subprocess, no args currently)

  **Acceptance Criteria**:
  - [ ] `--mode daily` (default) produces identical output to current behavior
  - [ ] `--mode full` produces `daily_metrics_full.csv` with 600-650 rows and 14 columns
  - [ ] `alert_count` column exists with integer values
  - [ ] Full mode does NOT read or write `ingestion_state.json`

  **QA Scenarios**:

  ```
  Scenario: Daily mode still works identically
    Tool: Bash
    Preconditions: ingestion_state.json exists with valid date
    Steps:
      1. cp data/ads/daily_metrics.csv /tmp/daily_metrics_backup.csv
      2. python3 scripts/calculate_daily_metrics.py --mode daily
      3. diff <(head -5 /tmp/daily_metrics_backup.csv) <(head -5 data/ads/daily_metrics.csv)
    Expected Result: No diff — daily mode output unchanged
    Failure Indicators: Different output, different row count, script crash
    Evidence: .sisyphus/evidence/task-3-daily-unchanged.txt

  Scenario: Full mode generates correct row count and columns
    Tool: Bash
    Steps:
      1. python3 scripts/calculate_daily_metrics.py --mode full
      2. python3 -c "import pandas as pd; df=pd.read_csv('data/ads/daily_metrics_full.csv'); assert 600<=len(df)<=650, f'Expected 600-650 rows, got {len(df)}'; assert len(df.columns)==14, f'Expected 14 cols, got {len(df.columns)}'; assert 'alert_count' in df.columns; print(f'OK: {len(df)} rows, {len(df.columns)} cols')"
      3. python3 -c "import pandas as pd; df=pd.read_csv('data/ads/daily_metrics_full.csv'); d=df['simulated_date'].min(); print(f'Date range: {d} to {df[\"simulated_date\"].max()}')"
    Expected Result: 600-650 rows, 14 columns, date range 2016-09-04 to 2018-10-17
    Failure Indicators: Row count < 600 or > 650, missing alert_count column, script crash
    Evidence: .sisyphus/evidence/task-3-full-mode.txt

  Scenario: Full mode does not touch ingestion_state.json
    Tool: Bash
    Steps:
      1. cp data/system/ingestion_state.json /tmp/state_backup.json
      2. python3 scripts/calculate_daily_metrics.py --mode full
      3. diff data/system/ingestion_state.json /tmp/state_backup.json
    Expected Result: No diff — ingestion_state.json untouched
    Failure Indicators: File modified (different checksum or content)
    Evidence: .sisyphus/evidence/task-3-state-untouched.txt
  ```

  **Commit**: YES
  - Message: `feat(metrics): add --mode full to calculate_daily_metrics.py`
  - Files: `scripts/calculate_daily_metrics.py`, `scripts/config.py`

- [ ] 4. Define Pydantic AI decision output schemas

  **What to do**:
  - Create `scripts/ai_schemas.py` with Pydantic v2 models:
    - `StrategyRecommendation`: recommendation_id, event_id, title, detail (structured text with [问题][证据][判断][建议动作][预期收益][风险][验收指标]), target_object, expected_impact, risk_level, requires_approval, owner_role, decision_type, confidence, success_metric, impact_score
    - `ActionTask`: task_id, title, description, owner_role, source_event, source_strategy, priority, deadline, status
    - `ReviewRetro`: review_id, strategy_id, outcome, actual_impact, is_effective, lessons_learned, promote_to_rule, reviewed_at
    - `DecisionReport`: generated_at, mode, total_alerts, strategies_count, tasks_count, top_findings, recommendations_summary
  - Use `pydantic.BaseModel` with `Field(description=...)` for each field
  - Include `model_dump_json(indent=2)` serialization
  - Add `validate_strategy_detail()` function that checks the structured text has all 7 Chinese section headers

  **Must NOT do**:
  - Do NOT import these models in any existing script yet (only Task 8 uses them)
  - Do NOT require pydantic as a new dependency if already available; add to requirements if not

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Pure data model definition — no runtime logic
  - **Skills**: [`writing-plans`]

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2, 3)
  - **Blocks**: Task 8
  - **Blocked By**: None

  **References**:
  - `config/feishu_base_schema.yml:154-206` — recommendations table fields to match
  - `config/feishu_base_schema.yml:107-153` — action_tasks table fields to match
  - `config/feishu_base_schema.yml:207-238` — review_retro table fields to match
  - `scripts/run_wake_agent.py:35-48` — Existing ID generation pattern (`gen_id()`)

  **Acceptance Criteria**:
  - [ ] `scripts/ai_schemas.py` has 4 Pydantic models with all required fields
  - [ ] `StrategyRecommendation` has `decision_type`, `confidence`, `success_metric`, `impact_score` fields
  - [ ] `validate_strategy_detail()` correctly validates/denies structured text format

  **QA Scenarios**:

  ```
  Scenario: Pydantic models parse valid data correctly
    Tool: Bash
    Steps:
      1. python3 -c "
from scripts.ai_schemas import StrategyRecommendation
s = StrategyRecommendation(
    recommendation_id='rec_001',
    event_id='evt_001',
    title='Test Strategy',
    detail='【问题】test\n【证据】test\n【判断】test\n【建议动作】test\n【预期收益】test\n【风险】test\n【验收指标】test',
    target_object='seller_123',
    expected_impact='Reduce cancel rate',
    risk_level='medium',
    requires_approval=False,
    owner_role='seller_ops',
    decision_type='investigate',
    confidence='high',
    success_metric='cancel_rate_7d < 0.05',
    impact_score=8
)
print(s.model_dump_json(indent=2))
       "
    Expected Result: Valid JSON output, no exceptions
    Failure Indicators: Pydantic ValidationError, import error
    Evidence: .sisyphus/evidence/task-4-schema-valid.json

  Scenario: Schema rejects invalid data
    Tool: Bash
    Steps:
      1. python3 -c "
from scripts.ai_schemas import StrategyRecommendation
try:
    s = StrategyRecommendation(recommendation_id='rec_001')  # missing required fields
    print('FAIL: should have raised ValidationError')
except Exception as e:
    print(f'OK: correctly rejected - {type(e).__name__}')
       "
    Expected Result: "OK: correctly rejected - ValidationError"
    Failure Indicators: No exception raised, wrong exception type
    Evidence: .sisyphus/evidence/task-4-schema-reject.txt
  ```

  **Commit**: YES
  - Message: `feat(schemas): add Pydantic models for AI decision output`
  - Files: `scripts/ai_schemas.py`

- [ ] 5. Add 7 new alert handler functions to run_alert_detection.py

  **What to do**:
  - Add handler functions for NEW rules to `scripts/run_alert_detection.py`:
    - `check_gmv_spike()` — GMV 7d avg > prev_14d_avg * 1.20 (mirror of gmv_drop)
    - `check_order_drop()` — order_count 7d avg < prev_14d_avg * 0.80
    - `check_low_review_cluster()` — low_review_rate > 0.15 AND total_reviews >= 10
    - `check_seller_risk()` — seller_count 7d avg < prev_14d_avg * 0.70
  - Add STUB handlers for 3 deferred rules:
    - `check_category_decline()` — return [] (log "requires per-category aggregation, deferred to Phase II")
    - `check_region_anomaly()` — return [] (log "requires per-region aggregation, deferred to Phase II")
    - `check_marketing_seller_gap()` — return [] (log "requires marketing funnel data, deferred to Phase II")
  - Add `impact_score` computation to ALL handlers (existing + new):
    ```python
    def compute_impact_score(rule_id, current_value, baseline_value, affected_orders=None):
        scores = {'gmv_drop': 3, 'gmv_spike': 2, 'order_drop': 3, 'cancel_rate_spike': 2,
                  'late_delivery_spike': 3, 'review_score_drop': 2, 'low_review_cluster': 2,
                  'seller_risk': 3}
        return scores.get(rule_id, 1)
    ```
  - Register all 7 new handlers in `RULE_HANDLERS` dict

  **Must NOT do**:
  - Do NOT change existing handler logic (gmv_drop, late_delivery_spike, review_score_drop, cancel_rate_spike)
  - Do NOT implement actual dimension-level aggregation for deferred rules

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Requires understanding of existing alert detection patterns, time-series window logic, and careful handler implementation
  - **Skills**: [`writing-plans`]

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 6, 7, 8 — all modify different files)
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 6 (co-located in same file), Task 9
  - **Blocked By**: Tasks 1, 3

  **References**:
  - `scripts/run_alert_detection.py:65-119` — `check_gmv_drop()` as template pattern for gmv_spike and order_drop
  - `scripts/run_alert_detection.py:122-155` — `check_late_delivery_spike()` as template for threshold-based checks
  - `scripts/run_alert_detection.py:276-282` — `RULE_HANDLERS` dict to extend
  - `config/alert_rules.yml` — New rule definitions for condition text and metadata

  **Acceptance Criteria**:
  - [ ] 4 new active handler functions operational
  - [ ] 3 stub handlers return empty [] with log message
  - [ ] `impact_score` computed for EVERY alert (new and existing)
  - [ ] All 11 entries in `RULE_HANDLERS` dict

  **QA Scenarios**:

  ```
  Scenario: All 11 handlers registered and 8 produce results
    Tool: Bash
    Preconditions: daily_metrics_full.csv exists (from Task 3)
    Steps:
      1. python3 -c "
import sys; sys.path.insert(0,'.')
from scripts.run_alert_detection import RULE_HANDLERS
assert len(RULE_HANDLERS) == 11, f'Expected 11 handlers, got {len(RULE_HANDLERS)}'
print(f'OK: {len(RULE_HANDLERS)} handlers registered')
for k,v in RULE_HANDLERS.items():
    print(f'  {k}: {v.__name__}')
       "
    Expected Result: 11 handlers listed, including new ones: check_gmv_spike, check_order_drop, check_low_review_cluster, check_seller_risk
    Failure Indicators: Missing handlers, import error
    Evidence: .sisyphus/evidence/task-5-handlers-registered.txt

  Scenario: New active handlers generate alerts on full data
    Tool: Bash
    Preconditions: daily_metrics_full.csv exists
    Steps:
      1. python3 scripts/run_alert_detection.py --mode full
      2. python3 -c "
import pandas as pd
df = pd.read_csv('data/ads/metric_alerts_full.csv')
rules = df['rule_id'].unique()
print(f'Total alerts: {len(df)}')
print(f'Rule types: {list(rules)}')
print(f'Rule count: {len(rules)}')
# Check for new rule types
new_rules = {'gmv_spike','order_drop','low_review_cluster','seller_risk'}
found = [r for r in rules if r in new_rules]
print(f'New rules that fired: {found}')
       "
    Expected Result: ≥10 alerts total, ≥7 distinct rule_ids, at least 2 new rule types firing
    Failure Indicators: <10 alerts, only old rule types, script crash
    Evidence: .sisyphus/evidence/task-5-alert-types.txt
  ```

  **Commit**: YES
  - Message: `feat(alerts): add 7 new alert handler functions`
  - Files: `scripts/run_alert_detection.py`

- [ ] 6. Add `--mode full` to run_alert_detection.py + Top-N impact_score filtering

  **What to do**:
  - Add `argparse` with `--mode` (daily|full) and `--top-alerts` (int, default=200) arguments
  - In `main()`: when `--mode full`:
    - Skip `load_ingestion_state()` — read `daily_metrics_full.csv` instead of `daily_metrics.csv`
    - Use separate dedup scope: `metric_alerts_full.csv` instead of `metric_alerts.csv`
    - After generating all alerts, apply Top-N filtering:
      - Sort by `impact_score` DESC, then `severity` (high>medium>low), then `current_value` change magnitude
      - Limit to `--top-alerts` count (default 200)
      - Log how many were filtered out
    - Write to `METRIC_ALERTS_FULL_FILE`
  - Add `METRIC_ALERTS_FULL_FILE` to `scripts/config.py`
  - When `--mode daily`: preserve EXACT existing behavior

  **Must NOT do**:
  - Do NOT change daily mode dedup logic or output path

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Modifies alert detection pipeline's mode branching, dedup logic, and filtering
  - **Skills**: [`writing-plans`]

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 5, 7, 8)
  - **Parallel Group**: Wave 2 (but MUST run AFTER Task 5 in same file — sequential within file)
  - **Blocks**: Task 9, Task 11
  - **Blocked By**: Tasks 1, 3, 5

  **References**:
  - `scripts/run_alert_detection.py:285-319` — `load_existing_alerts()` and `append_alerts()` for dedup pattern
  - `scripts/run_alert_detection.py:347-389` — `main()` function to modify
  - `scripts/calculate_daily_metrics.py` — Task 3 pattern for `--mode full` branching
  - `scripts/config.py:75` — `METRIC_ALERTS_FILE` for path pattern

  **Acceptance Criteria**:
  - [ ] `--mode full` generates `metric_alerts_full.csv` with 10-200 rows
  - [ ] `--top-alerts 50` limits output to ≤50 rows
  - [ ] Daily mode dedup scope separate (full alerts don't pollute daily)
  - [ ] `--mode daily` produces identical output to pre-change

  **QA Scenarios**:

  ```
  Scenario: Full mode with Top-50 filtering
    Tool: Bash
    Preconditions: daily_metrics_full.csv exists
    Steps:
      1. python3 scripts/run_alert_detection.py --mode full --top-alerts 50
      2. python3 -c "
import pandas as pd
df = pd.read_csv('data/ads/metric_alerts_full.csv')
assert 10 <= len(df) <= 50, f'Expected 10-50, got {len(df)}'
# Check sorted by impact_score desc
if 'impact_score' in df.columns:
    scores = df['impact_score'].tolist()
    assert scores == sorted(scores, reverse=True), 'Not sorted by impact_score desc'
print(f'OK: {len(df)} alerts, sorted by impact_score')
       "
    Expected Result: 10-50 alerts, sorted by impact_score DESC
    Failure Indicators: >50 alerts, unsorted, script crash
    Evidence: .sisyphus/evidence/task-6-top50.txt

  Scenario: Daily mode unchanged
    Tool: Bash
    Steps:
      1. cp data/ads/metric_alerts.csv /tmp/alerts_backup.csv 2>/dev/null || touch /tmp/alerts_backup.csv
      2. python3 scripts/run_alert_detection.py --mode daily
      3. echo "Daily mode completed with exit code $?"
    Expected Result: Exit code 0, no crash
    Failure Indicators: Non-zero exit code, script crash
    Evidence: .sisyphus/evidence/task-6-daily-unchanged.txt
  ```

  **Commit**: YES (combined with Task 5 changes if not yet committed)
  - Message: `feat(alerts): add --mode full + Top-N impact_score filtering`
  - Files: `scripts/run_alert_detection.py`, `scripts/config.py`

- [ ] 7. Add `--mode full` to build_aip_context_bundle.py (monthly window logic)

  **What to do**:
  - Add `argparse` with `--mode` (daily|full) argument to `scripts/build_aip_context_bundle.py`
  - When `--mode full`:
    - Skip `resolve_snapshot_date()` (no `ingestion_state.json` read)
    - Instead of 7d/14d windows, generate **monthly snapshots**:
      - For each month in the data range, compute a metrics summary
      - Include full-range aggregates (overall means, totals, trends)
    - Read from `daily_metrics_full.csv` and `metric_alerts_full.csv`
    - Write output to `AIP_CONTEXT_BUNDLE_FULL_FILE`
  - When `--mode daily`: preserve EXACT existing behavior
  - Add `AIP_CONTEXT_BUNDLE_FULL_FILE` to `scripts/config.py`
  - Monthly snapshot format:
    ```json
    {
      "mode": "full",
      "generated_at": "...",
      "date_range": {"start": "2016-09-04", "end": "2018-10-17"},
      "total_days": 634,
      "monthly_snapshots": [
        {"month": "2016-09", "gmv": ..., "order_count": ..., ...},
        ...
      ],
      "full_range_summary": {
        "total_gmv": ..., "avg_daily_gmv": ..., "gmv_trend": "up",
        "total_orders": ..., "avg_review_score": ...
      },
      "top_alerts": [...],
      "metric_peaks": {...}
    }
    ```

  **Must NOT do**:
  - Do NOT modify daily mode's 7d/14d window logic
  - Do NOT read `ingestion_state.json` in full mode

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Requires new time window computation logic, structural change to context bundle format
  - **Skills**: [`writing-plans`]

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 5, 8)
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 8, Task 11
  - **Blocked By**: Task 3

  **References**:
  - `scripts/build_aip_context_bundle.py:50-71` — `compute_time_windows()` to replace/adapt for monthly
  - `scripts/build_aip_context_bundle.py:30-38` — `resolve_snapshot_date()` pattern to bypass in full mode
  - `scripts/build_aip_context_bundle.py:74-80` — `safe_float()` utility to reuse
  - `scripts/config.py:65-66` — `AIP_CONTEXT_BUNDLE_FILE` for path pattern

  **Acceptance Criteria**:
  - [ ] `--mode full` generates `aip_context_bundle_full.json` with monthly snapshots
  - [ ] Bundle includes `full_range_summary` with total_gmv, avg_daily_gmv, gmv_trend
  - [ ] `--mode daily` produces identical `aip_context_bundle.json` to pre-change
  - [ ] Full mode does NOT reference `ingestion_state.json`

  **QA Scenarios**:

  ```
  Scenario: Full mode generates monthly bundle
    Tool: Bash
    Preconditions: daily_metrics_full.csv exists
    Steps:
      1. python3 scripts/build_aip_context_bundle.py --mode full
      2. python3 -c "
import json
b = json.load(open('data/aip/aip_context_bundle_full.json'))
assert b['mode'] == 'full'
assert 'date_range' in b
assert 'monthly_snapshots' in b
assert 'full_range_summary' in b
assert 'total_gmv' in b['full_range_summary']
months = len(b['monthly_snapshots'])
assert months >= 24, f'Expected ≥24 months, got {months}'
print(f'OK: {months} monthly snapshots, total_gmv={b[\"full_range_summary\"][\"total_gmv\"]:.0f}')
       "
    Expected Result: ≥24 monthly snapshots, total_gmv > 0, all required sections present
    Failure Indicators: Missing sections, <24 months, script crash
    Evidence: .sisyphus/evidence/task-7-monthly-bundle.txt

  Scenario: Daily mode unchanged
    Tool: Bash
    Steps:
      1. cp data/aip/aip_context_bundle.json /tmp/bundle_backup.json 2>/dev/null || echo "no existing bundle"
      2. python3 scripts/build_aip_context_bundle.py --mode daily
      3. echo "Daily mode completed with exit code $?"
    Expected Result: Exit code 0
    Failure Indicators: Non-zero exit code
    Evidence: .sisyphus/evidence/task-7-daily-unchanged.txt
  ```

  **Commit**: YES
  - Message: `feat(aip): add --mode full with monthly window logic`
  - Files: `scripts/build_aip_context_bundle.py`, `scripts/config.py`

- [ ] 8. Create run_ai_decision_engine.py with LLM structured output

  **What to do**:
  - Create `scripts/run_ai_decision_engine.py` — the core AI decision orchestrator
  - CLI: `--mode full`, `--top-alerts 20`, `--dry-run` (print prompt, skip API call)
  - **Step 1 — Load inputs**:
    - Read `aip_context_bundle_full.json` (or daily version based on mode)
    - Read `metric_alerts_full.csv`, sort by impact_score DESC, take top N
    - Read `config/llm_config.yml` for model/endpoint settings
    - Read `config/owner_mapping.yml` for role assignments
    - Read `config/action_registry.yml` for action templates
  - **Step 2 — Build LLM prompt**:
    - System prompt: "You are an e-commerce operations AI. Analyze the provided business metrics and alerts, generate structured strategy recommendations..."
    - User prompt: JSON context with metrics summary, top alerts, date range
    - Require structured output matching `StrategyRecommendation` schema (from Task 4)
  - **Step 3 — Call LLM API**:
    - Use `openai` Python SDK (OpenAI-compatible) with config from `llm_config.yml`
    - API key from env var `LLM_API_KEY`
    - Use `response_format={"type": "json_object"}` for structured output
    - On failure: log warning, fall back to rule-based heuristic strategies, do NOT halt
  - **Step 4 — Validate & save outputs**:
    - Parse LLM response into Pydantic models, validate with `StrategyRecommendation`
    - Filter: only keep strategies meeting all 8 quality criteria
    - Generate action tasks from strategies where `decision_type != monitor_only`
    - Write `outputs/ai/strategy_recommendations.json`
    - Write `outputs/ai/action_tasks.json`
    - Write `outputs/ai/decision_report.md` (narrative summary)
    - Write `outputs/ai/review_retro_draft.json` (empty for now — Task 10 fills)

  **Must NOT do**:
  - Do NOT hardcode API keys
  - Do NOT crash the pipeline on LLM failure (fall back to heuristics)
  - Do NOT call Feishu cloud APIs

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: New script, LLM integration, structured output parsing, error handling — most complex task
  - **Skills**: [`writing-plans`]

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 5, 7, 9)
  - **Parallel Group**: Wave 2
  - **Blocks**: Tasks 9, 10, 11
  - **Blocked By**: Tasks 2, 4, 7

  **References**:
  - `scripts/run_wake_agent.py:50-95` — Existing metric analysis pattern to study (will be replaced by LLM)
  - `scripts/run_wake_agent.py:35-48` — `gen_id()`, `save_json()`, `save_text()` utilities to reuse
  - `scripts/feishu_client.py` — Existing HTTP client pattern for API call structure
  - `config/llm_config.yml` — Created in Task 2
  - `scripts/ai_schemas.py` — Created in Task 4
  - `config/feishu_base_schema.yml:154-206` — recommendations table field requirements

  **Acceptance Criteria**:
  - [ ] Script runs without crash (or graceful fallback if no API key)
  - [ ] `outputs/ai/strategy_recommendations.json` has ≥15 valid strategies OR ≥5 heuristic fallback strategies
  - [ ] Each strategy has all 8 quality criteria in detail field
  - [ ] `outputs/ai/action_tasks.json` has tasks for non-monitor_only strategies
  - [ ] `outputs/ai/decision_report.md` is non-empty markdown
  - [ ] `--dry-run` prints the prompt without making API call

  **QA Scenarios**:

  ```
  Scenario: Dry-run mode prints prompt without API call
    Tool: Bash
    Steps:
      1. python3 scripts/run_ai_decision_engine.py --mode full --top-alerts 5 --dry-run 2>&1 | head -30
    Expected Result: Prints system prompt + user prompt content, no API error, exit code 0
    Failure Indicators: API connection error, crash before printing prompt
    Evidence: .sisyphus/evidence/task-8-dry-run.txt

  Scenario: Full run generates output files (with or without LLM key)
    Tool: Bash
    Preconditions: aip_context_bundle_full.json exists
    Steps:
      1. python3 scripts/run_ai_decision_engine.py --mode full --top-alerts 20 2>&1
      2. test -f outputs/ai/strategy_recommendations.json && echo "OK: strategies exist" || echo "FAIL"
      3. test -f outputs/ai/action_tasks.json && echo "OK: tasks exist" || echo "FAIL"
      4. test -f outputs/ai/decision_report.md && echo "OK: report exists" || echo "FAIL"
      5. python3 -c "
import json
s = json.load(open('outputs/ai/strategy_recommendations.json'))
print(f'Strategies: {len(s) if isinstance(s,list) else s.get(\"count\",\"unknown\")}')
       "
    Expected Result: All 3 output files exist, strategies count ≥ 5
    Failure Indicators: Missing output files, crash, 0 strategies
    Evidence: .sisyphus/evidence/task-8-outputs.txt

  Scenario: Each strategy has 8-part structured detail
    Tool: Bash
    Steps:
      1. python3 -c "
import json
s = json.load(open('outputs/ai/strategy_recommendations.json'))
items = s if isinstance(s, list) else s.get('recommendations', [])
required = ['【问题】','【证据】','【判断】','【建议动作】','【预期收益】','【风险】','【验收指标】']
for i, item in enumerate(items[:5]):
    detail = item.get('detail', '')
    missing = [r for r in required if r not in detail]
    if missing:
        print(f'Item {i}: MISSING {missing}')
    else:
        print(f'Item {i}: OK - all 7 sections present')
       "
    Expected Result: All checked items have all 7 section headers
    Failure Indicators: Missing section headers in detail field
    Evidence: .sisyphus/evidence/task-8-structured-detail.txt
  ```

  **Commit**: YES
  - Message: `feat(ai): create run_ai_decision_engine.py with LLM structured output`
  - Files: `scripts/run_ai_decision_engine.py`

- [ ] 9. Add `--mode full` to generate_feishu_sandbox.py (read _full inputs)

  **What to do**:
  - Add `argparse` with `--mode` (daily|full) to `scripts/generate_feishu_sandbox.py`
  - When `--mode full`:
    - Read from `_full` suffixed sources: `daily_metrics_full.csv`, `metric_alerts_full.csv`, `outputs/ai/strategy_recommendations.json`, `outputs/ai/action_tasks.json`
    - Output to `_full` suffixed Feishu CSVs: `daily_metrics_for_feishu_full.csv`, `alert_events_for_feishu_full.csv`, `strategy_recommendations_for_feishu_full.csv`, `action_tasks_for_feishu_full.csv`, `execution_reviews_for_feishu_full.csv`
    - For `strategy_recommendations`: map AI Decision Engine fields to Feishu schema (recommendation_id, event_id, title, detail, target_object, expected_impact, risk_level, requires_approval, owner → owner_role, approval_status, status, created_at)
    - For `action_tasks`: map task_id, title, description, owner_role → owner, source_event, source_strategy, priority, deadline, status
    - For `execution_reviews`: use `review_retro_draft.json` from Task 10 (NOT empty)
    - Generate `latest_daily_metrics_for_feishu_full.csv` using the LAST business date row
  - When `--mode daily`: preserve EXACT existing behavior
  - Add `_full` path constants to `scripts/config.py`

  **Must NOT do**:
  - Do NOT change daily mode output paths
  - Do NOT generate empty `execution_reviews_for_feishu_full.csv` (must read from review_retro_draft.json)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Follows existing transform pattern, just adds `_full` source/target path branching
  - **Skills**: [`writing-plans`]

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 5, 8 — different files)
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 10, Task 11
  - **Blocked By**: Tasks 5, 6, 8

  **References**:
  - `scripts/generate_feishu_sandbox.py:52-73` — `transform_daily_metrics()` pattern to replicate
  - `scripts/generate_feishu_sandbox.py:110-166` — `transform_strategy_recommendations()` pattern
  - `scripts/generate_feishu_sandbox.py:168-208` — `transform_action_tasks()` pattern
  - `scripts/generate_feishu_sandbox.py:233-237` — `transform_execution_reviews()` — currently always empty, needs modification
  - `config/feishu_base_schema.yml` — All 5 table field definitions
  - `config/feishu_field_mapping.yml` — Field name mappings

  **Acceptance Criteria**:
  - [ ] `--mode full` generates 5 non-empty `_full` Feishu CSVs
  - [ ] `execution_reviews_for_feishu_full.csv` no longer empty (5-10 rows after Task 10)
  - [ ] Each CSV column count matches `feishu_base_schema.yml` field count
  - [ ] `--mode daily` produces identical output to pre-change

  **QA Scenarios**:

  ```
  Scenario: Full mode generates all 5 Feishu CSVs
    Tool: Bash
    Preconditions: Tasks 3-8 completed, review_retro_draft.json exists (or empty)
    Steps:
      1. python3 scripts/generate_feishu_sandbox.py --mode full
      2. for f in data/feishu/*_full.csv; do echo "$f: $(wc -l < $f) rows"; done
    Expected Result: 5 CSV files in data/feishu/ with _full suffix, all have data rows
    Failure Indicators: Missing files, empty files (except review_retro before Task 10)
    Evidence: .sisyphus/evidence/task-9-feishu-csvs.txt

  Scenario: CSV columns match schema
    Tool: Bash
    Steps:
      1. python3 -c "
import yaml, pandas as pd, os
schema = yaml.safe_load(open('config/feishu_base_schema.yml'))
for table in schema['tables']:
    tid = table['table_id']
    if tid == 'review_retro': fname = 'execution_reviews_for_feishu_full.csv'
    elif tid == 'recommendations': fname = 'strategy_recommendations_for_feishu_full.csv'
    elif tid == 'alert_events': fname = 'alert_events_for_feishu_full.csv'
    else: fname = f'{tid}_for_feishu_full.csv'
    path = f'data/feishu/{fname}'
    if os.path.exists(path):
        df = pd.read_csv(path)
        expected = len(table['fields'])
        actual = len(df.columns)
        status = 'OK' if actual == expected else f'MISMATCH (expected {expected}, got {actual})'
        print(f'{tid}: {status}')
       "
    Expected Result: All 5 tables show "OK"
    Failure Indicators: Column count mismatch for any table
    Evidence: .sisyphus/evidence/task-9-schema-check.txt
  ```

  **Commit**: YES
  - Message: `feat(feishu): add --mode full to generate_feishu_sandbox.py`
  - Files: `scripts/generate_feishu_sandbox.py`, `scripts/config.py`

- [ ] 10. Generate 5-10 deterministic review_retro sample records

  **What to do**:
  - Create/update `scripts/generate_review_retro_samples.py` — generates `outputs/ai/review_retro_draft.json`
  - Generate 5-10 deterministic sample records (NOT LLM-generated, NOT random):
    - Each record pairs a strategy with a simulated outcome based on historical "hindsight"
    - For example: strategy targeting cancel_rate → check if cancel_rate actually dropped 7 days later in historical data
  - Record structure:
    ```json
    {
      "review_id": "rev_001",
      "strategy_id": "rec_xxx",
      "outcome": "已完成异常订单抽样排查，识别3个高取消率品类",
      "actual_impact": "后续7日取消率从8.9%回落到4.7%，降幅47%",
      "is_effective": true,
      "lessons_learned": "促销峰值后取消率异常应设置7日观察窗口，避免单日误报",
      "promote_to_rule": true,
      "reviewed_at": "2017-12-07"
    }
    ```
  - Hindsight logic: For each strategy with a target date, read `daily_metrics_full.csv` for the 7 days after that date and compute actual metric change
  - At least 3 records with `is_effective: true`, at least 2 with `is_effective: false` (mixed results)
  - At least 2 records with `promote_to_rule: true`
  - Integrate into `generate_feishu_sandbox.py` full mode path: read `review_retro_draft.json` for `transform_execution_reviews()`

  **Must NOT do**:
  - Do NOT use LLM API for generating these records (deterministic templates)
  - Do NOT generate random data (must be based on actual metric after-changes)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Deterministic data generation with simple CSV lookup
  - **Skills**: [`writing-plans`]

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 11)
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 11 (integrated into feishu sandbox)
  - **Blocked By**: Tasks 8, 9

  **References**:
  - `scripts/generate_feishu_sandbox.py:233-237` — `transform_execution_reviews()` — currently always empty
  - `config/feishu_base_schema.yml:207-238` — review_retro table field definitions
  - `data/ads/daily_metrics_full.csv` — Historical data for hindsight computation
  - `outputs/ai/strategy_recommendations.json` — Strategies to pair with reviews

  **Acceptance Criteria**:
  - [ ] `outputs/ai/review_retro_draft.json` has 5-10 records
  - [ ] At least 3 with `is_effective: true`, at least 2 with `is_effective: false`
  - [ ] At least 2 with `promote_to_rule: true`
  - [ ] All records have valid `strategy_id` referencing actual strategies

  **QA Scenarios**:

  ```
  Scenario: Review retro has correct record count and mixed results
    Tool: Bash
    Steps:
      1. python3 -c "
import json
data = json.load(open('outputs/ai/review_retro_draft.json'))
reviews = data if isinstance(data, list) else data.get('reviews', [])
assert 5 <= len(reviews) <= 10, f'Expected 5-10, got {len(reviews)}'
effective = sum(1 for r in reviews if r.get('is_effective'))
ineffective = len(reviews) - effective
promoted = sum(1 for r in reviews if r.get('promote_to_rule'))
print(f'Total: {len(reviews)} | Effective: {effective} | Ineffective: {ineffective} | Promoted: {promoted}')
       "
    Expected Result: 5-10 total, effective ≥ 3, ineffective ≥ 2, promoted ≥ 2
    Failure Indicators: Wrong count, all same outcome, no promoted rules
    Evidence: .sisyphus/evidence/task-10-retro-counts.txt

  Scenario: Review retro integrates into feishu sandbox
    Tool: Bash
    Preconditions: review_retro_draft.json exists
    Steps:
      1. python3 scripts/generate_feishu_sandbox.py --mode full
      2. python3 -c "
import pandas as pd
df = pd.read_csv('data/feishu/execution_reviews_for_feishu_full.csv')
assert len(df) >= 5, f'Expected ≥5 rows, got {len(df)}'
assert 'review_id' in df.columns
assert 'is_effective' in df.columns
print(f'OK: {len(df)} review rows in feishu CSV')
       "
    Expected Result: execution_reviews_for_feishu_full.csv has ≥5 rows (no longer empty)
    Failure Indicators: Empty CSV, missing columns
    Evidence: .sisyphus/evidence/task-10-feishu-integration.txt
  ```

  **Commit**: YES
  - Message: `feat(feishu): generate review_retro sample records`
  - Files: `scripts/generate_review_retro_samples.py`, `scripts/generate_feishu_sandbox.py` (if modified), `outputs/ai/review_retro_draft.json`

- [ ] 11. Pipeline orchestration: run_full_decision_pipeline.py wrapper

  **What to do**:
  - Create `scripts/run_full_decision_pipeline.py` — a standalone orchestrator for full mode
  - Does NOT modify `run_daily_pipeline.py` — completely separate
  - Steps executed sequentially (like daily pipeline pattern):
    1. `calculate_daily_metrics.py --mode full`
    2. `run_alert_detection.py --mode full --top-alerts 200`
    3. `build_aip_context_bundle.py --mode full`
    4. `run_ai_decision_engine.py --mode full --top-alerts 20`
    5. `generate_review_retro_samples.py` (Task 10 script)
    6. `generate_feishu_sandbox.py --mode full`
  - Each step runs as subprocess, exit code checked, failure logged
  - Prints step-by-step progress with timing
  - At end: summary of all output files with row counts
  - Add `run_full_decision_pipeline.py` to `README.md` under "运行方法"

  **Must NOT do**:
  - Do NOT modify `run_daily_pipeline.py`
  - Do NOT call Feishu cloud APIs (local CSV only for Phase I)
  - Do NOT skip steps on failure (report and continue)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple subprocess orchestration following existing `run_daily_pipeline.py` pattern
  - **Skills**: [`writing-plans`]

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 10)
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 12
  - **Blocked By**: Tasks 6, 7, 8, 9, 10

  **References**:
  - `scripts/run_daily_pipeline.py:1-92` — Existing pipeline orchestrator pattern to follow
  - `scripts/run_daily_pipeline.py:6-15` — `STEPS` list structure
  - `scripts/run_daily_pipeline.py:54-76` — Subprocess execution pattern with timing

  **Acceptance Criteria**:
  - [ ] `scripts/run_full_decision_pipeline.py` executes all 6 steps
  - [ ] Each step's output files verified to exist after execution
  - [ ] Summary printed at end with file paths and row counts
  - [ ] Non-zero exit from any step is logged but does NOT halt pipeline (except step 1 failure)

  **QA Scenarios**:

  ```
  Scenario: Full pipeline runs all steps
    Tool: Bash
    Steps:
      1. python3 scripts/run_full_decision_pipeline.py 2>&1 | tee /tmp/pipeline_output.txt
      2. grep -c "OK" /tmp/pipeline_output.txt
    Expected Result: At least 6 "OK" lines (one per step), exit code 0
    Failure Indicators: Crash, missing step outputs, exit code non-zero
    Evidence: .sisyphus/evidence/task-11-pipeline-run.txt

  Scenario: All output files exist after pipeline
    Tool: Bash
    Steps:
      1. for f in data/ads/daily_metrics_full.csv data/ads/metric_alerts_full.csv data/aip/aip_context_bundle_full.json outputs/ai/strategy_recommendations.json outputs/ai/action_tasks.json outputs/ai/review_retro_draft.json data/feishu/daily_metrics_for_feishu_full.csv data/feishu/alert_events_for_feishu_full.csv data/feishu/strategy_recommendations_for_feishu_full.csv data/feishu/action_tasks_for_feishu_full.csv data/feishu/execution_reviews_for_feishu_full.csv; do test -f "$f" && echo "OK: $f" || echo "MISSING: $f"; done
    Expected Result: All 11 files show "OK"
    Failure Indicators: Any file "MISSING"
    Evidence: .sisyphus/evidence/task-11-all-files.txt
  ```

  **Commit**: YES
  - Message: `feat(orchestration): add run_full_decision_pipeline.py wrapper`
  - Files: `scripts/run_full_decision_pipeline.py`, `README.md`

- [ ] 12. End-to-end integration QA — run ALL steps and verify outputs

  **What to do**:
  - Run the full pipeline: `python3 scripts/run_full_decision_pipeline.py`
  - Verify every output file exists with correct format
  - Verify daily mode still works: `python3 scripts/calculate_daily_metrics.py --mode daily` produces unchanged output
  - Verify `ingestion_state.json` is untouched by full mode
  - Verify schema compliance: all 5 Feishu CSVs match `feishu_base_schema.yml` column counts
  - Verify structured decision format: all strategies have 7-section detail
  - Generate integration test evidence file with all verification results
  - If any check fails, report exact failure and file path

  **Must NOT do**:
  - Do NOT modify any source code in this task (verification only)

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Comprehensive integration testing across all components
  - **Skills**: [`writing-plans`]

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Sequential (final integration gate before FINAL wave)
  - **Blocks**: F1-F4
  - **Blocked By**: Task 11

  **References**:
  - `.sisyphus/plans/phase-i-full-decision-mode.md` — This plan for acceptance criteria
  - `config/feishu_base_schema.yml` — Schema to validate against
  - `scripts/run_full_decision_pipeline.py` — Pipeline to execute

  **Acceptance Criteria**:
  - [ ] All 11 output files exist
  - [ ] `daily_metrics_full.csv`: 600-650 rows, 14 columns
  - [ ] `metric_alerts_full.csv`: ≥10 rows, ≥7 rule types
  - [ ] `strategy_recommendations.json`: ≥5 strategies
  - [ ] All 5 Feishu CSVs match schema column counts
  - [ ] Daily mode unchanged
  - [ ] `ingestion_state.json` untouched

  **QA Scenarios**:

  ```
  Scenario: Complete integration verification
    Tool: Bash
    Steps:
      1. python3 -c "
import os, json, pandas as pd, yaml

results = []
# Check 1: All files exist
files = [
    'data/ads/daily_metrics_full.csv',
    'data/ads/metric_alerts_full.csv',
    'data/aip/aip_context_bundle_full.json',
    'outputs/ai/strategy_recommendations.json',
    'outputs/ai/action_tasks.json',
    'outputs/ai/review_retro_draft.json',
]
for f in files:
    if os.path.exists(f):
        results.append(f'PASS: {f} exists')
    else:
        results.append(f'FAIL: {f} MISSING')

# Check 2: Row counts
df = pd.read_csv('data/ads/daily_metrics_full.csv')
if 600 <= len(df) <= 650: results.append(f'PASS: daily_metrics_full {len(df)} rows')
else: results.append(f'FAIL: daily_metrics_full {len(df)} rows (expected 600-650)')

# Check 3: Schema compliance
schema = yaml.safe_load(open('config/feishu_base_schema.yml'))
expected_cols = {t['table_id']: len(t['fields']) for t in schema['tables']}
feishu_map = {'daily_metrics':'daily_metrics_for_feishu_full.csv','alert_events':'alert_events_for_feishu_full.csv',
              'recommendations':'strategy_recommendations_for_feishu_full.csv','action_tasks':'action_tasks_for_feishu_full.csv',
              'review_retro':'execution_reviews_for_feishu_full.csv'}
for tid, fname in feishu_map.items():
    path = f'data/feishu/{fname}'
    if os.path.exists(path):
        actual = len(pd.read_csv(path).columns)
        expected = expected_cols[tid]
        if actual == expected: results.append(f'PASS: {tid} schema {actual}/{expected} cols')
        else: results.append(f'FAIL: {tid} schema {actual} vs {expected} cols')

for r in results: print(r)
       "
    Expected Result: All "PASS", no "FAIL"
    Failure Indicators: Any FAIL line
    Evidence: .sisyphus/evidence/task-12-integration-qa.txt

  Scenario: Daily mode preserved after all full mode changes
    Tool: Bash
    Steps:
      1. python3 scripts/calculate_daily_metrics.py --mode daily 2>&1
      2. test -f data/ads/daily_metrics.csv && echo "PASS: daily_metrics.csv still generated" || echo "FAIL"
      3. python3 -c "import json; s=json.load(open('data/system/ingestion_state.json')); print(f'State file intact: {list(s.keys())}')"
    Expected Result: Daily mode works, state file unchanged
    Failure Indicators: Daily mode crash, state file corruption
    Evidence: .sisyphus/evidence/task-12-daily-preserved.txt
  ```

  **Commit**: NO (verification only, evidence committed separately if needed)

---

## Final Verification Wave

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in .sisyphus/evidence/. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `python3 -c "import scripts.config; print('OK')"` on all modified scripts. Review all changed files for: bare `except:`, print-debugging left in, hardcoded paths (should use config.py), missing `--mode` argparse. Check AI slop: excessive comments, over-abstraction, generic variable names.
  Output: `Imports [PASS/FAIL] | Lint [PASS/FAIL] | Files [N clean/N issues] | VERDICT`

- [ ] F3. **Real Manual QA** — `unspecified-high`
  Start from clean state. Execute EVERY QA scenario from EVERY task — follow exact steps, capture evidence. Test cross-task integration (full pipeline run). Test edge cases: empty intermediate tables, missing config files, LLM API timeout. Save to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff (git log/diff). Verify 1:1 — everything in spec was built, nothing beyond spec was built. Check "Must NOT do" compliance. Detect cross-task contamination. Flag unaccounted changes.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

- **1**: `feat(config): expand alert_rules 5→11 with owner mappings` — config/alert_rules.yml, config/owner_mapping.yml
- **2**: `feat(config): add llm_config.yml for AI decision engine` — config/llm_config.yml
- **3**: `feat(metrics): add --mode full to calculate_daily_metrics.py` — scripts/calculate_daily_metrics.py
- **4**: `feat(schemas): add Pydantic models for AI decision output` — scripts/ai_schemas.py
- **5**: `feat(alerts): add 7 new alert handler functions` — scripts/run_alert_detection.py
- **6**: `feat(alerts): add --mode full + Top-N impact_score filtering` — scripts/run_alert_detection.py
- **7**: `feat(aip): add --mode full with monthly window logic` — scripts/build_aip_context_bundle.py
- **8**: `feat(ai): create run_ai_decision_engine.py with LLM structured output` — scripts/run_ai_decision_engine.py
- **9**: `feat(feishu): add --mode full to generate_feishu_sandbox.py` — scripts/generate_feishu_sandbox.py
- **10**: `feat(feishu): generate review_retro sample records` — scripts/generate_feishu_sandbox.py
- **11**: `feat(orchestration): add run_full_decision_pipeline.py wrapper` — scripts/run_full_decision_pipeline.py

---

## Success Criteria

### Verification Commands
```bash
# Full metrics
python3 scripts/calculate_daily_metrics.py --mode full
python3 -c "import pandas as pd; df=pd.read_csv('data/ads/daily_metrics_full.csv'); assert 600<=len(df)<=650; assert len(df.columns)==14; print(f'OK: {len(df)} rows, {len(df.columns)} cols')"

# Full alerts
python3 scripts/run_alert_detection.py --mode full
python3 -c "import pandas as pd; df=pd.read_csv('data/ads/metric_alerts_full.csv'); assert len(df)>=10; rules=df['rule_id'].nunique(); assert rules>=7; print(f'OK: {len(df)} alerts, {rules} rule types')"

# AI Decision Engine
python3 scripts/run_ai_decision_engine.py --mode full --top-alerts 20
python3 -c "import json; d=json.load(open('outputs/ai/strategy_recommendations.json')); assert len(d)>=15; print(f'OK: {len(d)} strategies')"

# Feishu CSVs
python3 scripts/generate_feishu_sandbox.py --mode full
python3 -c "
import os, pandas as pd
for f in ['daily_metrics_for_feishu_full.csv','alert_events_for_feishu_full.csv','strategy_recommendations_for_feishu_full.csv','action_tasks_for_feishu_full.csv','execution_reviews_for_feishu_full.csv']:
    df=pd.read_csv(f'data/feishu/{f}')
    print(f'{f}: {len(df)} rows')
print('OK: all 5 CSVs generated')
"

# Daily mode preserved
python3 scripts/calculate_daily_metrics.py
python3 -c "import pandas as pd; df=pd.read_csv('data/ads/daily_metrics.csv'); assert len(df)>0; print(f'Daily mode OK: {len(df)} rows')"
```

### Final Checklist
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent
- [ ] All 5 Feishu CSVs generated with correct schema
- [ ] Daily mode unchanged
- [ ] ingestion_state.json untouched
