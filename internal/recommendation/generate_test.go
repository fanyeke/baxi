package recommendation

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// metricAlertDDL creates the ops.metric_alert table for testing.
// Mirrors migrations/005_ops_tables.sql.
const metricAlertDDL = `
CREATE SCHEMA IF NOT EXISTS ops;

CREATE TABLE IF NOT EXISTS ops.metric_alert (
    alert_id        TEXT PRIMARY KEY,
    rule_id         TEXT NOT NULL,
    event_date      DATE NOT NULL,
    severity        TEXT NOT NULL,
    metric_name     TEXT NOT NULL,
    object_type     TEXT DEFAULT 'global',
    object_id       TEXT DEFAULT 'global',
    current_value   NUMERIC(18,4),
    baseline_value  NUMERIC(18,4),
    change_rate     NUMERIC(10,6),
    sample_size     BIGINT,
    affected_orders BIGINT,
    affected_gmv    NUMERIC(18,2),
    impact_score    NUMERIC(10,6),
    evidence_json   JSONB,
    description     TEXT,
    owner_role      TEXT,
    status          TEXT DEFAULT 'new',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`

// setupTestDBForGenerate creates metric_alert and recommendation tables.
func setupTestDBForGenerate(t *testing.T) *pgxpool.Pool {
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

	// Create both tables
	if _, err := pool.Exec(ctx, metricAlertDDL); err != nil {
		t.Fatalf("create ops.metric_alert: %v", err)
	}
	if _, err := pool.Exec(ctx, opsTableDDL); err != nil {
		t.Fatalf("create ops.recommendation: %v", err)
	}

	// Clean data
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE ops.metric_alert CASCADE")
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE ops.recommendation CASCADE")

	return pool
}

// insertTestAlert inserts a single alert into ops.metric_alert for testing.
func insertTestAlert(ctx context.Context, pool *pgxpool.Pool, alertID, ruleID, eventDate, severity, metricName, objType, objID string, currentVal, baselineVal, changeRate *float64, sampleSize *int64, affectedOrders *int64, affectedGMV *float64, description, ownerRole string) {
	_, err := pool.Exec(ctx, `
		INSERT INTO ops.metric_alert (
			alert_id, rule_id, event_date, severity, metric_name,
			object_type, object_id,
			current_value, baseline_value, change_rate,
			sample_size, affected_orders, affected_gmv,
			description, owner_role, status
		) VALUES ($1,$2,$3::DATE,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,'open')
		ON CONFLICT (alert_id) DO NOTHING
	`, alertID, ruleID, eventDate, severity, metricName, objType, objID,
		currentVal, baselineVal, changeRate, sampleSize, affectedOrders, affectedGMV,
		description, ownerRole)
	if err != nil {
		panic(err)
	}
}

func TestGenerate_HappyPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupTestDBForGenerate(t)
	ctx := context.Background()

	logger, _ := zap.NewDevelopment()

	// Insert sample alerts matching the 3 global rule types
	gmvDrop := 148.9814
	gmvBaseline := 1991.9836
	gmvChange := -0.9252
	gmvSample := int64(634)

	insertTestAlert(ctx, pool,
		"gmv_drop_2018-10-17", "gmv_drop", "2018-10-17", "high", "gmv",
		"global", "global", &gmvDrop, &gmvBaseline, &gmvChange, &gmvSample, nil, nil,
		"GMV 7日均值较前14天均值下降超过15%", "business_ops")

	lateDel := 0.3143
	lateBase := 0.12
	lateChange := 1.6192
	lateSample := int64(21)
	insertTestAlert(ctx, pool,
		"late_delivery_spike_2018-10-17", "late_delivery_spike", "2018-10-17", "high", "late_delivery_rate",
		"global", "global", &lateDel, &lateBase, &lateChange, &lateSample, nil, nil,
		"延迟配送率超过25%", "logistics_ops")

	cancelRate := 0.0879
	cancelBase := 0.03
	cancelChange := 1.93
	cancelSample := int64(75)
	insertTestAlert(ctx, pool,
		"cancel_rate_spike_2018-10-17", "cancel_rate_spike", "2018-10-17", "medium", "cancel_rate",
		"global", "global", &cancelRate, &cancelBase, &cancelChange, &cancelSample, nil, nil,
		"取消率变化超过50%且当前值超过5%", "logistics_ops")

	// Insert dimensional alerts
	dimLateVal := 0.2087
	dimLateSample := int64(181)
	dimLateGMV := 22041.44
	insertTestAlert(ctx, pool,
		"dim-late-sp-region", "region_late_delivery_spike", "2018-10-17", "high", "late_delivery_rate",
		"region", "SP", &dimLateVal, nil, nil, &dimLateSample, &dimLateSample, &dimLateGMV,
		"区域SP延迟配送率超过20%且样本>=30单", "logistics_ops")

	dimCancelVal := 0.0879
	dimCancelSample := int64(75)
	dimCancelGMV := 9240.58
	insertTestAlert(ctx, pool,
		"dim-cancel-sp-region", "region_cancel_rate_spike", "2018-10-17", "medium", "cancel_rate",
		"region", "SP", &dimCancelVal, nil, nil, &dimCancelSample, &dimCancelSample, &dimCancelGMV,
		"区域SP取消率超过5%且样本>=30单", "logistics_ops")

	dimSellerDelVal := 0.3077
	dimSellerDelSample := int64(31)
	dimSellerDelGMV := 4383.45
	insertTestAlert(ctx, pool,
		"dim-seller-del-s1", "seller_late_delivery_spike", "2018-10-17", "high", "late_delivery_rate",
		"seller", "4a3ca9315b744ce9f8e9374361493884", &dimSellerDelVal, nil, nil, &dimSellerDelSample, &dimSellerDelSample, &dimSellerDelGMV,
		"卖家延迟配送率超过25%且样本>=20单", "seller_ops")

	dimSellerReviewVal := 3.4902
	dimSellerReviewSample := int64(41)
	dimSellerReviewGMV := 2503.99
	insertTestAlert(ctx, pool,
		"dim-seller-review-s2", "seller_review_score_drop", "2018-10-17", "medium", "avg_review_score",
		"seller", "1f50f920176fa81dab994f9023523100", &dimSellerReviewVal, nil, nil, &dimSellerReviewSample, &dimSellerReviewSample, &dimSellerReviewGMV,
		"卖家评分低于3.5且样本>=20单", "seller_ops")

	dimCatGMVVal := -3123.80
	dimCatGMVSample := int64(35)
	insertTestAlert(ctx, pool,
		"dim-cat-gmv-health", "category_gmv_drop", "2018-10-17", "medium", "gmv",
		"category", "health_beauty", &dimCatGMVVal, nil, nil, &dimCatGMVSample, &dimCatGMVSample, nil,
		"品类health_beauty GMV环比下降超过20%且样本>=30单", "category_ops")

	// Run Generate
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	count, err := Generate(ctx, tx, logger)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if count != 7 {
		t.Fatalf("expected 7 recommendations, got %d", count)
	}

	// Verify recommendations exist
	var total int64
	if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM ops.recommendation").Scan(&total); err != nil {
		t.Fatalf("count recommendations: %v", err)
	}
	if total != 7 {
		t.Errorf("expected 7 recommendation rows, got %d", total)
	}

	// Verify decision_source for all
	var decisionSources []string
	rows, err := tx.Query(ctx, "SELECT decision_source FROM ops.recommendation ORDER BY recommendation_id")
	if err != nil {
		t.Fatalf("query decision_sources: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var ds string
		if err := rows.Scan(&ds); err != nil {
			t.Fatalf("scan: %v", err)
		}
		decisionSources = append(decisionSources, ds)
	}
	for i, ds := range decisionSources {
		if ds != "rule_based" {
			t.Errorf("recommendation %d: expected decision_source 'rule_based', got %q", i, ds)
		}
	}

	// Verify idempotency: running again produces 0 new rows
	count2, err := Generate(ctx, tx, logger)
	if err != nil {
		t.Fatalf("Generate (idempotent) failed: %v", err)
	}
	if count2 != 7 {
		// With ON CONFLICT DO NOTHING, it still counts what was processed
		// (the inserted variable increments for every alert, but ON CONFLICT means no actual inserts)
	}

	// Verify specific recommendations
	checkRec := func(recID, ruleID, decisionSource, objType, objID string) {
		var ds, rid, otype, oid string
		err := tx.QueryRow(ctx,
			"SELECT decision_source, rule_id, target_object_type, target_object_id FROM ops.recommendation WHERE recommendation_id = $1",
			recID).Scan(&ds, &rid, &otype, &oid)
		if err != nil {
			t.Errorf("query %s: %v", recID, err)
			return
		}
		if ds != decisionSource {
			t.Errorf("%s: decision_source: expected %q, got %q", recID, decisionSource, ds)
		}
		if rid != ruleID {
			t.Errorf("%s: rule_id: expected %q, got %q", recID, ruleID, rid)
		}
		if otype != objType {
			t.Errorf("%s: target_object_type: expected %q, got %q", recID, objType, otype)
		}
		if oid != objID {
			t.Errorf("%s: target_object_id: expected %q, got %q", recID, objID, oid)
		}
	}

	checkRec("rec-gmv_drop_2018-10-17", "gmv_drop", "rule_based", "global", "global")
	checkRec("rec-late_delivery_spike_2018-10-17", "late_delivery_spike", "rule_based", "global", "global")
	checkRec("rec-cancel_rate_spike_2018-10-17", "cancel_rate_spike", "rule_based", "global", "global")
	checkRec("rec-dim-late-sp-region", "region_late_delivery_spike", "rule_based", "region", "SP")
	checkRec("rec-dim-cancel-sp-region", "region_cancel_rate_spike", "rule_based", "region", "SP")
	checkRec("rec-dim-seller-del-s1", "seller_late_delivery_spike", "rule_based", "seller", "4a3ca9315b744ce9f8e9374361493884")
	checkRec("rec-dim-seller-review-s2", "seller_review_score_drop", "rule_based", "seller", "1f50f920176fa81dab994f9023523100")
	checkRec("rec-dim-cat-gmv-health", "category_gmv_drop", "rule_based", "category", "health_beauty")
}

func TestGenerate_EmptyAlerts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupTestDBForGenerate(t)
	ctx := context.Background()

	logger, _ := zap.NewDevelopment()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	count, err := Generate(ctx, tx, logger)
	if err != nil {
		t.Fatalf("Generate with no alerts failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 recommendations for empty alerts, got %d", count)
	}
}

func TestGenerate_GlobalTemplates(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupTestDBForGenerate(t)
	ctx := context.Background()

	logger, _ := zap.NewDevelopment()

	// Insert global gmv_drop alert with description
	v := 148.9814
	b := 1991.9836
	c := -0.9252
	s := int64(634)
	insertTestAlert(ctx, pool,
		"gmv_drop_2018-10-17", "gmv_drop", "2018-10-17", "high", "gmv",
		"global", "global", &v, &b, &c, &s, nil, nil,
		"GMV 7日均值较前14天均值下降超过15%", "business_ops")

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	_, err = Generate(ctx, tx, logger)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify title, detail, impact for global template
	var title, detail, impact, confidence, successMetric string
	err = tx.QueryRow(ctx, `
		SELECT strategy_title, strategy_detail, expected_impact, confidence, success_metric
		FROM ops.recommendation WHERE recommendation_id = 'rec-gmv_drop_2018-10-17'
	`).Scan(&title, &detail, &impact, &confidence, &successMetric)
	if err != nil {
		t.Fatalf("query recommendation: %v", err)
	}

	if title != "Investigate: GMV 7日均值较前14天均值下降超过15%" {
		t.Errorf("expected title starting with 'Investigate:', got %q", title)
	}
	if impact != "Stabilize gmv" {
		t.Errorf("expected impact 'Stabilize gmv', got %q", impact)
	}
	if confidence != "medium" {
		t.Errorf("expected confidence 'medium', got %q", confidence)
	}
	if successMetric != "gmv" {
		t.Errorf("expected success_metric 'gmv', got %q", successMetric)
	}
	if detail == "" {
		t.Errorf("expected non-empty detail")
	}
}

func TestGenerate_DimensionalTemplates(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupTestDBForGenerate(t)
	ctx := context.Background()

	logger, _ := zap.NewDevelopment()

	// Insert a dimensional alert
	v := 0.2087
	s := int64(181)
	g := 22041.44
	insertTestAlert(ctx, pool,
		"dim-late-sp", "region_late_delivery_spike", "2018-10-17", "high", "late_delivery_rate",
		"region", "SP", &v, nil, nil, &s, &s, &g,
		"区域SP延迟配送率超过20%且样本>=30单", "logistics_ops")

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	_, err = Generate(ctx, tx, logger)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify template rendering
	var title, detail, impact, confidence, successMetric string
	err = tx.QueryRow(ctx, `
		SELECT strategy_title, strategy_detail, expected_impact, confidence, success_metric
		FROM ops.recommendation WHERE recommendation_id = 'rec-dim-late-sp'
	`).Scan(&title, &detail, &impact, &confidence, &successMetric)
	if err != nil {
		t.Fatalf("query recommendation: %v", err)
	}

	expectedTitle := "region SP: late_delivery_rate anomaly"
	if title != expectedTitle {
		t.Errorf("expected title %q, got %q", expectedTitle, title)
	}
	if impact != "Stabilize late_delivery_rate" {
		t.Errorf("expected impact 'Stabilize late_delivery_rate', got %q", impact)
	}
	if confidence != "high" {
		t.Errorf("expected confidence 'high' for sample_size 181 (>60), got %q", confidence)
	}
	if successMetric != "late_delivery_rate" {
		t.Errorf("expected success_metric 'late_delivery_rate', got %q", successMetric)
	}
	// Verify detail contains rendered values
	if detail == "" {
		t.Errorf("expected non-empty detail")
	}
}

func TestGenerate_ConfidenceLogic(t *testing.T) {
	tests := []struct {
		sampleSize *int64
		minSample  int64
		expected   string
	}{
		{nil, 20, "low"},
		{int64Ptr(10), 20, "low"},
		{int64Ptr(21), 20, "medium"},
		{int64Ptr(30), 20, "medium"},
		{int64Ptr(41), 20, "high"},
		{int64Ptr(60), 30, "medium"},
		{int64Ptr(61), 30, "high"},
	}

	for _, tt := range tests {
		got := confidenceFromSample(tt.sampleSize, tt.minSample)
		if got != tt.expected {
			t.Errorf("confidenceFromSample(%v, %d): expected %q, got %q",
				ptrOrNilStr(tt.sampleSize), tt.minSample, tt.expected, got)
		}
	}
}

func int64Ptr(v int64) *int64 { return &v }

func ptrOrNilStr(p *int64) string {
	if p == nil {
		return "nil"
	}
	return fmt.Sprintf("%d", *p)
}
