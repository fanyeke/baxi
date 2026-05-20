# Olist 表关系分析

生成时间: 2026-05-20 22:27:24

---

## 表概览

| 数据集 | 表名 | 行数 | 列数 | 可能主键 | 可能外键 |
|--------|------|------|------|----------|----------|
| 主数据集 | olist_customers_dataset | 99,441 | 5 | customer_id | customer_id, customer_unique_id |
| 主数据集 | olist_geolocation_dataset | 1,000,163 | 5 | - | - |
| 主数据集 | olist_order_items_dataset | 112,650 | 7 | - | order_id, order_item_id, product_id, seller_id |
| 主数据集 | olist_order_payments_dataset | 103,886 | 5 | - | order_id |
| 主数据集 | olist_order_reviews_dataset | 99,224 | 7 | - | review_id, order_id |
| 主数据集 | olist_orders_dataset | 99,441 | 8 | order_id, customer_id | order_id, customer_id |
| 主数据集 | olist_products_dataset | 32,951 | 9 | product_id | product_id |
| 主数据集 | olist_sellers_dataset | 3,095 | 4 | seller_id | seller_id |
| 主数据集 | product_category_name_translation | 71 | 2 | product_category_name, product_category_name_english | - |

---

## 表关系推断

### 主数据集内部关系

- **olist_customers_dataset.customer_id -> olist_orders_dataset.customer_id**
- **olist_order_items_dataset.order_id -> olist_order_payments_dataset.order_id (待验证)**
- **olist_order_items_dataset.order_id -> olist_order_reviews_dataset.order_id (待验证)**
- **olist_order_items_dataset.order_id -> olist_orders_dataset.order_id**
- **olist_order_items_dataset.product_id -> olist_products_dataset.product_id**
- **olist_order_items_dataset.seller_id -> olist_sellers_dataset.seller_id**
- **olist_order_payments_dataset.order_id -> olist_order_items_dataset.order_id (待验证)**
- **olist_order_payments_dataset.order_id -> olist_order_reviews_dataset.order_id (待验证)**
- **olist_order_payments_dataset.order_id -> olist_orders_dataset.order_id**
- **olist_order_reviews_dataset.order_id -> olist_order_items_dataset.order_id (待验证)**
- **olist_order_reviews_dataset.order_id -> olist_order_payments_dataset.order_id (待验证)**
- **olist_order_reviews_dataset.order_id -> olist_orders_dataset.order_id**
- **olist_orders_dataset.order_id -> olist_order_items_dataset.order_id (待验证)**
- **olist_orders_dataset.order_id -> olist_order_payments_dataset.order_id (待验证)**
- **olist_orders_dataset.order_id -> olist_order_reviews_dataset.order_id (待验证)**
- **olist_orders_dataset.customer_id -> olist_customers_dataset.customer_id**
- **olist_products_dataset.product_id -> olist_order_items_dataset.product_id (待验证)**
- **olist_sellers_dataset.seller_id -> olist_order_items_dataset.seller_id (待验证)**

---

## 数据质量说明

### 主数据集

- **olist_geolocation_dataset**: 存在 261831 行重复数据 (26.18%)
- **olist_order_reviews_dataset**: 以下字段存在缺失值
  - `review_comment_title`: 87656 (88.34%)
  - `review_comment_message`: 58247 (58.70%)
- **olist_orders_dataset**: 以下字段存在缺失值
  - `order_approved_at`: 160 (0.16%)
  - `order_delivered_carrier_date`: 1783 (1.79%)
  - `order_delivered_customer_date`: 2965 (2.98%)
- **olist_products_dataset**: 以下字段存在缺失值
  - `product_category_name`: 610 (1.85%)
  - `product_name_lenght`: 610 (1.85%)
  - `product_description_lenght`: 610 (1.85%)
  - `product_photos_qty`: 610 (1.85%)
  - `product_weight_g`: 2 (0.01%)
  - `product_length_cm`: 2 (0.01%)
  - `product_height_cm`: 2 (0.01%)
  - `product_width_cm`: 2 (0.01%)

---

**注**: 以上关系为初步推断，需要通过实际查询验证。
