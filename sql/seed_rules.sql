-- v0.2 Seed rules for Olist Operations Decision Backend
-- Note: Rules are primarily loaded from config/alert_rules.yml
-- This file provides a reference mapping for rule-to-table correspondence

-- Insert rule IDs as metadata reference (informational only, rule engine reads YAML)
INSERT OR IGNORE INTO pipeline_runs (run_id, run_type, mode, status, started_at, finished_at, input_count, output_count, error_message)
VALUES 
  ('seed-init', 'seed', 'init', 'completed', datetime('now'), datetime('now'), 0, 0, NULL);
