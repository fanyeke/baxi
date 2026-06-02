package pipeline

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuditRecorder_NewUUID_Uniqueness(t *testing.T) {
	ids := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		id := newUUID()
		assert.False(t, ids[id], "duplicate UUID: %s", id)
		ids[id] = true
	}
}

func TestAuditRecorder_NewUUID_Length(t *testing.T) {
	id := newUUID()
	assert.Len(t, id, 36, "UUID should be 36 chars (including hyphens)")
}

func TestAuditRecorder_NewUUID_FormatSegments(t *testing.T) {
	id := newUUID()
	// Format: 8-4-4-4-12
	segments := []int{8, 4, 4, 4, 12}
	pos := 0
	for i, segLen := range segments {
		if i > 0 {
			assert.Equal(t, byte('-'), id[pos], "expected hyphen at position %d", pos)
			pos++
		}
		for j := 0; j < segLen; j++ {
			c := id[pos+j]
			validHex := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')
			assert.True(t, validHex, "invalid hex char %c at position %d", c, pos+j)
		}
		pos += segLen
	}
}

func TestAuditRecorder_NilPool(t *testing.T) {
	// Verify AuditRecorder can be created with nil pool (for testing)
	recorder := &AuditRecorder{Pool: nil, Logger: nil}
	assert.Nil(t, recorder.Pool)
	assert.Nil(t, recorder.Logger)
}
