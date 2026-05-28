-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS raw.olist_customers (
    customer_id            TEXT PRIMARY KEY,
    customer_unique_id     TEXT,
    customer_zip_code_prefix TEXT,
    customer_city          TEXT,
    customer_state         TEXT,
    ingested_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source_file            TEXT,
    source_row_number      BIGINT,
    raw_hash               TEXT
);

CREATE TABLE IF NOT EXISTS raw.olist_orders (
    order_id                       TEXT PRIMARY KEY,
    customer_id                    TEXT,
    order_status                   TEXT,
    order_purchase_timestamp       TIMESTAMPTZ,
    order_approved_at              TIMESTAMPTZ,
    order_delivered_carrier_date   TIMESTAMPTZ,
    order_delivered_customer_date  TIMESTAMPTZ,
    order_estimated_delivery_date  TIMESTAMPTZ,
    ingested_at                    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source_file                    TEXT,
    source_row_number              BIGINT,
    raw_hash                       TEXT
);

CREATE TABLE IF NOT EXISTS raw.olist_order_items (
    order_id           TEXT,
    order_item_id      BIGINT,
    product_id         TEXT,
    seller_id          TEXT,
    shipping_limit_date TIMESTAMPTZ,
    price              NUMERIC(18,2),
    freight_value      NUMERIC(18,2),
    ingested_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source_file        TEXT,
    source_row_number  BIGINT,
    raw_hash           TEXT,
    PRIMARY KEY (order_id, order_item_id)
);

CREATE TABLE IF NOT EXISTS raw.olist_order_payments (
    order_id            TEXT,
    payment_sequential  BIGINT,
    payment_type        TEXT,
    payment_installments BIGINT,
    payment_value       NUMERIC(18,2),
    ingested_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source_file         TEXT,
    source_row_number   BIGINT,
    raw_hash            TEXT,
    PRIMARY KEY (order_id, payment_sequential)
);

CREATE TABLE IF NOT EXISTS raw.olist_order_reviews (
    review_id               TEXT PRIMARY KEY,
    order_id                TEXT,
    review_score            NUMERIC(4,2),
    review_comment_title    TEXT,
    review_comment_message  TEXT,
    review_creation_date    TIMESTAMPTZ,
    review_answer_timestamp TIMESTAMPTZ,
    ingested_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source_file             TEXT,
    source_row_number       BIGINT,
    raw_hash                TEXT
);

CREATE TABLE IF NOT EXISTS raw.olist_products (
    product_id               TEXT PRIMARY KEY,
    product_category_name    TEXT,
    product_name_lenght      BIGINT,
    product_description_lenght BIGINT,
    product_photos_qty       BIGINT,
    product_weight_g         BIGINT,
    product_length_cm        BIGINT,
    product_height_cm        BIGINT,
    product_width_cm         BIGINT,
    ingested_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source_file              TEXT,
    source_row_number        BIGINT,
    raw_hash                 TEXT
);

CREATE TABLE IF NOT EXISTS raw.olist_sellers (
    seller_id             TEXT PRIMARY KEY,
    seller_zip_code_prefix TEXT,
    seller_city           TEXT,
    seller_state          TEXT,
    ingested_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source_file           TEXT,
    source_row_number     BIGINT,
    raw_hash              TEXT
);

CREATE TABLE IF NOT EXISTS raw.olist_geolocation (
    id                      BIGSERIAL PRIMARY KEY,
    geolocation_zip_code_prefix TEXT,
    geolocation_lat         NUMERIC(10,6),
    geolocation_lng         NUMERIC(10,6),
    geolocation_city        TEXT,
    geolocation_state       TEXT,
    ingested_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source_file             TEXT,
    source_row_number       BIGINT,
    raw_hash                TEXT
);

CREATE TABLE IF NOT EXISTS raw.product_category_name_translation (
    product_category_name        TEXT PRIMARY KEY,
    product_category_name_english TEXT,
    ingested_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source_file                  TEXT,
    source_row_number            BIGINT,
    raw_hash                     TEXT
);

CREATE TABLE IF NOT EXISTS raw.marketing_qualified_leads (
    mql_id            TEXT PRIMARY KEY,
    first_contact_date DATE,
    landing_page_id   TEXT,
    origin            TEXT,
    ingested_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source_file       TEXT,
    source_row_number BIGINT,
    raw_hash          TEXT
);

CREATE TABLE IF NOT EXISTS raw.closed_deals (
    mql_id                        TEXT PRIMARY KEY,
    seller_id                     TEXT,
    sdr_id                        TEXT,
    sr_id                         TEXT,
    won_date                      DATE,
    business_segment              TEXT,
    lead_type                     TEXT,
    lead_behaviour_profile        TEXT,
    has_company                   BOOLEAN,
    has_gtin                      BOOLEAN,
    average_stock                 TEXT,
    business_type                 TEXT,
    declared_product_catalog_size NUMERIC(18,2),
    declared_monthly_revenue      NUMERIC(18,2),
    ingested_at                   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source_file                   TEXT,
    source_row_number             BIGINT,
    raw_hash                      TEXT
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS raw.olist_customers CASCADE;
DROP TABLE IF EXISTS raw.olist_orders CASCADE;
DROP TABLE IF EXISTS raw.olist_order_items CASCADE;
DROP TABLE IF EXISTS raw.olist_order_payments CASCADE;
DROP TABLE IF EXISTS raw.olist_order_reviews CASCADE;
DROP TABLE IF EXISTS raw.olist_products CASCADE;
DROP TABLE IF EXISTS raw.olist_sellers CASCADE;
DROP TABLE IF EXISTS raw.olist_geolocation CASCADE;
DROP TABLE IF EXISTS raw.product_category_name_translation CASCADE;
DROP TABLE IF EXISTS raw.marketing_qualified_leads CASCADE;
DROP TABLE IF EXISTS raw.closed_deals CASCADE;

-- +goose StatementEnd
