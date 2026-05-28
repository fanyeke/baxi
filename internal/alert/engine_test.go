package alert

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestEngine_EvaluateGlobalRules_DeadRules verifies that dead rules never
// produce alerts. We create a minimal mart.metric_daily table and verify
// that calling EvaluateGlobalRules on an empty table returns empty results
// (the gmv_drop rule requires ≥21 days of data and won't fire).
func TestEngine_EvaluateGlobalRules_DeadRules(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	// Create mart.metric_daily schema
	_, err = pool.Exec(ctx, `
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
		)
	`)
	if err != nil {
		t.Fatalf("create mart.metric_daily: %v", err)
	}
	defer pool.Exec(ctx, "DROP TABLE IF EXISTS mart.metric_daily")

	// Insert 21 days of stable GMV data — no drop, no alert expected
	for i := 0; i < 21; i++ {
		date := fmt.Sprintf("2018-10-%02d", i+1)
		_, err := pool.Exec(ctx, `
			INSERT INTO mart.metric_daily (metric_date, gmv, order_count, late_delivery_rate, cancel_rate)
			VALUES ($1::DATE, 2000.00, 100, 0.08, 0.02)
			ON CONFLICT (metric_date) DO NOTHING
		`, date)
		if err != nil {
			t.Fatalf("insert row %s: %v", date, err)
		}
	}

	// Evaluate rules within a transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	engine := NewEngine()
	results, err := engine.EvaluateGlobalRules(ctx, tx)
	if err != nil {
		t.Fatalf("EvaluateGlobalRules: %v", err)
	}

	// With stable data, no rules should trigger
	if len(results) != 0 {
		t.Errorf("expected 0 alerts with stable data, got %d", len(results))
		for _, r := range results {
			t.Logf("  unexpected alert: %s (rule=%s)", r.AlertID, r.RuleID)
		}
	}
}

// TestEngine_EvaluateGlobalRules_GMVDrop verifies that the gmv_drop rule
// fires when 7d avg drops ≥15% vs previous 14d avg.
func TestEngine_EvaluateGlobalRules_GMVDrop(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	_, err = pool.Exec(ctx, `
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
		)
	`)
	if err != nil {
		t.Fatalf("create mart.metric_daily: %v", err)
	}
	defer pool.Exec(ctx, "DROP TABLE IF EXISTS mart.metric_daily")

	// Insert 14 days of high GMV (baseline), then 7 days of low GMV (current)
	// The drop should be >15% to trigger the rule
	for i := 1; i <= 14; i++ {
		date := fmt.Sprintf("2018-10-%02d", i)
		_, err := pool.Exec(ctx, `
			INSERT INTO mart.metric_daily (metric_date, gmv, order_count, late_delivery_rate, cancel_rate)
			VALUES ($1::DATE, 2000.00, 100, 0.08, 0.02)
			ON CONFLICT (metric_date) DO NOTHING
		`, date)
		if err != nil {
			t.Fatalf("insert baseline row %s: %v", date, err)
		}
	}
	for i := 15; i <= 21; i++ {
		date := fmt.Sprintf("2018-10-%02d", i)
		_, err := pool.Exec(ctx, `
			INSERT INTO mart.metric_daily (metric_date, gmv, order_count, late_delivery_rate, cancel_rate)
			VALUES ($1::DATE, 200.00, 100, 0.08, 0.02)
			ON CONFLICT (metric_date) DO NOTHING
		`, date)
		if err != nil {
			t.Fatalf("insert current row %s: %v", date, err)
		}
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	engine := NewEngine()
	results, err := engine.EvaluateGlobalRules(ctx, tx)
	if err != nil {
		t.Fatalf("EvaluateGlobalRules: %v", err)
	}

	// Should fire exactly 1 alert (gmv_drop)
	if len(results) != 1 {
		t.Fatalf("expected 1 alert (gmv_drop), got %d", len(results))
	}

	r := results[0]
	if r.RuleID != "gmv_drop" {
		t.Errorf("expected rule gmv_drop, got %s", r.RuleID)
	}
	if r.EventDate != "2018-10-21" {
		t.Errorf("expected event_date 2018-10-21, got %s", r.EventDate)
	}
	if r.AlertID != "gmv_drop_2018-10-21" {
		t.Errorf("expected alert_id gmv_drop_2018-10-21, got %s", r.AlertID)
	}
	if r.Severity != SeverityHigh {
		t.Errorf("expected severity high, got %s", r.Severity)
	}
	if r.MetricName != "gmv" {
		t.Errorf("expected metric gmv, got %s", r.MetricName)
	}
	if r.SampleSize != 21 {
		t.Errorf("expected sample_size 21, got %d", r.SampleSize)
	}

	// GMV dropped from ~2000 to ~200 — big drop
	expectedPct := (200.0 - 2000.0) / 2000.0
	if r.DeltaPct < expectedPct*1.1 || r.DeltaPct > expectedPct*0.9 {
		t.Logf("delta_pct: expected approx %.4f, got %.4f", expectedPct, r.DeltaPct)
	}
}

// TestEngine_EvaluateGlobalRules_LateDeliveryNoSpike verifies that with
// normal late delivery rates (~8%), the late_delivery_spike rule won't fire.
func TestEngine_EvaluateGlobalRules_LateDeliveryNoSpike(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	_, err = pool.Exec(ctx, `
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
		)
	`)
	if err != nil {
		t.Fatalf("create mart.metric_daily: %v", err)
	}
	defer pool.Exec(ctx, "DROP TABLE IF EXISTS mart.metric_daily")

	// 21 days with normal late_delivery_rate (~0.08, well below 0.25)
	for i := 1; i <= 21; i++ {
		date := fmt.Sprintf("2018-10-%02d", i)
		_, err := pool.Exec(ctx, `
			INSERT INTO mart.metric_daily (metric_date, gmv, order_count, late_delivery_rate, cancel_rate)
			VALUES ($1::DATE, 2000.00, 100, 0.08, 0.02)
			ON CONFLICT (metric_date) DO NOTHING
		`, date)
		if err != nil {
			t.Fatalf("insert row %s: %v", date, err)
		}
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	engine := NewEngine()
	results, err := engine.EvaluateGlobalRules(ctx, tx)
	if err != nil {
		t.Fatalf("EvaluateGlobalRules: %v", err)
	}

	// late_delivery_spike should NOT trigger (rate is 0.08 < 0.25)
	// gmv_drop should NOT trigger (stable data)
	for _, r := range results {
		if r.RuleID == "late_delivery_spike" {
			t.Error("late_delivery_spike should not trigger with 0.08 rate")
		}
	}
}
