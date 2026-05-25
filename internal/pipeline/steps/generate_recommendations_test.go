package steps

import (
	"context"
	"os"
	"testing"

	"baxi/internal/pipeline"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// generateRecOpsDDL creates the ops schema needed for integration testing.
const generateRecOpsDDL = `
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

CREATE TABLE IF NOT EXISTS ops.recommendation (
    recommendation_id   TEXT PRIMARY KEY,
    alert_id            TEXT,
    decision_source     TEXT NOT NULL DEFAULT 'heuristic',
    rule_id             TEXT,
    strategy_title      TEXT NOT NULL,
    strategy_detail     TEXT,
    target_object_type  TEXT,
    target_object_id    TEXT,
    expected_impact     TEXT,
    risk_level          TEXT,
    confidence          TEXT,
    requires_approval   BOOLEAN DEFAULT FALSE,
    approval_status     TEXT DEFAULT 'draft',
    execution_status    TEXT DEFAULT 'draft',
    owner_role          TEXT,
    success_metric      TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`

// setupGenerateRecTestDB creates ops tables and returns a clean pool.
func setupGenerateRecTestDB(t *testing.T) *pgxpool.Pool {
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

	if _, err := pool.Exec(ctx, generateRecOpsDDL); err != nil {
		t.Fatalf("create ops tables: %v", err)
	}

	_, _ = pool.Exec(ctx, "TRUNCATE TABLE ops.metric_alert CASCADE")
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE ops.recommendation CASCADE")

	return pool
}

func TestGenerateRecommendationsStep_Name(t *testing.T) {
	step := NewGenerateRecommendationsStep()
	if got := step.Name(); got != "generate_recommendations" {
		t.Errorf("expected name 'generate_recommendations', got %q", got)
	}
}

func TestGenerateRecommendationsStep_HappyPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGenerateRecTestDB(t)
	ctx := context.Background()

	logger, _ := zap.NewDevelopment()

	// Insert a few test alerts
	for _, alert := range []struct {
		id, rule, date, sev, metric, objType, objID string
		cur, base, chg                              float64
		sample                                      int64
	}{
		{"gmv_drop_2018-10-17", "gmv_drop", "2018-10-17", "high", "gmv", "global", "global", 148.98, 1991.98, -0.9252, 634},
		{"dim-late-sp", "region_late_delivery_spike", "2018-10-17", "high", "late_delivery_rate", "region", "SP", 0.2087, 0, 0, 181},
	} {
		_, err := pool.Exec(ctx, `
			INSERT INTO ops.metric_alert (alert_id, rule_id, event_date, severity, metric_name, object_type, object_id, current_value, baseline_value, change_rate, sample_size, status)
			VALUES ($1,$2,$3::DATE,$4,$5,$6,$7,$8,$9,$10,$11,'open')
			ON CONFLICT (alert_id) DO NOTHING
		`, alert.id, alert.rule, alert.date, alert.sev, alert.metric, alert.objType, alert.objID, alert.cur, alert.base, alert.chg, alert.sample)
		if err != nil {
			t.Fatalf("insert alert %s: %v", alert.id, err)
		}
	}

	step := NewGenerateRecommendationsStep()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	output, err := step.Run(ctx, tx, pipeline.StepInput{
		Logger: logger,
	})
	if err != nil {
		t.Fatalf("step Run failed: %v", err)
	}

	if output.InputCount != 2 {
		t.Errorf("expected InputCount=2, got %d", output.InputCount)
	}
	if output.OutputCount != 2 {
		t.Errorf("expected OutputCount=2, got %d", output.OutputCount)
	}

	// Verify recommendations were created
	var count int64
	if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM ops.recommendation").Scan(&count); err != nil {
		t.Fatalf("count recommendations: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 recommendation rows, got %d", count)
	}
}

func TestGenerateRecommendationsStep_Idempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGenerateRecTestDB(t)
	ctx := context.Background()

	logger, _ := zap.NewDevelopment()

	// Insert one alert
	_, err := pool.Exec(ctx, `
		INSERT INTO ops.metric_alert (alert_id, rule_id, event_date, severity, metric_name, object_type, object_id, current_value, status)
		VALUES ('gmv_drop_2018-10-17','gmv_drop','2018-10-17'::DATE,'high','gmv','global','global',148.98,'open')
		ON CONFLICT (alert_id) DO NOTHING
	`)
	if err != nil {
		t.Fatalf("insert alert: %v", err)
	}

	step := NewGenerateRecommendationsStep()

	// First run
	tx1, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx1: %v", err)
	}
	output1, err := step.Run(ctx, tx1, pipeline.StepInput{Logger: logger})
	if err != nil {
		tx1.Rollback(ctx)
		t.Fatalf("first Run failed: %v", err)
	}
	if err := tx1.Commit(ctx); err != nil {
		t.Fatalf("commit tx1: %v", err)
	}

	if output1.OutputCount != 1 {
		t.Errorf("first run: expected OutputCount=1, got %d", output1.OutputCount)
	}

	// Second run — should produce 0 new recommendations due to ON CONFLICT
	tx2, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx2: %v", err)
	}
	_, err = step.Run(ctx, tx2, pipeline.StepInput{Logger: logger})
	if err != nil {
		tx2.Rollback(ctx)
		t.Fatalf("second Run failed: %v", err)
	}
	tx2.Rollback(ctx)

	// ON CONFLICT DO NOTHING means duplicate inserts are silently ignored.
	// The actual row count should remain 1.
	var actualCount int64
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM ops.recommendation").Scan(&actualCount)
	if actualCount != 1 {
		t.Errorf("expected 1 recommendation row after idempotent re-run, got %d", actualCount)
	}
}
