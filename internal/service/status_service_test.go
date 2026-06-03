package service

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/repository"
)

func setupSvcStatusTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	// Create schemas if they don't exist (existing migrations should have them)
	ctx := context.Background()
	for _, stmt := range []string{
		"CREATE SCHEMA IF NOT EXISTS ops",
		"CREATE SCHEMA IF NOT EXISTS dwd",
		"CREATE SCHEMA IF NOT EXISTS mart",
		"CREATE SCHEMA IF NOT EXISTS audit",
	} {
		_, err := pool.Exec(ctx, stmt)
		require.NoError(t, err)
	}

	return pool
}

func TestStatusService_GetStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupSvcStatusTestDB(t)
	ctx := context.Background()

	// Insert test data into ops.metric_alert
	_, err := pool.Exec(ctx, `
		INSERT INTO ops.metric_alert (alert_id, rule_id, event_date, severity, metric_name, status, created_at)
		VALUES ('st-alert-1', 'rule-1', CURRENT_DATE, 'high', 'revenue', 'new', NOW())
		ON CONFLICT (alert_id) DO NOTHING
	`)
	require.NoError(t, err)

	// Use the deprecated StatusRepository wrapper (matches service's expected type)
	repo := statusRepo.NewRepository(nil)
	repo.SetPool(pool)
	svc := NewStatusService(repo, pool, "postgres://baxi:baxi_dev@localhost:5432/baxi?sslmode=disable")

	resp, err := svc.GetStatus(ctx)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify response structure
	assert.True(t, resp.Database.Exists)
	assert.Equal(t, "localhost:5432/baxi", resp.Database.Path)
	assert.NotContains(t, resp.Database.Path, "baxi_dev")
	assert.Equal(t, "0.6.0", resp.Version)

	// Verify table counts are returned
	require.NotEmpty(t, resp.Database.Tables)
	alertCount, ok := resp.Database.Tables["alert_events"]
	assert.True(t, ok, "expected alert_events in table counts")
	assert.GreaterOrEqual(t, alertCount, 1)

	taskCount, ok := resp.Database.Tables["action_tasks"]
	assert.True(t, ok, "expected action_tasks in table counts")
	assert.GreaterOrEqual(t, taskCount, 0)

	outboxCount, ok := resp.Database.Tables["event_outbox"]
	assert.True(t, ok, "expected event_outbox in table counts")
	assert.GreaterOrEqual(t, outboxCount, 0)

	// Pipeline run may or may not exist
	if resp.LastPipelineRun != nil {
		assert.NotEmpty(t, resp.LastPipelineRun.RunID)
	}
}

func TestStatusService_GetStatus_Structure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupSvcStatusTestDB(t)
	ctx := context.Background()

	repo := statusRepo.NewRepository(nil)
	repo.SetPool(pool)
	svc := NewStatusService(repo, pool, "postgres://baxi:baxi_dev@localhost:5432/baxi?sslmode=disable")

	resp, err := svc.GetStatus(ctx)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Always expected fields
	assert.True(t, resp.Database.Exists)
	assert.Equal(t, "0.6.0", resp.Version)

	// Table counts should have all expected keys
	expectedTables := []string{"alert_events", "action_tasks", "event_outbox", "dwd_order_level", "dwd_item_level", "metric_daily", "metric_dimension_daily"}
	for _, name := range expectedTables {
		_, ok := resp.Database.Tables[name]
		assert.True(t, ok, "expected table %s in response", name)
	}
}
