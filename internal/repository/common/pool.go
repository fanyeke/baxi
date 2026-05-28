// Package common provides shared repository infrastructure.
package common

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolProvider provides access to database connections.
// This replaces the anti-pattern of passing *pgxpool.Pool as a parameter
// to every repository method. Store a PoolProvider in the repository struct.
type PoolProvider struct {
	pool *pgxpool.Pool
}

// NewPoolProvider creates a new PoolProvider.
func NewPoolProvider(pool *pgxpool.Pool) *PoolProvider {
	return &PoolProvider{pool: pool}
}

// Query executes a query and returns rows.
func (p *PoolProvider) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return p.pool.Query(ctx, sql, args...)
}

// QueryRow executes a query and returns a single row.
func (p *PoolProvider) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return p.pool.QueryRow(ctx, sql, args...)
}

// Begin starts a transaction.
func (p *PoolProvider) Begin(ctx context.Context) (pgx.Tx, error) {
	return p.pool.Begin(ctx)
}

// Pool returns the underlying pool for direct access if needed.
func (p *PoolProvider) Pool() *pgxpool.Pool {
	return p.pool
}
