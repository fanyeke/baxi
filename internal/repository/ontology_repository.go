// DEPRECATED: Use baxi/internal/repository/ontology instead.
// This file is a compatibility layer during migration.

package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"baxi/internal/repository/common"
	ontologyRepo "baxi/internal/repository/ontology"
)

// WithRole returns a context with the given role for RBAC enforcement.
// DEPRECATED: Use ontology.WithRole instead.
func WithRole(ctx context.Context, role string) context.Context {
	return ontologyRepo.WithRole(ctx, role)
}

// OntologyRepo provides object queries against dwd/mart/ops tables.
// DEPRECATED: Use ontology.Repository instead.
type OntologyRepo struct {
	inner      *ontologyRepo.Repository
	v2Compiler ontologyRepo.V2QueryCompiler
}

// NewOntologyRepo creates a new OntologyRepo (DEPRECATED).
func NewOntologyRepo() *OntologyRepo {
	return &OntologyRepo{}
}

// SetV2Compiler sets the v2 query compiler, enabling schema-driven queries.
func (r *OntologyRepo) SetV2Compiler(qc ontologyRepo.V2QueryCompiler) {
	r.v2Compiler = qc
	if r.inner != nil {
		r.inner.SetV2Compiler(qc)
	}
}

func (r *OntologyRepo) ensureInitialized(pool *pgxpool.Pool) *ontologyRepo.Repository {
	if r.inner == nil {
		r.inner = ontologyRepo.NewRepository(common.NewPoolProvider(pool))
		if r.v2Compiler != nil {
			r.inner.SetV2Compiler(r.v2Compiler)
		}
	}
	return r.inner
}

// QueryByObjectType queries objects by type (DEPRECATED).
func (r *OntologyRepo) QueryByObjectType(ctx context.Context, pool *pgxpool.Pool, objectType string, filters ObjectFilters) (*ObjectQueryResult, error) {
	result, err := r.ensureInitialized(pool).QueryByObjectType(ctx, objectType, ontologyRepo.ObjectFilters(filters))
	if err != nil {
		return nil, err
	}
	return toObjectQueryResult(result), nil
}

// GetObjectByID retrieves a single object by its ID (DEPRECATED).
func (r *OntologyRepo) GetObjectByID(ctx context.Context, pool *pgxpool.Pool, objectType, objectID string) (*ObjectInstance, error) {
	obj, err := r.ensureInitialized(pool).GetObjectByID(ctx, objectType, objectID)
	if err != nil {
		return nil, err
	}
	return toObjectInstance(obj), nil
}

// GetObjectMetrics retrieves metrics for a specific object (DEPRECATED).
func (r *OntologyRepo) GetObjectMetrics(ctx context.Context, pool *pgxpool.Pool, objectType, objectID string) (*ObjectMetrics, error) {
	metrics, err := r.ensureInitialized(pool).GetObjectMetrics(ctx, objectType, objectID)
	if err != nil {
		return nil, err
	}
	return toObjectMetrics(metrics), nil
}

// SearchObjects searches for objects matching the given filters (DEPRECATED).
func (r *OntologyRepo) SearchObjects(ctx context.Context, pool *pgxpool.Pool, objectType string, filters SearchFilters) (*SearchResult, error) {
	result, err := r.ensureInitialized(pool).SearchObjects(ctx, objectType, ontologyRepo.SearchFilters(filters))
	if err != nil {
		return nil, err
	}
	return toSearchResult(result), nil
}

// ──── Type conversion helpers ────────────────────────────────────────────────

func toObjectInstance(src *ontologyRepo.ObjectInstance) *ObjectInstance {
	if src == nil {
		return nil
	}
	return &ObjectInstance{
		ObjectType: src.ObjectType,
		ID:         src.ID,
		Properties: src.Properties,
	}
}

func toObjectInstanceSlice(src []ontologyRepo.ObjectInstance) []ObjectInstance {
	if src == nil {
		return nil
	}
	dst := make([]ObjectInstance, len(src))
	for i, s := range src {
		dst[i] = ObjectInstance{
			ObjectType: s.ObjectType,
			ID:         s.ID,
			Properties: s.Properties,
		}
	}
	return dst
}

func toObjectQueryResult(src *ontologyRepo.ObjectQueryResult) *ObjectQueryResult {
	if src == nil {
		return nil
	}
	return &ObjectQueryResult{
		Rows:  toObjectInstanceSlice(src.Rows),
		Total: src.Total,
	}
}

func toObjectMetrics(src *ontologyRepo.ObjectMetrics) *ObjectMetrics {
	if src == nil {
		return nil
	}
	return &ObjectMetrics{
		ObjectType: src.ObjectType,
		ID:         src.ID,
		Metrics:    src.Metrics,
	}
}

func toSearchResult(src *ontologyRepo.SearchResult) *SearchResult {
	if src == nil {
		return nil
	}
	return &SearchResult{
		Rows:  toObjectInstanceSlice(src.Rows),
		Total: src.Total,
	}
}
