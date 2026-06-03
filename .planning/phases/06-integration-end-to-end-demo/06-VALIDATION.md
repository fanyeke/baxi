---
phase: 06
slug: integration-end-to-end-demo
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-03
---

# Phase 06 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework (Go)** | Go testing + testify v1.9.0 |
| **Framework (Frontend)** | Vitest ^4.1.7 + Testing Library |
| **Config file (Go)** | `go.mod` + `.golangci.yml` |
| **Config file (Frontend)** | `frontend/vitest.config.ts` |
| **Quick run (Go)** | `go test -short ./internal/...` |
| **Quick run (Frontend)** | `cd frontend && npx vitest run --reporter=verbose` |
| **Full suite (Go internal)** | `go test ./internal/...` |
| **Full suite (Go E2E)** | `go test -tags=integration ./test/...` (requires Docker) |
| **Full suite (Frontend)** | `cd frontend && npm test` |
| **Estimated runtime** | ~120 seconds |

---

## Sampling Rate

- **After every task commit (Go):** `go test -short ./internal/...`
- **After every task commit (Frontend):** `cd frontend && npx vitest run --changed`
- **After every plan wave:** Full frontend suite + Go internal tests
- **Before verification:** Full suite (Go internal + frontend + E2E with Docker)
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 06-01-01 | 01 | 1 | INT-02 | — | Test compilation fixed | unit | `go test ./internal/service/...` | ✅ | ⬜ pending |
| 06-01-02 | 01 | 1 | INT-02 | — | Test compilation fixed | unit | `go test ./internal/api/handler/...` | ✅ | ⬜ pending |
| 06-01-03 | 01 | 1 | INT-02 | — | Test compilation fixed | unit | `go test ./internal/decision/...` | ✅ | ⬜ pending |
| 06-02-01 | 02 | 1 | INT-01 | T-01 | Frontend types match backend DTOs | unit | `cd frontend && npx vitest run` | ✅ | ⬜ pending |
| 06-02-02 | 02 | 1 | INT-04 | — | Frontend unit tests pass | unit | `cd frontend && npx vitest run` | ✅ | ⬜ pending |
| 06-03-01 | 03 | 2 | INT-05 | — | Demo walkthrough verifiable | manual | Manual trigger pipeline → check frontend | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Fix 4 Go test compilation errors (listed in RESEARCH.md)
- [ ] Fix 10 frontend unit test assertion mismatches
- [ ] Align frontend governance types with backend DTOs

*Existing infrastructure covers all phase requirements after fixes.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Full closed-loop demo | INT-05 | Requires running pipeline + chained governance/decision/action/alert flow | Start API + worker, trigger pipeline, check frontend for audit trail |
| Frontend pages load live data | INT-01 | Requires running backend with real data | Start API + frontend, navigate to each page, verify data displays |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
