# Baxi — 电商治理与决策平台

## 项目概述

Baxi 是一个多语言治理 + 分析平台，原始数据源为 Olist 巴西电商公开数据集。项目已完成从 Python/SQLite 到 Go/PostgreSQL 的全面迁移。

**核心能力**：
- Go 数据管道引擎（摄入 → DWD 层 → 指标聚合 → 异常检测 → 建议生成）
- 规则驱动决策引擎含沙盘模拟
- React 19 管理控制台（Vite + TanStack Query + Radix UI）
- 飞书集成（消息推送、多维表格同步）
- MCP Server（31 个工具，11 个业务域）/ Pi Agent 集成

## 技术栈

| 层 | 技术 | 说明 |
|----|------|------|
| **后端** | Go 1.23 (chi, pgx) | API 服务、Worker、CLI |
| **数据库** | PostgreSQL 16 | 主存储 |
| **前端** | React 19, Vite 6, TanStack Query 5, Tailwind CSS v4, Radix UI | 管理控制台 |
| **数据管道** | Go + Step 接口编排 | 7 步管道 + 审计记录 |
| **渠道适配** | Go Adapter 模式 | 飞书、GitHub Issue、CLI、手动审核 |
| **容器化** | Docker | 多阶段构建，CGO_ENABLED=0 |
| **MCP Server** | Go (mark3labs/mcp-go) | 31 个工具，Pi Agent 集成 |

## 快速开始

### 前提条件

- Go 1.23
- Docker 和 Docker Compose
- Node.js 18+（仅前端开发）
- PostgreSQL 16（通过 Docker）

### 启动数据库

```bash
make up              # docker compose up postgres
make migrate         # goose migrations up
```

### 配置环境变量

```bash
cp .env.example .env
# 编辑 .env 填入必要配置（API_BEARER_TOKEN 等）
```

### 运行

```bash
# 启动 API 服务（端口 8080）
make api

# 启动 Worker
make worker

# 运行数据管道
make pipeline

# 构建二进制
make build

# 运行 Go 测试
make test            # go test ./... (Go only)
```

### 前端开发

```bash
cd frontend
npm install
npm run dev          # React 开发服务器 :5173
```

## 项目结构

```
baxi/
├── cmd/              # Go 入口点（baxi-api, baxi-cli, baxi-worker, baxi-mcp）
├── internal/         # Go 核心：44 个包
│   ├── api/          # chi HTTP 服务（15 个 handler、6 个中间件）
│   ├── pipeline/     # 管道步骤定义与编排（7 步）
│   ├── decision/     # 决策引擎（case engine + context builder）
│   ├── service/      # 业务服务层
│   ├── repository/   # 数据访问层（pgx 实现）
│   ├── governance/   # 数据治理（分类、血缘、访问策略）
│   ├── worker/       # 后台 Worker（事件分发）
│   ├── action/       # 动作注册与执行
│   ├── alert/        # 维度级异常检测
│   ├── adapter/      # 渠道适配（飞书、GitHub、CLI、手动）
│   ├── review/       # 审核与审批
│   ├── recommendation/ # 建议生成
│   ├── config/       # 环境配置结构体
│   ├── mcp/          # MCP 协议 Server（31 个工具）
│   └── llm/          # LLM 决策提供者抽象
├── pi-extension/     # Pi Agent TypeScript 扩展
├── frontend/         # React 19 SPA
├── config/           # YAML 治理/业务配置（28 文件）
├── migrations/       # Goose SQL 迁移
├── test/             # Go 集成 + 安全 E2E 测试
├── docs/             # 技术文档
├── data/             # 原始 CSV + 中间数据
└── scripts/          # 工具脚本（含冻结分析脚本）
```

## 可用命令

```bash
make up              # docker compose up postgres
make api             # go run ./cmd/baxi-api
make worker          # go run ./cmd/baxi-worker
make build           # go build 所有二进制
make pipeline        # Go CLI 管道运行
make migrate         # goose 迁移 up
make mcp             # 启动 MCP 服务器（stdio）
make test            # go test ./...（仅 Go）
```

**Make 管道命令**：
```bash
make pipeline-ingest          # 仅数据摄入
make pipeline-dwd             # DWD 层构建
make pipeline-metrics         # 指标构建
make pipeline-compare         # 基线对比验证
make test-pipeline            # 管道测试
```

## 版本演进

| 版本 | 核心能力 | 状态 |
|------|---------|------|
| v0.1-v0.5.3 | Python/SQLite 原型 | ✅ 已归档 |
| v1.0 | Go/PostgreSQL 迁移完成 | ✅ 当前 |
| Phase I | 全量数据 + AI 决策引擎 (LLM 代码就绪，待激活) | 🟡 核心完成 |
| Phase II+ | 维度告警扩展 / 真实 LLM 决策 / 自动调度 | ❌ 未启动 |

## 参考资源

- [Brazilian E-Commerce Dataset on Kaggle](https://www.kaggle.com/datasets/olistbr/brazilian-ecommerce) — 主数据集
- [Olist 官方文档](https://olist.com/)

---

**最后更新**: 2026-05-28
**当前版本**: v1.0（Go/PostgreSQL）
**活跃维护**: Go API 服务、数据管道、React 控制台
