package decision

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	decisionRepo "baxi/internal/repository/decision"
	"github.com/stretchr/testify/assert"
)

// --- Mocks ---

type mockLineageEventRepo struct {
	createLineageEventFn     func(ctx context.Context, event *DecisionLineageEvent) error
	getLineageEventsByCaseFn func(ctx context.Context, caseID string) ([]DecisionLineageEvent, error)
	createDataSnapshotFn     func(ctx context.Context, snapshot *DecisionDataSnapshot) error
	getDataSnapshotsByCaseFn func(ctx context.Context, caseID string) ([]DecisionDataSnapshot, error)
}

func (m *mockLineageEventRepo) CreateLineageEvent(ctx context.Context, event *DecisionLineageEvent) error {
	return m.createLineageEventFn(ctx, event)
}

func (m *mockLineageEventRepo) GetLineageEventsByCase(ctx context.Context, caseID string) ([]DecisionLineageEvent, error) {
	return m.getLineageEventsByCaseFn(ctx, caseID)
}

func (m *mockLineageEventRepo) CreateDataSnapshot(ctx context.Context, snapshot *DecisionDataSnapshot) error {
	return m.createDataSnapshotFn(ctx, snapshot)
}

func (m *mockLineageEventRepo) GetDataSnapshotsByCase(ctx context.Context, caseID string) ([]DecisionDataSnapshot, error) {
	return m.getDataSnapshotsByCaseFn(ctx, caseID)
}

// --- Compile-time interface checks ---

var _ LineageEventRepository = (*mockLineageEventRepo)(nil)

// --- Helper: createTestLineageAdapter ---

func createTestLineageAdapter(eventRepo LineageEventRepository) *DecisionLineageAdapter {
	return &DecisionLineageAdapter{
		lineageSvc: nil,
		caseRepo:   nil,
		eventRepo:  eventRepo,
	}
}

// --- Tests: GetDecisionLineage ---

func TestGetDecisionLineage_ReturnsFullChain(t *testing.T) {
	caseID := "dc_test_1"
	now := time.Now()

	events := []DecisionLineageEvent{
		{
			EventID:        "le_1",
			CaseID:         caseID,
			EventType:      LineageEventCaseCreated,
			EventTimestamp: now.Add(-10 * time.Minute),
			Actor:          "system",
		},
		{
			EventID:        "le_2",
			CaseID:         caseID,
			EventType:      LineageEventContextBuilt,
			EventTimestamp: now.Add(-5 * time.Minute),
			Actor:          "context_builder",
		},
		{
			EventID:        "le_3",
			CaseID:         caseID,
			EventType:      LineageEventDecisionGenerated,
			EventTimestamp: now,
			Actor:          "decision_engine",
		},
	}

	snapshots := []DecisionDataSnapshot{
		{
			SnapshotID:   "ds_1",
			CaseID:       caseID,
			SnapshotType: SnapshotTypeAlertContext,
			RowCount:     1,
			CapturedAt:   now.Add(-10 * time.Minute),
		},
		{
			SnapshotID:   "ds_2",
			CaseID:       caseID,
			SnapshotType: SnapshotTypeDecisionOutput,
			RowCount:     1,
			CapturedAt:   now,
		},
	}

	eventRepo := &mockLineageEventRepo{
		getLineageEventsByCaseFn: func(ctx context.Context, cid string) ([]DecisionLineageEvent, error) {
			assert.Equal(t, caseID, cid)
			return events, nil
		},
		getDataSnapshotsByCaseFn: func(ctx context.Context, cid string) ([]DecisionDataSnapshot, error) {
			assert.Equal(t, caseID, cid)
			return snapshots, nil
		},
	}

	adapter := createTestLineageAdapter(eventRepo)
	chain, err := adapter.GetDecisionLineage(context.Background(), caseID)

	assert.NoError(t, err)
	assert.NotNil(t, chain)
	assert.Equal(t, caseID, chain.CaseID)
	assert.Len(t, chain.Events, 3)
	assert.Len(t, chain.Snapshots, 2)
	assert.Equal(t, LineageEventCaseCreated, chain.Events[0].EventType)
	assert.Equal(t, LineageEventDecisionGenerated, chain.Events[2].EventType)
}

func TestGetDecisionLineage_EmptyResults(t *testing.T) {
	caseID := "dc_empty"

	eventRepo := &mockLineageEventRepo{
		getLineageEventsByCaseFn: func(ctx context.Context, cid string) ([]DecisionLineageEvent, error) {
			return nil, nil
		},
		getDataSnapshotsByCaseFn: func(ctx context.Context, cid string) ([]DecisionDataSnapshot, error) {
			return nil, nil
		},
	}

	adapter := createTestLineageAdapter(eventRepo)
	chain, err := adapter.GetDecisionLineage(context.Background(), caseID)

	assert.NoError(t, err)
	assert.NotNil(t, chain)
	assert.Equal(t, caseID, chain.CaseID)
	assert.Empty(t, chain.Events)
	assert.Empty(t, chain.Snapshots)
}

func TestGetDecisionLineage_EventsError(t *testing.T) {
	eventRepo := &mockLineageEventRepo{
		getLineageEventsByCaseFn: func(ctx context.Context, cid string) ([]DecisionLineageEvent, error) {
			return nil, errors.New("database error")
		},
		getDataSnapshotsByCaseFn: func(ctx context.Context, cid string) ([]DecisionDataSnapshot, error) {
			t.Fatal("should not be called when events fail")
			return nil, nil
		},
	}

	adapter := createTestLineageAdapter(eventRepo)
	chain, err := adapter.GetDecisionLineage(context.Background(), "dc_1")

	assert.Error(t, err)
	assert.Nil(t, chain)
	assert.Contains(t, err.Error(), "get lineage events")
}

func TestGetDecisionLineage_SnapshotsError(t *testing.T) {
	eventRepo := &mockLineageEventRepo{
		getLineageEventsByCaseFn: func(ctx context.Context, cid string) ([]DecisionLineageEvent, error) {
			return []DecisionLineageEvent{}, nil
		},
		getDataSnapshotsByCaseFn: func(ctx context.Context, cid string) ([]DecisionDataSnapshot, error) {
			return nil, errors.New("database error")
		},
	}

	adapter := createTestLineageAdapter(eventRepo)
	chain, err := adapter.GetDecisionLineage(context.Background(), "dc_1")

	assert.Error(t, err)
	assert.Nil(t, chain)
	assert.Contains(t, err.Error(), "get data snapshots")
}

// --- Tests: GetContextLineage ---

func TestGetContextLineage_WithConfigVersions(t *testing.T) {
	caseID := "dc_ctx_1"
	objectType := "seller"

	caseRepo := &mockCaseRepoForLineage{
		getCaseByIDFn: func(ctx context.Context, cid string) (*decisionRepo.DecisionCaseRow, error) {
			return &decisionRepo.DecisionCaseRow{
				CaseID:                caseID,
				ObjectType:            &objectType,
				AlertRulesVersion:     strPtr("v1.2.0"),
				AlertRulesHash:        strPtr("abc123"),
				ActionRegistryVersion: strPtr("v2.0.0"),
				ActionRegistryHash:    strPtr("def456"),
			}, nil
		},
	}

	snapshots := []DecisionDataSnapshot{
		{
			SnapshotID:   "ds_1",
			CaseID:       caseID,
			SnapshotType: SnapshotTypeObjectContext,
			RowCount:     5,
		},
	}

	eventRepo := &mockLineageEventRepo{
		getDataSnapshotsByCaseFn: func(ctx context.Context, cid string) ([]DecisionDataSnapshot, error) {
			return snapshots, nil
		},
	}

	adapter := &DecisionLineageAdapter{
		lineageSvc: nil,
		caseRepo:   nil,
		eventRepo:  eventRepo,

	}

	ctx, err := adapter.GetContextLineageWithCaseRepo(context.Background(), caseID, caseRepo)

	assert.NoError(t, err)
	assert.NotNil(t, ctx)
	assert.Equal(t, caseID, ctx.CaseID)
	assert.Equal(t, "v1.2.0", ctx.ConfigVersions["alert_rules_version"])
	assert.Equal(t, "abc123", ctx.ConfigVersions["alert_rules_hash"])
	assert.Equal(t, "v2.0.0", ctx.ConfigVersions["action_registry_version"])
	assert.Equal(t, "def456", ctx.ConfigVersions["action_registry_hash"])
	assert.Len(t, ctx.Snapshots, 1)
}

func TestGetContextLineage_EmptyConfigVersions(t *testing.T) {
	caseID := "dc_ctx_2"

	caseRepo := &mockCaseRepoForLineage{
		getCaseByIDFn: func(ctx context.Context, cid string) (*decisionRepo.DecisionCaseRow, error) {
			return &decisionRepo.DecisionCaseRow{
				CaseID: caseID,
			}, nil
		},
	}

	eventRepo := &mockLineageEventRepo{
		getDataSnapshotsByCaseFn: func(ctx context.Context, cid string) ([]DecisionDataSnapshot, error) {
			return nil, nil
		},
	}

	adapter := &DecisionLineageAdapter{
		lineageSvc: nil,
		caseRepo:   nil,
		eventRepo:  eventRepo,

	}

	ctx, err := adapter.GetContextLineageWithCaseRepo(context.Background(), caseID, caseRepo)

	assert.NoError(t, err)
	assert.NotNil(t, ctx)
	assert.Equal(t, caseID, ctx.CaseID)
	assert.Empty(t, ctx.ConfigVersions["alert_rules_version"])
	assert.Empty(t, ctx.UpstreamTables)
	assert.Empty(t, ctx.Snapshots)
}

func TestGetContextLineage_CaseNotFound(t *testing.T) {
	caseRepo := &mockCaseRepoForLineage{
		getCaseByIDFn: func(ctx context.Context, cid string) (*decisionRepo.DecisionCaseRow, error) {
			return nil, errors.New("case not found")
		},
	}

	adapter := &DecisionLineageAdapter{
		lineageSvc: nil,
		caseRepo:   nil,
		eventRepo:  nil,

	}

	ctx, err := adapter.GetContextLineageWithCaseRepo(context.Background(), "nonexistent", caseRepo)

	assert.Error(t, err)
	assert.Nil(t, ctx)
	assert.Contains(t, err.Error(), "get case")
}

func TestGetContextLineage_SnapshotsError(t *testing.T) {
	caseRepo := &mockCaseRepoForLineage{
		getCaseByIDFn: func(ctx context.Context, cid string) (*decisionRepo.DecisionCaseRow, error) {
			return &decisionRepo.DecisionCaseRow{CaseID: "dc_1"}, nil
		},
	}

	eventRepo := &mockLineageEventRepo{
		getDataSnapshotsByCaseFn: func(ctx context.Context, cid string) ([]DecisionDataSnapshot, error) {
			return nil, errors.New("snapshot query failed")
		},
	}

	adapter := &DecisionLineageAdapter{
		lineageSvc: nil,
		caseRepo:   nil,
		eventRepo:  eventRepo,

	}

	ctx, err := adapter.GetContextLineageWithCaseRepo(context.Background(), "dc_1", caseRepo)

	assert.Error(t, err)
	assert.Nil(t, ctx)
	assert.Contains(t, err.Error(), "get data snapshots")
}

// --- Tests: RecordDecisionLineage ---

func TestRecordDecisionLineage_Success(t *testing.T) {
	var capturedEvent *DecisionLineageEvent

	eventRepo := &mockLineageEventRepo{
		createLineageEventFn: func(ctx context.Context, event *DecisionLineageEvent) error {
			capturedEvent = event
			return nil
		},
	}

	adapter := createTestLineageAdapter(eventRepo)
	eventData := json.RawMessage(`{"status": "created"}`)

	err := adapter.RecordDecisionLineage(context.Background(), LineageEventRecord{
		CaseID:      "dc_1",
		EventType:   LineageEventCaseCreated,
		Actor:       "system",
		EventData:   eventData,
		ContextHash: "hash123",
		ConfigHash:  "config456",
	})

	assert.NoError(t, err)
	assert.NotNil(t, capturedEvent)
	assert.Contains(t, capturedEvent.EventID, "le_")
	assert.Equal(t, "dc_1", capturedEvent.CaseID)
	assert.Equal(t, LineageEventCaseCreated, capturedEvent.EventType)
	assert.Equal(t, "system", capturedEvent.Actor)
	assert.Equal(t, eventData, capturedEvent.EventData)
	assert.Equal(t, "hash123", capturedEvent.ContextHash)
	assert.Equal(t, "config456", capturedEvent.ConfigHash)
	assert.False(t, capturedEvent.EventTimestamp.IsZero())
}

func TestRecordDecisionLineage_DBError(t *testing.T) {
	eventRepo := &mockLineageEventRepo{
		createLineageEventFn: func(ctx context.Context, event *DecisionLineageEvent) error {
			return errors.New("insert failed")
		},
	}

	adapter := createTestLineageAdapter(eventRepo)
	err := adapter.RecordDecisionLineage(context.Background(), LineageEventRecord{
		CaseID:    "dc_1",
		EventType: LineageEventCaseCreated,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create lineage event")
}

func TestRecordDecisionLineage_AllEventTypes(t *testing.T) {
	eventTypes := []LineageEventType{
		LineageEventCaseCreated,
		LineageEventContextBuilt,
		LineageEventDecisionGenerated,
		LineageEventProposalCreated,
		LineageEventProposalApproved,
		LineageEventProposalRejected,
		LineageEventActionApplied,
		LineageEventCaseClosed,
		LineageEventCaseFailed,
		LineageEventFallbackUsed,
		LineageEventValidationFailed,
	}

	for _, et := range eventTypes {
		t.Run(string(et), func(t *testing.T) {
			var capturedEvent *DecisionLineageEvent
			eventRepo := &mockLineageEventRepo{
				createLineageEventFn: func(ctx context.Context, event *DecisionLineageEvent) error {
					capturedEvent = event
					return nil
				},
			}

			adapter := createTestLineageAdapter(eventRepo)
			err := adapter.RecordDecisionLineage(context.Background(), LineageEventRecord{
				CaseID:    "dc_1",
				EventType: et,
			})

			assert.NoError(t, err)
			assert.Equal(t, et, capturedEvent.EventType)
		})
	}
}

// --- Tests: RecordDataSnapshot ---

func TestRecordDataSnapshot_Success(t *testing.T) {
	var capturedSnapshot *DecisionDataSnapshot

	eventRepo := &mockLineageEventRepo{
		createDataSnapshotFn: func(ctx context.Context, snapshot *DecisionDataSnapshot) error {
			capturedSnapshot = snapshot
			return nil
		},
	}

	adapter := createTestLineageAdapter(eventRepo)
	snapshotJSON := json.RawMessage(`{"metric": "gmv", "value": 1000}`)

	err := adapter.RecordDataSnapshot(context.Background(), DataSnapshotRecord{
		CaseID:       "dc_1",
		SnapshotType: SnapshotTypeObjectContext,
		SnapshotJSON: snapshotJSON,
		SourceTable:  "dwd_order_level",
		RowCount:     100,
	})

	assert.NoError(t, err)
	assert.NotNil(t, capturedSnapshot)
	assert.Contains(t, capturedSnapshot.SnapshotID, "ds_")
	assert.Equal(t, "dc_1", capturedSnapshot.CaseID)
	assert.Equal(t, SnapshotTypeObjectContext, capturedSnapshot.SnapshotType)
	assert.Equal(t, snapshotJSON, capturedSnapshot.SnapshotJSON)
	assert.Equal(t, "dwd_order_level", capturedSnapshot.SourceTable)
	assert.Equal(t, 100, capturedSnapshot.RowCount)
	assert.False(t, capturedSnapshot.CapturedAt.IsZero())
}

func TestRecordDataSnapshot_DBError(t *testing.T) {
	eventRepo := &mockLineageEventRepo{
		createDataSnapshotFn: func(ctx context.Context, snapshot *DecisionDataSnapshot) error {
			return errors.New("insert failed")
		},
	}

	adapter := createTestLineageAdapter(eventRepo)
	err := adapter.RecordDataSnapshot(context.Background(), DataSnapshotRecord{
		CaseID:       "dc_1",
		SnapshotType: SnapshotTypeAlertContext,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create data snapshot")
}

func TestRecordDataSnapshot_AllSnapshotTypes(t *testing.T) {
	snapshotTypes := []SnapshotType{
		SnapshotTypeAlertContext,
		SnapshotTypeObjectContext,
		SnapshotTypeGovernance,
		SnapshotTypeDecisionInput,
		SnapshotTypeDecisionOutput,
		SnapshotTypeProposalPayload,
	}

	for _, st := range snapshotTypes {
		t.Run(string(st), func(t *testing.T) {
			var capturedSnapshot *DecisionDataSnapshot
			eventRepo := &mockLineageEventRepo{
				createDataSnapshotFn: func(ctx context.Context, snapshot *DecisionDataSnapshot) error {
					capturedSnapshot = snapshot
					return nil
				},
			}

			adapter := createTestLineageAdapter(eventRepo)
			err := adapter.RecordDataSnapshot(context.Background(), DataSnapshotRecord{
				CaseID:       "dc_1",
				SnapshotType: st,
			})

			assert.NoError(t, err)
			assert.Equal(t, st, capturedSnapshot.SnapshotType)
		})
	}
}

// --- Tests: ID Generation ---

func TestGenerateLineageEventID(t *testing.T) {
	id := GenerateLineageEventID()
	assert.Contains(t, id, "le_")
	assert.Len(t, id, 20)
}

func TestGenerateDataSnapshotID(t *testing.T) {
	id := GenerateDataSnapshotID()
	assert.Contains(t, id, "ds_")
	assert.Len(t, id, 20)
}

func TestGenerateLineageEventID_Uniqueness(t *testing.T) {
	ids := make(map[string]struct{})
	for i := 0; i < 100; i++ {
		id := GenerateLineageEventID()
		ids[id] = struct{}{}
	}
	assert.Len(t, ids, 100, "all generated lineage event IDs should be unique")
}

func TestGenerateDataSnapshotID_Uniqueness(t *testing.T) {
	ids := make(map[string]struct{})
	for i := 0; i < 100; i++ {
		id := GenerateDataSnapshotID()
		ids[id] = struct{}{}
	}
	assert.Len(t, ids, 100, "all generated data snapshot IDs should be unique")
}

type mockCaseRepoForLineage struct {
	getCaseByIDFn func(ctx context.Context, caseID string) (*decisionRepo.DecisionCaseRow, error)
}

func (m *mockCaseRepoForLineage) GetCaseByID(ctx context.Context, caseID string) (*decisionRepo.DecisionCaseRow, error) {
	return m.getCaseByIDFn(ctx, caseID)
}

func (a *DecisionLineageAdapter) GetContextLineageWithCaseRepo(ctx context.Context, caseID string, caseRepo *mockCaseRepoForLineage) (*ContextLineage, error) {
	caseRow, err := caseRepo.GetCaseByID(ctx, caseID)
	if err != nil {
		return nil, fmt.Errorf("get case %s: %w", caseID, err)
	}

	var upstreamTables []string
	if a.lineageSvc != nil && caseRow.ObjectType != nil && *caseRow.ObjectType != "" {
		lineageResult, err := a.lineageSvc.GetUpstream(ctx, *caseRow.ObjectType)
		if err == nil {
			upstreamTables = lineageResult
		}
	}
	if upstreamTables == nil {
		upstreamTables = []string{}
	}

	configVersions := make(map[string]string)
	if caseRow.AlertRulesVersion != nil {
		configVersions["alert_rules_version"] = *caseRow.AlertRulesVersion
	}
	if caseRow.AlertRulesHash != nil {
		configVersions["alert_rules_hash"] = *caseRow.AlertRulesHash
	}
	if caseRow.ActionRegistryVersion != nil {
		configVersions["action_registry_version"] = *caseRow.ActionRegistryVersion
	}
	if caseRow.ActionRegistryHash != nil {
		configVersions["action_registry_hash"] = *caseRow.ActionRegistryHash
	}

	snapshots, err := a.eventRepo.GetDataSnapshotsByCase(ctx, caseID)
	if err != nil {
		return nil, fmt.Errorf("get data snapshots for case %s: %w", caseID, err)
	}
	if snapshots == nil {
		snapshots = []DecisionDataSnapshot{}
	}

	return &ContextLineage{
		CaseID:         caseID,
		UpstreamTables: upstreamTables,
		ConfigVersions: configVersions,
		Snapshots:      snapshots,
	}, nil
}
