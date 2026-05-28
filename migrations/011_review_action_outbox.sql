-- +goose Up
-- +goose StatementBegin

-- Migration 011: Extend schema for Phase 7 review/action/outbox state machine.

-- ============================================================
-- 1. Extend ai.action_proposal.apply_status CHECK
--    Add: 'applying', 'applied', 'failed'
-- ============================================================
ALTER TABLE ai.action_proposal
    DROP CONSTRAINT IF EXISTS chk_action_proposal_apply_status;

ALTER TABLE ai.action_proposal
    ADD CONSTRAINT chk_action_proposal_apply_status
    CHECK (apply_status IN ('proposed', 'approved', 'rejected', 'applying', 'applied', 'failed'));

-- ============================================================
-- 2. Reconcile ai.action_proposal.action_type CHECK
--    Replace 'escalate_to_human' with 'create_outbox_message'
-- ============================================================
ALTER TABLE ai.action_proposal
    DROP CONSTRAINT IF EXISTS chk_action_proposal_action_type;

ALTER TABLE ai.action_proposal
    ADD CONSTRAINT chk_action_proposal_action_type
    CHECK (action_type IN ('create_followup_task', 'notify_owner', 'export_report', 'create_outbox_message'));

-- ============================================================
-- 3. Add CHECK constraint to ai.review_record.verdict
-- ============================================================
ALTER TABLE ai.review_record
    ADD CONSTRAINT chk_review_record_verdict
    CHECK (verdict IN ('approve', 'reject', 'cancel'));

-- ============================================================
-- 4. Add index on ai.review_record(proposal_id)
-- ============================================================
CREATE INDEX IF NOT EXISTS idx_review_record_proposal_id
    ON ai.review_record(proposal_id);

-- ============================================================
-- 5. Add partial index on ops.outbox_event for worker polling
-- ============================================================
CREATE INDEX IF NOT EXISTS idx_outbox_event_status_created
    ON ops.outbox_event(status, created_at)
    WHERE status IN ('pending', 'failed');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- ============================================================
-- 1. Drop partial index on ops.outbox_event
-- ============================================================
DROP INDEX IF EXISTS ops.idx_outbox_event_status_created;

-- ============================================================
-- 2. Drop index on ai.review_record(proposal_id)
-- ============================================================
DROP INDEX IF EXISTS ai.idx_review_record_proposal_id;

-- ============================================================
-- 3. Restore ai.action_proposal.action_type CHECK
-- ============================================================
ALTER TABLE ai.action_proposal
    DROP CONSTRAINT IF EXISTS chk_action_proposal_action_type;

ALTER TABLE ai.action_proposal
    ADD CONSTRAINT chk_action_proposal_action_type
    CHECK (action_type IN ('create_followup_task', 'notify_owner', 'export_report', 'escalate_to_human'));

-- ============================================================
-- 4. Restore ai.action_proposal.apply_status CHECK
-- ============================================================
ALTER TABLE ai.action_proposal
    DROP CONSTRAINT IF EXISTS chk_action_proposal_apply_status;

ALTER TABLE ai.action_proposal
    ADD CONSTRAINT chk_action_proposal_apply_status
    CHECK (apply_status IN ('proposed', 'approved', 'rejected'));

-- ============================================================
-- 5. Drop CHECK constraint on ai.review_record.verdict
-- ============================================================
ALTER TABLE ai.review_record
    DROP CONSTRAINT IF EXISTS chk_review_record_verdict;

-- +goose StatementEnd
