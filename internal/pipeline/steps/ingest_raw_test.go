package steps

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"baxi/internal/ingest"
	"baxi/internal/pipeline"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// Test CSV data — minimal rows matching real CSV headers.
// Each key is the CSV filename; value is the full CSV text (header + data).
var testCSVs = map[string]string{
	"olist_customers_dataset.csv": `"customer_id","customer_unique_id","customer_zip_code_prefix","customer_city","customer_state"
"c1","u1","01001","sao paulo","SP"
"c2","u2","02001","rio de janeiro","RJ"
`,
	"olist_orders_dataset.csv": `"order_id","customer_id","order_status","order_purchase_timestamp","order_approved_at","order_delivered_carrier_date","order_delivered_customer_date","order_estimated_delivery_date"
"o1","c1","delivered","2017-01-01 10:00:00","2017-01-01 11:00:00","2017-01-05 10:00:00","2017-01-10 10:00:00","2017-01-15"
"o2","c1","shipped","2017-02-01 10:00:00","2017-02-01 11:00:00","2017-02-05 10:00:00",,
`,
	"olist_order_items_dataset.csv": `"order_id","order_item_id","product_id","seller_id","shipping_limit_date","price","freight_value"
"o1","1","p1","s1","2017-01-20 10:00:00","100.00","10.00"
"o1","2","p2","s2","2017-01-20 10:00:00","50.00","5.00"
`,
	"olist_order_payments_dataset.csv": `"order_id","payment_sequential","payment_type","payment_installments","payment_value"
"o1","1","credit_card","1","100.00"
"o2","1","boleto","1","200.00"
`,
	"olist_order_reviews_dataset.csv": `"review_id","order_id","review_score","review_comment_title","review_comment_message","review_creation_date","review_answer_timestamp"
"r1","o1","4","Good","Nice product","2017-01-11 10:00:00","2017-01-12 10:00:00"
"r2","o2","3","OK","Average","2017-02-06 10:00:00","2017-02-07 10:00:00"
`,
	"olist_products_dataset.csv": `"product_id","product_category_name","product_name_lenght","product_description_lenght","product_photos_qty","product_weight_g","product_length_cm","product_height_cm","product_width_cm"
"p1","electronics","20","100","3","500","30","10","20"
"p2","furniture","15","80","2","5000","100","50","60"
`,
	"olist_sellers_dataset.csv": `"seller_id","seller_zip_code_prefix","seller_city","seller_state"
"s1","01001","sao paulo","SP"
"s2","02001","rio de janeiro","RJ"
`,
	"olist_geolocation_dataset.csv": `"geolocation_zip_code_prefix","geolocation_lat","geolocation_lng","geolocation_city","geolocation_state"
"01001","-23.550520","-46.633309","sao paulo","SP"
"02001","-22.906847","-43.172896","rio de janeiro","RJ"
`,
	"product_category_name_translation.csv": `product_category_name,product_category_name_english
electronics,electronics
furniture,furniture
`,
	"olist_marketing_qualified_leads_dataset.csv": `mql_id,first_contact_date,landing_page_id,origin
"m1","2017-01-01","lp1","organic_search"
"m2","2017-02-01","lp2","paid_search"
`,
	"olist_closed_deals_dataset.csv": `mql_id,seller_id,sdr_id,sr_id,won_date,business_segment,lead_type,lead_behaviour_profile,has_company,has_gtin,average_stock,business_type,declared_product_catalog_size,declared_monthly_revenue
"m1","s1","sdr1","sr1","2017-01-15","segment_a","type_a","profile_a","true","false","medium","b2b","100","5000.00"
"m2","s2","sdr2","sr2","2017-02-01","segment_b","type_b","profile_b","false","true","large","b2c","200","10000.00"
`,
}

// expectedRows maps CSV filename -> expected row count after ingestion.
var expectedRows = map[string]int64{
	"olist_customers_dataset.csv":                 2,
	"olist_orders_dataset.csv":                    2,
	"olist_order_items_dataset.csv":               2,
	"olist_order_payments_dataset.csv":            2,
	"olist_order_reviews_dataset.csv":             2,
	"olist_products_dataset.csv":                  2,
	"olist_sellers_dataset.csv":                   2,
	"olist_geolocation_dataset.csv":               2,
	"product_category_name_translation.csv":       2,
	"olist_marketing_qualified_leads_dataset.csv": 2,
	"olist_closed_deals_dataset.csv":              2,
}

// DDL to create raw tables for testing. This mirrors migrations/002_raw_tables.sql
// but excludes the audit/metadata columns not present in CSV files since they
// have defaults or accept NULL.
const rawTableDDL = `
CREATE SCHEMA IF NOT EXISTS raw;

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
`

// setupTestDB creates database tables needed for the ingest test.
func setupTestDB(t *testing.T) *pgxpool.Pool {
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

	// Clean any leftover data from previous runs
	for _, mapping := range ingest.AllTableMappings() {
		_, _ = pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s", mapping.TableName))
	}

	return pool
}

// writeTestCSVs writes the test CSV files to the given directory.
func writeTestCSVs(t *testing.T, dir string) {
	t.Helper()
	for name, content := range testCSVs {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write test CSV %s: %v", name, err)
		}
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestIngestRaw_Name(t *testing.T) {
	step := NewIngestRawStep()
	if got := step.Name(); got != "ingest_raw" {
		t.Errorf("expected name 'ingest_raw', got %q", got)
	}
}

func TestIngestRaw_MissingRequiredCSV(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupTestDB(t)
	defer pool.Close()

	// Use an empty temp dir so CSVs don't exist
	emptyDir := t.TempDir()

	step := NewIngestRawStep()
	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(context.Background())

	_, err = step.Run(context.Background(), tx, pipeline.StepInput{
		DataDir: emptyDir,
		Logger:  zap.NewNop(),
	})
	if err == nil {
		t.Fatal("expected error when required CSV is missing")
	}
}

func TestIngestRaw_SuccessfulLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupTestDB(t)
	defer pool.Close()

	// Write test CSVs to a temp directory
	dataDir := t.TempDir()
	writeTestCSVs(t, dataDir)

	step := NewIngestRawStep()
	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(context.Background())

	output, err := step.Run(context.Background(), tx, pipeline.StepInput{
		DataDir: dataDir,
		Logger:  zap.NewNop(),
	})
	if err != nil {
		t.Fatalf("step run failed: %v", err)
	}

	// Verify aggregated counts
	var expectedTotal int64
	for _, cnt := range expectedRows {
		expectedTotal += cnt
	}

	if output.InputCount != expectedTotal {
		t.Errorf("expected input_count %d, got %d", expectedTotal, output.InputCount)
	}
	if output.OutputCount != expectedTotal {
		t.Errorf("expected output_count %d, got %d", expectedTotal, output.OutputCount)
	}

	// Verify per-table row counts
	for _, m := range ingest.AllTableMappings() {
		var count int64
		q := fmt.Sprintf("SELECT COUNT(*) FROM %s", m.TableName)
		if err := tx.QueryRow(context.Background(), q).Scan(&count); err != nil {
			t.Fatalf("query %s: %v", m.TableName, err)
		}
		expected := expectedRows[m.CSVFile]
		// Optional tables might not exist if the CSV wasn't required, but
		// in this test we wrote all CSVs so they should all be present.
		if m.Required || testCSVs[m.CSVFile] != "" {
			if count != expected {
				t.Errorf("table %s: expected %d rows, got %d", m.TableName, expected, count)
			}
		}
	}
}

func TestIngestRaw_IdempotentReRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupTestDB(t)
	defer pool.Close()

	dataDir := t.TempDir()
	writeTestCSVs(t, dataDir)

	step := NewIngestRawStep()

	// First run
	func() {
		tx, err := pool.Begin(context.Background())
		if err != nil {
			t.Fatalf("begin tx: %v", err)
		}
		defer tx.Rollback(context.Background())

		_, err = step.Run(context.Background(), tx, pipeline.StepInput{
			DataDir: dataDir,
			Logger:  zap.NewNop(),
		})
		if err != nil {
			t.Fatalf("first run failed: %v", err)
		}

		// Commit so data is visible to next transaction
		if err := tx.Commit(context.Background()); err != nil {
			t.Fatalf("commit first run: %v", err)
		}
	}()

	// Second run — should TRUNCATE then reload same data
	func() {
		tx, err := pool.Begin(context.Background())
		if err != nil {
			t.Fatalf("begin tx: %v", err)
		}
		defer tx.Rollback(context.Background())

		output, err := step.Run(context.Background(), tx, pipeline.StepInput{
			DataDir: dataDir,
			Logger:  zap.NewNop(),
		})
		if err != nil {
			t.Fatalf("second run failed: %v", err)
		}

		var expectedTotal int64
		for _, cnt := range expectedRows {
			expectedTotal += cnt
		}
		if output.InputCount != expectedTotal {
			t.Errorf("re-run: expected input_count %d, got %d", expectedTotal, output.InputCount)
		}
		if output.OutputCount != expectedTotal {
			t.Errorf("re-run: expected output_count %d, got %d", expectedTotal, output.OutputCount)
		}

		// Verify no duplicate rows (each table should have exact expected rows)
		for _, m := range ingest.AllTableMappings() {
			var count int64
			q := fmt.Sprintf("SELECT COUNT(*) FROM %s", m.TableName)
			if err := tx.QueryRow(context.Background(), q).Scan(&count); err != nil {
				t.Fatalf("query %s: %v", m.TableName, err)
			}
			expected := expectedRows[m.CSVFile]
			if count != expected {
				t.Errorf("re-run table %s: expected %d rows, got %d", m.TableName, expected, count)
			}
		}
	}()
}
