// Package configloader loads YAML governance configs, computes SHA256 hashes,
// and syncs them to gov.* database tables.
package configloader

import (
	"crypto/sha256"
	"fmt"
)

// computeHash returns the SHA256 hex digest of the given content.
func computeHash(content []byte) string {
	h := sha256.Sum256(content)
	return fmt.Sprintf("%x", h)
}
