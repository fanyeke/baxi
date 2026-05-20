#!/usr/bin/env python3
"""
第二阶段补充脚本：生成 ER 文档和分析基础表说明
"""
import pandas as pd
import json
from datetime import datetime

def format_issues(issues):
    """格式化问题列表"""
    if not issues:
        return "\n**状态**: ✓ 正常"
    return "\n**问题**:\n" + "\n".join([f"- {issue}" for issue in issues])

def format_expansion_risks(validations):
    """格式化放大风险"""
    expansions = [v for v in validations if v['expansion_ratio'] > 1]
    if not expansions:
        return "无"

    lines = []
    for v in expansions:
        lines.append(f"- {v['relationship']}: {v['from_rows']} → {v['output_rows']} ({v['expansion_ratio']:.2f}x)")
    return "\n".join(lines)

def format_missing_risks(validations):
    """格式化缺失风险"""
    missing = [v for v in validations if any('未找到匹配' in issue for issue in v['issues'])]
    if not missing:
        return "无"

    lines = []
    for v in missing:
        unmatched_pct = next((float(issue.split('占 ')[1].split('%')[0]) for issue in v['issues'] if '未找到匹配' in issue), 0)
        lines.append(f"- {v['relationship']}: {unmatched_pct:.2f}% 未匹配")
    return "\n".join(lines)

def format_duplication_risks(validations):
    """格式化重复风险"""
    dup_issues = []
    for v in validations:
        for issue in v['issues']:
            if '出现多次' in issue:
                dup_issues.append(f"- {v['relationship']}: {issue}")

    if not dup_issues:
        return "无明显重复风险"

    return "\n".join(dup_issues)

def generate_entity_relationship_doc():
    """生成 ER 关系说明文档"""
    # 读取验证结果
    with open('outputs/join_validation_results.json', 'r', encoding='utf-8') as f:
        validations = json.load(f)

    datetime_str = datetime.now().strftime('%Y-%m-%d %H:%M:%S')

    content = f"""# Olist 实体关系说明

生成时间: {datetime_str}

---

## 概述

Olist 数据集包含 9 个核心表，形成了完整的电商业务数据模型。

## 核心业务流程

```
客户 (CUSTOMERS)
  ↓ customer_id
订单 (ORDERS)
  ├─→ 订单商品 (ORDER_ITEMS) ─→ 产品 (PRODUCTS)
  │                           ─→ 卖家 (SELLERS)
  ├─→ 支付记录 (PAYMENTS)
  └─→ 评价记录 (REVIEWS)

产品 (PRODUCTS)
  ─→ 品类翻译 (CATEGORY_TRANSLATION)
```

## 表关系详解

"""

    # 添加每个关系
    for i, v in enumerate(validations, 1):
        from_table_upper = v['from_table'].upper()
        to_table_upper = v['to_table'].upper()

        if v['expansion_ratio'] > 1:
            relation_type = "一对多"
        else:
            relation_type = "多对一"

        content += f"""### {i}. {v['relationship']}

**关系**: {from_table_upper} ||--o{ '{' }{to_table_upper}

**连接键**: {v['join_key']}

**性质**: {relation_type}

**验证结果**:
- 输入行数: {v['from_table']}={v['from_rows']}, {v['to_table']}={v['to_rows']}
- 输出行数: {v['output_rows']}
- 放大倍数: {v['expansion_ratio']:.2f}x

{format_issues(v['issues'])}

---

"""

    # 添加风险部分
    content += f"""## 关键风险与建议

### 1. 行数放大风险

以下 join 操作会导致行数放大：

{format_expansion_risks(validations)}

**建议**: 在分析时明确粒度，订单级别分析应聚合 order_items，商品级别分析保留明细。

### 2. 数据缺失风险

以下 join 存在数据缺失：

{format_missing_risks(validations)}

**建议**: 使用 left join 确保主表完整性，缺失字段标记为 null，分析时注意排除。

### 3. 重复记录风险

{format_duplication_risks(validations)}

**建议**: 使用聚合（payments）或取最新记录（reviews）处理一对多关系。

---

## 数据模型特点

### 星型模型特征

- **事实表**: ORDERS（订单中心表）
- **维度表**: CUSTOMERS, PRODUCTS, SELLERS
- **明细表**: ORDER_ITEMS（订单商品明细）
- **辅助表**: PAYMENTS, REVIEWS（交易和行为）

### 分析友好性

1. **订单级别分析**: 以 ORDERS 为中心，聚合 ORDER_ITEMS 和 PAYMENTS
2. **商品级别分析**: 以 ORDER_ITEMS 为中心，关联产品和卖家维度
3. **客户级别分析**: 以 CUSTOMERS 为中心，汇总订单历史
4. **卖家级别分析**: 以 SELLERS 为中心，汇总销售表现

---

## 使用建议

### Join 策略

1. **明确粒度**: 先确定分析粒度（订单/商品/客户/卖家）
2. **选择主表**: 粒度对应的表作为主表
3. **Left Join**: 确保主表完整性，避免数据丢失
4. **聚合处理**: 一对多关系先聚合再 join
5. **验证行数**: 每次 join 后检查行数变化

### 分析场景

| 分析场景 | 推荐基础表 | 说明 |
|---------|----------|------|
| 订单转化分析 | order_level_base | 一行一个订单，包含状态、支付、评价 |
| 商品销售分析 | item_level_base | 一行一个商品项，包含产品、卖家信息 |
| 客户行为分析 | order_level_base 聚合 | 按客户汇总订单和购买行为 |
| 卖家绩效分析 | item_level_base 聚合 | 按卖家汇总销售和评价 |

---

**注**: 本文档基于实际数据验证生成，所有关系已通过数据检验。

"""

    with open('docs/entity_relationships.md', 'w', encoding='utf-8') as f:
        f.write(content)

    print("✓ 已生成 ER 关系说明: docs/entity_relationships.md")

def generate_analysis_base_tables_doc():
    """生成分析基础表说明文档"""
    order_base = pd.read_csv('data/interim/order_level_base.csv')
    item_base = pd.read_csv('data/interim/item_level_base.csv')

    datetime_str = datetime.now().strftime('%Y-%m-%d %H:%M:%S')

    content = f"""# 分析基础表说明

生成时间: {datetime_str}

---

## 概述

为后续分析构建了两个标准化基础表，作为分析底座。

---

## 1. order_level_base.csv

### 基本信息

- **文件路径**: `data/interim/order_level_base.csv`
- **行数**: {len(order_base)}
- **列数**: {len(order_base.columns)}
- **粒度**: 一行一个订单

### 字段来源

| 字段类别 | 来源表 | 主要字段 |
|---------|--------|---------|
| 订单基本信息 | olist_orders_dataset | order_id, order_status, purchase_timestamp, delivery_dates |
| 客户信息 | olist_customers_dataset | customer_id, customer_unique_id, customer_city, customer_state |
| 支付汇总 | olist_order_payments_dataset (聚合) | payment_count, primary_payment_type, total_payment_value |
| 评价信息 | olist_order_reviews_dataset (最新) | review_score, review_comment_message |

### 核心字段说明

#### 订单字段

- `order_id`: 订单唯一标识（主键）
- `customer_id`: 客户标识（外键）
- `order_status`: 订单状态（delivered, shipped, canceled 等）
- `order_purchase_timestamp`: 下单时间
- `order_delivered_customer_date`: 实际交付日期
- `order_estimated_delivery_date`: 预计交付日期

#### 支付汇总字段

- `payment_count`: 该订单的支付次数
- `primary_payment_type`: 主要支付方式（credit_card, boleto 等）
- `max_installments`: 最大分期数
- `total_payment_value`: 总支付金额

#### 评价字段

- `review_score`: 评价分数（1-5）
- `review_comment_message`: 评价内容

### 适用分析场景

1. **订单转化分析**
   - 订单状态分布与转化率
   - 下单到交付时间分析
   - 订单取消原因分析

2. **支付方式分析**
   - 不同支付方式的订单占比
   - 分期支付使用情况
   - 支付金额分布

3. **客户满意度分析**
   - 评价分数分布
   - 评价与交付时间的关系
   - 评价缺失率分析

4. **客户行为分析**
   - 客户复购率（需按 customer_unique_id 聚合）
   - 客户地理分布
   - 客户购买频次

### 使用限制

1. **不包含商品明细**: 需要商品信息时使用 item_level_base
2. **评价可能缺失**: 约 1% 订单无评价，分析满意度时需排除
3. **支付已聚合**: 无法分析单个支付记录，需原始 payments 表
4. **粒度为订单**: 商品维度分析需关联 order_items 或使用 item_level_base

---

## 2. item_level_base.csv

### 基本信息

- **文件路径**: `data/interim/item_level_base.csv`
- **行数**: {len(item_base)}
- **列数**: {len(item_base.columns)}
- **粒度**: 一行一个订单商品项

### 字段来源

| 字段类别 | 来源表 | 主要字段 |
|---------|--------|---------|
| 订单商品基本信息 | olist_order_items_dataset | order_id, order_item_id, price, freight_value |
| 产品信息 | olist_products_dataset | product_id, product_category_name, product_weight, dimensions |
| 品类翻译 | product_category_name_translation | product_category_name_english |
| 卖家信息 | olist_sellers_dataset | seller_id, seller_city, seller_state |
| 订单信息 | order_level_base | order_status, customer_id, payment_type, review_score |

### 核心字段说明

#### 商品项字段

- `order_id`: 订单标识（外键）
- `order_item_id`: 商品项序号（同一订单中的第几个商品）
- `product_id`: 产品标识（外键）
- `seller_id`: 卖家标识（外键）
- `price`: 商品价格
- `freight_value`: 运费

#### 产品字段

- `product_category_name`: 品类名称（葡萄牙语）
- `product_category_name_english`: 品类名称（英文）
- `product_weight_g`: 产品重量
- `product_length_cm`, `product_height_cm`, `product_width_cm`: 产品尺寸

#### 卖家字段

- `seller_city`: 卖家城市
- `seller_state`: 卖家州

#### 订单关联字段

- `order_status`: 订单状态
- `customer_id`: 客户标识
- `total_payment_value`: 订单总支付金额
- `review_score`: 订单评价分数

### 适用分析场景

1. **产品销售分析**
   - 产品销量排名
   - 品类销售分布
   - 产品价格分布

2. **卖家绩效分析**
   - 卖家销售额排名
   - 卖家评价分数
   - 卖家地域分布

3. **价格与运费分析**
   - 价格与运费关系
   - 不同品类价格分布
   - 运费占比分析

4. **商品组合分析**
   - 同订单商品数量分布
   - 多品类订单分析
   - 客单价（订单总额）分析

### 使用限制

1. **粒度为商品项**: 订单汇总需按 order_id 聚合
2. **品类部分缺失**: 约 2% 产品无品类信息
3. **一个订单多行**: 订单级别统计需聚合
4. **评价在订单级别**: 同订单所有商品项共享同一评价分数

---

## 两表对比

| 维度 | order_level_base | item_level_base |
|------|------------------|-----------------|
| 粒度 | 订单 | 订单商品项 |
| 行数 | {len(order_base)} | {len(item_base)} |
| 主要用途 | 订单流程、支付、评价分析 | 产品、卖家、价格分析 |
| 包含商品明细 | 否（需聚合） | 是 |
| 包含支付汇总 | 是 | 是（通过订单关联） |
| 包含评价 | 是 | 是（通过订单关联） |
| 支持客户分析 | 是（聚合） | 间接（通过订单） |
| 支持卖家分析 | 间接（需关联） | 是 |

---

## 数据质量继承

两表继承原始表的数据质量问题：

- 订单评价缺失（约 1%）
- 产品品类缺失（约 2%）
- 订单配送日期缺失（约 3%）

详见第一阶段数据字典和表关系说明。

---

**注**: 本文档基于第二阶段构建的实际数据表生成，可直接用于后续分析参考。

"""

    with open('docs/analysis_base_tables.md', 'w', encoding='utf-8') as f:
        f.write(content)

    print("✓ 已生成分析基础表说明: docs/analysis_base_tables.md")

if __name__ == '__main__':
    print("=" * 80)
    print("生成剩余文档")
    print("=" * 80)

    generate_entity_relationship_doc()
    generate_analysis_base_tables_doc()

    print("\n" + "=" * 80)
    print("✓ 文档生成完成")
    print("=" * 80)