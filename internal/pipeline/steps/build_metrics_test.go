package steps

import (
	"context"
	"fmt"
	"os"
	"testing"

	"baxi/internal/ingest"
	"baxi/internal/pipeline"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Test DDL for mart.metric_dimension_daily. Mirrors migrations/004_mart_tables.sql.
// ---------------------------------------------------------------------------

const metricDimensionDailyDDL = `
CREATE SCHEMA IF NOT EXISTS mart;

CREATE TABLE IF NOT EXISTS mart.metric_dimension_daily (
    metric_date      DATE NOT NULL,
    dimension_type   TEXT NOT NULL,
    dimension_value  TEXT NOT NULL,
    metric_name      TEXT NOT NULL,
    metric_value     NUMERIC(18,4),
    sample_size      BIGINT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (metric_date, dimension_type, dimension_value, metric_name)
);
`

// ---------------------------------------------------------------------------
// Test setup helper
// ---------------------------------------------------------------------------

// setupBuildMetricDimTestDB creates raw, dwd, and mart tables needed for
// metric_dimension_daily tests. It loads raw tables via IngestRawStep,
// builds DWD tables via BuildDWDSOrderLevelStep and BuildDWDItemLevelStep,
// then creates the target mart table.
func setupBuildMetricDimTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	t.Cleanup(pool.Close)

	ctx := context.Background()

	// Create raw tables
	if _, err := pool.Exec(ctx, rawTableDDL); err != nil {
		t.Fatalf("create raw tables: %v", err)
	}

	// Create dwd tables
	if _, err := pool.Exec(ctx, dwdOrderLevelDDL); err != nil {
		t.Fatalf("create dwd.order_level: %v", err)
	}
	if _, err := pool.Exec(ctx, dwdItemLevelDDL); err != nil {
		t.Fatalf("create dwd.item_level: %v", err)
	}

	// Create mart tables
	if _, err := pool.Exec(ctx, metricDimensionDailyDDL); err != nil {
		t.Fatalf("create mart.metric_dimension_daily: %v", err)
	}

	// Clean data from previous runs
	for _, mapping := range ingest.AllTableMappings() {
		_, _ = pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s", mapping.TableName))
	}
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE dwd.order_level")
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE dwd.item_level")
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE mart.metric_dimension_daily")

	return pool
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestBuildMetricDim_Name(t *testing.T) {
	step := NewBuildMetricDimensionDailyStep()
	if got := step.Name(); got != "build_metric_dimension_daily" {
		t.Errorf("expected name 'build_metric_dimension_daily', got %q", got)
	}
}

func TestBuildMetricDim_HappyPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupBuildMetricDimTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	dataDir := t.TempDir()
	writeTestCSVs(t, dataDir)

	// Step 1: Load raw tables
	ingestStep := NewIngestRawStep()
	func() {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin tx for ingest: %v", err)
		}
		defer tx.Rollback(ctx)

		_, err = ingestStep.Run(ctx, tx, pipeline.StepInput{
			DataDir: dataDir,
			Logger:  zap.NewNop(),
		})
		if err != nil {
			t.Fatalf("ingest step failed: %v", err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit ingest: %v", err)
		}
	}()

	// Step 2: Build dwd.order_level
	orderStep := NewBuildDWDSOrderLevelStep()
	func() {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin tx for order build: %v", err)
		}
		defer tx.Rollback(ctx)

		_, err = orderStep.Run(ctx, tx, pipeline.StepInput{
			RunID:  "test-run-001",
			Logger: zap.NewNop(),
		})
		if err != nil {
			t.Fatalf("dwd.order_level build failed: %v", err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit order build: %v", err)
		}
	}()

	// Step 3: Build dwd.item_level
	itemStep := NewBuildDWDItemLevelStep()
	func() {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin tx for item build: %v", err)
		}
		defer tx.Rollback(ctx)

		_, err = itemStep.Run(ctx, tx, pipeline.StepInput{
			RunID:  "test-run-001",
			Logger: zap.NewNop(),
		})
		if err != nil {
			t.Fatalf("dwd.item_level build failed: %v", err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit item build: %v", err)
		}
	}()

	// Step 4: Build mart.metric_dimension_daily
	metricStep := NewBuildMetricDimensionDailyStep()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx for metric build: %v", err)
	}
	defer tx.Rollback(ctx)

	output, err := metricStep.Run(ctx, tx, pipeline.StepInput{
		RunID:  "test-run-001",
		Logger: zap.NewNop(),
	})
	if err != nil {
		t.Fatalf("metric build step failed: %v", err)
	}

	// Verify output row count
	// Test data: o1 has 2 items (s1+electronics, s2+furniture), o2 has 0 items
	// So:
	//   seller: 2 unique sellers × 7 metrics = 14 rows
	//   category: 2 unique categories × 7 metrics = 14 rows
	//   region: 1 unique state (SP) × 7 metrics = 7 rows
	//   Total = 35
	expectedTotal := int64(35)
	if output.OutputCount != expectedTotal {
		t.Errorf("expected output_count %d, got %d", expectedTotal, output.OutputCount)
	}

	// Count rows in target table
	var martCount int64
	if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM mart.metric_dimension_daily").Scan(&martCount); err != nil {
		t.Fatalf("count mart.metric_dimension_daily: %v", err)
	}
	if martCount != expectedTotal {
		t.Errorf("mart.metric_dimension_daily: expected %d rows, got %d", expectedTotal, martCount)
	}

	// Verify dimension types are all present
	dimQuery := `SELECT dimension_type, COUNT(*) FROM mart.metric_dimension_daily GROUP BY dimension_type ORDER BY dimension_type`
	type dimCount struct {
		dimType string
		count   int64
	}
	rows, err := tx.Query(ctx, dimQuery)
	if err != nil {
		t.Fatalf("query dimension counts: %v", err)
	}
	defer rows.Close()

	var dimCounts []dimCount
	for rows.Next() {
		var dc dimCount
		if err := rows.Scan(&dc.dimType, &dc.count); err != nil {
			t.Fatalf("scan dimension count: %v", err)
		}
		dimCounts = append(dimCounts, dc)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows iteration: %v", err)
	}

	if len(dimCounts) != 3 {
		t.Fatalf("expected 3 dimension types, got %d: %+v", len(dimCounts), dimCounts)
	}

	// Each dimension should have 14 rows (2 unique values × 7 metrics each)
	// except region which has 7 rows (1 state × 7 metrics)
	for _, dc := range dimCounts {
		switch dc.dimType {
		case "seller", "category":
			if dc.count != 14 {
				t.Errorf("dimension %s: expected 14 rows, got %d", dc.dimType, dc.count)
			}
		case "region":
			if dc.count != 7 {
				t.Errorf("dimension %s: expected 7 rows, got %d", dc.dimType, dc.count)
			}
		default:
			t.Errorf("unexpected dimension type: %s", dc.dimType)
		}
	}

	// Verify metric names are all present
	metricQuery := `SELECT DISTINCT metric_name FROM mart.metric_dimension_daily ORDER BY metric_name`
	mRows, err := tx.Query(ctx, metricQuery)
	if err != nil {
		t.Fatalf("query metric names: %v", err)
	}
	defer mRows.Close()

	expectedMetrics := []string{"avg_order_value", "avg_review_score", "cancel_rate", "customer_count", "gmv", "late_delivery_rate", "order_count"}
	var actualMetrics []string
	for mRows.Next() {
		var m string
		if err := mRows.Scan(&m); err != nil {
			t.Fatalf("scan metric name: %v", err)
		}
		actualMetrics = append(actualMetrics, m)
	}
	if err := mRows.Err(); err != nil {
		t.Fatalf("rows iteration: %v", err)
	}

	if len(actualMetrics) != len(expectedMetrics) {
		t.Errorf("expected %d metrics, got %d: %v", len(expectedMetrics), len(actualMetrics), actualMetrics)
	}
	for i, m := range expectedMetrics {
		if i < len(actualMetrics) && actualMetrics[i] != m {
			t.Errorf("metric[%d]: expected %q, got %q", i, m, actualMetrics[i])
		}
	}

	// Verify sample gmv value for seller s1 on 2017-01-01
	var gmvVal float64
	var sampleSize int64
	err = tx.QueryRow(ctx, `
		SELECT metric_value, sample_size FROM mart.metric_dimension_daily
		WHERE metric_date = '2017-01-01'
		  AND dimension_type = 'seller'
		  AND dimension_value = 's1'
		  AND metric_name = 'gmv'
	`).Scan(&gmvVal, &sampleSize)
	if err != nil {
		t.Fatalf("query seller s1 gmv: %v", err)
	}
	if gmvVal != 100.00 {
		t.Errorf("seller s1 gmv: expected 100.00, got %f", gmvVal)
	}
	if sampleSize != 1 {
		t.Errorf("seller s1 sample_size: expected 1, got %d", sampleSize)
	}

	// Verify gmv for category electronics on 2017-01-01
	err = tx.QueryRow(ctx, `
		SELECT metric_value FROM mart.metric_dimension_daily
		WHERE metric_date = '2017-01-01'
		  AND dimension_type = 'category'
		  AND dimension_value = 'electronics'
		  AND metric_name = 'gmv'
	`).Scan(&gmvVal)
	if err != nil {
		t.Fatalf("query category electronics gmv: %v", err)
	}
	if gmvVal != 100.00 {
		t.Errorf("category electronics gmv: expected 100.00, got %f", gmvVal)
	}

	// Verify gmv for region SP on 2017-01-01 (both items = 150)
	err = tx.QueryRow(ctx, `
		SELECT metric_value FROM mart.metric_dimension_daily
		WHERE metric_date = '2017-01-01'
		  AND dimension_type = 'region'
		  AND dimension_value = 'SP'
		  AND metric_name = 'gmv'
	`).Scan(&gmvVal)
	if err != nil {
		t.Fatalf("query region SP gmv: %v", err)
	}
	if gmvVal != 150.00 {
		t.Errorf("region SP gmv: expected 150.00, got %f", gmvVal)
	}

	// Verify avg_review_score for seller s1
	var reviewScore float64
	err = tx.QueryRow(ctx, `
		SELECT metric_value FROM mart.metric_dimension_daily
		WHERE metric_date = '2017-01-01'
		  AND dimension_type = 'seller'
		  AND dimension_value = 's1'
		  AND metric_name = 'avg_review_score'
	`).Scan(&reviewScore)
	if err != nil {
		t.Fatalf("query seller s1 avg_review_score: %v", err)
	}
	if reviewScore != 4.0 {
		t.Errorf("seller s1 avg_review_score: expected 4.0, got %f", reviewScore)
	}

	// Verify cancel_rate for seller s1 is 0 (order delivered, not cancelled)
	var cancelRate float64
	err = tx.QueryRow(ctx, `
		SELECT metric_value FROM mart.metric_dimension_daily
		WHERE metric_date = '2017-01-01'
		  AND dimension_type = 'seller'
		  AND dimension_value = 's1'
		  AND metric_name = 'cancel_rate'
	`).Scan(&cancelRate)
	if err != nil {
		t.Fatalf("query seller s1 cancel_rate: %v", err)
	}
	if cancelRate != 0.0 {
		t.Errorf("seller s1 cancel_rate: expected 0.0, got %f", cancelRate)
	}
}

// ---------------------------------------------------------------------------
// Metric Daily Tests
// ---------------------------------------------------------------------------

const metricDailyDDL = `
CREATE SCHEMA IF NOT EXISTS mart;

CREATE TABLE IF NOT EXISTS mart.metric_daily (
    metric_date              DATE PRIMARY KEY,
    gmv                      NUMERIC(18,2),
    order_count              BIGINT,
    customer_count           BIGINT,
    seller_count             BIGINT,
    avg_order_value          NUMERIC(18,2),
    freight_value            NUMERIC(18,2),
    avg_review_score         NUMERIC(4,2),
    low_review_rate          NUMERIC(10,6),
    late_delivery_rate       NUMERIC(10,6),
    cancel_rate              NUMERIC(10,6),
    payment_installment_rate NUMERIC(10,6),
    marketing_seller_share   NUMERIC(10,6),
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`

// setupBuildMetricDailyTestDB creates raw, dwd, and mart.metric_daily tables.
func setupBuildMetricDailyTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	t.Cleanup(pool.Close)

	ctx := context.Background()

	// Create raw tables
	if _, err := pool.Exec(ctx, rawTableDDL); err != nil {
		t.Fatalf("create raw tables: %v", err)
	}

	// Create dwd tables
	if _, err := pool.Exec(ctx, dwdOrderLevelDDL); err != nil {
		t.Fatalf("create dwd.order_level: %v", err)
	}
	if _, err := pool.Exec(ctx, dwdItemLevelDDL); err != nil {
		t.Fatalf("create dwd.item_level: %v", err)
	}

	// Create mart table
	if _, err := pool.Exec(ctx, metricDailyDDL); err != nil {
		t.Fatalf("create mart.metric_daily: %v", err)
	}

	// Clean data from previous runs
	for _, mapping := range ingest.AllTableMappings() {
		_, _ = pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s", mapping.TableName))
	}
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE dwd.order_level")
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE dwd.item_level")
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE mart.metric_daily")

	return pool
}

func TestBuildMetricDaily_Name(t *testing.T) {
	step := NewBuildMetricDailyStep()
	if got := step.Name(); got != "build_metric_daily" {
		t.Errorf("expected name 'build_metric_daily', got %q", got)
	}
}

func TestBuildMetricDaily_HappyPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupBuildMetricDailyTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	dataDir := t.TempDir()
	writeTestCSVs(t, dataDir)

	// Step 1: Load raw tables
	ingestStep := NewIngestRawStep()
	func() {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin tx for ingest: %v", err)
		}
		defer tx.Rollback(ctx)

		_, err = ingestStep.Run(ctx, tx, pipeline.StepInput{
			DataDir: dataDir,
			Logger:  zap.NewNop(),
		})
		if err != nil {
			t.Fatalf("ingest step failed: %v", err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit ingest: %v", err)
		}
	}()

	// Step 2: Build dwd.order_level
	orderStep := NewBuildDWDSOrderLevelStep()
	func() {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin tx for order build: %v", err)
		}
		defer tx.Rollback(ctx)

		_, err = orderStep.Run(ctx, tx, pipeline.StepInput{
			RunID:  "test-run-001",
			Logger: zap.NewNop(),
		})
		if err != nil {
			t.Fatalf("dwd.order_level build failed: %v", err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit order build: %v", err)
		}
	}()

	// Step 3: Build dwd.item_level
	itemStep := NewBuildDWDItemLevelStep()
	func() {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin tx for item build: %v", err)
		}
		defer tx.Rollback(ctx)

		_, err = itemStep.Run(ctx, tx, pipeline.StepInput{
			RunID:  "test-run-001",
			Logger: zap.NewNop(),
		})
		if err != nil {
			t.Fatalf("dwd.item_level build failed: %v", err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit item build: %v", err)
		}
	}()

	// Step 4: Build mart.metric_daily
	metricStep := NewBuildMetricDailyStep()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx for metric build: %v", err)
	}
	defer tx.Rollback(ctx)

	output, err := metricStep.Run(ctx, tx, pipeline.StepInput{
		RunID:  "test-run-001",
		Logger: zap.NewNop(),
	})
	if err != nil {
		t.Fatalf("metric daily build failed: %v", err)
	}

	// Verify input count (distinct purchase dates)
	if output.InputCount != 2 {
		t.Errorf("expected input_count 2, got %d", output.InputCount)
	}

	// Verify output count (one row per date)
	if output.OutputCount != 2 {
		t.Errorf("expected output_count 2, got %d", output.OutputCount)
	}

	var martCount int64
	if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM mart.metric_daily").Scan(&martCount); err != nil {
		t.Fatalf("count mart.metric_daily: %v", err)
	}
	if martCount != 2 {
		t.Errorf("expected 2 rows in mart.metric_daily, got %d", martCount)
	}

	// Verify sample values for 2017-01-01 (o1: delivered, payment=100, 2 items)
	type metricRow struct {
		gmv                   float64
		orderCount            int64
		customerCount         int64
		sellerCount           int64
		avgOrderValue         float64
		freightValue          float64
		avgReviewScore        float64
		lowReviewRate         float64
		lateDeliveryRate      float64
		cancelRate            float64
		paymentInstallmentRate float64
		marketingSellerShare  float64
	}

	var day1 metricRow
	err = tx.QueryRow(ctx, `
		SELECT gmv::TEXT, order_count, customer_count, seller_count,
		       avg_order_value::TEXT, freight_value::TEXT, avg_review_score::TEXT,
		       low_review_rate::TEXT, late_delivery_rate::TEXT, cancel_rate::TEXT,
		       payment_installment_rate::TEXT, marketing_seller_share::TEXT
		FROM mart.metric_daily WHERE metric_date = '2017-01-01'
	`).Scan(
		&day1.gmv, &day1.orderCount, &day1.customerCount, &day1.sellerCount,
		&day1.avgOrderValue, &day1.freightValue, &day1.avgReviewScore,
		&day1.lowReviewRate, &day1.lateDeliveryRate, &day1.cancelRate,
		&day1.paymentInstallmentRate, &day1.marketingSellerShare,
	)
	if err != nil {
		t.Fatalf("query 2017-01-01: %v", err)
	}

	// o1: delivered, payment=100, 2 items (s1+100.00, s2+50.00), freight=15, review=4
	if day1.gmv != 100.00 {
		t.Errorf("gmv: expected 100.00, got %f", day1.gmv)
	}
	if day1.orderCount != 1 {
		t.Errorf("order_count: expected 1, got %d", day1.orderCount)
	}
	if day1.customerCount != 1 {
		t.Errorf("customer_count: expected 1, got %d", day1.customerCount)
	}
	if day1.sellerCount != 2 {
		t.Errorf("seller_count: expected 2, got %d", day1.sellerCount)
	}
	if day1.avgOrderValue != 100.00 {
		t.Errorf("avg_order_value: expected 100.00, got %f", day1.avgOrderValue)
	}
	if day1.freightValue != 15.00 {
		t.Errorf("freight_value: expected 15.00, got %f", day1.freightValue)
	}
	if day1.avgReviewScore != 4.00 {
		t.Errorf("avg_review_score: expected 4.00, got %f", day1.avgReviewScore)
	}
	if day1.lowReviewRate != 0.00 {
		t.Errorf("low_review_rate: expected 0.00, got %f", day1.lowReviewRate)
	}
	if day1.lateDeliveryRate != 0.00 {
		t.Errorf("late_delivery_rate: expected 0.00, got %f", day1.lateDeliveryRate)
	}
	if day1.cancelRate != 0.00 {
		t.Errorf("cancel_rate: expected 0.00, got %f", day1.cancelRate)
	}
	if day1.paymentInstallmentRate != 0.00 {
		t.Errorf("payment_installment_rate: expected 0.00, got %f", day1.paymentInstallmentRate)
	}
	if day1.marketingSellerShare != 0.00 {
		t.Errorf("marketing_seller_share: expected 0.00, got %f", day1.marketingSellerShare)
	}

	// Verify sample values for 2017-02-01 (o2: shipped, payment=200, no items)
	var day2 metricRow
	err = tx.QueryRow(ctx, `
		SELECT gmv::TEXT, order_count, customer_count, seller_count,
		       avg_order_value::TEXT, freight_value::TEXT, avg_review_score::TEXT,
		       low_review_rate::TEXT, late_delivery_rate::TEXT, cancel_rate::TEXT,
		       payment_installment_rate::TEXT, marketing_seller_share::TEXT
		FROM mart.metric_daily WHERE metric_date = '2017-02-01'
	`).Scan(
		&day2.gmv, &day2.orderCount, &day2.customerCount, &day2.sellerCount,
		&day2.avgOrderValue, &day2.freightValue, &day2.avgReviewScore,
		&day2.lowReviewRate, &day2.lateDeliveryRate, &day2.cancelRate,
		&day2.paymentInstallmentRate, &day2.marketingSellerShare,
	)
	if err != nil {
		t.Fatalf("query 2017-02-01: %v", err)
	}

	if day2.gmv != 200.00 {
		t.Errorf("gmv: expected 200.00, got %f", day2.gmv)
	}
	if day2.orderCount != 1 {
		t.Errorf("order_count: expected 1, got %d", day2.orderCount)
	}
	if day2.customerCount != 1 {
		t.Errorf("customer_count: expected 1, got %d", day2.customerCount)
	}
	if day2.sellerCount != 0 {
		t.Errorf("seller_count: expected 0, got %d", day2.sellerCount)
	}
	if day2.avgOrderValue != 200.00 {
		t.Errorf("avg_order_value: expected 200.00, got %f", day2.avgOrderValue)
	}
	if day2.freightValue != 0.00 {
		t.Errorf("freight_value: expected 0.00, got %f", day2.freightValue)
	}
	if day2.avgReviewScore != 3.00 {
		t.Errorf("avg_review_score: expected 3.00, got %f", day2.avgReviewScore)
	}
	if day2.lowReviewRate != 0.00 {
		t.Errorf("low_review_rate: expected 0.00, got %f", day2.lowReviewRate)
	}
	if day2.lateDeliveryRate != 0.00 {
		t.Errorf("late_delivery_rate: expected 0.00, got %f", day2.lateDeliveryRate)
	}
	if day2.cancelRate != 0.00 {
		t.Errorf("cancel_rate: expected 0.00, got %f", day2.cancelRate)
	}
	if day2.paymentInstallmentRate != 0.00 {
		t.Errorf("payment_installment_rate: expected 0.00, got %f", day2.paymentInstallmentRate)
	}
	if day2.marketingSellerShare != 0.00 {
		t.Errorf("marketing_seller_share: expected 0.00, got %f", day2.marketingSellerShare)
	}
}

func TestBuildMetricDaily_Idempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupBuildMetricDailyTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	dataDir := t.TempDir()
	writeTestCSVs(t, dataDir)

	// Load raw + build DWD tables (same setup as HappyPath)
	ingestStep := NewIngestRawStep()
	func() {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin tx for ingest: %v", err)
		}
		defer tx.Rollback(ctx)
		_, err = ingestStep.Run(ctx, tx, pipeline.StepInput{DataDir: dataDir, Logger: zap.NewNop()})
		if err != nil {
			t.Fatalf("ingest failed: %v", err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit ingest: %v", err)
		}
	}()

	orderStep := NewBuildDWDSOrderLevelStep()
	func() {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin tx for order build: %v", err)
		}
		defer tx.Rollback(ctx)
		_, err = orderStep.Run(ctx, tx, pipeline.StepInput{RunID: "test-ido", Logger: zap.NewNop()})
		if err != nil {
			t.Fatalf("order build failed: %v", err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit order build: %v", err)
		}
	}()

	itemStep := NewBuildDWDItemLevelStep()
	func() {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin tx for item build: %v", err)
		}
		defer tx.Rollback(ctx)
		_, err = itemStep.Run(ctx, tx, pipeline.StepInput{RunID: "test-ido", Logger: zap.NewNop()})
		if err != nil {
			t.Fatalf("item build failed: %v", err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit item build: %v", err)
		}
	}()

	metricStep := NewBuildMetricDailyStep()

	// First run — should produce 2 rows
	func() {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin tx for first run: %v", err)
		}
		defer tx.Rollback(ctx)

		output, err := metricStep.Run(ctx, tx, pipeline.StepInput{
			RunID:  "test-ido-1",
			Logger: zap.NewNop(),
		})
		if err != nil {
			t.Fatalf("first run failed: %v", err)
		}
		if output.OutputCount != 2 {
			t.Errorf("first run: expected 2 rows, got %d", output.OutputCount)
		}

		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit first run: %v", err)
		}
	}()

	// Second run — TRUNCATE + INSERT should produce same 2 rows
	func() {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin tx for second run: %v", err)
		}
		defer tx.Rollback(ctx)

		output, err := metricStep.Run(ctx, tx, pipeline.StepInput{
			RunID:  "test-ido-2",
			Logger: zap.NewNop(),
		})
		if err != nil {
			t.Fatalf("second run failed: %v", err)
		}

		if output.OutputCount != 2 {
			t.Errorf("second run: expected 2 rows, got %d", output.OutputCount)
		}

		// Total should still be 2 (TRUNCATE replaces all data)
		var total int64
		if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM mart.metric_daily").Scan(&total); err != nil {
			t.Fatalf("count after re-run: %v", err)
		}
		if total != 2 {
			t.Errorf("idempotent re-run: expected 2 total rows, got %d", total)
		}
	}()
}

func TestBuildMetricDim_Idempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupBuildMetricDimTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	dataDir := t.TempDir()
	writeTestCSVs(t, dataDir)

	// Load raw + build DWD tables (same setup as HappyPath)
	ingestStep := NewIngestRawStep()
	func() {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin tx for ingest: %v", err)
		}
		defer tx.Rollback(ctx)
		_, err = ingestStep.Run(ctx, tx, pipeline.StepInput{DataDir: dataDir, Logger: zap.NewNop()})
		if err != nil {
			t.Fatalf("ingest failed: %v", err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit ingest: %v", err)
		}
	}()

	orderStep := NewBuildDWDSOrderLevelStep()
	func() {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin tx for order build: %v", err)
		}
		defer tx.Rollback(ctx)
		_, err = orderStep.Run(ctx, tx, pipeline.StepInput{RunID: "test-ido", Logger: zap.NewNop()})
		if err != nil {
			t.Fatalf("order build failed: %v", err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit order build: %v", err)
		}
	}()

	itemStep := NewBuildDWDItemLevelStep()
	func() {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin tx for item build: %v", err)
		}
		defer tx.Rollback(ctx)
		_, err = itemStep.Run(ctx, tx, pipeline.StepInput{RunID: "test-ido", Logger: zap.NewNop()})
		if err != nil {
			t.Fatalf("item build failed: %v", err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit item build: %v", err)
		}
	}()

	metricStep := NewBuildMetricDimensionDailyStep()

	// First run
	func() {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin tx for first run: %v", err)
		}
		defer tx.Rollback(ctx)

		output, err := metricStep.Run(ctx, tx, pipeline.StepInput{
			RunID:  "test-ido-1",
			Logger: zap.NewNop(),
		})
		if err != nil {
			t.Fatalf("first run failed: %v", err)
		}
		if output.OutputCount != 35 {
			t.Errorf("first run: expected 35 rows, got %d", output.OutputCount)
		}

		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit first run: %v", err)
		}
	}()

	// Second run — ON CONFLICT DO UPDATE should upsert same 35 rows
	func() {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin tx for second run: %v", err)
		}
		defer tx.Rollback(ctx)

		output, err := metricStep.Run(ctx, tx, pipeline.StepInput{
			RunID:  "test-ido-2",
			Logger: zap.NewNop(),
		})
		if err != nil {
			t.Fatalf("second run failed: %v", err)
		}

		// Second run should upsert (ON CONFLICT DO UPDATE means RowsAffected
		// reports rows that triggered the insert OR update)
		// For idempotency, the important check is that total rows don't change.
		if output.OutputCount != 35 {
			t.Errorf("second run: expected 35 rows upserted, got %d", output.OutputCount)
		}

		var total int64
		if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM mart.metric_dimension_daily").Scan(&total); err != nil {
			t.Fatalf("count after re-run: %v", err)
		}
		if total != 35 {
			t.Errorf("idempotent re-run: expected 35 total rows, got %d", total)
		}
	}()
}
