package governance

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ──── ResolveLevel ─────────────────────────────────────────────────────────

func TestResolveLevel_PII(t *testing.T) {
	assert.Equal(t, "L3", ResolveLevel("pii"))
}

func TestResolveLevel_Sensitive(t *testing.T) {
	assert.Equal(t, "L3", ResolveLevel("sensitive"))
}

func TestResolveLevel_Internal(t *testing.T) {
	assert.Equal(t, "L2", ResolveLevel("internal"))
}

func TestResolveLevel_DerivedSensitive(t *testing.T) {
	assert.Equal(t, "L2", ResolveLevel("derived_sensitive"))
}

func TestResolveLevel_PublicInternal(t *testing.T) {
	assert.Equal(t, "L1", ResolveLevel("public_internal"))
}

func TestResolveLevel_Unknown(t *testing.T) {
	assert.Equal(t, "L2", ResolveLevel("unknown_level"))
}

func TestResolveLevel_Empty(t *testing.T) {
	assert.Equal(t, "L2", ResolveLevel(""))
}

// ──── ClassificationService New ────────────────────────────────────────────

func TestNewClassificationService(t *testing.T) {
	svc := NewClassificationService(nil)
	assert.NotNil(t, svc)
}
