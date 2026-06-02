package governance

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"baxi/internal/ontology"
)

// ──── sensitivityToLevel ─────────────────────────────────────────────────

func TestSensitivityToLevel_KnownLevels(t *testing.T) {
	assert.Equal(t, 0, sensitivityToLevel("L0"))
	assert.Equal(t, 1, sensitivityToLevel("L1"))
	assert.Equal(t, 2, sensitivityToLevel("L2"))
	assert.Equal(t, 3, sensitivityToLevel("L3"))
	assert.Equal(t, 4, sensitivityToLevel("L4"))
}

func TestSensitivityToLevel_UnknownLevels(t *testing.T) {
	assert.Equal(t, 0, sensitivityToLevel(""))
	assert.Equal(t, 0, sensitivityToLevel("unknown"))
	assert.Equal(t, 0, sensitivityToLevel("L5"))
	assert.Equal(t, 0, sensitivityToLevel("l3"))
}

func TestSensitivityToLevel_CaseSensitive(t *testing.T) {
	assert.Equal(t, 0, sensitivityToLevel("l0"))
	assert.Equal(t, 0, sensitivityToLevel("l1"))
}

// ──── levelToPriority ────────────────────────────────────────────────────

func TestLevelToPriority_KnownLevels(t *testing.T) {
	assert.Equal(t, 1, levelToPriority("L1"))
	assert.Equal(t, 2, levelToPriority("L2"))
	assert.Equal(t, 3, levelToPriority("L3"))
}

func TestLevelToPriority_UnknownLevelDefaults(t *testing.T) {
	assert.Equal(t, 2, levelToPriority(""))
	assert.Equal(t, 2, levelToPriority("unknown"))
	assert.Equal(t, 2, levelToPriority("L4"))
}

func TestLevelToPriority_CaseSensitive(t *testing.T) {
	assert.Equal(t, 2, levelToPriority("l1"))
	assert.Equal(t, 2, levelToPriority("l2"))
	assert.Equal(t, 2, levelToPriority("l3"))
}

// ──── priorityToLevel ────────────────────────────────────────────────────

func TestPriorityToLevel_KnownPriorities(t *testing.T) {
	assert.Equal(t, "L3", priorityToLevel(3))
	assert.Equal(t, "L3", priorityToLevel(4))
	assert.Equal(t, "L3", priorityToLevel(5))
	assert.Equal(t, "L2", priorityToLevel(2))
	assert.Equal(t, "L1", priorityToLevel(1))
	assert.Equal(t, "L1", priorityToLevel(0))
}

func TestPriorityToLevel_NegativePriority(t *testing.T) {
	assert.Equal(t, "L1", priorityToLevel(-1))
}

// ──── getOntologySensitivity (requires mock) ──────────────────────────────

func TestGetOntologySensitivity_FieldFound(t *testing.T) {
	registry := &mockRegistry{
		properties: map[string]map[string]ontology.ObjectProperty{
			"customer": {
				"email": {Name: "email", Sensitivity: "L3", LLMReadable: false},
			},
		},
	}
	// We can't call getOntologySensitivity directly since it's on MarkingAdapter
	// But we can test through GetFieldMarking
	classification := &mockClassification{
		results: map[string]classificationResult{},
	}

	adapter := newTestAdapter(classification, registry)
	marking, err := adapter.GetFieldMarking(nil, "customer", "email")
	assert.NoError(t, err)
	assert.Equal(t, "L3", marking.Sensitivity)
}

func TestGetOntologySensitivity_FieldNotFound(t *testing.T) {
	registry := &mockRegistry{
		properties: map[string]map[string]ontology.ObjectProperty{
			"customer": {
				"name": {Name: "name", Sensitivity: "L1", LLMReadable: true},
			},
		},
	}

	classification := &mockClassification{
		results: map[string]classificationResult{},
	}

	adapter := newTestAdapter(classification, registry)
	marking, err := adapter.GetFieldMarking(nil, "customer", "missing_field")
	assert.NoError(t, err)
	assert.Equal(t, "", marking.Sensitivity)
}

func TestGetOntologySensitivity_ObjectTypeNotFound(t *testing.T) {
	registry := &mockRegistry{
		properties: map[string]map[string]ontology.ObjectProperty{},
	}

	classification := &mockClassification{
		results: map[string]classificationResult{},
	}

	adapter := newTestAdapter(classification, registry)
	_, err := adapter.GetFieldMarking(nil, "nonexistent", "field")
	assert.NoError(t, err)
}
