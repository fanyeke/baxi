# Baxi — Governance + Analytics Platform

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

### Active

- [ ] Fix all 6 API endpoints returning 501 Not Implemented (DecideLLM, Compare, Replay, ListLLMDecisions, ListEvals, BatchDispatch)
- [ ] Replace generic 500 errors with proper HTTP status codes and structured error details
- [ ] Remove Python/SQLite migration remnants from pipeline preview and Makefile
- [ ] Remove dead code: unreachable llm CLI subcommand, placeholder worker.go, deprecated repository shims
- [ ] Fix known bugs: silently ignored JSON decode errors, marshaling failures, pagination type assertion
- [ ] Complete frontend-to-backend integration for all decision and governance features
- [ ] Ensure E2E integration and security tests pass cleanly
- [ ] Achieve demonstrable closed loop: pipeline → governance → decision → action → alert → feedback

### Out of Scope

- Multi-tenant isolation — single-tenant deployment for demo
- Production-grade auth (OAuth, RBAC) — single bearer token is sufficient for demo
- Horizontal scaling / k8s deployment — Docker Compose is sufficient
- New channel adapters beyond Feishu/GitHub — existing two are sufficient
- Performance optimization beyond basic cleanup — functional correctness first

## Context

**Brownfield codebase** migrated from Python/SQLite to Go/PostgreSQL. All Python code removed, but migration artifacts remain in pipeline preview, Makefile targets, and `migration_baseline/` directory. The codebase has established patterns: interface-driven design, per-step transactions, config-driven governance, provider pattern for LLM swappability.

**Frontend** uses React 19 with Vite, TanStack Query for server state, Radix UI for accessible primitives, and Tailwind CSS v4 for styling. Tests use Vitest + Testing Library + Playwright.

**Backend** uses Go 1.23 with chi/v5 router, pgx/v5 for PostgreSQL, zap for structured logging, testify + testcontainers for testing.

## Constraints

- **Tech stack**: Fixed — Go 1.23 backend, React 19 frontend, PostgreSQL 15, Docker Compose
- **Timeline**: Demo-ready — focus on completeness and bug fixes, not new features
- **Dependencies**: Feishu/Lark and GitHub integrations must remain functional
- **Compatibility**: MCP server contract must not break Pi Agent integration
- **Security**: Fix SQL injection risks and auth gaps before demo

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Keep existing Go/PostgreSQL stack | Migration already completed, works well | ✓ Good |
| Fix bugs before adding features | User wants demonstrable closed loop, not new capabilities | — Pending |
| Remove deprecated repository shims | Clean up dual APIs, enforce PoolProvider pattern | — Pending |
| Implement 501 stubs rather than remove | Frontend already expects these endpoints | — Pending |

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
*Last updated: 2026-06-03 after initialization*
