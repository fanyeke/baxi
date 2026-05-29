-- +goose Up
-- +goose StatementBegin

-- Migration 028: Ensure next_retry_at column exists on ops.outbox_event.
-- Migration 014 already adds this column; this migration uses IF NOT EXISTS
-- so it is safe regardless of which migrations have been applied.
-- The column enables exponential backoff for outbox dispatch retries.

ALTER TABLE ops.outbox_event
  ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMPTZ;

COMMENT ON COLUMN ops.outbox_event.next_retry_at IS 'Scheduled retry time for exponential backoff; NULL = retry immediately';

-- +goose StatementEnd

-- +goose StatementBegin

CREATE INDEX IF NOT EXISTS idx_outbox_event_pending
  ON ops.outbox_event(status, next_retry_at, created_at)
  WHERE status = 'pending';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS ops.idx_outbox_event_pending;

ALTER TABLE ops.outbox_event DROP COLUMN IF EXISTS next_retry_at;

-- +goose StatementEnd
