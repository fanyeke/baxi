package review

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Sandbox represents a persistent proposal sandbox.
type Sandbox struct {
	SandboxID    string                 `json:"sandbox_id"`
	CaseID       string                 `json:"case_id"`
	ProposalID   *string                `json:"proposal_id,omitempty"`
	SandboxData  map[string]interface{} `json:"sandbox_data"`
	Status       string                 `json:"status"`
	ComparedWith []string               `json:"compared_with,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    *time.Time             `json:"updated_at,omitempty"`
}

// SandboxService manages persistent proposal sandboxes.
type SandboxService struct {
	pool *pgxpool.Pool
}

// NewSandboxService creates a new SandboxService.
func NewSandboxService(pool *pgxpool.Pool) *SandboxService {
	return &SandboxService{pool: pool}
}

// CreateSandbox creates a new sandbox with the given case ID and initial data.
func (s *SandboxService) CreateSandbox(ctx context.Context, caseID string, data map[string]interface{}) (string, error) {
	sandboxID := "sbx_" + uuid.NewString()

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshal sandbox data: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO ai.proposal_sandbox (sandbox_id, case_id, sandbox_data, status, created_at)
		VALUES ($1, $2, $3, 'draft', NOW())
	`, sandboxID, caseID, dataJSON)
	if err != nil {
		return "", fmt.Errorf("insert proposal_sandbox: %w", err)
	}

	return sandboxID, nil
}

// AddProposalToSandbox links a proposal to an existing sandbox.
func (s *SandboxService) AddProposalToSandbox(ctx context.Context, sandboxID, proposalID string) error {
	res, err := s.pool.Exec(ctx, `
		UPDATE ai.proposal_sandbox
		SET proposal_id = $1, updated_at = NOW()
		WHERE sandbox_id = $2
	`, proposalID, sandboxID)
	if err != nil {
		return fmt.Errorf("update proposal_sandbox: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("sandbox %s not found", sandboxID)
	}
	return nil
}

// CompareSandbox compares two sandboxes and returns their differences.
func (s *SandboxService) CompareSandbox(ctx context.Context, sandboxID1, sandboxID2 string) (*ComparisonResult, error) {
	sb1, err := s.GetSandbox(ctx, sandboxID1)
	if err != nil {
		return nil, fmt.Errorf("get sandbox 1: %w", err)
	}
	if sb1 == nil {
		return nil, fmt.Errorf("sandbox %s not found", sandboxID1)
	}

	sb2, err := s.GetSandbox(ctx, sandboxID2)
	if err != nil {
		return nil, fmt.Errorf("get sandbox 2: %w", err)
	}
	if sb2 == nil {
		return nil, fmt.Errorf("sandbox %s not found", sandboxID2)
	}

	diffs := compareData(sb1.SandboxData, sb2.SandboxData)

	return &ComparisonResult{
		Sandbox1ID: sandboxID1,
		Sandbox2ID: sandboxID2,
		Differences: diffs,
	}, nil
}

// GetSandbox retrieves a sandbox by its ID.
func (s *SandboxService) GetSandbox(ctx context.Context, sandboxID string) (*Sandbox, error) {
	var sb Sandbox
	var dataJSON []byte
	var proposalID *string
	var updatedAt *time.Time

	err := s.pool.QueryRow(ctx, `
		SELECT sandbox_id, case_id, proposal_id, sandbox_data, status, compared_with, created_at, updated_at
		FROM ai.proposal_sandbox
		WHERE sandbox_id = $1
	`, sandboxID).Scan(
		&sb.SandboxID,
		&sb.CaseID,
		&proposalID,
		&dataJSON,
		&sb.Status,
		&sb.ComparedWith,
		&sb.CreatedAt,
		&updatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query proposal_sandbox: %w", err)
	}

	sb.ProposalID = proposalID
	sb.UpdatedAt = updatedAt
	if len(dataJSON) > 0 {
		if err := json.Unmarshal(dataJSON, &sb.SandboxData); err != nil {
			sb.SandboxData = make(map[string]interface{})
		}
	} else {
		sb.SandboxData = make(map[string]interface{})
	}
	if sb.ComparedWith == nil {
		sb.ComparedWith = []string{}
	}

	return &sb, nil
}

// ComparisonResult holds the result of comparing two sandboxes.
type ComparisonResult struct {
	Sandbox1ID  string
	Sandbox2ID  string
	Differences []Difference
}

// Difference represents a single field difference between two sandboxes.
type Difference struct {
	Field  string
	Value1 interface{}
	Value2 interface{}
}

// compareData compares two data maps and returns a list of differences.
func compareData(d1, d2 map[string]interface{}) []Difference {
	var diffs []Difference

	allKeys := make(map[string]bool)
	for k := range d1 {
		allKeys[k] = true
	}
	for k := range d2 {
		allKeys[k] = true
	}

	for k := range allKeys {
		v1, has1 := d1[k]
		v2, has2 := d2[k]

		if !has1 {
			diffs = append(diffs, Difference{Field: k, Value1: nil, Value2: v2})
			continue
		}
		if !has2 {
			diffs = append(diffs, Difference{Field: k, Value1: v1, Value2: nil})
			continue
		}

		v1Str, _ := json.Marshal(v1)
		v2Str, _ := json.Marshal(v2)
		if string(v1Str) != string(v2Str) {
			diffs = append(diffs, Difference{Field: k, Value1: v1, Value2: v2})
		}
	}

	return diffs
}
