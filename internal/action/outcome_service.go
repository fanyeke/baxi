package action

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ActionOutcome represents a single outcome record for an executed action proposal.
type ActionOutcome struct {
	OutcomeID        string                 `json:"outcome_id"`
	CaseID           string                 `json:"case_id"`
	ProposalID       string                 `json:"proposal_id"`
	ActionType       string                 `json:"action_type"`
	ExecutionStatus  string                 `json:"execution_status"`
	BusinessResult   string                 `json:"business_result,omitempty"`
	BusinessImpact   map[string]interface{} `json:"business_impact,omitempty"`
	IsEffective      *bool                  `json:"is_effective,omitempty"`
	RecordedBy       string                 `json:"recorded_by"`
	RecordedAt       time.Time              `json:"recorded_at"`
	Notes            string                 `json:"notes,omitempty"`
}

// RecordOutcomeInput is the input for recording an action outcome.
type RecordOutcomeInput struct {
	CaseID          string
	ProposalID      string
	ActionType      string
	ExecutionStatus string
	BusinessResult  string
	BusinessImpact  map[string]interface{}
	IsEffective     *bool
	RecordedBy      string
	Notes           string
}

// OutcomeService provides outcome recording and retrieval for action proposals.
type OutcomeService struct {
	pool *pgxpool.Pool
}

// NewOutcomeService creates a new OutcomeService.
func NewOutcomeService(pool *pgxpool.Pool) *OutcomeService {
	return &OutcomeService{pool: pool}
}

// RecordOutcome persists an action outcome.
func (s *OutcomeService) RecordOutcome(ctx context.Context, input RecordOutcomeInput) (*ActionOutcome, error) {
	outcomeID := fmt.Sprintf("out_%d", time.Now().UnixNano())

	var impactJSON *json.RawMessage
	if input.BusinessImpact != nil {
		data, err := json.Marshal(input.BusinessImpact)
		if err != nil {
			return nil, fmt.Errorf("marshal business impact: %w", err)
		}
		raw := json.RawMessage(data)
		impactJSON = &raw
	}

	query := `
		INSERT INTO ai.action_outcome (
			outcome_id, case_id, proposal_id, action_type,
			execution_status, business_result, business_impact_json,
			is_effective, recorded_by, recorded_at, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), $10)
	`

	_, err := s.pool.Exec(ctx, query,
		outcomeID, input.CaseID, input.ProposalID, input.ActionType,
		input.ExecutionStatus, input.BusinessResult, impactJSON,
		input.IsEffective, input.RecordedBy, input.Notes,
	)
	if err != nil {
		return nil, fmt.Errorf("insert action outcome: %w", err)
	}

	return &ActionOutcome{
		OutcomeID:        outcomeID,
		CaseID:           input.CaseID,
		ProposalID:       input.ProposalID,
		ActionType:       input.ActionType,
		ExecutionStatus:  input.ExecutionStatus,
		BusinessResult:   input.BusinessResult,
		BusinessImpact:   input.BusinessImpact,
		IsEffective:      input.IsEffective,
		RecordedBy:       input.RecordedBy,
		RecordedAt:       time.Now(),
		Notes:            input.Notes,
	}, nil
}

// GetOutcomeByProposal retrieves an outcome by proposal ID.
func (s *OutcomeService) GetOutcomeByProposal(ctx context.Context, proposalID string) (*ActionOutcome, error) {
	query := `
		SELECT outcome_id, case_id, proposal_id, action_type,
		       execution_status, business_result, business_impact_json,
		       is_effective, recorded_by, recorded_at, notes
		FROM ai.action_outcome
		WHERE proposal_id = $1
	`

	var outcome ActionOutcome
	var businessResult, recordedBy, notes *string
	var businessImpactJSON *json.RawMessage
	var recordedAt time.Time

	err := s.pool.QueryRow(ctx, query, proposalID).Scan(
		&outcome.OutcomeID, &outcome.CaseID, &outcome.ProposalID, &outcome.ActionType,
		&outcome.ExecutionStatus, &businessResult, &businessImpactJSON,
		&outcome.IsEffective, &recordedBy, &recordedAt, &notes,
	)
	if err != nil {
		return nil, fmt.Errorf("query action outcome: %w", err)
	}

	if businessResult != nil {
		outcome.BusinessResult = *businessResult
	}
	if recordedBy != nil {
		outcome.RecordedBy = *recordedBy
	}
	if notes != nil {
		outcome.Notes = *notes
	}
	if businessImpactJSON != nil {
		json.Unmarshal(*businessImpactJSON, &outcome.BusinessImpact)
	}
	outcome.RecordedAt = recordedAt

	return &outcome, nil
}

// GetOutcomesByCase retrieves all outcomes for a given case.
func (s *OutcomeService) GetOutcomesByCase(ctx context.Context, caseID string) ([]ActionOutcome, error) {
	query := `
		SELECT outcome_id, case_id, proposal_id, action_type,
		       execution_status, business_result, business_impact_json,
		       is_effective, recorded_by, recorded_at, notes
		FROM ai.action_outcome
		WHERE case_id = $1
		ORDER BY recorded_at DESC
	`

	rows, err := s.pool.Query(ctx, query, caseID)
	if err != nil {
		return nil, fmt.Errorf("query action outcomes: %w", err)
	}
	defer rows.Close()

	var outcomes []ActionOutcome
	for rows.Next() {
		var o ActionOutcome
		var businessResult, recordedBy, notes *string
		var businessImpactJSON *json.RawMessage
		var recordedAt time.Time

		if err := rows.Scan(
			&o.OutcomeID, &o.CaseID, &o.ProposalID, &o.ActionType,
			&o.ExecutionStatus, &businessResult, &businessImpactJSON,
			&o.IsEffective, &recordedBy, &recordedAt, &notes,
		); err != nil {
			return nil, fmt.Errorf("scan action outcome: %w", err)
		}

		if businessResult != nil {
			o.BusinessResult = *businessResult
		}
		if recordedBy != nil {
			o.RecordedBy = *recordedBy
		}
		if notes != nil {
			o.Notes = *notes
		}
		if businessImpactJSON != nil {
			json.Unmarshal(*businessImpactJSON, &o.BusinessImpact)
		}
		o.RecordedAt = recordedAt

		outcomes = append(outcomes, o)
	}

	if outcomes == nil {
		outcomes = []ActionOutcome{}
	}
	return outcomes, nil
}
