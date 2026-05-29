//go:build integration

package e2e_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"baxi/internal/decision"
	"baxi/internal/llm"
	"baxi/internal/pipeline"
	"baxi/internal/pipeline/steps"
	"baxi/internal/repository"
	"baxi/internal/testutil"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// setupTestDB starts a test PostgreSQL container, runs migrations, and returns
// a pool and cleanup function.
func setupTestDB(t *testing.T) (context.Context, *pgxpool.Pool, func()) {
	t.Helper()
	ctx := context.Background()

	pg, err := testutil.StartPostgres(ctx)
	require.NoError(t, err, "start postgres container")

	pool, err := pgxpool.New(ctx, pg.ConnectionString())
	require.NoError(t, err, "create pgxpool")

	err = pg.RunMigrations(ctx, "../../migrations")
	require.NoError(t, err, "run migrations")

	cleanup := func() {
		pool.Close()
		_ = pg.Terminate(ctx)
	}
	return ctx, pool, cleanup
}

// setupPipeline creates a pipeline.Runner with all 9 production steps in
// dependency order. The DataDir is provided at Run time via RunInput.
func setupPipeline(t *testing.T, pool *pgxpool.Pool) *pipeline.Runner {
	t.Helper()

	return &pipeline.Runner{
		DB: pool,
		Steps: []pipeline.Step{
			steps.NewIngestRawStep(),
			steps.NewBuildDWDSOrderLevelStep(),
			steps.NewBuildDWDItemLevelStep(),
			steps.NewBuildMetricDailyStep(),
			steps.NewBuildMetricDimensionDailyStep(),
			steps.NewDetectAlertsStep(),
			steps.NewGenerateRecommendationsStep(),
			steps.NewGenerateTasksStep(),
			steps.NewCreateOutboxStep(),
		},
		Log: zap.NewNop(),
	}
}

// queryInt is a convenience helper that executes a scalar COUNT/integer query.
func queryInt(t *testing.T, pool *pgxpool.Pool, ctx context.Context, sql string, args ...interface{}) int {
	t.Helper()
	var n int
	err := pool.QueryRow(ctx, sql, args...).Scan(&n)
	require.NoError(t, err, "query: %s", sql)
	return n
}

// ---------------------------------------------------------------------------
// Decision flow helpers
// ---------------------------------------------------------------------------

// fetchHighSeverityAlert queries the first high-severity alert from ops.metric_alert.
type alertRow struct {
	AlertID       string
	RuleID        string
	Severity      string
	MetricName    string
	ObjectType    string
	ObjectID      string
	CurrentValue  float64
	BaselineValue float64
	ChangeRate    float64
}

func fetchHighSeverityAlert(t *testing.T, pool *pgxpool.Pool, ctx context.Context) alertRow {
	t.Helper()
	var a alertRow
	err := pool.QueryRow(ctx, `
		SELECT alert_id, rule_id, severity, metric_name,
		       object_type, object_id,
		       COALESCE(current_value, 0),
		       COALESCE(baseline_value, 0),
		       COALESCE(change_rate, 0)
		FROM ops.metric_alert
		WHERE severity IN ('high', 'critical')
		ORDER BY
			CASE severity WHEN 'critical' THEN 0 WHEN 'high' THEN 1 ELSE 2 END,
			alert_id
		LIMIT 1
	`).Scan(&a.AlertID, &a.RuleID, &a.Severity, &a.MetricName,
		&a.ObjectType, &a.ObjectID,
		&a.CurrentValue, &a.BaselineValue, &a.ChangeRate)
	require.NoError(t, err, "fetch high-severity alert")
	return a
}

// buildDecisionContext constructs a DecisionContext directly from an alert row,
// bypassing the ContextBuilder (which requires ontology infrastructure that the
// pipeline does not populate). This is the correct E2E approach: the pipeline
// produces alerts, and the decision engine consumes them.
func buildDecisionContext(a alertRow, caseID string) *decision.DecisionContext {
	allowedActions := []string{
		"create_followup_task",
		"notify_owner",
		"export_report",
		"escalate_to_human",
	}
	forbiddenActions := []string{
		"execute_dispatch",
		"modify_raw_data",
		"write_dwd",
		"write_mart",
	}

	return &decision.DecisionContext{
		DecisionCaseID: caseID,
		SourceType:     strPtr("alert"),
		SourceID:       &a.AlertID,
		Trigger: decision.TriggerInfo{
			AlertID:       a.AlertID,
			RuleID:        a.RuleID,
			Severity:      a.Severity,
			MetricName:    a.MetricName,
			CurrentValue:  a.CurrentValue,
			BaselineValue: a.BaselineValue,
			DeltaPct:      a.ChangeRate,
		},
		ObjectContext: decision.ObjectContextData{
			ObjectType: a.ObjectType,
			ObjectID:   a.ObjectID,
			Properties: map[string]interface{}{
				"alert_id":        a.AlertID,
				"rule_id":         a.RuleID,
				"severity":        a.Severity,
				"metric_name":     a.MetricName,
				"current_value":   a.CurrentValue,
				"baseline_value":  a.BaselineValue,
				"delta_pct":       a.ChangeRate,
			},
		},
		Governance: decision.GovernanceData{
			Classification:   "internal",
			RedactionApplied: false,
			Role:             "agent_readonly",
		},
		AllowedActions:   allowedActions,
		ForbiddenActions: forbiddenActions,
	}
}

func strPtr(s string) *string { return &s }

// buildDecisionEngine creates a DecisionEngine with a RuleBasedProvider and
// real DB-backed repositories. No real LLM calls are made.
func buildDecisionEngine(pool *pgxpool.Pool) *decision.DecisionEngine {
	decisionRepo := repository.NewDecisionRepository()
	ruleProvider := llm.NewRuleBasedProvider()
	auditLogger := llm.NewDBAuditLogger(pool)
	return decision.NewDecisionEngine(ruleProvider, decisionRepo, pool, auditLogger)
}

// ---------------------------------------------------------------------------
// Test 1: Full pipeline run through to decision dispatch
// ---------------------------------------------------------------------------

func TestFullPipelineToDispatch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, pool, cleanup := setupTestDB(t)
	defer cleanup()

	runner := setupPipeline(t, pool)

	// Step 1: Run the full pipeline
	input := pipeline.RunInput{
		RunType: "full",
		Mode:    "test",
		DataDir: "../../data/raw",
	}
	err := runner.Run(ctx, input)
	require.NoError(t, err, "pipeline run should succeed")

	// Step 2: Verify audit.pipeline_run — 1 completed run
	completedRuns := queryInt(t, pool, ctx,
		`SELECT COUNT(*) FROM audit.pipeline_run WHERE status = 'completed'`)
	require.Equal(t, 1, completedRuns, "expected 1 completed pipeline_run")

	// Step 3: Verify audit.pipeline_step_run — 9 completed steps
	completedSteps := queryInt(t, pool, ctx,
		`SELECT COUNT(*) FROM audit.pipeline_step_run WHERE status = 'completed'`)
	require.Equal(t, 9, completedSteps, "expected 9 completed pipeline_step_run rows")

	// Step 4: Verify dwd.order_level count = 99441
	orderLevelCount := queryInt(t, pool, ctx,
		`SELECT COUNT(*) FROM dwd.order_level`)
	require.Equal(t, 99441, orderLevelCount, "dwd.order_level row count")

	// Step 5: Verify dwd.item_level count = 112650
	itemLevelCount := queryInt(t, pool, ctx,
		`SELECT COUNT(*) FROM dwd.item_level`)
	require.Equal(t, 112650, itemLevelCount, "dwd.item_level row count")

	// Step 6: Verify mart.metric_daily count = 634
	metricDailyCount := queryInt(t, pool, ctx,
		`SELECT COUNT(*) FROM mart.metric_daily`)
	require.Equal(t, 634, metricDailyCount, "mart.metric_daily row count")

	// Step 7: Verify ops.metric_alert count > 0
	alertCount := queryInt(t, pool, ctx,
		`SELECT COUNT(*) FROM ops.metric_alert`)
	require.Greater(t, alertCount, 0, "expected at least 1 metric_alert")

	// Step 8: Verify ops.recommendation count = ops.metric_alert count
	recCount := queryInt(t, pool, ctx,
		`SELECT COUNT(*) FROM ops.recommendation`)
	require.Equal(t, alertCount, recCount,
		"recommendation count should equal alert count")

	// Step 9: Verify ops.task count = ops.recommendation count
	taskCount := queryInt(t, pool, ctx,
		`SELECT COUNT(*) FROM ops.task`)
	require.Equal(t, recCount, taskCount,
		"task count should equal recommendation count")

	// Step 10: Verify ops.outbox_event count = ops.task count
	outboxCount := queryInt(t, pool, ctx,
		`SELECT COUNT(*) FROM ops.outbox_event`)
	require.Equal(t, taskCount, outboxCount,
		"outbox_event count should equal task count")

	// ------------------------------------------------------------------
	// Decision flow (steps 11-17): alert → case → decision → approve → dispatch
	// ------------------------------------------------------------------

	// Step 11: Query a high-severity alert
	alert := fetchHighSeverityAlert(t, pool, ctx)
	require.NotEmpty(t, alert.AlertID, "must find at least one high/critical alert")

	// Step 12: Create a decision case directly in ai.decision_case
	caseID := fmt.Sprintf("e2e-case-%s", alert.AlertID)
	_, err = pool.Exec(ctx, `
		INSERT INTO ai.decision_case (case_id, status, source_type, source_id,
		                              object_type, object_id, severity, created_at)
		VALUES ($1, 'created', 'alert', $2, $3, $4, $5, NOW())
	`, caseID, alert.AlertID, alert.ObjectType, alert.ObjectID, alert.Severity)
	require.NoError(t, err, "insert decision_case")

	// Step 13: Build DecisionContext and generate a decision via DecisionEngine
	decCtx := buildDecisionContext(alert, caseID)
	engine := buildDecisionEngine(pool)

	output, err := engine.GenerateDecision(ctx, caseID, decCtx)
	require.NoError(t, err, "GenerateDecision should succeed")
	require.NotNil(t, output, "decision output should not be nil")
	require.NotEmpty(t, output.DecisionType, "decision type should be set")
	require.NotEmpty(t, output.Summary, "decision summary should be set")
	require.True(t, output.RequiresHumanReview, "high/critical should require human review")

	// Step 14: Verify ai.llm_decision record was created
	llmDecisionCount := queryInt(t, pool, ctx,
		`SELECT COUNT(*) FROM ai.llm_decision WHERE case_id = $1`, caseID)
	require.Equal(t, 1, llmDecisionCount, "expected 1 llm_decision record")

	// Step 15: Create an action_proposal from the decision output
	// Valid action_types per migration 011: create_followup_task, notify_owner, export_report, create_outbox_message
	actionType := "create_followup_task"
	if len(output.RecommendedActions) > 0 {
		validTypes := map[string]bool{
			"create_followup_task": true, "notify_owner": true,
			"export_report": true, "create_outbox_message": true,
		}
		for _, a := range output.RecommendedActions {
			if validTypes[a.ActionType] {
				actionType = a.ActionType
				break
			}
		}
	}
	proposalID := fmt.Sprintf("e2e-prop-%s", alert.AlertID)
	_, err = pool.Exec(ctx, `
		INSERT INTO ai.action_proposal (
			proposal_id, case_id, action_type, apply_status, title,
			requires_human_review, created_at
		) VALUES ($1, $2, $3, 'proposed', $4, true, NOW())
	`, proposalID, caseID, actionType, output.Summary)
	require.NoError(t, err, "insert action_proposal")

	// Step 16: Approve the proposal via review_record
	_, err = pool.Exec(ctx, `
		INSERT INTO ai.review_record (
			review_id, proposal_id, reviewer_id, verdict, feedback, reviewed_at
		) VALUES ($1, $2, 'e2e-reviewer', 'approve', 'E2E auto-approve', NOW())
	`, fmt.Sprintf("e2e-review-%s", alert.AlertID), proposalID)
	require.NoError(t, err, "insert review_record")

	_, err = pool.Exec(ctx, `
		UPDATE ai.action_proposal SET apply_status = 'approved' WHERE proposal_id = $1
	`, proposalID)
	require.NoError(t, err, "update proposal to approved")

	// Verify approved status
	var applyStatus string
	err = pool.QueryRow(ctx,
		`SELECT apply_status FROM ai.action_proposal WHERE proposal_id = $1`,
		proposalID).Scan(&applyStatus)
	require.NoError(t, err)
	require.Equal(t, "approved", applyStatus)

	// Step 17: Create an outbox event from the approved proposal and mark dispatched
	outboxEventID := fmt.Sprintf("e2e-evt-%s", alert.AlertID)
	payloadJSON := fmt.Sprintf(
		`{"proposal_id":"%s","case_id":"%s","action_type":"%s","title":"%s"}`,
		proposalID, caseID, output.DecisionType, output.Summary)
	_, err = pool.Exec(ctx, `
		INSERT INTO ops.outbox_event (
			event_id, event_type, source_type, source_id,
			payload_json, target_channel, status, created_at, dispatch_attempts
		) VALUES ($1, $2, 'action_execution', $3, $4, 'local_cli', 'pending', NOW(), 0)
	`, outboxEventID, output.DecisionType, proposalID, payloadJSON)
	require.NoError(t, err, "insert outbox_event")

	// Simulate dispatch: update status to 'dispatched'
	_, err = pool.Exec(ctx, `
		UPDATE ops.outbox_event SET status = 'dispatched' WHERE event_id = $1
	`, outboxEventID)
	require.NoError(t, err, "mark outbox_event as dispatched")

	// Verify final outbox_event status
	var eventStatus string
	err = pool.QueryRow(ctx,
		`SELECT status FROM ops.outbox_event WHERE event_id = $1`,
		outboxEventID).Scan(&eventStatus)
	require.NoError(t, err)
	require.Equal(t, "dispatched", eventStatus)
}

// ---------------------------------------------------------------------------
// Test 2: Pipeline idempotency — running twice yields identical row counts
// ---------------------------------------------------------------------------

func TestPipelineIdempotency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, pool, cleanup := setupTestDB(t)
	defer cleanup()

	runner := setupPipeline(t, pool)
	input := pipeline.RunInput{
		RunType: "full",
		Mode:    "test",
		DataDir: "../../data/raw",
	}

	// First run
	err := runner.Run(ctx, input)
	require.NoError(t, err, "first pipeline run")

	// Snapshot row counts after first run
	orderAfter1 := queryInt(t, pool, ctx, `SELECT COUNT(*) FROM dwd.order_level`)
	itemAfter1 := queryInt(t, pool, ctx, `SELECT COUNT(*) FROM dwd.item_level`)
	metricAfter1 := queryInt(t, pool, ctx, `SELECT COUNT(*) FROM mart.metric_daily`)
	alertAfter1 := queryInt(t, pool, ctx, `SELECT COUNT(*) FROM ops.metric_alert`)
	recAfter1 := queryInt(t, pool, ctx, `SELECT COUNT(*) FROM ops.recommendation`)
	taskAfter1 := queryInt(t, pool, ctx, `SELECT COUNT(*) FROM ops.task`)
	outboxAfter1 := queryInt(t, pool, ctx, `SELECT COUNT(*) FROM ops.outbox_event`)

	// Second run
	err = runner.Run(ctx, input)
	require.NoError(t, err, "second pipeline run")

	// Verify row counts are identical
	require.Equal(t, orderAfter1,
		queryInt(t, pool, ctx, `SELECT COUNT(*) FROM dwd.order_level`),
		"order_level count should be unchanged")
	require.Equal(t, itemAfter1,
		queryInt(t, pool, ctx, `SELECT COUNT(*) FROM dwd.item_level`),
		"item_level count should be unchanged")
	require.Equal(t, metricAfter1,
		queryInt(t, pool, ctx, `SELECT COUNT(*) FROM mart.metric_daily`),
		"metric_daily count should be unchanged")
	require.Equal(t, alertAfter1,
		queryInt(t, pool, ctx, `SELECT COUNT(*) FROM ops.metric_alert`),
		"metric_alert count should be unchanged")
	require.Equal(t, recAfter1,
		queryInt(t, pool, ctx, `SELECT COUNT(*) FROM ops.recommendation`),
		"recommendation count should be unchanged")
	require.Equal(t, taskAfter1,
		queryInt(t, pool, ctx, `SELECT COUNT(*) FROM ops.task`),
		"task count should be unchanged")
	require.Equal(t, outboxAfter1,
		queryInt(t, pool, ctx, `SELECT COUNT(*) FROM ops.outbox_event`),
		"outbox_event count should be unchanged")

	// Verify 2 pipeline_run records
	runCount := queryInt(t, pool, ctx,
		`SELECT COUNT(*) FROM audit.pipeline_run`)
	require.Equal(t, 2, runCount, "expected 2 pipeline_run records")
}

// ---------------------------------------------------------------------------
// Test 3: Alert chain integrity — counts must form a 1:1:1:1 chain
// ---------------------------------------------------------------------------

func TestAlertChainIntegrity(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, pool, cleanup := setupTestDB(t)
	defer cleanup()

	runner := setupPipeline(t, pool)
	err := runner.Run(ctx, pipeline.RunInput{
		RunType: "full",
		Mode:    "test",
		DataDir: "../../data/raw",
	})
	require.NoError(t, err, "pipeline run")

	alertCount := queryInt(t, pool, ctx, `SELECT COUNT(*) FROM ops.metric_alert`)
	recCount := queryInt(t, pool, ctx, `SELECT COUNT(*) FROM ops.recommendation`)
	taskCount := queryInt(t, pool, ctx, `SELECT COUNT(*) FROM ops.task`)
	outboxCount := queryInt(t, pool, ctx, `SELECT COUNT(*) FROM ops.outbox_event`)

	require.Greater(t, alertCount, 0, "expected at least 1 alert")
	require.Equal(t, alertCount, recCount,
		"alert count should equal recommendation count")
	require.Equal(t, recCount, taskCount,
		"recommendation count should equal task count")
	require.Equal(t, taskCount, outboxCount,
		"task count should equal outbox_event count")
}
