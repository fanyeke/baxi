package common

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// MockQuerier is a mock implementation of the Querier interface for testing.
type MockQuerier struct {
	QueryFunc    func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRowFunc func(ctx context.Context, sql string, args ...interface{}) pgx.Row
	ExecFunc     func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
	BeginFunc    func(ctx context.Context) (pgx.Tx, error)

	// Call tracking
	QueryCalls    []QueryCall
	QueryRowCalls []QueryCall
	ExecCalls     []QueryCall
}

// QueryCall records a single call to Query, QueryRow, or Exec.
type QueryCall struct {
	SQL  string
	Args []interface{}
}

func (m *MockQuerier) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	m.QueryCalls = append(m.QueryCalls, QueryCall{SQL: sql, Args: args})
	if m.QueryFunc != nil {
		return m.QueryFunc(ctx, sql, args...)
	}
	return &MockRows{}, nil
}

func (m *MockQuerier) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	m.QueryRowCalls = append(m.QueryRowCalls, QueryCall{SQL: sql, Args: args})
	if m.QueryRowFunc != nil {
		return m.QueryRowFunc(ctx, sql, args...)
	}
	return &MockRow{}
}

func (m *MockQuerier) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	m.ExecCalls = append(m.ExecCalls, QueryCall{SQL: sql, Args: args})
	if m.ExecFunc != nil {
		return m.ExecFunc(ctx, sql, args...)
	}
	return pgconn.CommandTag{}, nil
}

func (m *MockQuerier) Begin(ctx context.Context) (pgx.Tx, error) {
	if m.BeginFunc != nil {
		return m.BeginFunc(ctx)
	}
	return &MockTx{}, nil
}

// MockRows implements pgx.Rows for testing.
type MockRows struct {
	rowsData      [][]interface{}
	index         int
	lastNextIndex int
	closed        bool
	errFunc       func() error
	scanFunc      func(dest ...interface{}) error
}

// NewMockRows creates a MockRows with the given row data.
func NewMockRows(rows [][]interface{}) *MockRows {
	return &MockRows{rowsData: rows}
}

func (m *MockRows) Next() bool {
	if m.index < len(m.rowsData) {
		m.lastNextIndex = m.index
		m.index++
		return true
	}
	return false
}

func (m *MockRows) Scan(dest ...interface{}) error {
	if m.scanFunc != nil {
		return m.scanFunc(dest...)
	}
	if m.lastNextIndex >= len(m.rowsData) {
		return fmt.Errorf("no more rows")
	}
	row := m.rowsData[m.lastNextIndex]
	for i, d := range dest {
		if i < len(row) {
			setScanValue(d, row[i])
		}
	}
	return nil
}

func (m *MockRows) Close() {
	m.closed = true
}

func (m *MockRows) Err() error {
	if m.errFunc != nil {
		return m.errFunc()
	}
	return nil
}

func (m *MockRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (m *MockRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (m *MockRows) Values() ([]any, error) {
	if m.lastNextIndex >= len(m.rowsData) {
		return nil, fmt.Errorf("no current row")
	}
	return m.rowsData[m.lastNextIndex], nil
}

func (m *MockRows) RawValues() [][]byte {
	return nil
}

func (m *MockRows) Conn() *pgx.Conn {
	return nil
}

// MockRow implements pgx.Row for testing.
type MockRow struct {
	values   []interface{}
	err      error
	scanFunc func(dest ...interface{}) error
}

// NewMockRow creates a MockRow with the given values.
func NewMockRow(values ...interface{}) *MockRow {
	return &MockRow{values: values}
}

// NewMockRowError creates a MockRow that returns an error on Scan.
func NewMockRowError(err error) *MockRow {
	return &MockRow{err: err}
}

func (m *MockRow) Scan(dest ...interface{}) error {
	if m.scanFunc != nil {
		return m.scanFunc(dest...)
	}
	if m.err != nil {
		return m.err
	}
	for i, d := range dest {
		if i < len(m.values) {
			setScanValue(d, m.values[i])
		}
	}
	return nil
}

// MockTx implements pgx.Tx for testing.
type MockTx struct {
	queryFunc    func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	queryRowFunc func(ctx context.Context, sql string, args ...interface{}) pgx.Row
	execFunc     func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
	commitFunc   func() error
	rollbackFunc func() error

	CommitCalled   bool
	RollbackCalled bool
}

func (m *MockTx) Begin(ctx context.Context) (pgx.Tx, error) {
	return &MockTx{}, nil
}

func (m *MockTx) Commit(ctx context.Context) error {
	m.CommitCalled = true
	if m.commitFunc != nil {
		return m.commitFunc()
	}
	return nil
}

func (m *MockTx) Rollback(ctx context.Context) error {
	m.RollbackCalled = true
	if m.rollbackFunc != nil {
		return m.rollbackFunc()
	}
	return nil
}

func (m *MockTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}

func (m *MockTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return nil
}

func (m *MockTx) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}

func (m *MockTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, nil
}

func (m *MockTx) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	if m.execFunc != nil {
		return m.execFunc(ctx, sql, arguments...)
	}
	return pgconn.CommandTag{}, nil
}

func (m *MockTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if m.queryFunc != nil {
		return m.queryFunc(ctx, sql, args...)
	}
	return &MockRows{}, nil
}

func (m *MockTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if m.queryRowFunc != nil {
		return m.queryRowFunc(ctx, sql, args...)
	}
	return &MockRow{}
}

func (m *MockTx) Conn() *pgx.Conn {
	return nil
}

// MockCommandTag creates a pgconn.CommandTag with a specific RowsAffected value.
func MockCommandTag(rowsAffected int64) pgconn.CommandTag {
	return pgconn.NewCommandTag(fmt.Sprintf("INSERT 0 %d", rowsAffected))
}

// setScanValue sets the value of a scan destination.
func setScanValue(dest interface{}, value interface{}) {
	if value == nil {
		return
	}

	switch d := dest.(type) {
	case *string:
		if s, ok := value.(string); ok {
			*d = s
		}
	case *int:
		if i, ok := value.(int); ok {
			*d = i
		}
	case *int64:
		switch v := value.(type) {
		case int64:
			*d = v
		case int:
			*d = int64(v)
		case float64:
			*d = int64(v)
		}
	case *float64:
		if f, ok := value.(float64); ok {
			*d = f
		}
	case *bool:
		if b, ok := value.(bool); ok {
			*d = b
		}
	case *[]byte:
		if b, ok := value.([]byte); ok {
			*d = b
		}
	case **string:
		if s, ok := value.(string); ok {
			*d = &s
		} else if s, ok := value.(*string); ok {
			*d = s
		}
	case *time.Time:
		if t, ok := value.(time.Time); ok {
			*d = t
		}
	}
}
