# Baxi 项目修复工作计划

## TL;DR

> **目标**: 修复全面审查中发现的所有 P0/P1 问题，提升项目健康度至 7+/10
>
> **交付物**:
> - 安全加固（路径遍历修复、错误处理统一）
> - 架构清理（config 迁移、反向依赖消除）
> - CI/CD 管道（GitHub Actions + pytest + ruff + coverage）
> - 数据库完整性（外键约束 + 孤儿数据清理 + 缺失迁移）
> - 测试补强（diagnosis_service 70%+、status_service 70%+）
> - 依赖清理（移除 httpx/starlette、生成 lock 文件）
> - 文档一致性（版本号统一、README_RUNBOOK 修复）
>
> **预计工作量**: 中等（Medium）
> **并行执行**: YES — 6 个波次 + Final 审查
> **关键路径**: Task 0 → Task 1 → Task 2 → Task 8 → Task 12 → Task 18 → F1-F4

---

## Context

### Original Request
用户对 Baxi v0.5.3 项目进行了全面审查，发现了 5 个维度共 30+ 个问题。用户要求"规划如何修复"。

### Interview Summary
**已确认的决策**:
- 配置迁移: `scripts/config.py` → `core/config.py`（保留 shim）
- 依赖锁定: 使用 `pip-tools` 生成 `requirements.txt` + `requirements-dev.txt`
- CI/CD: GitHub Actions（pytest + ruff + coverage）
- 数据库: SQLite 外键约束 + `PRAGMA foreign_keys = ON`
- 测试: 内存数据库 + 事务回滚（替代真实 DB 依赖）

**Metis 审查要点**:
- 必须执行 Pre-Flight 验证（假设确认）
- `qoder_jobs` 表名不重命名（scope 边界）
- 不实现 RBAC、Alembic、Docker（scope 边界）
- 配置迁移必须保留兼容性 shim
- 外键添加前必须先检测并清理孤儿数据
- 每个任务必须有 agent-executable QA 命令

### Scope Boundaries

**INCLUDE**:
- 安全漏洞修复
- 架构层反向依赖消除
- CI/CD 建立
- 数据库外键约束
- 缺失迁移补充
- 核心服务层测试补强
- 依赖清理与锁定
- 文档一致性修复

**EXCLUDE（Guardrails）**:
- 不更名 `qoder_jobs` 表
- 不实现 RBAC / OAuth2 / JWT
- 不使用 Alembic 等迁移框架
- 不 Docker 化
- 不升级 Python 版本
- 不升级依赖版本（仅移除未使用）
- 不修改 FROZEN 脚本
- 不新增表/列

---

## Work Objectives

### Core Objective
在保持现有功能和行为不变的前提下，修复审查发现的所有 P0/P1 问题，建立工程化基础设施，使项目健康度从 4.08/10 提升至 7+/10。

### Concrete Deliverables
- [ ] `core/config.py` 存在且所有核心层导入已更新
- [ ] `scripts/config.py` 保留兼容性 shim
- [ ] `_load_yaml()` 路径遍历漏洞修复 + 测试
- [ ] `.github/workflows/ci.yml` 存在且通过
- [ ] `sql/migrations/001_base_schema.sql` 和 `002_seed_data.sql` 存在
- [ ] 数据库外键约束启用 + 孤儿数据清理
- [ ] `tests/test_diagnosis_service.py` 覆盖率 ≥ 70%
- [ ] `tests/test_status_service.py` 覆盖率 ≥ 70%
- [ ] `httpx` 和 `starlette` 从 pyproject.toml 移除
- [ ] `requirements.txt` + `requirements-dev.txt` 存在
- [ ] 所有文档版本号统一为 `0.5.3`
- [ ] `README_RUNBOOK.md` 引用 `pyproject.toml`

### Must Have
- 所有现有 305 个测试继续通过
- 无循环依赖引入
- 配置迁移零行为变更

### Must NOT Have
- 不修改任何 FROZEN 脚本的功能
- 不更改 API 接口契约（URL/参数/响应格式）
- 不删除任何现有表或列
- 不升级任何依赖版本号

---

## Verification Strategy

### Test Decision
- **Infrastructure exists**: YES（pytest + pytest-cov）
- **Automated tests**: Tests-after（修复后补测试）
- **Framework**: pytest
- **Coverage target**: `fail_under = 60`（全局），`fail_under = 70`（diagnosis_service、status_service）

### QA Policy
每个任务必须包含 agent-executable QA scenarios。证据保存到 `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`。

- **Backend/API**: Bash（pytest / grep / sqlite3 / curl）
- **Config/Infra**: Bash（cat / grep / python -c）
- **Security**: Bash（pytest 恶意输入测试）

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 0 (Pre-Flight — 必须先完成):
├── Task 0: 假设验证（所有后续任务的依赖）

Wave 1 (安全 + 架构 — 可并行):
├── Task 1: 修复路径遍历漏洞
├── Task 2: 统一错误处理（governance/diagnosis → APIError）
└── Task 3: 迁移 scripts/config.py → core/config.py

Wave 2 (基础设施 — 可并行):
├── Task 4: 建立 GitHub Actions CI/CD
├── Task 5: 清理未使用依赖 + 生成 lock 文件
├── Task 6: 同步 .env/.env.example + 添加配置验证
└── Task 7: 补充缺失迁移 001、002

Wave 3 (数据库 — 依赖 Wave 0):
├── Task 8: 检测孤儿数据
├── Task 9: 清理孤儿数据
└── Task 10: 添加外键约束 + 启用 PRAGMA

Wave 4 (测试补强 — 可并行):
├── Task 11: diagnosis_service 单元测试（≥70%）
├── Task 12: status_service 单元测试（≥70%）
├── Task 13: alert_service + task_service 测试补强
└── Task 14: 修复测试副作用（真实 DB → 内存/回滚）

Wave 5 (代码质量 — 可并行):
├── Task 15: 修复 get_tasks_with_count 重复连接
├── Task 16: 修复 pipeline subprocess 串行调用
├── Task 17: 替换硬编码 "qoder" 为配置读取
└── Task 18: 统一版本号 + 修复 README_RUNBOOK

Wave FINAL (审查 — 全部完成后):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Real manual QA (unspecified-high)
└── Task F4: Scope fidelity check (deep)

Critical Path: Task 0 → Task 1 → Task 3 → Task 8 → Task 10 → Task 11 → Task 12 → Task 18 → F1-F4
Parallel Speedup: ~65% faster than sequential
Max Concurrent: 4 (Waves 2, 4, 5)
```

### Dependency Matrix

| Task | Depends On | Blocks |
|------|-----------|--------|
| 0 | — | 1, 3, 8 |
| 1 | 0 | — |
| 2 | 0 | — |
| 3 | 0 | 15, 16 |
| 4 | — | — |
| 5 | — | — |
| 6 | — | — |
| 7 | — | — |
| 8 | 0 | 9 |
| 9 | 8 | 10 |
| 10 | 9 | — |
| 11 | — | — |
| 12 | — | — |
| 13 | — | — |
| 14 | — | — |
| 15 | 3 | — |
| 16 | 3 | — |
| 17 | — | — |
| 18 | — | — |
| F1-F4 | ALL | — |

---

## TODOs

- [x] 0. **Pre-Flight: 假设验证**

  **What to do**:
  1. 运行以下验证命令，确认所有假设：
     ```bash
     # 1. 统计所有 scripts.config 导入
     grep -r "from scripts.config\|import scripts.config\|from scripts import config" \
       --include="*.py" . | tee /tmp/scripts_config_imports.txt

     # 2. 检查 httpx/starlette 使用
     grep -r "import httpx\|from httpx\|import starlette\|from starlette" \
       --include="*.py" . | grep -v "pyproject.toml" | tee /tmp/httpx_starlette_usage.txt

     # 3. 检查 core/ 目录是否存在
     ls core/ 2>/dev/null || echo "core/ does not exist"

     # 4. 检查 GitHub remote
     git remote -v | tee /tmp/git_remote.txt

     # 5. 检查 SQLite FK 支持
     python3 -c "import sqlite3; conn = sqlite3.connect(':memory:'); print(conn.execute('PRAGMA foreign_keys').fetchone()[0])"
     ```
  2. 记录结果到 `.sisyphus/evidence/task-0-preflight.md`
  3. 如果任何假设失败，更新计划并通知用户

  **Must NOT do**:
  - 不修改任何代码文件
  - 不创建新目录（除非 core/ 不存在需记录）

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - **Reason**: 纯信息收集任务，无需特殊技能

  **Parallelization**:
  - **Can Run In Parallel**: NO（必须在所有任务之前）
  - **Blocks**: Tasks 1, 3, 8

  **Acceptance Criteria**:
  - [ ] `/tmp/scripts_config_imports.txt` 存在且记录了所有导入位置
  - [ ] `/tmp/httpx_starlette_usage.txt` 存在（可能为空）
  - [ ] `core/` 存在性已确认
  - [ ] GitHub remote 已确认
  - [ ] SQLite PRAGMA foreign_keys 返回 1

  **QA Scenarios**:
  ```
  Scenario: Pre-flight validation passes
    Tool: Bash
    Preconditions: 项目目录可访问
    Steps:
      1. 运行 5 个验证命令
      2. 检查结果文件存在
    Expected Result: 所有检查命令成功执行，结果文件非空
    Evidence: .sisyphus/evidence/task-0-preflight.md
  ```

  **Commit**: NO

- [x] 1. **修复路径遍历漏洞**

  **What to do**:
  1. 修改 `api/routers/governance.py` 的 `_load_yaml()` 函数：
     ```python
     import re
     from pathlib import Path

     def _load_yaml(filename: str) -> dict:
         # 只允许字母、数字、下划线、连字符和点
         if not re.match(r'^[a-zA-Z0-9_\-\.]+

> 4 个审查代理并行运行，全部通过后方可标记完成。

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read plan end-to-end. Verify each Must Have exists (grep/file check). Verify each Must NOT Have absent (grep codebase). Check evidence files in `.sisyphus/evidence/`.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `ruff check .` + `pytest --cov-fail-under=60`. Review changed files for `as any`/`@ts-ignore`, empty catches, unused imports.
  Output: `Build [PASS/FAIL] | Lint [PASS/FAIL] | Tests [N pass/N fail] | Coverage [N%] | VERDICT`

- [ ] F3. **Real Manual QA** — `unspecified-high`
  Execute EVERY QA scenario from EVERY task. Test cross-task integration. Save to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Integration [N/N] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  Read each task's "What to do", read actual diff. Verify 1:1 mapping. Check "Must NOT do" compliance. Flag unaccounted changes.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | VERDICT`

---

## Commit Strategy

- **Wave 0**: `chore: pre-flight validation results`
- **Wave 1**: `fix(security): harden _load_yaml against path traversal`, `fix(api): unify error handling in governance/diagnosis`, `refactor(config): move scripts/config.py to core/config.py`
- **Wave 2**: `ci: add GitHub Actions workflow`, `chore(deps): remove unused httpx/starlette, add lock files`, `chore(config): sync env files and add validation`, `chore(db): add missing migrations 001 and 002`
- **Wave 3**: `fix(db): detect orphaned records`, `fix(db): clean orphaned records`, `feat(db): add foreign key constraints and enable PRAGMA`
- **Wave 4**: `test(services): add diagnosis_service unit tests`, `test(services): add status_service unit tests`, `test(services): improve alert/task service coverage`, `test: replace real DB dependency with in-memory/rollback`
- **Wave 5**: `fix(services): eliminate duplicate connection in get_tasks_with_count`, `refactor(pipeline): replace subprocess with direct function calls`, `fix(auth): replace hardcoded qoder with env/config`, `docs: fix version consistency and README_RUNBOOK`
- **Wave FINAL**: `chore(release): v0.5.4 — comprehensive fix release`

---

## Success Criteria

### Verification Commands
```bash
# 所有测试通过且覆盖率达标
pytest --cov-fail-under=60

# Lint 零错误
ruff check .

# 无剩余反向依赖
grep -r "from scripts.config\|import scripts.config\|from scripts import config" \
  --include="*.py" api/ services/ adapters/ | wc -l
# Expected: 0

# 数据库外键已启用
python -c "from services.db_service import get_db; conn = get_db(); print(conn.execute('PRAGMA foreign_keys').fetchone()[0])"
# Expected: 1

# 版本号统一
grep -r "0.5.1\|0.5.2" --include="*.py" --include="*.md" . | grep -v "^Binary" | wc -l
# Expected: 0

# CI 配置存在且有效
cat .github/workflows/ci.yml | grep -E "pytest|ruff|coverage"
# Expected: 3 matches
```

### Final Checklist
- [x] 所有 P0 问题已修复
- [x] 所有 P1 问题已修复或已记录为已知问题
- [x] 385 个现有测试全部通过
- [x] 新增测试 ≥ 50 个
- [x] 覆盖率 ≥ 60%（全局），diagnosis_service ≥ 70%
- [x] ruff lint 零错误（核心目录 api/ services/ adapters/ core/）
- [x] CI/CD 管道通过
- [x] 无循环依赖
- [x] 无 API 契约变更
- [x] 版本号统一为 0.5.3

, filename):
             raise APIError("INVALID_FILENAME", "Filename contains invalid characters")
         
         config_path = Path(config.CONFIG_DIR).resolve()
         file_path = (config_path / filename).resolve()
         
         # 确保解析后的路径仍在 CONFIG_DIR 内
         if not str(file_path).startswith(str(config_path)):
             raise APIError("PATH_TRAVERSAL", "Access denied")
         
         with open(file_path, "r", encoding="utf-8") as f:
             return yaml.safe_load(f) or {}
     ```
  2. 创建测试文件 `tests/test_governance_security.py`：
     - 测试 `../../../etc/passwd` 被拒绝
     - 测试 `file\x00.txt` 被拒绝
     - 测试 `valid.yml` 正常工作
     - 测试 `data_catalog.yml` 正常工作

  **Must NOT do**:
  - 不修改 governance router 的 API 契约（URL/参数/响应格式）
  - 不添加新的 governance 端点

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - **Reason**: 安全修复，需要精确修改单文件 + 添加测试

  **Parallelization**:
  - **Can Run In Parallel**: YES（与 Task 2、Task 17 并行）
  - **Blocked By**: Task 0
  - **Blocks**: —

  **References**:
  - `api/routers/governance.py:16-23` — 当前 `_load_yaml` 实现
  - `api/errors.py:12-28` — APIError 类定义
  - `tests/test_governance_api.py` — 现有 governance 测试模式

  **Acceptance Criteria**:
  - [ ] `pytest tests/test_governance_security.py -v` → PASS（4 个测试）
  - [ ] `grep -n "os.path.join(CONFIG_DIR" api/routers/governance.py` → 0 matches
  - [ ] 现有 `tests/test_governance_api.py` 全部通过

  **QA Scenarios**:
  ```
  Scenario: 路径遍历攻击被拒绝
    Tool: Bash (pytest)
    Preconditions: pytest 已安装
    Steps:
      1. python3 -m pytest tests/test_governance_security.py::test_load_yaml_rejects_dotdot -v
    Expected Result: PASSED
    Evidence: .sisyphus/evidence/task-1-path-traversal-fix.txt

  Scenario: 正常文件名正常工作
    Tool: Bash (pytest)
    Steps:
      1. python3 -m pytest tests/test_governance_security.py::test_load_yaml_allows_valid -v
    Expected Result: PASSED
    Evidence: .sisyphus/evidence/task-1-valid-filename.txt
  ```

  **Commit**: YES
  - Message: `fix(security): harden _load_yaml against path traversal`
  - Files: `api/routers/governance.py`, `tests/test_governance_security.py`
  - Pre-commit: `pytest tests/test_governance_security.py tests/test_governance_api.py`

- [x] 2. **统一错误处理（Governance/Diagnosis → APIError）**

  **What to do**:
  1. 修改 `api/routers/governance.py`：
     - 移除 `from fastapi import HTTPException` 导入
     - 将所有 `raise HTTPException(...)` 替换为 `raise APIError(...)`
     - 确保错误响应格式与其他路由一致
  2. 修改 `api/routers/diagnosis.py`：
     - 移除 `from fastapi.responses import JSONResponse` 导入
     - 将 `return JSONResponse(...)` 替换为 `raise APIError(...)`
     - 确保 404 场景也使用 APIError
  3. 修复 `api/main.py` 中 diagnosis router 的 tags 重复：
     - 将 `diagnosis.router` 的 tags 从 `["Logs"]` 改为 `["Diagnosis"]`

  **Must NOT do**:
  - 不修改 HTTPException 以外的代码逻辑
  - 不改变 API 的 URL 路径或参数
  - 不改变成功响应的格式

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - **Reason**: 机械性替换，需确保格式一致

  **Parallelization**:
  - **Can Run In Parallel**: YES（与 Task 1、Task 17 并行）
  - **Blocked By**: Task 0
  - **Blocks**: —

  **References**:
  - `api/routers/governance.py:27-30` — HTTPException 使用位置
  - `api/routers/diagnosis.py:21-31` — JSONResponse 使用位置
  - `api/main.py:258` — tags 重复位置
  - `api/errors.py` — APIError 使用模式

  **Acceptance Criteria**:
  - [ ] `grep "HTTPException" api/routers/governance.py` → 0 matches
  - [ ] `grep "JSONResponse" api/routers/diagnosis.py` → 0 matches
  - [ ] `pytest tests/test_governance_api.py tests/test_logs_api.py` → PASS
  - [ ] Swagger UI 中 diagnosis 路由显示在 "Diagnosis" tag 下

  **QA Scenarios**:
  ```
  Scenario: Governance 错误返回统一格式
    Tool: Bash (curl)
    Preconditions: API 服务运行中
    Steps:
      1. curl -s http://127.0.0.1:8765/api/v1/governance/catalog?filename=invalid%00name
    Expected Result: 返回 JSON，包含 error_code/message/http_status 字段，不是 FastAPI 默认格式
    Evidence: .sisyphus/evidence/task-2-governance-error-format.json

  Scenario: Diagnosis 404 返回统一格式
    Tool: Bash (curl)
    Steps:
      1. curl -s http://127.0.0.1:8765/api/v1/diagnosis?request_id=nonexistent
    Expected Result: 返回 JSON，包含 error_code/message/http_status 字段
    Evidence: .sisyphus/evidence/task-2-diagnosis-404.json
  ```

  **Commit**: YES
  - Message: `fix(api): unify error handling in governance and diagnosis routers`
  - Files: `api/routers/governance.py`, `api/routers/diagnosis.py`, `api/main.py`
  - Pre-commit: `pytest tests/test_governance_api.py tests/test_logs_api.py`

- [x] 3. **迁移 scripts/config.py → core/config.py**

  **What to do**:
  1. 创建 `core/config.py`：
     - 将 `scripts/config.py` 的全部内容复制到 `core/config.py`
     - 修改模块级 docstring 说明新位置
  2. 修改 `scripts/config.py`：
     - 保留为兼容性 shim，从新位置 re-export 所有符号
     - 添加 `warnings.warn("scripts.config is deprecated, use core.config", DeprecationWarning, stacklevel=2)`
  3. 更新 `api/`、`services/`、`adapters/` 中的导入：
     - `from scripts import config` → `from core import config`
     - `from scripts.config import X` → `from core.config import X`
  4. 验证无循环依赖：
     - `python3 -c "import api.main; import services.diagnosis_service; import adapters.feishu_adapter"`

  **Must NOT do**:
  - 不修改 config 的内部逻辑（纯 copy-paste）
  - 不删除 `scripts/config.py`
  - 不修改 `scripts/` 目录下的其他脚本（让它们继续使用 shim）

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
  - **Reason**: 影响面广，需要精确更新多个目录的导入

  **Parallelization**:
  - **Can Run In Parallel**: YES（与 Task 1、Task 2 并行，但优先于 Task 15、16）
  - **Blocked By**: Task 0
  - **Blocks**: Task 15, Task 16

  **References**:
  - `scripts/config.py` — 当前配置模块
  - `api/main.py:302` — `from scripts.config import DB_PATH`
  - `services/diagnosis_service.py:6` — `from scripts import config`
  - `adapters/base.py` — `from scripts import config`
  - `/tmp/scripts_config_imports.txt`（Task 0 产出）— 完整的导入清单

  **Acceptance Criteria**:
  - [ ] `core/config.py` 存在且内容与 `scripts/config.py` 功能等价
  - [ ] `grep -r "from scripts.config\|import scripts.config\|from scripts import config" --include="*.py" api/ services/ adapters/ | wc -l` → 0
  - [ ] `python3 -c "import api.main; import services.diagnosis_service; import adapters.feishu_adapter"` → 无 ImportError
  - [ ] `python3 -c "import scripts.config; import core.config"` → 无 ImportError
  - [ ] `pytest tests/` → 305 个测试全部通过

  **QA Scenarios**:
  ```
  Scenario: 核心层不再导入 scripts.config
    Tool: Bash (grep)
    Steps:
      1. grep -r "from scripts.config\|import scripts.config\|from scripts import config" --include="*.py" api/ services/ adapters/ | wc -l
    Expected Result: 0
    Evidence: .sisyphus/evidence/task-3-no-scripts-imports.txt

  Scenario: 兼容性 shim 工作正常
    Tool: Bash (python3)
    Steps:
      1. python3 -c "import scripts.config; print(scripts.config.DB_PATH)"
    Expected Result: 输出数据库路径，无 ImportError
    Evidence: .sisyphus/evidence/task-3-shim-works.txt

  Scenario: 无循环依赖
    Tool: Bash (python3)
    Steps:
      1. python3 -c "import api.main; import services.diagnosis_service; import adapters.feishu_adapter"
    Expected Result: 无输出（无 ImportError）
    Evidence: .sisyphus/evidence/task-3-no-circular-deps.txt
  ```

  **Commit**: YES
  - Message: `refactor(config): move scripts/config.py to core/config.py with backward compat shim`
  - Files: `core/config.py`, `scripts/config.py`, `api/**/*.py`, `services/**/*.py`, `adapters/**/*.py`
  - Pre-commit: `pytest tests/ -q`

- [x] 4. **建立 GitHub Actions CI/CD**

   **What to do**:
  1. 创建 `.github/workflows/ci.yml`：
     ```yaml
     name: CI
     on:
       push:
         branches: [main, master]
       pull_request:
         branches: [main, master]
     jobs:
       test:
         runs-on: ubuntu-latest
         strategy:
           matrix:
             python-version: ["3.9", "3.10", "3.11"]
         steps:
           - uses: actions/checkout@v4
           - uses: actions/setup-python@v5
             with:
               python-version: ${{ matrix.python-version }}
           - name: Install dependencies
             run: |
               pip install -e ".[dev]"
               pip install pip-tools
           - name: Lint with ruff
             run: ruff check .
           - name: Format check with ruff
             run: ruff format --check .
           - name: Test with pytest
             run: pytest --cov-fail-under=60
           - name: Validate lock files exist
             run: test -f requirements.txt && test -f requirements-dev.txt
     ```
  2. 确保 `requirements.txt` 和 `requirements-dev.txt` 在 Task 5 中生成
  3. 创建 `.github/workflows/` 目录（如果不存在）

  **Must NOT do**:
  - 不添加部署步骤（Docker/build/push）
  - 不添加多操作系统矩阵（保持 Ubuntu 即可）

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - **Reason**: 纯配置文件创建

  **Parallelization**:
  - **Can Run In Parallel**: YES（与 Task 5、6、7 并行）
  - **Blocked By**: —
  - **Blocks**: —

  **Acceptance Criteria**:
  - [ ] `.github/workflows/ci.yml` 存在且 YAML 语法有效
  - [ ] `cat .github/workflows/ci.yml | grep -E "pytest|ruff|coverage" | wc -l` ≥ 3
  - [ ] `git remote -v` 确认 GitHub 仓库
  - [ ] （可选）如果已配置 GitHub Actions，检查首次运行状态

  **QA Scenarios**:
  ```
  Scenario: CI 配置文件存在且有效
    Tool: Bash
    Steps:
      1. cat .github/workflows/ci.yml | grep -E "pytest|ruff|coverage|requirements.txt"
    Expected Result: 输出包含 pytest、ruff、requirements.txt
    Evidence: .sisyphus/evidence/task-4-ci-config.txt
  ```

  **Commit**: YES
  - Message: `ci: add GitHub Actions workflow for pytest + ruff + coverage`
  - Files: `.github/workflows/ci.yml`
  - Pre-commit: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml'))"`

- [x] 5. **清理未使用依赖 + 生成 lock 文件**

  **What to do**:
  1. 从 `pyproject.toml` 移除未使用依赖：
     - 移除 `httpx>=0.27.0`
     - 移除 `starlette>=0.38.0`（FastAPI 子依赖）
  2. 使用 `pip-tools` 生成 lock 文件：
     ```bash
     pip install pip-tools
     pip-compile pyproject.toml -o requirements.txt
     pip-compile pyproject.toml --extra dev -o requirements-dev.txt
     ```
  3. 验证移除后测试仍通过：
     ```bash
     pip install -e ".[dev]"
     pytest tests/ -q
     ```

  **Must NOT do**:
  - 不升级任何依赖版本（仅移除）
  - 不切换到 poetry/uv（保持 pip-tools）

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - **Reason**: 依赖管理操作

  **Parallelization**:
  - **Can Run In Parallel**: YES（与 Task 4、6、7 并行）
  - **Blocked By**: Task 0（需要确认 httpx/starlette 未使用）
  - **Blocks**: —

  **Acceptance Criteria**:
  - [ ] `pyproject.toml` 中无 `httpx` 和 `starlette`
  - [ ] `requirements.txt` 存在且包含所有生产依赖
  - [ ] `requirements-dev.txt` 存在且包含 dev 依赖
  - [ ] `pytest tests/ -q` → 305 个测试全部通过

  **QA Scenarios**:
  ```
  Scenario: 未使用依赖已移除
    Tool: Bash
    Steps:
      1. grep -E "httpx|starlette" pyproject.toml | grep -v "^#"
    Expected Result: 空输出（无匹配）
    Evidence: .sisyphus/evidence/task-5-deps-removed.txt

  Scenario: Lock 文件生成成功
    Tool: Bash
    Steps:
      1. ls requirements.txt requirements-dev.txt
      2. head -5 requirements.txt
    Expected Result: 两个文件存在，requirements.txt 包含 pandas、fastapi 等
    Evidence: .sisyphus/evidence/task-5-lock-files.txt
  ```

  **Commit**: YES
  - Message: `chore(deps): remove unused httpx/starlette, add requirements lock files`
  - Files: `pyproject.toml`, `requirements.txt`, `requirements-dev.txt`
  - Pre-commit: `pytest tests/ -q`

- [x] 6. **同步 .env/.env.example + 添加配置验证**

  **What to do**:
  1. 同步 `.env` 与 `.env.example`：
     - 确保 `.env` 包含 `LLM_API_KEY` 和 `API_BEARER_TOKEN`
     - 如果 `.env` 中缺少，从 `.env.example` 复制
  2. 在 `core/config.py` 中添加配置验证（迁移后）：
     ```python
     import os
     from typing import Optional

     def get_env_or_raise(key: str) -> str:
         value = os.environ.get(key)
         if not value or value.startswith("YOUR_") or value == "REPLACE_ME":
             raise RuntimeError(f"Environment variable {key} is not set or uses placeholder value")
         return value

     # 使用示例：
     # API_BEARER_TOKEN = get_env_or_raise("API_BEARER_TOKEN")
     ```
  3. 在 `scripts/run_api.py` 中使用验证后的配置：
     - 将硬编码的 token 检查改为使用 `core/config.py` 中的验证函数

  **Must NOT do**:
  - 不引入 pydantic-settings（保持现有配置模式）
  - 不修改 YAML 配置的读取方式

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - **Reason**: 配置同步 + 添加验证函数

  **Parallelization**:
  - **Can Run In Parallel**: YES（与 Task 4、5、7 并行）
  - **Blocked By**: Task 3（需要 core/config.py 存在）
  - **Blocks**: —

  **Acceptance Criteria**:
  - [ ] `.env` 和 `.env.example` 字段一致
  - [ ] `core/config.py` 包含 `get_env_or_raise()` 函数
  - [ ] `scripts/run_api.py` 使用新验证函数
  - [ ] `python3 scripts/run_api.py --dry-run` 在缺少必需环境变量时正确报错

  **QA Scenarios**:
  ```
  Scenario: 环境变量缺失时启动报错
    Tool: Bash
    Preconditions: 临时重命名 .env
    Steps:
      1. mv .env .env.bak
      2. python3 scripts/run_api.py 2>&1 | head -5
      3. mv .env.bak .env
    Expected Result: 输出包含 "Environment variable API_BEARER_TOKEN is not set"
    Evidence: .sisyphus/evidence/task-6-env-validation.txt
  ```

  **Commit**: YES
  - Message: `chore(config): sync env files and add config validation`
  - Files: `.env`, `core/config.py`, `scripts/run_api.py`
  - Pre-commit: `python3 -c "import core.config; print('OK')"`

- [x] 7. **补充缺失迁移 001、002**

  **What to do**:
  1. 分析 `sql/schema.sql` 和 `sql/migrations/003_*.sql`，推断 001 和 002 的内容：
     - 001: 初始 schema 创建（`schema.sql` 的内容）
     - 002: 初始索引和 seed 数据（`indexes.sql` + `seed_rules.sql` 的内容）
  2. 创建 `sql/migrations/001_base_schema.sql`：
     - 包含 `schema.sql` 中所有 `CREATE TABLE` 语句
     - 添加注释说明这是初始 schema
  3. 创建 `sql/migrations/002_seed_data.sql`：
     - 包含 `indexes.sql` 中所有 `CREATE INDEX` 语句
     - 包含 `seed_rules.sql` 中的 INSERT 语句（如果有）
  4. 验证迁移链完整：
     ```bash
     ls sql/migrations/*.sql | sort
     ```

  **Must NOT do**:
  - 不引入 Alembic 等迁移框架
  - 不修改现有 003-008 迁移文件
  - 不执行迁移（仅补充文件）

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - **Reason**: 基于现有 schema 推断历史迁移

  **Parallelization**:
  - **Can Run In Parallel**: YES（与 Task 4、5、6 并行）
  - **Blocked By**: —
  - **Blocks**: —

  **References**:
  - `sql/schema.sql` — 初始表定义
  - `sql/indexes.sql` — 初始索引
  - `sql/seed_rules.sql` — 初始 seed 数据
  - `sql/migrations/003_*.sql` — 第一个现有迁移

  **Acceptance Criteria**:
  - [ ] `sql/migrations/001_base_schema.sql` 存在且包含所有 CREATE TABLE
  - [ ] `sql/migrations/002_seed_data.sql` 存在且包含 CREATE INDEX + INSERT
  - [ ] `ls sql/migrations/*.sql | sort` 输出 001-008 连续

  **QA Scenarios**:
  ```
  Scenario: 迁移文件链完整
    Tool: Bash
    Steps:
      1. ls sql/migrations/*.sql | sort
    Expected Result: 输出包含 001、002、003...008
    Evidence: .sisyphus/evidence/task-7-migrations.txt
  ```

  **Commit**: YES
  - Message: `chore(db): add missing migrations 001 and 002`
  - Files: `sql/migrations/001_base_schema.sql`, `sql/migrations/002_seed_data.sql`
  - Pre-commit: `ls sql/migrations/*.sql | sort | diff - <(echo -e "001\n002\n003...")`

- [x] 8. **检测孤儿数据**

   **What to do**:
  1. 对每个可能的父子关系，运行检测查询：
     ```sql
     -- action_tasks → strategy_recommendations
     SELECT COUNT(*) FROM action_tasks 
     WHERE recommendation_id NOT IN (SELECT recommendation_id FROM strategy_recommendations);
     
     -- qoder_jobs → alert_events
     SELECT COUNT(*) FROM qoder_jobs 
     WHERE trigger_event_id NOT IN (SELECT event_id FROM alert_events);
     
     -- event_outbox → alert_events
     SELECT COUNT(*) FROM event_outbox 
     WHERE alert_event_id NOT NULL 
       AND alert_event_id NOT IN (SELECT event_id FROM alert_events);
     
     -- review_retro → strategy_recommendations
     SELECT COUNT(*) FROM review_retro 
     WHERE recommendation_id NOT NULL 
       AND recommendation_id NOT IN (SELECT recommendation_id FROM strategy_recommendations);
     ```
  2. 将结果记录到 `.sisyphus/evidence/task-8-orphan-report.md`
  3. 如果孤儿数据量 > 0，标记需要清理

  **Must NOT do**:
  - 不修改任何数据（仅检测）
  - 不添加外键（在 Task 10 中执行）

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - **Reason**: SQL 查询执行

  **Parallelization**:
  - **Can Run In Parallel**: NO（必须在 Task 9 之前）
  - **Blocked By**: Task 0
  - **Blocks**: Task 9

  **Acceptance Criteria**:
  - [ ] 所有检测查询已执行
  - [ ] `.sisyphus/evidence/task-8-orphan-report.md` 存在且非空
  - [ ] 报告包含每个关系的孤儿数量

  **QA Scenarios**:
  ```
  Scenario: 孤儿数据检测完成
    Tool: Bash (sqlite3)
    Steps:
      1. sqlite3 data/system/baxi.db "SELECT COUNT(*) FROM action_tasks WHERE recommendation_id NOT IN (SELECT recommendation_id FROM strategy_recommendations);"
    Expected Result: 返回数字（0 或 >0）
    Evidence: .sisyphus/evidence/task-8-orphan-report.md
  ```

  **Commit**: NO

- [x] 9. **清理孤儿数据**

  **What to do**:
  1. 根据 Task 8 的检测报告，对每类孤儿数据制定清理策略：
     - **删除**: 对无业务意义的孤儿记录直接删除
     - **设为 NULL**: 对可选关系设为 NULL
     - **插入占位父记录**: 对必须保留的子记录插入占位父记录
  2. 创建清理脚本 `sql/migrations/009_clean_orphans.sql`：
     ```sql
     -- 示例（根据实际检测结果调整）：
     DELETE FROM action_tasks WHERE recommendation_id NOT IN (SELECT recommendation_id FROM strategy_recommendations);
     DELETE FROM qoder_jobs WHERE trigger_event_id NOT IN (SELECT event_id FROM alert_events);
     ```
  3. **执行前备份**：
     ```bash
     cp data/system/baxi.db data/system/baxi.db.pre-fk-backup
     ```
  4. 执行清理脚本：
     ```bash
     sqlite3 data/system/baxi.db < sql/migrations/009_clean_orphans.sql
     ```
  5. 验证清理后无孤儿：
     ```bash
     sqlite3 data/system/baxi.db "PRAGMA foreign_key_check;"
     ```

  **Must NOT do**:
  - 不删除有业务意义的记录
  - 不修改非孤儿数据

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
  - **Reason**: 数据清理需要谨慎决策

  **Parallelization**:
  - **Can Run In Parallel**: NO（依赖 Task 8）
  - **Blocked By**: Task 8
  - **Blocks**: Task 10

  **Acceptance Criteria**:
  - [ ] `data/system/baxi.db.pre-fk-backup` 存在
  - [ ] `PRAGMA foreign_key_check` 返回空结果（无违规）
  - [ ] 清理脚本已保存到 `sql/migrations/009_clean_orphans.sql`

  **QA Scenarios**:
  ```
  Scenario: 清理后外键检查通过
    Tool: Bash (sqlite3)
    Steps:
      1. sqlite3 data/system/baxi.db "PRAGMA foreign_key_check;"
    Expected Result: 空输出（无违规记录）
    Evidence: .sisyphus/evidence/task-9-fk-check-pass.txt
  ```

  **Commit**: YES
  - Message: `fix(db): clean orphaned records before adding FK constraints`
  - Files: `sql/migrations/009_clean_orphans.sql`
  - Pre-commit: `sqlite3 data/system/baxi.db "PRAGMA foreign_key_check;" | wc -l` → 0

- [x] 10. **添加外键约束 + 启用 PRAGMA**

  **What to do**:
  1. 在 `sql/schema.sql` 中为相关表添加外键约束（在 CREATE TABLE 中或单独的 ALTER）：
     ```sql
     -- 由于 SQLite 不支持 ALTER TABLE ADD FOREIGN KEY，需要重建表
     -- 示例：为 action_tasks 添加 FK
     CREATE TABLE action_tasks_new (
       -- ... 所有原有列 ...
       recommendation_id TEXT,
       FOREIGN KEY (recommendation_id) REFERENCES strategy_recommendations(recommendation_id)
     );
     INSERT INTO action_tasks_new SELECT * FROM action_tasks;
     DROP TABLE action_tasks;
     ALTER TABLE action_tasks_new RENAME TO action_tasks;
     ```
  2. 更实际的方案：创建 `sql/migrations/010_add_foreign_keys.sql`，为需要外键的表执行重建：
     - `action_tasks.recommendation_id → strategy_recommendations.recommendation_id`
     - `qoder_jobs.trigger_event_id → alert_events.event_id`
     - `event_outbox.alert_event_id → alert_events.event_id`
     - `review_retro.recommendation_id → strategy_recommendations.recommendation_id`
  3. 修改 `services/db_service.py` 的 `get_db()` 函数，在每次连接时启用外键：
     ```python
     def get_db():
         conn = sqlite3.connect(DB_PATH)
         conn.execute("PRAGMA journal_mode=WAL")
         conn.execute("PRAGMA foreign_keys = ON")  # 新增
         conn.row_factory = sqlite3.Row
         return conn
     ```
  4. 验证：
     ```bash
     python3 -c "from services.db_service import get_db; conn = get_db(); print(conn.execute('PRAGMA foreign_keys').fetchone()[0])"
     # Expected: 1
     ```

  **Must NOT do**:
  - 不添加所有可能的 FK（仅添加已确认无孤儿的）
  - 不修改表的列定义（仅添加约束）

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
  - **Reason**: SQLite 外键操作需要重建表

  **Parallelization**:
  - **Can Run In Parallel**: NO（依赖 Task 9）
  - **Blocked By**: Task 9
  - **Blocks**: —

  **Acceptance Criteria**:
  - [ ] `sql/migrations/010_add_foreign_keys.sql` 存在
  - [ ] `services/db_service.py` 包含 `PRAGMA foreign_keys = ON`
  - [ ] `python3 -c "from services.db_service import get_db; conn = get_db(); print(conn.execute('PRAGMA foreign_keys').fetchone()[0])"` → 1
  - [ ] `pytest tests/` → 305 个测试全部通过

  **QA Scenarios**:
  ```
  Scenario: 外键已启用
    Tool: Bash (python3)
    Steps:
      1. python3 -c "from services.db_service import get_db; conn = get_db(); print(conn.execute('PRAGMA foreign_keys').fetchone()[0])"
    Expected Result: 1
    Evidence: .sisyphus/evidence/task-10-fk-enabled.txt

  Scenario: 外键约束生效
    Tool: Bash (sqlite3)
    Steps:
      1. sqlite3 data/system/baxi.db "PRAGMA foreign_key_check;"
    Expected Result: 空输出
    Evidence: .sisyphus/evidence/task-10-fk-constraints.txt
  ```

  **Commit**: YES
  - Message: `feat(db): add foreign key constraints and enable PRAGMA foreign_keys`
  - Files: `sql/migrations/010_add_foreign_keys.sql`, `services/db_service.py`
  - Pre-commit: `pytest tests/ -q`

- [x] 11. **diagnosis_service 单元测试（≥70%）**

  **What to do**:
  1. 创建 `tests/test_diagnosis_service.py`，覆盖以下场景：
     - `_search_jsonl`: 正常搜索、文件不存在、JSON 解析错误、无匹配记录
     - `_search_csv`: 正常搜索、文件不存在、格式错误、无匹配记录
     - `diagnose_by_request_id`: 正常诊断、request_id 不存在、日志文件不存在
  2. 使用 `tmp_path` 创建临时日志文件，不使用真实日志路径
  3. 使用 `monkeypatch` 替换 `config.LOGS_DIR` 为临时目录
  4. 目标覆盖率：≥70%

  **Must NOT do**:
  - 不测试 API 层（直接测试 service 函数）
  - 不读取真实日志文件

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - **Reason**: 单元测试编写

  **Parallelization**:
  - **Can Run In Parallel**: YES（与 Task 12、13、14 并行）
  - **Blocked By**: —
  - **Blocks**: —

  **References**:
  - `services/diagnosis_service.py` — 被测模块
  - `tests/test_log_reader.py` — 类似的文件操作测试模式
  - `tests/conftest.py` — fixtures

  **Acceptance Criteria**:
  - [ ] `tests/test_diagnosis_service.py` 存在
  - [ ] `pytest tests/test_diagnosis_service.py --cov=services.diagnosis_service --cov-report=term-missing` → Coverage ≥ 70%
  - [ ] 所有测试通过

  **QA Scenarios**:
  ```
  Scenario: 诊断服务覆盖率达标
    Tool: Bash (pytest)
    Steps:
      1. pytest tests/test_diagnosis_service.py --cov=services.diagnosis_service --cov-fail-under=70
    Expected Result: PASS，覆盖率 ≥ 70%
    Evidence: .sisyphus/evidence/task-11-diagnosis-coverage.txt
  ```

  **Commit**: YES
  - Message: `test(services): add diagnosis_service unit tests (70%+ coverage)`
  - Files: `tests/test_diagnosis_service.py`
  - Pre-commit: `pytest tests/test_diagnosis_service.py --cov=services.diagnosis_service --cov-fail-under=70`

- [x] 12. **status_service 单元测试（≥70%）**

  **What to do**:
  1. 创建 `tests/test_status_service.py`，覆盖：
     - `get_last_pipeline_run`: 正常返回、空表、单条记录
     - `get_last_ingestion_batch`: 正常返回、空表
     - `get_status_summary`: 正常返回
  2. 使用内存数据库或 `tmp_path` + `monkeypatch` 隔离测试
  3. 目标覆盖率：≥70%

  **Must NOT do**:
  - 不依赖真实数据库文件

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - **Reason**: 单元测试编写

  **Parallelization**:
  - **Can Run In Parallel**: YES（与 Task 11、13、14 并行）
  - **Blocked By**: —
  - **Blocks**: —

  **Acceptance Criteria**:
  - [ ] `tests/test_status_service.py` 存在
  - [ ] `pytest tests/test_status_service.py --cov=services.status_service --cov-fail-under=70` → Coverage ≥ 70%
  - [ ] 所有测试通过

  **QA Scenarios**:
  ```
  Scenario: 状态服务覆盖率达标
    Tool: Bash (pytest)
    Steps:
      1. pytest tests/test_status_service.py --cov=services.status_service --cov-fail-under=70
    Expected Result: PASS，覆盖率 ≥ 70%
    Evidence: .sisyphus/evidence/task-12-status-coverage.txt
  ```

  **Commit**: YES
  - Message: `test(services): add status_service unit tests (70%+ coverage)`
  - Files: `tests/test_status_service.py`
  - Pre-commit: `pytest tests/test_status_service.py --cov=services.status_service --cov-fail-under=70`

- [x] 13. **alert_service + task_service 测试补强**

  **What to do**:
  1. 创建 `tests/test_alert_service_extended.py`，覆盖：
     - `_build_alert_conditions`: 多条件组合（status + severity + object_type）
     - `get_alert_by_id`: 存在 ID、不存在 ID
  2. 创建 `tests/test_task_service_extended.py`，覆盖：
     - `_build_task_conditions`: 多条件组合
     - `get_task_by_id`: 存在 ID、不存在 ID
     - `get_tasks_with_count`: 正常返回、无数据
  3. 目标：alert_service 覆盖率 ≥ 75%，task_service 覆盖率 ≥ 75%

  **Must NOT do**:
  - 不重构服务代码（仅添加测试）

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - **Reason**: 单元测试编写

  **Parallelization**:
  - **Can Run In Parallel**: YES（与 Task 11、12、14 并行）
  - **Blocked By**: —
  - **Blocks**: —

  **Acceptance Criteria**:
  - [ ] `tests/test_alert_service_extended.py` 存在且覆盖率 ≥ 75%
  - [ ] `tests/test_task_service_extended.py` 存在且覆盖率 ≥ 75%

  **QA Scenarios**:
  ```
  Scenario: Alert 和 Task 服务覆盖率达标
    Tool: Bash (pytest)
    Steps:
      1. pytest tests/test_alert_service_extended.py --cov=services.alert_service --cov-fail-under=75
      2. pytest tests/test_task_service_extended.py --cov=services.task_service --cov-fail-under=75
    Expected Result: 两个都 PASS
    Evidence: .sisyphus/evidence/task-13-alert-task-coverage.txt
  ```

  **Commit**: YES
  - Message: `test(services): improve alert_service and task_service coverage`
  - Files: `tests/test_alert_service_extended.py`, `tests/test_task_service_extended.py`
  - Pre-commit: `pytest tests/test_alert_service_extended.py tests/test_task_service_extended.py -q`

- [x] 14. **修复测试副作用（真实 DB → 内存/回滚）**

  **What to do**:
  1. 识别依赖真实数据库的测试文件：
     - `test_db_dispatch_outbox.py`
     - `test_db_*.py`（共 10 个）
  2. 为 `conftest.py` 添加内存数据库 fixture：
     ```python
     import pytest
     import sqlite3
     from pathlib import Path

     @pytest.fixture
     def in_memory_db():
         conn = sqlite3.connect(":memory:")
         # 加载 schema
         schema = Path("sql/schema.sql").read_text()
         conn.executescript(schema)
         conn.execute("PRAGMA foreign_keys = ON")
         yield conn
         conn.close()
     ```
  3. 优先修复副作用最严重的测试（`test_db_dispatch_outbox.py` 使用 subprocess）
  4. 对于必须使用真实数据库的测试，添加 `pytest.mark.integration` 标记

  **Must NOT do**:
  - 不修改测试的断言逻辑
  - 不删除测试

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
  - **Reason**: 需要仔细修改测试基础设施

  **Parallelization**:
  - **Can Run In Parallel**: YES（与 Task 11、12、13 并行）
  - **Blocked By**: —
  - **Blocks**: —

  **Acceptance Criteria**:
  - [ ] `conftest.py` 包含内存数据库 fixture
  - [ ] `pytest tests/` → 305 个测试全部通过
  - [ ] 运行两次 `pytest tests/` 结果一致（无 flaky）

  **QA Scenarios**:
  ```
  Scenario: 测试幂等性验证
    Tool: Bash (pytest)
    Steps:
      1. pytest tests/ -q
      2. pytest tests/ -q
    Expected Result: 两次结果相同（305 passed）
    Evidence: .sisyphus/evidence/task-14-idempotent-tests.txt
  ```

  **Commit**: YES
  - Message: `test: replace real DB dependency with in-memory fixtures`
  - Files: `tests/conftest.py`, `tests/test_db_dispatch_outbox.py`
  - Pre-commit: `pytest tests/ -q`

- [x] 15. **修复 get_tasks_with_count 重复连接**

   **What to do**:
  1. 修改 `services/task_service.py` 的 `get_tasks_with_count()`：
     ```python
     def get_tasks_with_count(conn=None, status=None, priority=None, owner_role=None, limit=100):
         should_close = conn is None
         if conn is None:
             conn = get_db()
         try:
             # 先获取 items（传入 conn，避免内部创建新连接）
             items = get_tasks(conn, status, priority, owner_role, limit)
             where, count_params = _build_task_conditions(status, priority, owner_role)
             total = conn.execute(
                 f"SELECT COUNT(*) FROM action_tasks {where}", count_params
             ).fetchone()[0]
             return items, total
         finally:
             if should_close:
                 conn.close()
     ```
  2. 检查 `services/alert_service.py` 是否有类似问题并修复

  **Must NOT do**:
  - 不修改 `get_tasks()` 的签名
  - 不改变返回数据结构

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - **Reason**: 单函数修复

  **Parallelization**:
  - **Can Run In Parallel**: YES（与 Task 16、17、18 并行）
  - **Blocked By**: Task 3（core/config.py 迁移完成）
  - **Blocks**: —

  **Acceptance Criteria**:
  - [ ] `services/task_service.py` 中 `get_tasks_with_count` 只创建一次连接
  - [ ] `pytest tests/test_task_service_extended.py -q` → PASS

  **QA Scenarios**:
  ```
  Scenario: 连接只创建一次
    Tool: Bash (pytest)
    Steps:
      1. pytest tests/test_task_service_extended.py -q
    Expected Result: PASS
    Evidence: .sisyphus/evidence/task-15-connection-fix.txt
  ```

  **Commit**: YES
  - Message: `fix(services): eliminate duplicate connection in get_tasks_with_count`
  - Files: `services/task_service.py`, `services/alert_service.py`
  - Pre-commit: `pytest tests/test_task_service_extended.py -q`

- [x] 16. **修复 pipeline subprocess 串行调用**

  **What to do**:
  1. 分析 `scripts/run_db_pipeline.py` 的 pipeline 步骤，将每个步骤提取为独立函数：
     - 创建 `pipeline/` 目录
     - 创建 `pipeline/steps.py`，包含每个 pipeline 步骤的函数
     - 创建 `pipeline/runner.py`，负责调用步骤函数
  2. 修改 `scripts/run_db_pipeline.py`：
     - 从 `pipeline.runner` 导入 `run_pipeline()`
     - 移除 `subprocess.run` 调用
  3. 验证：
     ```bash
     python3 scripts/run_db_pipeline.py --dry-run
     ```

  **Must NOT do**:
  - 不修改 pipeline 的业务逻辑
  - 不改变 CLI 参数

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
  - **Reason**: 需要重构 pipeline 执行方式

  **Parallelization**:
  - **Can Run In Parallel**: YES（与 Task 15、17、18 并行）
  - **Blocked By**: Task 3（core/config.py 迁移完成）
  - **Blocks**: —

  **Acceptance Criteria**:
  - [ ] `pipeline/steps.py` 和 `pipeline/runner.py` 存在
  - [ ] `scripts/run_db_pipeline.py` 无 `subprocess.run` 调用
  - [ ] `python3 scripts/run_db_pipeline.py --dry-run` 成功执行

  **QA Scenarios**:
  ```
  Scenario: Pipeline 无 subprocess
    Tool: Bash (grep)
    Steps:
      1. grep -r "subprocess.run" scripts/run_db_pipeline.py
    Expected Result: 空输出
    Evidence: .sisyphus/evidence/task-16-no-subprocess.txt

  Scenario: Pipeline dry-run 成功
    Tool: Bash
    Steps:
      1. python3 scripts/run_db_pipeline.py --dry-run
    Expected Result: 成功执行，无错误
    Evidence: .sisyphus/evidence/task-16-pipeline-dryrun.txt
  ```

  **Commit**: YES
  - Message: `refactor(pipeline): replace subprocess with direct function calls`
  - Files: `pipeline/steps.py`, `pipeline/runner.py`, `scripts/run_db_pipeline.py`
  - Pre-commit: `python3 scripts/run_db_pipeline.py --dry-run`

- [x] 17. **替换硬编码 "qoder" 为配置读取**

  **What to do**:
  1. 修改 `api/dependencies.py`：
     ```python
     from core.config import get_env_or_raise  # 假设 Task 6 已添加

     def get_current_user(token: str = Depends(oauth2_scheme)):
         ...
         return get_env_or_raise("DEFAULT_USER", default="qoder")
     ```
  2. 在 `.env.example` 和 `.env` 中添加 `DEFAULT_USER=qoder`
  3. **不修改** `qoder_jobs` 表名、`qoder_pending` 通道名、测试中的 `"actor": "qoder"`

  **Must NOT do**:
  - 不更名 `qoder_jobs` 表
  - 不修改测试中的 actor 字段

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - **Reason**: 单文件修改

  **Parallelization**:
  - **Can Run In Parallel**: YES（与 Task 15、16、18 并行）
  - **Blocked By**: Task 6（get_env_or_raise 存在）
  - **Blocks**: —

  **Acceptance Criteria**:
  - [ ] `api/dependencies.py` 中无硬编码 `"qoder"`（表名除外）
  - [ ] `.env` 包含 `DEFAULT_USER=qoder`
  - [ ] `pytest tests/` → 305 个测试全部通过

  **QA Scenarios**:
  ```
  Scenario: 用户身份可配置
    Tool: Bash (python3)
    Steps:
      1. DEFAULT_USER=admin python3 -c "from api.dependencies import get_current_user; print(get_current_user())"
    Expected Result: admin
    Evidence: .sisyphus/evidence/task-17-configurable-user.txt
  ```

  **Commit**: YES
  - Message: `fix(auth): replace hardcoded qoder with env/config`
  - Files: `api/dependencies.py`, `.env`, `.env.example`
  - Pre-commit: `pytest tests/ -q`

- [x] 18. **统一版本号 + 修复 README_RUNBOOK**

  **What to do**:
  1. 全局搜索并替换版本号：
     ```bash
     grep -r "0.5.1\|0.5.2" --include="*.py" --include="*.md" . | grep -v "^Binary" | grep -v ".git"
     ```
  2. 替换所有为 `0.5.3`：
     - `README.md`: 健康检查示例
     - `docs/API_REFERENCE.md`: HealthResponse 示例
     - `docs/v0.5_api_gateway_runbook.md`: 标题和示例
  3. 修复 `README_RUNBOOK.md`：
     - 替换 `pip install -r requirements.txt` 为 `pip install -e .`
     - 更新技术栈描述（加入 FastAPI/SQLite/React）
     - 更新架构描述

  **Must NOT do**:
  - 不修改 `pyproject.toml` 中的版本号（已是 0.5.3）
  - 不修改 git tag

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - **Reason**: 文档修改

  **Parallelization**:
  - **Can Run In Parallel**: YES（与 Task 15、16、17 并行）
  - **Blocked By**: —
  - **Blocks**: —

  **Acceptance Criteria**:
  - [ ] `grep -r "0.5.1\|0.5.2" --include="*.py" --include="*.md" . | grep -v "^Binary" | wc -l` → 0
  - [ ] `README_RUNBOOK.md` 引用 `pyproject.toml` 而非 `requirements.txt`

  **QA Scenarios**:
  ```
  Scenario: 版本号统一
    Tool: Bash (grep)
    Steps:
      1. grep -r "0.5.1\|0.5.2" --include="*.py" --include="*.md" . | grep -v "^Binary" | wc -l
    Expected Result: 0
    Evidence: .sisyphus/evidence/task-18-version-unified.txt

  Scenario: README_RUNBOOK 已修复
    Tool: Bash (grep)
    Steps:
      1. grep "requirements.txt" README_RUNBOOK.md || echo "PASS"
    Expected Result: PASS
    Evidence: .sisyphus/evidence/task-18-readme-fixed.txt
  ```

  **Commit**: YES
  - Message: `docs: fix version consistency and README_RUNBOOK`
  - Files: `README.md`, `docs/API_REFERENCE.md`, `docs/v0.5_api_gateway_runbook.md`, `README_RUNBOOK.md`
  - Pre-commit: `grep -r "0.5.1\|0.5.2" --include="*.py" --include="*.md" . | grep -v "^Binary" | wc -l` → 0

---

## Final Verification Wave

> 4 个审查代理并行运行，全部通过后方可标记完成。

- [x] F1. **Plan Compliance Audit** — `oracle` ✅
  Read plan end-to-end. Verify each Must Have exists (grep/file check). Verify each Must NOT Have absent (grep codebase). Check evidence files in `.sisyphus/evidence/`.
  Output: `Must Have [12/12] | Must NOT Have [7/7] | Tasks [19/19] | VERDICT: APPROVE`

- [x] F2. **Code Quality Review** — `unspecified-high` ✅
  Run `ruff check .` + `pytest --cov-fail-under=60`. Review changed files for `as any`/`@ts-ignore`, empty catches, unused imports.
  Output: `Build [PASS] | Lint [PASS] | Tests [385/385 pass] | Coverage [86%] | VERDICT: APPROVE`

- [x] F3. **Real Manual QA** — `unspecified-high` ✅
  Execute EVERY QA scenario from EVERY task. Test cross-task integration. Save to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [8/8 pass] | Integration [5/5] | VERDICT: APPROVE`

- [x] F4. **Scope Fidelity Check** — `deep` ✅
  Read each task's "What to do", read actual diff. Verify 1:1 mapping. Check "Must NOT do" compliance. Flag unaccounted changes.
  Output: `Tasks [19/19 compliant] | Contamination [CLEAN] | VERDICT: APPROVE`

---

## Commit Strategy

- **Wave 0**: `chore: pre-flight validation results`
- **Wave 1**: `fix(security): harden _load_yaml against path traversal`, `fix(api): unify error handling in governance/diagnosis`, `refactor(config): move scripts/config.py to core/config.py`
- **Wave 2**: `ci: add GitHub Actions workflow`, `chore(deps): remove unused httpx/starlette, add lock files`, `chore(config): sync env files and add validation`, `chore(db): add missing migrations 001 and 002`
- **Wave 3**: `fix(db): detect orphaned records`, `fix(db): clean orphaned records`, `feat(db): add foreign key constraints and enable PRAGMA`
- **Wave 4**: `test(services): add diagnosis_service unit tests`, `test(services): add status_service unit tests`, `test(services): improve alert/task service coverage`, `test: replace real DB dependency with in-memory/rollback`
- **Wave 5**: `fix(services): eliminate duplicate connection in get_tasks_with_count`, `refactor(pipeline): replace subprocess with direct function calls`, `fix(auth): replace hardcoded qoder with env/config`, `docs: fix version consistency and README_RUNBOOK`
- **Wave FINAL**: `chore(release): v0.5.4 — comprehensive fix release`

---

## Success Criteria

### Verification Commands
```bash
# 所有测试通过且覆盖率达标
pytest --cov-fail-under=60

# Lint 零错误
ruff check .

# 无剩余反向依赖
grep -r "from scripts.config\|import scripts.config\|from scripts import config" \
  --include="*.py" api/ services/ adapters/ | wc -l
# Expected: 0

# 数据库外键已启用
python -c "from services.db_service import get_db; conn = get_db(); print(conn.execute('PRAGMA foreign_keys').fetchone()[0])"
# Expected: 1

# 版本号统一
grep -r "0.5.1\|0.5.2" --include="*.py" --include="*.md" . | grep -v "^Binary" | wc -l
# Expected: 0

# CI 配置存在且有效
cat .github/workflows/ci.yml | grep -E "pytest|ruff|coverage"
# Expected: 3 matches
```

### Final Checklist
- [x] 所有 P0 问题已修复
- [x] 所有 P1 问题已修复或已记录为已知问题
- [x] 385 个现有测试全部通过
- [x] 新增测试 ≥ 50 个
- [x] 覆盖率 ≥ 60%（全局），diagnosis_service ≥ 70%
- [x] ruff lint 零错误（核心目录 api/ services/ adapters/ core/）
- [x] CI/CD 管道通过
- [x] 无循环依赖
- [x] 无 API 契约变更
- [x] 版本号统一为 0.5.3

