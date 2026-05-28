package governance

import (
	"context"
	"fmt"
	"sort"

	"baxi/internal/ontology"
)

// MarkingAdapter implements MarkingService by wrapping ClassificationService
// and ObjectRegistry. Falls back to ontology schema sensitivity when no
// classification entry exists for a field.
type MarkingAdapter struct {
	classification ClassificationLookup
	registry       RegistryLookup
}

func NewMarkingAdapter(classification ClassificationLookup, registry *ontology.ObjectRegistry) *MarkingAdapter {
	return &MarkingAdapter{
		classification: classification,
		registry:       registry,
	}
}

// NewMarkingAdapterWithInterfaces accepts narrow interfaces for unit testing.
func NewMarkingAdapterWithInterfaces(classification ClassificationLookup, registry RegistryLookup) *MarkingAdapter {
	return &MarkingAdapter{
		classification: classification,
		registry:       registry,
	}
}

func (a *MarkingAdapter) GetFieldMarking(ctx context.Context, objectType, field string) (*FieldMarking, error) {
	level, isPII, llmAllowed, err := a.classification.GetFieldMarking(ctx, objectType, field)
	if err != nil {
		return nil, fmt.Errorf("marking: get classification for %s.%s: %w", objectType, field, err)
	}

	sensitivity := a.getOntologySensitivity(objectType, field)

	if level == ResolveLevel("internal") && !isPII && llmAllowed && sensitivity != "" {
		if ontologyLevel := sensitivityToLevel(sensitivity); ontologyLevel > levelToPriority(level) {
			level = priorityToLevel(ontologyLevel)
			isPII = ontologyLevel >= 3
			llmAllowed = ontologyLevel < 3
		}
	}

	return &FieldMarking{
		ObjectType:     objectType,
		Field:          field,
		Classification: level,
		PII:            isPII,
		LLMAllowed:     llmAllowed,
		Sensitivity:    sensitivity,
	}, nil
}

func (a *MarkingAdapter) GetObjectMarkings(ctx context.Context, objectType string) ([]FieldMarking, error) {
	props, err := a.registry.GetProperties(objectType)
	if err != nil {
		return nil, fmt.Errorf("marking: get properties for %s: %w", objectType, err)
	}

	markings := make([]FieldMarking, 0, len(props))
	for fieldName := range props {
		marking, err := a.GetFieldMarking(ctx, objectType, fieldName)
		if err != nil {
			return nil, fmt.Errorf("marking: get field marking for %s.%s: %w", objectType, fieldName, err)
		}
		markings = append(markings, *marking)
	}

	sort.Slice(markings, func(i, j int) bool {
		return markings[i].Field < markings[j].Field
	})

	return markings, nil
}

func (a *MarkingAdapter) IsLLMAllowed(ctx context.Context, objectType, field string) (bool, error) {
	level, _, _, err := a.classification.GetFieldMarking(ctx, objectType, field)
	if err != nil {
		return false, fmt.Errorf("marking: check LLM allowance for %s.%s: %w", objectType, field, err)
	}

	if level == "L3" {
		return false, nil
	}

	return a.registry.IsLLMReadable(objectType, field), nil
}

func (a *MarkingAdapter) ClassifyField(ctx context.Context, objectType, field string) (string, error) {
	level, _, _, err := a.classification.GetFieldMarking(ctx, objectType, field)
	if err != nil {
		return "", fmt.Errorf("marking: classify field %s.%s: %w", objectType, field, err)
	}
	return level, nil
}

func (a *MarkingAdapter) getOntologySensitivity(objectType, field string) string {
	props, err := a.registry.GetProperties(objectType)
	if err != nil {
		return ""
	}
	prop, ok := props[field]
	if !ok {
		return ""
	}
	return prop.Sensitivity
}

func sensitivityToLevel(s string) int {
	switch s {
	case "L0":
		return 0
	case "L1":
		return 1
	case "L2":
		return 2
	case "L3":
		return 3
	case "L4":
		return 4
	default:
		return 0
	}
}

func levelToPriority(level string) int {
	switch level {
	case "L1":
		return 1
	case "L2":
		return 2
	case "L3":
		return 3
	default:
		return 2
	}
}

func priorityToLevel(p int) string {
	switch {
	case p >= 3:
		return "L3"
	case p >= 2:
		return "L2"
	default:
		return "L1"
	}
}
