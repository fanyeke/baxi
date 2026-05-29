-- +goose Up
ALTER TABLE ai.decision_case ADD COLUMN resolution VARCHAR(50) DEFAULT NULL;
ALTER TABLE ai.decision_case ADD COLUMN case_resolution_comment TEXT DEFAULT NULL;

-- +goose Down
ALTER TABLE ai.decision_case DROP COLUMN IF EXISTS case_resolution_comment;
ALTER TABLE ai.decision_case DROP COLUMN IF EXISTS resolution;
