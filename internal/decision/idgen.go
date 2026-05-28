package decision

import (
	"crypto/rand"
	"fmt"
	"time"
)

const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// GenerateCaseID generates a unique case ID with prefix "dc".
// Format: dc_<timestamp>_<6char_hash>
func GenerateCaseID() string {
	return generateID("dc")
}

// GenerateProposalID generates a unique proposal ID with prefix "ap".
// Format: ap_<timestamp>_<6char_hash>
func GenerateProposalID() string {
	return generateID("ap")
}

// GenerateDecisionID generates a unique decision ID with prefix "de".
// Format: de_<timestamp>_<6char_hash>
func GenerateDecisionID() string {
	return generateID("de")
}

// GenerateLineageEventID generates a unique lineage event ID with prefix "le".
// Format: le_<timestamp>_<6char_hash>
func GenerateLineageEventID() string {
	return generateID("le")
}

// GenerateDataSnapshotID generates a unique data snapshot ID with prefix "ds".
// Format: ds_<timestamp>_<6char_hash>
func GenerateDataSnapshotID() string {
	return generateID("ds")
}

// generateID creates an ID string with the given prefix, current Unix timestamp,
// and a 6-character random hash containing only [A-Za-z0-9].
func generateID(prefix string) string {
	ts := time.Now().Unix()
	buf := make([]byte, 6)
	_, _ = rand.Read(buf)

	hash := make([]byte, 6)
	for i := 0; i < 6; i++ {
		hash[i] = alphabet[int(buf[i])%len(alphabet)]
	}

	return fmt.Sprintf("%s_%d_%s", prefix, ts, string(hash))
}
