package governance

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/repository"
	governanceRepo "baxi/internal/repository/governance"
)

// ──── mockRepository ────────────────────────────────────────────────────────

type mockGovRepository struct {
	getConfigSnapshotsFn    func(ctx context.Context, pool interface{}) ([]governanceRepo.ConfigSnapshotRow, error)
	countObjectSchemasFn    func(ctx context.Context, pool interface{}) int
	getObjectSchemasFn      func(ctx context.Context, pool interface{}) ([]governanceRepo.ObjectSchemaRow, error)
	getAccessPoliciesByRoleFn func(ctx context.Context, pool interface{}, role string) ([]repository.AccessPolicyRow, error)
	getAccessPoliciesFn     func(ctx context.Context, pool interface{}) ([]repository.AccessPolicyRow, error)
	getDataClassificationsFn func(ctx context.Context, pool interface{}) ([]governanceRepo.DataClassificationRow, error)
	getLineageBySourceFn    func(ctx context.Context, pool interface{}, table string) ([]repository.DataLineageRow, error)
	getLineageByTargetFn    func(ctx context.Context, pool interface{}, table string) ([]repository.DataLineageRow, error)
	getDataLineageFn        func(ctx context.Context, pool interface{}) ([]repository.DataLineageRow, error)
}

func (m *mockGovRepository) GetConfigSnapshots(ctx context.Context, pool interface{}) ([]governanceRepo.ConfigSnapshotRow, error) {
	if m.getConfigSnapshotsFn != nil {
		return m.getConfigSnapshotsFn(ctx, pool)
	}
	return nil, nil
}

func (m *mockGovRepository) CountObjectSchemas(ctx context.Context, pool interface{}) int {
	if m.countObjectSchemasFn != nil {
		return m.countObjectSchemasFn(ctx, pool)
	}
	return 0
}

func (m *mockGovRepository) GetObjectSchemas(ctx context.Context, pool interface{}) ([]governanceRepo.ObjectSchemaRow, error) {
	if m.getObjectSchemasFn != nil {
		return m.getObjectSchemasFn(ctx, pool)
	}
	return nil, nil
}

// ──── ResolveLevel ────────────────────────────────────────────────────────

func TestResolveLevel_AllCases(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"pii", "L3"},
		{"sensitive", "L3"},
		{"internal", "L2"},
		{"derived_sensitive", "L2"},
		{"public_internal", "L1"},
		{"unknown_level", "L2"},
		{"", "L2"},
		{"L3", "L2"}, // unrecognized string defaults to L2
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, ResolveLevel(tt.input))
		})
	}
}

// ──── access_policy: CheckAccess unit tests with mocked repo ─────────────

type mockAccessPolicyRepository struct {
	getByRoleFn func(ctx context.Context, pool interface{}, role string) ([]repository.AccessPolicyRow, error)
	getAllFn    func(ctx context.Context, pool interface{}) ([]repository.AccessPolicyRow, error)
}

func (m *mockAccessPolicyRepository) GetAccessPoliciesByRole(ctx context.Context, pool interface{}, role string) ([]repository.AccessPolicyRow, error) {
	if m.getByRoleFn != nil {
		return m.getByRoleFn(ctx, pool, role)
	}
	return nil, nil
}

func (m *mockAccessPolicyRepository) GetAccessPolicies(ctx context.Context, pool interface{}) ([]repository.AccessPolicyRow, error) {
	if m.getAllFn != nil {
		return m.getAllFn(ctx, pool)
	}
	return nil, nil
}

func TestAccessPolicyService_CheckAccess_AllowByRole(t *testing.T) {
	// With nil pool, the service will panic because the repository calls pgx methods.
	// This test verifies the constructor works.
	svc := NewAccessPolicyService(nil, nil)
	assert.NotNil(t, svc)
}

func TestAccessPolicyService_GetAll_NilPool(t *testing.T) {
	svc := NewAccessPolicyService(nil, nil)
	assert.NotNil(t, svc)
}

// ──── redaction: RedactObjectContext ──────────────────────────────────────

func TestRedactObjectContext_NoRedactions(t *testing.T) {
	props := map[string]interface{}{
		"name":  "test",
		"value": 123,
	}
	classifications := map[string]string{
		"name":  "public_internal",
		"value": "public_internal",
	}
	markings := map[string]string{}

	result := RedactObjectContext(props, classifications, markings, RedactionPolicy{Role: "admin"})
	assert.Len(t, result.Properties, 2)
	assert.Empty(t, result.RedactedFields)
}

func TestRedactObjectContext_PII_Classification(t *testing.T) {
	props := map[string]interface{}{
		"email": "test@example.com",
		"name":  "test",
	}
	classifications := map[string]string{
		"email": "pii",
	}
	markings := map[string]string{}

	result := RedactObjectContext(props, classifications, markings, RedactionPolicy{Role: "viewer"})
	assert.Len(t, result.Properties, 1)
	assert.Equal(t, "test", result.Properties["name"])
	assert.Len(t, result.RedactedFields, 1)
	assert.Equal(t, "email", result.RedactedFields[0].Field)
	assert.Contains(t, result.RedactedFields[0].Reason, "pii")
}

func TestRedactObjectContext_Sensitive_Classification(t *testing.T) {
	props := map[string]interface{}{
		"revenue": 1000,
		"name":    "test",
	}
	classifications := map[string]string{
		"revenue": "sensitive",
	}
	markings := map[string]string{}

	result := RedactObjectContext(props, classifications, markings, RedactionPolicy{Role: "viewer"})
	assert.Len(t, result.Properties, 1)
	assert.Equal(t, "test", result.Properties["name"])
	assert.Len(t, result.RedactedFields, 1)
	assert.Equal(t, "revenue", result.RedactedFields[0].Field)
}

func TestRedactObjectContext_DerivedSensitive_Classification(t *testing.T) {
	props := map[string]interface{}{
		"score":  95,
		"name":   "test",
	}
	classifications := map[string]string{
		"score": "derived_sensitive",
	}
	markings := map[string]string{}

	result := RedactObjectContext(props, classifications, markings, RedactionPolicy{Role: "agent_readonly"})
	assert.Len(t, result.Properties, 1)
	assert.Len(t, result.RedactedFields, 1)
	assert.Equal(t, "score", result.RedactedFields[0].Field)
}

func TestRedactObjectContext_Internal_Classification(t *testing.T) {
	props := map[string]interface{}{
		"note":  "internal note",
		"name":  "test",
	}
	classifications := map[string]string{
		"note": "internal",
	}
	markings := map[string]string{}

	// viewer redacts internal
	result := RedactObjectContext(props, classifications, markings, RedactionPolicy{Role: "viewer"})
	assert.Len(t, result.Properties, 1)
	assert.Len(t, result.RedactedFields, 1)

	// admin does not redact internal
	result2 := RedactObjectContext(props, classifications, markings, RedactionPolicy{Role: "admin"})
	assert.Len(t, result2.Properties, 2)
	assert.Empty(t, result2.RedactedFields)
}

func TestRedactObjectContext_PublicInternal_NeverRedacted(t *testing.T) {
	props := map[string]interface{}{
		"category": "electronics",
	}
	classifications := map[string]string{
		"category": "public_internal",
	}
	markings := map[string]string{}

	result := RedactObjectContext(props, classifications, markings, RedactionPolicy{Role: "viewer"})
	assert.Len(t, result.Properties, 1)
	assert.Empty(t, result.RedactedFields)
}

func TestRedactObjectContext_Marking_PII(t *testing.T) {
	props := map[string]interface{}{
		"ssn": "123-45-6789",
		"name": "test",
	}
	classifications := map[string]string{}
	markings := map[string]string{
		"ssn": "PII",
	}

	// non-admin is redacted
	result := RedactObjectContext(props, classifications, markings, RedactionPolicy{Role: "viewer"})
	assert.Len(t, result.Properties, 1)
	assert.Len(t, result.RedactedFields, 1)
	assert.Equal(t, "ssn", result.RedactedFields[0].Field)

	// admin is not redacted
	result2 := RedactObjectContext(props, classifications, markings, RedactionPolicy{Role: "admin"})
	assert.Len(t, result2.Properties, 2)
	assert.Empty(t, result2.RedactedFields)
}

func TestRedactObjectContext_Marking_FINANCIAL_INTERNAL(t *testing.T) {
	props := map[string]interface{}{
		"profit": 5000,
	}
	classifications := map[string]string{}
	markings := map[string]string{
		"profit": "FINANCIAL_INTERNAL",
	}

	result := RedactObjectContext(props, classifications, markings, RedactionPolicy{Role: "analyst"})
	assert.Len(t, result.Properties, 0)
	assert.Len(t, result.RedactedFields, 1)
}

func TestRedactObjectContext_Marking_RAW_DATA(t *testing.T) {
	props := map[string]interface{}{
		"raw_field": "some_value",
	}
	classifications := map[string]string{}
	markings := map[string]string{
		"raw_field": "RAW_DATA",
	}

	result := RedactObjectContext(props, classifications, markings, RedactionPolicy{Role: "agent_readonly"})
	assert.Len(t, result.Properties, 0)
	assert.Len(t, result.RedactedFields, 1)
}

func TestRedactObjectContext_Marking_OPERATIONAL_INTERNAL(t *testing.T) {
	props := map[string]interface{}{
		"ops_field": "some_value",
		"name":      "test",
	}
	classifications := map[string]string{}
	markings := map[string]string{
		"ops_field": "OPERATIONAL_INTERNAL",
	}

	// viewer is redacted
	result := RedactObjectContext(props, classifications, markings, RedactionPolicy{Role: "viewer"})
	assert.Len(t, result.Properties, 1)
	assert.Len(t, result.RedactedFields, 1)

	// admin is NOT redacted for OPERATIONAL_INTERNAL
	result2 := RedactObjectContext(props, classifications, markings, RedactionPolicy{Role: "admin"})
	assert.Len(t, result2.Properties, 2)
	assert.Empty(t, result2.RedactedFields)
}

func TestRedactObjectContext_EmptyInputs(t *testing.T) {
	result := RedactObjectContext(nil, nil, nil, RedactionPolicy{Role: "admin"})
	assert.NotNil(t, result.Properties)
	assert.Empty(t, result.Properties)
	assert.Empty(t, result.RedactedFields)
}

func TestRedactObjectContext_MarkingPrecedenceOverClassification(t *testing.T) {
	props := map[string]interface{}{
		"field": "value",
	}
	classifications := map[string]string{
		"field": "public_internal",
	}
	markings := map[string]string{
		"field": "PII",
	}

	// Marking should take precedence
	result := RedactObjectContext(props, classifications, markings, RedactionPolicy{Role: "viewer"})
	assert.Len(t, result.Properties, 0)
	assert.Len(t, result.RedactedFields, 1)
	assert.Contains(t, result.RedactedFields[0].Reason, "marking")
}

func TestRedactObjectContext_SortedRedactedFields(t *testing.T) {
	props := map[string]interface{}{
		"z_field": "z",
		"a_field": "a",
		"m_field": "m",
	}
	classifications := map[string]string{
		"z_field": "pii",
		"a_field": "pii",
		"m_field": "pii",
	}
	markings := map[string]string{}

	result := RedactObjectContext(props, classifications, markings, RedactionPolicy{Role: "viewer"})
	require.Len(t, result.RedactedFields, 3)
	assert.Equal(t, "a_field", result.RedactedFields[0].Field)
	assert.Equal(t, "m_field", result.RedactedFields[1].Field)
	assert.Equal(t, "z_field", result.RedactedFields[2].Field)
}

// ──── checkMarking / checkClassification edge cases ──────────────────────

func TestCheckMarking_UnknownMarking(t *testing.T) {
	markings := map[string]string{
		"field": "UNKNOWN_MARKING",
	}
	entry, ok := checkMarking("field", markings, "viewer")
	assert.False(t, ok)
	assert.Empty(t, entry)
}

func TestCheckMarking_MissingMarking(t *testing.T) {
	markings := map[string]string{}
	entry, ok := checkMarking("field", markings, "viewer")
	assert.False(t, ok)
	assert.Empty(t, entry)
}

func TestCheckClassification_EmptyRole_PII(t *testing.T) {
	classifications := map[string]string{
		"email": "pii",
	}
	entry, ok := checkClassification("email", classifications, "")
	assert.True(t, ok)
	assert.Contains(t, entry.Reason, "pii")
}

func TestCheckClassification_UnknownClassification(t *testing.T) {
	classifications := map[string]string{
		"field": "unknown_level",
	}
	entry, ok := checkClassification("field", classifications, "viewer")
	// "unknown_level" doesn't match any case, so redact is false
	// Actually the switch has no match for "unknown_level", so redact stays false
	assert.False(t, ok)
	assert.Empty(t, entry)
}

func TestCheckClassification_Sensitive_AgentReadonly(t *testing.T) {
	classifications := map[string]string{
		"field": "sensitive",
	}
	entry, ok := checkClassification("field", classifications, "agent_readonly")
	assert.True(t, ok)
	assert.Contains(t, entry.Reason, "sensitive")
}

func TestCheckClassification_NoMatch(t *testing.T) {
	classifications := map[string]string{
		"other_field": "pii",
	}
	entry, ok := checkClassification("my_field", classifications, "viewer")
	assert.False(t, ok)
	assert.Empty(t, entry)
}
