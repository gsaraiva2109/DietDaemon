-- 023_nudge_rule_config: per-user overrides of scheduler nudge rules.
-- enabled=0 disables a rule entirely; params_json holds a JSON object with a
-- subset of the rule's own field names (e.g. {"MinFraction": 0.9}) that
-- overrides just those tunables, leaving the rest at their hardcoded default.

CREATE TABLE IF NOT EXISTS nudge_rule_config (
    user_id     TEXT NOT NULL,
    rule_id     TEXT NOT NULL,
    enabled     INTEGER NOT NULL DEFAULT 1,
    params_json TEXT NOT NULL DEFAULT '{}',
    PRIMARY KEY (user_id, rule_id)
);
