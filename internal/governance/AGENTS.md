# governance: Data Governance

**Branch:** main

## OVERVIEW
Data governance enforcement: classification, lineage, access control, markings, and redaction. 10 files.

Exposed via MCP tools: `check_access`, `get_classification`. See `internal/mcp/tools_governance.go` for handler implementations.

## WHERE TO LOOK

| Task | File | Notes |
|------|------|-------|
| Access control | `access_policy.go` | Role-based access decisions |
| Data classification | `classification.go` | PII/sensitive/internal classification levels |
| Data lineage | `lineage.go` | Source-to-target field lineage tracking |
| Checkpoint rules | `checkpoint.go` | Pipeline stage gate rules |
| PII redaction | `redaction.go` | Field-level redaction for LLM context |
| Markings | `marking_adapter.go` | Data marking integration with decision engine |

## KEY PATTERNS

- **Config-driven**: Rules loaded from `config/*.yml` at startup
- **Governance in decision loop**: Classification + redaction applied before LLM context is built
- **Pool stored as field**: ClassificationService, LineageService, etc. store `pool *pgxpool.Pool` (inconsistent with PoolProvider pattern)

## ANTI-PATTERNS

- access_policy.go imports `baxi/internal/api/dto` (reverse dependency — should use model)
- Dual pool approach: stores pool as struct field AND passes it as parameter in some methods
- No direct tests for access_policy, checkpoint, or redaction services
