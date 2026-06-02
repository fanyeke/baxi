package ontology

import (
	"context"
	"os"
	"path/filepath"

	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"baxi/internal/repository/common"
	"baxi/internal/testutil"
)

const ontoDDL = `
CREATE SCHEMA IF NOT EXISTS ai;
CREATE SCHEMA IF NOT EXISTS dwd;
CREATE SCHEMA IF NOT EXISTS mart;
CREATE SCHEMA IF NOT EXISTS ops;
CREATE TABLE IF NOT EXISTS dwd.order_level (
    order_id                TEXT,
    order_status            TEXT,
    order_purchase_timestamp TIMESTAMPTZ,
    payment_value           DOUBLE PRECISION,
    payment_type            TEXT,
    review_score            INT,
    customer_unique_id      TEXT,
    customer_state          TEXT,
    is_late                 BOOLEAN DEFAULT false,
    is_cancelled            BOOLEAN DEFAULT false,
    delivery_status         TEXT
);
CREATE TABLE IF NOT EXISTS dwd.item_level (
    order_id            TEXT,
    product_id          TEXT,
    seller_id           TEXT,
    seller_state        TEXT,
    price               DOUBLE PRECISION,
    freight_value       DOUBLE PRECISION,
    product_category_name       TEXT,
    product_category_name_english TEXT
);
CREATE TABLE IF NOT EXISTS ops.metric_alert (
    alert_id    TEXT PRIMARY KEY,
    rule_id     TEXT,
    metric_name TEXT,
    severity    TEXT,
    current_value DOUBLE PRECISION,
    baseline_value DOUBLE PRECISION,
    status      TEXT,
    owner_role  TEXT,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
`
func setupRepo(t *testing.T) (*Repository, *common.PoolProvider) {
	t.Helper()
	pool := testutil.SetupTestPool(t)
	ctx := context.Background()
	_, err := pool.Exec(ctx, ontoDDL)
	require.NoError(t, err)
	for _, tbl := range []string{"dwd.order_level", "dwd.item_level", "ops.metric_alert"} {
		_, _ = pool.Exec(ctx, "TRUNCATE TABLE "+tbl+" CASCADE")
	}
	return NewRepository(common.NewPoolProvider(pool)), common.NewPoolProvider(pool)
}

func TestOntologyQueryObjects_NoRoleInContext(t *testing.T) {
	repo, _ := setupRepo(t)
	ctx := context.Background()
	f := ObjectFilters{ObjectType: "order", Limit: 10}
	result, err := repo.QueryByObjectType(ctx, "order", f)
	require.Error(t, err)
	// Default role is "analyst", which does not have access to dwd.order_level
	assert.Nil(t, result)
}

func TestOntologyQueryObjects_WithRole(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := WithRole(context.Background(), "admin")
	pool.Exec(ctx, `INSERT INTO dwd.order_level(order_id,order_status,review_score,customer_unique_id) VALUES('ord-1','delivered',4,'cust-1')`)

	f := ObjectFilters{ObjectType: "order", Limit: 10}
	result, err := repo.QueryByObjectType(ctx, "order", f)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.GreaterOrEqual(t, result.Total, 1)
}

func TestOntologyQueryObjects_Seller(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := WithRole(context.Background(), "admin")
	pool.Exec(ctx, `INSERT INTO dwd.item_level(order_id,order_item_id,seller_id,price) VALUES('ord-1',1,'slr-1',100.0)`)

	f := ObjectFilters{ObjectType: "seller", Limit: 10}
	result, err := repo.QueryByObjectType(ctx, "seller", f)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.GreaterOrEqual(t, result.Total, 1)
}

func TestOntologyQueryObjects_UnknownType(t *testing.T) {
	repo, _ := setupRepo(t)
	ctx := WithRole(context.Background(), "admin")
	f := ObjectFilters{ObjectType: "unknown_type", Limit: 10}
	result, err := repo.QueryByObjectType(ctx, "unknown_type", f)
	require.Error(t, err)
	assert.Nil(t, result)
}

func TestOntologySearchObjects(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := WithRole(context.Background(), "admin")
	pool.Exec(ctx, `INSERT INTO dwd.order_level(order_id,order_status,review_score,customer_unique_id) VALUES('ord-search','delivered',5,'cust-2')`)

	f := SearchFilters{ObjectType: "order", Query: "ord-search", Limit: 10}
	result, err := repo.SearchObjects(ctx, "order", f)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.GreaterOrEqual(t, result.Total, 1)
}

func TestOntologyObjectCount(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := WithRole(context.Background(), "admin")
	pool.Exec(ctx, `INSERT INTO dwd.order_daily(order_id,dt) VALUES('o1','2026-01-01')`)
	pool.Exec(ctx, `INSERT INTO dwd.order_daily(order_id,dt) VALUES('o2','2026-01-01')`)

	f := ObjectFilters{ObjectType: "order", Limit: 10}
	result, err := repo.QueryByObjectType(ctx, "order", f)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, 2, result.Total)
}

func TestOntologyResolveLimit(t *testing.T) {
	n, err := resolveLimit(10)
	require.NoError(t, err)
	assert.Equal(t, 10, n)
	n, err = resolveLimit(0)
	require.NoError(t, err)
	assert.Equal(t, 1000, n)
	_, err = resolveLimit(10001)
	require.Error(t, err)
	_, err = resolveLimit(-1)
	require.Error(t, err)
}

func TestOntologyTableAccessible(t *testing.T) {
	assert.True(t, tableAccessible("admin", "ops", "metric_alert"))
	assert.False(t, tableAccessible("viewer", "dwd", "order_daily"))
	assert.False(t, tableAccessible("admin", "dwd", "order_daily"))
	assert.True(t, tableAccessible("admin", "dwd", "order_level"))
	assert.True(t, tableAccessible("viewer", "ops", "metric_alert"))
	assert.False(t, tableAccessible("analyst", "dwd", "order_level"))
}

func TestV2Compiler_StableObjectsCompile(t *testing.T) {
	schemaPath := filepath.Join("..", "..", "..", "config", "aip_object_schema_v2.yml")
	data, err := os.ReadFile(schemaPath)
	require.NoError(t, err)

	var cfg struct {
		Version string `yaml:"version"`
		Objects map[string]struct {
			Maturity    string `yaml:"maturity"`
			DisplayName string `yaml:"display_name"`
			Grain       string `yaml:"grain"`
			Source      struct {
				Schema     string `yaml:"schema"`
				Table      string `yaml:"table"`
				PrimaryKey string `yaml:"primary_key"`
			} `yaml:"source"`
			Properties map[string]struct {
				Type         string `yaml:"type"`
				Source       string `yaml:"source"`
				IsPK         *bool  `yaml:"is_pk,omitempty"`
				Availability string `yaml:"availability,omitempty"`
			} `yaml:"properties"`
		} `yaml:"objects"`
	}
	err = yaml.Unmarshal(data, &cfg)
	require.NoError(t, err)
	require.NotEmpty(t, cfg.Objects, "v2 schema should contain object definitions")

	stableCount := 0
	for name, obj := range cfg.Objects {
		if obj.Maturity != "stable" {
			continue
		}
		stableCount++
		t.Run(name, func(t *testing.T) {
			// source.schema is required for query compilation.
			assert.NotEmpty(t, obj.Source.Schema, "source.schema must be set")
			// source.table is required for query compilation.
			assert.NotEmpty(t, obj.Source.Table, "source.table must be set")
			// source.primary_key is required for get-by-id queries.
			assert.NotEmpty(t, obj.Source.PrimaryKey, "source.primary_key must be set")
			// At least one property must be defined for column selection.
			assert.NotEmpty(t, obj.Properties, "at least one property must be defined")
			// grain is required by schema validation.
			assert.NotEmpty(t, obj.Grain, "grain must be set")

			// Verify at least one non-planned property exists (compiler skips planned).
			hasCompilable := false
			hasPK := false
			for _, prop := range obj.Properties {
				if prop.Availability != "planned" {
					hasCompilable = true
				}
				if prop.IsPK != nil && *prop.IsPK {
					hasPK = true
				}
			}
			assert.True(t, hasCompilable, "at least one non-planned property must exist")
			assert.True(t, hasPK || obj.Source.PrimaryKey != "", "at least one is_pk property or source.primary_key required")
		})
	}

	require.Greater(t, stableCount, 0, "at least one stable object must exist in v2 schema")
}
