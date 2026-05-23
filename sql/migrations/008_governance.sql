CREATE TABLE IF NOT EXISTS governance_checkpoints (
    checkpoint_id INTEGER PRIMARY KEY AUTOINCREMENT,
    action_type TEXT NOT NULL,
    endpoint TEXT NOT NULL,
    actor TEXT NOT NULL,
    request_id TEXT,
    justification TEXT,
    mode TEXT NOT NULL DEFAULT 'dry_run',
    status TEXT NOT NULL DEFAULT 'recorded',
    metadata_json TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE IF NOT EXISTS governance_health_results (
    result_id INTEGER PRIMARY KEY AUTOINCREMENT,
    check_id TEXT NOT NULL,
    check_type TEXT NOT NULL,
    status TEXT NOT NULL,
    detail TEXT,
    checked_at TEXT NOT NULL DEFAULT (datetime('now'))
);
