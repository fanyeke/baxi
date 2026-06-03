package ontology

import (
	"context"
	"fmt"

	"baxi/internal/repository"
)

type objectTypeProvider interface {
	GetObjectType(name string) (*ObjectType, error)
}

type OntologyAwareAdapter struct {
	querier  repository.ObjectQuerier
	registry objectTypeProvider
}

func NewOntologyAwareAdapter(querier repository.ObjectQuerier, registry objectTypeProvider) *OntologyAwareAdapter {
	return &OntologyAwareAdapter{querier: querier, registry: registry}
}

func (a *OntologyAwareAdapter) GetObjectByID(ctx context.Context, objectType, objectID string) (*repository.ObjectInstance, error) {
	allowed, err := a.allowedProperties(objectType)
	if err != nil {
		return nil, err
	}

	obj, err := a.querier.GetObjectByID(ctx, objectType, objectID)
	if err != nil {
		return nil, err
	}

	obj.Properties = filterProperties(obj.Properties, allowed)
	return obj, nil
}

func (a *OntologyAwareAdapter) QueryByObjectType(ctx context.Context, objectType string, filters repository.ObjectFilters) (*repository.ObjectQueryResult, error) {
	allowed, err := a.allowedProperties(objectType)
	if err != nil {
		return nil, err
	}

	result, err := a.querier.QueryByObjectType(ctx, objectType, filters)
	if err != nil {
		return nil, err
	}

	for i := range result.Rows {
		result.Rows[i].Properties = filterProperties(result.Rows[i].Properties, allowed)
	}
	return result, nil
}

func (a *OntologyAwareAdapter) GetObjectTypeSchema(_ context.Context, objectType string) (*repository.ObjectSchema, error) {
	ot, err := a.registry.GetObjectType(objectType)
	if err != nil {
		return nil, fmt.Errorf("ontology schema: %w", err)
	}

	props := make(map[string]repository.PropertySchema, len(ot.Properties))
	for name, p := range ot.Properties {
		props[name] = repository.PropertySchema{Name: p.Name, Type: p.Type, IsPK: p.IsPK}
	}

	return &repository.ObjectSchema{
		Name:        ot.Name,
		DisplayName: ot.DisplayName,
		Grain:       ot.Grain,
		PrimaryKey:  ot.PrimaryKey,
		Properties:  props,
	}, nil
}

func (a *OntologyAwareAdapter) allowedProperties(objectType string) (map[string]bool, error) {
	ot, err := a.registry.GetObjectType(objectType)
	if err != nil {
		return nil, fmt.Errorf("ontology schema lookup: %w", err)
	}

	allowed := make(map[string]bool, len(ot.Properties))
	for name := range ot.Properties {
		allowed[name] = true
	}
	return allowed, nil
}

func filterProperties(props map[string]any, allowed map[string]bool) map[string]any {
	filtered := make(map[string]any, len(allowed))
	for key, val := range props {
		if allowed[key] {
			filtered[key] = val
		}
	}
	return filtered
}
