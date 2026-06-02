//go:build integration

package decision_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"baxi/internal/action"
	"baxi/internal/decision"
	"baxi/internal/llm"
	"baxi/internal/ontology"
	"baxi/internal/repository"
	"baxi/internal/repository/common"
	ontRepo "baxi/internal/repository/ontology"
	"baxi/internal/testutil"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// ──── Path helpers ──────────────────────────────────────────────────────────

func projectRoot() string {
	dir, _ := os.Getwd()
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	return "."
}

func v2SchemaPath() string {
	return filepath.Join(projectRoot(), "config", "aip_object_schema_v2.yml")
}

func v1SchemaPath() string {
	return filepath.Join(projectRoot(), "config", "aip_object_schema.yml")
}

func contextRecipesPath() string {
	return filepath.Join(projectRoot(), "config", "context_recipes.yml")
}

func metricDefsPath() string {
	return filepath.Join(projectRoot(), "config", "metric_definitions.yml")
}

func actionRegistryPath() string {
	return filepath.Join(projectRoot(), "config", "action_registry.yml")
}

func migrationsDir() string {
	return filepath.Join(projectRoot(), "migrations")
}

// ──── Shared test environment setup ─────────────────────────────────────────

// recipeTestEnv holds the shared resources needed by golden recipe tests.
type recipeTestEnv struct {
	pool    *pgxpool.Pool
	builder *decision.RecipeContextBuilder
	close   func()
}

// setupRecipeTestEnv starts a PostgreSQL container, runs migrations, adds
// v2 compatibility columns, loads configs, wires a RecipeContextBuilder, and
// returns the test environment with a cleanup function.
func setupRecipeTestEnv(t *testing.T) *recipeTestEnv {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	pg, err := testutil.StartPostgres(ctx)
	require.NoError(t, err, "start postgres")

	pool, err := pgxpool.New(ctx, pg.ConnectionString())
	require.NoError(t, err, "create pool")

	require.NoError(t, pg.RunMigrations(ctx, migrationsDir()), "run migrations")

	addV2CompatibilityColumns(t, ctx, pool)

	// Load action registry.
	actionReg, err := action.NewActionRegistry(actionRegistryPath())
	require.NoError(t, err, "load action registry")

	// Load v1 + v2 object schemas.
	objRegistry, err := ontology.NewObjectRegistry(ctx, nil, nil, v1SchemaPath())
	require.NoError(t, err, "load v1 object registry")

	require.NoError(t, objRegistry.LoadV2Schema(v2SchemaPath()), "load v2 schema")
	v2Objects := objRegistry.AllObjectsV2()
	require.NotEmpty(t, v2Objects, "v2 objects must not be empty")

	// Load recipes.
	recipes, err := ontology.LoadContextRecipes(contextRecipesPath())
	require.NoError(t, err, "load context recipes")
	require.NotEmpty(t, recipes, "recipes must not be empty")

	// Load metric definitions.
	metricDefs, err := ontology.LoadMetricDefinitions(metricDefsPath())
	require.NoError(t, err, "load metric definitions")

	// Wire services.
	metricResolver := ontology.NewMetricResolver(metricDefs)
	metricQuery := ontology.NewMetricQueryResolver(metricResolver, pool)
	linkExec := ontRepo.NewLinkExecutor(common.NewPoolProvider(pool))
	qc := ontology.NewQueryCompiler(v2Objects, 10000)

	caseProvider := repository.NewDecisionRepository()
	builder := decision.NewRecipeContextBuilder(
		caseProvider, qc, metricQuery, linkExec, pool,
		action.NewActionTypeProviderAdapter(actionReg), recipes,
	)

	closeFn := func() {
		pool.Close()
		if err := pg.Terminate(context.Background()); err != nil {
			t.Logf("terminate postgres: %v", err)
		}
	}

	return &recipeTestEnv{
		pool:    pool,
		builder: builder,
		close:   closeFn,
	}
}

// addV2CompatibilityColumns adds columns to dwd.item_level that the v2
// ontology schema references but the current migrations do not create.
func addV2CompatibilityColumns(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	cols := []string{
		"seller_city TEXT",
		"late_delivery_rate NUMERIC(10,6)",
		"order_status TEXT",
		"payment_value NUMERIC(18,2)",
		"review_score NUMERIC(4,2)",
		"delivery_status TEXT",
		"order_purchase_timestamp TIMESTAMPTZ",
	}
	for _, colDef := range cols {
		_, err := pool.Exec(ctx,
			fmt.Sprintf("ALTER TABLE dwd.item_level ADD COLUMN IF NOT EXISTS %s", colDef))
		require.NoError(t, err, "add column: %s", colDef)
	}
}

// ──── Assertion helpers ─────────────────────────────────────────────────────

// verifyEnvelopeBasics checks the common assertions that every golden test
// must satisfy: context_hash, object_context, evidence, allowed_actions.
func verifyEnvelopeBasics(t *testing.T, envelope *llm.LLMSafeContextEnvelope) {
	t.Helper()
	require.NotEmpty(t, envelope.ContextHash, "context_hash must not be empty")
	require.NotNil(t, envelope.ObjectContext, "object_context must not be nil")
	require.NotEmpty(t, envelope.Evidence, "evidence must not be empty")
	require.NotEmpty(t, envelope.AllowedActions, "allowed_actions must not be empty")
}

// verifyEvidenceStructure checks that every evidence item has non-empty Type,
// Key, and a non-nil Value.
func verifyEvidenceStructure(t *testing.T, envelope *llm.LLMSafeContextEnvelope) {
	t.Helper()
	for i, item := range envelope.Evidence {
		require.NotEmpty(t, item.Type, "evidence[%d].Type must not be empty", i)
		require.NotEmpty(t, item.Key, "evidence[%d].Key must not be empty", i)
		require.NotNil(t, item.Value, "evidence[%d].Value must not be nil", i)
	}
}

// verifyPlannedFieldsAbsent checks that properties marked as planned in the
// v2 schema are absent from the object context.
func verifyPlannedFieldsAbsent(t *testing.T, envelope *llm.LLMSafeContextEnvelope, plannedFields []string) {
	t.Helper()
	for _, field := range plannedFields {
		_, exists := envelope.ObjectContext.Properties[field]
		require.False(t, exists, "planned field %q must be absent from object_context properties", field)
	}
}

// ──── Golden case: SellerLateDelivery ───────────────────────────────────────

func TestSellerLateDelivery(t *testing.T) {
	env := setupRecipeTestEnv(t)
	defer env.close()
	ctx := context.Background()

	// Seed fixture data: seller in dwd.item_level with linked orders.
	_, err := env.pool.Exec(ctx, `
		INSERT INTO dwd.item_level (
			order_id, order_item_id, product_id, seller_id,
			product_category_name, seller_state, seller_city, price, freight_value
		) VALUES
			('ORD_SL_001', 1, 'PROD_SL_001', 'SELLER_001',
			 'bed_bath_table', 'SP', 'Sao Paulo', 199.90, 15.50),
			('ORD_SL_002', 1, 'PROD_SL_002', 'SELLER_001',
			 'health_beauty', 'SP', 'Sao Paulo', 49.90, 5.00)
	`)
	require.NoError(t, err, "seed seller item rows")

	_, err = env.pool.Exec(ctx, `
		INSERT INTO dwd.order_level (
			order_id, customer_id, customer_unique_id, order_status,
			order_purchase_timestamp, payment_value, review_score,
			is_late, is_cancelled
		) VALUES
			('ORD_SL_001', 'CUST_SL_001', 'UNIQ_SL_001', 'delivered',
			 '2026-05-01 10:00:00+00', 199.90, 4.5, true, false),
			('ORD_SL_002', 'CUST_SL_002', 'UNIQ_SL_002', 'shipped',
			 '2026-05-15 14:30:00+00', 49.90, 3.0, false, false)
	`)
	require.NoError(t, err, "seed order rows")

	_, err = env.pool.Exec(ctx, `
		INSERT INTO ai.decision_case (
			case_id, status, source_type, source_id, object_type, object_id,
			severity, alert_id, created_at
		) VALUES (
			'CASE_SL_001', 'open', 'metric_alert', 'ALERT_SL_001',
			'seller', 'SELLER_001', 'high', 'seller_late_delivery_spike', NOW()
		)
	`)
	require.NoError(t, err, "seed decision case")

	envelope, err := env.builder.BuildEnvelope(ctx, "CASE_SL_001", "seller_late_delivery_alert")
	require.NoError(t, err, "BuildEnvelope should succeed")

	verifyEnvelopeBasics(t, envelope)
	verifyEvidenceStructure(t, envelope)

	// Verify recipe-specific evidence: alert_id and rule_id items.
	hasAlertID := false
	hasRuleID := false
	for _, item := range envelope.Evidence {
		if item.Key == "alert_id" {
			hasAlertID = true
		}
		if item.Key == "rule_id" {
			hasRuleID = true
		}
	}
	require.True(t, hasAlertID, "evidence must contain alert_id")
	require.True(t, hasRuleID, "evidence must contain rule_id")

	// Verify object_context matches the seeded data.
	require.Equal(t, "seller", envelope.ObjectContext.ObjectType)
	require.Equal(t, "SELLER_001", envelope.ObjectContext.ObjectID)
	require.NotNil(t, envelope.ObjectContext.Properties)

	// Verify allowed_actions include expected actions.
	require.Contains(t, envelope.AllowedActions, "notify_owner")
	require.Contains(t, envelope.AllowedActions, "create_followup_task")
	require.Contains(t, envelope.AllowedActions, "export_report")

	// Verify governance and redaction summary are populated.
	require.NotZero(t, envelope.Governance)
	require.NotZero(t, envelope.RedactionSummary)

	t.Logf("SellerLateDelivery: hash=%s evidence=%d actions=%d",
		envelope.ContextHash, len(envelope.Evidence), len(envelope.AllowedActions))
}

// ──── Golden case: OrderAnomaly ─────────────────────────────────────────────

func TestOrderAnomaly(t *testing.T) {
	env := setupRecipeTestEnv(t)
	defer env.close()
	ctx := context.Background()

	// The order_anomaly_alert recipe references region properties and links.
	// Seed a region via dwd.order_level (region source table).
	_, err := env.pool.Exec(ctx, `
		INSERT INTO dwd.item_level (
			order_id, order_item_id, product_id, seller_id,
			product_category_name, seller_state, seller_city, price, freight_value
		) VALUES
			('ORD_OA_001', 1, 'PROD_OA_001', 'SELLER_OA_001',
			 'bed_bath_table', 'SP', 'Sao Paulo', 199.90, 15.50),
			('ORD_OA_002', 1, 'PROD_OA_002', 'SELLER_OA_002',
			 'health_beauty', 'SP', 'Sao Paulo', 49.90, 5.00)
	`)
	require.NoError(t, err, "seed item rows for region test")

	_, err = env.pool.Exec(ctx, `
		INSERT INTO dwd.order_level (
			order_id, customer_id, customer_unique_id, customer_state, order_status,
			order_purchase_timestamp, payment_value, review_score,
			is_late, is_cancelled
		) VALUES
			('ORD_OA_001', 'CUST_OA_001', 'UNIQ_OA_001', 'SP', 'delivered',
			 '2026-05-01 10:00:00+00', 199.90, 4.5, true, false),
			('ORD_OA_002', 'CUST_OA_002', 'UNIQ_OA_002', 'SP', 'shipped',
			 '2026-05-15 14:30:00+00', 49.90, 3.0, false, false)
	`)
	require.NoError(t, err, "seed order rows for region test")

	_, err = env.pool.Exec(ctx, `
		INSERT INTO ai.decision_case (
			case_id, status, source_type, source_id, object_type, object_id,
			severity, alert_id, created_at
		) VALUES (
			'CASE_OA_001', 'open', 'metric_alert', 'ALERT_OA_001',
			'region', 'SP', 'high', 'order_anomaly', NOW()
		)
	`)
	require.NoError(t, err, "seed decision case for order anomaly")

	envelope, err := env.builder.BuildEnvelope(ctx, "CASE_OA_001", "order_anomaly_alert")
	require.NoError(t, err, "BuildEnvelope should succeed")

	verifyEnvelopeBasics(t, envelope)
	verifyEvidenceStructure(t, envelope)

	// Verify object_context.
	require.Equal(t, "region", envelope.ObjectContext.ObjectType)
	require.Equal(t, "SP", envelope.ObjectContext.ObjectID)
	require.NotNil(t, envelope.ObjectContext.Properties)

	// Verify allowed_actions include expected actions.
	require.Contains(t, envelope.AllowedActions, "create_followup_task")
	require.Contains(t, envelope.AllowedActions, "notify_owner")
	require.Contains(t, envelope.AllowedActions, "export_report")

	require.NotZero(t, envelope.Governance)
	require.NotZero(t, envelope.RedactionSummary)

	t.Logf("OrderAnomaly: hash=%s evidence=%d actions=%d",
		envelope.ContextHash, len(envelope.Evidence), len(envelope.AllowedActions))
}

// ──── Golden case: CustomerChurnRisk ────────────────────────────────────────

func TestCustomerChurnRisk(t *testing.T) {
	env := setupRecipeTestEnv(t)
	defer env.close()
	ctx := context.Background()

	// Seed a customer via dwd.order_level (customer source table).
	// customer_city is a planned field and must be absent from context.
	_, err := env.pool.Exec(ctx, `
		INSERT INTO dwd.item_level (
			order_id, order_item_id, product_id, seller_id,
			product_category_name, seller_state, seller_city, price, freight_value
		) VALUES
			('ORD_CC_001', 1, 'PROD_CC_001', 'SELLER_CC_001',
			 'bed_bath_table', 'SP', 'Sao Paulo', 199.90, 15.50),
			('ORD_CC_002', 1, 'PROD_CC_002', 'SELLER_CC_002',
			 'health_beauty', 'SP', 'Sao Paulo', 149.90, 10.00)
	`)
	require.NoError(t, err, "seed item rows for customer")

	_, err = env.pool.Exec(ctx, `
		INSERT INTO dwd.order_level (
			order_id, customer_id, customer_unique_id, customer_state, order_status,
			order_purchase_timestamp, payment_value, review_score,
			is_late, is_cancelled
		) VALUES
			('ORD_CC_001', 'CUST_001', 'UNIQ_001', 'SP', 'delivered',
			 '2026-04-01 10:00:00+00', 199.90, 4.5, false, false),
			('ORD_CC_002', 'CUST_001', 'UNIQ_001', 'SP', 'delivered',
			 '2026-05-01 10:00:00+00', 149.90, 3.5, false, false)
	`)
	require.NoError(t, err, "seed order rows for customer")

	_, err = env.pool.Exec(ctx, `
		INSERT INTO ai.decision_case (
			case_id, status, source_type, source_id, object_type, object_id,
			severity, alert_id, created_at
		) VALUES (
			'CASE_CC_001', 'open', 'metric_alert', 'ALERT_CC_001',
			'customer', 'CUST_001', 'high', 'customer_churn_risk', NOW()
		)
	`)
	require.NoError(t, err, "seed decision case for customer churn")

	envelope, err := env.builder.BuildEnvelope(ctx, "CASE_CC_001", "customer_churn_risk_alert")
	require.NoError(t, err, "BuildEnvelope should succeed")

	verifyEnvelopeBasics(t, envelope)
	verifyEvidenceStructure(t, envelope)

	// Verify object_context.
	require.Equal(t, "customer", envelope.ObjectContext.ObjectType)
	require.Equal(t, "CUST_001", envelope.ObjectContext.ObjectID)
	require.NotNil(t, envelope.ObjectContext.Properties)

	// customer_city has availability: planned in the v2 schema, must be absent.
	verifyPlannedFieldsAbsent(t, envelope, []string{"customer_city"})

	// Verify expected actions.
	require.Contains(t, envelope.AllowedActions, "create_followup_task")
	require.Contains(t, envelope.AllowedActions, "notify_owner")

	require.NotZero(t, envelope.Governance)
	require.NotZero(t, envelope.RedactionSummary)

	t.Logf("CustomerChurnRisk: hash=%s evidence=%d actions=%d",
		envelope.ContextHash, len(envelope.Evidence), len(envelope.AllowedActions))
}

// ──── Golden case: ProductPerformanceDrop ───────────────────────────────────

func TestProductPerformanceDrop(t *testing.T) {
	env := setupRecipeTestEnv(t)
	defer env.close()
	ctx := context.Background()

	// Seed a product in dwd.item_level with review links via dwd.order_level.
	_, err := env.pool.Exec(ctx, `
		INSERT INTO dwd.item_level (
			order_id, order_item_id, product_id, seller_id,
			product_category_name, seller_state, seller_city, price, freight_value
		) VALUES
			('ORD_PP_001', 1, 'PROD_001', 'SELLER_PP_001',
			 'bed_bath_table', 'SP', 'Sao Paulo', 199.90, 15.50),
			('ORD_PP_002', 1, 'PROD_001', 'SELLER_PP_002',
			 'bed_bath_table', 'RJ', 'Rio de Janeiro', 179.90, 12.00)
	`)
	require.NoError(t, err, "seed item rows for product")

	_, err = env.pool.Exec(ctx, `
		INSERT INTO dwd.order_level (
			order_id, customer_id, customer_unique_id, customer_state, order_status,
			order_purchase_timestamp, payment_value, review_score,
			is_late, is_cancelled
		) VALUES
			('ORD_PP_001', 'CUST_PP_001', 'UNIQ_PP_001', 'SP', 'delivered',
			 '2026-05-01 10:00:00+00', 199.90, 2.0, false, false),
			('ORD_PP_002', 'CUST_PP_002', 'UNIQ_PP_002', 'RJ', 'delivered',
			 '2026-05-10 14:30:00+00', 179.90, 1.5, false, false)
	`)
	require.NoError(t, err, "seed order rows for product")

	_, err = env.pool.Exec(ctx, `
		INSERT INTO ai.decision_case (
			case_id, status, source_type, source_id, object_type, object_id,
			severity, alert_id, created_at
		) VALUES (
			'CASE_PP_001', 'open', 'metric_alert', 'ALERT_PP_001',
			'product', 'PROD_001', 'high', 'product_performance_drop', NOW()
		)
	`)
	require.NoError(t, err, "seed decision case for product performance")

	envelope, err := env.builder.BuildEnvelope(ctx, "CASE_PP_001", "product_performance_drop_alert")
	require.NoError(t, err, "BuildEnvelope should succeed")

	verifyEnvelopeBasics(t, envelope)
	verifyEvidenceStructure(t, envelope)

	// Verify object_context.
	require.Equal(t, "product", envelope.ObjectContext.ObjectType)
	require.Equal(t, "PROD_001", envelope.ObjectContext.ObjectID)
	require.NotNil(t, envelope.ObjectContext.Properties)

	// Verify expected actions.
	require.Contains(t, envelope.AllowedActions, "create_followup_task")
	require.Contains(t, envelope.AllowedActions, "notify_owner")
	require.Contains(t, envelope.AllowedActions, "export_report")

	require.NotZero(t, envelope.Governance)
	require.NotZero(t, envelope.RedactionSummary)

	t.Logf("ProductPerformanceDrop: hash=%s evidence=%d actions=%d",
		envelope.ContextHash, len(envelope.Evidence), len(envelope.AllowedActions))
}

// ──── Golden case: InventoryStockout ────────────────────────────────────────

func TestInventoryStockout(t *testing.T) {
	env := setupRecipeTestEnv(t)
	defer env.close()
	ctx := context.Background()

	// The inventory_stockout_alert recipe targets category objects.
	// Seed category data via dwd.item_level.
	_, err := env.pool.Exec(ctx, `
		INSERT INTO dwd.item_level (
			order_id, order_item_id, product_id, seller_id,
			product_category_name, seller_state, seller_city, price, freight_value
		) VALUES
			('ORD_IS_001', 1, 'PROD_IS_001', 'SELLER_IS_001',
			 'bed_bath_table', 'SP', 'Sao Paulo', 199.90, 15.50),
			('ORD_IS_002', 1, 'PROD_IS_002', 'SELLER_IS_002',
			 'bed_bath_table', 'RJ', 'Rio de Janeiro', 149.90, 10.00)
	`)
	require.NoError(t, err, "seed item rows for category")

	_, err = env.pool.Exec(ctx, `
		INSERT INTO dwd.order_level (
			order_id, customer_id, customer_unique_id, customer_state, order_status,
			order_purchase_timestamp, payment_value, review_score,
			is_late, is_cancelled
		) VALUES
			('ORD_IS_001', 'CUST_IS_001', 'UNIQ_IS_001', 'SP', 'delivered',
			 '2026-05-01 10:00:00+00', 199.90, 4.0, false, false),
			('ORD_IS_002', 'CUST_IS_002', 'UNIQ_IS_002', 'RJ', 'delivered',
			 '2026-05-10 14:30:00+00', 149.90, 3.5, false, false)
	`)
	require.NoError(t, err, "seed order rows for category")

	_, err = env.pool.Exec(ctx, `
		INSERT INTO ai.decision_case (
			case_id, status, source_type, source_id, object_type, object_id,
			severity, alert_id, created_at
		) VALUES (
			'CASE_IS_001', 'open', 'metric_alert', 'ALERT_IS_001',
			'category', 'bed_bath_table', 'high', 'inventory_stockout', NOW()
		)
	`)
	require.NoError(t, err, "seed decision case for inventory stockout")

	envelope, err := env.builder.BuildEnvelope(ctx, "CASE_IS_001", "inventory_stockout_alert")
	require.NoError(t, err, "BuildEnvelope should succeed")

	verifyEnvelopeBasics(t, envelope)
	verifyEvidenceStructure(t, envelope)

	// Verify object_context.
	require.Equal(t, "category", envelope.ObjectContext.ObjectType)
	require.Equal(t, "bed_bath_table", envelope.ObjectContext.ObjectID)
	require.NotNil(t, envelope.ObjectContext.Properties)

	// Verify expected actions.
	require.Contains(t, envelope.AllowedActions, "create_followup_task")
	require.Contains(t, envelope.AllowedActions, "notify_owner")
	require.Contains(t, envelope.AllowedActions, "export_report")

	require.NotZero(t, envelope.Governance)
	require.NotZero(t, envelope.RedactionSummary)

	t.Logf("InventoryStockout: hash=%s evidence=%d actions=%d",
		envelope.ContextHash, len(envelope.Evidence), len(envelope.AllowedActions))
}
