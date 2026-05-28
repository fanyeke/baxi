// DEPRECATED: Use baxi/internal/repository/governance instead.
// This file is a compatibility layer during migration.

package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"baxi/internal/repository/common"
	governanceRepo "baxi/internal/repository/governance"
)

// ConfigSnapshotRow represents a row from gov.config_snapshot.
// DEPRECATED: Use governance.ConfigSnapshotRow instead.
type ConfigSnapshotRow = governanceRepo.ConfigSnapshotRow

// GovernanceRepository provides data access for governance (DEPRECATED).
// Use governance.Repository instead for new code.
type GovernanceRepository struct {
	inner *governanceRepo.Repository
}

// NewGovernanceRepository creates a new GovernanceRepository (DEPRECATED).
func NewGovernanceRepository() *GovernanceRepository {
	return &GovernanceRepository{}
}

// SetPool initializes the inner repository with a pool provider.
func (r *GovernanceRepository) SetPool(pool *pgxpool.Pool) {
	r.inner = governanceRepo.NewRepository(common.NewPoolProvider(pool))
}

// ensureInitialized lazily initializes the inner repo if needed.
func (r *GovernanceRepository) ensureInitialized(pool *pgxpool.Pool) *governanceRepo.Repository {
	if r.inner == nil {
		r.SetPool(pool)
	}
	return r.inner
}

// GetConfigSnapshots queries gov.config_snapshot for all loaded configuration entries (DEPRECATED).
func (r *GovernanceRepository) GetConfigSnapshots(ctx context.Context, pool *pgxpool.Pool) ([]ConfigSnapshotRow, error) {
	return r.ensureInitialized(pool).GetConfigSnapshots(ctx)
}

// CountTableRows returns the number of rows in the given schema.table (DEPRECATED).
func (r *GovernanceRepository) CountTableRows(ctx context.Context, pool *pgxpool.Pool, schema, table string) int {
	return r.ensureInitialized(pool).CountTableRows(ctx, schema, table)
}

// GetObjectSchemas queries all rows from gov.object_schema (DEPRECATED).
func (r *GovernanceRepository) GetObjectSchemas(ctx context.Context, pool *pgxpool.Pool) ([]ObjectSchemaRow, error) {
	schemas, err := r.ensureInitialized(pool).GetObjectSchemas(ctx)
	if err != nil {
		return nil, err
	}
	// Convert governanceRepo.ObjectSchemaRow to repository.ObjectSchemaRow
	results := make([]ObjectSchemaRow, len(schemas))
	for i, s := range schemas {
		results[i] = ObjectSchemaRow{
			ObjectType:  s.ObjectType,
			ObjectName:  s.ObjectName,
			SchemaJSONB: s.SchemaJSONB,
			Version:     s.Version,
		}
	}
	return results, nil
}

// CountObjectSchemas returns the number of rows in gov.object_schema (DEPRECATED).
func (r *GovernanceRepository) CountObjectSchemas(ctx context.Context, pool *pgxpool.Pool) int {
	return r.ensureInitialized(pool).CountObjectSchemas(ctx)
}

// GetDataClassifications queries all rows from gov.data_classification (DEPRECATED).
func (r *GovernanceRepository) GetDataClassifications(ctx context.Context, pool *pgxpool.Pool) ([]DataClassificationRow, error) {
	classifications, err := r.ensureInitialized(pool).GetDataClassifications(ctx)
	if err != nil {
		return nil, err
	}
	// Convert governanceRepo.DataClassificationRow to repository.DataClassificationRow
	results := make([]DataClassificationRow, len(classifications))
	for i, c := range classifications {
		results[i] = DataClassificationRow{
			FieldPath:           c.FieldPath,
			ClassificationLevel: c.ClassificationLevel,
			SensitivityScore:    c.SensitivityScore,
			Description:         c.Description,
		}
	}
	return results, nil
}

// GetByFieldPath queries a single classification by field_path (DEPRECATED).
func (r *GovernanceRepository) GetByFieldPath(ctx context.Context, pool *pgxpool.Pool, fieldPath string) (*DataClassificationRow, error) {
	row, err := r.ensureInitialized(pool).GetByFieldPath(ctx, fieldPath)
	if err != nil {
		return nil, err
	}
	return &DataClassificationRow{
		FieldPath:           row.FieldPath,
		ClassificationLevel: row.ClassificationLevel,
		SensitivityScore:    row.SensitivityScore,
		Description:         row.Description,
	}, nil
}

// GetDataLineage queries all rows from gov.data_lineage (DEPRECATED).
func (r *GovernanceRepository) GetDataLineage(ctx context.Context, pool *pgxpool.Pool) ([]DataLineageRow, error) {
	lineage, err := r.ensureInitialized(pool).GetDataLineage(ctx)
	if err != nil {
		return nil, err
	}
	// Convert governanceRepo.DataLineageRow to repository.DataLineageRow
	results := make([]DataLineageRow, len(lineage))
	for i, l := range lineage {
		results[i] = DataLineageRow{
			SourceTable:         l.SourceTable,
			SourceColumn:        l.SourceColumn,
			TargetTable:         l.TargetTable,
			TargetColumn:        l.TargetColumn,
			TransformationLogic: l.TransformationLogic,
			Confidence:          l.Confidence,
		}
	}
	return results, nil
}

// GetLineageBySource queries lineage rows where source_table matches (DEPRECATED).
func (r *GovernanceRepository) GetLineageBySource(ctx context.Context, pool *pgxpool.Pool, sourceTable string) ([]DataLineageRow, error) {
	lineage, err := r.ensureInitialized(pool).GetLineageBySource(ctx, sourceTable)
	if err != nil {
		return nil, err
	}
	// Convert governanceRepo.DataLineageRow to repository.DataLineageRow
	results := make([]DataLineageRow, len(lineage))
	for i, l := range lineage {
		results[i] = DataLineageRow{
			SourceTable:         l.SourceTable,
			SourceColumn:        l.SourceColumn,
			TargetTable:         l.TargetTable,
			TargetColumn:        l.TargetColumn,
			TransformationLogic: l.TransformationLogic,
			Confidence:          l.Confidence,
		}
	}
	return results, nil
}

// GetLineageByTarget queries lineage rows where target_table matches (DEPRECATED).
func (r *GovernanceRepository) GetLineageByTarget(ctx context.Context, pool *pgxpool.Pool, targetTable string) ([]DataLineageRow, error) {
	lineage, err := r.ensureInitialized(pool).GetLineageByTarget(ctx, targetTable)
	if err != nil {
		return nil, err
	}
	// Convert governanceRepo.DataLineageRow to repository.DataLineageRow
	results := make([]DataLineageRow, len(lineage))
	for i, l := range lineage {
		results[i] = DataLineageRow{
			SourceTable:         l.SourceTable,
			SourceColumn:        l.SourceColumn,
			TargetTable:         l.TargetTable,
			TargetColumn:        l.TargetColumn,
			TransformationLogic: l.TransformationLogic,
			Confidence:          l.Confidence,
		}
	}
	return results, nil
}

// GetAccessPolicies queries all rows from gov.access_policy (DEPRECATED).
func (r *GovernanceRepository) GetAccessPolicies(ctx context.Context, pool *pgxpool.Pool) ([]AccessPolicyRow, error) {
	policies, err := r.ensureInitialized(pool).GetAccessPolicies(ctx)
	if err != nil {
		return nil, err
	}
	// Convert governanceRepo.AccessPolicyRow to repository.AccessPolicyRow
	results := make([]AccessPolicyRow, len(policies))
	for i, p := range policies {
		results[i] = AccessPolicyRow{
			PolicyName:       p.PolicyName,
			ResourceType:     p.ResourceType,
			ResourcePattern:  p.ResourcePattern,
			Action:           p.Action,
			PrincipalType:    p.PrincipalType,
			PrincipalPattern: p.PrincipalPattern,
			Effect:           p.Effect,
			ConditionsJSONB:  p.ConditionsJSONB,
		}
	}
	return results, nil
}

// GetAccessPoliciesByRole queries access policies for a specific role (DEPRECATED).
func (r *GovernanceRepository) GetAccessPoliciesByRole(ctx context.Context, pool *pgxpool.Pool, role string) ([]AccessPolicyRow, error) {
	policies, err := r.ensureInitialized(pool).GetAccessPoliciesByRole(ctx, role)
	if err != nil {
		return nil, err
	}
	// Convert governanceRepo.AccessPolicyRow to repository.AccessPolicyRow
	results := make([]AccessPolicyRow, len(policies))
	for i, p := range policies {
		results[i] = AccessPolicyRow{
			PolicyName:       p.PolicyName,
			ResourceType:     p.ResourceType,
			ResourcePattern:  p.ResourcePattern,
			Action:           p.Action,
			PrincipalType:    p.PrincipalType,
			PrincipalPattern: p.PrincipalPattern,
			Effect:           p.Effect,
			ConditionsJSONB:  p.ConditionsJSONB,
		}
	}
	return results, nil
}
