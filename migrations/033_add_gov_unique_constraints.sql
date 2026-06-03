-- +goose Up
-- +goose StatementBegin

-- Migration 033: Add unique constraints to match ON CONFLICT targets in configloader

-- 1. Deduplicate gov.data_lineage on (source_table, target_table, transformation_logic)
-- Keep the row with the lowest lineage_id for each duplicate group.
DELETE FROM gov.data_lineage
WHERE lineage_id IN (
    SELECT lineage_id
    FROM (
        SELECT lineage_id,
               ROW_NUMBER() OVER (
                   PARTITION BY source_table, target_table, transformation_logic
                   ORDER BY lineage_id
               ) AS rn
        FROM gov.data_lineage
    ) sub
    WHERE rn > 1
);

-- 2. Add unique constraint for gov.data_classification ON CONFLICT target
ALTER TABLE gov.data_classification
ADD CONSTRAINT uq_data_classification_field_path_level
UNIQUE (field_path, classification_level);

-- 3. Add unique constraint for gov.data_lineage ON CONFLICT target
ALTER TABLE gov.data_lineage
ADD CONSTRAINT uq_data_lineage_source_target_transform
UNIQUE (source_table, target_table, transformation_logic);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE gov.data_lineage
DROP CONSTRAINT IF EXISTS uq_data_lineage_source_target_transform;

ALTER TABLE gov.data_classification
DROP CONSTRAINT IF EXISTS uq_data_classification_field_path_level;

-- +goose StatementEnd
