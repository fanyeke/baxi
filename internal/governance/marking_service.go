package governance

import (
	"context"

	"baxi/internal/ontology"
)

// FieldMarking holds unified marking information for a single field,
// combining classification data (from data_classification.yml / DB)
// with ontology schema metadata (sensitivity, LLM readability).
type FieldMarking struct {
	ObjectType     string `json:"object_type"`
	Field          string `json:"field"`
	Classification string `json:"classification"` // L1, L2, L3
	PII            bool   `json:"pii"`
	LLMAllowed     bool   `json:"llm_allowed"`
	Sensitivity    string `json:"sensitivity"` // from ontology schema (L0–L4)
}

// MarkingService provides unified data classification and field marking.
//
// It combines two data sources:
//   - ClassificationService: field-level classification (pii, sensitive, internal, etc.)
//   - ObjectRegistry: ontology schema with sensitivity levels and LLM readability
//
// When a field has no explicit classification entry, the adapter falls back
// to the ontology schema's Sensitivity value for that property.
type MarkingService interface {
	// GetFieldMarking returns the unified marking for a specific object_type and field.
	// Falls back to ontology schema sensitivity when no classification entry exists.
	GetFieldMarking(ctx context.Context, objectType, field string) (*FieldMarking, error)

	// GetObjectMarkings returns markings for all fields of the given object type.
	// Fields are enriched with classification data where available.
	GetObjectMarkings(ctx context.Context, objectType string) ([]FieldMarking, error)

	// IsLLMAllowed returns true if the field may be included in LLM context.
	// A field is LLM-allowed when its classification level is not L3 (pii/sensitive)
	// AND the ontology schema marks it as LLM-readable.
	IsLLMAllowed(ctx context.Context, objectType, field string) (bool, error)

	// ClassifyField returns the canonical classification level (L1, L2, L3) for a field.
	// Falls back to "L2" (internal) when no classification entry exists.
	ClassifyField(ctx context.Context, objectType, field string) (string, error)
}

// ClassificationLookup is the narrow interface the adapter needs from ClassificationService.
// This enables unit testing without a real database.
type ClassificationLookup interface {
	// GetFieldMarking returns (level, isPII, llmAllowed, error) for a field.
	GetFieldMarking(ctx context.Context, objectType, property string) (string, bool, bool, error)
}

// RegistryLookup is the narrow interface the adapter needs from ObjectRegistry.
// This enables unit testing without a real registry.
type RegistryLookup interface {
	// GetProperties returns the properties map for the given object type.
	GetProperties(objectType string) (map[string]ontology.ObjectProperty, error)
	// IsLLMReadable checks whether the named property is LLM-readable.
	IsLLMReadable(objectType, property string) bool
}
