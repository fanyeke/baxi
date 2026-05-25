package testutil

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestStartPostgres_Smoke verifies that the test PostgreSQL container starts,
// migrations run, and the audit.pipeline_run table is queryable.
func TestStartPostgres_Smoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	pg, err := StartPostgres(ctx)
	if err != nil {
		t.Fatalf("StartPostgres: %v", err)
	}
	defer func() {
		if err := pg.Terminate(ctx); err != nil {
			t.Errorf("Terminate: %v", err)
		}
	}()

	// Run all migrations (001_init_schemas through 008_audit_tables)
	if err := pg.RunMigrations(ctx, "../../migrations"); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	// Connect and verify: audit.pipeline_run should exist and be empty
	pool, err := pgxpool.New(ctx, pg.ConnectionString())
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	defer pool.Close()

	var count int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM audit.pipeline_run").Scan(&count); err != nil {
		t.Fatalf("query audit.pipeline_run: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 rows in audit.pipeline_run, got %d", count)
	}

	t.Logf("smoke test passed: migrations applied, audit.pipeline_run has %d rows", count)
}
