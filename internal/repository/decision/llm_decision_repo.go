// Package decision provides repository access for the decision domain.
package decision

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"baxi/internal/repository/common"
)

// LLMDecision represents a simplified LLM decision with core fields.
// Maps to the ai.llm_decision table. DecisionJSON maps to the output_json column.
type LLMDecision struct {
	DecisionID   string           `json:"decision_id"`
	CaseID       string           `json:"case_id"`
	RecipeID     *string          `json:"recipe_id,omitempty"`
	ContextHash  *string          `json:"context_hash,omitempty"`
	DecisionJSON *json.RawMessage `json:"decision_json,omitempty"`
	Severity     *string          `json:"severity,omitempty"`
	Confidence   *float64         `json:"confidence,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
}

// LLMDecisionRepository provides focused data access for the ai.llm_decision table.
type LLMDecisionRepository struct {
	common.Querier
}

// NewLLMDecisionRepository creates a new LLMDecisionRepository.
func NewLLMDecisionRepository(querier common.Querier) *LLMDecisionRepository {
	return &LLMDecisionRepository{Querier: querier}
}

// CreateDecision inserts a new row into ai.llm_decision with core fields.
// Uses parameterized queries (no string interpolation).
func (r *LLMDecisionRepository) CreateDecision(ctx context.Context, d *LLMDecision) error {
	query := `
		INSERT INTO ai.llm_decision (
			decision_id, case_id, recipe_id, context_hash,
			output_json, severity, confidence, created_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8
		)
	`

	_, err := r.Exec(ctx, query,
		d.DecisionID,
		d.CaseID,
		d.RecipeID,
		d.ContextHash,
		d.DecisionJSON,
		d.Severity,
		d.Confidence,
		d.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert ai.llm_decision: %w", err)
	}
	return nil
}

// GetDecisionByID retrieves a single LLM decision by its decision_id.
// Uses parameterized queries (no string interpolation).
func (r *LLMDecisionRepository) GetDecisionByID(ctx context.Context, id string) (*LLMDecision, error) {
	query := `
		SELECT decision_id, case_id, recipe_id, context_hash,
		       output_json, severity, confidence, created_at
		FROM ai.llm_decision
		WHERE decision_id = $1
	`

	var d LLMDecision
	err := r.QueryRow(ctx, query, id).Scan(
		&d.DecisionID,
		&d.CaseID,
		&d.RecipeID,
		&d.ContextHash,
		&d.DecisionJSON,
		&d.Severity,
		&d.Confidence,
		&d.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("query ai.llm_decision by id: %w", err)
	}
	return &d, nil
}
