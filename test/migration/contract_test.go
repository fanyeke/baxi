//go:build integration

// Package migration verifies that the database schema produced by goose
// migrations matches what repository structs expect. This is a schema
// contract test — it does not insert or query functional data.
package migration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"baxi/internal/testutil"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// migrationsDir walks up the directory tree to find the migrations/ folder.
func migrationsDir() string {
	dir, _ := os.Getwd()
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, "migrations")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		dir = filepath.Dir(dir)
	}
	return "migrations"
}

// setupTestDB starts a test PostgreSQL container, runs all migrations, and
// returns the pool. The container and pool are cleaned up automatically.
func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	pool, err := pgxpool.New(ctx, pg.ConnectionString())
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	require.NoError(t, pg.RunMigrations(ctx, migrationsDir()))
	return pool
}

// columnInfo holds the result of an information_schema.columns query.
type columnInfo struct {
	ColumnName    string
	IsNullable    string
	DataType      string
	ColumnDefault *string
}

// queryColumns returns column metadata for a given table from
// information_schema.columns.
func queryColumns(ctx context.Context, t *testing.T, pool *pgxpool.Pool, schema, table string) []columnInfo {
	t.Helper()
	rows, err := pool.Query(ctx, `
		SELECT column_name, is_nullable, data_type, column_default
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position
	`, schema, table)
	require.NoError(t, err)
	defer rows.Close()

	var cols []columnInfo
	for rows.Next() {
		var c columnInfo
		require.NoError(t, rows.Scan(&c.ColumnName, &c.IsNullable, &c.DataType, &c.ColumnDefault))
		cols = append(cols, c)
	}
	require.NoError(t, rows.Err())
	return cols
}

// tableExists checks whether a table exists in the given schema.
func tableExists(ctx context.Context, t *testing.T, pool *pgxpool.Pool, schema, table string) bool {
	t.Helper()
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = $1 AND table_name = $2
		)
	`, schema, table).Scan(&exists)
	require.NoError(t, err)
	return exists
}

// indexExists checks whether an index with the given name exists in the schema.
func indexExists(ctx context.Context, t *testing.T, pool *pgxpool.Pool, schema, indexName string) bool {
	t.Helper()
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_indexes
			WHERE schemaname = $1 AND indexname = $2
		)
	`, schema, indexName).Scan(&exists)
	require.NoError(t, err)
	return exists
}

// constraintExists checks whether a constraint with the given name exists on a table.
func constraintExists(ctx context.Context, t *testing.T, pool *pgxpool.Pool, schema, constraintName string) bool {
	t.Helper()
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.table_constraints
			WHERE constraint_schema = $1 AND constraint_name = $2
		)
	`, schema, constraintName).Scan(&exists)
	require.NoError(t, err)
	return exists
}

// columnMap builds a map of column name → columnInfo for quick lookups.
func columnMap(cols []columnInfo) map[string]columnInfo {
	m := make(map[string]columnInfo, len(cols))
	for _, c := range cols {
		m[c.ColumnName] = c
	}
	return m
}

// -------------------------------------------------------------------------
// Schema existence tests
// -------------------------------------------------------------------------

func TestMigrationContract_SchemasExist(t *testing.T) {
	t.Parallel()
	pool := setupTestDB(t)
	ctx := context.Background()

	expectedSchemas := []string{"raw", "dwd", "mart", "ops", "gov", "ai", "audit"}
	for _, s := range expectedSchemas {
		var exists bool
		err := pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = $1)`, s,
		).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "schema %q must exist", s)
	}
}

// -------------------------------------------------------------------------
// ops.outbox_event contract
// -------------------------------------------------------------------------

func TestMigrationContract_OutboxEvent(t *testing.T) {
	t.Parallel()
	pool := setupTestDB(t)
	ctx := context.Background()

	require.True(t, tableExists(ctx, t, pool, "ops", "outbox_event"),
		"ops.outbox_event table must exist")

	cols := queryColumns(ctx, t, pool, "ops", "outbox_event")
	cm := columnMap(cols)

	expected := map[string]struct {
		dataType   string
		notNull    bool
	}{
		"event_id":          {"text", true},
		"event_type":        {"text", true},
		"source_type":       {"text", true},
		"source_id":         {"text", true},
		"payload_json":      {"jsonb", true},
		"target_channel":    {"text", true},
		"status":            {"text", false},
		"dispatch_attempts": {"bigint", false},
		"last_dispatch_at":  {"timestamp with time zone", false},
		"external_ref":      {"text", false},
		"adapter_name":      {"text", false},
		"created_at":        {"timestamp with time zone", true},
		"processed_at":      {"timestamp with time zone", false},
		"error_message":     {"text", false},
		"next_retry_at":     {"timestamp with time zone", false},
	}

	for name, exp := range expected {
		c, ok := cm[name]
		if !ok {
			t.Errorf("column %q missing from ops.outbox_event", name)
			continue
		}
		assert.Equal(t, exp.dataType, c.DataType,
			"ops.outbox_event.%s type mismatch", name)
		if exp.notNull {
			assert.Equal(t, "NO", c.IsNullable,
				"ops.outbox_event.%s should be NOT NULL", name)
		}
	}

	// Verify key indexes on ops.outbox_event
	assert.True(t, indexExists(ctx, t, pool, "ops", "idx_ops_outbox_event_created"),
		"idx_ops_outbox_event_created should exist")
	assert.True(t, indexExists(ctx, t, pool, "ops", "idx_outbox_event_pending"),
		"idx_outbox_event_pending should exist")
}

// -------------------------------------------------------------------------
// ai.decision_case contract
// -------------------------------------------------------------------------

func TestMigrationContract_DecisionCase(t *testing.T) {
	t.Parallel()
	pool := setupTestDB(t)
	ctx := context.Background()

	require.True(t, tableExists(ctx, t, pool, "ai", "decision_case"),
		"ai.decision_case table must exist")

	cols := queryColumns(ctx, t, pool, "ai", "decision_case")
	cm := columnMap(cols)

	expected := map[string]struct {
		dataType   string
		notNull    bool
	}{
		"case_id":                   {"text", true},
		"alert_id":                  {"text", false},
		"case_type":                 {"text", false},
		"status":                    {"text", false},
		"context_json":              {"jsonb", false},
		"created_at":                {"timestamp with time zone", false},
		"resolved_at":               {"timestamp with time zone", false},
		"source_type":               {"text", false},
		"source_id":                 {"text", false},
		"object_type":               {"text", false},
		"object_id":                 {"text", false},
		"severity":                  {"text", false},
		"context_hash":              {"text", false},
		"governance_snapshot_json":  {"jsonb", false},
		"created_by":                {"text", false},
		"error_message":             {"text", false},
		"updated_at":                {"timestamp with time zone", false},
	}

	for name, exp := range expected {
		c, ok := cm[name]
		if !ok {
			t.Errorf("column %q missing from ai.decision_case", name)
			continue
		}
		assert.Equal(t, exp.dataType, c.DataType,
			"ai.decision_case.%s type mismatch", name)
		if exp.notNull {
			assert.Equal(t, "NO", c.IsNullable,
				"ai.decision_case.%s should be NOT NULL", name)
		}
	}

	// Verify CHECK constraint
	assert.True(t, constraintExists(ctx, t, pool, "ai", "chk_decision_case_status"),
		"chk_decision_case_status should exist")

	// Verify indexes
	assert.True(t, indexExists(ctx, t, pool, "ai", "idx_ai_decision_case_status"),
		"idx_ai_decision_case_status should exist")
	assert.True(t, indexExists(ctx, t, pool, "ai", "idx_ai_decision_case_alert"),
		"idx_ai_decision_case_alert should exist")
	assert.True(t, indexExists(ctx, t, pool, "ai", "idx_ai_decision_case_source"),
		"idx_ai_decision_case_source should exist")
	assert.True(t, indexExists(ctx, t, pool, "ai", "idx_ai_decision_case_active_source"),
		"idx_ai_decision_case_active_source should exist")
}

// -------------------------------------------------------------------------
// ai.action_proposal contract
// -------------------------------------------------------------------------

func TestMigrationContract_ActionProposal(t *testing.T) {
	t.Parallel()
	pool := setupTestDB(t)
	ctx := context.Background()

	require.True(t, tableExists(ctx, t, pool, "ai", "action_proposal"),
		"ai.action_proposal table must exist")

	cols := queryColumns(ctx, t, pool, "ai", "action_proposal")
	cm := columnMap(cols)

	expected := map[string]struct {
		dataType   string
		notNull    bool
	}{
		"proposal_id":           {"text", true},
		"case_id":               {"text", false},
		"decision_id":           {"text", false},
		"action_type":           {"text", false},
		"payload":               {"jsonb", false},
		"apply_status":          {"text", false},
		"created_at":            {"timestamp with time zone", false},
		"applied_at":            {"timestamp with time zone", false},
		"applied_by":            {"text", false},
		"title":                 {"text", true},
		"description":           {"text", false},
		"risk_level":            {"text", false},
		"requires_human_review": {"boolean", false},
	}

	for name, exp := range expected {
		c, ok := cm[name]
		if !ok {
			t.Errorf("column %q missing from ai.action_proposal", name)
			continue
		}
		assert.Equal(t, exp.dataType, c.DataType,
			"ai.action_proposal.%s type mismatch", name)
		if exp.notNull {
			assert.Equal(t, "NO", c.IsNullable,
				"ai.action_proposal.%s should be NOT NULL", name)
		}
	}

	// Verify CHECK constraints
	assert.True(t, constraintExists(ctx, t, pool, "ai", "chk_action_proposal_apply_status"),
		"chk_action_proposal_apply_status should exist")
	assert.True(t, constraintExists(ctx, t, pool, "ai", "chk_action_proposal_action_type"),
		"chk_action_proposal_action_type should exist")
	assert.True(t, constraintExists(ctx, t, pool, "ai", "chk_action_proposal_requires_review"),
		"chk_action_proposal_requires_review should exist")

	// Verify index
	assert.True(t, indexExists(ctx, t, pool, "ai", "idx_ai_action_proposal_case"),
		"idx_ai_action_proposal_case should exist")
}

// -------------------------------------------------------------------------
// ai.llm_decision contract
// -------------------------------------------------------------------------

func TestMigrationContract_LLMDecision(t *testing.T) {
	t.Parallel()
	pool := setupTestDB(t)
	ctx := context.Background()

	require.True(t, tableExists(ctx, t, pool, "ai", "llm_decision"),
		"ai.llm_decision table must exist")

	cols := queryColumns(ctx, t, pool, "ai", "llm_decision")
	cm := columnMap(cols)

	expected := map[string]struct {
		dataType   string
		notNull    bool
	}{
		"decision_id":      {"text", true},
		"case_id":          {"text", false},
		"model_version":    {"text", false},
		"prompt_hash":      {"text", false},
		"output_json":      {"jsonb", false},
		"confidence":       {"numeric", false},
		"created_at":       {"timestamp with time zone", false},
		"status":           {"text", false},
		"fallback_reason":  {"text", false},
		"validation_errors": {"jsonb", false},
		"provider":         {"text", false},
		"model":            {"text", false},
		"prompt_id":        {"text", false},
		"prompt_version":   {"text", false},
		"context_hash":     {"text", false},
		"input_json":       {"jsonb", false},
		"raw_output":       {"text", false},
		"parsed_output_json": {"jsonb", false},
		"validation_status": {"text", false},
		"fallback_used":    {"boolean", false},
		"token_prompt":     {"integer", false},
		"token_completion": {"integer", false},
		"latency_ms":       {"integer", false},
	}

	for name, exp := range expected {
		c, ok := cm[name]
		if !ok {
			t.Errorf("column %q missing from ai.llm_decision", name)
			continue
		}
		assert.Equal(t, exp.dataType, c.DataType,
			"ai.llm_decision.%s type mismatch", name)
		if exp.notNull {
			assert.Equal(t, "NO", c.IsNullable,
				"ai.llm_decision.%s should be NOT NULL", name)
		}
	}

	// Verify indexes
	assert.True(t, indexExists(ctx, t, pool, "ai", "idx_ai_llm_decision_case"),
		"idx_ai_llm_decision_case should exist")
	assert.True(t, indexExists(ctx, t, pool, "ai", "idx_llm_decision_provider"),
		"idx_llm_decision_provider should exist")
	assert.True(t, indexExists(ctx, t, pool, "ai", "idx_llm_decision_fallback"),
		"idx_llm_decision_fallback should exist")
}

// -------------------------------------------------------------------------
// ai.review_record contract
// -------------------------------------------------------------------------

func TestMigrationContract_ReviewRecord(t *testing.T) {
	t.Parallel()
	pool := setupTestDB(t)
	ctx := context.Background()

	require.True(t, tableExists(ctx, t, pool, "ai", "review_record"),
		"ai.review_record table must exist")

	cols := queryColumns(ctx, t, pool, "ai", "review_record")
	cm := columnMap(cols)

	expected := map[string]struct {
		dataType   string
		notNull    bool
	}{
		"review_id":   {"text", true},
		"proposal_id": {"text", false},
		"reviewer_id": {"text", false},
		"verdict":     {"text", false},
		"feedback":    {"text", false},
		"reviewed_at": {"timestamp with time zone", false},
	}

	for name, exp := range expected {
		c, ok := cm[name]
		if !ok {
			t.Errorf("column %q missing from ai.review_record", name)
			continue
		}
		assert.Equal(t, exp.dataType, c.DataType,
			"ai.review_record.%s type mismatch", name)
		if exp.notNull {
			assert.Equal(t, "NO", c.IsNullable,
				"ai.review_record.%s should be NOT NULL", name)
		}
	}

	// Verify CHECK constraint
	assert.True(t, constraintExists(ctx, t, pool, "ai", "chk_review_record_verdict"),
		"chk_review_record_verdict should exist")

	// Verify indexes
	assert.True(t, indexExists(ctx, t, pool, "ai", "idx_review_record_proposal_id"),
		"idx_review_record_proposal_id should exist")
}

// -------------------------------------------------------------------------
// ai.decision_eval_result contract (migration 012)
// -------------------------------------------------------------------------

func TestMigrationContract_DecisionEvalResult(t *testing.T) {
	t.Parallel()
	pool := setupTestDB(t)
	ctx := context.Background()

	require.True(t, tableExists(ctx, t, pool, "ai", "decision_eval_result"),
		"ai.decision_eval_result table must exist")

	cols := queryColumns(ctx, t, pool, "ai", "decision_eval_result")
	cm := columnMap(cols)

	expected := map[string]struct {
		dataType   string
		notNull    bool
	}{
		"eval_id":          {"text", true},
		"decision_case_id": {"text", true},
		"llm_decision_id":  {"text", false},
		"eval_rule_id":     {"text", false},
		"eval_status":      {"text", true},
		"score":            {"numeric", false},
		"details_json":     {"jsonb", false},
		"created_at":       {"timestamp with time zone", false},
	}

	for name, exp := range expected {
		c, ok := cm[name]
		if !ok {
			t.Errorf("column %q missing from ai.decision_eval_result", name)
			continue
		}
		assert.Equal(t, exp.dataType, c.DataType,
			"ai.decision_eval_result.%s type mismatch", name)
		if exp.notNull {
			assert.Equal(t, "NO", c.IsNullable,
				"ai.decision_eval_result.%s should be NOT NULL", name)
		}
	}

	// Verify indexes
	assert.True(t, indexExists(ctx, t, pool, "ai", "idx_eval_result_case"),
		"idx_eval_result_case should exist")
	assert.True(t, indexExists(ctx, t, pool, "ai", "idx_eval_result_decision"),
		"idx_eval_result_decision should exist")
}
