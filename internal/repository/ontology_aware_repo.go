package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"baxi/internal/repository/common"
	ontologyRepo "baxi/internal/repository/ontology"
	outboxSub "baxi/internal/repository/outbox"
)

// Type aliases for types originally defined in interfaces.go (deleted).
// These types now live in the ontology subpackage.
type ObjectInstance = ontologyRepo.ObjectInstance
type ObjectFilters = ontologyRepo.ObjectFilters
type ObjectQueryResult = ontologyRepo.ObjectQueryResult
type ObjectMetrics = ontologyRepo.ObjectMetrics
type SearchFilters = ontologyRepo.SearchFilters
type SearchResult = ontologyRepo.SearchResult

// WithRole returns a context with the given role for RBAC enforcement.
// Deprecated: Use ontology.WithRole instead.
func WithRole(ctx context.Context, role string) context.Context {
	return ontologyRepo.WithRole(ctx, role)
}

// ObjectSchemaRepository provides access to gov.object_schema table.
// Deprecated: This interface is retained for backward compatibility.
type ObjectSchemaRepository interface {
	GetAll(ctx context.Context, pool *pgxpool.Pool) ([]ObjectSchemaRow, error)
	GetByObjectType(ctx context.Context, pool *pgxpool.Pool, objectType string) (*ObjectSchemaRow, error)
	Upsert(ctx context.Context, pool *pgxpool.Pool, params ObjectSchemaUpsertParams) error
}

// ObjectSchemaRow represents a row from gov.object_schema.
type ObjectSchemaRow struct {
	ObjectType  string
	ObjectName  string
	SchemaJSONB []byte
	Version     string
}

// ObjectSchemaUpsertParams holds parameters for upserting an object schema.
type ObjectSchemaUpsertParams struct {
	ObjectType  string
	ObjectName  string
	SchemaJSONB []byte
	Version     string
}

type ObjectSchema struct {
	Name        string
	DisplayName string
	Grain       string
	PrimaryKey  string
	Properties  map[string]PropertySchema
}

type PropertySchema struct {
	Name string
	Type string
	IsPK bool
}

type OntologyAwareRepo interface {
	GetObjectByID(ctx context.Context, pool *pgxpool.Pool, objectType, objectID string) (*ObjectInstance, error)
	QueryByObjectType(ctx context.Context, pool *pgxpool.Pool, objectType string, filters ObjectFilters) (*ObjectQueryResult, error)
	GetObjectTypeSchema(ctx context.Context, objectType string) (*ObjectSchema, error)
}

// OntologyRepo provides object queries against dwd/mart/ops tables.
// Deprecated: Use ontology.Repository instead. Retained for backward compatibility
// with callers that have not yet migrated to the new signature.
// TODO: Remove when all callers are updated to use ontology.Repository directly.
type OntologyRepo struct {
	inner *ontologyRepo.Repository
}

// NewOntologyRepo creates a new OntologyRepo (DEPRECATED).
func NewOntologyRepo() *OntologyRepo {
	return &OntologyRepo{}
}

func (r *OntologyRepo) ensureInitialized(pool *pgxpool.Pool) *ontologyRepo.Repository {
	if r.inner == nil {
		r.inner = ontologyRepo.NewRepository(common.NewPoolProvider(pool))
	}
	return r.inner
}

// SetV2Compiler sets the v2 query compiler (DEPRECATED).
func (r *OntologyRepo) SetV2Compiler(qc ontologyRepo.V2QueryCompiler) {
	if r.inner != nil {
		r.inner.SetV2Compiler(qc)
	}
}

// GetObjectByID retrieves a single object by its ID (DEPRECATED).
func (r *OntologyRepo) GetObjectByID(ctx context.Context, pool *pgxpool.Pool, objectType, objectID string) (*ObjectInstance, error) {
	return r.ensureInitialized(pool).GetObjectByID(ctx, objectType, objectID)
}

// QueryByObjectType queries objects by type (DEPRECATED).
func (r *OntologyRepo) QueryByObjectType(ctx context.Context, pool *pgxpool.Pool, objectType string, filters ObjectFilters) (*ObjectQueryResult, error) {
	return r.ensureInitialized(pool).QueryByObjectType(ctx, objectType, ontologyRepo.ObjectFilters(filters))
}

// GetObjectMetrics retrieves metrics for a specific object (DEPRECATED).
func (r *OntologyRepo) GetObjectMetrics(ctx context.Context, pool *pgxpool.Pool, objectType, objectID string) (*ObjectMetrics, error) {
	return r.ensureInitialized(pool).GetObjectMetrics(ctx, objectType, objectID)
}

// SearchObjects searches for objects matching the given filters (DEPRECATED).
func (r *OntologyRepo) SearchObjects(ctx context.Context, pool *pgxpool.Pool, objectType string, filters SearchFilters) (*SearchResult, error) {
	return r.ensureInitialized(pool).SearchObjects(ctx, objectType, ontologyRepo.SearchFilters(filters))
}

// ──── Outbox backward compatibility (from deleted outbox_repository.go) ────────

// OutboxRow represents a single row from ops.outbox_event for read queries.
type OutboxRow = outboxSub.OutboxRow

// OutboxFilters holds optional WHERE clause filters for listing outbox events.
type OutboxFilters = outboxSub.OutboxFilters

// OutboxDetail represents a full outbox event row for detail/management queries.
type OutboxDetail = outboxSub.OutboxDetail

// OutboxRepository provides read-only access to ops.outbox_event.
// Deprecated: Use outbox.Repository instead.
type OutboxRepository struct {
	inner *outboxSub.Repository
}

// NewOutboxRepository creates a new OutboxRepository (DEPRECATED).
func NewOutboxRepository() *OutboxRepository {
	return &OutboxRepository{}
}

func (r *OutboxRepository) ensureInit(pool *pgxpool.Pool) {
	if r.inner == nil {
		r.inner = outboxSub.NewRepository(common.NewPoolProvider(pool))
	}
}

// GetDetail retrieves a full outbox event row by event_id (DEPRECATED).
func (r *OutboxRepository) GetDetail(ctx context.Context, pool *pgxpool.Pool, eventID string) (*OutboxDetail, error) {
	r.ensureInit(pool)
	return r.inner.GetDetail(ctx, eventID)
}

// ListOutboxEvents queries ops.outbox_event with optional filters and pagination (DEPRECATED).
func (r *OutboxRepository) ListOutboxEvents(
	ctx context.Context,
	pool *pgxpool.Pool,
	filters OutboxFilters,
	limit, offset int,
) ([]OutboxRow, int, error) {
	r.ensureInit(pool)
	return r.inner.ListOutboxEvents(ctx, filters, limit, offset)
}
