# Phase 1: Go + PostgreSQL Foundation

## 阶段目标

Phase 1 的目标是建立 Go + PostgreSQL + Docker Compose 的最小可运行工程骨架。

完成后具备：
1. PostgreSQL 可以通过 Docker Compose 启动
2. Go 项目可以编译
3. migration 工具可以初始化数据库 schema
4. baxi-api 可以启动并提供 /api/v1/health
5. baxi-worker 可以启动但暂不处理业务
6. Makefile 可以统一管理本地命令
7. 不修改旧 Python 业务逻辑

## 新增目录结构

```
.
├── cmd/
│   ├── baxi-api/
│   │   └── main.go
│   └── baxi-worker/
│       └── main.go
├── internal/
│   ├── api/
│   │   ├── health.go
│   │   └── server.go
│   ├── config/
│   │   └── config.go
│   ├── db/
│   │   └── postgres.go
│   ├── logger/
│   │   └── logger.go
│   └── worker/
│       └── worker.go
├── migrations/
│   └── 001_init_schemas.sql
├── docker-compose.yml
├── Dockerfile.api
├── Dockerfile.worker
├── Makefile
├── go.mod
├── go.sum
├── .env.example
├── .dockerignore
└── docs/
    └── migration/
        └── phase-1-foundation.md
```

## 技术选型

| 组件 | 选型 | 说明 |
|------|------|------|
| Go | 1.22+ | 编译型、高性能 |
| PostgreSQL | 16 | 关系型数据库 |
| HTTP Router | chi/v5 | 轻量级路由 |
| DB Driver | pgx/v5 | PostgreSQL 原生驱动 |
| Migration | goose/v3 | SQL 迁移工具 |
| Logger | zap | 结构化日志 |
| Config | 环境变量 | .env.example 作为参考 |

## 本地启动

### 前置条件

- Go 1.22+
- Docker & Docker Compose v2
- goose CLI (`go install github.com/pressly/goose/v3/cmd/goose@latest`)

### 1. 启动 PostgreSQL

```bash
make up
# 或: docker compose up -d postgres
```

验证：
```bash
docker compose ps | grep baxi-postgres
# 应看到: baxi-postgres  healthy
```

### 2. 执行 Migration

```bash
export DATABASE_URL="postgres://baxi:baxi_dev@localhost:5432/baxi?sslmode=disable"
make migrate
```

验证：
```bash
psql "$DATABASE_URL" -c "SELECT schema_name FROM information_schema.schemata WHERE schema_name IN ('raw','dwd','mart','ops','gov','ai','audit') ORDER BY schema_name;"
# 应输出 7 个 schema
```

### 3. 启动 API

```bash
export DATABASE_URL="postgres://baxi:baxi_dev@localhost:5432/baxi?sslmode=disable"
make api
```

验证：
```bash
curl http://localhost:8080/api/v1/health
# 应返回: {"status":"ok","service":"baxi-api"}
```

### 4. 启动 Worker

```bash
export DATABASE_URL="postgres://baxi:baxi_dev@localhost:5432/baxi?sslmode=disable"
make worker
# 应输出: baxi-worker started
```

## 验收命令

```bash
# 1. Infrastructure
docker compose up -d postgres
docker compose ps | grep baxi-postgres | grep healthy

# 2. Migration
export DATABASE_URL="postgres://baxi:baxi_dev@localhost:5432/baxi?sslmode=disable"
make migrate
psql "$DATABASE_URL" -c "SELECT schema_name FROM information_schema.schemata WHERE schema_name IN ('raw','dwd','mart','ops','gov','ai','audit') ORDER BY schema_name;"

# 3. Go compilation
go test ./...
go vet ./...

# 4. API
make api &
curl http://localhost:8080/api/v1/health

# 5. Worker
timeout 5s make worker

# 6. Scope check
git diff --name-only
```

## Schema 用途说明

| Schema | 用途 | Phase |
|--------|------|-------|
| raw | 原始 Olist CSV 导入表 | Phase 2 |
| dwd | 订单级、商品级明细宽表 | Phase 2+ |
| mart | 指标快照、日指标、维度指标 | Phase 3+ |
| ops | alert、task、recommendation、outbox | Phase 3+ |
| gov | 数据分类、对象 schema、血缘、权限 | Phase 4+ |
| ai | decision_case、llm_decision、action_proposal | Phase 4+ |
| audit | pipeline_run、api_log、audit_log、error_log | Phase 2+ |

## 非目标 (Out of Scope)

1. ❌ 不迁移 CSV Pipeline
2. ❌ 不迁移 SQLite 表
3. ❌ 不实现完整 PostgreSQL 业务表
4. ❌ 不接入 LLM
5. ❌ 不重写 FastAPI 业务端点
6. ❌ 不改 React 前端
7. ❌ 不改现有 YAML 治理配置语义

## 提交历史

```
547be35 chore: add docker postgres foundation
e367fdb feat: add go api and worker skeleton
```

## 下一步 (Phase 2 预览)

Phase 2 将在 Phase 1 基础设施上：
1. 设计 PostgreSQL 业务表结构（orders, customers, products, sellers 等）
2. 创建 CSV 导入 pipeline 的 Go 版本
3. 实现完整的 health check（含 DB ping）
4. 添加基础测试
