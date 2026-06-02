package pipeline

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ──── newUUID ──────────────────────────────────────────────────────────────

var uuidV4Pattern = regexp.MustCompile(
	`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`,
)

func TestNewUUID_FormatIsValidV4(t *testing.T) {
	id := newUUID()
	assert.Regexp(t, uuidV4Pattern, id, "UUID %q does not match v4 pattern", id)
}

func TestNewUUID_NonEmpty(t *testing.T) {
	id := newUUID()
	assert.NotEmpty(t, id)
}

func TestNewUUID_Unique(t *testing.T) {
	ids := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		id := newUUID()
		assert.False(t, ids[id], "duplicate UUID generated: %s", id)
		ids[id] = true
	}
}

func TestNewUUID_VersionIs4(t *testing.T) {
	id := newUUID()
	// The 13th character (0-indexed: position 14) is the version
	assert.Equal(t, byte('4'), id[14], "UUID version nibble should be 4")
}

func TestNewUUID_VariantIsRFC4122(t *testing.T) {
	id := newUUID()
	// The 17th character (0-indexed: position 19) is the variant top bits
	// Valid variants: 8, 9, a, b
	c := id[19]
	assert.True(t,
		c == '8' || c == '9' || c == 'a' || c == 'b',
		"UUID variant nibble should be 8/9/a/b, got %c", c,
	)
}

func TestNewUUID_ContainsOnlyHexAndDashes(t *testing.T) {
	id := newUUID()
	assert.Regexp(t, uuidV4Pattern, id)
}
