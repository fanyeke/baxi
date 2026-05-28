package ontology

import (
	"context"
	"fmt"

	"baxi/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ObjectQueryService provides typed, per-object-type query methods for business
// objects backed by the OntologyRepo. It enforces default LIMIT bounds and
// role-based access control.
type ObjectQueryService struct {
	repo *repository.OntologyRepo
	pool *pgxpool.Pool
}

// SellerFilters holds optional filters for seller searches.
type SellerFilters struct {
	State  string
	MinGMV float64
	Limit  int
	Offset int
}

// CategoryFilters holds pagination parameters for category searches.
type CategoryFilters struct {
	Limit  int
	Offset int
}

// RegionFilters holds pagination parameters for region searches.
type RegionFilters struct {
	Limit  int
	Offset int
}

// ObjectContext is a lightweight representation of an object suitable for
// LLM context building.
type ObjectContext struct {
	ObjectType string                 `json:"object_type"`
	ObjectID   string                 `json:"object_id"`
	Properties map[string]interface{} `json:"properties"`
}

// NewObjectQueryService creates a new ObjectQueryService.
func NewObjectQueryService(repo *repository.OntologyRepo, pool *pgxpool.Pool) *ObjectQueryService {
	return &ObjectQueryService{
		repo: repo,
		pool: pool,
	}
}

func getRole(ctx context.Context) string {
	if role, ok := ctx.Value("role").(string); ok && role != "" {
		return role
	}
	return "analyst"
}

func withRole(ctx context.Context) context.Context {
	return repository.WithRole(ctx, getRole(ctx))
}

func resolveLimit(requested int) int {
	if requested <= 0 {
		return 1000
	}
	if requested > 10000 {
		return 10000
	}
	return requested
}

// GetOrder retrieves a single order by its ID.
func (s *ObjectQueryService) GetOrder(ctx context.Context, orderID string) (*repository.ObjectInstance, error) {
	return s.repo.GetObjectByID(withRole(ctx), s.pool, TypeOrder, orderID)
}

// GetSeller retrieves a single seller by its ID.
func (s *ObjectQueryService) GetSeller(ctx context.Context, sellerID string) (*repository.ObjectInstance, error) {
	return s.repo.GetObjectByID(withRole(ctx), s.pool, TypeSeller, sellerID)
}

// GetMetricAlert retrieves a single metric alert by its ID.
func (s *ObjectQueryService) GetMetricAlert(ctx context.Context, alertID string) (*repository.ObjectInstance, error) {
	return s.repo.GetObjectByID(withRole(ctx), s.pool, TypeMetricAlert, alertID)
}

// SearchSellers searches for sellers matching the given filters.
func (s *ObjectQueryService) SearchSellers(ctx context.Context, filters SellerFilters) (*repository.SearchResult, error) {
	query := ""
	if filters.State != "" {
		query = filters.State
	}

	return s.repo.SearchObjects(withRole(ctx), s.pool, TypeSeller, repository.SearchFilters{
		ObjectType: TypeSeller,
		Query:      query,
		Limit:      resolveLimit(filters.Limit),
		Offset:     filters.Offset,
	})
}

// SearchCategories searches for categories.
func (s *ObjectQueryService) SearchCategories(ctx context.Context, filters CategoryFilters) (*repository.SearchResult, error) {
	return s.repo.SearchObjects(withRole(ctx), s.pool, TypeCategory, repository.SearchFilters{
		ObjectType: TypeCategory,
		Limit:      resolveLimit(filters.Limit),
		Offset:     filters.Offset,
	})
}

// SearchRegions searches for regions.
func (s *ObjectQueryService) SearchRegions(ctx context.Context, filters RegionFilters) (*repository.SearchResult, error) {
	return s.repo.SearchObjects(withRole(ctx), s.pool, TypeRegion, repository.SearchFilters{
		ObjectType: TypeRegion,
		Limit:      resolveLimit(filters.Limit),
		Offset:     filters.Offset,
	})
}

// GetSellerMetrics retrieves metrics for a specific seller.
func (s *ObjectQueryService) GetSellerMetrics(ctx context.Context, sellerID string) (*repository.ObjectMetrics, error) {
	return s.repo.GetObjectMetrics(withRole(ctx), s.pool, TypeSeller, sellerID)
}

// GetCategoryMetrics retrieves metrics for a specific category.
func (s *ObjectQueryService) GetCategoryMetrics(ctx context.Context, categoryName string) (*repository.ObjectMetrics, error) {
	return s.repo.GetObjectMetrics(withRole(ctx), s.pool, TypeCategory, categoryName)
}

// BuildObjectContext fetches an object by type and ID and returns a lightweight
// ObjectContext suitable for LLM context building.
func (s *ObjectQueryService) BuildObjectContext(ctx context.Context, objectType, objectID string) (*ObjectContext, error) {
	instance, err := s.repo.GetObjectByID(withRole(ctx), s.pool, objectType, objectID)
	if err != nil {
		return nil, fmt.Errorf("build object context: %w", err)
	}

	return &ObjectContext{
		ObjectType: instance.ObjectType,
		ObjectID:   instance.ID,
		Properties: instance.Properties,
	}, nil
}
