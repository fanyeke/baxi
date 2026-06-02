package steps

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"baxi/internal/pipeline"
)

// mockTxSteps is a minimal pgx.Tx for pipeline step tests.
type mockTxSteps struct {
	execHandler  func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	queryHandler func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	queryRowFunc func(ctx context.Context, sql string, args ...any) pgx.Row
}

func (m *mockTxSteps) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	if m.execHandler != nil {
		return m.execHandler(ctx, sql, arguments...)
	}
	return pgconn.CommandTag{}, nil
}

func (m *mockTxSteps) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if m.queryHandler != nil {
		return m.queryHandler(ctx, sql, args...)
	}
	return &mockStepsRows{}, nil
}

func (m *mockTxSteps) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if m.queryRowFunc != nil {
		return m.queryRowFunc(ctx, sql, args...)
	}
	return &mockStepsRow{}
}

func (m *mockTxSteps) Begin(context.Context) (pgx.Tx, error)                       { return m, nil }
func (m *mockTxSteps) Commit(context.Context) error                                 { return nil }
func (m *mockTxSteps) Rollback(context.Context) error                               { return nil }
func (m *mockTxSteps) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (m *mockTxSteps) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults      { return nil }
func (m *mockTxSteps) LargeObjects() pgx.LargeObjects                               { return pgx.LargeObjects{} }
func (m *mockTxSteps) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (m *mockTxSteps) Conn() *pgx.Conn { return nil }

type mockStepsRow struct {
	scanFn func(dest ...any) error
}

func (r *mockStepsRow) Scan(dest ...any) error {
	if r.scanFn != nil {
		return r.scanFn(dest...)
	}
	return nil
}

type mockStepsRows struct {
	data   [][]interface{}
	pos    int
	closed bool
}

func (r *mockStepsRows) Next() bool {
	if r.pos >= len(r.data) {
		return false
	}
	r.pos++
	return true
}

func (r *mockStepsRows) Scan(dest ...any) error { return nil }
func (r *mockStepsRows) Close()                 { r.closed = true }
func (r *mockStepsRows) Err() error             { return nil }
func (r *mockStepsRows) CommandTag() pgconn.CommandTag              { return pgconn.CommandTag{} }
func (r *mockStepsRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *mockStepsRows) Values() ([]interface{}, error)             { return nil, nil }
func (r *mockStepsRows) RawValues() [][]byte                        { return nil }
func (r *mockStepsRows) Conn() *pgx.Conn                            { return nil }

func newTestInput() pipeline.StepInput {
	return pipeline.StepInput{
		RunID:   "test-run-001",
		Logger:  zap.NewNop(),
		DataDir: "/tmp/test",
	}
}

// --- DetectAlertsStep.Run tests ---

func TestDetectAlertsStep_Run_NoAlerts(t *testing.T) {
	step := NewDetectAlertsStep()
	tx := &mockTxSteps{
		queryRowFunc: func(_ context.Context, sql string, _ ...any) pgx.Row {
			return &mockStepsRow{
				scanFn: func(dest ...any) error {
					// Return a valid date for getLatestDate
					if len(dest) > 0 {
						if tp, ok := dest[0].(*time.Time); ok {
							*tp = time.Date(2018, 10, 21, 0, 0, 0, 0, time.UTC)
						}
					}
					return nil
				},
			}
		},
		queryHandler: func(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
			// Return empty results for all queries
			return &mockStepsRows{data: [][]interface{}{}}, nil
		},
	}

	output, err := step.Run(context.Background(), tx, newTestInput())
	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, int64(0), output.OutputCount)
}

func TestDetectAlertsStep_Run_GlobalRuleError(t *testing.T) {
	step := NewDetectAlertsStep()
	tx := &mockTxSteps{
		queryRowFunc: func(_ context.Context, sql string, _ ...any) pgx.Row {
			return &mockStepsRow{
				scanFn: func(dest ...any) error {
					return assert.AnError
				},
			}
		},
	}

	_, err := step.Run(context.Background(), tx, newTestInput())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "detect_alerts")
}

func TestDetectAlertsStep_Run_ExecError(t *testing.T) {
	step := NewDetectAlertsStep()
	tx := &mockTxSteps{
		queryRowFunc: func(_ context.Context, sql string, _ ...any) pgx.Row {
			return &mockStepsRow{
				scanFn: func(dest ...any) error {
					return nil
				},
			}
		},
		queryHandler: func(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
			return &mockStepsRows{data: [][]interface{}{}}, nil
		},
		execHandler: func(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, assert.AnError
		},
	}

	_, err := step.Run(context.Background(), tx, newTestInput())
	assert.Error(t, err)
}
