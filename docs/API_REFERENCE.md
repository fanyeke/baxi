# Olist Decision Backend — API 接口参考手册

> **版本**: v0.5.3 · **最后更新**: 2026-05-24  
> **技术栈**: Python 3.9+ · FastAPI · Pydantic v2 · SQLite (WAL)  
> **默认端口**: 8765 · **基础路径**: `/api/v1`

---

## 目录

1. [概览与架构](#1-概览与架构)
2. [快速开始](#2-快速开始)
3. [认证与授权](#3-认证与授权)
4. [横切关注点](#4-横切关注点)
5. [统一错误格式](#5-统一错误格式)
6. [完整端点参考](#6-完整端点参考)
7. [请求与响应 Schema 详解](#7-请求与响应-schema-详解)
8. [运维指导](#8-运维指导)
9. [前端集成指南](#9-前端集成指南)
10. [附录：数据表参考](#10-附录数据表参考)

---

## 1. 概览与架构

### 1.1 系统定位

Baxi API 是 **Olist 巴西电商决策沙盘** 的后端网关，对外提供：
- 业务数据只读查询（订单、告警、任务、指标）
- 飞书/外部渠道的事件调度（Outbox 模式）
- 飞书多维表格双向同步
- 数据治理与诊断
- 数据管道预览

### 1.2 分层架构

```
Client (React console / CLI / curl)
   │  HTTP + Bearer Token
   ▼
┌──────────────────────────────┐
│  FastAPI Gateway (8765)      │
│  ├─ Middleware: Request ID   │
│  ├─ Middleware: Security Hdr │
│  ├─ Middleware: Rate Limit   │
│  └─ Exception → Structured   │
├──────────────────────────────┤
│  Routers (10 个)             │
│  health / status / alerts /  │
│  tasks / outbox / logs /     │
│  feishu / pipeline /         │
│  diagnosis / governance      │
├──────────────────────────────┤
│  Services (业务逻辑层)       │
│  db / alert / task /         │
│  dispatch / feishu / status  │
│  pipeline / log_reader /     │
│  diagnosis                   │
├──────────────────────────────┤
│  Adapters (渠道适配层)       │
│  Feishu / GitHub / LocalCLI  │
│  / Manual                    │
├──────────────────────────────┤
│  SQLite (olist_ops.db, WAL)  │
│  + 飞书多维表格              │
└──────────────────────────────┘
```

### 1.3 端点总览（21 个端点）

| # | 方法 | 路径 | 分组 | 认证 | 作用 |
|---|------|------|------|------|------|
| 1 | GET | `/api/v1/health` | Health | ❌ | 健康检查 |
| 2 | GET | `/api/v1/status` | Status | ✅ | 系统状态 |
| 3 | GET | `/api/v1/alerts` | Alerts | ✅ | 告警列表 |
| 4 | GET | `/api/v1/tasks` | Tasks | ✅ | 任务列表 |
| 5 | GET | `/api/v1/outbox` | Outbox | ✅ | 发件箱列表 |
| 6 | POST | `/api/v1/outbox/dispatch` | Outbox | ✅ | 执行调度 |
| 7 | GET | `/api/v1/logs/errors` | Logs | ✅ | 错误日志 |
| 8 | GET | `/api/v1/logs/audit` | Logs | ✅ | 审计日志 |
| 9 | GET | `/api/v1/logs/recent` | Logs | ✅ | 最近请求 |
| 10 | POST | `/api/v1/feishu/export` | Feishu | ✅ | 导出 CSV |
| 11 | POST | `/api/v1/feishu/sync` | Feishu | ✅ | 同步到飞书 |
| 12 | POST | `/api/v1/feishu/status/import` | Feishu | ✅ | 从飞书回流 |
| 13 | POST | `/api/v1/pipeline/run` | Pipeline | ✅ | 管道预览 |
| 14 | GET | `/api/v1/logs/diagnosis` | Logs | ✅ | 请求诊断 |
| 15 | GET | `/api/v1/governance/catalog` | Governance | ✅ | 数据目录 |
| 16 | GET | `/api/v1/governance/classification` | Governance | ✅ | 数据分类 |
| 17 | GET | `/api/v1/governance/markings` | Governance | ✅ | 数据标记 |
| 18 | GET | `/api/v1/governance/lineage` | Governance | ✅ | 数据血缘 |
| 19 | GET | `/api/v1/governance/checkpoints` | Governance | ✅ | 检查点规则 |
| 20 | GET | `/api/v1/governance/health` | Governance | ✅ | 健康检查规则 |
| 21 | GET | `/api/v1/governance/status` | Governance | ✅ | 治理状态汇总 |

---

## 2. 快速开始

### 2.1 安装依赖

```bash
cd /path/to/baxi
python3 -m venv venv && source venv/bin/activate
pip install -e .
# 或按 pyproject.toml 手动安装
```

### 2.2 配置环境变量

复制 `config/feishu_table_ids.yml.example` 到 `config/feishu_table_ids.yml` 并填入真实飞书令牌。

在项目根目录创建 `.env` 文件：

```bash
# 必填：API 访问令牌
API_BEARER_TOKEN=your-secure-random-token-here

# 可选：飞书凭据（不配置则飞书操作使用 dry-run 默认值）
FEISHU_APP_ID=cli_xxx
FEISHU_APP_SECRET=xxx
FEISHU_BASE_APP_TOKEN=xxx
FEISHU_CHAT_ID=oc_xxx

# 可选：启用 OpenAPI 文档
ENABLE_DOCS=1

# 可选：CORS 允许的来源（默认 http://localhost:5173）
CORS_ORIGINS=http://localhost:5173,http://localhost:5174

# 可选：调试模式（500 错误会显示堆栈信息）
DEBUG=1
```

快速生成安全 token：

```bash
python3 -c "import secrets; print(secrets.token_urlsafe(32))"
```

### 2.3 启动服务

```bash
python3 scripts/run_api.py --port 8765
# 或直接用 uvicorn
uvicorn api.main:app --host 0.0.0.0 --port 8765
```

### 2.4 验证

```bash
# 健康检查（无需认证）
curl http://127.0.0.1:8765/api/v1/health
# → {"status":"ok","version":"0.5.1","db_connected":true}

# OpenAPI 文档（需 ENABLE_DOCS=1）
open http://127.0.0.1:8765/docs
```

---

## 3. 认证与授权

### 3.1 认证方式

**HTTP Bearer Token**，通过 `Authorization` 请求头传递。

```
Authorization: Bearer <token>
```

校验流程：
1. 从 `Authorization` 头提取 `Bearer <token>`
2. 使用 **常量时间比较**（`hmac.compare_digest`）与 `API_BEARER_TOKEN` 环境变量比对
3. 通过 → 请求继续；失败 → 返回 401 或 403

### 3.2 各端点认证要求

| 分组 | 需要认证 |
|------|----------|
| Health | ❌ 不需要 |
| Status / Alerts / Tasks / Outbox / Logs / Feishu / Pipeline / Diagnosis / Governance | ✅ 全部需要 |

### 3.3 示例

```bash
# ✅ 正确
curl -H "Authorization: Bearer $API_BEARER_TOKEN" \
  http://127.0.0.1:8765/api/v1/status

# ❌ 缺少头 → 401 AUTH_REQUIRED
curl http://127.0.0.1:8765/api/v1/status

# ❌ token 错误 → 403 INVALID_TOKEN
curl -H "Authorization: Bearer wrong-token" \
  http://127.0.0.1:8765/api/v1/status
```

### 3.4 注意事项

- 当前版本 (`v0.5.3`) 使用**单一静态 token**，所有请求被视为同一 actor（`qoder`）
- 不支持多用户 / RBAC（即使配置文件中有 `access_policy.yml`，API 层未启用）
- Token 不应硬编码在代码或配置文件中，应通过环境变量传递

---

## 4. 横切关注点

### 4.1 Request ID 追踪

**每个请求都有唯一 `X-Request-ID`** 用于端到端追踪。

- 客户端可传入 `X-Request-ID` 头（建议 UUID）
- 未传入时服务端生成 UUID
- 响应头包含相同 `X-Request-ID`
- 所有错误响应和日志均携带

```bash
curl -H "X-Request-ID: req-001" \
     -H "Authorization: Bearer $TOKEN" \
     http://127.0.0.1:8765/api/v1/alerts
# 响应头包含: X-Request-ID: req-001
```

### 4.2 速率限制

**Token-bucket 算法**，按 `(源 IP, rate_class)` 限流。

| 端点类别 | 路径前缀 | 限制（窗口 60s） |
|----------|---------|------------------|
| `health` | `/api/v1/health` | 30 次/分钟 |
| `dispatch` | `/api/v1/outbox/dispatch` | 30 次/分钟 |
| `pipeline` | `/api/v1/pipeline/run` | 10 次/分钟 |
| 其他 | — | 300 次/分钟 |

超出限制 → **HTTP 429** `RATE_LIMITED`

```json
{
  "request_id": "abc123",
  "error_code": "RATE_LIMITED",
  "message": "Too many requests. Please slow down.",
  "diagnosis": "Rate limit exceeded for this endpoint",
  "suggested_action": "Wait before retrying"
}
```

**实现说明**：
- 内存存储（进程重启后清零），不适合多 worker 部署
- OPTIONS 请求不计入限流
- 支持反向代理：通过 `X-Forwarded-For` 读取真实 IP（需配置 `TRUSTED_PROXY_IPS` 环境变量）

### 4.3 安全响应头

所有响应都会包含以下安全头：

| Header | 值 | 作用 |
|--------|-----|------|
| `X-Content-Type-Options` | `nosniff` | 防止 MIME 嗅探 |
| `X-Frame-Options` | `DENY` | 禁止 iframe 嵌入 |
| `Strict-Transport-Security` | `max-age=31536000; includeSubDomains` | 强制 HTTPS |
| `X-XSS-Protection` | `1; mode=block` | 旧浏览器 XSS 防护 |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | 限制 Referrer 泄露 |
| `Content-Security-Policy` | 视路径不同 | 防止 XSS / 数据注入 |

对于 `/docs`、`/redoc`、`/openapi.json` 路径，CSP 会放宽允许 jsdelivr.net 和 unpkg.com CDN。

### 4.4 CORS

默认允许 `http://localhost:5173`（vite 开发服务器），通过 `CORS_ORIGINS` 环境变量配置多个来源（逗号分隔）。

```bash
CORS_ORIGINS=http://localhost:5173,https://console.example.com
```

允许的方法：`GET`、`POST`、`OPTIONS`  
允许的请求头：`Authorization`、`Content-Type`、`X-Request-ID`

---

## 5. 统一错误格式

**所有错误都返回这个统一结构**，前端可以统一处理。

```json
{
  "request_id": "abc-123-def",
  "error_code": "AUTH_REQUIRED",
  "message": "Authorization header is required",
  "diagnosis": "No Bearer token provided",
  "suggested_action": "Add 'Authorization: Bearer <token>' header"
}
```

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `request_id` | string | 请求 ID，用于诊断日志关联 |
| `error_code` | string | 机器可读错误码（见下表） |
| `message` | string | 人类可读错误描述 |
| `diagnosis` | string | 问题诊断（说明为什么出错） |
| `suggested_action` | string | 建议的修复动作 |

### 错误码速查

| HTTP 状态 | error_code | 触发场景 |
|----------|------------|----------|
| 401 | `AUTH_REQUIRED` | 缺少 Authorization 头，或格式错误 |
| 403 | `INVALID_TOKEN` | token 与 `API_BEARER_TOKEN` 不匹配 |
| 404 | `NOT_FOUND` | 诊断请求中的 `request_id` 在日志中找不到 |
| 429 | `RATE_LIMITED` | 速率限制超出 |
| 422 | `VALIDATION_ERROR` | 请求体 Pydantic 校验失败 |
| 503 | `DB_MISSING` | SQLite DB 文件不存在（需先运行数据管道） |
| 503 | `DB_UNAVAILABLE` | DB 无法连接 |
| 503 | `DB_QUERY_FAILED` | SQL 查询失败 |
| 500 | `INTERNAL_ERROR` | 未捕获的异常 |
| 500 | `CONFIG_MISSING` | 必需配置文件缺失 |
| 500/400 | `DISPATCH_FAILED` | 调度执行失败 |
| 500/400 | `FEISHU_DISPATCH_FAILED` | 飞书操作失败 |

### 错误码分类建议

- **4xx**：客户端问题（参数错误 / 认证失败 / 限流）→ 修复客户端代码
- **5xx**：服务端问题 → 查 `diagnosis` 和 `suggested_action`，或联系管理员
- **503**：依赖不可用 → 运行数据管道初始化

---

## 6. 完整端点参考

### 6.1 Health 分组

#### `GET /api/v1/health`

系统健康检查，**无需认证**，用于负载均衡/探针。

**请求参数**：无

**响应** — `HealthResponse`：
```json
{
  "status": "ok",
  "version": "0.5.1",
  "db_connected": true
}
```

**字段说明**：
- `status`: 字符串，固定为 `"ok"`
- `version`: API 版本号
- `db_connected`: SQLite DB 文件是否存在

**示例**：
```bash
curl http://127.0.0.1:8765/api/v1/health
```

---

### 6.2 Status 分组

#### `GET /api/v1/status`

系统状态（DB 表计数 + 最近一次管道运行 + 迁移状态）。

**请求参数**：无

**响应** — `StatusResponse`：
```json
{
  "database": {
    "path": "data/olist_ops.db",
    "exists": true,
    "tables": {
      "pipeline_runs": 42,
      "ingestion_batches": 634,
      "dwd_order_level": 99441,
      "dwd_item_level": 112650,
      "metric_daily": 634,
      "metric_dimension_daily": 5400,
      "alert_events": 230,
      "strategy_recommendations": 85,
      "action_tasks": 120,
      "review_retro": 15,
      "event_outbox": 45,
      "qoder_jobs": 8
    }
  },
  "last_pipeline_run": {
    "run_id": "run_20260524_001",
    "run_type": "full",
    "mode": "full",
    "status": "completed",
    "started_at": "2026-05-24T01:00:00",
    "finished_at": "2026-05-24T01:05:23",
    "input_count": 99441,
    "output_count": 230,
    "error_message": null
  },
  "version": "0.5.1",
  "migration_status": {
    "status": "ok",
    "failed": []
  }
}
```

**示例**：
```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:8765/api/v1/status
```

---

### 6.3 Alerts 分组

#### `GET /api/v1/alerts`

告警事件列表（来自规则引擎评估结果）。

**Query 参数**：

| 参数 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `status` | string | 否 | — | 筛选：`new` / `investigating` / `resolved` / `ignored` |
| `severity` | string | 否 | — | 筛选：`low` / `medium` / `high` / `critical` |
| `object_type` | string | 否 | — | 筛选：`global` / `seller` / `category` / `region` |
| `object_id` | string | 否 | — | 筛选具体维度 ID |
| `limit` | int | 否 | 100 | 1~1000 |

**响应** — `AlertListResponse`：
```json
{
  "items": [
    {
      "event_id": "evt_abc123",
      "rule_id": "gmv_drop",
      "event_date": "2026-05-23",
      "severity": "high",
      "metric_name": "gmv",
      "object_type": "global",
      "object_id": "global",
      "current_value": 42500.5,
      "baseline_value": 58000.0,
      "change_rate": -0.267,
      "owner_role": "finance",
      "status": "new",
      "impact_score": 8.5
    }
  ],
  "total": 230
}
```

**典型用法**：
```bash
# 最近 50 条高危告警
curl -H "Authorization: Bearer $TOKEN" \
  "http://127.0.0.1:8765/api/v1/alerts?severity=high&limit=50"

# 某卖家相关告警
curl -H "Authorization: Bearer $TOKEN" \
  "http://127.0.0.1:8765/api/v1/alerts?object_type=seller&object_id=seller_123"
```

---

### 6.4 Tasks 分组

#### `GET /api/v1/tasks`

任务列表（由决策引擎生成）。

**Query 参数**：

| 参数 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `status` | string | 否 | — | `todo` / `in_progress` / `done` / `cancelled` / `blocked` |
| `priority` | string | 否 | — | `low` / `medium` / `high` / `critical` |
| `owner_role` | string | 否 | — | 负责人角色 |
| `limit` | int | 否 | 100 | 1~1000 |

**响应** — `TaskListResponse`：
```json
{
  "items": [
    {
      "task_id": "task_xyz456",
      "task_title": "调查卖家 #123 GMV 异常下跌",
      "task_description": "...",
      "status": "todo",
      "priority": "high",
      "owner_role": "ops",
      "owner_user_id": null,
      "due_at": "2026-05-30T00:00:00",
      "created_at": "2026-05-23T12:00:00",
      "completed_at": null,
      "feedback": null,
      "recommendation_id": "rec_001",
      "event_id": "evt_abc123",
      "target_object_type": "seller",
      "target_object_id": "seller_123"
    }
  ],
  "total": 120
}
```

---

### 6.5 Outbox 分组

#### `GET /api/v1/outbox`

发件箱列表（待调度的事件队列）。

**Query 参数**：

| 参数 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `status` | string | 否 | — | `pending` / `dispatching` / `dispatched` / `skipped` / `failed` |
| `channel` | string | 否 | — | 目标渠道：`feishu_message` / `github_issue` / `local_cli` / `manual` |
| `limit` | int | 否 | 100 | 1~1000 |

**响应** — `OutboxListResponse`：
```json
{
  "items": [
    {
      "outbox_id": "out_abc123",
      "event_type": "alert",
      "source_type": "event",
      "source_id": "evt_abc123",
      "target_channel": "feishu_message",
      "status": "pending",
      "created_at": "2026-05-23T12:00:00",
      "dispatch_attempts": 0,
      "last_dispatch_at": null
    }
  ],
  "total": 45
}
```

---

#### `POST /api/v1/outbox/dispatch`

**执行调度**：从发件箱取出 pending 事件，通过对应 adapter 发到目标渠道。

**请求体** — `DispatchRequest`：
```json
{
  "channel": "feishu_message",  // 可选，筛选渠道
  "limit": 100,                 // 1~1000，默认 100
  "apply": false                // false = dry-run（预览）; true = 真正执行
}
```

**响应** — `DispatchResponse`：
```json
{
  "request_id": "req_xxx",
  "dry_run": true,
  "processed": 3,
  "results": [
    {
      "outbox_id": "out_abc123",
      "status": "preview",
      "adapter_name": "FeishuAdapter",
      "message": "[Alert] gmv_drop: gmv",
      "external_ref": null,
      "error": null
    },
    {
      "outbox_id": "out_def456",
      "status": "failed",
      "adapter_name": null,
      "message": null,
      "external_ref": null,
      "error": "No adapter found for channel: unknown"
    }
  ]
}
```

**状态说明**：
- `preview` (dry-run 模式下)：预览将要发送的内容
- `dispatched`：成功发往目标渠道
- `skipped`：已被其他 worker 抢占，或 adapter 决定跳过
- `failed`：发送失败（最多重试 3 次，超出后置为 `failed` 终态）

**典型用法**：
```bash
# 预览（默认 dry-run，安全）
curl -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"channel":"feishu_message","limit":5}' \
  http://127.0.0.1:8765/api/v1/outbox/dispatch

# 真正执行（谨慎！会对所有匹配事件实际调度）
curl -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"apply":true,"channel":"feishu_message","limit":5}' \
  http://127.0.0.1:8765/api/v1/outbox/dispatch
```

---

### 6.6 Logs 分组

#### `GET /api/v1/logs/errors`

解析 `logs/api/error.log` (JSONL 格式)，返回最近的错误。

**Query 参数**：

| 参数 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `request_id` | string | 否 | — | 按 request_id 过滤 |
| `error_code` | string | 否 | — | 按错误码过滤（应用层过滤） |
| `limit` | int | 否 | 100 | 1~500 |

**响应** — `ErrorLogListResponse`：
```json
{
  "items": [
    {
      "ts": "2026-05-24T01:23:45.123",
      "level": "ERROR",
      "message": "Unhandled exception: division by zero",
      "request_id": "req_xxx",
      "error_code": "INTERNAL_ERROR",
      "diagnosis": "[CREDENTIAL_REDACTED]",  // 凭据自动脱敏
      "suggested_action": "Check server logs",
      "actor": "qoder"
    }
  ],
  "total": 1
}
```

---

#### `GET /api/v1/logs/audit`

解析审计日志 CSV，跟踪所有写操作（dispatch / feishu）。

**Query 参数**：

| 参数 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `outbox_id` | string | 否 | — | 按 outbox_id 过滤 |
| `status` | string | 否 | — | 按状态过滤 |
| `limit` | int | 否 | 100 | 1~500 |
| `source` | string | 否 | `"dispatch"` | `"dispatch"` 或 `"feishu"` |

**响应** — `AuditLogListResponse`：
```json
{
  "items": [
    {
      "timestamp": "2026-05-24T01:23:45.123",
      "outbox_id": "out_abc123",
      "target_channel": "feishu_message",
      "adapter_name": "FeishuAdapter",
      "mode": "apply",
      "status": "dispatched",
      "external_ref": "msg_xxx",
      "error": null,
      "request_id": "req_xxx",
      "source": "api"
    }
  ],
  "total": 45
}
```

---

#### `GET /api/v1/logs/recent`

解析 `logs/api/api.log` (JSONL 格式)，返回最近的请求。

**Query 参数**：

| 参数 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `limit` | int | 否 | 50 | 1~500 |

**响应** — `RecentLogListResponse`：
```json
{
  "items": [
    {
      "ts": "2026-05-24T01:23:45.123",
      "level": "INFO",
      "message": "GET /api/v1/alerts 200 12ms",
      "request_id": "req_xxx",
      "method": "GET",
      "path": "/api/v1/alerts",
      "actor": "qoder"
    }
  ],
  "total": 50
}
```

---

#### `GET /api/v1/logs/diagnosis`

**请求诊断**：根据 `request_id` 聚合 error.log + dispatch 审计 + feishu 审计，给出完整故事线。

**Query 参数**：

| 参数 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `request_id` | string | **是** | — | 要诊断的请求 ID，min_length=1 |

**响应**（成功 200）— `DiagnosisResponse`：
```json
{
  "request_id": "req_xxx",
  "summary": "Dispatch failed: Feishu credentials not configured",
  "error_code": "FEISHU_DISPATCH_FAILED",
  "diagnosis": "FEISHU_APP_ID or FEISHU_APP_SECRET not set",
  "suggested_action": "Configure Feishu credentials in .env",
  "related_logs": [
    {
      "source": "error.log",
      "ts": "2026-05-24T01:23:45.123",
      "error_code": "FEISHU_DISPATCH_FAILED",
      "message": "Feishu send_message failed",
      "diagnosis": "..."
    },
    {
      "source": "audit_dispatch.csv",
      "timestamp": "2026-05-24T01:23:45.123",
      "outbox_id": "out_abc123",
      "status": "failed",
      "error": "credentials not configured"
    }
  ]
}
```

**响应**（未找到 404）：
```json
{
  "request_id": "req_unknown",
  "error_code": "NOT_FOUND",
  "message": "No logs found for request_id: req_unknown",
  "diagnosis": "The request_id was not found in error.log, audit CSV, or Feishu audit CSV.",
  "suggested_action": "Verify the request_id is correct. Logs may have been rotated."
}
```

---

### 6.7 Feishu 分组

#### `POST /api/v1/feishu/export`

将 DB 数据导出为 CSV 文件（准备用于飞书多维表格导入）。

#### `POST /api/v1/feishu/sync`

将数据同步到飞书多维表格。

#### `POST /api/v1/feishu/status/import`

从飞书拉取任务状态/反馈回流到本地 DB。

**三个端点共享相同的请求/响应结构。**

**请求体** — `FeishuExportRequest` / `FeishuSyncRequest` / `FeishuStatusImportRequest`：
```json
{
  "tables": ["alert_events", "action_tasks"],  // 可选，不传则所有配置的表
  "apply": false                                // false = dry-run; true = 真正操作
}
```

**限制**：
- `tables` 最多 20 项
- 表名只允许字母数字和下划线
- 可用表名由 `config/feishu_table_ids.yml` 决定（默认：`daily_metrics`, `alert_events`, `strategy_recommendations`, `action_tasks`, `review_retro`）

**响应** — `FeishuExportResponse` / `FeishuSyncResponse` / `FeishuStatusImportResponse`：
```json
{
  "status": "preview",      // preview | exported | synced | imported | failed | not_configured
  "message": "Dry-run: no files written",
  "tables": [
    {
      "name": "alert_events",
      "status": "preview",
      "rows": 0,
      "file": "",
      "created": 0,
      "updated": 0,
      "pulled": 0,
      "imported": 0,
      "skipped": 0
    }
  ]
}
```

**典型用法**：
```bash
# 预览导出
curl -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}' \
  http://127.0.0.1:8765/api/v1/feishu/export

# 真正同步到飞书（需配好凭据）
curl -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"apply":true,"tables":["alert_events"]}' \
  http://127.0.0.1:8765/api/v1/feishu/sync
```

---

### 6.8 Pipeline 分组

#### `POST /api/v1/pipeline/run`

**预览**数据管道命令（不会执行任何操作）。

**请求体** — `PipelineRunRequest`：
```json
{
  "pipeline_type": "daily"  // daily | full | db_full（默认 daily）
}
```

注意：服务端允许未识别的 `pipeline_type`，会返回带警告的响应（HTTP 200）。

**响应** — `PipelineRunResponse`：
```json
{
  "command": "python3 scripts/run_daily_pipeline.py",
  "pipeline_type": "daily",
  "estimated_duration": "~30 seconds (daily mode)",
  "required_env_vars": [
    "API_BEARER_TOKEN",
    "FEISHU_APP_ID",
    "FEISHU_APP_SECRET",
    "FEISHU_BASE_APP_TOKEN",
    "FEISHU_CHAT_ID"
  ],
  "warnings": [
    "Env var FEISHU_APP_ID not set — Feishu operations will use dry-run defaults"
  ],
  "description": "8-step daily pipeline: ingest → quality → metrics → alerts → AIP → wake → Feishu"
}
```

**预定义管道类型**：

| 类型 | 脚本 | 描述 | 估算时长 |
|------|------|------|----------|
| `daily` | `run_daily_pipeline.py` | 单日 8 步管道 | ~30 秒 |
| `full` | `run_full_pipeline.py` | 全量 5 步评估 | ~5 分钟 |
| `db_full` | `run_db_pipeline.py --mode full --dimensional` | SQLite 全量管道 | ~2 分钟 |

---

### 6.9 Governance 分组

所有 Governance 端点返回 YAML 配置文件的内容（JSON 形式）。

#### `GET /api/v1/governance/catalog`
返回 `config/data_catalog.yml` 内容。

#### `GET /api/v1/governance/classification`
返回 `config/data_classification.yml` 内容（数据分类策略）。

#### `GET /api/v1/governance/markings`
返回 `config/data_markings.yml` 内容（数据标记规则）。

#### `GET /api/v1/governance/lineage`
返回 `config/data_lineage.yml` 内容（数据血缘）。

#### `GET /api/v1/governance/checkpoints`
返回 `config/checkpoint_rules.yml` 内容（检查点规则）。

#### `GET /api/v1/governance/health`
返回 `config/health_checks.yml` 内容（健康检查规则）。

**以上 6 个端点响应** — `GovernanceConfigResponse`：
```json
{
  "assets": [
    {"name": "dwd_order_level", "owner": "data_team", "classification": "internal"}
  ]
  // YAML 中任意字段均通过 extra="allow" 返回
}
```

#### `GET /api/v1/governance/status`

汇总检查 9 个治理配置文件的存在性和可解析性。

**响应** — `GovernanceStatusResponse`：
```json
{
  "governance_layer": "active",
  "configs": {
    "data_catalog.yml": "loaded",
    "data_classification.yml": "loaded",
    "data_markings.yml": "loaded",
    "data_lineage.yml": "loaded",
    "checkpoint_rules.yml": "loaded",
    "retention_policies.yml": "loaded",
    "health_checks.yml": "loaded",
    "decision_eval_rules.yml": "loaded",
    "access_policy.yml": "loaded"
  }
}
```

每个 config 值为 `"loaded"` 或 `"error"`（解析失败或缺失）。

---

## 7. 请求与响应 Schema 详解

### 7.1 请求模型

#### `DispatchRequest`
| 字段 | 类型 | 默认 | 校验 | 说明 |
|------|------|------|------|------|
| `channel` | string \| null | null | — | 筛选渠道 |
| `limit` | int | 100 | 1~1000 | 最多处理多少条 |
| `apply` | bool | false | — | `false`=dry-run, `true`=apply |

#### `FeishuExportRequest` / `FeishuSyncRequest` / `FeishuStatusImportRequest`
| 字段 | 类型 | 默认 | 校验 | 说明 |
|------|------|------|------|------|
| `tables` | string[] \| null | null | 最多 20，字母+数字+下划线 | 表名列表 |
| `apply` | bool | false | — | dry-run / apply |

#### `PipelineRunRequest`
| 字段 | 类型 | 默认 | 说明 |
|------|------|------|------|
| `pipeline_type` | string | `"daily"` | 管道类型 |

### 7.2 响应模型索引

| 模型名 | 用途 | 关键字段 |
|--------|------|----------|
| `HealthResponse` | health 端点 | status, version, db_connected |
| `StatusResponse` | status 端点 | database, last_pipeline_run, version, migration_status |
| `AlertItem` / `AlertListResponse` | alerts 端点 | event_id, rule_id, severity, impact_score |
| `TaskItem` / `TaskListResponse` | tasks 端点 | task_id, status, priority, owner_role |
| `OutboxItem` / `OutboxListResponse` | outbox 列表 | outbox_id, target_channel, status, dispatch_attempts |
| `DispatchResultItem` / `DispatchResponse` | dispatch 端点 | request_id, dry_run, processed, results[] |
| `ErrorLogEntry` / `ErrorLogListResponse` | error 日志 | ts, error_code, message, diagnosis |
| `RecentLogEntry` / `RecentLogListResponse` | 最近请求 | ts, method, path, request_id |
| `AuditLogEntry` / `AuditLogListResponse` | 审计日志 | outbox_id, target_channel, mode, status, error |
| `FeishuTableResult` / `FeishuExportResponse` / `FeishuSyncResponse` / `FeishuStatusImportResponse` | feishu 操作 | 单个 table 的统计 (rows/created/updated/pulled/imported/skipped) |
| `PipelineRunResponse` | pipeline 端点 | command, required_env_vars, warnings |
| `GovernanceConfigResponse` | governance 端点 | extra="allow"，返回 YAML 全部内容 |
| `GovernanceStatusResponse` | governance/status 端点 | governance_layer, configs |
| `DiagnosisLogEntry` / `DiagnosisResponse` | diagnosis 端点 | request_id, summary, related_logs |
| `ErrorResponse` | 所有错误 | request_id, error_code, message, diagnosis, suggested_action |

---

## 8. 运维指导

### 8.1 健康检查

推荐监控探针：

```bash
# 健康检查（无需认证，30 req/min 限流）
curl http://127.0.0.1:8765/api/v1/health
# 期望: {"status":"ok","version":"...","db_connected":true}

# 系统状态（需认证）
curl -H "Authorization: Bearer $TOKEN" http://127.0.0.1:8765/api/v1/status
# 关注: migration_status.status 不为 "failed"
```

### 8.2 日志位置

| 文件 | 路径 | 用途 |
|------|------|------|
| 应用日志 | `logs/api/api.log` | 所有请求（INFO+） |
| 错误日志 | `logs/api/error.log` | ERROR+ 级别 |
| 审计日志 | `logs/api/audit.log` | 写操作（INFO+） |
| Dispatch 审计 | `data/system/api_audit_dispatch.csv` | dispatch 历史 |
| Feishu 审计 | `data/system/api_audit_feishu.csv` | 飞书操作历史 |

### 8.3 常见错误排查

| 现象 | 排查 |
|------|------|
| 503 `DB_MISSING` | `data/olist_ops.db` 不存在，需先运行 `python3 scripts/run_db_pipeline.py --mode full` |
| 503 critical table missing | DB 文件存在但 schema 不完整，跑迁移 `python3 scripts/db_migrate.py` |
| 429 `RATE_LIMITED` | 客户端请求过快；如果是健康检查频繁，调整探针间隔 |
| 401 / 403 认证失败 | 检查 `Authorization: Bearer <token>` 头与 `API_BEARER_TOKEN` 环境变量是否一致 |
| 500 `FEISHU_DISPATCH_FAILED` | 检查 `.env` 中的飞书凭据是否正确配置 |
| 500 `INTERNAL_ERROR` | 查看 `logs/api/error.log`，使用 `GET /logs/diagnosis?request_id=<rid>` 追踪 |

### 8.4 限流调优

环境变量 `TRUSTED_PROXY_IPS` 配置可信反向代理（逗号分隔），允许从 `X-Forwarded-For` 读取真实客户端 IP。

限流参数在 `api/main.py` 中硬编码：
```python
_rate_limit_config = {
    "health": ("/api/v1/health", (30, 60)),         # 30/60s
    "dispatch": ("/api/v1/outbox/dispatch", (30, 60)),
    "pipeline": ("/api/v1/pipeline/run", (10, 60)),
}
_DEFAULT_RATE = (300, 60)   # 默认 300/60s
```

### 8.5 数据库迁移

启动时 **自动检查并应用缺失的迁移**，无需手动跑 migration 命令。如果关键表（`dwd_order_level`、`alert_events`）缺失，会在响应中的 `migration_status.failed` 数组中报告，API 仍然会启动但数据可能不完整。

---

## 9. 前端集成指南

### 9.1 通用请求模板

```typescript
// lib/api.ts
const BASE_URL = "http://127.0.0.1:8765/api/v1";

async function api<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, {
    ...options,
    headers: {
      Authorization: `Bearer ${import.meta.env.VITE_API_TOKEN}`,
      "Content-Type": "application/json",
      "X-Request-ID": crypto.randomUUID(),  // 推荐：生成 request_id
      ...(options.headers || {}),
    },
  });

  if (!res.ok) {
    const err = await res.json();
    throw new ApiError(err);
  }
  return res.json();
}

class ApiError extends Error {
  constructor(public readonly data: {
    request_id: string;
    error_code: string;
    diagnosis: string;
    suggested_action: string;
  }) {
    super(data.message);
  }
}
```

### 9.2 TanStack Query 示例

```tsx
function useAlerts(severity?: string) {
  return useQuery({
    queryKey: ["alerts", severity],
    queryFn: () => {
      const params = severity ? `?severity=${severity}` : "";
      return api<AlertListResponse>(`/alerts${params}`);
    },
  });
}

function useDispatch() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: DispatchRequest) =>
      api<DispatchResponse>("/outbox/dispatch", {
        method: "POST",
        body: JSON.stringify(body),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["outbox"] }),
  });
}
```

### 9.3 React 控制台

项目自带 React 控制台，构建后通过同一端口提供静态服务：

```bash
cd frontend && npm run build  # 构建到 frontend/dist/
cd .. && python3 scripts/run_api.py  # 自动 mount /console
```

访问：`http://127.0.0.1:8765/console`

---

## 10. 附录：数据表参考

### 10.1 SQLite 表结构概览

| 表名 | 粒度 | 描述 | API 可见 |
|------|------|------|----------|
| `pipeline_runs` | 一次管道运行 | 管道运行历史 | ✅ (status) |
| `ingestion_batches` | 一个 ingestion 批次 | 数据摄取批次 | ✅ (status) |
| `dwd_order_level` | 一个订单 | 订单级宽表 | ✅ (status.tables) |
| `dwd_item_level` | 一个订单商品行 | 商品级宽表 | ✅ (status.tables) |
| `metric_daily` | 一天 | 全局每日指标 | ✅ (status.tables) |
| `metric_dimension_daily` | 一天 × 一维度 × 一 ID | 维度级每日指标 | ✅ (status.tables) |
| `alert_events` | 一个告警 | 规则触发的告警 | ✅ (alerts) |
| `strategy_recommendations` | 一条建议 | 策略建议 | ✅ (status.tables) |
| `action_tasks` | 一个任务 | 行动任务 | ✅ (tasks) |
| `review_retro` | 一次复盘 | 复盘记录 | ✅ (status.tables) |
| `event_outbox` | 一个调度事件 | 发件箱队列 | ✅ (outbox) |
| `qoder_jobs` | 一个 agent job | agent 任务 | ✅ (status.tables) |
| `governance_checkpoints` | 治理检查点 | 审计记录 | — |
| `governance_health_results` | 健康检查结果 | 治理健康审计 | — |

### 10.2 状态机

| 实体 | 状态 | 终态 |
|------|------|------|
| `alert_events.status` | `new` → `investigating` → `strategy_generated` → `task_created` → `resolved` | `resolved`, `ignored` |
| `action_tasks.status` | `todo` → `in_progress` → (`blocked`) → `done` | `done`, `cancelled` |
| `event_outbox.status` | `pending` → `dispatching` → (`pending` 重试) → `dispatched` / `skipped` / `failed` | `dispatched`, `skipped`, `failed` |
| `strategy_recommendations.status` | `draft` → `pending_review` → `approved` → `executing` → `completed` | `completed`, `rejected`, `invalidated` |

### 10.3 维度告警对象

| `object_type` | `object_id` 示例 | 描述 |
|---------------|------------------|------|
| `global` | `global` | 全局（无维度下钻） |
| `seller` | `seller_123` | 单个卖家 |
| `category` | `electronics` | 单个商品品类 |
| `region` | `SP` | 单个地区 |

---

## 变更日志

| 版本 | 主要变更 |
|------|----------|
| v0.5 | 首个 API 网关：FastAPI + 6 个核心端点 + Bearer Token + 限流 |
| v0.5.1 | 新增 Logs / Feishu / Pipeline / Governance 端点；结构化审计；前端 Alpha |
| v0.5.2 | 新增 `/logs/diagnosis`；前端硬化；日志诊断优化 |
| v0.5.3 | Governance 7 个端点 + response model；请求 schema validator；测试隔离优化 |

---

## 联系与问题反馈

如有问题请通过 GitHub Issues 报告，附带：
1. 请求的 `X-Request-ID` 响应头
2. HTTP 状态码和完整错误响应体
3. 使用 `GET /logs/diagnosis?request_id=<rid>` 获取的诊断结果
