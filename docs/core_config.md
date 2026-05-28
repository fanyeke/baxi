# 核心配置 — internal/config

## 配置加载

Go 配置通过 `internal/config/config.go` 的 `Load()` 函数加载，从环境变量读取。

```go
cfg, err := config.Load()
```

## 配置项

| 字段 | 环境变量 | 类型 | 默认值 | 说明 |
|------|---------|------|--------|------|
| `DatabaseURL` | `DATABASE_URL` | string | — | PostgreSQL 连接串（**必填**） |
| `APIPort` | `API_PORT` | string | `8080` | API 监听端口 |
| `LogLevel` | `LOG_LEVEL` | string | `info` | 日志级别 |
| `APIBearerToken` | `API_BEARER_TOKEN` | string | — | API 认证令牌（**必填**） |
| `CORSAllowedOrigins` | `CORS_ALLOWED_ORIGINS` | string | `http://localhost:5173` | CORS 允许来源 |

### LLM 配置

| 字段 | 环境变量 | 默认值 |
|------|---------|--------|
| `LLMAPIKey` | `LLM_API_KEY` | — |
| `LLMAPIBase` | `LLM_API_BASE` | — |
| `LLMModel` | `LLM_MODEL` | `gpt-4o-mini` |
| `LLMTemperature` | `LLM_TEMPERATURE` | `0.7` |
| `LLMMaxTokens` | `LLM_MAX_TOKENS` | `1024` |
| `LLMTimeoutSeconds` | `LLM_TIMEOUT_SECONDS` | `60` |
| `LLMEnabled` | `LLM_ENABLED` | `false` |
| `LLMProvider` | `LLM_PROVIDER` | `disabled` |
| `LLMStoreRawOutput` | `LLM_STORE_RAW_OUTPUT` | `false` |
| `LLMMaxRetries` | `LLM_MAX_RETRIES` | `3` |

### Worker 配置

| 字段 | 环境变量 | 默认值 |
|------|---------|--------|
| `ActionApplyDryRun` | `ACTION_APPLY_DRY_RUN` | `true` |
| `WorkerTickInterval` | `WORKER_TICK_INTERVAL` | `30s` |
| `WorkerBatchSize` | `WORKER_BATCH_SIZE` | `10` |

### 飞书集成

| 字段 | 环境变量 | 说明 |
|------|---------|------|
| `FeishuWebhookURL` | `FEISHU_WEBHOOK_URL` | 飞书 Webhook 地址 |
| `GitHubToken` | `GITHUB_TOKEN` | GitHub API Token |

配置在服务启动时自动加载，缺少 `DATABASE_URL` 或 `API_BEARER_TOKEN` 时会报错。
