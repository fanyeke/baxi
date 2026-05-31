# review: 审核与审批

**Branch:** main

## OVERVIEW
提案审批流程（approve/reject/cancel）+ 沙盘（sandbox）模拟。5 个生产文件，处理 action_proposal 的状态转换。

公开的 MCP 工具：`approve_proposal`, `reject_proposal`, `cancel_proposal`, `get_proposal_by_id`, `list_review_records`。参见 `internal/mcp/tools_review.go`。

## WHERE TO LOOK

| 任务 | 文件 | 说明 |
|------|------|------|
| 审核服务 | `service.go` | 审批/驳回/取消，事务安全的状态转换（7 步事务） |
| 审核领域 | `domain.go` | Verdict 类型（approve/reject/cancel）、ReviewRecord、ReviewRequest |
| 数据访问 | `repository.go` | ReviewRepository（无状态），ActionProposalRow 本地类型避免循环依赖 |
| 沙盘服务 | `sandbox.go` | 提案沙盘的 CRUD + 对比（差异计算） |

## KEY PATTERNS

- **事务安全的状态转换**: `transitionProposal` 使用 `SELECT ... FOR UPDATE` 锁定行，7 步原子操作
- **审计日志**: 每次状态转换自动写入 `audit.audit_log` + 可选的 lineage 事件
- **沙盘对比**: `compareData` 递归比较两个沙盘的 map 差异
- **无状态 Repository**: `ReviewRepository` 无字段，所有方法接收 `pool` 参数
- **可选 LineageRecorder**: 通过 `WithLineageRecorder` 依赖注入，默认不记录
- **编译时接口检查**: domain.go 中的 Validate() 方法验证必填字段

## ANTI-PATTERNS

- `ActionProposalRow` 在 `repository.go` 本地定义（为避免循环依赖）— 与 `internal/repository` 的 action proposal 类型重复
- `ListReviewRecords` 在 service.go 中直接调用 `s.repo.ListReviewRecords(ctx, s.pool, ...)` — pool 重复传入
- 沙盘没有独立验证/审批流程 — 只是数据容器
