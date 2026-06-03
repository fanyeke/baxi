---
phase: 06-integration-end-to-end-demo
plan: 02
subsystem: frontend
tags: [typescript, react, governance, pipeline, type-alignment]
requires:
  - phase: 01-core-api-completion
    provides: Backend DTO shape (CatalogResponse with objects/datasets, Pipeline endpoint with config field)
provides:
  - Governance 页面类型匹配后端 DTO
  - Pipeline 请求体字段名对齐
affects: [phase 06 integration demo verification]

tech-stack:
  added: []
  patterns: ["后端 DTO 作为前端类型 source of truth"]

key-files:
  created: []
  modified:
    - frontend/src/api/governance.ts
    - frontend/src/pages/Governance.tsx
    - frontend/src/pages/__tests__/Governance.test.tsx
    - frontend/src/pages/Pipeline.tsx

key-decisions:
  - "删除 CatalogAsset 接口，新增 CatalogObject/CatalogDataset 匹配后端 DTO"
  - "CatalogTab 从 data.assets 迁移到 data.objects，7 列减为 5 列"
  - "Pipeline 请求体字段名从 pipeline_type 改为 config"

requirements-completed: [INT-01]

duration: 12 min
completed: 2026-06-03
---

# Phase 6 Plan 2: 前端类型对齐（Governance + Pipeline）Summary

**Governance API 类型和页面组件对齐后端 DTO，Pipeline 请求体字段名修复**

## Performance

- **Duration:** 12 min
- **Started:** 2026-06-03T23:33:00Z
- **Completed:** 2026-06-03T23:45:00Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Governance API 类型定义：删除 `CatalogAsset`，新增 `CatalogObject`/`CatalogDataset`，`CatalogResponse` 使用 `objects`/`datasets` 字段
- Governance 页面 CatalogTab：从 `data.assets` 迁移到 `data.objects`，列从 7 列调整为 5 列（对象类型、来源数据集、主键、属性数、链接数）
- Governance SummaryStats：`assets.length` → `objects.length`，标签"数据资产"→"数据对象"
- Pipeline.tsx：请求体字段名从 `{ pipeline_type: type }` 改为 `{ config: type }`，匹配后端期望
- 测试更新：mock 数据从旧 `assets` 形状改为新 `objects`/`datasets` 形状，所有 6 个测试通过

## Task Commits

Each task was committed atomically:

1. **Task 1: 修复 governance API 类型 + Governance 页面组件 + Governance 测试** - `98c9987` (feat)
2. **Task 2: 修复 Pipeline.tsx 请求体字段名** - `3802cc6` (feat)

## Files Modified

- `frontend/src/api/governance.ts` - 删除 CatalogAsset，新增 CatalogObject/CatalogDataset，更新 CatalogResponse
- `frontend/src/pages/Governance.tsx` - CatalogTab 和 SummaryStats 使用 data.objects
- `frontend/src/pages/__tests__/Governance.test.tsx` - mock 数据更新，测试检查新字段
- `frontend/src/pages/Pipeline.tsx` - 请求体字段 pipeline_type → config

## Decisions Made

- 后端 DTO 作为前端类型的唯一 source of truth，前端不保留独立类型定义
- CatalogTab 列数从 7 列（含描述/粒度/状态等通用列）精简为 5 列（匹配实际数据特征）

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Governance 和 Pipeline 前端类型已对齐后端 DTO
- 后续计划 06-03 可以依赖这些类型进行集成验证
- 前端 TypeScript 编译零错误，Governance 测试全部通过

---

*Phase: 06-integration-end-to-end-demo*
*Completed: 2026-06-03*
