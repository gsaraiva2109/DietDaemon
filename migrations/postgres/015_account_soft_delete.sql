-- Add soft-delete support to accounts so a deletion request goes through a
-- recovery window before progress photos are purged (day 30) and the
-- account is hard-deleted (day 90), mirroring the chat_sessions pattern in
-- 002_chat_soft_delete.sql.
ALTER TABLE accounts ADD COLUMN deleted_at TEXT;
ALTER TABLE accounts ADD COLUMN photos_purged_at TEXT;
CREATE INDEX IF NOT EXISTS idx_accounts_deleted_at ON accounts(deleted_at);
