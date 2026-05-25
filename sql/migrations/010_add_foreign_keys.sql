-- Task 10: Add foreign key constraints
-- Note: SQLite does not support ALTER TABLE ADD FOREIGN KEY.
-- Tables must be rebuilt with the constraint included.

PRAGMA foreign_keys = OFF;

-- 1. action_tasks.recommendation_id -> strategy_recommendations.recommendation_id
CREATE TABLE action_tasks_new (
  task_id TEXT PRIMARY KEY,
  recommendation_id TEXT,
  event_id TEXT,
  task_title TEXT NOT NULL,
  task_description TEXT,
  target_object_type TEXT,
  target_object_id TEXT,
  task_source TEXT DEFAULT 'heuristic_strategy',
  owner_role TEXT,
  owner_user_id TEXT,
  priority TEXT DEFAULT 'medium',
  due_at TEXT,
  status TEXT DEFAULT 'todo',
  feedback TEXT,
  completed_at TEXT,
  created_at TEXT NOT NULL,
  FOREIGN KEY (recommendation_id) REFERENCES strategy_recommendations(recommendation_id)
);
INSERT INTO action_tasks_new SELECT * FROM action_tasks;
DROP TABLE action_tasks;
ALTER TABLE action_tasks_new RENAME TO action_tasks;

-- 2. qoder_jobs.trigger_event_id -> alert_events.event_id
CREATE TABLE qoder_jobs_new (
  job_id TEXT PRIMARY KEY,
  trigger_event_id TEXT,
  job_type TEXT NOT NULL,
  job_title TEXT NOT NULL,
  job_context_json TEXT,
  dispatch_channel TEXT NOT NULL,
  dispatch_status TEXT DEFAULT 'pending',
  external_ref TEXT,
  created_at TEXT NOT NULL,
  dispatched_at TEXT,
  completed_at TEXT,
  FOREIGN KEY (trigger_event_id) REFERENCES alert_events(event_id)
);
INSERT INTO qoder_jobs_new SELECT * FROM qoder_jobs;
DROP TABLE qoder_jobs;
ALTER TABLE qoder_jobs_new RENAME TO qoder_jobs;

-- 3. review_retro.recommendation_id -> strategy_recommendations.recommendation_id
CREATE TABLE review_retro_new (
  review_id TEXT PRIMARY KEY,
  status TEXT DEFAULT 'draft',
  feedback TEXT,
  recommendation_id TEXT,
  task_id TEXT,
  review_type TEXT DEFAULT 'simulated',
  review_source TEXT DEFAULT 'hindsight_rule',
  actual_result TEXT,
  actual_impact TEXT,
  is_effective INTEGER,
  lesson_learned TEXT,
  promote_to_rule INTEGER DEFAULT 0,
  reviewed_at TEXT,
  FOREIGN KEY (recommendation_id) REFERENCES strategy_recommendations(recommendation_id)
);
INSERT INTO review_retro_new SELECT * FROM review_retro;
DROP TABLE review_retro;
ALTER TABLE review_retro_new RENAME TO review_retro;

PRAGMA foreign_keys = ON;
