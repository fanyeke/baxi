package common

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoolProvider_New(t *testing.T) {
	assert.NotPanics(t, func() {
		p := NewPoolProvider(nil)
		assert.Nil(t, p.Pool())
	})
}

func TestPoolProvider_PoolReturnsNil(t *testing.T) {
	p := NewPoolProvider(nil)
	assert.Nil(t, p.Pool())
}

func TestPoolProvider_NewWithPool(t *testing.T) {
	// Create a minimal PoolProvider without an actual database pool.
	p := &PoolProvider{}
	assert.Nil(t, p.Pool())
}

func TestPoolProvider_PoolGetter(t *testing.T) {
	// The PoolProvider should return whatever pool was passed to NewPoolProvider.
	// With nil, Pool() should return nil.
	p := NewPoolProvider(nil)
	assert.Nil(t, p.Pool())
}

func TestPoolProvider_QueryWithNilPool(t *testing.T) {
	p := NewPoolProvider(nil)
	// Calling Query on nil pool should panic because pgxpool.Pool is not nil-safe.
	assert.Panics(t, func() {
		_, _ = p.Query(context.Background(), "SELECT 1")
	})
}

func TestPoolProvider_QueryRowWithNilPool(t *testing.T) {
	p := NewPoolProvider(nil)
	assert.Panics(t, func() {
		_ = p.QueryRow(context.Background(), "SELECT 1")
	})
}

func TestPoolProvider_BeginWithNilPool(t *testing.T) {
	p := NewPoolProvider(nil)
	assert.Panics(t, func() {
		_, _ = p.Begin(context.Background())
	})
}

// TestPoolProvider_ConstructorAndFields ensures the struct fields are accessible.
func TestPoolProvider_ConstructorAndFields(t *testing.T) {
	p := NewPoolProvider(nil)
	require.NotNil(t, p)
	assert.Nil(t, p.Pool())
}
