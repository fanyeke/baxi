package governance

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/ontology"
	"baxi/internal/repository"
	governanceRepo "baxi/internal/repository/governance"
)

// ──── mockConfigSnapshotProvider ────────────────────────────────────────────

type testConfigSnapshotProvider struct {
	snapshots []governanceRepo.ConfigSnapshotRow
	err       error
}

func (m *testConfigSnapshotProvider) GetConfigSnapshots(ctx context.Context) ([]governanceRepo.ConfigSnapshotRow, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.snapshots, nil
}

// ──── CheckpointService additional tests ────────────────────────────────────

func TestCheckpointService_RequiresCheckpoint_ConfigWithYMLKey(t *testing.T) {
	cfg := checkpointRulesConfig{
		Checkpoints: []CheckpointRule{
			{Action: "custom_action", RequiresReason: true, RequiresHumanReview: false},
		},
	}
	cfgJSON, _ := json.Marshal(cfg)

	provider := &testConfigSnapshotProvider{
		snapshots: []governanceRepo.ConfigSnapshotRow{
			{ConfigKey: "checkpoint_rules.yml", Status: string(cfgJSON)},
		},
	}
	svc := NewCheckpointServiceWithProvider(provider)

	assert.True(t, svc.RequiresCheckpoint(context.Background(), "custom_action"))
}

func TestCheckpointService_GetRules_EmptySnapshots(t *testing.T) {
	provider := &testConfigSnapshotProvider{
		snapshots: []governanceRepo.ConfigSnapshotRow{},
	}
	svc := NewCheckpointServiceWithProvider(provider)

	rules := svc.GetRules(context.Background())
	assert.NotNil(t, rules)
	assert.Len(t, rules, 3)
}

func TestCheckpointService_GetRules_DeduplicatesConfigRule(t *testing.T) {
	cfg := checkpointRulesConfig{
		Checkpoints: []CheckpointRule{
			{Action: "execute_dispatch", RequiresReason: false, RequiresHumanReview: false},
		},
	}
	cfgJSON, _ := json.Marshal(cfg)

	provider := &testConfigSnapshotProvider{
		snapshots: []governanceRepo.ConfigSnapshotRow{
			{ConfigKey: "checkpoint_rules", Status: string(cfgJSON)},
		},
	}
	svc := NewCheckpointServiceWithProvider(provider)

	rules := svc.GetRules(context.Background())
	// execute_dispatch should appear only once
	count := 0
	for _, r := range rules {
		if r.Action == "execute_dispatch" {
			count++
		}
	}
	assert.Equal(t, 1, count)
}

func TestCheckpointService_GetRules_ProviderError(t *testing.T) {
	provider := &testConfigSnapshotProvider{err: fmt.Errorf("db error")}
	svc := NewCheckpointServiceWithProvider(provider)

	rules := svc.GetRules(context.Background())
	// When provider errors, GetRules returns nil (not built-ins)
	assert.Nil(t, rules)
}

// ──── MarkingAdapter additional tests ───────────────────────────────────────

type testClassificationLookup struct {
	level      string
	isPII      bool
	llmAllowed bool
	err        error
}

func (m *testClassificationLookup) GetFieldMarking(ctx context.Context, objectType, property string) (string, bool, bool, error) {
	if m.err != nil {
		return "", false, false, m.err
	}
	return m.level, m.isPII, m.llmAllowed, nil
}

type testRegistryLookup struct {
	props     map[string]map[string]string
	_readable map[string]map[string]bool
}

func (m *testRegistryLookup) GetProperties(objectType string) (map[string]ontology.ObjectProperty, error) {
	if m.props == nil {
		return nil, fmt.Errorf("unknown object type: %s", objectType)
	}
	p, ok := m.props[objectType]
	if !ok {
		return nil, fmt.Errorf("unknown object type: %s", objectType)
	}
	result := make(map[string]ontology.ObjectProperty)
	for k, v := range p {
		result[k] = ontology.ObjectProperty{Sensitivity: v}
	}
	return result, nil
}

func (m *testRegistryLookup) IsLLMReadable(objectType, property string) bool {
	if m._readable == nil {
		return true
	}
	if obj, ok := m._readable[objectType]; ok {
		if val, ok := obj[property]; ok {
			return val
		}
	}
	return true
}

func TestMarkingAdapter_GetFieldMarking_OntologyUpgrade(t *testing.T) {
	classification := &testClassificationLookup{
		level:      "L2",
		isPII:      false,
		llmAllowed: true,
	}
	registry := &testRegistryLookup{
		props: map[string]map[string]string{
			"customer": {"email": "L4"},
		},
	}
	adapter := NewMarkingAdapterWithInterfaces(classification, registry)

	marking, err := adapter.GetFieldMarking(context.Background(), "customer", "email")
	require.NoError(t, err)
	// L2 default with ontology L4 should upgrade to L3
	assert.Equal(t, "L3", marking.Classification)
	assert.True(t, marking.PII)
	assert.False(t, marking.LLMAllowed)
	assert.Equal(t, "L4", marking.Sensitivity)
}

func TestMarkingAdapter_GetFieldMarking_ClassificationError(t *testing.T) {
	classification := &testClassificationLookup{err: fmt.Errorf("db error")}
	registry := &testRegistryLookup{}
	adapter := NewMarkingAdapterWithInterfaces(classification, registry)

	_, err := adapter.GetFieldMarking(context.Background(), "customer", "email")
	assert.Error(t, err)
}

func TestMarkingAdapter_GetObjectMarkings_Sorted(t *testing.T) {
	classification := &testClassificationLookup{
		level:      "L1",
		isPII:      false,
		llmAllowed: true,
	}
	registry := &testRegistryLookup{
		props: map[string]map[string]string{
			"customer": {"email": "", "name": ""},
		},
	}
	adapter := NewMarkingAdapterWithInterfaces(classification, registry)

	markings, err := adapter.GetObjectMarkings(context.Background(), "customer")
	require.NoError(t, err)
	assert.Len(t, markings, 2)
	// Should be sorted by field name
	assert.Equal(t, "email", markings[0].Field)
	assert.Equal(t, "name", markings[1].Field)
}

func TestMarkingAdapter_GetObjectMarkings_RegistryError(t *testing.T) {
	classification := &testClassificationLookup{}
	registry := &testRegistryLookup{}
	adapter := NewMarkingAdapterWithInterfaces(classification, registry)

	_, err := adapter.GetObjectMarkings(context.Background(), "unknown")
	assert.Error(t, err)
}

func TestMarkingAdapter_IsLLMAllowed_L3(t *testing.T) {
	classification := &testClassificationLookup{
		level:      "L3",
		llmAllowed: false,
	}
	registry := &testRegistryLookup{}
	adapter := NewMarkingAdapterWithInterfaces(classification, registry)

	allowed, err := adapter.IsLLMAllowed(context.Background(), "customer", "email")
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestMarkingAdapter_IsLLMAllowed_L1(t *testing.T) {
	classification := &testClassificationLookup{
		level:      "L1",
		llmAllowed: true,
	}
	registry := &testRegistryLookup{
		_readable: map[string]map[string]bool{
			"customer": {"name": true},
		},
	}
	adapter := NewMarkingAdapterWithInterfaces(classification, registry)

	allowed, err := adapter.IsLLMAllowed(context.Background(), "customer", "name")
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestMarkingAdapter_IsLLMAllowed_ClassificationError(t *testing.T) {
	classification := &testClassificationLookup{err: fmt.Errorf("error")}
	registry := &testRegistryLookup{}
	adapter := NewMarkingAdapterWithInterfaces(classification, registry)

	_, err := adapter.IsLLMAllowed(context.Background(), "customer", "email")
	assert.Error(t, err)
}

func TestMarkingAdapter_ClassifyField_Success(t *testing.T) {
	classification := &testClassificationLookup{
		level:      "L2",
		llmAllowed: true,
	}
	registry := &testRegistryLookup{}
	adapter := NewMarkingAdapterWithInterfaces(classification, registry)

	level, err := adapter.ClassifyField(context.Background(), "customer", "name")
	require.NoError(t, err)
	assert.Equal(t, "L2", level)
}

func TestMarkingAdapter_ClassifyField_Error(t *testing.T) {
	classification := &testClassificationLookup{err: fmt.Errorf("error")}
	registry := &testRegistryLookup{}
	adapter := NewMarkingAdapterWithInterfaces(classification, registry)

	_, err := adapter.ClassifyField(context.Background(), "customer", "name")
	assert.Error(t, err)
}

func TestMarkingAdapter_getOntologySensitivity_UnknownType(t *testing.T) {
	registry := &testRegistryLookup{}
	adapter := NewMarkingAdapterWithInterfaces(nil, registry)

	sensitivity := adapter.getOntologySensitivity("unknown", "field")
	assert.Empty(t, sensitivity)
}

func TestMarkingAdapter_getOntologySensitivity_UnknownField(t *testing.T) {
	registry := &testRegistryLookup{
		props: map[string]map[string]string{
			"customer": {"name": "L1"},
		},
	}
	adapter := NewMarkingAdapterWithInterfaces(nil, registry)

	sensitivity := adapter.getOntologySensitivity("customer", "unknown")
	assert.Empty(t, sensitivity)
}

// ──── SensitivityToLevel tests ──────────────────────────────────────────────

func TestSensitivityToLevel_All(t *testing.T) {
	assert.Equal(t, 0, sensitivityToLevel("L0"))
	assert.Equal(t, 1, sensitivityToLevel("L1"))
	assert.Equal(t, 2, sensitivityToLevel("L2"))
	assert.Equal(t, 3, sensitivityToLevel("L3"))
	assert.Equal(t, 4, sensitivityToLevel("L4"))
	assert.Equal(t, 0, sensitivityToLevel("unknown"))
}

func TestLevelToPriority_All(t *testing.T) {
	assert.Equal(t, 1, levelToPriority("L1"))
	assert.Equal(t, 2, levelToPriority("L2"))
	assert.Equal(t, 3, levelToPriority("L3"))
	assert.Equal(t, 2, levelToPriority("unknown"))
}

func TestPriorityToLevel_All(t *testing.T) {
	assert.Equal(t, "L3", priorityToLevel(3))
	assert.Equal(t, "L3", priorityToLevel(4))
	assert.Equal(t, "L2", priorityToLevel(2))
	assert.Equal(t, "L1", priorityToLevel(1))
	assert.Equal(t, "L1", priorityToLevel(0))
}

// ──── filterByRole additional test ──────────────────────────────────────────

func TestFilterByRole_NoMatchResult(t *testing.T) {
	policies := []repository.AccessPolicyRow{
		{PrincipalPattern: "admin", Effect: "allow"},
		{PrincipalPattern: "analyst", Effect: "allow"},
	}
	filtered := filterByRole(policies, "viewer")
	assert.NotNil(t, filtered)
	assert.Empty(t, filtered)
}

// ──── deprecatedLineageAdapter tests ────────────────────────────────────────

func TestConvertLineageRow(t *testing.T) {
	input := repository.DataLineageRow{
		SourceTable:         "src",
		SourceColumn:        "col1",
		TargetTable:         "tgt",
		TargetColumn:        "col2",
		TransformationLogic: "direct",
		Confidence:          0.95,
	}
	result := convertLineageRow(input)
	assert.Equal(t, "src", result.SourceTable)
	assert.Equal(t, "col1", result.SourceColumn)
	assert.Equal(t, "tgt", result.TargetTable)
	assert.Equal(t, "col2", result.TargetColumn)
	assert.Equal(t, "direct", result.TransformationLogic)
	assert.InDelta(t, 0.95, result.Confidence, 0.001)
}

// ──── NewClassificationService ──────────────────────────────────────────────

func TestNewClassificationService_NilPool(t *testing.T) {
	repo := repository.NewGovernanceRepository()
	svc := NewClassificationService(nil, repo)
	assert.NotNil(t, svc)
}

func TestResolveLevel_AllLevels(t *testing.T) {
	assert.Equal(t, "L3", ResolveLevel("pii"))
	assert.Equal(t, "L3", ResolveLevel("sensitive"))
	assert.Equal(t, "L2", ResolveLevel("internal"))
	assert.Equal(t, "L2", ResolveLevel("derived_sensitive"))
	assert.Equal(t, "L1", ResolveLevel("public_internal"))
	assert.Equal(t, "L2", ResolveLevel("unknown"))
	assert.Equal(t, "L2", ResolveLevel(""))
}
