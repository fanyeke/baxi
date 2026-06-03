package api

import (
	"context"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"baxi/internal/api/handler"
	"baxi/internal/config"
	"baxi/internal/repository/common"
	taskRepo "baxi/internal/repository/task"
	"baxi/internal/service"
)

// Server represents the HTTP API server.
type Server struct {
	ctx                  context.Context
	cancel               context.CancelFunc
	router               chi.Router
	logger               *zap.Logger
	pool                 *pgxpool.Pool
	http                 *http.Server
	taskSvc              *service.TaskService
	bearerToken          string
	corsAllowedOrigins   string
	cfg                  *config.Config
	handlerMu            sync.Mutex
	decisionHandlerVal   *handler.DecisionHandler
	actionHandlerVal     *handler.ActionHandler
	llmHandlerVal        *handler.LLMHandler
	feishuHandlerVal     *handler.FeishuHandler
	diagnosisHandlerVal  *handler.DiagnosisHandler
	outboxHandlerVal     *handler.OutboxHandler
	governanceHandlerVal *handler.GovernanceHandler
	qoderHandlerVal      *handler.QoderHandler
	statusHandlerVal     *handler.StatusHandler
	logHandlerVal        *handler.LogHandler
	agentLogHandlerVal   *handler.AgentLogHandler
	alertHandlerVal      *handler.AlertHandler
	reviewHandlerVal     *handler.ReviewHandler
	pipelineHandlerVal   *handler.PipelineHandler
	sandboxHandlerVal    *handler.SandboxHandler
}

// New creates a new API server instance.
func New(logger *zap.Logger, pool *pgxpool.Pool, cfg *config.Config) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Server{
		ctx:                ctx,
		cancel:             cancel,
		router:             chi.NewRouter(),
		logger:             logger,
		pool:               pool,
		cfg:                cfg,
		taskSvc:            service.NewTaskService(taskRepo.NewRepository(common.NewPoolProvider(pool))),
		bearerToken:        os.Getenv("API_BEARER_TOKEN"),
		corsAllowedOrigins: os.Getenv("CORS_ALLOWED_ORIGINS"),
	}
	s.setupRoutes()
	return s
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
	defer s.cancel()
	return s.http.Shutdown(ctx)
}
