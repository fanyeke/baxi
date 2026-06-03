//go:build integration

package governance

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/model"
	"baxi/internal/repository/common"
	governanceRepo "baxi/internal/repository/governance"
	"baxi/internal/testutil"
)

const govServiceDDL = `
CREATE SCHEMA IF NOT EXISTS gov;

CREATE TABLE IF NOT EXISTS gov.config_snapshot (
    snapshot_id     BIGSERIAL PRIMARY KEY,
    config_key      TEXT NOT NULL,
    config_type     TEXT,
    source_path     TEXT,
    content_jsonb   JSONB,
    content_hash    TEXT,
    loaded_at       TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS gov.data_classification (
    field_path           TEXT PRIMARY KEY,
    classification_level TEXT NOT NULL,
    sensitivity_score    DOUBLE PRECISION DEFAULT 0,
    description          TEXT DEFAULT ''
);

CREATE TABLE IF NOT EXISTS gov.data_lineage (
    lineage_id           BIGSERIAL PRIMARY KEY,
    source_table         TEXT NOT NULL,
    source_column        TEXT,
    target_table         TEXT NOT NULL,
    target_column        TEXT,
    transformation_logic TEXT DEFAULT '',
    confidence           DOUBLE PRECISION DEFAULT 1.0
);

CREATE TABLE IF NOT EXISTS gov.access_policy (
    policy_id         BIGSERIAL PRIMARY KEY,
    policy_name       TEXT NOT NULL,
    resource_type     TEXT,
    resource_pattern  TEXT,
    action            TEXT,
    principal_type    TEXT,
    principal_pattern TEXT,
    effect            TEXT,
    conditions_jsonb  JSONB
);
`

func setupGovServiceDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	pool, err := pgxpool.New(ctx, pg.ConnectionString())
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	_, err = pool.Exec(ctx, govServiceDDL)
	require.NoError(t, err)

	return pool
}

func insertClassification(t *testing.T, pool *pgxpool.Pool, fieldPath, level string, score float64) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO gov.data_classification (field_path, classification_level, sensitivity_score, description)
		VALUES ($1, $2, $3, $4)
	`, fieldPath, level, score, "test classification")
	require.NoError(t, err)
}

func insertLineage(t *testing.T, pool *pgxpool.Pool, sourceTable, sourceCol, targetTable, targetCol string, confidence float64) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO gov.data_lineage (source_table, source_column, target_table, target_column, confidence)
		VALUES ($1, $2, $3, $4, $5)
	`, sourceTable, sourceCol, targetTable, targetCol, confidence)
	require.NoError(t, err)
}

func insertAccessPolicy(t *testing.T, pool *pgxpool.Pool, name, resourcePattern, action, principalPattern, effect string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO gov.access_policy (policy_name, resource_pattern, action, principal_pattern, effect)
		VALUES ($1, $2, $3, $4, $5)
	`, name, resourcePattern, action, principalPattern, effect)
	require.NoError(t, err)
}

func insertConfigSnapshot(t *testing.T, pool *pgxpool.Pool, configKey string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO gov.config_snapshot (config_key, config_type, source_path, content_jsonb)
		VALUES ($1, 'yaml', $2, '{}'::jsonb)
	`, configKey, configKey)
	require.NoError(t, err)
}

func TestClassificationService_GetClassification_Known(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovServiceDB(t)
	insertClassification(t, pool, "customer.email", "pii", 0.9)

	provider := common.NewPoolProvider(pool)
	govRepo := governanceRepo.NewRepository(provider)
	svc := NewClassificationService(govRepo)

	level, err := svc.GetClassification(context.Background(), "customer.email")
	require.NoError(t, err)
	assert.Equal(t, "L3", level)
}

func TestClassificationService_GetClassification_Unknown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovServiceDB(t)
	provider := common.NewPoolProvider(pool)
	govRepo := governanceRepo.NewRepository(provider)
	svc := NewClassificationService(govRepo)

	level, err := svc.GetClassification(context.Background(), "unknown.field")
	require.NoError(t, err)
	assert.Equal(t, "L2", level)
}

func TestClassificationService_GetClassification_Sensitive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovServiceDB(t)
	insertClassification(t, pool, "order.revenue", "sensitive", 0.8)

	provider := common.NewPoolProvider(pool)
	govRepo := governanceRepo.NewRepository(provider)
	svc := NewClassificationService(govRepo)

	level, err := svc.GetClassification(context.Background(), "order.revenue")
	require.NoError(t, err)
	assert.Equal(t, "L3", level)
}

func TestClassificationService_GetFieldMarking_Found(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovServiceDB(t)
	insertClassification(t, pool, "customer.email", "pii", 0.95)

	provider := common.NewPoolProvider(pool)
	govRepo := governanceRepo.NewRepository(provider)
	svc := NewClassificationService(govRepo)

	level, isPII, llmAllowed, err := svc.GetFieldMarking(context.Background(), "customer", "email")
	require.NoError(t, err)
	assert.Equal(t, "L3", level)
	assert.True(t, isPII)
	assert.False(t, llmAllowed)
}

func TestClassificationService_GetFieldMarking_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovServiceDB(t)
	provider := common.NewPoolProvider(pool)
	govRepo := governanceRepo.NewRepository(provider)
	svc := NewClassificationService(govRepo)

	level, isPII, llmAllowed, err := svc.GetFieldMarking(context.Background(), "product", "price")
	require.NoError(t, err)
	assert.Equal(t, "L2", level)
	assert.False(t, isPII)
	assert.True(t, llmAllowed)
}

func TestClassificationService_GetFieldMarking_PublicInternal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovServiceDB(t)
	insertClassification(t, pool, "product.name", "public_internal", 0.1)

	provider := common.NewPoolProvider(pool)
	govRepo := governanceRepo.NewRepository(provider)
	svc := NewClassificationService(govRepo)

	level, isPII, llmAllowed, err := svc.GetFieldMarking(context.Background(), "product", "name")
	require.NoError(t, err)
	assert.Equal(t, "L1", level)
	assert.False(t, isPII)
	assert.True(t, llmAllowed)
}

func TestClassificationService_GetAll(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovServiceDB(t)
	insertClassification(t, pool, "a.field", "internal", 0.5)
	insertClassification(t, pool, "b.field", "pii", 0.9)

	provider := common.NewPoolProvider(pool)
	govRepo := governanceRepo.NewRepository(provider)
	svc := NewClassificationService(govRepo)

	rows, err := svc.GetAll(context.Background())
	require.NoError(t, err)
	assert.Len(t, rows, 2)
}

func TestLineageService_GetLineage_BothDirections(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovServiceDB(t)
	ctx := context.Background()

	insertLineage(t, pool, "raw.orders", "id", "dwd.order_level", "order_id", 1.0)
	insertLineage(t, pool, "dwd.order_level", "order_id", "mart.metric_daily", "order_id", 0.9)
	insertLineage(t, pool, "dwd.order_level", "customer_id", "mart.metric_dimension_daily", "customer_id", 0.8)

	provider := common.NewPoolProvider(pool)
	govRepo := governanceRepo.NewRepository(provider)
	svc := NewLineageService(govRepo)

	result, err := svc.GetLineage(ctx, "dwd.order_level")
	require.NoError(t, err)
	assert.Equal(t, "dwd.order_level", result.Resource)
	assert.Contains(t, result.Upstream, "raw.orders")
	assert.Contains(t, result.Downstream, "mart.metric_daily")
	assert.Contains(t, result.Downstream, "mart.metric_dimension_daily")
}

func TestLineageService_GetUpstream(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovServiceDB(t)
	ctx := context.Background()

	insertLineage(t, pool, "raw.orders", "id", "dwd.order_level", "order_id", 1.0)
	insertLineage(t, pool, "raw.customers", "id", "dwd.order_level", "customer_id", 1.0)

	provider := common.NewPoolProvider(pool)
	govRepo := governanceRepo.NewRepository(provider)
	svc := NewLineageService(govRepo)

	upstream, err := svc.GetUpstream(ctx, "dwd.order_level")
	require.NoError(t, err)
	assert.Len(t, upstream, 2)
	assert.Contains(t, upstream, "raw.orders")
	assert.Contains(t, upstream, "raw.customers")
}

func TestLineageService_GetDownstream(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovServiceDB(t)
	ctx := context.Background()

	insertLineage(t, pool, "dwd.order_level", "order_id", "mart.metric_daily", "order_id", 1.0)

	provider := common.NewPoolProvider(pool)
	govRepo := governanceRepo.NewRepository(provider)
	svc := NewLineageService(govRepo)

	downstream, err := svc.GetDownstream(ctx, "dwd.order_level")
	require.NoError(t, err)
	assert.Len(t, downstream, 1)
	assert.Contains(t, downstream, "mart.metric_daily")
}

func TestLineageService_GetLineage_NoResults(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovServiceDB(t)
	ctx := context.Background()

	provider := common.NewPoolProvider(pool)
	govRepo := governanceRepo.NewRepository(provider)
	svc := NewLineageService(govRepo)

	result, err := svc.GetLineage(ctx, "nonexistent.table")
	require.NoError(t, err)
	assert.Empty(t, result.Upstream)
	assert.Empty(t, result.Downstream)
}

func TestLineageService_GetAll(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovServiceDB(t)
	ctx := context.Background()

	insertLineage(t, pool, "a", "c1", "b", "c1", 1.0)
	insertLineage(t, pool, "b", "c1", "c", "c1", 0.8)

	provider := common.NewPoolProvider(pool)
	govRepo := governanceRepo.NewRepository(provider)
	svc := NewLineageService(govRepo)

	rows, err := svc.GetAll(ctx)
	require.NoError(t, err)
	assert.Len(t, rows, 2)
}

func TestAccessPolicyService_CheckAccess_Allow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovServiceDB(t)
	ctx := context.Background()

	insertAccessPolicy(t, pool, "allow-order-read", "order", "read", "analyst", "allow")

	provider := common.NewPoolProvider(pool)
	govRepo := governanceRepo.NewRepository(provider)
	svc := NewAccessPolicyService(govRepo)

	decision := svc.CheckAccess(ctx, "analyst", "order", "read")
	assert.Equal(t, model.AccessAllowed, decision)
}

func TestAccessPolicyService_CheckAccess_MatchingPolicy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovServiceDB(t)
	ctx := context.Background()

	insertAccessPolicy(t, pool, "deny-order-delete", "order", "delete", "analyst", "allow")

	provider := common.NewPoolProvider(pool)
	govRepo := governanceRepo.NewRepository(provider)
	svc := NewAccessPolicyService(govRepo)

	decision := svc.CheckAccess(ctx, "analyst", "order", "delete")
	assert.Equal(t, model.AccessAllowed, decision)
}

func TestAccessPolicyService_CheckAccess_DefaultDeny(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovServiceDB(t)
	ctx := context.Background()

	provider := common.NewPoolProvider(pool)
	govRepo := governanceRepo.NewRepository(provider)
	svc := NewAccessPolicyService(govRepo)

	decision := svc.CheckAccess(ctx, "viewer", "order", "read")
	assert.Equal(t, model.AccessDenied, decision)
}

func TestAccessPolicyService_CheckAccess_WrongRole(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovServiceDB(t)
	ctx := context.Background()

	insertAccessPolicy(t, pool, "admin-only", "order", "read", "admin", "allow")

	provider := common.NewPoolProvider(pool)
	govRepo := governanceRepo.NewRepository(provider)
	svc := NewAccessPolicyService(govRepo)

	decision := svc.CheckAccess(ctx, "analyst", "order", "read")
	assert.Equal(t, model.AccessDenied, decision)
}

func TestAccessPolicyService_CheckAccess_WildcardResource(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovServiceDB(t)
	ctx := context.Background()

	insertAccessPolicy(t, pool, "allow-all", "*", "read", "analyst", "allow")

	provider := common.NewPoolProvider(pool)
	govRepo := governanceRepo.NewRepository(provider)
	svc := NewAccessPolicyService(govRepo)

	decision := svc.CheckAccess(ctx, "analyst", "product", "read")
	assert.Equal(t, model.AccessAllowed, decision)
}

func TestAccessPolicyService_CheckAccess_Conditional(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovServiceDB(t)
	ctx := context.Background()

	_, err := pool.Exec(ctx, `
		INSERT INTO gov.access_policy
			(policy_name, resource_pattern, action, principal_pattern,
			effect, conditions_jsonb)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, "conditional-policy", "order", "read", "analyst", "allow", `{"time": {"before": "2025-01-01"}}`)
	require.NoError(t, err)

	provider := common.NewPoolProvider(pool)
	govRepo := governanceRepo.NewRepository(provider)
	svc := NewAccessPolicyService(govRepo)

	decision := svc.CheckAccess(ctx, "analyst", "order", "read")
	assert.Equal(t, model.AccessConditional, decision)
}

func TestAccessPolicyService_GetAll(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovServiceDB(t)
	ctx := context.Background()

	insertAccessPolicy(t, pool, "policy-1", "*", "read", "admin", "allow")

	provider := common.NewPoolProvider(pool)
	govRepo := governanceRepo.NewRepository(provider)
	svc := NewAccessPolicyService(govRepo)

	policies, err := svc.GetAll(ctx)
	require.NoError(t, err)
	assert.Len(t, policies, 1)
}

func TestCheckpointService_RequiresCheckpoint_SensitiveAction(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovServiceDB(t)
	ctx := context.Background()

	provider := common.NewPoolProvider(pool)
	govRepo := governanceRepo.NewRepository(provider)
	svc := NewCheckpointService(govRepo)

	assert.True(t, svc.RequiresCheckpoint(ctx, "execute_dispatch"))
	assert.True(t, svc.RequiresCheckpoint(ctx, "modify_business_policy"))
	assert.True(t, svc.RequiresCheckpoint(ctx, "trigger_pipeline"))
}

func TestCheckpointService_RequiresCheckpoint_NonSensitive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovServiceDB(t)
	ctx := context.Background()

	provider := common.NewPoolProvider(pool)
	govRepo := governanceRepo.NewRepository(provider)
	svc := NewCheckpointService(govRepo)

	assert.False(t, svc.RequiresCheckpoint(ctx, "view_dashboard"))
	assert.False(t, svc.RequiresCheckpoint(ctx, "export_report"))
}

func TestCheckpointService_GetRules_HasBuiltins(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovServiceDB(t)
	ctx := context.Background()

	provider := common.NewPoolProvider(pool)
	govRepo := governanceRepo.NewRepository(provider)
	svc := NewCheckpointService(govRepo)

	rules := svc.GetRules(ctx)
	assert.Len(t, rules, 3)
	actions := make(map[string]bool)
	for _, r := range rules {
		actions[r.Action] = true
	}
	assert.True(t, actions["execute_dispatch"])
	assert.True(t, actions["modify_business_policy"])
	assert.True(t, actions["trigger_pipeline"])
}

func TestCheckpointService_GetRules_WithConfigSnapshot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGovServiceDB(t)
	ctx := context.Background()

	insertConfigSnapshot(t, pool, "checkpoint_rules.yml")

	provider := common.NewPoolProvider(pool)
	govRepo := governanceRepo.NewRepository(provider)
	svc := NewCheckpointService(govRepo)

	rules := svc.GetRules(ctx)
	assert.NotEmpty(t, rules)
}

func TestResolveLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"pii", "L3"},
		{"sensitive", "L3"},
		{"internal", "L2"},
		{"derived_sensitive", "L2"},
		{"public_internal", "L1"},
		{"unknown_level", "L2"},
		{"", "L2"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := ResolveLevel(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestRedactObjectContext_AdminSeesAll(t *testing.T) {
	props := map[string]interface{}{
		"email":   "test@example.com",
		"name":    "John",
		"revenue": 1000.0,
	}
	classifications := map[string]string{
		"email":   "pii",
		"revenue": "sensitive",
	}
	markings := map[string]string{}

	result := RedactObjectContext(props, classifications, markings, RedactionPolicy{Role: "admin"})
	assert.Contains(t, result.Properties, "email")
	assert.Contains(t, result.Properties, "name")
	assert.Contains(t, result.Properties, "revenue")
	assert.Empty(t, result.RedactedFields)
}

func TestRedactObjectContext_ViewerRedactsPII(t *testing.T) {
	props := map[string]interface{}{
		"email":   "test@example.com",
		"name":    "John",
		"revenue": 1000.0,
	}
	classifications := map[string]string{
		"email":   "pii",
		"revenue": "sensitive",
	}
	markings := map[string]string{}

	result := RedactObjectContext(props, classifications, markings, RedactionPolicy{Role: "viewer"})
	assert.NotContains(t, result.Properties, "email")
	assert.NotContains(t, result.Properties, "revenue")
	assert.Contains(t, result.Properties, "name")
	assert.Len(t, result.RedactedFields, 2)
}

func TestRedactObjectContext_AnalystSeesInternal(t *testing.T) {
	props := map[string]interface{}{
		"email":    "test@example.com",
		"category": "electronics",
	}
	classifications := map[string]string{
		"email":    "pii",
		"category": "internal",
	}
	markings := map[string]string{}

	result := RedactObjectContext(props, classifications, markings, RedactionPolicy{Role: "analyst"})
	assert.NotContains(t, result.Properties, "email")
	assert.Contains(t, result.Properties, "category")
	assert.Len(t, result.RedactedFields, 1)
	assert.Equal(t, "email", result.RedactedFields[0].Field)
}

func TestRedactObjectContext_MarkingTakesPriority(t *testing.T) {
	props := map[string]interface{}{
		"email": "test@example.com",
	}
	classifications := map[string]string{
		"email": "public_internal",
	}
	markings := map[string]string{
		"email": "PII",
	}

	result := RedactObjectContext(props, classifications, markings, RedactionPolicy{Role: "analyst"})
	assert.NotContains(t, result.Properties, "email")
	assert.Len(t, result.RedactedFields, 1)
	assert.Equal(t, "marking: PII", result.RedactedFields[0].Reason)
}

func TestRedactObjectContext_NoClassOrMarking(t *testing.T) {
	props := map[string]interface{}{
		"name":    "John",
		"address": "123 Main St",
	}
	classifications := map[string]string{}
	markings := map[string]string{}

	result := RedactObjectContext(props, classifications, markings, RedactionPolicy{Role: "viewer"})
	assert.Contains(t, result.Properties, "name")
	assert.Contains(t, result.Properties, "address")
	assert.Empty(t, result.RedactedFields)
}

func TestRedactObjectContext_SortedRedactedFields(t *testing.T) {
	props := map[string]interface{}{
		"zeta":   "last",
		"alpha":  "first",
		"gamma":  "middle",
		"normal": "visible",
	}
	classifications := map[string]string{
		"zeta":  "pii",
		"alpha": "pii",
		"gamma": "pii",
	}
	markings := map[string]string{}

	result := RedactObjectContext(props, classifications, markings, RedactionPolicy{Role: "analyst"})
	assert.Len(t, result.RedactedFields, 3)
	assert.Equal(t, "alpha", result.RedactedFields[0].Field)
	assert.Equal(t, "gamma", result.RedactedFields[1].Field)
	assert.Equal(t, "zeta", result.RedactedFields[2].Field)
}

func TestRedactObjectContext_MarkingsAllRoles(t *testing.T) {
	props := map[string]interface{}{
		"pii_field":   "secret",
		"financial":   "secret",
		"raw_data":    "secret",
		"operational": "internal",
	}
	classifications := map[string]string{}
	markings := map[string]string{
		"pii_field":   "PII",
		"financial":   "FINANCIAL_INTERNAL",
		"raw_data":    "RAW_DATA",
		"operational": "OPERATIONAL_INTERNAL",
	}

	adminResult := RedactObjectContext(props, classifications, markings, RedactionPolicy{Role: "admin"})
	assert.Len(t, adminResult.Properties, 4)
	assert.Empty(t, adminResult.RedactedFields)

	analystResult := RedactObjectContext(props, classifications, markings, RedactionPolicy{Role: "analyst"})
	assert.Len(t, analystResult.Properties, 1)
	assert.Equal(t, "operational", analystResult.Properties["operational"])

	viewerResult := RedactObjectContext(props, classifications, markings, RedactionPolicy{Role: "viewer"})
	assert.Empty(t, viewerResult.Properties)
	assert.Len(t, viewerResult.RedactedFields, 4)
}
