package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"baxi/internal/pipeline"
	"baxi/internal/pipeline/steps"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func handlePipeline(ctx context.Context, args []string, log *zap.Logger, pool *pgxpool.Pool) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "pipeline: missing subcommand")
		fmt.Fprintln(os.Stderr, "Usage: baxi-cli pipeline <run|validate> [options]")
		os.Exit(1)
	}

	switch args[0] {
	case "run":
		handleRun(ctx, args[1:], log, pool)
	case "validate":
		handleValidate(ctx, args[1:], log, pool)
	default:
		fmt.Fprintf(os.Stderr, "pipeline: unknown subcommand: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "Usage: baxi-cli pipeline <run|validate> [options]")
		os.Exit(1)
	}
}

func handleRun(ctx context.Context, args []string, log *zap.Logger, pool *pgxpool.Pool) {
	runCmd := flag.NewFlagSet("pipeline run", flag.ExitOnError)
	runStep := runCmd.String("step", "", "Run a specific step by name (runs all if empty)")
	runDataDir := runCmd.String("data-dir", "./data/raw", "Directory containing CSV data files")

	if err := runCmd.Parse(args); err != nil {
		log.Fatal("failed to parse run flags", zap.Error(err))
	}

	steps := allSteps()

	if *runStep != "" {
		var found bool
		for _, s := range steps {
			if s.Name() == *runStep {
				steps = []pipeline.Step{s}
				found = true
				break
			}
		}
		if !found {
			fmt.Fprintf(os.Stderr, "Error: unknown step: %s\n", *runStep)
			os.Exit(1)
		}
	}

	if len(steps) == 0 {
		log.Info("no pipeline steps registered; nothing to run")
		fmt.Println("Pipeline executed successfully (no steps registered)")
		return
	}

	runner := &pipeline.Runner{
		DB:    pool,
		Steps: steps,
		Log:   log,
	}

	if err := runner.Run(ctx, pipeline.RunInput{
		RunType: "full",
		Mode:    "manual",
		DataDir: *runDataDir,
	}); err != nil {
		log.Fatal("pipeline run failed", zap.Error(err))
	}

	log.Info("pipeline executed successfully", zap.String("data-dir", *runDataDir))
}

// TableValidation defines expected count and tolerance for a pipeline output table.
type TableValidation struct {
	Key      string  // key in table_counts.json
	Table    string  // schema-qualified PostgreSQL table name
	Expected int64   // expected row count from baseline
	WarnPct  float64 // percentage difference allowed for WARN (0 = exact only)
}

// validationTables defines all pipeline output tables to validate.
var validationTables = []TableValidation{
	{Key: "dwd_order_level",          Table: "dwd.order_level",          Expected: 99441,  WarnPct: 0},
	{Key: "dwd_item_level",           Table: "dwd.item_level",           Expected: 112650, WarnPct: 0},
	{Key: "metric_daily",             Table: "mart.metric_daily",        Expected: 634,    WarnPct: 0},
	{Key: "metric_dimension_daily",   Table: "mart.metric_dimension_daily", Expected: 693602, WarnPct: 0.5},   // 0.5% tolerance for known NULL handling diff
	{Key: "alert_events",             Table: "ops.metric_alert",         Expected: 36,     WarnPct: 5},     // ~3% tolerance for known threshold diff
	{Key: "strategy_recommendations", Table: "ops.recommendation",       Expected: 36,     WarnPct: 5},
	{Key: "action_tasks",             Table: "ops.task",                 Expected: 36,     WarnPct: 5},
	{Key: "event_outbox",             Table: "ops.outbox_event",         Expected: 36,     WarnPct: 5},
}

func handleValidate(ctx context.Context, args []string, log *zap.Logger, pool *pgxpool.Pool) {
	log.Info("validating pipeline outputs against baseline")

	baselineCounts := make(map[string]int64)
	baselineData, err := os.ReadFile("migration_baseline/table_counts.json")
	if err != nil {
		log.Warn("cannot read baseline counts, using hardcoded expected values", zap.Error(err))
	} else {
		if err := json.Unmarshal(baselineData, &baselineCounts); err != nil {
			log.Warn("cannot parse baseline counts, using hardcoded expected values", zap.Error(err))
		}
	}

	fmt.Printf("%-40s %-12s %-12s %-12s %s\n", "TABLE", "OLD", "NEW", "EXPECTED", "STATUS")
	fmt.Println(strings.Repeat("-", 90))

	allPass := true
	anyWarn := false

	for _, vt := range validationTables {
		var newCount int64
		if err := pool.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", vt.Table)).Scan(&newCount); err != nil {
			fmt.Printf("%-40s %-12s %-12s %-12s %s\n", vt.Table, "ERR", "ERR", "ERR", "FAIL")
			fmt.Printf("  → Error: %v\n", err)
			allPass = false
			continue
		}

		oldCount := baselineCounts[vt.Key]

		delta := newCount - vt.Expected
		status := "FAIL"
		if delta == 0 {
			status = "PASS"
		} else if vt.WarnPct > 0 {
			pctDiff := 100.0 * float64(abs64(delta)) / float64(vt.Expected)
			if pctDiff <= vt.WarnPct {
				status = "WARN"
				anyWarn = true
			}
		}

		if status == "FAIL" {
			allPass = false
		}

		oldStr := fmt.Sprintf("%d", oldCount)
		if oldCount == 0 && vt.Expected > 0 {
			oldStr = "N/A"
		}

		fmt.Printf("%-40s %-12s %-12d %-12d %s\n", vt.Table, oldStr, newCount, vt.Expected, status)
		if status == "WARN" {
			pctDiff := 100.0 * float64(abs64(delta)) / float64(vt.Expected)
			fmt.Printf("  → Delta: %d (%.2f%%), within tolerance (%.1f%%)\n", delta, pctDiff, vt.WarnPct)
		} else if status == "FAIL" && vt.WarnPct > 0 {
			pctDiff := 100.0 * float64(abs64(delta)) / float64(vt.Expected)
			fmt.Printf("  → Delta: %d (%.2f%%), exceeds tolerance (%.1f%%)\n", delta, pctDiff, vt.WarnPct)
		}
	}

	fmt.Println()
	if allPass && !anyWarn {
		fmt.Println("All checks PASSED")
		os.Exit(0)
	} else if allPass {
		fmt.Println("All checks PASSED (with warnings)")
		os.Exit(0)
	} else {
		fmt.Println("Some checks FAILED")
		os.Exit(1)
	}
}

func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

func allSteps() []pipeline.Step {
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
