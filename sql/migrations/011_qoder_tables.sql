-- v0.5.3 Migration 011: Qoder runs and reports tables
-- Create tables for tracking Qoder execution runs and generated reports.
-- All timestamps stored as TEXT (ISO 8601).

-- 1. Qoder runs: track each Qoder execution run
CREATE TABLE IF NOT EXISTS qoder_runs (
  run_id TEXT PRIMARY KEY,
  run_type TEXT NOT NULL,
  mode TEXT NOT NULL DEFAULT 'read_only',
  status TEXT NOT NULL,
  started_at TEXT NOT NULL,
  finished_at TEXT,
  request_id TEXT,
  actor TEXT DEFAULT 'qoder',
  can_apply INTEGER DEFAULT 0,
  error_message TEXT
);

-- 2. Qoder reports: reports generated from Qoder runs
CREATE TABLE IF NOT EXISTS qoder_reports (
  report_id TEXT PRIMARY KEY,
  run_id TEXT,
  run_type TEXT NOT NULL,
  summary TEXT NOT NULL,
  findings_json TEXT,
  recommended_human_actions_json TEXT,
  risk_level TEXT,
  used_endpoints_json TEXT,
  no_apply_performed INTEGER NOT NULL DEFAULT 1,
  business_side_effect INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  request_id TEXT
);

-- Indexes for qoder_runs
CREATE INDEX IF NOT EXISTS idx_qoder_runs_started_at ON qoder_runs(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_qoder_runs_status ON qoder_runs(status);
CREATE INDEX IF NOT EXISTS idx_qoder_runs_type ON qoder_runs(run_type);

-- Indexes for qoder_reports
CREATE INDEX IF NOT EXISTS idx_qoder_reports_created_at ON qoder_reports(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_qoder_reports_run_id ON qoder_reports(run_id);
CREATE INDEX IF NOT EXISTS idx_qoder_reports_risk_level ON qoder_reports(risk_level);