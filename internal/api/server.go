package api

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"baxi/internal/api/handler"
	apimw "baxi/internal/api/middleware"
	"baxi/internal/repository"
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
	svc := service.NewOutboxService(repo, s.pool)
	return handler.NewOutboxHandler(svc)
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
