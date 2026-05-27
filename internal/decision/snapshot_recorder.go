package decision

import "context"

// SnapshotRecorder persists decision-related data snapshots for audit and replay.
// Implementations should be best-effort: failures are logged but do not block
// the main decision flow.
type SnapshotRecorder interface {
	RecordSnapshot(ctx context.Context, record DataSnapshotRecord) error
	RecordEvent(ctx context.Context, record LineageEventRecord) error
}

// noopSnapshotRecorder is a no-op implementation used when lineage tracking is disabled.
type noopSnapshotRecorder struct{}

func (n *noopSnapshotRecorder) RecordSnapshot(ctx context.Context, record DataSnapshotRecord) error {
	return nil
}

func (n *noopSnapshotRecorder) RecordEvent(ctx context.Context, record LineageEventRecord) error {
	return nil
}

// NewNoopSnapshotRecorder returns a SnapshotRecorder that does nothing.
func NewNoopSnapshotRecorder() SnapshotRecorder {
	return &noopSnapshotRecorder{}
}
