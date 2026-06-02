package governance

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedactObjectContext_AdminSeesAll(t *testing.T) {
	properties := map[string]interface{}{
		"email": "test@example.com",
		"name":  "John Doe",
	}
	classifications := map[string]string{
		"email": "pii",
	}
	markings := map[string]string{
		"email": "PII",
	}
	policy := RedactionPolicy{Role: "admin"}

	result := RedactObjectContext(properties, classifications, markings, policy)

	assert.Len(t, result.Properties, 2)
	assert.Contains(t, result.Properties, "email")
	assert.Contains(t, result.Properties, "name")
	assert.Empty(t, result.RedactedFields)
}

func TestRedactObjectContext_ViewerRedactsPII(t *testing.T) {
	properties := map[string]interface{}{
		"email":      "test@example.com",
		"name":       "John Doe",
		"order_id":   "12345",
		"gmv":        100.50,
	}
	classifications := map[string]string{
		"email": "pii",
		"name":  "sensitive",
		"gmv":   "public_internal",
	}
	markings := map[string]string{}
	policy := RedactionPolicy{Role: "viewer"}

	result := RedactObjectContext(properties, classifications, markings, policy)

	assert.Contains(t, result.Properties, "order_id")
	assert.Contains(t, result.Properties, "gmv")
	assert.NotContains(t, result.Properties, "email")
	assert.NotContains(t, result.Properties, "name")

	assert.Len(t, result.RedactedFields, 2)
	redactedFields := make(map[string]bool)
	for _, r := range result.RedactedFields {
		redactedFields[r.Field] = true
	}
	assert.True(t, redactedFields["email"])
	assert.True(t, redactedFields["name"])
}

func TestRedactObjectContext_EmptyProperties(t *testing.T) {
	result := RedactObjectContext(
		map[string]interface{}{},
		map[string]string{},
		map[string]string{},
		RedactionPolicy{Role: "admin"},
	)
	assert.Empty(t, result.Properties)
	assert.Empty(t, result.RedactedFields)
}

func TestRedactObjectContext_NilProperties(t *testing.T) {
	result := RedactObjectContext(nil, nil, nil, RedactionPolicy{Role: "admin"})
	assert.Empty(t, result.Properties)
	assert.Empty(t, result.RedactedFields)
}

func TestRedactObjectContext_ViewerRedactsInternal(t *testing.T) {
	properties := map[string]interface{}{
		"internal_field": "secret",
		"public_field":   "open",
	}
	classifications := map[string]string{
		"internal_field": "internal",
	}
	policy := RedactionPolicy{Role: "viewer"}

	result := RedactObjectContext(properties, classifications, nil, policy)

	assert.Contains(t, result.Properties, "public_field")
	assert.NotContains(t, result.Properties, "internal_field")
	assert.Len(t, result.RedactedFields, 1)
	assert.Equal(t, "internal_field", result.RedactedFields[0].Field)
	assert.Contains(t, result.RedactedFields[0].Reason, "classification: internal")
}

func TestRedactObjectContext_ViewerDoesNotRedactPublicInternal(t *testing.T) {
	properties := map[string]interface{}{
		"public_field": "open",
	}
	classifications := map[string]string{
		"public_field": "public_internal",
	}
	policy := RedactionPolicy{Role: "viewer"}

	result := RedactObjectContext(properties, classifications, nil, policy)

	assert.Contains(t, result.Properties, "public_field")
	assert.Empty(t, result.RedactedFields)
}

func TestRedactObjectContext_AgentReadonlyRedactsSensitive(t *testing.T) {
	properties := map[string]interface{}{
		"sensitive_field": "secret_val",
		"internal_field":  "internal_val",
		"public_field":   "open",
	}
	classifications := map[string]string{
		"sensitive_field": "sensitive",
		"internal_field":  "internal",
	}
	policy := RedactionPolicy{Role: "agent_readonly"}

	result := RedactObjectContext(properties, classifications, nil, policy)

	assert.Contains(t, result.Properties, "public_field")
	assert.Contains(t, result.Properties, "internal_field")
	assert.NotContains(t, result.Properties, "sensitive_field")
	assert.Len(t, result.RedactedFields, 1)
	assert.Equal(t, "sensitive_field", result.RedactedFields[0].Field)
}

func TestRedactObjectContext_MarkingTakesPriority(t *testing.T) {
	properties := map[string]interface{}{
		"field_a": "value_a",
	}
	classifications := map[string]string{
		"field_a": "public_internal",
	}
	markings := map[string]string{
		"field_a": "PII",
	}
	policy := RedactionPolicy{Role: "viewer"}

	result := RedactObjectContext(properties, classifications, markings, policy)

	assert.NotContains(t, result.Properties, "field_a")
	assert.Len(t, result.RedactedFields, 1)
	assert.Equal(t, "field_a", result.RedactedFields[0].Field)
	assert.Contains(t, result.RedactedFields[0].Reason, "marking: PII")
}

func TestRedactObjectContext_AdminSeesMarkedFields(t *testing.T) {
	properties := map[string]interface{}{
		"financial": "12345",
	}
	markings := map[string]string{
		"financial": "FINANCIAL_INTERNAL",
	}
	policy := RedactionPolicy{Role: "admin"}

	result := RedactObjectContext(properties, nil, markings, policy)

	assert.Contains(t, result.Properties, "financial")
	assert.Empty(t, result.RedactedFields)
}

func TestRedactObjectContext_ViewerRedactsOperationalInternal(t *testing.T) {
	properties := map[string]interface{}{
		"ops_field": "ops_data",
	}
	markings := map[string]string{
		"ops_field": "OPERATIONAL_INTERNAL",
	}
	policy := RedactionPolicy{Role: "viewer"}

	result := RedactObjectContext(properties, nil, markings, policy)

	assert.NotContains(t, result.Properties, "ops_field")
	assert.Len(t, result.RedactedFields, 1)
	assert.Equal(t, "ops_field", result.RedactedFields[0].Field)
	assert.Contains(t, result.RedactedFields[0].Reason, "marking: OPERATIONAL_INTERNAL")
}

func TestRedactObjectContext_NonViewerSeesOperationalInternal(t *testing.T) {
	properties := map[string]interface{}{
		"ops_field": "ops_data",
	}
	markings := map[string]string{
		"ops_field": "OPERATIONAL_INTERNAL",
	}
	policy := RedactionPolicy{Role: "admin"}

	result := RedactObjectContext(properties, nil, markings, policy)

	assert.Contains(t, result.Properties, "ops_field")
	assert.Empty(t, result.RedactedFields)
}

func TestRedactObjectContext_RedactedFieldsSorted(t *testing.T) {
	properties := map[string]interface{}{
		"z_field": "z_val",
		"a_field": "a_val",
		"m_field": "m_val",
	}
	classifications := map[string]string{
		"z_field": "pii",
		"a_field": "pii",
		"m_field": "pii",
	}
	policy := RedactionPolicy{Role: "viewer"}

	result := RedactObjectContext(properties, classifications, nil, policy)

	assert.Len(t, result.RedactedFields, 3)
	assert.Equal(t, "a_field", result.RedactedFields[0].Field)
	assert.Equal(t, "m_field", result.RedactedFields[1].Field)
	assert.Equal(t, "z_field", result.RedactedFields[2].Field)
}

func TestRedactObjectContext_PublicFieldPassesThrough(t *testing.T) {
	properties := map[string]interface{}{
		"email": "test@example.com",
		"name":  "John",
	}
	classifications := map[string]string{
		"email": "pii",
	}
	policy := RedactionPolicy{Role: "viewer"}

	result := RedactObjectContext(properties, classifications, nil, policy)

	assert.NotContains(t, result.Properties, "email")
	assert.Contains(t, result.Properties, "name")
	assert.Equal(t, "John", result.Properties["name"])
}

func TestRedactObjectContext_DerivedSensitiveRedactedForViewer(t *testing.T) {
	properties := map[string]interface{}{
		"derived_field": "derived_val",
	}
	classifications := map[string]string{
		"derived_field": "derived_sensitive",
	}
	policy := RedactionPolicy{Role: "viewer"}

	result := RedactObjectContext(properties, classifications, nil, policy)

	assert.NotContains(t, result.Properties, "derived_field")
	assert.Len(t, result.RedactedFields, 1)
}

func TestRedactObjectContext_DerivedSensitiveNotRedactedForAdmin(t *testing.T) {
	properties := map[string]interface{}{
		"derived_field": "derived_val",
	}
	classifications := map[string]string{
		"derived_field": "derived_sensitive",
	}
	policy := RedactionPolicy{Role: "admin"}

	result := RedactObjectContext(properties, classifications, nil, policy)

	assert.Contains(t, result.Properties, "derived_field")
	assert.Empty(t, result.RedactedFields)
}

func TestRedactObjectContext_RawDataMarkingRedactedForNonAdmin(t *testing.T) {
	properties := map[string]interface{}{
		"raw_field": "raw_data",
	}
	markings := map[string]string{
		"raw_field": "RAW_DATA",
	}
	policy := RedactionPolicy{Role: "viewer"}

	result := RedactObjectContext(properties, nil, markings, policy)

	assert.NotContains(t, result.Properties, "raw_field")
	assert.Len(t, result.RedactedFields, 1)
	assert.Contains(t, result.RedactedFields[0].Reason, "marking: RAW_DATA")
}

func TestRedactObjectContext_RawDataMarkingNotRedactedForAdmin(t *testing.T) {
	properties := map[string]interface{}{
		"raw_field": "raw_data",
	}
	markings := map[string]string{
		"raw_field": "RAW_DATA",
	}
	policy := RedactionPolicy{Role: "admin"}

	result := RedactObjectContext(properties, nil, markings, policy)

	assert.Contains(t, result.Properties, "raw_field")
	assert.Empty(t, result.RedactedFields)
}

// ──── checkMarking ──────────────────────────────────────────────────────────

func TestCheckMarking_NoMarking(t *testing.T) {
	_, ok := checkMarking("field_x", map[string]string{}, "viewer")
	assert.False(t, ok)
}

func TestCheckMarking_NilMarkings(t *testing.T) {
	_, ok := checkMarking("field_x", nil, "viewer")
	assert.False(t, ok)
}

func TestCheckMarking_PIIRedactedForViewer(t *testing.T) {
	entry, ok := checkMarking("email", map[string]string{"email": "PII"}, "viewer")
	assert.True(t, ok)
	assert.Equal(t, "email", entry.Field)
	assert.Equal(t, "PII", entry.Rule)
}

func TestCheckMarking_PIINotRedactedForAdmin(t *testing.T) {
	_, ok := checkMarking("email", map[string]string{"email": "PII"}, "admin")
	assert.False(t, ok)
}

func TestCheckMarking_FinancialInternalRedactedForViewer(t *testing.T) {
	entry, ok := checkMarking("fin", map[string]string{"fin": "FINANCIAL_INTERNAL"}, "viewer")
	assert.True(t, ok)
	assert.Equal(t, "FINANCIAL_INTERNAL", entry.Rule)
}

func TestCheckMarking_FinancialInternalNotRedactedForAdmin(t *testing.T) {
	_, ok := checkMarking("fin", map[string]string{"fin": "FINANCIAL_INTERNAL"}, "admin")
	assert.False(t, ok)
}

// ──── checkClassification ──────────────────────────────────────────────────

func TestCheckClassification_NoClassification(t *testing.T) {
	_, ok := checkClassification("field_x", map[string]string{}, "viewer")
	assert.False(t, ok)
}

func TestCheckClassification_NilClassifications(t *testing.T) {
	_, ok := checkClassification("field_x", nil, "viewer")
	assert.False(t, ok)
}

func TestCheckClassification_PIIRedactedForViewerAndReadonly(t *testing.T) {
	for _, role := range []string{"viewer", "agent_readonly"} {
		t.Run(role, func(t *testing.T) {
			entry, ok := checkClassification("email", map[string]string{"email": "pii"}, role)
			assert.True(t, ok, "pii should be redacted for %s", role)
			assert.Equal(t, "email", entry.Field)
			assert.Equal(t, "pii", entry.Rule)
		})
	}
	t.Run("admin", func(t *testing.T) {
		_, ok := checkClassification("email", map[string]string{"email": "pii"}, "admin")
		assert.False(t, ok, "pii should not be redacted for admin")
	})
}

func TestCheckClassification_SensitiveRedactedForViewer(t *testing.T) {
	_, ok := checkClassification("s", map[string]string{"s": "sensitive"}, "viewer")
	assert.True(t, ok)
}

func TestCheckClassification_SensitiveNotRedactedForAdmin(t *testing.T) {
	_, ok := checkClassification("s", map[string]string{"s": "sensitive"}, "admin")
	assert.False(t, ok)
}

func TestCheckClassification_InternalRedactedForViewer(t *testing.T) {
	entry, ok := checkClassification("i", map[string]string{"i": "internal"}, "viewer")
	assert.True(t, ok)
	assert.Equal(t, "internal", entry.Rule)
}

func TestCheckClassification_InternalNotRedactedForAdmin(t *testing.T) {
	_, ok := checkClassification("i", map[string]string{"i": "internal"}, "admin")
	assert.False(t, ok)
}

func TestCheckClassification_PublicInternalNeverRedacted(t *testing.T) {
	for _, role := range []string{"viewer", "admin", "agent_readonly"} {
		t.Run(role, func(t *testing.T) {
			_, ok := checkClassification("p", map[string]string{"p": "public_internal"}, role)
			assert.False(t, ok)
		})
	}
}

func TestCheckClassification_UnknownLevelNotRedacted(t *testing.T) {
	_, ok := checkClassification("x", map[string]string{"x": "unknown"}, "viewer")
	assert.False(t, ok)
}

// ──── RedactionPolicy ──────────────────────────────────────────────────────

func TestRedactionPolicy_EmptyRole(t *testing.T) {
	properties := map[string]interface{}{
		"email": "test@example.com",
	}
	classifications := map[string]string{
		"email": "pii",
	}
	result := RedactObjectContext(properties, classifications, nil, RedactionPolicy{Role: ""})
	assert.NotContains(t, result.Properties, "email")
	assert.Len(t, result.RedactedFields, 1)
}
