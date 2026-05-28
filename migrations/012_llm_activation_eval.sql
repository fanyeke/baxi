-- +goose Up
-- +goose StatementBegin

-- Migration 012: Extend ai.llm_decision with LLM billing/audit columns
-- and create ai.decision_eval_result table for Phase 8 LLM activation/eval.

-- ============================================================
-- 1. Add LLM audit columns to ai.llm_decision
--    Columns like provider, model, prompt tracking, I/O payload,
--    validation status, fallback tracking, and token usage.
-- ============================================================
ALTER TABLE ai.llm_decision
  ADD COLUMN IF NOT EXISTS provider TEXT,
  ADD COLUMN IF NOT EXISTS model TEXT,
  ADD COLUMN IF NOT EXISTS prompt_id TEXT,
  ADD COLUMN IF NOT EXISTS prompt_version TEXT,
  ADD COLUMN IF NOT EXISTS prompt_hash TEXT,
  ADD COLUMN IF NOT EXISTS context_hash TEXT,
  ADD COLUMN IF NOT EXISTS input_json JSONB,
  ADD COLUMN IF NOT EXISTS raw_output TEXT,
  ADD COLUMN IF NOT EXISTS parsed_output_json JSONB,
  ADD COLUMN IF NOT EXISTS validation_status TEXT,
  ADD COLUMN IF NOT EXISTS fallback_used BOOLEAN DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS fallback_reason TEXT,
  ADD COLUMN IF NOT EXISTS token_prompt INT,
  ADD COLUMN IF NOT EXISTS token_completion INT,
  ADD COLUMN IF NOT EXISTS latency_ms INT;

-- ============================================================
-- 2. Create ai.decision_eval_result table
--    Stores evaluation scores for LLM decisions.
-- ============================================================
CREATE TABLE IF NOT EXISTS ai.decision_eval_result (
  eval_id TEXT PRIMARY KEY,
  decision_case_id TEXT NOT NULL,
  llm_decision_id TEXT,
  eval_rule_id TEXT,
  eval_status TEXT NOT NULL,
  score NUMERIC(4,2),
  details_json JSONB,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================================
-- 3. Indexes for query performance
-- ============================================================
CREATE INDEX IF NOT EXISTS idx_llm_decision_provider
    ON ai.llm_decision(provider, created_at);

CREATE INDEX IF NOT EXISTS idx_llm_decision_fallback
    ON ai.llm_decision(fallback_used, created_at);

CREATE INDEX IF NOT EXISTS idx_eval_result_case
    ON ai.decision_eval_result(decision_case_id);

CREATE INDEX IF NOT EXISTS idx_eval_result_decision
    ON ai.decision_eval_result(llm_decision_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- ============================================================
-- 1. Drop indexes
-- ============================================================
DROP INDEX IF EXISTS ai.idx_eval_result_decision;
DROP INDEX IF EXISTS ai.idx_eval_result_case;
DROP INDEX IF EXISTS ai.idx_llm_decision_fallback;
DROP INDEX IF EXISTS ai.idx_llm_decision_provider;

-- ============================================================
-- 2. Drop table
-- ============================================================
DROP TABLE IF EXISTS ai.decision_eval_result;

-- ============================================================
-- 3. Drop columns from ai.llm_decision
-- ============================================================
ALTER TABLE ai.llm_decision
  DROP COLUMN IF EXISTS provider,
  DROP COLUMN IF EXISTS model,
  DROP COLUMN IF EXISTS prompt_id,
  DROP COLUMN IF EXISTS prompt_version,
  DROP COLUMN IF EXISTS prompt_hash,
  DROP COLUMN IF EXISTS context_hash,
  DROP COLUMN IF EXISTS input_json,
  DROP COLUMN IF EXISTS raw_output,
  DROP COLUMN IF EXISTS parsed_output_json,
  DROP COLUMN IF EXISTS validation_status,
  DROP COLUMN IF EXISTS fallback_used,
  DROP COLUMN IF EXISTS fallback_reason,
  DROP COLUMN IF EXISTS token_prompt,
  DROP COLUMN IF EXISTS token_completion,
  DROP COLUMN IF EXISTS latency_ms;

-- +goose StatementEnd
