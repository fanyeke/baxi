-- +goose Up
-- +goose StatementBegin

-- Migration 008: Enhance ai.decision_case, ai.action_proposal, ai.llm_decision
-- with missing columns, CHECK constraints, indexes, and idempotency index.

-- ============================================================
-- 1. ADD COLUMNS to ai.decision_case
-- ============================================================
ALTER TABLE ai.decision_case
    ADD COLUMN IF NOT EXISTS source_type TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS source_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS object_type TEXT,
    ADD COLUMN IF NOT EXISTS object_id TEXT,
    ADD COLUMN IF NOT EXISTS severity TEXT,
    ADD COLUMN IF NOT EXISTS context_hash TEXT,
    ADD COLUMN IF NOT EXISTS governance_snapshot_json JSONB,
    ADD COLUMN IF NOT EXISTS created_by TEXT,
    ADD COLUMN IF NOT EXISTS error_message TEXT,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ;

-- ============================================================
-- 2. ADD COLUMNS to ai.action_proposal
-- ============================================================
ALTER TABLE ai.action_proposal
    ADD COLUMN IF NOT EXISTS title TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS description TEXT,
    ADD COLUMN IF NOT EXISTS risk_level TEXT,
    ADD COLUMN IF NOT EXISTS requires_human_review BOOLEAN DEFAULT TRUE;

-- ============================================================
-- 3. ADD COLUMNS to ai.llm_decision
-- ============================================================
ALTER TABLE ai.llm_decision
    ADD COLUMN IF NOT EXISTS status TEXT,
    ADD COLUMN IF NOT EXISTS fallback_reason TEXT,
    ADD COLUMN IF NOT EXISTS validation_errors JSONB;

-- ============================================================
-- 4. CHECK CONSTRAINTS
-- ============================================================

-- ai.decision_case: status must be a known value (include 'open' for existing rows)
ALTER TABLE ai.decision_case
    ADD CONSTRAINT chk_decision_case_status
    CHECK (status IN ('open', 'created', 'context_built', 'decision_generated', 'proposal_generated', 'review_required', 'closed', 'failed'));

-- ai.action_proposal: apply_status limited to allowed values
ALTER TABLE ai.action_proposal
    ADD CONSTRAINT chk_action_proposal_apply_status
    CHECK (apply_status IN ('proposed', 'approved', 'rejected'));

-- ai.action_proposal: action_type limited to defined types
ALTER TABLE ai.action_proposal
    ADD CONSTRAINT chk_action_proposal_action_type
    CHECK (action_type IN ('create_followup_task', 'notify_owner', 'export_report', 'escalate_to_human'));

-- ai.action_proposal: requires_human_review must always be TRUE
ALTER TABLE ai.action_proposal
    ADD CONSTRAINT chk_action_proposal_requires_review
    CHECK (requires_human_review = TRUE);

-- ============================================================
-- 5. INDEXES
-- ============================================================

-- Partial unique index for decision case idempotency:
-- Ensures at most one active case per source_type + source_id combination.
CREATE UNIQUE INDEX IF NOT EXISTS idx_ai_decision_case_active_source
    ON ai.decision_case(source_type, source_id)
    WHERE status NOT IN ('closed', 'failed');

-- Standard index for source-based lookups
CREATE INDEX IF NOT EXISTS idx_ai_decision_case_source
    ON ai.decision_case(source_type, source_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- ============================================================
-- REVERSE: Drop indexes
-- ============================================================
DROP INDEX IF EXISTS ai.idx_ai_decision_case_source;
DROP INDEX IF EXISTS ai.idx_ai_decision_case_active_source;

-- ============================================================
-- REVERSE: Drop CHECK constraints
-- ============================================================
ALTER TABLE ai.action_proposal DROP CONSTRAINT IF EXISTS chk_action_proposal_requires_review;
ALTER TABLE ai.action_proposal DROP CONSTRAINT IF EXISTS chk_action_proposal_action_type;
ALTER TABLE ai.action_proposal DROP CONSTRAINT IF EXISTS chk_action_proposal_apply_status;
ALTER TABLE ai.decision_case DROP CONSTRAINT IF EXISTS chk_decision_case_status;

-- ============================================================
-- REVERSE: Drop added columns from ai.llm_decision
-- ============================================================
ALTER TABLE ai.llm_decision DROP COLUMN IF EXISTS validation_errors;
ALTER TABLE ai.llm_decision DROP COLUMN IF EXISTS fallback_reason;
ALTER TABLE ai.llm_decision DROP COLUMN IF EXISTS status;

-- ============================================================
-- REVERSE: Drop added columns from ai.action_proposal
-- ============================================================
ALTER TABLE ai.action_proposal DROP COLUMN IF EXISTS requires_human_review;
ALTER TABLE ai.action_proposal DROP COLUMN IF EXISTS risk_level;
ALTER TABLE ai.action_proposal DROP COLUMN IF EXISTS description;
ALTER TABLE ai.action_proposal DROP COLUMN IF EXISTS title;

-- ============================================================
-- REVERSE: Drop added columns from ai.decision_case
-- ============================================================
ALTER TABLE ai.decision_case DROP COLUMN IF EXISTS updated_at;
ALTER TABLE ai.decision_case DROP COLUMN IF EXISTS error_message;
ALTER TABLE ai.decision_case DROP COLUMN IF EXISTS created_by;
ALTER TABLE ai.decision_case DROP COLUMN IF EXISTS governance_snapshot_json;
ALTER TABLE ai.decision_case DROP COLUMN IF EXISTS context_hash;
ALTER TABLE ai.decision_case DROP COLUMN IF EXISTS severity;
ALTER TABLE ai.decision_case DROP COLUMN IF EXISTS object_id;
ALTER TABLE ai.decision_case DROP COLUMN IF EXISTS object_type;
ALTER TABLE ai.decision_case DROP COLUMN IF EXISTS source_id;
ALTER TABLE ai.decision_case DROP COLUMN IF EXISTS source_type;

-- +goose StatementEnd
