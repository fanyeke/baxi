package decision

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/repository/common"
)

func TestMockCreateCase(t *testing.T) {
	var capturedSQL string
	var capturedArgs []interface{}
	mock := &common.MockQuerier{
		ExecFunc: func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
			capturedSQL = sql
			capturedArgs = args
			return common.MockCommandTag(1), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	row := &DecisionCaseRow{
		CaseID:  "case-1",
		Status:  "open",
	}

	err := repo.CreateCase(ctx, row)
	require.NoError(t, err)
	assert.Contains(t, capturedSQL, "INSERT INTO ai.decision_case")
	assert.Equal(t, "case-1", capturedArgs[0])
}

func TestMockCreateCase_Error(t *testing.T) {
	mock := &common.MockQuerier{
		ExecFunc: func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, fmt.Errorf("duplicate key")
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	row := &DecisionCaseRow{CaseID: "case-1", Status: "open"}

	err := repo.CreateCase(ctx, row)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate key")
}

func TestMockGetCaseByID(t *testing.T) {
	mock := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRow(
				"case-1", nil, nil, "open", nil,
				"2026-01-01T00:00:00Z", nil,
				nil, nil, nil, nil,
				nil, nil, nil,
				nil, nil, nil,
				nil, nil, nil, nil, nil, nil,
			)
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	row, err := repo.GetCaseByID(ctx, "case-1")

	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "case-1", row.CaseID)
	assert.Equal(t, "open", row.Status)
}

func TestMockGetCaseByID_NotFound(t *testing.T) {
	mock := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRowError(fmt.Errorf("no rows in result set"))
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	row, err := repo.GetCaseByID(ctx, "missing")

	assert.Error(t, err)
	assert.Nil(t, row)
}

func TestMockUpdateCaseStatus(t *testing.T) {
	var capturedArgs []interface{}
	mock := &common.MockQuerier{
		ExecFunc: func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
			capturedArgs = args
			return common.MockCommandTag(1), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	err := repo.UpdateCaseStatus(ctx, "case-1", "resolved", nil, nil, nil)

	require.NoError(t, err)
	require.Len(t, capturedArgs, 5)
	assert.Equal(t, "resolved", capturedArgs[0])
	assert.Equal(t, "case-1", capturedArgs[4])
}

func TestMockUpdateCaseStatus_NotFound(t *testing.T) {
	mock := &common.MockQuerier{
		ExecFunc: func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
			return common.MockCommandTag(0), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	err := repo.UpdateCaseStatus(ctx, "missing", "resolved", nil, nil, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMockListCases(t *testing.T) {
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{
				{"c1", nil, nil, "open", nil, "2026-01-01T00:00:00Z", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, 2},
				{"c2", nil, nil, "resolved", nil, "2026-01-01T00:00:00Z", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, 2},
			}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	rows, total, err := repo.ListCases(ctx, CaseFilter{})

	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, rows, 2)
	assert.Equal(t, "c1", rows[0].CaseID)
}

func TestMockListCases_WithStatusFilter(t *testing.T) {
	var capturedSQL string
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			capturedSQL = sql
			return common.NewMockRows([][]interface{}{
				{"c1", nil, nil, "open", nil, "2026-01-01T00:00:00Z", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, 1},
			}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	status := "open"
	_, _, err := repo.ListCases(ctx, CaseFilter{Status: &status})

	require.NoError(t, err)
	assert.Contains(t, capturedSQL, "status = $")
}

func TestMockCreateDecision(t *testing.T) {
	mock := &common.MockQuerier{
		ExecFunc: func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
			return common.MockCommandTag(1), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	outputJSON := json.RawMessage(`{"action":"approve"}`)
	row := &LLMDecisionRow{
		DecisionID: "dec-1",
		CaseID:     "cd1",
		OutputJSON: &outputJSON,
	}

	err := repo.CreateDecision(ctx, row)
	require.NoError(t, err)
}

func TestMockCreateProposal(t *testing.T) {
	mock := &common.MockQuerier{
		ExecFunc: func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
			return common.MockCommandTag(1), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	row := &ActionProposalRow{
		ProposalID: "prop-1",
		CaseID:     "case-1",
		ActionType: "approve",
		Title:      "Approve order",
	}

	err := repo.CreateProposal(ctx, row)
	require.NoError(t, err)
}

func TestMockListProposalsByCase(t *testing.T) {
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{
				{"prop-1", "case-1", nil, "approve", nil, "pending", "2026-01-01T00:00:00Z", nil, nil, "Approve order", nil, nil, false},
			}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	proposals, err := repo.ListProposalsByCase(ctx, "case-1")

	require.NoError(t, err)
	assert.Len(t, proposals, 1)
	assert.Equal(t, "prop-1", proposals[0].ProposalID)
	assert.Equal(t, "approve", proposals[0].ActionType)
}
