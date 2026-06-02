package alert

import (
	"context"

	"baxi/internal/testutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/repository/common"
)

const alertDDL = `
CREATE SCHEMA IF NOT EXISTS ops;
CREATE TABLE IF NOT EXISTS ops.metric_alert (
    alert_id     TEXT PRIMARY KEY,
    rule_id      TEXT NOT NULL,
    event_date   DATE NOT NULL,
    severity     TEXT NOT NULL,
    metric_name  TEXT NOT NULL,
    object_type  TEXT DEFAULT 'global',
    object_id    TEXT DEFAULT 'global',
    current_value NUMERIC(18,4),
    baseline_value NUMERIC(18,4),
    change_rate  NUMERIC(10,6),
    impact_score NUMERIC(10,6),
    owner_role   TEXT,
    status       TEXT DEFAULT 'new',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`
func setupRepo(t *testing.T) (*Repository, *common.PoolProvider) {
	t.Helper()
	pool := testutil.SetupTestPool(t)
	ctx := context.Background()
	_, err := pool.Exec(ctx, alertDDL)
	require.NoError(t, err)
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE ops.metric_alert CASCADE")
	return NewRepository(common.NewPoolProvider(pool)), common.NewPoolProvider(pool)
}

func ins(t *testing.T, pool *common.PoolProvider, id, rule, date, sev, metric, status string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO ops.metric_alert(alert_id,rule_id,event_date,severity,metric_name,status,owner_role)
		VALUES($1,$2,$3,$4,$5,$6,$7)`, id, rule, date, sev, metric, status, "owner")
	require.NoError(t, err)
}

func TestAlertListAlerts(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	ins(t, pool, "a1", "r1", "2026-01-01", "high", "m1", "new")
	ins(t, pool, "a2", "r2", "2026-01-02", "med", "m2", "new")
	rows, total, err := repo.ListAlerts(ctx, "", "", "", "", "", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, rows, 2)
}

func TestAlertFilterByStatus(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	ins(t, pool, "a1", "r1", "2026-01-01", "high", "m1", "new")
	ins(t, pool, "a2", "r2", "2026-01-02", "med", "m2", "resolved")
	rows, total, err := repo.ListAlerts(ctx, "", "new", "", "", "", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "a1", rows[0].AlertID)
}

func TestAlertSortBySeverity(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	ins(t, pool, "a1", "r1", "2026-01-01", "low", "m1", "new")
	ins(t, pool, "a2", "r2", "2026-01-02", "high", "m2", "new")
	rows, _, err := repo.ListAlerts(ctx, "", "", "", "", "severity_desc", 10, 0)
	require.NoError(t, err)
	assert.Len(t, rows, 2)
	assert.Equal(t, "a1", rows[0].AlertID) // "low" > "high" alphabetically with DESC
}

func TestAlertGetByID(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	ins(t, pool, "a1", "r1", "2026-01-01", "high", "m1", "new")
	a, err := repo.GetAlertByID(ctx, "a1")
	require.NoError(t, err)
	require.NotNil(t, a)
	assert.Equal(t, "high", a.Severity)
}

func TestAlertGetByIDNotFound(t *testing.T) {
	repo, _ := setupRepo(t)
	ctx := context.Background()
	a, err := repo.GetAlertByID(ctx, "missing")
	assert.Error(t, err)
	assert.Nil(t, a)
}
