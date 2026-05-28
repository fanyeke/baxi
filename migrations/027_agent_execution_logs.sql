-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS ai.agent_execution (
    execution_id    TEXT PRIMARY KEY,
    session_id      TEXT,
    tool_name       TEXT NOT NULL,
    input_args      JSONB,
    output_result   JSONB,
    status          TEXT NOT NULL,
    error_message   TEXT,
    duration_ms     BIGINT,
    llm_model       TEXT,
    llm_tokens      BIGINT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_ai_agent_execution_session
    ON ai.agent_execution(session_id, created_at);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_ai_agent_execution_tool
    ON ai.agent_execution(tool_name, created_at);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_ai_agent_execution_status
    ON ai.agent_execution(status, created_at);
-- +goose StatementEnd

-- +goose StatementBegin
COMMENT ON TABLE ai.agent_execution IS 'Pi Agent execution logs for decision making';
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS audit.mcp_call (
    call_id         BIGSERIAL PRIMARY KEY,
    request_id      TEXT,
    server_name     TEXT NOT NULL,
    tool_name       TEXT NOT NULL,
    input_args      JSONB,
    output_result   JSONB,
    status          TEXT NOT NULL,
    error_message   TEXT,
    duration_ms     BIGINT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_audit_mcp_call_server
    ON audit.mcp_call(server_name, created_at);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_audit_mcp_call_tool
    ON audit.mcp_call(tool_name, created_at);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_audit_mcp_call_status
    ON audit.mcp_call(status, created_at);
-- +goose StatementEnd

-- +goose StatementBegin
COMMENT ON TABLE audit.mcp_call IS 'MCP tool call logs from Pi Agent';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS audit.mcp_call;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS ai.agent_execution;
-- +goose StatementEnd
