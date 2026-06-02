package common

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──── PoolProvider tests ────────────────────────────────────────────────────

func TestNewPoolProvider(t *testing.T) {
	pp := NewPoolProvider(nil)
	assert.NotNil(t, pp)
	assert.Nil(t, pp.Pool())
}

func TestPoolProvider_Pool(t *testing.T) {
	pp := NewPoolProvider(nil)
	assert.Nil(t, pp.Pool())
}

// ──── MockRows tests ────────────────────────────────────────────────────────

func TestMockRows_Next(t *testing.T) {
	rows := NewMockRows([][]interface{}{
		{"a", 1},
		{"b", 2},
	})
	assert.True(t, rows.Next())
	assert.True(t, rows.Next())
	assert.False(t, rows.Next())
}

func TestMockRows_Scan(t *testing.T) {
	rows := NewMockRows([][]interface{}{
		{"hello", 42},
	})
	require.True(t, rows.Next())

	var s string
	var i int
	err := rows.Scan(&s, &i)
	require.NoError(t, err)
	assert.Equal(t, "hello", s)
	assert.Equal(t, 42, i)
}

func TestMockRows_ScanFunc(t *testing.T) {
	rows := NewMockRows([][]interface{}{{"a"}})
	rows.scanFunc = func(dest ...interface{}) error {
		return fmt.Errorf("scan error")
	}
	require.True(t, rows.Next())

	var s string
	err := rows.Scan(&s)
	assert.Error(t, err)
}

func TestMockRows_Close(t *testing.T) {
	rows := NewMockRows([][]interface{}{{"a"}})
	assert.False(t, rows.closed)
	rows.Close()
	assert.True(t, rows.closed)
}

func TestMockRows_Err(t *testing.T) {
	rows := NewMockRows([][]interface{}{})
	assert.NoError(t, rows.Err())

	rows.errFunc = func() error { return fmt.Errorf("iteration error") }
	assert.Error(t, rows.Err())
}

func TestMockRows_CommandTag(t *testing.T) {
	rows := NewMockRows([][]interface{}{})
	tag := rows.CommandTag()
	assert.NotNil(t, tag)
}

func TestMockRows_FieldDescriptions(t *testing.T) {
	rows := NewMockRows([][]interface{}{})
	assert.Nil(t, rows.FieldDescriptions())
}

func TestMockRows_Values(t *testing.T) {
	rows := NewMockRows([][]interface{}{{"a", 1}})
	require.True(t, rows.Next())

	vals, err := rows.Values()
	require.NoError(t, err)
	assert.Len(t, vals, 2)
}

func TestMockRows_Values_NoCurrentRow(t *testing.T) {
	rows := NewMockRows([][]interface{}{})
	_, err := rows.Values()
	assert.Error(t, err)
}

func TestMockRows_RawValues(t *testing.T) {
	rows := NewMockRows([][]interface{}{})
	assert.Nil(t, rows.RawValues())
}

func TestMockRows_Conn(t *testing.T) {
	rows := NewMockRows([][]interface{}{})
	assert.Nil(t, rows.Conn())
}

func TestMockRows_Empty(t *testing.T) {
	rows := NewMockRows([][]interface{}{})
	assert.False(t, rows.Next())
}

func TestMockRows_MultipleScanCalls(t *testing.T) {
	rows := NewMockRows([][]interface{}{
		{"first", 1},
		{"second", 2},
	})

	require.True(t, rows.Next())
	var s string
	var i int
	require.NoError(t, rows.Scan(&s, &i))
	assert.Equal(t, "first", s)
	assert.Equal(t, 1, i)

	require.True(t, rows.Next())
	require.NoError(t, rows.Scan(&s, &i))
	assert.Equal(t, "second", s)
	assert.Equal(t, 2, i)

	assert.False(t, rows.Next())
}

// ──── MockRow tests ─────────────────────────────────────────────────────────

func TestNewMockRow(t *testing.T) {
	row := NewMockRow("hello", 42, true)
	assert.NotNil(t, row)
}

func TestMockRow_Scan(t *testing.T) {
	row := NewMockRow("hello", 42, true)

	var s string
	var i int
	var b bool
	err := row.Scan(&s, &i, &b)
	require.NoError(t, err)
	assert.Equal(t, "hello", s)
	assert.Equal(t, 42, i)
	assert.True(t, b)
}

func TestMockRow_ScanError(t *testing.T) {
	row := NewMockRowError(fmt.Errorf("scan error"))
	var s string
	err := row.Scan(&s)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scan error")
}

func TestMockRow_ScanFunc(t *testing.T) {
	row := NewMockRow("a")
	row.scanFunc = func(dest ...interface{}) error {
		return fmt.Errorf("custom scan error")
	}
	var s string
	err := row.Scan(&s)
	assert.Error(t, err)
}

func TestMockRow_NilValues(t *testing.T) {
	row := NewMockRow(nil, nil)
	var s string
	var i int
	err := row.Scan(&s, &i)
	require.NoError(t, err)
	// Nil values don't get set, so s and i remain zero values
	assert.Equal(t, "", s)
	assert.Equal(t, 0, i)
}

// ──── MockTx tests ──────────────────────────────────────────────────────────

func TestMockTx_Commit(t *testing.T) {
	tx := &MockTx{}
	err := tx.Commit(context.Background())
	require.NoError(t, err)
	assert.True(t, tx.CommitCalled)
}

func TestMockTx_CommitError(t *testing.T) {
	tx := &MockTx{
		commitFunc: func() error { return fmt.Errorf("commit failed") },
	}
	err := tx.Commit(context.Background())
	assert.Error(t, err)
	assert.True(t, tx.CommitCalled)
}

func TestMockTx_Rollback(t *testing.T) {
	tx := &MockTx{}
	err := tx.Rollback(context.Background())
	require.NoError(t, err)
	assert.True(t, tx.RollbackCalled)
}

func TestMockTx_RollbackError(t *testing.T) {
	tx := &MockTx{
		rollbackFunc: func() error { return fmt.Errorf("rollback failed") },
	}
	err := tx.Rollback(context.Background())
	assert.Error(t, err)
	assert.True(t, tx.RollbackCalled)
}

func TestMockTx_Begin(t *testing.T) {
	tx := &MockTx{}
	nested, err := tx.Begin(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, nested)
}

func TestMockTx_CopyFrom(t *testing.T) {
	tx := &MockTx{}
	n, err := tx.CopyFrom(context.Background(), pgx.Identifier{"table"}, []string{"col"}, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)
}

func TestMockTx_SendBatch(t *testing.T) {
	tx := &MockTx{}
	result := tx.SendBatch(context.Background(), &pgx.Batch{})
	assert.Nil(t, result)
}

func TestMockTx_LargeObjects(t *testing.T) {
	tx := &MockTx{}
	lo := tx.LargeObjects()
	assert.NotNil(t, lo)
}

func TestMockTx_Prepare(t *testing.T) {
	tx := &MockTx{}
	desc, err := tx.Prepare(context.Background(), "stmt", "SELECT 1")
	require.NoError(t, err)
	assert.Nil(t, desc)
}

func TestMockTx_Exec(t *testing.T) {
	tx := &MockTx{}
	tag, err := tx.Exec(context.Background(), "INSERT INTO t VALUES (1)")
	require.NoError(t, err)
	assert.NotNil(t, tag)
}

func TestMockTx_ExecFunc(t *testing.T) {
	tx := &MockTx{
		execFunc: func(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, fmt.Errorf("exec error")
		},
	}
	_, err := tx.Exec(context.Background(), "INSERT INTO t VALUES (1)")
	assert.Error(t, err)
}

func TestMockTx_Query(t *testing.T) {
	tx := &MockTx{}
	rows, err := tx.Query(context.Background(), "SELECT 1")
	require.NoError(t, err)
	assert.NotNil(t, rows)
}

func TestMockTx_QueryFunc(t *testing.T) {
	tx := &MockTx{
		queryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return NewMockRows([][]interface{}{{"result"}}), nil
		},
	}
	rows, err := tx.Query(context.Background(), "SELECT 1")
	require.NoError(t, err)
	require.NotNil(t, rows)
	assert.True(t, rows.Next())
}

func TestMockTx_QueryRow(t *testing.T) {
	tx := &MockTx{}
	row := tx.QueryRow(context.Background(), "SELECT 1")
	assert.NotNil(t, row)
}

func TestMockTx_QueryRowFunc(t *testing.T) {
	tx := &MockTx{
		queryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return NewMockRow("result")
		},
	}
	row := tx.QueryRow(context.Background(), "SELECT 1")
	var s string
	err := row.Scan(&s)
	require.NoError(t, err)
	assert.Equal(t, "result", s)
}

func TestMockTx_Conn(t *testing.T) {
	tx := &MockTx{}
	assert.Nil(t, tx.Conn())
}

// ──── MockCommandTag ────────────────────────────────────────────────────────

func TestMockCommandTag(t *testing.T) {
	tag := MockCommandTag(5)
	assert.NotNil(t, tag)
}

// ──── MockQuerier tests ─────────────────────────────────────────────────────

func TestMockQuerier_Query(t *testing.T) {
	mq := &MockQuerier{}
	rows, err := mq.Query(context.Background(), "SELECT 1")
	require.NoError(t, err)
	assert.NotNil(t, rows)
	assert.Len(t, mq.QueryCalls, 1)
	assert.Equal(t, "SELECT 1", mq.QueryCalls[0].SQL)
}

func TestMockQuerier_QueryWithFunc(t *testing.T) {
	mq := &MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return NewMockRows([][]interface{}{{"custom"}}), nil
		},
	}
	rows, err := mq.Query(context.Background(), "SELECT 1")
	require.NoError(t, err)
	require.NotNil(t, rows)
	assert.True(t, rows.Next())
}

func TestMockQuerier_QueryRow(t *testing.T) {
	mq := &MockQuerier{}
	row := mq.QueryRow(context.Background(), "SELECT 1")
	assert.NotNil(t, row)
	assert.Len(t, mq.QueryRowCalls, 1)
}

func TestMockQuerier_QueryRowWithFunc(t *testing.T) {
	mq := &MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return NewMockRow("custom")
		},
	}
	row := mq.QueryRow(context.Background(), "SELECT 1")
	var s string
	err := row.Scan(&s)
	require.NoError(t, err)
	assert.Equal(t, "custom", s)
}

func TestMockQuerier_Exec(t *testing.T) {
	mq := &MockQuerier{}
	tag, err := mq.Exec(context.Background(), "INSERT INTO t VALUES (1)")
	require.NoError(t, err)
	assert.NotNil(t, tag)
	assert.Len(t, mq.ExecCalls, 1)
}

func TestMockQuerier_ExecWithFunc(t *testing.T) {
	mq := &MockQuerier{
		ExecFunc: func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, fmt.Errorf("exec error")
		},
	}
	_, err := mq.Exec(context.Background(), "INSERT INTO t VALUES (1)")
	assert.Error(t, err)
}

func TestMockQuerier_Begin(t *testing.T) {
	mq := &MockQuerier{}
	tx, err := mq.Begin(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, tx)
}

func TestMockQuerier_BeginWithFunc(t *testing.T) {
	mq := &MockQuerier{
		BeginFunc: func(ctx context.Context) (pgx.Tx, error) {
			return &MockTx{}, nil
		},
	}
	tx, err := mq.Begin(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, tx)
}

// ──── setScanValue tests ────────────────────────────────────────────────────

func TestSetScanValue_String(t *testing.T) {
	var s string
	setScanValue(&s, "hello")
	assert.Equal(t, "hello", s)
}

func TestSetScanValue_Int(t *testing.T) {
	var i int
	setScanValue(&i, 42)
	assert.Equal(t, 42, i)
}

func TestSetScanValue_Int64(t *testing.T) {
	var i64 int64
	setScanValue(&i64, int64(42))
	assert.Equal(t, int64(42), i64)

	var i64_2 int64
	setScanValue(&i64_2, 42)
	assert.Equal(t, int64(42), i64_2)

	var i64_3 int64
	setScanValue(&i64_3, 42.0)
	assert.Equal(t, int64(42), i64_3)
}

func TestSetScanValue_Float64(t *testing.T) {
	var f float64
	setScanValue(&f, 3.14)
	assert.InDelta(t, 3.14, f, 0.001)
}

func TestSetScanValue_Bool(t *testing.T) {
	var b bool
	setScanValue(&b, true)
	assert.True(t, b)
}

func TestSetScanValue_Bytes(t *testing.T) {
	var b []byte
	setScanValue(&b, []byte("hello"))
	assert.Equal(t, []byte("hello"), b)
}

func TestSetScanValue_StringPtr(t *testing.T) {
	var sp *string
	setScanValue(&sp, "hello")
	require.NotNil(t, sp)
	assert.Equal(t, "hello", *sp)
}

func TestSetScanValue_StringPtrFromPtr(t *testing.T) {
	var sp *string
	orig := "hello"
	setScanValue(&sp, &orig)
	require.NotNil(t, sp)
	assert.Equal(t, "hello", *sp)
}

func TestSetScanValue_Time(t *testing.T) {
	var t2 time.Time
	now := time.Now().UTC().Truncate(time.Second)
	setScanValue(&t2, now)
	assert.Equal(t, now, t2)
}

func TestSetScanValue_NilValue(t *testing.T) {
	var s string
	setScanValue(&s, nil)
	// Should not panic, s remains empty
	assert.Equal(t, "", s)
}

func TestSetScanValue_UnknownType(t *testing.T) {
	var ch chan int
	setScanValue(&ch, 42)
	// Should not panic, ch remains nil
	assert.Nil(t, ch)
}
