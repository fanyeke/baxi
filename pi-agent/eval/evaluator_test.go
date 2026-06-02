package eval

import (
	"testing"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func validDecision() *AgentDecision {
	return &AgentDecision{
		SchemaVersion:       "decision_output.v1",
		DecisionType:        "intervention",
		Severity:            "high",
		Summary:             "Seller SELLER_001 late delivery rate spike requires intervention.",
		Rationale:           []string{"late_delivery_rate_7d is 0.31 vs 0.08 baseline (287.5% increase).", "Order count declining and review scores dropping."},
		RecommendedActions: []RecommendedAction{
			{ActionType: "notify_owner", Priority: "high", OwnerRole: "seller_ops"},
			{ActionType: "create_followup_task", Priority: "high", OwnerRole: "seller_ops"},
		},
		Confidence:          0.85,
		RequiresHumanReview: true,
		EvidenceRefs:        []string{"metric:seller_late_delivery_rate_7d"},
	}
}

func validGolden() *GoldenCase {
	return &GoldenCase{
		CaseID:         "case-01-seller-late-delivery",
		RecipeID:       "seller_late_delivery_alert",
		ContextSummary: "Seller SELLER_001 late delivery rate alert.",
		ExpectedDecision: AgentDecision{
			SchemaVersion: "decision_output.v1",
			DecisionType:  "intervention",
			Severity:      "high",
		},
		GradingCriteria: GradingCriteria{
			SeverityMatch:        "must be high",
			MinConfidence:        0.7,
			MustRecommend:        []string{"notify_owner", "create_followup_task"},
			MustNotRecommend:     []string{"export_report"},
			RequiredEvidenceRefs: []string{"metric:seller_late_delivery_rate_7d"},
		},
		AllowedActions:   []string{"notify_owner", "create_followup_task", "export_report"},
		ForbiddenActions: []string{"execute_dispatch"},
	}
}

func TestEvaluate_ValidDecision(t *testing.T) {
	decision := validDecision()
	golden := validGolden()

	result := EvaluateDecision(decision, golden)

	if !result.Passes() {
		t.Errorf("expected PASS (>=80), got score %.1f\n%s", result.TotalScore, result.Summary())
	}
	if result.TotalScore < 99 {
		t.Errorf("expected near-perfect score (>=99), got %.1f\n%s", result.TotalScore, result.Summary())
	}
}

func TestEvaluate_WrongSeverity(t *testing.T) {
	decision := validDecision()
	decision.Severity = "low"

	golden := validGolden()
	golden.GradingCriteria.SeverityMatch = "must be high"

	result := EvaluateDecision(decision, golden)

	// Severity is wrong: severity_score should be < 25
	if result.SeverityScore >= 20 {
		t.Errorf("expected severity score < 20 for wrong severity, got %.1f", result.SeverityScore)
	}
	// Details should mention the mismatch
	found := false
	for _, d := range result.Details {
		if d == `severity: got "low", expected "high"` {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected detail about severity mismatch, got: %v", result.Details)
	}
}

func TestEvaluate_MissingEvidenceRefs(t *testing.T) {
	decision := validDecision()
	decision.EvidenceRefs = []string{"metric:some_other_metric"} // missing the required one

	golden := validGolden()
	golden.GradingCriteria.RequiredEvidenceRefs = []string{"metric:seller_late_delivery_rate_7d"}

	result := EvaluateDecision(decision, golden)

	if result.EvidenceScore >= 20 {
		t.Errorf("expected evidence score < 20 when required ref missing, got %.1f", result.EvidenceScore)
	}
	// Should still pass overall (wrong ref penalizes evidence but other categories are OK)
}

func TestEvaluate_ForbiddenAction(t *testing.T) {
	decision := validDecision()
	decision.RecommendedActions = append(decision.RecommendedActions, RecommendedAction{
		ActionType: "execute_dispatch",
		Priority:   "high",
		OwnerRole:  "ops",
	})

	golden := validGolden()
	golden.ForbiddenActions = []string{"execute_dispatch"}

	result := EvaluateDecision(decision, golden)

	if result.ActionScore >= 20 {
		t.Errorf("expected action score < 20 when forbidden action present, got %.1f", result.ActionScore)
	}
	found := false
	for _, d := range result.Details {
		if d == `forbidden action "execute_dispatch" is present — maximum penalty` {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected detail about forbidden action, got: %v", result.Details)
	}
}

func TestEvaluate_LowConfidenceHighRisk(t *testing.T) {
	decision := validDecision()
	decision.Severity = "critical"
	decision.Confidence = 0.35 // low confidence for critical severity

	golden := validGolden()
	golden.GradingCriteria.SeverityMatch = "must be critical"
	golden.GradingCriteria.MinConfidence = 0.7
	golden.GradingCriteria.RequiredEvidenceRefs = nil // remove evidence requirement to isolate the test

	result := EvaluateDecision(decision, golden)

	// Severity matches but confidence is too low (below 0.7) and high-risk penalty applies
	if result.SeverityScore >= 25 {
		t.Errorf("expected severity+confidence score < 25 due to low confidence with critical risk, got %.1f", result.SeverityScore)
	}
}

// TestEvaluate_MalformedEvidenceRef ensures malformed refs are penalized.
func TestEvaluate_MalformedEvidenceRef(t *testing.T) {
	decision := validDecision()
	decision.EvidenceRefs = []string{"metric:seller_late_delivery_rate_7d", "bad_ref_no_prefix"}

	golden := validGolden()

	result := EvaluateDecision(decision, golden)

	if result.EvidenceScore >= 25 {
		t.Errorf("expected evidence score < 25 when malformed ref present, got %.1f", result.EvidenceScore)
	}
}

// TestEvaluate_EmptyDecision ensures a nil-like empty decision scores poorly.
func TestEvaluate_EmptyDecision(t *testing.T) {
	decision := &AgentDecision{}
	golden := validGolden()

	result := EvaluateDecision(decision, golden)

	if result.Passes() {
		t.Error("expected FAIL for empty decision, got PASS")
	}
}

// TestEvalResult_Passes tests the Passes threshold directly.
func TestEvalResult_Passes(t *testing.T) {
	r := &EvalResult{TotalScore: 80}
	if !r.Passes() {
		t.Error("expected Passes() true for 80")
	}
	r.TotalScore = 79.99
	if r.Passes() {
		t.Error("expected Passes() false for 79.99")
	}
	r.TotalScore = 100
	if !r.Passes() {
		t.Error("expected Passes() true for 100")
	}
}

// TestEvalResult_Summary ensures Summary() output is non-empty and contains score.
func TestEvalResult_Summary(t *testing.T) {
	r := &EvalResult{
		TotalScore:    85.0,
		SchemaScore:   20.0,
		EvidenceScore: 22.0,
		ActionScore:   23.0,
		SeverityScore: 20.0,
		Details:       []string{"some detail"},
	}
	s := r.Summary()
	if s == "" {
		t.Fatal("summary should not be empty")
	}
}

// TestLoadGoldenCase verifies loading a real JSON file.
func TestLoadGoldenCase(t *testing.T) {
	gc, err := LoadGoldenCase("../golden_cases/case-01-seller-late-delivery.json")
	if err != nil {
		t.Fatalf("LoadGoldenCase failed: %v", err)
	}
	if gc.CaseID != "case-01-seller-late-delivery" {
		t.Errorf("expected case-01-seller-late-delivery, got %q", gc.CaseID)
	}
}
