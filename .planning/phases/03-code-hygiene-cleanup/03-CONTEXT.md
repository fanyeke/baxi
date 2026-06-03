# Phase 03: Code Hygiene & Cleanup - Context

**Gathered:** 2026-06-03
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 3 清理 Python/SQLite 迁移遗留物、死代码和弃用的 repository shim，实现代码库的干净可构建状态。不涉及新增功能、BUG 修复或安全加固（分属 Phase 4/5）。

**在范围内（从 REQUIREMENTS.md）：**
- HYG-01: Pipeline 预览显示 Go 命令而非 Python 脚本
- HYG-02: Makefile 不再引用 Python 脚本进行验证
- HYG-03: 弃用的 repository shim 文件全部删除（6 个文件 + interfaces.go）
- HYG-04: 所有调用方从弃用 repository 迁移到子包 + PoolProvider
- HYG-05: 死 CLI 子命令 `llm.go` 移除（不接入 main.go）
- HYG-06: 占位文件 `internal/worker/worker.go` 移除
- HYG-07: 迁移基线目录 `migration_baseline/` 删除

**不在范围内：**
- BUG 修复（Phase 4）
- 安全加固（Phase 5）
- CLI 重构到 `internal/cli/`（后续阶段）
- E2E 测试迁移到 `internal/`（后续阶段）

</domain>

<decisions>
## Implementation Decisions

### Repository Shim 迁移策略
- **D-01:** 批量替换策略——用静态分析一次性找出所有调用旧 API 的地方，全部迁移到子包 PoolProvider API，然后一次性删除 6 个 shim 文件。
- **Rationale:** 代码库规模不大，全量替换风险可控。逐个迁移步骤繁琐且收益递减。
- **D-02:** 迁移后运行 `go build ./...` + `go test ./...` 验证无遗漏。不额外写迁移专用测试。
- **D-03:** `internal/repository/interfaces.go`（定义旧接口的地方）随 shim 文件一并删除。旧接口在子包中已有对应定义。
- **D-04:** 所有 6 个 shim 文件一次性删除（不分组）。
- **D-05:** 清理 repository 目录下其他不符合子包模式的残余旧文件（除 6 个 shim 外）。

### 死 CLI 代码 (llm.go)
- **D-06:** `cmd/baxi-cli/llm.go` 直接删除。不接入 main.go。
- **Rationale:** llm status/metrics 功能缺失不大，维护成本高。删除最简洁。
- **D-07:** `cmd/baxi-cli/client.go` 暂时保留（还被 decision.go 等其他子命令使用）。不在此阶段搬移到 internal/。
- **D-08:** 不扫描清理 `cmd/baxi-cli/` 下其他文件的死代码——只删 `llm.go`。

### Python 遗留物处理
- **D-09:** Makefile 中 `api-compare` 目标（调用 `python3 scripts/migration/compare_api_baseline.py`）删除。相关 Python 验证能力已由 Go E2E 测试覆盖。
- **D-10:** 全量扫描 Makefile，**删除所有** Python 脚本引用。
- **D-11:** Pipeline 预览（`internal/service/pipeline_service.go`）返回的命令更新为 `go run ./cmd/baxi-cli pipeline run`，替换原有的 `python3 scripts/run_daily_pipeline.py`。
- **D-12:** `scripts/migration/*.py` 删除。

### 迁移基线目录处理
- **D-13:** `migration_baseline/` 目录直接删除（不归档）。Git 历史中已有记录。
- **D-14:** 同步更新 README.md、AGENTS.md 等文档中过时的 SQLite/Python 引用。

### the agent's Discretion
- 具体找出所有调用方的方式（grep 或 AST）由 planer/executor 决定。
- pipeline preview 字符串替换的具体格式由 planer/executor 决定。
- Makefile 清理后 `pipeline-run` 目标指向的具体 Go 命令由 planer/executor 决定。

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements & Roadmap
- `.planning/REQUIREMENTS.md` §HYG-01~HYG-07 — 本阶段 7 个需求
- `.planning/ROADMAP.md` §Phase 3 — Phase 3 目标和成功标准

### 弃用的 Repository Shim（将被删除）
- `internal/repository/governance_repository.go` — 弃用 shim
- `internal/repository/decision_repository.go` — 弃用 shim
- `internal/repository/ontology_repository.go` — 弃用 shim
- `internal/repository/outbox_repository.go` — 弃用 shim
- `internal/repository/log_repository.go` — 弃用 shim
- `internal/repository/context_repository.go` — 弃用 shim
- `internal/repository/interfaces.go` — 旧接口定义（随 shim 删除）

### Subpackage Repositories（迁移目标）
- `internal/repository/decision/` — 决策相关数据库操作（PoolProvider 模式）
- `internal/repository/governance/` — 治理相关数据库操作
- `internal/repository/ontology/` — 本体相关数据库操作
- `internal/repository/outbox/` — Outbox 相关数据库操作
- `internal/repository/log_repository.go` 的子包对应
- `internal/repository/common/pool.go` — PoolProvider 定义

### 死代码/占位文件（将被删除）
- `cmd/baxi-cli/llm.go` — 不可达的死 CLI 子命令
- `internal/worker/worker.go` — 占位文件（实际逻辑在 dispatch_worker.go）

### Pipeline 预览
- `internal/service/pipeline_service.go` — 需要更新 Python→Go 命令

### 迁移基线（将被删除）
- `migration_baseline/` — 包含 sqlite_schema.sql 和 Python 脚本
- `scripts/migration/compare_api_baseline.py` — Makefile api-compare 调用的 Python 脚本

### 代码库诊断
- `.planning/codebase/CONCERNS.md` §Python/SQLite Migration Remnants — 详细的问题描述
- `.planning/codebase/CONCERNS.md` §Deprecated Repository Compatibility Layer — 详细的问题描述
- `.planning/codebase/CONCERNS.md` §Dead/Unreachable Code — 详细的问题描述

### 文档（需更新）
- `README.md` — 可能含过时 SQLite/Python 引用
- `AGENTS.md` — 可能含过时 SQLite/Python 引用
- `docs/` 目录下文档 — 需检查并更新

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- Subpackage repositories（`internal/repository/decision/` 等）已实现 PoolProvider 模式，是迁移的目标 API
- `internal/repository/common/pool.go` — PoolProvider 定义，可被所有 subpackage 复用
- `cmd/baxi-cli/decision.go`、`pipeline.go`、`governance.go` — 可参考其模式判断 client.go 的依赖关系

### Established Patterns
- PoolProvider 注入模式：repository subpackage 通过 `common.NewPoolProvider(pool)` 创建 provider
- Thin main.go 模式：baxi-api 和 baxi-worker 遵循，baxi-cli 是反例
- Subpackage 组织：`internal/repository/{domain}/` 结构，每个子包有独立的 repository.go

### Integration Points
- 弃用 shim 文件的调用方跨越多个包：`internal/api/handler/`、`internal/service/`、`internal/decision/` 等——需要 grep 找到所有引用
- `internal/worker/worker.go` 的移除不影响 `internal/worker/dispatch_worker.go`（独立文件）
- `cmd/baxi-cli/main.go` 删除 `llm` 分发——需要确保 `main.go` 中没有注册 `llm` 命令
- Pipeline preview 字符串引用在 `internal/service/pipeline_service.go` 第 37、77 行附近

</code_context>

<specifics>
## Specific Ideas

- 批量替换前先用 `grep -rn "baxi/internal/repository\.\|baxi/internal/repository/governance_repository\|..."` 枚举所有调用方，确认覆盖完整
- Pipeline preview 字符串应精确显示：`go run ./cmd/baxi-cli pipeline run`
- 删除 migration_baseline/ 前确认没有其他 Git 引用指向其中文件

</specifics>

<deferred>
## Deferred Ideas

- CLI 逻辑重构到 `internal/cli/`（baxi-cli 919 行在 package main 的反模式）——后续阶段
- BatchDispatch 增强（过滤条件、自动重试）——继续延期
- E2E 测试从 `test/` 迁移到 `internal/`——后续阶段
- No golangci-lint config（CONCERNS 中提到）——需独立阶段或工具配置

</deferred>

---

*Phase: 03-code-hygiene-cleanup*
*Context gathered: 2026-06-03*
