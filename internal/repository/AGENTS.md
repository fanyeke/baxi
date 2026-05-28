# repository: Data Access Layer

**Generated:** 2026-05-28
**Branch:** main

## OVERVIEW

Domain-organized repository layer with PoolProvider injection. 10 subpackages within `internal/repository/`:
- `common/` — PoolProvider base infrastructure
- `governance/` — Governance config snapshots, object schemas
- `decision/` — Decision cases, LLM decisions, proposals
- `task/` — Task queries
- `alert/` — Alert event queries
- `outbox/` — Outbox event queries
- `log/` — Log querying
- `context/` — Qoder context data
- `status/` — System status (table counts, pipeline runs)
- `ontology/` — Object queries across dwd/mart/ops tables

Flat compatibility files (`*_repository.go`) still exist in the parent package and delegate to the new subpackages.

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Add a governance query | `repository/governance/` | Implement interface from `interfaces.go` |
| Add a decision query | `repository/decision/` | PoolProvider injected via constructor |
| Add a task query | `repository/task/` | Same pattern |
| Define shared interfaces | `repository/interfaces.go` | May still use old pool-as-param pattern |

## KEY PATTERNS

- **PoolProvider injection**: Each subpackage repository embeds `*common.PoolProvider` via constructor
- **Domain isolation**: Subpackages prevent cross-imports between governance, decision, task repos
- **Backward compat**: Flat `*_repository.go` files delegate to subpackages via `ensureInitialized(pool)` lazy init
- **Inline DDL in tests**: Tests use `CREATE TABLE IF NOT EXISTS` rather than migrations
- **Raw SQL**: No query builder — handwritten SQL with parameterized queries

## ANTI-PATTERNS

- **interfaces.go still uses pool-as-param**: Interface methods still pass `pool *pgxpool.Pool` even though implementations use PoolProvider
- **Tests bypass migrations**: Inline DDL diverges from production schema
- **No query builder**: Schema changes require grep across all repo files
- **Dual identity**: Some callers use the old flat interface, others use the new subpackages
