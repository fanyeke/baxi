package pipeline_test

import (
	"context"
	"os"
	"testing"

	"baxi/internal/pipeline"
	"baxi/internal/pipeline/steps"
	"baxi/internal/testutil"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func TestFullPipeline_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	pg, err := testutil.StartPostgres(ctx)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	defer func() {
		if err := pg.Terminate(ctx); err != nil {
			t.Errorf("terminate postgres: %v", err)
		}
	}()

	if err := pg.RunMigrations(ctx, "../../migrations"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	pool, err := pgxpool.New(ctx, pg.ConnectionString())
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	defer pool.Close()

	dataDir := "../../data/raw"
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		t.Fatalf("data directory not found at %s (run from project root)", dataDir)
	}

	runner := &pipeline.Runner{
		DB:    pool,
		Steps: allIntegrationSteps(),
		Log:   zap.NewNop(),
	}

	if err := runner.Run(ctx, pipeline.RunInput{
		RunType: "full",
		Mode:    "test",
		DataDir: dataDir,
	}); err != nil {
		t.Fatalf("pipeline run failed: %v", err)
	}

	// Verify audit records — 1 completed pipeline run, 9 completed step runs.
	assertRowCount(t, pool, ctx, "audit.pipeline_run WHERE status = 'completed'", 1)
	assertRowCount(t, pool, ctx, "audit.pipeline_step_run WHERE status = 'completed'", 9)

	// Verify core DWD and mart table row counts match the Python + SQLite
	// baseline exactly. These are deterministic from the raw CSV data.
	assertRowCount(t, pool, ctx, "dwd.order_level", 99441)
	assertRowCount(t, pool, ctx, "dwd.item_level", 112650)
	assertRowCount(t, pool, ctx, "mart.metric_daily", 634)

	// mart.metric_dimension_daily and ops tables differ slightly from the
	// baseline because the raw Olist reviews CSV contains 789 duplicate
	// review_id values. The Python/SQLite pipeline loaded all 99,224 review
	// rows (no PK enforcement), while the Go/PostgreSQL pipeline deduplicates
	// them at ingest time via INSERT … ON CONFLICT DO NOTHING.
	//
	// This dedup propagates a ~0.5 % difference into the dimension-level
	// metrics, and causes one additional global GMV drop alert to fire
	// (37 vs 36). The ops chain (alert → recommendation → task → outbox)
	// remains self-consistent.
	assertRowCount(t, pool, ctx, "mart.metric_dimension_daily", 690326)
	assertRowCount(t, pool, ctx, "ops.metric_alert", 37)
	assertRowCount(t, pool, ctx, "ops.recommendation", 37)
	assertRowCount(t, pool, ctx, "ops.task", 37)
	assertRowCount(t, pool, ctx, "ops.outbox_event", 37)
}

func allIntegrationSteps() []pipeline.Step {
	return []pipeline.Step{
		steps.NewIngestRawStep(),
		steps.NewBuildDWDSOrderLevelStep(),
		steps.NewBuildDWDItemLevelStep(),
		steps.NewBuildMetricDailyStep(),
		steps.NewBuildMetricDimensionDailyStep(),
		steps.NewDetectAlertsStep(),
		steps.NewGenerateRecommendationsStep(),
		steps.NewGenerateTasksStep(),
		steps.NewCreateOutboxStep(),
	}
}

func assertRowCount(t *testing.T, pool *pgxpool.Pool, ctx context.Context, tableOrWhere string, expected int64) {
	t.Helper()
	var count int64
	q := "SELECT COUNT(*) FROM " + tableOrWhere
	if err := pool.QueryRow(ctx, q).Scan(&count); err != nil {
		t.Errorf("count query failed (%s): %v", tableOrWhere, err)
		return
	}
	if count != expected {
		t.Errorf("row count mismatch: %s = %d, want %d", tableOrWhere, count, expected)
	}
}
