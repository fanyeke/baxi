package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
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
	GetObjectByID(ctx context.Context, pool *pgxpool.Pool, objectType, objectID string) (*ObjectInstance, error)
	QueryByObjectType(ctx context.Context, pool *pgxpool.Pool, objectType string, filters ObjectFilters) (*ObjectQueryResult, error)
	GetObjectTypeSchema(ctx context.Context, objectType string) (*ObjectSchema, error)
}
