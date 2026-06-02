package alert

import (
	"strings"
	"testing"
	"time"
)

// formatTimestamp is not tested in existing test files

func TestFormatTimestamp_Format(t *testing.T) {
	ts := formatTimestamp()

	// Should match ISO-8601 with microseconds
	expectedFormat := "2006-01-02T15:04:05.000000"
	_, err := time.Parse(expectedFormat, ts)
	if err != nil {
		t.Errorf("formatTimestamp() = %q, expected format %q, parse error: %v", ts, expectedFormat, err)
	}
}

func TestFormatTimestamp_ContainsISO8601(t *testing.T) {
	ts := formatTimestamp()

	if !strings.Contains(ts, "T") {
		t.Errorf("formatTimestamp() = %q, expected 'T' separator", ts)
	}

	// Should have microseconds (6 digits after last .)
	parts := strings.Split(ts, ".")
	if len(parts) != 2 {
		t.Errorf("formatTimestamp() = %q, expected format with microseconds (.xxxxxx)", ts)
	} else if len(parts[1]) != 6 {
		t.Errorf("formatTimestamp() = %q, expected 6 fractional digits, got %d", ts, len(parts[1]))
	}
}

func TestFormatTimestamp_RoundTrip(t *testing.T) {
	ts := formatTimestamp()
	parsed, err := time.Parse("2006-01-02T15:04:05.000000", ts)
	if err != nil {
		t.Fatalf("failed to parse formatTimestamp(): %v", err)
	}

	// The parsed time should be within 1 second of now
	diff := time.Since(parsed)
	if diff < 0 {
		diff = -diff
	}
	if diff > 5*time.Second {
		t.Errorf("formatTimestamp() = %q, parsed time is %v away from now", ts, diff)
	}
}
