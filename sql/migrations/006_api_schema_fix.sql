-- v0.6: API schema fix
-- Add missing impact tracking columns to alert_events and target object columns to action_tasks.
-- event_outbox dispatch columns are in 005_dispatch_adapters.sql
-- NOTE: Caller must check column existence before running (SQLite has no IF NOT EXISTS for ADD COLUMN).

-- alert_events: impact tracking columns (used by db_dimensional_rule_engine.py)
ALTER TABLE alert_events ADD COLUMN affected_orders INTEGER;
ALTER TABLE alert_events ADD COLUMN affected_gmv REAL;
ALTER TABLE alert_events ADD COLUMN impact_score REAL;

-- action_tasks: target object columns (used by db_dimensional_rule_engine.py)
ALTER TABLE action_tasks ADD COLUMN target_object_type TEXT;
ALTER TABLE action_tasks ADD COLUMN target_object_id TEXT;
