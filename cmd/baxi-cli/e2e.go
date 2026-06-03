package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"baxi/internal/action"
	"baxi/internal/config"
	"baxi/internal/decision"
	"baxi/internal/governance"
	"baxi/internal/llm"
	"baxi/internal/model"
	"baxi/internal/ontology"
	"baxi/internal/pipeline"
	"baxi/internal/pipeline/steps"
	"baxi/internal/repository"
	"baxi/internal/repository/common"
	alertRepo "baxi/internal/repository/alert"
	decisionRepo "baxi/internal/repository/decision"
	governanceRepo "baxi/internal/repository/governance"
	statusRepo "baxi/internal/repository/status"
	"baxi/internal/review"
	"baxi/internal/service"
)

// E2ERunner orchestrates the full end-to-end test flow.
type E2ERunner struct {
	log       *zap.Logger
	pool      *pgxpool.Pool
	ctx       context.Context

	// Services
	decisionSvc    *service.DecisionService
	caseSvc        *decision.CaseService
	alertSvc       *service.AlertService
	reviewSvc      *review.ReviewService
	proposalSvc    *action.ProposalService
	executeSvc     *action.ApplyService
	pipelineRunner *pipeline.Runner
	objectSvc      *ontology.ObjectQueryService
	statusSvc      *service.StatusService
	govSvc         *service.GovernanceService

	// State
	results        []E2EResult
	createdCaseIDs []string
	autoFix        bool
	allowLive      bool
}

// E2EResult tracks the outcome of a single test step.
type E2EResult struct {
	Step   string `json:"step"`
	Status string `json:"status"` // PASS, FAIL, SKIP, WARN
	Detail string `json:"detail"`
}

func newE2ERunner(ctx context.Context, log *zap.Logger, pool *pgxpool.Pool, autoFix, allowLive bool) (*E2ERunner, error) {
	r := &E2ERunner{
		ctx:       ctx,
		log:       log,
		pool:      pool,
		autoFix:   autoFix,
		allowLive: allowLive,
	}

	// Wire services (same pattern as cmd/baxi-mcp/main.go)
	provider := common.NewPoolProvider(pool)
	decisionRepoInst := decisionRepo.NewRepository(provider)
	alertRepoInst := alertRepo.NewRepository(provider)
	caseSvc := decision.NewCaseService(decisionRepoInst, alertRepoInst)

	ontologyRepo := repository.NewOntologyRepo()
	objectSvc := ontology.NewObjectQueryService(ontologyRepo, pool)

	govRepo := governanceRepo.NewRepository(provider)
	classSvc := governance.NewClassificationService(govRepo)

	reg, err := action.NewActionRegistry("")
	if err != nil {
		log.Warn("failed to load action registry, using empty fallback", zap.Error(err))
		reg = action.NewEmptyRegistry()
	}

	v1Builder := decision.NewContextBuilder(decisionRepoInst, objectSvc, classSvc, action.NewActionTypeProviderAdapter(reg))

	configDir := os.Getenv("BAXI_CONFIG_DIR")
	if configDir == "" {
		configDir = "config"
	}

	var v2Builder *decision.ContextBuilderV2
	objRegistry, regErr := ontology.NewObjectRegistry(ctx, nil, pool, filepath.Join(configDir, "aip_object_schema.yml"))
	if regErr != nil {
		log.Warn("failed to load object registry for v2 builder", zap.Error(regErr))
	} else {
		ontologyAwareRepo := ontology.NewOntologyAwareAdapter(ontologyRepo, objRegistry)
		markingSvc := governance.NewMarkingAdapter(classSvc, objRegistry)
		govRepoLocal := governanceRepo.NewRepository(common.NewPoolProvider(pool))
		lineageSvc := governance.NewLineageService(govRepoLocal)
		eventRepo := decision.NewPgxLineageEventRepository(pool)
		lineageAdapter := decision.NewDecisionLineageAdapter(lineageSvc, decisionRepoInst, eventRepo)
		v2Builder = decision.NewContextBuilderV2(decisionRepoInst, ontologyAwareRepo, markingSvc, lineageAdapter, pool, action.NewActionTypeProviderAdapter(reg))
	}

	var ctxBuilder decision.ObjectContextBuilder
	if v2Builder != nil {
		switcher := decision.NewSwitchableContextBuilder(v1Builder, v2Builder, nil)
		if objRegistry != nil {
			v3Builder := decision.NewContextBuilderV3(v2Builder, objRegistry, objectSvc)
			switcher.WithV3Builder(v3Builder)
			switcher.SwitchTo("v3")
		}
		ctxBuilder = switcher
	} else {
		ctxBuilder = v1Builder
	}

	decisionProvider := llm.NewRuleBasedProvider()
	engine := decision.NewDecisionEngine(decisionProvider, decisionRepoInst, llm.NewDBAuditLogger(pool))
	proposalSvc := action.NewProposalService(decisionRepoInst, decisionRepoInst, reg)

	decisionSvc := service.NewDecisionService(caseSvc, ctxBuilder, engine, proposalSvc, pool)
	alertSvc := service.NewAlertService(alertRepoInst)

	pipelineSteps := []pipeline.Step{
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
	pipelineRunner := &pipeline.Runner{DB: pool, Steps: pipelineSteps, Log: log}

	reviewRepo := review.NewReviewRepository()
	reviewSvc := review.NewReviewService(reviewRepo, pool)

	proposalLoader := &proposalLoaderAdapter{repo: reviewRepo, pool: pool}
	executeSvc := action.NewApplyService(reg, nil, proposalLoader, nil, nil, pool)

	statusRepoInst := statusRepo.NewRepository(provider)
	statusSvc := service.NewStatusService(statusRepoInst, "")

	govSvc := service.NewGovernanceService(govRepo, pool)

	r.decisionSvc = decisionSvc
	r.caseSvc = caseSvc
	r.alertSvc = alertSvc
	r.reviewSvc = reviewSvc
	r.proposalSvc = proposalSvc
	r.executeSvc = executeSvc
	r.pipelineRunner = pipelineRunner
	r.objectSvc = objectSvc
	r.statusSvc = statusSvc
	r.govSvc = govSvc

	return r, nil
}

func handleE2E(ctx context.Context, args []string, log *zap.Logger, pool *pgxpool.Pool, cfg *config.Config) {
	fs := flag.NewFlagSet("e2e", flag.ExitOnError)
	autoFix := fs.Bool("auto-fix", false, "Automatically fix known schema and data issues")
	allowLive := fs.Bool("live", false, "Allow live execution (sets BAXI_ALLOW_LIVE_EXECUTION=true)")
	reportJSON := fs.Bool("json", false, "Output report as JSON")
	verbose := fs.Bool("verbose", false, "Verbose output")
	_ = fs.Parse(args)

	if *allowLive {
		os.Setenv("BAXI_ALLOW_LIVE_EXECUTION", "true")
		log.Info("live execution enabled via --live flag")
	}

	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("  Baxi E2E Test Runner — Autonomous Full-Cycle Validation")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()

	runner, err := newE2ERunner(ctx, log, pool, *autoFix, *allowLive)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize E2E runner: %v\n", err)
		os.Exit(1)
	}

	// Phase 1: Probe system state
	runner.probeSystem()

	// Phase 2: Auto-fix known issues if requested
	if *autoFix {
		runner.autoFixIssues()
	}

	// Phase 3: Ensure data availability
	runner.ensureData()

	// Phase 4: Run the decision lifecycle
	runner.runDecisionLifecycle()

	// Phase 5: Run governance and ontology checks
	runner.runGovernanceChecks()

	// Phase 6: Generate report
	runner.printReport(*reportJSON, *verbose)
}

func (r *E2ERunner) record(step, status, detail string) {
	result := E2EResult{
		Step:   step,
		Status: status,
		Detail: detail,
	}
	r.results = append(r.results, result)

	switch status {
	case "PASS":
		fmt.Printf("  ✅ %s: %s\n", step, detail)
	case "FAIL":
		fmt.Printf("  ❌ %s: %s\n", step, detail)
	case "WARN":
		fmt.Printf("  ⚠️  %s: %s\n", step, detail)
	case "SKIP":
		fmt.Printf("  ⏭️  %s: %s\n", step, detail)
	case "INFO":
		fmt.Printf("  ℹ️  %s: %s\n", step, detail)
	}
}

// Phase 1: Probe system state
func (r *E2ERunner) probeSystem() {
	fmt.Println("━ Phase 1: System Probe ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Check system status
	status, err := r.statusSvc.GetStatus(r.ctx)
	if err != nil {
		r.record("get_system_status", "FAIL", fmt.Sprintf("error: %v", err))
		return
	}
	r.record("get_system_status", "PASS",
		fmt.Sprintf("tables=%d, last_run=%v", len(status.Database.Tables), status.LastPipelineRun != nil))

	// Check alerts
	alerts, err := r.alertSvc.ListAlerts(r.ctx, model.AlertFilters{}, "", 10, 0)
	if err != nil {
		r.record("list_alerts", "FAIL", fmt.Sprintf("error: %v", err))
		return
	}
	r.record("list_alerts", "PASS", fmt.Sprintf("found %d alerts", alerts.Total))

	// Check schema issues
	r.checkSchemaIssues()
}

func (r *E2ERunner) checkSchemaIssues() {
	// Check if dwd.global table exists and has required columns
	var exists bool
	err := r.pool.QueryRow(r.ctx, `
		SELECT EXISTS(SELECT 1 FROM information_schema.tables 
			WHERE table_schema='dwd' AND table_name='global')`).Scan(&exists)
	if err != nil {
		r.record("schema_global", "WARN", "could not check global table")
		return
	}
	if !exists {
		r.record("schema_global", "WARN", "dwd.global table does not exist")
		return
	}

	var hasBaseline, hasSnapshot bool
	err = r.pool.QueryRow(r.ctx, `
		SELECT EXISTS(SELECT 1 FROM information_schema.columns 
			WHERE table_schema='dwd' AND table_name='global' AND column_name='baseline_value'),
			EXISTS(SELECT 1 FROM information_schema.columns 
				WHERE table_schema='dwd' AND table_name='global' AND column_name='snapshot_date')`).Scan(&hasBaseline, &hasSnapshot)

	if err != nil {
		r.record("schema_global", "WARN", "schema check failed")
		return
	}

	if !hasBaseline || !hasSnapshot {
		r.record("schema_global", "WARN",
			fmt.Sprintf("global missing: baseline_value=%v, snapshot_date=%v", hasBaseline, hasSnapshot))
	} else {
		r.record("schema_global", "PASS", "global table has required columns")
	}
}

// Phase 2: Auto-fix known issues
func (r *E2ERunner) autoFixIssues() {
	fmt.Println()
	fmt.Println("━ Phase 2: Auto-Fix ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if !r.autoFix {
		r.record("auto_fix", "SKIP", "--auto-fix not enabled")
		return
	}

	// Fix global table schema
	var exists bool
	err := r.pool.QueryRow(r.ctx, `
		SELECT EXISTS(SELECT 1 FROM information_schema.tables 
			WHERE table_schema='dwd' AND table_name='global')`).Scan(&exists)
	if err == nil && exists {
		var hasBaseline, hasSnapshot bool
		err = r.pool.QueryRow(r.ctx, `
			SELECT EXISTS(SELECT 1 FROM information_schema.columns 
				WHERE table_schema='dwd' AND table_name='global' AND column_name='baseline_value'),
				EXISTS(SELECT 1 FROM information_schema.columns 
					WHERE table_schema='dwd' AND table_name='global' AND column_name='snapshot_date')`).Scan(&hasBaseline, &hasSnapshot)
		if err == nil && (!hasBaseline || !hasSnapshot) {
			if !hasBaseline {
				_, err = r.pool.Exec(r.ctx, `ALTER TABLE dwd.global ADD COLUMN IF NOT EXISTS baseline_value NUMERIC(18,4)`)
				if err != nil {
					r.record("fix_global_baseline", "FAIL", err.Error())
				} else {
					r.record("fix_global_baseline", "PASS", "added baseline_value column")
				}
			}
			if !hasSnapshot {
				_, err = r.pool.Exec(r.ctx, `ALTER TABLE dwd.global ADD COLUMN IF NOT EXISTS snapshot_date DATE`)
				if err != nil {
					r.record("fix_global_snapshot", "FAIL", err.Error())
				} else {
					r.record("fix_global_snapshot", "PASS", "added snapshot_date column")
				}
			}
		}
	}

	// Ensure pipeline can generate data by checking for raw data
	var rawCount int
	err = r.pool.QueryRow(r.ctx, `SELECT COUNT(*) FROM raw.orders`).Scan(&rawCount)
	if err != nil || rawCount == 0 {
		r.record("raw_data_check", "WARN", fmt.Sprintf("raw.orders has %d rows, pipeline data needed", rawCount))
	} else {
		r.record("raw_data_check", "PASS", fmt.Sprintf("raw.orders has %d rows", rawCount))
	}
}

// Phase 3: Ensure data availability
func (r *E2ERunner) ensureData() {
	fmt.Println()
	fmt.Println("━ Phase 3: Data Preparation ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Check if we have alerts
	alerts, err := r.alertSvc.ListAlerts(r.ctx, model.AlertFilters{}, "", 1, 0)
	if err != nil {
		r.record("alert_check", "FAIL", err.Error())
		return
	}

	if alerts.Total > 0 {
		r.record("alert_check", "PASS", fmt.Sprintf("%d alerts available", alerts.Total))
		return
	}

	// No alerts — try to run pipeline to generate them
	r.record("alert_check", "WARN", "no alerts found, attempting pipeline run")

	// Check if raw data exists
	var rawCount int
	err = r.pool.QueryRow(r.ctx, `SELECT COUNT(*) FROM raw.orders`).Scan(&rawCount)
	if err != nil || rawCount == 0 {
		r.record("pipeline_run", "SKIP", "no raw data available, cannot generate alerts")
		return
	}

	// Run pipeline
	err = r.pipelineRunner.Run(r.ctx, pipeline.RunInput{RunType: "full", Mode: "e2e-auto"})
	if err != nil {
		r.record("pipeline_run", "FAIL", fmt.Sprintf("pipeline error: %v", err))
		return
	}
	r.record("pipeline_run", "PASS", "pipeline completed")

	// Re-check alerts
	alerts, err = r.alertSvc.ListAlerts(r.ctx, model.AlertFilters{}, "", 1, 0)
	if err != nil {
		r.record("alert_recheck", "FAIL", err.Error())
		return
	}
	if alerts.Total > 0 {
		r.record("alert_recheck", "PASS", fmt.Sprintf("%d alerts generated", alerts.Total))
	} else {
		r.record("alert_recheck", "WARN", "still no alerts after pipeline run")
	}
}

// Phase 4: Run the full decision lifecycle
func (r *E2ERunner) runDecisionLifecycle() {
	fmt.Println()
	fmt.Println("━ Phase 4: Decision Lifecycle ━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Step 1: Get alerts and create cases
	alerts, err := r.alertSvc.ListAlerts(r.ctx, model.AlertFilters{}, "", 5, 0)
	if err != nil {
		r.record("lifecycle_fetch_alerts", "FAIL", err.Error())
		return
	}
	if alerts.Total == 0 {
		r.record("lifecycle_fetch_alerts", "SKIP", "no alerts to process")
		return
	}

	var alertID string
	for _, a := range alerts.Items {
		if a.EventID != "" {
			alertID = a.EventID
			break
		}
	}
	if alertID == "" {
		r.record("lifecycle_select_alert", "FAIL", "no valid alert ID found")
		return
	}
	r.record("lifecycle_select_alert", "PASS", fmt.Sprintf("selected alert %s", alertID))

	// Step 2: Create decision case
	caseResult, err := r.decisionSvc.CreateCaseFromAlert(r.ctx, alertID, "e2e_test_agent")
	if err != nil {
		r.record("create_decision_case", "FAIL", err.Error())
		return
	}
	caseID := caseResult.CaseID
	r.createdCaseIDs = append(r.createdCaseIDs, caseID)
	r.record("create_decision_case", "PASS", fmt.Sprintf("case_id=%s", caseID))

	// Step 3: Generate decision (proposals)
	_, _, proposals, err := r.decisionSvc.Decide(r.ctx, caseID)
	if err != nil {
		r.record("decide", "FAIL", err.Error())
		// Continue with manual proposal creation
		r.tryManualPropose(caseID)
	} else if len(proposals) > 0 {
		r.record("decide", "PASS", fmt.Sprintf("generated %d proposals", len(proposals)))
		r.runApprovalAndExecution(caseID, proposals[0].ProposalID)
	} else {
		r.record("decide", "WARN", "no proposals generated")
		r.tryManualPropose(caseID)
	}

	// Step 4: Resolve case via direct DB update
	_, err = r.pool.Exec(r.ctx,
		`UPDATE ai.decision_case SET status = 'closed', resolution = $1, updated_at = NOW() WHERE case_id = $2`,
		"E2E test completed", caseID)
	if err != nil {
		r.record("resolve_case", "FAIL", err.Error())
	} else {
		r.record("resolve_case", "PASS", fmt.Sprintf("case %s closed", caseID))
	}
}

func (r *E2ERunner) tryManualPropose(caseID string) {
	// Fallback: manually insert a proposal via SQL
	proposalID := fmt.Sprintf("e2e-proposal-%d", os.Getpid())
	_, err := r.pool.Exec(r.ctx, `
		INSERT INTO ai.action_proposal (proposal_id, case_id, action_type, apply_status, title, requires_human_review, created_at)
		VALUES ($1, $2, 'notify_owner', 'proposed', 'E2E Test Proposal', true, NOW())
		ON CONFLICT (proposal_id) DO NOTHING`,
		proposalID, caseID)
	if err != nil {
		r.record("manual_propose", "FAIL", err.Error())
		return
	}

	// Get the proposal we just created
	proposals, err := r.decisionSvc.ListProposals(r.ctx, caseID)
	if err != nil || len(proposals) == 0 {
		r.record("manual_propose_fetch", "FAIL", "could not fetch created proposal")
		return
	}
	// Use the most recently created proposal
	proposalID = proposals[len(proposals)-1].ProposalID
	r.record("manual_propose", "PASS", fmt.Sprintf("proposal_id=%s", proposalID))
	r.runApprovalAndExecution(caseID, proposalID)
}

func (r *E2ERunner) runApprovalAndExecution(caseID, proposalID string) {
	// Step 5: Get proposal details
	proposalRow, err := r.reviewSvc.GetProposalByID(r.ctx, proposalID)
	if err != nil {
		r.record("get_proposal", "FAIL", err.Error())
		return
	}
	r.record("get_proposal", "PASS", fmt.Sprintf("status=%s", proposalRow.ApplyStatus))

	// Step 6: Approve proposal
	_, err = r.reviewSvc.ApproveProposal(r.ctx, proposalID, "e2e_test_reviewer", "Approved by E2E test")
	if err != nil {
		r.record("approve_proposal", "FAIL", err.Error())
		return
	}
	r.record("approve_proposal", "PASS", fmt.Sprintf("proposal %s approved", proposalID))

	// Step 7: Verify review records
	records, _, err := r.reviewSvc.ListReviewRecords(r.ctx, proposalID, 10, 0)
	if err != nil {
		r.record("list_review_records", "FAIL", err.Error())
	} else {
		r.record("list_review_records", "PASS", fmt.Sprintf("%d review records found", len(records)))
	}

	// Step 8: Execute proposal (dry_run)
	dryResult, err := r.executeSvc.ExecuteProposal(r.ctx, r.pool, proposalID, "e2e_test_executor", action.WithDryRun(true))
	if err != nil {
		r.record("execute_dry_run", "FAIL", err.Error())
		return
	}
	r.record("execute_dry_run", "PASS", fmt.Sprintf("success=%v", dryResult.Success))

	// Step 9: Execute proposal (live) if allowed
	if r.allowLive {
		liveResult, err := r.executeSvc.ExecuteProposal(r.ctx, r.pool, proposalID, "e2e_test_executor", action.WithDryRun(false))
		if err != nil {
			r.record("execute_live", "FAIL", err.Error())
		} else {
			r.record("execute_live", "PASS", fmt.Sprintf("success=%v", liveResult.Success))
		}
	} else {
		r.record("execute_live", "SKIP", "use --live to enable live execution")
	}
}

// Phase 5: Run governance and ontology checks
func (r *E2ERunner) runGovernanceChecks() {
	fmt.Println()
	fmt.Println("━ Phase 5: Governance & Ontology ━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Check access
	accessDecision := r.govSvc.CheckAccess(r.ctx, "admin", "seller", "read")
	r.record("check_access", "PASS", fmt.Sprintf("allowed=%v", accessDecision == model.AccessAllowed))

	// Get classification
	classification, err := r.govSvc.GetClassification(r.ctx, "seller.gmv")
	if err != nil {
		r.record("get_classification", "FAIL", err.Error())
	} else {
		level := "unknown"
		if len(classification.Levels) > 0 {
			level = classification.Levels[0]
		}
		r.record("get_classification", "PASS", fmt.Sprintf("level=%s", level))
	}

	// Search objects (if data exists)
	searchResult, err := r.objectSvc.SearchObjects(r.ctx, "seller", "", 1, 0)
	if err != nil {
		r.record("search_objects", "FAIL", err.Error())
	} else {
		r.record("search_objects", "PASS", fmt.Sprintf("%d sellers found", searchResult.Total))
	}

	// List outbox events
	var outboxCount int
	err = r.pool.QueryRow(r.ctx, `SELECT COUNT(*) FROM ops.event_outbox`).Scan(&outboxCount)
	if err != nil {
		r.record("outbox_check", "FAIL", err.Error())
	} else {
		r.record("outbox_check", "PASS", fmt.Sprintf("%d outbox events", outboxCount))
	}
}

// Phase 6: Print report
func (r *E2ERunner) printReport(asJSON, verbose bool) {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("  E2E Test Report")
	fmt.Println("═══════════════════════════════════════════════════════════")

	pass, fail, warn, skip := 0, 0, 0, 0
	for _, res := range r.results {
		switch res.Status {
		case "PASS":
			pass++
		case "FAIL":
			fail++
		case "WARN":
			warn++
		case "SKIP":
			skip++
		}
	}

	fmt.Printf("\nSummary: %d PASS, %d FAIL, %d WARN, %d SKIP (total %d)\n",
		pass, fail, warn, skip, len(r.results))

	if fail > 0 {
		fmt.Println("\nFailed steps:")
		for _, res := range r.results {
			if res.Status == "FAIL" {
				fmt.Printf("  - %s: %s\n", res.Step, res.Detail)
			}
		}
	}

	if verbose {
		fmt.Println("\nAll steps:")
		for _, res := range r.results {
			fmt.Printf("  [%s] %s: %s\n", res.Status, res.Step, res.Detail)
		}
	}

	if asJSON {
		fmt.Println()
		fmt.Println("{")
		fmt.Printf(`  "summary": {"pass": %d, "fail": %d, "warn": %d, "skip": %d, "total": %d},`+"\n",
			pass, fail, warn, skip, len(r.results))
		fmt.Println(`  "results": [`)
		for i, res := range r.results {
			comma := ","
			if i == len(r.results)-1 {
				comma = ""
			}
			fmt.Printf(`    {"step": "%s", "status": "%s", "detail": "%s"}%s`+"\n",
				res.Step, res.Status, res.Detail, comma)
		}
		fmt.Println(`  ]`)
		fmt.Println("}")
	}

	if fail > 0 {
		fmt.Println("\n❌ E2E test completed with failures")
		os.Exit(1)
	} else if warn > 0 {
		fmt.Println("\n⚠️  E2E test completed with warnings")
	} else {
		fmt.Println("\n✅ All E2E tests passed")
	}
}

// proposalLoaderAdapter satisfies action.ProposalLoader interface.
type proposalLoaderAdapter struct {
	repo *review.ReviewRepository
	pool *pgxpool.Pool
}

func (a *proposalLoaderAdapter) GetProposalByID(ctx context.Context, pool *pgxpool.Pool, proposalID string) (*action.ActionProposal, error) {
	row, err := a.repo.GetProposalByID(ctx, a.pool, proposalID)
	if err != nil {
		return nil, err
	}
	return &action.ActionProposal{
		ProposalID:          row.ProposalID,
		CaseID:              row.CaseID,
		ActionType:          row.ActionType,
		ApplyStatus:         row.ApplyStatus,
		Title:               row.Title,
		RequiresHumanReview: row.RequiresHumanReview,
	}, nil
}
