-- v0.2 SQLite Schema: Olist Operations Decision Backend
-- 12 core tables for facts, states, rules, events, and triggers
-- All timestamps stored as TEXT (ISO 8601)

-- 1. Pipeline runs: record every execution (ingest, metrics, rules, export, trigger)
CREATE TABLE IF NOT EXISTS pipeline_runs (
  run_id TEXT PRIMARY KEY,
  run_type TEXT NOT NULL,
  mode TEXT NOT NULL,
  status TEXT NOT NULL,
  started_at TEXT NOT NULL,
  finished_at TEXT,
  input_count INTEGER DEFAULT 0,
  output_count INTEGER DEFAULT 0,
  error_message TEXT
);

-- 2. Ingestion batches: control what gets imported
CREATE TABLE IF NOT EXISTS ingestion_batches (
  batch_id TEXT PRIMARY KEY,
  source_name TEXT NOT NULL,
  ingestion_mode TEXT NOT NULL,
  date_start TEXT,
  date_end TEXT,
  source_file TEXT,
  row_count INTEGER DEFAULT 0,
  status TEXT NOT NULL,
  created_at TEXT NOT NULL
);

-- 3. Order-level DWD: one row per order (PK: order_id)
CREATE TABLE IF NOT EXISTS dwd_order_level (
  order_id TEXT PRIMARY KEY,
  customer_id TEXT,
  customer_unique_id TEXT,
  order_status TEXT,
  order_purchase_timestamp TEXT,
  purchase_date TEXT,
  customer_state TEXT,
  payment_type TEXT,
  payment_installments INTEGER,
  payment_value REAL,
  review_score REAL,
  delivered_customer_date TEXT,
  estimated_delivery_date TEXT,
  delivery_days REAL,
  delay_days REAL,
  is_late INTEGER,
  is_cancelled INTEGER,
  ingestion_batch_id TEXT,
  loaded_at TEXT
);

-- 4. Item-level DWD: one row per order_item (PK: item_key = order_id || '_' || order_item_id)
CREATE TABLE IF NOT EXISTS dwd_item_level (
  item_key TEXT PRIMARY KEY,
  order_id TEXT,
  order_item_id INTEGER,
  product_id TEXT,
  seller_id TEXT,
  product_category_name TEXT,
  product_category_name_english TEXT,
  seller_state TEXT,
  price REAL,
  freight_value REAL,
  ingestion_batch_id TEXT,
  loaded_at TEXT
);

-- 5. Daily metrics: one row per date (PK: metric_date)
CREATE TABLE IF NOT EXISTS metric_daily (
  metric_date TEXT PRIMARY KEY,
  gmv REAL,
  order_count INTEGER,
  customer_count INTEGER,
  seller_count INTEGER,
  avg_order_value REAL,
  freight_value REAL,
  avg_review_score REAL,
  low_review_rate REAL,
  late_delivery_rate REAL,
  cancel_rate REAL,
  payment_installment_rate REAL,
  marketing_seller_share REAL,
  created_at TEXT NOT NULL
);

-- 6. Dimension-level daily metrics: (PK: metric_date, dimension_type, dimension_value, metric_name)
CREATE TABLE IF NOT EXISTS metric_dimension_daily (
  metric_date TEXT NOT NULL,
  dimension_type TEXT NOT NULL,
  dimension_value TEXT NOT NULL,
  metric_name TEXT NOT NULL,
  metric_value REAL,
  sample_size INTEGER,
  created_at TEXT NOT NULL,
  PRIMARY KEY (metric_date, dimension_type, dimension_value, metric_name)
);

-- 7. Alert events: triggered anomalies
CREATE TABLE IF NOT EXISTS alert_events (
  event_id TEXT PRIMARY KEY,
  rule_id TEXT NOT NULL,
  event_date TEXT NOT NULL,
  severity TEXT NOT NULL,
  metric_name TEXT NOT NULL,
  object_type TEXT DEFAULT 'global',
  object_id TEXT DEFAULT 'global',
  current_value REAL,
  baseline_value REAL,
  change_rate REAL,
  sample_size INTEGER,
  evidence_json TEXT,
  description TEXT,
  owner_role TEXT,
  status TEXT DEFAULT 'new',
  created_at TEXT NOT NULL
);

-- 8. Strategy recommendations: decisions derived from events
CREATE TABLE IF NOT EXISTS strategy_recommendations (
  recommendation_id TEXT PRIMARY KEY,
  event_id TEXT,
  decision_source TEXT NOT NULL DEFAULT 'heuristic',
  rule_id TEXT,
  strategy_title TEXT NOT NULL,
  strategy_detail TEXT,
  target_object_type TEXT,
  target_object_id TEXT,
  expected_impact TEXT,
  risk_level TEXT,
  confidence TEXT,
  requires_approval INTEGER DEFAULT 0,
  approval_status TEXT DEFAULT 'draft',
  execution_status TEXT DEFAULT 'draft',
  owner_role TEXT,
  success_metric TEXT,
  created_at TEXT NOT NULL
);

-- 9. Action tasks: execution items
CREATE TABLE IF NOT EXISTS action_tasks (
  task_id TEXT PRIMARY KEY,
  recommendation_id TEXT,
  event_id TEXT,
  task_title TEXT NOT NULL,
  task_description TEXT,
  task_source TEXT DEFAULT 'heuristic_strategy',
  owner_role TEXT,
  owner_user_id TEXT,
  priority TEXT DEFAULT 'medium',
  due_at TEXT,
  status TEXT DEFAULT 'todo',
  feedback TEXT,
  completed_at TEXT,
  created_at TEXT NOT NULL
);

-- 10. Review/retro: post-execution analysis
CREATE TABLE IF NOT EXISTS review_retro (
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
  reviewed_at TEXT
);

-- 11. Event outbox: pending trigger events for external dispatch
CREATE TABLE IF NOT EXISTS event_outbox (
  outbox_id TEXT PRIMARY KEY,
  event_type TEXT NOT NULL,
  source_type TEXT NOT NULL,
  source_id TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  target_channel TEXT NOT NULL,
  status TEXT DEFAULT 'pending',
  dispatch_attempts INTEGER DEFAULT 0,
  last_dispatch_at TEXT,
  external_ref TEXT,
  adapter_name TEXT,
  created_at TEXT NOT NULL,
  processed_at TEXT,
  error_message TEXT
);

-- 12. Qoder jobs: external execution tracking
CREATE TABLE IF NOT EXISTS qoder_jobs (
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
  completed_at TEXT
);
