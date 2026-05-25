-- v0.2 Migration 002: Seed Data and Indexes
-- Creates performance indexes and inserts initial seed data.
-- Rules are primarily loaded from config/alert_rules.yml.

-- === Indexes ===

-- dwd_order_level: date-based queries
CREATE INDEX IF NOT EXISTS idx_dwd_order_purchase_date ON dwd_order_level(purchase_date);
CREATE INDEX IF NOT EXISTS idx_dwd_order_customer ON dwd_order_level(customer_unique_id);
CREATE INDEX IF NOT EXISTS idx_dwd_order_status ON dwd_order_level(order_status);
CREATE INDEX IF NOT EXISTS idx_dwd_order_batch ON dwd_order_level(ingestion_batch_id);
CREATE INDEX IF NOT EXISTS idx_dwd_order_cancelled ON dwd_order_level(is_cancelled);
CREATE INDEX IF NOT EXISTS idx_dwd_order_late ON dwd_order_level(is_late);

-- dwd_item_level: seller/category queries, FK-style lookups
CREATE INDEX IF NOT EXISTS idx_dwd_item_order ON dwd_item_level(order_id);
CREATE INDEX IF NOT EXISTS idx_dwd_item_seller ON dwd_item_level(seller_id);
CREATE INDEX IF NOT EXISTS idx_dwd_item_category ON dwd_item_level(product_category_name);
CREATE INDEX IF NOT EXISTS idx_dwd_item_batch ON dwd_item_level(ingestion_batch_id);

-- metric_daily: range queries for rolling windows
CREATE INDEX IF NOT EXISTS idx_metric_date ON metric_daily(metric_date);

-- metric_dimension_daily: dimension lookups
CREATE INDEX IF NOT EXISTS idx_metric_dim_date_type ON metric_dimension_daily(metric_date, dimension_type);
CREATE INDEX IF NOT EXISTS idx_metric_dim_value ON metric_dimension_daily(dimension_type, dimension_value);

-- alert_events: querying by date, rule, severity
CREATE INDEX IF NOT EXISTS idx_alert_event_date ON alert_events(event_date);
CREATE INDEX IF NOT EXISTS idx_alert_rule ON alert_events(rule_id);
CREATE INDEX IF NOT EXISTS idx_alert_severity ON alert_events(severity);
CREATE INDEX IF NOT EXISTS idx_alert_status ON alert_events(status);
CREATE INDEX IF NOT EXISTS idx_alert_owner ON alert_events(owner_role);

-- strategy_recommendations: status/source queries
CREATE INDEX IF NOT EXISTS idx_strategy_event ON strategy_recommendations(event_id);
CREATE INDEX IF NOT EXISTS idx_strategy_status ON strategy_recommendations(execution_status);
CREATE INDEX IF NOT EXISTS idx_strategy_source ON strategy_recommendations(decision_source);

-- action_tasks: status/priority queries
CREATE INDEX IF NOT EXISTS idx_action_status ON action_tasks(status);
CREATE INDEX IF NOT EXISTS idx_action_priority ON action_tasks(priority);
CREATE INDEX IF NOT EXISTS idx_action_owner ON action_tasks(owner_role);
CREATE INDEX IF NOT EXISTS idx_action_due ON action_tasks(due_at);

-- event_outbox: status/channel dispatch
CREATE INDEX IF NOT EXISTS idx_outbox_status_channel ON event_outbox(status, target_channel);
CREATE INDEX IF NOT EXISTS idx_outbox_status ON event_outbox(status);
CREATE INDEX IF NOT EXISTS idx_outbox_channel ON event_outbox(target_channel);

-- qoder_jobs: dispatch status
CREATE INDEX IF NOT EXISTS idx_qoder_status ON qoder_jobs(dispatch_status);
CREATE INDEX IF NOT EXISTS idx_qoder_channel ON qoder_jobs(dispatch_channel);

-- pipeline_runs: query by type/status
CREATE INDEX IF NOT EXISTS idx_pipeline_type ON pipeline_runs(run_type);
CREATE INDEX IF NOT EXISTS idx_pipeline_status ON pipeline_runs(status);

-- ingestion_batches: query by source/status
CREATE INDEX IF NOT EXISTS idx_ingest_source ON ingestion_batches(source_name);
CREATE INDEX IF NOT EXISTS idx_ingest_status ON ingestion_batches(status);

-- === Seed Data ===

INSERT OR IGNORE INTO pipeline_runs (run_id, run_type, mode, status, started_at, finished_at, input_count, output_count, error_message)
VALUES
  ('seed-init', 'seed', 'init', 'completed', datetime('now'), datetime('now'), 0, 0, NULL);
