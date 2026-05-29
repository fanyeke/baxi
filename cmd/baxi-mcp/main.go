package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
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
	ctxBuilder := decision.NewContextBuilder(decisionRepo, objectSvc, classSvc, pool.Pool, action.NewActionTypeProviderAdapter(reg))

	decisionProvider := llm.NewRuleBasedProvider()
	engine := decision.NewDecisionEngine(decisionProvider, decisionRepo, pool.Pool, llm.NewDBAuditLogger(pool.Pool))
	proposalSvc := action.NewProposalService(decisionRepo, decisionRepo, reg, pool.Pool)

	decisionSvc := service.NewDecisionService(caseSvc, ctxBuilder, engine, proposalSvc, pool.Pool)
	alertSvc := service.NewAlertService(alertRepo, pool.Pool)
	govSvc := &governanceServiceAdapter{
		svc: service.NewGovernanceService(govRepo, pool.Pool),
	}

	// Pipeline runner (simple stub)
	pipelineSteps := []pipeline.Step{
		steps.NewIngestRawStep(),
		steps.NewBuildDWDSOrderLevelStep(),
	}
	pipelineRunner := &pipeline.Runner{DB: pool.Pool, Steps: pipelineSteps, Log: zapLog}
	pipelineSvc := &pipelineRunService{runner: pipelineRunner}

	// Wire review service
	reviewRepo := review.NewReviewRepository()
	reviewSvc := review.NewReviewService(reviewRepo, pool.Pool)

	// Wire outbox and pipeline info services
	outboxSvc := &outboxServiceAdapter{pool: pool.Pool}
	pipelineInfoSvc := &pipelineInfoAdapter{pool: pool.Pool}

	// Minimal stubs for services not yet implemented
	executeSvc := &executeServiceAdapter{}
	statusSvc := &statusServiceAdapter{}
	searchSvc := &searchServiceAdapter{}

	// Create MCP server with stdio transport
	mcpSrv, err := mcp.NewServer(
		decisionSvc, engine, ctxBuilder, proposalSvc, alertSvc, govSvc, pipelineSvc,
		reviewSvc, outboxSvc, pipelineInfoSvc,
		executeSvc, pool.Pool,
		statusSvc, searchSvc,
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

// executeServiceAdapter is a minimal stub for executing action proposals.
type executeServiceAdapter struct{}

func (a *executeServiceAdapter) ExecuteProposal(ctx context.Context, pool *pgxpool.Pool, proposalID string, actorID string, opts ...action.ExecuteOption) (*action.ExecutionResult, error) {
	return &action.ExecutionResult{Success: true, DryRun: true}, nil
}

// statusServiceAdapter is a minimal stub for system status.
type statusServiceAdapter struct{}

func (a *statusServiceAdapter) GetStatus(ctx context.Context) (*model.SystemStatus, error) {
	return &model.SystemStatus{}, nil
}

// searchServiceAdapter is a minimal stub for object search.
type searchServiceAdapter struct{}

func (a *searchServiceAdapter) SearchObjects(ctx context.Context, objectType, query string, limit, offset int) (*model.SearchResult, error) {
	return &model.SearchResult{Items: []map[string]interface{}{}, Total: 0}, nil
}
