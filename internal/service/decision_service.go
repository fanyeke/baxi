package service

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"

	"baxi/internal/action"
	"baxi/internal/decision"
	"baxi/internal/eval"
	"baxi/internal/llm"
)

// DecisionService composes case, context, engine, and proposal services into a
// single business orchestration layer for the decision workflow.
type DecisionService struct {
	caseSvc      CaseService
	ctxBuilder   ContextBuilder
	engine       DecisionEngine
	proposalSvc  ProposalService
	pool         *pgxpool.Pool
	metrics      *eval.MetricsCollector
	replaySvc    *eval.ReplayService
	ruleProvider llm.DecisionProvider
}

// CaseService defines the decision case operations needed by DecisionService.
type CaseService interface {
	CreateCaseFromAlert(ctx context.Context, alertID, createdBy string) (*decision.DecisionCase, error)
	GetCase(ctx context.Context, caseID string) (*decision.DecisionCase, error)
	ListCases(ctx context.Context, filter decision.CaseFilter) (*decision.CaseList, error)
}

// ContextBuilder defines the decision context building operation.
type ContextBuilder interface {
	BuildDecisionContext(ctx context.Context, caseID string) (*decision.DecisionContext, error)
}

// DecisionEngine defines the decision generation operation.
type DecisionEngine interface {
	GenerateDecision(ctx context.Context, caseID string, context *decision.DecisionContext) (*llm.DecisionOutput, error)
}

// ProposalService defines the action proposal operations needed by DecisionService.
type ProposalService interface {
	GenerateProposals(ctx context.Context, caseID, decisionID string, dec *llm.DecisionOutput, contextHash string) ([]action.ActionProposal, error)
	ListProposals(ctx context.Context, caseID string) ([]action.ActionProposal, error)
}

var (
	_ CaseService     = (*decision.CaseService)(nil)
	_ ContextBuilder  = (*decision.ContextBuilder)(nil)
	_ DecisionEngine  = (*decision.DecisionEngine)(nil)
	_ ProposalService = (*action.ProposalService)(nil)
)

// NewDecisionService creates a new DecisionService.
func NewDecisionService(
	caseSvc CaseService,
	ctxBuilder ContextBuilder,
	engine DecisionEngine,
	proposalSvc ProposalService,
	pool *pgxpool.Pool,
) *DecisionService {
	return &DecisionService{
		caseSvc:     caseSvc,
		ctxBuilder:  ctxBuilder,
		engine:      engine,
		proposalSvc: proposalSvc,
		pool:        pool,
	}
}

// WithMetrics attaches a MetricsCollector to the DecisionService.
func (s *DecisionService) WithMetrics(m *eval.MetricsCollector) *DecisionService {
	s.metrics = m
	return s
}

// WithReplayService attaches a ReplayService to the DecisionService.
func (s *DecisionService) WithReplayService(r *eval.ReplayService) *DecisionService {
	s.replaySvc = r
	return s
}

// WithRuleProvider attaches a rule-based DecisionProvider to the DecisionService.
func (s *DecisionService) WithRuleProvider(p llm.DecisionProvider) *DecisionService {
	s.ruleProvider = p
	return s
}

// CreateCaseFromAlert delegates to caseSvc.CreateCaseFromAlert and returns the case.
func (s *DecisionService) CreateCaseFromAlert(ctx context.Context, alertID, createdBy string) (*decision.DecisionCase, error) {
	return s.caseSvc.CreateCaseFromAlert(ctx, alertID, createdBy)
}

// BuildContext delegates to ctxBuilder.BuildDecisionContext and returns the context.
func (s *DecisionService) BuildContext(ctx context.Context, caseID string) (*decision.DecisionContext, error) {
	return s.ctxBuilder.BuildDecisionContext(ctx, caseID)
}

// Decide orchestrates the full decision workflow: get case, build context,
// generate decision, and create action proposals.
func (s *DecisionService) Decide(ctx context.Context, caseID string) (*decision.DecisionContext, *llm.DecisionOutput, []action.ActionProposal, error) {
	_, err := s.caseSvc.GetCase(ctx, caseID)
	if err != nil {
		return nil, nil, nil, err
	}

	decCtx, err := s.ctxBuilder.BuildDecisionContext(ctx, caseID)
	if err != nil {
		return nil, nil, nil, err
	}

	output, err := s.engine.GenerateDecision(ctx, caseID, decCtx)
	if err != nil {
		return nil, nil, nil, err
	}

	// Compute context hash for traceability: links this decision to its exact LLM context.
	contextHash := ""
	llmSafeCtx := llm.LLMSafeContext{
		CaseID: decCtx.DecisionCaseID,
		Trigger: llm.TriggerInfo{
			AlertID:       decCtx.Trigger.AlertID,
			RuleID:        decCtx.Trigger.RuleID,
			Severity:      decCtx.Trigger.Severity,
			MetricName:    decCtx.Trigger.MetricName,
			CurrentValue:  decCtx.Trigger.CurrentValue,
			BaselineValue: decCtx.Trigger.BaselineValue,
			DeltaPct:      decCtx.Trigger.DeltaPct,
		},
		ObjectContext: llm.ObjectContext{
			ObjectType: decCtx.ObjectContext.ObjectType,
			ObjectID:   decCtx.ObjectContext.ObjectID,
			Properties: decCtx.ObjectContext.Properties,
		},
		GovernanceInfo: llm.GovernanceInfo{
			Classification:   decCtx.Governance.Classification,
			RedactionApplied: decCtx.Governance.RedactionApplied,
			RedactedFields:   decCtx.Governance.RedactedFields,
			Role:             decCtx.Governance.Role,
		},
		AllowedActions:   decCtx.AllowedActions,
		ForbiddenActions: decCtx.ForbiddenActions,
	}
	if hash, err := decision.ComputeContextHash(llmSafeCtx); err == nil {
		contextHash = hash
	}

	decisionID := decision.GenerateDecisionID()
	proposals, err := s.proposalSvc.GenerateProposals(ctx, caseID, decisionID, output, contextHash)
	if err != nil {
		return nil, nil, nil, err
	}

	return decCtx, output, proposals, nil
}

// GetCase delegates to caseSvc.GetCase.
func (s *DecisionService) GetCase(ctx context.Context, caseID string) (*decision.DecisionCase, error) {
	return s.caseSvc.GetCase(ctx, caseID)
}

// ListCases delegates to caseSvc.ListCases.
func (s *DecisionService) ListCases(ctx context.Context, filter decision.CaseFilter) (*decision.CaseList, error) {
	return s.caseSvc.ListCases(ctx, filter)
}

// ListProposals delegates to proposalSvc.ListProposals.
func (s *DecisionService) ListProposals(ctx context.Context, caseID string) ([]action.ActionProposal, error) {
	return s.proposalSvc.ListProposals(ctx, caseID)
}

// DecideLLM is the explicit LLM decision path.
// It reuses the same Decide logic and adds LLM-specific metadata tracking.
func (s *DecisionService) DecideLLM(ctx context.Context, caseID string) (*decision.DecisionContext, *llm.DecisionOutput, []action.ActionProposal, error) {
	decCtx, output, proposals, err := s.Decide(ctx, caseID)
	if err != nil {
		return nil, nil, nil, err
	}

	// Track the decision in metrics.
	if s.metrics != nil {
		s.metrics.RecordDecision("llm", 0)
	}

	return decCtx, output, proposals, nil
}

// Compare runs both LLM and rule-based decisions and returns a comparison.
func (s *DecisionService) Compare(ctx context.Context, caseID string) (*eval.DecisionComparison, error) {
	_, err := s.caseSvc.GetCase(ctx, caseID)
	if err != nil {
		return nil, err
	}

	decCtx, err := s.ctxBuilder.BuildDecisionContext(ctx, caseID)
	if err != nil {
		return nil, err
	}

	// Generate LLM decision.
	llmOutput, err := s.engine.GenerateDecision(ctx, caseID, decCtx)
	if err != nil {
		return nil, err
	}

	// Generate rule-based decision for comparison.
	var ruleOutput *llm.DecisionOutput
	if s.ruleProvider != nil {
		safeCtx := llm.LLMSafeContext{
			CaseID: caseID,
		}
		ruleOutput, err = s.ruleProvider.GenerateDecision(ctx, safeCtx)
		if err != nil {
			ruleOutput = nil
		}
	}

	if ruleOutput == nil {
		ruleOutput = &llm.DecisionOutput{
			DecisionType: "monitor_only",
			Severity:     "low",
			Summary:      "rule-based decision unavailable",
			Confidence:   0.0,
		}
	}

	comparison := eval.Compare(caseID, llmOutput, ruleOutput)
	return comparison, nil
}

// Replay replays a previous decision. If dryRun is true, returns original
// decision data without calling the provider again.
func (s *DecisionService) Replay(ctx context.Context, caseID string, dryRun bool) (*eval.ReplayResult, error) {
	if s.replaySvc == nil {
		return nil, errors.New("replay service not configured")
	}
	return s.replaySvc.Replay(ctx, caseID, eval.ReplayOptions{DryRun: dryRun})
}

// ListLLMDecisions lists LLM decisions for a case.
// Returns a list of LLM decision records from ai.llm_decision table.
func (s *DecisionService) ListLLMDecisions(ctx context.Context, caseID string) (interface{}, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT decision_id, case_id, provider, model, confidence,
		       validation_status, fallback_used, output_json, created_at
		FROM ai.llm_decision
		WHERE case_id = $1
		ORDER BY created_at DESC
	`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type llmDecisionRow struct {
		DecisionID       string          `json:"decision_id"`
		CaseID           string          `json:"case_id"`
		Provider         string          `json:"provider"`
		Model            string          `json:"model"`
		Confidence       float64         `json:"confidence"`
		ValidationStatus string          `json:"validation_status"`
		FallbackUsed     bool            `json:"fallback_used"`
		OutputJSON       json.RawMessage `json:"output_json,omitempty"`
		CreatedAt        string          `json:"created_at"`
	}

	var decisions []llmDecisionRow
	for rows.Next() {
		var d llmDecisionRow
		var createdAt string
		if err := rows.Scan(&d.DecisionID, &d.CaseID, &d.Provider, &d.Model,
			&d.Confidence, &d.ValidationStatus, &d.FallbackUsed, &d.OutputJSON, &createdAt); err != nil {
			return nil, err
		}
		d.CreatedAt = createdAt
		decisions = append(decisions, d)
	}

	if decisions == nil {
		decisions = []llmDecisionRow{}
	}
	return decisions, nil
}

// ListEvals lists eval results for a case.
// Returns a list of decision_eval_result records from ai schema.
func (s *DecisionService) ListEvals(ctx context.Context, caseID string) (interface{}, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT eval_id, llm_decision_id, decision_case_id, eval_rule_id, eval_status, score, details_json, created_at
		FROM ai.decision_eval_result
		WHERE decision_case_id = $1
		ORDER BY created_at DESC
	`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type evalRow struct {
		EvalID        string          `json:"eval_id"`
		LlmDecisionID string          `json:"llm_decision_id"`
		CaseID        string          `json:"decision_case_id"`
		EvalRuleID    string          `json:"eval_rule_id"`
		EvalStatus    string          `json:"eval_status"`
		Score         float64         `json:"score,omitempty"`
		DetailsJSON   json.RawMessage `json:"details_json,omitempty"`
		CreatedAt     string          `json:"created_at"`
	}

	var evals []evalRow
	for rows.Next() {
		var e evalRow
		var createdAt string
		if err := rows.Scan(&e.EvalID, &e.LlmDecisionID, &e.CaseID, &e.EvalRuleID,
			&e.EvalStatus, &e.Score, &e.DetailsJSON, &createdAt); err != nil {
			return nil, err
		}
		e.CreatedAt = createdAt
		evals = append(evals, e)
	}

	if evals == nil {
		evals = []evalRow{}
	}
	return evals, nil
}
