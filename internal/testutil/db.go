// Package testutil provides test helpers for PostgreSQL integration tests
// using testcontainers-go. All downstream pipeline tests should use these
// helpers for consistent test setup.
package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PostgresContainer wraps a test PostgreSQL container with its connection string.
type PostgresContainer struct {
	Container *postgres.PostgresContainer
	ConnStr   string
}

// StartPostgres starts a PostgreSQL 15-alpine container with a test database.
// Returns the container wrapper or an error.
func StartPostgres(ctx context.Context) (*PostgresContainer, error) {
	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("docker.io/postgres:15-alpine"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategyAndDeadline(120*time.Second,
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(120*time.Second),
			wait.ForListeningPort("5432/tcp").
				WithStartupTimeout(120*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("start postgres container: %w", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, fmt.Errorf("get connection string: %w", err)
	}

	return &PostgresContainer{
		Container: pgContainer,
		ConnStr:   connStr,
	}, nil
}

// ConnectionString returns the PostgreSQL connection URL with sslmode=disable.
func (c *PostgresContainer) ConnectionString() string {
	return c.ConnStr
}

// RunMigrations runs all goose migrations from the given directory.
// Uses the existing pgx bridge pattern (pgxpool → database/sql) for goose compatibility.
func (c *PostgresContainer) RunMigrations(ctx context.Context, migrationsDir string) error {
	pool, err := pgxpool.New(ctx, c.ConnStr)
	if err != nil {
		return fmt.Errorf("connect pool for migrations: %w", err)
	}
	defer pool.Close()

	db := stdlib.OpenDBFromPool(pool)
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			fmt.Printf("warning: closing stdlib db: %v\n", closeErr)
		}
	}()

	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("run goose migrations from %q: %w", migrationsDir, err)
	}

	return nil
}

// RunMigrationsWithOptions runs goose migrations with a specific set of options.
// Useful for selective migration runs (e.g., up-by-one, reset, redo).
func (c *PostgresContainer) RunMigrationsWithOptions(ctx context.Context, migrationsDir string, fn func(db *sql.DB) error) error {
	pool, err := pgxpool.New(ctx, c.ConnStr)
	if err != nil {
		return fmt.Errorf("connect pool for migrations: %w", err)
	}
	defer pool.Close()

	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()

	return fn(db)
}

// Terminate stops and removes the container.
func (c *PostgresContainer) Terminate(ctx context.Context) error {
	if err := c.Container.Terminate(ctx); err != nil {
		return fmt.Errorf("terminate postgres container: %w", err)
	}
	return nil
}
