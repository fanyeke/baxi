-- +goose Up
-- Add write-tracing fields for Pi Agent traceability:
--   evidence_refs, recipe_id on action_proposal
--   recipe_id, context_hash, severity on llm_decision

ALTER TABLE ai.action_proposal
    ADD COLUMN IF NOT EXISTS evidence_refs TEXT,
    ADD COLUMN IF NOT EXISTS recipe_id TEXT;

ALTER TABLE ai.llm_decision
    ADD COLUMN IF NOT EXISTS recipe_id TEXT,
    ADD COLUMN IF NOT EXISTS context_hash TEXT,
    ADD COLUMN IF NOT EXISTS severity TEXT;

-- +goose Down

ALTER TABLE ai.action_proposal
    DROP COLUMN IF EXISTS evidence_refs,
    DROP COLUMN IF EXISTS recipe_id;

ALTER TABLE ai.llm_decision
    DROP COLUMN IF EXISTS recipe_id,
    DROP COLUMN IF EXISTS context_hash,
    DROP COLUMN IF EXISTS severity;
