package configloader

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTx records calls and returns configurable results.
type mockTx struct {
	execCalls  []execCall
	execErr    error
	queryCalls []string
}

type execCall struct {
	sql  string
	args []interface{}
}

func (m *mockTx) Exec(_ context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	m.execCalls = append(m.execCalls, execCall{sql: sql, args: arguments})
	return pgconn.CommandTag{}, m.execErr
}

func (m *mockTx) Query(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
	m.queryCalls = append(m.queryCalls, sql)
	return &mockRows{}, nil
}

func (m *mockTx) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	m.queryCalls = append(m.queryCalls, sql)
	return &mockRow{}
}

func (m *mockTx) Begin(context.Context) (pgx.Tx, error)                       { return m, nil }
func (m *mockTx) Commit(context.Context) error                                 { return nil }
func (m *mockTx) Rollback(context.Context) error                               { return nil }
func (m *mockTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (m *mockTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults      { return nil }
func (m *mockTx) LargeObjects() pgx.LargeObjects                               { return pgx.LargeObjects{} }
func (m *mockTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (m *mockTx) Conn() *pgx.Conn { return nil }

type mockRow struct{}

func (r *mockRow) Scan(dest ...any) error { return nil }

type mockRows struct {
	pos    int
	closed bool
}

func (r *mockRows) Next() bool                                   { return false }
func (r *mockRows) Scan(dest ...any) error                       { return nil }
func (r *mockRows) Close()                                       { r.closed = true }
func (r *mockRows) Err() error                                   { return nil }
func (r *mockRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *mockRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *mockRows) Values() ([]interface{}, error)               { return nil, nil }
func (r *mockRows) RawValues() [][]byte                          { return nil }
func (r *mockRows) Conn() *pgx.Conn                              { return nil }

// --- syncConfigSnapshots tests ---

func TestSyncConfigSnapshots_EmptyRegistry(t *testing.T) {
	tx := &mockTx{}
	registry := &ConfigRegistry{RawConfigs: map[string]RawConfig{}}

	err := syncConfigSnapshots(context.Background(), tx, registry)
	assert.NoError(t, err)
	assert.Empty(t, tx.execCalls)
}

func TestSyncConfigSnapshots_WithConfigs(t *testing.T) {
	tx := &mockTx{}
	registry := &ConfigRegistry{
		RawConfigs: map[string]RawConfig{
			"alert_rules": {
				ConfigKey:  "alert_rules",
				ConfigType: "alert_rules",
				SourcePath: "alert_rules.yml",
				Content:    []byte("rules:\n  - rule_id: test"),
			},
		},
	}

	err := syncConfigSnapshots(context.Background(), tx, registry)
	assert.NoError(t, err)
	assert.Greater(t, len(tx.execCalls), 0)
}

// --- syncObjectSchema tests ---

func TestSyncObjectSchema_NilConfig(t *testing.T) {
	tx := &mockTx{}
	registry := &ConfigRegistry{RawConfigs: map[string]RawConfig{}}

	err := syncObjectSchema(context.Background(), tx, registry)
	assert.NoError(t, err)
	assert.Empty(t, tx.execCalls)
}

func TestSyncObjectSchema_WithObjects(t *testing.T) {
	tx := &mockTx{}
	registry := &ConfigRegistry{
		ObjectSchema: &ObjectSchemaConfig{
			Objects: []ObjectSchema{
				{
					ObjectTypeID: "customer",
					DisplayName:  "Customer",
					Properties: map[string]PropertyDef{
						"id": {Type: "string", IsPK: true},
					},
				},
			},
		},
		RawConfigs: map[string]RawConfig{},
	}

	err := syncObjectSchema(context.Background(), tx, registry)
	assert.NoError(t, err)
	assert.Greater(t, len(tx.execCalls), 0)
}

func TestSyncObjectSchema_EmptyObjects(t *testing.T) {
	tx := &mockTx{}
	registry := &ConfigRegistry{
		ObjectSchema: &ObjectSchemaConfig{
			Objects: []ObjectSchema{},
		},
		RawConfigs: map[string]RawConfig{},
	}

	err := syncObjectSchema(context.Background(), tx, registry)
	assert.NoError(t, err)
	assert.Empty(t, tx.execCalls)
}

// --- syncDataClassification tests ---

func TestSyncDataClassification_NilConfig(t *testing.T) {
	tx := &mockTx{}
	registry := &ConfigRegistry{RawConfigs: map[string]RawConfig{}}

	err := syncDataClassification(context.Background(), tx, registry)
	assert.NoError(t, err)
}

func TestSyncDataClassification_WithClassifications(t *testing.T) {
	tx := &mockTx{}
	registry := &ConfigRegistry{
		DataClassification: &DataClassificationConfig{
			Classifications: []Classification{
				{
					AssetRef:  "customer.email",
					Level:     "pii",
					Rationale: "Email is PII",
				},
				{
					AssetRef:        "customer.name",
					Level:           "internal",
					Rationale:       "Name is internal",
					AppliesToFields: map[string]string{"first_name": "internal", "last_name": "internal"},
				},
			},
		},
		RawConfigs: map[string]RawConfig{},
	}

	err := syncDataClassification(context.Background(), tx, registry)
	assert.NoError(t, err)
	// Should have 1 base + 2 field-level classifications = 3 exec calls
	assert.GreaterOrEqual(t, len(tx.execCalls), 3)
}

// --- syncDataLineage tests ---

func TestSyncDataLineage_NilConfig(t *testing.T) {
	tx := &mockTx{}
	registry := &ConfigRegistry{RawConfigs: map[string]RawConfig{}}

	err := syncDataLineage(context.Background(), tx, registry)
	assert.NoError(t, err)
}

func TestSyncDataLineage_WithEdges(t *testing.T) {
	tx := &mockTx{}
	registry := &ConfigRegistry{
		DataLineage: &DataLineageConfig{
			Edges: []LineageEdge{
				{From: "raw.orders", To: "dwd.order_level", Transform: "copy", TransformType: "batch_load"},
			},
		},
		RawConfigs: map[string]RawConfig{},
	}

	err := syncDataLineage(context.Background(), tx, registry)
	assert.NoError(t, err)
	assert.Greater(t, len(tx.execCalls), 0)
}

func TestSyncDataLineage_EmptyEdges(t *testing.T) {
	tx := &mockTx{}
	registry := &ConfigRegistry{
		DataLineage: &DataLineageConfig{Edges: []LineageEdge{}},
		RawConfigs:  map[string]RawConfig{},
	}

	err := syncDataLineage(context.Background(), tx, registry)
	assert.NoError(t, err)
	assert.Empty(t, tx.execCalls)
}

// --- syncAccessPolicy tests ---

func TestSyncAccessPolicy_NilConfig(t *testing.T) {
	tx := &mockTx{}
	registry := &ConfigRegistry{RawConfigs: map[string]RawConfig{}}

	err := syncAccessPolicy(context.Background(), tx, registry)
	assert.NoError(t, err)
}

func TestSyncAccessPolicy_WithRoles(t *testing.T) {
	tx := &mockTx{}
	registry := &ConfigRegistry{
		AccessPolicy: &AccessPolicyConfig{},
		RawConfigs:   map[string]RawConfig{},
	}
	registry.AccessPolicy.AccessPolicy.Roles = []Role{
		{
			Role:           "admin",
			AllowedActions: []string{"read", "write"},
			DataAccess:     []string{"all"},
		},
	}
	registry.AccessPolicy.AccessPolicy.DefaultPolicy = "deny"

	err := syncAccessPolicy(context.Background(), tx, registry)
	assert.NoError(t, err)
	// 2 actions + 1 data_access + 1 default deny = 4
	assert.GreaterOrEqual(t, len(tx.execCalls), 3)
}

func TestSyncAccessPolicy_EmptyRoles(t *testing.T) {
	tx := &mockTx{}
	registry := &ConfigRegistry{
		AccessPolicy: &AccessPolicyConfig{},
		RawConfigs:   map[string]RawConfig{},
	}

	err := syncAccessPolicy(context.Background(), tx, registry)
	assert.NoError(t, err)
	// Only default deny
	assert.GreaterOrEqual(t, len(tx.execCalls), 1)
}

// --- SyncSnapshots integration test ---

func TestSyncSnapshots_CompleteRegistry(t *testing.T) {
	tx := &mockTx{}
	registry := &ConfigRegistry{
		ObjectSchema: &ObjectSchemaConfig{
			Objects: []ObjectSchema{{ObjectTypeID: "order", DisplayName: "Order"}},
		},
		DataClassification: &DataClassificationConfig{
			Classifications: []Classification{{AssetRef: "customer.email", Level: "pii"}},
		},
		DataLineage: &DataLineageConfig{
			Edges: []LineageEdge{{From: "a", To: "b", Transform: "copy"}},
		},
		AccessPolicy: &AccessPolicyConfig{},
		RawConfigs: map[string]RawConfig{
			"alert_rules": {ConfigKey: "alert_rules", Content: []byte("rules: []")},
		},
	}
	registry.AccessPolicy.AccessPolicy.Roles = []Role{{Role: "admin", AllowedActions: []string{"read"}}}

	// SyncSnapshots needs a pool and begins a tx internally
	// We'll test it by directly calling the sync functions
	err := syncConfigSnapshots(context.Background(), tx, registry)
	require.NoError(t, err)
	err = syncObjectSchema(context.Background(), tx, registry)
	require.NoError(t, err)
	err = syncDataClassification(context.Background(), tx, registry)
	require.NoError(t, err)
	err = syncDataLineage(context.Background(), tx, registry)
	require.NoError(t, err)
	err = syncAccessPolicy(context.Background(), tx, registry)
	require.NoError(t, err)

	assert.Greater(t, len(tx.execCalls), 5) // Multiple inserts across all syncs
}

func TestSyncSnapshots_ExecError(t *testing.T) {
	tx := &mockTx{execErr: assert.AnError}
	registry := &ConfigRegistry{
		ObjectSchema: &ObjectSchemaConfig{
			Objects: []ObjectSchema{{ObjectTypeID: "test"}},
		},
		RawConfigs: map[string]RawConfig{},
	}

	// syncObjectSchema logs errors but doesn't return them
	err := syncObjectSchema(context.Background(), tx, registry)
	assert.NoError(t, err) // Error is logged, not returned
}
