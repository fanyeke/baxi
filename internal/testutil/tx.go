package testutil

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WithTransaction runs fn inside a transaction that is always rolled back
// after fn completes, providing test isolation. Any data written within fn
// is discarded when the transaction rolls back.
func WithTransaction(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := fn(tx); err != nil {
		return fmt.Errorf("transaction function: %w", err)
	}

	return nil
}
