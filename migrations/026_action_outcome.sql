-- +goose Up
CREATE TABLE IF NOT EXISTS ai.action_outcome (
    outcome_id TEXT PRIMARY KEY,
    case_id TEXT NOT NULL,
    proposal_id TEXT NOT NULL,
    action_type TEXT NOT NULL,
    execution_status TEXT NOT NULL,
    business_result TEXT,
    business_impact_json JSONB,
    is_effective BOOLEAN,
    recorded_by TEXT NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_action_outcome_case ON ai.action_outcome(case_id);
CREATE INDEX IF NOT EXISTS idx_action_outcome_proposal ON ai.action_outcome(proposal_id);

-- +goose Down
DROP TABLE IF EXISTS ai.action_outcome;
