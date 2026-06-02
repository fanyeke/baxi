package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──── mockTx for LoadFixtureCSV tests ───────────────────────────────────────

type testMockTx struct {
	execFunc func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
	execCalls []execCall
}

type execCall struct {
	sql  string
	args []interface{}
}

func (m *testMockTx) Begin(ctx context.Context) (pgx.Tx, error) {
	return nil, nil
}

func (m *testMockTx) Commit(ctx context.Context) error {
	return nil
}

func (m *testMockTx) Rollback(ctx context.Context) error {
	return nil
}

func (m *testMockTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}

func (m *testMockTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return nil
}

func (m *testMockTx) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}

func (m *testMockTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, nil
}

func (m *testMockTx) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	m.execCalls = append(m.execCalls, execCall{sql: sql, args: arguments})
	if m.execFunc != nil {
		return m.execFunc(ctx, sql, arguments...)
	}
	return pgconn.CommandTag{}, nil
}

func (m *testMockTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}

func (m *testMockTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return nil
}

func (m *testMockTx) Conn() *pgx.Conn {
	return nil
}

// ──── LoadFixtureCSV tests ──────────────────────────────────────────────────

func TestLoadFixtureCSV_Success(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "test.csv")
	content := "id,name,value\n1,test,100\n2,test2,200\n"
	require.NoError(t, os.WriteFile(csvPath, []byte(content), 0644))

	tx := &testMockTx{}
	err := LoadFixtureCSV(tx, "test_table", csvPath)
	require.NoError(t, err)
	assert.Len(t, tx.execCalls, 2)
	assert.Contains(t, tx.execCalls[0].sql, "INSERT INTO test_table")
}

func TestLoadFixtureCSV_FileNotFound(t *testing.T) {
	tx := &testMockTx{}
	err := LoadFixtureCSV(tx, "test_table", "/nonexistent/file.csv")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "open fixture file")
}

func TestLoadFixtureCSV_TooFewRows(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "header_only.csv")
	content := "id,name,value\n"
	require.NoError(t, os.WriteFile(csvPath, []byte(content), 0644))

	tx := &testMockTx{}
	err := LoadFixtureCSV(tx, "test_table", csvPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "need header + 1 row")
}

func TestLoadFixtureCSV_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "empty.csv")
	require.NoError(t, os.WriteFile(csvPath, []byte(""), 0644))

	tx := &testMockTx{}
	err := LoadFixtureCSV(tx, "test_table", csvPath)
	assert.Error(t, err)
}

func TestLoadFixtureCSV_ExecError(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "test.csv")
	content := "id,name\n1,test\n"
	require.NoError(t, os.WriteFile(csvPath, []byte(content), 0644))

	tx := &testMockTx{
		execFunc: func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, fmt.Errorf("insert failed")
		},
	}
	err := LoadFixtureCSV(tx, "test_table", csvPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insert into test_table")
}

// ──── PostgresContainer tests ───────────────────────────────────────────────

func TestPostgresContainer_ConnectionString(t *testing.T) {
	c := &PostgresContainer{ConnStr: "postgres://test:test@localhost:5432/testdb"}
	assert.Equal(t, "postgres://test:test@localhost:5432/testdb", c.ConnectionString())
}

func TestPostgresContainer_Terminate_NilContainer(t *testing.T) {
	c := &PostgresContainer{Container: nil}
	err := c.Terminate(context.Background())
	assert.NoError(t, err)
}

// ──── requireDBSkip tests ───────────────────────────────────────────────────

func TestRequireDBSkip_NotShort(t *testing.T) {
	// When not running with -short, requireDBSkip should not skip
	// We can't easily test the skip path without modifying test flags
	// But we can verify the function doesn't panic
	if !testing.Short() {
		requireDBSkip(t)
	}
}
