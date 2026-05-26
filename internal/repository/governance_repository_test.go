package repository

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const govTableDDL = `
CREATE SCHEMA IF NOT EXISTS gov;

CREATE TABLE IF NOT EXISTS gov.config_snapshot (
    snapshot_id  BIGSERIAL PRIMARY KEY,
    config_key   TEXT NOT NULL,
    config_type  TEXT,
    source_path  TEXT,
    content_jsonb JSONB,
    content_hash TEXT,
    loaded_at    TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_config_key_hash UNIQUE (config_key, content_hash)
);
`

func setupGovTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	ctx := context.Background()
	_, err = pool.Exec(ctx, govTableDDL)
	require.NoError(t, err)

	_, _ = pool.Exec(ctx, "TRUNCATE TABLE gov.config_snapshot CASCADE")
	return pool
}

func insertTestConfigSnapshot(t *testing.T, pool *pgxpool.Pool, configKey string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO gov.config_snapshot (config_key, config_type, source_path, content_jsonb)
		VALUES ($1, 'yaml', $1, '{}'::jsonb)
	`, configKey)
	require.NoError(t, err)
}

func TestGovernanceRepository_GetConfigSnapshots_WithData(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovTestDB(t)
	ctx := context.Background()

	insertTestConfigSnapshot(t, pool, "data_catalog.yml")
	insertTestConfigSnapshot(t, pool, "access_policy.yml")
	insertTestConfigSnapshot(t, pool, "health_checks.yml")

	repo := NewGovernanceRepository()
	configs, err := repo.GetConfigSnapshots(ctx, pool)
	require.NoError(t, err)
	assert.Len(t, configs, 3)

	keys := make(map[string]string)
	for _, c := range configs {
		keys[c.ConfigKey] = c.Status
	}
	assert.Equal(t, "loaded", keys["data_catalog.yml"])
	assert.Equal(t, "loaded", keys["access_policy.yml"])
	assert.Equal(t, "loaded", keys["health_checks.yml"])
}

func TestGovernanceRepository_GetConfigSnapshots_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovTestDB(t)
	ctx := context.Background()

	repo := NewGovernanceRepository()
	configs, err := repo.GetConfigSnapshots(ctx, pool)
	require.NoError(t, err)
	assert.Empty(t, configs)
}

func TestGovernanceRepository_GetConfigSnapshots_Ordered(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovTestDB(t)
	ctx := context.Background()

	insertTestConfigSnapshot(t, pool, "zzz_later.yml")
	insertTestConfigSnapshot(t, pool, "aaa_first.yml")
	insertTestConfigSnapshot(t, pool, "mmm_middle.yml")

	repo := NewGovernanceRepository()
	configs, err := repo.GetConfigSnapshots(ctx, pool)
	require.NoError(t, err)
	require.Len(t, configs, 3)

	assert.Equal(t, "aaa_first.yml", configs[0].ConfigKey)
	assert.Equal(t, "mmm_middle.yml", configs[1].ConfigKey)
	assert.Equal(t, "zzz_later.yml", configs[2].ConfigKey)
}

func TestGovernanceRepository_CountTableRows_ExistingTable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovTestDB(t)
	ctx := context.Background()

	insertTestConfigSnapshot(t, pool, "test_config.yml")

	repo := NewGovernanceRepository()
	count := repo.CountTableRows(ctx, pool, "gov", "config_snapshot")
	assert.Equal(t, 1, count)
}

func TestGovernanceRepository_CountTableRows_MissingTable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovTestDB(t)
	ctx := context.Background()

	repo := NewGovernanceRepository()
	count := repo.CountTableRows(ctx, pool, "gov", "nonexistent_table")
	assert.Equal(t, 0, count)
}
