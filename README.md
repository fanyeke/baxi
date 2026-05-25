# Olist 巴西电商数据分析项目

## 项目概述

本项目对巴西电商 Olist 的公开数据集进行探索性分析，旨在理解数据结构、识别数据质量问题，并挖掘业务洞察。

## 数据集介绍

本项目包含两个互补的数据集：

### 1. 主数据集：Brazilian E-Commerce by Olist

包含 2016 年至 2018 年间巴西多个城市的电商订单信息，涵盖客户、订单、产品、卖家、支付、评价等多个维度的数据。

- **数据集**: [Brazilian E-Commerce Public Dataset by Olist](https://www.kaggle.com/datasets/olistbr/brazilian-ecommerce)
- **时间范围**: 2016-09 至 2018-10
- **记录数量**: 约 100,000 条订单
- **文件数量**: 9 个 CSV 文件

### 2. 营销漏斗数据集：Marketing Funnel by Olist ⏳

记录 Olist 如何获取卖家（商家）以及这些卖家的转化过程，是理解 Olist 商业模式的关键数据。

- **数据集**: [Marketing Funnel Dataset](https://www.kaggle.com/datasets/olistbr/marketing-funnel)
- **记录数量**: 约 8,000 条营销线索
- **文件数量**: 2 个 CSV 文件
- **状态**: **待下载**（见 `docs/marketing_funnel_guide.md`）

**关联关系**: Marketing Funnel 数据集通过 `seller_id` 与主数据集的卖家表关联，形成完整的"营销 → 转化 → 销售"业务链路。

## 项目阶段

### 第一阶段（Phase 1）：原始数据理解 ✅ (FROZEN)

本阶段完成时间：2026-05-20

#### 目标

- 自动识别所有原始 CSV 文件，不修改原始数据
- 对每张表进行基础画像分析
- 初步判断表之间的关联关系
- 产出可复现的分析代码和结果文档

#### 数据文件

本项目包含以下 9 个数据文件：

1. **olist_customers_dataset.csv** - 客户数据（99,441 行）
2. **olist_geolocation_dataset.csv** - 地理位置数据（1,000,163 行）
3. **olist_order_items_dataset.csv** - 订单商品数据（112,650 行）
4. **olist_order_payments_dataset.csv** - 订单支付数据（103,886 行）
5. **olist_order_reviews_dataset.csv** - 订单评价数据（99,224 行）
6. **olist_orders_dataset.csv** - 订单主数据（99,441 行）
7. **olist_products_dataset.csv** - 产品数据（32,951 行）
8. **olist_sellers_dataset.csv** - 卖家数据（3,095 行）
9. **product_category_name_translation.csv** - 产品类别翻译（71 行）

#### 产出文件

##### 1. 数据画像汇总
- **文件**: `outputs/data_profile_summary.csv`
- **内容**: 每张表的字段级统计信息
- **指标**: 行数、列数、数据类型、缺失率、唯一值数、主键/外键推断等

##### 2. 数据字典
- **文件**: `docs/data_dictionary.md`
- **内容**: 每张表的详细字段说明
- **包括**: 字段类型、缺失情况、日期范围、可能的主键和外键

##### 3. 表关系说明
- **文件**: `docs/table_relationship_notes.md`
- **内容**: 表之间的关联关系分析
- **包括**: 外键关系推断、数据质量说明

#### 关键发现

##### 数据质量

1. **重复数据**:
   - `olist_geolocation_dataset` 存在 261,831 行重复数据（26.18%）

2. **缺失值**:
   - `olist_order_reviews_dataset`:
     - `review_comment_title`: 87,656 条缺失（88.34%）
     - `review_comment_message`: 58,247 条缺失（58.70%）
   - `olist_orders_dataset`:
     - `order_approved_at`: 160 条缺失（0.16%）
     - `order_delivered_carrier_date`: 1,783 条缺失（1.79%）
     - `order_delivered_customer_date`: 2,965 条缺失（2.98%）
   - `olist_products_dataset`:
     - 多个字段存在 610 条缺失（1.85%）
     - 产品尺寸字段存在 2 条缺失（0.01%）

##### 主键识别

- `olist_customers_dataset`: `customer_id` (唯一值)
- `olist_orders_dataset`: `order_id` (唯一值)
- `olist_products_dataset`: `product_id` (唯一值)
- `olist_sellers_dataset`: `seller_id` (唯一值)

##### 表关系

核心关联结构：
```
olist_customers (客户)
    ↓ customer_id
olist_orders (订单)
    ↓ order_id
    ├── olist_order_items (订单商品)
    │       ↓ product_id
    │   olist_products (产品)
    │       ↓ seller_id
    │   olist_sellers (卖家)
    ├── olist_order_payments (支付信息)
    └── olist_order_reviews (订单评价)

olist_geolocation (地理位置) ← 通过 zip_code_prefix 关联
```

#### 分析脚本

- **文件**: `explore_data.py`
- **功能**: 自动扫描所有 CSV 文件并生成完整的数据画像
- **可复现性**: 运行脚本即可重新生成所有分析结果

#### 运行方法

```bash
# 安装依赖
pip install pandas

# 运行数据探索脚本（主数据集）
python3 explore_data.py

# 运行扩展脚本（包含 Marketing Funnel 数据集）
python3 explore_data_extended.py
```

### 第一阶段补充：Marketing Funnel 数据集分析 ⏳

#### 状态说明

Marketing Funnel 数据集尚未下载到项目中。下载完成后，将自动补充分析。

#### 补充内容

添加 Marketing Funnel 数据集后，将分析以下两个表：

1. **olist_marketing_qualified_leads_dataset.csv** (MQL)
   - 营销合格线索记录
   - 约 8,000 条数据
   - 字段包括：mql_id、first_contact_date、landing_page、origin 等

2. **olist_closed_deals_dataset.csv** (Closed Deals)
   - 成交交易记录
   - 记录成功转化为平台卖家的数据
   - 通过 seller_id 关联主数据集

#### 下载指引

详细的下载步骤请参考：**docs/marketing_funnel_guide.md**

快速下载：
```bash
# 从 Kaggle 下载（需要 Kaggle 账号）
# https://www.kaggle.com/datasets/olistbr/marketing-funnel

# 或使用 Kaggle API
kaggle datasets download -d olistbr/marketing-funnel
unzip marketing-funnel.zip

# 确保文件位于项目根目录
ls olist_marketing_qualified_leads_dataset.csv
ls olist_closed_deals_dataset.csv
```

#### 补充分析后的产出

下载并运行 `python3 explore_data_extended.py` 后，将更新：

1. **outputs/data_profile_summary.csv** - 新增约 20 行营销漏斗字段分析
2. **docs/data_dictionary.md** - 新增营销漏斗数据集章节
3. **docs/table_relationship_notes.md** - 新增跨数据集关联关系
4. **README.md** - 更新数据集统计和关联说明

#### 业务价值补充

Marketing Funnel 数据集将建立完整的业务链路分析：

```
营销获客 (MQL)
    ↓ mql_id
成交转化 (Closed Deals)
    ↓ seller_id
卖家入驻 (Sellers)
    ↓ seller_id
订单销售 (Order Items)
    ↓ product_id
产品交付 (Products + Orders)
```

这将支持：
- 营销渠道效果分析
- 线索转化率评估
- landing page 优化建议
- 营销投入与销售业绩关联分析

### 第二阶段（Phase 2）：核心数据模型搭建 ✅ (FROZEN)

本阶段完成时间：2026-05-20

#### 目标

- 验证关键关联关系，检查 join 后的数据完整性
- 输出正式 ER 关系说明与图谱
- 构建标准化分析基础表
- 说明表粒度、字段来源、适用场景

#### 关键产出

##### 1. 关系验证结果

验证了 7 个关键 join 关系，记录完整的输入输出统计：

| 关系 | 左表行数 | 右表行数 | 输出行数 | 放大倍数 |
|------|---------|---------|---------|---------|
| customers → orders | 99,441 | 99,441 | 99,441 | 1.00x |
| orders → order_items | 99,441 | 112,650 | 113,425 | 1.14x |
| order_items → products | 112,650 | 32,951 | 112,650 | 1.00x |
| order_items → sellers | 112,650 | 3,095 | 112,650 | 1.00x |
| orders → payments | 99,441 | 103,886 | 103,887 | 1.04x |
| orders → reviews | 99,441 | 99,224 | 99,992 | 1.01x |
| products → category_translation | 32,951 | 71 | 32,951 | 1.00x |

**关键发现**：
- orders → order_items 正常放大 1.14x（多商品订单）
- orders → payments 正常放大 1.04x（组合支付）
- orders → reviews 正常放大 1.01x（多条评价）
- 无数据丢失，主表数据完整保留

##### 2. ER 关系说明

- **文件**: `docs/entity_relationships.md` (243 行)
- **内容**: 完整的表关系详细说明
- **图谱**: `outputs/erd.mmd` (Mermaid ER 图，81 行)

**星型模型结构**：
- 事实表：ORDERS（订单中心）
- 维度表：CUSTOMERS, PRODUCTS, SELLERS
- 明细表：ORDER_ITEMS
- 辅助表：PAYMENTS, REVIEWS

##### 3. 分析基础表

#### order_level_base.csv

- **路径**: `data/interim/order_level_base.csv`
- **粒度**: 一行一个订单
- **行数**: 99,441 行 × 22 列 (35 MB)
- **来源**: orders + customers + payments汇总 + reviews最新

**适用场景**：
- 订单转化分析（订单状态、交付时间）
- 支付方式分析（支付类型、分期情况）
- 客户满意度分析（评价分数分布）
- 客户行为分析（复购率、购买频次）

#### item_level_base.csv

- **路径**: `data/interim/item_level_base.csv`
- **粒度**: 一行一个订单商品项
- **行数**: 112,650 行 × 36 列 (51 MB)
- **来源**: order_items + products + sellers + 订单信息

**适用场景**：
- 产品销售分析（销量排名、品类分布）
- 卖家绩效分析（销售额、评价分数）
- 价格与运费分析（价格分布、运费占比）
- 商品组合分析（客单价、商品数量）

##### 4. 基础表说明文档

- **文件**: `docs/analysis_base_tables.md` (212 行)
- **内容**: 粒度、字段来源、适用场景、使用限制
- **对比**: 两张表的差异与选择建议

#### 分析脚本

- **文件**: `build_data_model.py` (数据模型构建)
- **文件**: `generate_docs.py` (文档生成)
- **可复现性**: 运行脚本即可重新生成所有结果

#### 运行方法

```bash
# 构建数据模型（包含验证和基础表）
python3 build_data_model.py

# 生成补充文档（如果主脚本中断）
python3 generate_docs.py
```

## 后续计划

**当前版本**: v0.5.3（含规则决策引擎、SQLite 后端、FastAPI 网关、React 控制台、飞书集成、数据治理层）

### Phase 3：全局业务分析 ✅ (FROZEN)

月度趋势分析（GMV/订单量/客户增长）、订单状态分布、支付方式分析、品类销售排名、卖家GMV排名、地域分析。
- 脚本: `scripts/phase03_overall_business_analysis.py`
- 报告: `reports/overall_business_analysis.md`
- 产出: 12 CSV + 13 图表

### Phase 4：履约与客户体验分析 ✅ (FROZEN)

订单交付时效、评价分数影响、地域/品类/卖家交付表现对比。
- 脚本: `scripts/phase04_fulfillment_experience_analysis.py`
- 报告: `reports/fulfillment_customer_experience_analysis.md`
- 产出: 14 CSV + 11 图表

### Phase 5：营销漏斗 + 卖家绩效分析 ✅ (FROZEN)

营销线索来源、渠道质量、卖家GMV排名、质量修订。
- 脚本: `scripts/phase05_marketing_funnel_analysis.py`, `phase05_seller_performance_analysis.py`, `phase05_quality_revision.py`
- 报告: `reports/marketing_funnel_seller_growth_analysis.md`

### Phase 6：（跳过）

### Phase 7：决策沙盘模拟器 ✅ (FROZEN)

What-If 场景模拟，GMV/评分联动预测。
- 脚本: `scripts/phase07_simulation_engine.py`, `phase07_calibration_revision.py`
- 报告: `reports/scenario_simulation_analysis.md`, `reports/simulation_readiness_report.md`

### Phase 8：飞书沙盘准备

飞书多维表格 scenario 参数构建。
- 产出: `data/processed/feishu_*.csv`

### Phase 9：飞书沙盘集成与部署 ✅

飞书部署、UI 验收、执行报告。
- 文档: `docs/phase9_*.md`

### Phase 10：Waker 只读验证 ✅

读写契约验证。
- 文档: `docs/waker_read_write_contract.md`, `docs/phase10_*.md`

### 版本演进

| 版本 | 核心能力 | 状态 |
|------|---------|------|
| v0.1 | 规则驱动决策沙盘 (heuristic) | ✅ DONE |
| v0.2 | SQLite 后端 + 12 表 Schema + 配置化治理 | ✅ DONE |
| v0.3 | 维度级异常检测 (seller/category/region) | ✅ DONE |
| v0.3.1 | 飞书沙盘集成 + 决策质量校准 | ✅ DONE |
| v0.4 | 分发适配器 (Feishu/GitHub/Local/Manual) | ✅ DONE |
| v0.5 | API 网关 (FastAPI:8765, OpenAPI, Bearer Token) | ✅ DONE |
| v0.5.3 | React 控制台 Alpha (7 pages, TanStack Query) | ✅ DONE |
| v0.5.3 | 控制台 Beta 硬化 (P0修复 + 日志诊断) | ✅ DONE |
| v0.5.3 | 数据治理层 + API 接口参考文档 + Schema 校验 + 测试隔离优化 | ✅ DONE |
| Phase I | 全量数据 + AI 决策引擎 (LLM 代码就绪，待激活) | 🟡 核心完成 |
| Phase II+ | 维度告警扩展 / 真实 LLM 决策 / 自动调度 | ❌ 未启动 |

## 运行说明

### 脚本状态：❄️ FROZEN

所有分析脚本已集中至 `scripts/` 目录并重命名为 `phaseXX_*.py` 格式。由于原始数据文件已从根目录移至 `data/raw/`，**脚本中的硬编码路径已失效，不能直接运行**。

详细说明见：[scripts/_FROZEN.md](scripts/_FROZEN.md)

### 🚀 API 网关（活跃开发）

`core/config.py` 已实现集中化路径常量管理。FastAPI API 网关是项目当前最活跃的对外接口。

**快速启动**：

```bash
# 1. 安装依赖
pip install -e .

# 2. 配置环境变量（参考 .env.example）
cat > .env <<EOF
API_BEARER_TOKEN=$(python3 -c "import secrets; print(secrets.token_urlsafe(32))")
FEISHU_APP_ID=YOUR_APP_ID
FEISHU_APP_SECRET=YOUR_APP_SECRET
FEISHU_BASE_APP_TOKEN=YOUR_APP_TOKEN
FEISHU_CHAT_ID=YOUR_CHAT_ID
EOF

# 3. 启动 API
python3 scripts/run_api.py --port 8765
```

**验证**：

```bash
# 健康检查（无需认证）
curl http://127.0.0.1:8765/api/v1/health
# → {"status":"ok","version":"0.5.3","db_connected":true}

# 启用 OpenAPI 文档（可选）
ENABLE_DOCS=1 python3 scripts/run_api.py
# → http://127.0.0.1:8765/docs 查看 Swagger UI
```

**📖 完整接口文档**：[`docs/API_REFERENCE.md`](docs/API_REFERENCE.md)

文档涵盖 21 个端点、统一错误格式、认证机制、速率限制、前端集成示例和状态机说明。

### Phase B 计划（已完成 ✅）

`core/config.py` 已实现集中管理路径常量。新代码应通过 `from core.config import` 引用路径。`scripts/config.py` 是向后兼容的 shim（已标记 DeprecationWarning），新代码不应再导入。

## 技术栈

- **Python 3.9+** - 主要编程语言
- **FastAPI + Uvicorn** - API 网关
- **Pydantic v2** - 请求/响应模型与校验
- **SQLite (WAL)** - 数据存储
- **Pandas / NumPy** - 数据处理和分析
- **React + TanStack Query** - 控制台前端
- **lark-oapi** - 飞书集成

## 项目结构概览

| 目录 | 内容 | 文件数 |
|------|------|--------|
| `api/` | FastAPI 网关（路由、认证、错误处理、Schema） | 18 .py |
| `services/` | 业务逻辑层（DB、告警、任务、调度、飞书、日志） | 9 .py |
| `adapters/` | 渠道适配层（Feishu、GitHub、LocalCLI、Manual） | 5 .py |
| `config/` | 治理/业务 YAML 配置（告警规则、指标、权限、血缘等） | 27 .yml |
| `core/` | 核心配置模块（路径常量、环境变量读取） | 1 .py |
| `pipeline/` | 管道步骤定义与编排（steps + runner） | 2 .py |
| `tests/` | 单元 + 集成测试 | 33 test_*.py |
| `data/raw/` | 原始数据源 | 12 |
| `data/interim/` | 中间分析表 | 3 |
| `data/processed/` | 飞书沙盘产物 | 4 |
| `scripts/` | 数据管道脚本 + 分析脚本（❄️ FROZEN） | 14 .py + 1 .md |
| `sql/` | Schema + 迁移脚本 | 1 schema + 10 migrations |
| `outputs/charts/` | 分析图表 | ~34 .png |
| `outputs/tables/` | 分析结果表 | ~47 .csv |
| `reports/` | 分析报告 | 6 .md |
| `docs/` | 技术文档（含接口参考、运行手册） | 19 文件 |
| `frontend/` | React 控制台（Vite + TanStack Query） | — |

## 项目结构

```
.
├── README.md                          # 项目说明
├── pyproject.toml                     # Python 包配置 + 依赖
├── pytest.ini                         # 测试配置（含覆盖率）
├── .env / .env.example                # 环境变量（API 令牌 + 飞书凭据）
│
├── api/                               # ★ FastAPI 网关 (v0.5.3)
│   ├── main.py                        # App factory + middleware
│   ├── auth.py                        # Bearer Token 常量时间校验
│   ├── dependencies.py                # DI: get_db / get_current_user
│   ├── errors.py                      # APIError + 统一错误处理
│   ├── schemas.py                     # Pydantic v2 模型（请求/响应）
│   ├── logging_config.py              # JSON 日志 + ContextVar request_id
│   └── routers/                       # 10 个路由分组
│       ├── health.py, status.py       # 系统状态
│       ├── alerts.py, tasks.py        # 业务数据查询
│       ├── outbox.py                  # 调度（Outbox 模式）
│       ├── feishu.py                  # 飞书双向同步
│       ├── pipeline.py                # 管道预览
│       ├── logs.py, diagnosis.py      # 日志与诊断
│       └── governance.py              # 数据治理层
│
├── services/                          # 业务逻辑层
│   ├── db_service.py                  # SQLite 连接 + 表校验
│   ├── alert_service.py               # 告警查询
│   ├── task_service.py                # 任务查询
│   ├── dispatch_service.py            # 事件调度（Outbox 核心）
│   ├── feishu_service.py              # 飞书操作
│   ├── pipeline_service.py            # 管道预览
│   ├── log_reader.py                  # JSONL 尾部读取
│   ├── diagnosis_service.py           # 跨源请求诊断
│   └── status_service.py              # 系统状态
│
├── adapters/                          # 渠道适配层（策略模式 + 工厂）
│   ├── base.py                        # ChannelAdapter ABC + resolve_adapter()
│   ├── feishu_adapter.py              # 飞书消息推送
│   ├── github_issue_adapter.py        # GitHub Issue（预览模式）
│   ├── local_cli_adapter.py           # 本地 CLI 审计日志
│   └── manual_adapter.py              # 手动审核队列
│
├── core/                              # 核心配置模块
│   └── config.py                      # 路径常量、环境变量读取、Feishu 凭据加载
│
├── config/                            # 治理 + 业务 YAML 配置（27 个）
│   ├── alert_rules.yml                # 告警规则
│   ├── metrics.yml                    # 指标定义
│   ├── feishu_table_ids.yml.example   # 飞书表 ID 模板
│   ├── data_classification.yml        # 数据分类
│   ├── access_policy.yml              # 访问策略
│   └── ...                            # 血缘、标记、检查点等
│
├── sql/                               # 数据库 Schema + 迁移
│   ├── schema.sql                     # 14 表定义
│   ├── indexes.sql                    # 22 个查询索引
│   └── migrations/                    # 6 个版本迁移
│
├── tests/                             # 测试（27 个文件，305+ 用例）
│   ├── conftest.py                    # 共享 fixtures
│   ├── test_api_gateway.py            # API 集成测试
│   └── test_*.py                      # ...（共 33 个测试文件，385+ 用例）
│
├── data/                              # 数据
│   ├── raw/                           # 原始 CSV（❄️ 不可修改）
│   ├── interim/                       # 中间分析表
│   ├── processed/                     # 飞书产物
│   └── system/                        # 运行时状态（审计 CSV 等）
│
├── pipeline/                          # 管道步骤定义与编排
│   ├── steps.py                       # 9 个管道步骤函数
│   └── runner.py                      # 管道编排与执行
│
├── scripts/                           # 数据管道脚本（❄️ FROZEN）
│   ├── config.py                      # 集中路径常量（⚠️ 已弃用，请从 core.config 导入）
│   ├── run_api.py                     # API 启动入口（✅ 活跃）
│   ├── run_db_pipeline.py             # DB 管道入口（✅ 活跃，内部调用 pipeline/）
│   ├── feishu_client.py               # 飞书 SDK 客户端（✅ 活跃）
│   ├── _FROZEN.md                     # 冻结脚本说明
│   └── phase01~07_*.py                # 分析脚本（❄️ 路径已失效）
│
├── frontend/                          # React 控制台
├── outputs/                           # 分析产物（图表、CSV、ER 图谱）
├── reports/                           # 分析报告
└── docs/                              # 技术文档
    ├── API_REFERENCE.md               # ★ API 接口参考手册
    ├── v0.5_api_gateway_runbook.md    # v0.5 网关运维权手册
    └── ...
```

## 参考资源

### 数据集

- [Brazilian E-Commerce Dataset on Kaggle](https://www.kaggle.com/datasets/olistbr/brazilian-ecommerce) - 主数据集
- [Marketing Funnel Dataset on Kaggle](https://www.kaggle.com/datasets/olistbr/marketing-funnel) - 营销漏斗数据集
- [Olist Documentation](https://olist.com/) - 官方文档

### 分析工具

- [Claude Code](https://claude.ai/code) - AI 辅助编程工具
- [Pandas Documentation](https://pandas.pydata.org/) - 数据处理库
- [FastAPI Documentation](https://fastapi.tiangolo.com/) - API 框架

### 项目文档

- [`docs/API_REFERENCE.md`](docs/API_REFERENCE.md) — **接口参考手册**（21 个端点、Schema、错误码、前端集成）
- [`README_RUNBOOK.md`](README_RUNBOOK.md) — 端到端运维手册
- [`docs/v0.5_api_gateway_runbook.md`](docs/v0.5_api_gateway_runbook.md) — v0.5 网关运行手册
- [`docs/data_dictionary.md`](docs/data_dictionary.md) — 数据字典
- [`docs/entity_relationships.md`](docs/entity_relationships.md) — 实体关系说明
- [`scripts/_FROZEN.md`](scripts/_FROZEN.md) — 冻结脚本状态说明

## 许可证

本项目使用的数据集遵循 Olist 的使用条款，仅供学习和研究使用。

---

**最后更新**: 2026-05-24
**当前版本**: v0.5.3
**活跃维护**: API 网关、服务层、适配层、测试套件