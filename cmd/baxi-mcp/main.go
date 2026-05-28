package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

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

	// Create MCP server with stdio transport
	mcpSrv, err := mcp.NewServer(decisionSvc, engine, ctxBuilder, proposalSvc, alertSvc, govSvc, pipelineSvc)
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
	return "run-id", nil
}
