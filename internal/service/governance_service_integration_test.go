package service

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/repository/common"
	governanceRepo "baxi/internal/repository/governance"
)

const svcGovDDL = `
CREATE SCHEMA IF NOT EXISTS gov;

CREATE TABLE IF NOT EXISTS gov.config_snapshot (
    config_key TEXT PRIMARY KEY,
    status     TEXT DEFAULT 'loaded'
);

CREATE TABLE IF NOT EXISTS gov.object_schema (
    object_type  TEXT PRIMARY KEY,
    object_name  TEXT NOT NULL,
    schema_jsonb JSONB DEFAULT '{}',
    version      TEXT DEFAULT '1.0'
);

CREATE TABLE IF NOT EXISTS gov.data_classification (
    field_path           TEXT PRIMARY KEY,
    classification_level TEXT NOT NULL,
    sensitivity_score    NUMERIC(10,4) DEFAULT 0,
    description          TEXT DEFAULT ''
);

CREATE TABLE IF NOT EXISTS gov.data_lineage (
    id                  SERIAL PRIMARY KEY,
    source_table        TEXT NOT NULL,
    source_column       TEXT NOT NULL,
    target_table        TEXT NOT NULL,
    target_column       TEXT NOT NULL,
    transformation_logic TEXT DEFAULT '',
    confidence          NUMERIC(10,6) DEFAULT 0.9
);

CREATE TABLE IF NOT EXISTS gov.access_policy (
    policy_name       TEXT PRIMARY KEY,
    resource_type     TEXT NOT NULL,
    resource_pattern  TEXT NOT NULL,
    action            TEXT NOT NULL,
    principal_type    TEXT DEFAULT 'role',
    principal_pattern TEXT NOT NULL,
    effect            TEXT NOT NULL,
    conditions_jsonb  JSONB DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS gov.checkpoint_rule (
    rule_id               TEXT PRIMARY KEY,
    action                TEXT NOT NULL,
    requires_reason       BOOLEAN DEFAULT false,
    requires_human_review BOOLEAN DEFAULT false
);
`

func setupGovTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping governance integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	ctx := context.Background()
	_, err = pool.Exec(ctx, svcGovDDL)
	require.NoError(t, err)

	for _, tbl := range []string{
		"gov.config_snapshot",
		"gov.object_schema",
		"gov.data_classification",
		"gov.data_lineage",
		"gov.access_policy",
		"gov.checkpoint_rule",
	} {
		_, _ = pool.Exec(ctx, "TRUNCATE "+tbl+" CASCADE")
	}

	return pool
}

func TestGovernanceService_GetStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovTestDB(t)
	ctx := context.Background()

	// Insert test data
	_, err := pool.Exec(ctx, `INSERT INTO gov.config_snapshot (config_key) VALUES ('data_classification'), ('data_lineage')`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO gov.object_schema (object_type, object_name) VALUES ('order', 'Order'), ('customer', 'Customer')`)
	require.NoError(t, err)

	
	provider := common.NewPoolProvider(pool)
	repo := governanceRepo.NewRepository(provider)
	svc := NewGovernanceService(repo, pool)

	resp, err := svc.GetStatus(ctx)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "active", resp.GovernanceLayer)
	assert.Equal(t, 2, len(resp.Configs))
	assert.Equal(t, "loaded", resp.Configs["data_classification"])
	assert.Equal(t, "loaded", resp.Configs["data_lineage"])
	assert.Equal(t, 2, resp.ObjectSchemaCount)
}

func TestGovernanceService_GetClassification(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovTestDB(t)
	ctx := context.Background()

	_, err := pool.Exec(ctx, `INSERT INTO gov.data_classification (field_path, classification_level, sensitivity_score, description) VALUES
		('customer.email', 'pii', 0.95, 'Customer email address'),
		('customer.name', 'internal', 0.3, 'Customer name'),
		('order.amount', 'sensitive', 0.8, 'Order amount')`)
	require.NoError(t, err)

	
	provider := common.NewPoolProvider(pool)
	repo := governanceRepo.NewRepository(provider)
	svc := NewGovernanceService(repo, pool)

	t.Run("all classifications", func(t *testing.T) {
		resp, err := svc.GetClassification(ctx, "")
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, 3, len(resp.Resources))
		// pii→L3, internal→L2, sensitive→L3 (unique levels: L3, L2)
		assert.Contains(t, resp.Levels, "L3")
		assert.Contains(t, resp.Levels, "L2")
	})

	t.Run("filter by field path", func(t *testing.T) {
		resp, err := svc.GetClassification(ctx, "customer.email")
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Resources, 1)
		assert.Equal(t, "customer.email", resp.Resources[0].Resource)
		assert.Equal(t, "L3", resp.Resources[0].Classification)
	})

	t.Run("unknown field path returns default", func(t *testing.T) {
		resp, err := svc.GetClassification(ctx, "unknown.field")
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Resources, 1)
		assert.Equal(t, "unknown.field", resp.Resources[0].Resource)
		assert.Equal(t, "L2", resp.Resources[0].Classification) // default is internal→L2
	})
}

func TestGovernanceService_GetFieldMarking(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovTestDB(t)
	ctx := context.Background()

	_, err := pool.Exec(ctx, `INSERT INTO gov.data_classification (field_path, classification_level, sensitivity_score, description) VALUES
		('customer.email', 'pii', 0.95, 'Customer email'),
		('customer.name', 'internal', 0.3, 'Customer name'),
		('seller.rating', 'sensitive', 0.75, 'Seller rating'),
		('order.amount', 'public_internal', 0.2, 'Order amount')`)
	require.NoError(t, err)

	
	provider := common.NewPoolProvider(pool)
	repo := governanceRepo.NewRepository(provider)
	svc := NewGovernanceService(repo, pool)

	t.Run("all markings when no filter", func(t *testing.T) {
		resp, err := svc.GetFieldMarking(ctx, "", "")
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Markings, 4)
	})

	t.Run("filter by object type and property", func(t *testing.T) {
		resp, err := svc.GetFieldMarking(ctx, "customer", "email")
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp.Markings, 1)
		assert.Equal(t, "customer", resp.Markings[0].ObjectType)
		assert.Equal(t, "email", resp.Markings[0].Field)
		assert.Equal(t, "L3", resp.Markings[0].Classification)
		assert.True(t, resp.Markings[0].PII)
		assert.False(t, resp.Markings[0].LLMAllowed) // L3 → LLM not allowed
	})
}

func TestGovernanceService_GetCatalog(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovTestDB(t)
	ctx := context.Background()

	_, err := pool.Exec(ctx, `INSERT INTO gov.object_schema (object_type, object_name, version) VALUES
		('customer', 'Customer', '1.0'),
		('order', 'Order', '1.1'),
		('product', 'Product', '1.0')`)
	require.NoError(t, err)

	
	provider := common.NewPoolProvider(pool)
	repo := governanceRepo.NewRepository(provider)
	svc := NewGovernanceService(repo, pool)

	resp, err := svc.GetCatalog(ctx)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Len(t, resp.Objects, 3)
	assert.Len(t, resp.Datasets, 3)

	// Verify object mappings
	objMap := make(map[string]string)
	for _, o := range resp.Objects {
		objMap[o.ObjectType] = o.SourceDataset
	}
	assert.Equal(t, "olist_customers", objMap["customer"])
	assert.Equal(t, "olist_orders", objMap["order"])
	assert.Equal(t, "olist_products", objMap["product"])
}

func TestGovernanceService_GetHealthChecks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovTestDB(t)
	ctx := context.Background()

	// Create gov tables with data
	_, err := pool.Exec(ctx, `INSERT INTO gov.config_snapshot (config_key) VALUES ('test')`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO gov.data_classification (field_path, classification_level) VALUES ('test.field', 'internal')`)
	require.NoError(t, err)

	
	provider := common.NewPoolProvider(pool)
	repo := governanceRepo.NewRepository(provider)
	svc := NewGovernanceService(repo, pool)

	resp, err := svc.GetHealthChecks(ctx)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "healthy", resp.Status)
	assert.Len(t, resp.Checks, 5)

	checkMap := make(map[string]string)
	for _, c := range resp.Checks {
		checkMap[c.Name] = c.Status
	}
	assert.Equal(t, "healthy", checkMap["config_snapshot"])
	assert.Equal(t, "healthy", checkMap["data_classification"])
	assert.Equal(t, "unknown", checkMap["data_lineage"])   // empty table
	assert.Equal(t, "unknown", checkMap["access_policy"])  // empty table
	assert.Equal(t, "unknown", checkMap["object_schema"])  // empty table
}

func TestGovernanceService_GetLineage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovTestDB(t)
	ctx := context.Background()

	_, err := pool.Exec(ctx, `INSERT INTO gov.data_lineage (source_table, source_column, target_table, target_column) VALUES
		('raw.orders', 'order_id', 'dwd.dwd_order_level', 'order_id'),
		('raw.orders', 'seller_id', 'dwd.dwd_order_level', 'seller_id'),
		('dwd.dwd_order_level', 'order_id', 'metric.metric_daily', 'order_id')`)
	require.NoError(t, err)

	
	provider := common.NewPoolProvider(pool)
	repo := governanceRepo.NewRepository(provider)
	svc := NewGovernanceService(repo, pool)

	resp, err := svc.GetLineage(ctx, "dwd.dwd_order_level")
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "dwd.dwd_order_level", resp.Resource)
	assert.Contains(t, resp.Upstream, "raw.orders")
	assert.Contains(t, resp.Downstream, "metric.metric_daily")
}
