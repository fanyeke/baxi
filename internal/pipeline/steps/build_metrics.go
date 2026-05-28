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

// BuildMetricDailyStep builds the mart.metric_daily table by aggregating
// dwd.order_level and dwd.item_level data at daily granularity.
// Expected output: 634 rows (one per day with orders, matching baseline).
//
// The step uses TRUNCATE + INSERT (full reload) for idempotency.
// Item-level metrics (freight_value, seller_count) are aggregated via
// a pre-joined subquery to avoid row fan-out from the 1:N order→items join.
type BuildMetricDailyStep struct{}

// NewBuildMetricDailyStep creates a new BuildMetricDailyStep.
func NewBuildMetricDailyStep() *BuildMetricDailyStep {
	return &BuildMetricDailyStep{}
}

// Name returns the step name for audit logging.
func (s *BuildMetricDailyStep) Name() string {
	return "build_metric_daily"
}

// Run executes the step within the given transaction.
// It TRUNCATEs mart.metric_daily, then INSERTs aggregated daily metrics
// from dwd.order_level LEFT JOIN dwd.item_level.
//
// Metrics computed per the Python baseline (db_calculate_metrics.py):
//   - order_count:       DISTINCT order_id count
//   - customer_count:    DISTINCT customer_unique_id count
//   - seller_count:      DISTINCT seller_id from item_level
//   - gmv:               SUM of payment_value (order-level)
//   - avg_order_value:   gmv / order_count
//   - freight_value:     SUM of item_level freight_value
//   - avg_review_score:  AVG of review_score (order-level)
//   - low_review_rate:   reviews ≤ 2 / reviews with score
//   - late_delivery_rate: delivered AND late / delivered orders
//   - cancel_rate:       cancelled / total orders
//   - payment_installment_rate: >1 installment / total orders
//   - marketing_seller_share: 0 (placeholder)
func (s *BuildMetricDailyStep) Run(ctx context.Context, tx pgx.Tx, input pipeline.StepInput) (*pipeline.StepOutput, error) {
	// TRUNCATE for full reload idempotency
	if _, err := tx.Exec(ctx, "TRUNCATE TABLE mart.metric_daily"); err != nil {
		return nil, fmt.Errorf("truncate mart.metric_daily: %w", err)
	}

	insertSQL := `
INSERT INTO mart.metric_daily (
    metric_date, gmv, order_count, customer_count, seller_count,
    avg_order_value, freight_value, avg_review_score,
    low_review_rate, late_delivery_rate, cancel_rate,
    payment_installment_rate, marketing_seller_share, created_at
)
WITH daily_item_metrics AS (
    SELECT
        o.purchase_date,
        COALESCE(SUM(i.freight_value), 0) AS freight_value,
        COUNT(DISTINCT i.seller_id) AS seller_count
    FROM dwd.order_level o
    LEFT JOIN dwd.item_level i ON o.order_id = i.order_id
    WHERE o.purchase_date IS NOT NULL
    GROUP BY o.purchase_date
)
SELECT
    o.purchase_date AS metric_date,
    ROUND(COALESCE(SUM(o.payment_value), 0)::numeric, 2) AS gmv,
    COUNT(DISTINCT o.order_id) AS order_count,
    COUNT(DISTINCT o.customer_unique_id) AS customer_count,
    COALESCE(MAX(dim.seller_count), 0) AS seller_count,
    ROUND(
        (COALESCE(SUM(o.payment_value), 0)
         / NULLIF(COUNT(DISTINCT o.order_id), 0))::numeric, 2
    ) AS avg_order_value,
    COALESCE(MAX(dim.freight_value), 0) AS freight_value,
    ROUND(COALESCE(AVG(o.review_score), 0)::numeric, 4) AS avg_review_score,
    ROUND(
        COALESCE(
            SUM(CASE WHEN o.review_score IS NOT NULL AND o.review_score <= 2 THEN 1 ELSE 0 END)::numeric
            / NULLIF(COUNT(CASE WHEN o.review_score IS NOT NULL THEN 1 END), 0), 0
        ), 4
    ) AS low_review_rate,
    ROUND(
        COALESCE(
            SUM(CASE WHEN o.order_status = 'delivered' AND o.is_late THEN 1 ELSE 0 END)::numeric
            / NULLIF(COUNT(CASE WHEN o.order_status = 'delivered' THEN 1 END), 0), 0
        ), 4
    ) AS late_delivery_rate,
    ROUND(
        COALESCE(
            SUM(CASE WHEN o.is_cancelled THEN 1 ELSE 0 END)::numeric
            / NULLIF(COUNT(o.order_id), 0), 0
        ), 4
    ) AS cancel_rate,
    ROUND(
        COALESCE(
            SUM(CASE WHEN o.payment_installments > 1 THEN 1 ELSE 0 END)::numeric
            / NULLIF(COUNT(o.order_id), 0), 0
        ), 4
    ) AS payment_installment_rate,
    0 AS marketing_seller_share,
    NOW() AS created_at
FROM dwd.order_level o
LEFT JOIN daily_item_metrics dim ON o.purchase_date = dim.purchase_date
WHERE o.purchase_date IS NOT NULL
GROUP BY o.purchase_date, dim.seller_count, dim.freight_value
ORDER BY o.purchase_date;
`
	result, err := tx.Exec(ctx, insertSQL)
	if err != nil {
		return nil, fmt.Errorf("build mart.metric_daily: %w", err)
	}

	outputCount := result.RowsAffected()

	// Count input rows as number of distinct purchase dates in dwd.order_level
	var inputCount int64
	countSQL := `SELECT COUNT(DISTINCT purchase_date) FROM dwd.order_level WHERE purchase_date IS NOT NULL`
	if err := tx.QueryRow(ctx, countSQL).Scan(&inputCount); err != nil {
		return nil, fmt.Errorf("count distinct purchase dates: %w", err)
	}

	input.Logger.Info("mart.metric_daily built",
		zap.Int64("distinct_dates", inputCount),
		zap.Int64("inserted_rows", outputCount),
	)

	return &pipeline.StepOutput{
		InputCount:  inputCount,
		OutputCount: outputCount,
	}, nil
}

// BuildMetricDimensionDailyStep populates mart.metric_dimension_daily with
// per-dimension metrics (seller, category, region) in EAV format.
// Expected output: 693,602 rows (3 dimensions × 7 metrics per unique date+dim-value).
type BuildMetricDimensionDailyStep struct{}

// NewBuildMetricDimensionDailyStep creates a new BuildMetricDimensionDailyStep.
func NewBuildMetricDimensionDailyStep() *BuildMetricDimensionDailyStep {
	return &BuildMetricDimensionDailyStep{}
}

// Name returns the step name for audit logging.
func (s *BuildMetricDimensionDailyStep) Name() string {
	return "build_metric_dimension_daily"
}

// Run executes the step within the given transaction.
// It computes 7 metrics per dimension (seller, category, region) across
// all dates, unpivoting each aggregated row into 7 metric_name rows.
//
// The step is idempotent: ON CONFLICT (metric_date, dimension_type, dimension_value, metric_name)
// DO UPDATE replaces the metric_value and sample_size.
//
// Dimensions:
//   - seller:   group by dwd.item_level.seller_id
//   - category: group by dwd.item_level.product_category_name_english
//   - region:   group by dwd.order_level.customer_state
//
// Metrics per dimension (all 7):
//   gmv, order_count, customer_count, avg_order_value,
//   avg_review_score, late_delivery_rate, cancel_rate
func (s *BuildMetricDimensionDailyStep) Run(ctx context.Context, tx pgx.Tx, input pipeline.StepInput) (*pipeline.StepOutput, error) {
	insertSQL := `
WITH
seller_agg AS (
    SELECT
        o.purchase_date                                          AS metric_date,
        i.seller_id                                              AS dimension_value,
        ROUND(COALESCE(SUM(i.price), 0)::numeric, 4)             AS gmv,
        COUNT(DISTINCT o.order_id)::numeric                      AS order_count,
        COUNT(DISTINCT o.customer_unique_id)::numeric             AS customer_count,
        ROUND(COALESCE(SUM(i.price)::numeric /
            NULLIF(COUNT(DISTINCT o.order_id), 0), 0), 4)        AS avg_order_value,
        ROUND(COALESCE(AVG(o.review_score), 0)::numeric, 4)     AS avg_review_score,
        ROUND(COALESCE(
            SUM(CASE WHEN o.order_status = 'delivered' AND o.is_late THEN 1 ELSE 0 END)::numeric
            / NULLIF(SUM(CASE WHEN o.order_status = 'delivered' THEN 1 ELSE 0 END), 0), 0
        ), 4)                                                    AS late_delivery_rate,
        ROUND(COALESCE(
            SUM(CASE WHEN o.is_cancelled THEN 1 ELSE 0 END)::numeric
            / NULLIF(COUNT(o.order_id), 0), 0
        ), 4)                                                    AS cancel_rate,
        COUNT(DISTINCT o.order_id)                                AS sample_size
    FROM dwd.item_level i
    JOIN dwd.order_level o ON i.order_id = o.order_id
    WHERE o.purchase_date IS NOT NULL
      AND i.seller_id IS NOT NULL
    GROUP BY o.purchase_date, i.seller_id
),
category_agg AS (
    SELECT
        o.purchase_date                                          AS metric_date,
        i.product_category_name_english                           AS dimension_value,
        ROUND(COALESCE(SUM(i.price), 0)::numeric, 4)             AS gmv,
        COUNT(DISTINCT o.order_id)::numeric                      AS order_count,
        COUNT(DISTINCT o.customer_unique_id)::numeric             AS customer_count,
        ROUND(COALESCE(SUM(i.price)::numeric /
            NULLIF(COUNT(DISTINCT o.order_id), 0), 0), 4)        AS avg_order_value,
        ROUND(COALESCE(AVG(o.review_score), 0)::numeric, 4)     AS avg_review_score,
        ROUND(COALESCE(
            SUM(CASE WHEN o.order_status = 'delivered' AND o.is_late THEN 1 ELSE 0 END)::numeric
            / NULLIF(SUM(CASE WHEN o.order_status = 'delivered' THEN 1 ELSE 0 END), 0), 0
        ), 4)                                                    AS late_delivery_rate,
        ROUND(COALESCE(
            SUM(CASE WHEN o.is_cancelled THEN 1 ELSE 0 END)::numeric
            / NULLIF(COUNT(o.order_id), 0), 0
        ), 4)                                                    AS cancel_rate,
        COUNT(DISTINCT o.order_id)                                AS sample_size
    FROM dwd.item_level i
    JOIN dwd.order_level o ON i.order_id = o.order_id
    WHERE o.purchase_date IS NOT NULL
      AND i.product_category_name_english IS NOT NULL
    GROUP BY o.purchase_date, i.product_category_name_english
),
region_agg AS (
    SELECT
        o.purchase_date                                          AS metric_date,
        o.customer_state                                         AS dimension_value,
        ROUND(COALESCE(SUM(i.price), 0)::numeric, 4)             AS gmv,
        COUNT(DISTINCT o.order_id)::numeric                      AS order_count,
        COUNT(DISTINCT o.customer_unique_id)::numeric             AS customer_count,
        ROUND(COALESCE(SUM(i.price)::numeric /
            NULLIF(COUNT(DISTINCT o.order_id), 0), 0), 4)        AS avg_order_value,
        ROUND(COALESCE(AVG(o.review_score), 0)::numeric, 4)     AS avg_review_score,
        ROUND(COALESCE(
            SUM(CASE WHEN o.order_status = 'delivered' AND o.is_late THEN 1 ELSE 0 END)::numeric
            / NULLIF(SUM(CASE WHEN o.order_status = 'delivered' THEN 1 ELSE 0 END), 0), 0
        ), 4)                                                    AS late_delivery_rate,
        ROUND(COALESCE(
            SUM(CASE WHEN o.is_cancelled THEN 1 ELSE 0 END)::numeric
            / NULLIF(COUNT(o.order_id), 0), 0
        ), 4)                                                    AS cancel_rate,
        COUNT(DISTINCT o.order_id)                                AS sample_size
    FROM dwd.item_level i
    JOIN dwd.order_level o ON i.order_id = o.order_id
    WHERE o.purchase_date IS NOT NULL
      AND o.customer_state IS NOT NULL
    GROUP BY o.purchase_date, o.customer_state
)
INSERT INTO mart.metric_dimension_daily (
    metric_date, dimension_type, dimension_value,
    metric_name, metric_value, sample_size, created_at
)

-- Seller dimension (7 metric rows per date-seller)
SELECT metric_date, 'seller', dimension_value, 'gmv',             gmv,             sample_size, NOW() FROM seller_agg
UNION ALL
SELECT metric_date, 'seller', dimension_value, 'order_count',     order_count,     sample_size, NOW() FROM seller_agg
UNION ALL
SELECT metric_date, 'seller', dimension_value, 'customer_count',  customer_count,  sample_size, NOW() FROM seller_agg
UNION ALL
SELECT metric_date, 'seller', dimension_value, 'avg_order_value', avg_order_value, sample_size, NOW() FROM seller_agg
UNION ALL
SELECT metric_date, 'seller', dimension_value, 'avg_review_score', avg_review_score, sample_size, NOW() FROM seller_agg
UNION ALL
SELECT metric_date, 'seller', dimension_value, 'late_delivery_rate', late_delivery_rate, sample_size, NOW() FROM seller_agg
UNION ALL
SELECT metric_date, 'seller', dimension_value, 'cancel_rate',     cancel_rate,     sample_size, NOW() FROM seller_agg

UNION ALL

-- Category dimension (7 metric rows per date-category)
SELECT metric_date, 'category', dimension_value, 'gmv',             gmv,             sample_size, NOW() FROM category_agg
UNION ALL
SELECT metric_date, 'category', dimension_value, 'order_count',     order_count,     sample_size, NOW() FROM category_agg
UNION ALL
SELECT metric_date, 'category', dimension_value, 'customer_count',  customer_count,  sample_size, NOW() FROM category_agg
UNION ALL
SELECT metric_date, 'category', dimension_value, 'avg_order_value', avg_order_value, sample_size, NOW() FROM category_agg
UNION ALL
SELECT metric_date, 'category', dimension_value, 'avg_review_score', avg_review_score, sample_size, NOW() FROM category_agg
UNION ALL
SELECT metric_date, 'category', dimension_value, 'late_delivery_rate', late_delivery_rate, sample_size, NOW() FROM category_agg
UNION ALL
SELECT metric_date, 'category', dimension_value, 'cancel_rate',     cancel_rate,     sample_size, NOW() FROM category_agg

UNION ALL

-- Region dimension (7 metric rows per date-region)
SELECT metric_date, 'region', dimension_value, 'gmv',             gmv,             sample_size, NOW() FROM region_agg
UNION ALL
SELECT metric_date, 'region', dimension_value, 'order_count',     order_count,     sample_size, NOW() FROM region_agg
UNION ALL
SELECT metric_date, 'region', dimension_value, 'customer_count',  customer_count,  sample_size, NOW() FROM region_agg
UNION ALL
SELECT metric_date, 'region', dimension_value, 'avg_order_value', avg_order_value, sample_size, NOW() FROM region_agg
UNION ALL
SELECT metric_date, 'region', dimension_value, 'avg_review_score', avg_review_score, sample_size, NOW() FROM region_agg
UNION ALL
SELECT metric_date, 'region', dimension_value, 'late_delivery_rate', late_delivery_rate, sample_size, NOW() FROM region_agg
UNION ALL
SELECT metric_date, 'region', dimension_value, 'cancel_rate',     cancel_rate,     sample_size, NOW() FROM region_agg
ON CONFLICT (metric_date, dimension_type, dimension_value, metric_name)
DO UPDATE SET
    metric_value = EXCLUDED.metric_value,
    sample_size  = EXCLUDED.sample_size,
    created_at   = NOW();
`

	result, err := tx.Exec(ctx, insertSQL)
	if err != nil {
		return nil, fmt.Errorf("build mart.metric_dimension_daily: %w", err)
	}

	outputCount := result.RowsAffected()

	// Count rows in the source tables for input metrics
	var inputCount int64
	countSQL := `SELECT COUNT(*) FROM dwd.item_level`
	if err := tx.QueryRow(ctx, countSQL).Scan(&inputCount); err != nil {
		return nil, fmt.Errorf("count dwd.item_level: %w", err)
	}

	input.Logger.Info("mart.metric_dimension_daily built",
		zap.Int64("input_item_rows", inputCount),
		zap.Int64("inserted_rows", outputCount),
	)

	return &pipeline.StepOutput{
		InputCount:  inputCount,
		OutputCount: outputCount,
	}, nil
}
