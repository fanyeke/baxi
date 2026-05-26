package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"baxi/internal/action"
	"baxi/internal/api/dto"
	"baxi/internal/api/handler"
	apimw "baxi/internal/api/middleware"
	"baxi/internal/decision"
	"baxi/internal/governance"
	"baxi/internal/llm"
	"baxi/internal/ontology"
	"baxi/internal/outbox"
	"baxi/internal/repository"
	"baxi/internal/review"
	"baxi/internal/service"
)

// Server represents the HTTP API server.
type Server struct {
	router           chi.Router
	logger           *zap.Logger
	pool             *pgxpool.Pool
	http             *http.Server
	taskSvc          *service.TaskService
	bearerToken      string
	corsAllowedOrigins string
	decisionHandlerVal  *handler.DecisionHandler
	actionHandlerVal    *handler.ActionHandler
}

// New creates a new API server instance.
func New(logger *zap.Logger, pool *pgxpool.Pool) *Server {
	s := &Server{
		router:             chi.NewRouter(),
		logger:             logger,
		pool:               pool,
		taskSvc:            service.NewTaskService(repository.NewTaskRepository(), pool),
		bearerToken:        os.Getenv("API_BEARER_TOKEN"),
		corsAllowedOrigins: os.Getenv("CORS_ALLOWED_ORIGINS"),
	}
	s.setupRoutes()
	return s
}

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
		ctxBuilder := decision.NewContextBuilder(decisionRepo, objectSvc, classSvc, s.pool)

		ruleProvider := llm.NewRuleBasedProvider()
		engine := decision.NewDecisionEngine(ruleProvider, decisionRepo, s.pool)

		proposalSvc := action.NewProposalService(decisionRepo, decisionRepo, s.pool)

		svc := service.NewDecisionService(caseSvc, ctxBuilder, engine, proposalSvc, s.pool)
		s.decisionHandlerVal = handler.NewDecisionHandler(svc)
	}
	return s.decisionHandlerVal
}

// reviewHandlerSvc adapts review.ReviewService to handler.ReviewService.
type reviewHandlerSvc struct {
	svc  *review.ReviewService
	repo *review.ReviewRepository
	pool *pgxpool.Pool
}

type outboxServiceAdapter struct {
	readSvc   *service.OutboxService
	readRepo  *repository.OutboxRepository
	writeRepo *outbox.OutboxRepository
	pool      *pgxpool.Pool
	executors map[string]action.ActionExecutor
}

func (a *outboxServiceAdapter) List(ctx context.Context, filters dto.OutboxFilters, limit, offset int) (*dto.OutboxListResponse, error) {
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
			ProposalID:  detail.EventID,
			ActionType:  detail.EventType,
			CaseID:      detail.SourceID,
			Title:       detail.EventType + " - " + detail.SourceID,
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

// actionHandlerSvc adapts action.ApplyService and review.ReviewRepository to handler.ActionService.
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

func (s *Server) actionHandler() *handler.ActionHandler {
	if s.actionHandlerVal == nil {
		repo := review.NewReviewRepository()
		reg, _ := action.NewActionRegistry("")
		loader := &proposalLoaderAdapter{repo: repo}
		applySvc := action.NewApplyService(reg, nil, loader)
		adapter := &actionHandlerSvc{
			applySvc: applySvc,
			repo:     repo,
			pool:     s.pool,
		}
		s.actionHandlerVal = handler.NewActionHandler(adapter, s.pool)
	}
	return s.actionHandlerVal
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

func (s *Server) setupRoutes() {
	s.router.Use(apimw.RequestIDMiddleware)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.Timeout(30 * time.Second))
	s.router.Use(apimw.NewCORSMiddleware(s.corsAllowedOrigins))

	s.router.Route("/api/v1", func(r chi.Router) {
		// Public routes (no auth required)
		r.Get("/health", s.handleHealth)

		// Protected routes (auth required when API_BEARER_TOKEN is set)
		r.Group(func(r chi.Router) {
			if s.bearerToken != "" {
				r.Use(apimw.NewAuthMiddleware(s.bearerToken))
			}

			r.Get("/status", s.statusHandler().HandleStatus)
			r.Get("/alerts", s.alertHandler().HandleListAlerts)
			r.Get("/tasks", s.handleListTasks)
			r.Get("/outbox", s.outboxHandler().HandleListOutbox)
			r.Get("/outbox/{id}", s.outboxHandler().HandleGetDetail)
			r.Post("/outbox/{id}/dispatch", s.outboxHandler().HandleDispatch)
			r.Post("/outbox/{id}/cancel", s.outboxHandler().HandleCancel)
			r.Get("/governance/status", s.governanceHandler().HandleGovernanceStatus)
			r.Get("/governance/catalog", s.governanceHandler().HandleCatalog)
			r.Get("/governance/classification", s.governanceHandler().HandleClassification)
			r.Get("/governance/markings", s.governanceHandler().HandleMarkings)
			r.Get("/governance/lineage", s.governanceHandler().HandleLineage)
			r.Get("/governance/checkpoints", s.governanceHandler().HandleCheckpoints)
			r.Get("/governance/health", s.governanceHandler().HandleHealth)
			r.Get("/logs/recent", s.logHandler().HandleListRecent)
			r.Get("/logs/errors", s.logHandler().HandleListErrors)
			r.Get("/logs/audit", s.logHandler().HandleListAudit)
			r.Get("/qoder/capabilities", s.qoderHandler().HandleCapabilities)
			r.Get("/qoder/context", s.qoderHandler().HandleContext)

			// Decision case endpoints
			r.Post("/decisions/cases", s.decisionHandler().CreateCase)
			r.Get("/decisions/cases", s.decisionHandler().ListCases)
			r.Get("/decisions/cases/{case_id}", s.decisionHandler().GetCase)
			r.Post("/decisions/cases/{case_id}/context", s.decisionHandler().BuildContext)
			r.Post("/decisions/cases/{case_id}/decide", s.decisionHandler().Decide)
			r.Get("/decisions/cases/{case_id}/proposals", s.decisionHandler().ListProposals)

			// Review endpoints
			r.Post("/proposals/{id}/approve", s.reviewHandler().HandleApprove)
			r.Post("/proposals/{id}/reject", s.reviewHandler().HandleReject)
			r.Post("/proposals/{id}/cancel", s.reviewHandler().HandleCancel)
			r.Get("/proposals/{id}/review", s.reviewHandler().HandleGetReview)

			// Action execution endpoints
			r.Post("/proposals/{id}/execute", s.actionHandler().HandleExecute)
			r.Get("/proposals/{id}/status", s.actionHandler().HandleStatus)
		})
	})
}

// Start starts the HTTP server on the given address.
func (s *Server) Start(addr string) error {
	s.http = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.logger.Info("starting API server", zap.String("addr", addr))
	return s.http.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down API server")
	return s.http.Shutdown(ctx)
}
