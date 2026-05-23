# Phase Consolidation：项目收敛与稳定化 v0.1

## TL;DR

> **Quick Summary**: 将 37 个 Python 脚本、10+ 个阶段的项目收敛为 4 入口命令、4 大阶段、可复现环境、最小测试集的 **v0.1-heuristic-decision-sandbox** 版本。不新增功能，只做整理/校准/文档/测试。
>
> **Deliverables**:
> - `reports/project_status_matrix.md` — 模块状态矩阵
> - `README_RUNBOOK.md` — 运行手册
> - `requirements.txt` — 依赖声明
> - 4 个标准化入口命令（daily/full/sync/pull）
> - `decision_source` / `review_type` 字段追加
> - 最小 pytest 测试集（3 文件）
> - `reports/feishu_dashboard_validation.md` — 看板校准
> - `reports/v0.1_release_summary.md` — 版本报告
>
> **Estimated Effort**: Medium（9 个任务，4 waves）
> **Parallel Execution**: YES — max 5 tasks per wave
> **Critical Path**: T1 → T2 → T3 → T6 → T7 → T9

---

## Context

### Original Request
用户提出 **"Phase Consolidation：项目收敛与稳定化"** 方案，共 9 个收敛任务。核心诉求：把当前散乱的 10+ 个阶段、37 个脚本压缩为一个清晰、可运行、可验收的 v0.1 版本。状态重新定义为 **"规则驱动的 AI-ready 决策沙盘"**（非 LLM 驱动）。

### Interview Summary

**已确认决策**:
- **Phase 1-7 处理**: 方案 B（冻结为历史资产），只修复生成 3 个基础表的脚本
- **测试框架**: pytest
- **全量管道入口**: 新建 `scripts/run_full_pipeline.py` 作为轻量 wrapper，封装现有 --mode full 脚本
- **数据自举**: Kaggle 下载 → RUNBOOK 写明下载步骤
- **飞书范围**: v0.1 仅 dry-run 验证，不要求真实 API 凭证

**项目当前状态定义**（v0.1 基线）:
| 能力 | 状态 |
|------|------|
| 数据分析（Phase 1-7） | ✅ DONE（FROZEN 历史资产） |
| 指标与异常检测 | ✅ DONE |
| 规则/启发式策略生成 | ✅ DONE |
| 飞书多维表格承接 | ✅ PARTIAL（已接入，需校准） |
| 看板展示 | ✅ DONE（HTML + 飞书） |
| 状态回流脚本 | ✅ DONE |
| 真实大模型决策 | ❌ NOT_STARTED |
| Qoder Wake 接入 | ❌ NOT_STARTED |
| 定时任务上线 | ❌ NOT_STARTED |
| 自动触发任务 | ❌ NOT_STARTED |

### Metis Review

**Identified Gaps** (addressed):
- **G1 数据自举**: 已确认 Kaggle 下载策略，RUNBOOK 会包含下载步骤
- **G2 飞书模式**: v0.1 限定 dry-run 验证（`.env` 为占位符，无需真实凭证）
- **G3 脚本分类**: 已枚举全部 37 脚本，分为 4 入口 + 内部组件 + FROZEN 三组
- **G4 Python 版本**: 固定 Python ≥ 3.10
- **G5 ingestion_state 初始化**: 新增任务处理 fresh clone 场景
- **G6 data_quality_checks WARN**: 作为已知问题纳入调查范围
- **G7 Git-clean 验证**: 最终验收步骤包含 `git status` 检查

---

## Work Objectives

### Core Objective
把 Olist 项目从"多阶段散乱探索产物"收敛为"单版本清晰可运行数据产品"。产出 v0.1-heuristic-decision-sandbox。

### Concrete Deliverables
1. `reports/project_status_matrix.md` — 全模块 DONE/PARTIAL/FROZEN/NOT_STARTED 状态
2. `README_RUNBOOK.md` — 环境安装、数据下载、4 入口命令、排错指南
3. `requirements.txt` — 完整 Python 依赖（≥9 个包）
4. 标准化 4 个入口命令（daily / full / sync / pull）
5. `decision_source` / `review_type` 字段追加到策略/任务/复盘输出
6. 最小 pytest 测试集（`tests/test_*.py` × 3）
7. `reports/feishu_dashboard_validation.md` — 核心组件口径校准
8. Phase 1-7 基础表生成脚本修复（3 个脚本）
9. `reports/v0.1_release_summary.md` — 版本验收清单

### Definition of Done
- [ ] `python3 -m pytest tests/ -v` → 100% pass
- [ ] `python3 scripts/run_daily_pipeline.py` → exit 0
- [ ] `python3 scripts/run_full_pipeline.py` → exit 0，生成全部 `_full` 文件
- [ ] `python3 scripts/sync_feishu_bitable.py --all --dry-run` → exit 0
- [ ] `python3 scripts/pull_feishu_status.py --dry-run` → exit 0
- [ ] `pip install -r requirements.txt` 在干净 venv 中成功
- [ ] 策略输出的 `decision_source` 均为 `"heuristic"`
- [ ] `git status` 仅显示预期变更文件

### Must Have
- 4 个入口命令均可运行
- requirements.txt 完整可安装
- RUNBOOK 覆盖数据下载 → 运行 → 排错全流程
- 状态矩阵准确反映当前情况
- 决策产物含有 `decision_source` / `review_type` 标注
- 基础表脚本可复现（从 raw CSV → interim CSV）

### Must NOT Have (Guardrails)
- **G1**: 不新增功能（不接入 LLM、不接入 Qoder、不加定时任务）
- **G2**: 不修改 data/raw/ 原始数据
- **G3**: 不创建新的数据管道逻辑（只整理现有）
- **G4**: 飞书同步仅 dry-run（不要求真实凭证）
- **G5**: Phase 1-7 EDA 脚本不解冻（只修复 3 个基础表生成脚本）
- **G6**: Python ≥ 3.10 固定

---

## Verification Strategy

### Test Decision
- **Infrastructure exists**: NO（需新建 pytest 基础设施）
- **Automated tests**: Tests-after（先补测试文件，再补实现修复）
- **Framework**: pytest

### QA Policy
- **pytest**: 验证输出文件完整性（PK 唯一、字段存在、schema 一致性）
- **Bash**: 验证 4 个入口命令 exit code = 0
- **Bash**: 验证 `pip install -r requirements.txt` 成功
- **Bash**: 验证 `git status` 清洁度

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1（立即开始 — 文档 + 环境，MAX PARALLEL）:
├── Task 1: project_status_matrix.md + 4 大阶段冻结文档 [quick]
├── Task 2: requirements.txt + .env.example + README_RUNBOOK.md [quick]
├── Task 3: 修复 Phase 1-7 3 个基础表生成脚本 [deep]
└── Task 4: 新增 decision_source / review_type 字段 [quick]

Wave 2（Wave 1 后 — 入口收敛 + 测试，MAX PARALLEL）:
├── Task 5: 4 个入口命令收敛（run_full_pipeline.py 新建） [deep]
├── Task 6: 最小 pytest 测试集 [quick]
├── Task 7: 飞书看板核心口径校准 [quick]
└── Task 8: ingestion_state 初始化修复 [quick]

Wave 2（Wave 1 后 — 入口收敛 + 测试 + 校准，MAX PARALLEL）:
├── Task 5: run_full_pipeline.py + 4 入口标准化 [deep]
├── Task 6: 最小 pytest 测试集 [quick]
├── Task 7: 飞书看板核心口径校准 [quick]
└── Task 8: ingestion_state 初始化修复 [quick]

Wave FINAL（全部前序任务后 — 打包 + 验收）:
├── Task 9: v0.1 版本打包 + 全流程验收 [deep]
└── F1-F4: 4 个 parallel review agents
```

### Dependency Matrix

| Task | Depends On | Blocks | Wave |
|------|-----------|--------|------|
| 1 | — | — | 1 |
| 2 | — | 5, 6, 8 | 1 |
| 3 | — | 5 | 1 |
| 4 | — | 5 | 1 |
| 5 | 2, 3, 4 | 6, 7, 9 | 2 |
| 6 | 2, 5 | 9 | 2 |
| 7 | 5 | 9 | 2 |
| 8 | 2 | — | 2 |
| 9 | 5, 6, 7, 8 | F1-F4 | FINAL |
| F1 | 9 | — | FINAL |
| F2 | 9 | — | FINAL |
| F3 | 9 | — | FINAL |
| F4 | 9 | — | FINAL |

**Critical Path**: Task 2 → Task 5 → Task 6 → Task 9 → (F1-F4 async)
**Parallel Speedup**: ~60% vs sequential
**Max Concurrent**: 4（Wave 1 & 2）+ 4（Wave FINAL reviews）

---

## TODOs

### Wave 1 — 文档 + 环境 + 字段

- [ ] 1. 创建 project_status_matrix.md + 4 大阶段冻结文档

  **What to do**:
  - 创建 `reports/project_status_matrix.md`，枚举全项目模块并标注状态
  - 状态枚举值：DONE / PARTIAL / READY_FOR_TEST / BLOCKED / NOT_STARTED / FROZEN
  - 至少覆盖以下模块组：
    | 模块组 | 包含内容 | 状态 |
    |--------|---------|------|
    | 数据分析资产 | Phase 1-7 原始分析、基础表、图表、报告 | FROZEN |
    | 数据产品层 | AIP 层、12 指标、异常检测、Context Bundle、治理配置 | DONE |
    | 飞书工作台 | 表/字段/视图/仪表盘/同步脚本 | PARTIAL |
    | 决策闭环沙盘 | 规则建议、任务、复盘、状态回流 | PARTIAL |
    | AI 大模型决策 | Qoder/LLM 接入 | NOT_STARTED |
    | 自动化调度 | 定时任务/自动触发 | NOT_STARTED |
    | 环境与测试 | requirements.txt/runbook/pytest | NOT_STARTED |
  - 更新 README.md：修正项目状态描述（"AI 决策闭环已完成" → "规则驱动的 AI-ready 决策沙盘已完成"），增加 4 大阶段说明，更新阶段状态标记
  - 创建 `docs/phase_freeze_notice.md`：正式声明 Phase 1-7 为 FROZEN 历史资产，说明 FROZEN 含义和例外（3 个基础表脚本在 T3 中修复）

  **Must NOT do**:
  - 不修改 Phase 1-7 EDA 脚本
  - 不删除或移动任何 FROZEN 文件

  **Recommended Agent Profile**:
  - **Category**: `quick` — 纯文档写作，不涉及代码
  - **Skills**: [`writing-plans`]

  **Parallelization**:
  - **Can Run In Parallel**: YES（与 T2、T3、T4 独立）
  - **Parallel Group**: Wave 1
  - **Blocks**: None
  - **Blocked By**: None

  **References**:
  - `README.md:33-37` — 当前阶段状态标记（需更新）
  - `scripts/_FROZEN.md` — FROZEN 机制参考

  **Acceptance Criteria**:
  - [ ] `reports/project_status_matrix.md` 存在，覆盖 ≥7 个模块组
  - [ ] README.md 阶段状态描述已更新
  - [ ] `docs/phase_freeze_notice.md` 存在

  **QA Scenarios**:

  ```
  Scenario: 状态矩阵覆盖所有关键模块
    Tool: Bash (grep)
    Steps:
      1. grep -c "|" reports/project_status_matrix.md → ≥10 (至少 10 行表格)
      2. grep -c "DONE\|PARTIAL\|FROZEN\|NOT_STARTED\|BLOCKED\|READY_FOR_TEST" reports/project_status_matrix.md → ≥5
    Expected: 表格行数 ≥10，状态标签覆盖 ≥5 种
    Evidence: .sisyphus/evidence/task-1-status-matrix.txt

  Scenario: README 状态描述已更新
    Tool: Bash (grep)
    Steps:
      1. grep "规则驱动" README.md → 至少 1 条匹配
      2. grep "AI-ready" README.md → 至少 1 条匹配
    Expected: README 不含旧描述 "AI 决策闭环已完成"
    Evidence: .sisyphus/evidence/task-1-readme-update.txt
  ```

  **Commit**: YES
  - Message: `docs(consolidation): add project status matrix, phase freeze notice, README status update`
  - Files: `reports/project_status_matrix.md`, `docs/phase_freeze_notice.md`, `README.md`

- [ ] 2. 创建 requirements.txt + README_RUNBOOK.md + 更新 .env.example

  **What to do**:
  - 创建 `requirements.txt`（完整依赖列表）：
    ```
    pandas>=2.0.0
    numpy>=1.24.0
    pyyaml>=6.0
    requests>=2.28.0
    pydantic>=2.0.0
    openai>=1.0.0
    python-dotenv>=1.0.0
    matplotlib>=3.7.0
    seaborn>=0.12.0
    pytest>=7.0.0
    ```
  - 创建 `README_RUNBOOK.md`，覆盖以下 10 小节：
    1. **前置条件** — Python ≥3.10，Git 已安装
    2. **环境安装** — `python3 -m venv venv && source venv/bin/activate && pip install -r requirements.txt`
    3. **数据下载** — Kaggle CLI 下载 2 个数据集（主数据集 + Marketing Funnel）到 `data/raw/`，包含具体命令和 URL
    4. **构建基础表** — `python3 scripts/phase02_build_data_model.py`（或修复后的等价命令），生成 `order_level_base.csv` + `item_level_base.csv`
    5. **入口 1: 每日模拟** — `python3 scripts/run_daily_pipeline.py`，说明输出
    6. **入口 2: 全量评估** — `python3 scripts/run_full_pipeline.py`，说明输出
    7. **入口 3: 飞书同步** — dry-run 和 apply 两种模式
    8. **入口 4: 状态回流** — dry-run 和 apply 两种模式
    9. **常见错误** — 至少列 5 个典型问题及解决方法（如 pandas 找不到 CSV、权限不够、JSON 解析失败等）
    10. **当前限制** — 明确标注：不是大模型决策、飞书同步需手动建表、FROZEN 脚本不能直接运行
  - 更新 `.env.example`：确保包含 LLM_API_KEY + FEISHU_* 变量

  **Must NOT do**:
  - 不创建真实的 `.env` 文件
  - 不在 RUNBOOK 中包含未验证的命令

  **Recommended Agent Profile**:
  - **Category**: `quick` — 文档 + 小文件创建
  - **Skills**: [`writing-plans`]

  **Parallelization**:
  - **Can Run In Parallel**: YES（与 T1、T3、T4 独立）
  - **Parallel Group**: Wave 1
  - **Blocks**: T5, T6, T8
  - **Blocked By**: None

  **References**:
  - `scripts/run_daily_pipeline.py` — 入口 1 实现参考
  - `scripts/sync_feishu_bitable.py:1-20` — 入口 3 CLI 参考
  - `scripts/pull_feishu_status.py:1-20` — 入口 4 CLI 参考
  - `.env.example` — 环境变量模板

  **Acceptance Criteria**:
  - [ ] `requirements.txt` 存在，≥9 个包
  - [ ] `pip install -r requirements.txt` 在干净 venv 中成功（不做完整测试，但从文件内容能推断）
  - [ ] `README_RUNBOOK.md` 存在，覆盖 10 小节
  - [ ] `.env.example` 含 LLM_API_KEY + FEISHU_APP_ID + FEISHU_APP_SECRET + FEISHU_BASE_APP_TOKEN + FEISHU_CHAT_ID

  **QA Scenarios**:

  ```
  Scenario: requirements.txt 可解析
    Tool: Bash (python3)
    Steps:
      1. python3 -c "
  with open('requirements.txt') as f:
      pkgs = [l.strip() for l in f if l.strip() and not l.startswith('#')]
  assert len(pkgs) >= 9, f'Expected ≥9 packages, got {len(pkgs)}'
  print(f'OK: {len(pkgs)} packages')
  "
    Expected: ≥9 个包声明
    Evidence: .sisyphus/evidence/task-2-requirements.txt

  Scenario: RUNBOOK 覆盖全部 10 小节
    Tool: Bash (grep)
    Steps:
      1. grep -c "^##" README_RUNBOOK.md → ≥10
    Expected: ≥10 个标题（每小节一个）
    Evidence: .sisyphus/evidence/task-2-runbook-sections.txt
  ```

  **Commit**: YES（与 T1 同组）
  - Message: `docs(consolidation): add requirements.txt, README_RUNBOOK.md, .env.example update`

- [ ] 3. 修复 Phase 1-7 3 个基础表生成脚本（Scheme B）

  **What to do**:
  - 仅修复以下 3 个脚本使其可用：
    1. **`scripts/phase02_build_data_model.py`**（或等效脚本）：生成 `data/interim/order_level_base.csv` 和 `data/interim/item_level_base.csv`
    2. **`scripts/phase02_generate_docs.py`**：确保可生成或确认产物已固化
    3. **`scripts/calculate_channel_thresholds.py`**：生成 `data/interim/channel_classification.csv`
  - 修复策略：
    - 更新文件路径：从硬编码 `olist_*.csv`（根目录）改为 `from scripts.config import RAW_DIR, INTERIM_DIR` + `os.path.join(RAW_DIR, 'olist_orders_dataset.csv')`
    - 或在 README_RUNBOOK 中记录 workaround（如：运行前 `cp data/raw/*.csv .`）
  - 验证：运行修复后的脚本，确认 3 个基础表能成功生成且行数/列数与预期一致

  **Must NOT do**:
  - 不解冻其他 Phase 1-7 EDA 脚本
  - 不改变基础表的列名/结构/内容

  **Recommended Agent Profile**:
  - **Category**: `deep` — 需理解 Phase 2 数据模型的 join 逻辑和路径关系
  - **Skills**: [`writing-plans`]

  **Parallelization**:
  - **Can Run In Parallel**: YES（与 T1、T2、T4 独立）
  - **Parallel Group**: Wave 1
  - **Blocks**: T5
  - **Blocked By**: None

  **References**:
  - `scripts/config.py:12-14` — RAW_DIR, INTERIM_DIR 路径常量
  - `scripts/phase02_build_data_model.py` — 当前脚本（硬编码路径需修复）
  - `scripts/calculate_channel_thresholds.py` — 渠道分类脚本
  - `data/interim/order_level_base.csv` — 目标输出（99,441行×22列）
  - `data/interim/item_level_base.csv` — 目标输出（112,650行×36列）
  - `data/raw/olist_orders_dataset.csv` — 核心源数据
  - `data/raw/olist_order_items_dataset.csv` — 商品明细源

  **Acceptance Criteria**:
  - [ ] `order_level_base.csv` 可重新生成，行数 ≈99,441，列数 = 22
  - [ ] `item_level_base.csv` 可重新生成，行数 ≈112,650，列数 = 36
  - [ ] `channel_classification.csv` 可重新生成
  - [ ] 运行命令在 RUNBOOK 中可引用

  **QA Scenarios**:

  ```
  Scenario: 基础表生成脚本可运行并产出正确行数
    Tool: Bash (python3)
    Preconditions: data/raw/ 包含原始 CSV
    Steps:
      1. python3 scripts/phase02_build_data_model.py 2>&1 | tail -5
      2. python3 -c "
  import pandas as pd
  ol = pd.read_csv('data/interim/order_level_base.csv')
  il = pd.read_csv('data/interim/item_level_base.csv')
  assert 90000 <= len(ol) <= 110000, f'order_level_base rows: {len(ol)}'
  assert 100000 <= len(il) <= 130000, f'item_level_base rows: {len(il)}'
  print(f'OK: order_level={len(ol)}, item_level={len(il)}')
  "
    Expected: 脚本 exit 0，行数在合理范围
    Failure Indicators: 脚本 crash、行数为 0、路径错误
    Evidence: .sisyphus/evidence/task-3-base-tables.txt
  ```

  **Commit**: YES
  - Message: `fix(phase2): repair base table generation scripts with config.py paths`
  - Files: `scripts/phase02_build_data_model.py`, `scripts/calculate_channel_thresholds.py`

- [ ] 4. 添加 decision_source / review_type 字段到决策产物

  **What to do**:
  - 在以下文件中新增字段：
    1. **`scripts/ai_schemas.py`** — 更新 Pydantic 模型：
       - `StrategyRecommendation` 新增: `decision_source: str = "heuristic"`, `model_name: Optional[str] = None`, `is_simulated: bool = True`
       - `ActionTask` 新增: `task_source: str = "heuristic"`, `source_rule: Optional[str] = None`, `requires_human_confirmation: bool = True`
       - `ReviewRetro` 新增: `review_type: str = "simulated"`, `review_source: str = "hindsight_rule"`
    2. **`scripts/run_wake_agent.py`** — 生成策略/任务时填入 `decision_source = "heuristic"`
    3. **`scripts/run_ai_decision_engine.py`** — 生成策略时填入 `decision_source = "heuristic"`（LLM 模式为空字符串，回退模式下为 "heuristic"），生成任务时填入 `task_source`，生成复盘时填入 `review_type`
    4. **`scripts/generate_review_retro_samples.py`** — 生成复盘时填入 `review_type = "simulated"`, `review_source = "hindsight_rule"`
  - 确保所有新增字段写入输出 JSON 文件
  - 更新 `config/feishu_base_schema.yml` 对应的策略/任务/复盘表字段定义（如需要）

  **Must NOT do**:
  - 不改变现有字段名或值
  - 不改变现有 JSON 结构（增量添加字段）

  **Recommended Agent Profile**:
  - **Category**: `quick` — 小范围字段追加，不改变逻辑
  - **Skills**: [`writing-plans`]

  **Parallelization**:
  - **Can Run In Parallel**: YES（与 T1、T2、T3 独立）
  - **Parallel Group**: Wave 1
  - **Blocks**: T5
  - **Blocked By**: None

  **References**:
  - `scripts/ai_schemas.py` — Pydantic 模型定义（StrategyRecommendation, ActionTask, ReviewRetro）
  - `scripts/run_wake_agent.py:50-95` — 策略生成代码
  - `scripts/run_ai_decision_engine.py:150-220` — LLM 策略生成 + 回退逻辑
  - `scripts/generate_review_retro_samples.py` — 复盘样例生成
  - `config/feishu_base_schema.yml:154-238` — recommendations / action_tasks / review_retro 字段定义

  **Acceptance Criteria**:
  - [ ] `StrategyRecommendation` 模型含 `decision_source`, `model_name`, `is_simulated`
  - [ ] `ActionTask` 模型含 `task_source`, `source_rule`, `requires_human_confirmation`
  - [ ] `ReviewRetro` 模型含 `review_type`, `review_source`
  - [ ] 运行 full pipeline 后，输出 JSON 中包含新字段
  - [ ] 现有字段和行为不变

  **QA Scenarios**:

  ```
  Scenario: Pydantic 模型包含新字段
    Tool: Bash (python3)
    Steps:
      1. python3 -c "
  from scripts.ai_schemas import StrategyRecommendation, ActionTask, ReviewRetro
  s = StrategyRecommendation(recommendation_id='t1', event_id='e1', title='t', detail='【问题】x\n【证据】x\n【判断】x\n【建议动作】x\n【预期收益】x\n【风险】x\n【验收指标】x', target_object='o', expected_impact='x', risk_level='low', requires_approval=False, owner_role='business_ops', decision_type='investigate', confidence='high', success_metric='x', impact_score=7)
  assert s.decision_source == 'heuristic'
  assert s.model_name is None
  assert s.is_simulated == True
  a = ActionTask(task_id='t1', title='t', description='d', owner_role='business_ops', source_event='e1', source_strategy='s1', priority='medium', deadline='2026-01-01', status='todo')
  assert a.task_source == 'heuristic'
  assert a.requires_human_confirmation == True
  r = ReviewRetro(review_id='r1', strategy_id='s1', outcome='o', actual_impact='i', is_effective=True, lessons_learned='l', promote_to_rule=False, reviewed_at='2026-01-01')
  assert r.review_type == 'simulated'
  assert r.review_source == 'hindsight_rule'
  print('OK: all new fields present with correct defaults')
  "
    Expected: 所有新字段存在且默认值正确
    Evidence: .sisyphus/evidence/task-4-new-fields.json
  ```

  **Commit**: YES（与 T1+T2+T3 同 Wave 提交）
  - Message: `feat(schemas): add decision_source/task_source/review_type fields to AI decision outputs`

### Wave 2 — 入口收敛 + 测试 + 校准

- [ ] 5. 4 个入口命令收敛（新建 run_full_pipeline.py + 标准化调用约定）

  **What to do**:
  - **新建 `scripts/run_full_pipeline.py`**（~60 行）作为入口 2 的轻量 wrapper：
    ```python
    # 顺序调用现有 --mode full 脚本（subprocess 模式），不重复逻辑
    steps = [
        ("calculate_daily_metrics", "calculate_daily_metrics.py --mode full"),
        ("run_alert_detection", "run_alert_detection.py --mode full"),
        ("build_aip_context_bundle", "build_aip_context_bundle.py --mode full"),
        ("run_ai_decision_engine", "run_ai_decision_engine.py --mode full"),
        ("generate_feishu_sandbox", "generate_feishu_sandbox.py --mode full"),
    ]
    ```
  - **标准化 4 个入口命令**：确保每个入口有统一的行为约定：
    1. `run_daily_pipeline.py` — 8 步顺序，exit 0 = 成功
    2. `run_full_pipeline.py` — 5 步顺序（--mode full），exit 0 = 成功
    3. `sync_feishu_bitable.py --all --dry-run` — exit 0; `--apply` 真实写入
    4. `pull_feishu_status.py --dry-run` — exit 0
  - 验证：运行所有 4 个入口确认 exit code 正确
  - 更新 `README_RUNBOOK.md` 中入口命令文档

  **Must NOT do**:
  - 不修改现有 --mode full 脚本的实现
  - 不改变 run_daily_pipeline.py 的 8 步结构
  - 不重复实现逻辑（纯 subprocess 调用）

  **Recommended Agent Profile**:
  - **Category**: `deep` — 需理解 pipeline 调用链和 subprocess 隔离模式
  - **Skills**: [`writing-plans`]

  **Parallelization**:
  - **Can Run In Parallel**: YES（与 T6、T7、T8 独立）
  - **Parallel Group**: Wave 2
  - **Blocks**: T6, T7, T9
  - **Blocked By**: T2, T3, T4

  **References**:
  - `scripts/run_daily_pipeline.py:6-15` — STEPS 列表模式
  - `scripts/run_daily_pipeline.py:54-77` — subprocess.run 调用模式
  - `scripts/config.py` — 路径常量
  - `scripts/calculate_daily_metrics.py` — --mode full 参数
  - `scripts/run_alert_detection.py` — --mode full 参数
  - `scripts/run_ai_decision_engine.py` — --mode full 参数

  **Acceptance Criteria**:
  - [ ] `python3 scripts/run_full_pipeline.py` → exit 0
  - [ ] `python3 scripts/run_daily_pipeline.py` → exit 0
  - [ ] `python3 scripts/sync_feishu_bitable.py --all --dry-run` → exit 0
  - [ ] `python3 scripts/pull_feishu_status.py --dry-run` → exit 0
  - [ ] 4 个入口在 RUNBOOK 中有文档

  **QA Scenarios**:

  ```
  Scenario: run_full_pipeline.py 生成 _full 输出
    Tool: Bash
    Steps:
      1. python3 scripts/run_full_pipeline.py 2>&1 | tail -10
      2. test -f data/ads/daily_metrics_full.csv && echo "OK: daily_metrics_full" || echo "FAIL"
      3. test -f outputs/ai/strategy_recommendations.json && echo "OK: strategies" || echo "FAIL"
    Expected: exit 0，关键文件存在
    Evidence: .sisyphus/evidence/task-5-full-pipeline.txt

  Scenario: 4 个入口全部 exit 0
    Tool: Bash
    Steps:
      1. for cmd in "run_daily_pipeline.py" "run_full_pipeline.py"; do python3 scripts/$cmd; echo "$cmd exit=$?"; done
      2. python3 scripts/sync_feishu_bitable.py --all --dry-run; echo "sync exit=$?"
      3. python3 scripts/pull_feishu_status.py --dry-run; echo "pull exit=$?"
    Expected: 全部 exit 0
    Evidence: .sisyphus/evidence/task-5-entry-points.txt
  ```

  **Commit**: YES
  - Message: `feat(consolidation): add run_full_pipeline.py wrapper, standardize 4 entry points`
  - Files: `scripts/run_full_pipeline.py`, `README_RUNBOOK.md`

- [ ] 6. 建立最小 pytest 测试集

  **What to do**:
  - 创建 `tests/__init__.py`（空文件），`tests/conftest.py`（sys.path 设置 + 项目根 fixture）
  - 创建 **`tests/test_data_outputs.py`** — 验证 ADS + AIP 输出（≥6 测试）：
    - `daily_metrics_full.csv` PK 唯一、行数 600-650、14 列含 alert_count
    - `metric_alerts_full.csv` alert_id 唯一
    - `aip_context_bundle_full.json` 含 metrics/events/objects/allowed_actions，monthly_snapshots ≥ 24
  - 创建 **`tests/test_decision_outputs.py`** — 验证 AI 决策输出（≥6 测试）：
    - 每条策略 detail 含 `【问题】` + `【证据】`，有 owner_role，decision_source == "heuristic"
    - 每条任务有 task_source，每条复盘有 review_type
    - 策略数 ≥ 5
  - 创建 **`tests/test_feishu_outputs.py`** — 验证飞书 CSV（≥5 测试）：
    - 5 个 _full CSV 存在，≥4 个有数据，execution_reviews ≥ 3 行
    - CSV 列无重复
  - 验证：`python3 -m pytest tests/ -v` → 全部 PASS

  **Must NOT do**:
  - 不引入 mock（直接读文件 assertion）
  - 不测试未定义的产出

  **Recommended Agent Profile**:
  - **Category**: `quick` — 标准 pytest 文件读取 + assertion
  - **Skills**: [`writing-plans`]

  **Parallelization**:
  - **Can Run In Parallel**: YES（与 T5、T7、T8 独立）
  - **Parallel Group**: Wave 2
  - **Blocks**: T9
  - **Blocked By**: T2, T5

  **References**:
  - `scripts/config.py:74-78` — DAILY_METRICS_FULL_FILE, METRIC_ALERTS_FULL_FILE 等
  - `config/feishu_base_schema.yml` — 飞书表字段定义
  - `outputs/ai/strategy_recommendations.json` — 策略输出格式
  - `data/aip/aip_context_bundle_full.json` — Context Bundle 格式

  **Acceptance Criteria**:
  - [ ] `python3 -m pytest tests/ -v` → 全部 PASS（≥14 测试）
  - [ ] tests/__init__.py, conftest.py 存在
  - [ ] 3 个 test_*.py 文件各 ≥5 测试

  **QA Scenarios**:

  ```
  Scenario: pytest 全部通过
    Tool: Bash
    Steps:
      1. python3 -m pytest tests/ -v 2>&1 | tail -30
    Expected: 全部 PASS，0 FAILED，0 ERROR
    Evidence: .sisyphus/evidence/task-6-pytest-output.txt
  ```

  **Commit**: YES
  - Message: `test(consolidation): add minimal pytest suite for data/decision/feishu outputs`
  - Files: `tests/__init__.py`, `tests/conftest.py`, `tests/test_data_outputs.py`, `tests/test_decision_outputs.py`, `tests/test_feishu_outputs.py`

- [ ] 7. 飞书看板核心口径校准文档

  **What to do**:
  - 创建 `reports/feishu_dashboard_validation.md`，逐组件检查本地数据期望值：
    | 组件 | 数据源 | 期望值 | 通过 |
    |------|--------|--------|------|
    | 累计 GMV | daily_metrics_full | SUM(gmv) | ? |
    | 累计订单数 | daily_metrics_full | ≈99,441 | ? |
    | 最新业务日 GMV | daily_metrics_full | 2018-10-17 日值 | ? |
    | 异常事件数 | metric_alerts_full | COUNT  | ? |
    | 待处理任务数 | action_tasks_full | COUNT status=todo | ? |
    | 平均评分范围 | daily_metrics_full | 3.5-4.5 | ? |
  - 分析已知 4 个问题（配置变更提示/COUNT_ALL/异常不一致/review_retro 空白）并给出建议
  - 不操作真实飞书看板，仅基于本地 CSV 数据给出期望值

  **Must NOT do**:
  - 不操作真实飞书看板
  - 不修改飞书表结构

  **Recommended Agent Profile**:
  - **Category**: `quick` — 文档，从 CSV 计算数值
  - **Skills**: [`writing-plans`]

  **Parallelization**:
  - **Can Run In Parallel**: YES（与 T5、T6、T8 独立）
  - **Parallel Group**: Wave 2
  - **Blocks**: T9
  - **Blocked By**: T5

  **References**:
  - `data/ads/daily_metrics_full.csv` — 核心数值
  - `data/ads/metric_alerts_full.csv` — 异常数值
  - `data/feishu/action_tasks_for_feishu_full.csv` — 任务数
  - `config/feishu_field_mapping.yml` — 字段映射
  - `config/feishu_base_schema.yml` — 表结构

  **Acceptance Criteria**:
  - [ ] `reports/feishu_dashboard_validation.md` 存在
  - [ ] ≥6 个组件有检查结果
  - [ ] 每个标注当前值/期望值/通过状态
  - [ ] 4 个已知问题有分析记录

  **QA Scenarios**:

  ```
  Scenario: 校验文档覆盖核心组件
    Tool: Bash (grep)
    Steps:
      1. grep -c "|" reports/feishu_dashboard_validation.md → ≥10
      2. grep -c "通过\|PASS\|WARN" reports/feishu_dashboard_validation.md → ≥3
    Expected: 表格 ≥10 行，结果标签 ≥3 种
    Evidence: .sisyphus/evidence/task-7-dashboard.txt
  ```

  **Commit**: YES
  - Message: `docs(consolidation): add feishu dashboard core metrics calibration`

- [ ] 8. 修复 ingestion_state.json 冷启动问题

  **What to do**:
  - 检查 `scripts/simulate_daily_ingestion.py` 是否有初始化逻辑
  - 如果 `ingestion_state.json` 不存在时 crash，添加 auto-init：
    ```python
    if not os.path.exists(INGESTION_STATE_FILE):
        initial = {"next_simulated_date": "2016-09-04",
                   "last_completed_simulated_date": None,
                   "status": "initialized"}
        json.dump(initial, open(INGESTION_STATE_FILE, 'w'), indent=2)
    ```
  - 处理 `last_completed_simulated_date` 为 None 的边界
  - 验证：删除 state 文件 → 运行脚本 → 自动创建，日期为 2016-09-04

  **Must NOT do**:
  - 不改变现有日期推进逻辑
  - 不改变 state 文件字段结构

  **Recommended Agent Profile**:
  - **Category**: `quick` — 小修复
  - **Skills**: [`writing-plans`]

  **Parallelization**:
  - **Can Run In Parallel**: YES（与 T5、T6、T7 独立）
  - **Parallel Group**: Wave 2
  - **Blocks**: None
  - **Blocked By**: T2

  **References**:
  - `scripts/simulate_daily_ingestion.py` — 当前脚本
  - `scripts/config.py:41` — INGESTION_STATE_FILE 路径
  - `data/system/ingestion_state.json` — 当前格式

  **Acceptance Criteria**:
  - [ ] 删除 state 文件后运行脚本，文件自动创建
  - [ ] 初始日期为 `2016-09-04`
  - [ ] 已有文件时不覆盖

  **QA Scenarios**:

  ```
  Scenario: 冷启动自动初始化
    Tool: Bash
    Steps:
      1. cp data/system/ingestion_state.json /tmp/state_bak.json
      2. rm data/system/ingestion_state.json
      3. python3 scripts/simulate_daily_ingestion.py 2>&1
      4. python3 -c "import json; s=json.load(open('data/system/ingestion_state.json')); assert s['next_simulated_date']=='2016-09-04'; print('OK')"
      5. cp /tmp/state_bak.json data/system/ingestion_state.json
    Expected: 文件被创建，初始日期正确
    Evidence: .sisyphus/evidence/task-8-cold-start.txt
  ```

  **Commit**: YES
  - Message: `fix(pipeline): auto-initialize ingestion_state.json on cold start`
  - Files: `scripts/simulate_daily_ingestion.py`

### Wave FINAL — 版本打包

- [ ] 9. v0.1 版本打包（release_summary + 全流程验收）

  **What to do**:
  - 创建 `reports/v0.1_release_summary.md`，包含：
    1. **版本定义**：`v0.1-heuristic-decision-sandbox`（无 LLM、无 Qoder、无定时任务）
    2. **版本能力清单**：✅ 数据分析 / 指标异常 / 规则策略 / 飞书承接 / 看板 / 状态回流 / ❌ LLM决策 / Qoder / 定时 / 自动触发
    3. **4 入口命令验收**：每个入口的运行结果（exit code + 关键输出）
    4. **测试结果**：pytest 全量通过报告（测试数/通过数）
    5. **环境复现**：`pip install -r requirements.txt` 验证
    6. **已知限制**：FROZEN 脚本、飞书空凭证、review_retro 采样数据
    7. **下一步**：Phase II（维度级异常）、Qoder 接入、定时任务
  - 运行全流程验收：
    ```bash
    python3 -m pytest tests/ -v          # 全部 PASS
    python3 scripts/run_daily_pipeline.py   # exit 0
    python3 scripts/run_full_pipeline.py    # exit 0
    git status --short                     # 仅预期文件
    ```
  - 将验收结果填入 release_summary

  **Must NOT do**:
  - 不新增功能
  - 不在验收中加入未完成的模块

  **Recommended Agent Profile**:
  - **Category**: `deep` — 需运行全流程验收并汇总
  - **Skills**: [`writing-plans`]

  **Parallelization**:
  - **Can Run In Parallel**: NO（依赖所有前序任务完成）
  - **Parallel Group**: Final
  - **Blocks**: F1-F4
  - **Blocked By**: T5, T6, T7, T8

  **References**:
  - `reports/project_status_matrix.md` — 模块状态（T1）
  - `README_RUNBOOK.md` — 运行手册（T2）
  - `reports/feishu_dashboard_validation.md` — 看板校准（T7）
  - `scripts/run_full_pipeline.py` — 入口 2（T5）

  **Acceptance Criteria**:
  - [ ] `reports/v0.1_release_summary.md` 存在，含 7 个小节
  - [ ] pytest 全部通过
  - [ ] 4 个入口全部 exit 0
  - [ ] git status 仅显示预期文件
  - [ ] 版本定义准确（heuristic，非 LLM）

  **QA Scenarios**:

  ```
  Scenario: 全流程验收通过
    Tool: Bash
    Steps:
      1. python3 -m pytest tests/ -v 2>&1 | grep -E "passed|failed"
      2. python3 scripts/run_full_pipeline.py 2>&1 | tail -3
      3. git status --short
    Expected: All tests passed, exit 0, only expected files modified
    Evidence: .sisyphus/evidence/task-9-release-acceptance.txt
  ```

  **Commit**: YES
  - Message: `docs(release): v0.1-heuristic-decision-sandbox release summary`
  - Files: `reports/v0.1_release_summary.md`

---
## Final Verification Wave (MANDATORY — after ALL implementation tasks)

> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay".

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Verify: task_count.csv has correct totals, all must-have deliverables exist, no guardrail violations (no new features, no LLM integration, no frozen scripts modified beyond 3 base table scripts)

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run: `python3 -m pytest tests/ -v`, `python3 -m py_compile scripts/run_full_pipeline.py`. Check: no hardcoded credentials, no broken imports, all 4 entry points importable.

- [ ] F3. **Real Manual QA** — `unspecified-high`
  Execute: all 4 entry point commands, verify exit codes. Run pytest. Verify git status clean. Verify RUNBOOK steps work.

- [ ] F4. **Scope Fidelity Check** — `deep`
  Compare deliverables against plan. Verify: no scope creep (no new LLM/Qoder/cron features), 37 scripts classified correctly, 3 base table scripts only changes to frozen scripts.

---

## Commit Strategy

- **Wave 1** (T1+T2+T3+T4): `docs(consolidation): project status matrix, runbook, requirements, phase freeze, decision fields`
- **Wave 2** (T5+T6+T7+T8): `feat(consolidation): entry point convergence, tests, dashboard calibration, state fix`
- **Wave FINAL** (T9): `docs(release): v0.1-heuristic-decision-sandbox release summary`

---

## Success Criteria

### Verification Commands
```bash
# 环境验证
pip install -r requirements.txt
python3 -c "import pandas, numpy, yaml, requests, pydantic, openai, dotenv, matplotlib, seaborn; print('All imports OK')"

# 测试验证
python3 -m pytest tests/ -v

# 入口命令验证
python3 scripts/run_daily_pipeline.py
python3 scripts/run_full_pipeline.py
python3 scripts/sync_feishu_bitable.py --all --dry-run
python3 scripts/pull_feishu_status.py --dry-run

# Git 清洁度
git status --short

# 基础表复现
python3 scripts/phase02_build_data_model.py
```

### Final Checklist
- [ ] `requirements.txt` 存在且 `pip install -r` 成功
- [ ] `README_RUNBOOK.md` 覆盖 10 个小节
- [ ] `reports/project_status_matrix.md` 覆盖全部模块
- [ ] 4 个入口命令全部 exit 0
- [ ] `pytest` 全部通过（≥10 测试）
- [ ] 策略/任务/复盘输出均有 `decision_source` 或 `review_type`
- [ ] 基础表脚本可生成 `order_level_base.csv` / `item_level_base.csv`
- [ ] 飞书看板核心指标校准完成
- [ ] `git status` 仅预期文件变更
- [ ] `reports/v0.1_release_summary.md` 存在
