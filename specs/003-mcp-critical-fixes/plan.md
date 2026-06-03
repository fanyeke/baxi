# MCP Critical Bug Fixes

**Plan ID**: 003-mcp-critical-fixes
**Date**: 2026-06-03
**Status**: In Progress

## Overview

根据 MCP 功能完整性评估报告验证后，确认 3 个需要修复的问题（1 个构建阻断、1 个测试数据错误、1 个功能缺失），其余 5 个已验证无需改动。

## 验证结果

| 报告中的问题 | 验证结果 | 说明 |
|---|---|---|
| list_alerts NULL scan | ✅ **确认修复** | `AlertRow.OwnerRole/Status` 是 `*string`，model 层是 `string`，编译失败 |
| resolve_case 状态冲突 | ❌ **已修复** | `main.go:274` 已用 `'closed'`，无需改动 |
| get_decision_context 崩溃 | ❌ **已有防护** | `context_builder.go:167-168` 返回明确错误，不会崩溃 |
| propose_action 错误信息 | ⚠️ **设计如此** | handler 返回 service 层错误，`success:false` 是有效业务状态 |
| search_objects 列名泄露 | ❌ **当前正确** | V2 compiler 用 clean property name 作为 key，非 expression 文本 |
| build_context 不可用 | ✅ **确认修复** | `LoadV2Schema()` 从未在 main.go 中调用 |

## 修复项

### Fix 1: Alert 类型不匹配 (P0 — 构建阻断)

**问题**: 3 处映射站点将 `*string` 直接赋给 `string`，编译失败。

**方案**: 在 service 层使用内联 nil 安全解引用（与 `task_service.go:61-63` 同模式）。

**文件**:

| 文件 | 行 | 改动 |
|---|---|---|
| `internal/service/alert_service.go` | 60-61 | `OwnerRole: row.OwnerRole` → 内联解引用 |
| `internal/service/qoder_service.go` | 205-206 | `OwnerRole: repoRow.OwnerRole` → 内联解引用 |

**模式**:
```go
ownerRole := ""
if row.OwnerRole != nil {
    ownerRole = *row.OwnerRole
}
status := ""
if row.Status != nil {
    status = *row.Status
}
```

### Fix 2: 测试数据违反 CHECK 约束 (P1)

**问题**: `repository_test.go` 4 处使用 `'resolved'`，CHECK 约束不允许。

**文件**: `internal/repository/decision/repository_test.go`

| 行 | 改动 |
|---|---|
| 124 | `'resolved'` → `'closed'` |
| 128 | `assert.Equal(t, "resolved"` → `"closed"` |
| 135 | INSERT `'resolved'` → `'closed'` |
| 146 | INSERT `'resolved'` → `'closed'` |

### Fix 3: build_context 服务不可用 (P2)

**问题**: `LoadV2Schema()` 从未在 main.go 中调用，`AllObjectsV2()` 返回空 map。

**文件**: `cmd/baxi-mcp/main.go`，在第 157 行后添加:

```go
// Load v2 schema to enable build_context service
if objRegistry != nil {
    v2SchemaPath := filepath.Join(configDir, "aip_object_schema_v2.yml")
    if v2Err := objRegistry.LoadV2Schema(v2SchemaPath); v2Err != nil {
        zapLog.Warn("failed to load v2 schema, build_context will be unavailable", zap.Error(v2Err))
    }
}
```

## 执行顺序

1. Fix 1 (P0) — 解除构建阻断
2. Fix 2 (P1) — 修测试数据
3. Fix 3 (P2) — 启用 build_context

## 验证

```bash
go build ./...
go test ./internal/service/... ./internal/repository/decision/... -v -count=1
go test ./test/integration/... -v -count=1 -run TestMCP
```
