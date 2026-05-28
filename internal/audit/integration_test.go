package audit

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"baxi/internal/testutil"
)

func setupTestDB(t *testing.T) (*testutil.PostgresContainer, *pgxpool.Pool, pgx.Tx) {
	t.Helper()
	ctx := context.Background()

	pg, err := testutil.StartPostgres(ctx)
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, pg.ConnStr)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, "CREATE SCHEMA IF NOT EXISTS audit")
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		CREATE TABLE audit.audit_log (
			audit_id      BIGSERIAL PRIMARY KEY,
			category      TEXT,
			action        TEXT,
			actor         TEXT,
			resource_type TEXT,
			resource_id   TEXT,
			metadata      JSONB,
			created_at    TIMESTAMPTZ DEFAULT NOW()
		)
	`)
	require.NoError(t, err)

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		tx.Rollback(ctx)
		pool.Close()
		pg.Terminate(ctx)
	})

	return pg, pool, tx
}

func TestAuditIntegration_LogProposalReviewed(t *testing.T) {
	ctx := context.Background()
	_, pool, tx := setupTestDB(t)

	audit := NewIntegration()

	err := audit.LogProposalReviewed(ctx, tx, "prop_001", "reviewer_alice", "approve", "Looks good")
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))

	var category, action, actor, resourceType, resourceID string
	var metadataJSON []byte
	err = pool.QueryRow(ctx, `
		SELECT category, action, actor, resource_type, resource_id, metadata
		FROM audit.audit_log
		WHERE resource_id = 'prop_001'
	`).Scan(&category, &action, &actor, &resourceType, &resourceID, &metadataJSON)
	require.NoError(t, err)

	assert.Equal(t, "review", category)
	assert.Equal(t, "proposal_reviewed", action)
	assert.Equal(t, "reviewer_alice", actor)
	assert.Equal(t, "action_proposal", resourceType)
	assert.Equal(t, "prop_001", resourceID)

	var metadata map[string]interface{}
	require.NoError(t, json.Unmarshal(metadataJSON, &metadata))
	assert.Equal(t, "approve", metadata["verdict"])
	assert.Equal(t, "Looks good", metadata["feedback"])
}

func TestAuditIntegration_LogProposalExecuted_DryRun(t *testing.T) {
	ctx := context.Background()
	_, pool, tx := setupTestDB(t)

	audit := NewIntegration()

	err := audit.LogProposalExecuted(ctx, tx, "prop_002", "actor_bob", true, true, "")
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))

	var category, action, actor, resourceType, resourceID string
	var metadataJSON []byte
	err = pool.QueryRow(ctx, `
		SELECT category, action, actor, resource_type, resource_id, metadata
		FROM audit.audit_log
		WHERE resource_id = 'prop_002'
	`).Scan(&category, &action, &actor, &resourceType, &resourceID, &metadataJSON)
	require.NoError(t, err)

	assert.Equal(t, "action_apply", category)
	assert.Equal(t, "proposal_executed", action)
	assert.Equal(t, "actor_bob", actor)
	assert.Equal(t, "action_proposal", resourceType)

	var metadata map[string]interface{}
	require.NoError(t, json.Unmarshal(metadataJSON, &metadata))
	assert.Equal(t, true, metadata["success"])
	assert.Equal(t, true, metadata["dry_run"])
	_, hasError := metadata["error"]
	assert.False(t, hasError, "error should be omitted when empty")
}

func TestAuditIntegration_LogProposalExecuted_Failed(t *testing.T) {
	ctx := context.Background()
	_, pool, tx := setupTestDB(t)

	audit := NewIntegration()

	err := audit.LogProposalExecuted(ctx, tx, "prop_003", "actor_charlie", false, false, "connection timeout")
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))

	var action string
	var metadataJSON []byte
	err = pool.QueryRow(ctx, `
		SELECT action, metadata
		FROM audit.audit_log
		WHERE resource_id = 'prop_003'
	`).Scan(&action, &metadataJSON)
	require.NoError(t, err)

	assert.Equal(t, "proposal_execution_failed", action)

	var metadata map[string]interface{}
	require.NoError(t, json.Unmarshal(metadataJSON, &metadata))
	assert.Equal(t, false, metadata["success"])
	assert.Equal(t, false, metadata["dry_run"])
	assert.Equal(t, "connection timeout", metadata["error"])
}

func TestAuditIntegration_LogOutboxDispatched(t *testing.T) {
	ctx := context.Background()
	_, pool, tx := setupTestDB(t)

	audit := NewIntegration()

	err := audit.LogOutboxDispatched(ctx, tx, "evt_001", "feishu", true, "")
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))

	var category, action, actor, resourceType, resourceID string
	var metadataJSON []byte
	err = pool.QueryRow(ctx, `
		SELECT category, action, actor, resource_type, resource_id, metadata
		FROM audit.audit_log
		WHERE resource_id = 'evt_001'
	`).Scan(&category, &action, &actor, &resourceType, &resourceID, &metadataJSON)
	require.NoError(t, err)

	assert.Equal(t, "outbox", category)
	assert.Equal(t, "outbox_dispatched", action)
	assert.Equal(t, "system", actor)
	assert.Equal(t, "outbox_event", resourceType)
	assert.Equal(t, "evt_001", resourceID)

	var metadata map[string]interface{}
	require.NoError(t, json.Unmarshal(metadataJSON, &metadata))
	assert.Equal(t, "feishu", metadata["channel"])
	assert.Equal(t, true, metadata["success"])
	_, hasError := metadata["error"]
	assert.False(t, hasError, "error should be omitted when empty")
}

func TestAuditIntegration_LogOutboxDispatched_Failed(t *testing.T) {
	ctx := context.Background()
	_, pool, tx := setupTestDB(t)

	audit := NewIntegration()

	err := audit.LogOutboxDispatched(ctx, tx, "evt_002", "github", false, "rate limit exceeded")
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))

	var action string
	var metadataJSON []byte
	err = pool.QueryRow(ctx, `
		SELECT action, metadata
		FROM audit.audit_log
		WHERE resource_id = 'evt_002'
	`).Scan(&action, &metadataJSON)
	require.NoError(t, err)

	assert.Equal(t, "outbox_dispatch_failed", action)

	var metadata map[string]interface{}
	require.NoError(t, json.Unmarshal(metadataJSON, &metadata))
	assert.Equal(t, "github", metadata["channel"])
	assert.Equal(t, false, metadata["success"])
	assert.Equal(t, "rate limit exceeded", metadata["error"])
}

func TestAuditIntegration_EmptyActorDefaultsToSystem(t *testing.T) {
	ctx := context.Background()
	_, pool, tx := setupTestDB(t)

	audit := NewIntegration()

	err := audit.LogProposalReviewed(ctx, tx, "prop_004", "", "reject", "Insufficient evidence")
	require.NoError(t, err)

	err = audit.LogProposalExecuted(ctx, tx, "prop_005", "", false, false, "executor panic")
	require.NoError(t, err)

	require.NoError(t, tx.Commit(ctx))

	var reviewerActor, executorActor string
	err = pool.QueryRow(ctx, `
		SELECT actor FROM audit.audit_log WHERE resource_id = 'prop_004'
	`).Scan(&reviewerActor)
	require.NoError(t, err)
	assert.Equal(t, "system", reviewerActor)

	err = pool.QueryRow(ctx, `
		SELECT actor FROM audit.audit_log WHERE resource_id = 'prop_005'
	`).Scan(&executorActor)
	require.NoError(t, err)
	assert.Equal(t, "system", executorActor)
}

func TestAuditIntegration_TransactionRollback(t *testing.T) {
	ctx := context.Background()
	_, pool, tx := setupTestDB(t)

	audit := NewIntegration()

	err := audit.LogProposalReviewed(ctx, tx, "prop_006", "reviewer_dave", "approve", "OK")
	require.NoError(t, err)

	require.NoError(t, tx.Rollback(ctx))

	var count int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit.audit_log WHERE resource_id = 'prop_006'
	`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "audit record should not exist after rollback")
}

func TestAuditIntegration_DryRunExecutionHasCorrectActionName(t *testing.T) {
	ctx := context.Background()
	_, pool, tx := setupTestDB(t)

	audit := NewIntegration()

	err := audit.LogProposalExecuted(ctx, tx, "prop_007", "system_tester", true, true, "")
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))

	var action string
	err = pool.QueryRow(ctx, `
		SELECT action FROM audit.audit_log WHERE resource_id = 'prop_007'
	`).Scan(&action)
	require.NoError(t, err)

	assert.Equal(t, "proposal_executed", action)
}
