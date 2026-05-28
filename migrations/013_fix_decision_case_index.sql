-- +goose Up
-- +goose StatementBegin

-- Migration 013: Fix unique index on ai.decision_case to allow NULL source_type/source_id.
--
-- Problem:
--   Columns are NOT NULL DEFAULT '' and the partial unique index covers all active rows,
--   preventing multiple cases without a real source trigger (e.g., manual creation).
--
-- Fix:
--   1. Drop NOT NULL from source_type and source_id so true NULL is allowed
--   2. Recreate the unique index to exclude NULL rows:
--        - Cases with NULL source_type/source_id can freely coexist
--        - Cases with the same non-null source enforce at-most-one-active

ALTER TABLE ai.decision_case
    ALTER COLUMN source_type DROP NOT NULL,
    ALTER COLUMN source_id DROP NOT NULL;

DROP INDEX IF EXISTS ai.idx_ai_decision_case_active_source;

CREATE UNIQUE INDEX idx_ai_decision_case_active_source
    ON ai.decision_case(source_type, source_id)
    WHERE source_type IS NOT NULL
      AND source_id IS NOT NULL
      AND status NOT IN ('closed', 'failed');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS ai.idx_ai_decision_case_active_source;

CREATE UNIQUE INDEX idx_ai_decision_case_active_source
    ON ai.decision_case(source_type, source_id)
    WHERE status NOT IN ('closed', 'failed');

ALTER TABLE ai.decision_case
    ALTER COLUMN source_type SET NOT NULL,
    ALTER COLUMN source_id SET NOT NULL;

-- +goose StatementEnd
