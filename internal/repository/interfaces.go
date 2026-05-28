// Package repository provides data access for querying database tables.
package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ConfigSnapshotRepository provides access to gov.config_snapshot table.
type ConfigSnapshotRepository interface {
	GetConfigSnapshots(ctx context.Context, pool *pgxpool.Pool) ([]ConfigSnapshotRow, error)
	UpsertSnapshot(ctx context.Context, pool *pgxpool.Pool, params UpsertSnapshotParams) error
}

// ObjectSchemaRepository provides access to gov.object_schema table.
type ObjectSchemaRepository interface {
	GetAll(ctx context.Context, pool *pgxpool.Pool) ([]ObjectSchemaRow, error)
	GetByObjectType(ctx context.Context, pool *pgxpool.Pool, objectType string) (*ObjectSchemaRow, error)
	Upsert(ctx context.Context, pool *pgxpool.Pool, params ObjectSchemaUpsertParams) error
}

// DataClassificationRepository provides access to gov.data_classification table.
type DataClassificationRepository interface {
	GetAll(ctx context.Context, pool *pgxpool.Pool) ([]DataClassificationRow, error)
	GetByFieldPath(ctx context.Context, pool *pgxpool.Pool, fieldPath string) (*DataClassificationRow, error)
	Upsert(ctx context.Context, pool *pgxpool.Pool, params ClassificationUpsertParams) error
}

// DataLineageRepository provides access to gov.data_lineage table.
type DataLineageRepository interface {
	GetAll(ctx context.Context, pool *pgxpool.Pool) ([]DataLineageRow, error)
	GetBySource(ctx context.Context, pool *pgxpool.Pool, sourceTable string) ([]DataLineageRow, error)
	Upsert(ctx context.Context, pool *pgxpool.Pool, params LineageUpsertParams) error
}

// AccessPolicyRepository provides access to gov.access_policy table.
type AccessPolicyRepository interface {
	GetAll(ctx context.Context, pool *pgxpool.Pool) ([]AccessPolicyRow, error)
	GetByRole(ctx context.Context, pool *pgxpool.Pool, role string) ([]AccessPolicyRow, error)
	Upsert(ctx context.Context, pool *pgxpool.Pool, params AccessPolicyUpsertParams) error
}

// OntologyRepository provides object queries against dwd/mart/ops tables.
type OntologyRepository interface {
	QueryByObjectType(ctx context.Context, pool *pgxpool.Pool, objectType string, filters ObjectFilters) (*ObjectQueryResult, error)
	GetObjectByID(ctx context.Context, pool *pgxpool.Pool, objectType, objectID string) (*ObjectInstance, error)
	GetObjectMetrics(ctx context.Context, pool *pgxpool.Pool, objectType, objectID string) (*ObjectMetrics, error)
	SearchObjects(ctx context.Context, pool *pgxpool.Pool, objectType string, filters SearchFilters) (*SearchResult, error)
}

// ContextRepository encapsulates Qoder context data queries.
type ContextRepository interface {
	GetLastPipelineRun(ctx context.Context, pool *pgxpool.Pool) (*PipelineRunInfo, error)
	GetAlerts(ctx context.Context, pool *pgxpool.Pool, severity string, limit int) ([]AlertSummary, error)
	GetOpenTasks(ctx context.Context, pool *pgxpool.Pool, limit int) ([]TaskSummary, error)
	GetPendingOutbox(ctx context.Context, pool *pgxpool.Pool, limit int) ([]OutboxSummary, error)
}

// ──── Params for upsert operations ────────────────────────────────────────────

// UpsertSnapshotParams holds parameters for upserting a config snapshot.
type UpsertSnapshotParams struct {
	ConfigKey    string
	ConfigType   string
	SourcePath   string
	ContentJSONB []byte
	ContentHash  string
}

// ObjectSchemaUpsertParams holds parameters for upserting an object schema.
type ObjectSchemaUpsertParams struct {
	ObjectType  string
	ObjectName  string
	SchemaJSONB []byte
	Version     string
}

// ClassificationUpsertParams holds parameters for upserting a data classification.
type ClassificationUpsertParams struct {
	FieldPath           string
	ClassificationLevel string
	SensitivityScore    float64
	Description         string
}

// LineageUpsertParams holds parameters for upserting a data lineage record.
type LineageUpsertParams struct {
	SourceTable         string
	SourceColumn        string
	TargetTable         string
	TargetColumn        string
	TransformationLogic string
	Confidence          float64
}

// AccessPolicyUpsertParams holds parameters for upserting an access policy.
type AccessPolicyUpsertParams struct {
	PolicyName       string
	ResourceType     string
	ResourcePattern  string
	Action           string
	PrincipalType    string
	PrincipalPattern string
	Effect           string
	ConditionsJSONB  []byte
}

// ──── Row types for queries ──────────────────────────────────────────────────

// ObjectSchemaRow represents a row from gov.object_schema.
type ObjectSchemaRow struct {
	ObjectType  string
	ObjectName  string
	SchemaJSONB []byte
	Version     string
}

// DataClassificationRow represents a row from gov.data_classification.
type DataClassificationRow struct {
	FieldPath           string
	ClassificationLevel string
	SensitivityScore    float64
	Description         string
}

// DataLineageRow represents a row from gov.data_lineage.
type DataLineageRow struct {
	SourceTable         string
	SourceColumn        string
	TargetTable         string
	TargetColumn        string
	TransformationLogic string
	Confidence          float64
}

// AccessPolicyRow represents a row from gov.access_policy.
type AccessPolicyRow struct {
	PolicyName       string
	ResourceType     string
	ResourcePattern  string
	Action           string
	PrincipalType    string
	PrincipalPattern string
	Effect           string
	ConditionsJSONB  []byte
}

// ──── Ontology query types ───────────────────────────────────────────────────

// ObjectFilters holds optional filters for object queries.
type ObjectFilters struct {
	ObjectType string
	Limit      int
	Offset     int
	Filters    map[string]interface{}
}

// ObjectQueryResult holds the result of a paginated object query.
type ObjectQueryResult struct {
	Rows  []ObjectInstance
	Total int
}

// ObjectInstance represents a single object instance from a dwd/mart/ops query.
type ObjectInstance struct {
	ObjectType string
	ID         string
	Properties map[string]interface{}
}

// ObjectMetrics holds metric values for a specific object.
type ObjectMetrics struct {
	ObjectType string
	ID         string
	Metrics    map[string]float64
}

// SearchFilters holds parameters for searching objects.
type SearchFilters struct {
	ObjectType string
	Query      string
	Limit      int
	Offset     int
}

// SearchResult holds the result of a paginated search.
type SearchResult struct {
	Rows  []ObjectInstance
	Total int
}

// ──── Context query types ────────────────────────────────────────────────────

// PipelineRunInfo summarizes a pipeline execution.
type PipelineRunInfo struct {
	RunID       int64
	Status      string
	StartedAt   string
	CompletedAt string
}

// AlertSummary is a compact representation of an alert.
type AlertSummary struct {
	AlertID  string
	Severity string
	Metric   string
	Status   string
}

// TaskSummary is a compact representation of a task.
type TaskSummary struct {
	TaskID    string
	Title     string
	Status    string
	OwnerRole string
}

// OutboxSummary is a compact representation of an outbox event.
type OutboxSummary struct {
	EventID   string
	EventType string
	Status    string
}
