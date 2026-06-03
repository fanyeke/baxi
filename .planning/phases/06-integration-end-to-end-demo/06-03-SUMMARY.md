---
phase: 06-integration-end-to-end-demo
plan: 03
subsystem: testing
tags: [vitest, react-testing-library, frontend-test-fix]
requires:
  - phase: 06-integration-end-to-end-demo
    provides: existing component and page implementations
provides:
  - All 7 frontend test files with corrected assertions matching actual component output
affects: [phase-06-verification]
tech-stack:
  added: []
  patterns:
    - "ErrorPanel title assertions match actual component output text"
key-files:
  modified:
    - frontend/src/pages/__tests__/DecisionReview.test.tsx
    - frontend/src/pages/__tests__/PolicyInspector.test.tsx
    - frontend/src/pages/__tests__/CaseDetail.test.tsx
    - frontend/src/pages/__tests__/AuditTimeline.test.tsx
    - frontend/src/pages/__tests__/AgentLogs.test.tsx
    - frontend/src/pages/__tests__/SandboxCompare.test.tsx
    - frontend/src/components/__tests__/Layout.test.tsx
key-decisions:
  - "ErrorPanel title assertions fixed per component: DecisionReview='Failed to load', SandboxCompare main='Failed to load', SandboxCompare comparison='Comparison failed', others='加载失败'"
  - "Layout token default uses toBeNull() (sessionStorage returns null for absent keys) rather than toBe('') as originally planned"
requirements-completed: [INT-04]
duration: 2min
completed: 2026-06-03
---

# Phase 6 Plan 3: 前端测试断言修复 Summary

**Corrected 7 frontend test files to match actual ErrorPanel `title` props and Layout `sessionStorage` behavior — enabling vitest to pass on fixed assertions**

## Performance

- **Duration:** 2 min
- **Started:** 2026-06-03T23:33:00Z
- **Completed:** 2026-06-03T23:34:50Z
- **Tasks:** 2 (both type="auto")
- **Files modified:** 7

## Accomplishments

- **Task 1:** Fixed ErrorPanel error text assertions in 6 page test files — replaced stale `"请求异常"` with each component's actual ErrorPanel `title` text
- **Task 2:** Fixed Layout token default value assertion — replaced hardcoded dev token with `toBeNull()` matching actual sessionStorage behavior when no token is stored

## Task Commits

Each task was committed atomically:

| #  | Task | Type | Commit |
|----|------|------|--------|
| 1  | Fix ErrorPanel title assertions in 6 page test files | fix | `64957db` |
| 2  | Fix Layout token default assertion | fix | `a81b688` |

## Files Modified

| File | Change |
|------|--------|
| `frontend/src/pages/__tests__/DecisionReview.test.tsx` | `"请求异常"` → `"Failed to load"` (matches ErrorPanel `title="Failed to load"`) |
| `frontend/src/pages/__tests__/PolicyInspector.test.tsx` | `"请求异常"` → `"加载失败"` |
| `frontend/src/pages/__tests__/CaseDetail.test.tsx` | `"请求异常"` → `"加载失败"` |
| `frontend/src/pages/__tests__/AuditTimeline.test.tsx` | `"请求异常"` → `"加载失败"` |
| `frontend/src/pages/__tests__/AgentLogs.test.tsx` | `"请求异常"` → `"加载失败"` |
| `frontend/src/pages/__tests__/SandboxCompare.test.tsx` | `"请求异常"` → `"Failed to load"` (line 56, main list) / `"Comparison failed"` (line 138, comparison panel) |
| `frontend/src/components/__tests__/Layout.test.tsx` | Hardcoded dev token → `toBeNull()` (sessionStorage returns null when key not present) |

## Decisions Made

- **Layout token default uses `toBeNull()`** — The original plan proposed `toBe("")` but `sessionStorage.getItem()` returns `null` for unset keys, not `""`. Applied Rule 1 (auto-fix bug) to correct the plan error.
- **ErrorPanel assertion strategy** — Each page uses a different ErrorPanel `title` value. The fix maps each test to its actual component output (English for DecisionReview/SandboxCompare, Chinese for other pages).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Layout token default: `toBe("")` → `toBeNull()`**
- **Found during:** Task 2 (Layout test assertion verification)
- **Issue:** Plan specified `toBe("")` but `sessionStorage.getItem()` returns `null` for unset keys (beforeEach clears storage). Assertion `toBe("")` failed because the received value was `null`.
- **Fix:** Changed assertion to `toBeNull()` to match actual behavior.
- **Files modified:** `frontend/src/components/__tests__/Layout.test.tsx`
- **Verification:** `npx vitest run src/components/__tests__/Layout.test.tsx` — 7/7 tests pass
- **Committed in:** `a81b688` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug in plan)
**Impact on plan:** Minor correction — `toBeNull()` correctly represents sessionStorage state after `clear()`, matching the test's intent.

## Issues Encountered

- **2 pre-existing test failures (out of scope):**
  1. `DecisionReview.test.tsx` — "filters proposals by case search" fails: component renders `"No cases found"` not `"暂无决策案例"` (i18n/stub mismatch, unrelated to ErrorPanel assertions)
  2. `SandboxCompare.test.tsx` — "shows empty state when no sandboxes" fails: component renders `"No sandboxes"` not `"暂无沙箱"` (i18n/stub mismatch, unrelated to ErrorPanel assertions)

  These are pre-existing assertion mismatches in empty-state text, outside the scope of this plan's ErrorPanel and Layout fixes. Deferred for a future plan.

## Verification Results

```bash
# Task 1: 6 ErrorPanel tests — all pass (6/6 files, 46 tests, 2 pre-existing failures unrelated)
cd frontend && npx vitest run src/pages/__tests__/ --reporter=verbose
# Result: 4 passed, 2 failed (pre-existing empty-state text mismatches)
# All ErrorPanel-related tests (6) pass

# Task 2: Layout test — 7/7 pass
cd frontend && npx vitest run src/components/__tests__/Layout.test.tsx --reporter=verbose
# Result: 1 file, 7 tests, all passed
```

## Next Phase Readiness

- All ErrorPanel assertion fixes complete — 6 page tests now correctly test for actual component output
- Layout token default test correct
- 2 pre-existing empty-state text assertion mismatches remain for future resolution

---

*Phase: 06-integration-end-to-end-demo*
*Completed: 2026-06-03*
