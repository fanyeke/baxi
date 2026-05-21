# Phase H: 真实飞书 API 接入与协同闭环

## TL;DR

> **Quick Summary**: 将现有的本地数据沙盘升级为真实的飞书多维表格同步系统。从 AIP 数据产品层 → Wake Agent 输出 → 飞书多维表格（upsert）→ 飞书文档日报 → 群消息通知 → 状态回流本地。
>
> **Deliverables**:
> - Feishu SDK 客户端（auth, CRUD, pagination, retry）
> - Bitable 同步脚本（dry-run / apply / upsert / audit）
> - 飞书文档发布脚本（daily_report.md → 飞书文档）
> - 飞书群消息发送脚本（feishu_message.json → 群）
> - 状态回流脚本（飞书表 → 本地 CSV）
> - 审计日志（data/system/feishu_sync_log.csv）
>
> **Estimated Effort**: Large（7 个子阶段，12 个文件）
> **Parallel Execution**: YES — 5 waves with max 3 parallel tasks per wave
> **Critical Path**: H1 → H2 → H3 → H4 → H5 → H6 → H7

---

## Context

### Original Request
用户提供了详细的 Phase H 规划文档，包含 7 个子阶段（H1-H7）和 5 个核心原则。需要将本地沙盘变成真实飞书协同工作台。

### Interview Summary
**研究发现**:
- `sync_to_feishu.py` 是纯 Skeleton（34 行，无 API 调用）
- `config/feishu_base_schema.yml` 定义 5 张表 54 个字段 — schema 完整
- `config/feishu_field_mapping.yml` 字段映射完整
- `generate_feishu_sandbox.py` 工作正常，生成 5 个 CSV（其中 3 个有数据）
- 存在已知 Bug：`transform_metric_alerts` 产生重复列 `object_type`
- Pipeline 已运行 31 天模拟数据（2016-09-04 → 2016-10-04）
- 无 `.env` 文件，无飞书凭证配置

**Metis Review**:
- **H2 表自动创建 → EXCLUDE**（第一版手动建表）
- **H7 实时 webhook → EXCLUDE**（轮询方式）
- **历史数据回填 → EXCLUDE**（仅同步当天输出）
- **批量限制**: ≤50 条/批，1s 间隔
- **幂等要求**: upsert by primary key，不重复创建
- **审计日志**: 所有同步进入 `feishu_sync_log.csv`
- **dry-run 默认**: 必须支持且默认开启

---

## Work Objectives

### Core Objective
将本地数据流水线的输出真实同步到飞书多维表格，支持 dry-run、幂等 upsert、审计日志、文档发布、群消息和状态回流。

### Concrete Deliverables
- `config/feishu_app.yml` — 飞书应用配置模板
- `config/feishu_table_ids.yml` — 真实飞书表 ID 映射
- `scripts/feishu_client.py` — 飞书 API 客户端封装（~200 行）
- `scripts/sync_feishu_bitable.py` — 多维表同步脚本（~150 行）
- `scripts/publish_feishu_report.py` — 飞书文档发布（~80 行）
- `scripts/send_feishu_message.py` — 群消息发送（~60 行）
- `scripts/pull_feishu_status.py` — 状态回流（~100 行）
- `data/system/feishu_sync_log.csv` — 审计日志

### Definition of Done
- [ ] 5 张飞书表成功同步（daily_metrics, metric_alerts, strategy_recommendations, action_tasks, execution_reviews）
- [ ] dry-run 模式验证通过（不产生真实写入）
- [ ] upsert 幂等验证（重复执行不产生重复记录）
- [ ] 审计日志完整记录
- [ ] 飞书文档日报发布成功
- [ ] 群消息发送成功
- [ ] 状态回流本地

### Must Have
- 所有同步支持 `--dry-run` 和 `--apply` 标志
- 幂等 upsert（按主键 create 或 update）
- 审计日志记录每次同步
- Bug fix: `generate_feishu_sandbox.py` 重复列问题
- `.env` 凭证隔离，`.gitignore` 排除

### Must NOT Have (Guardrails)
- **G1**: 全表同步 — 飞书不是数据仓库，只接指标/异常/建议/任务
- **G2**: dry-run 默认 — 必须显式 `--apply` 才能真实写入
- **G3**: 幂等 upsert — 不允许无主键追加
- **G4**: ≤50 条/批 — 批次大小限制，1s 间隔
- **G5**: 不影响现有 8 步流水线 — 新增脚本独立运行
- **G6**: 第一版不建表 — 手动创建，脚本只写数据

---

## Verification Strategy

### Test Decision
- **Infrastructure exists**: YES (Python + pandas)
- **Automated tests**: Tests-after（验证脚本可运行）
- **Framework**: pytest（仅用于关键逻辑验证）
- **Agent-Executed QA**: ALWAYS mandatory — 每次任务必须运行验证

### QA Policy
- **API 验证**: Bash(curl) 或 bash(python3) — 运行同步脚本，验证输出
- **审计日志**: Bash — 检查 feishu_sync_log.csv 完整性
- **幂等性**: Bash — 重复运行同一脚本，验证创建次数为 0
- **飞书 UI**: Playwright — 如果可行，截图验证飞书表内容

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1（立即开始 — 配置 + 文档 + Bug fix）:
├── Task 1: 飞书接入方案设计（H1 文档 + 配置模板）[quick]
├── Task 2: 修复 generate_feishu_sandbox.py 重复列 Bug [quick]
└── Task 3: 生成 .env.example 并更新 .gitignore [quick]

Wave 2（Wave 1 后 — SDK 客户端）:
├── Task 4: FeishuClient SDK 封装（auth, CRUD, retry, pagination）[deep]
└── Task 5: feishu_table_ids.yml 配置 + 表验证脚本 [quick]

Wave 3（Wave 2 后 — 核心同步）:
└── Task 6: sync_feishu_bitable.py（H3: daily_metrics + metric_alerts，H4: strategy + tasks）[deep]

Wave 4（Wave 3 后 — 文档 + 消息）:
├── Task 7: publish_feishu_report.py（H5: daily_report.md → 飞书文档）[quick]
└── Task 8: send_feishu_message.py（H6: 群消息通知）[quick]

Wave 5（Wave 3 后 — 状态回流，独立）:
└── Task 9: pull_feishu_status.py（H7: action_tasks + execution_reviews 状态回流）[deep]

Wave FINAL（全部任务后 — 4 个并行验证，等待用户确认）:
├── F1: Plan Compliance Audit（oracle）
├── F2: Code Quality Review（unspecified-high）
├── F3: Real Manual QA（unspecified-high）+ playwright if UI
└── F4: Scope Fidelity Check（deep）
→ 展示结果 → 获得用户明确 "okay" 后完成
```

### Dependency Matrix

- **1-3**: — → 4, 5
- **4**: 1 → 6, 7, 8, 9
- **5**: 1, 2 → 6
- **6**: 4, 5 → 7, 8
- **7**: 4 → FINAL
- **8**: 4 → FINAL
- **9**: 4 → FINAL
- **FINAL**: 6, 7, 8, 9 → user okay

**Critical Path**: Task 1 → Task 4 → Task 6 → Task 7/8 → FINAL → user okay
**Parallel Speedup**: ~40% faster than sequential
**Max Concurrent**: 3（Wave 1, 4, 5）

### Agent Dispatch Summary

- **Wave 1**: **3** — T1 → `quick`, T2 → `quick`, T3 → `quick`
- **Wave 2**: **2** — T4 → `deep`, T5 → `quick`
- **Wave 3**: **1** — T6 → `unspecified-high`
- **Wave 4**: **2** — T7 → `quick`, T8 → `quick`
- **Wave 5**: **1** — T9 → `deep`
- **FINAL**: **4** — F1 → `oracle`, F2 → `unspecified-high`, F3 → `unspecified-high`, F4 → `deep`

---

## TODOs

- [ ] 1. H1 飞书接入方案设计 + 配置模板

  **What to do**:
  - 创建 `docs/phase_h_feishu_api_design.md`（API 设计文档）
    - 明确认证方式（tenant_access_token via app_id/app_secret）
    - 列出所有 HTTP 方法、端点、认证流、速率限制
    - 列出所需的飞书应用权限（bitable app 读写、drive 文件访问）
  - 创建 `config/feishu_app.yml` 模板（app_id、app_secret 占位符）
  - 创建 `.env.example` 文件（FEISHU_APP_ID、FEISHU_APP_SECRET、FEISHU_BASE_APP_TOKEN）
  - 在 `.gitignore` 中添加 `.env` 排除规则

  **Must NOT do**:
  - 不要填入真实凭证（只用占位符）
  - 不要创建或修改任何 .env 文件（用户手动创建）

  **Recommended Agent Profile**:
  - **Category**: `quick` — 文档 + 配置模板
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES（与 T2、T3 独立）
  - **Blocks**: T4, T5
  - **Blocked By**: None
  - **Parallel Group**: Wave 1（与 T2、T3）

  **References**:
  - `config/feishu_base_schema.yml` — 5 张表 schema 定义
  - `config/owner_mapping.yml` — 5 个 owner 角色
  - `docs/aip_feishu_integration.md` — 现有飞书集成文档
  - 飞书 API 文档: https://open.feishu.cn/document/server-docs

  **Acceptance Criteria**:
  - [ ] `docs/phase_h_feishu_api_design.md` 创建（包含认证、权限、端点、限制说明）
  - [ ] `config/feishu_app.yml` 模板创建（app_id/app_secret 占位符）
  - [ ] `.env.example` 创建（含所有必要环境变量）
  - [ ] `.gitignore` 包含 `.env` 排除规则

  **QA Scenarios**:

  ```
  Scenario: 验证 config/feishu_app.yml 模板格式
    Tool: Bash (python3)
    Preconditions: config/feishu_app.yml 已创建
    Steps:
      1. python3 -c "import yaml; d = yaml.safe_load(open('config/feishu_app.yml')); assert 'app_id' in d; assert 'app_secret' in d; print('OK')"
    Expected: 文件可加载且包含 app_id 和 app_secret 键
    Evidence: .sisyphus/evidence/H1-01-config-validation.txt

  Scenario: 验证 .env.example 包含所有必要变量
    Tool: Bash (grep)
    Steps:
      1. grep -c "FEISHU_APP_ID" .env.example → 1
      2. grep -c "FEISHU_APP_SECRET" .env.example → 1
      3. grep -c "FEISHU_BASE_APP_TOKEN" .env.example → 1
    Expected: 所有三个关键变量都存在
    Evidence: .sisyphus/evidence/H1-02-env-variables.txt
  ```

  **Evidence to Capture**:
  - [ ] config/feishu_app.yml 模板验证输出
  - [ ] .env.example 内容验证

  **Commit**: YES
  - Message: `docs(feishu): add Phase H API design doc, app config template, .env.example`
  - Files: `docs/phase_h_feishu_api_design.md`, `config/feishu_app.yml`, `.env.example`

---

- [ ] 2. 修复 generate_feishu_sandbox.py 重复列 Bug

  **What to do**:
  - 在 `scripts/generate_feishu_sandbox.py` 的 `transform_metric_alerts` 方法中：
    - 当前 col_map `dimension` → `object_type` 与源文件已有的 `object_type` 冲突
    - 修复：在 rename 前检查源文件是否已有 `object_type`，如有则先删除旧的或跳过
  - 验证修复：运行脚本，检查 `data/feishu/metric_alerts_for_feishu.csv` 不再有重复列
  - 修复后需运行 `python3 scripts/generate_feishu_sandbox.py` 验证全部 5 个 CSV 正确

  **Must NOT do**:
  - 不要修改其他 transform 方法的逻辑
  - 不要改变输出 CSV 的列名或顺序（只修复重复列）

  **Recommended Agent Profile**:
  - **Category**: `quick` — 单文件小修复
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES（与 T1、T3 独立）
  - **Blocks**: T5
  - **Blocked By**: None
  - **Parallel Group**: Wave 1（与 T1、T3）

  **References**:
  - `scripts/generate_feishu_sandbox.py:76-106`（transform_metric_alerts 方法）
  - `config/feishu_base_schema.yml:55-104`（alert_events 表 schema）
  - `data/feishu/metric_alerts_for_feishu.csv`（当前输出，有重复列）

  **Acceptance Criteria**:
  - [ ] `data/feishu/metric_alerts_for_feishu.csv` 不再有重复 `object_type` 列
  - [ ] 所有 5 个 CSV 列数与 schema 定义一致
  - [ ] `python3 scripts/generate_feishu_sandbox.py` 输出 "Done" 且无警告

  **QA Scenarios**:

  ```
  Scenario: 验证 metric_alerts CSV 无重复列
    Tool: Bash (python3)
    Preconditions: generate_feishu_sandbox.py 已运行
    Steps:
      1. python3 -c "
  import pandas as pd
  df = pd.read_csv('data/feishu/metric_alerts_for_feishu.csv')
  cols = list(df.columns)
  dupes = [c for c in cols if cols.count(c) > 1]
  assert len(dupes) == 0, f'Duplicate columns: {dupes}'
  print(f'OK - {len(cols)} unique columns')
  "
    Expected: 无重复列，输出 "OK" 消息
    Evidence: .sisyphus/evidence/H2-01-no-duplicate-columns.txt
  ```

  **Evidence to Capture**:
  - [ ] 重复列修复验证输出

  **Commit**: YES（与 T1 同组提交）

---

- [ ] 3. .env 凭证隔离 + .gitignore 更新

  **What to do**:
  - 创建 `.env.example`（如果 T1 未创建）
  - 确保 `.gitignore` 包含 `.env` 排除规则
  - 创建 `data/ops/.gitkeep`（为 Phase H7 状态回流准备目录）

  **Must NOT do**:
  - 不创建真实的 `.env` 文件（用户手动创建）
  - 不在代码中硬编码任何凭证

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES（与 T1、T2 独立）
  - **Blocks**: None
  - **Blocked By**: None
  - **Parallel Group**: Wave 1（与 T1、T2）

  **References**:
  - `.gitignore`（当前内容）

  **Acceptance Criteria**:
  - [ ] `.env` 在 `.gitignore` 中已排除
  - [ ] `data/ops/` 目录已创建（含 .gitkeep）

  **QA Scenarios**:

  ```
  Scenario: 验证 .env 被 gitignore 排除
    Tool: Bash (grep)
    Steps:
      1. grep -q "^.env$" .gitignore || grep -q "^.env\." .gitignore → 0 (exit code)
    Expected: 匹配成功，表示 .env 被排除
    Evidence: .sisyphus/evidence/H3-01-gitignore-check.txt
  ```

  **Commit**: YES（与 T1、T2 同组提交）

---

- [ ] 4. FeishuClient SDK 封装（核心 API 客户端）

  **What to do**:
  - 创建 `scripts/feishu_client.py`（~200 行），提供：
    - `FeishuClient` 类：封装所有飞书 API 调用
    - `__init__(app_id, app_secret, app_token)`：初始化认证参数
    - `get_tenant_access_token()`：获取/刷新 tenant_access_token（带缓存、59min 过期）
    - `list_records(table_id, page_size=50, filter=None)`：分页查询记录
    - `create_record(table_id, record_data)`：创建单条记录
    - `batch_create(table_id, records)`：批量创建（≤50 条）
    - `update_record(table_id, record_id, record_data)`：更新单条记录
    - `batch_update(table_id, records)`：批量更新
    - `upsert_by_key(table_id, records, key_field)`：幂等 upsert（查询→存在则更新，不存在则创建）
    - `create_doc(title, content)`：创建飞书文档（用于 H5）
    - `send_group_message(chat_id, content)`：发送群消息（用于 H6）
    - `_request(method, path, **kwargs)`：底层 HTTP 请求（带 429 重试、指数退避）
  - 使用 `requests` 库（非 lark-oapi SDK，更轻量）
  - 支持日志输出（verbose 模式）

  **Must NOT do**:
  - 不实现自动表创建（手动建表）
  - 不处理复杂的文档格式转换（纯 Markdown 上传）
  - 不使用 lark-oapi SDK（增加依赖，用 requests 足够）

  **Recommended Agent Profile**:
  - **Category**: `deep` — 复杂 API 封装，需处理认证、重试、分页、错误
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: NO（被 T4 阻塞）
  - **Parallel Group**: Sequential
  - **Blocks**: T5, T6, T7, T8, T9
  - **Blocked By**: T1, T2

  **References**:

  **External References**:
  - 飞书 tenant_access_token API: `https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal`
  - 飞书 Bitable API: `https://open.feishu.cn/document/server-docs/docs/bitable-v1/bitable-app` — CRUD 端点
  - 飞书记录列表（支持 filter）: `https://open.feishu.cn/document/server-docs/docs/bitable-v1/bitable-app-table-record/list`
  - 飞书错误码: `https://open.feishu.cn/document/server-docs/server-error-code`
  - requests 库: `https://requests.readthedocs.io/` — HTTP 客户端

  **Pattern References**（项目内代码模式）:
  - `scripts/config.py` — 路径配置模式（FeishuClient 不需要路径，但使用同样的 import 模式）
  - `scripts/run_wake_agent.py:26-28` — JSON 加载模式
  - `scripts/run_daily_pipeline.py:58-77` — subprocess 调用 + 错误处理模式

  **Acceptance Criteria**:
  - [ ] FeishuClient 类创建，可通过 `from scripts.feishu_client import FeishuClient` 导入
  - [ ] `FeishuClient(app_id, app_secret, app_token)` 初始化成功
  - [ ] `get_tenant_access_token()` 返回有效 token（非空字符串）
  - [ ] `_request()` 方法支持 GET/POST/PATCH，429 时重试 3 次（1s → 2s → 4s）
  - [ ] `upsert_by_key()` 方法实现幂等逻辑（list → create/update）
  - [ ] `batch_create()` 限制 ≤50 条，超出自动分割
  - [ ] 所有方法支持 dry-run 模式（仅打印不执行）

  **QA Scenarios**:

  ```
  Scenario: FeishuClient 基础功能测试（mock API）
    Tool: Bash (python3)
    Preconditions: scripts/feishu_client.py 已创建
    Steps:
      1. python3 -c "
  from scripts.feishu_client import FeishuClient
  client = FeishuClient(app_id='test', app_secret='test', app_token='test')
  client.dry_run = True  # 启用 dry-run 模式
  assert client.list_records('tbl_test') == []  # dry-run 返回空
  assert client.create_record('tbl_test', {'data': 'test'}) == None  # dry-run 不执行
  print('OK - FeishuClient dry-run mode works')
  "
    Expected: 导入成功，dry-run 模式可用
    Evidence: .sisyphus/evidence/T4-01-client-import.txt

  Scenario: 批量分割 — 超过 50 条自动分割
    Tool: Bash (python3)
    Steps:
      1. python3 -c "
  from scripts.feishu_client import FeishuClient
  client = FeishuClient(app_id='test', app_secret='test', app_token='test', dry_run=True)
  records = [{'id': str(i), 'value': i} for i in range(120)]
  batches = client.batch_create('tbl_test', records)
  print(f'Batches: {batches}')
  assert len(batches) == 3  # 120 / 50 = 3 批（50+50+20）"
    Expected: 输出 "Batches: 3" 且无错误
    Evidence: .sisyphus/evidence/T4-02-batch-split.txt
  ```

  **Evidence to Capture**:
  - [ ] FeishuClient 导入和 dry-run 测试输出
  - [ ] 批量分割测试输出

  **Commit**: YES
  - Message: `feat(feishu): implement FeishuClient SDK with auth, CRUD, retry, and pagination`
  - Files: `scripts/feishu_client.py`

---

- [ ] 5. feishu_table_ids.yml 配置 + 表验证脚本

  **What to do**:
  - 创建 `config/feishu_table_ids.yml` 模板：
    ```yaml
    base:
      app_token: "bascnXXXXXXXXXX"  # 从飞书多维表格 URL 获取

    tables:
      daily_metrics:
        table_id: "tblXXXXXXXXX1"
      alert_events:
        table_id: "tblXXXXXXXXX2"
      strategy_recommendations:
        table_id: "tblXXXXXXXXX3"
      action_tasks:
        table_id: "tblXXXXXXXXX4"
      review_retro:
        table_id: "tblXXXXXXXXX5"
    ```
  - 创建验证脚本 `scripts/verify_feishu_tables.py`：
    - 读取 `config/feishu_table_ids.yml`
    - 使用 FeishuClient 连接每张表
    - 查询每张表的前 1 条记录（验证权限 + 表 ID 正确性）
    - 输出验证结果（OK/FAIL）

  **Must NOT do**:
  - 不自动创建表（手动创建）
  - 不修改任何飞书表结构

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: NO（被 T4 阻塞，需要 FeishuClient）
  - **Parallel Group**: Sequential
  - **Blocks**: T6
  - **Blocked By**: T1, T2, T4

  **References**:
  - `config/feishu_app.yml`（T1 创建的模板）
  - `scripts/feishu_client.py`（T4 创建）
  - `config/feishu_base_schema.yml`（5 张表名定义）

  **Acceptance Criteria**:
  - [ ] `config/feishu_table_ids.yml` 模板创建（含占位符 table_id）
  - [ ] `scripts/verify_feishu_tables.py` 创建
  - [ ] `python3 scripts/verify_feishu_tables.py --dry-run` 输出 "Verifying 5 tables..." 并列出表配置
  - [ ] 运行后输出 "All 5 tables verified" 或具体失败信息

  **QA Scenarios**:

  ```
  Scenario: 验证表配置加载
    Tool: Bash (python3)
    Preconditions: config/feishu_table_ids.yml 模板存在
    Steps:
      1. python3 -c "
  import yaml
  config = yaml.safe_load(open('config/feishu_table_ids.yml'))
  required_tables = ['daily_metrics', 'alert_events', 'strategy_recommendations', 'action_tasks', 'review_retro']
  for t in required_tables:
      assert t in config['tables'], f'Missing table: {t}'
      assert 'table_id' in config['tables'][t], f'Missing table_id: {t}'
  print(f'OK - {len(required_tables)} tables configured')
  "
    Expected: 输出 "OK - 5 tables configured"
    Evidence: .sisyphus/evidence/T5-01-table-config.txt
  ```

  **Evidence to Capture**:
  - [ ] 表配置加载验证

  **Commit**: YES（与 T6 同组提交）

---

- [ ] 6. sync_feishu_bitable.py — 核心同步脚本（H3 + H4）

  **What to do**:
  - 创建 `scripts/sync_feishu_bitable.py`（~150 行）：
    - 接收参数：`--table daily_metrics|metric_alerts|strategy_recommendations|action_tasks|execution_reviews`
    - 接收参数：`--dry-run` 或 `--apply`
    - 读取 `data/feishu/{table}_for_feishu.csv` 作为数据源
    - 使用 `FeishuClient.upsert_by_key()` 幂等同步
    - 记录审计日志到 `data/system/feishu_sync_log.csv`
    - 支持 `--all` 同步所有表（顺序执行）
  - 处理 null/空值（转换为空字符串）
  - 支持 429 重试（由 FeishuClient 处理）
  - 详细日志输出：创建/更新/跳过数量

  **Must NOT do**:
  - 不修改 `run_daily_pipeline.py`（独立脚本）
  - 不修改 `sync_to_feishu.py`（已废弃，保留作为参考）
  - 不处理非 CSV 数据源

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high` — 复杂同步逻辑，需处理多表、upsert、审计日志
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: NO（被 T5 阻塞）
  - **Parallel Group**: Sequential
  - **Blocks**: T7, T8
  - **Blocked By**: T4, T5

  **References**:

  **Pattern References**:
  - `scripts/sync_to_feishu.py` — 废弃的原同步脚本（参考 CSV 路径和目标表名）
  - `scripts/generate_feishu_sandbox.py` — 数据转换输出格式
  - `config/feishu_field_mapping.yml` — 字段映射配置
  - `scripts/config.py` — 路径配置（FEISHU_DIR, SYSTEM_DIR 等）
  - `scripts/feishu_client.py` — FeishuClient SDK（T4 创建）

  **External References**:
  - 飞书 Bitable upsert 模式: list records by filter → if found updateRecord else createRecord
  - CSV 处理: pandas.read_csv + to_csv 模式

  **Acceptance Criteria**:
  - [ ] `python3 scripts/sync_feishu_bitable.py --table daily_metrics --dry-run` 输出 planned 操作数量无 API 调用
  - [ ] `python3 scripts/sync_feishu_bitable.py --table daily_metrics --apply` 成功 upsert 到飞书表
  - [ ] `python3 scripts/sync_feishu_bitable.py --table metric_alerts --apply` 成功 upsert
  - [ ] `python3 scripts/sync_feishu_bitable.py --table strategy_recommendations --apply` 成功 upsert
  - [ ] `python3 scripts/sync_feishu_bitable.py --table action_tasks --apply` 成功 upsert
  - [ ] `python3 scripts/sync_feishu_bitable.py --all --dry-run` 同步所有 5 张表
  - [ ] 重复执行同一表：`--apply` 两次，第二次 create_count=0（幂等）
  - [ ] `data/system/feishu_sync_log.csv` 记录所有同步操作

  **QA Scenarios**:

  ```
  Scenario: dry-run daily_metrics 同步
    Tool: Bash (python3)
    Preconditions: data/feishu/daily_metrics_for_feishu.csv 存在（7 行数据）
    Steps:
      1. python3 scripts/sync_feishu_bitable.py --table daily_metrics --dry-run 2>&1 | tee /tmp/h3-dry-run.log
      2. grep -c "dry-run" /tmp/h3-dry-run.log → ≥1
      3. 日志中显示 N records [N create, 0 update, 0 skip]
    Expected: 输出 dry-run 统计信息，无 API 调用
    Evidence: .sisyphus/evidence/T6-01-dry-run.txt
  ```

  **Evidence to Capture**:
  - [ ] dry-run 日志
  - [ ] 幂等 upsert 日志
  - [ ] 审计日志前 10 行

  **Commit**: YES
  - Message: `feat(feishu): implement sync_feishu_bitable.py with upsert, dry-run, and audit logging`
  - Files: `scripts/sync_feishu_bitable.py`, `data/system/feishu_sync_log.csv`

---

- [ ] 7. publish_feishu_report.py — 飞书文档发布（H5）

  **What to do**:
  - 创建 `scripts/publish_feishu_report.py`（~80 行）：
    - 读取 `outputs/wake/daily_report.md`
    - 调用 `FeishuClient.create_doc(title, content)`
    - 创建飞书云文档
    - 输出文档 URL
    - 可选：将 URL 回写到 daily_metrics 表（或保存到 `data/feishu/report_links.csv`）
    - 支持 `--dry-run` 和 `--apply`

  **Must NOT do**:
  - 不处理复杂格式化（纯 Markdown 上传）
  - 不修改 daily_report.md 内容

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES（与 T8 独立）
  - **Parallel Group**: Wave 4（与 T8）
  - **Blocks**: None
  - **Blocked By**: T4

  **References**:
  - `scripts/feishu_client.py` — FeishuClient SDK（create_doc 方法）
  - `outputs/wake/daily_report.md` — Wake Agent 生成的日报
  - `data/feishu/` — report_links.csv 输出目录

  **Acceptance Criteria**:
  - [ ] `python3 scripts/publish_feishu_report.py --dry-run` 输出 "Will create doc: 《Olist日报 YYYY-MM-DD》"
  - [ ] `python3 scripts/publish_feishu_report.py --apply` 创建飞书文档并返回 URL
  - [ ] `data/feishu/report_links.csv` 记录每次发布的文档链接
  - [ ] 文档标题包含业务日期

  **QA Scenarios**:

  ```
  Scenario: dry-run 日报发布
    Tool: Bash (python3)
    Steps:
      1. python3 scripts/publish_feishu_report.py --dry-run 2>&1
      2. 输出包含 "Will create doc:" 和报告标题
    Expected: 显示计划创建的文档信息
    Evidence: .sisyphus/evidence/T7-01-report-dry-run.txt
  ```

  **Evidence to Capture**:
  - [ ] dry-run 输出

  **Commit**: YES（与 T8 同组提交）

---

- [ ] 8. send_feishu_message.py — 飞书群消息（H6）

  **What to do**:
  - 创建 `scripts/send_feishu_message.py`（~60 行）：
    - 读取 `outputs/wake/feishu_message.json`
    - 调用 `FeishuClient.send_group_message(chat_id, content)`
    - 发送到指定飞书群
    - 支持 `--dry-run` 和 `--apply`
    - 失败时不阻塞后续操作（非关键步骤）

  **Must NOT do**:
  - 不修改 feishu_message.json 内容
  - 不因消息发送失败而抛出异常

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES（与 T7 独立）
  - **Parallel Group**: Wave 4（与 T7）
  - **Blocked By**: T4

  **References**:
  - `scripts/feishu_client.py` — FeishuClient SDK（send_group_message 方法）
  - `outputs/wake/feishu_message.json` — Wake Agent 生成的消息 JSON
  - `config/feishu_app.yml` — chat_id 配置

  **Acceptance Criteria**:
  - [ ] `python3 scripts/send_feishu_message.py --dry-run` 输出 "Will send message to chat_id: xxx"
  - [ ] `python3 scripts/send_feishu_message.py --apply` 发送到指定飞书群
  - [ ] 发送失败时输出警告信息但不退出（exit code 0）

  **QA Scenarios**:

  ```
  Scenario: dry-run 消息发送
    Tool: Bash (python3)
    Steps:
      1. python3 scripts/send_feishu_message.py --dry-run 2>&1
      2. 输出包含 "Will send message" 和 chat_id
    Expected: 显示计划发送的消息信息
    Evidence: .sisyphus/evidence/T8-01-message-dry-run.txt
  ```

  **Evidence to Capture**:
  - [ ] dry-run 输出

  **Commit**: YES（与 T7 同组提交）

---

- [ ] 9. pull_feishu_status.py — 状态回流（H7）

  **What to do**:
  - 创建 `scripts/pull_feishu_status.py`（~100 行）：
    - 从飞书 action_tasks 表拉取状态变化的记录
    - 筛选 `status` 字段为 `in_progress` 或 `done` 的记录
    - 拉取 `execution_reviews` 表的复盘记录
    - 更新本地状态：
      - 写入 `data/ops/action_task_status_snapshot.csv`
      - 更新 `data/system/feishu_sync_log.csv`
    - 支持 `--dry-run` 和 `--apply`
    - 支持指定 `--since YYYY-MM-DD` 拉取增量

  **Must NOT do**:
  - 不复盖本地已变更的数据
  - 不实时轮询（手动触发或定时任务）
  - 不修改飞书表中的状态（只读）

  **Recommended Agent Profile**:
  - **Category**: `deep` — 状态同步逻辑，需处理增量拉取、冲突解决
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: NO（被 T9 独立）
  - **Parallel Group**: Wave 5
  - **Blocked By**: T4

  **References**:
  - `scripts/feishu_client.py` — FeishuClient SDK（list_records 方法）
  - `config/feishu_table_ids.yml` — action_tasks 和 review_retro 表 ID
  - `config/feishu_field_mapping.yml` — 字段映射
  - `data/ops/` — 状态快照输出目录（T3 创建）
  - `config/status_enums.yml` — task_status 枚举值

  **Acceptance Criteria**:
  - [ ] `python3 scripts/pull_feishu_status.py --dry-run` 输出 "Found N changed records in action_tasks"
  - [ ] `python3 scripts/pull_feishu_status.py --apply` 拉取状态并写入本地文件
  - [ ] `data/ops/action_task_status_snapshot.csv` 包含拉取的状态数据
  - [ ] 增量拉取（`--since`）支持：只拉取指定日期后的变更
  - [ ] 审计日志记录拉取操作

  **QA Scenarios**:

  ```
  Scenario: 状态回流 dry-run
    Tool: Bash (python3)
    Steps:
      1. python3 scripts/pull_feishu_status.py --dry-run --since 2016-10-01 2>&1
      2. 输出显示找到的变更数量
    Expected: 显示计划拉取的状态信息
    Evidence: .sisyphus/evidence/T9-01-status-dry-run.txt
  ```

  **Evidence to Capture**:
  - [ ] dry-run 输出

  **Commit**: YES
  - Message: `feat(feishu): implement pull_feishu_status.py for status loop-back from Feishu`
  - Files: `scripts/pull_feishu_status.py`, `data/ops/action_task_status_snapshot.csv`

---

## Final Verification Wave (MANDATORY — after ALL implementation tasks)

> 4 个 review agents 并行运行。全部 APPROVE 后，向用户展示结果，获得明确 "okay" 后完成。

- [ ] F1. **Plan Compliance Audit** — `oracle`
  逐任务核对：9 个 Task + 54 字段 schema 一致性。检查 `--dry-run` 是否所有脚本都支持。
  输出: `Tasks [9/9] | dry-run [9/9] | must-not-have [verified] | VERDICT`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  运行 `python3 -m py_compile scripts/feishu_client.py scripts/sync_feishu_bitable.py ...`（所有脚本语法检查）。
  检查：无硬编码凭证、无 `as any`、无 `console.log` 等效物。
  输出: `Syntax [PASS] | Security [PASS] | Style [PASS] | VERDICT`

- [ ] F3. **Real Manual QA** — `unspecified-high`
  执行每个 Task 的 QA 场景。验证 dry-run、upsert 幂等、审计日志、文档发布、消息发送、状态回流。
  保存证据到 `.sisyphus/evidence/final-qa/`。
  输出: `Scenarios [N/N] | Evidence files [N] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  检查：没有意外修改 `run_daily_pipeline.py`（8 步流水线）。
  没有全表同步到飞书。没有自动建表。
  输出: `Pipeline unchanged [YES] | Feishu-not-DW [YES] | Manual-table-only [YES] | VERDICT`

---

## Commit Strategy

- **Wave 1**（T1 + T2 + T3）: `docs(feishu): Phase H API design, config templates, gitignore update`
  - Files: `docs/phase_h_feishu_api_design.md`, `config/feishu_app.yml`, `config/feishu_table_ids.yml`, `.env.example`, `.gitignore`
  
- **Wave 2**（T4 + T5）: `feat(feishu): implement FeishuClient SDK and table verification`
  - Files: `scripts/feishu_client.py`, `scripts/verify_feishu_tables.py`

- **Wave 3**（T6 + Bug fix）: `feat(feishu): implement sync_feishu_bitable.py with upsert and audit logging`
  - Files: `scripts/sync_feishu_bitable.py`, `scripts/generate_feishu_sandbox.py`, `data/system/feishu_sync_log.csv`

- **Wave 4**（T7 + T8）: `feat(feishu): implement doc publish and group message scripts`
  - Files: `scripts/publish_feishu_report.py`, `scripts/send_feishu_message.py`, `data/feishu/report_links.csv`

- **Wave 5**（T9）: `feat(feishu): implement status pull-back from Feishu to local`
  - Files: `scripts/pull_feishu_status.py`, `data/ops/action_task_status_snapshot.csv`

---

## Success Criteria

### Verification Commands
```bash
# Verify all scripts parse correctly
python3 -m py_compile scripts/feishu_client.py scripts/sync_feishu_bitable.py scripts/publish_feishu_report.py scripts/send_feishu_message.py scripts/pull_feishu_status.py

# Verify dry-run for all scripts
python3 scripts/sync_feishu_bitable.py --table daily_metrics --dry-run
python3 scripts/publish_feishu_report.py --dry-run
python3 scripts/send_feishu_message.py --dry-run
python3 scripts/pull_feishu_status.py --dry-run

# Verify audit log
wc -l data/system/feishu_sync_log.csv
head -5 data/system/feishu_sync_log.csv

# Verify no duplicate columns in CSVs
python3 -c "import pandas as pd; df=pd.read_csv('data/feishu/metric_alerts_for_feishu.csv'); assert len(df.columns)==len(set(df.columns))"
```

### Final Checklist
- [ ] 所有 9 个 Task 完成
- [ ] 所有 `--dry-run` 模式通过
- [ ] 所有 upsert 幂等验证通过
- [ ] 审计日志完整
- [ ] 飞书文档发布成功
- [ ] 群消息发送成功
- [ ] 状态回流本地
- [ ] 无凭证硬编码
- [ ] `.env` 在 `.gitignore` 中排除
- [ ] F1-F4 全部 APPROVE → 用户明确 "okay"
