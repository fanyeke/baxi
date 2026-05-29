package action

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"baxi/internal/decision"
	"baxi/internal/llm"
	"baxi/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ActionProposal is the domain model for ai.action_proposal.
type ActionProposal struct {
	ProposalID          string                 `json:"proposal_id"`
	CaseID              string                 `json:"case_id"`
	DecisionID          string                 `json:"decision_id"`
	ActionType          string                 `json:"action_type"`
	Title               string                 `json:"title"`
	Description         string                 `json:"description"`
	Payload             map[string]interface{} `json:"payload"`
	RiskLevel           string                 `json:"risk_level"`
	RequiresHumanReview bool                   `json:"requires_human_review"`
	ApplyStatus         string                 `json:"apply_status"`
	CreatedAt           time.Time              `json:"created_at"`
}

// ProposalRepository defines the interface for action proposal storage operations.
type ProposalRepository interface {
	CreateProposal(ctx context.Context, pool *pgxpool.Pool, row *repository.ActionProposalRow) error
	ListProposalsByCase(ctx context.Context, pool *pgxpool.Pool, caseID string) ([]repository.ActionProposalRow, error)
}

// CaseStatusUpdater defines the interface for updating decision case status.
type CaseStatusUpdater interface {
	UpdateCaseStatus(ctx context.Context, pool *pgxpool.Pool, caseID string, status string, contextJSON *json.RawMessage, contextHash *string, governanceSnapshot *json.RawMessage) error
}

// ProposalService generates and manages action proposals from decisions.
type ProposalService struct {
	repo     ProposalRepository
	caseSvc  CaseStatusUpdater
	registry *ActionRegistry
	pool     *pgxpool.Pool
}

// NewProposalService creates a new ProposalService.
func NewProposalService(repo ProposalRepository, caseSvc CaseStatusUpdater, registry *ActionRegistry, pool *pgxpool.Pool) *ProposalService {
	return &ProposalService{
		repo:     repo,
		caseSvc:  caseSvc,
		registry: registry,
		pool:     pool,
	}
}

// GenerateProposals creates action proposals from a decision output.
// For each recommended action, a proposal is created and persisted.
// The case status is then updated to "proposal_generated".
func (s *ProposalService) GenerateProposals(ctx context.Context, caseID, decisionID string, dec *llm.DecisionOutput, contextHash string) ([]ActionProposal, error) {
	var proposals []ActionProposal

	for _, action := range dec.RecommendedActions {
		proposalID := decision.GenerateProposalID()

		// Phase 4: Validate payload against action registry schema
		if s.registry != nil {
			if errs := s.registry.ValidatePayload(action.ActionType, action.Payload); len(errs) > 0 {
				// Payload invalid — skip this proposal, log validation failure
				continue
			}
		}

		// Build title: "{action_type}: {decision.summary}" truncated to 200 chars
		title := fmt.Sprintf("%s: %s", action.ActionType, dec.Summary)
		if len(title) > 200 {
			title = title[:200]
		}

		// Build description from rationale
		description := strings.Join(dec.Rationale, "; ")

		// Map severity to risk level
		riskLevel := mapRiskLevel(dec.Severity)

		// Marshal payload if present
		var payloadJSON *json.RawMessage
		if action.Payload != nil {
			raw, err := json.Marshal(action.Payload)
			if err != nil {
				return nil, fmt.Errorf("marshal payload for action %s: %w", action.ActionType, err)
			}
			msg := json.RawMessage(raw)
			payloadJSON = &msg
		}

		// Phase 4: Resolve action schema version
		schemaVersion := "v1"
		if s.registry != nil {
			if cfg, ok := s.registry.GetActionConfig(action.ActionType); ok && cfg.Version != "" {
				schemaVersion = cfg.Version
			}
		}

		// Phase 6: all proposals require human review
		row := &repository.ActionProposalRow{
			ProposalID:          proposalID,
			CaseID:              caseID,
			DecisionID:          &decisionID,
			ActionType:          action.ActionType,
			Payload:             payloadJSON,
			ApplyStatus:         "proposed",
			CreatedAt:           time.Now(),
			Title:               title,
			Description:         &description,
			RiskLevel:           &riskLevel,
			RequiresHumanReview: true,
			ContextHash:         &contextHash,
			ActionSchemaVersion: &schemaVersion,
		}

		if err := s.repo.CreateProposal(ctx, s.pool, row); err != nil {
			return nil, fmt.Errorf("create proposal for action %s: %w", action.ActionType, err)
		}

		proposals = append(proposals, *rowToProposal(row))
	}

	// Update case status to signal proposals have been generated
	if err := s.caseSvc.UpdateCaseStatus(ctx, s.pool, caseID, "proposal_generated", nil, nil, nil); err != nil {
		return nil, fmt.Errorf("update case status to proposal_generated: %w", err)
	}

	if proposals == nil {
		proposals = []ActionProposal{}
	}

	return proposals, nil
}

// ListProposals returns all action proposals for a given case.
func (s *ProposalService) ListProposals(ctx context.Context, caseID string) ([]ActionProposal, error) {
	rows, err := s.repo.ListProposalsByCase(ctx, s.pool, caseID)
	if err != nil {
		return nil, fmt.Errorf("list proposals for case %s: %w", caseID, err)
	}

	proposals := make([]ActionProposal, len(rows))
	for i := range rows {
		proposals[i] = *rowToProposal(&rows[i])
	}

	return proposals, nil
}

// rowToProposal maps an ActionProposalRow to an ActionProposal domain struct.
func rowToProposal(row *repository.ActionProposalRow) *ActionProposal {
	p := &ActionProposal{
		ProposalID:          row.ProposalID,
		CaseID:              row.CaseID,
		ActionType:          row.ActionType,
		Title:               row.Title,
		ApplyStatus:         row.ApplyStatus,
		CreatedAt:           row.CreatedAt,
		RequiresHumanReview: row.RequiresHumanReview,
	}

	if row.DecisionID != nil {
		p.DecisionID = *row.DecisionID
	}
	if row.Description != nil {
		p.Description = *row.Description
	}
	if row.RiskLevel != nil {
		p.RiskLevel = *row.RiskLevel
	}
	if row.Payload != nil {
		var payload map[string]interface{}
		if err := json.Unmarshal(*row.Payload, &payload); err == nil {
			p.Payload = payload
		}
	}

	return p
}

// mapRiskLevel maps a decision severity to a risk level string.
func mapRiskLevel(severity string) string {
	switch severity {
	case "critical", "high":
		return "high"
	case "medium":
		return "medium"
	case "low":
		return "low"
	default:
		return "medium"
	}
}
