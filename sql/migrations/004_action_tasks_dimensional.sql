-- v0.3.1 Migration: Action Tasks Dimensional Columns
-- Adds target_object_type and target_object_id to action_tasks for dimensional tracking.
-- This file is documentation; actual migration is done via scripts/db_migrate.py

-- ALTER TABLE action_tasks ADD COLUMN target_object_type TEXT;
-- ALTER TABLE action_tasks ADD COLUMN target_object_id TEXT;
