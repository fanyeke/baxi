# 分析基础表说明

生成时间: 2026-05-22 11:03:08

---

## 概述

为后续分析构建了两个标准化基础表，作为分析底座。

---

## 1. order_level_base.csv

### 基本信息

- **文件路径**: `data/interim/order_level_base.csv`
- **行数**: 99441
- **列数**: 22
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
- **行数**: 112650
- **列数**: 36
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
| 行数 | 99441 | 112650 |
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

