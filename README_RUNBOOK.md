# Baxi 运维手册

## 技术栈
- **后端**: Go 1.22+ (chi, pgx)
- **数据库**: PostgreSQL 15
- **前端**: React 19 + Vite
- **数据管道**: Go Step 接口编排（7 步）

## 前提条件
- Go 1.22+
- Docker & Docker Compose
- Node.js 18+（前端开发用）

## 快速启动
```bash
make up              # 启动 PostgreSQL
make migrate         # 运行数据库迁移
make api             # 启动 API 服务（端口 8080）
make worker          # 启动 Worker
make pipeline        # 运行数据管道
```

## 常用命令

| 命令 | 说明 |
|------|------|
| `make mcp` | 启动 MCP 服务器（stdio） |
| `make build` | 编译所有二进制 |
| `make test` | 运行 Go 测试（`go test ./...`） |
| `make pipeline-ingest` | 仅数据摄入步骤 |
| `make pipeline-dwd` | 仅 DWD 层构建 |
| `cd frontend && npm run dev` | React 开发服务器（端口 5173） |

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `DATABASE_URL` | PostgreSQL 连接串 | **必填** |
| `API_BEARER_TOKEN` | API 访问令牌 | **必填** |
| `API_PORT` | API 服务端口 | `8080` |
| `LOG_LEVEL` | 日志级别 | `info` |
| `CORS_ALLOWED_ORIGINS` | CORS 允许的源 | `http://localhost:5173` |

## 架构概要

```
Client → Go chi API (端口 8080) → Service Layer → Repository → PostgreSQL
                                          ↕
                                    Pipeline (7 步)
                                       ↓
                                Worker (事件分发)
```

API 采用分层架构：Handler → Service → Repository。Handler 定义本地接口用于测试 mock。
Repository 使用 PoolProvider 注入，无需手动传递连接池。

## 错误排查

| 现象 | 排查 |
|------|------|
| API 返回 401 | 检查 `API_BEARER_TOKEN` 环境变量 |
| API 返回 503 | 检查 PostgreSQL 连接（`DATABASE_URL`） |
| Pipeline 失败 | 检查数据文件在 `data/` 目录下是否存在 |
| Worker 无响应 | 检查数据库连接和 Outbox 表中是否有待处理事件 |
