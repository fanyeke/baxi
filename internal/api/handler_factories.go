package api

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"baxi/internal/action"
	"baxi/internal/adapter"
	"baxi/internal/api/handler"
	"baxi/internal/decision"
	"baxi/internal/eval"
	"baxi/internal/governance"
	"baxi/internal/llm"
	"baxi/internal/model"
	"baxi/internal/ontology"
	"baxi/internal/outbox"
	"baxi/internal/pipeline"
	"baxi/internal/pipeline/steps"
	"baxi/internal/repository"
	"baxi/internal/review"
	"baxi/internal/service"
)

// outboxHandler lazily initializes the outbox handler.
func (s *Server) outboxHandler() *handler.OutboxHandler {
	repo := repository.NewOutboxRepository()
	writeRepo := outbox.NewOutboxRepository()
	svc := service.NewOutboxService(repo, s.pool)
	adapter := &outboxServiceAdapter{
		readSvc:   svc,
		readRepo:  repo,
		writeRepo: writeRepo,
		pool:      s.pool,
		executors: s.actionExecutors(),
	}
	return handler.NewOutboxHandler(adapter)
}

// actionExecutors lazily initializes the action executors.
func (s *Server) actionExecutors() map[string]action.ActionExecutor {
	executors := make(map[string]action.ActionExecutor)
	executors["noop"] = action.NewNoOpExecutor()
	return executors
}

// governanceHandler lazily initializes the governance handler.
func (s *Server) governanceHandler() *handler.GovernanceHandler {
	repo := repository.NewGovernanceRepository()
	svc := service.NewGovernanceService(repo, s.pool)
	return handler.NewGovernanceHandler(svc, svc)
}

// qoderHandler lazily initializes the Qoder handler.
func (s *Server) qoderHandler() *handler.QoderHandler {
	ctxRepo := repository.NewContextRepository()
	svc := service.NewQoderService(ctxRepo, s.pool)
	return handler.NewQoderHandler(svc)
}

// statusHandler lazily initializes the status handler.
func (s *Server) statusHandler() *handler.StatusHandler {
	repo := repository.NewStatusRepository()
	dbURL := ""
	if s.pool != nil {
		dbURL = s.pool.Config().ConnString()
	}
	svc := service.NewStatusService(repo, s.pool, dbURL)
	return handler.NewStatusHandler(svc)
}

// logHandler lazily initializes the logs handler.
func (s *Server) logHandler() *handler.LogHandler {
	repo := repository.NewLogRepository()
	svc := service.NewLogService(repo, s.pool)
	return handler.NewLogHandler(svc)
}

// alertHandler lazily initializes the alerts handler.
func (s *Server) alertHandler() *handler.AlertHandler {
	repo := repository.NewAlertRepository()
	svc := service.NewAlertService(repo, s.pool)
	return handler.NewAlertHandler(svc)
}

// llmHandler lazily initializes the LLM handler.
func (s *Server) llmHandler() *handler.LLMHandler {
	if s.llmHandlerVal == nil {
		s.llmHandlerVal = handler.NewLLMHandler(s.cfg, eval.NewMetricsCollector())
	}
	return s.llmHandlerVal
}

// feishuHandler lazily initializes the Feishu handler.
func (s *Server) feishuHandler() *handler.FeishuHandler {
	if s.feishuHandlerVal == nil {
		feishuWebhookURL := ""
		if s.cfg != nil {
			feishuWebhookURL = s.cfg.FeishuWebhookURL
		}
		feishuAdapter := adapter.NewFeishuAdapter(adapter.FeishuConfig{
			WebhookURL: feishuWebhookURL,
			Enabled:    feishuWebhookURL != "",
		})
		svc := handler.NewFeishuService(feishuAdapter)
		s.feishuHandlerVal = handler.NewFeishuHandler(svc)
	}
	return s.feishuHandlerVal
}

// diagnosisHandler lazily initializes the diagnosis handler.
func (s *Server) diagnosisHandler() *handler.DiagnosisHandler {
	if s.diagnosisHandlerVal == nil {
		svc := service.NewDiagnosisService(
			"logs/api/error.log",
			"data/system/api_audit_dispatch.csv",
			"data/system/api_audit_feishu.csv",
		)
		s.diagnosisHandlerVal = handler.NewDiagnosisHandler(svc)
	}
	return s.diagnosisHandlerVal
}

// decisionHandler lazily initializes the decision handler.
func (s *Server) decisionHandler() *handler.DecisionHandler {
	if s.decisionHandlerVal == nil {
		decisionRepo := repository.NewDecisionRepository()
		alertRepo := repository.NewAlertRepository()
		caseSvc := decision.NewCaseService(decisionRepo, alertRepo, s.pool)

		ontologyRepo := repository.NewOntologyRepo()
		objectSvc := ontology.NewObjectQueryService(ontologyRepo, s.pool)
		govRepo := repository.NewGovernanceRepository()
		classSvc := governance.NewClassificationService(s.pool, govRepo)
		reg, err := action.NewActionRegistry("")
		if err != nil {
			log.Printf("WARNING: failed to load action registry: %v, using empty fallback", err)
			reg = action.NewEmptyRegistry()
		}
		ctxBuilder := decision.NewContextBuilder(decisionRepo, objectSvc, classSvc, s.pool, action.NewActionTypeProviderAdapter(reg))

		var decisionProvider llm.DecisionProvider
		decisionProvider = llm.NewRuleBasedProvider()
		if s.cfg != nil {
			promptReg, _ := llm.NewPromptRegistry()
			factory := llm.NewProviderFactory(s.cfg, promptReg)
			if p, err := factory.CreateProvider(); err == nil {
				decisionProvider = p
			} else {
				log.Printf("WARNING: failed to create LLM provider: %v, falling back to rule-based", err)
			}
		}
		engine := decision.NewDecisionEngine(decisionProvider, decisionRepo, s.pool, llm.NewDBAuditLogger(s.pool))

		proposalSvc := action.NewProposalService(decisionRepo, decisionRepo, reg, s.pool)

		replayRepo := eval.NewPGReplayRepository(s.pool)
		auditLogger := llm.NewDBAuditLogger(s.pool)
		replaySvc := eval.NewReplayService(replayRepo, decisionProvider, auditLogger)

		ruleProvider := llm.NewRuleBasedProvider()

		svc := service.NewDecisionService(caseSvc, ctxBuilder, engine, proposalSvc, s.pool).
			WithMetrics(eval.NewMetricsCollector()).
			WithReplayService(replaySvc).
			WithRuleProvider(ruleProvider)
		s.decisionHandlerVal = handler.NewDecisionHandler(svc)
	}
	return s.decisionHandlerVal
}

// reviewHandler lazily initializes the review handler.
func (s *Server) reviewHandler() *handler.ReviewHandler {
	repo := review.NewReviewRepository()
	svc := review.NewReviewService(repo, s.pool)
	adapter := &reviewHandlerSvc{
		svc:  svc,
		repo: repo,
		pool: s.pool,
	}
	return handler.NewReviewHandler(adapter)
}

// pipelineHandler lazily initializes the pipeline handler.
func (s *Server) pipelineHandler() *handler.PipelineHandler {
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
	runner := &pipeline.Runner{
		DB:    s.pool,
		Steps: pipelineSteps,
		Log:   s.logger,
	}
	svc := &pipelineRunService{ctx: s.ctx, runner: runner, log: s.logger}
	return handler.NewPipelineHandler(svc)
}

// actionHandler lazily initializes the action handler.
func (s *Server) actionHandler() *handler.ActionHandler {
	if s.actionHandlerVal == nil {
		repo := review.NewReviewRepository()
		reg, err := action.NewActionRegistry("")
		if err != nil {
			log.Printf("WARNING: failed to load action registry: %v, using empty fallback", err)
			reg = action.NewEmptyRegistry()
		}
		loader := &proposalLoaderAdapter{repo: repo}
		feishuWebhookURL := ""
		githubToken := ""
		if s.cfg != nil {
			feishuWebhookURL = s.cfg.FeishuWebhookURL
			githubToken = s.cfg.GitHubToken
		}
		feishuExec := adapter.NewFeishuAdapter(adapter.FeishuConfig{
			WebhookURL: feishuWebhookURL,
			Enabled:    feishuWebhookURL != "",
		})
		githubExec := adapter.NewGitHubAdapter(adapter.GitHubConfig{
			Token:   githubToken,
			Enabled: githubToken != "",
		})
		executors := map[string]action.ActionExecutor{
			"feishu": feishuExec,
			"github": githubExec,
		}
		applySvc := action.NewApplyService(reg, executors, loader, nil, nil, s.pool)
		adapter := &actionHandlerSvc{
			applySvc: applySvc,
			repo:     repo,
			pool:     s.pool,
		}
		s.actionHandlerVal = handler.NewActionHandler(adapter, s.pool)
	}
	return s.actionHandlerVal
}

// Adapter structs and implementations

type reviewHandlerSvc struct {
	svc  *review.ReviewService
	repo *review.ReviewRepository
	pool *pgxpool.Pool
}

func (a *reviewHandlerSvc) ApproveProposal(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error) {
	return a.svc.ApproveProposal(ctx, proposalID, reviewerID, feedback)
}

func (a *reviewHandlerSvc) RejectProposal(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error) {
	return a.svc.RejectProposal(ctx, proposalID, reviewerID, feedback)
}

func (a *reviewHandlerSvc) CancelProposal(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error) {
	return a.svc.CancelProposal(ctx, proposalID, reviewerID, feedback)
}

func (a *reviewHandlerSvc) GetReviewByProposal(ctx context.Context, proposalID string) (*review.ReviewRecord, error) {
	records, err := a.repo.GetReviewsByProposal(ctx, a.pool, proposalID)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	return &records[0], nil
}

type outboxServiceAdapter struct {
	readSvc   *service.OutboxService
	readRepo  *repository.OutboxRepository
	writeRepo *outbox.OutboxRepository
	pool      *pgxpool.Pool
	executors map[string]action.ActionExecutor
}

func (a *outboxServiceAdapter) List(ctx context.Context, filters model.OutboxFilters, limit, offset int) (*model.OutboxListResponse, error) {
	return a.readSvc.List(ctx, filters, limit, offset)
}

func (a *outboxServiceAdapter) GetEvent(ctx context.Context, id string) (*handler.OutboxDetailItem, error) {
	detail, err := a.readRepo.GetDetail(ctx, a.pool, id)
	if err != nil || detail == nil {
		return nil, err
	}
	item := &handler.OutboxDetailItem{
		EventID:          detail.EventID,
		EventType:        detail.EventType,
		SourceType:       detail.SourceType,
		SourceID:         detail.SourceID,
		TargetChannel:    detail.TargetChannel,
		Status:           detail.Status,
		CreatedAt:        detail.CreatedAt,
		DispatchAttempts: detail.DispatchAttempts,
		LastDispatchAt:   detail.LastDispatchAt,
		ErrorMessage:     detail.ErrorMessage,
	}
	if detail.Payload != nil {
		item.Payload = string(detail.Payload)
	}
	return item, nil
}

func (a *outboxServiceAdapter) DispatchEvent(ctx context.Context, id string) error {
	detail, err := a.readRepo.GetDetail(ctx, a.pool, id)
	if err != nil {
		return err
	}
	if detail == nil {
		return handler.ErrEventNotFound{}
	}
	if detail.Status != "pending" && detail.Status != "failed" {
		return handler.ErrInvalidState{Status: detail.Status}
	}

	executor, ok := a.executors[detail.TargetChannel]
	if !ok {
		executor = action.NewNoOpExecutor()
	}

	var proposal action.ActionProposal
	if len(detail.Payload) == 0 || json.Unmarshal(detail.Payload, &proposal) != nil {
		proposal = action.ActionProposal{
			ProposalID: detail.EventID,
			ActionType: detail.EventType,
			CaseID:     detail.SourceID,
			Title:      detail.EventType + " - " + detail.SourceID,
		}
	}

	result, err := executor.Execute(ctx, proposal, true)
	if err != nil || !result.Success {
		tx, txErr := a.pool.Begin(ctx)
		if txErr != nil {
			return txErr
		}
		defer tx.Rollback(ctx)
		errMsg := "dispatch failure"
		if err != nil {
			errMsg = err.Error()
		} else {
			errMsg = result.Error
		}
		if markErr := a.writeRepo.MarkFailed(ctx, tx, id, errMsg); markErr != nil {
			return markErr
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			return commitErr
		}
		if err != nil {
			return err
		}
		return fmt.Errorf("dispatch failed: %s", result.Error)
	}

	tx, txErr := a.pool.Begin(ctx)
	if txErr != nil {
		return txErr
	}
	defer tx.Rollback(ctx)
	if err := a.writeRepo.MarkDispatched(ctx, tx, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (a *outboxServiceAdapter) BatchDispatch(ctx context.Context, dryRun bool, channel string, limit int) (*handler.BatchDispatchResponse, error) {
	events, err := a.writeRepo.GetPendingEvents(ctx, a.pool, limit)
	if err != nil {
		return nil, fmt.Errorf("get pending events: %w", err)
	}

	if channel != "" {
		var filtered []outbox.OutboxEvent
		for _, e := range events {
			if e.TargetChannel == channel {
				filtered = append(filtered, e)
			}
		}
		events = filtered
	}

	resp := &handler.BatchDispatchResponse{
		DryRun:   dryRun,
		EventIDs: make([]string, 0, len(events)),
	}

	if dryRun {
		resp.Dispatched = len(events)
		for _, e := range events {
			resp.EventIDs = append(resp.EventIDs, e.EventID)
		}
		return resp, nil
	}

	for _, e := range events {
		if err := a.DispatchEvent(ctx, e.EventID); err != nil {
			resp.Failed++
		} else {
			resp.Dispatched++
			resp.EventIDs = append(resp.EventIDs, e.EventID)
		}
	}

	return resp, nil
}

func (a *outboxServiceAdapter) CancelEvent(ctx context.Context, id string) error {
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := a.writeRepo.MarkCancelled(ctx, tx, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type proposalLoaderAdapter struct {
	repo *review.ReviewRepository
}

func (p *proposalLoaderAdapter) GetProposalByID(ctx context.Context, pool *pgxpool.Pool, proposalID string) (*action.ActionProposal, error) {
	row, err := p.repo.GetProposalByID(ctx, pool, proposalID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	ap := &action.ActionProposal{
		ProposalID:          row.ProposalID,
		CaseID:              row.CaseID,
		ActionType:          row.ActionType,
		Title:               row.Title,
		ApplyStatus:         row.ApplyStatus,
		CreatedAt:           row.CreatedAt,
		RequiresHumanReview: row.RequiresHumanReview,
	}
	if row.DecisionID != nil {
		ap.DecisionID = *row.DecisionID
	}
	if row.Description != nil {
		ap.Description = *row.Description
	}
	if row.RiskLevel != nil {
		ap.RiskLevel = *row.RiskLevel
	}
	if row.Payload != nil {
		var payload map[string]interface{}
		if err := json.Unmarshal(*row.Payload, &payload); err == nil {
			ap.Payload = payload
		}
	}
	return ap, nil
}

type actionHandlerSvc struct {
	applySvc *action.ApplyService
	repo     *review.ReviewRepository
	pool     *pgxpool.Pool
}

func (a *actionHandlerSvc) ExecuteProposal(ctx context.Context, pool *pgxpool.Pool, proposalID, actorID string, opts ...action.ExecuteOption) (*action.ExecutionResult, error) {
	return a.applySvc.ExecuteProposal(ctx, pool, proposalID, actorID, opts...)
}

func (a *actionHandlerSvc) GetProposalByID(ctx context.Context, pool *pgxpool.Pool, proposalID string) (*action.ActionProposal, error) {
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
	}
	if row.Payload != nil {
		var payload map[string]interface{}
		if err := json.Unmarshal(*row.Payload, &payload); err == nil {
			p.Payload = payload
		}
	}
	return p, nil
}

type pipelineRunService struct {
	ctx    context.Context
	runner *pipeline.Runner
	log    *zap.Logger
}

func (s *pipelineRunService) Run(ctx context.Context, config string) (string, error) {
	runID := newPipelineRunID()
	s.log.Info("pipeline run requested",
		zap.String("run_id", runID),
		zap.String("config", config),
	)
	go func() {
		input := pipeline.RunInput{
			RunType: config,
			Mode:    "api",
			DataDir: "./data/raw",
		}
		if err := s.runner.Run(s.ctx, input); err != nil {
			s.log.Error("pipeline run failed",
				zap.String("run_id", runID),
				zap.String("config", config),
				zap.Error(err),
			)
			return
		}
		s.log.Info("pipeline run completed",
			zap.String("run_id", runID),
			zap.String("config", config),
		)
	}()
	return runID, nil
}

func newPipelineRunID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
