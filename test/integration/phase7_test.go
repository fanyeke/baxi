package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"baxi/internal/action"
	"baxi/internal/adapter"
	"baxi/internal/outbox"
	"baxi/internal/review"
	"baxi/internal/testutil"
	"baxi/internal/worker"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func migrationsDir() string {
	dir, _ := os.Getwd()
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, "migrations")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		dir = filepath.Dir(dir)
	}
	return "migrations"
}

func actionRegistryPath() string {
	dir, _ := os.Getwd()
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, "config", "action_registry.yml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		dir = filepath.Dir(dir)
	}
	return "config/action_registry.yml"
}

func TestPhase7_FullApprovalFlow(t *testing.T) {
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

	caseID := "case_test001"
	proposalID := "prop_test001"

	_, err = pool.Exec(ctx,
		`INSERT INTO ai.decision_case (case_id, status, created_at) VALUES ($1, 'proposal_generated', NOW())`,
		caseID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx,
		`INSERT INTO ai.action_proposal (proposal_id, case_id, action_type, apply_status, title, created_at)
		 VALUES ($1, $2, 'notify_owner', 'proposed', 'Test notification', NOW())`,
		proposalID, caseID)
	require.NoError(t, err)

	reviewRepo := review.NewReviewRepository()
	reviewSvc := review.NewReviewService(reviewRepo, pool)

	record, err := reviewSvc.ApproveProposal(ctx, proposalID, "reviewer-1", "Looks good")
	require.NoError(t, err)
	require.Equal(t, proposalID, record.ProposalID)
	require.Equal(t, "approve", string(record.Verdict))

	var applyStatus string
	err = pool.QueryRow(ctx, `SELECT apply_status FROM ai.action_proposal WHERE proposal_id = $1`, proposalID).Scan(&applyStatus)
	require.NoError(t, err)
	require.Equal(t, "approved", applyStatus)

	var reviewCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM ai.review_record WHERE proposal_id = $1`, proposalID).Scan(&reviewCount)
	require.NoError(t, err)
	require.Equal(t, 1, reviewCount)

	actionRegistry, err := action.NewActionRegistry(actionRegistryPath())
	require.NoError(t, err)

	feishuAdapter := adapter.NewFeishuAdapter(adapter.FeishuConfig{WebhookURL: "http://localhost:9999/test"})
	githubAdapter := adapter.NewGitHubAdapter(adapter.GitHubConfig{Token: "test-token"})
	executors := map[string]action.ActionExecutor{
		"feishu": feishuAdapter,
		"github": githubAdapter,
	}

	loader := &proposalLoaderAdapter{repo: reviewRepo}
	applySvc := action.NewApplyService(actionRegistry, executors, loader)

	result, err := applySvc.ExecuteProposal(ctx, pool, proposalID, "actor-1", action.WithDryRun(true))
	require.NoError(t, err)
	require.True(t, result.Success)
	require.True(t, result.DryRun)
	require.Empty(t, result.OutboxEventID)

	result, err = applySvc.ExecuteProposal(ctx, pool, proposalID, "actor-1", action.WithDryRun(false))
	require.NoError(t, err)
	require.True(t, result.Success)
	require.False(t, result.DryRun)
	require.NotEmpty(t, result.OutboxEventID)

	var finalStatus string
	err = pool.QueryRow(ctx, `SELECT apply_status FROM ai.action_proposal WHERE proposal_id = $1`, proposalID).Scan(&finalStatus)
	require.NoError(t, err)
	require.Equal(t, "applied", finalStatus)

	var eventStatus string
	err = pool.QueryRow(ctx, `SELECT status FROM ops.outbox_event WHERE event_id = $1`, result.OutboxEventID).Scan(&eventStatus)
	require.NoError(t, err)
	require.Equal(t, "pending", eventStatus)

	var execLogCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit.audit_log WHERE category = 'action_apply' AND action = 'execute' AND resource_id = $1`, proposalID).Scan(&execLogCount)
	require.NoError(t, err)
	require.Equal(t, 1, execLogCount)

	var reviewLogCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit.audit_log WHERE category = 'review' AND action = 'proposal_approved' AND resource_id = $1`, proposalID).Scan(&reviewLogCount)
	require.NoError(t, err)
	require.Equal(t, 1, reviewLogCount)
}

func TestPhase7_RejectionFlow(t *testing.T) {
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

	caseID := "case_test002"
	proposalID := "prop_test002"

	_, err = pool.Exec(ctx,
		`INSERT INTO ai.decision_case (case_id, status, created_at) VALUES ($1, 'proposal_generated', NOW())`,
		caseID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx,
		`INSERT INTO ai.action_proposal (proposal_id, case_id, action_type, apply_status, title, created_at)
		 VALUES ($1, $2, 'notify_owner', 'proposed', 'Test notification', NOW())`,
		proposalID, caseID)
	require.NoError(t, err)

	reviewRepo := review.NewReviewRepository()
	reviewSvc := review.NewReviewService(reviewRepo, pool)

	record, err := reviewSvc.RejectProposal(ctx, proposalID, "reviewer-2", "Not appropriate")
	require.NoError(t, err)
	require.Equal(t, "reject", string(record.Verdict))

	var applyStatus string
	err = pool.QueryRow(ctx, `SELECT apply_status FROM ai.action_proposal WHERE proposal_id = $1`, proposalID).Scan(&applyStatus)
	require.NoError(t, err)
	require.Equal(t, "rejected", applyStatus)

	actionRegistry, err := action.NewActionRegistry(actionRegistryPath())
	require.NoError(t, err)

	executors := map[string]action.ActionExecutor{
		"feishu": adapter.NewFeishuAdapter(adapter.FeishuConfig{}),
	}
	loader := &proposalLoaderAdapter{repo: reviewRepo}
	applySvc := action.NewApplyService(actionRegistry, executors, loader)

	_, err = applySvc.ExecuteProposal(ctx, pool, proposalID, "actor-1", action.WithDryRun(false))
	require.Error(t, err)
	require.ErrorIs(t, err, action.ErrNotApproved)

	var rejectLogCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit.audit_log WHERE category = 'review' AND action = 'proposal_rejected' AND resource_id = $1`, proposalID).Scan(&rejectLogCount)
	require.NoError(t, err)
	require.Equal(t, 1, rejectLogCount)
}

func TestPhase7_Security_UnapprovedExecution(t *testing.T) {
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

	caseID := "case_test003"
	proposalID := "prop_test003"

	_, err = pool.Exec(ctx,
		`INSERT INTO ai.decision_case (case_id, status, created_at) VALUES ($1, 'proposal_generated', NOW())`,
		caseID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx,
		`INSERT INTO ai.action_proposal (proposal_id, case_id, action_type, apply_status, title, created_at)
		 VALUES ($1, $2, 'notify_owner', 'proposed', 'Test notification', NOW())`,
		proposalID, caseID)
	require.NoError(t, err)

	actionRegistry, err := action.NewActionRegistry(actionRegistryPath())
	require.NoError(t, err)

	executors := map[string]action.ActionExecutor{
		"feishu": adapter.NewFeishuAdapter(adapter.FeishuConfig{}),
	}
	loader := &proposalLoaderAdapter{repo: review.NewReviewRepository()}
	applySvc := action.NewApplyService(actionRegistry, executors, loader)

	_, err = applySvc.ExecuteProposal(ctx, pool, proposalID, "attacker", action.WithDryRun(false))
	require.Error(t, err)
	require.ErrorIs(t, err, action.ErrNotApproved)

	var execLogCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit.audit_log WHERE resource_id = $1 AND action IN ('execute', 'execute_failed')`, proposalID).Scan(&execLogCount)
	require.NoError(t, err)
	require.Equal(t, 0, execLogCount)

	var outboxCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM ops.outbox_event WHERE source_id = $1`, proposalID).Scan(&outboxCount)
	require.NoError(t, err)
	require.Equal(t, 0, outboxCount)
}

func TestPhase7_Whitelist_NonWhitelistedAction(t *testing.T) {
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

	caseID := "case_test004"
	proposalID := "prop_test004"

	_, err = pool.Exec(ctx,
		`INSERT INTO ai.decision_case (case_id, status, created_at) VALUES ($1, 'proposal_generated', NOW())`,
		caseID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx,
		`INSERT INTO ai.action_proposal (proposal_id, case_id, action_type, apply_status, title, created_at)
		 VALUES ($1, $2, 'notify_owner', 'approved', 'Test notification', NOW())`,
		proposalID, caseID)
	require.NoError(t, err)

	tmpConfig := `actions: {}`
	tmpPath := filepath.Join(t.TempDir(), "action_registry.yml")
	require.NoError(t, os.WriteFile(tmpPath, []byte(tmpConfig), 0644))

	actionRegistry, err := action.NewActionRegistry(tmpPath)
	require.NoError(t, err)

	executors := map[string]action.ActionExecutor{
		"feishu": adapter.NewFeishuAdapter(adapter.FeishuConfig{WebhookURL: "http://localhost:9999/test"}),
	}
	loader := &proposalLoaderAdapter{repo: review.NewReviewRepository()}
	applySvc := action.NewApplyService(actionRegistry, executors, loader)

	_, err = applySvc.ExecuteProposal(ctx, pool, proposalID, "actor-1", action.WithDryRun(false))
	require.Error(t, err)
	require.ErrorIs(t, err, action.ErrActionNotAllowed)

	var outboxCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM ops.outbox_event WHERE source_id = $1`, proposalID).Scan(&outboxCount)
	require.NoError(t, err)
	require.Equal(t, 0, outboxCount)
}

func TestPhase7_ConcurrentApprovals(t *testing.T) {
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

	const numProposals = 5
	proposalIDs := make([]string, numProposals)
	caseIDs := make([]string, numProposals)

	for i := 0; i < numProposals; i++ {
		caseIDs[i] = fmt.Sprintf("case_concurrent_%d", i)
		proposalIDs[i] = fmt.Sprintf("prop_concurrent_%d", i)

		_, err = pool.Exec(ctx,
			`INSERT INTO ai.decision_case (case_id, status, created_at) VALUES ($1, 'proposal_generated', NOW())`,
			caseIDs[i])
		require.NoError(t, err)

		_, err = pool.Exec(ctx,
			`INSERT INTO ai.action_proposal (proposal_id, case_id, action_type, apply_status, title, created_at)
			 VALUES ($1, $2, 'notify_owner', 'proposed', 'Concurrent test proposal', NOW())`,
			proposalIDs[i], caseIDs[i])
		require.NoError(t, err)
	}

	reviewRepo := review.NewReviewRepository()
	reviewSvc := review.NewReviewService(reviewRepo, pool)

	var wg sync.WaitGroup
	results := make(chan error, numProposals)

	for i := 0; i < numProposals; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := reviewSvc.ApproveProposal(ctx, proposalIDs[idx], fmt.Sprintf("reviewer-%d", idx), "Approved in concurrent test")
			results <- err
		}(i)
	}

	wg.Wait()
	close(results)

	successCount := 0
	for err := range results {
		require.NoError(t, err)
		successCount++
	}
	require.Equal(t, numProposals, successCount)

	var approvedCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM ai.action_proposal WHERE apply_status = 'approved'`).Scan(&approvedCount)
	require.NoError(t, err)
	require.Equal(t, numProposals, approvedCount)

	var reviewCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM ai.review_record WHERE verdict = 'approve'`).Scan(&reviewCount)
	require.NoError(t, err)
	require.Equal(t, numProposals, reviewCount)
}

func TestPhase7_WorkerDispatch(t *testing.T) {
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

	outboxRepo := outbox.NewOutboxRepository()
	feishuAdapter := adapter.NewFeishuAdapter(adapter.FeishuConfig{})
	executors := map[string]action.ActionExecutor{
		"feishu": feishuAdapter,
	}

	config := worker.DispatchConfig{
		PollInterval: 100 * time.Millisecond,
		BatchSize:    10,
		MaxRetries:   3,
		DryRun:       false,
	}
	w := worker.NewDispatchWorker(outboxRepo, pool, executors, config)

	numEvents := 3
	proposalID := "prop_worker_test"
	caseID := "case_worker_test"

	_, err = pool.Exec(ctx,
		`INSERT INTO ai.decision_case (case_id, status, created_at) VALUES ($1, 'proposal_generated', NOW())`,
		caseID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx,
		`INSERT INTO ai.action_proposal (proposal_id, case_id, action_type, apply_status, title, created_at)
		 VALUES ($1, $2, 'notify_owner', 'approved', 'Worker test proposal', NOW())`,
		proposalID, caseID)
	require.NoError(t, err)

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)

	for i := 0; i < numEvents; i++ {
		eventID := fmt.Sprintf("evt_worker_%d", i)
		payloadStr := fmt.Sprintf(`{"proposal_id":"%s","case_id":"%s","action_type":"notify_owner","title":"Worker test"}`, proposalID, caseID)
		_, err = tx.Exec(ctx,
			`INSERT INTO ops.outbox_event (event_id, event_type, source_type, source_id, payload_json, target_channel, status, created_at, dispatch_attempts)
			 VALUES ($1, 'notify_owner', 'action_execution', $2, $3, 'feishu', 'pending', NOW(), 0)`,
			eventID, proposalID, payloadStr)
		require.NoError(t, err)
	}
	require.NoError(t, tx.Commit(ctx))

	var pendingCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM ops.outbox_event WHERE status = 'pending'`).Scan(&pendingCount)
	require.NoError(t, err)
	require.Equal(t, numEvents, pendingCount)

	workerCtx, cancel := context.WithCancel(ctx)

	done := make(chan struct{})
	go func() {
		_ = w.Run(workerCtx)
		close(done)
	}()

	time.Sleep(500 * time.Millisecond)
	cancel()
	<-done

	var dispatchedCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM ops.outbox_event WHERE status = 'dispatched'`).Scan(&dispatchedCount)
	require.NoError(t, err)
	require.Equal(t, numEvents, dispatchedCount)
}

type proposalLoaderAdapter struct {
	repo *review.ReviewRepository
}

func (p *proposalLoaderAdapter) GetProposalByID(ctx context.Context, pool *pgxpool.Pool, proposalID string) (*action.ActionProposal, error) {
	row, err := p.repo.GetProposalByID(ctx, pool, proposalID)
	if err != nil || row == nil {
		return nil, err
	}
	ap := &action.ActionProposal{
		ProposalID:          row.ProposalID,
		CaseID:              row.CaseID,
		ActionType:          row.ActionType,
		Title:               row.Title,
		ApplyStatus:         row.ApplyStatus,
		CreatedAt:           row.CreatedAt,
		RequiresHumanReview: row.RequiresHumanReview,
	}
	if row.DecisionID != nil {
		ap.DecisionID = *row.DecisionID
	}
	if row.Description != nil {
		ap.Description = *row.Description
	}
	if row.RiskLevel != nil {
		ap.RiskLevel = *row.RiskLevel
	}
	return ap, nil
}
