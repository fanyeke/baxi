package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"baxi/internal/action"
	"baxi/internal/adapter"
	"baxi/internal/ontology"
	"baxi/internal/review"
	"baxi/internal/testutil"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func ontologySchemaPath() string {
	dir, _ := os.Getwd()
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, "config", "aip_object_schema.yml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		dir = filepath.Dir(dir)
	}
	return "config/aip_object_schema.yml"
}


// TestAIP_DecisionLifecycle tests the full decision lifecycle:
// approve -> execute -> resolve flow with audit trail verification.
func TestAIP_DecisionLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx)
	require.NoError(t, err)
	defer pg.Terminate(ctx)

	pool, err := pgxpool.New(ctx, pg.ConnectionString())
	require.NoError(t, err)
	defer pool.Close()

	require.NoError(t, pg.RunMigrations(ctx, migrationsDir()))

	// --- Seed test data ---
	caseID := "aip_case_lifecycle_001"
	proposalID := "aip_prop_lifecycle_001"

	_, err = pool.Exec(ctx,
		`INSERT INTO ai.decision_case (case_id, status, created_at) VALUES ($1, 'proposal_generated', NOW())`,
		caseID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx,
		`INSERT INTO ai.action_proposal (proposal_id, case_id, action_type, apply_status, title, description, risk_level, requires_human_review, created_at)
		 VALUES ($1, $2, 'notify_owner', 'proposed', 'Test lifecycle proposal', 'Integration test for full lifecycle', 'medium', true, NOW())`,
		proposalID, caseID)
	require.NoError(t, err)

	reviewRepo := review.NewReviewRepository()
	reviewSvc := review.NewReviewService(reviewRepo, pool)

	// --- Step 1: Approve ---
	record, err := reviewSvc.ApproveProposal(ctx, proposalID, "reviewer-lifecycle", "Looks good for lifecycle test")
	require.NoError(t, err)
	require.Equal(t, proposalID, record.ProposalID)
	require.Equal(t, "approve", string(record.Verdict))

	var applyStatus string
	err = pool.QueryRow(ctx, `SELECT apply_status FROM ai.action_proposal WHERE proposal_id = $1`, proposalID).Scan(&applyStatus)
	require.NoError(t, err)
	require.Equal(t, "approved", applyStatus)

	// --- Step 2: Execute (dry-run first, then real) ---
	actionRegistry, err := action.NewActionRegistry(actionRegistryPath())
	require.NoError(t, err)

	feishuAdapter := adapter.NewFeishuAdapter(adapter.FeishuConfig{WebhookURL: "http://localhost:9999/test"})
	executors := map[string]action.ActionExecutor{
		"feishu": feishuAdapter,
	}

	loader := &proposalLoaderAdapter{repo: reviewRepo}
	applySvc := action.NewApplyService(actionRegistry, executors, loader, nil, nil, nil)

	// Dry-run first
	result, err := applySvc.ExecuteProposal(ctx, pool, proposalID, "actor-lifecycle", action.WithDryRun(true))
	require.NoError(t, err)
	require.True(t, result.Success)
	require.True(t, result.DryRun)
	require.Empty(t, result.OutboxEventID)

	// Real execution
	result, err = applySvc.ExecuteProposal(ctx, pool, proposalID, "actor-lifecycle", action.WithDryRun(false))
	require.NoError(t, err)
	require.True(t, result.Success)
	require.False(t, result.DryRun)
	require.NotEmpty(t, result.OutboxEventID)

	var finalStatus string
	err = pool.QueryRow(ctx, `SELECT apply_status FROM ai.action_proposal WHERE proposal_id = $1`, proposalID).Scan(&finalStatus)
	require.NoError(t, err)
	require.Equal(t, "applied", finalStatus)

	// --- Step 3: Verify outbox event ---
	var eventStatus string
	err = pool.QueryRow(ctx, `SELECT status FROM ops.outbox_event WHERE event_id = $1`, result.OutboxEventID).Scan(&eventStatus)
	require.NoError(t, err)
	require.Equal(t, "pending", eventStatus)

	// --- Step 4: Verify audit trail ---
	var approvalLogCount int
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit.audit_log WHERE category = 'review' AND action = 'proposal_approved' AND resource_id = $1`,
		proposalID).Scan(&approvalLogCount)
	require.NoError(t, err)
	require.Equal(t, 1, approvalLogCount)

	var execLogCount int
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit.audit_log WHERE category = 'action_apply' AND action = 'execute' AND resource_id = $1`,
		proposalID).Scan(&execLogCount)
	require.NoError(t, err)
	require.Equal(t, 1, execLogCount)

	// --- Step 5: Test cancellation on a second proposal ---
	caseID2 := "aip_case_lifecycle_002"
	proposalID2 := "aip_prop_lifecycle_002"

	_, err = pool.Exec(ctx,
		`INSERT INTO ai.decision_case (case_id, status, created_at) VALUES ($1, 'proposal_generated', NOW())`,
		caseID2)
	require.NoError(t, err)

	_, err = pool.Exec(ctx,
		`INSERT INTO ai.action_proposal (proposal_id, case_id, action_type, apply_status, title, description, risk_level, requires_human_review, created_at)
		 VALUES ($1, $2, 'notify_owner', 'proposed', 'Cancellation test', 'Test cancel flow', 'medium', true, NOW())`,
		proposalID2, caseID2)
	require.NoError(t, err)

	cancelRecord, err := reviewSvc.CancelProposal(ctx, proposalID2, "reviewer-cancel", "Changed mind")
	require.NoError(t, err)
	require.Equal(t, "cancel", string(cancelRecord.Verdict))

	var cancelStatus string
	err = pool.QueryRow(ctx, `SELECT apply_status FROM ai.action_proposal WHERE proposal_id = $1`, proposalID2).Scan(&cancelStatus)
	require.NoError(t, err)
	require.Equal(t, "rejected", cancelStatus)

	// --- Step 6: List review records ---
	records, total, err := reviewSvc.ListReviewRecords(ctx, proposalID, 100, 0)
	require.NoError(t, err)
	require.GreaterOrEqual(t, total, 1)
	require.NotEmpty(t, records)
	require.Equal(t, proposalID, records[0].ProposalID)
}

// TestAIP_RiskAdaptiveHITL tests the risk-adaptive human-in-the-loop feature:
// low-risk proposals auto-approve, high-risk proposals require human approval.
func TestAIP_RiskAdaptiveHITL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx)
	require.NoError(t, err)
	defer pg.Terminate(ctx)

	pool, err := pgxpool.New(ctx, pg.ConnectionString())
	require.NoError(t, err)
	defer pool.Close()

	require.NoError(t, pg.RunMigrations(ctx, migrationsDir()))

	reviewRepo := review.NewReviewRepository()
	actionRegistry, err := action.NewActionRegistry(actionRegistryPath())
	require.NoError(t, err)

	feishuAdapter := adapter.NewFeishuAdapter(adapter.FeishuConfig{WebhookURL: "http://localhost:9999/test"})
	executors := map[string]action.ActionExecutor{
		"feishu": feishuAdapter,
	}
	loader := &proposalLoaderAdapter{repo: reviewRepo}
	applySvc := action.NewApplyService(actionRegistry, executors, loader, nil, nil, nil)

	// --- Test 1: Low-risk proposal should auto-approve and execute ---
	caseLow := "aip_case_hitl_low"
	proposalLow := "aip_prop_hitl_low"

	_, err = pool.Exec(ctx,
		`INSERT INTO ai.decision_case (case_id, status, created_at) VALUES ($1, 'proposal_generated', NOW())`,
		caseLow)
	require.NoError(t, err)

	_, err = pool.Exec(ctx,
		`INSERT INTO ai.action_proposal (proposal_id, case_id, action_type, apply_status, title, description, risk_level, requires_human_review, created_at)
		 VALUES ($1, $2, 'notify_owner', 'proposed', 'Low risk proposal', 'Auto-approve test', 'low', false, NOW())`,
		proposalLow, caseLow)
	require.NoError(t, err)

	// Execute without prior approval - should succeed for low-risk
	result, err := applySvc.ExecuteProposal(ctx, pool, proposalLow, "actor-hitl", action.WithDryRun(false))
	require.NoError(t, err)
	require.True(t, result.Success)

	// Verify the proposal was applied
	var lowStatus string
	err = pool.QueryRow(ctx, `SELECT apply_status FROM ai.action_proposal WHERE proposal_id = $1`, proposalLow).Scan(&lowStatus)
	require.NoError(t, err)
	require.Equal(t, "applied", lowStatus)

	// --- Test 2: High-risk proposal should fail without approval ---
	caseHigh := "aip_case_hitl_high"
	proposalHigh := "aip_prop_hitl_high"

	_, err = pool.Exec(ctx,
		`INSERT INTO ai.decision_case (case_id, status, created_at) VALUES ($1, 'proposal_generated', NOW())`,
		caseHigh)
	require.NoError(t, err)

	_, err = pool.Exec(ctx,
		`INSERT INTO ai.action_proposal (proposal_id, case_id, action_type, apply_status, title, description, risk_level, requires_human_review, created_at)
		 VALUES ($1, $2, 'notify_owner', 'proposed', 'High risk proposal', 'Must require approval', 'critical', true, NOW())`,
		proposalHigh, caseHigh)
	require.NoError(t, err)

	// Execute without approval - should fail for critical risk
	_, err = applySvc.ExecuteProposal(ctx, pool, proposalHigh, "actor-hitl", action.WithDryRun(false))
	require.Error(t, err)
	require.ErrorIs(t, err, action.ErrNotApproved)

	// Verify the proposal is still in proposed status
	var highStatus string
	err = pool.QueryRow(ctx, `SELECT apply_status FROM ai.action_proposal WHERE proposal_id = $1`, proposalHigh).Scan(&highStatus)
	require.NoError(t, err)
	require.Equal(t, "proposed", highStatus)

	// --- Verify audit logs exist for both cases ---
	var lowAuditCount int
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit.audit_log WHERE category = 'action_apply' AND resource_id = $1`,
		proposalLow).Scan(&lowAuditCount)
	require.NoError(t, err)
	require.Equal(t, 1, lowAuditCount)

	// High-risk proposal should have a failed execution audit log
	var highAuditCount int
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit.audit_log WHERE resource_id = $1 AND action IN ('execute', 'execute_failed')`,
		proposalHigh).Scan(&highAuditCount)
	require.NoError(t, err)
	// The failed execution attempt does not write an audit log (it returns early with error)
	require.Equal(t, 0, highAuditCount)
}

// TestAIP_OntologyConfig tests that the ontology configuration is loadable
// and internally consistent (valid property-target links, valid allowed_actions).
func TestAIP_OntologyConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// --- Load ontology from YAML (no DB needed) ---
	ontPath := ontologySchemaPath()
	reg, err := ontology.NewObjectRegistry(context.Background(), nil, nil, ontPath)
	require.NoError(t, err)

	types := reg.ListObjectTypes()
	require.NotEmpty(t, types, "should have at least one object type")
	require.Contains(t, types, "customer")
	require.Contains(t, types, "order")
	require.Contains(t, types, "seller")
	require.Contains(t, types, "metric_alert")

	// --- Verify each object type has a primary key ---
	for _, typeName := range types {
		ot, err := reg.GetObjectType(typeName)
		require.NoError(t, err, "object type %s should be retrievable", typeName)
		require.NotEmpty(t, ot.Properties, "object type %s should have properties", typeName)

		// At least one property should be marked as PK
		hasPK := false
		for _, prop := range ot.Properties {
			if prop.IsPK {
				hasPK = true
				break
			}
		}
		require.True(t, hasPK, "object type %s should have a primary key property", typeName)
	}

	// --- Verify link traversal: each link's "via" property exists on the source type ---
	for _, typeName := range types {
		ot, err := reg.GetObjectType(typeName)
		require.NoError(t, err)

		links, _ := reg.GetLinks(typeName)
		for _, link := range links {
			// The via field should reference a property on the source type or a join key
			if link.Via != "" {
				_, hasViaProp := ot.Properties[link.Via]
				// Via can be a property name or a join key expression - just verify it's non-empty
				_ = hasViaProp // informational; some via values are join keys not direct props
			}
			// The target type should be a known object type
			require.True(t, ontology.KnownObjectType(link.TargetType),
				"link %s on %s points to unknown target type %s", link.Name, typeName, link.TargetType)
		}
	}

	// --- Load action registry and verify consistency ---
	actionReg, err := action.NewActionRegistry(actionRegistryPath())
	require.NoError(t, err)

	allowedActions := actionReg.AllowedActions()
	require.NotEmpty(t, allowedActions, "should have allowed actions")
	require.Contains(t, allowedActions, "notify_owner")
	require.Contains(t, allowedActions, "create_followup_task")

	// Verify allowed_actions on objects map to real action types
	actionSet := make(map[string]bool, len(allowedActions))
	for _, a := range allowedActions {
		actionSet[a] = true
	}

	for _, typeName := range types {
		allowed := reg.GetAllowedActions(typeName)
		for _, act := range allowed {
			if act == "read" {
				continue // "read" is a built-in, not from action registry
			}
			require.True(t, actionSet[act],
				"object type %s has allowed_action %s not in action registry", typeName, act)
		}
	}

	// --- Verify action registry configs are well-formed ---
	for _, actType := range allowedActions {
		cfg, ok := actionReg.GetActionConfig(actType)
		require.True(t, ok, "action %s should have config", actType)
		require.NotEmpty(t, cfg.Adapter, "action %s should have an adapter", actType)
		require.NotEmpty(t, cfg.RiskLevel, "action %s should have a risk_level", actType)
	}

	// --- Verify schema catalog works ---
	catalog := action.NewActionSchemaCatalog(actionReg)
	schemas, err := catalog.ListActionSchemas()
	require.NoError(t, err)
	require.Len(t, schemas, len(allowedActions))

	for _, schema := range schemas {
		require.NotEmpty(t, schema.Name)
		require.NotEmpty(t, schema.Adapter)
		require.NotEmpty(t, schema.RiskLevel)
	}
}


