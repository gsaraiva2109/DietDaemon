-- 025_nudge_rule_config: per-user overrides for macro/health/digest nudge
-- rules. One row per (user, rule_id); params_json holds the rule-specific
-- shape (Rule/HealthRule/DigestRule each differ) so one table covers all
-- three rule kinds without a sparse wide schema.

CREATE TABLE IF NOT EXISTS nudge_rule_config (
    user_id     TEXT NOT NULL REFERENCES users(id),
    rule_id     TEXT NOT NULL,
    enabled     INTEGER NOT NULL DEFAULT 1,
    params_json TEXT NOT NULL DEFAULT '{}',
    PRIMARY KEY (user_id, rule_id)
);
