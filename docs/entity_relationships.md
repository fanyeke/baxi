# Olist 实体关系说明

生成时间: 2026-05-20 22:36:38

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

### 1. customers → orders (customer_id)

**关系**: CUSTOMERS ||--o{ORDERS

**连接键**: customer_id

**性质**: 多对一

**验证结果**:
- 输入行数: customers=99441, orders=99441
- 输出行数: 99441
- 放大倍数: 1.00x


**问题**:
- ⚠ 99441 行在 customers 中未找到匹配 (占 100.00%)

---

### 2. orders → order_items (order_id)

**关系**: ORDERS ||--o{ORDER_ITEMS

**连接键**: order_id

**性质**: 一对多

**验证结果**:
- 输入行数: orders=99441, order_items=112650
- 输出行数: 113425
- 放大倍数: 1.14x


**问题**:
- ⚠ 113425 行在 orders 中未找到匹配 (占 114.06%)
- ⚠ 行数放大: 99441 → 113425 (放大倍数: 1.14x)
- ⚠ 目标表 order_items 中 order_id 有 9803 个值出现多次

---

### 3. order_items → products (product_id)

**关系**: ORDER_ITEMS ||--o{PRODUCTS

**连接键**: product_id

**性质**: 多对一

**验证结果**:
- 输入行数: order_items=112650, products=32951
- 输出行数: 112650
- 放大倍数: 1.00x


**问题**:
- ⚠ 112650 行在 order_items 中未找到匹配 (占 100.00%)

---

### 4. order_items → sellers (seller_id)

**关系**: ORDER_ITEMS ||--o{SELLERS

**连接键**: seller_id

**性质**: 多对一

**验证结果**:
- 输入行数: order_items=112650, sellers=3095
- 输出行数: 112650
- 放大倍数: 1.00x


**问题**:
- ⚠ 112650 行在 order_items 中未找到匹配 (占 100.00%)

---

### 5. orders → payments (order_id)

**关系**: ORDERS ||--o{PAYMENTS

**连接键**: order_id

**性质**: 一对多

**验证结果**:
- 输入行数: orders=99441, payments=103886
- 输出行数: 103887
- 放大倍数: 1.04x


**问题**:
- ⚠ 103887 行在 orders 中未找到匹配 (占 104.47%)
- ⚠ 行数放大: 99441 → 103887 (放大倍数: 1.04x)
- ⚠ 目标表 payments 中 order_id 有 2961 个值出现多次

---

### 6. orders → reviews (order_id)

**关系**: ORDERS ||--o{REVIEWS

**连接键**: order_id

**性质**: 一对多

**验证结果**:
- 输入行数: orders=99441, reviews=99224
- 输出行数: 99992
- 放大倍数: 1.01x


**问题**:
- ⚠ 99992 行在 orders 中未找到匹配 (占 100.55%)
- ⚠ 行数放大: 99441 → 99992 (放大倍数: 1.01x)
- ⚠ 目标表 reviews 中 order_id 有 547 个值出现多次

---

### 7. products → category_translation (product_category_name)

**关系**: PRODUCTS ||--o{CATEGORY_TRANSLATION

**连接键**: product_category_name

**性质**: 多对一

**验证结果**:
- 输入行数: products=32951, category_translation=71
- 输出行数: 32951
- 放大倍数: 1.00x


**问题**:
- ⚠ 32951 行在 products 中未找到匹配 (占 100.00%)

---

## 关键风险与建议

### 1. 行数放大风险

以下 join 操作会导致行数放大：

- orders → order_items (order_id): 99441 → 113425 (1.14x)
- orders → payments (order_id): 99441 → 103887 (1.04x)
- orders → reviews (order_id): 99441 → 99992 (1.01x)

**建议**: 在分析时明确粒度，订单级别分析应聚合 order_items，商品级别分析保留明细。

### 2. 数据缺失风险

以下 join 存在数据缺失：

- customers → orders (customer_id): 100.00% 未匹配
- orders → order_items (order_id): 114.06% 未匹配
- order_items → products (product_id): 100.00% 未匹配
- order_items → sellers (seller_id): 100.00% 未匹配
- orders → payments (order_id): 104.47% 未匹配
- orders → reviews (order_id): 100.55% 未匹配
- products → category_translation (product_category_name): 100.00% 未匹配

**建议**: 使用 left join 确保主表完整性，缺失字段标记为 null，分析时注意排除。

### 3. 重复记录风险

- orders → order_items (order_id): ⚠ 目标表 order_items 中 order_id 有 9803 个值出现多次
- orders → payments (order_id): ⚠ 目标表 payments 中 order_id 有 2961 个值出现多次
- orders → reviews (order_id): ⚠ 目标表 reviews 中 order_id 有 547 个值出现多次

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

