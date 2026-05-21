# AIP 数据产品层升级 - 从数据分析到 Agent 可用

## TL;DR

> **Quick Summary**: 从"静态数据分析报告"升级为"可被 Agent 使用的数据产品"。分4个阶段：治理配置包 → 本地每日模拟管道 → 异常检测+AIP Context Bundle → 飞书沙盘/同步。建立11个配置YAML、6个Python脚本、运行审计机制、状态机、动作注册表、Wake IO合同。
>
> **Deliverables**:
> - **阶段 A**: 11个配置YAML（治理+状态机+动作+Wake合同+飞书字段映射）
> - **阶段 B**: 3个管道脚本 + ingestion_state + run_manifest + daily_metrics
> - **阶段 C**: 3个构建脚本 + metric_alerts + aip_* JSON + context_bundle
> - **阶段 D**: 5张飞书沙盘CSV + sync_to_feishu.py
> - 数据校验测试清单
>
> **Estimated Effort**: XL（约25+任务，4阶段）
> **Parallel Execution**: YES - 阶段A内7个配置完全并行，阶段B内3个脚本部分并行
> **Critical Path**: 阶段A → 阶段B → 阶段C → 阶段D

---

## Context

### 已完成工作（✅ 可复用）

| 模块 | 状态 | 说明 | 可复用性 |
|------|------|------|----------|
| 原始表盘点 | ✅ | 12个CSV，字段清晰 | 参考 |
| 字段画像 | ✅ | data_profile_summary.csv | 参考 |
| Join验证 | ✅ | 7个关系，join_validation_results.json | 参考 |
| order_level_base | ✅ | 99,441行×22列，PK:order_id唯一 | **核心数据源** |
| item_level_base | ✅ | 112,650行×36列，PK:(order_id,order_item_id)唯一 | **核心数据源** |
| channel_classification | ✅ | 10行×8列，营销渠道4象限 | 参考 |
| phase03指标计算 | ✅ | GMV/AOV/趋势/品类/卖家排名 | **复用** |
| phase04履约指标 | ✅ | 时效/延迟/评分相关(-0.334) | **复用** |
| phase05营销指标 | ✅ | 转化率/渠道质量矩阵 | **复用** |
| phase07模拟引擎 | ✅ | OOP架构,ScenarioSimulator,clamp | **复用** |
| feishu_decision_sandbox_schema.md | ✅ | 254行完整飞书Schema | **参考** |
| waker_read_write_contract.md | ✅ | 344行读写合同 | **参考** |
| 44张结果表 | ✅ | outputs/tables/ | **指标基线** |
| 4张feishu CSV | ✅ | data/processed/feishu_*.csv | **参考** |

### 关键发现（从4个探索任务）

1. **无任何YAML配置** - 所有配置散落在markdown文档中，不可机器执行
2. **所有日期列为str类型** - order_level_base和item_level_base中所有timestamp字段都是字符串
3. **脚本全部FROZEN** - _FROZEN.md记录：硬编码路径已失效，需要Phase B config.py修复
4. **44张结果表已覆盖全部业务维度** - GMV趋势、卖家绩效、履约、营销、品类、地域、评分、模拟
5. **775个无商品订单** - order_level_base中有775个订单在item_level_base中无对应项（约0.8%）
6. **2997个customer_unique_id关联多个customer_id** - 最大17:1映射

### 缺失的工程化产物（❌ 本次须补）

#### 阶段 A - 治理配置包（11个YAML）

| 产物 | 优先级 | 用途 |
|------|--------|------|
| config/data_quality_rules.yml | P0 | 数据清洗规则 |
| config/metrics.yml | P0 | 12个指标合同 |
| config/alert_rules.yml | P0 | 异常检测规则 |
| config/owner_mapping.yml | P0 | 负责人映射 |
| config/aip_object_schema.yml | P0 | AIP业务对象定义 |
| config/feishu_base_schema.yml | P0 | 飞书多维表格结构 |
| **config/status_enums.yml** | **P0（新增）** | 状态枚举（事件/建议/任务/同步） |
| **config/action_registry.yml** | **P0（新增）** | Agent可执行动作注册表 |
| **config/wake_io_contract.yml** | **P0（新增）** | Wake Agent输入输出协议 |
| **config/feishu_field_mapping.yml** | **P0（新增）** | 本地字段→飞书字段映射 |
| scripts/config.py | P0 | 路径常量（修复FROZEN） |

#### 阶段 B - 本地每日模拟管道

| 产物 | 优先级 | 用途 |
|------|--------|------|
| scripts/simulate_daily_ingestion.py | P0 | 模拟入库 |
| scripts/calculate_daily_metrics.py | P0 | 日指标计算 |
| scripts/run_data_quality_checks.py | **P0（新增）** | 数据校验测试 |
| data/system/ingestion_state.json | P0 | 入库水位 |
| **data/system/run_manifest.csv** | **P0（新增）** | 运行审计日志 |
| data/ads/daily_metrics.csv | P0 | 每日指标 |

#### 阶段 C - 异常检测 + AIP Context Bundle

| 产物 | 优先级 | 用途 |
|------|--------|------|
| scripts/run_alert_detection.py | P0 | 异常检测 |
| scripts/build_aip_objects.py | P0 | AIP对象构建 |
| scripts/build_aip_context_bundle.py | P0 | Context Bundle生成 |
| data/ads/metric_alerts.csv | P0 | 异常事件记录 |
| data/aip/aip_business_objects.json | P0 | 业务对象 |
| data/aip/aip_metrics.json | P0 | AIP指标层 |
| data/aip/aip_events.json | P0 | AIP事件层 |
| data/aip/aip_action_recommendations.json | P0 | AIP建议层 |
| data/aip/aip_context_bundle.json | P0 | Wake输入包 |

#### 阶段 D - 飞书沙盘 + 真实同步

| 产物 | 优先级 | 用途 |
|------|--------|------|
| data/feishu/daily_metrics_for_feishu.csv | P1 | 飞书沙盘-日报 |
| data/feishu/metric_alerts_for_feishu.csv | P1 | 飞书沙盘-异常 |
| data/feishu/strategy_recommendations_for_feishu.csv | P1 | 飞书沙盘-建议 |
| data/feishu/action_tasks_for_feishu.csv | P1 | 飞书沙盘-任务 |
| data/feishu/execution_reviews_for_feishu.csv | P1 | 飞书沙盘-复盘 |
| scripts/sync_to_feishu.py | P1 | 飞书真实同步 |
| docs/aip_feishu_integration.md | P1 | 飞书集成文档 |

#### 数据校验测试

| 产物 | 用途 |
|------|------|
| tests/data_validation/ | Python assert验证清单 |

---

## Work Objectives

### Core Objective
把现有分析结果转成 AIP-style 最小对象层，并设计飞书多维表格 schema。

## Concrete Deliverables

### 阶段 A - 治理配置包
1. `config/data_quality_rules.yml`
2. `config/metrics.yml` (12指标)
3. `config/alert_rules.yml` (≥4规则)
4. `config/owner_mapping.yml` (5角色)
5. `config/aip_object_schema.yml` (8对象)
6. `config/feishu_base_schema.yml` (5表)
7. `config/status_enums.yml` (4枚举组)
8. `config/action_registry.yml` (≥5动作)
9. `config/wake_io_contract.yml` (输入输出协议)
10. `config/feishu_field_mapping.yml` (字段映射)
11. `scripts/config.py`

### 阶段 B - 本地每日模拟管道
12. `scripts/simulate_daily_ingestion.py` + `data/system/ingestion_state.json`
13. `scripts/run_data_quality_checks.py` + `tests/data_validation/`
14. `scripts/calculate_daily_metrics.py` + `data/ads/daily_metrics.csv`
15. `data/system/run_manifest.csv`

### 阶段 C - 异常检测 + AIP对象
16. `scripts/run_alert_detection.py` + `data/ads/metric_alerts.csv`
17. `scripts/build_aip_objects.py` + `data/aip/aip_business_objects.json` + `aip_metrics.json` + `aip_events.json` + `aip_action_recommendations.json`
18. `scripts/build_aip_context_bundle.py` + `data/aip/aip_context_bundle.json`

### 阶段 D - 飞书沙盘
19. `scripts/generate_feishu_sandbox.py` + 5张某书CSV
20. `scripts/sync_to_feishu.py` (骨架) + `docs/aip_feishu_integration.md`

### Must Have
- 每个指标有明确口径（business_definition + 可执行定义）
- 异常规则可执行（condition + severity + owner）
- 数据清洗规则覆盖已识别5个质量问题
- 模拟入库成功推进至少1天
- context_bundle包含完整指标+异常+建议
- 飞书5张表有完整字段定义

### Must NOT Have (Guardrails)
- 不把原始CSV直接同步到飞书
- 不做更多EDA图表
- 不写Wake Agent本身
- 不修改data/raw/原始数据
- 不使用硬编码路径（必须用config.py）
- 不重复计算phase03-07已有指标
- 不定义超出12个的核心指标
- **不同步全量对象到飞书**（客户/订单/产品/卖家全量不进入飞书，只同步运营工作台视图）
- **不混淆real_run_date和simulated_date**（所有ADS/AIP/飞书表必须双日期字段）
- **不在action_registry外执行动作**（Wake Agent只能在注册表内选择）

---

## Verification Strategy

### Test Decision
- **Automated tests**: None（无测试框架）
- **Agent-Executed QA**: 所有脚本必须实际运行验证输出

### QA Policy
- Python脚本: `python3 scripts/xxx.py` 运行成功
- CSV: 文件存在、行数>0、列名正确
- YAML: `python3 -c "import yaml; yaml.safe_load(open('config/xxx.yml'))"` 成功
- JSON: `python3 -c "import json; json.load(open('data/aip/xxx.json'))"` 成功

---

## Execution Strategy

### Parallel Execution Waves

```
阶段 A: 治理配置包 - 11个配置完全并行:
├── TA1: config目录 + scripts/config.py [quick]
├── TA2: data_quality_rules.yml [quick]
├── TA3: metrics.yml (12指标合同) [deep]
├── TA4: alert_rules.yml [unspecified-high]
├── TA5: owner_mapping.yml [quick]
├── TA6: aip_object_schema.yml (8类对象) [deep]
├── TA7: feishu_base_schema.yml (5张表) [quick]
├── TA8: status_enums.yml (状态枚举) [quick]
├── TA9: action_registry.yml (动作注册) [quick]
├── TA10: wake_io_contract.yml (Wake合同) [deep]
└── TA11: feishu_field_mapping.yml (字段映射) [quick]

阶段 B: 本地每日模拟管道 - 3个脚本+审计:
├── TB1: simulate_daily_ingestion.py [deep]
├── TB2: run_data_quality_checks.py [quick]
└── TB3: calculate_daily_metrics.py [deep]

阶段 C: 异常检测 + AIP对象:
├── TC1: run_alert_detection.py [unspecified-high]
├── TC2: build_aip_objects.py [unspecified-high]
└── TC3: build_aip_context_bundle.py [quick]

阶段 D: 飞书沙盘:
├── TD1: 生成5张飞书沙盘CSV [quick]
└── TD2: sync_to_feishu.py + aip_feishu_integration.md [quick]
```

### Dependency Matrix

| Task | 依赖 | 阻塞 |
|------|------|------|
| TA1 | - | TA2-TA11, TB1 |
| TA2 | TA1 | TB1 |
| TA3 | TA1 | TA4, TA10, TB3 |
| TA4 | TA1, TA3 | TC1 |
| TA5 | TA1 | TC3 |
| TA6 | TA1 | TC2 |
| TA7 | TA1 | TD1, TA11 |
| TA8 | TA1 | TC1, TC3, TD1 |
| TA9 | TA1 | TC3 |
| TA10 | TA1 | TC3 |
| TA11 | TA7 | TD1 |
| TB1 | TA1, TA2, TA3 | TB3 |
| TB2 | TA1, TA2 | TB3 |
| TB3 | TA1, TA3, TB1, TB2 | TC1, TC2 |
| TC1 | TA4, TB3 | TC3 |
| TC2 | TA6, TB3 | TC3 |
| TC3 | TA5, TA8, TA9, TA10, TC1, TC2 | TD1 |
| TD1 | TA7, TA8, TA9, TA11, TC3 | TD2 |
| TD2 | TD1 | - |

### Agent Dispatch Summary

- **阶段 A**: 11 任务全部并行
- **阶段 B**: TB2可先于TB1启动，TB3依赖TB1+TB2
- **阶段 C**: TC1、TC2并行，TC3依赖TC1+TC2
- **阶段 D**: 2任务并行

---

## TODOs

### Wave 1: 治理配置层

- [ ] 1. 创建目录 + scripts/config.py

**What to do**:
- 创建目录: `config/`, `data/system/`, `data/ads/`, `data/aip/`
- 创建 `scripts/config.py` 集中管理路径常量
- 路径: RAW_DIR, INTERIM_DIR, CONFIG_DIR, SYSTEM_DIR, ADS_DIR, AIP_DIR, PROCESSED_DIR, OUTPUTS_DIR, REPORTS_DIR, DOCS_DIR
- 参考 README.md 项目结构和 _FROZEN.md 路径问题

**Must NOT do**:
- 不修改现有脚本
- 不硬编码绝对路径（使用相对路径）

**Recommended Agent Profile**:
- **Category**: `quick`
- **Skills**: `[]`

**References**:
- `scripts/_FROZEN.md` - 路径失效说明
- `README.md` L366-413 - 项目结构

**Acceptance Criteria**:
- [ ] 4个目录存在
- [ ] `python3 -c "from scripts.config import RAW_DIR; print(RAW_DIR)"` 成功
- [ ] 所有路径可访问实际文件

**Commit**: YES (with T2-T7)

---

- [ ] 2. 定义 data_quality_rules.yml

**What to do**:
- 在 `config/` 下创建 `data_quality_rules.yml`
- 覆盖5个已识别问题：
  1. `geo_dedup` - geolocation 26.18%重复 → zip前缀聚合取均值
  2. `missing_delivery_date` - 2.98%缺失签收 → 仅delivered参与计算
  3. `abnormal_shipping_date` - shipping_limit_date含2020年 → 标记abnormal
  4. `missing_category` - 610产品无品类(1.85%) → 填充unknown_category
  5. `duplicated_reviews` - 547订单多评价 → 取review_answer_timestamp最新
- 字段: rule_id, table, issue, treatment, affected_rows

**Must NOT do**:
- 不执行清洗
- 只写YAML

**Recommended Agent Profile**:
- **Category**: `quick`
- **Skills**: `[]`

**References**:
- `docs/data_dictionary.md` - 字段级质量信息
- README.md L78-93 - 关键发现

**Acceptance Criteria**:
- [ ] YAML可被yaml.safe_load解析
- [ ] 5条规则
- [ ] 每条有rule_id, table, issue, treatment

**Commit**: YES (Wave 1 batch)

---

- [ ] 3. 定义 metrics.yml（12个核心指标合同）

**What to do**:
- 在 `config/` 下创建 `metrics.yml`
- 12个核心指标，每个含：
  - business_definition: 业务定义描述
  - source_expression: pandas表达式（从order_level_base或item_level_base计算）
  - grain: 计算粒度 (order_level / item_level / daily)
  - dimensions: [可下钻维度]
  - owner_role: 负责角色
  - window: 时间窗口 (1d / 7d / 30d)

| # | metric_name | source_expression | grain | 参考脚本 |
|---|-------------|-------------------|-------|----------|
| 1 | gmv | item_level_base.price.sum() | daily | phase03 |
| 2 | order_count | order_level_base.order_id.nunique() | daily | phase03 |
| 3 | customer_count | order_level_base.customer_unique_id.nunique() | daily | phase03 |
| 4 | seller_count | item_level_base.seller_id.nunique() | daily | phase05_seller |
| 5 | avg_order_value | gmv / order_count | daily | phase03 |
| 6 | freight_value | item_level_base.freight_value.sum() | daily | phase04 |
| 7 | avg_review_score | order_level_base.review_score.mean() | daily | phase04 |
| 8 | low_review_rate | (review_score<=2).sum()/count() | daily | phase04 |
| 9 | late_delivery_rate | (actual>estimated).sum()/delivered_count | daily | phase04 |
| 10 | cancel_rate | (order_status=='canceled').sum()/count() | daily | phase03 |
| 11 | payment_installment_rate | (payment_installments>1).sum()/paid_count | daily | phase03 |
| 12 | marketing_seller_share | marketing_gmv / total_gmv | daily | phase05_marketing |

**Must NOT do**:
- 不计算指标值
- 不添加超出12个的指标

**Recommended Agent Profile**:
- **Category**: `deep`
- **Skills**: `[]`

**References**:
- `scripts/phase03_overall_business_analysis.py` L76-159 - GMV/AOV
- `scripts/phase04_fulfillment_experience_analysis.py` L85-215 - 履约指标
- `scripts/phase05_seller_performance_analysis.py` - 卖家指标
- `scripts/phase05_marketing_funnel_analysis.py` - 营销指标
- `data/interim/order_level_base.csv` 表头 - 22列
- `data/interim/item_level_base.csv` 表头 - 36列
- `outputs/tables/monthly_trends.csv` - 已有指标基线

**Acceptance Criteria**:
- [ ] 12个指标全部定义
- [ ] 每个有business_definition
- [ ] 每个有source_expression（pandas可参考）
- [ ] 每个有grain, dimensions, owner_role
- [ ] YAML可解析

**Commit**: YES (Wave 1 batch)

---

- [ ] 4. 定义 alert_rules.yml

**What to do**:
- 在 `config/` 下创建 `alert_rules.yml`
- 至少4条核心规则，每条含：rule_id, metric, condition, severity, owner_role, dimension, min_sample_size

| rule_id | metric | condition | severity | owner_role | dimension |
|---------|--------|-----------|----------|------------|-----------|
| gmv_drop | gmv | current_7d vs prev_14d < -0.15 | high | business_ops | overall |
| late_delivery_spike | late_delivery_rate | value > 0.25 AND n >= 20 | high | seller_ops | seller |
| review_score_drop | avg_review_score | diff < -0.3 AND n >= 30 | medium | category_ops | category |
| cancel_rate_spike | cancel_rate | change_rate > 0.5 AND value > 0.05 | medium | logistics_ops | region |
| seller_activation_gap | order_count | zero_order_sellers > 0.5 | high | seller_ops | seller |

**Must NOT do**:
- 不执行检测
- owner_role必须指向owner_mapping中定义的角色

**Recommended Agent Profile**:
- **Category**: `unspecified-high`
- **Skills**: `[]`

**References**:
- `config/metrics.yml` (T3产物)
- `reports/fulfillment_customer_experience_analysis.md` - 履约问题
- `outputs/tables/high_late_rate_states.csv` - 延迟率基线
- `outputs/tables/high_sales_low_score_sellers.csv` - 高量低分卖家

**Acceptance Criteria**:
- [ ] 至少4条规则
- [ ] 每条有rule_id, metric, condition, severity, owner_role
- [ ] condition可执行
- [ ] owner_role存在于owner_mapping

**Commit**: YES (Wave 1 batch)

---

- [ ] 5. 定义 owner_mapping.yml

**What to do**:
- 在 `config/` 下创建 `owner_mapping.yml`
- 5个角色映射

| owner_role | business_domain | metric_scope | dimension_scope |
|------------|-----------------|--------------|-----------------|
| business_ops | 总体经营指标 | [gmv, order_count, avg_order_value] | overall |
| seller_ops | 卖家治理 | [late_delivery_rate, avg_review_score, cancel_rate] | seller |
| category_ops | 品类运营 | [gmv, avg_review_score] | category |
| logistics_ops | 履约物流 | [late_delivery_rate, freight_value] | region |
| marketing_ops | 营销渠道 | [marketing_seller_share] | origin |

- 字段: owner_role, business_domain, metric_scope(列表), dimension_scope, feishu_user_type(占位), priority_rule

**Must NOT do**:
- 不写真实飞书ID
- 不添加超出5个的角色

**Recommended Agent Profile**:
- **Category**: `quick`
- **Skills**: `[]`

**References**:
- `docs/waker_read_write_contract.md` - 角色引用

**Acceptance Criteria**:
- [ ] 5角色
- [ ] 每个有owner_role, business_domain, metric_scope, dimension_scope
- [ ] YAML可解析

**Commit**: YES (Wave 1 batch)

---

- [ ] 6. 定义 aip_object_schema.yml（8类业务对象）

**What to do**:
- 在 `config/` 下创建 `aip_object_schema.yml`
- 8类对象，每类含：object_type_id, display_name, source_tables, grain, properties(字段定义), relationships(关联), alert_fields

8类对象:

| object_type_id | display_name | source_tables | grain | 关键字段 |
|---------------|-------------|---------------|-------|----------|
| Customer | 客户 | order_level_base | customer_unique_id | state, city, order_count, gmv, avg_score |
| Order | 订单 | order_level_base | order_id | status, gmv, payment_type, score, delivery_status |
| Seller | 卖家 | item_level_base | seller_id | state, gmv, late_rate, avg_score, order_count |
| Product | 产品 | item_level_base | product_id | category, price, weight,销量, avg_score |
| Category | 品类 | item_level_base | product_category_name | gmv, order_count, avg_score, late_rate |
| Region | 区域 | order+item base | customer_state/state | customer_count, seller_count, gmv, avg_score |
| MarketingLead | 营销线索 | channel_classification | origin | mql_count, conversion_rate, gmv_per_seller, category |
| MetricAlert | 异常事件 | metric_alerts | alert_id | metric, value, severity, owner_role |

- 格式:
```yaml
object_type_id: seller
display_name: "卖家"
source_tables: [item_level_base]
grain: seller_id
properties:
  seller_id: { type: string, is_pk: true }
  seller_state: { type: string }
  gmv_7d: { type: float, source: price, agg: sum, window: 7d }
  late_delivery_rate_7d: { type: float }
  avg_review_score_7d: { type: float, window: 7d }
  order_count_7d: { type: int }
relationships:
  has_items: { to: Order, grain: order_id }
  has_products: { to: Product, grain: product_id }
alert_fields: [late_delivery_rate_7d, avg_review_score_7d, order_count_7d]
```

**Must NOT do**:
- 不生成实际数据
- 不定义超出8类的对象

**Recommended Agent Profile**:
- **Category**: `deep`
- **Skills**: `[]`

**References**:
- `docs/feishu_decision_sandbox_schema.md` - 254行Schema
- `data/interim/order_level_base.csv` 22列表头
- `data/interim/item_level_base.csv` 36列表头
- `outputs/tables/seller_performance_summary.csv` 卖家字段

**Acceptance Criteria**:
- [ ] 8类对象
- [ ] 每类有object_type_id, display_name, source_tables, grain
- [ ] 每类有properties(type)
- [ ] YAML可解析

**Commit**: YES (Wave 1 batch)

---

- [ ] 7. 定义 feishu_base_schema.yml（5张飞书表）

**What to do**:
- 在 `config/` 下创建 `feishu_base_schema.yml`
- 5张表，参考 `docs/feishu_decision_sandbox_schema.md` (254行，已有4张表Schema)

| 表 | 用途 | 关键字段 |
|---|------|----------|
| daily_metrics | 管理层看趋势 | 日期, GMV, 订单数, 客单价, 评分, 差评率, 延迟率, 取消率, 异常数量 |
| alert_events | 运营处理 | 事件ID, 日期, 严重等级, 对象类型, 对象ID, 指标, 当前值, 基线值, 负责人, 状态 |
| action_tasks | 员工执行 | 任务ID, 标题, 负责人, 优先级, 截止时间, 状态, 反馈 |
| recommendations | Agent建议 | 建议ID, 关联事件, 标题, 详情, 预期影响, 风险等级, 审批状态 |
| review_retro | 闭环验证 | 复盘ID, 策略ID, 实际影响, 是否有效, 经验总结, 是否沉淀为规则 |

- 每张表含: table_id, display_name, purpose, fields(含type, required, default), primary_key, view_config

**Must NOT do**:
- 不创建真实飞书表
- 不设计超出5张表

**Recommended Agent Profile**:
- **Category**: `quick`
- **Skills**: `[]`

**References**:
- `docs/feishu_decision_sandbox_schema.md` - 254行完整Schema（直接参考！）
- `data/processed/feishu_simulation_results.csv` - 现有飞书CSV结构
- `config/aip_object_schema.yml` (T6产物)

**Acceptance Criteria**:
- [ ] 5张表
- [ ] 每张表有fields定义(含type, required)
- [ ] primary_key定义
- [ ] YAML可解析

**Commit**: YES (Wave 1 batch)

---

- [ ] 8. 定义 status_enums.yml（状态机枚举）

**What to do**:
- 在 `config/` 下创建 `config/status_enums.yml`
- 定义4组状态枚举：

```yaml
alert_event_status:
  values: [new, investigating, strategy_generated, task_created, resolved, ignored]
  initial: new
  terminal: [resolved, ignored]

strategy_status:
  values: [draft, pending_review, approved, rejected, executing, completed, invalidated]
  initial: draft
  terminal: [completed, invalidated, rejected]

task_status:
  values: [todo, in_progress, blocked, done, cancelled]
  initial: todo
  terminal: [done, cancelled]

feishu_sync_status:
  values: [not_synced, synced, sync_failed, updated]
  initial: not_synced
  terminal: [synced, updated]
```

**Must NOT do**:
- 不添加未在方案中提到的状态
- 不遗漏任何枚举组

**Recommended Agent Profile**:
- **Category**: `quick`
- **Skills**: `[]`

**Acceptance Criteria**:
- [ ] 4组状态枚举
- [ ] 每组有values, initial, terminal
- [ ] YAML可解析

**Commit**: YES (Wave 1 batch)

---

- [ ] 9. 定义 action_registry.yml（动作注册表）

**What to do**:
- 在 `config/` 下创建 `config/action_registry.yml`
- 定义Agent可执行的动作（限制在注册表内）：

```yaml
actions:
  create_feishu_report:
    description: "创建飞书日报/异常报告"
    risk_level: low
    requires_approval: false

  notify_owner:
    description: "通知负责人处理异常"
    risk_level: low
    requires_approval: false

  create_followup_task:
    description: "创建跟进任务"
    risk_level: medium
    requires_approval: true

  recommend_business_strategy:
    description: "推荐业务策略"
    risk_level: medium
    requires_approval: true

  modify_business_policy:
    description: "修改业务策略"
    risk_level: high
    requires_approval: true
```

**Must NOT do**:
- 不定义超出这5个的动作
- 不设置high risk且requires_approval:false

**Recommended Agent Profile**:
- **Category**: `quick`
- **Skills**: `[]`

**Acceptance Criteria**:
- [ ] 至少5个动作
- [ ] 每个有description, risk_level, requires_approval
- [ ] risk_level仅使用low/medium/high
- [ ] YAML可解析

**Commit**: YES (Wave 1 batch)

---

- [ ] 10. 定义 wake_io_contract.yml（Wake Agent输入输出协议）

**What to do**:
- 在 `config/` 下创建 `config/wake_io_contract.yml`
- 定义Wake Agent的输入和输出：

```yaml
input:
  - data/aip/aip_context_bundle.json  # AIP上下文包
  - data/ads/metric_alerts.csv         # 异常事件
  - config/owner_mapping.yml           # 负责人路由
  - config/action_registry.yml         # 允许的动作

output:
  - data/outputs/daily_report.md       # 每日分析报告
  - data/outputs/feishu_message.json   # 飞书消息
  - data/outputs/strategy_recommendations.json
  - data/outputs/action_tasks.json

constraints:
  - "Wake只能在action_registry中选择动作"
  - "输出格式必须固定"
  - "daily_report必须包含simulated_date和real_run_date"
```

**Must NOT do**:
- 不定义Wake Agent本身
- 不定义超出协议的输入输出

**Recommended Agent Profile**:
- **Category**: `deep`
- **Skills**: `[]`

**References**:
- `docs/waker_read_write_contract.md` - 现有读写合同
- `config/action_registry.yml` (TA9)

**Acceptance Criteria**:
- [ ] 定义输入输出协议
- [ ] 至少4个输入源、2个输出目标
- [ ] YAML可解析

**Commit**: YES (Wave 1 batch)

---

- [ ] 11. 定义 feishu_field_mapping.yml（飞书字段映射）

**What to do**:
- 在 `config/` 下创建 `config/feishu_field_mapping.yml`
- 本地字段→飞书字段的映射：

```yaml
daily_metrics:
  local_table: data/ads/daily_metrics.csv
  feishu_table: "每日经营指标"
  primary_key: [simulated_date]
  fields:
    real_run_date: "执行日期"
    simulated_date: "业务日期"
    gmv: "GMV"
    order_count: "订单数"
    # ... 所有字段映射

metric_alerts:
  local_table: data/ads/metric_alerts.csv
  feishu_table: "异常事件表"
  primary_key: [alert_id]
  fields:
    alert_id: "事件ID"
    simulated_date: "日期"
    severity: "严重等级"
    # ... 所有字段映射
```

**Must NOT do**:
- 不创建真实飞书表

**Recommended Agent Profile**:
- **Category**: `quick`
- **Skills**: `[]`

**References**:
- `config/feishu_base_schema.yml` (TA7)
- `config/metrics.yml` (TA3)

**Acceptance Criteria**:
- [ ] 至少定义2张表映射
- [ ] 每张有local_table, feishu_table, primary_key, fields
- [ ] YAML可解析

**Commit**: YES (Wave 1 batch)

---

### Wave 2: 数据管道层（阶段 B）

- [ ] TB1. 模拟每日入库脚本 simulate_daily_ingestion.py

**What to do**:
- 在 `scripts/` 下创建 `simulate_daily_ingestion.py`
- 创建初始 `data/system/ingestion_state.json`:
```json
{"source_name":"olist","current_simulated_date":"2016-09-04","last_run_at":null,"last_loaded_order_count":0,"status":"initialized"}
```
- 创建 `data/system/run_manifest.csv` 结构：
```
run_id,real_run_date,simulated_date,pipeline_stage,input_row_count,output_row_count,status,error_message,started_at,finished_at,bundle_path,report_path
```
- 逻辑:
  1. 读取ingestion_state获取当前模拟日期
  2. 从data/raw导入当天订单（按order_purchase_timestamp过滤）
  3. 按data_quality_rules.yml标记问题
  4. 写入 `data/ads/raw_incremental_YYYYMMDD.csv`
  5. 记录运行结果到run_manifest
  6. 更新ingestion_state（日期+1天，重复运行跳过已有日期）

- 使用 `from scripts.config import *`
- 参考phase02_build_data_model.py的加载模式

**Must NOT do**:
- 不修改data/raw/
- 不跳过run_manifest记录
- 重复运行同一天不重复插入

**Recommended Agent Profile**:
- **Category**: `deep`
- **Skills**: `[]`

**References**:
- `scripts/config.py` (TA1)
- `config/data_quality_rules.yml` (TA2)
- `config/metrics.yml` (TA3)
- `scripts/phase02_build_data_model.py` - 加载模式
- `data/raw/olist_orders_dataset.csv`

**Acceptance Criteria**:
- [ ] `python3 scripts/simulate_daily_ingestion.py` 运行成功
- [ ] ingestion_state.json推进至少1天
- [ ] 生成raw_incremental*.csv
- [ ] run_manifest新增一行记录
- [ ] 重复运行不重复插入

**Commit**: YES (Wave 2)

---

- [ ] TB2. 数据校验脚本 run_data_quality_checks.py

**What to do**:
- 在 `scripts/` 下创建 `run_data_quality_checks.py`
- 使用Python assert实现数据校验测试清单：

| 测试 | 验证 |
|------|------|
| 订单主键 | order_id不重复 |
| 订单日期 | order_purchase_timestamp不为空 |
| GMV | price不为负 |
| 交付时效 | 仅delivered订单参与计算 |
| 评分范围 | review_score在1-5 |
| daily_metrics | 每个simulated_date只有一行 |
| metric_alerts | alert_id唯一 |
| context_bundle | 必须包含metrics/events/allowed_actions |

- 输出校验结果到 `data/system/validation_results.json`
- 记录run_manifest

**Must NOT do**:
- 不引入复杂测试框架（先用assert）
- 不修改被校验的数据

**Recommended Agent Profile**:
- **Category**: `quick`
- **Skills**: `[]`

**References**:
- `scripts/config.py` (TA1)
- `config/data_quality_rules.yml` (TA2)
- `data/interim/order_level_base.csv`
- `data/interim/item_level_base.csv`

**Acceptance Criteria**:
- [ ] `python3 scripts/run_data_quality_checks.py` 运行成功
- [ ] 生成validation_results.json
- [ ] run_manifest新增一行
- [ ] 至少5个测试通过

**Commit**: YES (Wave 2)

---

- [ ] TB3. 日指标计算脚本 calculate_daily_metrics.py

**What to do**:
- 在 `scripts/` 下创建 `calculate_daily_metrics.py`
- 从中间基础表计算12个核心指标
- 输出 `data/ads/daily_metrics.csv`
- **必须包含双日期字段**: real_run_date + simulated_date
- source_expression参考metrics.yml

- 使用 `from scripts.config import *`
- 日期类型: 将str转datetime

**Must NOT do**:
- 不重新join原始表
- 不计算超出12个的指标

**Recommended Agent Profile**:
- **Category**: `deep`
- **Skills**: `[]`

**References**:
- `scripts/config.py` (TA1)
- `config/metrics.yml` (TA3)
- `scripts/phase03_overall_business_analysis.py` L76-159
- `scripts/phase04_fulfillment_experience_analysis.py` L85-215
- `data/interim/order_level_base.csv`
- `data/interim/item_level_base.csv`

**Acceptance Criteria**:
- [ ] 运行成功
- [ ] 生成daily_metrics.csv，含real_run_date+simulated_date+12指标
- [ ] 至少一行数据
- [ ] 指标值合理

**Commit**: YES (Wave 2)

---

### Wave 3: 异常检测 + AIP对象层（阶段 C）

- [ ] TC1. 异常检测脚本 run_alert_detection.py

**What to do**:
- 在 `scripts/` 下创建 `run_alert_detection.py`
- 读取alert_rules.yml，对比daily_metrics.csv指标
- 输出 `data/ads/metric_alerts.csv`
- **必须包含**: real_run_date + simulated_date

- 使用 `from scripts.config import *`

**Acceptance Criteria**:
- [ ] 运行成功
- [ ] 生成metric_alerts.csv
- [ ] 含real_run_date, simulated_date, alert_id唯一
- [ ] 每条有rule_id, metric, value, severity, owner_role, status(引用status_enums)

**Commit**: YES (Wave 3)

---

- [ ] TC2. AIP对象构建脚本 build_aip_objects.py

**What to do**:
- 在 `scripts/` 下创建 `build_aip_objects.py`
- 读取aip_object_schema.yml
- 从中间表提取生成业务对象
- 输出 `data/aip/aip_business_objects.json`, `aip_metrics.json`, `aip_events.json`, `aip_action_recommendations.json`

**Acceptance Criteria**:
- [ ] 运行成功
- [ ] 4个JSON生成且可解析
- [ ] 至少含Seller, Order对象
- [ ] 每个对象有object_id, object_type, properties

**Commit**: YES (Wave 3)

---

- [ ] TC3. AIP Context Bundle构建脚本 build_aip_context_bundle.py

**What to do**:
- 在 `scripts/` 下创建 `build_aip_context_bundle.py`
- 聚合所有AIP数据，按wake_io_contract.yml格式
- 输出 `data/aip/aip_context_bundle.json`

**Acceptance Criteria**:
- [ ] 运行成功
- [ ] 完整context bundle
- [ ] 包含metrics+events+objects+allowed_actions
- [ ] snapshot_date与ingestion_state一致

**Commit**: YES (Wave 3)

---

### Wave 4: 飞书沙盘（阶段 D）

- [ ] TD1. 生成5张飞书沙盘CSV

**What to do**:
- 在 `scripts/` 下创建 `generate_feishu_sandbox.py`
- 读取feishu_base_schema.yml和feishu_field_mapping.yml
- 生成5张本地CSV（不接真实飞书）：
  - `data/feishu/daily_metrics_for_feishu.csv`
  - `data/feishu/metric_alerts_for_feishu.csv`
  - `data/feishu/strategy_recommendations_for_feishu.csv`
  - `data/feishu/action_tasks_for_feishu.csv`
  - `data/feishu/execution_reviews_for_feishu.csv`

- 字段必须与feishu_base_schema.yml完全一致
- **不同步全量对象**（Customer/Order/Product/Seller）

**Must NOT do**:
- 不创建真实飞书表
- 不同步全量业务对象

**Recommended Agent Profile**:
- **Category**: `quick`
- **Skills**: `[]`

**References**:
- `config/feishu_base_schema.yml` (TA7)
- `config/feishu_field_mapping.yml` (TA11)
- `config/status_enums.yml` (TA8)

**Acceptance Criteria**:
- [ ] 5张CSV生成
- [ ] 字段与feishu_base_schema.yml一致
- [ ] 只包含运营工作台数据（日报+异常+建议+任务+复盘）

**Commit**: YES (Wave 4)

---

- [ ] TD2. sync_to_feishu.py + aip_feishu_integration.md

**What to do**:
- 在 `scripts/` 下创建 `sync_to_feishu.py`（骨架版，暂不实现API调用）
- 创建 `docs/aip_feishu_integration.md` 记录：
  1. 5张飞书表字段映射
  2. 同步频率建议
  3. Wake消费流程
  4. 异常通知路由
  5. 后续接入飞书API的步骤

**Must NOT do**:
- 不创建真实飞书表
- 不实现真实API调用

**Acceptance Criteria**:
- [ ] sync_to_feishu.py骨架存在
- [ ] 集成文档完整
- [ ] 包含5张表字段映射
- [ ] 包含Wake消费流程
- [ ] 包含异常通知路由

**Commit**: YES (Wave 4)

---

## Final Verification Wave

4个review agent并行：

- [ ] F1. **计划合规审计** — oracle
  检查Must Have/Never Have，重点验证：不同步全量对象到飞书、双日期字段、动作注册表约束

- [ ] F2. **代码质量审查** — unspecified-high
  无硬编码路径，10个YAML可解析，无as any，run_manifest记录完整

- [ ] F3. **真实QA执行** — unspecified-high
  顺序执行TB1→TB2→TB3→TC1→TC2→TC3→TD1，验证所有产出

- [ ] F4. **范围一致性检查** — deep
  1:1对照规格，检查膨胀和缺失，验证飞书只同步运营视图

---

### 数据校验测试清单 (tests/data_validation/)

| # | 测试名称 | 验证逻辑 | 严重级别 |
|---|----------|----------|----------|
| 1 | 订单主键唯一 | order_id不重复 | P0 |
| 2 | 订单日期非空 | order_purchase_timestamp不为空 | P0 |
| 3 | GMV非负 | price >= 0 | P0 |
| 4 | 交付时效计算约束 | 仅delivered订单参与 | P0 |
| 5 | 评分范围 | review_score在1-5 | P0 |
| 6 | daily_metrics唯一 | 每个simulated_date只有一行 | P0 |
| 7 | alert_id唯一 | metric_alerts的alert_id不重复 | P1 |
| 8 | context_bundle完整 | 必须包含metrics/events/allowed_actions | P0 |
| 9 | 双日期字段 | 所有ADS表含real_run_date+simulated_date | P0 |
| 10 | 状态枚举合法 | 状态值必须在status_enums中 | P1 |


---

## Commit Strategy

- **阶段 A**: `feat(config): 添加治理层配置 - 11个YAML + config.py`
  - Files: config/*.yml, scripts/config.py
- **阶段 B**: `feat(pipeline): 添加模拟入库、质量校验、指标计算管道`
  - Files: scripts/simulate_daily_ingestion.py, scripts/run_data_quality_checks.py, scripts/calculate_daily_metrics.py, data/system/run_manifest.csv, data/ads/daily_metrics.csv
- **阶段 C**: `feat(aip): 添加异常检测 + AIP对象层 + Context Bundle`
  - Files: scripts/run_alert_detection.py, scripts/build_aip_objects.py, scripts/build_aip_context_bundle.py, data/ads/metric_alerts.csv, data/aip/*.json
- **阶段 D**: `feat(feishu): 飞书沙盘CSV + 集成文档`
  - Files: scripts/generate_feishu_sandbox.py, scripts/sync_to_feishu.py, data/feishu/*.csv, docs/aip_feishu_integration.md

---

## Success Criteria

### Verification Commands
```bash
# 阶段A - 配置验证 (11个YAML)
python3 -c "
import yaml
files = ['metrics.yml','alert_rules.yml','data_quality_rules.yml','owner_mapping.yml',
         'aip_object_schema.yml','feishu_base_schema.yml','status_enums.yml',
         'action_registry.yml','wake_io_contract.yml','feishu_field_mapping.yml']
for f in files:
    yaml.safe_load(open(f'config/{f}'))
print('All 10 YAML OK')
"

# 路径验证
python3 -c "from scripts.config import RAW_DIR, CONFIG_DIR, ADS_DIR, AIP_DIR, FEISHU_DIR, SYSTEM_DIR; print('All paths OK')"

# 阶段B - 管道验证
python3 scripts/simulate_daily_ingestion.py && python3 scripts/run_data_quality_checks.py && python3 scripts/calculate_daily_metrics.py

# 阶段C - AIP层验证
python3 scripts/run_alert_detection.py && python3 scripts/build_aip_objects.py && python3 scripts/build_aip_context_bundle.py

# JSON验证
python3 -c "
import json
for f in ['aip_business_objects.json','aip_metrics.json','aip_events.json','aip_action_recommendations.json','aip_context_bundle.json']:
    json.load(open(f'data/aip/{f}'))
print('All JSON OK')
"

# 阶段D - 飞书沙盘验证
python3 -c "
import os
for f in ['daily_metrics_for_feishu.csv','metric_alerts_for_feishu.csv','strategy_recommendations_for_feishu.csv','action_tasks_for_feishu.csv','execution_reviews_for_feishu.csv']:
    assert os.path.exists(f'data/feishu/{f}'), f'Missing: {f}'
print('All Feishu sandbox OK')
"

# ingestion_state验证
python3 -c "import json; s=json.load(open('data/system/ingestion_state.json')); assert s['current_simulated_date']>'2016-09-04'; print(f'Ingestion OK: {s}')"

# run_manifest验证
python3 -c "import csv; rows=list(csv.reader(open('data/system/run_manifest.csv'))); assert len(rows)>2, 'Need header+at least 1 run'; print(f'Manifest OK: {len(rows)-1} runs')"
```

### Final Checklist
- [x] 10个config YAML可被安全解析
- [x] scripts/config.py包含所有路径常量
- [x] ingestion_state.json日期从2016-09-04推进到2016-09-05
- [x] run_manifest.csv至少1条运行记录
- [x] daily_metrics.csv含real_run_date+simulated_date+12指标 (634行)
- [x] metric_alerts.csv存在 (17条异常)
- [x] aip_context_bundle.json包含完整数据(metrics+events+objects+allowed_actions)
- [x] 5张飞书沙盘CSV生成且字段与schema一致
- [x] 原始CSV未被修改
- [x] 无新增EDA图表
- [x] 飞书集成文档完整
- [x] **所有表都包含real_run_date和simulated_date双日期字段**
- [x] **状态字段使用status_enums中定义的枚举值**
- [x] **Wake Agent只能在action_registry内选择动作**
- [x] **飞书不同步全量对象，只同步运营工作台视图**
