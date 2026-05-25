# Baxi v0.5.3 — AIP-inspired Data Governance Pack

## TL;DR

> **Quick Summary**: 为 Baxi 补齐数据治理层，参照 Palantir AIP/Foundry 的治理模式，产出 10 份治理文档 + 9 个配置文件 + API 端点 + 前端治理中心页面，使 Baxi 从"功能系统"升级为"可治理系统"。
>
> **Deliverables**:
> - `docs/external/palantir_aip_foundry/` — 13 份 Palantir 参考文档 + manifest.yml
> - `docs/governance/` — 10 份 Baxi 治理规范文档
> - `config/` — 9 个治理配置文件
> - `api/routers/governance.py` — 治理数据 API（GET 端点）
> - `frontend/src/pages/Governance.tsx` — 治理中心前端页面
> - `sql/migrations/008_governance.sql` — checkpoint/health 审计表
> - `tests/` — TDD 测试覆盖
>
> **Estimated Effort**: Large
> **Parallel Execution**: YES — 4 waves, max 10 parallel tasks
> **Critical Path**: Wave 1 配置 → Wave 2 文档 → Wave 3 后端 API → Wave 4 前端 → Final 验证

---

## Context

### Original Request
用户基于 Palantir AIP/Foundry 的治理思路，为 Baxi 项目设计了一套完整的数据治理方案，覆盖 6 类治理能力：数据保护与治理、本体/对象层建模、数据血缘与生命周期、权限与敏感标记、监控健康与审计、AI 工作流评估。

### Interview Summary

**Key Discussions**:
- **范围**: 全部 10 份治理文档 + 9 个配置文件 + API + 前端，不做删减
- **测试策略**: TDD（pytest + vitest，先写测试再实现）
- **脚本状态**: 纯增量，不修改任何 FROZEN 脚本
- **前端**: 新增独立"治理中心"页面，只读仪表盘
- **Palantir 文档**: 计划中直接 webfetch，失败则用摘要笔记
- **时间**: 深思熟虑（天级），非快速增量
- **项目**: 个人项目，无 CI/CD

**Research Findings** (from codebase exploration):
| 发现 | 影响 |
|------|------|
| 已有 9 个 AIP 业务对象 + 18 个 YAML 配置 | 治理配置可复用现有 schema 模式 |
| 已有 dry-run/apply 模式 + 审计 CSV | checkpoint 机制可扩展现有模式 |
| 已有 data_quality_rules.yml（5 条规则） | health_checks 应增强而非替换已有规则 |
| docs/external/ 未在 .gitignore | 需添加排除规则 |
| 所有脚本 FROZEN | 治理层完全独立，零耦合 |
| React 控制台有 7 页，用 TanStack Query | 新页面需遵循相同数据获取模式 |
| FastAPI 有 9 个 router | governance router 遵循 outbox.py 模式 |

### Metis Review

**Identified Gaps** (addressed in plan):
| Gap | Resolution |
|-----|-----------|
| 治理 API 认证方式 | 复用现有 Bearer Token（单用户项目，治理只读） |
| 治理数据持久化 | 新增 SQLite 表 + migration 008（checkpoints, health_results） |
| 前端治理中心具体内容 | 定义 6 个面板：目录、分类、血缘图、标记、检查点、健康 |
| data_health_checks.yml vs data_quality_rules.yml | 增强：health_checks 引用 data_quality_rules.yml 并添加新规则 |
| 配置文件目录结构 | 保持 config/ 扁平结构（匹配现有 18 个 config 惯例） |
| ontology_model.md vs aip_object_schema.yml | 扩展：ontology_model.md 在现有对象上加治理注解 |
| 治理 API 限流 | 使用现有 "other" 限流类（300 req/60s） |

---

## Work Objectives

### Core Objective
为 Baxi 建立完整的数据治理体系，使每个数据资产（表、字段、API、导出）都有明确的分类、敏感标记、血缘关系和生命周期策略，并通过前端治理中心可视化呈现。

### Concrete Deliverables

**文档（10 份）**:
- `docs/governance/baxi_data_governance_policy.md`
- `docs/governance/baxi_ontology_model.md`
- `docs/governance/baxi_data_lineage_model.md`
- `docs/governance/baxi_data_marking_policy.md`
- `docs/governance/baxi_checkpoint_policy.md`
- `docs/governance/baxi_retention_policy.md`
- `docs/governance/baxi_data_health_checks.md`
- `docs/governance/baxi_decision_eval_policy.md`
- `docs/governance/baxi_access_control_model.md`
- `docs/governance/baxi_aip_alignment.md`

**配置文件（9 个）**:
- `config/data_catalog.yml`
- `config/data_classification.yml`
- `config/data_markings.yml`
- `config/data_lineage.yml`
- `config/checkpoint_rules.yml`
- `config/retention_policies.yml`
- `config/health_checks.yml`
- `config/decision_eval_rules.yml`
- `config/access_policy.yml`

**外部参考文档**:
- `docs/external/palantir_aip_foundry/manifest.yml`
- `docs/external/palantir_aip_foundry/` 下 13 个 .html/.md 文件

**代码**:
- `api/routers/governance.py` — 治理数据 API
- `api/main.py` — 注册 governance router（单行修改）
- `scripts/config.py` — 注册治理 YAML 路径常量（追加）
- `frontend/src/api/governance.ts` — 前端 API 类型与 hooks
- `frontend/src/pages/Governance.tsx` — 治理中心页面
- `frontend/src/App.tsx` — 添加 /governance 路由（单行修改）
- `sql/migrations/008_governance.sql` — 治理审计表
- `sql/schema.sql` — 追加 governance table schemas（追加）

**测试**:
- `tests/test_governance_configs.py` — 配置校验测试
- `tests/test_governance_api.py` — API 端点测试
- `frontend/src/pages/__tests__/Governance.test.tsx` — 前端组件测试

### Definition of Done
- [ ] `pytest tests/test_governance*.py -v` → ALL PASS
- [ ] `cd frontend && npx vitest run` → ALL PASS
- [ ] `python -c "import yaml; yaml.safe_load(open('config/<%=config%>.yml'))"` 对全部 9 个配置通过
- [ ] `curl -H "Authorization: Bearer $TOKEN" http://localhost:8765/api/v1/governance/catalog` → 200 + JSON
- [ ] 前端 `/governance` 页面渲染 6 个面板，数据从 API 获取
- [ ] `docs/external/` 被 git 忽略（git status 不显示）

### Must Have
- 所有 10 份治理文档 + 9 个配置文件产出
- Palantir 参考文档本地下载 + manifest
- 治理 API 端点（catalog, classification, markings, lineage, checkpoints, health）
- 前端治理中心页面（只读仪表盘，至少含 4 个面板）
- .gitignore 更新
- TDD 测试覆盖

### Must NOT Have (Guardrails)
- ❌ **不修改** `scripts/` 目录下任何 FROZEN 文件
- ❌ **不修改** 现有 18 个 YAML 配置文件
- ❌ **不修改** 现有 12 张 SQLite 表的结构
- ❌ **不修改** 现有 9 个 API router 的逻辑
- ❌ **不修改** 现有前端 7 个页面的代码
- ❌ **不引入** 新业务功能、LLM 集成、CI/CD
- ❌ **不过度抽象** — 治理是只读展示层，不需要 service 层、ORM、缓存

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed.

### Test Decision
- **Infrastructure exists**: YES（pytest + vitest）
- **Automated tests**: TDD（RED → GREEN → REFACTOR）
- **Framework**: pytest（backend）、vitest（frontend）
- **每个任务遵循**: 先写测试 → 确认失败 → 最小实现 → 测试通过

### QA Policy
每个任务包含 Agent-Executed QA Scenarios：
- **Backend/API**: `Bash (curl)` — 发送请求、断言状态码和响应字段
- **Config**: `Bash (python)` — yaml.safe_load + schema 校验
- **Frontend**: `Playwright` — 导航页面、检查 DOM、截图
- **文档**: `Bash (cat)` — 文件存在 + 字数校验

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Start Immediately — 基础设施 + 全部配置 + 外部文档下载):
├── T1: .gitignore + 目录脚手架 [quick]
├── T2: Palantir 参考文档下载 P0 (8 docs) [unspecified-high]
├── T3: Palantir 参考文档下载 P1 (5 docs) + manifest.yml [unspecified-high]
├── T4: config/data_catalog.yml [quick]
├── T5: config/data_classification.yml [quick]
├── T6: config/data_markings.yml [quick]
├── T7: config/data_lineage.yml [quick]
├── T8: 版本号升级 v0.5.2→v0.5.3 [quick]
└── T9: 配置 YAML schema 校验测试 (TDD RED) [quick]

Wave 2 (After Wave 1 configs — 治理策略配置 + 策略文档，MAX PARALLEL):
├── T10: config/checkpoint_rules.yml [quick]
├── T11: config/retention_policies.yml [quick]
├── T12: config/health_checks.yml [quick]
├── T13: config/decision_eval_rules.yml [quick]
├── T14: config/access_policy.yml [quick]
├── T15: docs/governance/baxi_data_governance_policy.md [writing]
├── T16: docs/governance/baxi_ontology_model.md [writing]
├── T17: docs/governance/baxi_data_lineage_model.md [writing]
├── T18: docs/governance/baxi_data_marking_policy.md [writing]
└── T19: docs/governance/baxi_access_control_model.md [writing]

Wave 3 (After Wave 2 — 操作治理文档 + 后端 API，MAX PARALLEL):
├── T20: docs/governance/baxi_checkpoint_policy.md [writing]
├── T21: docs/governance/baxi_retention_policy.md [writing]
├── T22: docs/governance/baxi_data_health_checks.md [writing]
├── T23: docs/governance/baxi_decision_eval_policy.md [writing]
├── T24: docs/governance/baxi_aip_alignment.md [writing]
├── T25: scripts/config.py 注册治理 YAML 路径常量 [quick]
├── T26: api/routers/governance.py — API 端点 (TDD) [deep]
├── T27: api/main.py — 注册 governance router [quick]
└── T28: sql/migrations/008_governance.sql + schema.sql 追加 [quick]

Wave 4 (After Wave 3 API — 前端，顺序依赖):
├── T29: frontend/src/api/governance.ts — 类型 + hooks (TDD RED) [quick]
├── T30: frontend/src/pages/Governance.tsx — 治理中心页面 [visual-engineering]
├── T31: frontend/src/App.tsx — 添加 /governance 路由 [quick]
└── T32: frontend/src/pages/__tests__/Governance.test.tsx (TDD GREEN) [quick]

Wave FINAL (After ALL tasks — 4 parallel reviews, then user okay):
├── F1: Plan Compliance Audit (oracle)
├── F2: Code Quality Review (unspecified-high)
├── F3: Real Manual QA (unspecified-high)
└── F4: Scope Fidelity Check (deep)
→ Present results → Get explicit user okay
```

### Critical Path
T1 → T4-T7 (configs) → T10-T14 (action configs) + T15-T19 (policy docs) → T25-T28 (API + migration) → T29-T32 (frontend) → F1-F4 (verification)

### Parallel Speedup
~75% faster than sequential — Wave 2 runs 10 tasks in parallel, Wave 3 runs 9 tasks in parallel.

### Agent Dispatch Summary
- **Wave 1**: 9 tasks — T1-T6,T8,T9 → `quick`, T2-T3 → `unspecified-high`, T7 → `quick`
- **Wave 2**: 10 tasks — T10-T14 → `quick`, T15-T19 → `writing`
- **Wave 3**: 9 tasks — T20-T24 → `writing`, T25,T27,T28 → `quick`, T26 → `deep`
- **Wave 4**: 4 tasks — T29,T31,T32 → `quick`, T30 → `visual-engineering`
- **FINAL**: 4 tasks — F1 → `oracle`, F2 → `unspecified-high`, F3 → `unspecified-high`, F4 → `deep`

---

## TODOs

### Wave 1 — 基础设施 + 数据目录配置 + 外部文档

- [x] 1. `.gitignore` 更新 + 目录脚手架创建

  **What to do**:
  - 在 `.gitignore` 末尾追加 `docs/external/` 行
  - 创建目录结构：`docs/external/palantir_aip_foundry/` 及其子目录（`00_platform/` ~ `08_lifecycle/`）
  - 创建目录结构：`docs/governance/`
  - 创建目录结构：`config/`（已存在，确认即可）
  - 创建目录结构：`.sisyphus/evidence/`

  **Must NOT do**:
  - 不修改 `.gitignore` 中已有规则
  - 不创建 `docs/external/` 以外的目录

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: 纯文件系统操作，无业务逻辑
  - **Skills**: [`git-master`]
    - `git-master`: 用于 .gitignore 修改后的 git 状态验证

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T2-T9)
  - **Blocks**: All subsequent tasks (directory structure prerequisite)
  - **Blocked By**: None

  **References**:
  - `.gitignore:1-30` — 现有忽略规则，确认 `docs/external` 未被覆盖
  - `pyproject.toml:1-5` — 项目根目录确认

  **Acceptance Criteria**:
  - [ ] `grep 'docs/external' .gitignore` → 返回匹配行
  - [ ] `ls docs/external/palantir_aip_foundry/00_platform/` → 目录存在
  - [ ] `ls docs/governance/` → 目录存在

  **QA Scenarios**:
  ```
  Scenario: .gitignore 包含 docs/external 规则
    Tool: Bash (grep)
    Steps:
      1. grep 'docs/external' .gitignore
    Expected Result: 返回包含 "docs/external" 的行
    Evidence: .sisyphus/evidence/task-1-gitignore.txt

  Scenario: 所有治理子目录已创建
    Tool: Bash (ls)
    Steps:
      1. for dir in docs/external/palantir_aip_foundry/00_platform docs/external/palantir_aip_foundry/01_governance docs/external/palantir_aip_foundry/02_ontology docs/external/palantir_aip_foundry/03_lineage docs/external/palantir_aip_foundry/04_observability docs/external/palantir_aip_foundry/05_security docs/external/palantir_aip_foundry/06_accountability docs/external/palantir_aip_foundry/07_ai_governance docs/external/palantir_aip_foundry/08_lifecycle docs/governance; do ls -d "$dir" 2>/dev/null || echo "MISSING: $dir"; done
    Expected Result: 所有目录存在，无 MISSING 输出
    Evidence: .sisyphus/evidence/task-1-scaffold.txt
  ```

  **Commit**: YES (groups with T8)
  - Message: `chore(governance): add .gitignore rule and directory scaffold for governance pack`

- [x] 2. Palantir AIP/Foundry P0 参考文档下载（8 份）

  **What to do**:
  - 使用 webfetch 从 palantir.com/docs/foundry 下载以下 8 份文档并保存为 .md：
    1. AIP Overview → `00_platform/aip_overview.md`
    2. AIP Architecture → `00_platform/aip_architecture.md`
    3. Data Protection and Governance → `01_governance/data_protection_and_governance.md`
    4. Ontology Overview → `02_ontology/ontology_overview.md`
    5. Object Backend Overview → `02_ontology/object_backend_overview.md`
    6. Data Lineage Overview → `03_lineage/data_lineage_overview.md`
    7. Markings → `05_security/markings.md`
    8. Checkpoints Overview → `06_accountability/checkpoints_overview.md`
  - 若某文档 webfetch 失败（401/403），记录原因到 `_fetch_errors.md`
  - 文档保存为 Markdown 格式（webfetch format="markdown"）

  **Must NOT do**:
  - 不提交下载的文档到 git（.gitignore 已排除）
  - 不手动编写文档内容（仅 webfetch）

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: 需要处理网络请求、可能的重试、格式转换
  - **Skills**: [`scrapling-skill`]
    - `scrapling-skill`: 用于处理可能需要浏览器渲染的页面

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T1, T3-T9)
  - **Blocks**: T15（治理文档写作需要参考）
  - **Blocked By**: T1（目录脚手架）

  **References**:
  - Palantir 文档 URL 列表（来自用户 spec）：
    - `https://www.palantir.com/docs/foundry/aip/overview/`
    - `https://www.palantir.com/docs/foundry/architecture-center/aip-architecture`
    - `https://www.palantir.com/docs/foundry/security/data-protection-and-governance/`
    - `https://www.palantir.com/docs/foundry/ontology/overview/`
    - `https://www.palantir.com/docs/foundry/object-backend/overview/`
    - `https://www.palantir.com/docs/foundry/data-lineage/overview/`
    - `https://www.palantir.com/docs/foundry/security/markings/`
    - `https://www.palantir.com/docs/foundry/checkpoints/overview/`

  **Acceptance Criteria**:
  - [ ] 至少 5/8 份文档成功下载（>500 bytes）
  - [ ] `_fetch_errors.md` 记录所有失败原因

  **QA Scenarios**:
  ```
  Scenario: P0 文档已下载且非空
    Tool: Bash (find + wc)
    Steps:
      1. find docs/external/palantir_aip_foundry -name "*.md" | sort
      2. for f in $(find docs/external/palantir_aip_foundry -name "*.md"); do echo "$f: $(wc -c < "$f") bytes"; done
    Expected Result: 至少 5 个 .md 文件，每个 >500 bytes
    Evidence: .sisyphus/evidence/task-2-p0-docs.txt
  ```

  **Commit**: NO（文档不提交 git）

- [x] 3. Palantir AIP/Foundry P1 参考文档下载（5 份）+ manifest.yml

  **What to do**:
  - 使用 webfetch 下载 P1 文档：
    1. Data Health → `04_observability/data_health.md`
    2. Data Lifetime → `08_lifecycle/data_lifetime.md`
    3. AIP Evals Overview → `07_ai_governance/aip_evals_overview.md`
    4. AIP Logic Overview → `07_ai_governance/aip_logic_overview.md`
    5. Interoperability → `00_platform/interoperability.md`
  - 创建 `docs/external/palantir_aip_foundry/manifest.yml`
  - manifest.yml 内容：按用户 spec 格式，每条记录包含 id、title、source、local_file、fetched_at、used_for

  **Must NOT do**:
  - 不跳过 manifest.yml（这是治理文档的"目录的目录"）

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: web fetch + structured YAML 生成

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T1-T2, T4-T9)
  - **Blocks**: T15（治理文档写作参考）
  - **Blocked By**: T1（目录脚手架）

  **Acceptance Criteria**:
  - [ ] `python -c "import yaml; yaml.safe_load(open('docs/external/palantir_aip_foundry/manifest.yml'))"` → 成功解析
  - [ ] manifest.yml 包含至少 13 条记录

  **QA Scenarios**:
  ```
  Scenario: manifest.yml 合法且完整
    Tool: Bash (python)
    Steps:
      1. python3 -c "
  import yaml
  m = yaml.safe_load(open('docs/external/palantir_aip_foundry/manifest.yml'))
  print(f'Total entries: {len(m)}')
  for e in m:
      required = ['id', 'title', 'source', 'local_file']
      missing = [k for k in required if k not in e]
      if missing:
          print(f'  {e.get(\"id\", \"?\")}: MISSING {missing}')
      else:
          print(f'  {e[\"id\"]}: OK')
  "
    Expected Result: Total entries >= 13，所有条目 required fields OK
    Evidence: .sisyphus/evidence/task-3-manifest.txt
  ```

  **Commit**: YES（manifest.yml 提交；文档不提交）
  - Message: `docs(governance): add Palantir reference manifest`

- [x] 4. `config/data_catalog.yml` — 数据资产目录

  **What to do**:
  - 创建 Baxi 数据资产清单，覆盖 5 类资产：
    1. **raw_files**: 11 个 CSV 文件（data/raw/）
    2. **sqlite_tables**: 12 张表（dwd_order_level ~ qoder_jobs）
    3. **api_endpoints**: 14 个端点（/api/v1/*）
    4. **feishu_exports**: 5 个飞书多维表格
    5. **logs**: API 日志、审计日志
  - 每条记录包含：asset_id、asset_type、name、location、description、grain（如适用）、row_count（如适用）、owner
  - 每条记录增加 `status: active | deprecated | deleted` 字段（对应 Foundry lineage 的 Deleted 节点概念）
  - 格式遵循现有 YAML 惯例（如 `alert_rules.yml` 的 `rules:` 列表模式）

  **Must NOT do**:
  - 不包含敏感数据内容（仅元数据）
  - 不引用不存在的文件路径

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: 数据盘点，基于已知信息结构化写入

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T1-T3, T5-T9)
  - **Blocks**: T26（API catalog 端点）
  - **Blocked By**: None（可从探索结果获取信息）

  **References**:
  - `data/raw/` — 列出所有 CSV 文件名和行数
  - `sql/schema.sql` — 12 张表的完整定义（表名、列、主键）
  - `api/main.py:330-380` — 所有注册的路由端点
  - `config/feishu_base_schema.yml` — 5 个飞书表格定义
  - `config/alert_rules.yml:1-30` — YAML 结构模式参考（rules: 列表）

  **Acceptance Criteria**:
  - [ ] `python -c "import yaml; c=yaml.safe_load(open('config/data_catalog.yml')); print(len(c['assets']))"` → >= 30 assets
  - [ ] 每个 asset 包含 asset_id、asset_type、name、location

  **QA Scenarios**:
  ```
  Scenario: catalog 包含所有已知资产类型
    Tool: Bash (python)
    Steps:
      1. python3 -c "
  import yaml
  c = yaml.safe_load(open('config/data_catalog.yml'))
  types = set(a['asset_type'] for a in c['assets'])
  print(f'Asset types: {sorted(types)}')
  print(f'Total assets: {len(c[\"assets\"])}')
  "
    Expected Result: types 包含 raw_file, sqlite_table, api_endpoint, feishu_export, log_source
    Evidence: .sisyphus/evidence/task-4-catalog.txt

  Scenario: catalog 引用真实文件路径
    Tool: Bash (python)
    Steps:
      1. python3 -c "
  import os, yaml
  c = yaml.safe_load(open('config/data_catalog.yml'))
  missing = []
  for a in c['assets']:
      if 'location' in a and os.path.exists(a['location']):
          continue
      elif a['asset_type'] in ('api_endpoint', 'feishu_export'):
          continue
      else:
          missing.append(f'{a[\"asset_id\"]}: {a.get(\"location\", \"N/A\")}')
  print(f'Missing locations: {len(missing)}')
  for m in missing: print(f'  {m}')
  "
    Expected Result: Missing locations: 0
    Evidence: .sisyphus/evidence/task-4-catalog-paths.txt
  ```

  **Commit**: YES
  - Message: `feat(governance): add data catalog config`

- [x] 5. `config/data_classification.yml` — 数据分级分类

  **What to do**:
  - 为所有数据资产分配敏感等级，使用 5 级分类：
    - `public_internal`: 聚合指标、dashboard 摘要
    - `internal`: DWD 表、metrics、alert_events、API 内部端点
    - `sensitive`: seller_id、customer_unique_id、原始订单数据
    - `pii`: 个人身份信息字段
    - `derived_sensitive`: 从 sensitive 数据派生的指标
  - 每条记录：asset_ref（指向 catalog asset_id）、level、rationale、applies_to_fields（如适用）
  - 字段级分类示例：
    ```yaml
    - asset_ref: dwd_order_level
      level: sensitive
      rationale: "Contains customer purchase history with seller linkages"
      applies_to_fields:
        - customer_unique_id: pii
        - seller_id: internal
        - payment_value: internal
        - review_score: internal
    ```

  **Must NOT do**:
  - 不修改原始数据
  - 不在分类中暴露真实数据值

  **Future Direction** (注释标记):
  - 在配置顶部添加 `# Future: Sensitive Data Scanner — automatic detection of PII/sensitive patterns` 注释
  - 参考 Foundry Sensitive Data Scanner 概念：自动扫描新数据、匹配预定义敏感模式、触发标记/告警

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: 基于已知 schema 的字段级标注

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T1-T4, T6-T9)
  - **Blocks**: T26（API classification 端点）
  - **Blocked By**: T4（data_catalog.yml 提供 asset_id 引用）

  **References**:
  - `config/data_catalog.yml`（由 T4 产出）— asset_id 引用
  - `sql/schema.sql:101-192` — 所有表的列定义
  - `docs/data_dictionary.md` — 字段含义参考
  - `api/routers/outbox.py:30-40` — API 端点涉敏操作参考

  **Acceptance Criteria**:
  - [ ] `python -c "import yaml; c=yaml.safe_load(open('config/data_classification.yml')); print(len(c['classifications']))"` → >= 15 classifications
  - [ ] 5 个分类级别全部被使用

  **QA Scenarios**:
  ```
  Scenario: 所有 5 个敏感等级都被使用
    Tool: Bash (python)
    Steps:
      1. python3 -c "
  import yaml
  c = yaml.safe_load(open('config/data_classification.yml'))
  levels = set(cl['level'] for cl in c['classifications'])
  print(f'Levels used: {sorted(levels)}')
  "
    Expected Result: Levels used: ['derived_sensitive', 'internal', 'pii', 'public_internal', 'sensitive']
    Evidence: .sisyphus/evidence/task-5-classification.txt
  ```

  **Commit**: YES
  - Message: `feat(governance): add data classification config`

- [x] 6. `config/data_markings.yml` — 数据敏感标记（基于 Foundry Markings 强制控制模型）

  **What to do**:
  - 定义标记（Markings）及 Foundry 核心控制属性：
    ```yaml
    # Markings are MANDATORY controls (restrict access). Roles are DISCRETIONARY (expand access).
    # Markings are CONJUNCTIVE (AND): user must satisfy ALL markings on a resource.
    # Markings INHERIT along file hierarchy AND data dependencies.
    markings:
      PII:
        # Mandatory control — even Owners cannot bypass without expand_access permission
        mandatory_control: true
        # Binary all-or-nothing access
        access_type: binary
        # Conjunctive — must satisfy this AND all other markings on the resource
        conjunctive: true
        # Inheritance paths (Foundry: file hierarchy + data dependency)
        inheritance:
          - file_hierarchy    # Projects/folders inherit to children
          - data_dependency   # Upstream data markings flow to downstream derived data
        applies_to:
          - raw_customers.customer_unique_id
          - dwd_order_level.customer_unique_id
        policy: "Do not expose in frontend or Feishu export"
        expand_access_permission: "data_protection_officer"  # centralized removal control

      OPERATIONAL_INTERNAL:
        mandatory_control: true
        access_type: binary
        conjunctive: true
        inheritance:
          - data_dependency
        applies_to:
          - alert_events
          - strategy_recommendations
          - action_tasks
        policy: "Visible to operators only"

      FINANCIAL_INTERNAL:
        mandatory_control: true
        access_type: binary
        conjunctive: true
        inheritance:
          - data_dependency
        applies_to:
          - dwd_order_level.payment_value
          - dwd_item_level.price
          - metric_daily.gmv
        policy: "Aggregated GMV is public_internal; raw payment values are internal"

      RAW_DATA:
        mandatory_control: true
        access_type: binary
        conjunctive: true
        inheritance:
          - data_dependency
        applies_to:
          - raw_customers
          - raw_orders
          - raw_order_items
        policy: "Raw data may contain unprocessed PII. Pipeline stage: raw → processed markings should change"

    # Pipeline stages pattern (Foundry example: Raw Data Marking → processed → lower sensitivity)
    pipeline_stage_markings:
      - stage: raw
        marking: RAW_DATA
      - stage: processed
        marking: null  # Hashed/encrypted → remove RAW_DATA, apply appropriate marking
    ```
  - 每条标记包含：mandatory_control、access_type、conjunctive、inheritance、applies_to、policy、expand_access_permission

  **Must NOT do**:
  - 不创建与 classification 冲突的标记
  - 不在标记中包含实际敏感数据

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: 基于已知数据模型 + Foundry Markings 模型的结构化标注

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T1-T5, T7-T9)
  - **Blocks**: T26（API markings 端点）
  - **Blocked By**: T5（data_classification.yml 提供分类基础）

  **References**:
  - `config/data_classification.yml`（由 T5 产出）— 分类级别参考
  - `sql/schema.sql:60-80` — dwd_order_level 所有字段
  - `sql/schema.sql:101-118` — alert_events 字段
  - Palantir Markings 下载文档 — `docs/external/palantir_aip_foundry/05_security/markings.md`
    - 核心概念：mandatory vs discretionary control、binary all-or-nothing access、conjunctive (AND)
    - 双路径继承：file hierarchy + data dependency propagation
    - Expand Access permission：集中化的 marking 移除控制

  **Acceptance Criteria**:
  - [ ] `python -c "import yaml; m=yaml.safe_load(open('config/data_markings.yml')); print(len(m['markings']))"` → >= 4 markings
  - [ ] 每个 marking 包含 mandatory_control、inheritance、policy
  - [ ] 至少 1 个 marking 使用 `data_dependency` 继承路径

  **QA Scenarios**:
  ```
  Scenario: markings 包含 Foundry 核心控制属性
    Tool: Bash (python)
    Steps:
      1. python3 -c "
  import yaml
  m = yaml.safe_load(open('config/data_markings.yml'))
  for k, v in m['markings'].items():
      mandatory = v.get('mandatory_control')
      inheritance = v.get('inheritance', [])
      conjunctive = v.get('conjunctive')
      print(f'{k}: mandatory={mandatory}, inheritance={inheritance}, conjunctive={conjunctive}')
  "
    Expected Result: 所有 marking mandatory=True, inheritance 非空, conjunctive=True
    Evidence: .sisyphus/evidence/task-6-markings.txt
  ```

  **Commit**: YES
  - Message: `feat(governance): add data markings config (Foundry Mandatory Control model)`

- [x] 7. `config/data_lineage.yml` — 数据血缘关系

  **What to do**:
  - 记录 Baxi 数据管道的完整血缘图，使用 Foundry lineage graph element 类型：
    ```yaml
    nodes:
      - id: raw_orders_csv
        type: source           # Foundry: external data source
        label: "olist_orders_dataset.csv"
        status: active
      - id: dwd_order_level
        type: dataset          # Foundry: internal dataset/table
        label: "dwd_order_level"
        status: active
      - id: metric_daily
        type: dataset
        label: "metric_daily"
        status: active
      - id: alert_events
        type: object_type      # Foundry: Ontology object type (business object)
        label: "alert_events"
        status: active
      - id: strategy_recommendations
        type: object_type
        label: "strategy_recommendations"
        status: active
      - id: feishu_daily_metrics
        type: sync             # Foundry: external sync destination
        label: "飞书-每日经营指标"
        status: active
      # (issues nodes for known data quality problems)
      - id: issue_geo_duplicates
        type: issue            # Foundry: data quality issue marker
        label: "Geo dedup (261,831 rows)"
        status: active
        linked_to: raw_geolocation_csv
    edges:
      - from: raw_orders_csv
        to: dwd_order_level
        transform: "ingestion: CSV → SQLite"
        transform_type: batch_load
      - from: dwd_order_level
        to: metric_daily
        transform: "aggregation: daily GMV/orders/customers"
        transform_type: sql_aggregation
      - from: metric_daily
        to: alert_events
        transform: "rule engine: anomaly detection"
        transform_type: heuristic_rule
      - from: alert_events
        to: strategy_recommendations
        transform: "strategy generation: heuristic → recommendation"
        transform_type: heuristic_rule
      - from: strategy_recommendations
        to: action_tasks
        transform: "task creation: recommendation → work item"
        transform_type: template_instantiation
      - from: action_tasks
        to: event_outbox
        transform: "dispatch: task → outbox event"
        transform_type: channel_routing
      - from: event_outbox
        to: feishu_daily_metrics
        transform: "sync: outbox → Feishu Bitable"
        transform_type: api_sync
    ```
  - 覆盖完整链路：raw CSV → DWD → metrics → alerts → strategies → tasks → outbox → Feishu
  - 每个 edge 包含 transform 描述和 transform_type
  - 节点类型遵循 Foundry lineage graph elements：source, dataset, object_type, artifact, issue, sync, deleted

  **Must NOT do**:
  - 不包含实际数据内容
  - 不遗漏关键的 alert→strategy→task→outbox→feishu 链路

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: 已知管道结构，结构化 DAG 定义

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T1-T6, T8-T9)
  - **Blocks**: T26（API lineage 端点）
  - **Blocked By**: None

  **References**:
  - `sql/schema.sql:1-200` — 所有表及关系
  - `scripts/db_rule_engine.py:1-50` — 规则引擎输入输出表
  - `scripts/db_dispatch_outbox.py:1-50` — 分发流程
  - `docs/entity_relationships.md` — ER 关系文档
  - `services/feishu_service.py` — 飞书同步链路

  **Acceptance Criteria**:
  - [ ] `python -c "import yaml; l=yaml.safe_load(open('config/data_lineage.yml')); print(f'nodes={len(l[\"nodes\"])}, edges={len(l[\"edges\"])}')"` → nodes >= 15, edges >= 12
  - [ ] 包含 raw → dwd → metric → alert → strategy → task → outbox → feishu 完整链路

  **QA Scenarios**:
  ```
  Scenario: 血缘图包含关键节点和边
    Tool: Bash (python)
    Steps:
      1. python3 -c "
  import yaml
  l = yaml.safe_load(open('config/data_lineage.yml'))
  node_ids = {n['id'] for n in l['nodes']}
  required = ['raw_orders_csv', 'dwd_order_level', 'metric_daily', 'alert_events', 'strategy_recommendations', 'action_tasks', 'event_outbox']
  missing = [n for n in required if n not in node_ids]
  print(f'Nodes: {len(l[\"nodes\"])} total, missing required: {missing}')
  print(f'Edges: {len(l[\"edges\"])} total')
  "
    Expected Result: missing required: []，edges >= 12
    Evidence: .sisyphus/evidence/task-7-lineage.txt
  ```

  **Commit**: YES
  - Message: `feat(governance): add data lineage config`

- [x] 8. 版本号升级 v0.5.2 → v0.5.3

  **What to do**:
  - 查找项目中所有包含 "0.5.2" 版本号的文件
  - 将版本号从 "0.5.2" 更新为 "0.5.3"
  - 涉及文件：`pyproject.toml`（`version = "0.5.2"` → `"0.5.3"`）、`frontend/package.json`（`"version": "0.5.2"` → `"0.5.3"`）、OpenAPI spec docs/

  **Must NOT do**:
  - 不修改版本号之外的任何内容

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: 搜索替换，无逻辑变更

  **Parallelization**:
  - **Can Run In Parallel**: NO（顺序：需在所有其他 Wave 1 任务之前完成？实际上可以并行，版本号与其他任务无关）
  - **Parallel Group**: Wave 1 (with T1-T7, T9)
  - **Blocks**: None
  - **Blocked By**: None

  **References**:
  - `pyproject.toml:3` — `version = "0.5.1"`（实际当前版本可能是 0.5.1 或 0.5.2）
  - `frontend/package.json:3` — `"version": "0.5.1"`
  - `docs/openapi-v0.5.2.json` — OpenAPI spec 文件名

  **Acceptance Criteria**:
  - [ ] `grep -r "0.5.3" pyproject.toml frontend/package.json` → 匹配

  **QA Scenarios**:
  ```
  Scenario: 版本号已更新
    Tool: Bash (grep)
    Steps:
      1. grep 'version.*0.5.3' pyproject.toml
      2. grep '"version".*0.5.3' frontend/package.json
    Expected Result: 两个文件都返回匹配
    Evidence: .sisyphus/evidence/task-8-version.txt
  ```

  **Commit**: YES（与 T1 合并）
  - Message: `chore: bump version to 0.5.3`

- [x] 9. 配置 YAML Schema 校验测试（TDD RED 阶段）

  **What to do**:
  - 创建 `tests/test_governance_configs.py`
  - 写 RED 测试：测试每个治理配置文件的 YAML 结构和必填字段
  - 此时所有治理配置文件尚未创建，测试应 FAIL
  - 测试结构：
    ```python
    class TestGovernanceConfigs:
        def test_data_catalog_has_assets(self):
            """RED: config/data_catalog.yml should exist and have assets list"""
            # Will fail until T4 completes
            with open('config/data_catalog.yml') as f:
                data = yaml.safe_load(f)
            assert 'assets' in data
            assert len(data['assets']) > 0

        def test_data_classification_levels(self): ...
        def test_data_markings_structure(self): ...
        def test_data_lineage_dag_valid(self): ...
        def test_checkpoint_rules_format(self): ...
        # ... (one test per config file)
    ```

  **Must NOT do**:
  - 不写 GREEN 实现（这是 RED 阶段）
  - 不创建 config 文件（那是 T4-T7, T10-T14 的任务）

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: 测试模板编写，无复杂逻辑

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T1-T8)
  - **Blocks**: None（测试预计 FAIL 直到各 config 创建完）
  - **Blocked By**: None

  **References**:
  - `tests/` 目录 — 现有测试文件命名和结构惯例
  - `pyproject.toml` — pytest 配置
  - `config/alert_rules.yml` — 现有 YAML 结构模式

  **Acceptance Criteria**:
  - [ ] `pytest tests/test_governance_configs.py -v --tb=short 2>&1 | tail -20` → 显示 FAILED（RED 阶段）
  - [ ] 测试文件包含至少 8 个 test_ 函数（对应 8 个 config）

  **QA Scenarios**:
  ```
  Scenario: 测试文件存在且 RED
    Tool: Bash (pytest)
    Steps:
      1. pytest tests/test_governance_configs.py -v --tb=line 2>&1
    Expected Result: 所有测试 FAILED（因为 config 文件尚未创建），exit code != 0
    Evidence: .sisyphus/evidence/task-9-configs-red.txt
  ```

  **Commit**: YES
  - Message: `test(governance): add config validation tests (RED)`

### Wave 2 — 操作治理配置 + 策略文档

- [x] 10. `config/checkpoint_rules.yml` — 敏感操作检查点

  **What to do**:
  - 定义需要 justification 的敏感操作，使用 Foundry Checkpoints scope 模型：
    ```yaml
    # Checkpoint scopes (Foundry: Organization > Space > Endpoint)
    # Organization: applies across all spaces
    # Space: applies within a specific space/project
    # Endpoint: applies to a specific API endpoint
    checkpoints:
      feishu_sync_apply:
        scope: endpoint
        endpoint: "POST /api/v1/feishu/sync"
        requires_justification: true
        prompt: "Why is this Feishu sync being applied?"
        record_fields: [actor, request_id, tables, dry_run, apply, justification]
        checkpoint_types: [export, sync]    # Foundry: checkpoint type classification
      outbox_dispatch_apply:
        scope: endpoint
        endpoint: "POST /api/v1/outbox/dispatch"
        requires_justification: true
        prompt: "Why are these outbox events being dispatched?"
        record_fields: [actor, request_id, event_count, channel, dry_run, apply, justification]
        checkpoint_types: [export, dispatch]
      pipeline_run:
        scope: endpoint
        endpoint: "POST /api/v1/pipeline/run"
        requires_justification: false  # daily pipeline is routine
      feishu_export:
        scope: endpoint
        endpoint: "POST /api/v1/feishu/export"
        requires_justification: true
        prompt: "Why is this data being exported to Feishu?"
        checkpoint_types: [export]
      data_deletion:
        scope: organization          # Organization-wide: any data deletion requires justification
        requires_justification: true
        prompt: "Why is this data being deleted? Confirm lineage-aware cascade impact."
        checkpoint_types: [deletion, sensitive_action]
    ```
  - 每条规则：scope、endpoint、requires_justification、prompt、checkpoint_types、record_fields

  **Must NOT do**:
  - 不修改现有 API 路由代码（checkpoint 是配置层，执行时由 governance API 读取）
  - 不为 routine 操作（如 health check）添加 checkpoint

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: 基于已知 API 的结构化规则定义

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with T11-T19)
  - **Blocks**: T20（checkpoint 策略文档）、T26（API checkpoint 端点）
  - **Blocked By**: None

  **References**:
  - `api/routers/outbox.py:60-80` — POST dispatch 的 dry-run/apply 模式
  - `api/routers/feishu.py:40-60` — POST feishu 操作
  - `api/routers/pipeline.py` — POST pipeline/run
  - `config/action_registry.yml` — 现有 action 权限模式

  **Acceptance Criteria**:
  - [ ] `python -c "import yaml; c=yaml.safe_load(open('config/checkpoint_rules.yml')); print(len(c['checkpoints']))"` → >= 3 checkpoints
  - [ ] 至少包含 feishu_sync_apply 和 outbox_dispatch_apply

  **QA Scenarios**:
  ```
  Scenario: checkpoint 规则引用真实 API 端点
    Tool: Bash (python)
    Steps:
      1. python3 -c "
  import yaml
  c = yaml.safe_load(open('config/checkpoint_rules.yml'))
  for k, v in c['checkpoints'].items():
      has_endpoint = 'endpoint' in v
      has_prompt = 'prompt' in v if v.get('requires_justification') else True
      print(f'{k}: endpoint={has_endpoint}, prompt_ok={has_prompt}')
  "
    Expected Result: 所有规则 endpoint=True, prompt_ok=True
    Evidence: .sisyphus/evidence/task-10-checkpoints.txt
  ```

  **Commit**: YES
  - Message: `feat(governance): add checkpoint rules config`

- [x] 11. `config/retention_policies.yml` — 数据保留与删除策略
- [x] 12. `config/health_checks.yml` — 数据健康检查规则
- [x] 13. `config/decision_eval_rules.yml` — 决策评估规则
- [x] 14. `config/access_policy.yml` — 访问控制策略

  **What to do**:
  - 整合现有 `action_registry.yml` 和 `owner_mapping.yml`，定义完整访问控制模型：
    ```yaml
    access_policy:
      roles:
        - role: business_ops
          allowed_actions: [create_feishu_report, notify_owner, recommend_business_strategy, modify_business_policy]
          data_access: [metric_daily, alert_events, strategy_recommendations, action_tasks]
        - role: seller_ops
          allowed_actions: [notify_owner, create_followup_task]
          data_access: [dwd_item_level, metric_dimension_daily, alert_events]
        - role: category_ops
          allowed_actions: [notify_owner, create_followup_task, recommend_business_strategy]
          data_access: [dwd_item_level, metric_dimension_daily]
        - role: marketing_ops
          allowed_actions: [recommend_business_strategy]
          data_access: [metric_daily, dwd_order_level]
      default_policy: "deny_all"
    ```
  - 每条角色：allowed_actions、data_access（SQLite 表名）、classification 限制

  **Must NOT do**:
  - 不修改 `action_registry.yml` 或 `owner_mapping.yml`
  - 不实现 RBAC 引擎（策略定义层）

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: 基于已有权限数据的结构化整合

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with T10-T13, T15-T19)
  - **Blocks**: T19（access control 策略文档）
  - **Blocked By**: T5（data_classification.yml）

  **References**:
  - `config/action_registry.yml` — 现有 action 权限表
  - `config/owner_mapping.yml` — 角色-用户映射
  - `config/data_classification.yml`（T5 产出）— 数据敏感等级

  **Acceptance Criteria**:
  - [ ] `python -c "import yaml; a=yaml.safe_load(open('config/access_policy.yml')); print(len(a['access_policy']['roles']))"` → >= 4 roles
  - [ ] 包含 default_policy

  **QA Scenarios**:
  ```
  Scenario: access_policy 角色定义完整
    Tool: Bash (python)
    Steps:
      1. python3 -c "
  import yaml
  a = yaml.safe_load(open('config/access_policy.yml'))
  for r in a['access_policy']['roles']:
      has_actions = 'allowed_actions' in r
      has_data = 'data_access' in r
      print(f'{r[\"role\"]}: actions={has_actions}, data={has_data}')
  "
    Expected Result: 所有角色 actions=True, data=True
    Evidence: .sisyphus/evidence/task-14-access.txt
  ```

  **Commit**: YES
  - Message: `feat(governance): add access policy config`

- [x] 15. `docs/governance/baxi_data_governance_policy.md` — 数据治理总纲
- [x] 16. `docs/governance/baxi_ontology_model.md` — 本体/对象模型
- [x] 17. `docs/governance/baxi_data_lineage_model.md` — 数据血缘模型
- [x] 18. `docs/governance/baxi_data_marking_policy.md` — 数据标记策略
- [x] 19. `docs/governance/baxi_access_control_model.md` — 访问控制模型

  **What to do**:
  - 描述 Baxi 的访问控制模型，约 800-1200 字
  - 结构：
    1. Access Model Overview: Bearer Token + 角色基础
    2. Roles & Permissions: 4 个角色及其权限矩阵
    3. Data Access Matrix: 角色 × 数据表访问权限
    4. Action Authorization: dry-run/apply 流程中的授权
    5. Audit Trail: 如何审计访问行为

  **Must NOT do**:
  - 不在文档中暴露实际的 Bearer Token 值

  **Recommended Agent Profile**:
  - **Category**: `writing`
    - Reason: 安全文档

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with T10-T18)
  - **Blocks**: None
  - **Blocked By**: T14（access_policy.yml）

  **Acceptance Criteria**:
  - [ ] 文件 >= 600 字
  - [ ] 包含 Roles & Permissions 和 Data Access Matrix 章节

  **QA Scenarios**:
  ```
  Scenario: access control 文档描述 4 个角色
    Tool: Bash (grep)
    Steps:
      1. grep -c "business_ops\|seller_ops\|category_ops\|marketing_ops" docs/governance/baxi_access_control_model.md
    Expected Result: >= 4
    Evidence: .sisyphus/evidence/task-19-access-doc.txt
  ```

  **Commit**: YES
  - Message: `docs(governance): add access control model`

### Wave 3 — 操作治理文档 + 后端 API + 数据库迁移

- [x] 20. `docs/governance/baxi_checkpoint_policy.md` — 检查点策略
- [x] 21. `docs/governance/baxi_retention_policy.md` — 数据保留策略
- [x] 22. `docs/governance/baxi_data_health_checks.md` — 数据健康检查
- [x] 23. `docs/governance/baxi_decision_eval_policy.md` — 决策评估策略
- [x] 24. `docs/governance/baxi_aip_alignment.md` — AIP 对齐文档

  **What to do**:
  - 编写 Baxi 与 Palantir AIP/Foundry 的对齐分析，约 1200-1800 字
  - 结构：
    1. AIP Overview: Palantir AIP 的核心治理理念摘要
    2. Baxi Capability Map: 按 AIP Architecture 的 **12 个能力类别**逐项对比：
       1. Secure LLM Integration — Baxi 现状（heuristic, LLM 代码就绪未激活）
       2. End-to-end Observability — logging + audit CSVs
       3. Context Engineering — raw CSV → DWD → metrics pipeline
       4. Ontology System — aip_object_schema.yml (9 objects, semantic + kinetic)
       5. Vector/Compute/Tool Services — Pandas/SQLite (simple, not Foundry-scale)
       6. Security & Governance — 本版本（v0.5.3）补齐
       7. Agent Lifecycle — rule engines (heuristic agents)
       8. Operational Automation — outbox dispatch + feishu sync
       9. Development Environments — scripts/ + api/ (pro-code only)
       10. Human + AI Applications — React console + Feishu Bitable
       11. Package/Release/Deploy — git + manual deploy (simple)
       12. Enterprise Automation — 未来方向（LLM + Qoder）
    3. Gap Analysis: Baxi 已有什么、缺什么（按 12 类别表格）
    4. Roadmap: 本版本（v0.5.3）及后续版本的计划
    5. References: 指向 manifest.yml 和下载的参考文档

  **Must NOT do**:
  - 不声称 Baxi 是 Palantir 的替代品

  **Recommended Agent Profile**:
  - **Category**: `writing`

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with T20-T23, T25-T28)
  - **Blocked By**: T2-T3（Palantir 文档参考）、T15-T19（策略文档基础）

  **Acceptance Criteria**:
  - [ ] 文件 >= 800 字
  - [ ] 包含 Capacity Map 和 Gap Analysis 章节

  **QA Scenarios**:
  ```
  Scenario: AIP alignment 对比表
    Tool: Bash
    Steps:
      1. grep -c "✅\|❌\|🟡" docs/governance/baxi_aip_alignment.md
    Expected Result: >= 4（至少 4 个对比项）
    Evidence: .sisyphus/evidence/task-24-alignment.txt
  ```

  **Commit**: YES
  - Message: `docs(governance): add AIP alignment analysis`

- [x] 25. `scripts/config.py` — 注册治理 YAML 路径常量
- [x] 26. `api/routers/governance.py` — 治理数据 API 端点 (TDD)
- [x] 27. `api/main.py` — 注册 governance router
- [x] 28. SQLite governance 审计表（migration 008）

  **What to do**:
  - 创建 `sql/migrations/008_governance.sql`：
    ```sql
    -- Governance audit tables for checkpoint records and health check results
    CREATE TABLE IF NOT EXISTS governance_checkpoints (
        checkpoint_id INTEGER PRIMARY KEY AUTOINCREMENT,
        action_type TEXT NOT NULL,        -- 'feishu_sync_apply', 'outbox_dispatch_apply', etc.
        endpoint TEXT NOT NULL,           -- API endpoint called
        actor TEXT NOT NULL,              -- who performed the action
        request_id TEXT,                  -- API request ID
        justification TEXT,               -- user-provided reason
        mode TEXT NOT NULL DEFAULT 'dry_run',  -- 'dry_run' or 'apply'
        status TEXT NOT NULL DEFAULT 'recorded',
        metadata_json TEXT,               -- additional context as JSON
        created_at TEXT NOT NULL DEFAULT (datetime('now'))
    );

    CREATE TABLE IF NOT EXISTS governance_health_results (
        result_id INTEGER PRIMARY KEY AUTOINCREMENT,
        check_id TEXT NOT NULL,           -- references health_checks.yml check id
        check_type TEXT NOT NULL,         -- 'lineage', 'governance', 'quality', 'audit'
        status TEXT NOT NULL,             -- 'pass', 'fail', 'warn', 'error'
        detail TEXT,                      -- human-readable result
        checked_at TEXT NOT NULL DEFAULT (datetime('now'))
    );
    ```
  - 在 `sql/schema.sql` 末尾追加这两个表定义
  - 初始化 migration：执行 SQL

  **Must NOT do**:
  - 不修改现有 12 张表的结构
  - 不删除现有数据

  **Recommended Agent Profile**:
  - **Category**: `quick`

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with T20-T25)
  - **Blocks**: None
  - **Blocked By**: None

  **References**:
  - `sql/schema.sql:1-200` — 现有表定义模式
  - `sql/migrations/005_dispatch_adapters.sql` — 迁移文件格式
  - `sql/indexes.sql` — 索引定义（可选为新表添加索引）

  **Acceptance Criteria**:
  - [ ] `sqlite3 data/olist_ops.db ".tables" | grep governance` → 匹配
  - [ ] `sqlite3 data/olist_ops.db ".schema governance_checkpoints"` → 返回 DDL

  **QA Scenarios**:
  ```
  Scenario: governance 表已创建
    Tool: Bash (sqlite3)
    Steps:
      1. sqlite3 data/olist_ops.db ".tables" | tr ' ' '\n' | grep governance
    Expected Result: governance_checkpoints, governance_health_results
    Evidence: .sisyphus/evidence/task-28-migration.txt
  ```

  **Commit**: YES
  - Message: `feat(governance): add governance audit tables (migration 008)`

### Wave 4 — 前端治理中心

- [x] 29. `frontend/src/api/governance.ts` — 前端 API 类型定义 + TanStack Query hooks
- [x] 30. `frontend/src/pages/Governance.tsx` — 治理中心页面
- [x] 31. `frontend/src/App.tsx` — 添加 /governance 路由
- [x] 32. `frontend/src/pages/__tests__/Governance.test.tsx` — 前端组件测试（TDD GREEN）

  **What to do**:
  - 创建 Vitest + Testing Library 测试：
    ```typescript
    describe('Governance Page', () => {
      it('renders all 6 tabs', async () => { ... });
      it('shows loading state initially', async () => { ... });
      it('displays catalog data after fetch', async () => { ... });
      it('handles API error gracefully', async () => { ... });
      it('navigates between tabs', async () => { ... });
    });
    ```
  - Mock TanStack Query hooks 返回值

  **Must NOT do**:
  - 不测试 TanStack Query 内部行为（只测试组件渲染）

  **Recommended Agent Profile**:
  - **Category**: `quick`

  **Parallelization**:
  - **Can Run In Parallel**: NO（依赖 T30）
  - **Blocked By**: T30（Governance 组件）

  **References**:
  - `frontend/src/pages/__tests__/` — 现有测试文件模式
  - `frontend/vitest.config.ts` — vitest 配置

  **Acceptance Criteria**:
  - [ ] `cd frontend && npx vitest run src/pages/__tests__/Governance.test.tsx` → PASS
  - [ ] 至少 5 个 test case

  **QA Scenarios**:
  ```
  Scenario: 前端测试全部通过
    Tool: Bash
    Steps:
      1. cd frontend && npx vitest run src/pages/__tests__/Governance.test.tsx 2>&1
    Expected Result: Tests: 5+ passed, 0 failed
    Evidence: .sisyphus/evidence/task-32-vitest.txt
  ```

  **Commit**: YES
  - Message: `test(frontend): add governance page tests`

---

## Final Verification Wave (MANDATORY — after ALL implementation tasks)

> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.
>
> **Do NOT auto-proceed after verification. Wait for user's explicit approval.**

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists. For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found.
  - Verify all 10 governance docs exist in `docs/governance/`
  - Verify all 9 config files exist in `config/`
  - Verify governance API endpoints respond correctly
  - Verify frontend /governance route works
  - Verify .gitignore includes docs/external/
  - Verify NO modifications to FROZEN scripts
  - Verify NO modifications to existing 18 configs
  - Verify NO modifications to existing 9 API routers
  - Check evidence files exist in `.sisyphus/evidence/`
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  - Backend: Run `ruff check api/routers/governance.py tests/test_governance*.py`
  - Frontend: Run `cd frontend && npx tsc --noEmit`
  - Run `cd frontend && npx vitest run`
  - Run `pytest tests/test_governance*.py -v`
  - Review for AI slop: excessive comments, unused imports, console.log in prod, `as any`/`@ts-ignore`
  Output: `Lint [PASS/FAIL] | TypeScript [PASS/FAIL] | Tests [N pass/N fail] | VERDICT`

- [ ] F3. **Real Manual QA** — `unspecified-high` (+ `playwright` skill)
  - Start API server: `cd /home/zzz/project/baxi && python -m uvicorn api.main:app --port 8765`
  - Start frontend: `cd frontend && npx vite --port 5173`
  - Execute EVERY QA scenario from EVERY task — follow exact steps, capture evidence
  - Test cross-task integration: API endpoints returning config data, frontend consuming API
  - Test edge cases: missing config file (404), invalid YAML, auth failure, empty data
  - Save to `.sisyphus/evidence/final-qa/`
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  - For each task: read "What to do", read actual diff (git diff)
  - Verify 1:1 — everything in spec was built, nothing beyond spec was built
  - Check "Must NOT do" compliance per task
  - Detect cross-task contamination: Wave N task touching Wave M task's files
  - Flag unaccounted changes (files modified outside plan scope)
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

| Wave | Tasks | Commit Message |
|------|-------|---------------|
| 1 | T1+T8 | `chore(governance): add .gitignore rule, directory scaffold, bump to v0.5.3` |
| 1 | T3 | `docs(governance): add Palantir reference manifest` |
| 1 | T4 | `feat(governance): add data catalog config` |
| 1 | T5 | `feat(governance): add data classification config` |
| 1 | T6 | `feat(governance): add data markings config` |
| 1 | T7 | `feat(governance): add data lineage config` |
| 1 | T9 | `test(governance): add config validation tests (RED)` |
| 2 | T10 | `feat(governance): add checkpoint rules config` |
| 2 | T11 | `feat(governance): add retention policies config` |
| 2 | T12 | `feat(governance): add health checks config` |
| 2 | T13 | `feat(governance): add decision eval rules config` |
| 2 | T14 | `feat(governance): add access policy config` |
| 2 | T15 | `docs(governance): add data governance policy` |
| 2 | T16 | `docs(governance): add ontology model` |
| 2 | T17 | `docs(governance): add data lineage model` |
| 2 | T18 | `docs(governance): add data marking policy` |
| 2 | T19 | `docs(governance): add access control model` |
| 3 | T20-T24 | `docs(governance): add checkpoint, retention, health, eval, AIP docs` |
| 3 | T25 | `feat(governance): register governance config paths` |
| 3 | T26+T27 | `feat(governance): add governance API endpoints + router` |
| 3 | T28 | `feat(governance): add governance audit tables (migration 008)` |
| 4 | T29 | `feat(frontend): add governance API types and hooks` |
| 4 | T30+T31 | `feat(frontend): add governance center page + route` |
| 4 | T32 | `test(frontend): add governance page tests` |

---

## Success Criteria

### Verification Commands
```bash
# Backend: all governance tests pass
pytest tests/test_governance*.py -v --tb=short

# Frontend: TypeScript compiles + tests pass
cd frontend && npx tsc --noEmit && npx vitest run

# Config: all 9 governance YAML files parse correctly
for f in config/data_{catalog,classification,markings,lineage}.yml config/{checkpoint_rules,retention_policies,health_checks,decision_eval_rules,access_policy}.yml; do python3 -c "import yaml; yaml.safe_load(open('$f')); print(f'OK: $f')" || echo "FAIL: $f"; done

# API: governance endpoints respond
TOKEN=$(grep API_BEARER_TOKEN .env | cut -d= -f2)
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8765/api/v1/governance/catalog | jq '.assets | length'
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8765/api/v1/governance/lineage | jq '.nodes | length'

# Git: docs/external/ is ignored
git status --porcelain | grep 'docs/external/' && echo "FAIL: should be gitignored" || echo "OK: gitignored"
```

### Final Checklist
- [ ] All 10 governance docs present in `docs/governance/`
- [ ] All 9 config files present in `config/`
- [ ] Palantir reference docs downloaded + manifest.yml
- [ ] `.gitignore` excludes `docs/external/`
- [ ] Version bumped to 0.5.3
- [ ] Governance API router registered, 7 endpoints working
- [ ] Governance API tests all pass (pytest)
- [ ] Frontend /governance page renders with 6 tabs
- [ ] Frontend tests all pass (vitest)
- [ ] Migration 008 applied (governance tables exist)
- [ ] All "Must NOT Have" verified (no contamination of existing code)
- [ ] All evidence files captured in `.sisyphus/evidence/`
