# 测试指南

> 33 个测试文件，385+ 用例，覆盖率 86%。

## 快速开始

```bash
# 运行全部测试
pytest

# 运行全部测试（简洁输出）
pytest -q

# 运行特定测试文件
pytest tests/test_diagnosis_service.py -v

# 运行集成测试（需要真实数据库）
pytest -m integration

# 排除集成测试
pytest -m "not integration"
```

---

## 测试分类

### 单元测试（默认）

使用内存数据库，无外部依赖，运行快：

- `test_diagnosis_service.py` — 诊断服务（100% 覆盖）
- `test_status_service.py` — 状态服务（100% 覆盖）
- `test_alert_service_extended.py` — 告警服务（92% 覆盖）
- `test_task_service_extended.py` — 任务服务（91% 覆盖）
- `test_governance_security.py` — 安全测试（路径遍历）
- `test_pyproject_deps.py` — 依赖验证

### 集成测试（`-m integration`）

使用真实数据库 `data/olist_ops.db`，验证端到端行为：

- `test_db_schema.py` — Schema 验证
- `test_db_ingestion.py` — 数据摄取
- `test_db_metrics.py` — 指标计算
- `test_db_rule_engine.py` — 规则引擎
- `test_db_dimension_metrics.py` — 维度指标
- `test_db_dimensional_rules.py` — 维度规则
- `test_db_feishu_export.py` — Feishu 导出
- `test_db_event_outbox_routing.py` — Outbox 路由

### API 测试

- `test_api_gateway.py` — FastAPI 端点集成测试
- `test_pipeline_api.py` — Pipeline API 测试
- `test_governance_api.py` — Governance API 测试
- `test_logs_api.py` — 日志 API 测试
- `test_feishu_api.py` — 飞书 API 测试

---

## Fixtures

### 内存数据库

```python
def test_something(in_memory_db):
    # in_memory_db 是 sqlite3.Connection，已加载 schema
    cur = in_memory_db.execute("SELECT COUNT(*) FROM alert_events")
    assert cur.fetchone()[0] == 0
```

### 临时文件数据库

```python
def test_something(temp_db_path):
    # temp_db_path 是 Path，指向带 schema 的临时 SQLite 文件
    # 适用于需要文件路径的测试（如 subprocess）
    pass
```

---

## 覆盖率

```bash
# 运行并生成覆盖率报告（默认已启用）
pytest --cov

# 指定模块覆盖率
pytest tests/test_diagnosis_service.py --cov=services.diagnosis_service

# 覆盖率阈值（失败低于阈值）
pytest --cov-fail-under=60
```

---

## 测试结构

```
tests/
├── conftest.py              # 共享 fixtures（内存 DB、临时 DB）
├── test_api_gateway.py      # API 集成
├── test_db_*.py             # 数据库集成测试（-m integration）
├── test_*_service.py        # 服务层单元测试
├── test_governance_*.py     # Governance + 安全
├── test_feishu_*.py         # 飞书相关
├── test_pipeline_api.py     # Pipeline API
└── test_pyproject_deps.py   # 依赖验证
```

---

## 编写新测试

```python
import pytest
from services.alert_service import get_alerts

def test_get_alerts_empty(in_memory_db):
    """测试空表返回空列表."""
    result = get_alerts(conn=in_memory_db)
    assert result == []
```

**原则**：
- 优先使用 `in_memory_db` fixture（隔离、快速）
- 需要真实数据时用 `pytest.mark.integration`
- 不要依赖真实数据库文件（除非集成测试）
