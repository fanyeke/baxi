# Requirements: Baxi Demo-Ready Platform

**Defined:** 2026-06-03
**Core Value:** A complete, demonstrable closed-loop governance and analytics platform with no critical bugs

## v1 Requirements

### API Completeness

- [ ] **API-01**: `POST /api/v1/decisions/llm` — DecideLLM endpoint fully implements LLM-assisted decision flow with context building, validation, and repair retry
- [ ] **API-02**: `POST /api/v1/decisions/compare` — Compare endpoint accepts two decision IDs and returns structured comparison with diff visualization
- [ ] **API-03**: `POST /api/v1/decisions/replay` — Replay endpoint re-executes a decision with original or modified context and returns new result
- [ ] **API-04**: `GET /api/v1/decisions/llm` — ListLLMDecisions endpoint returns paginated list of LLM-assisted decisions with filters
- [ ] **API-05**: `GET /api/v1/evals` — ListEvals endpoint returns evaluation metrics and scores for decision quality
- [ ] **API-06**: `POST /api/v1/outbox/dispatch` — HandleBatchDispatch endpoint processes pending outbox events in batch
- [ ] **API-07**: All implemented endpoints return OpenAPI-documented response schemas

### Error Handling

- [ ] **ERR-01**: Service errors map to appropriate HTTP status codes (400 Bad Request, 404 Not Found, 409 Conflict, 502 Bad Gateway) instead of generic 500
- [ ] **ERR-02**: Error responses include structured JSON with `code`, `message`, and optional `details` fields
- [ ] **ERR-03**: Malformed JSON in request bodies returns 400 with parse error details (not silently ignored)
- [ ] **ERR-04**: Database connection failures return 503 with retry-after guidance
- [ ] **ERR-05**: Validation errors return 400 with field-level error details

### Code Hygiene

- [ ] **HYG-01**: Pipeline preview returns correct Go commands (`go run ./cmd/baxi-cli pipeline run`) not Python scripts
- [ ] **HYG-02**: Makefile targets no longer reference Python scripts for verification
- [ ] **HYG-03**: Deprecated repository shim files removed (governance_repository.go, decision_repository.go, ontology_repository.go, outbox_repository.go, log_repository.go, context_repository.go)
- [ ] **HYG-04**: All callers migrated from deprecated repositories to subpackage repositories with PoolProvider
- [ ] **HYG-05**: Dead CLI subcommand `llm.go` removed or wired into main.go dispatch
- [ ] **HYG-06**: Placeholder `internal/worker/worker.go` removed
- [ ] **HYG-07**: Migration baseline directory archived or removed (sqlite_schema.sql, Python scripts)

### Bug Fixes

- [ ] **BUG-01**: `internal/api/handler/action.go` returns 400 on JSON decode failure instead of proceeding with defaults
- [ ] **BUG-02**: `internal/alert/engine.go` handles JSON marshal errors explicitly (no silent data loss)
- [ ] **BUG-03**: `internal/feishu/client.go` handles page_token type assertion failure with proper error propagation
- [ ] **BUG-04**: Goose migration sequence is continuous (no missing 015, 025 — audit and fix gaps)
- [ ] **BUG-05**: SQL injection risk in ontology repository eliminated (schema.table always sanitized, allowlist check before query)

### Security

- [ ] **SEC-01**: Auth middleware supports token rotation or JWT with claims and expiry
- [ ] **SEC-02**: CORS origin check validates scheme explicitly (http vs https)
- [ ] **SEC-03**: Docker Compose does not hardcode credentials in plain text

### Integration & Testing

- [ ] **INT-01**: Frontend pages for decisions, governance, pipeline, alerts all connect to working backend endpoints
- [ ] **INT-02**: E2E integration tests (`test/integration/phase7_test.go`) pass cleanly
- [ ] **INT-03**: Security E2E tests (`test/security/phase7_test.go`) pass cleanly
- [ ] **INT-04**: Frontend unit tests (`frontend/src/pages/__tests__/*.test.tsx`) pass cleanly
- [ ] **INT-05**: Full closed-loop demo works: trigger pipeline → governance rules fire → decision created → action executed → alert sent → result visible in frontend

## v2 Requirements

### Performance

- **PERF-01**: Pipeline execution optimized for sub-minute runs on demo datasets
- **PERF-02**: Frontend page load under 2 seconds with React.lazy code splitting

### Features

- **FEAT-01**: Real-time WebSocket alerts instead of polling
- **FEAT-02**: Advanced decision analytics dashboard with trend charts
- **FEAT-03**: Multi-language support (i18n) for frontend

## Out of Scope

| Feature | Reason |
|---------|--------|
| Multi-tenant isolation | Single-tenant demo deployment sufficient |
| OAuth / social login | Bearer token auth sufficient for demo |
| Kubernetes deployment | Docker Compose sufficient for demo |
| New channel adapters | Feishu + GitHub sufficient |
| Performance benchmarking | Functional correctness prioritized |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| API-01 | Phase 1 | Pending |
| API-02 | Phase 1 | Pending |
| API-03 | Phase 1 | Pending |
| API-04 | Phase 1 | Pending |
| API-05 | Phase 1 | Pending |
| API-06 | Phase 1 | Pending |
| API-07 | Phase 1 | Pending |
| ERR-01 | Phase 2 | Pending |
| ERR-02 | Phase 2 | Pending |
| ERR-03 | Phase 2 | Pending |
| ERR-04 | Phase 2 | Pending |
| ERR-05 | Phase 2 | Pending |
| HYG-01 | Phase 3 | Pending |
| HYG-02 | Phase 3 | Pending |
| HYG-03 | Phase 3 | Pending |
| HYG-04 | Phase 3 | Pending |
| HYG-05 | Phase 3 | Pending |
| HYG-06 | Phase 3 | Pending |
| HYG-07 | Phase 3 | Pending |
| BUG-01 | Phase 4 | Pending |
| BUG-02 | Phase 4 | Pending |
| BUG-03 | Phase 4 | Pending |
| BUG-04 | Phase 4 | Pending |
| BUG-05 | Phase 4 | Pending |
| SEC-01 | Phase 5 | Pending |
| SEC-02 | Phase 5 | Pending |
| SEC-03 | Phase 5 | Pending |
| INT-01 | Phase 6 | Pending |
| INT-02 | Phase 6 | Pending |
| INT-03 | Phase 6 | Pending |
| INT-04 | Phase 6 | Pending |
| INT-05 | Phase 6 | Pending |

**Coverage:**
- v1 requirements: 32 total
- Mapped to phases: 32
- Unmapped: 0 ✓

**Phase mapping summary:**

| Phase | Name | Requirements | Count |
|-------|------|--------------|-------|
| 1 | Core API Completion | API-01..API-07 | 7 |
| 2 | Error Handling & Observability | ERR-01..ERR-05 | 5 |
| 3 | Code Hygiene & Cleanup | HYG-01..HYG-07 | 7 |
| 4 | Bug Fixes & Stability | BUG-01..BUG-05 | 5 |
| 5 | Security Hardening | SEC-01..SEC-03 | 3 |
| 6 | Integration & End-to-End Demo | INT-01..INT-05 | 5 |

---
*Requirements defined: 2026-06-03*
*Last updated: 2026-06-03 after initial definition*
