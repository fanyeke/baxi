package decision

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──── NewDecisionLineageAdapter ─────────────────────────────────────────

func TestNewDecisionLineageAdapter(t *testing.T) {
	adapter := NewDecisionLineageAdapter(nil, nil, nil)
	assert.NotNil(t, adapter)
	assert.Nil(t, adapter.lineageSvc)
	assert.Nil(t, adapter.caseRepo)
	assert.Nil(t, adapter.eventRepo)
}

// ──── GetDecisionLineage ────────────────────────────────────────────────

func TestDecisionLineageAdapter_GetDecisionLineage(t *testing.T) {
	events := []DecisionLineageEvent{
		{EventID: "evt-1", CaseID: "case-1", EventType: LineageEventContextBuilt},
		{EventID: "evt-2", CaseID: "case-1", EventType: LineageEventDecisionGenerated},
	}
	snapshots := []DecisionDataSnapshot{
		{SnapshotID: "snap-1", CaseID: "case-1", SnapshotType: SnapshotTypeObjectContext},
	}

	eventRepo := &mockLineageEventRepo{
		getLineageEventsByCaseFn: func(ctx context.Context, caseID string) ([]DecisionLineageEvent, error) {
			assert.Equal(t, "case-1", caseID)
			return events, nil
		},
		getDataSnapshotsByCaseFn: func(ctx context.Context, caseID string) ([]DecisionDataSnapshot, error) {
			return snapshots, nil
		},
	}

	adapter := NewDecisionLineageAdapter(nil, nil, eventRepo)
	chain, err := adapter.GetDecisionLineage(context.Background(), "case-1")

	require.NoError(t, err)
	assert.Equal(t, "case-1", chain.CaseID)
	assert.Len(t, chain.Events, 2)
	assert.Len(t, chain.Snapshots, 1)
	assert.Equal(t, "evt-1", chain.Events[0].EventID)
	assert.Equal(t, SnapshotTypeObjectContext, chain.Snapshots[0].SnapshotType)
}

func TestDecisionLineageAdapter_GetDecisionLineage_EmptyResults(t *testing.T) {
	eventRepo := &mockLineageEventRepo{
		getLineageEventsByCaseFn: func(ctx context.Context, caseID string) ([]DecisionLineageEvent, error) {
			return nil, nil
		},
		getDataSnapshotsByCaseFn: func(ctx context.Context, caseID string) ([]DecisionDataSnapshot, error) {
			return nil, nil
		},
	}

	adapter := NewDecisionLineageAdapter(nil, nil, eventRepo)
	chain, err := adapter.GetDecisionLineage(context.Background(), "case-2")

	require.NoError(t, err)
	assert.NotNil(t, chain.Events)
	assert.Empty(t, chain.Events)
	assert.NotNil(t, chain.Snapshots)
	assert.Empty(t, chain.Snapshots)
}

func TestDecisionLineageAdapter_GetDecisionLineage_EventError(t *testing.T) {
	eventRepo := &mockLineageEventRepo{
		getLineageEventsByCaseFn: func(ctx context.Context, caseID string) ([]DecisionLineageEvent, error) {
			return nil, assert.AnError
		},
	}

	adapter := NewDecisionLineageAdapter(nil, nil, eventRepo)
	_, err := adapter.GetDecisionLineage(context.Background(), "case-1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get lineage events")
}

func TestDecisionLineageAdapter_GetDecisionLineage_SnapshotError(t *testing.T) {
	eventRepo := &mockLineageEventRepo{
		getLineageEventsByCaseFn: func(ctx context.Context, caseID string) ([]DecisionLineageEvent, error) {
			return nil, nil
		},
		getDataSnapshotsByCaseFn: func(ctx context.Context, caseID string) ([]DecisionDataSnapshot, error) {
			return nil, assert.AnError
		},
	}

	adapter := NewDecisionLineageAdapter(nil, nil, eventRepo)
	_, err := adapter.GetDecisionLineage(context.Background(), "case-1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get data snapshots")
}

// ──── RecordDecisionLineage ─────────────────────────────────────────────

func TestDecisionLineageAdapter_RecordDecisionLineage(t *testing.T) {
	var recordedEvent *DecisionLineageEvent
	eventRepo := &mockLineageEventRepo{
		createLineageEventFn: func(ctx context.Context, event *DecisionLineageEvent) error {
			recordedEvent = event
			return nil
		},
	}

	adapter := NewDecisionLineageAdapter(nil, nil, eventRepo)
	err := adapter.RecordDecisionLineage(context.Background(), LineageEventRecord{
		CaseID:    "case-1",
		EventType: LineageEventContextBuilt,
		Actor:     "system",
	})

	require.NoError(t, err)
	require.NotNil(t, recordedEvent)
	assert.Equal(t, "case-1", recordedEvent.CaseID)
	assert.Equal(t, LineageEventContextBuilt, recordedEvent.EventType)
	assert.Equal(t, "system", recordedEvent.Actor)
	assert.NotEmpty(t, recordedEvent.EventID)
}

func TestDecisionLineageAdapter_RecordDecisionLineage_Error(t *testing.T) {
	eventRepo := &mockLineageEventRepo{
		createLineageEventFn: func(ctx context.Context, event *DecisionLineageEvent) error {
			return assert.AnError
		},
	}

	adapter := NewDecisionLineageAdapter(nil, nil, eventRepo)
	err := adapter.RecordDecisionLineage(context.Background(), LineageEventRecord{
		CaseID:    "case-1",
		EventType: LineageEventContextBuilt,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create lineage event")
}

// ──── RecordDataSnapshot ────────────────────────────────────────────────

func TestDecisionLineageAdapter_RecordDataSnapshot(t *testing.T) {
	var recordedSnapshot *DecisionDataSnapshot
	eventRepo := &mockLineageEventRepo{
		createDataSnapshotFn: func(ctx context.Context, snapshot *DecisionDataSnapshot) error {
			recordedSnapshot = snapshot
			return nil
		},
	}

	adapter := NewDecisionLineageAdapter(nil, nil, eventRepo)
	err := adapter.RecordDataSnapshot(context.Background(), DataSnapshotRecord{
		CaseID:       "case-1",
		SnapshotType: SnapshotTypeObjectContext,
		SourceTable:  "dwd.order_level",
		RowCount:     100,
	})

	require.NoError(t, err)
	require.NotNil(t, recordedSnapshot)
	assert.Equal(t, "case-1", recordedSnapshot.CaseID)
	assert.Equal(t, SnapshotTypeObjectContext, recordedSnapshot.SnapshotType)
	assert.Equal(t, "dwd.order_level", recordedSnapshot.SourceTable)
	assert.Equal(t, 100, recordedSnapshot.RowCount)
	assert.NotEmpty(t, recordedSnapshot.SnapshotID)
}

func TestDecisionLineageAdapter_RecordDataSnapshot_Error(t *testing.T) {
	eventRepo := &mockLineageEventRepo{
		createDataSnapshotFn: func(ctx context.Context, snapshot *DecisionDataSnapshot) error {
			return assert.AnError
		},
	}

	adapter := NewDecisionLineageAdapter(nil, nil, eventRepo)
	err := adapter.RecordDataSnapshot(context.Background(), DataSnapshotRecord{
		CaseID:       "case-1",
		SnapshotType: SnapshotTypeObjectContext,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create data snapshot")
}

// ──── NewPgxLineageEventRepository ──────────────────────────────────────

func TestNewPgxLineageEventRepository(t *testing.T) {
	repo := NewPgxLineageEventRepository(nil)
	assert.NotNil(t, repo)
}

// ──── LineageEventType constants ────────────────────────────────────────

func TestLineageEventTypes_Extra(t *testing.T) {
	types := []LineageEventType{
		LineageEventContextBuilt,
		LineageEventDecisionGenerated,
		LineageEventProposalCreated,
		LineageEventActionApplied,
		LineageEventDispatchSucceeded,
		LineageEventDispatchFailed,
		LineageEventRepairAttempted,
		LineageEventRepairSucceeded,
		LineageEventRepairFailed,
	}
	for _, lt := range types {
		assert.NotEmpty(t, string(lt))
	}
}

// ──── SnapshotType constants ────────────────────────────────────────────

func TestSnapshotTypes_Extra(t *testing.T) {
	types := []SnapshotType{
		SnapshotTypeLLMSafeContext,
		SnapshotTypeLLMRawOutput,
		SnapshotTypeLLMParsedOutput,
		SnapshotTypeLLMValidation,
		SnapshotTypeLLMRepairAttempt,
	}
	for _, st := range types {
		assert.NotEmpty(t, string(st))
	}
}

// ──── ID generation ─────────────────────────────────────────────────────

func TestGenerateProposalID_Extra(t *testing.T) {
	id := GenerateProposalID()
	assert.NotEmpty(t, id)
	assert.Contains(t, id, "ap_")
}

func TestGenerateDecisionID_Extra(t *testing.T) {
	id := GenerateDecisionID()
	assert.NotEmpty(t, id)
	assert.Contains(t, id, "de_")
}
