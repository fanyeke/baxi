package pipeline

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type mockStep struct {
	name string
	succ bool
}

func (m *mockStep) Name() string { return m.name }

func (m *mockStep) Run(_ context.Context, _ pgx.Tx, _ StepInput) (*StepOutput, error) {
	if !m.succ {
		return nil, fmt.Errorf("mock step %s failed", m.name)
	}
	return &StepOutput{InputCount: 5, OutputCount: 5}, nil
}

type stepSpy struct {
	*mockStep
	called bool
}

func (s *stepSpy) Run(ctx context.Context, tx pgx.Tx, input StepInput) (*StepOutput, error) {
	s.called = true
	return s.mockStep.Run(ctx, tx, input)
}

func TestRunnerAuditRecordsCreated(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	log := zap.NewNop()
	step := &stepSpy{mockStep: &mockStep{name: "test-step", succ: true}}

	runner := &Runner{
		DB:    pool,
		Steps: []Step{step},
		Log:   log,
	}

	err := runner.Run(context.Background(), RunInput{
		RunType: "full",
		Mode:    "manual",
		DataDir: "/tmp/test",
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !step.called {
		t.Fatal("expected step to be called")
	}

	var runStatus, runType, runMode string
	err = pool.QueryRow(context.Background(),
		`SELECT status, run_type, mode FROM audit.pipeline_run ORDER BY started_at DESC LIMIT 1`,
	).Scan(&runStatus, &runType, &runMode)
	if err != nil {
		t.Fatalf("query pipeline_run: %v", err)
	}
	if runStatus != "completed" {
		t.Errorf("expected run status 'completed', got %q", runStatus)
	}
	if runType != "full" {
		t.Errorf("expected run_type 'full', got %q", runType)
	}
	if runMode != "manual" {
		t.Errorf("expected mode 'manual', got %q", runMode)
	}

	var stepStatus, stepName string
	var stepInputCount, stepOutputCount int64
	err = pool.QueryRow(context.Background(),
		`SELECT status, step_name, input_count, output_count
		 FROM audit.pipeline_step_run ORDER BY started_at DESC LIMIT 1`,
	).Scan(&stepStatus, &stepName, &stepInputCount, &stepOutputCount)
	if err != nil {
		t.Fatalf("query pipeline_step_run: %v", err)
	}
	if stepStatus != "completed" {
		t.Errorf("expected step status 'completed', got %q", stepStatus)
	}
	if stepName != "test-step" {
		t.Errorf("expected step_name 'test-step', got %q", stepName)
	}
	if stepInputCount != 5 {
		t.Errorf("expected input_count 5, got %d", stepInputCount)
	}
	if stepOutputCount != 5 {
		t.Errorf("expected output_count 5, got %d", stepOutputCount)
	}
}

func TestRunnerSuccess(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	log := zap.NewNop()
	steps := []Step{
		&mockStep{name: "step-a", succ: true},
		&mockStep{name: "step-b", succ: true},
		&mockStep{name: "step-c", succ: true},
	}

	runner := &Runner{
		DB:    pool,
		Steps: steps,
		Log:   log,
	}

	err := runner.Run(context.Background(), RunInput{
		RunType: "full",
		Mode:    "manual",
		DataDir: "/tmp/test",
	})
	if err != nil {
		t.Fatalf("expected no error on successful run, got: %v", err)
	}

	var count int
	err = pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM audit.pipeline_step_run WHERE status = 'completed'`,
	).Scan(&count)
	if err != nil {
		t.Fatalf("query completed steps: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 completed steps, got %d", count)
	}

	var runStatus string
	err = pool.QueryRow(context.Background(),
		`SELECT status FROM audit.pipeline_run ORDER BY started_at DESC LIMIT 1`,
	).Scan(&runStatus)
	if err != nil {
		t.Fatalf("query run status: %v", err)
	}
	if runStatus != "completed" {
		t.Errorf("expected run 'completed', got %q", runStatus)
	}
}

func TestRunnerFailure(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	log := zap.NewNop()
	steps := []Step{
		&mockStep{name: "step-ok", succ: true},
		&mockStep{name: "step-fail", succ: false},
		&mockStep{name: "step-never", succ: true},
	}

	runner := &Runner{
		DB:    pool,
		Steps: steps,
		Log:   log,
	}

	err := runner.Run(context.Background(), RunInput{
		RunType: "partial",
		Mode:    "manual",
		DataDir: "/tmp/test",
	})
	if err == nil {
		t.Fatal("expected error on failed step, got nil")
	}

	var completed int
	err = pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM audit.pipeline_step_run WHERE status = 'completed'`,
	).Scan(&completed)
	if err != nil {
		t.Fatalf("query completed steps: %v", err)
	}
	if completed != 1 {
		t.Errorf("expected 1 completed step, got %d", completed)
	}

	var failed int
	err = pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM audit.pipeline_step_run WHERE status = 'failed'`,
	).Scan(&failed)
	if err != nil {
		t.Fatalf("query failed steps: %v", err)
	}
	if failed != 1 {
		t.Errorf("expected 1 failed step, got %d", failed)
	}

	var runStatus, errMsg string
	err = pool.QueryRow(context.Background(),
		`SELECT status, COALESCE(error_message, '') FROM audit.pipeline_run ORDER BY started_at DESC LIMIT 1`,
	).Scan(&runStatus, &errMsg)
	if err != nil {
		t.Fatalf("query run status: %v", err)
	}
	if runStatus != "failed" {
		t.Errorf("expected run 'failed', got %q", runStatus)
	}
	if errMsg == "" {
		t.Error("expected non-empty error_message on failed run")
	}

	var third int
	err = pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM audit.pipeline_step_run WHERE step_name = 'step-never'`,
	).Scan(&third)
	if err != nil {
		t.Fatalf("query step-never: %v", err)
	}
	if third != 0 {
		t.Errorf("expected 0 records for unexecuted step, got %d", third)
	}
}

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping pipeline integration tests")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	t.Cleanup(pool.Close)

	ctx := context.Background()
	schema := `
	CREATE SCHEMA IF NOT EXISTS audit;
	CREATE TABLE IF NOT EXISTS audit.pipeline_run (
		run_id        TEXT PRIMARY KEY,
		run_type      TEXT NOT NULL,
		mode          TEXT NOT NULL,
		status        TEXT NOT NULL,
		started_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		finished_at   TIMESTAMPTZ,
		input_count   BIGINT DEFAULT 0,
		output_count  BIGINT DEFAULT 0,
		error_message TEXT
	);
	CREATE TABLE IF NOT EXISTS audit.pipeline_step_run (
		step_run_id    TEXT PRIMARY KEY,
		pipeline_run_id TEXT,
		step_name      TEXT,
		step_order     BIGINT,
		status         TEXT,
		started_at     TIMESTAMPTZ DEFAULT NOW(),
		finished_at    TIMESTAMPTZ,
		input_count    BIGINT,
		output_count   BIGINT,
		error_message  TEXT
	);`
	_, err = pool.Exec(ctx, schema)
	if err != nil {
		t.Fatalf("migrate test schema: %v", err)
	}
	return pool
}
