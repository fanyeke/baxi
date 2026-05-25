package api

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Server represents the HTTP API server.
type Server struct {
	router chi.Router
	logger *zap.Logger
	pool   *pgxpool.Pool
	http   *http.Server
}

// New creates a new API server instance.
func New(logger *zap.Logger, pool *pgxpool.Pool) *Server {
	s := &Server{
		router: chi.NewRouter(),
		logger: logger,
		pool:   pool,
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.Timeout(30 * time.Second))

	s.router.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", s.handleHealth)
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
