# Phase 03: Code Hygiene & Cleanup - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-03
**Phase:** 03-code-hygiene-cleanup
**Areas discussed:** Repository 迁移策略, 死 CLI 代码 (llm.go), Python 遗留物处理, 迁移基线目录处理

---

## Repository 迁移策略

| Option | Description | Selected |
|--------|-------------|----------|
| 逐个迁移 | 每迁移一个调用方到子包 API 后立即测试。风险低但耗时长。 | |
| 批量替换 | 用静态分析一次找出所有调用方，统一替换，然后一次删除所有 shim。 | ✓ |
| 编译+测试验证 | 迁移后运行 go build ./... + go test ./... 验证 | ✓ |
| 额外 grep 残留检查 | 除编译和测试外，额外 grep 确认无旧接口引用残留 | |
| 一并删除 interfaces.go | 删除 6 个 shim 时一并删除 interfaces.go（旧接口定义） | ✓ |
| 暂时保留 interfaces.go | 先保留等后续处理 | |
| 一次性全删 | 所有 6 个 shim + interfaces.go 一次删除 | ✓ |
| 分批删除 | 按文件逐批删除 | |
| 只删 shim | 严格按 HYG-03/04 执行 | |
| 一并清理残余 | 清理 repository 目录下其他不符合子包模式的旧文件 | ✓ |
| 不额外写测试 | 迁移验证只做编译+测试 | ✓ |
| 视情况补测试 | 如新迁移路径未被测试覆盖则补写 | |

**User's choice:** 批量替换 → 编译+测试验证 → 一并删除 interfaces.go → 一次性全删 → 一并清理残余旧文件 → 不额外写测试

---

## 死 CLI 代码 (llm.go)

| Option | Description | Selected |
|--------|-------------|----------|
| 直接删除 | 删除 llm.go，简洁零维护 | ✓ |
| 接入 main.go | 在 main.go 中添加 llm 命令分发 | |
| 暂时保留 client.go | client.go 还被其他子命令使用 | ✓ |
| 一并搬到 internal | client.go 搬到 internal/api/client/ | |
| 只删 llm.go | 不做额外 CLI 扫描 | ✓ |
| 一并扫描清理 | 扫描 cmd/baxi-cli/ 下所有死代码 | |

**User's choice:** 直接删除 llm.go → 保留 client.go → 只删 llm.go

---

## Python 遗留物处理

| Option | Description | Selected |
|--------|-------------|----------|
| 删除 Python 目标和脚本 | 删除 api-compare Makefile 目标和 scripts/migration/*.py | ✓ |
| 用 Go 重写 | 用 Go 重写 compare_api_baseline.py 功能 | |
| 全量扫描删干净 | 扫描整个 Makefile，删除所有 Python 引用 | ✓ |
| 只删明确引用的 | 只删 api-compare 和 pipeline-run 中的 Python 引用 | |
| go run 命令 | Pipeline preview 显示 'go run ./cmd/baxi-cli pipeline run' | ✓ |
| make pipeline | Pipeline preview 显示 'make pipeline' | |
| 你决定 | 让 planer 决定 preview 格式 | |

**User's choice:** 删除 Python 目标和脚本 → 全量扫描删干净 → go run 命令

---

## 迁移基线目录处理

| Option | Description | Selected |
|--------|-------------|----------|
| 直接删除 | 删除 migration_baseline/ 目录 | ✓ |
| 归档到外部 | 备份到外部位置然后从仓库删除 | |
| 你决定 | 让 planer 决定 | |
| 一并更新文档 | 同步更新 README.md/AGENTS.md 中过时引用 | ✓ |
| 只删目录 | 只删目录不动文档 | |

**User's choice:** 直接删除 → 一并更新文档

---

## the agent's Discretion

- 具体找出所有调用方的方式（grep 或 AST 分析）
- Pipeline preview 字符串替换的具体格式
- Makefile 清理后 pipeline-run 目标指向的具体命令

## Deferred Ideas

- CLI 逻辑重构到 internal/cli/（后续阶段）
- BatchDispatch 增强（继续延期）
- E2E 测试从 test/ 迁移到 internal/（后续阶段）
- golangci-lint 配置（需独立阶段）
