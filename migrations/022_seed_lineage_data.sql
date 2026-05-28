-- +goose Up
-- +goose StatementBegin

-- Migration 022: Seed lineage data from config/data_lineage.yml
-- Creates gov.lineage_node + gov.lineage_edge tables and populates them.

-- ============================================================
-- 1. Create lineage tables (not yet in schema)
-- ============================================================
CREATE TABLE IF NOT EXISTS gov.lineage_node (
    node_id     TEXT PRIMARY KEY,
    node_type   TEXT NOT NULL,
    label       TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'active',
    linked_to   TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE gov.lineage_node IS 'Data lineage nodes: sources, datasets, object types, syncs, issues';
COMMENT ON COLUMN gov.lineage_node.node_type IS 'Node kind: source, dataset, object_type, sync, issue';

CREATE TABLE IF NOT EXISTS gov.lineage_edge (
    edge_id         BIGSERIAL PRIMARY KEY,
    source_node_id  TEXT NOT NULL REFERENCES gov.lineage_node(node_id),
    target_node_id  TEXT NOT NULL REFERENCES gov.lineage_node(node_id),
    description     TEXT,
    transform_type  TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_lineage_edge UNIQUE (source_node_id, target_node_id, transform_type)
);

COMMENT ON TABLE gov.lineage_edge IS 'Data lineage edges: directed transformations between nodes';

CREATE INDEX IF NOT EXISTS idx_gov_lineage_edge_source ON gov.lineage_edge(source_node_id);
CREATE INDEX IF NOT EXISTS idx_gov_lineage_edge_target ON gov.lineage_edge(target_node_id);
CREATE INDEX IF NOT EXISTS idx_gov_lineage_node_type ON gov.lineage_node(node_type);

-- ============================================================
-- 2. Lineage nodes (26 nodes from YAML + 1 implied review_retro)
-- ============================================================
INSERT INTO gov.lineage_node (node_id, node_type, label, status, linked_to)
VALUES
    -- Source nodes (11 CSV files)
    ('raw_orders_csv',              'source',       'olist_orders_dataset.csv',                          'active', NULL),
    ('raw_customers_csv',           'source',       'olist_customers_dataset.csv',                       'active', NULL),
    ('raw_order_items_csv',         'source',       'olist_order_items_dataset.csv',                     'active', NULL),
    ('raw_order_payments_csv',      'source',       'olist_order_payments_dataset.csv',                  'active', NULL),
    ('raw_order_reviews_csv',       'source',       'olist_order_reviews_dataset.csv',                   'active', NULL),
    ('raw_products_csv',            'source',       'olist_products_dataset.csv',                        'active', NULL),
    ('raw_sellers_csv',             'source',       'olist_sellers_dataset.csv',                         'active', NULL),
    ('raw_geolocation_csv',         'source',       'olist_geolocation_dataset.csv (26% dupes)',         'active', NULL),
    ('raw_category_translation_csv','source',       'product_category_name_translation.csv',             'active', NULL),
    ('raw_mql_csv',                 'source',       'olist_marketing_qualified_leads_dataset.csv',       'active', NULL),
    ('raw_closed_deals_csv',        'source',       'olist_closed_deals_dataset.csv',                    'active', NULL),

    -- Issue node
    ('issue_geo_duplicates',        'issue',        'Geolocation 261,831 dup rows',                      'active', 'raw_geolocation_csv'),

    -- Dataset nodes (DWD + metrics + outbox)
    ('dwd_order_level',             'dataset',      'dwd_order_level (1r/order, 22c)',                   'active', NULL),
    ('dwd_item_level',              'dataset',      'dwd_item_level (1r/item, 18c)',                     'active', NULL),
    ('metric_daily',                'dataset',      'metric_daily (1r/day, 13m)',                        'active', NULL),
    ('metric_dimension_daily',      'dataset',      'metric_dimension_daily (by seller/category/region)','active', NULL),
    ('event_outbox',                'dataset',      'event_outbox (dispatch queue)',                     'active', NULL),

    -- Object type nodes
    ('alert_events',                'object_type',  'alert_events',                                      'active', NULL),
    ('strategy_recommendations',    'object_type',  'strategy_recommendations',                          'active', NULL),
    ('action_tasks',                'object_type',  'action_tasks',                                      'active', NULL),
    ('review_retro',                'object_type',  'review_retro',                                      'active', NULL),

    -- Sync nodes (Feishu)
    ('feishu_daily_metrics',        'sync',         '飞书-每日经营指标',                                  'active', NULL),
    ('feishu_alert_events',         'sync',         '飞书-异常事件表',                                    'active', NULL),
    ('feishu_action_tasks',         'sync',         '飞书-负责人任务表',                                  'active', NULL),
    ('feishu_recommendations',      'sync',         '飞书-策略建议表',                                    'active', NULL),
    ('feishu_review_retro',         'sync',         '飞书-执行复盘表',                                    'active', NULL)
ON CONFLICT (node_id) DO UPDATE SET
    node_type = EXCLUDED.node_type,
    label = EXCLUDED.label,
    status = EXCLUDED.status,
    linked_to = EXCLUDED.linked_to;

-- ============================================================
-- 3. Lineage edges (17 edges from YAML)
-- ============================================================
INSERT INTO gov.lineage_edge (source_node_id, target_node_id, description, transform_type)
VALUES
    -- Source → DWD (batch_load)
    ('raw_orders_csv',          'dwd_order_level',      'CSV ingestion: orders + payments + reviews',       'batch_load'),
    ('raw_customers_csv',       'dwd_order_level',      'join: customer_id + customer_unique_id',           'batch_load'),
    ('raw_order_items_csv',     'dwd_item_level',        'CSV ingestion: items + products + sellers',        'batch_load'),
    ('raw_products_csv',        'dwd_item_level',        'join: product_category_name_english',              'batch_load'),
    ('raw_sellers_csv',         'dwd_item_level',        'join: seller_state',                               'batch_load'),

    -- DWD → Metrics (sql_aggregation)
    ('dwd_order_level',         'metric_daily',          'daily GMV, orders, review_score, late_rate, cancel_rate',  'sql_aggregation'),
    ('dwd_item_level',          'metric_daily',          'daily seller_count, freight_value',                        'sql_aggregation'),
    ('dwd_order_level',         'metric_dimension_daily','dimension grouping seller/category/region',                'sql_aggregation'),
    ('dwd_item_level',          'metric_dimension_daily','dimension grouping seller/category/region',                'sql_aggregation'),

    -- Metrics → Alert (heuristic_rule)
    ('metric_daily',            'alert_events',          'global anomaly detection (5 rules)',                'heuristic_rule'),
    ('metric_dimension_daily',  'alert_events',          'dimensional anomaly detection',                     'heuristic_rule'),

    -- Alert → Recommendations → Tasks (heuristic_rule, template)
    ('alert_events',            'strategy_recommendations', 'heuristic template to recommendation',           'heuristic_rule'),
    ('strategy_recommendations','action_tasks',              'recommendation to work item',                   'template_instantiation'),

    -- Tasks → Outbox → Feishu syncs (channel_routing, api_sync)
    ('action_tasks',            'event_outbox',          'severity to channel routing',                       'channel_routing'),
    ('event_outbox',            'feishu_daily_metrics',  'export metrics to Feishu',                          'api_sync'),
    ('event_outbox',            'feishu_alert_events',   'dispatch alert to Feishu chat',                     'api_sync'),

    -- Feishu → Review (api_sync)
    ('feishu_action_tasks',     'review_retro',          'import human status to SQLite',                     'api_sync')
ON CONFLICT (source_node_id, target_node_id, transform_type) DO UPDATE SET
    description = EXCLUDED.description;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Reverse: truncate lineage data then drop tables
TRUNCATE TABLE gov.lineage_edge CASCADE;
TRUNCATE TABLE gov.lineage_node CASCADE;
DROP TABLE IF EXISTS gov.lineage_edge;
DROP TABLE IF EXISTS gov.lineage_node;

-- +goose StatementEnd
