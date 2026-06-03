package repository

import (
	"context"
)

// ObjectQuerier abstracts the OntologyRepo methods used by OntologyAwareAdapter.
// Defined here so the adapter in ontology/ can depend on this interface
// without importing the concrete OntologyRepo.
type ObjectQuerier interface {
	GetObjectByID(ctx context.Context, objectType, objectID string) (*ObjectInstance, error)
	QueryByObjectType(ctx context.Context, objectType string, filters ObjectFilters) (*ObjectQueryResult, error)
}
