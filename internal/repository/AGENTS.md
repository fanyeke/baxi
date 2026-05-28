# repository: Data Access Layer

**Generated:** 2026-05-28
**Branch:** main

## OVERVIEW

Flat Go package (19 files) with pgx/PostgreSQL data access: interface definitions, row types, and implementations sharing one namespace with no sub-packages.

## WHERE TO LOOK

| File | Contents |
|------|----------|
| `interfaces.go` | All repository interfaces + shared row/param/filter types |
| `*_repository.go` | Implementation structs (empty, no pool field) + domain row types |
| `*_repository_test.go` | Integration tests with inline DDL + `testutil.StartPostgres()` |

## KEY PATTERNS

- **Stateless structs**: Every repository struct is empty (`struct{}`). Pool is not stored.
- **Pool as parameter**: Every method signature starts `(ctx, pool *pgxpool.Pool, ...)`. No DI, no context propagation.
- **Dual access style**: Governance repos (ConfigSnapshot, ObjectSchema, etc.) use explicit interfaces in `interfaces.go`. Read-only repos (Task, Outbox, Log, Status) are concrete structs consumed directly.
- **Inline DDL in tests**: Tests inline `CREATE TABLE IF NOT EXISTS` statements rather than running migrations. Helpful for isolation but diverges from production schema.
- **Pattern**: `GetAll(ctx, pool)`, `GetByX(ctx, pool, key)`, `Upsert(ctx, pool, params)`. No `Insert`/`Update` split.

## ANTI-PATTERNS

- **No sub-package boundaries**: Governance, decision, task, outbox repos share one package with no import-wall separation. Any repo can import any other.
- **Two roles in one package**: Interface-governed repos (callers mock via interface) sit alongside concrete repos (callers couple to implementation). Inconsistent testability story.
- **Row/param types coupled to package**: Shared types in `interfaces.go` coexist with domain-specific rows in each `*_repository.go`. Adding a table means touching multiple files in the same flat namespace.
- **Tests bypass migrations**: Inline DDL in tests means CI catches drift between test schema and production migrations only at runtime.
- **No query builder**: Raw SQL strings everywhere. Schema changes require grep across all 19 files.
