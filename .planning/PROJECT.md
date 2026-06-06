# Baxi — Governance + Analytics Platform

## Current Milestone: v1.1 MCP 信息收束

**Goal:** 对 MCP Server 进行信息收束改造，Agent 接触 MCP 时无法拼凑项目架构，也无法获取不该拿的业务数据

**Target features:**
- 泛化服务器身份（改名 + instructions 模糊化）
- 工具名按业务能力重新分组命名（抹掉 internal 包映射）
- 裁剪 describe_ontology / get_system_status 输出
- get_object / get_linked_objects 字段级过滤
- search_objects / run_pipeline 输入加固

## What This Is

Baxi is a multi-language data governance and analytics platform that runs data pipelines, enforces governance rules, and makes LLM-assisted decisions with full audit trails. It connects to external channels (Feishu/Lark, GitHub) for alerting and action execution, and exposes an MCP server for Pi Agent integration.

## Core Value

A complete, demonstrable closed-loop system: data flows through pipelines, governance rules catch issues, decisions are made with context, actions are executed, and results feed back — all observable through API, CLI, and web frontend.

## Requirements

### Validated

- ✓ **Pipeline orchestration** — 7-step ETL pipeline with per-step transactions (`internal/pipeline/`)
- ✓ **Governance engine** — Classification, lineage, access policy, redaction via 28 YAML configs (`internal/governance/`)
- ✓ **Decision engine** — LLM-driven decisions with validation, repair retry, rule-based fallback (`internal/decision/`)
- ✓ **MCP server** — 31 tools over stdio for Pi Agent integration (`internal/mcp/`)
- ✓ **HTTP API** — chi router with 14 handlers, JWT auth middleware (`internal/api/`)
- ✓ **React frontend** — 11 pages with TanStack Query, Radix UI, Tailwind CSS (`frontend/`)
- ✓ **PostgreSQL persistence** — pgx/v5 with 29 goose migrations (`migrations/`)
- ✓ **Channel adapters** — Feishu webhook + OpenAPI, GitHub issue creation (`internal/adapter/`)
- ✓ **Alert engine** — Dimensional anomaly detection (`internal/alert/`)
- ✓ **Outbox worker** — Polls and dispatches pending events (`internal/worker/`)
- ✓ **Action registry** — Whitelist-enforced action config with proposal/apply workflow (`internal/action/`)
- ✓ **Ontology V2** — AIP semantic object schema, link resolution, action binding (`internal/ontology/`)
- ✓ **Core API Completion** — All 6 endpoints returning 501 implemented (Phase 1, completed 2026-06-03)
- ✓ **Error Handling & Observability** — Proper HTTP status codes and structured error details (Phase 2, completed 2026-06-03)
- ✓ **Code Hygiene & Cleanup** — Python/SQLite remnants, dead code, deprecated shims removed (Phase 3, completed 2026-06-03)
- ✓ **Bug Fixes & Stability** — JSON decode errors, marshal failures, SQL injection risks fixed (Phase 4, completed 2026-06-03)
- ✓ **Security Hardening** — CORS scheme validation, auth fixes (Phase 5, completed 2026-06-03)
- ✓ **Integration & End-to-End Demo** — Frontend-to-backend wired, E2E tests pass (Phase 6, completed 2026-06-03)

### Active

- [ ] **INT-01**: MCP 服务器身份泛化 — 改名 + instructions 模糊化，Agent 无法识别项目身份
- [ ] **INT-02**: MCP 工具名抽象 — 按业务能力重新分组命名，抹掉 internal 包映射
- [ ] **INT-03**: describe_ontology 输出裁剪 — 移除 / 混淆 schema 细节
- [ ] **INT-04**: get_system_status 输出裁剪 — 移除 table_counts 等架构泄露
- [ ] **INT-05**: get_object / get_linked_objects 字段级过滤 — 利用 LLMReadable/sensitivity 标记过滤
- [ ] **INT-06**: search_objects 安全加固 — 结果限制、分页上限、字段过滤
- [ ] **INT-07**: run_pipeline 输入加固 — config 改为 allowlist，data_dir 固定

### Out of Scope

- Multi-tenant isolation — single-tenant deployment for demo
- Production-grade auth (OAuth, RBAC) — single bearer token is sufficient for demo
- Horizontal scaling / k8s deployment — Docker Compose is sufficient
- New channel adapters beyond Feishu/GitHub — existing two are sufficient
- Performance optimization beyond basic cleanup — functional correctness first

## Context

**Milestone v1 已完成** — 6 个阶段闭合并交付了一个可演示的闭环治理平台。所有 API 端点正常返回，前端连接后端，E2E 测试通过。

**Milestone v1.1 启动** — 重点转向 MCP 信息收束。现有 MCP Server（~31 个工具，12 组注册函数）在服务器身份自述、工具命名、输出内容三个维度上过度暴露了项目架构信息。已有的 `LLMReadable` / `sensitivity` 标记在 handler 层未被利用做字段级过滤。

**Frontend** uses React 19 with Vite, TanStack Query for server state, Radix UI for accessible primitives, and Tailwind CSS v4 for styling. Tests use Vitest + Testing Library + Playwright.

**Backend** uses Go 1.23 with chi/v5 router, pgx/v5 for PostgreSQL, zap for structured logging, testify + testcontainers for testing.

## Constraints

- **Tech stack**: Fixed — Go 1.23 backend, React 19 frontend, PostgreSQL 15, Docker Compose
- **Timeline**: Demo-ready — focus on completeness and bug fixes, not new features
- **Dependencies**: Feishu/Lark and GitHub integrations must remain functional
- **Compatibility**: MCP protocol compatibility must remain intact — Pi Agent integration must not break
- **Security**: Agent must not be able to infer project architecture or access unauthorized data through MCP

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Keep existing Go/PostgreSQL stack | Migration already completed, works well | ✓ Good |
| Fix bugs before adding features | Closed-loop demo deliverable prioritized | ✓ Done (v1) |
| MCP 信息收束 — 通用身份 + 严格裁剪 | Agent 不应能从 MCP 推断项目架构 | ✓ Decided (v1.1) |
| MCP 工具抽象 — 重新分组命名 | 用业务能力命名代替领域命名，抹掉 internal 包映射 | ✓ Decided (v1.1) |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-06-06 after v1.1 milestone initialization*
