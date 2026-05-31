# Baxi API 接口参考手册

> **版本**: v1.0 · **最后更新**: 2026-05-31
> **技术栈**: Go 1.23 · chi v5 · pgx v5 · PostgreSQL 16
> **默认端口**: 8080 · **基础路径**: `/api/v1`

---

## 概览

Baxi API 是电商治理决策平台的后端网关，使用 Go chi 路由框架，pgx 连接 PostgreSQL 16。所有端点位于 `/api/v1` 下，通过 8080 端口提供服务。

### 分层架构

```
Client (React console / CLI / curl / MCP)
   │  HTTP + Bearer Token
   ▼
┌──────────────────────────────┐
│  chi Router (:8080)          │
│  ├─ RequestID Middleware     │
│  ├─ Logger (zap)             │
│  ├─ Recovery (panic→500)     │
│  ├─ Timeout (30s)            │
│  ├─ CORS                     │
│  └─ Auth (BEARER)            │
├──────────────────────────────┤
│  Handlers (15 个)            │
│  dto → handler → service     │
├──────────────────────────────┤
│  Services (业务逻辑层)       │
├──────────────────────────────┤
│  Repository (pgx + pool)     │
├──────────────────────────────┤
│  PostgreSQL 16               │
└──────────────────────────────┘
```

### 端点总览（50+ 端点）

| 分组 | 端点数 | 认证 |
|------|--------|------|
| Health | 1 | ❌ |
| Status / Alerts / Tasks | 3 | ✅ |
| Outbox | 4 | ✅ |
| Logs / Agent Logs | 6 | ✅ |
| Governance | 7 | ✅ |
| Feishu | 3 | ✅ |
| Pipeline | 1 | ✅ |
| LLM | 2 | ✅ |
| Decisions | 11 | ✅ |
| Proposals / Review | 4 | ✅ |
| Action | 2 | ✅ |
| Sandbox | 5 | ✅ |
| Qoder | 2 | ✅ |

所有路由定义见 `internal/api/routes.go`。

---

## 2. 认证与授权

### 认证方式

**HTTP Bearer Token**，通过 `Authorization` 请求头传递。

```
Authorization: Bearer <token>
```

校验流程：

1. 从 `Authorization` 头提取 `Bearer <token>`
2. 使用 `crypto/subtle.ConstantTimeCompare` 常量时间比较
3. Token 最短 32 字符，拒绝已知弱 token（`test-token`、`changeme` 等）
4. 通过 → 请求继续；失败 → 401 `UNAUTHORIZED`

### 端点认证要求

| 分组 | 需要认证 |
|------|----------|
| Health (`/api/v1/health`) | ❌ 不需要 |
| 其他所有端点 | ✅ 需要 |

### 示例

```bash
# ✅ 正确
curl -H "Authorization: Bearer $API_BEARER_TOKEN" \
  http://127.0.0.1:8080/api/v1/status

# ❌ 缺少头 → 401 UNAUTHORIZED
curl http://127.0.0.1:8080/api/v1/status
```

### 注意事项

- 当前使用单一静态 token，所有请求视为同一 actor
- 不支持多用户 / RBAC（API 层未启用角色检查）
- Token 通过 `API_BEARER_TOKEN` 环境变量配置

---

## 3. 统一错误格式

所有错误返回 5 字段 JSON 结构，定义于 `internal/api/middleware/error.go`。

```json
{
  "request_id": "req_abc123_def456",
  "error_code": "UNAUTHORIZED",
  "message": "Missing Authorization header",
  "diagnosis": "Request must include an Authorization: Bearer <token> header",
  "suggested_action": "Provide a valid API bearer token in the Authorization header"
}
```

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `request_id` | string | 请求 ID，全链路追踪用 |
| `error_code` | string | 机器可读错误码 |
| `message` | string | 人类可读描述 |
| `diagnosis` | string | 问题诊断 |
| `suggested_action` | string | 建议的修复动作 |

### 请求 ID 追踪

每个请求都有唯一 `X-Request-ID`：

- 客户端可传入 `X-Request-ID` 头（建议 UUID）
- 未传入时服务端生成，格式 `req_<ts>_<8chr>`
- 响应头包含相同 `X-Request-ID`
- 所有错误响应和日志均携带

```bash
curl -H "X-Request-ID: req-001" \
     -H "Authorization: Bearer $TOKEN" \
     http://127.0.0.1:8080/api/v1/alerts
```

### 分页

支持所有列表端点。通过 `limit` 和 `offset` 查询参数控制，响应包含分页元数据。

**请求参数**：

| 参数 | 类型 | 默认 | 范围 | 说明 |
|------|------|------|------|------|
| `limit` | int | 100 | 1~1000 | 每页记录数 |
| `offset` | int | 0 | >= 0 | 偏移量 |

**响应格式**：

```json
{
  "items": [],
  "pagination": {
    "limit": 100,
    "offset": 0,
    "total": 230
  }
}
```

部分旧端点（alerts、tasks、outbox）使用向后兼容的 `{"items": [...], "total": N}` 格式。

### 中间件栈

全部定义在 `internal/api/middleware/` 中：

| 中间件 | 文件 | 说明 |
|--------|------|------|
| RequestID | `request_id.go` | 生成/传播请求 ID |
| RealIP | chi 内置 | 从 X-Forwarded-For 读取真实 IP |
| Logger | chi 内置 | zap 访问日志 |
| Recovery | `error.go` | panic 捕获 → 500 错误响应 |
| Timeout | chi 内置 | 30 秒超时 |
| CORS | `cors.go` | 按 `CORS_ALLOWED_ORIGINS` 配置 |
| Auth | `auth.go` | Bearer Token 校验 |

---

## 4. 端点参考

以下按分组列出所有端点。`internal/api/handler/` 中每个文件对应一个分组。

---

## 5. 公共端点

### `GET /api/v1/health`

系统健康检查，无需认证，用于负载均衡探针。

**响应**：

```json
{
  "status": "ok",
  "version": "1.0",
  "db_connected": true
}
```

**参考**: `internal/api/handler/status.go`

---

## 6. 状态与告警

### `GET /api/v1/status`

系统状态：数据库表计数、最近管道运行、版本号。

**Query 参数**：无

**参考**: `internal/api/handler/status.go` · `internal/api/dto/status.go`

### `GET /api/v1/alerts`

告警事件列表，支持筛选。

**Query 参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| `severity` | string | 筛选：`low` / `medium` / `high` / `critical` |
| `status` | string | 筛选：`new` / `investigating` / `resolved` / `ignored` |
| `object_type` | string | 筛选：`global` / `seller` / `category` / `region` |
| `rule_id` | string | 按规则 ID 筛选 |
| `limit` | int | 1~1000，默认 100 |
| `offset` | int | 偏移量 |

**参考**: `internal/api/handler/alerts.go` · `internal/api/dto/alert.go`

---

## 7. 任务与发件箱

### `GET /api/v1/tasks`

任务列表，支持状态、优先级、负责人筛选。

**Query 参数**：`status`、`priority`、`owner`、`limit`、`offset`

**参考**: `internal/api/handler/tasks.go` · `internal/api/dto/task.go`

### `GET /api/v1/outbox`

发件箱列表（待调度事件队列）。

**Query 参数**：`status`、`channel`、`event_type`、`limit`、`offset`

### `GET /api/v1/outbox/{id}`

发件箱单条详情。

### `POST /api/v1/outbox/dispatch`

批量调度：从发件箱取出 pending 事件，通过对应 adapter 发到目标渠道。

**请求体**：

```json
{
  "channel": "feishu_message",
  "limit": 100,
  "apply": false
}
```

- `channel`: 筛选渠道（可选）
- `limit`: 最多处理条数，1~1000
- `apply`: false = 预览，true = 实际执行

### `POST /api/v1/outbox/{id}/dispatch`

单条发件箱调度。

### `POST /api/v1/outbox/{id}/cancel`

取消发件箱事件。

**参考**: `internal/api/handler/outbox.go` · `internal/api/dto/outbox.go`

---

## 8. 日志与诊断

### `GET /api/v1/logs/recent`

最近请求日志。

**Query 参数**：`limit`、`offset`

### `GET /api/v1/logs/errors`

错误日志。

**Query 参数**：`request_id`、`error_code`、`limit`、`offset`

### `GET /api/v1/logs/audit`

审计日志。

**Query 参数**：`outbox_id`、`status`、`source`、`limit`、`offset`

### `GET /api/v1/logs/diagnosis`

请求诊断：根据 `request_id` 聚合错误日志 + 审计记录。

**Query 参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `request_id` | string | ✅ | 要诊断的请求 ID |

**参考**: `internal/api/handler/diagnosis.go` · `internal/api/dto/diagnosis.go`

### `GET /api/v1/logs/agent`

Agent 操作日志列表。

**Query 参数**：`agent_id`、`action`、`status`、`limit`、`offset`

### `POST /api/v1/logs/agent`

创建 Agent 操作日志。

**请求体**：

```json
{
  "agent_id": "pi-agent-1",
  "action": "read_status",
  "status": "success",
  "details": {}
}
```

**参考**: `internal/api/handler/agent_logs.go`

---

## 9. 治理

治理端点从数据库读取数据分类、血缘、标记、检查点规则和健康检查结果。

| 端点 | 参考 DTO |
|------|----------|
| `GET /governance/status` | `GovernanceStatusResponse` |
| `GET /governance/catalog` | `CatalogResponse` |
| `GET /governance/classification` | `ClassificationResponse` |
| `GET /governance/markings` | `FieldMarkingResponse` |
| `GET /governance/lineage` | `LineageResponse` |
| `GET /governance/checkpoints` | `CheckpointsResponse` |
| `GET /governance/health` | `HealthChecksResponse` |

**参考**: `internal/api/handler/governance.go` · `internal/api/dto/governance.go`

---

## 10. 飞书集成

三个端点共享相同的请求结构。

### `POST /api/v1/feishu/export`

导出数据为 CSV（准备导入飞书多维表格）。

### `POST /api/v1/feishu/sync`

将数据同步到飞书多维表格。

### `POST /api/v1/feishu/status/import`

从飞书拉取任务状态/反馈回流到本地数据库。

**请求体**：

```json
{
  "tables": ["alert_events", "action_tasks"],
  "apply": false
}
```

- `tables`: 可选，不传则操作所有配置的表（最多 20 项）
- `apply`: false = 预览，true = 实际执行

**参考**: `internal/api/handler/feishu.go`

---

## 11. 决策引擎

### `POST /api/v1/decisions/cases`

创建决策案例。

**请求体**：

```json
{
  "source_type": "alert",
  "source_id": "evt_abc123"
}
```

### `GET /api/v1/decisions/cases`

案例列表。

**Query 参数**：`source_type`、`status`、`severity`、`limit`、`offset`

### `GET /api/v1/decisions/cases/{case_id}`

案例详情。

### `POST /api/v1/decisions/cases/{case_id}/context`

构建决策上下文。无请求体，触发上下文组装。

### `POST /api/v1/decisions/cases/{case_id}/decide`

执行规则决策。生成建议列表。

### `GET /api/v1/decisions/cases/{case_id}/proposals`

案例的建议列表。

### `POST /api/v1/decisions/cases/{case_id}/decide/llm`

使用 LLM 辅助决策。

### `POST /api/v1/decisions/cases/{case_id}/compare`

比较两种决策方案。

### `POST /api/v1/decisions/cases/{case_id}/replay`

重放决策过程。

### `GET /api/v1/decisions/cases/{case_id}/llm-decisions`

LLM 决策记录列表。

### `GET /api/v1/decisions/cases/{case_id}/evals`

决策评估结果列表。

**参考**: `internal/api/handler/decision.go` · `internal/api/dto/decision.go`

---

## 12. 审批与执行

### `POST /api/v1/proposals/{id}/approve`

审批通过提案。

### `POST /api/v1/proposals/{id}/reject`

驳回提案。

### `POST /api/v1/proposals/{id}/cancel`

取消提案。

### `GET /api/v1/proposals/{id}/review`

获取提案审核详情。

### `POST /api/v1/proposals/{id}/execute`

执行提案。

### `GET /api/v1/proposals/{id}/status`

查询执行状态。

**参考**: `internal/api/handler/review.go` · `internal/api/handler/action.go`

---

## 13. 沙盘模拟

### `POST /api/v1/sandboxes`

创建沙盘。

**请求体**：

```json
{
  "case_id": "case_abc123",
  "data": {}
}
```

### `GET /api/v1/sandboxes`

沙盘列表。

### `GET /api/v1/sandboxes/compare`

沙盘对比。

**Query 参数**：`sandbox_1`、`sandbox_2`

### `GET /api/v1/sandboxes/{id}`

沙盘详情。

### `POST /api/v1/sandboxes/{id}/proposals`

向沙盘添加提案。

**请求体**：

```json
{
  "proposal_id": "prop_xyz789"
}
```

**参考**: `internal/api/handler/handler_sandbox.go` · `internal/api/dto/sandbox.go`

---

## 14. 管道

### `POST /api/v1/pipeline/run`

运行数据管道。

**请求体**：

```json
{
  "config": "daily"
}
```

- `config`: 管道配置名称，如 `ingest_raw`、`full`、`daily`

**响应**：

```json
{
  "run_id": "run_20260531_001",
  "status": "started"
}
```

**参考**: `internal/api/handler/pipeline.go` · `internal/api/dto/pipeline.go`

---

## 15. Qoder

### `GET /api/v1/qoder/capabilities`

Qoder 能力矩阵。返回操作模式、协议版本和权限标志。

```json
{
  "mode": "read_only",
  "version": "0.6.0",
  "can_read_status": true,
  "can_read_alerts": true,
  "can_read_tasks": true,
  "can_read_outbox": true,
  "can_read_governance": true,
  "can_read_logs": true,
  "can_write_reports": false,
  "can_execute_actions": false
}
```

**参考**: `internal/api/handler/qoder.go` · `internal/api/dto/qoder.go`

### `GET /api/v1/qoder/context`

聚合系统上下文，供 AI Agent 决策使用。返回系统状态、汇总计数、告警、任务、发件箱等。

**Query 参数**：

| 参数 | 类型 | 默认 | 范围 | 说明 |
|------|------|------|------|------|
| `severity` | string | — | — | 筛选告警级别 |
| `limit_alerts` | int | 10 | 0~100 | 最大告警数 |
| `limit_tasks` | int | 10 | 0~100 | 最大任务数 |
| `limit_outbox` | int | 10 | 0~100 | 最大发件箱数 |
| `include_logs` | bool | false | — | 是否附加最近错误诊断 |

```json
{
  "request_id": "ctx_a1b2c3d4",
  "system": {
    "last_pipeline_run": { "...": "..." }
  },
  "summary": {
    "total_alerts": 8,
    "total_open_tasks": 5,
    "total_pending_outbox": 3
  },
  "top_alerts": [],
  "open_tasks": [],
  "pending_outbox": [],
  "ontology": {},
  "governance": {},
  "agent_policy": {}
}
```

---

## 16. LLM

### `GET /api/v1/llm/status`

LLM 配置状态。

```json
{
  "enabled": false,
  "provider": "",
  "model": "",
  "fallback_enabled": false,
  "raw_output_storage": false
}
```

### `GET /api/v1/llm/metrics`

LLM 调用指标。

**参考**: `internal/api/handler/llm.go`

---

## 17. 前端集成

React 控制台位于 `frontend/` 目录，通过同一端口 `/console` 提供静态服务。

```typescript
const BASE_URL = "http://127.0.0.1:8080/api/v1";

const res = await fetch(`${BASE_URL}/status`, {
  headers: {
    Authorization: `Bearer ${import.meta.env.VITE_API_TOKEN}`,
    "Content-Type": "application/json",
    "X-Request-ID": crypto.randomUUID(),
  },
});
```

### TanStack Query 示例

```tsx
function useAlerts(severity?: string) {
  return useQuery({
    queryKey: ["alerts", severity],
    queryFn: () => {
      const params = severity ? `?severity=${severity}` : "";
      return api(`/alerts${params}`);
    },
  });
}
```

### 错误处理

```tsx
if (!res.ok) {
  const err = await res.json();
  // err = { request_id, error_code, message, diagnosis, suggested_action }
  throw new Error(err.message);
}
```
