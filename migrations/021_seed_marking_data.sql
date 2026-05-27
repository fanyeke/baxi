-- +goose Up
-- +goose StatementBegin

-- Migration 021: Seed marking data from config/data_markings.yml
-- Populates gov.marking_definition, gov.marking_assignment, gov.pipeline_stage_marking

-- ============================================================
-- 1. Marking definitions (4 markings)
-- ============================================================
INSERT INTO gov.marking_definition (marking_id, display_name, description, mandatory_control, access_type, conjunctive, inheritance_rules, policy, expand_access_permission, is_active)
VALUES
    ('PII',                  'Personal Identifying Information',
        'Customer identity fields requiring strictest protection',
        TRUE, 'binary', TRUE,
        '{file_hierarchy,data_dependency}',
        'Do not expose in frontend or Feishu export',
        'data_protection_officer', TRUE),

    ('OPERATIONAL_INTERNAL', 'Operational Internal',
        'Internal operational data: alerts, recommendations, tasks, outbox',
        TRUE, 'binary', TRUE,
        '{data_dependency}',
        'Visible to data_operators and business_ops roles only',
        'data_protection_officer', TRUE),

    ('FINANCIAL_INTERNAL',   'Financial Internal',
        'Raw payment and pricing fields; aggregated GMV is public_internal',
        TRUE, 'binary', TRUE,
        '{data_dependency}',
        'Aggregated GMV is public_internal; raw payment values are internal',
        'data_protection_officer', TRUE),

    ('RAW_DATA',             'Raw Data',
        'Raw CSV data that may contain unprocessed PII',
        TRUE, 'binary', TRUE,
        '{data_dependency}',
        'Raw data may contain unprocessed PII. Pipeline ingestion should hash/remove before downstream use',
        'data_protection_officer', TRUE)
ON CONFLICT (marking_id) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    description = EXCLUDED.description,
    mandatory_control = EXCLUDED.mandatory_control,
    access_type = EXCLUDED.access_type,
    conjunctive = EXCLUDED.conjunctive,
    inheritance_rules = EXCLUDED.inheritance_rules,
    policy = EXCLUDED.policy,
    expand_access_permission = EXCLUDED.expand_access_permission,
    is_active = EXCLUDED.is_active,
    updated_at = NOW();

-- ============================================================
-- 2. Marking assignments (maps markings to resource paths)
-- ============================================================

-- PII assignments (4 resources)
INSERT INTO gov.marking_assignment (marking_id, resource_type, resource_path)
VALUES
    ('PII', 'column', 'raw_customers.customer_unique_id'),
    ('PII', 'column', 'raw_orders.customer_id'),
    ('PII', 'column', 'dwd_order_level.customer_unique_id'),
    ('PII', 'column', 'dwd_order_level.customer_id')
ON CONFLICT (marking_id, resource_type, resource_path) DO UPDATE SET
    is_active = TRUE;

-- OPERATIONAL_INTERNAL assignments (5 resources)
INSERT INTO gov.marking_assignment (marking_id, resource_type, resource_path)
VALUES
    ('OPERATIONAL_INTERNAL', 'table', 'alert_events'),
    ('OPERATIONAL_INTERNAL', 'table', 'strategy_recommendations'),
    ('OPERATIONAL_INTERNAL', 'table', 'action_tasks'),
    ('OPERATIONAL_INTERNAL', 'table', 'review_retro'),
    ('OPERATIONAL_INTERNAL', 'table', 'event_outbox')
ON CONFLICT (marking_id, resource_type, resource_path) DO UPDATE SET
    is_active = TRUE;

-- FINANCIAL_INTERNAL assignments (4 resources)
INSERT INTO gov.marking_assignment (marking_id, resource_type, resource_path)
VALUES
    ('FINANCIAL_INTERNAL', 'column', 'dwd_order_level.payment_value'),
    ('FINANCIAL_INTERNAL', 'column', 'dwd_item_level.price'),
    ('FINANCIAL_INTERNAL', 'column', 'dwd_item_level.freight_value'),
    ('FINANCIAL_INTERNAL', 'column', 'raw_order_payments.payment_value')
ON CONFLICT (marking_id, resource_type, resource_path) DO UPDATE SET
    is_active = TRUE;

-- RAW_DATA assignments (8 resources)
INSERT INTO gov.marking_assignment (marking_id, resource_type, resource_path)
VALUES
    ('RAW_DATA', 'table', 'raw_customers'),
    ('RAW_DATA', 'table', 'raw_orders'),
    ('RAW_DATA', 'table', 'raw_order_items'),
    ('RAW_DATA', 'table', 'raw_order_payments'),
    ('RAW_DATA', 'table', 'raw_order_reviews'),
    ('RAW_DATA', 'table', 'raw_products'),
    ('RAW_DATA', 'table', 'raw_sellers'),
    ('RAW_DATA', 'table', 'raw_geolocation')
ON CONFLICT (marking_id, resource_type, resource_path) DO UPDATE SET
    is_active = TRUE;

-- ============================================================
-- 3. Pipeline stage markings
-- ============================================================
-- Note: 'processed' stage has marking null in YAML; skipped since marking_id is NOT NULL.
INSERT INTO gov.pipeline_stage_marking (stage_name, marking_id)
VALUES
    ('raw', 'RAW_DATA')
ON CONFLICT (stage_name) DO UPDATE SET
    marking_id = EXCLUDED.marking_id;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Reverse: truncate seeded marking data (order: stage_marking → assignments → definitions)
TRUNCATE TABLE gov.pipeline_stage_marking CASCADE;
TRUNCATE TABLE gov.marking_assignment CASCADE;
TRUNCATE TABLE gov.marking_definition CASCADE;

-- +goose StatementEnd
