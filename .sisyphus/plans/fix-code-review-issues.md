# 代码审查问题修复计划

## TL;DR

> **目标**: 修复代码审查发现的 7 个问题（1个P0、3个P1、3个P2），提升安全性和代码质量
>
> **范围**: 涉及 5 个核心模块（services/db_service.py, api/routers/outbox.py, services/feishu_service.py, services/alert_service.py, services/task_service.py）+ 配置清理
>
> **预计工作量**: 中等（~6-8 小时，3 个并行工作流）
>
> **风险**: 低-中等（有完整测试覆盖，TDD 验证）
>
> **成功标准**: 所有 385+ 现有测试通过 + 新增测试覆盖修复场景

---

## Context

### 原始审查报告
`reports/CODE_REVIEW_2026-05-24.md` 识别了 7 个问题，按优先级分为：

- **P0（立即修复）**: SQL 注入风险、裸 except Exception
- **P1（短期修复）**: 重复代码模式、子进程参数校验、版本号不一致
- **P2（后续优化）**: print 语句、废弃代码清理

### Metis 差距分析要点

**关键发现**:
1. SQL 注入风险已通过 `validate_table_name()` 白名单部分缓解，但需要防御纵深（调用 `validate_sql_identifier()`）
2. 裸 except 并非 `except:`（会捕获 KeyboardInterrupt），而是 `except Exception:`，但仍然过宽
3. alert_service 和 task_service 有 ~85% 结构重复，但存在关键差异（排序字段、参数数量）
4. 废弃模块 `scripts/config.py` 与 README 存在矛盾（README 仍建议从此导入）

**必须设置的防护栏**:
- 不引入 ORM 或查询构建器
- 不改变函数签名或返回类型
- 不添加新依赖
- 子进程修复只加校验，不包装抽象层
- 每个问题一个原子提交
- 先写/更新测试，再实现（TDD）

---

## Work Objectives

### Core Objective
修复代码审查中识别的 7 个问题，不引入回归，保持现有 API 契约和测试覆盖率。

### Concrete Deliverables
- `services/db_service.py` — 加强 SQL 校验（防御纵深）
- `services/alert_service.py` + `services/task_service.py` — 提取共享查询工具函数
- `api/routers/outbox.py` — 收窄异常捕获范围
- `services/feishu_service.py` — 收窄异常 + 子进程参数校验
- `core/config.py` — 移除 print 语句
- `frontend/package.json` — 同步版本号
- `scripts/config.py` + `README.md` — 标记废弃时间线

### Definition of Done
- [ ] 所有 385+ 现有测试通过（pytest）
- [ ] 新增测试覆盖每个修复场景
- [ ] 覆盖率不低于 86%
- [ ] 无函数签名变更
- [ ] 无新依赖引入

### Must Have
- P0 问题必须完全修复
- 每个提交对应一个独立问题
- TDD 流程：先测试（红）→ 实现（绿）→ 重构

### Must NOT Have (Guardrails)
- 不引入 ORM、查询构建器、Repository 模式
- 不改变函数签名或返回类型
- 不添加 SQLAlchemy 或新依赖
- 子进程修复只加校验，不创建抽象类
- 不修改 dispatch_service.py（即使它有类似重复模式）
- 不重命名文件或移动模块

---

## Verification Strategy

### Test Decision
- **基础设施**: 已存在（pytest + pytest-cov）
- **测试策略**: TDD（RED-GREEN-REFACTOR）
- **框架**: pytest

### QA Policy
每个任务必须包含 Agent-Executed QA Scenarios：
- **后端 API**: Bash (curl) 或 pytest 验证
- **服务层**: pytest 单元测试
- **前端**: 版本号文件内容校验（Bash cat + grep）

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Start Immediately - 5 个独立任务可并行):
├── 任务1: 修复 SQL 注入 (db_service.py) [quick]
├── 任务2: 修复裸 except (outbox.py + feishu_service.py) [quick]
├── 任务4: 子进程参数校验 (feishu_service.py) [quick]
├── 任务6: 移除 print 语句 (core/config.py) [quick]
└── 任务5: 同步版本号 (frontend/package.json) [quick]

Wave 2 (After Wave 1 - 最复杂任务):
└── 任务3: 抽象重复代码 (alert_service + task_service) [unspecified-high]

Wave 3 (After Wave 2 - 清理工作):
└── 任务7: 清理废弃代码 (scripts/config.py + README) [quick]

Wave FINAL (After ALL tasks - 4 个并行审查):
├── F1: 计划合规审计 (oracle)
├── F2: 代码质量审查 (unspecified-high)
├── F3: 全量测试执行 (unspecified-high)
└── F4: 范围保真检查 (deep)
```

### Dependency Matrix

| 任务 | 前置依赖 | 阻塞 |
|------|----------|------|
| 1 (SQL) | 无 | 无 |
| 2 (except) | 无 | 无 |
| 3 (重复代码) | 无 | 无 |
| 4 (subprocess) | 无 | 无 |
| 5 (版本号) | 无 | 无 |
| 6 (print) | 无 | 无 |
| 7 (废弃代码) | 3 | 无 |
| F1-F4 | 1-7 | 无 |

> 注：任务3虽然理论可与其他并行，但因涉及重构最复杂，建议单独 wave 以便专注审查。

### Agent Dispatch Summary

- **Wave 1**: 5 个 quick 任务并行
- **Wave 2**: 1 个 unspecified-high 任务（重复代码重构）
- **Wave 3**: 1 个 quick 任务（废弃代码清理）
- **FINAL**: 4 个审查任务并行

---

## TODOs

- [x] 1. 修复 SQL 注入风险（db_service.py）

  **What to do**:
  - 在 `services/db_service.py:69` 的 `get_table_counts()` 中，`validate_table_name()` 白名单校验之后，再调用 `core.config.validate_sql_identifier()` 进行二次校验
  - 在 `services/db_service.py:86` 的 `get_table_info()` 中，确保已有 `validate_table_name()` 之外，也调用 `validate_sql_identifier()` 进行二次校验
  - 新增测试 `test_get_table_info_rejects_invalid_names()`：验证 PRAGMA 查询不会对非法表名执行
  - 新增测试 `test_get_table_counts_defense_in_depth()`：验证同时调用两个校验函数

  **Must NOT do**:
  - 不改变 `get_table_counts()` 或 `get_table_info()` 的函数签名
  - 不引入 ORM 或查询构建器
  - 不改变现有错误行为（仍返回空 dict，只是校验更严格）

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - **理由**: 简单的校验层添加，测试修改也很直接

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1（与任务2、4、5、6并行）
  - **Blocks**: 无
  - **Blocked By**: 无

  **References**:
  - `services/db_service.py:11-25` — `ALLOWED_PUBLIC_TABLES` 白名单
  - `services/db_service.py:65-70` — `get_table_counts()` 现有逻辑
  - `services/db_service.py:75-87` — `get_table_info()` 现有逻辑
  - `core/config.py:112-133` — `validate_sql_identifier()` 函数
  - `tests/test_db_service.py` — 现有测试模式

  **Acceptance Criteria**:
  - [ ] `validate_sql_identifier()` 在 `get_table_counts()` 和 `get_table_info()` 中被调用
  - [ ] 新测试 `test_get_table_info_rejects_invalid_names()` 通过
  - [ ] 新测试 `test_get_table_counts_defense_in_depth()` 通过
  - [ ] 现有所有 db_service 测试通过

  **QA Scenarios**:
  ```
  Scenario: 合法表名正常查询
    Tool: pytest
    Preconditions: SQLite 内存数据库已初始化
    Steps:
      1. 调用 get_table_counts(conn) 验证返回包含 pipeline_runs
      2. 调用 get_table_info(conn, "pipeline_runs") 验证返回列信息
    Expected Result: 正常返回数据，无异常
    Evidence: pytest 输出

  Scenario: 非法表名被拦截
    Tool: pytest
    Preconditions: SQLite 内存数据库已初始化
    Steps:
      1. 调用 get_table_info(conn, "users; DROP TABLE users--") 验证抛出 ValueError
      2. 调用 get_table_counts(conn) 验证不会统计 sqlite_master 等系统表
    Expected Result: ValueError 被抛出，包含 "Invalid SQL identifier"
    Evidence: pytest 输出
  ```

  **Evidence to Capture**:
  - [ ] pytest 输出截图/文本
  - [ ] coverage 报告

  **Commit**: YES
  - Message: `fix(security): add defense-in-depth SQL validation to db_service`
  - Files: `services/db_service.py`, `tests/test_db_service.py`


- [x] 2. 修复裸 except Exception（outbox.py + feishu_service.py）

  **What to do**:
  - `api/routers/outbox.py:81`: 将 `except Exception as e:` 收窄为具体异常类型（至少区分 `DatabaseError`、`ConnectionError`、业务异常）。对于确实需要捕获的异常，添加内联注释说明原因
  - `services/feishu_service.py:179,216,253`: 三处相同的 `except Exception` 模式。分析每处可能抛出的具体异常（lark-oapi 的异常类型、网络异常等），收窄捕获范围。如果三处模式相同且应保持一致，提取为 `_handle_feishu_error(e, context: str)` 辅助函数
  - 新增/更新测试验证关键异常（如数据库错误）能正确传播或处理

  **Must NOT do**:
  - 不改变现有错误响应的数据结构（返回的 dict 格式必须一致）
  - 不创建装饰器、上下文管理器或错误处理抽象类
  - 不将 feishu_service 的三处 except 合并为通用错误处理器（除非只是提取为同文件内的辅助函数）

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - **理由**: 异常收窄是局部修改，不影响架构

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1（与任务1、4、5、6并行）
  - **Blocks**: 无
  - **Blocked By**: 无

  **References**:
  - `api/routers/outbox.py:80-90` — 当前 except 块
  - `services/feishu_service.py:175-185` — 第一处 except
  - `services/feishu_service.py:212-222` — 第二处 except
  - `services/feishu_service.py:249-259` — 第三处 except
  - `tests/test_api_gateway.py` — API 测试
  - `tests/test_feishu_service.py` — Feishu 服务测试

  **Acceptance Criteria**:
  - [ ] outbox.py 的 except 收窄并带有注释说明
  - [ ] feishu_service.py 三处 except 收窄或提取辅助函数
  - [ ] 新增测试 `test_outbox_dispatch_propagates_db_errors()` 通过（验证数据库错误不被吞没）
  - [ ] 新增测试 `test_feishu_service_error_format_consistency()` 通过
  - [ ] 现有所有相关测试通过

  **QA Scenarios**:
  ```
  Scenario: outbox dispatch 正常异常处理
    Tool: pytest
    Preconditions: 内存数据库含 event_outbox 表
    Steps:
      1. 模拟 dispatch_one 抛出 RuntimeError
      2. 验证 API 返回 500 或正确错误响应
    Expected Result: 不返回 200，错误信息被记录
    Evidence: pytest 输出

  Scenario: feishu_service 错误格式一致性
    Tool: pytest
    Preconditions: 无
    Steps:
      1. 调用三个可能抛异常的 feishu 方法
      2. 验证返回的错误 dict 结构一致
    Expected Result: 所有错误响应包含 status、message 字段
    Evidence: pytest 输出
  ```

  **Evidence to Capture**:
  - [ ] pytest 输出

  **Commit**: YES
  - Message: `fix(exceptions): narrow except Exception and add comments`
  - Files: `api/routers/outbox.py`, `services/feishu_service.py`, `tests/test_api_gateway.py`, `tests/test_feishu_service.py`


- [x] 3. 抽象重复代码模式（alert_service.py + task_service.py）

  **What to do**:
  - 创建 `services/_query_utils.py` 新文件，包含共享的 `_build_conditions()` 函数
  - 该函数接受参数：连接、表名、列名映射、参数字典、排序字段
  - 修改 `services/alert_service.py` 和 `services/task_service.py`，调用新的工具函数
  - 保留现有函数签名不变（只修改内部实现）
  - 保留连接管理模式（`should_close`、`get_db()`、`try/finally`）在每个服务文件中，不提取
  - `dispatch_service.py` 的 `get_outbox_with_count()` **不修改**（明确排除在范围外）

  **Must NOT do**:
  - 不创建抽象基类、接口或继承层次结构
  - 不改变 `get_alerts()`、`get_tasks()` 等函数的签名或返回类型
  - 不修改 `dispatch_service.py`
  - 不将 `insert_alert()` 逻辑纳入共享工具（alert_service 有此函数，task_service 无对应函数）

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
  - **理由**: 这是所有任务中最复杂的，需要仔细处理两个服务之间的细微差异（排序字段、参数数量）

  **Parallelization**:
  - **Can Run In Parallel**: NO（建议单独 Wave 以便专注审查）
  - **Parallel Group**: Wave 2
  - **Blocks**: 无
  - **Blocked By**: 无

  **References**:
  - `services/alert_service.py:4-57` — `_build_alert_conditions()` + `get_alerts()` + `get_alerts_with_count()`
  - `services/task_service.py:4-55` — `_build_task_conditions()` + `get_tasks()` + `get_tasks_with_count()`
  - `tests/test_alert_service_extended.py` — alert_service 测试
  - `tests/test_task_service_extended.py` — task_service 测试

  **Acceptance Criteria**:
  - [ ] 新文件 `services/_query_utils.py` 创建并通过测试
  - [ ] `services/alert_service.py` 使用新工具函数，所有现有测试通过
  - [ ] `services/task_service.py` 使用新工具函数，所有现有测试通过
  - [ ] 新测试 `test_build_conditions()` 覆盖所有参数组合
  - [ ] 函数签名无变更（grep 验证）

  **QA Scenarios**:
  ```
  Scenario: 共享工具函数正确处理多参数查询
    Tool: pytest
    Preconditions: 内存数据库含 alert_events 和 action_tasks 表
    Steps:
      1. 调用 get_alerts(conn, status="new", severity="high")
      2. 调用 get_tasks(conn, status="todo", priority="medium")
      3. 验证 WHERE 子句构建正确
    Expected Result: 两个查询都返回正确过滤结果
    Evidence: pytest 输出

  Scenario: 排序字段差异保持
    Tool: pytest
    Preconditions: 内存数据库含测试数据
    Steps:
      1. 调用 get_alerts() 验证按 event_date DESC 排序
      2. 调用 get_tasks() 验证按 created_at DESC 排序
    Expected Result: 排序行为与修改前一致
    Evidence: pytest 输出
  ```

  **Evidence to Capture**:
  - [ ] pytest 全量测试结果
  - [ ] 函数签名 diff（git diff）

  **Commit**: YES
  - Message: `refactor(services): extract shared _build_conditions utility`
  - Files: `services/_query_utils.py`, `services/alert_service.py`, `services/task_service.py`, `tests/test_query_utils.py`


- [x] 4. 子进程参数校验（feishu_service.py）

  **What to do**:
  - 在 `services/feishu_service.py` 的 `_run_script()` 方法中，`subprocess.run(cmd, ...)` 调用之前添加参数校验
  - 校验内容：`cmd` 必须是非空列表，`cmd[0]` 必须是已知可执行路径（位于 `SCRIPTS_DIR` 下），`cmd[1:]` 中的参数不含 shell 元字符
  - 如果校验失败，抛出 `ValueError` 而非执行 subprocess
  - 新增测试覆盖非法 cmd 场景

  **Must NOT do**:
  - 不创建 `SubprocessRunner` 或 `CommandValidator` 抽象类
  - 不修改 `subprocess.run()` 的调用方式（仍保持 `capture_output=True` 等参数）
  - 不修改 timeout 逻辑

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - **理由**: 局部校验逻辑添加

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1（与任务1、2、5、6并行）
  - **Blocks**: 无
  - **Blocked By**: 无

  **References**:
  - `services/feishu_service.py:77-100` — `_run_script()` 方法
  - `tests/test_feishu_service.py` — 现有测试

  **Acceptance Criteria**:
  - [ ] `_run_script()` 在调用 subprocess 前验证 `cmd` 参数
  - [ ] 新测试 `test_subprocess_cmd_rejects_empty_list()` 通过
  - [ ] 新测试 `test_subprocess_cmd_rejects_invalid_script()` 通过
  - [ ] 现有 feishu 测试全部通过

  **QA Scenarios**:
  ```
  Scenario: 合法脚本正常执行
    Tool: pytest
    Steps:
      1. 调用 FeishuService._run_script("feishu_client.py", ["--help"])
    Expected Result: 正常返回输出
    Evidence: pytest 输出

  Scenario: 非法脚本路径被拒绝
    Tool: pytest
    Steps:
      1. 调用 _run_script("../../../etc/passwd", [])
    Expected Result: 抛出 ValueError，不执行 subprocess
    Evidence: pytest 输出
  ```

  **Evidence to Capture**:
  - [ ] pytest 输出

  **Commit**: YES
  - Message: `fix(security): validate subprocess cmd parameters in feishu_service`
  - Files: `services/feishu_service.py`, `tests/test_feishu_service.py`


- [x] 5. 同步前端版本号（frontend/package.json）

  **What to do**:
  - 将 `frontend/package.json` 中的 `"version": "0.5.1"` 改为 `"0.5.3"`
  - 在 frontend 目录下运行 `npm install`（或 `bun install`）验证无依赖冲突
  - 检查是否有其他版本号引用（如 `frontend/src/` 中的常量、Dockerfile 等）

  **Must NOT do**:
  - 不修改任何业务逻辑代码
  - 不修改 API 契约
  - 不引入版本管理基础设施（如统一版本源）

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - **理由**: 单行修改 + 验证

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1（与任务1、2、4、6并行）
  - **Blocks**: 无
  - **Blocked By**: 无

  **References**:
  - `frontend/package.json:4` — 版本号字段

  **Acceptance Criteria**:
  - [ ] `frontend/package.json` 中 version 为 "0.5.3"
  - [ ] `npm install` 或 `bun install` 在 frontend/ 下成功执行
  - [ ] 无其他 "0.5.1" 引用残留（grep 检查）

  **QA Scenarios**:
  ```
  Scenario: 版本号已同步
    Tool: Bash
    Steps:
      1. grep '"version"' frontend/package.json
    Expected Result: 输出包含 "0.5.3"
    Evidence: 终端输出

  Scenario: 依赖安装无冲突
    Tool: Bash
    Steps:
      1. cd frontend && npm install
    Expected Result: 命令成功退出，无错误
    Evidence: 终端输出
  ```

  **Evidence to Capture**:
  - [ ] package.json 内容截图
  - [ ] npm install 输出

  **Commit**: YES
  - Message: `chore(version): sync frontend version to 0.5.3`
  - Files: `frontend/package.json`


- [x] 6. 移除 print 语句（core/config.py）

  **What to do**:
  - 分析 `core/config.py:183-193` 的 `if __name__ == '__main__':` 块
  - 决定策略：A) 保留但改为 logging.info()；B) 移除整个块；C) 保留但添加注释说明仅用于调试
  - 建议策略 A（改为日志）以保持开发者便利性
  - 如果需要，添加简单的日志配置（使用已存在的 api/logging_config.py 模式或标准 logging）

  **Must NOT do**:
  - 不引入新的日志依赖（使用 Python 标准库 logging）
  - 不改变 `ensure_dirs_exist()` 的行为

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - **理由**: 简单的 print → logging 替换

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1（与任务1、2、4、5并行）
  - **Blocks**: 无
  - **Blocked By**: 无

  **References**:
  - `core/config.py:183-193` — `if __name__ == '__main__'` 块
  - `api/logging_config.py` — 日志配置参考

  **Acceptance Criteria**:
  - [ ] `core/config.py` 中无裸 `print()` 语句（或仅在明确注释的调试块中）
  - [ ] 直接运行 `python core/config.py` 仍能输出路径信息（通过 logging）
  - [ ] 现有测试通过

  **QA Scenarios**:
  ```
  Scenario: 直接运行 core/config.py 输出路径
    Tool: Bash
    Steps:
      1. python core/config.py
    Expected Result: 输出路径信息（通过 logging，不是 print）
    Evidence: 终端输出

  Scenario: 导入时无输出
    Tool: Bash (python -c)
    Steps:
      1. python -c "import core.config"
    Expected Result: 无输出（logging 默认不输出到控制台）
    Evidence: 终端输出
  ```

  **Evidence to Capture**:
  - [ ] 终端输出

  **Commit**: YES
  - Message: `fix(config): replace print with logging in core/config.py`
  - Files: `core/config.py`


- [x] 7. 清理废弃代码（scripts/config.py + README.md）

  **What to do**:
  - 搜索所有 `scripts.config` 的导入引用：`grep -r "from scripts.config\|from scripts import config\|scripts.config" --include="*.py" .`
  - 如果无活动引用：在 `scripts/config.py` 顶部添加 `DeprecationWarning`，指定移除版本（如 v0.6.0）
  - 如果有活动引用：先将这些文件迁移到 `from core.config import ...`，然后再添加废弃警告
  - 更新 `README.md`：删除 "新代码应通过 from scripts import config 引用路径" 的说明，改为 "从 core.config 导入"

  **Must NOT do**:
  - 不删除 `scripts/config.py`（除非确认无任何引用，包括外部用户脚本）
  - 不修改 `scripts/__init__.py` 的导出（除非必要）

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - **理由**: 搜索 + 文档更新 + 废弃标记

  **Parallelization**:
  - **Can Run In Parallel**: NO（建议 Wave 3 最后执行，以便基于前面修改的上下文）
  - **Parallel Group**: Wave 3
  - **Blocks**: 无
  - **Blocked By**: 无（但建议在其他修改完成后执行，避免冲突）

  **References**:
  - `scripts/config.py` — 废弃模块
  - `README.md` — 需要更新的文档
  - `core/config.py` — 正确的新导入源

  **Acceptance Criteria**:
  - [ ] `grep` 显示无活动引用（或所有引用已迁移）
  - [ ] `scripts/config.py` 包含 `DeprecationWarning`
  - [ ] `README.md` 已更新，不再推荐从 scripts.config 导入
  - [ ] 现有测试通过

  **QA Scenarios**:
  ```
  Scenario: 废弃警告触发
    Tool: Bash (python -c)
    Steps:
      1. python -c "import scripts.config"
    Expected Result: 显示 DeprecationWarning
    Evidence: 终端输出

  Scenario: README 已更新
    Tool: Bash (grep)
    Steps:
      1. grep "scripts import config" README.md
    Expected Result: 无输出（旧说明已删除）
    Evidence: 终端输出
  ```

  **Evidence to Capture**:
  - [ ] grep 搜索结果
  - [ ] DeprecationWarning 触发截图

  **Commit**: YES
  - Message: `docs(deprecation): mark scripts/config.py deprecated and update README`
  - Files: `scripts/config.py`, `README.md`

---

## Final Verification Wave

> 4 个审查任务并行执行。全部通过后才能标记工作完成。

- [ ] F1. **计划合规审计** — `oracle`
  逐条检查计划中的 "Must Have" 和 "Must NOT Have"：
  - 读取每个修改后的文件，确认实现存在
  - 搜索代码库，确认没有禁止的模式（ORM、抽象基类、签名变更、新依赖）
  - 检查 .sisyphus/evidence/ 中的证据文件存在
  - 验证提交数量 = 7，每个提交对应一个问题
  **输出**: `Must Have [N/N] | Must NOT Have [N/N] | VERDICT`

- [ ] F2. **代码质量审查** — `unspecified-high`
  运行全量检查：
  ```bash
  # 语法检查
  python -m py_compile api/*.py api/routers/*.py services/*.py adapters/*.py core/*.py
  
  # 测试
  pytest -q
  
  # 覆盖率
  pytest --cov=. --cov-report=term-missing
  
  # 代码风格（如果已安装 ruff）
  ruff check api/ services/ adapters/ core/
  ```
  **审查项**: 
  - `as any` / `@ts-ignore` / 空 catch 块
  - 残留的 `print()` 语句
  - 未使用的导入
  - AI slop 模式（过度注释、过度抽象）
  **输出**: `Build [PASS/FAIL] | Tests [N pass/N fail] | Coverage [N%] | VERDICT`

- [ ] F3. **全量 QA 执行** — `unspecified-high`
  从干净状态开始，执行每个任务的 QA Scenarios：
  - 验证 happy path 和 failure path
  - 验证边缘情况（空输入、极大 limit 值等）
  - 捕获证据到 `.sisyphus/evidence/`
  **输出**: `Scenarios [N/N pass] | VERDICT`

- [ ] F4. **范围保真检查** — `deep`
  对每个任务：读取 "What to do"，比对实际 git diff：
  - 确认规格中的所有内容都已实现（无遗漏）
  - 确认未实现规格之外的内容（无蔓延）
  - 检查 "Must NOT do" 合规性
  - 检测跨任务污染（任务 N 修改了任务 M 的文件）
  **输出**: `Tasks [N/N compliant] | Creep [CLEAN/N issues] | VERDICT`

> **注意**: F1-F4 全部 APPROVE 后，向用户呈现结果并获取显式确认。如果任何一项 REJECT，修复问题后重新运行验证。

---

## Commit Strategy

| 提交 | 内容 | 文件 |
|------|------|------|
| Commit 1 | fix(security): 为 db_service SQL 查询添加防御纵深校验 | services/db_service.py, tests/test_db_service.py |
| Commit 2 | fix(exceptions): 收窄异常捕获范围并添加说明 | api/routers/outbox.py, services/feishu_service.py, tests/* |
| Commit 3 | refactor(services): 提取 alert/task 共享查询工具函数 | services/_query_utils.py, services/alert_service.py, services/task_service.py, tests/test_query_utils.py |
| Commit 4 | fix(security): 为 subprocess 调用添加参数校验 | services/feishu_service.py, tests/test_feishu_service.py |
| Commit 5 | fix(config): 移除 core/config.py 中的 print 语句 | core/config.py |
| Commit 6 | chore(version): 同步前端版本号至 0.5.3 | frontend/package.json |
| Commit 7 | docs(deprecation): 标记 scripts/config.py 废弃时间线并更新 README | scripts/config.py, README.md |

## Success Criteria

### Verification Commands
```bash
# 1. 全量测试通过
pytest -q
# Expected: 385+ passed, 0 failed

# 2. 覆盖率检查
pytest --cov=. --cov-report=term-missing
# Expected: coverage >= 86%

# 3. 前端版本号校验
grep '"version"' frontend/package.json
# Expected: "version": "0.5.3"

# 4. 无 print 语句残留
grep -n "print(" core/config.py
# Expected: 无输出（或仅在 __main__ 块内）

# 5. 废弃模块标记确认
grep -n "DeprecationWarning\|deprecated\|will be removed" scripts/config.py
# Expected: 显示废弃警告
```

### Final Checklist
- [ ] 所有 P0 问题已修复
- [ ] 所有 P1 问题已修复
- [ ] 所有 P2 问题已修复
- [ ] 385+ 测试全部通过
- [ ] 覆盖率 >= 86%
- [ ] 无函数签名变更
- [ ] 无新依赖引入
- [ ] 7 个原子提交，每个对应一个问题
- [ ] README 已更新（废弃模块说明）
