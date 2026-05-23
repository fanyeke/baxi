# Olist Brazil E-Commerce Data Pipeline — 运维手册

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** End-to-end runbook for deploying and operating the data pipeline, from environment setup to Feishu sync.

**Architecture:** Single-repo Python pipeline with 4 entry points (daily simulation, full evaluation, Feishu sync, status pullback), built on pandas/numpy data model with heuristic/AI decision engine and Feishu Bitable sync.

**Tech Stack:** Python 3.10+, pandas, numpy, pyyaml, requests, pydantic, openai, python-dotenv, matplotlib, seaborn

---

## 前置条件

- Python ≥ 3.10
- Git installed
- Kaggle account (for data download)

## 环境安装

```bash
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
```

## 数据下载

Two Kaggle datasets needed:

```bash
# Main dataset (9 CSVs, ~100K orders)
kaggle datasets download -d olistbr/brazilian-ecommerce -p data/raw/
unzip data/raw/brazilian-ecommerce.zip -d data/raw/
# Marketing funnel (2 CSVs, ~8K leads)
kaggle datasets download -d olistbr/marketing-funnel -p data/raw/
unzip data/raw/marketing-funnel.zip -d data/raw/
```

Confirm: `ls data/raw/olist_orders_dataset.csv`

## 构建基础表

```bash
python3 scripts/phase02_build_data_model.py
```

Produces: `data/interim/order_level_base.csv` (99,441 rows), `data/interim/item_level_base.csv` (112,650 rows)

## 入口 1：每日模拟模式

```bash
python3 scripts/run_daily_pipeline.py
```

8-step sequential pipeline: simulate ingestion → data quality → daily metrics → alert detection → AIP objects → context bundle → wake agent → feishu sandbox. Increments `data/system/ingestion_state.json` by 1 day per run.

## 入口 2：全量评估模式

```bash
python3 scripts/run_full_pipeline.py
```

5-step --mode full pipeline: daily_metrics_full → metric_alerts_full → aip_context_bundle_full → AI decision engine → feishu sandbox full. Produces `_full` suffixed outputs in data/ads/, data/aip/, outputs/ai/, data/feishu/.

## 入口 3：飞书同步

```bash
# Dry-run (preview only, no API calls)
python3 scripts/sync_feishu_bitable.py --all --dry-run
# Apply (real sync, requires FEISHU_* env vars)
python3 scripts/sync_feishu_bitable.py --all --apply
```

Syncs 5 tables: daily_metrics, alert_events, strategy_recommendations, action_tasks, execution_reviews. Requires `.env` with real Feishu credentials. See `config/feishu_table_ids.yml` for table ID configuration.

## 入口 4：状态回流

```bash
python3 scripts/pull_feishu_status.py --dry-run
python3 scripts/pull_feishu_status.py --apply
```

Pulls task status and review retro from Feishu back to local CSV.

## 常见错误

- **"No such file: olist_orders_dataset.csv"**: Run data download step first
- **"ModuleNotFoundError: No module named 'pandas'"**: Run `pip install -r requirements.txt`
- **"LLM API key not found"**: Copy `.env.example` to `.env` and set LLM_API_KEY (or skip: AI engine will fall back to heuristic rules)
- **"Feishu auth failed"**: Set real FEISHU_APP_ID/FEISHU_APP_SECRET in `.env` (or use --dry-run for local-only)
- **"ingestion_state.json not found"**: Script auto-initializes on first run (start date: 2016-09-04)
- **"data_quality_checks returned WARN"**: Non-critical; some quality checks may fail on small daily data. This is a known issue tracked for Phase II.

## 当前限制

- **不是大模型决策**：策略生成基于启发式规则（decision_source=heuristic），LLM API 调用仅作可选增强
- **飞书同步需手动建表**：脚本只读写数据，不自动创建飞书多维表格
- **FROZEN 脚本不能直接运行**：Phase 1-7 的 EDA 脚本（phase03-07）路径已失效，其分析结果已固化在 outputs/ 和 reports/ 中
- **无定时任务**：所有命令需手动触发，未配置 cron 或调度器
- **无自动触发**：异常检测不会自动通知，需通过飞书同步后人工查看
