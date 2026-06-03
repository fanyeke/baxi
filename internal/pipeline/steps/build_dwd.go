// Package steps implements individual pipeline steps that can be composed
// into a full pipeline run. Each step implements pipeline.Step.
package steps

import (
	"context"
	"fmt"

	"baxi/internal/pipeline"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// BuildDWDSOrderLevelStep builds the dwd.order_level table by joining raw tables
// (orders, customers, payments, reviews) at order granularity.
// Expected output: 99,441 rows (one per order).
type BuildDWDSOrderLevelStep struct{}

// NewBuildDWDSOrderLevelStep creates a new BuildDWDSOrderLevelStep.
func NewBuildDWDSOrderLevelStep() *BuildDWDSOrderLevelStep {
	return &BuildDWDSOrderLevelStep{}
}

// Name returns the step name for audit logging.
func (s *BuildDWDSOrderLevelStep) Name() string {
	return "build_dwd_order_level"
}

// Run executes the step within the given transaction.
// It inserts rows into dwd.order_level by LEFT JOINing raw tables:
//   - raw.olist_orders         (base: order_id, status, timestamps)
//   - raw.olist_customers      (customer_unique_id, state)
//   - raw.olist_order_payments (first payment row per order)
//   - raw.olist_order_reviews  (latest review per order)
//
// Computed columns:
//   - delivery_days: fractional days between purchase and customer delivery
//   - delay_days:    fractional days between estimated and actual delivery
//   - is_late:       TRUE if delivered after estimated, NULL if undelivered
//   - is_cancelled:  TRUE if order_status = 'canceled'
//
// The join is idempotent: ON CONFLICT (order_id) DO NOTHING.
func (s *BuildDWDSOrderLevelStep) Run(ctx context.Context, tx pgx.Tx, input pipeline.StepInput) (*pipeline.StepOutput, error) {
	insertSQL := `
INSERT INTO dwd.order_level (
    order_id, customer_id, customer_unique_id, order_status,
    order_purchase_timestamp, purchase_date, customer_state,
    payment_type, payment_installments, payment_value,
    review_score, delivered_customer_date, estimated_delivery_date,
    delivery_days, delay_days, is_late, is_cancelled,
    ingestion_batch_id, loaded_at, pipeline_run_id, record_hash
)
SELECT
    o.order_id,
    o.customer_id,
    c.customer_unique_id,
    o.order_status,
    o.order_purchase_timestamp,
    o.order_purchase_timestamp::DATE AS purchase_date,
    c.customer_state,
    pa.payment_type,
    pa.payment_installments,
    pa.payment_value,
    r.review_score,
    o.order_delivered_customer_date AS delivered_customer_date,
    o.order_estimated_delivery_date AS estimated_delivery_date,
    CASE
        WHEN o.order_delivered_customer_date IS NOT NULL AND o.order_purchase_timestamp IS NOT NULL
        THEN EXTRACT(EPOCH FROM (o.order_delivered_customer_date - o.order_purchase_timestamp)) / 86400
        ELSE NULL
    END AS delivery_days,
    CASE
        WHEN o.order_delivered_customer_date IS NOT NULL AND o.order_estimated_delivery_date IS NOT NULL
        THEN EXTRACT(EPOCH FROM (o.order_delivered_customer_date - o.order_estimated_delivery_date)) / 86400
        ELSE NULL
    END AS delay_days,
    CASE
        WHEN o.order_delivered_customer_date IS NOT NULL AND o.order_estimated_delivery_date IS NOT NULL
        THEN o.order_delivered_customer_date > o.order_estimated_delivery_date
        ELSE NULL
    END AS is_late,
    CASE WHEN o.order_status = 'canceled' THEN TRUE ELSE FALSE END AS is_cancelled,
    @batch_id         AS ingestion_batch_id,
    NOW()             AS loaded_at,
    @pipeline_run_id  AS pipeline_run_id,
    MD5(o.order_id || '-' || @pipeline_run_id) AS record_hash
FROM raw.olist_orders o
LEFT JOIN raw.olist_customers c ON o.customer_id = c.customer_id
LEFT JOIN (
    SELECT DISTINCT ON (order_id)
        order_id, payment_type, payment_installments, payment_value
    FROM raw.olist_order_payments
    ORDER BY order_id, payment_sequential
) pa ON o.order_id = pa.order_id
LEFT JOIN (
    SELECT DISTINCT ON (order_id)
        order_id, review_score
    FROM raw.olist_order_reviews
    ORDER BY order_id, review_answer_timestamp DESC NULLS LAST
) r ON o.order_id = r.order_id
ORDER BY o.order_id
ON CONFLICT (order_id) DO NOTHING;
`
	args := pgx.NamedArgs{
		"batch_id":        input.RunID,
		"pipeline_run_id": input.RunID,
	}

	result, err := tx.Exec(ctx, insertSQL, args)
	if err != nil {
		return nil, fmt.Errorf("build dwd.order_level: %w", err)
	}

	outputCount := result.RowsAffected()

	// Count input rows from the raw orders table
	var inputCount int64
	countSQL := `SELECT COUNT(*) FROM raw.olist_orders`
	if err := tx.QueryRow(ctx, countSQL).Scan(&inputCount); err != nil {
		return nil, fmt.Errorf("count raw.olist_orders: %w", err)
	}

	input.Logger.Info("dwd.order_level built",
		zap.Int64("input_rows", inputCount),
		zap.Int64("inserted_rows", outputCount),
	)

	return &pipeline.StepOutput{
		InputCount:  inputCount,
		OutputCount: outputCount,
	}, nil
}

// BuildDWDItemLevelStep builds the dwd.item_level table by joining raw tables
// (order_items, products, sellers, category_translation) at order_item granularity.
// Expected output: 112,650 rows (one per order item).
type BuildDWDItemLevelStep struct{}

// NewBuildDWDItemLevelStep creates a new BuildDWDItemLevelStep.
func NewBuildDWDItemLevelStep() *BuildDWDItemLevelStep {
	return &BuildDWDItemLevelStep{}
}

// Name returns the step name for audit logging.
func (s *BuildDWDItemLevelStep) Name() string {
	return "build_dwd_item_level"
}

// Run executes the step within the given transaction.
// It inserts rows into dwd.item_level by LEFT JOINing raw tables:
//   - raw.olist_order_items       (base)
//   - raw.olist_products          (product category)
//   - raw.olist_sellers           (seller state)
//   - raw.product_category_name_translation  (English category name)
//
// The join is idempotent: ON CONFLICT (order_id, order_item_id) DO NOTHING.
func (s *BuildDWDItemLevelStep) Run(ctx context.Context, tx pgx.Tx, input pipeline.StepInput) (*pipeline.StepOutput, error) {
	insertSQL := `
INSERT INTO dwd.item_level (
    order_id, order_item_id, product_id, seller_id,
    product_category_name, product_category_name_english,
    seller_state, price, freight_value,
    ingestion_batch_id, loaded_at, pipeline_run_id, record_hash
)
SELECT
    oi.order_id,
    oi.order_item_id,
    oi.product_id,
    oi.seller_id,
    p.product_category_name,
    COALESCE(pct.product_category_name_english, p.product_category_name) AS product_category_name_english,
    s.seller_state,
    oi.price,
    oi.freight_value,
    @batch_id       AS ingestion_batch_id,
    NOW()           AS loaded_at,
    @pipeline_run_id AS pipeline_run_id,
    MD5(oi.order_id || oi.order_item_id::TEXT || @pipeline_run_id) AS record_hash
FROM raw.olist_order_items oi
LEFT JOIN raw.olist_products p ON oi.product_id = p.product_id
LEFT JOIN raw.olist_sellers s ON oi.seller_id = s.seller_id
LEFT JOIN raw.product_category_name_translation pct
    ON p.product_category_name = pct.product_category_name
ON CONFLICT (order_id, order_item_id) DO NOTHING;
`
	args := pgx.NamedArgs{
		"batch_id":        input.RunID,
		"pipeline_run_id": input.RunID,
	}

	result, err := tx.Exec(ctx, insertSQL, args)
	if err != nil {
		return nil, fmt.Errorf("build dwd.item_level: %w", err)
	}

	outputCount := result.RowsAffected()

	var inputCount int64
	countSQL := `SELECT COUNT(*) FROM raw.olist_order_items`
	if err := tx.QueryRow(ctx, countSQL).Scan(&inputCount); err != nil {
		return nil, fmt.Errorf("count raw.olist_order_items: %w", err)
	}

	input.Logger.Info("dwd.item_level built",
		zap.Int64("input_rows", inputCount),
		zap.Int64("inserted_rows", outputCount),
	)

	return &pipeline.StepOutput{
		InputCount:  inputCount,
		OutputCount: outputCount,
	}, nil
}
