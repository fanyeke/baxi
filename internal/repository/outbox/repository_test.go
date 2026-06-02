package outbox

import (
	"context"

	"baxi/internal/testutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/repository/common"
)

const outboxTableDDL = `
CREATE SCHEMA IF NOT EXISTS ops;
CREATE TABLE IF NOT EXISTS ops.outbox_event (
    event_id          TEXT PRIMARY KEY,
    source_type       TEXT NOT NULL DEFAULT '',
    source_id         TEXT NOT NULL DEFAULT '',
    event_type        TEXT NOT NULL,
    status            TEXT DEFAULT 'pending',
    payload_json      JSONB NOT NULL DEFAULT '{}',
    target_channel    TEXT NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    dispatch_attempts BIGINT DEFAULT 0,
    next_retry_at     TIMESTAMPTZ,
    last_dispatch_at  TIMESTAMPTZ,
    error_message     TEXT,
    processed_at      TIMESTAMPTZ,
    external_ref      TEXT,
    adapter_name      TEXT
);
`
func setupRepo(t *testing.T) (*Repository, *common.PoolProvider) {
	t.Helper()
	pool := testutil.SetupTestPool(t)
	ctx := context.Background()
	_, err := pool.Exec(ctx, outboxTableDDL)
	require.NoError(t, err)
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE ops.outbox_event CASCADE")
	return NewRepository(common.NewPoolProvider(pool)), common.NewPoolProvider(pool)
}

func TestOutboxListOutboxEvents(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		id := "ev-" + itoa(i)
		_, err := pool.Exec(ctx, `INSERT INTO ops.outbox_event(event_id,event_type,source_type,source_id,payload_json,target_channel,status) VALUES($1,'t','s',$1,'{}','cli','pending')`, id)
		require.NoError(t, err)
	}
	rows, total, err := repo.ListOutboxEvents(ctx, OutboxFilters{}, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, rows, 3)
}

func TestOutboxFilterByStatus(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	_, err := pool.Exec(ctx, `INSERT INTO ops.outbox_event(event_id,event_type,source_type,source_id,payload_json,target_channel,status) VALUES('e1','t','s','s1','{}','cli','pending')`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO ops.outbox_event(event_id,event_type,source_type,source_id,payload_json,target_channel,status) VALUES('e2','t','s','s2','{}','cli','dispatched')`)
	require.NoError(t, err)
	f := OutboxFilters{Status: strPtr("pending")}
	rows, total, err := repo.ListOutboxEvents(ctx, f, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, rows, 1)
	assert.Equal(t, "e1", rows[0].OutboxID)
}

func TestOutboxGetDetail(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	_, err := pool.Exec(ctx, `INSERT INTO ops.outbox_event(event_id,event_type,source_type,source_id,payload_json,target_channel,status) VALUES('ev1','t','s','sid','{}','cli','pending')`)
	require.NoError(t, err)
	d, err := repo.GetDetail(ctx, "ev1")
	require.NoError(t, err)
	require.NotNil(t, d)
	assert.Equal(t, "ev1", d.EventID)
}

func TestOutboxGetDetailNotFound(t *testing.T) {
	repo, _ := setupRepo(t)
	ctx := context.Background()
	d, err := repo.GetDetail(ctx, "missing")
	require.NoError(t, err)
	assert.Nil(t, d)
}

func TestOutboxLimit(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		_, err := pool.Exec(ctx, `INSERT INTO ops.outbox_event(event_id,event_type,source_type,source_id,payload_json,target_channel,status) VALUES($1,'t','s',$1,'{}','cli','pending')`, "lev-"+itoa(i))
		require.NoError(t, err)
	}
	rows, total, err := repo.ListOutboxEvents(ctx, OutboxFilters{}, 2, 0)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, rows, 2)
}

func strPtr(s string) *string { return &s }
func itoa(n int) string {
	if n == 0 { return "0" }
	s := ""
	for n > 0 { s = string(rune('0'+n%10)) + s; n /= 10 }
	return s
}
