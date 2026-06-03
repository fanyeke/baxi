# Phase 05: Security Hardening - Context

**Gathered:** 2026-06-03
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 5 修复 CORS 中间件的 scheme 验证缺失。其他安全需求（JWT/token 轮换、Docker Compose 凭据）因项目为本地演示程序而跳过。

**在范围内（从 REQUIREMENTS.md）：**
- SEC-02: CORS origin 检查验证 scheme（http vs https）

**不在范围内：**
- SEC-01: Auth 中间件 token 轮换或 JWT——本地演示程序不需要。API_BEARER_TOKEN 模式保持不变
- SEC-03: Docker Compose 凭据管理——本地开发环境，写死凭据可接受
- SSL/TLS 配置——无公共网络部署
- 速率限制或多租户身份验证

</domain>

<decisions>
## Implementation Decisions

### SEC-02: CORS Scheme 验证
- **D-01:** 使用 `url.Parse()` 解析 Origin 头，精确比较 `scheme + host + port`（含端口标准化处理）。
- **D-02:** 保持 `CORS_ALLOWED_ORIGINS` 的现有逗号分隔格式不变。只修改验证逻辑。
- **Rationale:** 最小改动原则——不改配置格式，只加固验证逻辑。

### SEC-01 和 SEC-03
- **D-03:** SEC-01（JWT/token 轮换）跳过。项目声明"单 bearer token 对演示来说足够了"。
- **D-04:** SEC-03（Docker Compose 凭据）跳过。本地开发环境硬编码凭据可接受。
- **Rationale:** 项目是本地演示/测试程序，安全攻击面极小。

### the agent's Discretion
- `url.Parse` 返回结果中端口字段为空时的默认行为（使用 scheme 默认端口 80/443）
- URL 解析失败时的处理方式（记录错误并拒绝请求 vs 允许请求）
- 具体的测试方法

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### CORS Middleware（需修改）
- `internal/api/middleware/cors.go` — 需要添加 scheme 验证逻辑
- `docker-compose.yml` — 参考 CORS 来源列表格式

### 参考
- `.planning/codebase/CONCERNS.md` §CORS Origin Check Does Not Validate Scheme — 问题详细描述
- `.planning/REQUIREMENTS.md` §SEC-02 — 需求定义
- `.planning/ROADMAP.md` §Phase 5 — 阶段目标和成功标准

### 已确认无需修改
- `internal/api/middleware/auth.go` — SEC-01 跳过，不修改
- `docker-compose.yml` — SEC-03 跳过，不修改
- `frontend/src/api/client.ts` — 前端 token 处理不变

</canonical_refs>

<code_context>
## Existing Code Insights

### 需要修改的文件
- `internal/api/middleware/cors.go` — `parseOrigins()` 和 `originAllowed()` 函数

### 现有模式
- CORS 中间件使用 `strings.Split` 解析来源列表（在 Phase 2 已修复空格问题）
- `originAllowed()` 使用 `strings.EqualFold` 做不区分大小写的域名比较

### 集成点
- cors.go 是 chi 中间件链的一部分，在 `internal/api/middleware/middleware.go` 中注册
- 修改后影响所有 API 路由

</code_context>

<specifics>
## Specific Ideas

- 用 `net/url` 解析 Origin 头，提取 scheme+host+port
- 端口标准化：Origin 为 `http://example.com`（无端口）时默认 port=80；`https://` 默认 port=443
- `parseOrigins()` 中存储已解析的 URL 结构体，避免每次请求都重新解析
- URL 解析失败时建议拒绝请求（fail closed），不跳过验证

</specifics>

<deferred>
## Deferred Ideas

- SEC-01（JWT/token 轮换）——若部署到非本地环境再实现
- SEC-03（Docker Compose 凭据）——若公开部署再处理
- SSL/TLS 强制——无计划
- 速率限制——超出当前范围

</deferred>

---

*Phase: 05-Security-Hardening*
*Context gathered: 2026-06-03*
