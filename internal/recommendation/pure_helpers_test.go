package recommendation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ──── truncStr ────────────────────────────────────────────────────────────

func TestTruncStr_ShorterThanMax(t *testing.T) {
	assert.Equal(t, "hello", truncStr("hello", 10))
}

func TestTruncStr_EqualToMax(t *testing.T) {
	assert.Equal(t, "hello", truncStr("hello", 5))
}

func TestTruncStr_LongerThanMax(t *testing.T) {
	input := "hello world this is a long string"
	// With maxLen=10, truncStr should take first 10 chars and trim spaces
	assert.Len(t, truncStr(input, 10), 10)
}

func TestTruncStr_EmptyString(t *testing.T) {
	assert.Equal(t, "", truncStr("", 10))
}

func TestTruncStr_ZeroMaxLen(t *testing.T) {
	assert.Equal(t, "", truncStr("hello", 0))
}

func TestTruncStr_TrimsTrailingWhitespace(t *testing.T) {
	result := truncStr("hello world", 5)
	assert.Equal(t, "hello", result)
}

func TestTruncStr_Unicode(t *testing.T) {
	// Unicode characters are multi-byte, but truncStr uses len() which counts bytes
	result := truncStr("你好世界", 6)
	// "你好" is 6 bytes in UTF-8
	assert.Equal(t, "你好", result)
}

// ──── ptrFmt6d ────────────────────────────────────────────────────────────

func TestPtrFmt6d_Nil(t *testing.T) {
	assert.Equal(t, "0.0000", ptrFmt6d(nil))
}

func TestPtrFmt6d_Zero(t *testing.T) {
	v := 0.0
	assert.Equal(t, "0.0000", ptrFmt6d(&v))
}

func TestPtrFmt6d_Positive(t *testing.T) {
	v := 3.14159
	assert.Equal(t, "3.1416", ptrFmt6d(&v))
}

func TestPtrFmt6d_Negative(t *testing.T) {
	v := -2.71828
	assert.Equal(t, "-2.7183", ptrFmt6d(&v))
}

func TestPtrFmt6d_LargeValue(t *testing.T) {
	v := 123456.789012
	assert.Equal(t, "123456.7890", ptrFmt6d(&v))
}

// ──── ptrFmt2d ────────────────────────────────────────────────────────────

func TestPtrFmt2d_Nil(t *testing.T) {
	assert.Equal(t, "0.00", ptrFmt2d(nil))
}

func TestPtrFmt2d_Positive(t *testing.T) {
	v := 3.14159
	assert.Equal(t, "3.14", ptrFmt2d(&v))
}

func TestPtrFmt2d_Rounding(t *testing.T) {
	v := 3.145
	assert.Equal(t, "3.15", ptrFmt2d(&v))
}

func TestPtrFmt2d_Negative(t *testing.T) {
	v := -2.718
	assert.Equal(t, "-2.72", ptrFmt2d(&v))
}

// ──── ptrFmtInt ───────────────────────────────────────────────────────────

func TestPtrFmtInt_Nil(t *testing.T) {
	assert.Equal(t, "0", ptrFmtInt(nil))
}

func TestPtrFmtInt_Positive(t *testing.T) {
	v := int64(42)
	assert.Equal(t, "42", ptrFmtInt(&v))
}

func TestPtrFmtInt_Negative(t *testing.T) {
	v := int64(-7)
	assert.Equal(t, "-7", ptrFmtInt(&v))
}

func TestPtrFmtInt_Zero(t *testing.T) {
	v := int64(0)
	assert.Equal(t, "0", ptrFmtInt(&v))
}

func TestPtrFmtInt_Large(t *testing.T) {
	v := int64(999999999)
	assert.Equal(t, "999999999", ptrFmtInt(&v))
}

// ──── orElseFmt ───────────────────────────────────────────────────────────

func TestOrElseFmt_NonNil(t *testing.T) {
	v := 3.14
	fn := func(f float64) string { return "custom" }
	assert.Equal(t, "custom", orElseFmt(&v, "fallback", fn))
}

func TestOrElseFmt_Nil(t *testing.T) {
	fn := func(f float64) string { return "custom" }
	assert.Equal(t, "fallback", orElseFmt[float64](nil, "fallback", fn))
}

func TestOrElseFmt_NilWithEmptyFallback(t *testing.T) {
	fn := func(f float64) string { return "custom" }
	assert.Equal(t, "", orElseFmt[float64](nil, "", fn))
}

func TestOrElseFmt_PreservesResult(t *testing.T) {
	v := 42.0
	fn := func(f float64) string { return "the_answer" }
	assert.Equal(t, "the_answer", orElseFmt(&v, "fallback", fn))
}

// ──── deriveTaskID (additional edge cases) ────────────────────────────────

func TestDeriveTaskID_CustomPrefix(t *testing.T) {
	assert.Equal(t, "task-custom-id", deriveTaskID("custom-id"))
}

func TestDeriveTaskID_Empty(t *testing.T) {
	assert.Equal(t, "task-", deriveTaskID(""))
}

func TestDeriveTaskID_DimrecOnly(t *testing.T) {
	assert.Equal(t, "dimtask-", deriveTaskID("dimrec-"))
}

func TestDeriveTaskID_RecOnly(t *testing.T) {
	assert.Equal(t, "task-", deriveTaskID("rec-"))
}

// ──── deriveTaskSource (additional edge cases) ────────────────────────────

func TestDeriveTaskSource_Empty(t *testing.T) {
	assert.Equal(t, "heuristic_strategy", deriveTaskSource(""))
}

func TestDeriveTaskSource_DimrecOnly(t *testing.T) {
	assert.Equal(t, "dimensional_rule", deriveTaskSource("dimrec-"))
}

// ──── derivePriority (additional edge cases) ──────────────────────────────

func TestDerivePriority_AllCases(t *testing.T) {
	assert.Equal(t, "high", derivePriority("high"))
	assert.Equal(t, "medium", derivePriority("medium"))
	assert.Equal(t, "low", derivePriority("low"))
	assert.Equal(t, "medium", derivePriority("HIGH"))
	assert.Equal(t, "medium", derivePriority(""))
	assert.Equal(t, "medium", derivePriority("critical"))
}

// ──── confidenceFromSample ──────────────────────────────────────────────────

func TestConfidenceFromSample_Nil(t *testing.T) {
	assert.Equal(t, "low", confidenceFromSample(nil, 100))
}

func TestConfidenceFromSample_Sufficient(t *testing.T) {
	v := int64(200)
	assert.Equal(t, "medium", confidenceFromSample(&v, 100))
}

func TestConfidenceFromSample_Insufficient(t *testing.T) {
	v := int64(50)
	assert.Equal(t, "low", confidenceFromSample(&v, 100))
}

func TestConfidenceFromSample_ExactlyMin(t *testing.T) {
	v := int64(100)
	assert.Equal(t, "low", confidenceFromSample(&v, 100))
}

func TestConfidenceFromSample_ZeroSample(t *testing.T) {
	v := int64(0)
	assert.Equal(t, "low", confidenceFromSample(&v, 100))
}

func TestConfidenceFromSample_MinSampleZero(t *testing.T) {
	v := int64(5)
	assert.Equal(t, "high", confidenceFromSample(&v, 0))
}

// ──── roundTo ───────────────────────────────────────────────────────────────

func TestRoundTo_ThreeDecimals(t *testing.T) {
	assert.Equal(t, 3.142, roundTo(3.14159, 3))
}

func TestRoundTo_TwoDecimals(t *testing.T) {
	assert.Equal(t, 3.14, roundTo(3.14159, 2))
}

func TestRoundTo_ZeroDecimals(t *testing.T) {
	assert.Equal(t, 3.0, roundTo(3.14159, 0))
}

func TestRoundTo_Negative(t *testing.T) {
	assert.Equal(t, -3.14, roundTo(-3.14159, 2))
}

func TestRoundTo_Exact(t *testing.T) {
	assert.Equal(t, 1.5, roundTo(1.5, 1))
}

func TestRoundTo_Large(t *testing.T) {
	assert.Equal(t, 123456.79, roundTo(123456.789, 2))
}

// ──── Template functions ────────────────────────────────────────────────────

func TestTemplateSellerDelivery(t *testing.T) {
	ad := AlertData{
		ObjectID:   "seller-TestSeller",
		MetricName:   "delivery_rate",
		CurrentValue: ptrF(85.5),
		BaselineValue: ptrF(95.0),
		ChangeRate:   ptrF(-10.0),

	}
	title, detail, impact, successMetric, confidence, requiresApproval := templateSellerDelivery(ad)
	assert.Contains(t, title, "TestSeller", "title should contain seller name")
	assert.Contains(t, detail, "delivery_rate", "detail should mention metric")
	assert.NotEmpty(t, impact)
	assert.NotEmpty(t, successMetric)
	assert.NotEmpty(t, confidence)
	assert.False(t, requiresApproval)
}

func TestTemplateSellerReview(t *testing.T) {
	ad := AlertData{ObjectID: "seller-TestSeller", MetricName: "review_score"}
	title, detail, _, _, _, _ := templateSellerReview(ad)
	assert.Contains(t, title, "TestSeller")
	assert.Contains(t, detail, "review_score")
}

func TestTemplateCategoryGMV(t *testing.T) {
	ad := AlertData{
		ObjectID:   "cat-123",
		MetricName: "gmv",
	}
	title, detail, impact, _, confidence, requiresApproval := templateCategoryGMV(ad)
	assert.Contains(t, title, "cat-123")
	assert.Contains(t, detail, "GMV")
	assert.NotEmpty(t, impact)
	assert.NotEmpty(t, confidence)
	assert.False(t, requiresApproval)
}

func TestTemplateCategoryReview(t *testing.T) {
	ad := AlertData{
		ObjectID:   "cat-456",
		MetricName: "review_score",
		CurrentValue: ptrF(4.2),
		BaselineValue: ptrF(4.5),
	}
	title, detail, _, _, _, _ := templateCategoryReview(ad)
	assert.Contains(t, title, "cat-456")
	assert.Contains(t, detail, "low_review_rate")
}

func TestTemplateRegionCancel(t *testing.T) {
	ad := AlertData{
		ObjectID:   "region-cn",
		MetricName: "cancel_rate",
	}
	title, detail, _, _, _, _ := templateRegionCancel(ad)
	assert.Contains(t, title, "region-cn")
	assert.Contains(t, detail, "cancel_rate")
}

func TestTemplateRegionDelivery(t *testing.T) {
	ad := AlertData{
		ObjectID:   "region-us",
		MetricName: "delivery_rate",
	}
	title, detail, _, _, _, _ := templateRegionDelivery(ad)
	assert.Contains(t, title, "region-us")
	assert.Contains(t, detail, "delivery_rate")
}

func TestTemplateGlobalRule(t *testing.T) {
	ad := AlertData{
		RuleID:     "gmv_drop",
		MetricName: "gmv",
		Description: "GMV drop detected",
		CurrentValue: ptrF(1000000),
		BaselineValue: ptrF(1500000),
	}
	title, detail, impact, _, _, _ := templateGlobalRule(ad)
	assert.Contains(t, title, "GMV drop")
	assert.Contains(t, detail, "gmv")
	assert.NotEmpty(t, impact)
}

func ptrF(v float64) *float64 { return &v }
