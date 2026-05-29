// Package eval provides evaluation and replay capabilities for LLM decisions.
package eval

import (
	"context"
	"encoding/json"
	"fmt"

	"baxi/internal/llm"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReplayData holds the data needed to replay a previous LLM decision.
// InputContext is the JSON representation of the LLMSafeContext that was
// originally sent to the LLM provider. OriginalOutput is the parsed
// DecisionOutput that the provider returned.
type ReplayData struct {
	CaseID             string
	OriginalDecisionID string
	InputContext       json.RawMessage
	OriginalOutput     *llm.DecisionOutput
	Provider           string
	Model              string
	PromptVersion      string
	ContextHash        string
}

// ReplayOptions configures the behavior of a decision replay.
type ReplayOptions struct {
	ContextHash   string // optional: specify exact context version to replay
	Provider      string // optional: "openai", "rule_based", or empty for original
	PromptVersion string // optional: specific prompt version
	DryRun        bool   // default false: executes provider call
}

// DecisionDiff holds the comparison result between original and replayed decisions.
type DecisionDiff struct {
	DecisionTypeMatch bool    `json:"decision_type_match"`
	SeverityMatch     bool    `json:"severity_match"`
	ConfidenceDiff    float64 `json:"confidence_diff"`
	ActionOverlap     float64 `json:"action_overlap"`
	SummaryChanged    bool    `json:"summary_changed"`
	RationaleChanged  bool    `json:"rationale_changed"`
}

// ReplayResult holds the outcome of a decision replay.
type ReplayResult struct {
	OriginalDecision *llm.DecisionOutput   `json:"original_decision"`
	ReplayedDecision *llm.DecisionOutput   `json:"replayed_decision,omitempty"`
	Diff             *DecisionDiff         `json:"diff,omitempty"`
	ContextHash      string                `json:"context_hash"`
	PromptVersion    string                `json:"prompt_version"`
	Model            string                `json:"model"`
	DryRun           bool                  `json:"dry_run"`
	ValidationResult *llm.ValidationResult `json:"validation_result,omitempty"`
}

// DecisionRepository defines the storage interface needed for replay.
type DecisionRepository interface {
	GetLLMDecisionByCaseID(ctx context.Context, caseID string) (*ReplayData, error)
}

// ReplayService replays previous LLM decisions for audit and comparison.
// It fetches the original decision data from the repository and optionally
// calls the LLM provider again with the same context to produce a comparison.
type ReplayService struct {
	decisionRepo DecisionRepository
	provider     llm.DecisionProvider
	auditLogger  llm.LLMAuditLogger
}

// NewReplayService creates a new ReplayService.
func NewReplayService(repo DecisionRepository, provider llm.DecisionProvider, auditLogger llm.LLMAuditLogger) *ReplayService {
	return &ReplayService{
		decisionRepo: repo,
		provider:     provider,
		auditLogger:  auditLogger,
	}
}

// Replay replays a previous LLM decision identified by caseID.
//
// If opts.DryRun is true, the method returns the original decision data without
// calling the LLM provider. This is useful for auditing what decision was
// made without incurring cost or side effects.
//
// If opts.DryRun is false, the method calls the LLM provider with the same
// input context as the original decision and returns both the original
// and replayed decisions with a diff for comparison.
//
// The method does NOT auto-approve or auto-apply any replayed decision.
func (s *ReplayService) Replay(ctx context.Context, caseID string, opts ReplayOptions) (*ReplayResult, error) {
	data, err := s.decisionRepo.GetLLMDecisionByCaseID(ctx, caseID)
	if err != nil {
		return nil, fmt.Errorf("fetch replay data for case %s: %w", caseID, err)
	}

	// Filter by context hash if specified
	if opts.ContextHash != "" && data.ContextHash != opts.ContextHash {
		return nil, fmt.Errorf("context hash mismatch: expected %s, got %s", opts.ContextHash, data.ContextHash)
	}

	result := &ReplayResult{
		OriginalDecision: data.OriginalOutput,
		ContextHash:      data.ContextHash,
		PromptVersion:    data.PromptVersion,
		Model:            data.Model,
		DryRun:           opts.DryRun,
	}

	if opts.DryRun {
		return result, nil
	}

	var inputContext llm.LLMSafeContext
	if err := json.Unmarshal(data.InputContext, &inputContext); err != nil {
		return nil, fmt.Errorf("unmarshal input context for case %s: %w", caseID, err)
	}

	s.auditLogger.LogDecisionReplayed(ctx, caseID, data.OriginalDecisionID)

	replayedOutput, err := s.provider.GenerateDecision(ctx, inputContext)
	if err != nil {
		return nil, fmt.Errorf("replay provider call for case %s: %w", caseID, err)
	}

	result.ReplayedDecision = replayedOutput

	// Compute diff between original and replayed decisions
	if data.OriginalOutput != nil && replayedOutput != nil {
		diff := computeDecisionDiff(data.OriginalOutput, replayedOutput)
		result.Diff = diff
	}

	// Validate replayed output
	validationResult := llm.ValidateDecision(replayedOutput, inputContext.AllowedActions)
	result.ValidationResult = validationResult

	return result, nil
}

// ReplayLegacy calls Replay with a legacy dryRun boolean for backward compatibility.
func (s *ReplayService) ReplayLegacy(ctx context.Context, caseID string, dryRun bool) (*ReplayResult, error) {
	return s.Replay(ctx, caseID, ReplayOptions{DryRun: dryRun})
}

// computeDecisionDiff compares original and replayed decisions and returns a diff.
func computeDecisionDiff(original, replayed *llm.DecisionOutput) *DecisionDiff {
	if original == nil || replayed == nil {
		return nil
	}

	diff := &DecisionDiff{
		DecisionTypeMatch: original.DecisionType == replayed.DecisionType,
		SeverityMatch:     original.Severity == replayed.Severity,
		ConfidenceDiff:    absFloat(original.Confidence - replayed.Confidence),
		SummaryChanged:    original.Summary != replayed.Summary,
		RationaleChanged:  !stringSlicesEqual(original.Rationale, replayed.Rationale),
	}

	// Compute action overlap using Jaccard index
	origActions := make(map[string]bool)
	for _, a := range original.RecommendedActions {
		origActions[a.ActionType] = true
	}
	matchCount := 0
	for _, a := range replayed.RecommendedActions {
		if origActions[a.ActionType] {
			matchCount++
		}
	}
	union := len(original.RecommendedActions) + len(replayed.RecommendedActions) - matchCount
	if union > 0 {
		diff.ActionOverlap = float64(matchCount) / float64(union)
	} else {
		diff.ActionOverlap = 1.0
	}

	return diff
}

func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// PGReplayRepository implements DecisionRepository using a pgx connection pool.
// It queries the ai.llm_decision table for the latest decision for a given case.
type PGReplayRepository struct {
	pool *pgxpool.Pool
}

// NewPGReplayRepository creates a PGReplayRepository.
func NewPGReplayRepository(pool *pgxpool.Pool) *PGReplayRepository {
	return &PGReplayRepository{pool: pool}
}

// GetLLMDecisionByCaseID retrieves the latest LLM decision data for a case.
func (r *PGReplayRepository) GetLLMDecisionByCaseID(ctx context.Context, caseID string) (*ReplayData, error) {
	query := `
		SELECT decision_id, case_id, input_json, output_json,
		       COALESCE(provider, ''), COALESCE(model, ''),
		       COALESCE(prompt_version, ''), COALESCE(context_hash, '')
		FROM ai.llm_decision
		WHERE case_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var (
		decisionID    string
		cid           string
		inputJSON     *json.RawMessage
		outputJSON    *json.RawMessage
		provider      string
		model         string
		promptVersion string
		contextHash   string
	)

	err := r.pool.QueryRow(ctx, query, caseID).Scan(
		&decisionID, &cid, &inputJSON, &outputJSON,
		&provider, &model, &promptVersion, &contextHash,
	)
	if err != nil {
		return nil, fmt.Errorf("query ai.llm_decision by case_id: %w", err)
	}

	var originalOutput *llm.DecisionOutput
	if outputJSON != nil {
		var out llm.DecisionOutput
		if err := json.Unmarshal(*outputJSON, &out); err != nil {
			return nil, fmt.Errorf("unmarshal output_json: %w", err)
		}
		originalOutput = &out
	}

	if inputJSON == nil {
		return nil, fmt.Errorf("input_json is nil for decision %s", decisionID)
	}

	return &ReplayData{
		CaseID:             cid,
		OriginalDecisionID: decisionID,
		InputContext:       *inputJSON,
		OriginalOutput:     originalOutput,
		Provider:           provider,
		Model:              model,
		PromptVersion:      promptVersion,
		ContextHash:        contextHash,
	}, nil
}
