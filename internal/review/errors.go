package review

import "errors"

// Sentinel errors used across the review package.
var (
	// ErrSandboxNotFound is returned when a sandbox is not found.
	ErrSandboxNotFound = errors.New("sandbox not found")
)
