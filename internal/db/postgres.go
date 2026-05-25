package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

// Pool wraps a pgxpool.Pool with a logger.
type Pool struct {
	*pgxpool.Pool
	logger *zap.Logger
}

// NewPool creates a new PostgreSQL connection pool.
func NewPool(ctx context.Context, databaseURL string, logger *zap.Logger) (*Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	logger.Info("connected to database")
	return &Pool{Pool: pool, logger: logger}, nil
}

// NewStdDB creates a *sql.DB from the pgx pool for use with goose.
// This is the critical bridge: pgx native pool → database/sql.
func NewStdDB(p *Pool) *sql.DB {
	return stdlib.OpenDBFromPool(p.Pool)
}

// Close shuts down the pool.
func (p *Pool) Close() {
	if p.Pool != nil {
		p.Pool.Close()
		p.logger.Info("database connection pool closed")
	}
}
