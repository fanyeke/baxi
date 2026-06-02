package recommendation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTaskGenerator_NewTaskGenerator(t *testing.T) {
	gen := NewTaskGenerator()
	assert.NotNil(t, gen)
}

func TestPtrF(t *testing.T) {
	v := 3.14
	result := ptrF(3.14)
	assert.NotNil(t, result)
	assert.Equal(t, v, *result)
}

func TestPtrF_Zero(t *testing.T) {
	result := ptrF(0.0)
	assert.NotNil(t, result)
	assert.Equal(t, 0.0, *result)
}

func TestPtrF_Negative(t *testing.T) {
	result := ptrF(-1.5)
	assert.NotNil(t, result)
	assert.Equal(t, -1.5, *result)
}

func TestConfidenceFromSample_HighThreshold(t *testing.T) {
	v := int64(100)
	assert.Equal(t, "high", confidenceFromSample(&v, 30))
}

func TestConfidenceFromSample_MediumThreshold(t *testing.T) {
	v := int64(50)
	assert.Equal(t, "medium", confidenceFromSample(&v, 30))
}

func TestConfidenceFromSample_LowThreshold(t *testing.T) {
	v := int64(30)
	assert.Equal(t, "low", confidenceFromSample(&v, 30))
}

func TestTemplateSellerDelivery_WithSampleSize(t *testing.T) {
	ad := AlertData{
		ObjectID:     "seller-123",
		RuleID:       "seller_late_delivery_spike",
		CurrentValue: ptrF(0.35),
		SampleSize:   int64Ptr(50),
		AffectedGMV:  ptrF(1000.50),
	}
	title, detail, impact, successMetric, confidence, approval := templateSellerDelivery(ad)
	assert.Contains(t, title, "seller-123")
	assert.Contains(t, detail, "0.3500")
	assert.Contains(t, detail, "50")
	assert.Contains(t, detail, "1000.50")
	assert.Contains(t, detail, "seller_late_delivery_spike")
	assert.NotEmpty(t, impact)
	assert.NotEmpty(t, successMetric)
	assert.Equal(t, "high", confidence)
	assert.False(t, approval)
}

func TestTemplateSellerReview_WithSampleSize(t *testing.T) {
	ad := AlertData{
		ObjectID:     "seller-456",
		RuleID:       "seller_review_score_drop",
		CurrentValue: ptrF(2.8),
		SampleSize:   int64Ptr(25),
		AffectedGMV:  ptrF(500.00),
	}
	title, detail, impact, successMetric, confidence, _ := templateSellerReview(ad)
	assert.Contains(t, title, "seller-456")
	assert.Contains(t, detail, "2.8000")
	assert.Contains(t, detail, "seller_review_score_drop")
	assert.NotEmpty(t, impact)
	assert.NotEmpty(t, successMetric)
	assert.Equal(t, "medium", confidence)
}

func TestTemplateCategoryGMV_WithSampleSize(t *testing.T) {
	ad := AlertData{
		ObjectID:     "cat-electronics",
		RuleID:       "category_gmv_drop",
		CurrentValue: ptrF(5000.0),
		SampleSize:   int64Ptr(100),
	}
	title, detail, impact, successMetric, confidence, _ := templateCategoryGMV(ad)
	assert.Contains(t, title, "cat-electronics")
	assert.Contains(t, detail, "5000.0000")
	assert.Contains(t, detail, "category_gmv_drop")
	assert.NotEmpty(t, impact)
	assert.NotEmpty(t, successMetric)
	assert.Equal(t, "high", confidence)
}

func TestTemplateCategoryReview_WithSampleSize(t *testing.T) {
	ad := AlertData{
		ObjectID:     "cat-health",
		RuleID:       "category_low_review_cluster",
		CurrentValue: ptrF(20.0),
		SampleSize:   int64Ptr(60),
		AffectedGMV:  ptrF(3000.00),
	}
	title, detail, impact, successMetric, confidence, _ := templateCategoryReview(ad)
	assert.Contains(t, title, "cat-health")
	assert.Contains(t, detail, "20.0000")
	assert.Contains(t, detail, "category_low_review_cluster")
	assert.NotEmpty(t, impact)
	assert.NotEmpty(t, successMetric)
	assert.Equal(t, "medium", confidence)
}

func TestTemplateRegionCancel_WithSampleSize(t *testing.T) {
	ad := AlertData{
		ObjectID:     "region-north",
		RuleID:       "region_cancel_rate_spike",
		CurrentValue: ptrF(8.5),
		SampleSize:   int64Ptr(50),
		AffectedGMV:  ptrF(2000.00),
	}
	title, detail, impact, successMetric, confidence, _ := templateRegionCancel(ad)
	assert.Contains(t, title, "region-north")
	assert.Contains(t, detail, "8.5000")
	assert.Contains(t, detail, "region_cancel_rate_spike")
	assert.NotEmpty(t, impact)
	assert.NotEmpty(t, successMetric)
	assert.Equal(t, "medium", confidence)
}

func TestTemplateRegionDelivery_WithSampleSize(t *testing.T) {
	ad := AlertData{
		ObjectID:     "region-south",
		RuleID:       "region_late_delivery_spike",
		CurrentValue: ptrF(25.0),
		SampleSize:   int64Ptr(40),
		AffectedGMV:  ptrF(1500.00),
	}
	title, detail, impact, successMetric, confidence, _ := templateRegionDelivery(ad)
	assert.Contains(t, title, "region-south")
	assert.Contains(t, detail, "25.0000")
	assert.Contains(t, detail, "region_late_delivery_spike")
	assert.NotEmpty(t, impact)
	assert.NotEmpty(t, successMetric)
	assert.Equal(t, "medium", confidence)
}

func TestTemplateGlobalRule_AllFields(t *testing.T) {
	ad := AlertData{
		RuleID:         "global_rule",
		MetricName:     "total_gmv",
		Description:    "GMV is dropping",
		EventDate:      "2024-01-01",
		CurrentValue:   ptrF(1000000.0),
		BaselineValue:  ptrF(1500000.0),
		ChangeRate:     ptrF(-0.33),
	}
	title, detail, impact, successMetric, confidence, approval := templateGlobalRule(ad)
	assert.Contains(t, title, "GMV is dropping")
	assert.Contains(t, detail, "global_rule")
	assert.Contains(t, detail, "total_gmv")
	assert.Contains(t, detail, "2024-01-01")
	assert.Contains(t, detail, "1000000.0000")
	assert.Contains(t, detail, "1500000.0000")
	assert.Contains(t, detail, "-33.00%")
	assert.NotEmpty(t, impact)
	assert.Equal(t, "total_gmv", successMetric)
	assert.Equal(t, "medium", confidence)
	assert.False(t, approval)
}

func TestTemplateGlobalRule_NilValues(t *testing.T) {
	ad := AlertData{
		RuleID:     "test_rule",
		MetricName: "test_metric",
	}
	_, detail, _, _, _, _ := templateGlobalRule(ad)
	assert.Contains(t, detail, "N/A")
}

func TestDeriveTaskID_LongID(t *testing.T) {
	assert.Equal(t, "task-very-long-identifier-here", deriveTaskID("rec-very-long-identifier-here"))
}

func TestDeriveTaskSource_DimrecPrefix(t *testing.T) {
	assert.Equal(t, "dimensional_rule", deriveTaskSource("dimrec-seller_123"))
}

func TestDeriveTaskSource_RecPrefix(t *testing.T) {
	assert.Equal(t, "heuristic_strategy", deriveTaskSource("rec-gmv_drop"))
}

func TestDerivePriority_UnknownReturnsMedium(t *testing.T) {
	assert.Equal(t, "medium", derivePriority("unknown"))
}

func TestRuleTemplateMap_AllRulesMapped(t *testing.T) {
	expectedRules := []string{
		"seller_late_delivery_spike",
		"seller_review_score_drop",
		"category_gmv_drop",
		"category_low_review_cluster",
		"region_cancel_rate_spike",
		"region_late_delivery_spike",
	}
	for _, rule := range expectedRules {
		_, ok := ruleTemplateMap[rule]
		assert.True(t, ok, "rule %s should be in ruleTemplateMap", rule)
	}
}

func TestRuleTemplateMap_Count(t *testing.T) {
	assert.Equal(t, 6, len(ruleTemplateMap))
}

func TestTruncStr_EdgeCases(t *testing.T) {
	assert.Equal(t, "abcde", truncStr("abcde", 5))
	assert.Equal(t, "abcde", truncStr("abcdef", 5))
	assert.Len(t, truncStr("a very long string that goes on and on", 10), 10)
}

func TestPtrFmt6d_VariousValues(t *testing.T) {
	tests := []struct {
		name     string
		input    *float64
		expected string
	}{
		{"nil", nil, "0.0000"},
		{"zero", ptrF(0.0), "0.0000"},
		{"small", ptrF(0.0001), "0.0001"},
		{"large", ptrF(999999.9999), "999999.9999"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, ptrFmt6d(tc.input))
		})
	}
}

func TestPtrFmtInt_VariousValues(t *testing.T) {
	tests := []struct {
		name     string
		input    *int64
		expected string
	}{
		{"nil", nil, "0"},
		{"zero", ptrI(0), "0"},
		{"max_int32", ptrI(2147483647), "2147483647"},
		{"negative", ptrI(-100), "-100"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, ptrFmtInt(tc.input))
		})
	}
}

func TestOrElseFmt_PreservesNonNil(t *testing.T) {
	v := 42.0
	fn := func(f float64) string { return "got_value" }
	assert.Equal(t, "got_value", orElseFmt(&v, "default", fn))
}

func TestOrElseFmt_NilReturnsDefault(t *testing.T) {
	fn := func(f float64) string { return "got_value" }
	assert.Equal(t, "default", orElseFmt[float64](nil, "default", fn))
}

func ptrI(v int64) *int64 { return &v }
