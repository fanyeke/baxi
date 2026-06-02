// Package decision provides repository access for the decision domain.
// This is a domain subpackage of the repository layer with pool injection.
package decision

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"baxi/internal/repository/common"
)

// DecisionCaseRow represents a single row from ai.decision_case.
type DecisionCaseRow struct {
	CaseID                 string
	AlertID                *string
	CaseType               *string
	Status                 string
	ContextJSON            *json.RawMessage
	CreatedAt              time.Time
	ResolvedAt             *time.Time
	SourceType             *string
	SourceID               *string
	ObjectType             *string
	ObjectID               *string
	Severity               *string
	ContextHash            *string
	GovernanceSnapshotJSON *json.RawMessage
	CreatedBy              *string
	ErrorMessage           *string
	UpdatedAt              *time.Time
	AlertRulesVersion      *string
	AlertRulesHash         *string
	ActionRegistryVersion  *string
	ActionRegistryHash     *string
	ContextSnapshotJSON    *json.RawMessage
	DataSnapshotJSON       *json.RawMessage
}

// LLMDecisionRow represents a single row from ai.llm_decision.
type LLMDecisionRow struct {
	DecisionID       string
	CaseID           string
	ModelVersion     *string
	PromptHash       *string
	OutputJSON       *json.RawMessage
	Confidence       *float64
	CreatedAt        time.Time
	Status           *string
	FallbackReason   *string
	ValidationErrors *json.RawMessage
	RecipeID         *string
	ContextHash      *string
	Severity         *string
}

// ActionProposalRow represents a single row from ai.action_proposal.
type ActionProposalRow struct {
	ProposalID          string
	CaseID              string
	DecisionID          *string
	ActionType          string
	Payload             *json.RawMessage
	ApplyStatus         string
	CreatedAt           time.Time
	AppliedAt           *time.Time
	AppliedBy           *string
	Title               string
	Description         *string
	RiskLevel           *string
	RequiresHumanReview bool
	ContextHash         *string // Phase 2: links proposal to the exact LLM context used
	ActionSchemaVersion *string // Phase 4: links proposal to the action schema version
	EvidenceRefs        *string // JSON array of evidence reference IDs
	RecipeID            *string // recipe that triggered this decision
}

// CaseFilter holds optional WHERE clause filters for listing decision cases.
// Only non-nil fields are applied to the query.
type CaseFilter struct {
	SourceType *string
	SourceID   *string
	Status     *string
	Severity   *string
	Limit      int
	Offset     int
}

// Repository provides data access for ai.decision_case, ai.llm_decision,
// and ai.action_proposal tables.
type Repository struct {
	common.Querier
}

// NewRepository creates a new decision Repository.
func NewRepository(querier common.Querier) *Repository {
	return &Repository{Querier: querier}
}

// CreateCase inserts a new row into ai.decision_case.
func (r *Repository) CreateCase(ctx context.Context, row *DecisionCaseRow) error {
	query := `
		INSERT INTO ai.decision_case (
			case_id, alert_id, case_type, status, context_json,
			created_at, resolved_at,
			source_type, source_id, object_type, object_id,
			severity, context_hash, governance_snapshot_json,
			created_by, error_message, updated_at,
			alert_rules_version, alert_rules_hash,
			action_registry_version, action_registry_hash,
			context_snapshot_json, data_snapshot_json
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7,
			$8, $9, $10, $11,
			$12, $13, $14,
			$15, $16, $17,
			$18, $19,
			$20, $21,
			$22, $23
		)
	`

	_, err := r.Exec(ctx, query,
		row.CaseID,
		row.AlertID,
		row.CaseType,
		row.Status,
		row.ContextJSON,
		row.CreatedAt,
		row.ResolvedAt,
		row.SourceType,
		row.SourceID,
		row.ObjectType,
		row.ObjectID,
		row.Severity,
		row.ContextHash,
		row.GovernanceSnapshotJSON,
		row.CreatedBy,
		row.ErrorMessage,
		row.UpdatedAt,
		row.AlertRulesVersion,
		row.AlertRulesHash,
		row.ActionRegistryVersion,
		row.ActionRegistryHash,
		row.ContextSnapshotJSON,
		row.DataSnapshotJSON,
	)
	if err != nil {
		return fmt.Errorf("insert ai.decision_case: %w", err)
	}
	return nil
}

// GetCaseByID retrieves a single decision case by its case_id.
func (r *Repository) GetCaseByID(ctx context.Context, caseID string) (*DecisionCaseRow, error) {
	query := `
		SELECT case_id, alert_id, case_type, status, context_json,
		       created_at, resolved_at,
		       source_type, source_id, object_type, object_id,
		       severity, context_hash, governance_snapshot_json,
		       created_by, error_message, updated_at,
		       alert_rules_version, alert_rules_hash,
		       action_registry_version, action_registry_hash,
		       context_snapshot_json, data_snapshot_json
		FROM ai.decision_case
		WHERE case_id = $1
	`

	var row DecisionCaseRow
	err := r.QueryRow(ctx, query, caseID).Scan(
		&row.CaseID,
		&row.AlertID,
		&row.CaseType,
		&row.Status,
		&row.ContextJSON,
		&row.CreatedAt,
		&row.ResolvedAt,
		&row.SourceType,
		&row.SourceID,
		&row.ObjectType,
		&row.ObjectID,
		&row.Severity,
		&row.ContextHash,
		&row.GovernanceSnapshotJSON,
		&row.CreatedBy,
		&row.ErrorMessage,
		&row.UpdatedAt,
		&row.AlertRulesVersion,
		&row.AlertRulesHash,
		&row.ActionRegistryVersion,
		&row.ActionRegistryHash,
		&row.ContextSnapshotJSON,
		&row.DataSnapshotJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("query ai.decision_case by id: %w", err)
	}
	return &row, nil
}

// GetCaseBySource retrieves a single decision case by source_type and source_id.
// If sourceType or sourceID is nil, matches NULL in the database.
func (r *Repository) GetCaseBySource(ctx context.Context, sourceType, sourceID string) (*DecisionCaseRow, error) {
	query := `
		SELECT case_id, alert_id, case_type, status, context_json,
		       created_at, resolved_at,
		       source_type, source_id, object_type, object_id,
		       severity, context_hash, governance_snapshot_json,
		       created_by, error_message, updated_at,
		       alert_rules_version, alert_rules_hash,
		       action_registry_version, action_registry_hash,
		       context_snapshot_json, data_snapshot_json
		FROM ai.decision_case
		WHERE (source_type = $1 OR (source_type IS NULL AND $1 IS NULL))
		  AND (source_id = $2 OR (source_id IS NULL AND $2 IS NULL))
	`

	var (
		sType *string
		sID   *string
	)
	if sourceType != "" {
		sType = &sourceType
	}
	if sourceID != "" {
		sID = &sourceID
	}

	var row DecisionCaseRow
	err := r.QueryRow(ctx, query, sType, sID).Scan(
		&row.CaseID,
		&row.AlertID,
		&row.CaseType,
		&row.Status,
		&row.ContextJSON,
		&row.CreatedAt,
		&row.ResolvedAt,
		&row.SourceType,
		&row.SourceID,
		&row.ObjectType,
		&row.ObjectID,
		&row.Severity,
		&row.ContextHash,
		&row.GovernanceSnapshotJSON,
		&row.CreatedBy,
		&row.ErrorMessage,
		&row.UpdatedAt,
		&row.AlertRulesVersion,
		&row.AlertRulesHash,
		&row.ActionRegistryVersion,
		&row.ActionRegistryHash,
		&row.ContextSnapshotJSON,
		&row.DataSnapshotJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("query ai.decision_case by source: %w", err)
	}
	return &row, nil
}

// UpdateCaseStatus updates the status and related metadata for a decision case.
func (r *Repository) UpdateCaseStatus(
	ctx context.Context,
	caseID string,
	status string,
	contextJSON *json.RawMessage,
	contextHash *string,
	governanceSnapshot *json.RawMessage,
) error {
	query := `
		UPDATE ai.decision_case
		SET status = $1,
		    context_json = $2,
		    context_hash = $3,
		    governance_snapshot_json = $4,
		    updated_at = NOW()
		WHERE case_id = $5
	`

	res, err := r.Exec(ctx, query, status, contextJSON, contextHash, governanceSnapshot, caseID)
	if err != nil {
		return fmt.Errorf("update ai.decision_case status: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("decision case %s not found", caseID)
	}
	return nil
}

// ListCases queries ai.decision_case with optional filters and pagination.
// Uses COUNT(*) OVER() to return the total count in a single query.
// Results are ordered by created_at DESC.
func (r *Repository) ListCases(
	ctx context.Context,
	filter CaseFilter,
) ([]DecisionCaseRow, int, error) {
	query := `
		SELECT case_id, alert_id, case_type, status, context_json,
		       created_at, resolved_at,
		       source_type, source_id, object_type, object_id,
		       severity, context_hash, governance_snapshot_json,
		       created_by, error_message, updated_at,
		       alert_rules_version, alert_rules_hash,
		       action_registry_version, action_registry_hash,
		       context_snapshot_json, data_snapshot_json,
		       COUNT(*) OVER() AS total_count
		FROM ai.decision_case
		WHERE 1=1`

	args := make([]interface{}, 0, 6)
	argIdx := 1

	if filter.SourceType != nil {
		query += fmt.Sprintf(" AND source_type = $%d", argIdx)
		args = append(args, *filter.SourceType)
		argIdx++
	}
	if filter.SourceID != nil {
		query += fmt.Sprintf(" AND source_id = $%d", argIdx)
		args = append(args, *filter.SourceID)
		argIdx++
	}
	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.Severity != nil {
		query += fmt.Sprintf(" AND severity = $%d", argIdx)
		args = append(args, *filter.Severity)
		argIdx++
	}

	query += " ORDER BY created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query ai.decision_case: %w", err)
	}
	defer rows.Close()

	var results []DecisionCaseRow
	var total int

	for rows.Next() {
		var row DecisionCaseRow
		var rowTotal int
		if err := rows.Scan(
			&row.CaseID,
			&row.AlertID,
			&row.CaseType,
			&row.Status,
			&row.ContextJSON,
			&row.CreatedAt,
			&row.ResolvedAt,
			&row.SourceType,
			&row.SourceID,
			&row.ObjectType,
			&row.ObjectID,
			&row.Severity,
			&row.ContextHash,
			&row.GovernanceSnapshotJSON,
			&row.CreatedBy,
			&row.ErrorMessage,
			&row.UpdatedAt,
			&row.AlertRulesVersion,
			&row.AlertRulesHash,
			&row.ActionRegistryVersion,
			&row.ActionRegistryHash,
			&row.ContextSnapshotJSON,
			&row.DataSnapshotJSON,
			&rowTotal,
		); err != nil {
			return nil, 0, fmt.Errorf("scan decision_case row: %w", err)
		}
		results = append(results, row)
		total = rowTotal
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate decision_case rows: %w", err)
	}

	return results, total, nil
}

// CreateDecision inserts a new row into ai.llm_decision.
func (r *Repository) CreateDecision(ctx context.Context, row *LLMDecisionRow) error {
	query := `
		INSERT INTO ai.llm_decision (
			decision_id, case_id, model_version, prompt_hash,
			output_json, confidence, created_at,
			status, fallback_reason, validation_errors,
			recipe_id, context_hash, severity
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7,
			$8, $9, $10,
			$11, $12, $13
		)
	`

	_, err := r.Exec(ctx, query,
		row.DecisionID,
		row.CaseID,
		row.ModelVersion,
		row.PromptHash,
		row.OutputJSON,
		row.Confidence,
		row.CreatedAt,
		row.Status,
		row.FallbackReason,
		row.ValidationErrors,
		row.RecipeID,
		row.ContextHash,
		row.Severity,
	)
	if err != nil {
		return fmt.Errorf("insert ai.llm_decision: %w", err)
	}
	return nil
}

// GetDecisionByID retrieves a single LLM decision by its decision_id.
func (r *Repository) GetDecisionByID(ctx context.Context, decisionID string) (*LLMDecisionRow, error) {
	query := `
		SELECT decision_id, case_id, model_version, prompt_hash,
		       output_json, confidence, created_at,
		       status, fallback_reason, validation_errors,
		       recipe_id, context_hash, severity
		FROM ai.llm_decision
		WHERE decision_id = $1
	`

	var row LLMDecisionRow
	err := r.QueryRow(ctx, query, decisionID).Scan(
		&row.DecisionID,
		&row.CaseID,
		&row.ModelVersion,
		&row.PromptHash,
		&row.OutputJSON,
		&row.Confidence,
		&row.CreatedAt,
		&row.Status,
		&row.FallbackReason,
		&row.ValidationErrors,
		&row.RecipeID,
		&row.ContextHash,
		&row.Severity,
	)
	if err != nil {
		return nil, fmt.Errorf("query ai.llm_decision by id: %w", err)
	}
	return &row, nil
}

// CreateProposal inserts a new row into ai.action_proposal.
func (r *Repository) CreateProposal(ctx context.Context, row *ActionProposalRow) error {
	query := `
		INSERT INTO ai.action_proposal (
			proposal_id, case_id, decision_id, action_type,
			payload, apply_status, created_at,
			applied_at, applied_by,
			title, description, risk_level, requires_human_review,
			context_hash, action_schema_version,
			evidence_refs, recipe_id
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7,
			$8, $9,
			$10, $11, $12, $13,
			$14, $15,
			$16, $17
		)
	`

	_, err := r.Exec(ctx, query,
		row.ProposalID,
		row.CaseID,
		row.DecisionID,
		row.ActionType,
		row.Payload,
		row.ApplyStatus,
		row.CreatedAt,
		row.AppliedAt,
		row.AppliedBy,
		row.Title,
		row.Description,
		row.RiskLevel,
		row.RequiresHumanReview,
		row.ContextHash,
		row.ActionSchemaVersion,
		row.EvidenceRefs,
		row.RecipeID,
	)
	if err != nil {
		return fmt.Errorf("insert ai.action_proposal: %w", err)
	}
	return nil
}

// ListProposalsByCase retrieves all action proposals for a given case.
// Results are ordered by created_at ASC.
func (r *Repository) ListProposalsByCase(ctx context.Context, caseID string) ([]ActionProposalRow, error) {
	query := `
		SELECT proposal_id, case_id, decision_id, action_type,
		       payload, apply_status, created_at,
		       applied_at, applied_by,
		       title, description, risk_level, requires_human_review,
		       context_hash, action_schema_version,
		       evidence_refs, recipe_id
		FROM ai.action_proposal
		WHERE case_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.Query(ctx, query, caseID)
	if err != nil {
		return nil, fmt.Errorf("query ai.action_proposal by case: %w", err)
	}
	defer rows.Close()

	var results []ActionProposalRow
	for rows.Next() {
		var row ActionProposalRow
		if err := rows.Scan(
			&row.ProposalID,
			&row.CaseID,
			&row.DecisionID,
			&row.ActionType,
			&row.Payload,
			&row.ApplyStatus,
			&row.CreatedAt,
			&row.AppliedAt,
			&row.AppliedBy,
			&row.Title,
			&row.Description,
			&row.RiskLevel,
			&row.RequiresHumanReview,
			&row.ContextHash,
			&row.ActionSchemaVersion,
			&row.EvidenceRefs,
			&row.RecipeID,
		); err != nil {
			return nil, fmt.Errorf("scan action_proposal row: %w", err)
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate action_proposal rows: %w", err)
	}

	if results == nil {
		results = []ActionProposalRow{}
	}

	return results, nil
}
