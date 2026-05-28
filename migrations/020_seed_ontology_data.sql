-- +goose Up
-- +goose StatementBegin

-- Migration 020: Seed ontology data from config/aip_object_schema.yml
-- Populates gov.object_type_registry, gov.object_property, gov.object_relationship

-- ============================================================
-- 1. Object type registry (8 object types)
-- ============================================================
INSERT INTO gov.object_type_registry (object_type_id, display_name, source_tables, grain, is_active)
VALUES
    ('customer',       '客户',     '{order_level_base}',                      'customer_unique_id',        TRUE),
    ('order',          '订单',     '{order_level_base}',                      'order_id',                  TRUE),
    ('seller',         '卖家',     '{item_level_base}',                       'seller_id',                 TRUE),
    ('product',        '产品',     '{item_level_base}',                       'product_id',                TRUE),
    ('category',       '品类',     '{item_level_base}',                       'product_category_name',     TRUE),
    ('region',         '区域',     '{order_level_base,item_level_base}',      'state',                     TRUE),
    ('marketing_lead', '营销线索', '{channel_classification}',                'origin',                    TRUE),
    ('metric_alert',   '异常事件', '{metric_alerts}',                         'alert_id',                  TRUE)
ON CONFLICT (object_type_id) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    source_tables = EXCLUDED.source_tables,
    grain = EXCLUDED.grain,
    is_active = EXCLUDED.is_active,
    updated_at = NOW();

-- ============================================================
-- 2. Object properties (all properties for all 8 object types)
-- ============================================================

-- customer properties (8)
INSERT INTO gov.object_property (object_type_id, property_name, property_type, is_pk, source_column, aggregation)
VALUES
    ('customer', 'customer_unique_id',  'string',   TRUE,  NULL,                          NULL),
    ('customer', 'customer_state',      'string',   FALSE, NULL,                          NULL),
    ('customer', 'customer_city',       'string',   FALSE, NULL,                          NULL),
    ('customer', 'order_count',         'int',      FALSE, 'order_id',                    'nunique'),
    ('customer', 'gmv_total',           'float',    FALSE, 'total_payment_value',         'sum'),
    ('customer', 'avg_review_score',    'float',    FALSE, NULL,                          NULL),
    ('customer', 'first_order_date',    'datetime', FALSE, 'order_purchase_timestamp',    'min'),
    ('customer', 'last_order_date',     'datetime', FALSE, 'order_purchase_timestamp',    'max')
ON CONFLICT (object_type_id, property_name) DO UPDATE SET
    property_type = EXCLUDED.property_type,
    is_pk = EXCLUDED.is_pk,
    source_column = EXCLUDED.source_column,
    aggregation = EXCLUDED.aggregation;

-- order properties (7)
INSERT INTO gov.object_property (object_type_id, property_name, property_type, is_pk, source_column, aggregation)
VALUES
    ('order', 'order_id',                     'string',   TRUE,  NULL, NULL),
    ('order', 'order_status',                 'string',   FALSE, NULL, NULL),
    ('order', 'order_purchase_timestamp',      'datetime', FALSE, NULL, NULL),
    ('order', 'total_payment_value',           'float',    FALSE, NULL, NULL),
    ('order', 'payment_type',                  'string',   FALSE, NULL, NULL),
    ('order', 'review_score',                  'float',    FALSE, NULL, NULL),
    ('order', 'delivery_status',               'string',   FALSE, NULL, NULL)
ON CONFLICT (object_type_id, property_name) DO UPDATE SET
    property_type = EXCLUDED.property_type,
    is_pk = EXCLUDED.is_pk,
    source_column = EXCLUDED.source_column,
    aggregation = EXCLUDED.aggregation;

-- seller properties (7)
INSERT INTO gov.object_property (object_type_id, property_name, property_type, is_pk, source_column, aggregation)
VALUES
    ('seller', 'seller_id',             'string',  TRUE,  NULL,      NULL),
    ('seller', 'seller_state',          'string',  FALSE, NULL,      NULL),
    ('seller', 'seller_city',           'string',  FALSE, NULL,      NULL),
    ('seller', 'gmv',                   'float',   FALSE, 'price',   'sum'),
    ('seller', 'order_count',           'int',     FALSE, 'order_id','nunique'),
    ('seller', 'avg_review_score',      'float',   FALSE, NULL,      NULL),
    ('seller', 'late_delivery_rate',    'float',   FALSE, NULL,      NULL)
ON CONFLICT (object_type_id, property_name) DO UPDATE SET
    property_type = EXCLUDED.property_type,
    is_pk = EXCLUDED.is_pk,
    source_column = EXCLUDED.source_column,
    aggregation = EXCLUDED.aggregation;

-- product properties (8)
INSERT INTO gov.object_property (object_type_id, property_name, property_type, is_pk, source_column, aggregation)
VALUES
    ('product', 'product_id',                        'string',  TRUE,  NULL,      NULL),
    ('product', 'product_category_name',              'string',  FALSE, NULL,      NULL),
    ('product', 'product_category_name_english',      'string',  FALSE, NULL,      NULL),
    ('product', 'price',                              'float',   FALSE, NULL,      NULL),
    ('product', 'freight_value',                      'float',   FALSE, NULL,      NULL),
    ('product', 'product_weight_g',                   'float',   FALSE, NULL,      NULL),
    ('product', 'sales_count',                        'int',     FALSE, 'order_id','count'),
    ('product', 'avg_review_score',                   'float',   FALSE, NULL,      NULL)
ON CONFLICT (object_type_id, property_name) DO UPDATE SET
    property_type = EXCLUDED.property_type,
    is_pk = EXCLUDED.is_pk,
    source_column = EXCLUDED.source_column,
    aggregation = EXCLUDED.aggregation;

-- category properties (6)
INSERT INTO gov.object_property (object_type_id, property_name, property_type, is_pk, source_column, aggregation)
VALUES
    ('category', 'product_category_name',          'string',  TRUE,  NULL,      NULL),
    ('category', 'product_category_name_english',  'string',  FALSE, NULL,      NULL),
    ('category', 'gmv',                            'float',   FALSE, 'price',   'sum'),
    ('category', 'order_count',                    'int',     FALSE, 'order_id','nunique'),
    ('category', 'avg_review_score',               'float',   FALSE, NULL,      NULL),
    ('category', 'late_delivery_rate',             'float',   FALSE, NULL,      NULL)
ON CONFLICT (object_type_id, property_name) DO UPDATE SET
    property_type = EXCLUDED.property_type,
    is_pk = EXCLUDED.is_pk,
    source_column = EXCLUDED.source_column,
    aggregation = EXCLUDED.aggregation;

-- region properties (6)
INSERT INTO gov.object_property (object_type_id, property_name, property_type, is_pk, source_column, aggregation)
VALUES
    ('region', 'state',               'string',  TRUE,  NULL, NULL),
    ('region', 'customer_count',      'int',     FALSE, NULL, NULL),
    ('region', 'seller_count',        'int',     FALSE, NULL, NULL),
    ('region', 'gmv',                 'float',   FALSE, NULL, NULL),
    ('region', 'avg_review_score',    'float',   FALSE, NULL, NULL),
    ('region', 'avg_delivery_days',   'float',   FALSE, NULL, NULL)
ON CONFLICT (object_type_id, property_name) DO UPDATE SET
    property_type = EXCLUDED.property_type,
    is_pk = EXCLUDED.is_pk,
    source_column = EXCLUDED.source_column,
    aggregation = EXCLUDED.aggregation;

-- marketing_lead properties (6)
INSERT INTO gov.object_property (object_type_id, property_name, property_type, is_pk, source_column, aggregation)
VALUES
    ('marketing_lead', 'origin',            'string',  TRUE,  NULL, NULL),
    ('marketing_lead', 'mql_count',         'int',     FALSE, NULL, NULL),
    ('marketing_lead', 'conversion_count',  'int',     FALSE, NULL, NULL),
    ('marketing_lead', 'conversion_rate',   'float',   FALSE, NULL, NULL),
    ('marketing_lead', 'gmv_per_seller',    'float',   FALSE, NULL, NULL),
    ('marketing_lead', 'category',          'string',  FALSE, NULL, NULL)
ON CONFLICT (object_type_id, property_name) DO UPDATE SET
    property_type = EXCLUDED.property_type,
    is_pk = EXCLUDED.is_pk,
    source_column = EXCLUDED.source_column,
    aggregation = EXCLUDED.aggregation;

-- metric_alert properties (8)
INSERT INTO gov.object_property (object_type_id, property_name, property_type, is_pk, source_column, aggregation)
VALUES
    ('metric_alert', 'alert_id',         'string',  TRUE,  NULL, NULL),
    ('metric_alert', 'rule_id',          'string',  FALSE, NULL, NULL),
    ('metric_alert', 'metric',           'string',  FALSE, NULL, NULL),
    ('metric_alert', 'severity',         'string',  FALSE, NULL, NULL),
    ('metric_alert', 'current_value',    'float',   FALSE, NULL, NULL),
    ('metric_alert', 'baseline_value',   'float',   FALSE, NULL, NULL),
    ('metric_alert', 'owner_role',       'string',  FALSE, NULL, NULL),
    ('metric_alert', 'status',           'string',  FALSE, NULL, NULL)
ON CONFLICT (object_type_id, property_name) DO UPDATE SET
    property_type = EXCLUDED.property_type,
    is_pk = EXCLUDED.is_pk,
    source_column = EXCLUDED.source_column,
    aggregation = EXCLUDED.aggregation;

-- ============================================================
-- 3. Object relationships (seller has_items → order, has_products → product)
-- ============================================================
INSERT INTO gov.object_relationship (source_object_type, target_object_type, relationship_name, join_key, cardinality)
VALUES
    ('seller', 'order',   'has_items',    'order_id',    'many_to_many'),
    ('seller', 'product', 'has_products', 'product_id',  'many_to_many')
ON CONFLICT (source_object_type, target_object_type, relationship_name) DO UPDATE SET
    join_key = EXCLUDED.join_key,
    cardinality = EXCLUDED.cardinality;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Reverse: truncate seeded ontology data (order: relationships → properties → types)
TRUNCATE TABLE gov.object_relationship CASCADE;
TRUNCATE TABLE gov.object_property CASCADE;
TRUNCATE TABLE gov.object_type_registry CASCADE;

-- +goose StatementEnd
