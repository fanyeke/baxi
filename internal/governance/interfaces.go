package governance

import (
	"context"

	governanceRepo "baxi/internal/repository/governance"
)

// ConfigSnapshotProvider is a narrow interface for config snapshot lookups.
// Enables unit testing of CheckpointService without a real database.
type ConfigSnapshotProvider interface {
	GetConfigSnapshots(ctx context.Context) ([]governanceRepo.ConfigSnapshotRow, error)
}

// LineageProvider is a narrow interface for data lineage lookups.
// Enables unit testing of LineageService without a real database.
type LineageProvider interface {
	GetLineageBySource(ctx context.Context, sourceTable string) ([]governanceRepo.DataLineageRow, error)
	GetLineageByTarget(ctx context.Context, targetTable string) ([]governanceRepo.DataLineageRow, error)
	GetDataLineage(ctx context.Context) ([]governanceRepo.DataLineageRow, error)
}
