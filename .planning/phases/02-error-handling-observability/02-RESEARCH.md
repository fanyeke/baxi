# Phase 2: Error Handling & Observability - Research

**Researched:** 2026-06-03
**Domain:** Go HTTP API 错误处理与结构化响应
**Confidence:** HIGH

## 总结

Baxi 后端 API 已有基本的错误处理基础设施：`middleware/error.go` 定义了 `APIError` 结构体（5字段：request_id, error_code, message, diagnosis, suggested_action）、`WriteError()` 函数和 `RecoveryMiddleware()`。`handler/error_helper.go` 提供了便捷包装 `writeError()`、`classifyError()` 和默认的 diagnosis/action 文本。

**但存在关键缺陷：**
1. 缺少 `SERVICE_UNAVAILABLE`（503）和 `CONFLICT`（409）错误码常量 — 当前冲突误用 `BAD_REQUEST`
2. `details` 字段不存在 — ERR-02 要求可选 `details` 字段
3. `classifyError()` 仅映射 `pgx.ErrNoRows` → 404，其余全部 → 500，无法检测数据库连接失败
4. `action.go` 和 `outbox.go` 的 JSON 解码错误被静默忽略（`_ = json.NewDecoder...`）
5. 验证错误返回通用信息，无字段级错误详情
6. 无 `Retry-After` 头部的 503 响应
7. 部分 handler 的 nil/not-found 检测依赖于字符串匹配（如 `sandbox.go` 的 `errors.New("sandbox " + id + " not found")`），而非类型化错误

**核心推荐：** 扩展现有 `middleware/error.go` 基础设施（添加错误码常量、`details` 字段、DB 错误检测），而不是重写。在 `handler/error_helper.go` 中添加验证错误辅助函数。修复所有 handler 中的 JSON 解码静默忽略和错误的状态码映射。

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| ERR-01 | Service errors map to appropriate HTTP status codes (400, 404, 409, 502) instead of generic 500 | 已识别6个handler需要修复状态码映射；需添加 `CONFLICT`, `SERVICE_UNAVAILABLE` 常量；需扩展 `classifyError()` |
| ERR-02 | Error responses include structured JSON with `code`, `message`, and optional `details` fields | `APIError` 已有 `error_code` 和 `message` ；需添加 `details` 字段；现有格式已包含额外字段但需要保持兼容 |
| ERR-03 | Malformed JSON returns 400 with parse error details | `action.go` 和 `outbox.go` 静默忽略解码错误；其他handler已返回400但缺少具体解析错误详情 |
| ERR-04 | Database connection failures return 503 with retry-after guidance | 需要检测 `pgx` 连接错误；添加 `SERVICE_UNAVAILABLE` 错误码；设置 `Retry-After` 头部 |
| ERR-05 | Validation errors return 400 with field-level error details | 需添加 `ValidationError` 结构和 `WriteValidationError()` 函数；扩展 `details` 字段以支持字段级错误 |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Error response struct | API / Backend | — | `APIError` 定义在 `internal/api/middleware/` ，属于 API 层 |
| Error writing utilities | API / Backend | — | `writeError()` 等辅助函数在 `internal/api/handler/` |
| Validation error details | API / Backend | — | JSON 解码和请求验证都在 handler 层完成 |
| DB failure detection | API / Backend | Database / Storage | `pgx` 错误类型需要在 API 层识别并映射到 503 |
| Malformed JSON detection | API / Backend | — | `json.NewDecoder(r.Body).Decode()` 在 handler 中执行 |
| Error code constants | API / Backend | — | 定义在 `internal/api/middleware/error.go` |
| Retry-After header | API / Backend | — | 在 503 响应中由 handler 或 middleware 设置 |

## Standard Stack

### Core - Error Response Infrastructure (existing, 需要扩展)

| Library | 版本 | 用途 | 为什么是标准 |
|---------|------|------|-------------|
| `middleware/APIError` | 现存 | 结构化 JSON 错误响应结构体 | 已在代码库中使用，需扩展添加 `details` 字段 |
| `middleware/WriteError()` | 现存 | 写入结构化错误响应的核心函数 | 所有中间件（auth, rbac）都使用此函数 |
| `handler/writeError()` | 现存 | handler 层的便捷包装 | 所有 14 个 handler 都使用此包装 |
| `handler/classifyError()` | 现存 | 错误类型到 HTTP 状态码的映射 | 需要扩展以支持更多错误类型 |
| `middleware/RecoveryMiddleware` | 现存 | panic 恢复中间件 | 已在路由栈中使用 |

### Standard Go Error Detection Libraries

| Library | 版本 | 用途 | 何时使用 |
|---------|------|------|----------|
| `errors.Is()` | stdlib | 错误链匹配 | 检测 `pgx.ErrNoRows`、自定义错误类型 |
| `errors.As()` | stdlib | 错误类型断言 | 检测包装的数据库连接错误 |
| `pgx` 错误类型 | v5.5.5 | PostgreSQL 驱动错误 | `pgx.ErrNoRows`（404）、连接错误（503） |

### Validation (field-level errors)

| 方法 | 用法 | 说明 |
|------|------|------|
| 手动 if-check + `FieldError` | handler 中直接检查 | 项目现有模式，不引入新依赖 |
| `go-playground/validator/v10` | 结构体标签验证 | 可选增强，但为保持轻量不建议 Phase 2 引入 |

**不建议:** 引入第三方验证库。现有手动验证模式对 demo 足够了，只需将错误响应升级为包含字段级详情。如果未来需要，`go-playground/validator/v10` 是 Go 生态标准 [CITED: github.com/go-playground/validator]。

### Alternatives Considered

| 替代方案 | 当前方案 | 权衡 |
|---------|----------|------|
| 重写整个错误处理层 | 扩展现有基础设施 | 现有 `APIError` / `WriteError()` 已满足大多数需求，重写破坏前端兼容性 |
| 引入 validator 库 | 手动验证 + 结构化错误 | 新依赖增加复杂度，手动验证在现有代码库中已有一致模式 |
| 创建 HTTP 错误中间件 | 在 handler 中明确处理 | 中间件方案会隐藏 handler 中具体的错误上下文，handler 层处理更精确 |

**版本确认：**
```bash
# go-playground/validator 可用但未在项目中引入
# 现有 pgx v5.5.5 提供错误类型
```

## Architecture Patterns

### 系统架构图

```
Request
  │
  ▼
chi Router ──► RequestIDMiddleware (propagate/generate request_id)
  │
  ▼
RecoveryMiddleware (catch panics → JSON error)
  │
  ▼
CORSMiddleware
  │
  ▼
AuthMiddleware (validate Bearer token → 401 on failure)
  │
  ▼
Handler (parse request + validate + call service)
  │
  ├── JSON decode fail? ──► 400 + parse error details ──► Response
  │
  ├── Validation fail? ────► 400 + field-level errors ──► Response
  │
  ├── Service error?
  │   ├── pgx.ErrNoRows?       ──► 404 NOT_FOUND
  │   ├── DB connection err?   ──► 503 SERVICE_UNAVAILABLE + Retry-After
  │   ├── ErrProposalNotFound? ──► 404 NOT_FOUND
  │   ├── ErrNotApproved?      ──► 403 FORBIDDEN
  │   ├── ErrInvalidState?     ──► 409 CONFLICT
  │   └── Other?               ──► 500 INTERNAL_ERROR
  │
  └── Success ──► httputil.JSON(w, 2xx, data)
```

### 推荐的项目结构变更

```
internal/api/
├── middleware/
│   └── error.go              # 扩展：添加 SERVICE_UNAVAILABLE, CONFLICT 常量；添加 details 字段
│   └── error_test.go          # 扩展：添加新错误码常量的测试
│
├── handler/
│   └── error_helper.go        # 扩展：增强 classifyError()；添加 validationError 辅助函数
│   └── error_helper_test.go   # 扩展：添加验证错误测试、DB 错误测试
│
├── dto/
│   └── error.go               # 新增：ValidationError, FieldError 类型定义
```

### Pattern 1: 结构化错误响应（扩展现有）

**当前状态:** `APIError` 有 5 字段但缺少 `details`。ERR-02 要求 `code`, `message`, 可选 `details`。

**推荐扩展:**
```go
// internal/api/middleware/error.go — 扩展 APIError
type APIError struct {
    RequestID       string      `json:"request_id"`
    ErrorCode       string      `json:"error_code"`
    Message         string      `json:"message"`
    Diagnosis       string      `json:"diagnosis"`
    SuggestedAction string      `json:"suggested_action"`
    Details         interface{} `json:"details,omitempty"`  // 新增：可选字段级错误详情
}
```

### Pattern 2: 字段级验证错误（ERR-05）

**推荐数据结构:**
```go
// internal/api/dto/error.go — 新增
type FieldError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
    Code    string `json:"code,omitempty"`
}

type ValidationError struct {
    Fields []FieldError `json:"fields"`
}
```

**使用方式:**
```go
// handler 中使用
writeValidationError(w, r, "validation failed", []dto.FieldError{
    {Field: "source_type", Message: "source_type is required", Code: "required"},
    {Field: "source_id", Message: "source_id must be a valid UUID", Code: "invalid_format"},
})
// 响应:
// {
//   "request_id": "req_xxx",
//   "error_code": "VALIDATION_FAILED",
//   "message": "validation failed",
//   "details": {
//     "fields": [
//       {"field": "source_type", "message": "source_type is required", "code": "required"}
//     ]
//   }
// }
```

### Pattern 3: 数据库错误检测（ERR-04）

```go
// classifyError 扩展
func classifyError(err error) (int, string) {
    if errors.Is(err, pgx.ErrNoRows) {
        return http.StatusNotFound, middleware.NOT_FOUND
    }
    // 检测数据库连接错误
    if isDatabaseConnectionError(err) {
        return http.StatusServiceUnavailable, middleware.SERVICE_UNAVAILABLE
    }
    return http.StatusInternalServerError, middleware.INTERNAL_ERROR
}

func isDatabaseConnectionError(err error) bool {
    // pgx 连接错误通常包装在 errors 链中
    // 检测 "connection refused", "no such host", "timeout" 等
    if err == nil {
        return false
    }
    msg := strings.ToLower(err.Error())
    return strings.Contains(msg, "connection refused") ||
        strings.Contains(msg, "no such host") ||
        strings.Contains(msg, "connect: connection refused") ||
        strings.Contains(msg, "pool closed") ||
        strings.Contains(msg, "failed to connect")
}
```

### Anti-Patterns to Avoid

- **字符串匹配错误类型:** `sandbox.go` 中 `errors.New("sandbox " + id + " not found")` 字符串匹配 — 应使用类型化错误如 `ErrSandboxNotFound`
- **通用 500 掩盖具体错误:** 所有 handler 中 `default: 500 INTERNAL_ERROR` — 应分类识别特定错误类型
- **`_ = json.Decode` 静默忽略:** `action.go` 和 `outbox.go` — 应显式处理解码错误并返回 400
- **冲突误用 BAD_REQUEST:** 409 Conflict 应使用 `CONFLICT` 错误码而非 `BAD_REQUEST`

## Don't Hand-Roll

| 问题 | 不要自己构建 | 使用现有方案 | 原因 |
|------|-------------|-------------|------|
| 错误响应结构体 | 创建新的错误 DTO | 扩展现有 `middleware/APIError` | 现有结构已在前端使用，向前兼容 |
| JSON 序列化 | 手动 JSON 编码 | `encoding/json` + `json.NewEncoder` | stdlib 已足够，无需框架 |
| 错误类型定义 | 创建通用错误接口 | Go 标准 `errors` 包 + 类型断言 | 简单、可组合、不需要框架 |
| 请求日志 | 手动日志记录 | chi `middleware.Logger`（已在使用） | 已经配置好 |
| 路由参数解析 | 手动 URL 解析 | `chi.URLParam()`（已在使用） | 已在所有 handler 中使用 |

## Code Examples

### 示例 1: 扩展 APIError 添加 details 字段

Source: `internal/api/middleware/error.go`（现有代码扩展）

```go
// 新增错误码常量
const (
    UNAUTHORIZED      = "UNAUTHORIZED"
    FORBIDDEN         = "FORBIDDEN"
    BAD_REQUEST       = "BAD_REQUEST"
    NOT_FOUND         = "NOT_FOUND"
    CONFLICT          = "CONFLICT"          // 新增
    DB_QUERY_FAILED   = "DB_QUERY_FAILED"
    INTERNAL_ERROR    = "INTERNAL_ERROR"
    VALIDATION_FAILED = "VALIDATION_FAILED"
    SERVICE_UNAVAILABLE = "SERVICE_UNAVAILABLE" // 新增
)

// 扩展 APIError 结构体
type APIError struct {
    RequestID       string      `json:"request_id"`
    ErrorCode       string      `json:"error_code"`
    Message         string      `json:"message"`
    Diagnosis       string      `json:"diagnosis"`
    SuggestedAction string      `json:"suggested_action"`
    Details         interface{} `json:"details,omitempty"`  // 新增
}
```

### 示例 2: 写验证错误

Source: `internal/api/handler/error_helper.go`（新增函数）

```go
// FieldError 表示字段级验证错误
type FieldError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
    Code    string `json:"code,omitempty"`
}

// writeValidationError 写验证错误，包含字段级详情
func writeValidationError(w http.ResponseWriter, r *http.Request, message string, fields []FieldError) {
    details := map[string]interface{}{
        "fields": fields,
    }
    middleware.WriteErrorWithDetails(w, r, http.StatusBadRequest, middleware.VALIDATION_FAILED,
        message, defaultDiagnosis(middleware.VALIDATION_FAILED),
        defaultAction(middleware.VALIDATION_FAILED), details)
}

// 使用示例：
// writeValidationError(w, r, "validation failed", []FieldError{
//     {Field: "source_type", Message: "source_type is required", Code: "required"},
// })
```

### 示例 3: 修复静默 JSON 解码忽略

Source: `internal/api/handler/action.go`（bug 修复）

```go
// 修复前：
func (h *ActionHandler) HandleExecute(w http.ResponseWriter, r *http.Request) {
    var req executeRequest
    if r.Body != nil {
        _ = json.NewDecoder(r.Body).Decode(&req)  // BUG: 静默忽略错误
    }
    // ...继续使用零值
}

// 修复后：
func (h *ActionHandler) HandleExecute(w http.ResponseWriter, r *http.Request) {
    var req executeRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeErrorWithDetails(w, r, http.StatusBadRequest, middleware.BAD_REQUEST,
            "invalid request body", "JSON decode failed",
            "Provide a valid JSON request body",
            map[string]string{"parse_error": err.Error()})
        return
    }
    // ...
}
```

### 示例 4: 503 数据库连接失败 + Retry-After

Source: `internal/api/middleware/error.go` + handler 使用

```go
// writeDatabaseError 检测并写数据库错误
func writeDatabaseError(w http.ResponseWriter, r *http.Request, err error) {
    if isDatabaseConnectionError(err) {
        w.Header().Set("Retry-After", "5")  // 建议 5 秒后重试
        writeError(w, r, http.StatusServiceUnavailable, middleware.SERVICE_UNAVAILABLE,
            "Database temporarily unavailable")
        return
    }
    writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR,
        "internal server error")
}
```

## Common Pitfalls

### Pitfall 1: 静默 JSON 解码错误
**问题:** `action.go:HandleExecute` 和 `outbox.go:HandleBatchDispatch` 中 `_ = json.NewDecoder(r.Body).Decode(&req)` 忽略了解码错误，请求以零值继续处理
**原因:** 代码意图允许空 body（默认用 `dry_run=true`），但错误处理过于宽松
**如何避免:** 如果 body 为空则优雅处理（检查 `r.Body == nil` 或 `r.ContentLength == 0`），否则返回 400
**警告信号:** handler 方法包含 `_ = json.NewDecoder` 或 `// Default to ... if body parsing fails`

### Pitfall 2: 字符串匹配错误类型替代类型断言
**问题:** `sandbox.go` 使用 `errors.New("sandbox " + id + " not found")` 和字符串匹配，脆弱且不可靠
**原因:** 缺少类型化错误，服务层返回的也是原始字符串
**如何避免:** 使用 `errors.New("sandbox not found")` 作为包级别哨兵错误，用 `errors.Is()` 匹配
**警告信号:** handler 中包含 `err.Error() == "..."` 或 `strings.Contains(err.Error(), ...)`

### Pitfall 3: 409 Conflict 使用 BAD_REQUEST 错误码
**问题:** `outbox.go` 和 `review.go` 中使用 `middleware.BAD_REQUEST` 作为冲突响应的错误码
**原因:** `middleware/error.go` 中没有定义 `CONFLICT` 常量
**如何避免:** 添加 `CONFLICT = "CONFLICT"` 常量并在冲突响应中使用
**警告信号:** 使用 `http.StatusConflict` 但错误码是 `BAD_REQUEST`

### Pitfall 4: 数据库错误暴露内部信息
**问题:** 当前错误响应将数据库错误细节（表名、错误消息）返回给客户端
**原因:** `classifyError()` 生成的诊断文本包含内部细节
**如何避免:** 数据库错误始终返回通用消息，错误详情记录在日志中
**警告信号:** 错误响应中包含 "table", "column", "constraint" 等数据库术语

## State of the Art

| 旧方式 | 当前方式 | 何时变更 | 影响 |
|--------|---------|----------|------|
| 通用 500 错误 | 结构化错误码 + 详情 | Phase 2 | 前端可针对不同错误码做出不同响应 |
| 无验证错误详情 | 字段级验证错误 | Phase 2 | 前端可高亮具体输入字段 |
| 静默 JSON 忽略 | 明确的 400 解析错误 | Phase 2 | 排除一大类调试困难的 bug |
| 冲突被归类为 BAD_REQUEST | 使用 CONFLICT 错误码 | Phase 2 | 前端可区分验证错误与状态冲突 |
| DB 错误统一为 500 | 503 Service Unavailable | Phase 2 | 正确语义 + Retry-After 指导客户端重试 |

**已弃用/过时:**
- `_ = json.NewDecoder(r.Body).Decode(&req)`：静默忽略模式。应全部替换为显式错误处理
- 字符串匹配错误类型：应使用 `errors.Is()` + 类型化哨兵错误

## Assumptions Log

| # | 声明 | 章节 | 如果错误的风险 |
|---|------|------|---------------|
| A1 | 前端期望接收 5 字段错误格式（request_id, error_code, message, diagnosis, suggested_action） | Error Response | 添加 `details` 字段可能破坏旧前端响应解析；需要确认前端代码 |

该声明基于 `middleware/error.go` 注释 "Matches the old FastAPI error format exactly" 和 `AGENTS.md` 中的 "Error responses follow pre-existing error format for frontend compatibility"。前端的实际消费代码未在此研究中验证。

## 需要修改的文件清单

### 核心扩展

| 文件 | 修改内容 | 对应需求 |
|------|---------|---------|
| `internal/api/middleware/error.go` | 添加 `SERVICE_UNAVAILABLE`, `CONFLICT` 常量；扩展 `APIError.Details` 字段；添加 `WriteErrorWithDetails()` 函数 | ERR-01, ERR-02 |
| `internal/api/handler/error_helper.go` | 扩展 `classifyError()` 支持 DB 错误；添加 `writeValidationError()`、`writeDatabaseError()` 辅助函数 | ERR-01, ERR-04, ERR-05 |
| `internal/api/dto/error.go`（新增） | 定义 `FieldError`, `ValidationError` 结构体 | ERR-05 |

### Bug 修复（JSON 解码）

| 文件 | 修改内容 | 对应需求 |
|------|---------|---------|
| `internal/api/handler/action.go` | `HandleExecute`: 替换 `_ = json.NewDecoder` 为显式错误处理 + 400 响应 | ERR-03 |
| `internal/api/handler/outbox.go` | `HandleBatchDispatch`: 替换静默默认值为显式错误响应 | ERR-03 |

### 状态码修复

| 文件 | 修改内容 | 对应需求 |
|------|---------|---------|
| `internal/api/handler/outbox.go` | `HandleDispatch`/`HandleCancel`: 409 冲突响应改为使用 `CONFLICT` 错误码 | ERR-01 |
| `internal/api/handler/review.go` | `handleReviewAction`: 409 冲突响应改为使用 `CONFLICT` 错误码 | ERR-01 |
| `internal/api/handler/decision.go` | `Replay`: 503 响应改为使用 `SERVICE_UNAVAILABLE` 错误码 | ERR-01 |

### 验证错误增强

| 文件 | 修改内容 | 对应需求 |
|------|---------|---------|
| `internal/api/handler/decision.go` | `CreateCase`: 验证错误使用 `writeValidationError()` | ERR-05 |
| `internal/api/handler/action.go` | `HandleExecute`/`HandleStatus`: 验证错误使用 `writeValidationError()` | ERR-05 |
| `internal/api/handler/outbox.go` | `HandleDispatch`/`HandleCancel`/`HandleGetDetail`: 空 ID 校验使用 `writeValidationError()` | ERR-05 |
| `internal/api/handler/pipeline.go` | `HandleRun`: 空 config 校验使用 `writeValidationError()` | ERR-05 |
| `internal/api/handler/review.go` | `handleReviewAction`: 空 reviewer_id 校验使用 `writeValidationError()` | ERR-05 |
| `internal/api/handler/handler_sandbox.go` | `HandleCreate`/`HandleAddProposal`: 空字段校验使用 `writeValidationError()` | ERR-05 |

### Not-Found 增强

| 文件 | 修改内容 | 对应需求 |
|------|---------|---------|
| `internal/api/handler/handler_sandbox.go` | 字符串匹配错误改为使用类型化错误 + `errors.Is()` | ERR-01 |
| `internal/api/handler/action.go` | `HandleStatus`: 添加 `pgx.ErrNoRows` 检测 | ERR-01 |

### 数据库 503 增强

| 文件 | 修改内容 | 对应需求 |
|------|---------|---------|
| `internal/api/handler/error_helper.go` | 添加 `isDatabaseConnectionError()` 函数 + `writeDatabaseError()` | ERR-04 |
| 所有 handler 中的 `default:` 分支 | 替换 `writeError(w, r, 500, INTERNAL_ERROR, ...)` 为 `writeDatabaseError()` | ERR-04 |

### 测试文件

| 文件 | 修改内容 |
|------|---------|
| `internal/api/middleware/error_test.go` | 添加 `SERVICE_UNAVAILABLE`, `CONFLICT` 测试；添加带 `details` 的测试 |
| `internal/api/handler/error_helper_test.go` | 添加验证错误测试、DB 错误分类测试、冲突测试 |

## Sources

### Primary (HIGH confidence)
- [VERIFIED: codebase] `internal/api/middleware/error.go` - 现有错误结构体和写错误函数
- [VERIFIED: codebase] `internal/api/handler/error_helper.go` - 现有错误分类和便捷包装
- [VERIFIED: codebase] `internal/api/handler/action.go:65` - JSON 解码静默忽略
- [VERIFIED: codebase] `internal/api/handler/outbox.go:195-198` - JSON 解码静默忽略
- [VERIFIED: codebase] `internal/api/handler/handler_sandbox.go:112,135` - 字符串匹配错误
- [VERIFIED: codebase] `internal/api/handler/decision.go:379` - 503 使用 INTERNAL_ERROR 错误码
- [VERIFIED: codebase] `internal/api/handler/outbox.go:90,121` - 409 使用 BAD_REQUEST 错误码

### Secondary (MEDIUM confidence)
- [CITED: pkg.go.dev/github.com/jackc/pgx/v5] pgx 错误类型参考

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - 所有错误处理基础设施在代码库中已验证
- Architecture: HIGH - 每个 handler 的错误模式在代码库中已验证
- Pitfalls: HIGH - 所有已知问题在代码库中直接观察到

**Research date:** 2026-06-03
**Valid until:** 2026-07-03 (30 天 - Go 标准库和 pgx 稳定)
