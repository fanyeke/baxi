# Phase 04: Bug Fixes & Stability - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-03
**Phase:** 04-bug-fixes-stability
**Areas discussed:** BUG-01 (confirmed fixed), BUG-02, BUG-03, BUG-04, BUG-05

---

## BUG-01: action.go JSON Decode (confirmed fixed in Phase 2)

| Option | Description | Selected |
|--------|-------------|----------|
| 标记为已完成 | 当前代码已返回 400 错误信息，BUG-01 要求已满足 | ✓ |
| 让我验证 | 自己检查确认修复是否完整 | |

**User's choice:** 标记为已完成
**Notes:** 已在 Phase 2 的 commit aa363e7 修复。当前 action.go:69-72 正确返回 400。

---

## BUG-02: Alert Engine JSON Marshal Error Handling

| Option | Description | Selected |
|--------|-------------|----------|
| 记录日志 + 继续（推荐） | zap.Logger.Error() + 空证据继续 | ✓ |
| 返回错误给调用方 | 中断整次告警评估，调用方收到 500 | |
| 记录日志 + 继续（slog） | 用现有 slog 记录警告，改动最小 | |

**Follow-up 1 — 日志内容:**

| Option | Description | Selected |
|--------|-------------|----------|
| 证据键名清单 + 错误信息（推荐） | 记录 keys + err，不记录完整内容 | ✓ |
| 完整证据内容 + 错误 | 更详细但可能含大量无用数据 | |

**Follow-up 2 — 空证据行为:**

| Option | Description | Selected |
|--------|-------------|----------|
| 创建告警但证据为空（推荐） | 仍创建 AlertResult，告警不会完全丢失 | ✓ |
| 跳过此次告警 | 更保守，可能错过重要异常 | |

**User's choice:** 记录日志 + 继续，使用 zap.Logger.Error 记录证据键名清单，创建空证据告警
**Notes:** 告警系统的首要职责是发出告警，证据是辅助信息。Marshal 失败不应导致告警丢失。

---

## BUG-03: Feishu page_token Assertion Failure

| Option | Description | Selected |
|--------|-------------|----------|
| 记录错误 + 中断分页（推荐） | zap.Error 记录后 break 退出循环 | ✓ |
| 记录错误 + 返回错误给调用方 | 中断所有处理并向上传播错误 | |
| 记录警告 + 用空 token 继续 | 可能出现无限循环，风险较大 | |

**Follow-up — 错误传播:**

| Option | Description | Selected |
|--------|-------------|----------|
| 不返回错误（推荐） | 返回已获取数据和 nil error | ✓ |
| 返回部分数据 + 错误 | 返回数据和一个分页警告错误 | |

**User's choice:** 记录错误 + 中断分页，不返回错误
**Notes:** 分页中断是部分故障不是完全故障，已获取的数据仍然有效。

---

## BUG-04: Migration Sequence Gaps

| Option | Description | Selected |
|--------|-------------|----------|
| 添加空占位迁移（推荐） | 创建 015 和 025 空文件，含注释 | ✓ |
| 文档说明后保持现状 | 仅记录缺口原因，不添加新文件 | |
| 重编号现有迁移文件 | 改动大，影响已部署数据库的 goose 表 | |

**User's choice:** 添加空占位迁移
**Notes:** Git 历史中从未存在 015/025——是创建时跳过编号，非被删除。

---

## BUG-05: Ontology SQL Injection Hardening

| Option | Description | Selected |
|--------|-------------|----------|
| 允许列表 + pgx.Identifier（推荐） | 验证 objectType + Sanitize schema.table | ✓ |
| 全面迁移到 V2 编译器 | 弃用所有 V1 回退路径 | |
| 两者都做 | V1 加固 + 迁移到 V2 | |

**Follow-up — 弃用标记:**

| Option | Description | Selected |
|--------|-------------|----------|
| 添加 GODEPRECATED 注释（推荐） | 加固代码旁加注释说明弃用 | ✓ |
| 不加标记 | 只修复不添加文档 | |

**User's choice:** 允许列表 + pgx.Identifier，添加 GODEPRECATED 注释
**Notes:** V2 compiler 已在 compiler.go 中使用 pgx.Identifier 消毒，V1 路径应遵循相同标准。

---

## the agent's Discretion

- 告警引擎中 zap logger 的具体注入方式（Engine 结构体字段 vs 参数传递）
- 占位迁移文件中注释的具体措辞
- 允许列表检查的具体实现细节

## Deferred Ideas

None — discussion stayed within phase scope.
