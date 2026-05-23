-- v0.7: Add status/feedback columns to review_retro for Feishu status feedback loop
-- NOTE: Caller must check column existence before running (SQLite has no IF NOT EXISTS for ADD COLUMN).

ALTER TABLE review_retro ADD COLUMN status TEXT DEFAULT 'draft';
ALTER TABLE review_retro ADD COLUMN feedback TEXT;
