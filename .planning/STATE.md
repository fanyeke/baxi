---
gsd_state_version: 1.0
milestone: v1.1
milestone_name: MCP 信息收束
status: planning
last_updated: "2026-06-06T06:00:00.000Z"
last_activity: 2026-06-06
progress:
  total_phases: 5
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
  percent: 0
---

# Project State: Baxi

## Project Reference

See: .planning/PROJECT.md (updated 2026-06-03)

**Core value:** Agent 通过 MCP 接触项目时，无法通过工具名称、服务器自述、返回数据拼凑出项目架构，也无法获取不应暴露的业务数据
**Current focus:** Phase 07 — Foundation: 服务器身份泛化 & 工具名抽象

## Current Position

Phase: Phase 7 — Foundation: 服务器身份泛化 & 工具名抽象
Plan: TBD
Status: Roadmapped, awaiting approval
Last activity: 2026-06-06 — Milestone v1.1 roadmap created

## Performance Metrics

**Velocity:**

- Total plans completed: 0
- Average duration: —
- Total execution time: —

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 7 - Foundation | 0 | TBD | - |
| 8 - Output: Schema/Status | 0 | TBD | - |
| 9 - Output: Object Data | 0 | TBD | - |
| 10 - Input: Search/Pipeline | 0 | TBD | - |
| 11 - Compatibility & Error | 0 | TBD | - |

**Recent Trend:**

- Last 5 plans: —
- Trend: —

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- v1.1: MCP 信息收束 — 通用身份 + 严格裁剪，Agent 不应能从 MCP 推断项目架构
- v1.1: MCP 工具抽象 — 用业务能力命名代替领域命名，抹掉 internal 包映射
- v1.1: Dual registration（旧名称兼容别名）确保 Pi Agent 集成不中断
- v1.1: Handler 级过滤（非 middleware）— mcp-go v0.41.1 无后处理管道，过滤在 NewToolResultJSON 之前完成
- v1.1: LLMReadable 优先（非 Sensitivity）— MVP 只用 LLMReadable 标记，Sensitivity 过滤延后

### Pending Todos

None yet.

### Blockers/Concerns

- E2E tests in `test/` import `baxi/internal/*` by full module path — fragile to refactoring
- No golangci-lint config — varying style across packages
- Build constraint `//go:build integration` means `go test ./...` skips E2E tests silently
- **Tool rename breaks 3 test files + 1 Pi extension** — must update simultaneously in one atomic commit
- **Error messages leak SQL/schema details** — need `sanitizeError()` helper and audit ~15-20 call sites
- **LLMReadable flag coverage** — not all properties may be consistently flagged; need audit during Phase 9

## Deferred Items

Items acknowledged and carried forward from previous milestone close:

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| *(none)* | | | |

## Session Continuity

Last session: 2026-06-06
Stopped at: Milestone v1.1 roadmap created — Phases 7-11 defined
Resume file: /home/zzz/project/baxi/.planning/ROADMAP.md

## Phase Dependencies

```
Phase 7 (Foundation: Identity + Names) — no deps
  ↓
Phase 8 (Output: Schema/Status) — depends on Phase 7
  ↓
Phase 9 (Output: Object Data) — depends on Phase 8 (uses output_filter.go)
Phase 10 (Input: Search/Pipeline) — depends on Phase 8 (uses output_filter.go)
  ↓
Phase 11 (Compatibility & Error Sanitization) — depends on Phases 7-10
```

Note: Phases 8, 9, 10 can be parallelized after Phase 7 completes.
