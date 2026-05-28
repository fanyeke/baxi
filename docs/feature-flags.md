# Feature Flags

Baxi uses environment-variable-driven feature flags to control gradual rollout of new implementations.
All flags default to **false** (off). Set the env var to `1`, `true`, or `yes` to enable.

## Flag Reference

| Flag | Env Var | Default | Purpose |
|------|---------|---------|---------|
| `OntologyAwareRepo` | `USE_ONTOLOGY_AWARE_REPO` | off | Route queries through OntologyAwareAdapter (filters results against schema) |
| `MarkingService` | `USE_MARKING_SERVICE` | off | Use MarkingAdapter for field classification (combines DB classification + ontology sensitivity) |
| `DecisionLineageService` | `USE_DECISION_LINEAGE_SERVICE` | off | Enable DecisionLineageService for context lineage tracking |
| `NewContextBuilder` | `USE_NEW_CONTEXT_BUILDER` | off | Switch from ContextBuilder v1 to v2 (uses OntologyAwareRepo + MarkingService + Lineage) |
| `DualWrite` | `USE_DUAL_WRITE` | off | Write to both Python (SQLite) and Go (PostgreSQL) backends simultaneously |
| `GoPrimaryWrite` | `USE_GO_PRIMARY_WRITE` | off | Go backend is the primary writer; Python becomes read-only |

## Activation Order

Flags should be enabled in this sequence during migration:

1. `USE_ONTOLOGY_AWARE_REPO` -- verify schema filtering works
2. `USE_MARKING_SERVICE` -- verify classification + redaction
3. `USE_DECISION_LINEAGE_SERVICE` -- verify lineage tracking
4. `USE_NEW_CONTEXT_BUILDER` -- enable v2 context builder (depends on 1-3)
5. `USE_DUAL_WRITE` -- parallel write to both backends
6. `USE_GO_PRIMARY_WRITE` -- Go becomes primary writer

## Rollback

To disable a flag, unset the env var or set it to `0`/`false`/`no`. The system falls back to the previous behavior immediately (no restart needed for flags read per-request; config-level flags require restart).

## Code References

- Definition: `internal/feature/flags.go`
- Tests: `internal/feature/flags_test.go`
- Usage: `internal/decision/switchable_context_builder.go`, `internal/decision/context_builder_v2.go`
