-- +goose Up
-- +goose StatementBegin

-- Migration 031: Relax the HITL check constraint to allow low-risk proposals
-- to skip human review. This supports risk-adaptive HITL (Human-In-The-Loop)
-- where low-risk actions can auto-approve.
--
-- Original constraint (from 010): requires_human_review = TRUE
-- New constraint:             requires_human_review = TRUE OR risk_level = 'low'

ALTER TABLE ai.action_proposal DROP CONSTRAINT IF EXISTS chk_action_proposal_requires_review;
ALTER TABLE ai.action_proposal ADD CONSTRAINT chk_action_proposal_requires_review
    CHECK (
        requires_human_review = true
        OR risk_level = 'low'
    );

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Restore the original strict HITL constraint.
ALTER TABLE ai.action_proposal DROP CONSTRAINT IF EXISTS chk_action_proposal_requires_review;
ALTER TABLE ai.action_proposal ADD CONSTRAINT chk_action_proposal_requires_review
    CHECK (requires_human_review = true);

-- +goose StatementEnd
