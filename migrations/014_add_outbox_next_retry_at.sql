-- +goose Up
-- +goose StatementBegin

-- Migration 014: Add next_retry_at column to ops.outbox_event for exponential
-- backoff support. The worker uses this column to determine when a failed event
-- should be retried (NULL = retry immediately, otherwise retry after timestamp).

ALTER TABLE ops.outbox_event
  ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMPTZ;

COMMENT ON COLUMN ops.outbox_event.next_retry_at IS 'Scheduled retry time for exponential backoff; NULL = retry immediately';

-- +goose StatementEnd

-- +goose StatementBegin

CREATE INDEX IF NOT EXISTS idx_outbox_event_pending
  ON ops.outbox_event(status, next_retry_at, created_at)
  WHERE status = 'pending';

-- +goose StatementEnd

-- +goose StatementBegin

DROP INDEX IF EXISTS ops.idx_ops_outbox_event_status;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS ops.idx_outbox_event_pending;

ALTER TABLE ops.outbox_event DROP COLUMN IF EXISTS next_retry_at;

CREATE INDEX IF NOT EXISTS idx_ops_outbox_event_status ON ops.outbox_event(status);

-- +goose StatementEnd
