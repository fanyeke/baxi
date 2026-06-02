package ontology

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ──── resolveLimit ──────────────────────────────────────────────────────

func TestResolveLimit_Zero(t *testing.T) {
	assert.Equal(t, 1000, resolveLimit(0))
}

func TestResolveLimit_Negative(t *testing.T) {
	assert.Equal(t, 1000, resolveLimit(-5))
}

func TestResolveLimit_Normal(t *testing.T) {
	assert.Equal(t, 50, resolveLimit(50))
}

func TestResolveLimit_AtMax(t *testing.T) {
	assert.Equal(t, 10000, resolveLimit(10000))
}

func TestResolveLimit_AboveMax(t *testing.T) {
	assert.Equal(t, 10000, resolveLimit(50000))
}

func TestResolveLimit_One(t *testing.T) {
	assert.Equal(t, 1, resolveLimit(1))
}

// ──── getRole ───────────────────────────────────────────────────────────

func TestGetRole_DefaultAnalyst(t *testing.T) {
	ctx := context.Background()
	role := getRole(ctx)
	assert.Equal(t, "analyst", role)
}

func TestGetRole_FromContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), "role", "admin")
	role := getRole(ctx)
	assert.Equal(t, "admin", role)
}

func TestGetRole_EmptyRole(t *testing.T) {
	ctx := context.WithValue(context.Background(), "role", "")
	role := getRole(ctx)
	assert.Equal(t, "analyst", role)
}

// ──── withRole ──────────────────────────────────────────────────────────

func TestWithRole_Propagates(t *testing.T) {
	ctx := context.WithValue(context.Background(), "role", "viewer")
	newCtx := withRole(ctx)
	role := getRole(newCtx)
	assert.Equal(t, "viewer", role)
}

func TestWithRole_DefaultWhenEmpty(t *testing.T) {
	ctx := context.Background()
	newCtx := withRole(ctx)
	role := getRole(newCtx)
	assert.Equal(t, "analyst", role)
}

// ──── NewObjectQueryService ─────────────────────────────────────────────

func TestNewObjectQueryService_NilPool(t *testing.T) {
	svc := NewObjectQueryService(nil, nil)
	assert.NotNil(t, svc)
	assert.Nil(t, svc.repo)
	assert.Nil(t, svc.pool)
}

// ──── SellerFilters ─────────────────────────────────────────────────────

func TestSellerFilters_Fields(t *testing.T) {
	f := SellerFilters{
		State:  "SP",
		MinGMV: 1000,
		Limit:  20,
		Offset: 0,
	}
	assert.Equal(t, "SP", f.State)
	assert.Equal(t, 1000.0, f.MinGMV)
	assert.Equal(t, 20, f.Limit)
}

// ──── ObjectContext ─────────────────────────────────────────────────────

func TestObjectContext_Fields(t *testing.T) {
	ctx := ObjectContext{
		ObjectType: "order",
		ObjectID:   "ord-123",
		Properties: map[string]interface{}{"total": 100},
	}
	assert.Equal(t, "order", ctx.ObjectType)
	assert.Equal(t, "ord-123", ctx.ObjectID)
	assert.Equal(t, 100, ctx.Properties["total"])
}
