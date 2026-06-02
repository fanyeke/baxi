package governance

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	governanceRepo "baxi/internal/repository/governance"
)

// mockLineageProvider implements LineageProvider for testing.
type mockLineageProvider struct {
	sourceRows []governanceRepo.DataLineageRow
	targetRows []governanceRepo.DataLineageRow
	allRows    []governanceRepo.DataLineageRow
	err        error
}

func (m *mockLineageProvider) GetLineageBySource(ctx context.Context, sourceTable string) ([]governanceRepo.DataLineageRow, error) {
	if m.err != nil {
		return nil, m.err
	}
	out := make([]governanceRepo.DataLineageRow, 0)
	for _, r := range m.sourceRows {
		if r.SourceTable == sourceTable {
			out = append(out, r)
		}
	}
	if out == nil {
		return []governanceRepo.DataLineageRow{}, nil
	}
	return out, nil
}

func (m *mockLineageProvider) GetLineageByTarget(ctx context.Context, targetTable string) ([]governanceRepo.DataLineageRow, error) {
	if m.err != nil {
		return nil, m.err
	}
	out := make([]governanceRepo.DataLineageRow, 0)
	for _, r := range m.targetRows {
		if r.TargetTable == targetTable {
			out = append(out, r)
		}
	}
	if out == nil {
		return []governanceRepo.DataLineageRow{}, nil
	}
	return out, nil
}

func (m *mockLineageProvider) GetDataLineage(ctx context.Context) ([]governanceRepo.DataLineageRow, error) {
	if m.err != nil {
		return nil, m.err
	}
	out := make([]governanceRepo.DataLineageRow, len(m.allRows))
	copy(out, m.allRows)
	return out, nil
}

func TestLineageService_GetLineage_BothDirections(t *testing.T) {
	prov := &mockLineageProvider{
		sourceRows: []governanceRepo.DataLineageRow{
			{SourceTable: "orders", TargetTable: "order_items"},
			{SourceTable: "orders", TargetTable: "order_payments"},
		},
		targetRows: []governanceRepo.DataLineageRow{
			{SourceTable: "customers", TargetTable: "orders"},
			{SourceTable: "products", TargetTable: "orders"},
		},
	}
	svc := NewLineageServiceWithProvider(prov)

	result, err := svc.GetLineage(context.Background(), "orders")
	assert.NoError(t, err)
	assert.Equal(t, "orders", result.Resource)
	assert.ElementsMatch(t, []string{"customers", "products"}, result.Upstream)
	assert.ElementsMatch(t, []string{"order_items", "order_payments"}, result.Downstream)
}

func TestLineageService_GetUpstream(t *testing.T) {
	prov := &mockLineageProvider{
		targetRows: []governanceRepo.DataLineageRow{
			{SourceTable: "customers", TargetTable: "orders"},
			{SourceTable: "products", TargetTable: "orders"},
			{SourceTable: "sellers", TargetTable: "orders"},
		},
	}
	svc := NewLineageServiceWithProvider(prov)

	upstream, err := svc.GetUpstream(context.Background(), "orders")
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"customers", "products", "sellers"}, upstream)
}

func TestLineageService_GetUpstream_Deduplicates(t *testing.T) {
	prov := &mockLineageProvider{
		targetRows: []governanceRepo.DataLineageRow{
			{SourceTable: "customers", TargetTable: "orders"},
			{SourceTable: "customers", TargetTable: "orders"}, // duplicate
		},
	}
	svc := NewLineageServiceWithProvider(prov)

	upstream, err := svc.GetUpstream(context.Background(), "orders")
	assert.NoError(t, err)
	assert.Len(t, upstream, 1)
	assert.Equal(t, "customers", upstream[0])
}

func TestLineageService_GetDownstream(t *testing.T) {
	prov := &mockLineageProvider{
		sourceRows: []governanceRepo.DataLineageRow{
			{SourceTable: "orders", TargetTable: "order_items"},
			{SourceTable: "orders", TargetTable: "order_payments"},
			{SourceTable: "orders", TargetTable: "order_reviews"},
		},
	}
	svc := NewLineageServiceWithProvider(prov)

	downstream, err := svc.GetDownstream(context.Background(), "orders")
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"order_items", "order_payments", "order_reviews"}, downstream)
}

func TestLineageService_GetDownstream_Deduplicates(t *testing.T) {
	prov := &mockLineageProvider{
		sourceRows: []governanceRepo.DataLineageRow{
			{SourceTable: "orders", TargetTable: "order_items"},
			{SourceTable: "orders", TargetTable: "order_items"}, // duplicate
		},
	}
	svc := NewLineageServiceWithProvider(prov)

	downstream, err := svc.GetDownstream(context.Background(), "orders")
	assert.NoError(t, err)
	assert.Len(t, downstream, 1)
	assert.Equal(t, "order_items", downstream[0])
}

func TestLineageService_GetLineage_NoResults(t *testing.T) {
	svc := NewLineageServiceWithProvider(&mockLineageProvider{})

	result, err := svc.GetLineage(context.Background(), "nonexistent")
	assert.NoError(t, err)
	assert.Empty(t, result.Upstream)
	assert.Empty(t, result.Downstream)
	assert.Equal(t, "nonexistent", result.Resource)
}

func TestLineageService_GetUpstream_NoResults(t *testing.T) {
	svc := NewLineageServiceWithProvider(&mockLineageProvider{})

	upstream, err := svc.GetUpstream(context.Background(), "nonexistent")
	assert.NoError(t, err)
	assert.Empty(t, upstream)
}

func TestLineageService_GetDownstream_NoResults(t *testing.T) {
	svc := NewLineageServiceWithProvider(&mockLineageProvider{})

	downstream, err := svc.GetDownstream(context.Background(), "nonexistent")
	assert.NoError(t, err)
	assert.Empty(t, downstream)
}

func TestLineageService_GetAll(t *testing.T) {
	prov := &mockLineageProvider{
		allRows: []governanceRepo.DataLineageRow{
			{SourceTable: "a", TargetTable: "b"},
			{SourceTable: "b", TargetTable: "c"},
		},
	}
	svc := NewLineageServiceWithProvider(prov)

	all, err := svc.GetAll(context.Background())
	assert.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestLineageService_GetAll_NoResults(t *testing.T) {
	svc := NewLineageServiceWithProvider(&mockLineageProvider{})

	all, err := svc.GetAll(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, all)
}

func TestLineageService_GetLineage_UpstreamError(t *testing.T) {
	prov := &mockLineageProvider{
		err: assert.AnError,
	}
	svc := NewLineageServiceWithProvider(prov)

	_, err := svc.GetLineage(context.Background(), "orders")
	assert.Error(t, err)
}

func TestLineageService_GetUpstream_ProviderError(t *testing.T) {
	prov := &mockLineageProvider{
		err: assert.AnError,
	}
	svc := NewLineageServiceWithProvider(prov)

	_, err := svc.GetUpstream(context.Background(), "orders")
	assert.Error(t, err)
}

func TestLineageService_GetDownstream_ProviderError(t *testing.T) {
	prov := &mockLineageProvider{
		err: assert.AnError,
	}
	svc := NewLineageServiceWithProvider(prov)

	_, err := svc.GetDownstream(context.Background(), "orders")
	assert.Error(t, err)
}

func TestLineageService_GetAll_ProviderError(t *testing.T) {
	prov := &mockLineageProvider{
		err: assert.AnError,
	}
	svc := NewLineageServiceWithProvider(prov)

	_, err := svc.GetAll(context.Background())
	assert.Error(t, err)
}
