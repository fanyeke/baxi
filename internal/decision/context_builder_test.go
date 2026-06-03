package decision

import (
	"context"
	"errors"
	"testing"

	"baxi/internal/governance"
	"baxi/internal/ontology"
	"baxi/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

// --- Mocks ---

type mockDecisionCaseDataProvider struct {
	getCaseByIDFn     func(ctx context.Context, pool *pgxpool.Pool, caseID string) (*repository.DecisionCaseRow, error)
	getCaseBySourceFn func(ctx context.Context, pool *pgxpool.Pool, sourceType, sourceID string) (*repository.DecisionCaseRow, error)
}

func (m *mockDecisionCaseDataProvider) GetCaseByID(ctx context.Context, pool *pgxpool.Pool, caseID string) (*repository.DecisionCaseRow, error) {
	return m.getCaseByIDFn(ctx, pool, caseID)
}

func (m *mockDecisionCaseDataProvider) GetCaseBySource(ctx context.Context, pool *pgxpool.Pool, sourceType, sourceID string) (*repository.DecisionCaseRow, error) {
	return m.getCaseBySourceFn(ctx, pool, sourceType, sourceID)
}

type mockObjectDataProvider struct {
	buildObjectContextFn func(ctx context.Context, objectType, objectID string) (*ontology.ObjectContext, error)
	getMetricAlertFn     func(ctx context.Context, alertID string) (*repository.ObjectInstance, error)
}

func (m *mockObjectDataProvider) BuildObjectContext(ctx context.Context, objectType, objectID string) (*ontology.ObjectContext, error) {
	return m.buildObjectContextFn(ctx, objectType, objectID)
}

func (m *mockObjectDataProvider) GetMetricAlert(ctx context.Context, alertID string) (*repository.ObjectInstance, error) {
	return m.getMetricAlertFn(ctx, alertID)
}

type mockClassificationProvider struct {
	getFieldMarkingFn func(ctx context.Context, objectType, property string) (string, bool, bool, error)
}

func (m *mockClassificationProvider) GetFieldMarking(ctx context.Context, objectType, property string) (string, bool, bool, error) {
	return m.getFieldMarkingFn(ctx, objectType, property)
}

// --- Tests ---

func TestContextBuilder_BuildDecisionContext_WithTriggerData(t *testing.T) {
	caseSvc := &mockDecisionCaseDataProvider{
		getCaseByIDFn: func(ctx context.Context, pool *pgxpool.Pool, caseID string) (*repository.DecisionCaseRow, error) {
			alertID := "alert-1"
			severity := "high"
			objectType := "seller"
			objectID := "seller-42"
			return &repository.DecisionCaseRow{
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

	objectProvider := &mockObjectDataProvider{
		getMetricAlertFn: func(ctx context.Context, alertID string) (*repository.ObjectInstance, error) {
			return &repository.ObjectInstance{
				ObjectType: "metric_alert",
				ID:         alertID,
				Properties: map[string]interface{}{
					"rule_id":        "rule-1",
					"metric_name":    "gmv_drop",
					"current_value":  100.0,
					"baseline_value": 150.0,
					"delta_pct":      -33.3,
				},
			}, nil
		},
		buildObjectContextFn: func(ctx context.Context, objectType, objectID string) (*ontology.ObjectContext, error) {
			return &ontology.ObjectContext{
				ObjectType: objectType,
				ObjectID:   objectID,
				Properties: map[string]interface{}{
					"name":  "Test Seller",
					"state": "SP",
				},
			}, nil
		},
	}

	classProvider := &mockClassificationProvider{
		getFieldMarkingFn: func(ctx context.Context, objectType, property string) (string, bool, bool, error) {
			return "L1", false, true, nil
		},
	}

	builder := NewContextBuilder(caseSvc, objectProvider, classProvider, nil)
	decisionCtx, err := builder.BuildDecisionContext(context.Background(), "dc-1")

	assert.NoError(t, err)
	assert.NotNil(t, decisionCtx)
	assert.Equal(t, "dc-1", decisionCtx.DecisionCaseID)
	assert.Equal(t, "alert", *decisionCtx.SourceType)
	assert.Equal(t, "alert-1", *decisionCtx.SourceID)
	assert.Equal(t, "alert-1", decisionCtx.Trigger.AlertID)
	assert.Equal(t, "rule-1", decisionCtx.Trigger.RuleID)
	assert.Equal(t, "high", decisionCtx.Trigger.Severity)
	assert.Equal(t, "gmv_drop", decisionCtx.Trigger.MetricName)
	assert.Equal(t, 100.0, decisionCtx.Trigger.CurrentValue)
	assert.Equal(t, 150.0, decisionCtx.Trigger.BaselineValue)
	assert.InDelta(t, -33.3, decisionCtx.Trigger.DeltaPct, 0.1)
}

func TestContextBuilder_BuildDecisionContext_AppliesRedaction(t *testing.T) {
	caseSvc := &mockDecisionCaseDataProvider{
		getCaseByIDFn: func(ctx context.Context, pool *pgxpool.Pool, caseID string) (*repository.DecisionCaseRow, error) {
			alertID := "alert-1"
			severity := "high"
			objectType := "seller"
			objectID := "seller-42"
			return &repository.DecisionCaseRow{
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

	objectProvider := &mockObjectDataProvider{
		getMetricAlertFn: func(ctx context.Context, alertID string) (*repository.ObjectInstance, error) {
			return &repository.ObjectInstance{
				ObjectType: "metric_alert",
				ID:         alertID,
				Properties: map[string]interface{}{
					"rule_id": "rule-1",
				},
			}, nil
		},
		buildObjectContextFn: func(ctx context.Context, objectType, objectID string) (*ontology.ObjectContext, error) {
			return &ontology.ObjectContext{
				ObjectType: objectType,
				ObjectID:   objectID,
				Properties: map[string]interface{}{
					"name":    "Test Seller",
					"email":   "seller@example.com",
					"revenue": 10000.0,
					"state":   "SP",
				},
			}, nil
		},
	}

	classProvider := &mockClassificationProvider{
		getFieldMarkingFn: func(ctx context.Context, objectType, property string) (string, bool, bool, error) {
			switch property {
			case "email":
				return "L3", true, false, nil
			case "revenue":
				return "L3", false, false, nil
			case "name", "state":
				return "L2", false, true, nil
			default:
				return "L1", false, true, nil
			}
		},
	}

	builder := NewContextBuilder(caseSvc, objectProvider, classProvider, nil)
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

func TestContextBuilder_BuildDecisionContext_GovernanceData(t *testing.T) {
	caseSvc := &mockDecisionCaseDataProvider{
		getCaseByIDFn: func(ctx context.Context, pool *pgxpool.Pool, caseID string) (*repository.DecisionCaseRow, error) {
			alertID := "alert-1"
			severity := "medium"
			objectType := "seller"
			objectID := "seller-1"
			return &repository.DecisionCaseRow{
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

	objectProvider := &mockObjectDataProvider{
		getMetricAlertFn: func(ctx context.Context, alertID string) (*repository.ObjectInstance, error) {
			return &repository.ObjectInstance{
				ObjectType: "metric_alert",
				ID:         alertID,
				Properties: map[string]interface{}{},
			}, nil
		},
		buildObjectContextFn: func(ctx context.Context, objectType, objectID string) (*ontology.ObjectContext, error) {
			return &ontology.ObjectContext{
				ObjectType: objectType,
				ObjectID:   objectID,
				Properties: map[string]interface{}{
					"name":  "Test",
					"email": "test@test.com",
				},
			}, nil
		},
	}

	classProvider := &mockClassificationProvider{
		getFieldMarkingFn: func(ctx context.Context, objectType, property string) (string, bool, bool, error) {
			if property == "email" {
				return "L3", true, false, nil
			}
			return "L2", false, true, nil
		},
	}

	builder := NewContextBuilder(caseSvc, objectProvider, classProvider, nil)
	decisionCtx, err := builder.BuildDecisionContext(context.Background(), "dc-1")

	assert.NoError(t, err)
	assert.NotNil(t, decisionCtx)

	assert.Equal(t, "L3", decisionCtx.Governance.Classification)
	assert.True(t, decisionCtx.Governance.RedactionApplied)
	assert.Equal(t, "agent_readonly", decisionCtx.Governance.Role)
	assert.Contains(t, decisionCtx.Governance.RedactedFields, "email")

	assert.Equal(t, []string{"create_followup_task", "notify_owner", "export_report", "escalate_to_human"}, decisionCtx.AllowedActions)
	assert.Equal(t, []string{"execute_dispatch", "modify_raw_data", "write_dwd", "write_mart"}, decisionCtx.ForbiddenActions)
}

func TestContextBuilder_BuildLLMSafeContext_GeneratesHash(t *testing.T) {
	decisionCtx := &DecisionContext{
		DecisionCaseID: "dc-1",
		SourceType:     strPtr("alert"),
		SourceID:       strPtr("alert-1"),
		Trigger: TriggerInfo{
			AlertID:  "alert-1",
			Severity: "high",
		},
		ObjectContext: ObjectContextData{
			ObjectType: "seller",
			ObjectID:   "seller-1",
			Properties: map[string]interface{}{
				"name": "Test Seller",
			},
		},
		Governance: GovernanceData{
			Classification:   "L2",
			RedactionApplied: false,
			Role:             "agent_readonly",
		},
		AllowedActions:   []string{"create_followup_task", "notify_owner", "export_report", "escalate_to_human"},
		ForbiddenActions: []string{"execute_dispatch", "modify_raw_data", "write_dwd", "write_mart"},
	}

	builder := NewContextBuilder(nil, nil, nil, nil)
	llmCtx, err := builder.BuildLLMSafeContext(context.Background(), decisionCtx)

	assert.NoError(t, err)
	assert.NotNil(t, llmCtx)
	assert.Equal(t, "dc-1", llmCtx.CaseID)
	assert.NotEmpty(t, llmCtx.ContextHash)
	assert.Len(t, llmCtx.ContextHash, 64)
}

func TestContextBuilder_BuildDecisionContext_RedactedFieldsInGovernance(t *testing.T) {
	caseSvc := &mockDecisionCaseDataProvider{
		getCaseByIDFn: func(ctx context.Context, pool *pgxpool.Pool, caseID string) (*repository.DecisionCaseRow, error) {
			alertID := "alert-1"
			severity := "high"
			objectType := "seller"
			objectID := "seller-1"
			return &repository.DecisionCaseRow{
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

	objectProvider := &mockObjectDataProvider{
		getMetricAlertFn: func(ctx context.Context, alertID string) (*repository.ObjectInstance, error) {
			return &repository.ObjectInstance{
				ObjectType: "metric_alert",
				ID:         alertID,
				Properties: map[string]interface{}{},
			}, nil
		},
		buildObjectContextFn: func(ctx context.Context, objectType, objectID string) (*ontology.ObjectContext, error) {
			return &ontology.ObjectContext{
				ObjectType: objectType,
				ObjectID:   objectID,
				Properties: map[string]interface{}{
					"name":      "Test",
					"email":     "test@test.com",
					"phone":     "1234567890",
					"revenue":   10000.0,
					"public_id": "pub-123",
				},
			}, nil
		},
	}

	classProvider := &mockClassificationProvider{
		getFieldMarkingFn: func(ctx context.Context, objectType, property string) (string, bool, bool, error) {
			switch property {
			case "email", "phone":
				return "L3", true, false, nil
			case "revenue":
				return "L3", false, false, nil
			case "public_id":
				return "L1", false, true, nil
			default:
				return "L2", false, true, nil
			}
		},
	}

	builder := NewContextBuilder(caseSvc, objectProvider, classProvider, nil)
	decisionCtx, err := builder.BuildDecisionContext(context.Background(), "dc-1")

	assert.NoError(t, err)
	assert.NotNil(t, decisionCtx)

	assert.Len(t, decisionCtx.Governance.RedactedFields, 3)
	assert.Contains(t, decisionCtx.Governance.RedactedFields, "email")
	assert.Contains(t, decisionCtx.Governance.RedactedFields, "phone")
	assert.Contains(t, decisionCtx.Governance.RedactedFields, "revenue")
	assert.NotContains(t, decisionCtx.Governance.RedactedFields, "name")
	assert.NotContains(t, decisionCtx.Governance.RedactedFields, "public_id")
}

func TestComputeContextHash(t *testing.T) {
	data := map[string]string{"key": "value"}
	hash1, err := ComputeContextHash(data)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash1)
	assert.Len(t, hash1, 64)

	hash2, err := ComputeContextHash(data)
	assert.NoError(t, err)
	assert.Equal(t, hash1, hash2)

	hash3, err := ComputeContextHash(map[string]string{"key": "different"})
	assert.NoError(t, err)
	assert.NotEqual(t, hash1, hash3)
}

func TestContextBuilder_BuildDecisionContext_CaseNotFound(t *testing.T) {
	caseSvc := &mockDecisionCaseDataProvider{
		getCaseByIDFn: func(ctx context.Context, pool *pgxpool.Pool, caseID string) (*repository.DecisionCaseRow, error) {
			return nil, errors.New("case not found")
		},
	}

	builder := NewContextBuilder(caseSvc, nil, nil, nil)
	decisionCtx, err := builder.BuildDecisionContext(context.Background(), "dc-missing")

	assert.Error(t, err)
	assert.Nil(t, decisionCtx)
	assert.Contains(t, err.Error(), "fetch case")
}

func TestContextBuilder_BuildDecisionContext_NoAlertID(t *testing.T) {
	caseSvc := &mockDecisionCaseDataProvider{
		getCaseByIDFn: func(ctx context.Context, pool *pgxpool.Pool, caseID string) (*repository.DecisionCaseRow, error) {
			severity := "medium"
			objectType := "seller"
			objectID := "seller-1"
			return &repository.DecisionCaseRow{
				CaseID:     "dc-1",
				SourceType: strPtr("manual"),
				SourceID:   strPtr("manual-1"),
				Severity:   &severity,
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
				Properties: map[string]interface{}{
					"name": "Test",
				},
			}, nil
		},
	}

	classProvider := &mockClassificationProvider{
		getFieldMarkingFn: func(ctx context.Context, objectType, property string) (string, bool, bool, error) {
			return "L1", false, true, nil
		},
	}

	builder := NewContextBuilder(caseSvc, objectProvider, classProvider, nil)
	decisionCtx, err := builder.BuildDecisionContext(context.Background(), "dc-1")

	assert.NoError(t, err)
	assert.NotNil(t, decisionCtx)
	assert.Empty(t, decisionCtx.Trigger.AlertID)
	assert.Equal(t, "medium", decisionCtx.Trigger.Severity)
}

func TestContextBuilder_BuildDecisionContext_AlertError(t *testing.T) {
	caseSvc := &mockDecisionCaseDataProvider{
		getCaseByIDFn: func(ctx context.Context, pool *pgxpool.Pool, caseID string) (*repository.DecisionCaseRow, error) {
			alertID := "alert-1"
			severity := "high"
			objectType := "seller"
			objectID := "seller-1"
			return &repository.DecisionCaseRow{
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

	objectProvider := &mockObjectDataProvider{
		getMetricAlertFn: func(ctx context.Context, alertID string) (*repository.ObjectInstance, error) {
			return nil, errors.New("alert service unavailable")
		},
	}

	builder := NewContextBuilder(caseSvc, objectProvider, nil, nil)
	decisionCtx, err := builder.BuildDecisionContext(context.Background(), "dc-1")

	assert.Error(t, err)
	assert.Nil(t, decisionCtx)
	assert.Contains(t, err.Error(), "fetch alert")
}

func TestMapClassification(t *testing.T) {
	assert.Equal(t, "pii", mapClassification("L3", true))
	assert.Equal(t, "sensitive", mapClassification("L3", false))
	assert.Equal(t, "internal", mapClassification("L2", false))
	assert.Equal(t, "public_internal", mapClassification("L1", false))
	assert.Equal(t, "internal", mapClassification("unknown", false))
}

func TestResolveOverallClassification(t *testing.T) {
	assert.Equal(t, "L1", resolveOverallClassification(map[string]string{}))
	assert.Equal(t, "L2", resolveOverallClassification(map[string]string{"f1": "internal"}))
	assert.Equal(t, "L3", resolveOverallClassification(map[string]string{"f1": "pii"}))
	assert.Equal(t, "L3", resolveOverallClassification(map[string]string{"f1": "sensitive"}))
	assert.Equal(t, "L3", resolveOverallClassification(map[string]string{"f1": "internal", "f2": "pii"}))
}

func TestRedactObjectContext_Integration(t *testing.T) {
	properties := map[string]interface{}{
		"name":    "Test",
		"email":   "test@test.com",
		"revenue": 10000.0,
		"state":   "SP",
	}
	classifications := map[string]string{
		"email":   "pii",
		"revenue": "sensitive",
		"state":   "internal",
	}
	markings := map[string]string{}
	policy := governance.RedactionPolicy{Role: "agent_readonly"}

	result := governance.RedactObjectContext(properties, classifications, markings, policy)

	assert.NotNil(t, result.Properties["name"])
	assert.NotNil(t, result.Properties["state"])
	assert.Nil(t, result.Properties["email"])
	assert.Nil(t, result.Properties["revenue"])

	assert.Len(t, result.RedactedFields, 2)
	redactedNames := make([]string, len(result.RedactedFields))
	for i, rf := range result.RedactedFields {
		redactedNames[i] = rf.Field
	}
	assert.Contains(t, redactedNames, "email")
	assert.Contains(t, redactedNames, "revenue")
}
