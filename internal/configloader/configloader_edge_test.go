package configloader

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──── Parse function error paths ──────────────────────────────────────────

func TestParseFunctions_InvalidYAML(t *testing.T) {
	tests := []struct {
		name string
		fn   func([]byte) (any, error)
	}{
		{"parseDataClassification", func(b []byte) (any, error) { return parseDataClassification(b) }},
		{"parseAccessPolicy", func(b []byte) (any, error) { return parseAccessPolicy(b) }},
		{"parseDataLineage", func(b []byte) (any, error) { return parseDataLineage(b) }},
		{"parseDataMarkings", func(b []byte) (any, error) { return parseDataMarkings(b) }},
		{"parseHealthChecks", func(b []byte) (any, error) { return parseHealthChecks(b) }},
		{"parseCheckpointRules", func(b []byte) (any, error) { return parseCheckpointRules(b) }},
		{"parseAlertRules", func(b []byte) (any, error) { return parseAlertRules(b) }},
		{"parseMetrics", func(b []byte) (any, error) { return parseMetrics(b) }},
		{"parseAny", func(b []byte) (any, error) { return parseAny(b) }},
	}
	for _, tc := range tests {
		t.Run(tc.name+"_invalid", func(t *testing.T) {
			_, err := tc.fn([]byte(": invalid"))
			require.Error(t, err)
		})
	}
}

func TestParseFunctions_EmptyInput(t *testing.T) {
	fnsThatReturnNilForEmpty := map[string]bool{"parseAny": true}
	tests := []struct {
		name string
		fn   func([]byte) (any, error)
	}{
		{"parseDataClassification", func(b []byte) (any, error) { return parseDataClassification(b) }},
		{"parseAccessPolicy", func(b []byte) (any, error) { return parseAccessPolicy(b) }},
		{"parseDataLineage", func(b []byte) (any, error) { return parseDataLineage(b) }},
		{"parseDataMarkings", func(b []byte) (any, error) { return parseDataMarkings(b) }},
		{"parseHealthChecks", func(b []byte) (any, error) { return parseHealthChecks(b) }},
		{"parseCheckpointRules", func(b []byte) (any, error) { return parseCheckpointRules(b) }},
		{"parseAlertRules", func(b []byte) (any, error) { return parseAlertRules(b) }},
		{"parseMetrics", func(b []byte) (any, error) { return parseMetrics(b) }},
		{"parseAny", func(b []byte) (any, error) { return parseAny(b) }},
	}
	for _, tc := range tests {
		t.Run(tc.name+"_empty", func(t *testing.T) {
			result, err := tc.fn([]byte{})
			require.NoError(t, err)
			if !fnsThatReturnNilForEmpty[tc.name] {
				assert.NotNil(t, result)
			}
		})
	}
}

// ──── parseConfigByType branch coverage ────────────────────────────────────

func TestParseConfigByType_AllBranches(t *testing.T) {
	validYAML := []byte("objects:\n  - object_type_id: test")

	tests := []struct {
		configType string
	}{
		{"data_classification"},
		{"access_policy"},
		{"data_lineage"},
		{"data_markings"},
		{"health_checks"},
		{"checkpoint_rules"},
		{"alert_rules"},
		{"metrics"},
	}
	for _, tc := range tests {
		t.Run(tc.configType, func(t *testing.T) {
			result, err := parseConfigByType(tc.configType, validYAML)
			require.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

func TestParseConfigByType_CustomType(t *testing.T) {
	result, err := parseConfigByType("custom_type", []byte("key: value"))
	require.NoError(t, err)
	assert.NotNil(t, result)
	asMap, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "value", asMap["key"])
}

// ──── yamlToJSON edge cases ───────────────────────────────────────────────

func TestYAMLToJSON_NestedStructure(t *testing.T) {
	json, err := yamlToJSON([]byte("key: value\nnested:\n  inner: 42\n"))
	require.NoError(t, err)
	assert.Contains(t, string(json), "key")
	assert.Contains(t, string(json), "42")
}
