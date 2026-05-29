//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/testutil"
)

const ontologyTableDDL = `
CREATE SCHEMA IF NOT EXISTS dwd;

CREATE TABLE IF NOT EXISTS dwd.order_level (
    order_id                TEXT PRIMARY KEY,
    customer_unique_id      TEXT,
    customer_state          TEXT,
    customer_city           TEXT,
    order_status            TEXT,
    total_payment_value     DOUBLE PRECISION,
    payment_value           DOUBLE PRECISION,
    payment_type            TEXT,
    review_score            INTEGER,
    order_purchase_timestamp TIMESTAMPTZ,
    delivery_status         TEXT,
    seller_state            TEXT
);

CREATE TABLE IF NOT EXISTS dwd.item_level (
    item_key                     TEXT PRIMARY KEY,
    order_id                     TEXT,
    product_id                   TEXT,
    seller_id                    TEXT,
    seller_state                 TEXT,
    seller_city                  TEXT,
    product_category_name        TEXT,
    product_category_name_english TEXT,
    price                        DOUBLE PRECISION,
    freight_value                DOUBLE PRECISION,
    product_weight_g             DOUBLE PRECISION,
    review_score                 INTEGER
);

CREATE SCHEMA IF NOT EXISTS raw;

CREATE TABLE IF NOT EXISTS raw.marketing_qualified_leads (
    mql_id            TEXT PRIMARY KEY,
    first_contact_date TEXT,
    landing_page      TEXT,
    origin            TEXT
);

CREATE SCHEMA IF NOT EXISTS ops;

CREATE TABLE IF NOT EXISTS ops.metric_alert (
    alert_id       TEXT PRIMARY KEY,
    rule_id        TEXT,
    metric         TEXT,
    severity       TEXT,
    current_value  DOUBLE PRECISION,
    baseline_value DOUBLE PRECISION,
    status         TEXT
);
`

func setupOntologyTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	pool, err := pgxpool.New(ctx, pg.ConnectionString())
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	_, err = pool.Exec(ctx, ontologyTableDDL)
	require.NoError(t, err)

	return pool
}

func insertOrderLevelRow(t *testing.T, pool *pgxpool.Pool, row map[string]interface{}) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO dwd.order_level
			(order_id, customer_unique_id, customer_state, customer_city, order_status,
			 total_payment_value, payment_value, payment_type, review_score,
			 order_purchase_timestamp, delivery_status, seller_state)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, row["order_id"], row["customer_unique_id"], row["customer_state"],
		row["customer_city"], row["order_status"], row["total_payment_value"],
		row["payment_value"], row["payment_type"], row["review_score"],
		row["order_purchase_timestamp"], row["delivery_status"], row["seller_state"])
	require.NoError(t, err)
}

func insertItemLevelRow(t *testing.T, pool *pgxpool.Pool, row map[string]interface{}) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO dwd.item_level
			(item_key, order_id, product_id, seller_id, seller_state, seller_city,
			 product_category_name, product_category_name_english,
			 price, freight_value, product_weight_g, review_score)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, row["item_key"], row["order_id"], row["product_id"],
		row["seller_id"], row["seller_state"], row["seller_city"],
		row["product_category_name"], row["product_category_name_english"],
		row["price"], row["freight_value"], row["product_weight_g"],
		row["review_score"])
	require.NoError(t, err)
}

func insertMarketingLeadRow(t *testing.T, pool *pgxpool.Pool, mqlID, landingPage, origin string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO raw.marketing_qualified_leads (mql_id, first_contact_date, landing_page, origin)
		VALUES ($1, $2, $3, $4)
	`, mqlID, "2024-01-15", landingPage, origin)
	require.NoError(t, err)
}

func insertMetricAlertRow(t *testing.T, pool *pgxpool.Pool, alertID, ruleID, metric, severity, status string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO ops.metric_alert (alert_id, rule_id, metric, severity, current_value, baseline_value, status)
		VALUES ($1, $2, $3, $4, 100.0, 50.0, $5)
	`, alertID, ruleID, metric, severity, status)
	require.NoError(t, err)
}

func TestOntologyRepo_QueryByObjectType_Customer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOntologyTestDB(t)
	ctx := context.Background()
	repo := NewOntologyRepo()

	insertOrderLevelRow(t, pool, map[string]interface{}{
		"order_id":                 "o-1",
		"customer_unique_id":       "cust-1",
		"customer_state":           "SP",
		"customer_city":            "Sao Paulo",
		"order_status":             "delivered",
		"total_payment_value":      150.0,
		"payment_value":            150.0,
		"payment_type":             "credit_card",
		"review_score":             5,
		"order_purchase_timestamp": "2024-01-10T00:00:00Z",
		"delivery_status":          "on_time",
		"seller_state":             "RJ",
	})

	result, err := repo.QueryByObjectType(ctx, pool, "customer", ObjectFilters{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Rows, 1)
	assert.Equal(t, "cust-1", result.Rows[0].ID)
	assert.Equal(t, "customer", result.Rows[0].ObjectType)
}

func TestOntologyRepo_QueryByObjectType_WithFilters(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOntologyTestDB(t)
	ctx := context.Background()

	insertOrderLevelRow(t, pool, map[string]interface{}{
		"order_id": "o-1", "customer_unique_id": "cust-1",
		"customer_state": "SP", "order_status": "delivered",
		"total_payment_value": 100.0, "payment_value": 100.0,
		"review_score": 5, "order_purchase_timestamp": "2024-01-10T00:00:00Z",
	})
	insertOrderLevelRow(t, pool, map[string]interface{}{
		"order_id": "o-2", "customer_unique_id": "cust-2",
		"customer_state": "RJ", "order_status": "canceled",
		"total_payment_value": 50.0, "payment_value": 50.0,
		"review_score": 2, "order_purchase_timestamp": "2024-01-11T00:00:00Z",
	})

	repo := NewOntologyRepo()
	result, err := repo.QueryByObjectType(ctx, pool, "order", ObjectFilters{
		Limit:   10,
		Filters: map[string]interface{}{"order_status": "canceled"},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Rows, 1)
	assert.Equal(t, "o-2", result.Rows[0].ID)
}

func TestOntologyRepo_QueryByObjectType_Pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOntologyTestDB(t)
	ctx := context.Background()

	for i := 1; i <= 5; i++ {
		insertOrderLevelRow(t, pool, map[string]interface{}{
			"order_id": fmt.Sprintf("o-%d", i), "customer_unique_id": fmt.Sprintf("cust-%d", i),
			"customer_state": "SP", "order_status": "delivered",
			"total_payment_value": float64(i * 10), "payment_value": float64(i * 10),
			"review_score": i, "order_purchase_timestamp": "2024-01-10T00:00:00Z",
		})
	}

	repo := NewOntologyRepo()

	// Limit 2
	result, err := repo.QueryByObjectType(ctx, pool, "customer", ObjectFilters{Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, result.Total)
	assert.Len(t, result.Rows, 2)

	// Offset 3
	result2, err := repo.QueryByObjectType(ctx, pool, "customer", ObjectFilters{Limit: 10, Offset: 3})
	require.NoError(t, err)
	assert.Equal(t, 5, result2.Total)
	assert.Len(t, result2.Rows, 2)
}

func TestOntologyRepo_QueryByObjectType_UnknownType(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOntologyTestDB(t)
	repo := NewOntologyRepo()

	_, err := repo.QueryByObjectType(context.Background(), pool, "nonexistent", ObjectFilters{})
	assert.ErrorContains(t, err, "unknown object type")
}

func TestOntologyRepo_QueryByObjectType_InvalidLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOntologyTestDB(t)
	repo := NewOntologyRepo()

	_, err := repo.QueryByObjectType(context.Background(), pool, "customer", ObjectFilters{Limit: -1})
	assert.ErrorContains(t, err, "invalid limit")
}

func TestOntologyRepo_QueryByObjectType_ExceedsMaxLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOntologyTestDB(t)
	repo := NewOntologyRepo()

	_, err := repo.QueryByObjectType(context.Background(), pool, "customer", ObjectFilters{Limit: 99999})
	assert.ErrorContains(t, err, "exceeds maximum")
}

func TestOntologyRepo_QueryByObjectType_EmptyTable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOntologyTestDB(t)
	repo := NewOntologyRepo()

	result, err := repo.QueryByObjectType(context.Background(), pool, "customer", ObjectFilters{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 0, result.Total)
	assert.Empty(t, result.Rows)
}

// ──── Tests: GetObjectByID ────────────────────────────────────────────────

func TestOntologyRepo_GetObjectByID_Found(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOntologyTestDB(t)
	ctx := context.Background()
	repo := NewOntologyRepo()

	insertOrderLevelRow(t, pool, map[string]interface{}{
		"order_id": "o-42", "customer_unique_id": "cust-99",
		"customer_state": "MG", "order_status": "shipped",
		"total_payment_value": 200.0, "payment_value": 200.0,
		"review_score": 4, "order_purchase_timestamp": "2024-02-01T00:00:00Z",
	})

	obj, err := repo.GetObjectByID(ctx, pool, "order", "o-42")
	require.NoError(t, err)
	assert.Equal(t, "order", obj.ObjectType)
	assert.Equal(t, "o-42", obj.ID)
}

func TestOntologyRepo_GetObjectByID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOntologyTestDB(t)
	repo := NewOntologyRepo()

	_, err := repo.GetObjectByID(context.Background(), pool, "seller", "nonexistent")
	assert.ErrorContains(t, err, "not found")
}

func TestOntologyRepo_GetObjectByID_UnknownType(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOntologyTestDB(t)
	repo := NewOntologyRepo()

	_, err := repo.GetObjectByID(context.Background(), pool, "invalid_type", "x")
	assert.ErrorContains(t, err, "unknown object type")
}

func TestOntologyRepo_GetObjectByID_MarketingLead(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOntologyTestDB(t)
	ctx := context.Background()
	repo := NewOntologyRepo()

	insertMarketingLeadRow(t, pool, "mql-1", "/landing", "organic")

	obj, err := repo.GetObjectByID(ctx, pool, "marketing_lead", "mql-1")
	require.NoError(t, err)
	assert.Equal(t, "marketing_lead", obj.ObjectType)
	assert.Equal(t, "mql-1", obj.ID)
}

func TestOntologyRepo_GetObjectByID_MetricAlert(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOntologyTestDB(t)
	ctx := context.Background()
	repo := NewOntologyRepo()

	insertMetricAlertRow(t, pool, "alert-1", "rule-1", "gmv", "critical", "open")

	obj, err := repo.GetObjectByID(ctx, pool, "metric_alert", "alert-1")
	require.NoError(t, err)
	assert.Equal(t, "metric_alert", obj.ObjectType)
	assert.Equal(t, "alert-1", obj.ID)
}

// ──── Tests: GetObjectMetrics ─────────────────────────────────────────────

func TestOntologyRepo_GetObjectMetrics_Seller(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOntologyTestDB(t)
	ctx := context.Background()
	repo := NewOntologyRepo()

	insertItemLevelRow(t, pool, map[string]interface{}{
		"item_key": "ik-1", "order_id": "o-1", "product_id": "p-1",
		"seller_id": "seller-1", "seller_state": "SP", "seller_city": "Sao Paulo",
		"price": 100.0, "freight_value": 10.0, "product_weight_g": 500.0,
		"review_score": 4,
	})
	insertItemLevelRow(t, pool, map[string]interface{}{
		"item_key": "ik-2", "order_id": "o-2", "product_id": "p-2",
		"seller_id": "seller-1", "seller_state": "SP", "seller_city": "Sao Paulo",
		"price": 50.0, "freight_value": 5.0, "product_weight_g": 200.0,
		"review_score": 5,
	})

	metrics, err := repo.GetObjectMetrics(ctx, pool, "seller", "seller-1")
	require.NoError(t, err)
	assert.Equal(t, "seller-1", metrics.ID)
	assert.Equal(t, "seller", metrics.ObjectType)
	assert.Contains(t, metrics.Metrics, "total_sales")
	assert.Contains(t, metrics.Metrics, "total_items")
	assert.Contains(t, metrics.Metrics, "avg_review_score")
}

func TestOntologyRepo_GetObjectMetrics_Product(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOntologyTestDB(t)
	ctx := context.Background()
	repo := NewOntologyRepo()

	insertItemLevelRow(t, pool, map[string]interface{}{
		"item_key": "ik-1", "order_id": "o-1", "product_id": "p-1",
		"seller_id": "s-1", "product_category_name": "electronics",
		"product_category_name_english": "electronics",
		"price":                         200.0, "freight_value": 15.0, "product_weight_g": 1000.0,
		"review_score": 3,
	})

	metrics, err := repo.GetObjectMetrics(ctx, pool, "product", "p-1")
	require.NoError(t, err)
	assert.Contains(t, metrics.Metrics, "total_sold")
	assert.Contains(t, metrics.Metrics, "avg_price")
	assert.Contains(t, metrics.Metrics, "avg_freight")
	assert.Contains(t, metrics.Metrics, "avg_review_score")
}

func TestOntologyRepo_GetObjectMetrics_UnknownType(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOntologyTestDB(t)
	repo := NewOntologyRepo()

	_, err := repo.GetObjectMetrics(context.Background(), pool, "nonexistent", "x")
	assert.ErrorContains(t, err, "unknown object type")
}

// ──── Tests: SearchObjects ────────────────────────────────────────────────

func TestOntologyRepo_SearchObjects_Found(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOntologyTestDB(t)
	ctx := context.Background()
	repo := NewOntologyRepo()

	insertOrderLevelRow(t, pool, map[string]interface{}{
		"order_id": "o-abc", "customer_unique_id": "cust-abc",
		"customer_state": "SP", "customer_city": "Sao Paulo",
		"order_status":        "delivered",
		"total_payment_value": 100.0, "payment_value": 100.0,
		"review_score": 5, "order_purchase_timestamp": "2024-01-10T00:00:00Z",
	})

	result, err := repo.SearchObjects(ctx, pool, "customer", SearchFilters{
		Query: "abc", Limit: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Rows, 1)
}

func TestOntologyRepo_SearchObjects_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOntologyTestDB(t)
	ctx := context.Background()
	repo := NewOntologyRepo()

	insertOrderLevelRow(t, pool, map[string]interface{}{
		"order_id": "o-1", "customer_unique_id": "cust-1",
		"customer_state": "SP", "customer_city": "Sao Paulo",
		"order_status":        "delivered",
		"total_payment_value": 100.0, "payment_value": 100.0,
		"review_score": 5, "order_purchase_timestamp": "2024-01-10T00:00:00Z",
	})

	result, err := repo.SearchObjects(ctx, pool, "customer", SearchFilters{
		Query: "ZZZNOTFOUND", Limit: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, result.Total)
	assert.Empty(t, result.Rows)
}

func TestOntologyRepo_SearchObjects_EmptyQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOntologyTestDB(t)
	ctx := context.Background()
	repo := NewOntologyRepo()

	insertOrderLevelRow(t, pool, map[string]interface{}{
		"order_id": "o-1", "customer_unique_id": "cust-1",
		"customer_state":      "SP",
		"order_status":        "delivered",
		"total_payment_value": 100.0, "payment_value": 100.0,
		"review_score": 5, "order_purchase_timestamp": "2024-01-10T00:00:00Z",
	})

	// Empty query should return all
	result, err := repo.SearchObjects(ctx, pool, "customer", SearchFilters{Query: "", Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
}

// ──── Tests: RBAC (WithRole) ──────────────────────────────────────────────

func TestOntologyRepo_RBAC_RoleAccessDenied(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOntologyTestDB(t)
	repo := NewOntologyRepo()

	// viewer role should NOT have access to dwd.order_level (customer objects)
	ctx := WithRole(context.Background(), "viewer")
	_, err := repo.QueryByObjectType(ctx, pool, "customer", ObjectFilters{Limit: 10})
	assert.ErrorContains(t, err, "does not have access")
}

func TestOntologyRepo_RBAC_RoleAccessAllowed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOntologyTestDB(t)
	ctx := WithRole(context.Background(), "viewer")

	insertMetricAlertRow(t, pool, "alert-1", "rule-1", "gmv", "critical", "open")

	repo := NewOntologyRepo()
	// viewer has access to ops.metric_alert
	result, err := repo.QueryByObjectType(ctx, pool, "metric_alert", ObjectFilters{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
}

func TestOntologyRepo_RBAC_AdminHasFullAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOntologyTestDB(t)
	ctx := WithRole(context.Background(), "admin")

	insertOrderLevelRow(t, pool, map[string]interface{}{
		"order_id": "o-1", "customer_unique_id": "cust-1",
		"customer_state":      "SP",
		"order_status":        "delivered",
		"total_payment_value": 100.0, "payment_value": 100.0,
		"review_score": 5, "order_purchase_timestamp": "2024-01-10T00:00:00Z",
	})

	repo := NewOntologyRepo()
	result, err := repo.QueryByObjectType(ctx, pool, "customer", ObjectFilters{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
}

func TestOntologyRepo_RBAC_GetObjectByID_Denied(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOntologyTestDB(t)
	ctx := WithRole(context.Background(), "viewer")

	repo := NewOntologyRepo()
	_, err := repo.GetObjectByID(ctx, pool, "customer", "cust-1")
	assert.ErrorContains(t, err, "does not have access")
}

func TestOntologyRepo_RBAC_SearchObjects_Denied(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOntologyTestDB(t)
	ctx := WithRole(context.Background(), "viewer")

	repo := NewOntologyRepo()
	_, err := repo.SearchObjects(ctx, pool, "customer", SearchFilters{Query: "test", Limit: 10})
	assert.ErrorContains(t, err, "does not have access")
}
