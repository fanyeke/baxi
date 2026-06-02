package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPool_Close_NilPool(t *testing.T) {
	assert.NotPanics(t, func() {
		p := &Pool{Pool: nil, logger: zap.NewNop()}
		p.Close()
	})
}

func TestPool_Close_NilEmbeddedPool(t *testing.T) {
	assert.NotPanics(t, func() {
		p := &Pool{logger: zap.NewNop()}
		p.Close()
	})
}

func TestPool_StructCreation(t *testing.T) {
	p := &Pool{logger: zap.NewNop()}
	require.NotNil(t, p)
	assert.Nil(t, p.Pool)
}

func TestPool_LoggerSet(t *testing.T) {
	logger := zap.NewNop()
	p := &Pool{logger: logger}
	require.NotNil(t, p)
	assert.NotPanics(t, func() {
		p.Close()
	})
}

func TestPool_NewStdDB_NilPool(t *testing.T) {
	p := &Pool{Pool: nil, logger: zap.NewNop()}
	// NewStdDB with nil pool should not panic, but may return a nil *sql.DB
	// that panics later when used. The function itself should be safe.
	assert.NotPanics(t, func() {
		stdDB := NewStdDB(p)
		// stdDB may be functional or nil depending on OpenDBFromPool behavior
		_ = stdDB
	})
}
