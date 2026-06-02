package audit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ──── NewIntegration ────────────────────────────────────────────────────

func TestNewIntegration(t *testing.T) {
	integration := NewIntegration()
	assert.NotNil(t, integration)
}

// ──── defaultActor ──────────────────────────────────────────────────────

func TestDefaultActor_NonEmpty(t *testing.T) {
	assert.Equal(t, "alice", defaultActor("alice"))
	assert.Equal(t, "bob", defaultActor("bob"))
	assert.Equal(t, "system", defaultActor("system"))
}

func TestDefaultActor_Empty(t *testing.T) {
	assert.Equal(t, "system", defaultActor(""))
}
