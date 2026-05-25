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
// Test setup helpers
// ---------------------------------------------------------------------------

// DDL for dwd.order_level table. Mirrors migrations/003_dwd_tables.sql.
const dwdOrderLevelDDL = `
CREATE SCHEMA IF NOT EXISTS dwd;

CREATE TABLE IF NOT EXISTS dwd.order_level (
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
`

// setupBuildDWDTestDB creates raw tables + dwd.order_level for testing.
func setupBuildDWDTestDB(t *testing.T) *pgxpool.Pool {
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

	// Create raw tables (reuses DDL from ingest_raw_test.go)
	if _, err := pool.Exec(ctx, rawTableDDL); err != nil {
		t.Fatalf("create raw tables: %v", err)
	}

	// Create dwd.order_level table
	if _, err := pool.Exec(ctx, dwdOrderLevelDDL); err != nil {
		t.Fatalf("create dwd.order_level: %v", err)
	}

	// Clean any leftover data from previous runs
	for _, mapping := range ingest.AllTableMappings() {
		_, _ = pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s", mapping.TableName))
	}
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE dwd.order_level")

	return pool
}

// DDL for dwd.item_level table. Mirrors migrations/003_dwd_tables.sql.
const dwdItemLevelDDL = `
CREATE TABLE IF NOT EXISTS dwd.item_level (
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
`

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestBuildDWDOrder_Name(t *testing.T) {
	step := NewBuildDWDSOrderLevelStep()
	if got := step.Name(); got != "build_dwd_order_level" {
		t.Errorf("expected name 'build_dwd_order_level', got %q", got)
	}
}

func TestBuildDWDOrder_HappyPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupBuildDWDTestDB(t)
	defer pool.Close()

	// Write test CSVs to a temp directory
	dataDir := t.TempDir()
	writeTestCSVs(t, dataDir)

	// Step 1: Load raw tables using IngestRawStep
	ingestStep := NewIngestRawStep()
	func() {
		tx, err := pool.Begin(context.Background())
		if err != nil {
			t.Fatalf("begin tx for ingest: %v", err)
		}
		defer tx.Rollback(context.Background())

		_, err = ingestStep.Run(context.Background(), tx, pipeline.StepInput{
			DataDir: dataDir,
			Logger:  zap.NewNop(),
		})
		if err != nil {
			t.Fatalf("ingest step failed: %v", err)
		}

		if err := tx.Commit(context.Background()); err != nil {
			t.Fatalf("commit ingest: %v", err)
		}
	}()

	// Step 2: Build dwd.order_level
	buildStep := NewBuildDWDSOrderLevelStep()
	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("begin tx for build: %v", err)
	}
	defer tx.Rollback(context.Background())

	output, err := buildStep.Run(context.Background(), tx, pipeline.StepInput{
		RunID:   "test-run-001",
		DataDir: dataDir,
		Logger:  zap.NewNop(),
	})
	if err != nil {
		t.Fatalf("build step failed: %v", err)
	}

	// Verify input count matches number of orders in test data
	if output.InputCount != 2 {
		t.Errorf("expected input_count 2, got %d", output.InputCount)
	}
	// Verify output count matches expected DWD rows (one per order)
	if output.OutputCount != 2 {
		t.Errorf("expected output_count 2, got %d", output.OutputCount)
	}

	// Verify per-table row count
	var dwdCount int64
	if err := tx.QueryRow(context.Background(), "SELECT COUNT(*) FROM dwd.order_level").Scan(&dwdCount); err != nil {
		t.Fatalf("count dwd.order_level: %v", err)
	}
	if dwdCount != 2 {
		t.Errorf("expected 2 rows in dwd.order_level, got %d", dwdCount)
	}

	// Verify sample values for order o1 (delivered, all dates present)
	type orderRow struct {
		orderID        string
		customerID     string
		customerUnique string
		orderStatus    string
		customerState  string
		paymentType    string
		paymentInstall int64
		paymentValue   float64
		reviewScore    float64
		deliveryDays   *float64
		delayDays      *float64
		isLate         *bool
		isCancelled    bool
	}

	var o1 orderRow
	err = tx.QueryRow(context.Background(), `
		SELECT order_id, customer_id, customer_unique_id, order_status,
		       customer_state, payment_type, payment_installments, payment_value,
		       review_score, delivery_days, delay_days, is_late, is_cancelled
		FROM dwd.order_level WHERE order_id = 'o1'
	`).Scan(
		&o1.orderID, &o1.customerID, &o1.customerUnique, &o1.orderStatus,
		&o1.customerState, &o1.paymentType, &o1.paymentInstall, &o1.paymentValue,
		&o1.reviewScore, &o1.deliveryDays, &o1.delayDays, &o1.isLate, &o1.isCancelled,
	)
	if err != nil {
		t.Fatalf("query order o1: %v", err)
	}

	if o1.orderID != "o1" {
		t.Errorf("o1 order_id: expected 'o1', got %q", o1.orderID)
	}
	if o1.customerID != "c1" {
		t.Errorf("o1 customer_id: expected 'c1', got %q", o1.customerID)
	}
	if o1.customerUnique != "u1" {
		t.Errorf("o1 customer_unique_id: expected 'u1', got %q", o1.customerUnique)
	}
	if o1.orderStatus != "delivered" {
		t.Errorf("o1 order_status: expected 'delivered', got %q", o1.orderStatus)
	}
	if o1.customerState != "SP" {
		t.Errorf("o1 customer_state: expected 'SP', got %q", o1.customerState)
	}
	if o1.paymentType != "credit_card" {
		t.Errorf("o1 payment_type: expected 'credit_card', got %q", o1.paymentType)
	}
	if o1.paymentInstall != 1 {
		t.Errorf("o1 payment_installments: expected 1, got %d", o1.paymentInstall)
	}
	if o1.paymentValue != 100.00 {
		t.Errorf("o1 payment_value: expected 100.00, got %f", o1.paymentValue)
	}
	if o1.reviewScore != 4 {
		t.Errorf("o1 review_score: expected 4, got %f", o1.reviewScore)
	}
	if o1.deliveryDays == nil {
		t.Error("o1 delivery_days: expected non-nil")
	} else if *o1.deliveryDays <= 0 {
		t.Errorf("o1 delivery_days: expected positive, got %f", *o1.deliveryDays)
	}
	if o1.delayDays == nil {
		t.Error("o1 delay_days: expected non-nil")
	} else if *o1.delayDays > 0 {
		t.Errorf("o1 delay_days: expected <= 0 (delivered before estimate), got %f", *o1.delayDays)
	}
	if o1.isLate == nil {
		t.Error("o1 is_late: expected non-nil")
	} else if *o1.isLate {
		t.Error("o1 is_late: expected false (delivered before estimate)")
	}
	if o1.isCancelled {
		t.Error("o1 is_cancelled: expected false")
	}

	// Verify sample values for order o2 (shipped, no delivery dates)
	var o2 orderRow
	err = tx.QueryRow(context.Background(), `
		SELECT order_id, customer_id, customer_unique_id, order_status,
		       customer_state, payment_type, payment_installments, payment_value,
		       review_score, delivery_days, delay_days, is_late, is_cancelled
		FROM dwd.order_level WHERE order_id = 'o2'
	`).Scan(
		&o2.orderID, &o2.customerID, &o2.customerUnique, &o2.orderStatus,
		&o2.customerState, &o2.paymentType, &o2.paymentInstall, &o2.paymentValue,
		&o2.reviewScore, &o2.deliveryDays, &o2.delayDays, &o2.isLate, &o2.isCancelled,
	)
	if err != nil {
		t.Fatalf("query order o2: %v", err)
	}

	if o2.orderID != "o2" {
		t.Errorf("o2 order_id: expected 'o2', got %q", o2.orderID)
	}
	if o2.orderStatus != "shipped" {
		t.Errorf("o2 order_status: expected 'shipped', got %q", o2.orderStatus)
	}
	if o2.deliveryDays != nil {
		t.Errorf("o2 delivery_days: expected NULL, got %f", *o2.deliveryDays)
	}
	if o2.delayDays != nil {
		t.Errorf("o2 delay_days: expected NULL, got %f", *o2.delayDays)
	}
	if o2.isLate != nil {
		t.Errorf("o2 is_late: expected NULL, got %v", *o2.isLate)
	}
	if o2.isCancelled {
		t.Error("o2 is_cancelled: expected false")
	}
}

func TestBuildDWDOrder_IdempotentReRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupBuildDWDTestDB(t)
	defer pool.Close()

	dataDir := t.TempDir()
	writeTestCSVs(t, dataDir)

	// Load raw tables (same pattern as HappyPath)
	ingestStep := NewIngestRawStep()
	func() {
		tx, err := pool.Begin(context.Background())
		if err != nil {
			t.Fatalf("begin tx for ingest: %v", err)
		}
		defer tx.Rollback(context.Background())

		_, err = ingestStep.Run(context.Background(), tx, pipeline.StepInput{
			DataDir: dataDir,
			Logger:  zap.NewNop(),
		})
		if err != nil {
			t.Fatalf("ingest step failed: %v", err)
		}
		if err := tx.Commit(context.Background()); err != nil {
			t.Fatalf("commit ingest: %v", err)
		}
	}()

	buildStep := NewBuildDWDSOrderLevelStep()
	runID := "test-idempotent-run"

	// First run
	func() {
		tx, err := pool.Begin(context.Background())
		if err != nil {
			t.Fatalf("begin tx for first build: %v", err)
		}
		defer tx.Rollback(context.Background())

		output, err := buildStep.Run(context.Background(), tx, pipeline.StepInput{
			RunID:   runID,
			DataDir: dataDir,
			Logger:  zap.NewNop(),
		})
		if err != nil {
			t.Fatalf("first build failed: %v", err)
		}
		if output.OutputCount != 2 {
			t.Errorf("first run: expected 2 rows, got %d", output.OutputCount)
		}
		if err := tx.Commit(context.Background()); err != nil {
			t.Fatalf("commit first build: %v", err)
		}
	}()

	// Second run — ON CONFLICT DO NOTHING should prevent duplicates
	func() {
		tx, err := pool.Begin(context.Background())
		if err != nil {
			t.Fatalf("begin tx for second build: %v", err)
		}
		defer tx.Rollback(context.Background())

		output, err := buildStep.Run(context.Background(), tx, pipeline.StepInput{
			RunID:   runID,
			DataDir: dataDir,
			Logger:  zap.NewNop(),
		})
		if err != nil {
			t.Fatalf("second build failed: %v", err)
		}
		if output.OutputCount != 0 {
			t.Errorf("second run (idempotent): expected 0 new rows, got %d", output.OutputCount)
		}

		// Total should still be 2
		var total int64
		if err := tx.QueryRow(context.Background(), "SELECT COUNT(*) FROM dwd.order_level").Scan(&total); err != nil {
			t.Fatalf("count dwd.order_level: %v", err)
		}
		if total != 2 {
			t.Errorf("idempotent re-run: expected 2 total rows, got %d", total)
		}
	}()
}

// ---------------------------------------------------------------------------
// Item Level Tests
// ---------------------------------------------------------------------------

func TestBuildDWDItemLevel_Name(t *testing.T) {
	step := NewBuildDWDItemLevelStep()
	if got := step.Name(); got != "build_dwd_item_level" {
		t.Errorf("expected name 'build_dwd_item_level', got %q", got)
	}
}

func TestBuildDWDItemLevel_HappyPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	if _, err := pool.Exec(ctx, dwdItemLevelDDL); err != nil {
		t.Fatalf("create dwd.item_level: %v", err)
	}

	dataDir := t.TempDir()
	writeTestCSVs(t, dataDir)

	ingestStep := NewIngestRawStep()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx for ingest: %v", err)
	}
	_, err = ingestStep.Run(ctx, tx, pipeline.StepInput{
		DataDir: dataDir,
		Logger:  zap.NewNop(),
	})
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("ingest raw step failed: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit ingest: %v", err)
	}

	step := NewBuildDWDItemLevelStep()
	tx2, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx for build_dwd: %v", err)
	}
	defer tx2.Rollback(ctx)

	runID := "test-run-001"
	output, err := step.Run(ctx, tx2, pipeline.StepInput{
		RunID:   runID,
		DataDir: dataDir,
		Logger:  zap.NewNop(),
	})
	if err != nil {
		t.Fatalf("build_dwd_item_level step failed: %v", err)
	}

	if output.InputCount != 2 {
		t.Errorf("expected input_count 2, got %d", output.InputCount)
	}
	if output.OutputCount != 2 {
		t.Errorf("expected output_count 2, got %d", output.OutputCount)
	}

	var totalRows int64
	if err := tx2.QueryRow(ctx, "SELECT COUNT(*) FROM dwd.item_level").Scan(&totalRows); err != nil {
		t.Fatalf("count dwd.item_level: %v", err)
	}
	if totalRows != 2 {
		t.Errorf("dwd.item_level: expected 2 rows, got %d", totalRows)
	}

	rows, err := tx2.Query(ctx, `
		SELECT order_id, product_id, seller_id,
		       product_category_name, product_category_name_english, seller_state,
		       price::TEXT, freight_value::TEXT,
		       ingestion_batch_id, pipeline_run_id
		FROM dwd.item_level
		ORDER BY order_id, order_item_id
	`)
	if err != nil {
		t.Fatalf("query dwd.item_level: %v", err)
	}
	defer rows.Close()

	type itemRow struct {
		orderID, productID, sellerID      string
		catName, catNameEng, sellerState string
		price, freightValue               string
		batchID, pipelineRunID            string
	}

	var results []itemRow
	for rows.Next() {
		var r itemRow
		if err := rows.Scan(&r.orderID, &r.productID, &r.sellerID,
			&r.catName, &r.catNameEng, &r.sellerState,
			&r.price, &r.freightValue,
			&r.batchID, &r.pipelineRunID); err != nil {
			t.Fatalf("scan row: %v", err)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows iteration: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 result rows, got %d", len(results))
	}

	r1 := results[0]
	if r1.orderID != "o1" || r1.productID != "p1" || r1.sellerID != "s1" {
		t.Errorf("row1 ids: order=%s product=%s seller=%s", r1.orderID, r1.productID, r1.sellerID)
	}
	if r1.catName != "electronics" || r1.catNameEng != "electronics" {
		t.Errorf("row1 categories: %s / %s", r1.catName, r1.catNameEng)
	}
	if r1.sellerState != "SP" {
		t.Errorf("row1 seller_state: expected SP, got %s", r1.sellerState)
	}
	if r1.price != "100.00" || r1.freightValue != "10.00" {
		t.Errorf("row1 values: price=%s freight=%s", r1.price, r1.freightValue)
	}
	if r1.batchID != runID || r1.pipelineRunID != runID {
		t.Errorf("row1 ids: batch=%s pipeline=%s", r1.batchID, r1.pipelineRunID)
	}

	r2 := results[1]
	if r2.orderID != "o1" || r2.productID != "p2" || r2.sellerID != "s2" {
		t.Errorf("row2 ids: order=%s product=%s seller=%s", r2.orderID, r2.productID, r2.sellerID)
	}
	if r2.catName != "furniture" || r2.catNameEng != "furniture" {
		t.Errorf("row2 categories: %s / %s", r2.catName, r2.catNameEng)
	}
	if r2.sellerState != "RJ" {
		t.Errorf("row2 seller_state: expected RJ, got %s", r2.sellerState)
	}
	if r2.price != "50.00" || r2.freightValue != "5.00" {
		t.Errorf("row2 values: price=%s freight=%s", r2.price, r2.freightValue)
	}
	if r2.batchID != runID || r2.pipelineRunID != runID {
		t.Errorf("row2 ids: batch=%s pipeline=%s", r2.batchID, r2.pipelineRunID)
	}
}

func TestBuildDWDItemLevel_IdempotentReRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	if _, err := pool.Exec(ctx, dwdItemLevelDDL); err != nil {
		t.Fatalf("create dwd.item_level: %v", err)
	}

	dataDir := t.TempDir()
	writeTestCSVs(t, dataDir)

	ingestStep := NewIngestRawStep()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx for ingest: %v", err)
	}
	_, err = ingestStep.Run(ctx, tx, pipeline.StepInput{
		DataDir: dataDir,
		Logger:  zap.NewNop(),
	})
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("ingest raw step failed: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit ingest: %v", err)
	}

	step := NewBuildDWDItemLevelStep()

	tx1, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx first run: %v", err)
	}
	output, err := step.Run(ctx, tx1, pipeline.StepInput{
		RunID:  "run-1",
		Logger: zap.NewNop(),
	})
	if err != nil {
		tx1.Rollback(ctx)
		t.Fatalf("first run failed: %v", err)
	}
	if output.OutputCount != 2 {
		t.Errorf("first run: expected 2 rows, got %d", output.OutputCount)
	}
	if err := tx1.Commit(ctx); err != nil {
		t.Fatalf("commit first run: %v", err)
	}

	tx2, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx second run: %v", err)
	}
	defer tx2.Rollback(ctx)

	_, err = step.Run(ctx, tx2, pipeline.StepInput{
		RunID:  "run-2",
		Logger: zap.NewNop(),
	})
	if err != nil {
		t.Fatalf("second run failed: %v", err)
	}

	var totalRows int64
	if err := tx2.QueryRow(ctx, "SELECT COUNT(*) FROM dwd.item_level").Scan(&totalRows); err != nil {
		t.Fatalf("count dwd.item_level: %v", err)
	}
	if totalRows != 2 {
		t.Errorf("after re-run: expected 2 rows, got %d", totalRows)
	}
}
