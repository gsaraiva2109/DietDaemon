-- Add soft-delete support to chat_sessions so sessions can be restored
-- within a 30-day retention window before permanent purge.
ALTER TABLE chat_sessions ADD COLUMN deleted_at TEXT;
CREATE INDEX IF NOT EXISTS idx_chat_sessions_deleted_at ON chat_sessions(deleted_at);
