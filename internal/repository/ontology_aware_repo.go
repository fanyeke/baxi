package repository

import (
	"context"
)

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
	GetObjectByID(ctx context.Context, objectType, objectID string) (*ObjectInstance, error)
	QueryByObjectType(ctx context.Context, objectType string, filters ObjectFilters) (*ObjectQueryResult, error)
	GetObjectTypeSchema(ctx context.Context, objectType string) (*ObjectSchema, error)
}
