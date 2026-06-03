package decision

import (
	"context"
	"errors"
	"testing"

	"baxi/internal/feature"
	"baxi/internal/governance"
	"baxi/internal/ontology"
	"baxi/internal/repository"
	decisionRepo "baxi/internal/repository/decision"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

type mockOntologyAwareRepo struct {
	getObjectByIDFn func(ctx context.Context, pool *pgxpool.Pool, objectType, objectID string) (*repository.ObjectInstance, error)
}

func (m *mockOntologyAwareRepo) GetObjectByID(ctx context.Context, pool *pgxpool.Pool, objectType, objectID string) (*repository.ObjectInstance, error) {
	return m.getObjectByIDFn(ctx, pool, objectType, objectID)
}

func (m *mockOntologyAwareRepo) QueryByObjectType(ctx context.Context, pool *pgxpool.Pool, objectType string, filters repository.ObjectFilters) (*repository.ObjectQueryResult, error) {
	return nil, errors.New("not implemented")
}

func (m *mockOntologyAwareRepo) GetObjectTypeSchema(ctx context.Context, objectType string) (*repository.ObjectSchema, error) {
	return nil, errors.New("not implemented")
}

type mockMarkingService struct {
	getFieldMarkingFn func(ctx context.Context, objectType, field string) (*governance.FieldMarking, error)
}

func (m *mockMarkingService) GetFieldMarking(ctx context.Context, objectType, field string) (*governance.FieldMarking, error) {
	return m.getFieldMarkingFn(ctx, objectType, field)
}

func (m *mockMarkingService) GetObjectMarkings(ctx context.Context, objectType string) ([]governance.FieldMarking, error) {
	return nil, errors.New("not implemented")
}

func (m *mockMarkingService) IsLLMAllowed(ctx context.Context, objectType, field string) (bool, error) {
	return false, errors.New("not implemented")
}

func (m *mockMarkingService) ClassifyField(ctx context.Context, objectType, field string) (string, error) {
	return "", errors.New("not implemented")
}

type mockDecisionLineageService struct {
	getContextLineageFn func(ctx context.Context, caseID string) (*ContextLineage, error)
}

func (m *mockDecisionLineageService) GetDecisionLineage(ctx context.Context, caseID string) (*DecisionLineageChain, error) {
	return nil, errors.New("not implemented")
}

func (m *mockDecisionLineageService) GetContextLineage(ctx context.Context, caseID string) (*ContextLineage, error) {
	return m.getContextLineageFn(ctx, caseID)
}

func (m *mockDecisionLineageService) RecordDecisionLineage(ctx context.Context, record LineageEventRecord) error {
	return errors.New("not implemented")
}

type mockActionTypeProvider struct{}

func (m *mockActionTypeProvider) ListActionTypes() []string {
	return []string{"create_followup_task", "notify_owner", "export_report", "escalate_to_human"}
}
func (m *mockActionTypeProvider) IsActionAllowed(actionType string) bool {
	return true
}
func (m *mockActionTypeProvider) GetActionPolicy(actionType string) (ActionPolicy, bool) {
	return ActionPolicy{RiskLevel: "medium", RequiresApproval: false, AllowedBy: []string{"system"}}, true
}

var testActionTypes = &mockActionTypeProvider{}

func (m *mockDecisionLineageService) RecordDataSnapshot(ctx context.Context, record DataSnapshotRecord) error {
	return errors.New("not implemented")
}

func TestContextBuilderV2_BuildDecisionContext_WithTriggerData(t *testing.T) {
	caseSvc := &mockDecisionCaseDataProvider{
		getCaseByIDFn: func(ctx context.Context, caseID string) (*decisionRepo.DecisionCaseRow, error) {
			alertID := "alert-1"
			severity := "high"
			objectType := "seller"
			objectID := "seller-42"
			return &decisionRepo.DecisionCaseRow{
				CaseID:     "dc-1",
				SourceType: strPtr("alert"),
				SourceID:   strPtr("alert-1"),
				AlertID:    &alertID,
				Severity:   &severity,
				ObjectType: &objectType,
				ObjectID:   &objectID,
			}, nil
		},
	}

	ontologyRepo := &mockOntologyAwareRepo{
		getObjectByIDFn: func(ctx context.Context, pool *pgxpool.Pool, objectType, objectID string) (*repository.ObjectInstance, error) {
			if objectType == "metric_alert" {
				return &repository.ObjectInstance{
					ObjectType: "metric_alert",
					ID:         objectID,
					Properties: map[string]interface{}{
						"rule_id":        "rule-1",
						"metric_name":    "gmv_drop",
						"current_value":  100.0,
						"baseline_value": 150.0,
						"delta_pct":      -33.3,
						"severity":       "high",
					},
				}, nil
			}
			return &repository.ObjectInstance{
				ObjectType: objectType,
				ID:         objectID,
				Properties: map[string]interface{}{
					"name":  "Test Seller",
					"state": "SP",
				},
			}, nil
		},
	}

	markingSvc := &mockMarkingService{
		getFieldMarkingFn: func(ctx context.Context, objectType, field string) (*governance.FieldMarking, error) {
			return &governance.FieldMarking{
				ObjectType:     objectType,
				Field:          field,
				Classification: "L1",
				PII:            false,
				LLMAllowed:     true,
			}, nil
		},
	}

	lineageSvc := &mockDecisionLineageService{
		getContextLineageFn: func(ctx context.Context, caseID string) (*ContextLineage, error) {
			return &ContextLineage{
				CaseID:         caseID,
				UpstreamTables: []string{"dwd_order_level", "dwd_item_level"},
			}, nil
		},
	}

	builder := NewContextBuilderV2(caseSvc, ontologyRepo, markingSvc, lineageSvc, nil, testActionTypes)
	decisionCtx, err := builder.BuildDecisionContext(context.Background(), "dc-1")

	assert.NoError(t, err)
	assert.NotNil(t, decisionCtx)
	assert.Equal(t, "dc-1", decisionCtx.DecisionCaseID)
	assert.Equal(t, "alert", *decisionCtx.SourceType)
	assert.Equal(t, "alert-1", decisionCtx.Trigger.AlertID)
	assert.Equal(t, "rule-1", decisionCtx.Trigger.RuleID)
	assert.Equal(t, "high", decisionCtx.Trigger.Severity)
	assert.Equal(t, "gmv_drop", decisionCtx.Trigger.MetricName)
	assert.Equal(t, 100.0, decisionCtx.Trigger.CurrentValue)
	assert.Equal(t, 150.0, decisionCtx.Trigger.BaselineValue)
	assert.InDelta(t, -33.3, decisionCtx.Trigger.DeltaPct, 0.1)
}

func TestContextBuilderV2_BuildDecisionContext_AppliesRedaction(t *testing.T) {
	caseSvc := &mockDecisionCaseDataProvider{
		getCaseByIDFn: func(ctx context.Context, caseID string) (*decisionRepo.DecisionCaseRow, error) {
			alertID := "alert-1"
			severity := "high"
			objectType := "seller"
			objectID := "seller-42"
			return &decisionRepo.DecisionCaseRow{
				CaseID:     "dc-1",
				SourceType: strPtr("alert"),
				SourceID:   strPtr("alert-1"),
				AlertID:    &alertID,
				Severity:   &severity,
				ObjectType: &objectType,
				ObjectID:   &objectID,
			}, nil
		},
	}

	ontologyRepo := &mockOntologyAwareRepo{
		getObjectByIDFn: func(ctx context.Context, pool *pgxpool.Pool, objectType, objectID string) (*repository.ObjectInstance, error) {
			if objectType == "metric_alert" {
				return &repository.ObjectInstance{
					ObjectType: "metric_alert",
					ID:         objectID,
					Properties: map[string]interface{}{"rule_id": "rule-1"},
				}, nil
			}
			return &repository.ObjectInstance{
				ObjectType: objectType,
				ID:         objectID,
				Properties: map[string]interface{}{
					"name":    "Test Seller",
					"email":   "seller@example.com",
					"revenue": 10000.0,
					"state":   "SP",
				},
			}, nil
		},
	}

	markingSvc := &mockMarkingService{
		getFieldMarkingFn: func(ctx context.Context, objectType, field string) (*governance.FieldMarking, error) {
			switch field {
			case "email":
				return &governance.FieldMarking{Classification: "L3", PII: true}, nil
			case "revenue":
				return &governance.FieldMarking{Classification: "L3", PII: false}, nil
			case "name", "state":
				return &governance.FieldMarking{Classification: "L2", PII: false}, nil
			default:
				return &governance.FieldMarking{Classification: "L1", PII: false}, nil
			}
		},
	}

	builder := NewContextBuilderV2(caseSvc, ontologyRepo, markingSvc, nil, nil, testActionTypes)
	decisionCtx, err := builder.BuildDecisionContext(context.Background(), "dc-1")

	assert.NoError(t, err)
	assert.NotNil(t, decisionCtx)

	_, hasEmail := decisionCtx.ObjectContext.Properties["email"]
	_, hasRevenue := decisionCtx.ObjectContext.Properties["revenue"]
	assert.False(t, hasEmail, "email should be redacted")
	assert.False(t, hasRevenue, "revenue should be redacted")

	_, hasName := decisionCtx.ObjectContext.Properties["name"]
	_, hasState := decisionCtx.ObjectContext.Properties["state"]
	assert.True(t, hasName, "name should be kept")
	assert.True(t, hasState, "state should be kept")
}

func TestContextBuilderV2_BuildDecisionContext_PopulatesLineage(t *testing.T) {
	caseSvc := &mockDecisionCaseDataProvider{
		getCaseByIDFn: func(ctx context.Context, caseID string) (*decisionRepo.DecisionCaseRow, error) {
			severity := "medium"
			objectType := "seller"
			objectID := "seller-1"
			return &decisionRepo.DecisionCaseRow{
				CaseID:     "dc-1",
				SourceType: strPtr("alert"),
				SourceID:   strPtr("alert-1"),
				Severity:   &severity,
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
				Properties: map[string]interface{}{"name": "Test"},
			}, nil
		},
	}

	markingSvc := &mockMarkingService{
		getFieldMarkingFn: func(ctx context.Context, objectType, field string) (*governance.FieldMarking, error) {
			return &governance.FieldMarking{Classification: "L1", PII: false}, nil
		},
	}

	lineageSvc := &mockDecisionLineageService{
		getContextLineageFn: func(ctx context.Context, caseID string) (*ContextLineage, error) {
			return &ContextLineage{
				CaseID:         caseID,
				UpstreamTables: []string{"dwd_order_level", "dwd_item_level"},
				ConfigVersions: map[string]string{"alert_rules_version": "v1.2.0"},
			}, nil
		},
	}

	builder := NewContextBuilderV2(caseSvc, ontologyRepo, markingSvc, lineageSvc, nil, testActionTypes)
	decisionCtx, err := builder.BuildDecisionContext(context.Background(), "dc-1")

	assert.NoError(t, err)
	assert.NotNil(t, decisionCtx)
	assert.NotNil(t, decisionCtx.Governance.Lineage)
	assert.Equal(t, []string{"dwd_order_level", "dwd_item_level"}, decisionCtx.Governance.Lineage.Upstream)
}

func TestContextBuilderV2_BuildDecisionContext_LineageErrorNonFatal(t *testing.T) {
	caseSvc := &mockDecisionCaseDataProvider{
		getCaseByIDFn: func(ctx context.Context, caseID string) (*decisionRepo.DecisionCaseRow, error) {
			severity := "low"
			objectType := "seller"
			objectID := "seller-1"
			return &decisionRepo.DecisionCaseRow{
				CaseID:     "dc-1",
				SourceType: strPtr("alert"),
				SourceID:   strPtr("alert-1"),
				Severity:   &severity,
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
				Properties: map[string]interface{}{"name": "Test"},
			}, nil
		},
	}

	markingSvc := &mockMarkingService{
		getFieldMarkingFn: func(ctx context.Context, objectType, field string) (*governance.FieldMarking, error) {
			return &governance.FieldMarking{Classification: "L1", PII: false}, nil
		},
	}

	lineageSvc := &mockDecisionLineageService{
		getContextLineageFn: func(ctx context.Context, caseID string) (*ContextLineage, error) {
			return nil, errors.New("lineage service unavailable")
		},
	}

	builder := NewContextBuilderV2(caseSvc, ontologyRepo, markingSvc, lineageSvc, nil, testActionTypes)
	decisionCtx, err := builder.BuildDecisionContext(context.Background(), "dc-1")

	assert.NoError(t, err)
	assert.NotNil(t, decisionCtx)
	assert.Nil(t, decisionCtx.Governance.Lineage)
}

func TestContextBuilderV2_BuildDecisionContext_CaseNotFound(t *testing.T) {
	caseSvc := &mockDecisionCaseDataProvider{
		getCaseByIDFn: func(ctx context.Context, caseID string) (*decisionRepo.DecisionCaseRow, error) {
			return nil, errors.New("case not found")
		},
	}

	builder := NewContextBuilderV2(caseSvc, nil, nil, nil, nil, testActionTypes)
	decisionCtx, err := builder.BuildDecisionContext(context.Background(), "dc-missing")

	assert.Error(t, err)
	assert.Nil(t, decisionCtx)
	assert.Contains(t, err.Error(), "fetch case")
}

func TestContextBuilderV2_BuildDecisionContext_ObjectNotFound(t *testing.T) {
	caseSvc := &mockDecisionCaseDataProvider{
		getCaseByIDFn: func(ctx context.Context, caseID string) (*decisionRepo.DecisionCaseRow, error) {
			objectType := "seller"
			objectID := "seller-1"
			return &decisionRepo.DecisionCaseRow{
				CaseID:     "dc-1",
				ObjectType: &objectType,
				ObjectID:   &objectID,
			}, nil
		},
	}

	ontologyRepo := &mockOntologyAwareRepo{
		getObjectByIDFn: func(ctx context.Context, pool *pgxpool.Pool, objectType, objectID string) (*repository.ObjectInstance, error) {
			return nil, errors.New("object not found")
		},
	}

	builder := NewContextBuilderV2(caseSvc, ontologyRepo, nil, nil, nil, testActionTypes)
	decisionCtx, err := builder.BuildDecisionContext(context.Background(), "dc-1")

	assert.Error(t, err)
	assert.Nil(t, decisionCtx)
	assert.Contains(t, err.Error(), "get object")
}

func TestContextBuilderV2_BuildDecisionContext_AlertError(t *testing.T) {
	caseSvc := &mockDecisionCaseDataProvider{
		getCaseByIDFn: func(ctx context.Context, caseID string) (*decisionRepo.DecisionCaseRow, error) {
			alertID := "alert-1"
			severity := "high"
			objectType := "seller"
			objectID := "seller-1"
			return &decisionRepo.DecisionCaseRow{
				CaseID:     "dc-1",
				SourceType: strPtr("alert"),
				SourceID:   strPtr("alert-1"),
				AlertID:    &alertID,
				Severity:   &severity,
				ObjectType: &objectType,
				ObjectID:   &objectID,
			}, nil
		},
	}

	ontologyRepo := &mockOntologyAwareRepo{
		getObjectByIDFn: func(ctx context.Context, pool *pgxpool.Pool, objectType, objectID string) (*repository.ObjectInstance, error) {
			if objectType == "metric_alert" {
				return nil, errors.New("alert service unavailable")
			}
			return &repository.ObjectInstance{
				ObjectType: objectType,
				ID:         objectID,
				Properties: map[string]interface{}{"name": "Test"},
			}, nil
		},
	}

	builder := NewContextBuilderV2(caseSvc, ontologyRepo, nil, nil, nil, testActionTypes)
	decisionCtx, err := builder.BuildDecisionContext(context.Background(), "dc-1")

	assert.Error(t, err)
	assert.Nil(t, decisionCtx)
	assert.Contains(t, err.Error(), "fetch alert")
}

func TestContextBuilderV2_BuildDecisionContext_GovernanceData(t *testing.T) {
	caseSvc := &mockDecisionCaseDataProvider{
		getCaseByIDFn: func(ctx context.Context, caseID string) (*decisionRepo.DecisionCaseRow, error) {
			severity := "medium"
			objectType := "seller"
			objectID := "seller-1"
			return &decisionRepo.DecisionCaseRow{
				CaseID:     "dc-1",
				SourceType: strPtr("alert"),
				SourceID:   strPtr("alert-1"),
				Severity:   &severity,
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
				Properties: map[string]interface{}{
					"name":  "Test",
					"email": "test@test.com",
				},
			}, nil
		},
	}

	markingSvc := &mockMarkingService{
		getFieldMarkingFn: func(ctx context.Context, objectType, field string) (*governance.FieldMarking, error) {
			if field == "email" {
				return &governance.FieldMarking{Classification: "L3", PII: true}, nil
			}
			return &governance.FieldMarking{Classification: "L2", PII: false}, nil
		},
	}

	builder := NewContextBuilderV2(caseSvc, ontologyRepo, markingSvc, nil, nil, testActionTypes)
	decisionCtx, err := builder.BuildDecisionContext(context.Background(), "dc-1")

	assert.NoError(t, err)
	assert.NotNil(t, decisionCtx)

	assert.Equal(t, "L3", decisionCtx.Governance.Classification)
	assert.True(t, decisionCtx.Governance.RedactionApplied)
	assert.Equal(t, "agent_readonly", decisionCtx.Governance.Role)
	assert.Contains(t, decisionCtx.Governance.RedactedFields, "email")

	assert.Equal(t, []string{"create_followup_task", "notify_owner", "export_report", "escalate_to_human"}, decisionCtx.AllowedActions)
	assert.Equal(t, []string{"execute", "apply", "dispatch"}, decisionCtx.ForbiddenActions)
}

func TestContextBuilderV2_BuildDecisionContext_MarkingFallback(t *testing.T) {
	caseSvc := &mockDecisionCaseDataProvider{
		getCaseByIDFn: func(ctx context.Context, caseID string) (*decisionRepo.DecisionCaseRow, error) {
			objectType := "seller"
			objectID := "seller-1"
			return &decisionRepo.DecisionCaseRow{
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
				Properties: map[string]interface{}{
					"name":  "Test",
					"email": "test@test.com",
				},
			}, nil
		},
	}

	markingSvc := &mockMarkingService{
		getFieldMarkingFn: func(ctx context.Context, objectType, field string) (*governance.FieldMarking, error) {
			return nil, errors.New("marking not found")
		},
	}

	builder := NewContextBuilderV2(caseSvc, ontologyRepo, markingSvc, nil, nil, testActionTypes)
	decisionCtx, err := builder.BuildDecisionContext(context.Background(), "dc-1")

	assert.NoError(t, err)
	assert.NotNil(t, decisionCtx)
	assert.Equal(t, "L2", decisionCtx.Governance.Classification)
}

func TestSwitchableContextBuilder_DelegatesToOldByDefault(t *testing.T) {
	caseSvc := &mockDecisionCaseDataProvider{
		getCaseByIDFn: func(ctx context.Context, caseID string) (*decisionRepo.DecisionCaseRow, error) {
			objectType := "seller"
			objectID := "seller-1"
			return &decisionRepo.DecisionCaseRow{
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

	oldBuilder := NewContextBuilder(caseSvc, objectProvider, classProvider, testActionTypes)
	newBuilder := NewContextBuilderV2(nil, nil, nil, nil, nil, testActionTypes)

	flags := &feature.FeatureFlags{NewContextBuilder: false}
	switcher := NewSwitchableContextBuilder(oldBuilder, newBuilder, flags)

	decisionCtx, err := switcher.BuildDecisionContext(context.Background(), "dc-1")

	assert.NoError(t, err)
	assert.NotNil(t, decisionCtx)
	assert.Equal(t, "Old Builder", decisionCtx.ObjectContext.Properties["name"])
}

func TestSwitchableContextBuilder_DelegatesToNewWhenFlagOn(t *testing.T) {
	caseSvc := &mockDecisionCaseDataProvider{
		getCaseByIDFn: func(ctx context.Context, caseID string) (*decisionRepo.DecisionCaseRow, error) {
			objectType := "seller"
			objectID := "seller-1"
			return &decisionRepo.DecisionCaseRow{
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
				Properties: map[string]interface{}{"name": "New Builder"},
			}, nil
		},
	}

	markingSvc := &mockMarkingService{
		getFieldMarkingFn: func(ctx context.Context, objectType, field string) (*governance.FieldMarking, error) {
			return &governance.FieldMarking{Classification: "L1", PII: false}, nil
		},
	}

	oldBuilder := NewContextBuilder(nil, nil, nil, testActionTypes)
	newBuilder := NewContextBuilderV2(caseSvc, ontologyRepo, markingSvc, nil, nil, testActionTypes)

	flags := &feature.FeatureFlags{NewContextBuilder: true}
	switcher := NewSwitchableContextBuilder(oldBuilder, newBuilder, flags)

	decisionCtx, err := switcher.BuildDecisionContext(context.Background(), "dc-1")

	assert.NoError(t, err)
	assert.NotNil(t, decisionCtx)
	assert.Equal(t, "New Builder", decisionCtx.ObjectContext.Properties["name"])
}

func TestSwitchableContextBuilder_NilFlagsDelegatesToOld(t *testing.T) {
	caseSvc := &mockDecisionCaseDataProvider{
		getCaseByIDFn: func(ctx context.Context, caseID string) (*decisionRepo.DecisionCaseRow, error) {
			objectType := "seller"
			objectID := "seller-1"
			return &decisionRepo.DecisionCaseRow{
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

	oldBuilder := NewContextBuilder(caseSvc, objectProvider, classProvider, testActionTypes)
	switcher := NewSwitchableContextBuilder(oldBuilder, nil, nil)

	decisionCtx, err := switcher.BuildDecisionContext(context.Background(), "dc-1")

	assert.NoError(t, err)
	assert.NotNil(t, decisionCtx)
	assert.Equal(t, "Old Builder", decisionCtx.ObjectContext.Properties["name"])
}
