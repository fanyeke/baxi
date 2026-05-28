package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ObjectQuerier abstracts the OntologyRepo methods used by OntologyAwareAdapter.
// Defined here so the adapter in ontology/ can depend on this interface
// without importing the concrete OntologyRepo.
type ObjectQuerier interface {
	GetObjectByID(ctx context.Context, pool *pgxpool.Pool, objectType, objectID string) (*ObjectInstance, error)
	QueryByObjectType(ctx context.Context, pool *pgxpool.Pool, objectType string, filters ObjectFilters) (*ObjectQueryResult, error)
}
