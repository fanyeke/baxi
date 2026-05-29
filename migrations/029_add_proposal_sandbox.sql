-- +goose Up
-- +goose StatementBegin

-- Migration 029: Add proposal_sandbox table for persistent proposal sandbox/workspace.
-- Enables the create_sandbox, add_to_sandbox, compare_sandboxes, and get_sandbox MCP tools.

CREATE TABLE IF NOT EXISTS ai.proposal_sandbox (
    sandbox_id      TEXT PRIMARY KEY,
    case_id         TEXT NOT NULL REFERENCES ai.decision_case(case_id),
    proposal_id     TEXT REFERENCES ai.action_proposal(proposal_id),
    sandbox_data    JSONB NOT NULL DEFAULT '{}',
    status          TEXT NOT NULL DEFAULT 'draft',
    compared_with   TEXT[],
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ
);

COMMENT ON TABLE ai.proposal_sandbox IS 'Persistent proposal sandbox for comparing action proposals before execution';
COMMENT ON COLUMN ai.proposal_sandbox.sandbox_id IS 'Unique sandbox identifier (sbx_ prefix)';
COMMENT ON COLUMN ai.proposal_sandbox.case_id IS 'The decision case this sandbox belongs to';
COMMENT ON COLUMN ai.proposal_sandbox.proposal_id IS 'Optional linked action proposal';
COMMENT ON COLUMN ai.proposal_sandbox.sandbox_data IS 'Arbitrary JSON data stored in the sandbox';
COMMENT ON COLUMN ai.proposal_sandbox.status IS 'Sandbox status: draft, active, completed';
COMMENT ON COLUMN ai.proposal_sandbox.compared_with IS 'Array of sandbox IDs that this sandbox has been compared with';

CREATE INDEX IF NOT EXISTS idx_proposal_sandbox_case_id
    ON ai.proposal_sandbox(case_id);

CREATE INDEX IF NOT EXISTS idx_proposal_sandbox_status
    ON ai.proposal_sandbox(status);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS ai.idx_proposal_sandbox_case_id;
DROP INDEX IF EXISTS ai.idx_proposal_sandbox_status;
DROP TABLE IF EXISTS ai.proposal_sandbox;

-- +goose StatementEnd
