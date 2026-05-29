// Package governance provides repository access for the governance domain.
// This is a domain subpackage of the repository layer with pool injection.
package governance

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"baxi/internal/repository/common"
)

// Repository provides data access for governance configuration.
type Repository struct {
	*common.PoolProvider
}

// NewRepository creates a new Governance repository.
func NewRepository(provider *common.PoolProvider) *Repository {
	return &Repository{PoolProvider: provider}
}

// ConfigSnapshotRow represents a row from gov.config_snapshot.
type ConfigSnapshotRow struct {
	ConfigKey string
	Status    string // "loaded" if present in the table
}

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

// GetConfigSnapshots queries gov.config_snapshot for all loaded configuration entries.
// Returns the config key and empty string (all rows in config_snapshot are considered "loaded").
func (r *Repository) GetConfigSnapshots(ctx context.Context) ([]ConfigSnapshotRow, error) {
	query := `
		SELECT config_key
		FROM gov.config_snapshot
		ORDER BY config_key
	`

	rows, err := r.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query gov.config_snapshot: %w", err)
	}
	defer rows.Close()

	var results []ConfigSnapshotRow
	for rows.Next() {
		var row ConfigSnapshotRow
		if err := rows.Scan(&row.ConfigKey); err != nil {
			return nil, fmt.Errorf("scan config_snapshot row: %w", err)
		}
		row.Status = "loaded"
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate config_snapshot rows: %w", err)
	}

	if results == nil {
		results = []ConfigSnapshotRow{}
	}

	return results, nil
}

// CountTableRows returns the number of rows in the given schema.table.
// Returns 0 if the table does not exist or is empty (error is swallowed for missing tables).
func (r *Repository) CountTableRows(ctx context.Context, schema, table string) int {
	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, pgx.Identifier{schema, table}.Sanitize())
	var count int
	err := r.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		// Table may not exist or not be accessible; treat as 0
		return 0
	}
	return count
}

// GetObjectSchemas queries all rows from gov.object_schema.
func (r *Repository) GetObjectSchemas(ctx context.Context) ([]ObjectSchemaRow, error) {
	query := `
		SELECT object_type, object_name, schema_jsonb, version
		FROM gov.object_schema
		ORDER BY object_type
	`
	rows, err := r.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query gov.object_schema: %w", err)
	}
	defer rows.Close()

	var results []ObjectSchemaRow
	for rows.Next() {
		var row ObjectSchemaRow
		if err := rows.Scan(&row.ObjectType, &row.ObjectName, &row.SchemaJSONB, &row.Version); err != nil {
			return nil, fmt.Errorf("scan object_schema row: %w", err)
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate object_schema rows: %w", err)
	}
	if results == nil {
		results = []ObjectSchemaRow{}
	}
	return results, nil
}

// CountObjectSchemas returns the number of rows in gov.object_schema.
func (r *Repository) CountObjectSchemas(ctx context.Context) int {
	var count int
	err := r.QueryRow(ctx, `SELECT COUNT(*) FROM gov.object_schema`).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

// GetDataClassifications queries all rows from gov.data_classification.
func (r *Repository) GetDataClassifications(ctx context.Context) ([]DataClassificationRow, error) {
	query := `
		SELECT field_path, classification_level, sensitivity_score, description
		FROM gov.data_classification
		ORDER BY field_path
	`
	rows, err := r.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query gov.data_classification: %w", err)
	}
	defer rows.Close()

	var results []DataClassificationRow
	for rows.Next() {
		var row DataClassificationRow
		if err := rows.Scan(&row.FieldPath, &row.ClassificationLevel, &row.SensitivityScore, &row.Description); err != nil {
			return nil, fmt.Errorf("scan data_classification row: %w", err)
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate data_classification rows: %w", err)
	}
	if results == nil {
		results = []DataClassificationRow{}
	}
	return results, nil
}

// GetByFieldPath queries a single classification by field_path.
func (r *Repository) GetByFieldPath(ctx context.Context, fieldPath string) (*DataClassificationRow, error) {
	query := `
		SELECT field_path, classification_level, sensitivity_score, description
		FROM gov.data_classification
		WHERE field_path = $1
	`
	var row DataClassificationRow
	err := r.QueryRow(ctx, query, fieldPath).Scan(&row.FieldPath, &row.ClassificationLevel, &row.SensitivityScore, &row.Description)
	if err != nil {
		return nil, fmt.Errorf("query data_classification by field_path: %w", err)
	}
	return &row, nil
}

// GetDataLineage queries all rows from gov.data_lineage.
func (r *Repository) GetDataLineage(ctx context.Context) ([]DataLineageRow, error) {
	query := `
		SELECT source_table, source_column, target_table, target_column,
		       transformation_logic, confidence
		FROM gov.data_lineage
		ORDER BY source_table, target_table
	`
	rows, err := r.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query gov.data_lineage: %w", err)
	}
	defer rows.Close()

	var results []DataLineageRow
	for rows.Next() {
		var row DataLineageRow
		if err := rows.Scan(&row.SourceTable, &row.SourceColumn, &row.TargetTable, &row.TargetColumn, &row.TransformationLogic, &row.Confidence); err != nil {
			return nil, fmt.Errorf("scan data_lineage row: %w", err)
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate data_lineage rows: %w", err)
	}
	if results == nil {
		results = []DataLineageRow{}
	}
	return results, nil
}

// GetLineageBySource queries lineage rows where source_table matches.
func (r *Repository) GetLineageBySource(ctx context.Context, sourceTable string) ([]DataLineageRow, error) {
	query := `
		SELECT source_table, source_column, target_table, target_column,
		       transformation_logic, confidence
		FROM gov.data_lineage
		WHERE source_table = $1
		ORDER BY target_table
	`
	rows, err := r.Query(ctx, query, sourceTable)
	if err != nil {
		return nil, fmt.Errorf("query data_lineage by source: %w", err)
	}
	defer rows.Close()

	var results []DataLineageRow
	for rows.Next() {
		var row DataLineageRow
		if err := rows.Scan(&row.SourceTable, &row.SourceColumn, &row.TargetTable, &row.TargetColumn, &row.TransformationLogic, &row.Confidence); err != nil {
			return nil, fmt.Errorf("scan data_lineage row: %w", err)
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate data_lineage rows: %w", err)
	}
	if results == nil {
		results = []DataLineageRow{}
	}
	return results, nil
}

// GetLineageByTarget queries lineage rows where target_table matches.
func (r *Repository) GetLineageByTarget(ctx context.Context, targetTable string) ([]DataLineageRow, error) {
	query := `
		SELECT source_table, source_column, target_table, target_column,
		       transformation_logic, confidence
		FROM gov.data_lineage
		WHERE target_table = $1
		ORDER BY source_table
	`
	rows, err := r.Query(ctx, query, targetTable)
	if err != nil {
		return nil, fmt.Errorf("query data_lineage by target: %w", err)
	}
	defer rows.Close()

	var results []DataLineageRow
	for rows.Next() {
		var row DataLineageRow
		if err := rows.Scan(&row.SourceTable, &row.SourceColumn, &row.TargetTable, &row.TargetColumn, &row.TransformationLogic, &row.Confidence); err != nil {
			return nil, fmt.Errorf("scan data_lineage row: %w", err)
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate data_lineage rows: %w", err)
	}
	if results == nil {
		results = []DataLineageRow{}
	}
	return results, nil
}

// GetAccessPolicies queries all rows from gov.access_policy.
func (r *Repository) GetAccessPolicies(ctx context.Context) ([]AccessPolicyRow, error) {
	query := `
		SELECT policy_name, resource_type, resource_pattern, action,
		       principal_type, principal_pattern, effect, conditions_jsonb
		FROM gov.access_policy
		ORDER BY policy_name
	`
	rows, err := r.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query gov.access_policy: %w", err)
	}
	defer rows.Close()

	var results []AccessPolicyRow
	for rows.Next() {
		var row AccessPolicyRow
		if err := rows.Scan(&row.PolicyName, &row.ResourceType, &row.ResourcePattern, &row.Action, &row.PrincipalType, &row.PrincipalPattern, &row.Effect, &row.ConditionsJSONB); err != nil {
			return nil, fmt.Errorf("scan access_policy row: %w", err)
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate access_policy rows: %w", err)
	}
	if results == nil {
		results = []AccessPolicyRow{}
	}
	return results, nil
}

// GetAccessPoliciesByRole queries access policies for a specific role (principal_pattern).
func (r *Repository) GetAccessPoliciesByRole(ctx context.Context, role string) ([]AccessPolicyRow, error) {
	query := `
		SELECT policy_name, resource_type, resource_pattern, action,
		       principal_type, principal_pattern, effect, conditions_jsonb
		FROM gov.access_policy
		WHERE principal_pattern = $1
		ORDER BY policy_name
	`
	rows, err := r.Query(ctx, query, role)
	if err != nil {
		return nil, fmt.Errorf("query access_policy by role: %w", err)
	}
	defer rows.Close()

	var results []AccessPolicyRow
	for rows.Next() {
		var row AccessPolicyRow
		if err := rows.Scan(&row.PolicyName, &row.ResourceType, &row.ResourcePattern, &row.Action, &row.PrincipalType, &row.PrincipalPattern, &row.Effect, &row.ConditionsJSONB); err != nil {
			return nil, fmt.Errorf("scan access_policy row: %w", err)
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate access_policy rows: %w", err)
	}
	if results == nil {
		results = []AccessPolicyRow{}
	}
	return results, nil
}
