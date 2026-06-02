package alert

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// mockTx is a minimal pgx.Tx implementation for unit testing.
// It routes Query/QueryRow/Exec calls to pre-configured handlers.
type mockTx struct {
	execHandler  func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	queryHandler func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	queryRowFunc func(ctx context.Context, sql string, args ...any) pgx.Row
}

func (m *mockTx) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	if m.execHandler != nil {
		return m.execHandler(ctx, sql, arguments...)
	}
	return pgconn.CommandTag{}, nil
}

func (m *mockTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if m.queryHandler != nil {
		return m.queryHandler(ctx, sql, args...)
	}
	return &mockRows{}, nil
}

func (m *mockTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if m.queryRowFunc != nil {
		return m.queryRowFunc(ctx, sql, args...)
	}
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

// mockRow implements pgx.Row for QueryRow returns.
type mockRow struct {
	scanFn func(dest ...any) error
}

func (r *mockRow) Scan(dest ...any) error {
	if r.scanFn != nil {
		return r.scanFn(dest...)
	}
	return nil
}

// mockRows implements pgx.Rows for Query returns.
type mockRows struct {
	data   [][]interface{}
	pos    int
	closed bool
	err    error
}

func (r *mockRows) Next() bool {
	if r.closed || r.pos >= len(r.data) {
		return false
	}
	r.pos++
	return true
}

func (r *mockRows) Scan(dest ...any) error {
	if r.pos == 0 || r.pos > len(r.data) {
		return pgx.ErrNoRows
	}
	row := r.data[r.pos-1]
	for i, d := range dest {
		if i < len(row) {
			setDest(d, row[i])
		}
	}
	return nil
}

func (r *mockRows) Close()                                     { r.closed = true }
func (r *mockRows) Err() error                                { return r.err }
func (r *mockRows) CommandTag() pgconn.CommandTag             { return pgconn.CommandTag{} }
func (r *mockRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *mockRows) Values() ([]interface{}, error)            { return nil, nil }
func (r *mockRows) RawValues() [][]byte                       { return nil }
func (r *mockRows) Conn() *pgx.Conn                           { return nil }

// setDest assigns src to dest using type switches for pgx scan targets.
func setDest(dest, src interface{}) {
	switch d := dest.(type) {
	case *string:
		if s, ok := src.(string); ok {
			*d = s
		}
	case *float64:
		switch v := src.(type) {
		case float64:
			*d = v
		case int64:
			*d = float64(v)
		case int:
			*d = float64(v)
		}
	case *int64:
		switch v := src.(type) {
		case int64:
			*d = v
		case int:
			*d = int64(v)
		case float64:
			*d = int64(v)
		}
	case *bool:
		if b, ok := src.(bool); ok {
			*d = b
		}
	case *interface{}:
		*d = src
	}
}
