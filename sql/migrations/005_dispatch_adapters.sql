-- v0.4: dispatch_adapters migration
-- Add 4 dispatch tracking columns to event_outbox

ALTER TABLE event_outbox ADD COLUMN dispatch_attempts INTEGER DEFAULT 0;
ALTER TABLE event_outbox ADD COLUMN last_dispatch_at TEXT;
ALTER TABLE event_outbox ADD COLUMN external_ref TEXT;
ALTER TABLE event_outbox ADD COLUMN adapter_name TEXT;
