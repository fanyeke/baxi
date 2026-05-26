package configloader

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

func syncConfigSnapshots(ctx context.Context, pool *pgxpool.Pool, registry *ConfigRegistry) error {
	for configKey, raw := range registry.RawConfigs {
		jsonBytes, err := yamlToJSON(raw.Content)
		if err != nil {
			slog.Warn("failed to convert config to json for snapshot",
				"config_key", configKey, "error", err)
			continue
		}

		query := `
			INSERT INTO gov.config_snapshot
				(config_key, config_type, source_path, content_jsonb, content_hash, loaded_at)
			VALUES ($1, $2, $3, $4::jsonb, $5, NOW())
			ON CONFLICT (config_key, content_hash) DO NOTHING
		`
		_, err = pool.Exec(ctx, query,
			configKey,
			raw.ConfigType,
			raw.SourcePath,
			string(jsonBytes),
			raw.ContentHash,
		)
		if err != nil {
			slog.Warn("failed to upsert config snapshot",
				"config_key", configKey, "error", err)
		}
	}
	return nil
}

func syncObjectSchema(ctx context.Context, pool *pgxpool.Pool, registry *ConfigRegistry) error {
	if registry.ObjectSchema == nil {
		slog.Warn("ObjectSchema not loaded, skipping gov.object_schema sync")
		return nil
	}

	query := `
		INSERT INTO gov.object_schema
			(object_type, object_name, schema_jsonb, version, created_at)
		VALUES ($1, $2, $3::jsonb, $4, NOW())
		ON CONFLICT (object_type, version) DO UPDATE SET
			object_name = EXCLUDED.object_name,
			schema_jsonb = EXCLUDED.schema_jsonb
	`

	for _, obj := range registry.ObjectSchema.Objects {
		schemaJSON, err := json.Marshal(obj)
		if err != nil {
			slog.Warn("failed to marshal object schema",
				"object_type", obj.ObjectTypeID, "error", err)
			continue
		}

		version := "1"
		_, err = pool.Exec(ctx, query,
			obj.ObjectTypeID,
			obj.DisplayName,
			string(schemaJSON),
			version,
		)
		if err != nil {
			slog.Warn("failed to upsert object schema",
				"object_type", obj.ObjectTypeID, "error", err)
		}
	}
	return nil
}

func syncDataClassification(ctx context.Context, pool *pgxpool.Pool, registry *ConfigRegistry) error {
	if registry.DataClassification == nil {
		slog.Warn("DataClassification not loaded, skipping gov.data_classification sync")
		return nil
	}

	query := `
		INSERT INTO gov.data_classification
			(field_path, classification_level, sensitivity_score, description)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (field_path, classification_level) DO UPDATE SET
			sensitivity_score = EXCLUDED.sensitivity_score,
			description = EXCLUDED.description
	`

	for _, c := range registry.DataClassification.Classifications {
		score := sensitivityScore(c.Level)

		_, err := pool.Exec(ctx, query,
			c.AssetRef,
			c.Level,
			score,
			c.Rationale,
		)
		if err != nil {
			slog.Warn("failed to upsert data classification",
				"asset_ref", c.AssetRef, "error", err)
			continue
		}

		for field, fieldLevel := range c.AppliesToFields {
			fieldPath := c.AssetRef + "." + field
			fieldScore := sensitivityScore(fieldLevel)
			_, err := pool.Exec(ctx, query,
				fieldPath,
				fieldLevel,
				fieldScore,
				c.Rationale,
			)
			if err != nil {
				slog.Warn("failed to upsert field classification",
					"field_path", fieldPath, "error", err)
			}
		}
	}
	return nil
}

func syncDataLineage(ctx context.Context, pool *pgxpool.Pool, registry *ConfigRegistry) error {
	if registry.DataLineage == nil {
		slog.Warn("DataLineage not loaded, skipping gov.data_lineage sync")
		return nil
	}

	query := `
		INSERT INTO gov.data_lineage
			(source_table, source_column, target_table, target_column,
			 transformation_logic, confidence)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (source_table, target_table, transformation_logic) DO UPDATE SET
			confidence = EXCLUDED.confidence
	`

	for _, edge := range registry.DataLineage.Edges {
		confidence := lineageConfidence(edge.TransformType)
		_, err := pool.Exec(ctx, query,
			edge.From,
			"",
			edge.To,
			"",
			edge.Transform,
			confidence,
		)
		if err != nil {
			slog.Warn("failed to upsert data lineage edge",
				"from", edge.From, "to", edge.To, "error", err)
		}
	}
	return nil
}

func syncAccessPolicy(ctx context.Context, pool *pgxpool.Pool, registry *ConfigRegistry) error {
	if registry.AccessPolicy == nil {
		slog.Warn("AccessPolicy not loaded, skipping gov.access_policy sync")
		return nil
	}

	policyQuery := `
		INSERT INTO gov.access_policy
			(policy_name, resource_type, resource_pattern, action,
			 principal_type, principal_pattern, effect, conditions_jsonb)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb)
		ON CONFLICT (policy_name) DO UPDATE SET
			resource_pattern = EXCLUDED.resource_pattern,
			action = EXCLUDED.action
	`

	emptyConditions := "{}"

	for _, role := range registry.AccessPolicy.AccessPolicy.Roles {
		for _, action := range role.AllowedActions {
			policyName := role.Role + "_" + action
			_, err := pool.Exec(ctx, policyQuery,
				policyName,
				"api_action",
				action,
				action,
				"role",
				role.Role,
				"allow",
				emptyConditions,
			)
			if err != nil {
				slog.Warn("failed to upsert access policy (action)",
					"policy_name", policyName, "error", err)
			}
		}

		for _, access := range role.DataAccess {
			policyName := role.Role + "_access_" + access
			_, err := pool.Exec(ctx, policyQuery,
				policyName,
				"data_access",
				access,
				"read",
				"role",
				role.Role,
				"allow",
				emptyConditions,
			)
			if err != nil {
				slog.Warn("failed to upsert access policy (data_access)",
					"policy_name", policyName, "error", err)
			}
		}
	}

	_, err := pool.Exec(ctx, policyQuery,
		"default_deny_all",
		"policy",
		"*",
		"*",
		"*",
		"*",
		"deny",
		emptyConditions,
	)
	if err != nil {
		slog.Warn("failed to upsert default deny-all policy", "error", err)
	}
	return nil
}

func sensitivityScore(level string) float64 {
	switch level {
	case "pii":
		return 1.0
	case "sensitive":
		return 0.8
	case "derived_sensitive":
		return 0.75
	case "internal":
		return 0.6
	case "public_internal":
		return 0.4
	default:
		return 0.5
	}
}

func lineageConfidence(transformType string) float64 {
	switch transformType {
	case "batch_load":
		return 1.0
	case "sql_aggregation":
		return 0.9
	case "heuristic_rule":
		return 0.7
	case "template_instantiation":
		return 0.8
	case "channel_routing":
		return 0.85
	case "api_sync":
		return 0.95
	default:
		return 0.5
	}
}
