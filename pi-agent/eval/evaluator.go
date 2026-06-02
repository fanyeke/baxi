package eval

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// AgentDecision is the structured decision output from the Pi Agent.
type AgentDecision struct {
	SchemaVersion       string              `json:"schema_version"`
	DecisionType        string              `json:"decision_type"`
	Severity            string              `json:"severity"`
	Summary             string              `json:"summary"`
	Rationale           []string            `json:"rationale"`
	RecommendedActions  []RecommendedAction `json:"recommended_actions"`
	Confidence          float64             `json:"confidence"`
	RequiresHumanReview bool                `json:"requires_human_review"`
	EvidenceRefs        []string            `json:"evidence_refs,omitempty"`
	RequiresApproval    bool                `json:"requires_approval,omitempty"`
}

// RecommendedAction is a single action suggested in a decision.
type RecommendedAction struct {
	ActionType string                 `json:"action_type"`
	Priority   string                 `json:"priority"`
	OwnerRole  string                 `json:"owner_role"`
	Payload    map[string]interface{} `json:"payload"`
}

// GoldenCase represents a golden test case for evaluating decisions.
type GoldenCase struct {
	CaseID           string          `json:"case_id"`
	RecipeID         string          `json:"recipe_id"`
	ContextSummary   string          `json:"context_summary"`
	ExpectedDecision AgentDecision   `json:"expected_decision"`
	GradingCriteria  GradingCriteria `json:"grading_criteria"`
	AllowedActions   []string        `json:"allowed_actions,omitempty"`
	ForbiddenActions []string        `json:"forbidden_actions,omitempty"`
}

// GradingCriteria defines the criteria for scoring a decision.
type GradingCriteria struct {
	SeverityMatch        string   `json:"severity_match"`
	MinConfidence        float64  `json:"min_confidence"`
	MustRecommend        []string `json:"must_recommend"`
	MustNotRecommend     []string `json:"must_not_recommend"`
	RequiredEvidenceRefs []string `json:"required_evidence_refs"`
}

// EvalResult holds the scoring breakdown for an evaluation.
type EvalResult struct {
	TotalScore    float64  `json:"total_score"`
	SchemaScore   float64  `json:"schema_score"`
	EvidenceScore float64  `json:"evidence_score"`
	ActionScore   float64  `json:"action_score"`
	SeverityScore float64  `json:"severity_score"`
	Details       []string `json:"details"`
}

// ---------------------------------------------------------------------------
// Schema-level constants (mirrored from internal/llm for package independence)
// ---------------------------------------------------------------------------

var validDecisionTypes = map[string]bool{
	"monitor_only":  true,
	"investigate":   true,
	"optimize":      true,
	"intervention":  true,
	"experiment":    true,
}

var validSeverities = map[string]bool{
	"low":      true,
	"medium":   true,
	"high":     true,
	"critical": true,
}

// ---------------------------------------------------------------------------
// LoadGoldenCase
// ---------------------------------------------------------------------------

// LoadGoldenCase reads and parses a GoldenCase from a JSON file.
func LoadGoldenCase(path string) (*GoldenCase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read golden case file: %w", err)
	}

	var gc GoldenCase
	if err := json.Unmarshal(data, &gc); err != nil {
		return nil, fmt.Errorf("parse golden case JSON: %w", err)
	}

	if gc.CaseID == "" {
		return nil, fmt.Errorf("golden case missing case_id")
	}
	return &gc, nil
}

// ---------------------------------------------------------------------------
// EvaluateDecision
// ---------------------------------------------------------------------------

// EvaluateDecision scores a decision against a golden case.
func EvaluateDecision(decision *AgentDecision, golden *GoldenCase) *EvalResult {
	result := &EvalResult{}
	// --- 1. Schema Validation (25%) ---
	schemaScore, schemaDetails := evaluateSchema(decision)
	result.SchemaScore = schemaScore
	if schemaDetails != "" {
		result.Details = append(result.Details, "--- Schema ---")
		for _, line := range strings.Split(strings.TrimSpace(schemaDetails), "\n") {
			if line != "" {
				result.Details = append(result.Details, line)
			}
		}
	}

	// --- 2. Evidence Verification (25%) ---
	evidenceScore, evidenceDetails := evaluateEvidence(decision, golden)
	result.EvidenceScore = evidenceScore
	if evidenceDetails != "" {
		result.Details = append(result.Details, "--- Evidence ---")
		for _, line := range strings.Split(strings.TrimSpace(evidenceDetails), "\n") {
			if line != "" {
				result.Details = append(result.Details, line)
			}
		}
	}

	// --- 3. Action Allowlisting (25%) ---
	actionScore, actionDetails := evaluateActions(decision, golden)
	result.ActionScore = actionScore
	if actionDetails != "" {
		result.Details = append(result.Details, "--- Actions ---")
		for _, line := range strings.Split(strings.TrimSpace(actionDetails), "\n") {
			if line != "" {
				result.Details = append(result.Details, line)
			}
		}
	}

	// --- 4. Severity + Confidence (25%) ---
	severityScore, sevDetails := evaluateSeverityConfidence(decision, golden)
	result.SeverityScore = severityScore
	if sevDetails != "" {
		result.Details = append(result.Details, "--- Severity/Confidence ---")
		for _, line := range strings.Split(strings.TrimSpace(sevDetails), "\n") {
			if line != "" {
				result.Details = append(result.Details, line)
			}
		}
	}

	result.TotalScore = round2(schemaScore + evidenceScore + actionScore + severityScore)
	return result
}

// ---------------------------------------------------------------------------
// Scoring helpers
// ---------------------------------------------------------------------------

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func addDetail(details *strings.Builder, format string, args ...interface{}) {
	details.WriteString(fmt.Sprintf(format, args...))
	details.WriteString("\n")
}

// ---------------------------------------------------------------------------
// Schema Validation
// ---------------------------------------------------------------------------

func evaluateSchema(d *AgentDecision) (float64, string) {
	var b strings.Builder
	checks := 0
	passed := 0

	// 1. schema_version: empty or known version
	checks++
	if d.SchemaVersion == "" || d.SchemaVersion == "decision_output.v1" {
		passed++
	} else {
		addDetail(&b, "schema_version %q is not recognized", d.SchemaVersion)
	}

	// 2. decision_type: must be valid
	checks++
	if validDecisionTypes[d.DecisionType] {
		passed++
	} else {
		addDetail(&b, "decision_type %q is not valid", d.DecisionType)
	}

	// 3. severity: must be valid
	checks++
	if validSeverities[d.Severity] {
		passed++
	} else {
		addDetail(&b, "severity %q is not valid", d.Severity)
	}

	// 4. confidence in [0, 1]
	checks++
	if d.Confidence >= 0 && d.Confidence <= 1 {
		passed++
	} else {
		addDetail(&b, "confidence %.2f is out of range [0,1]", d.Confidence)
	}

	// 5. summary non-empty
	checks++
	if d.Summary != "" {
		passed++
	} else {
		addDetail(&b, "summary is empty")
	}

	// 6. rationale non-empty
	checks++
	if len(d.Rationale) > 0 {
		passed++
	} else {
		addDetail(&b, "rationale is empty")
	}

	// 7. recommended_actions non-empty
	checks++
	if len(d.RecommendedActions) > 0 {
		passed++
	} else {
		addDetail(&b, "recommended_actions is empty")
	}

	// 8. requires_human_review is true
	checks++
	if d.RequiresHumanReview {
		passed++
	} else {
		addDetail(&b, "requires_human_review is false")
	}

	score := round2((float64(passed) / float64(checks)) * 25)
	return score, b.String()
}

// ---------------------------------------------------------------------------
// Evidence Verification
// ---------------------------------------------------------------------------

func evaluateEvidence(d *AgentDecision, golden *GoldenCase) (float64, string) {
	var b strings.Builder
	required := golden.GradingCriteria.RequiredEvidenceRefs
	actual := d.EvidenceRefs
	if actual == nil {
		actual = []string{}
	}

	score := 25.0

	// --- Required evidence refs must be present ---
	if len(required) > 0 {
		actualSet := make(map[string]bool, len(actual))
		for _, ref := range actual {
			actualSet[ref] = true
		}
		missing := 0
		for _, req := range required {
			if !actualSet[req] {
				missing++
				addDetail(&b, "missing required evidence_ref: %s", req)
			}
		}
		if missing > 0 {
			// Deduct proportionally: each missing required ref reduces evidence score
			ratio := float64(len(required)-missing) / float64(len(required))
			score = round2(ratio * 25)
		}
	} else {
		// No required refs — full marks for presence, still check formatting
	}

	// --- Formatting check on all evidence_refs ---
	malformed := 0
	for _, ref := range actual {
		if !strings.HasPrefix(ref, "metric:") && !strings.HasPrefix(ref, "link:") {
			malformed++
			addDetail(&b, "evidence_ref %q does not start with metric: or link:", ref)
		}
	}
	if malformed > 0 && score > 0 {
		// Deduct 2 points per malformed ref, but don't go below 0
		deduction := float64(malformed) * 2
		score = round2(math.Max(0, score-deduction))
	}

	return score, b.String()
}

// ---------------------------------------------------------------------------
// Action Allowlisting
// ---------------------------------------------------------------------------

func evaluateActions(d *AgentDecision, golden *GoldenCase) (float64, string) {
	var b strings.Builder

	// Determine allowed actions — prefer GoldenCase.AllowedActions, fall back to inferred set
	allowedSet := make(map[string]bool)
	for _, a := range golden.AllowedActions {
		allowedSet[a] = true
	}
	// If golden.AllowedActions is empty, use the union of must_recommend + expected actions
	if len(golden.AllowedActions) == 0 {
		for _, a := range golden.GradingCriteria.MustRecommend {
			allowedSet[a] = true
		}
		for _, a := range golden.ExpectedDecision.RecommendedActions {
			allowedSet[a.ActionType] = true
		}
	}

	forbiddenSet := make(map[string]bool)
	for _, a := range golden.ForbiddenActions {
		forbiddenSet[a] = true
	}
	// Also add must_not_recommend to forbidden set for scoring purposes
	mustNotSet := make(map[string]bool)
	for _, a := range golden.GradingCriteria.MustNotRecommend {
		mustNotSet[a] = true
		forbiddenSet[a] = true
	}

	score := 25.0

	// --- Check recommended actions are in allowed set ---
	actionTypes := make(map[string]bool)
	for _, ra := range d.RecommendedActions {
		actionTypes[ra.ActionType] = true
		if !allowedSet[ra.ActionType] && !mustNotSet[ra.ActionType] {
			if len(golden.AllowedActions) > 0 {
				// Only penalize if golden explicitly defines allowed_actions
				addDetail(&b, "action %q is not in allowed_actions", ra.ActionType)
				score -= 5
			}
		}
	}

	// --- Must_recommend actions must be present ---
	for _, req := range golden.GradingCriteria.MustRecommend {
		if !actionTypes[req] {
			addDetail(&b, "must_recommend action %q is missing", req)
			score -= 5
		}
	}

	// --- Must_not_recommend actions must be absent (heavier penalty) ---
	for _, bad := range golden.GradingCriteria.MustNotRecommend {
		if actionTypes[bad] {
			addDetail(&b, "must_not_recommend action %q is present — heavy penalty", bad)
			score -= 8
		}
	}

	// --- Forbidden actions must be absent (heaviest penalty) ---
	for _, bad := range golden.ForbiddenActions {
		if actionTypes[bad] {
			addDetail(&b, "forbidden action %q is present — maximum penalty", bad)
			score -= 10
		}
	}

	score = round2(math.Max(0, score))
	return score, b.String()
}

// ---------------------------------------------------------------------------
// Severity + Confidence
// ---------------------------------------------------------------------------

func evaluateSeverityConfidence(d *AgentDecision, golden *GoldenCase) (float64, string) {
	var b strings.Builder
	score := 0.0

	// --- Severity match (10 points) ---
	sevMatch := golden.GradingCriteria.SeverityMatch
	expectedSev := strings.TrimPrefix(sevMatch, "must be ")
	if expectedSev == sevMatch {
		expectedSev = sevMatch // no prefix — use as-is
	}
	if expectedSev != "" && strings.EqualFold(d.Severity, expectedSev) {
		score += 10
	} else if expectedSev != "" {
		addDetail(&b, "severity: got %q, expected %q", d.Severity, expectedSev)
	}

	// --- Confidence threshold (10 points) ---
	minConf := golden.GradingCriteria.MinConfidence
	if d.Confidence >= minConf {
		score += 10
	} else {
		addDetail(&b, "confidence %.2f < minimum %.2f", d.Confidence, minConf)
	}

	// --- High-severity / low-confidence penalty (5 points) ---
	// High-risk decisions (high/critical severity) need sufficient confidence.
	isHighRisk := d.Severity == "high" || d.Severity == "critical"
	needsConfidence := 0.7
	if isHighRisk && d.Confidence < needsConfidence {
		addDetail(&b, "high-risk severity %q but confidence only %.2f (need >= %.1f)", d.Severity, d.Confidence, needsConfidence)
		score -= 5
	} else if isHighRisk && d.Confidence >= needsConfidence {
		score += 5 // bonus for appropriate pairing
	}

	score = round2(math.Max(0, score))
	return score, b.String()
}

// ---------------------------------------------------------------------------
// Result helpers
// ---------------------------------------------------------------------------

// Passes returns true if the total score is >= 80%.
func (r *EvalResult) Passes() bool {
	return r.TotalScore >= 80
}

// Summary returns a human-readable summary of the evaluation result.
func (r *EvalResult) Summary() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Score: %.1f/100", r.TotalScore))
	if r.Passes() {
		b.WriteString(" (PASS)")
	} else {
		b.WriteString(" (FAIL)")
	}
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("  Schema:    %.1f/25\n", r.SchemaScore))
	b.WriteString(fmt.Sprintf("  Evidence:  %.1f/25\n", r.EvidenceScore))
	b.WriteString(fmt.Sprintf("  Actions:   %.1f/25\n", r.ActionScore))
	b.WriteString(fmt.Sprintf("  Severity:  %.1f/25\n", r.SeverityScore))

	if len(r.Details) > 0 {
		b.WriteString("\nDetails:\n")
		for _, d := range r.Details {
			b.WriteString(fmt.Sprintf("  - %s\n", d))
		}
	}
	return b.String()
}
