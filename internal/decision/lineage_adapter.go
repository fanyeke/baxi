package decision

import (
	"context"
	"fmt"
	"time"

	"baxi/internal/governance"
	"baxi/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LineageEventRepository defines storage operations for ai.decision_lineage_event and ai.decision_data_snapshot.
type LineageEventRepository interface {
	CreateLineageEvent(ctx context.Context, pool *pgxpool.Pool, event *DecisionLineageEvent) error
	GetLineageEventsByCase(ctx context.Context, pool *pgxpool.Pool, caseID string) ([]DecisionLineageEvent, error)
	CreateDataSnapshot(ctx context.Context, pool *pgxpool.Pool, snapshot *DecisionDataSnapshot) error
	GetDataSnapshotsByCase(ctx context.Context, pool *pgxpool.Pool, caseID string) ([]DecisionDataSnapshot, error)
}

// DecisionLineageAdapter implements DecisionLineageService by composing
// LineageService (table-level lineage), DecisionRepository (case data),
// and LineageEventRepository (decision-specific lineage events).
type DecisionLineageAdapter struct {
	lineageSvc   *governance.LineageService
	caseRepo     *repository.DecisionRepository
	eventRepo    LineageEventRepository
	pool         *pgxpool.Pool
}

// NewDecisionLineageAdapter creates a new DecisionLineageAdapter.
func NewDecisionLineageAdapter(
	lineageSvc *governance.LineageService,
	caseRepo *repository.DecisionRepository,
	eventRepo LineageEventRepository,
	pool *pgxpool.Pool,
) *DecisionLineageAdapter {
	return &DecisionLineageAdapter{
		lineageSvc: lineageSvc,
		caseRepo:   caseRepo,
		eventRepo:  eventRepo,
		pool:       pool,
	}
}

// GetDecisionLineage returns the full lineage chain for a decision case,
// including all events and data snapshots in chronological order.
func (a *DecisionLineageAdapter) GetDecisionLineage(ctx context.Context, caseID string) (*DecisionLineageChain, error) {
	events, err := a.eventRepo.GetLineageEventsByCase(ctx, a.pool, caseID)
	if err != nil {
		return nil, fmt.Errorf("get lineage events for case %s: %w", caseID, err)
	}

	snapshots, err := a.eventRepo.GetDataSnapshotsByCase(ctx, a.pool, caseID)
	if err != nil {
		return nil, fmt.Errorf("get data snapshots for case %s: %w", caseID, err)
	}

	if events == nil {
		events = []DecisionLineageEvent{}
	}
	if snapshots == nil {
		snapshots = []DecisionDataSnapshot{}
	}

	return &DecisionLineageChain{
		CaseID:    caseID,
		Events:    events,
		Snapshots: snapshots,
	}, nil
}

// GetContextLineage returns lineage information needed for context building,
// including upstream tables from the object's data lineage and relevant data snapshots.
func (a *DecisionLineageAdapter) GetContextLineage(ctx context.Context, caseID string) (*ContextLineage, error) {
	caseRow, err := a.caseRepo.GetCaseByID(ctx, a.pool, caseID)
	if err != nil {
		return nil, fmt.Errorf("get case %s: %w", caseID, err)
	}

	var upstreamTables []string
	if caseRow.ObjectType != nil && *caseRow.ObjectType != "" {
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

	snapshots, err := a.eventRepo.GetDataSnapshotsByCase(ctx, a.pool, caseID)
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

// RecordDecisionLineage records a lineage event for a decision case.
func (a *DecisionLineageAdapter) RecordDecisionLineage(ctx context.Context, record LineageEventRecord) error {
	event := &DecisionLineageEvent{
		EventID:        GenerateLineageEventID(),
		CaseID:         record.CaseID,
		EventType:      record.EventType,
		EventTimestamp: time.Now(),
		Actor:          record.Actor,
		EventData:      record.EventData,
		ContextHash:    record.ContextHash,
		ConfigHash:     record.ConfigHash,
	}

	if err := a.eventRepo.CreateLineageEvent(ctx, a.pool, event); err != nil {
		return fmt.Errorf("create lineage event: %w", err)
	}

	return nil
}

// RecordDataSnapshot records a point-in-time data snapshot for a decision case.
func (a *DecisionLineageAdapter) RecordDataSnapshot(ctx context.Context, record DataSnapshotRecord) error {
	snapshot := &DecisionDataSnapshot{
		SnapshotID:   GenerateDataSnapshotID(),
		CaseID:       record.CaseID,
		SnapshotType: record.SnapshotType,
		SnapshotJSON: record.SnapshotJSON,
		SourceTable:  record.SourceTable,
		RowCount:     record.RowCount,
		CapturedAt:   time.Now(),
	}

	if err := a.eventRepo.CreateDataSnapshot(ctx, a.pool, snapshot); err != nil {
		return fmt.Errorf("create data snapshot: %w", err)
	}

	return nil
}

// pgxLineageEventRepository implements LineageEventRepository using pgx.
type pgxLineageEventRepository struct{}

// NewPgxLineageEventRepository creates a new pgx-based LineageEventRepository.
func NewPgxLineageEventRepository() *pgxLineageEventRepository {
	return &pgxLineageEventRepository{}
}

// CreateLineageEvent inserts a lineage event into ai.decision_lineage_event.
func (r *pgxLineageEventRepository) CreateLineageEvent(ctx context.Context, pool *pgxpool.Pool, event *DecisionLineageEvent) error {
	query := `
		INSERT INTO ai.decision_lineage_event (
			event_id, case_id, event_type, event_timestamp,
			actor, event_data, context_hash, config_hash
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8
		)
	`

	_, err := pool.Exec(ctx, query,
		event.EventID,
		event.CaseID,
		string(event.EventType),
		event.EventTimestamp,
		event.Actor,
		event.EventData,
		event.ContextHash,
		event.ConfigHash,
	)
	if err != nil {
		return fmt.Errorf("insert ai.decision_lineage_event: %w", err)
	}

	return nil
}

// GetLineageEventsByCase retrieves all lineage events for a case, ordered chronologically.
func (r *pgxLineageEventRepository) GetLineageEventsByCase(ctx context.Context, pool *pgxpool.Pool, caseID string) ([]DecisionLineageEvent, error) {
	query := `
		SELECT event_id, case_id, event_type, event_timestamp,
		       actor, event_data, context_hash, config_hash
		FROM ai.decision_lineage_event
		WHERE case_id = $1
		ORDER BY event_timestamp ASC
	`

	rows, err := pool.Query(ctx, query, caseID)
	if err != nil {
		return nil, fmt.Errorf("query ai.decision_lineage_event: %w", err)
	}
	defer rows.Close()

	var events []DecisionLineageEvent
	for rows.Next() {
		var event DecisionLineageEvent
		var eventType string
		if err := rows.Scan(
			&event.EventID,
			&event.CaseID,
			&eventType,
			&event.EventTimestamp,
			&event.Actor,
			&event.EventData,
			&event.ContextHash,
			&event.ConfigHash,
		); err != nil {
			return nil, fmt.Errorf("scan lineage event row: %w", err)
		}
		event.EventType = LineageEventType(eventType)
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate lineage event rows: %w", err)
	}

	return events, nil
}

// CreateDataSnapshot inserts a data snapshot into ai.decision_data_snapshot.
func (r *pgxLineageEventRepository) CreateDataSnapshot(ctx context.Context, pool *pgxpool.Pool, snapshot *DecisionDataSnapshot) error {
	query := `
		INSERT INTO ai.decision_data_snapshot (
			snapshot_id, case_id, snapshot_type,
			snapshot_json, source_table, row_count, captured_at
		) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7
		)
	`

	_, err := pool.Exec(ctx, query,
		snapshot.SnapshotID,
		snapshot.CaseID,
		string(snapshot.SnapshotType),
		snapshot.SnapshotJSON,
		snapshot.SourceTable,
		snapshot.RowCount,
		snapshot.CapturedAt,
	)
	if err != nil {
		return fmt.Errorf("insert ai.decision_data_snapshot: %w", err)
	}

	return nil
}

// GetDataSnapshotsByCase retrieves all data snapshots for a case, ordered by capture time.
func (r *pgxLineageEventRepository) GetDataSnapshotsByCase(ctx context.Context, pool *pgxpool.Pool, caseID string) ([]DecisionDataSnapshot, error) {
	query := `
		SELECT snapshot_id, case_id, snapshot_type,
		       snapshot_json, source_table, row_count, captured_at
		FROM ai.decision_data_snapshot
		WHERE case_id = $1
		ORDER BY captured_at ASC
	`

	rows, err := pool.Query(ctx, query, caseID)
	if err != nil {
		return nil, fmt.Errorf("query ai.decision_data_snapshot: %w", err)
	}
	defer rows.Close()

	var snapshots []DecisionDataSnapshot
	for rows.Next() {
		var snapshot DecisionDataSnapshot
		var snapshotType string
		if err := rows.Scan(
			&snapshot.SnapshotID,
			&snapshot.CaseID,
			&snapshotType,
			&snapshot.SnapshotJSON,
			&snapshot.SourceTable,
			&snapshot.RowCount,
			&snapshot.CapturedAt,
		); err != nil {
			return nil, fmt.Errorf("scan data snapshot row: %w", err)
		}
		snapshot.SnapshotType = SnapshotType(snapshotType)
		snapshots = append(snapshots, snapshot)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate data snapshot rows: %w", err)
	}

	return snapshots, nil
}

// Ensure compile-time interface compliance.
var _ DecisionLineageService = (*DecisionLineageAdapter)(nil)
