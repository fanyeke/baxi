-- +goose Up
-- +goose StatementBegin

-- ai.qoder_run: Qoder agent execution runs (mapped from qoder_runs, 19 rows)
CREATE TABLE ai.qoder_run (
    run_id TEXT PRIMARY KEY,
    run_type TEXT NOT NULL,
    mode TEXT NOT NULL DEFAULT 'read_only',
    status TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ,
    request_id TEXT,
    actor TEXT DEFAULT 'qoder',
    can_apply BOOLEAN DEFAULT FALSE,
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ai.qoder_report: Qoder agent analysis reports (mapped from qoder_reports, 18 rows)
CREATE TABLE ai.qoder_report (
    report_id TEXT PRIMARY KEY,
    run_id TEXT,
    run_type TEXT NOT NULL,
    summary TEXT NOT NULL,
    findings_json JSONB,
    recommended_human_actions_json JSONB,
    risk_level TEXT,
    used_endpoints_json JSONB,
    no_apply_performed BOOLEAN NOT NULL DEFAULT TRUE,
    business_side_effect BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    request_id TEXT
);

-- ai.decision_case: Alert + context aggregation for LLM (NEW)
CREATE TABLE ai.decision_case (
    case_id TEXT PRIMARY KEY,
    alert_id TEXT,
    case_type TEXT,
    status TEXT DEFAULT 'open',
    context_json JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

-- ai.llm_decision: Structured LLM decision output (NEW)
CREATE TABLE ai.llm_decision (
    decision_id TEXT PRIMARY KEY,
    case_id TEXT,
    model_version TEXT,
    prompt_hash TEXT,
    output_json JSONB,
    confidence NUMERIC(4,2),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ai.action_proposal: Proposed actions from LLM (NEW)
CREATE TABLE ai.action_proposal (
    proposal_id TEXT PRIMARY KEY,
    case_id TEXT,
    decision_id TEXT,
    action_type TEXT,
    payload JSONB,
    apply_status TEXT DEFAULT 'pending',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    applied_at TIMESTAMPTZ,
    applied_by TEXT
);

-- ai.review_record: Human review of AI proposals (NEW)
CREATE TABLE ai.review_record (
    review_id TEXT PRIMARY KEY,
    proposal_id TEXT,
    reviewer_id TEXT,
    verdict TEXT,
    feedback TEXT,
    reviewed_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for LLM decision query paths
CREATE INDEX idx_ai_decision_case_status ON ai.decision_case(status, created_at);
CREATE INDEX idx_ai_decision_case_alert ON ai.decision_case(alert_id);
CREATE INDEX idx_ai_llm_decision_case ON ai.llm_decision(case_id);
CREATE INDEX idx_ai_action_proposal_case ON ai.action_proposal(case_id, apply_status);
CREATE INDEX idx_ai_review_record_proposal ON ai.review_record(proposal_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS ai.idx_ai_review_record_proposal;
DROP INDEX IF EXISTS ai.idx_ai_action_proposal_case;
DROP INDEX IF EXISTS ai.idx_ai_llm_decision_case;
DROP INDEX IF EXISTS ai.idx_ai_decision_case_alert;
DROP INDEX IF EXISTS ai.idx_ai_decision_case_status;

DROP TABLE IF EXISTS ai.review_record;
DROP TABLE IF EXISTS ai.action_proposal;
DROP TABLE IF EXISTS ai.llm_decision;
DROP TABLE IF EXISTS ai.decision_case;
DROP TABLE IF EXISTS ai.qoder_report;
DROP TABLE IF EXISTS ai.qoder_run;

-- +goose StatementEnd
