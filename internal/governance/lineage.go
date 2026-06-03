package governance

import (
	"context"
	"fmt"

	governanceRepo "baxi/internal/repository/governance"
)

// LineageResult holds upstream and downstream tables for a resource.
type LineageResult struct {
	Resource   string   `json:"resource"`
	Upstream   []string `json:"upstream"`
	Downstream []string `json:"downstream"`
}

type lineageSnapshotRepo interface {
	GetLineageBySource(ctx context.Context, sourceTable string) ([]governanceRepo.DataLineageRow, error)
	GetLineageByTarget(ctx context.Context, targetTable string) ([]governanceRepo.DataLineageRow, error)
	GetDataLineage(ctx context.Context) ([]governanceRepo.DataLineageRow, error)
}

type lineageProviderAdapter struct {
	provider LineageProvider
}

func (a *lineageProviderAdapter) GetLineageBySource(ctx context.Context, sourceTable string) ([]governanceRepo.DataLineageRow, error) {
	return a.provider.GetLineageBySource(ctx, sourceTable)
}

func (a *lineageProviderAdapter) GetLineageByTarget(ctx context.Context, targetTable string) ([]governanceRepo.DataLineageRow, error) {
	return a.provider.GetLineageByTarget(ctx, targetTable)
}

func (a *lineageProviderAdapter) GetDataLineage(ctx context.Context) ([]governanceRepo.DataLineageRow, error) {
	return a.provider.GetDataLineage(ctx)
}

// LineageService provides data lineage lookup.
// Supports flat queries only — no graph traversal.
type LineageService struct {
	repo lineageSnapshotRepo
}

// NewLineageService creates a new LineageService.
func NewLineageService(repo lineageSnapshotRepo) *LineageService {
	return &LineageService{repo: repo}
}

// NewLineageServiceWithProvider creates a LineageService with a provider for testing.
func NewLineageServiceWithProvider(provider LineageProvider) *LineageService {
	return &LineageService{repo: &lineageProviderAdapter{provider: provider}}
}

// GetLineage returns both upstream and downstream lineage for a resource.
// Upstream: tables that feed into this resource (where target_table = resource).
// Downstream: tables this resource feeds into (where source_table = resource).
func (s *LineageService) GetLineage(ctx context.Context, resource string) (*LineageResult, error) {
	upstream, err := s.GetUpstream(ctx, resource)
	if err != nil {
		return nil, err
	}
	downstream, err := s.GetDownstream(ctx, resource)
	if err != nil {
		return nil, err
	}

	return &LineageResult{
		Resource:   resource,
		Upstream:   upstream,
		Downstream: downstream,
	}, nil
}

// GetUpstream returns tables that are upstream of the given resource.
// These are source tables whose data flows into the resource.
func (s *LineageService) GetUpstream(ctx context.Context, resource string) ([]string, error) {
	rows, err := s.repo.GetLineageByTarget(ctx, resource)
	if err != nil {
		return nil, fmt.Errorf("get upstream for %s: %w", resource, err)
	}

	seen := make(map[string]struct{})
	var result []string
	for _, row := range rows {
		if _, ok := seen[row.SourceTable]; !ok {
			seen[row.SourceTable] = struct{}{}
			result = append(result, row.SourceTable)
		}
	}
	if result == nil {
		result = []string{}
	}
	return result, nil
}

// GetDownstream returns tables that are downstream of the given resource.
// These are target tables that receive data from the resource.
func (s *LineageService) GetDownstream(ctx context.Context, resource string) ([]string, error) {
	rows, err := s.repo.GetLineageBySource(ctx, resource)
	if err != nil {
		return nil, fmt.Errorf("get downstream for %s: %w", resource, err)
	}

	seen := make(map[string]struct{})
	var result []string
	for _, row := range rows {
		if _, ok := seen[row.TargetTable]; !ok {
			seen[row.TargetTable] = struct{}{}
			result = append(result, row.TargetTable)
		}
	}
	if result == nil {
		result = []string{}
	}
	return result, nil
}

// GetAll returns all lineage rows from the database.
func (s *LineageService) GetAll(ctx context.Context) ([]governanceRepo.DataLineageRow, error) {
	return s.repo.GetDataLineage(ctx)
}
