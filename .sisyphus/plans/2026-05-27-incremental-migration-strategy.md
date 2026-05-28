# Baxi 渐进式迁移策略：Python/SQLite → Go/PostgreSQL

> **版本**: v1.0  
> **日期**: 2026-05-27  
> **状态**: 策略设计完成，待执行  
> **原则**: 向后兼容、可回滚、零停机、渐进式

---

## 1. 当前架构现状分析

### 1.1 双系统共存格局

```
┌─────────────────────────────────────────────────────────────────────┐
│                        当前生产环境                                  │
│                                                                     │
│  ┌──────────────────────┐        ┌──────────────────────┐          │
│  │  Python FastAPI :8765 │        │   Go chi API :8080   │          │
│  │  (生产流量入口)       │        │  (迁移验证环境)       │          │
│  └──────────┬───────────┘        └──────────┬───────────┘          │
│             │                                │                      │
│  ┌──────────▼───────────┐        ┌──────────▼───────────┐          │
│  │   SQLite (olist_ops)  │        │  PostgreSQL (baxi)   │          │
│  │   16 表, 906,526 行   │        │  42 表, 7 schemas    │          │
│  └──────────────────────┘        └──────────────────────┘          │
│                                                                     │
│  ┌──────────────────────┐        ┌──────────────────────┐          │
│  │  Python Pipeline      │        │  Go Pipeline (baxi-cli)│        │
│  │  9 步数据管道         │        │  7 阶段数据管道        │          │
│  └──────────────────────┘        └──────────────────────┘          │
│                                                                     │
│  ┌──────────────────────────────────────────────────────┐          │
│  │              React Frontend :5173                      │          │
│  │              (仅连接 Python API :8765)                 │          │
│  └──────────────────────────────────────────────────────┘          │
└─────────────────────────────────────────────────────────────────────┘
```

### 1.2 迁移进度矩阵

| 阶段 | 状态 | 完成度 | 关键产物 |
|------|------|--------|---------|
| Phase 0: Baseline Freeze | ✅ 完成 | 100% | `migration_baseline/` |
| Phase 1: Docker + PostgreSQL | ✅ 完成 | 100% | `docker-compose.yml`, `go.mod` |
| Phase 2: Schema Design | ✅ 完成 | 100% | 42 tables, 7 schemas, 16 migrations |
| Phase 3: Pipeline Migration | ✅ 完成 | 98% | Go pipeline, 6/8 tables exact match |
| Phase 4: API Migration | 🟡 进行中 | ~60% | 11 read-only endpoints designed |
| Phase 5: Governance Runtime | 📋 已设计 | 0% | ConfigLoader, ObjectRegistry |
| Phase 6: Decision/LLM Context | 📋 已设计 | 0% | decision_case, action_proposal |
| Phase 7: Review/Action/Outbox | 📋 已设计 | 0% | review_record, dispatch worker |

### 1.3 已知技术债务

| 问题 | 影响 | 优先级 |
|------|------|--------|
| Go 二进制文件提交到 git | 仓库膨胀 | P2 |
| 两个测试目录 (tests/ vs test/) | 混淆 | P2 |
| 两个迁移目录 (migrations/ vs sql/migrations/) | 不协调 | P1 |
| Python f-string SQL (19处) | SQL注入风险 | P1 |
| 前端仅连接 Python API | 迁移阻塞 | P0 |
| CI 仅测试 Python | Go/前端无覆盖 | P1 |

---

## 2. 迁移核心原则

### 2.1 五阶段渐进式策略

```
Stage 1: 并行运行 (Dual-Read)
    ↓ 验证通过
Stage 2: Go 为主读 (Go-Primary-Read)
    ↓ 验证通过
Stage 3: 双写 (Dual-Write)
    ↓ 验证通过
Stage 4: Go 为主写 (Go-Primary-Write)
    ↓ 验证通过
Stage 5: Python 退役 (Legacy Sunset)
```

### 2.2 回滚承诺

每个阶段必须满足：
- **数据回滚**: 可在 30 分钟内恢复到上一阶段
- **流量回滚**: 可在 5 分钟内将流量切回 Python
- **配置回滚**: 所有配置变更可逆
- **无数据丢失**: 回滚后数据完整性不受损

### 2.3 验收门禁

每个阶段通过以下门禁才能进入下一阶段：
1. **功能测试**: 所有端点响应一致
2. **性能测试**: p99 延迟不超过 Python 版本 20%
3. **数据一致性**: 行数和关键值匹配
4. **监控告警**: 无 P0/P1 告警
5. **人工确认**: 至少一位维护者 sign-off

---

## 3. 数据库迁移策略

### 3.1 Schema 迁移 (已完成)

**状态**: Phase 2 已完成，42 tables across 7 schemas

**已执行的 Schema 演进**:
```
001_init_schemas.sql      → 创建 7 个 schema
002_raw_tables.sql        → 11 raw staging tables
003_dwd_tables.sql        → 2 DWD tables
004_mart_tables.sql       → 3 mart tables
005_ops_tables.sql        → 7 ops tables
006_gov_tables.sql        → 7 gov tables
007_ai_tables.sql         → 6 ai tables
008_audit_tables.sql      → 6 audit tables
009_gov_indexes.sql       → Governance indexes
010_ai_tables_enhance.sql → AI table enhancements
011_review_action_outbox.sql → Review/action constraints
012_llm_activation_eval.sql → LLM eval tables
013_fix_decision_case_index.sql → Index fix
014_add_outbox_next_retry_at.sql → Outbox retry
016_config_versions.sql   → Config versioning
```

**回滚策略**:
```bash
# 逐级回滚
make migrate-down  # 回滚最后一个迁移

# 批量回滚到指定版本
goose -dir migrations postgres "$DATABASE_URL" down-to 001
```

### 3.2 数据迁移策略

#### 3.2.1 CSV → Raw 表 (已实现)

```
data/raw/*.csv  →  COPY  →  raw.olist_* (11 tables)
```

- **策略**: 全量 TRUNCATE + COPY (每次管道运行)
- **幂等性**: TRUNCATE 确保干净状态
- **验证**: 行数对比 `raw.olist_orders: 99,441`

#### 3.2.2 Raw → DWD 表 (已实现)

```
raw.olist_*  →  SQL JOIN  →  dwd.order_level (99,441)
                           →  dwd.item_level (112,650)
```

- **策略**: INSERT ... ON CONFLICT DO NOTHING
- **幂等性**: 天然幂等
- **验证**: 行数精确匹配

#### 3.2.3 DWD → Mart 表 (已实现)

```
dwd.order_level  →  聚合  →  mart.metric_daily (634)
dwd.item_level   →  聚合  →  mart.metric_dimension_daily (~693,602)
```

- **策略**: TRUNCATE + INSERT (全量模式)
- **偏差**: metric_dimension_daily 有 -0.47% 偏差 (已接受)
- **验证**: 行数 + 值容差 1e-9

#### 3.2.4 Mart → Ops 表 (已实现)

```
mart.metric_daily  →  规则引擎  →  ops.metric_alert (37)
                                 →  ops.recommendation (37)
                                 →  ops.task (37)
                                 →  ops.outbox_event (37)
```

- **偏差**: +1 alert (阈值边界差异，已接受)
- **级联一致性**: 37:37:37:37 比例保持

### 3.3 SQLite → PostgreSQL 数据同步 (过渡期)

**场景**: 需要在过渡期保持 SQLite 和 PostgreSQL 数据同步

**策略**: 单向同步 (SQLite → PostgreSQL)

```python
# scripts/migration/sqlite_to_pg_sync.py
# 1. 读取 SQLite 表
# 2. 转换类型 (INTEGER→BOOLEAN, TEXT→TIMESTAMPTZ)
# 3. 写入 PostgreSQL (INSERT ON CONFLICT)
# 4. 验证行数
```

**触发条件**:
- Python pipeline 运行后
- 手动触发: `make sync-sqlite-to-pg`

**回滚**: PostgreSQL 数据可通过重新运行 Go pipeline 恢复

---

## 4. 代码迁移策略

### 4.1 API 迁移策略 (Phase 4)

#### 4.1.1 端点迁移顺序

```
Wave 1: 基础端点 (无依赖)
  ├── GET /health          → Go (无 auth)
  ├── GET /status          → Go (聚合查询)
  └── GET /qoder/capabilities → Go (静态配置)

Wave 2: 数据查询端点
  ├── GET /alerts          → Go (ops.metric_alert)
  ├── GET /tasks           → Go (ops.task)
  └── GET /outbox          → Go (ops.outbox_event)

Wave 3: 日志端点
  ├── GET /logs/recent     → Go (audit.api_request_log)
  ├── GET /logs/errors     → Go (audit.error_log)
  └── GET /logs/audit      → Go (audit.audit_log)

Wave 4: 治理端点
  ├── GET /governance/status → Go (gov.*)
  └── GET /qoder/context    → Go (多表聚合)
```

#### 4.1.2 流量切换策略

```
Phase A: 影子模式 (Shadow Mode)
  ├── 前端请求 → Python API (生产)
  ├── Python API → 复制请求 → Go API (影子)
  ├── 对比响应差异
  └── 不返回 Go 响应给用户

Phase B: 金丝雀发布 (Canary)
  ├── 10% 流量 → Go API
  ├── 90% 流量 → Python API
  ├── 监控错误率、延迟
  └── 逐步增加 Go 比例

Phase C: 全量切换
  ├── 100% 流量 → Go API
  ├── Python API 保持运行 (备用)
  └── 7天观察期

Phase D: Python 退役
  ├── Python API 下线
  ├── 保留 legacy 分支
  └── 清理临时代码
```

#### 4.1.3 前端切换方案

**当前状态**: 前端硬编码连接 `http://localhost:8765`

**切换方案**:

```typescript
// frontend/src/config.ts
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8765';

// 切换到 Go API:
// VITE_API_BASE_URL=http://localhost:8080 npm run dev
```

**环境变量控制**:
```bash
# .env.production
VITE_API_BASE_URL=http://localhost:8080  # Go API

# .env.development
VITE_API_BASE_URL=http://localhost:8765  # Python API (开发)
```

### 4.2 Pipeline 迁移策略 (Phase 3)

**状态**: 已完成，6/8 tables exact match

**验证命令**:
```bash
# 运行 Go pipeline
make pipeline

# 对比基线
make pipeline-compare

# 验证结果
psql "$DATABASE_URL" -c "
  SELECT 'dwd.order_level' as tbl, COUNT(*) FROM dwd.order_level
  UNION ALL SELECT 'dwd.item_level', COUNT(*) FROM dwd.item_level
  UNION ALL SELECT 'mart.metric_daily', COUNT(*) FROM mart.metric_daily
"
```

**偏差接受决策**:

| 表 | 偏差 | 原因 | 决策 |
|----|------|------|------|
| metric_dimension_daily | -3,276 (-0.47%) | NULL 处理差异 | ✅ 接受 |
| metric_alert | +1 (+2.78%) | 阈值边界 | ✅ 接受 (新基线) |
| recommendation/task/outbox | +1 each | 级联 | ✅ 接受 |

### 4.3 配置迁移策略

#### 4.3.1 YAML → DB 配置 (Phase 5)

**策略**: 启动时加载 YAML → 写入 `gov.config_snapshot`

```
config/*.yml  →  ConfigLoader.LoadAll()  →  gov.config_snapshot
```

**优势**:
- 版本追踪 (content_hash)
- 运行时查询
- 审计日志

**保持兼容**:
- YAML 文件仍是 source of truth
- DB 是缓存/索引
- 手动同步: `make governance-load`

#### 4.3.2 配置热重载 (未来)

**当前**: 需要重启服务器

**未来方案**:
```go
// 内部/configloader/watcher.go
// 监听 config/ 目录变更
// 自动触发 SyncSnapshots
```

---

## 5. 测试策略

### 5.1 测试金字塔

```
                    ┌─────────────┐
                    │   E2E 测试   │  ← 10% (Playwright)
                    │  (关键路径)   │
                    ├─────────────┤
                    │  集成测试    │  ← 30% (testcontainers)
                    │  (API + DB)  │
                    ├─────────────┤
                    │   单元测试    │  ← 60% (go test / pytest)
                    │  (纯逻辑)    │
                    └─────────────┘
```

### 5.2 各阶段测试要求

| 阶段 | Go 测试 | Python 测试 | 前端测试 | E2E |
|------|---------|-------------|---------|-----|
| Phase 4 | 单元 + 集成 | 回归 | — | API 对比 |
| Phase 5 | 单元 + 集成 | — | — | 治理端点 |
| Phase 6 | 单元 + 集成 | — | — | 决策流程 |
| Phase 7 | 单元 + 集成 | — | — | 审核→执行 |

### 5.3 基线对比测试

**Python 对比脚本**:
```bash
# API 响应对比
python3 scripts/migration/compare_api_baseline.py \
  --baseline-dir migration_baseline/api_responses/ \
  --go-url http://localhost:8080/api/v1 \
  --token $API_BEARER_TOKEN

# Pipeline 输出对比
python3 scripts/migration/compare_csv.py \
  --baseline-dir migration_baseline/pipeline_outputs/ \
  --actual-dir /tmp/pipeline_outputs/ \
  --float-tolerance 1e-9
```

### 5.4 回归测试矩阵

**每次迁移必须验证**:

```bash
# 1. Go 测试
make test-go

# 2. Python 测试
make test-python

# 3. 前端测试
make test-frontend

# 4. Pipeline 一致性
make pipeline-compare

# 5. API 一致性
make api-compare
```

---

## 6. 回滚计划

### 6.1 每阶段回滚方案

#### Phase 4 回滚 (API 迁移)

```bash
# 1. 前端流量切回 Python
# 修改 frontend/.env
VITE_API_BASE_URL=http://localhost:8765

# 2. 重启前端
cd frontend && npm run build

# 3. 验证
curl http://localhost:8765/api/v1/health
```

**RTO**: 5 分钟  
**RPO**: 0 (只读操作)

#### Phase 5 回滚 (治理运行时)

```bash
# 1. 重启 Go API (治理配置从 YAML 重新加载)
make api

# 2. 如果需要完全回滚
git checkout migration/go-postgres~1
make build && make api
```

**RTO**: 10 分钟  
**RPO**: 0 (配置数据)

#### Phase 6 回滚 (决策管道)

```bash
# 1. 清理 ai.* 表数据
psql "$DATABASE_URL" -c "
  TRUNCATE ai.decision_case CASCADE;
  TRUNCATE ai.llm_decision CASCADE;
  TRUNCATE ai.action_proposal CASCADE;
"

# 2. 回滚代码
git checkout migration/go-postgres~1
make build && make api
```

**RTO**: 15 分钟  
**RPO**: 0 (决策数据可重新生成)

#### Phase 7 回滚 (审核/执行)

```bash
# 1. 停止 worker
docker compose stop worker

# 2. 停止 dispatch
# 设置环境变量
DISPATCH_DRY_RUN=true

# 3. 清理审核数据
psql "$DATABASE_URL" -c "
  TRUNCATE ai.review_record CASCADE;
  UPDATE ai.action_proposal SET apply_status = 'proposed';
"

# 4. 回滚代码
git checkout migration/go-postgres~1
make build && make api && make worker
```

**RTO**: 20 分钟  
**RPO**: 可能丢失最近的审核记录

### 6.2 灾难恢复

**场景**: PostgreSQL 数据损坏

```bash
# 1. 停止所有服务
docker compose down

# 2. 恢复 PostgreSQL 数据
docker compose up -d postgres
pg_restore -d baxi backup.dump

# 3. 重新运行 pipeline
make pipeline

# 4. 验证
make pipeline-compare
```

**备份策略**:
```bash
# 每日备份
pg_dump -Fc baxi > backup_$(date +%Y%m%d).dump

# 保留 7 天
find . -name "backup_*.dump" -mtime +7 -delete
```

---

## 7. 监控和告警策略

### 7.1 监控指标

#### 系统指标

| 指标 | 阈值 | 告警级别 |
|------|------|---------|
| API 延迟 p99 | > 500ms | P1 |
| API 错误率 | > 1% | P0 |
| DB 连接池使用率 | > 80% | P1 |
| Pipeline 运行时间 | > 30min | P2 |
| Outbox 待处理数 | > 100 | P1 |

#### 业务指标

| 指标 | 阈值 | 告警级别 |
|------|------|---------|
| Alert 生成数 | 偏差 > 20% | P1 |
| Task 完成率 | < 50% | P2 |
| Outbox 分发成功率 | < 90% | P1 |
| 决策案例数 | 异常增长 | P2 |

### 7.2 告警通道

```yaml
# config/alert_channels.yml
channels:
  - name: feishu
    type: webhook
    url: ${FEISHU_WEBHOOK_URL}
    severity: [P0, P1]
  
  - name: email
    type: email
    to: team@example.com
    severity: [P0]
  
  - name: pagerduty
    type: pagerduty
    service_key: ${PAGERDUTY_KEY}
    severity: [P0]
```

### 7.3 日志策略

**结构化日志** (JSON):
```json
{
  "timestamp": "2026-05-27T10:00:00Z",
  "level": "info",
  "service": "baxi-api",
  "request_id": "req_abc123",
  "method": "GET",
  "path": "/api/v1/alerts",
  "status": 200,
  "duration_ms": 45
}
```

**日志聚合**:
- 开发: stdout
- 生产: stdout → Docker logs → 日志收集器

### 7.4 健康检查

```bash
# API 健康检查
curl http://localhost:8080/api/v1/health
# {"status":"ok","service":"baxi-api","db":"connected"}

# Worker 健康检查
curl http://localhost:8081/health
# {"status":"ok","service":"baxi-worker","dispatch":"running"}
```

---

## 8. 实施路线图

### 8.1 阶段时间线

```
Week 1-2: Phase 4 完成
  ├── Wave 1: 基础端点
  ├── Wave 2: 数据查询端点
  ├── Wave 3: 日志端点
  └── Wave 4: 治理端点

Week 3: Phase 4 验证
  ├── 影子模式测试
  ├── API 基线对比
  └── 前端切换准备

Week 4: Phase 5 实施
  ├── ConfigLoader
  ├── ObjectRegistry
  ├── GovernanceService
  └── 6 个治理端点

Week 5: Phase 5 验证 + Phase 6 启动
  ├── 治理端点测试
  ├── Decision Case 创建
  └── LLM-safe 上下文

Week 6: Phase 6 完成
  ├── 决策生成
  ├── Action Proposal
  └── 6 个决策端点

Week 7: Phase 7 实施
  ├── Review 端点
  ├── Action 执行
  ├── Outbox Worker
  └── Dispatch 机制

Week 8: Phase 7 验证 + 收尾
  ├── 端到端测试
  ├── 性能测试
  └── 文档更新
```

### 8.2 里程碑

| 里程碑 | 目标日期 | 验收标准 |
|--------|---------|---------|
| M1: API 迁移完成 | Week 3 | 11 端点通过基线对比 |
| M2: 治理运行时 | Week 4 | 6 端点 + 配置加载 |
| M3: 决策管道 | Week 6 | 决策案例创建 + 上下文 |
| M4: 审核执行 | Week 7 | 审核→执行→分发 |
| M5: 生产就绪 | Week 8 | 全链路测试通过 |

### 8.3 人力需求

| 角色 | 人数 | 职责 |
|------|------|------|
| Go 后端 | 1-2 | API、Pipeline、Worker |
| Python 维护 | 0.5 | 回归测试、Bug 修复 |
| 前端 | 0.5 | 切换配置、测试 |
| QA | 0.5 | 基线对比、E2E 测试 |

---

## 9. 风险管理

### 9.1 风险矩阵

| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|---------|
| 数据不一致 | 中 | 高 | 基线对比 + 自动验证 |
| 性能下降 | 低 | 中 | 性能测试 + 优化 |
| 回滚失败 | 低 | 高 | 定期演练 + 备份 |
| 人力不足 | 中 | 中 | 优先级排序 + 自动化 |
| 依赖阻塞 | 低 | 中 | 提前识别 + 备选方案 |

### 9.2 应急预案

**场景 1: API 迁移后错误率飙升**
```bash
# 立即回滚
# 1. 前端切回 Python
# 2. 通知团队
# 3. 分析日志
# 4. 修复后重新迁移
```

**场景 2: Pipeline 输出偏差**
```bash
# 1. 暂停迁移
# 2. 对比详细日志
# 3. 修复 Go pipeline
# 4. 重新运行验证
```

**场景 3: 数据库性能问题**
```bash
# 1. 检查慢查询
# 2. 添加索引
# 3. 优化查询
# 4. 必要时扩容
```

---

## 10. 成功标准

### 10.1 技术指标

- [ ] 所有 24 个 API 端点迁移到 Go
- [ ] API 响应与 Python 基线 100% 兼容 (忽略已知差异)
- [ ] Pipeline 输出行数偏差 < 1%
- [ ] Go 测试覆盖率 > 60%
- [ ] 集成测试通过率 100%
- [ ] p99 延迟 < 500ms

### 10.2 业务指标

- [ ] 零停机迁移
- [ ] 零数据丢失
- [ ] 所有治理配置正确加载
- [ ] 决策管道端到端工作
- [ ] 审核→执行流程完整

### 10.3 运维指标

- [ ] 监控告警配置完成
- [ ] 日志聚合正常
- [ ] 备份恢复演练通过
- [ ] 文档更新完成

---

## 附录 A: 关键命令速查

```bash
# 启动环境
make up                    # 启动 PostgreSQL
make migrate               # 运行迁移
make api                   # 启动 Go API
make worker                # 启动 Go Worker

# Pipeline
make pipeline              # 运行完整管道
make pipeline-compare      # 对比基线
make pipeline-ingest       # 仅数据摄入

# 测试
make test-go               # Go 测试
make test-python           # Python 测试
make test-all              # 全部测试

# 验证
make api-compare           # API 基线对比
make governance-check      # 治理配置检查

# 回滚
make migrate-down          # 回滚迁移
git checkout legacy/python-sqlite  # 切回 Python
```

---

## 附录 B: 文件结构参考

```
baxi/
├── cmd/                    # Go 入口点
│   ├── baxi-api/          # API 服务器
│   ├── baxi-cli/          # CLI 工具
│   └── baxi-worker/       # 后台 Worker
├── internal/               # Go 核心包
│   ├── api/               # HTTP 处理器
│   ├── pipeline/          # 数据管道
│   ├── repository/        # 数据访问层
│   └── service/           # 业务逻辑层
├── migrations/             # Goose 迁移文件
├── config/                 # YAML 治理配置
├── api/                    # Python FastAPI (legacy)
├── services/               # Python 业务服务
├── frontend/               # React 前端
├── migration_baseline/     # 基线数据
└── docs/migration/         # 迁移文档
```

---

## 附录 C: 参考文档

- [迁移总计划](docs/migration/go-postgres-migration-plan.md)
- [Phase 1: Foundation](docs/migration/phase-1-foundation.md)
- [Phase 2: Schema Design](docs/migration/phase-2-schema-design.md)
- [Phase 3: Pipeline Migration](docs/migration/phase-3-pipeline-migration-plan.md)
- [Phase 3: Parity Report](docs/migration/phase-3-parity-report.md)
- [Phase 4: API Migration](docs/migration/phase-4-api-migration-plan.md)
- [Phase 5: Governance Runtime](docs/migration/phase-5-governance-ontology-runtime-plan.md)
- [Phase 6: Decision/LLM](docs/migration/phase-6-decision-case-llm-context-plan.md)
- [Phase 7: Review/Action/Outbox](docs/migration/phase-7-review-action-outbox-plan.md)

---

**文档维护者**: Migration Team  
**最后更新**: 2026-05-27  
**下次评审**: Phase 4 完成后
