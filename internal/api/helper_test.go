package api

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"baxi/internal/config"
)

// newTestServer creates an API server for testing with minimal dependencies.
// pool may be nil for tests that don't need database access.
func newTestServer(t *testing.T, pool *pgxpool.Pool) *Server {
	t.Helper()
	logger := zap.NewNop()
	cfg := &config.Config{}
	return New(logger, pool, cfg)
}
