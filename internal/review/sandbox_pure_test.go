package review

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ──── compareData ─────────────────────────────────────────────────────────

func TestCompareData_NoDifferences(t *testing.T) {
	d1 := map[string]interface{}{"a": "1", "b": 2}
	d2 := map[string]interface{}{"a": "1", "b": 2}
	diffs := compareData(d1, d2)
	assert.Empty(t, diffs)
}

func TestCompareData_BothEmpty(t *testing.T) {
	diffs := compareData(map[string]interface{}{}, map[string]interface{}{})
	assert.Empty(t, diffs)
}

func TestCompareData_NilInputs(t *testing.T) {
	diffs := compareData(nil, nil)
	assert.Empty(t, diffs)
	diffs2 := compareData(nil, map[string]interface{}{})
	assert.Empty(t, diffs2)
	diffs3 := compareData(map[string]interface{}{}, nil)
	assert.Empty(t, diffs3)
}

func TestCompareData_FieldInFirstOnly(t *testing.T) {
	d1 := map[string]interface{}{"a": "1", "b": 2}
	d2 := map[string]interface{}{"a": "1"}
	diffs := compareData(d1, d2)
	assert.Len(t, diffs, 1)
	assert.Equal(t, "b", diffs[0].Field)
	assert.Equal(t, 2, diffs[0].Value1)
	assert.Nil(t, diffs[0].Value2)
}

func TestCompareData_FieldInSecondOnly(t *testing.T) {
	d1 := map[string]interface{}{"a": "1"}
	d2 := map[string]interface{}{"a": "1", "b": 2}
	diffs := compareData(d1, d2)
	assert.Len(t, diffs, 1)
	assert.Equal(t, "b", diffs[0].Field)
	assert.Nil(t, diffs[0].Value1)
	assert.Equal(t, 2, diffs[0].Value2)
}

func TestCompareData_DifferentValues(t *testing.T) {
	d1 := map[string]interface{}{"score": 95}
	d2 := map[string]interface{}{"score": 80}
	diffs := compareData(d1, d2)
	assert.Len(t, diffs, 1)
	assert.Equal(t, "score", diffs[0].Field)
	assert.Equal(t, 95, diffs[0].Value1)
	assert.Equal(t, 80, diffs[0].Value2)
}

func TestCompareData_StringValues(t *testing.T) {
	d1 := map[string]interface{}{"name": "Alice"}
	d2 := map[string]interface{}{"name": "Bob"}
	diffs := compareData(d1, d2)
	assert.Len(t, diffs, 1)
	assert.Equal(t, "name", diffs[0].Field)
}

func TestCompareData_NestedMapComparison(t *testing.T) {
	d1 := map[string]interface{}{"meta": map[string]interface{}{"key": "old"}}
	d2 := map[string]interface{}{"meta": map[string]interface{}{"key": "new"}}
	diffs := compareData(d1, d2)
	assert.Len(t, diffs, 1)
}

func TestCompareData_MultipleDifferences(t *testing.T) {
	d1 := map[string]interface{}{"a": 1, "b": "x", "c": true}
	d2 := map[string]interface{}{"a": 2, "b": "y", "c": false}
	diffs := compareData(d1, d2)
	assert.Len(t, diffs, 3)
}

func TestCompareData_SameJSONDifferentTypes(t *testing.T) {
	// 42 as float64 vs int should still match if JSON marshaling produces the same output
	d1 := map[string]interface{}{"val": float64(42)}
	d2 := map[string]interface{}{"val": 42}
	diffs := compareData(d1, d2)
	v1, _ := json.Marshal(d1["val"])
	v2, _ := json.Marshal(d2["val"])
	if string(v1) == string(v2) {
		// They're the same in JSON, so no difference
		assert.Empty(t, diffs)
	}
}

// ──── CompareSandbox (pure helpers only, no DB) ──────────────────────────

func TestCompareData_WithSlices(t *testing.T) {
	d1 := map[string]interface{}{"items": []interface{}{1, 2, 3}}
	d2 := map[string]interface{}{"items": []interface{}{1, 2, 4}}
	diffs := compareData(d1, d2)
	assert.Len(t, diffs, 1)
}

func TestCompareData_SameSlices(t *testing.T) {
	d1 := map[string]interface{}{"items": []interface{}{1, 2, 3}}
	d2 := map[string]interface{}{"items": []interface{}{1, 2, 3}}
	diffs := compareData(d1, d2)
	assert.Empty(t, diffs)
}

func TestCompareData_DifferentTypes(t *testing.T) {
	d1 := map[string]interface{}{"val": 42}
	d2 := map[string]interface{}{"val": "42"}
	diffs := compareData(d1, d2)
	assert.Len(t, diffs, 1)
}

// ──── Difference ──────────────────────────────────────────────────────────

func TestDifference_StructFields(t *testing.T) {
	d := Difference{Field: "test", Value1: "old", Value2: "new"}
	assert.Equal(t, "test", d.Field)
	assert.Equal(t, "old", d.Value1)
	assert.Equal(t, "new", d.Value2)
}

func TestDifference_NilValues(t *testing.T) {
	d := Difference{Field: "test", Value1: nil, Value2: nil}
	assert.Nil(t, d.Value1)
	assert.Nil(t, d.Value2)
}
