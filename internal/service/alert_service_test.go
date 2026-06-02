package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/model"
	"baxi/internal/repository"
)

const svcAlertTableDDL = `
CREATE SCHEMA IF NOT EXISTS ops;

CREATE TABLE IF NOT EXISTS ops.metric_alert (
    alert_id        TEXT PRIMARY KEY,
    rule_id         TEXT NOT NULL,
    event_date      DATE NOT NULL,
    severity        TEXT NOT NULL,
    metric_name     TEXT NOT NULL,
    object_type     TEXT DEFAULT 'global',
    object_id       TEXT DEFAULT 'global',
    current_value   NUMERIC(18,4),
    baseline_value  NUMERIC(18,4),
    change_rate     NUMERIC(10,6),
    sample_size     BIGINT,
    affected_orders BIGINT,
    affected_gmv    NUMERIC(18,2),
    impact_score    NUMERIC(10,6),
    evidence_json   JSONB,
    description     TEXT,
    owner_role      TEXT,
    status          TEXT DEFAULT 'new',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`

func setupSvcAlertTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	ctx := context.Background()
	_, err = pool.Exec(ctx, svcAlertTableDDL)
	require.NoError(t, err)
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE ops.metric_alert CASCADE")
	return pool
}

func insertSvcTestAlert(t *testing.T, pool *pgxpool.Pool, id, rule, severity, metricName, objectType, objectID, status, ownerRole string, eventDate time.Time, currentVal *float64) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO ops.metric_alert (alert_id, rule_id, event_date, severity, metric_name, object_type, object_id, current_value, owner_role, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, id, rule, eventDate, severity, metricName, objectType, objectID, currentVal, ownerRole, status, time.Now())
	require.NoError(t, err)
}

func TestAlertService_ListAlerts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupSvcAlertTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC()
	yesterday := now.Add(-24 * time.Hour)

	insertSvcTestAlert(t, pool, "alert-1", "rule-1", "high", "revenue", "order", "order-1", "new", "admin", yesterday, float64Ptr(100.0))
	insertSvcTestAlert(t, pool, "alert-2", "rule-1", "medium", "revenue", "order", "order-2", "acknowledged", "analyst", now, float64Ptr(50.0))
	insertSvcTestAlert(t, pool, "alert-3", "rule-2", "low", "satisfaction", "customer", "cust-1", "resolved", "manager", now, float64Ptr(3.5))
	insertSvcTestAlert(t, pool, "alert-4", "rule-2", "high", "satisfaction", "customer", "cust-2", "new", "admin", now, float64Ptr(2.0))
	insertSvcTestAlert(t, pool, "alert-5", "rule-3", "critical", "fraud", "order", "order-3", "new", "analyst", now, nil)

	svc := NewAlertService(repository.NewAlertRepository(), pool)

	t.Run("list all alerts", func(t *testing.T) {
		resp, err := svc.ListAlerts(ctx, model.AlertFilters{}, "", 10, 0)
		require.NoError(t, err)
		assert.Equal(t, 5, resp.Total)
		assert.Len(t, resp.Items, 5)
	})

	t.Run("filter by severity", func(t *testing.T) {
		resp, err := svc.ListAlerts(ctx, model.AlertFilters{Severity: "high"}, "", 10, 0)
		require.NoError(t, err)
		assert.Equal(t, 2, resp.Total)
		for _, a := range resp.Items {
			assert.Equal(t, "high", a.Severity)
		}
	})

	t.Run("filter by status", func(t *testing.T) {
		resp, err := svc.ListAlerts(ctx, model.AlertFilters{Status: "new"}, "", 10, 0)
		require.NoError(t, err)
		assert.Equal(t, 3, resp.Total)
	})

	t.Run("filter by object type", func(t *testing.T) {
		resp, err := svc.ListAlerts(ctx, model.AlertFilters{ObjectType: "customer"}, "", 10, 0)
		require.NoError(t, err)
		assert.Equal(t, 2, resp.Total)
	})

	t.Run("filter by rule ID", func(t *testing.T) {
		resp, err := svc.ListAlerts(ctx, model.AlertFilters{RuleID: "rule-2"}, "", 10, 0)
		require.NoError(t, err)
		assert.Equal(t, 2, resp.Total)
	})

	t.Run("sort by severity descending", func(t *testing.T) {
		resp, err := svc.ListAlerts(ctx, model.AlertFilters{}, "severity_desc", 10, 0)
		require.NoError(t, err)
		require.Len(t, resp.Items, 5)
		})
	t.Run("pagination", func(t *testing.T) {
		resp, err := svc.ListAlerts(ctx, model.AlertFilters{}, "", 2, 1)
		require.NoError(t, err)
		assert.Equal(t, 5, resp.Total)
		assert.Len(t, resp.Items, 2)
	})

	t.Run("invalid sort defaults to created_at_desc", func(t *testing.T) {
		resp, err := svc.ListAlerts(ctx, model.AlertFilters{}, "invalid_sort", 10, 0)
		require.NoError(t, err)
		assert.Equal(t, 5, resp.Total)
	})

	t.Run("empty result with filter", func(t *testing.T) {
		resp, err := svc.ListAlerts(ctx, model.AlertFilters{Severity: "nonexistent"}, "", 10, 0)
		require.NoError(t, err)
		assert.Equal(t, 0, resp.Total)
		assert.Empty(t, resp.Items)
	})
}

func TestAlertService_Mapping(t *testing.T) {
	pool := setupSvcAlertTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC()
	currVal := 99.5
	insertSvcTestAlert(t, pool, "alert-map-1", "rule-x", "medium", "unit_cost", "product", "prod-1", "new", "admin", now, &currVal)

	svc := NewAlertService(repository.NewAlertRepository(), pool)
	resp, err := svc.ListAlerts(ctx, model.AlertFilters{}, "", 10, 0)
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)

	a := resp.Items[0]
	assert.Equal(t, "alert-map-1", a.EventID)
	assert.Equal(t, "rule-x", a.RuleID)
	assert.Equal(t, "medium", a.Severity)
	assert.Equal(t, "unit_cost", a.MetricName)
	assert.Equal(t, "product", a.ObjectType)
	assert.Equal(t, "prod-1", a.ObjectID)
	assert.NotNil(t, a.CurrentValue)
	assert.Equal(t, 99.5, *a.CurrentValue)
}

func float64Ptr(v float64) *float64 { return &v }
