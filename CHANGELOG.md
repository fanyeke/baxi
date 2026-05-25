# 变更日志

## v0.5.3 (2026-05-24)

### 安全加固
- **修复路径遍历漏洞**：`api/routers/governance.py` 的 `_load_yaml()` 现在验证文件名，防止 `../etc/passwd` 等路径遍历攻击
- **统一错误处理**：`api/routers/governance.py` 和 `api/routers/diagnosis.py` 统一使用 `APIError`，消除 `HTTPException`/`JSONResponse` 混用

### 架构清理
- **配置迁移**：`scripts/config.py` → `core/config.py`，保留向后兼容 shim
- **新增 `core/` 模块**：集中管理路径常量、环境变量读取、Feishu 凭据加载
- **新增 `pipeline/` 模块**：`pipeline/steps.py` + `pipeline/runner.py`，替代 `subprocess.run` 的直接函数调用管道

### CI/CD
- **GitHub Actions**：新增 `.github/workflows/ci.yml`（pytest + ruff + coverage）

### 数据库
- **外键约束**：Migration `010_add_foreign_keys.sql` 添加 3 个外键约束
- **PRAGMA foreign_keys = ON**：所有新连接自动启用外键检查
- **补充迁移**：新增 `001_base_schema.sql`、`002_seed_data.sql`

### 测试补强
- **新增 75 个测试**：总计 385 个用例，覆盖率从 ~60% → **86%**
- **诊断服务 100% 覆盖**：`tests/test_diagnosis_service.py`（17 个测试）
- **状态服务 100% 覆盖**：`tests/test_status_service.py`（10 个测试）
- **告警服务 92% 覆盖**：`tests/test_alert_service_extended.py`（24 个测试）
- **任务服务 91% 覆盖**：`tests/test_task_service_extended.py`（24 个测试）
- **测试副作用修复**：`tests/conftest.py` 添加内存数据库 fixture，消除真实 DB 依赖

### 代码质量
- **修复重复连接**：`services/task_service.py` 的 `get_tasks_with_count()` 现在只创建一次连接
- **可配置用户身份**：`api/dependencies.py` 的 `DEFAULT_USER` 改为从环境变量读取（默认 `qoder`）
- **依赖清理**：移除未使用的 `httpx` 和 `starlette`
- **版本统一**：所有版本引用更新为 `0.5.3`

---

## v0.5.2 (2026-05-20)

- 新增 `/logs/diagnosis` 端点
- 前端硬化（P0 修复 + 日志诊断优化）

## v0.5.1 (2026-05-18)

- 首个 API 网关：FastAPI + 6 个核心端点 + Bearer Token + 限流

## v0.5.0 (2026-05-15)

- Governance 7 个端点 + response model
- 请求 schema validator
- 测试隔离优化

## v0.4.0 (2026-05-10)

- 分发适配器：Feishu / GitHub / LocalCLI / Manual

## v0.3.1 (2026-05-05)

- 飞书沙盘集成
- 决策质量校准

## v0.3.0 (2026-05-01)

- 维度级异常检测（seller/category/region）

## v0.2.0 (2026-04-25)

- SQLite 后端 + 12 表 Schema + 配置化治理

## v0.1.0 (2026-04-20)

- 规则驱动决策沙盘（heuristic）
