// Package repository provides data access for querying database tables.
package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ConfigSnapshotRow represents a row from gov.config_snapshot.
type ConfigSnapshotRow struct {
	ConfigKey string
	Status    string // "loaded" if present in the table
}

// GovernanceRepository provides read-only access to gov.* tables.
type GovernanceRepository struct{}

// NewGovernanceRepository creates a new GovernanceRepository.
func NewGovernanceRepository() *GovernanceRepository {
	return &GovernanceRepository{}
}

// GetConfigSnapshots queries gov.config_snapshot for all loaded configuration entries.
// Returns the config key and empty string (all rows in config_snapshot are considered "loaded").
func (r *GovernanceRepository) GetConfigSnapshots(ctx context.Context, pool *pgxpool.Pool) ([]ConfigSnapshotRow, error) {
	query := `
		SELECT config_key
		FROM gov.config_snapshot
		ORDER BY config_key
	`

	rows, err := pool.Query(ctx, query)
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
func (r *GovernanceRepository) CountTableRows(ctx context.Context, pool *pgxpool.Pool, schema, table string) int {
	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s.%s`, schema, table)
	var count int
	err := pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		// Table may not exist or not be accessible; treat as 0
		return 0
	}
	return count
}

// ──── Object Schema ───────────────────────────────────────────────────────────

// GetObjectSchemas queries all rows from gov.object_schema.
func (r *GovernanceRepository) GetObjectSchemas(ctx context.Context, pool *pgxpool.Pool) ([]ObjectSchemaRow, error) {
	query := `
		SELECT object_type, object_name, schema_jsonb, version
		FROM gov.object_schema
		ORDER BY object_type
	`
	rows, err := pool.Query(ctx, query)
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
func (r *GovernanceRepository) CountObjectSchemas(ctx context.Context, pool *pgxpool.Pool) int {
	var count int
	err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM gov.object_schema`).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

// ──── Data Classification ─────────────────────────────────────────────────────

// GetDataClassifications queries all rows from gov.data_classification.
func (r *GovernanceRepository) GetDataClassifications(ctx context.Context, pool *pgxpool.Pool) ([]DataClassificationRow, error) {
	query := `
		SELECT field_path, classification_level, sensitivity_score, description
		FROM gov.data_classification
		ORDER BY field_path
	`
	rows, err := pool.Query(ctx, query)
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
func (r *GovernanceRepository) GetByFieldPath(ctx context.Context, pool *pgxpool.Pool, fieldPath string) (*DataClassificationRow, error) {
	query := `
		SELECT field_path, classification_level, sensitivity_score, description
		FROM gov.data_classification
		WHERE field_path = $1
	`
	var row DataClassificationRow
	err := pool.QueryRow(ctx, query, fieldPath).Scan(&row.FieldPath, &row.ClassificationLevel, &row.SensitivityScore, &row.Description)
	if err != nil {
		return nil, fmt.Errorf("query data_classification by field_path: %w", err)
	}
	return &row, nil
}

// ──── Data Lineage ────────────────────────────────────────────────────────────

// GetDataLineage queries all rows from gov.data_lineage.
func (r *GovernanceRepository) GetDataLineage(ctx context.Context, pool *pgxpool.Pool) ([]DataLineageRow, error) {
	query := `
		SELECT source_table, source_column, target_table, target_column,
		       transformation_logic, confidence
		FROM gov.data_lineage
		ORDER BY source_table, target_table
	`
	rows, err := pool.Query(ctx, query)
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
func (r *GovernanceRepository) GetLineageBySource(ctx context.Context, pool *pgxpool.Pool, sourceTable string) ([]DataLineageRow, error) {
	query := `
		SELECT source_table, source_column, target_table, target_column,
		       transformation_logic, confidence
		FROM gov.data_lineage
		WHERE source_table = $1
		ORDER BY target_table
	`
	rows, err := pool.Query(ctx, query, sourceTable)
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
func (r *GovernanceRepository) GetLineageByTarget(ctx context.Context, pool *pgxpool.Pool, targetTable string) ([]DataLineageRow, error) {
	query := `
		SELECT source_table, source_column, target_table, target_column,
		       transformation_logic, confidence
		FROM gov.data_lineage
		WHERE target_table = $1
		ORDER BY source_table
	`
	rows, err := pool.Query(ctx, query, targetTable)
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

// ──── Access Policy ───────────────────────────────────────────────────────────

// GetAccessPolicies queries all rows from gov.access_policy.
func (r *GovernanceRepository) GetAccessPolicies(ctx context.Context, pool *pgxpool.Pool) ([]AccessPolicyRow, error) {
	query := `
		SELECT policy_name, resource_type, resource_pattern, action,
		       principal_type, principal_pattern, effect, conditions_jsonb
		FROM gov.access_policy
		ORDER BY policy_name
	`
	rows, err := pool.Query(ctx, query)
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
func (r *GovernanceRepository) GetAccessPoliciesByRole(ctx context.Context, pool *pgxpool.Pool, role string) ([]AccessPolicyRow, error) {
	query := `
		SELECT policy_name, resource_type, resource_pattern, action,
		       principal_type, principal_pattern, effect, conditions_jsonb
		FROM gov.access_policy
		WHERE principal_pattern = $1
		ORDER BY policy_name
	`
	rows, err := pool.Query(ctx, query, role)
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
