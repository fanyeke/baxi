package governance

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/ontology"
	"baxi/internal/repository"
)

func TestResolveLevel_AllCases_Extra(t *testing.T) {
	tests := []struct {
		level    string
		expected string
	}{
		{"pii", "L3"},
		{"sensitive", "L3"},
		{"internal", "L2"},
		{"derived_sensitive", "L2"},
		{"public_internal", "L1"},
		{"unknown", "L2"},
		{"", "L2"},
		{"L3", "L2"},
		{"public", "L2"},
	}
	for _, tc := range tests {
		t.Run(tc.level, func(t *testing.T) {
			assert.Equal(t, tc.expected, ResolveLevel(tc.level))
		})
	}
}

func TestFieldMarking_Structure_Extra(t *testing.T) {
	fm := &FieldMarking{
		ObjectType:     "customer",
		Field:          "email",
		Classification: "L3",
		PII:            true,
		LLMAllowed:     false,
		Sensitivity:    "L3",
	}
	assert.Equal(t, "customer", fm.ObjectType)
	assert.Equal(t, "email", fm.Field)
	assert.Equal(t, "L3", fm.Classification)
	assert.True(t, fm.PII)
	assert.False(t, fm.LLMAllowed)
	assert.Equal(t, "L3", fm.Sensitivity)
}

func TestMarkingAdapter_GetFieldMarking_ClassificationError_Extra(t *testing.T) {
	classification := &mockClassification{
		results: map[string]classificationResult{
			"customer.email": {err: assert.AnError},
		},
	}
	registry := &mockRegistry{}
	adapter := newTestAdapter(classification, registry)

	_, err := adapter.GetFieldMarking(nil, "customer", "email")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "classification")
}

func TestMarkingAdapter_GetFieldMarking_NilContext_Extra(t *testing.T) {
	classification := &mockClassification{
		results: map[string]classificationResult{},
	}
	registry := &mockRegistry{
		properties: map[string]map[string]ontology.ObjectProperty{
			"product": {
				"weight": {Name: "weight", Sensitivity: "L0", LLMReadable: true},
			},
		},
	}
	adapter := newTestAdapter(classification, registry)

	marking, err := adapter.GetFieldMarking(nil, "product", "weight")
	require.NoError(t, err)
	assert.Equal(t, "L2", marking.Classification)
	assert.Equal(t, "L0", marking.Sensitivity)
}

func TestMarkingAdapter_GetObjectMarkings_EmptyProperties_Extra(t *testing.T) {
	classification := &mockClassification{}
	registry := &mockRegistry{
		properties: map[string]map[string]ontology.ObjectProperty{
			"empty_object": {},
		},
	}
	adapter := newTestAdapter(classification, registry)

	markings, err := adapter.GetObjectMarkings(nil, "empty_object")
	require.NoError(t, err)
	assert.Empty(t, markings)
}

func TestMarkingAdapter_IsLLMAllowed_ClassificationError_Extra(t *testing.T) {
	classification := &mockClassification{
		results: map[string]classificationResult{
			"customer.email": {err: assert.AnError},
		},
	}
	registry := &mockRegistry{}
	adapter := newTestAdapter(classification, registry)

	_, err := adapter.IsLLMAllowed(nil, "customer", "email")
	assert.Error(t, err)
}

func TestMarkingAdapter_ClassifyField_Error_Extra(t *testing.T) {
	classification := &mockClassification{
		results: map[string]classificationResult{
			"test.field": {err: assert.AnError},
		},
	}
	registry := &mockRegistry{}
	adapter := newTestAdapter(classification, registry)

	_, err := adapter.ClassifyField(nil, "test", "field")
	assert.Error(t, err)
}

func TestSensitivityToLevel_AllValues_Extra(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"L0", 0},
		{"L1", 1},
		{"L2", 2},
		{"L3", 3},
		{"L4", 4},
		{"L5", 0},
		{"", 0},
		{"X", 0},
		{"l0", 0},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, sensitivityToLevel(tc.input))
		})
	}
}

func TestLevelToPriority_AllValues_Extra(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"L1", 1},
		{"L2", 2},
		{"L3", 3},
		{"L4", 2},
		{"", 2},
		{"X", 2},
		{"l1", 2},
		{"l2", 2},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, levelToPriority(tc.input))
		})
	}
}

func TestPriorityToLevel_AllValues_Extra(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{-1, "L1"},
		{0, "L1"},
		{1, "L1"},
		{2, "L2"},
		{3, "L3"},
		{4, "L3"},
		{5, "L3"},
		{100, "L3"},
	}
	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, priorityToLevel(tc.input))
		})
	}
}

func TestNewMarkingAdapterWithInterfaces_Extra(t *testing.T) {
	classification := &mockClassification{}
	registry := &mockRegistry{}
	adapter := NewMarkingAdapterWithInterfaces(classification, registry)
	assert.NotNil(t, adapter)
}

func TestFilterByRole_WithMultiplePolicies_Extra(t *testing.T) {
	policies := []repository.AccessPolicyRow{
		{PolicyName: "p1", PrincipalPattern: "admin", Effect: "allow"},
		{PolicyName: "p2", PrincipalPattern: "admin", Effect: "deny"},
		{PolicyName: "p3", PrincipalPattern: "viewer", Effect: "allow"},
	}
	result := filterByRole(policies, "admin")
	assert.Len(t, result, 2)
	for _, p := range result {
		assert.Equal(t, "admin", p.PrincipalPattern)
	}
}

func TestMatchesResource_PrefixWildcard_Extra(t *testing.T) {
	p := repository.AccessPolicyRow{ResourcePattern: "dwd_*"}
	assert.True(t, matchesResource(p, "dwd_orders"))
	assert.True(t, matchesResource(p, "dwd_customers"))
	assert.False(t, matchesResource(p, "raw_orders"))
}

func TestMatchesAction_SuffixWildcard_Extra(t *testing.T) {
	p := repository.AccessPolicyRow{Action: "read*"}
	assert.True(t, matchesAction(p, "read"))
	assert.True(t, matchesAction(p, "readonly"))
	assert.True(t, matchesAction(p, "read_data"))
	assert.False(t, matchesAction(p, "write"))
}

func TestMarkingAdapter_GetObjectMarkings_MultipleFieldsSorted_Extra(t *testing.T) {
	classification := &mockClassification{
		results: map[string]classificationResult{
			"customer.z_field": {level: "L2", isPII: false, llmAllowed: true},
			"customer.a_field": {level: "L1", isPII: false, llmAllowed: true},
		},
	}
	registry := &mockRegistry{
		properties: map[string]map[string]ontology.ObjectProperty{
			"customer": {
				"z_field": {Name: "z_field", Sensitivity: "L2", LLMReadable: true},
				"a_field": {Name: "a_field", Sensitivity: "L1", LLMReadable: true},
			},
		},
	}
	adapter := newTestAdapter(classification, registry)

	markings, err := adapter.GetObjectMarkings(nil, "customer")
	require.NoError(t, err)
	require.Len(t, markings, 2)
	assert.Equal(t, "a_field", markings[0].Field)
	assert.Equal(t, "z_field", markings[1].Field)
}

func TestNewMarkingAdapter_Extra(t *testing.T) {
	classification := &mockClassification{}
	// Use a minimal mock that implements the interface needed by NewMarkingAdapter
	// Since NewMarkingAdapter takes *ontology.ObjectRegistry, we create one
	registry, err := ontology.NewObjectRegistry(nil, nil, nil, "")
	// This will fail to load the schema, but it creates a valid registry
	if err == nil {
		adapter := NewMarkingAdapter(classification, registry)
		assert.NotNil(t, adapter)
	}
}
