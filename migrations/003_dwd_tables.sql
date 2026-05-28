-- +goose Up
-- +goose StatementBegin

-- dwd.order_level: Order-level detail with customer, payment, review, delivery info
-- Source: SQLite dwd_order_level (99,441 rows)
CREATE TABLE dwd.order_level (
    order_id                TEXT                PRIMARY KEY,
    customer_id             TEXT,
    customer_unique_id      TEXT,
    order_status            TEXT,
    order_purchase_timestamp TIMESTAMPTZ,
    purchase_date           DATE,
    customer_state          TEXT,
    payment_type            TEXT,
    payment_installments    BIGINT,
    payment_value           NUMERIC(18,2),
    review_score            NUMERIC(4,2),
    delivered_customer_date TIMESTAMPTZ,
    estimated_delivery_date TIMESTAMPTZ,
    delivery_days           NUMERIC(10,6),
    delay_days              NUMERIC(10,6),
    is_late                 BOOLEAN,
    is_cancelled            BOOLEAN,
    ingestion_batch_id      TEXT,
    loaded_at               TIMESTAMPTZ,
    created_at              TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ,
    pipeline_run_id         TEXT,
    record_hash             TEXT
);

-- dwd.item_level: Item-level detail with product, seller, category info
-- Source: SQLite dwd_item_level (112,650 rows)
-- item_key replaced by natural composite PK (order_id, order_item_id)
CREATE TABLE dwd.item_level (
    order_id                    TEXT            NOT NULL,
    order_item_id               BIGINT          NOT NULL,
    product_id                  TEXT,
    seller_id                   TEXT,
    product_category_name       TEXT,
    product_category_name_english TEXT,
    seller_state                TEXT,
    price                       NUMERIC(18,2),
    freight_value               NUMERIC(18,2),
    ingestion_batch_id          TEXT,
    loaded_at                   TIMESTAMPTZ,
    created_at                  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ,
    pipeline_run_id             TEXT,
    record_hash                 TEXT,
    PRIMARY KEY (order_id, order_item_id)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS dwd.item_level;
DROP TABLE IF EXISTS dwd.order_level;

-- +goose StatementEnd
