package governance

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"baxi/internal/repository"
)

// LineageResult holds upstream and downstream tables for a resource.
type LineageResult struct {
	Resource   string   `json:"resource"`
	Upstream   []string `json:"upstream"`
	Downstream []string `json:"downstream"`
}

// LineageService provides data lineage lookup.
// Supports flat queries only — no graph traversal.
type LineageService struct {
	pool *pgxpool.Pool
	repo *repository.GovernanceRepository
}

// NewLineageService creates a new LineageService.
func NewLineageService(pool *pgxpool.Pool, repo *repository.GovernanceRepository) *LineageService {
	return &LineageService{pool: pool, repo: repo}
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
	rows, err := s.repo.GetLineageByTarget(ctx, s.pool, resource)
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
	rows, err := s.repo.GetLineageBySource(ctx, s.pool, resource)
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
func (s *LineageService) GetAll(ctx context.Context) ([]repository.DataLineageRow, error) {
	return s.repo.GetDataLineage(ctx, s.pool)
}
