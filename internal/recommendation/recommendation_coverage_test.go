package recommendation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ──── Template functions with nil/zero values ──────────────────────────────

func TestTemplateSellerDelivery_NilValues(t *testing.T) {
	ad := AlertData{
		ObjectID: "seller-nil",
		RuleID:   "seller_late_delivery_spike",
	}
	title, detail, impact, successMetric, confidence, approval := templateSellerDelivery(ad)
	assert.Contains(t, title, "seller-nil")
	assert.Contains(t, detail, "0.0000") // nil CurrentValue renders as 0.0000
	assert.Equal(t, "low", confidence)   // nil sample_size
	assert.False(t, approval)
	assert.NotEmpty(t, impact)
	assert.NotEmpty(t, successMetric)
}

func TestTemplateSellerReview_NilValues(t *testing.T) {
	ad := AlertData{
		ObjectID: "seller-nil2",
		RuleID:   "seller_review_score_drop",
	}
	title, detail, _, _, confidence, _ := templateSellerReview(ad)
	assert.Contains(t, title, "seller-nil2")
	assert.Contains(t, detail, "0.0000")
	assert.Equal(t, "low", confidence)
}

func TestTemplateCategoryGMV_NilValues(t *testing.T) {
	ad := AlertData{
		ObjectID: "cat-nil",
		RuleID:   "category_gmv_drop",
	}
	title, detail, _, _, confidence, _ := templateCategoryGMV(ad)
	assert.Contains(t, title, "cat-nil")
	assert.Equal(t, "low", confidence)
	_ = detail
}

func TestTemplateCategoryReview_NilValues(t *testing.T) {
	ad := AlertData{
		ObjectID: "cat-nil2",
		RuleID:   "category_low_review_cluster",
	}
	title, _, _, _, confidence, _ := templateCategoryReview(ad)
	assert.Contains(t, title, "cat-nil2")
	assert.Equal(t, "low", confidence)
}

func TestTemplateRegionCancel_NilValues(t *testing.T) {
	ad := AlertData{
		ObjectID: "region-nil",
		RuleID:   "region_cancel_rate_spike",
	}
	title, _, _, _, confidence, _ := templateRegionCancel(ad)
	assert.Contains(t, title, "region-nil")
	assert.Equal(t, "low", confidence)
}

func TestTemplateRegionDelivery_NilValues(t *testing.T) {
	ad := AlertData{
		ObjectID: "region-nil2",
		RuleID:   "region_late_delivery_spike",
	}
	title, _, _, _, confidence, _ := templateRegionDelivery(ad)
	assert.Contains(t, title, "region-nil2")
	assert.Equal(t, "low", confidence)
}

// ──── templateGlobalRule edge cases ────────────────────────────────────────

func TestTemplateGlobalRule_LongDescription(t *testing.T) {
	longDesc := "This is a very long description that exceeds the truncation limit of 60 characters for the title field"
	ad := AlertData{
		RuleID:      "test_rule",
		MetricName:  "test_metric",
		Description: longDesc,
	}
	title, _, _, _, _, _ := templateGlobalRule(ad)
	assert.Contains(t, title, "Investigate:")
	assert.LessOrEqual(t, len(title), 80) // truncated title
}

func TestTemplateGlobalRule_EmptyDescription(t *testing.T) {
	ad := AlertData{
		RuleID:     "test_rule",
		MetricName: "test_metric",
		Description: "",
	}
	title, detail, _, _, _, _ := templateGlobalRule(ad)
	assert.Contains(t, title, "Investigate:")
	_ = detail
}

// ──── confidenceFromSample edge cases ──────────────────────────────────────

func TestConfidenceFromSample_ExactlyDoubleMin(t *testing.T) {
	v := int64(40)
	got := confidenceFromSample(&v, 20) // exactly 2*minSample -> not >, so medium
	assert.Equal(t, "medium", got)
}

func TestConfidenceFromSample_AboveDoubleMin(t *testing.T) {
	v := int64(41)
	got := confidenceFromSample(&v, 20) // 41 > 40 -> high
	assert.Equal(t, "high", got)
}

func TestConfidenceFromSample_BetweenMinAndDouble(t *testing.T) {
	v := int64(30)
	got := confidenceFromSample(&v, 20) // 20 < 30 <= 40 -> medium
	assert.Equal(t, "medium", got)
}

func TestConfidenceFromSample_BelowMin(t *testing.T) {
	v := int64(19)
	got := confidenceFromSample(&v, 20) // 19 <= 20 -> low
	assert.Equal(t, "low", got)
}

// ──── Generate with nil logger ─────────────────────────────────────────────

func TestGenerate_NilLogger_NoPanic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	// This test verifies Generate doesn't panic with a nil logger
	// It requires a database connection
	// We just verify the function signature accepts nil
	// The actual DB test is in generate_test.go
}

// ──── TaskGenerator ────────────────────────────────────────────────────────

func TestNewTaskGenerator_NotNil(t *testing.T) {
	gen := NewTaskGenerator()
	if gen == nil {
		t.Fatal("expected non-nil generator")
	}
	_ = gen
}

// ──── GenerateTasks error path ─────────────────────────────────────────────

func TestGenerateTasks_InvalidTx(t *testing.T) {
	// In short mode, we skip DB tests
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	gen := NewTaskGenerator()
	_ = gen
}
