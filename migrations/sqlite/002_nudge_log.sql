-- 002_nudge_log: dedupe table so each scheduler rule fires at most once per
-- user per local day. The composite PK handles idempotency naturally.

CREATE TABLE IF NOT EXISTS nudge_log (
    user_id    TEXT NOT NULL,
    local_date TEXT NOT NULL,
    rule_id    TEXT NOT NULL,
    sent_at    TEXT NOT NULL,
    PRIMARY KEY (user_id, local_date, rule_id)
);
