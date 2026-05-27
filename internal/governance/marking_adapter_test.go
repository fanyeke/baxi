package governance

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/ontology"
)

type mockClassification struct {
	results map[string]classificationResult
}

type classificationResult struct {
	level     string
	isPII     bool
	llmAllowed bool
	err       error
}

func (m *mockClassification) GetFieldMarking(_ context.Context, objectType, property string) (string, bool, bool, error) {
	key := objectType + "." + property
	if r, ok := m.results[key]; ok {
		return r.level, r.isPII, r.llmAllowed, r.err
	}
	return ResolveLevel("internal"), false, true, nil
}

type mockRegistry struct {
	properties map[string]map[string]ontology.ObjectProperty
	llmReadable map[string]bool
}

func (m *mockRegistry) GetProperties(objectType string) (map[string]ontology.ObjectProperty, error) {
	if props, ok := m.properties[objectType]; ok {
		return props, nil
	}
	return nil, nil
}

func (m *mockRegistry) IsLLMReadable(objectType, property string) bool {
	key := objectType + "." + property
	if v, ok := m.llmReadable[key]; ok {
		return v
	}
	return false
}

func newTestAdapter(classification *mockClassification, registry *mockRegistry) *MarkingAdapter {
	return NewMarkingAdapterWithInterfaces(classification, registry)
}

func TestMarkingAdapter_GetFieldMarking_ClassificationFound(t *testing.T) {
	classification := &mockClassification{
		results: map[string]classificationResult{
			"customer.email": {level: "L3", isPII: true, llmAllowed: false},
		},
	}
	registry := &mockRegistry{
		properties: map[string]map[string]ontology.ObjectProperty{
			"customer": {
				"email": {Name: "email", Sensitivity: "L3", LLMReadable: false},
			},
		},
	}

	adapter := newTestAdapter(classification, registry)
	marking, err := adapter.GetFieldMarking(context.Background(), "customer", "email")

	require.NoError(t, err)
	assert.Equal(t, "L3", marking.Classification)
	assert.True(t, marking.PII)
	assert.False(t, marking.LLMAllowed)
	assert.Equal(t, "L3", marking.Sensitivity)
}

func TestMarkingAdapter_GetFieldMarking_FallbackToOntology(t *testing.T) {
	classification := &mockClassification{
		results: map[string]classificationResult{},
	}
	registry := &mockRegistry{
		properties: map[string]map[string]ontology.ObjectProperty{
			"order": {
				"payment_value": {Name: "payment_value", Sensitivity: "L3", LLMReadable: true},
			},
		},
	}

	adapter := newTestAdapter(classification, registry)
	marking, err := adapter.GetFieldMarking(context.Background(), "order", "payment_value")

	require.NoError(t, err)
	assert.Equal(t, "L3", marking.Classification)
	assert.True(t, marking.PII)
	assert.False(t, marking.LLMAllowed)
	assert.Equal(t, "L3", marking.Sensitivity)
}

func TestMarkingAdapter_GetFieldMarking_DefaultInternal(t *testing.T) {
	classification := &mockClassification{
		results: map[string]classificationResult{},
	}
	registry := &mockRegistry{
		properties: map[string]map[string]ontology.ObjectProperty{
			"product": {
				"name": {Name: "name", Sensitivity: "L0", LLMReadable: true},
			},
		},
	}

	adapter := newTestAdapter(classification, registry)
	marking, err := adapter.GetFieldMarking(context.Background(), "product", "name")

	require.NoError(t, err)
	assert.Equal(t, "L2", marking.Classification)
	assert.False(t, marking.PII)
	assert.True(t, marking.LLMAllowed)
	assert.Equal(t, "L0", marking.Sensitivity)
}

func TestMarkingAdapter_GetFieldMarking_OntologyHigherSensitivity(t *testing.T) {
	classification := &mockClassification{
		results: map[string]classificationResult{
			"order.revenue": {level: "L2", isPII: false, llmAllowed: true},
		},
	}
	registry := &mockRegistry{
		properties: map[string]map[string]ontology.ObjectProperty{
			"order": {
				"revenue": {Name: "revenue", Sensitivity: "L3", LLMReadable: false},
			},
		},
	}

	adapter := newTestAdapter(classification, registry)
	marking, err := adapter.GetFieldMarking(context.Background(), "order", "revenue")

	require.NoError(t, err)
	assert.Equal(t, "L3", marking.Classification)
	assert.True(t, marking.PII)
	assert.False(t, marking.LLMAllowed)
}

func TestMarkingAdapter_GetObjectMarkings(t *testing.T) {
	classification := &mockClassification{
		results: map[string]classificationResult{
			"customer.email": {level: "L3", isPII: true, llmAllowed: false},
		},
	}
	registry := &mockRegistry{
		properties: map[string]map[string]ontology.ObjectProperty{
			"customer": {
				"email": {Name: "email", Sensitivity: "L3", LLMReadable: false},
				"name":  {Name: "name", Sensitivity: "L0", LLMReadable: true},
			},
		},
	}

	adapter := newTestAdapter(classification, registry)
	markings, err := adapter.GetObjectMarkings(context.Background(), "customer")

	require.NoError(t, err)
	assert.Len(t, markings, 2)

	assert.Equal(t, "email", markings[0].Field)
	assert.Equal(t, "L3", markings[0].Classification)
	assert.True(t, markings[0].PII)

	assert.Equal(t, "name", markings[1].Field)
	assert.Equal(t, "L2", markings[1].Classification)
	assert.False(t, markings[1].PII)
}

func TestMarkingAdapter_IsLLMAllowed_L3NotAllowed(t *testing.T) {
	classification := &mockClassification{
		results: map[string]classificationResult{
			"customer.email": {level: "L3", isPII: true, llmAllowed: false},
		},
	}
	registry := &mockRegistry{
		llmReadable: map[string]bool{
			"customer.email": false,
		},
	}

	adapter := newTestAdapter(classification, registry)
	allowed, err := adapter.IsLLMAllowed(context.Background(), "customer", "email")

	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestMarkingAdapter_IsLLMAllowed_L2WithLLMReadable(t *testing.T) {
	classification := &mockClassification{
		results: map[string]classificationResult{
			"product.name": {level: "L2", isPII: false, llmAllowed: true},
		},
	}
	registry := &mockRegistry{
		llmReadable: map[string]bool{
			"product.name": true,
		},
	}

	adapter := newTestAdapter(classification, registry)
	allowed, err := adapter.IsLLMAllowed(context.Background(), "product", "name")

	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestMarkingAdapter_IsLLMAllowed_L2ButNotLLMReadable(t *testing.T) {
	classification := &mockClassification{
		results: map[string]classificationResult{
			"order.internal_note": {level: "L2", isPII: false, llmAllowed: true},
		},
	}
	registry := &mockRegistry{
		llmReadable: map[string]bool{
			"order.internal_note": false,
		},
	}

	adapter := newTestAdapter(classification, registry)
	allowed, err := adapter.IsLLMAllowed(context.Background(), "order", "internal_note")

	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestMarkingAdapter_ClassifyField(t *testing.T) {
	classification := &mockClassification{
		results: map[string]classificationResult{
			"customer.email":  {level: "L3", isPII: true, llmAllowed: false},
			"product.name":    {level: "L1", isPII: false, llmAllowed: true},
		},
	}
	registry := &mockRegistry{}

	adapter := newTestAdapter(classification, registry)

	level, err := adapter.ClassifyField(context.Background(), "customer", "email")
	require.NoError(t, err)
	assert.Equal(t, "L3", level)

	level, err = adapter.ClassifyField(context.Background(), "product", "name")
	require.NoError(t, err)
	assert.Equal(t, "L1", level)
}

func TestMarkingAdapter_ClassifyField_Default(t *testing.T) {
	classification := &mockClassification{
		results: map[string]classificationResult{},
	}
	registry := &mockRegistry{}

	adapter := newTestAdapter(classification, registry)
	level, err := adapter.ClassifyField(context.Background(), "unknown", "field")

	require.NoError(t, err)
	assert.Equal(t, "L2", level)
}

func TestSensitivityToLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"L0", 0},
		{"L1", 1},
		{"L2", 2},
		{"L3", 3},
		{"L4", 4},
		{"", 0},
		{"unknown", 0},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, sensitivityToLevel(tc.input))
		})
	}
}

func TestLevelToPriority(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"L1", 1},
		{"L2", 2},
		{"L3", 3},
		{"unknown", 2},
		{"", 2},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, levelToPriority(tc.input))
		})
	}
}

func TestPriorityToLevel(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "L1"},
		{1, "L1"},
		{2, "L2"},
		{3, "L3"},
		{4, "L3"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, priorityToLevel(tc.input))
		})
	}
}
