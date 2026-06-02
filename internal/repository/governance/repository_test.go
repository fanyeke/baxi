package governance

import (
	"context"

	"baxi/internal/testutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/repository/common"
)

const govDDL = `
CREATE SCHEMA IF NOT EXISTS gov;
CREATE TABLE IF NOT EXISTS gov.config_snapshot (
    id          BIGSERIAL PRIMARY KEY,
    config_key  TEXT NOT NULL,
    config_type TEXT,
    snapshot    JSONB,
    version     TEXT,
    status      TEXT DEFAULT 'active',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS gov.object_schema (
    object_schema_id BIGSERIAL PRIMARY KEY,
    object_type TEXT NOT NULL,
    object_name TEXT,
    schema_jsonb JSONB,
    version TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_object_type_version UNIQUE (object_type, version)
);
CREATE TABLE IF NOT EXISTS gov.data_classification (
    classification_id BIGSERIAL PRIMARY KEY,
    field_path TEXT NOT NULL,
    classification_level TEXT,
    sensitivity_score NUMERIC(4,2),
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
`
func setupRepo(t *testing.T) (*Repository, *common.PoolProvider) {
	t.Helper()
	pool := testutil.SetupTestPool(t)
	ctx := context.Background()
	_, err := pool.Exec(ctx, govDDL)
	require.NoError(t, err)
	for _, tbl := range []string{"gov.config_snapshot", "gov.object_schema", "gov.data_classification", "gov.data_lineage", "gov.access_policy"} {
		_, _ = pool.Exec(ctx, "TRUNCATE TABLE "+tbl+" CASCADE")
	}
	return NewRepository(common.NewPoolProvider(pool)), common.NewPoolProvider(pool)
}

func TestGovGetConfigSnapshots_Empty(t *testing.T) {
	repo, _ := setupRepo(t)
	ctx := context.Background()
	snapshots, err := repo.GetConfigSnapshots(ctx)
	require.NoError(t, err)
	assert.Empty(t, snapshots)
}

func TestGovGetConfigSnapshots(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	pool.Exec(ctx, `INSERT INTO gov.config_snapshot(config_key,config_type) VALUES('object_schema','aip_object_schema')`)
	pool.Exec(ctx, `INSERT INTO gov.config_snapshot(config_key,config_type) VALUES('access_policy','access_policy')`)
	snapshots, err := repo.GetConfigSnapshots(ctx)
	require.NoError(t, err)
	assert.Len(t, snapshots, 2)
}

func TestGovCountTableRows(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	pool.Exec(ctx, `INSERT INTO gov.config_snapshot(config_key,config_type) VALUES('k1','t1')`)
	pool.Exec(ctx, `INSERT INTO gov.config_snapshot(config_key,config_type) VALUES('k2','t2')`)
	n := repo.CountTableRows(ctx, "gov", "config_snapshot")
	assert.Equal(t, 2, n)
}

func TestGovGetObjectSchemas(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	pool.Exec(ctx, `INSERT INTO gov.object_schema(object_type,object_name,version) VALUES('order','Order','v1')`)
	schemas, err := repo.GetObjectSchemas(ctx)
	require.NoError(t, err)
	assert.Len(t, schemas, 1)
	assert.Equal(t, "order", schemas[0].ObjectType)
}

func TestGovCountObjectSchemas(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	pool.Exec(ctx, `INSERT INTO gov.object_schema(object_type,object_name,version) VALUES('o1','O1','v1')`)
	pool.Exec(ctx, `INSERT INTO gov.object_schema(object_type,object_name,version) VALUES('o2','O2','v1')`)
	n := repo.CountObjectSchemas(ctx)
	assert.Equal(t, 2, n)
}

func TestGovGetDataClassifications(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	pool.Exec(ctx, `INSERT INTO gov.data_classification(field_path,classification_level,sensitivity_score,description) VALUES('user.email','confidential',0.8,'Email address of the user')`)
	rows, err := repo.GetDataClassifications(ctx)
	require.NoError(t, err)
	assert.Len(t, rows, 1)
	assert.Equal(t, "user.email", rows[0].FieldPath)
}

func TestGovGetByFieldPath(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	pool.Exec(ctx, `INSERT INTO gov.data_classification(field_path,classification_level,sensitivity_score,description) VALUES('user.name','public',0.1,'User display name')`)
	row, err := repo.GetByFieldPath(ctx, "user.name")
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "public", row.ClassificationLevel)
}

func TestGovGetByFieldPath_NotFound(t *testing.T) {
	repo, _ := setupRepo(t)
	ctx := context.Background()
	row, err := repo.GetByFieldPath(ctx, "nonexistent")
	require.Error(t, err)
	assert.Nil(t, row)
}
