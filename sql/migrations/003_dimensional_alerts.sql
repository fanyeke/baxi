-- v0.3 Migration: Dimensional Alerts Schema Extensions
-- Adds columns to alert_events and strategy_recommendations for dimensional support.
-- This file is documentation; actual migration is done via scripts/db_migrate.py
-- to handle SQLite's lack of "IF NOT EXISTS" for ALTER TABLE ADD COLUMN.

-- alert_events extensions
-- ALTER TABLE alert_events ADD COLUMN affected_orders INTEGER;
-- ALTER TABLE alert_events ADD COLUMN affected_gmv REAL;
-- ALTER TABLE alert_events ADD COLUMN impact_score REAL;

-- strategy_recommendations extensions
-- ALTER TABLE strategy_recommendations ADD COLUMN confidence TEXT;
-- ALTER TABLE strategy_recommendations ADD COLUMN target_object_type TEXT;
-- ALTER TABLE strategy_recommendations ADD COLUMN target_object_id TEXT;
