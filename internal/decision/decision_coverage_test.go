package decision

import (
	"context"
	"testing"

	"baxi/internal/feature"
	"baxi/internal/governance"
	"baxi/internal/ontology"
	"baxi/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

func TestNoopSnapshotRecorder_RecordSnapshot(t *testing.T) {
	recorder := NewNoopSnapshotRecorder()
	err := recorder.RecordSnapshot(context.Background(), DataSnapshotRecord{
		CaseID:       "dc-1",
		SnapshotType: SnapshotTypeAlertContext,
	})
	assert.NoError(t, err)
}

func TestNoopSnapshotRecorder_RecordEvent(t *testing.T) {
	recorder := NewNoopSnapshotRecorder()
	err := recorder.RecordEvent(context.Background(), LineageEventRecord{
		CaseID:    "dc-1",
		EventType: LineageEventCaseCreated,
	})
	assert.NoError(t, err)
}

func TestNoopSnapshotRecorder_ImplementsInterface(t *testing.T) {
	var _ SnapshotRecorder = NewNoopSnapshotRecorder()
}

func TestDerefString_Nil(t *testing.T) {
	assert.Equal(t, "", derefString(nil))
}

func TestDerefString_NonNil(t *testing.T) {
	s := "hello"
	assert.Equal(t, "hello", derefString(&s))
}

func TestDerefString_Empty(t *testing.T) {
	s := ""
	assert.Equal(t, "", derefString(&s))
}

func TestGetStringProp_StringValue(t *testing.T) {
	props := map[string]interface{}{"key": "value"}
	assert.Equal(t, "value", getStringProp(props, "key"))
}

func TestGetStringProp_NonStringValue(t *testing.T) {
	props := map[string]interface{}{"key": 42}
	assert.Equal(t, "", getStringProp(props, "key"))
}

func TestGetStringProp_MissingKey(t *testing.T) {
	props := map[string]interface{}{"other": "value"}
	assert.Equal(t, "", getStringProp(props, "key"))
}

func TestGetStringProp_EmptyMap(t *testing.T) {
	props := map[string]interface{}{}
	assert.Equal(t, "", getStringProp(props, "key"))
}

func TestMapClassification_PII(t *testing.T) {
	assert.Equal(t, "pii", mapClassification("L3", true))
}

func TestMapClassification_Sensitive(t *testing.T) {
	assert.Equal(t, "sensitive", mapClassification("L3", false))
}

func TestMapClassification_Internal(t *testing.T) {
	assert.Equal(t, "internal", mapClassification("L2", false))
}

func TestMapClassification_PublicInternal(t *testing.T) {
	assert.Equal(t, "public_internal", mapClassification("L1", false))
}

func TestMapClassification_Default(t *testing.T) {
	assert.Equal(t, "internal", mapClassification("unknown", false))
}

func TestMapClassification_EmptyLevel(t *testing.T) {
	assert.Equal(t, "internal", mapClassification("", false))
}

func TestResolveOverallClassification_Empty(t *testing.T) {
	assert.Equal(t, "L1", resolveOverallClassification(map[string]string{}))
}

func TestResolveOverallClassification_AllInternal(t *testing.T) {
	assert.Equal(t, "L2", resolveOverallClassification(map[string]string{
		"f1": "internal",
		"f2": "internal",
	}))
}

func TestResolveOverallClassification_MixedLevels(t *testing.T) {
	assert.Equal(t, "L3", resolveOverallClassification(map[string]string{
		"f1": "internal",
		"f2": "pii",
	}))
}

func TestResolveOverallClassification_WithDerivedSensitive(t *testing.T) {
	assert.Equal(t, "L2", resolveOverallClassification(map[string]string{
		"f1": "derived_sensitive",
	}))
}

// helper to build a minimal oldBuilder (*ContextBuilder) for switchable tests
func minimalOldBuilder() *ContextBuilder {
	caseSvc := &mockDecisionCaseDataProvider{
		getCaseByIDFn: func(ctx context.Context, pool *pgxpool.Pool, caseID string) (*repository.DecisionCaseRow, error) {
			objectType := "seller"
			objectID := "seller-1"
			return &repository.DecisionCaseRow{
				CaseID:     "dc-1",
				ObjectType: &objectType,
				ObjectID:   &objectID,
			}, nil
		},
	}
	objectProvider := &mockObjectDataProvider{
		buildObjectContextFn: func(ctx context.Context, objectType, objectID string) (*ontology.ObjectContext, error) {
			return &ontology.ObjectContext{
				ObjectType: objectType,
				ObjectID:   objectID,
				Properties: map[string]interface{}{"name": "Old Builder"},
			}, nil
		},
	}
	classProvider := &mockClassificationProvider{
		getFieldMarkingFn: func(ctx context.Context, objectType, property string) (string, bool, bool, error) {
			return "L1", false, true, nil
		},
	}
	return NewContextBuilder(caseSvc, objectProvider, classProvider, nil, testActionTypes)
}

// --- SwitchableContextBuilder: v3 with nil v3Builder ---

func TestSwitchableContextBuilder_SwitchToV3_NilV3Builder(t *testing.T) {
	oldBuilder := minimalOldBuilder()
	switcher := NewSwitchableContextBuilder(oldBuilder, nil, nil)
	switcher.SwitchTo(BuilderV3)

	// v3Builder is nil, so it should fall through to oldBuilder
	result, err := switcher.BuildDecisionContext(context.Background(), "dc-1")
	assert.NoError(t, err)
	assert.Equal(t, "Old Builder", result.ObjectContext.Properties["name"])
}

func TestSwitchableContextBuilder_SwitchToV2_NilNewBuilder(t *testing.T) {
	oldBuilder := minimalOldBuilder()
	switcher := NewSwitchableContextBuilder(oldBuilder, nil, nil)
	switcher.SwitchTo(BuilderV2)

	// newBuilder is nil, so it should fall through to oldBuilder
	result, err := switcher.BuildDecisionContext(context.Background(), "dc-1")
	assert.NoError(t, err)
	assert.Equal(t, "Old Builder", result.ObjectContext.Properties["name"])
}

func TestSwitchableContextBuilder_SwitchToEmpty_FallsThrough(t *testing.T) {
	oldBuilder := minimalOldBuilder()
	switcher := NewSwitchableContextBuilder(oldBuilder, nil, nil)
	switcher.SwitchTo("") // empty version falls through

	result, err := switcher.BuildDecisionContext(context.Background(), "dc-1")
	assert.NoError(t, err)
	assert.Equal(t, "Old Builder", result.ObjectContext.Properties["name"])
}

func TestSwitchableContextBuilder_FlagOn_NilNewBuilder_FallsToOld(t *testing.T) {
	oldBuilder := minimalOldBuilder()
	flags := &feature.FeatureFlags{NewContextBuilder: true}
	switcher := NewSwitchableContextBuilder(oldBuilder, nil, flags)

	result, err := switcher.BuildDecisionContext(context.Background(), "dc-1")
	assert.NoError(t, err)
	assert.Equal(t, "Old Builder", result.ObjectContext.Properties["name"])
}

func TestSwitchableContextBuilder_V3WithPath(t *testing.T) {
	delegate := &mockObjectContextBuilder{
		buildFn: func(ctx context.Context, caseID string) (*DecisionContext, error) {
			return &DecisionContext{DecisionCaseID: "v3-result"}, nil
		},
	}
	v3 := NewContextBuilderV3(delegate, nil, nil)
	oldBuilder := minimalOldBuilder()
	switcher := NewSwitchableContextBuilder(oldBuilder, nil, nil)
	switcher.WithV3Builder(v3)
	switcher.SwitchTo(BuilderV3)

	result, err := switcher.BuildDecisionContext(context.Background(), "dc-1")
	assert.NoError(t, err)
	assert.Equal(t, "v3-result", result.DecisionCaseID)
}

func TestSwitchableContextBuilder_V2WithPath(t *testing.T) {
	caseSvc := &mockDecisionCaseDataProvider{
		getCaseByIDFn: func(ctx context.Context, pool *pgxpool.Pool, caseID string) (*repository.DecisionCaseRow, error) {
			objectType := "seller"
			objectID := "seller-1"
			return &repository.DecisionCaseRow{
				CaseID:     "dc-1",
				ObjectType: &objectType,
				ObjectID:   &objectID,
			}, nil
		},
	}
	ontologyRepo := &mockOntologyAwareRepo{
		getObjectByIDFn: func(ctx context.Context, pool *pgxpool.Pool, objectType, objectID string) (*repository.ObjectInstance, error) {
			return &repository.ObjectInstance{
				ObjectType: objectType,
				ID:         objectID,
				Properties: map[string]interface{}{"name": "v2-data"},
			}, nil
		},
	}
	markingSvc := &mockMarkingService{
		getFieldMarkingFn: func(ctx context.Context, objectType, field string) (*governance.FieldMarking, error) {
			return &governance.FieldMarking{Classification: "L1", PII: false}, nil
		},
	}
	newBuilder := NewContextBuilderV2(caseSvc, ontologyRepo, markingSvc, nil, nil, testActionTypes)
	oldBuilder := minimalOldBuilder()
	switcher := NewSwitchableContextBuilder(oldBuilder, newBuilder, nil)
	switcher.SwitchTo(BuilderV2)

	result, err := switcher.BuildDecisionContext(context.Background(), "dc-1")
	assert.NoError(t, err)
	assert.Equal(t, "dc-1", result.DecisionCaseID)
}
