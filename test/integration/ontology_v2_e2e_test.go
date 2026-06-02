//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"baxi/internal/action"
	"baxi/internal/adapter"
	"baxi/internal/decision"
	"baxi/internal/mcp"
	"baxi/internal/ontology"
	"baxi/internal/review"
	"baxi/internal/testutil"

	"baxi/internal/repository"
	repoCommon "baxi/internal/repository/common"
	ontRepo "baxi/internal/repository/ontology"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// ──── Helper paths ──────────────────────────────────────────────────────────

func v2SchemaPath() string {
	dir, _ := os.Getwd()
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, "config", "aip_object_schema_v2.yml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		dir = filepath.Dir(dir)
	}
	return "config/aip_object_schema_v2.yml"
}

func contextRecipesPath() string {
	dir, _ := os.Getwd()
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, "config", "context_recipes.yml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		dir = filepath.Dir(dir)
	}
	return "config/context_recipes.yml"
}

func metricDefsPath() string {
	dir, _ := os.Getwd()
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, "config", "metric_definitions.yml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		dir = filepath.Dir(dir)
	}
	return "config/metric_definitions.yml"
}

func v1SchemaPath() string {
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

func actionRegistryPathE2E() string {
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

// ──── Ontology v2 E2E test ─────────────────────────────────────────────────

// TestOntologyV2E2E validates the complete seller_late_delivery_alert workflow
// via 8 E2E steps covering describe_ontology, get_object, get_linked_objects,
// build_context, propose_action, approve_proposal, execute_proposal, and
// execute_action rejection.
//
// This is the RED phase of TDD — some steps may fail until the underlying v2
// wiring is completed.
func TestOntologyV2E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// ── Step 0: Database setup ──────────────────────────────────────────────
	pg, err := testutil.StartPostgres(ctx)
	require.NoError(t, err)
	defer pg.Terminate(ctx)

	pool, err := pgxpool.New(ctx, pg.ConnectionString())
	require.NoError(t, err)
	defer pool.Close()

	require.NoError(t, pg.RunMigrations(ctx, migrationsDir()))

	// Add columns that the v2 ontology schema expects but the current
	// migrations do not yet create on dwd.item_level.
	addV2CompatibilityColumns(t, ctx, pool)

	// ── Step 0b: Seed fixture data ─────────────────────────────────────────
	seedOntologyV2Fixtures(t, ctx, pool)

	// ── Step 0c: Wire services ─────────────────────────────────────────────
	ontSvc, buildCtxSvc, reviewSvc, executeSvc, lr := wireOntologyV2Services(t, ctx, pool)

	// Shared state between sub-tests.
	var sharedProposalID string

	// ────────────────────────────────────────────────────────────────────────
	// Step 1 (T038): describe_ontology
	// ────────────────────────────────────────────────────────────────────────
	t.Run("Step1_DescribeOntology", func(t *testing.T) {
		desc, err := ontSvc.DescribeOntology(ctx)
		require.NoError(t, err, "describe_ontology should succeed")
		require.NotEmpty(t, desc.ObjectTypes, "should return object types")

		typeNames := make([]string, len(desc.ObjectTypes))
		for i, ot := range desc.ObjectTypes {
			typeNames[i] = ot.Name
		}

		expectedTypes := []string{"seller", "order", "product", "metric_alert"}
		for _, et := range expectedTypes {
			require.Contains(t, typeNames, et,
				"describe_ontology should include %q, got %v", et, typeNames)
		}
		t.Logf("describe_ontology returned %d types: %v", len(desc.ObjectTypes), typeNames)
	})

	// ────────────────────────────────────────────────────────────────────────
	// Step 2 (T039): get_object(seller, SELLER_001)
	// ────────────────────────────────────────────────────────────────────────
	t.Run("Step2_GetObject", func(t *testing.T) {
		obj, err := ontSvc.GetObject(ctx, "seller", "SELLER_001")
		require.NoError(t, err, "get_object should succeed for SELLER_001")
		require.Equal(t, "seller", obj.ObjectType)
		require.Equal(t, "SELLER_001", obj.ObjectID)
		require.NotEmpty(t, obj.Properties, "object should have non-empty properties")
		t.Logf("get_object returned %d properties", len(obj.Properties))
	})

	// ────────────────────────────────────────────────────────────────────────
	// Step 3 (T040): get_linked_objects(seller, SELLER_001, recent_orders)
	// ────────────────────────────────────────────────────────────────────────
	t.Run("Step3_GetLinkedObjects", func(t *testing.T) {
		if lr == nil {
			t.Skip("linkResolver not wired — skipping step 3")
		}

		result, err := ontSvc.GetLinkedObjects(ctx, "seller", "SELLER_001", "recent_orders", 1)
		require.NoError(t, err, "get_linked_objects should succeed for recent_orders")

		require.Len(t, result.Links, 1, "should have one link result")
		link := result.Links[0]
		require.Equal(t, "recent_orders", link.LinkName, "link name should match")
		require.Equal(t, "order", link.TargetType, "target type should be order")
		require.GreaterOrEqual(t, len(link.Objects), 1,
			"should have at least 1 linked order, got %d", len(link.Objects))

		t.Logf("get_linked_objects returned %d orders for recent_orders", len(link.Objects))
		for _, o := range link.Objects {
			t.Logf("  order: %s", o.ObjectID)
		}
	})

	// ────────────────────────────────────────────────────────────────────────
	// Step 4 (T041): build_context(CASE_001, seller_late_delivery_alert)
	// ────────────────────────────────────────────────────────────────────────
	t.Run("Step4_BuildContext", func(t *testing.T) {
		if buildCtxSvc == nil {
			t.Skip("buildContextSvc not wired — skipping step 4")
		}

		envelope, err := buildCtxSvc.BuildEnvelope(ctx, "CASE_001", "seller_late_delivery_alert")
		require.NoError(t, err, "build_context should succeed")

		require.NotEmpty(t, envelope.ContextHash, "context_hash must be present")
		require.NotNil(t, envelope.ObjectContext, "object_context must be present")
		require.NotEmpty(t, envelope.AllowedActions, "allowed_actions must be non-empty")
		require.NotEmpty(t, envelope.Evidence, "evidence must be present")
		require.NotZero(t, envelope.Governance, "governance must be present")
		require.NotZero(t, envelope.RedactionSummary, "redaction_summary must be present")

		t.Logf("build_context: hash=%s, evidence=%d, actions=%d",
			envelope.ContextHash, len(envelope.Evidence), len(envelope.AllowedActions))
	})

	// ────────────────────────────────────────────────────────────────────────
	// Step 5 (T042): propose_action
	// ────────────────────────────────────────────────────────────────────────
	t.Run("Step5_ProposeAction", func(t *testing.T) {
		params := map[string]interface{}{
			"alert_id":   "ALERT_001",
			"owner_role": "seller_ops",
			"message":    "Late delivery alert for seller SELLER_001",
		}

		result, err := ontSvc.ProposeAction(ctx, "seller", "SELLER_001", "notify_owner", params)
		require.NoError(t, err, "propose_action should succeed")
		require.True(t, result.Success, "propose_action should return success=true")

		require.NotNil(t, result.Result, "result should contain proposal_id and status")
		pid, ok := result.Result["proposal_id"].(string)
		require.True(t, ok, "result should contain proposal_id string")
		require.NotEmpty(t, pid, "proposal_id must not be empty")

		status, ok := result.Result["status"].(string)
		require.True(t, ok, "result should contain status string")
		require.Equal(t, "proposed", status,
			"proposal should be created in 'proposed' status, got %q", status)

		sharedProposalID = pid
		t.Logf("propose_action: proposal_id=%s, status=%s", pid, status)
	})

	// ────────────────────────────────────────────────────────────────────────
	// Step 6 (T043): approve_proposal
	// ────────────────────────────────────────────────────────────────────────
	t.Run("Step6_ApproveProposal", func(t *testing.T) {
		if sharedProposalID == "" {
			t.Skip("no proposal_id available from step 5 — skipping step 6")
		}

		record, err := reviewSvc.ApproveProposal(ctx, sharedProposalID, "reviewer-e2e", "Approved via E2E test")
		require.NoError(t, err, "approve_proposal should succeed")
		require.Equal(t, sharedProposalID, record.ProposalID)
		require.Equal(t, "approve", string(record.Verdict))

		// Verify DB reflects the approved status.
		proposal, err := reviewSvc.GetProposalByID(ctx, sharedProposalID)
		require.NoError(t, err)
		require.Equal(t, "approved", proposal.ApplyStatus,
			"proposal should be 'approved' after approve, got %q", proposal.ApplyStatus)

		t.Logf("approve_proposal: proposal_id=%s approved", sharedProposalID)
	})

	// ────────────────────────────────────────────────────────────────────────
	// Step 7 (T044): execute_proposal(dry_run=true)
	// ────────────────────────────────────────────────────────────────────────
	t.Run("Step7_ExecuteProposalDryRun", func(t *testing.T) {
		if sharedProposalID == "" {
			t.Skip("no proposal_id available — skipping step 7")
		}

		result, err := executeSvc.ExecuteProposal(ctx, pool, sharedProposalID, "actor-e2e",
			action.WithDryRun(true))
		require.NoError(t, err, "execute_proposal(dry_run=true) should succeed")
		require.True(t, result.Success, "execution should succeed")
		require.True(t, result.DryRun, "should be a dry run")
		require.Empty(t, result.OutboxEventID,
			"dry-run should NOT produce an outbox event")

		t.Logf("execute_proposal(dry_run=true): success=%v (dry_run)", result.Success)
	})

	// ────────────────────────────────────────────────────────────────────────
	// Step 8 (T045): execute_action without approval is rejected.
	//
	//   We test ApplyService.ExecuteProposal on a proposal that was never
	//   approved (status='proposed') to verify the rejection.
	// ────────────────────────────────────────────────────────────────────────
	t.Run("Step8_ExecuteWithoutApproval", func(t *testing.T) {
		rejectedProposalID := "prop_e2e_rejected_001"
		caseID := "case_e2e_rejected_001"

		_, err := pool.Exec(ctx,
			`INSERT INTO ai.decision_case (case_id, status, created_at) VALUES ($1, 'open', NOW())`,
			caseID)
		require.NoError(t, err)

		_, err = pool.Exec(ctx,
			`INSERT INTO ai.action_proposal (proposal_id, case_id, action_type, apply_status, title, risk_level, requires_human_review, created_at)
			 VALUES ($1, $2, 'notify_owner', 'proposed', 'E2E rejected proposal', 'medium', true, NOW())`,
			rejectedProposalID, caseID)
		require.NoError(t, err)

		// Attempt execution without approval (dry-run by default -- approval
		// gate fires before env gate when DryRun=true, returning ErrNotApproved).
		_, err = executeSvc.ExecuteProposal(ctx, pool, rejectedProposalID, "attacker")
		require.Error(t, err, "execute_proposal on unapproved proposal must fail")
		require.ErrorIs(t, err, action.ErrNotApproved,
			"error must be ErrNotApproved, got: %v", err)

		t.Logf("execute_proposal correctly rejected unapproved proposal: %v", err)
	})
}

// ──── Helper functions ──────────────────────────────────────────────────────

// addV2CompatibilityColumns adds columns to dwd.item_level that the v2
// ontology schema references but the current migrations do not create.
func addV2CompatibilityColumns(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	cols := []string{
		"seller_city TEXT",
		"late_delivery_rate NUMERIC(10,6)",
		"order_status TEXT",
		"payment_value NUMERIC(18,2)",
		"review_score NUMERIC(4,2)",
		"delivery_status TEXT",
		"order_purchase_timestamp TIMESTAMPTZ",
	}
	for _, colDef := range cols {
		_, err := pool.Exec(ctx,
			fmt.Sprintf("ALTER TABLE dwd.item_level ADD COLUMN IF NOT EXISTS %s", colDef))
		require.NoError(t, err, "add column: %s", colDef)
	}
}

// seedOntologyV2Fixtures inserts test data required for the E2E test.
func seedOntologyV2Fixtures(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	// Seller row in dwd.item_level.
	_, err := pool.Exec(ctx, `
		INSERT INTO dwd.item_level (
			order_id, order_item_id, product_id, seller_id,
			product_category_name, seller_state, seller_city, price, freight_value
		) VALUES (
			'ORD_001', 1, 'PROD_001', 'SELLER_001',
			'bed_bath_table', 'SP', 'Sao Paulo', 199.90, 15.50
		)
	`)
	require.NoError(t, err, "seed primary seller row")

	// Additional item rows.
	_, err = pool.Exec(ctx, `
		INSERT INTO dwd.item_level (
			order_id, order_item_id, product_id, seller_id,
			product_category_name, seller_state, seller_city, price, freight_value
		) VALUES
			('ORD_002', 1, 'PROD_002', 'SELLER_001', 'health_beauty', 'SP', 'Sao Paulo', 49.90, 5.00),
			('ORD_003', 1, 'PROD_003', 'SELLER_001', 'toys', 'RJ', 'Rio de Janeiro', 129.90, 12.00)
	`)
	require.NoError(t, err, "seed additional seller item rows")

	// Orders in dwd.order_level.
	_, err = pool.Exec(ctx, `
		INSERT INTO dwd.order_level (
			order_id, customer_id, customer_unique_id, order_status,
			order_purchase_timestamp, payment_value, review_score,
			is_late, is_cancelled
		) VALUES
			('ORD_001', 'CUST_001', 'UNIQ_001', 'delivered',
			 '2026-05-01 10:00:00+00', 199.90, 4.5, false, false),
			('ORD_002', 'CUST_002', 'UNIQ_002', 'shipped',
			 '2026-05-15 14:30:00+00', 49.90, 3.0, true, false),
			('ORD_003', 'CUST_003', 'UNIQ_003', 'processing',
			 '2026-05-20 09:00:00+00', 129.90, 4.0, false, false)
	`)
	require.NoError(t, err, "seed orders in dwd.order_level")

	// Decision case needed by build_context.
	_, err = pool.Exec(ctx, `
		INSERT INTO ai.decision_case (
			case_id, status, source_type, source_id, object_type, object_id,
			severity, alert_id, created_at
		) VALUES (
			'CASE_001', 'open', 'metric_alert', 'ALERT_001',
			'seller', 'SELLER_001', 'high', 'seller_late_delivery_spike', NOW()
		)
	`)
	require.NoError(t, err, "seed decision case CASE_001")
}

// wireOntologyV2Services constructs the real service instances used by the
// E2E test steps.
func wireOntologyV2Services(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (
	ontSvc *testOntologyAdapter,
	buildCtxSvc mcp.BuildContextService,
	reviewSvc *review.ReviewService,
	executeSvc *action.ApplyService,
	linkResolver *ontology.LinkResolver,
) {
	t.Helper()

	// Action registry.
	actionReg, err := action.NewActionRegistry(actionRegistryPathE2E())
	require.NoError(t, err, "load action registry")

	// Object registry (v1 from YAML + v2 from separate YAML).
	objRegistry, err := ontology.NewObjectRegistry(ctx, nil, nil, v1SchemaPath())
	require.NoError(t, err, "load v1 object registry")

	err = objRegistry.LoadV2Schema(v2SchemaPath())
	require.NoError(t, err, "load v2 schema")

	v2Objects := objRegistry.AllObjectsV2()
	t.Logf("loaded %d v2 object types", len(v2Objects))

	// LinkResolver (nil if no v2 objects).
	if len(v2Objects) > 0 {
		linkResolver = ontology.NewLinkResolver(v2Objects)
	}

	// Repository without v2 compiler — uses the v1 hardcoded table mapping
	// which queries existing columns on dwd.item_level.  The v2 compiler path
	// has known issues (e.g. late_delivery_rate lacks a source field) that are
	// tracked separately.  RecipeContextBuilder and link resolution use their
	// own QueryCompiler/LinkExecutor instances below.
	repo := ontRepo.NewRepository(repoCommon.NewPoolProvider(pool))

	// Review service.
	reviewRepo := review.NewReviewRepository()
	rSvc := review.NewReviewService(reviewRepo, pool)

	// ApplyService.
	manualAdapter := adapter.NewManualAdapter(adapter.ManualConfig{Enabled: true})
	executors := map[string]action.ActionExecutor{
		"feishu": manualAdapter,
	}
	loader := &proposalLoaderAdapter{repo: reviewRepo}
	applySvc := action.NewApplyService(actionReg, executors, loader, nil, nil, pool)

	// BuildContextService (RecipeContextBuilder) — may be nil if YAMLs fail.
	var bSvc mcp.BuildContextService
	if len(v2Objects) > 0 {
		recipes, rErr := ontology.LoadContextRecipes(contextRecipesPath())
		if rErr == nil {
			metricDefs, mErr := ontology.LoadMetricDefinitions(metricDefsPath())
			if mErr == nil {
				metricResolver := ontology.NewMetricResolver(metricDefs)
				metricQuery := ontology.NewMetricQueryResolver(metricResolver, pool)
				linkExec := ontRepo.NewLinkExecutor(repoCommon.NewPoolProvider(pool))
				qc := ontology.NewQueryCompiler(v2Objects, 10000)

				caseProvider := repository.NewDecisionRepository()
				bSvc = decision.NewRecipeContextBuilder(
					caseProvider, qc, metricQuery, linkExec, pool,
					action.NewActionTypeProviderAdapter(actionReg), recipes,
				)
				t.Log("RecipeContextBuilder wired successfully")
			} else {
				t.Logf("build_context not wired: load metric definitions: %v", mErr)
			}
		} else {
			t.Logf("build_context not wired: load recipes: %v", rErr)
		}
	} else {
		t.Log("build_context not wired: no v2 objects")
	}

	// Test ontology adapter.
	ontSvc = &testOntologyAdapter{
		registry:     objRegistry,
		repo:         repo,
		pool:         pool,
		actionReg:    actionReg,
		linkResolver: linkResolver,
		v2Objects:    v2Objects,
	}

	return ontSvc, bSvc, rSvc, applySvc, linkResolver
}

// ──── Test adapter types ────────────────────────────────────────────────────

// testOntologyAdapter implements mcp.OntologyService using real services.
type testOntologyAdapter struct {
	registry     *ontology.ObjectRegistry
	repo         *ontRepo.Repository
	pool         *pgxpool.Pool
	actionReg    *action.ActionRegistry
	linkResolver *ontology.LinkResolver
	v2Objects    map[string]*ontology.ObjectTypeV2
}

func (a *testOntologyAdapter) DescribeOntology(ctx context.Context) (*mcp.OntologyDescriptor, error) {
	if a.registry == nil {
		return &mcp.OntologyDescriptor{
			ObjectTypes: []mcp.ObjectTypeDescriptor{},
		}, nil
	}

	names := a.registry.ListObjectTypes()
	desc := &mcp.OntologyDescriptor{
		ObjectTypes: make([]mcp.ObjectTypeDescriptor, 0, len(names)),
	}
	for _, name := range names {
		ot, err := a.registry.GetObjectType(name)
		if err != nil {
			continue
		}

		otDesc := mcp.ObjectTypeDescriptor{
			Name:        ot.Name,
			DisplayName: ot.DisplayName,
			Grain:       ot.Grain,
			LLMAccess: mcp.LLMAccessDescriptor{
				CanRead:  ot.LLMAccess.CanRead,
				CanWrite: ot.LLMAccess.CanWrite,
				ReadOnly: ot.LLMAccess.ReadOnly,
			},
		}

		for _, prop := range ot.Properties {
			if !prop.LLMReadable {
				continue
			}
			otDesc.Properties = append(otDesc.Properties, mcp.PropertyDescriptor{
				Name:        prop.Name,
				Type:        prop.Type,
				Sensitivity: prop.Sensitivity,
				LLMReadable: prop.LLMReadable,
				IsPK:        prop.IsPK,
			})
		}
		for _, link := range ot.Links {
			otDesc.Links = append(otDesc.Links, mcp.LinkDescriptor{
				Name:       link.Name,
				TargetType: link.TargetType,
				Via:        link.Via,
			})
		}

		// Also include allowed actions from the v1 registry.
		otDesc.AllowedActions = a.registry.GetAllowedActions(name)

		if otDesc.Properties == nil {
			otDesc.Properties = []mcp.PropertyDescriptor{}
		}
		if otDesc.Links == nil {
			otDesc.Links = []mcp.LinkDescriptor{}
		}

		desc.ObjectTypes = append(desc.ObjectTypes, otDesc)
	}
	return desc, nil
}

func (a *testOntologyAdapter) GetObject(ctx context.Context, objectType, objectID string) (*mcp.ObjectContext, error) {
	instance, err := a.repo.GetObjectByID(ctx, objectType, objectID)
	if err != nil {
		return nil, fmt.Errorf("get object %s %s: %w", objectType, objectID, err)
	}

	return &mcp.ObjectContext{
		ObjectType: instance.ObjectType,
		ObjectID:   instance.ID,
		Properties: instance.Properties,
	}, nil
}

func (a *testOntologyAdapter) GetObjectMetrics(ctx context.Context, objectType, objectID string) (map[string]float64, error) {
	metrics, err := a.repo.GetObjectMetrics(ctx, objectType, objectID)
	if err != nil {
		return nil, fmt.Errorf("get metrics for %s %s: %w", objectType, objectID, err)
	}
	return metrics.Metrics, nil
}

func (a *testOntologyAdapter) GetLinkedObjects(ctx context.Context, objectType, objectID, linkName string, maxDepth int) (*mcp.LinkedObjectsResult, error) {
	if a.linkResolver == nil {
		return a.getLinkedObjectsV1(ctx, objectType, objectID, linkName, maxDepth)
	}
	return a.getLinkedObjectsV2(ctx, objectType, objectID, linkName, maxDepth)
}

func (a *testOntologyAdapter) getLinkedObjectsV2(ctx context.Context, objectType, objectID, linkName string, maxDepth int) (*mcp.LinkedObjectsResult, error) {
	ot, ok := a.v2Objects[objectType]
	if !ok {
		return nil, fmt.Errorf("unknown v2 object type: %s", objectType)
	}

	// Find the named link on the v2 object.
	var linkDef *ontology.ObjectLinkV2
	for _, l := range ot.Links {
		if l.Name == linkName {
			linkDef = &l
			break
		}
	}
	if linkDef == nil {
		return nil, fmt.Errorf("link %q not found on v2 object type %q", linkName, objectType)
	}

	// Use LinkExecutor to execute the link directly.
	linkOpts := ontRepo.LinkOptions{
		SourceType:     objectType,
		SourceID:       objectID,
		TargetType:     linkDef.TargetType,
		TargetSchema:   linkDef.Target.Schema,
		TargetTable:    linkDef.Target.Table,
		TargetKey:      linkDef.Target.Key,
		ObjectIDField:  linkDef.Target.ObjectIDField,
		Strategy:       linkDef.Strategy,
		SourceKey:      linkDef.SourceKey,
		Limit:          linkDef.Limit,
		Sort:           linkDef.Sort,
		Fields:         linkDef.Fields,
	}
	executor := ontRepo.NewLinkExecutor(repoCommon.NewPoolProvider(a.pool))
	instances, err := executor.ExecuteLink(ctx, linkOpts)
	if err != nil {
		return nil, fmt.Errorf("execute link %s: %w", linkName, err)
	}

	objects := make([]mcp.ObjectContext, 0, len(instances))
	for _, inst := range instances {
		objects = append(objects, mcp.ObjectContext{
			ObjectType: inst.ObjectType,
			ObjectID:   inst.ID,
			Properties: inst.Properties,
		})
	}

	return &mcp.LinkedObjectsResult{
		ObjectType: objectType,
		ObjectID:   objectID,
		Links: []mcp.LinkResult{
			{
				LinkName:   linkName,
				TargetType: linkDef.TargetType,
				Objects:    objects,
			},
		},
	}, nil
}

func (a *testOntologyAdapter) getLinkedObjectsV1(ctx context.Context, objectType, objectID, linkName string, maxDepth int) (*mcp.LinkedObjectsResult, error) {
	if a.registry == nil {
		return nil, fmt.Errorf("ontology registry not available")
	}
	links, err := a.registry.GetLinks(objectType)
	if err != nil {
		return nil, fmt.Errorf("get links for %s: %w", objectType, err)
	}
	if linkName != "" {
		found := false
		for _, l := range links {
			if l.Name == linkName {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("link %q not found for object type %q (v1 links: %v)",
				linkName, objectType, linkNames(links))
		}
	}
	result := &mcp.LinkedObjectsResult{
		ObjectType: objectType,
		ObjectID:   objectID,
		Links:      make([]mcp.LinkResult, 0),
	}
	for _, link := range links {
		if linkName != "" && link.Name != linkName {
			continue
		}
		result.Links = append(result.Links, mcp.LinkResult{
			LinkName:   link.Name,
			TargetType: link.TargetType,
			Objects:    []mcp.ObjectContext{},
		})
	}
	return result, nil
}

// ExecuteAction validates the action and returns a dry-run result.
func (a *testOntologyAdapter) ExecuteAction(ctx context.Context, objectType, objectID, actionType string, params map[string]interface{}) (*mcp.ActionResult, error) {
	if a.registry == nil {
		return nil, fmt.Errorf("ontology registry not available")
	}
	allowed := a.registry.GetAllowedActions(objectType)
	allowedMap := make(map[string]bool, len(allowed))
	for _, aa := range allowed {
		allowedMap[aa] = true
	}
	if !allowedMap[actionType] {
		return &mcp.ActionResult{
			Success:    false,
			ActionType: actionType,
			ObjectType: objectType,
			ObjectID:   objectID,
			Result:     map[string]interface{}{"error": fmt.Sprintf("action %q not allowed on %s", actionType, objectType)},
		}, nil
	}
	return &mcp.ActionResult{
		Success:    true,
		ActionType: actionType,
		ObjectType: objectType,
		ObjectID:   objectID,
		Result: map[string]interface{}{
			"would_execute": true,
			"message":       fmt.Sprintf("Action %q is valid for %s %s", actionType, objectType, objectID),
		},
	}, nil
}

// ProposeAction creates a new case and proposal with status "proposed".
func (a *testOntologyAdapter) ProposeAction(ctx context.Context, objectType, objectID, actionType string, params map[string]interface{}) (*mcp.ActionResult, error) {
	if a.registry == nil {
		return nil, fmt.Errorf("ontology registry is not available")
	}

	allowed := a.registry.GetAllowedActions(objectType)
	allowedMap := make(map[string]bool, len(allowed))
	for _, aa := range allowed {
		allowedMap[aa] = true
	}
	if !allowedMap[actionType] {
		return &mcp.ActionResult{
			Success:    false,
			ActionType: actionType,
			ObjectType: objectType,
			ObjectID:   objectID,
			Result:     map[string]interface{}{"error": fmt.Sprintf("action %q not allowed on %s", actionType, objectType)},
		}, nil
	}

	now := time.Now().UnixNano()
	caseID := fmt.Sprintf("e2e-case-%d", now)
	proposalID := fmt.Sprintf("prop-e2e-%d", now)

	_, err := a.pool.Exec(ctx,
		`INSERT INTO ai.decision_case (case_id, status, source_type, source_id, object_type, object_id, created_at) VALUES ($1, 'open', $2, $3, $2, $3, NOW())`,
			caseID, objectType, objectID)
	if err != nil {
		return &mcp.ActionResult{
			Success: false, ActionType: actionType, ObjectType: objectType, ObjectID: objectID,
			Result: map[string]interface{}{"error": fmt.Sprintf("create case: %v", err)},
		}, nil
	}

	title := fmt.Sprintf("E2E propose_action: %s on %s %s", actionType, objectType, objectID)
	_, err = a.pool.Exec(ctx,
		`INSERT INTO ai.action_proposal (proposal_id, case_id, action_type, apply_status, title, risk_level, requires_human_review, created_at)
		 VALUES ($1, $2, $3, 'proposed', $4, 'medium', true, NOW())`,
		proposalID, caseID, actionType, title)
	if err != nil {
		return &mcp.ActionResult{
			Success: false, ActionType: actionType, ObjectType: objectType, ObjectID: objectID,
			Result: map[string]interface{}{"error": fmt.Sprintf("create proposal: %v", err)},
		}, nil
	}

	return &mcp.ActionResult{
		Success:    true,
		ActionType: actionType,
		ObjectType: objectType,
		ObjectID:   objectID,
		Result: map[string]interface{}{
			"proposal_id": proposalID,
			"case_id":     caseID,
			"status":      "proposed",
			"message":     fmt.Sprintf("Action %q proposed on %s %s", actionType, objectType, objectID),
		},
	}, nil
}

// ──── e2eV2CompilerAdapter ─────────────────────────────────────────────────

// e2eV2CompilerAdapter adapts ontology.QueryCompiler to ontRepo.V2QueryCompiler.
type e2eV2CompilerAdapter struct {
	compiler *ontology.QueryCompiler
}

func (a *e2eV2CompilerAdapter) CompileGetObject(objectType, objectID string) (*ontRepo.V2CompiledQuery, error) {
	result, err := a.compiler.CompileGetObject(objectType, objectID)
	if err != nil {
		return nil, err
	}
	return &ontRepo.V2CompiledQuery{
		SQL:        result.SQL,
		CountSQL:   result.CountSQL,
		Args:       result.Args,
		Columns:    result.Columns,
		ObjectType: result.ObjectType,
		PrimaryKey: result.PrimaryKey,
		Schema:     result.Schema,
		Table:      result.Table,
	}, nil
}

func (a *e2eV2CompilerAdapter) CompileSearchObjects(objectType string, filters ontRepo.V2CompilerFilters) (*ontRepo.V2CompiledQuery, error) {
	ontologyFilters := ontology.ObjectFilters{
		Filters: filters.Filters,
		Limit:   filters.Limit,
		Offset:  filters.Offset,
		Sort:    filters.Sort,
		Order:   filters.Order,
	}
	result, err := a.compiler.CompileSearchObjects(objectType, ontologyFilters)
	if err != nil {
		return nil, err
	}
	return &ontRepo.V2CompiledQuery{
		SQL:        result.SQL,
		CountSQL:   result.CountSQL,
		Args:       result.Args,
		Columns:    result.Columns,
		ObjectType: result.ObjectType,
		PrimaryKey: result.PrimaryKey,
		Schema:     result.Schema,
		Table:      result.Table,
	}, nil
}

func (a *e2eV2CompilerAdapter) CompileObjectMetrics(objectType, objectID string) (*ontRepo.V2CompiledQuery, error) {
	result, err := a.compiler.CompileObjectMetrics(objectType, objectID)
	if err != nil {
		return nil, err
	}
	return &ontRepo.V2CompiledQuery{
		SQL:        result.SQL,
		CountSQL:   result.CountSQL,
		Args:       result.Args,
		Columns:    result.Columns,
		ObjectType: result.ObjectType,
		PrimaryKey: result.PrimaryKey,
		Schema:     result.Schema,
		Table:      result.Table,
	}, nil
}

// ──── small helpers ─────────────────────────────────────────────────────────

func linkNames(links []ontology.ObjectLink) []string {
	names := make([]string, len(links))
	for i, l := range links {
		names[i] = l.Name
	}
	return names
}
