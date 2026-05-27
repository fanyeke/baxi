-- +goose Up
-- Add context_hash and action_schema_version to action_proposal for traceability.

ALTER TABLE ai.action_proposal
    ADD COLUMN IF NOT EXISTS context_hash TEXT,
    ADD COLUMN IF NOT EXISTS action_schema_version TEXT;

-- +goose Down
-- Remove added columns.

ALTER TABLE ai.action_proposal
    DROP COLUMN IF EXISTS context_hash,
    DROP COLUMN IF EXISTS action_schema_version;
