package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"baxi/internal/action"
	"baxi/internal/config"
	"baxi/internal/db"
	"baxi/internal/decision"
	"baxi/internal/governance"
	"baxi/internal/llm"
	"baxi/internal/logger"
	mcp "baxi/internal/mcp"
	"baxi/internal/model"
	"baxi/internal/ontology"
	"baxi/internal/pipeline"
	"baxi/internal/pipeline/steps"
	"baxi/internal/repository"
	"baxi/internal/repository/common"
	ontologyRepo "baxi/internal/repository/ontology"
	"baxi/internal/review"
	"baxi/internal/service"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	cfg, err := config.Load()
	if err != nil {
		os.Stderr.WriteString("failed to load config: " + err.Error() + "\n")
		os.Exit(1)
	}

	zapLog, err := logger.New(cfg.LogLevel)
	if err != nil {
		os.Stderr.WriteString("failed to initialize logger: " + err.Error() + "\n")
		os.Exit(1)
	}

	pool, err := db.NewPool(ctx, cfg.DatabaseURL, zapLog)
	if err != nil {
		zapLog.Fatal("failed to connect to database", zap.Error(err))
	}
	defer pool.Close()

	// Wire decision services (same pattern as handler_factories.go)
	decisionRepo := repository.NewDecisionRepository()
	alertRepo := repository.NewAlertRepository()
	caseSvc := decision.NewCaseService(decisionRepo, alertRepo, pool.Pool)

	ontologyRepo := repository.NewOntologyRepo()
	objectSvc := ontology.NewObjectQueryService(ontologyRepo, pool.Pool)
	govRepo := repository.NewGovernanceRepository()
	classSvc := governance.NewClassificationService(pool.Pool, govRepo)
	reg, err := action.NewActionRegistry("")
	if err != nil {
		zapLog.Warn("failed to load action registry, using empty fallback", zap.Error(err))
		reg = action.NewEmptyRegistry()
	}
	v1Builder := decision.NewContextBuilder(decisionRepo, objectSvc, classSvc, pool.Pool, action.NewActionTypeProviderAdapter(reg))

	configDir := os.Getenv("BAXI_CONFIG_DIR")
	if configDir == "" {
		configDir = "config"
	}

	// Build v2 builder (ontology-aware).
	var v2Builder *decision.ContextBuilderV2
	objRegistry, regErr := ontology.NewObjectRegistry(ctx, nil, pool.Pool, filepath.Join(configDir, "aip_object_schema.yml"))
	if regErr != nil {
		zapLog.Warn("failed to load object registry for v2 builder, v2/v3 unavailable", zap.Error(regErr))
	} else {
		ontologyAwareRepo := ontology.NewOntologyAwareAdapter(ontologyRepo, objRegistry)
		markingSvc := governance.NewMarkingAdapter(classSvc, objRegistry)
		govRepoLocal := repository.NewGovernanceRepository()
		lineageSvc := governance.NewLineageService(pool.Pool, govRepoLocal)
		eventRepo := decision.NewPgxLineageEventRepository()
		lineageAdapter := decision.NewDecisionLineageAdapter(lineageSvc, decisionRepo, eventRepo, pool.Pool)
		v2Builder = decision.NewContextBuilderV2(decisionRepo, ontologyAwareRepo, markingSvc, lineageAdapter, pool.Pool, action.NewActionTypeProviderAdapter(reg))
	}

	var ctxBuilder decision.ObjectContextBuilder
	if v2Builder != nil {
		switcher := decision.NewSwitchableContextBuilder(v1Builder, v2Builder, nil)
		// Only build v3 if we have the registry (needed for link traversal).
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
	engine := decision.NewDecisionEngine(decisionProvider, decisionRepo, pool.Pool, llm.NewDBAuditLogger(pool.Pool))
	proposalSvc := action.NewProposalService(decisionRepo, decisionRepo, reg, pool.Pool)

	decisionSvc := service.NewDecisionService(caseSvc, ctxBuilder, engine, proposalSvc, pool.Pool)
	alertSvc := service.NewAlertService(alertRepo, pool.Pool)
	govSvc := &governanceServiceAdapter{
		svc: service.NewGovernanceService(govRepo, pool.Pool),
	}

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
	pipelineRunner := &pipeline.Runner{DB: pool.Pool, Steps: pipelineSteps, Log: zapLog}
	pipelineSvc := &pipelineRunService{runner: pipelineRunner}

	// Wire review service
	reviewRepo := review.NewReviewRepository()
	reviewSvc := review.NewReviewService(reviewRepo, pool.Pool)

	// Wire outbox and pipeline info services
	outboxSvc := &outboxServiceAdapter{pool: pool.Pool}
	pipelineInfoSvc := &pipelineInfoAdapter{pool: pool.Pool}

	// Wire real ApplyService for execute_proposal
	proposalLoader := &proposalLoaderAdapter{repo: reviewRepo}
	executeSvc := action.NewApplyService(reg, nil, proposalLoader, nil, nil, pool.Pool)
	statusSvc := &statusServiceAdapter{pool: pool.Pool}
	searchSvc := &searchServiceAdapter{svc: objectSvc}

	// Wire ontology service for ontology MCP tools.
	// objRegistry may already be loaded above for the v2/v3 builder.
	if objRegistry == nil {
		objRegistry, regErr = ontology.NewObjectRegistry(ctx, nil, pool.Pool, filepath.Join(configDir, "aip_object_schema.yml"))
		if regErr != nil {
			zapLog.Warn("failed to load object registry, ontology tools will be unavailable", zap.Error(regErr))
		}
	}

	// Set up v2 QueryCompiler on the repository if v2 objects are available.
	if objRegistry != nil {
		v2Objects := objRegistry.AllObjectsV2()
		if len(v2Objects) > 0 {
			qc := ontology.NewQueryCompiler(v2Objects, 10000)
			ontologyRepo.SetV2Compiler(&v2CompilerAdapter{compiler: qc})
			zapLog.Info("v2 QueryCompiler enabled", zap.Int("object_types", len(v2Objects)))
		}
	}

	ontologySvc := &ontologyServiceAdapter{
		registry:  objRegistry,
		querySvc:  objectSvc,
		ontRepo:   ontologyRepo,
		pool:      pool.Pool,
		actionReg: reg,
		applySvc:  executeSvc,
		mcpRole:   os.Getenv("BAXI_MCP_ROLE"),
		log:       zapLog,
	}

	// Create adapters to satisfy extended MCP interfaces
	decisionSvcAdapter := &decisionServiceAdapter{svc: decisionSvc, pool: pool.Pool}
	reviewSvcAdapter := &reviewServiceAdapter{svc: reviewSvc}

	// Wire schema service for action schema MCP tools
	schemaCatalog := action.NewActionSchemaCatalog(reg)
	schemaSvc := &schemaServiceAdapter{catalog: schemaCatalog}

	// Wire sandbox service for proposal sandbox MCP tools
	sandboxService := review.NewSandboxService(pool.Pool)
	sandboxSvc := &sandboxServiceAdapter{svc: sandboxService}

	var buildContextSvc mcp.BuildContextService
	if objRegistry != nil {
		v2Objects := objRegistry.AllObjectsV2()
		if len(v2Objects) > 0 {
			recipes, recipeErr := ontology.LoadContextRecipes(filepath.Join(configDir, "context_recipes.yml"))
			if recipeErr != nil {
				zapLog.Warn("failed to load context recipes, build_context will be unavailable", zap.Error(recipeErr))
			} else {
				metricDefs, metricErr := ontology.LoadMetricDefinitions(filepath.Join(configDir, "metric_definitions.yml"))
				if metricErr != nil {
					zapLog.Warn("failed to load metric definitions, build_context will be unavailable", zap.Error(metricErr))
				} else {
					metricResolver := ontology.NewMetricResolver(metricDefs)
					metricQuery := ontology.NewMetricQueryResolver(metricResolver, pool.Pool)
					linkExec := newLinkExecutor(common.NewPoolProvider(pool.Pool))
					qc := ontology.NewQueryCompiler(v2Objects, 10000)
					buildContextSvc = decision.NewRecipeContextBuilder(
						decisionRepo, qc, metricQuery, linkExec, pool.Pool,
						action.NewActionTypeProviderAdapter(reg), recipes,
					)
					ontologySvc.linkResolver = ontology.NewLinkResolver(v2Objects)
					zapLog.Info("RecipeContextBuilder wired", zap.Int("recipes", len(recipes)), zap.Int("metrics", len(metricDefs)))
				}
			}
		}
	}
	if buildContextSvc == nil {
		zapLog.Warn("build_context service is not available (recipe loading failed or no v2 objects)")
	}

	mcpSrv, err := mcp.NewServer(
		decisionSvcAdapter, engine, ctxBuilder, buildContextSvc, proposalSvc, alertSvc, govSvc, pipelineSvc,
		reviewSvcAdapter, outboxSvc, pipelineInfoSvc,
		executeSvc, pool.Pool,
		statusSvc, searchSvc, ontologySvc,
		schemaSvc, sandboxSvc,
	)
	if err != nil {
		zapLog.Fatal("failed to create MCP server", zap.Error(err))
	}

	zapLog.Info("baxi-mcp server starting (stdio)")
	go func() {
		if err := mcpSrv.Run(); err != nil {
			zapLog.Fatal("MCP server error", zap.Error(err))
		}
	}()

	<-sigCh
	zapLog.Info("shutting down")
}

// decisionServiceAdapter wraps service.DecisionService to implement mcp.DecisionService,
// including the additional Decide and ResolveCase methods.
type decisionServiceAdapter struct {
	svc  *service.DecisionService
	pool *pgxpool.Pool
}

func (a *decisionServiceAdapter) CreateCaseFromAlert(ctx context.Context, alertID, createdBy string) (*decision.DecisionCase, error) {
	return a.svc.CreateCaseFromAlert(ctx, alertID, createdBy)
}

func (a *decisionServiceAdapter) GetCase(ctx context.Context, caseID string) (*decision.DecisionCase, error) {
	return a.svc.GetCase(ctx, caseID)
}

func (a *decisionServiceAdapter) ListCases(ctx context.Context, filter decision.CaseFilter) (*decision.CaseList, error) {
	return a.svc.ListCases(ctx, filter)
}

func (a *decisionServiceAdapter) Decide(ctx context.Context, caseID string) ([]action.ActionProposal, error) {
	_, _, proposals, err := a.svc.Decide(ctx, caseID)
	if err != nil {
		return nil, err
	}
	return proposals, nil
}

func (a *decisionServiceAdapter) ResolveCase(ctx context.Context, caseID, resolution, comment string) error {
	result, err := a.pool.Exec(ctx, `
		UPDATE ai.decision_case
		SET status = 'resolved',
		    resolved_at = NOW(),
		    resolution = $2,
		    case_resolution_comment = $3
		WHERE case_id = $1
	`, caseID, resolution, comment)
	if err != nil {
		return fmt.Errorf("resolve case %s: %w", caseID, err)
	}
	rows := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("case %s not found", caseID)
	}
	return nil
}

// reviewServiceAdapter wraps review.ReviewService to implement mcp.ReviewService,
// including the additional CancelProposal and GetProposalByID methods.
type reviewServiceAdapter struct {
	svc *review.ReviewService
}

func (a *reviewServiceAdapter) ApproveProposal(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error) {
	return a.svc.ApproveProposal(ctx, proposalID, reviewerID, feedback)
}

func (a *reviewServiceAdapter) RejectProposal(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error) {
	return a.svc.RejectProposal(ctx, proposalID, reviewerID, feedback)
}

func (a *reviewServiceAdapter) CancelProposal(ctx context.Context, proposalID, reviewerID, reason string) error {
	_, err := a.svc.CancelProposal(ctx, proposalID, reviewerID, reason)
	if err != nil {
		return err
	}
	return nil
}

func (a *reviewServiceAdapter) ListReviewRecords(ctx context.Context, proposalID string, limit, offset int) ([]review.ReviewRecord, int, error) {
	return a.svc.ListReviewRecords(ctx, proposalID, limit, offset)
}

func (a *reviewServiceAdapter) GetProposalByID(ctx context.Context, proposalID string) (*action.ActionProposal, error) {
	row, err := a.svc.GetProposalByID(ctx, proposalID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}

	p := &action.ActionProposal{
		ProposalID:          row.ProposalID,
		CaseID:              row.CaseID,
		ActionType:          row.ActionType,
		Title:               row.Title,
		ApplyStatus:         row.ApplyStatus,
		CreatedAt:           row.CreatedAt,
		RequiresHumanReview: row.RequiresHumanReview,
	}
	if row.DecisionID != nil {
		p.DecisionID = *row.DecisionID
	}
	if row.Description != nil {
		p.Description = *row.Description
	}
	if row.RiskLevel != nil {
		p.RiskLevel = *row.RiskLevel
	}
	if row.Payload != nil {
		var payload map[string]interface{}
		if err := json.Unmarshal(*row.Payload, &payload); err == nil {
			p.Payload = payload
		}
	}
	return p, nil
}

type governanceServiceAdapter struct {
	svc *service.GovernanceService
}

func (a *governanceServiceAdapter) CheckAccess(ctx context.Context, role, objectType, action string) (*model.AccessDecision, error) {
	result := a.svc.CheckAccess(ctx, role, objectType, action)
	return &result, nil
}

func (a *governanceServiceAdapter) GetClassification(ctx context.Context, fieldPath string) (*model.ClassificationResponse, error) {
	return a.svc.GetClassification(ctx, fieldPath)
}

type pipelineRunService struct {
	runner *pipeline.Runner
}

func (s *pipelineRunService) Run(ctx context.Context, config string) (string, error) {
	input := pipeline.RunInput{
		RunType: "full",
		Mode:    "mcp",
		DataDir: "./data/raw",
	}
	if config != "" {
		input.RunType = config
	}
	err := s.runner.Run(ctx, input)
	if err != nil {
		return "", err
	}
	return "pipeline-run-" + input.RunType, nil
}

// outboxServiceAdapter queries ops.outbox_event for the MCP tools.
type outboxServiceAdapter struct {
	pool *pgxpool.Pool
}

func (a *outboxServiceAdapter) ListOutboxEvents(ctx context.Context, status string, limit, offset int) ([]model.OutboxEvent, int, error) {
	var total int
	err := a.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM ops.outbox_event
		WHERE ($1 = '' OR status = $1)
	`, status).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count outbox events: %w", err)
	}

	rows, err := a.pool.Query(ctx, `
		SELECT event_id, source_type, event_type, status, created_at, dispatch_attempts
		FROM ops.outbox_event
		WHERE ($1 = '' OR status = $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query outbox events: %w", err)
	}
	defer rows.Close()

	var events []model.OutboxEvent
	for rows.Next() {
		var e model.OutboxEvent
		if err := rows.Scan(&e.OutboxID, &e.SourceType, &e.EventType, &e.Status, &e.CreatedAt, &e.DispatchAttempts); err != nil {
			return nil, 0, fmt.Errorf("scan outbox event: %w", err)
		}
		events = append(events, e)
	}
	return events, total, nil
}

// pipelineInfoAdapter queries audit.pipeline_run for the MCP tools.
type pipelineInfoAdapter struct {
	pool *pgxpool.Pool
}

func (a *pipelineInfoAdapter) GetLastRunStatus(ctx context.Context) (*model.PipelineRun, error) {
	var r model.PipelineRun
	var startedAt time.Time
	var finishedAt *time.Time
	var errMsg *string

	err := a.pool.QueryRow(ctx, `
		SELECT run_id, run_type, mode, status, started_at, finished_at, input_count, output_count, error_message
		FROM audit.pipeline_run
		ORDER BY started_at DESC
		LIMIT 1
	`).Scan(&r.RunID, &r.RunType, &r.Mode, &r.Status, &startedAt, &finishedAt, &r.InputCount, &r.OutputCount, &errMsg)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get last run: %w", err)
	}

	r.StartedAt = startedAt.Format(time.RFC3339)
	if finishedAt != nil {
		s := finishedAt.Format(time.RFC3339)
		r.FinishedAt = &s
	}
	if errMsg != nil && *errMsg != "" {
		r.ErrorMessage = errMsg
	}
	return &r, nil
}

func (a *pipelineInfoAdapter) ListRuns(ctx context.Context, limit int) ([]model.PipelineRun, error) {
	rows, err := a.pool.Query(ctx, `
		SELECT run_id, run_type, mode, status, started_at, finished_at, input_count, output_count, error_message
		FROM audit.pipeline_run
		ORDER BY started_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list runs: %w", err)
	}
	defer rows.Close()

	var runs []model.PipelineRun
	for rows.Next() {
		var r model.PipelineRun
		var startedAt time.Time
		var finishedAt *time.Time
		var errMsg *string

		if err := rows.Scan(&r.RunID, &r.RunType, &r.Mode, &r.Status, &startedAt, &finishedAt, &r.InputCount, &r.OutputCount, &errMsg); err != nil {
			return nil, fmt.Errorf("scan run: %w", err)
		}

		r.StartedAt = startedAt.Format(time.RFC3339)
		if finishedAt != nil {
			s := finishedAt.Format(time.RFC3339)
			r.FinishedAt = &s
		}
		if errMsg != nil && *errMsg != "" {
			r.ErrorMessage = errMsg
		}
		runs = append(runs, r)
	}
	return runs, nil
}

// proposalLoaderAdapter wraps *review.ReviewRepository to implement action.ProposalLoader.
type proposalLoaderAdapter struct {
	repo *review.ReviewRepository
}

func (a *proposalLoaderAdapter) GetProposalByID(ctx context.Context, pool *pgxpool.Pool, proposalID string) (*action.ActionProposal, error) {
	row, err := a.repo.GetProposalByID(ctx, pool, proposalID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}

	p := &action.ActionProposal{
		ProposalID:          row.ProposalID,
		CaseID:              row.CaseID,
		ActionType:          row.ActionType,
		Title:               row.Title,
		ApplyStatus:         row.ApplyStatus,
		CreatedAt:           row.CreatedAt,
		RequiresHumanReview: row.RequiresHumanReview,
	}

	if row.DecisionID != nil {
		p.DecisionID = *row.DecisionID
	}
	if row.Description != nil {
		p.Description = *row.Description
	}
	if row.RiskLevel != nil {
		p.RiskLevel = *row.RiskLevel
	} else {
		p.RiskLevel = "medium"
	}
	if row.Payload != nil {
		var payload map[string]interface{}
		if err := json.Unmarshal(*row.Payload, &payload); err == nil {
			p.Payload = payload
		}
	}

	return p, nil
}

// statusServiceAdapter queries the database for system status.
type statusServiceAdapter struct {
	pool *pgxpool.Pool
}

func (a *statusServiceAdapter) GetStatus(ctx context.Context) (*model.SystemStatus, error) {
	status := &model.SystemStatus{
		RecentErrors: []string{},
		TableCounts:  []model.TableCount{},
	}

	// 1. AlertCount from ops.metric_alert
	_ = a.pool.QueryRow(ctx, `SELECT COUNT(*) FROM ops.metric_alert`).Scan(&status.AlertCount)

	// 2. PipelineRun from audit.pipeline_run (same pattern as pipelineInfoAdapter)
	var r model.PipelineRun
	var startedAt time.Time
	var finishedAt *time.Time
	var errMsg *string

	err := a.pool.QueryRow(ctx, `
		SELECT run_id, run_type, mode, status, started_at, finished_at, input_count, output_count, error_message
		FROM audit.pipeline_run
		ORDER BY started_at DESC
		LIMIT 1
	`).Scan(&r.RunID, &r.RunType, &r.Mode, &r.Status, &startedAt, &finishedAt, &r.InputCount, &r.OutputCount, &errMsg)
	if err == nil {
		r.StartedAt = startedAt.Format(time.RFC3339)
		if finishedAt != nil {
			s := finishedAt.Format(time.RFC3339)
			r.FinishedAt = &s
		}
		if errMsg != nil && *errMsg != "" {
			r.ErrorMessage = errMsg
		}
		status.PipelineRun = &r
	}

	// 3. TableCounts for key tables
	rows, err := a.pool.Query(ctx, `
		SELECT table_name, row_count FROM (
			SELECT 'raw.orders' as table_name, (SELECT COUNT(*) FROM raw.orders) as row_count
			UNION ALL SELECT 'raw.sellers', (SELECT COUNT(*) FROM raw.sellers)
			UNION ALL SELECT 'raw.products', (SELECT COUNT(*) FROM raw.products)
			UNION ALL SELECT 'dwd.dwd_order_level', (SELECT COUNT(*) FROM dwd.dwd_order_level)
			UNION ALL SELECT 'dwd.dwd_item_level', (SELECT COUNT(*) FROM dwd.dwd_item_level)
			UNION ALL SELECT 'metric.metric_daily', (SELECT COUNT(*) FROM metric.metric_daily)
			UNION ALL SELECT 'ops.metric_alert', (SELECT COUNT(*) FROM ops.metric_alert)
			UNION ALL SELECT 'ai.decision_case', (SELECT COUNT(*) FROM ai.decision_case)
			UNION ALL SELECT 'ai.action_proposal', (SELECT COUNT(*) FROM ai.action_proposal)
		) t
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var tc model.TableCount
			if err := rows.Scan(&tc.TableName, &tc.RowCount); err == nil {
				status.TableCounts = append(status.TableCounts, tc)
			}
		}
	}

	// 4. RecentErrors from audit.audit_log
	errRows, err := a.pool.Query(ctx, `
		SELECT action || ': ' || COALESCE(resource_id, '') || ' | ' || COALESCE(metadata::text, '')
		FROM audit.audit_log
		ORDER BY created_at DESC
		LIMIT 10
	`)
	if err == nil {
		defer errRows.Close()
		for errRows.Next() {
			var msg string
			if err := errRows.Scan(&msg); err == nil {
				status.RecentErrors = append(status.RecentErrors, msg)
			}
		}
	}

	return status, nil
}

// searchServiceAdapter wraps ontology.ObjectQueryService for MCP object search.
type searchServiceAdapter struct {
	svc *ontology.ObjectQueryService
}

func (a *searchServiceAdapter) SearchObjects(ctx context.Context, objectType, query string, limit, offset int) (*model.SearchResult, error) {
	if a.svc == nil {
		return &model.SearchResult{Items: []map[string]interface{}{}, Total: 0}, nil
	}
	result, err := a.svc.SearchObjects(ctx, objectType, query, limit, offset)
	if err != nil {
		return &model.SearchResult{Items: []map[string]interface{}{}, Total: 0}, nil
	}
	return result, nil
}

// ontologyServiceAdapter wraps ontology and action services for MCP ontology tools.
type ontologyServiceAdapter struct {
	registry     *ontology.ObjectRegistry
	querySvc     *ontology.ObjectQueryService
	ontRepo      *repository.OntologyRepo
	pool         *pgxpool.Pool
	actionReg    *action.ActionRegistry
	applySvc     *action.ApplyService
	linkResolver *ontology.LinkResolver
	mcpRole      string // role used for allowed_by authorization checks
	log          *zap.Logger
}

func (a *ontologyServiceAdapter) DescribeOntology(ctx context.Context) (*mcp.OntologyDescriptor, error) {
	if a.registry == nil {
		a.log.Warn("DescribeOntology: object registry is nil, returning empty descriptor")
		return &mcp.OntologyDescriptor{ObjectTypes: []mcp.ObjectTypeDescriptor{}}, nil
	}

	a.log.Warn("DescribeOntology: using v1 object registry; v2 object types are not included in this path")

	names := a.registry.ListObjectTypes()
	desc := &mcp.OntologyDescriptor{
		ObjectTypes: make([]mcp.ObjectTypeDescriptor, 0, len(names)),
	}
	for _, name := range names {
		ot, err := a.registry.GetObjectType(name)
		if err != nil {
			continue
		}

		otDesc := mcp.ObjectTypeDescriptor{
			Name:           ot.Name,
			DisplayName:    ot.DisplayName,
			Grain:          ot.Grain,
			AllowedActions: ot.AllowedActions,
			LLMAccess: mcp.LLMAccessDescriptor{
				CanRead:  ot.LLMAccess.CanRead,
				CanWrite: ot.LLMAccess.CanWrite,
				ReadOnly: ot.LLMAccess.ReadOnly,
			},
		}

		for _, prop := range ot.Properties {
			if !prop.LLMReadable {
				continue
			}
			otDesc.Properties = append(otDesc.Properties, mcp.PropertyDescriptor{
				Name:        prop.Name,
				Type:        prop.Type,
				Sensitivity: prop.Sensitivity,
				LLMReadable: prop.LLMReadable,
				IsPK:        prop.IsPK,
			})
		}

		for _, link := range ot.Links {
			otDesc.Links = append(otDesc.Links, mcp.LinkDescriptor{
				Name:       link.Name,
				TargetType: link.TargetType,
				Via:        link.Via,
			})
		}

		if otDesc.Properties == nil {
			otDesc.Properties = []mcp.PropertyDescriptor{}
		}
		if otDesc.Links == nil {
			otDesc.Links = []mcp.LinkDescriptor{}
		}

		desc.ObjectTypes = append(desc.ObjectTypes, otDesc)
	}
	return desc, nil
}

func (a *ontologyServiceAdapter) GetObject(ctx context.Context, objectType, objectID string) (*mcp.ObjectContext, error) {
	if a.querySvc == nil {
		return nil, fmt.Errorf("ontology query service is not available")
	}

	obj, err := a.querySvc.BuildObjectContext(ctx, objectType, objectID)
	if err != nil {
		return nil, fmt.Errorf("get object %s %s: %w", objectType, objectID, err)
	}

	return &mcp.ObjectContext{
		ObjectType: obj.ObjectType,
		ObjectID:   obj.ObjectID,
		Properties: obj.Properties,
	}, nil
}

func (a *ontologyServiceAdapter) GetObjectMetrics(ctx context.Context, objectType, objectID string) (map[string]float64, error) {
	if a.ontRepo == nil || a.pool == nil {
		return nil, fmt.Errorf("ontology repository is not available")
	}
	metrics, err := a.ontRepo.GetObjectMetrics(ctx, a.pool, objectType, objectID)
	if err != nil {
		return nil, fmt.Errorf("get metrics for %s %s: %w", objectType, objectID, err)
	}
	return metrics.Metrics, nil
}

func (a *ontologyServiceAdapter) GetLinkedObjects(ctx context.Context, objectType, objectID, linkName string, maxDepth int) (*mcp.LinkedObjectsResult, error) {
	if a.registry == nil || a.querySvc == nil {
		return nil, fmt.Errorf("ontology services are not available")
	}

	if a.linkResolver != nil {
		v2Result, err := a.getLinkedObjectsV2(ctx, objectType, objectID, linkName, maxDepth)
		if err == nil {
			return v2Result, nil
		}
		a.log.Warn("v2 link resolution failed, falling back to v1",
			zap.String("object_type", objectType),
			zap.String("link_name", linkName),
			zap.Error(err))
	} else {
		a.log.Info("v2 link resolver not available, using v1 link resolution",
			zap.String("object_type", objectType),
			zap.String("link_name", linkName))
	}

	return a.getLinkedObjectsV1(ctx, objectType, objectID, linkName, maxDepth)
}

func (a *ontologyServiceAdapter) getLinkedObjectsV2(ctx context.Context, objectType, objectID, linkName string, maxDepth int) (*mcp.LinkedObjectsResult, error) {
	source := ontology.ObjectRef{ObjectType: objectType, ObjectID: objectID}
	opts := ontology.LinkOptions{MaxDepth: maxDepth}

	_, err := a.linkResolver.GetLinkedObjects(ctx, source, linkName, opts)
	if err != nil {
		return nil, err
	}

	plans, err := a.linkResolver.CompileAllLinks(ctx, source, opts)
	if err != nil {
		return nil, fmt.Errorf("compile links: %w", err)
	}

	var plan *ontology.CompiledLink
	for _, p := range plans {
		if p.LinkName == linkName {
			plan = p
			break
		}
	}
	if plan == nil {
		return nil, fmt.Errorf("link %q not found in compiled plans", linkName)
	}

	rows, err := a.pool.Query(ctx, plan.SQL, plan.Args...)
	if err != nil {
		return nil, fmt.Errorf("execute link query: %w", err)
	}
	defer rows.Close()

	objects := make([]mcp.ObjectContext, 0)
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			continue
		}
		props := make(map[string]interface{})
		for i, col := range plan.Columns {
			if i < len(values) {
				props[col] = values[i]
			}
		}
		objID := objectID
		if len(plan.Columns) > 0 {
			objID = fmt.Sprintf("%v", props[plan.Columns[0]])
		}
		objects = append(objects, mcp.ObjectContext{
			ObjectType: plan.TargetType,
			ObjectID:   objID,
			Properties: props,
		})
	}

	return &mcp.LinkedObjectsResult{
		ObjectType: objectType,
		ObjectID:   objectID,
		Links: []mcp.LinkResult{
			{
				LinkName:   linkName,
				TargetType: plan.TargetType,
				Objects:    objects,
			},
		},
	}, nil
}

func (a *ontologyServiceAdapter) getLinkedObjectsV1(ctx context.Context, objectType, objectID, linkName string, maxDepth int) (*mcp.LinkedObjectsResult, error) {
	links, err := a.registry.GetLinks(objectType)
	if err != nil {
		return nil, fmt.Errorf("get links for %s: %w", objectType, err)
	}

	if len(links) == 0 {
		return &mcp.LinkedObjectsResult{
			ObjectType: objectType,
			ObjectID:   objectID,
			Links:      []mcp.LinkResult{},
		}, nil
	}

	if linkName != "" {
		filtered := make([]ontology.ObjectLink, 0)
		for _, l := range links {
			if l.Name == linkName {
				filtered = append(filtered, l)
				break
			}
		}
		if len(filtered) == 0 {
			return nil, fmt.Errorf("link %q not found for object type %q", linkName, objectType)
		}
		links = filtered
	}

	result := &mcp.LinkedObjectsResult{
		ObjectType: objectType,
		ObjectID:   objectID,
		Links:      make([]mcp.LinkResult, 0, len(links)),
	}

	// Get the source object to extract Via field values from its properties.
	sourceObj, err := a.querySvc.BuildObjectContext(ctx, objectType, objectID)
	if err != nil {
		return nil, fmt.Errorf("get source object: %w", err)
	}

	for _, link := range links {
		linkResult := mcp.LinkResult{
			LinkName:   link.Name,
			TargetType: link.TargetType,
			Objects:    make([]mcp.ObjectContext, 0),
		}

		if viaVal, ok := sourceObj.Properties[link.Via]; ok && viaVal != nil {
			viaStr := fmt.Sprintf("%v", viaVal)
			if viaStr != "" {
				targetObj, err := a.querySvc.BuildObjectContext(ctx, link.TargetType, viaStr)
				if err == nil {
					linkResult.Objects = append(linkResult.Objects, mcp.ObjectContext{
						ObjectType: targetObj.ObjectType,
						ObjectID:   targetObj.ObjectID,
						Properties: targetObj.Properties,
					})
				}
			}
		}

		result.Links = append(result.Links, linkResult)
	}

	return result, nil
}

// schemaServiceAdapter wraps action.ActionSchemaCatalog to implement mcp.ActionSchemaService.
type schemaServiceAdapter struct {
	catalog *action.ActionSchemaCatalog
}

func (a *schemaServiceAdapter) ListActionSchemas(ctx context.Context) ([]mcp.ActionDefinition, error) {
	defs, err := a.catalog.ListActionSchemas()
	if err != nil {
		return nil, err
	}
	items := make([]mcp.ActionDefinition, 0, len(defs))
	for _, d := range defs {
		items = append(items, mcp.ActionDefinition{
			Name:          d.Name,
			Description:   d.Description,
			RiskLevel:     d.RiskLevel,
			PayloadSchema: d.PayloadSchema,
			AllowedBy:     d.AllowedBy,
			Adapter:       d.Adapter,
		})
	}
	return items, nil
}

func (a *schemaServiceAdapter) GetActionSchema(ctx context.Context, actionType string) (*mcp.ActionDefinition, error) {
	def, err := a.catalog.GetActionSchema(actionType)
	if err != nil {
		return nil, err
	}
	if def == nil {
		return nil, nil
	}
	return &mcp.ActionDefinition{
		Name:          def.Name,
		Description:   def.Description,
		RiskLevel:     def.RiskLevel,
		PayloadSchema: def.PayloadSchema,
		AllowedBy:     def.AllowedBy,
		Adapter:       def.Adapter,
	}, nil
}

// v2CompilerAdapter wraps ontology.QueryCompiler to implement
// ontologyRepo.V2QueryCompiler (avoids circular imports).
type v2CompilerAdapter struct {
	compiler *ontology.QueryCompiler
}

func (a *v2CompilerAdapter) CompileGetObject(objectType, objectID string) (*ontologyRepo.V2CompiledQuery, error) {
	result, err := a.compiler.CompileGetObject(objectType, objectID)
	if err != nil {
		return nil, err
	}
	return &ontologyRepo.V2CompiledQuery{
		SQL:        result.SQL,
		CountSQL:   result.CountSQL,
		Args:       result.Args,
		Columns:    result.Columns,
		ObjectType: result.ObjectType,
		PrimaryKey: result.PrimaryKey,
		Schema:     result.Schema,
		Table:      result.Table,
	}, nil
}

func (a *v2CompilerAdapter) CompileSearchObjects(objectType string, filters ontologyRepo.V2CompilerFilters) (*ontologyRepo.V2CompiledQuery, error) {
	ontologyFilters := ontology.ObjectFilters{
		Filters: filters.Filters,
		Limit:   filters.Limit,
		Offset:  filters.Offset,
		Sort:    filters.Sort,
		Order:   filters.Order,
	}
	result, err := a.compiler.CompileSearchObjects(objectType, ontologyFilters)
	if err != nil {
		return nil, err
	}
	return &ontologyRepo.V2CompiledQuery{
		SQL:        result.SQL,
		CountSQL:   result.CountSQL,
		Args:       result.Args,
		Columns:    result.Columns,
		ObjectType: result.ObjectType,
		PrimaryKey: result.PrimaryKey,
		Schema:     result.Schema,
		Table:      result.Table,
	}, nil
}

func (a *v2CompilerAdapter) CompileObjectMetrics(objectType, objectID string) (*ontologyRepo.V2CompiledQuery, error) {
	result, err := a.compiler.CompileObjectMetrics(objectType, objectID)
	if err != nil {
		return nil, err
	}
	return &ontologyRepo.V2CompiledQuery{
		SQL:        result.SQL,
		CountSQL:   result.CountSQL,
		Args:       result.Args,
		Columns:    result.Columns,
		ObjectType: result.ObjectType,
		PrimaryKey: result.PrimaryKey,
		Schema:     result.Schema,
		Table:      result.Table,
	}, nil
}

// sandboxServiceAdapter wraps review.SandboxService to implement mcp.SandboxService.
type sandboxServiceAdapter struct {
	svc *review.SandboxService
}

func (a *sandboxServiceAdapter) CreateSandbox(ctx context.Context, caseID string, data map[string]interface{}) (string, error) {
	return a.svc.CreateSandbox(ctx, caseID, data)
}

func (a *sandboxServiceAdapter) AddProposalToSandbox(ctx context.Context, sandboxID, proposalID string) error {
	return a.svc.AddProposalToSandbox(ctx, sandboxID, proposalID)
}

func (a *sandboxServiceAdapter) CompareSandbox(ctx context.Context, sandboxID1, sandboxID2 string) (*mcp.ComparisonResult, error) {
	result, err := a.svc.CompareSandbox(ctx, sandboxID1, sandboxID2)
	if err != nil {
		return nil, err
	}
	diffs := make([]mcp.Difference, 0, len(result.Differences))
	for _, d := range result.Differences {
		diffs = append(diffs, mcp.Difference{
			Field:  d.Field,
			Value1: d.Value1,
			Value2: d.Value2,
		})
	}
	return &mcp.ComparisonResult{
		Sandbox1ID:  result.Sandbox1ID,
		Sandbox2ID:  result.Sandbox2ID,
		Differences: diffs,
	}, nil
}

func (a *sandboxServiceAdapter) GetSandbox(ctx context.Context, sandboxID string) (*mcp.Sandbox, error) {
	sb, err := a.svc.GetSandbox(ctx, sandboxID)
	if err != nil {
		return nil, err
	}
	if sb == nil {
		return nil, nil
	}
	return &mcp.Sandbox{
		SandboxID:    sb.SandboxID,
		CaseID:       sb.CaseID,
		ProposalID:   stringPtrOrEmpty(sb.ProposalID),
		Data:         sb.SandboxData,
		Status:       sb.Status,
		ComparedWith: sb.ComparedWith,
		CreatedAt:    sb.CreatedAt.Format(time.RFC3339),
	}, nil
}

// stringPtrOrEmpty returns an empty string if the pointer is nil, or the dereferenced value.
func stringPtrOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// nullIfEmpty returns nil for an empty string, or the string pointer otherwise.
func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (a *ontologyServiceAdapter) ExecuteAction(ctx context.Context, objectType, objectID, actionType string, params map[string]interface{}) (*mcp.ActionResult, error) {
	if a.registry == nil {
		return nil, fmt.Errorf("ontology registry is not available")
	}

	// Check if the action is in the object type's AllowedActions list.
	allowedActions := a.registry.GetAllowedActions(objectType)
	allowed := false
	for _, aa := range allowedActions {
		if aa == actionType {
			allowed = true
			break
		}
	}
	if !allowed {
		return &mcp.ActionResult{
			Success:    false,
			ActionType: actionType,
			ObjectType: objectType,
			ObjectID:   objectID,
			Result:     map[string]interface{}{"error": fmt.Sprintf("action %q not allowed on %s", actionType, objectType)},
		}, nil
	}

	// Check if the action is allowed by the global action registry.
	if a.actionReg != nil && !a.actionReg.IsAllowed(actionType) {
		return &mcp.ActionResult{
			Success:    false,
			ActionType: actionType,
			ObjectType: objectType,
			ObjectID:   objectID,
			Result:     map[string]interface{}{"error": fmt.Sprintf("action %q is not allowed by the action registry", actionType)},
		}, nil
	}

	// Validate payload against action registry schema.
	if a.actionReg != nil && len(params) > 0 {
		if errs := a.actionReg.ValidatePayload(actionType, params); len(errs) > 0 {
			return &mcp.ActionResult{
				Success:    false,
				ActionType: actionType,
				ObjectType: objectType,
				ObjectID:   objectID,
				Result:     map[string]interface{}{"error": fmt.Sprintf("invalid action payload: %v", errs)},
			}, nil
		}
	}

	return &mcp.ActionResult{
		Success:    true,
		ActionType: actionType,
		ObjectType: objectType,
		ObjectID:   objectID,
		Result: map[string]interface{}{
			"would_execute": true,
			"message":       fmt.Sprintf("Action %q is valid for %s %s. Use propose_action to create a proposal for execution.", actionType, objectType, objectID),
		},
	}, nil
}

func newLinkExecutor(provider *common.PoolProvider) *ontologyRepo.LinkExecutor {
	return ontologyRepo.NewLinkExecutor(provider)
}

func (a *ontologyServiceAdapter) ProposeAction(ctx context.Context, objectType, objectID, actionType string, params map[string]interface{}, trace mcp.ProposeActionTrace) (*mcp.ActionResult, error) {
	if a.registry == nil {
		return nil, fmt.Errorf("ontology registry is not available")
	}

	allowedActions := a.registry.GetAllowedActions(objectType)
	allowed := false
	for _, aa := range allowedActions {
		if aa == actionType {
			allowed = true
			break
		}
	}
	if !allowed {
		return &mcp.ActionResult{
			Success:    false,
			ActionType: actionType,
			ObjectType: objectType,
			ObjectID:   objectID,
			Result:     map[string]interface{}{"error": fmt.Sprintf("action %q not allowed on %s", actionType, objectType)},
		}, nil
	}

	if a.actionReg != nil && !a.actionReg.IsAllowed(actionType) {
		return &mcp.ActionResult{
			Success:    false,
			ActionType: actionType,
			ObjectType: objectType,
			ObjectID:   objectID,
			Result:     map[string]interface{}{"error": fmt.Sprintf("action %q is not allowed by the action registry", actionType)},
		}, nil
	}

	if a.actionReg != nil && len(params) > 0 {
		if errs := a.actionReg.ValidatePayload(actionType, params); len(errs) > 0 {
			return &mcp.ActionResult{
				Success:    false,
				ActionType: actionType,
				ObjectType: objectType,
				ObjectID:   objectID,
				Result:     map[string]interface{}{"error": fmt.Sprintf("invalid action payload: %v", errs)},
			}, nil
		}
	}

	// Check allowed_by from the action registry config.
	// If the config defines specific roles (non-empty AllowedBy), the caller's
	// role must match at least one entry.
	if a.actionReg != nil {
		cfg, ok := a.actionReg.GetActionConfig(actionType)
		if ok && len(cfg.AllowedBy) > 0 {
			roleAllowed := false
			for _, role := range cfg.AllowedBy {
				if role == a.mcpRole {
					roleAllowed = true
					break
				}
			}
			if !roleAllowed {
				return &mcp.ActionResult{
					Success:    false,
					ActionType: actionType,
					ObjectType: objectType,
					ObjectID:   objectID,
					Result:     map[string]interface{}{"error": fmt.Sprintf("action %q requires one of roles %v, caller has role %q", actionType, cfg.AllowedBy, a.mcpRole)},
				}, nil
			}
		}
	}

	var caseID string
	if trace.CaseID != "" {
		caseID = trace.CaseID
	} else {
		caseID = fmt.Sprintf("mcp-%d", time.Now().UnixNano())
	}
	proposalID := fmt.Sprintf("mcp-proposal-%d", time.Now().UnixNano())

	payloadJSON, _ := json.Marshal(params)

	// Wrap the two INSERTs in a transaction to prevent orphaned decision_case rows.
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return &mcp.ActionResult{
			Success:    false,
			ActionType: actionType,
			ObjectType: objectType,
			ObjectID:   objectID,
			Result:     map[string]interface{}{"error": fmt.Sprintf("begin transaction: %v", err)},
		}, nil
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO ai.decision_case (case_id, status, source_type, source_id, object_type, object_id, created_at) VALUES ($1, 'open', $2, $3, $2, $3, NOW())`, caseID, objectType, objectID)
	if err != nil {
		return &mcp.ActionResult{
			Success:    false,
			ActionType: actionType,
			ObjectType: objectType,
			ObjectID:   objectID,
			Result:     map[string]interface{}{"error": fmt.Sprintf("create decision case: %v", err)},
		}, nil
	}

	title := fmt.Sprintf("MCP propose_action: %s on %s %s", actionType, objectType, objectID)
	_, err = tx.Exec(ctx,
		`INSERT INTO ai.action_proposal (proposal_id, case_id, decision_id, action_type, apply_status, title, risk_level, requires_human_review, payload, evidence_refs, context_hash, recipe_id, created_at)
		 VALUES ($1, $2, $3, $4, 'proposed', $5, 'medium', true, $6, $7, $8, $9, NOW())`,
		proposalID, caseID, nullIfEmpty(trace.DecisionID), actionType, title, payloadJSON,
		nullIfEmpty(trace.EvidenceRefs), nullIfEmpty(trace.ContextHash), nullIfEmpty(trace.RecipeID))
	if err != nil {
		return &mcp.ActionResult{
			Success:    false,
			ActionType: actionType,
			ObjectType: objectType,
			ObjectID:   objectID,
			Result:     map[string]interface{}{"error": fmt.Sprintf("create action proposal: %v", err)},
		}, nil
	}

	if err := tx.Commit(ctx); err != nil {
		return &mcp.ActionResult{
			Success:    false,
			ActionType: actionType,
			ObjectType: objectType,
			ObjectID:   objectID,
			Result:     map[string]interface{}{"error": fmt.Sprintf("commit transaction: %v", err)},
		}, nil
	}

	return &mcp.ActionResult{
		Success:    true,
		ActionType: actionType,
		ObjectType: objectType,
		ObjectID:   objectID,
		Result: map[string]interface{}{
			"proposal_id": proposalID,
			"case_id":     caseID,
			"status":      "proposed",
			"message":     fmt.Sprintf("Action %q proposed on %s %s", actionType, objectType, objectID),
		},
	}, nil
}
