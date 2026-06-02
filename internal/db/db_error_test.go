package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewPool_InvalidDSN(t *testing.T) {
	logger := zap.NewNop()
	pool, err := NewPool(context.Background(), "postgres://invalid:invalid@localhost:1/bad?sslmode=disable", logger)
	require.Error(t, err)
	assert.Nil(t, pool)
}

func TestNewPool_EmptyDSN(t *testing.T) {
	logger := zap.NewNop()
	pool, err := NewPool(context.Background(), "", logger)
	require.Error(t, err)
	assert.Nil(t, pool)
}

func TestPool_NewStdDB_NilEmbedded(t *testing.T) {
	p := &Pool{Pool: nil, logger: zap.NewNop()}
	assert.NotPanics(t, func() {
		stdDB := NewStdDB(p)
		_ = stdDB
	})
}
