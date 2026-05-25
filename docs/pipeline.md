# Pipeline 管道系统

> `pipeline/steps.py` + `pipeline/runner.py` — 替代 `subprocess.run` 的直接函数调用管道。

## 设计原则

- **函数替代子进程**：每个管道步骤都是纯 Python 函数，消除 `subprocess.run` 开销
- **统一编排**：`run_pipeline()` 负责步骤顺序、错误处理和报告
- **可测试**：每个步骤可独立调用，便于单元测试

---

## 管道步骤（9 个）

| 步骤函数 | 职责 | 对应原脚本 |
|----------|------|-----------|
| `step_db_init` | 初始化数据库 schema 和索引 | `db_init.py` |
| `step_db_ingest` | 数据摄取（CSV → SQLite） | `db_ingest.py` |
| `step_db_calculate_metrics` | 计算每日全局指标 | `db_calculate_metrics.py` |
| `step_db_calculate_dimension_metrics` | 计算维度级指标 | `db_calculate_dimension_metrics.py` |
| `step_db_rule_engine` | 规则引擎告警检测 | `db_rule_engine.py` |
| `step_db_dimensional_rule_engine` | 维度级规则检测 | `db_dimensional_rule_engine.py` |
| `step_db_generate_recommendations` | 生成策略建议 | `db_generate_recommendations.py` |
| `step_db_export_feishu` | 导出 Feishu CSV | `db_export_feishu.py` |
| `step_db_trigger_simulator` | 触发模拟器 | `db_trigger_simulator.py` |

---

## 运行管道

### 命令行入口

```bash
# 完整管道
python3 scripts/run_db_pipeline.py --mode full

# Dry-run（无副作用，默认行为）
python3 scripts/run_db_pipeline.py --mode full

# Apply（实际写入 DB）
python3 scripts/run_db_pipeline.py --mode full --apply

# 包含维度步骤
python3 scripts/run_db_pipeline.py --mode full --dimensional

# 指定日期范围
python3 scripts/run_db_pipeline.py --mode range --start 2017-01-01 --end 2017-12-31
```

### Python API

```python
from pipeline.runner import run_pipeline

# 完整管道
success = run_pipeline(mode="full", dry_run=False)

# 仅运行部分步骤（跳过 ingestion）
success = run_pipeline(
    mode="full",
    steps=["db_calculate_metrics", "db_rule_engine", "db_export_feishu"]
)
```

### 参数说明

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `mode` | str | `"full"` | `"full"` 或 `"range"` |
| `start` | str | `None` | 日期范围起始（`mode=range` 时必填） |
| `end` | str | `None` | 日期范围结束 |
| `dimensional` | bool | `False` | 是否包含维度级步骤 |
| `db_path` | str | `None` | 自定义 DB 路径（默认 `data/olist_ops.db`） |
| `dry_run` | bool | `True` | `True`=仅模拟不写入（默认安全）；传 `False` 或 `--apply` 才实际更新 outbox |
| `steps` | list | `None` | 指定步骤列表（`None`=全部） |

---

## 默认步骤列表

### 非维度模式（默认）

```
db_init → db_ingest → db_calculate_metrics → db_rule_engine
  → db_generate_recommendations → db_export_feishu → db_trigger_simulator
```

### 维度模式（`--dimensional`）

额外插入两个步骤：
```
... → db_calculate_metrics → db_calculate_dimension_metrics
  → db_rule_engine → db_dimensional_rule_engine → ...
```

---

## 错误处理

- **单步失败即中断**：任何步骤返回 `False` 或抛出异常，管道立即停止
- **自动报告**：每步耗时和结果打印到 stdout
- **返回值**：`run_pipeline()` 返回 `bool`，`True`=全部成功

---

## 迁移自旧子进程模式

```python
# ❌ 旧方式（subprocess）
subprocess.run(["python3", "scripts/db_ingest.py", "--mode", "full"])

# ✅ 新方式（直接函数调用）
from pipeline.steps import step_db_ingest
step_db_ingest(mode="full")
```
