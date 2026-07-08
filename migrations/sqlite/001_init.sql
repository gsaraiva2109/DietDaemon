-- 001_init: DietDaemon schema (SQLite). Greenfield rewrite — no real users/
-- data existed at the time of this rewrite, so this is the single source of
-- truth (no incremental migrations layered on top).

CREATE TABLE IF NOT EXISTS accounts (
    id         TEXT PRIMARY KEY,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- users: tenant-scoped (account_id) with all login identity columns folded in
-- (email/password auth, WebAuthn, locale). timezone drives local-day nudge
-- scheduling.
CREATE TABLE IF NOT EXISTS users (
    id                TEXT PRIMARY KEY,
    account_id        TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    timezone          TEXT NOT NULL DEFAULT '',
    email             TEXT,
    email_verified_at TEXT,
    status            TEXT NOT NULL DEFAULT 'active',
    display_name      TEXT,
    webauthn_handle   TEXT,           -- base64, NULL until first passkey
    locale            TEXT NOT NULL DEFAULT '',  -- BCP-47 tag (e.g. "en", "pt-BR"); '' = auto-detect
    created_at        TEXT NOT NULL DEFAULT (datetime('now'))
);
-- App lowercases email before write/compare; NULL emails are exempt (bot-only users).
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_account_email ON users(account_id, email) WHERE email IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_webauthn_handle ON users(webauthn_handle);
CREATE INDEX IF NOT EXISTS idx_users_account ON users(account_id);

CREATE TABLE IF NOT EXISTS meals (
    id          TEXT PRIMARY KEY,
    user_id     TEXT    NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    at_utc      TEXT    NOT NULL,
    raw_text    TEXT    NOT NULL,
    confidence  REAL    NOT NULL DEFAULT 0,
    parser_tier INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT    NOT NULL DEFAULT (datetime('now'))
);
-- Covers GetMealsInRange (user_id + date(at_utc) range scan).
CREATE INDEX IF NOT EXISTS idx_meals_user_at ON meals(user_id, at_utc);
CREATE INDEX IF NOT EXISTS idx_meals_user_date_at ON meals(user_id, date(at_utc));
-- Covers RecentMeals (user_id filter, created_at DESC LIMIT n).
CREATE INDEX IF NOT EXISTS idx_meals_user_created ON meals(user_id, created_at);

-- resolved_items: one row per parsed/resolved food item within a meal.
-- food_name/source/macro columns are a frozen snapshot at log time
-- (intentional — a later correction to `foods` must not rewrite history).
-- `position` replaces relying on SQLite's implicit rowid for item ordering
-- (not a portable concept on Postgres).
CREATE TABLE IF NOT EXISTS resolved_items (
    id               TEXT PRIMARY KEY,
    meal_id          TEXT    NOT NULL REFERENCES meals(id) ON DELETE CASCADE,
    position         INTEGER NOT NULL DEFAULT 0,
    raw_phrase       TEXT    NOT NULL DEFAULT '',
    quantity         REAL    NOT NULL DEFAULT 0,
    unit             TEXT    NOT NULL DEFAULT '',
    normalized_grams REAL    NOT NULL DEFAULT 0,
    food_id          TEXT    NOT NULL DEFAULT '',
    food_name        TEXT    NOT NULL DEFAULT '',
    source           TEXT    NOT NULL DEFAULT '',
    match_score      REAL    NOT NULL DEFAULT 0,
    kcal             REAL    NOT NULL DEFAULT 0,
    protein          REAL    NOT NULL DEFAULT 0,
    carbs            REAL    NOT NULL DEFAULT 0,
    fat              REAL    NOT NULL DEFAULT 0,
    fiber            REAL    NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_resolved_items_meal ON resolved_items(meal_id, position);

-- foods: global food catalog, one row per external food shared by every user.
-- Resolved once from USDA/TACO/OpenFoodFacts/etc.; never duplicated per user.
CREATE TABLE IF NOT EXISTS foods (
    food_id       TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    source        TEXT NOT NULL DEFAULT '',
    kcal_100g     REAL NOT NULL DEFAULT 0,
    protein_100g  REAL NOT NULL DEFAULT 0,
    carbs_100g    REAL NOT NULL DEFAULT 0,
    fat_100g      REAL NOT NULL DEFAULT 0,
    fiber_100g    REAL NOT NULL DEFAULT 0,
    category      TEXT NOT NULL DEFAULT '',
    brand         TEXT NOT NULL DEFAULT '',
    barcode       TEXT NOT NULL DEFAULT '',
    image_url     TEXT NOT NULL DEFAULT '',
    serving_size  REAL NOT NULL DEFAULT 0,
    serving_unit  TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at    TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_foods_barcode ON foods(barcode) WHERE barcode != '';
CREATE INDEX IF NOT EXISTS idx_foods_source ON foods(source);

-- user_food_stats: thin per-user usage-stats junction over the global foods
-- catalog. Was `food_library` — nutrition-fact columns moved to `foods`
-- above; only genuinely per-user data (how often/recently a user ate this
-- food) remains here.
CREATE TABLE IF NOT EXISTS user_food_stats (
    user_id     TEXT    NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    food_id     TEXT    NOT NULL REFERENCES foods(food_id) ON DELETE CASCADE,
    query_count INTEGER NOT NULL DEFAULT 0,
    last_used   TEXT    NOT NULL DEFAULT '',
    PRIMARY KEY (user_id, food_id)
);
CREATE INDEX IF NOT EXISTS idx_user_food_stats_last_used ON user_food_stats(user_id, last_used DESC);
CREATE INDEX IF NOT EXISTS idx_user_food_stats_frequent ON user_food_stats(user_id, query_count DESC, last_used DESC);

CREATE TABLE IF NOT EXISTS food_aliases (
    user_id          TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    alias_normalized TEXT NOT NULL,
    food_id          TEXT NOT NULL REFERENCES foods(food_id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, alias_normalized)
);
CREATE INDEX IF NOT EXISTS idx_food_aliases_food ON food_aliases(user_id, food_id);

-- food_vectors: global food embedding vectors for nearest-neighbor matching
-- (Tier-1/Tier-2). An embedding is a pure function of the food name, so it's
-- computed and stored once per food_id globally, never per user. vec is a
-- little-endian float32 BLOB; dim records the vector length for sanity
-- checks on load.
CREATE TABLE IF NOT EXISTS food_vectors (
    food_id TEXT PRIMARY KEY REFERENCES foods(food_id) ON DELETE CASCADE,
    dim     INTEGER NOT NULL,
    vec     BLOB NOT NULL
);

-- food_search: FTS5 full-text index over the global food catalog plus each
-- user's personal aliases. Rows come in two kinds:
--   - one global row per food (user_id = ''), synced from `foods`
--   - one row per user alias (user_id = <owner>), synced from `food_aliases`
-- Search is `MATCH ? AND (user_id = '' OR user_id = ?)`.
CREATE VIRTUAL TABLE IF NOT EXISTS food_search USING fts5(
    food_id,
    user_id,
    name,
    alias
);

CREATE TRIGGER IF NOT EXISTS foods_fts_insert AFTER INSERT ON foods BEGIN
    INSERT INTO food_search(food_id, user_id, name, alias)
    VALUES (NEW.food_id, '', NEW.name, '');
END;

CREATE TRIGGER IF NOT EXISTS foods_fts_update AFTER UPDATE ON foods BEGIN
    DELETE FROM food_search WHERE food_id = OLD.food_id AND user_id = '';
    INSERT INTO food_search(food_id, user_id, name, alias)
    VALUES (NEW.food_id, '', NEW.name, '');
END;

CREATE TRIGGER IF NOT EXISTS foods_fts_delete AFTER DELETE ON foods BEGIN
    DELETE FROM food_search WHERE food_id = OLD.food_id;
END;

CREATE TRIGGER IF NOT EXISTS food_aliases_fts_insert AFTER INSERT ON food_aliases BEGIN
    INSERT INTO food_search(food_id, user_id, name, alias)
    SELECT f.food_id, NEW.user_id, f.name, NEW.alias_normalized
    FROM foods f
    WHERE f.food_id = NEW.food_id;
END;

CREATE TRIGGER IF NOT EXISTS food_aliases_fts_delete AFTER DELETE ON food_aliases BEGIN
    DELETE FROM food_search
    WHERE food_id = OLD.food_id AND user_id = OLD.user_id AND alias = OLD.alias_normalized;
END;

-- pending_aliases: embedding-matched aliases awaiting user confirmation.
-- Replaces the silent write-back into food_aliases for near-miss matches:
-- the match is parked here until the user confirms or rejects it.
CREATE TABLE IF NOT EXISTS pending_aliases (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    phrase      TEXT NOT NULL,      -- raw phrase that triggered the match
    food_id     TEXT NOT NULL REFERENCES foods(food_id) ON DELETE CASCADE,
    match_score REAL NOT NULL,      -- embedding similarity score
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_pending_aliases_user ON pending_aliases(user_id);

-- source_precedence: per-user override of the nutrition source resolution
-- order (default order comes from the NUTRITION_SOURCE env var).
CREATE TABLE IF NOT EXISTS source_precedence (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source  TEXT NOT NULL,      -- source name, e.g. "openfoodfacts"
    rank    INTEGER NOT NULL,   -- 0 = tried first
    PRIMARY KEY (user_id, source)
);

CREATE TABLE IF NOT EXISTS daily_targets (
    user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    kcal    REAL NOT NULL DEFAULT 0,
    protein REAL NOT NULL DEFAULT 0,
    carbs   REAL NOT NULL DEFAULT 0,
    fat     REAL NOT NULL DEFAULT 0,
    fiber   REAL NOT NULL DEFAULT 0
);

-- daily_rollups: target_* columns are a frozen snapshot at the time of the
-- day's activity (intentional — a later targets change must not retroactively
-- rewrite historical days; see UpdateRollupTargets).
CREATE TABLE IF NOT EXISTS daily_rollups (
    user_id          TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    date             TEXT NOT NULL,
    consumed_kcal    REAL NOT NULL DEFAULT 0,
    consumed_protein REAL NOT NULL DEFAULT 0,
    consumed_carbs   REAL NOT NULL DEFAULT 0,
    consumed_fat     REAL NOT NULL DEFAULT 0,
    consumed_fiber   REAL NOT NULL DEFAULT 0,
    target_kcal      REAL NOT NULL DEFAULT 0,
    target_protein   REAL NOT NULL DEFAULT 0,
    target_carbs     REAL NOT NULL DEFAULT 0,
    target_fat       REAL NOT NULL DEFAULT 0,
    target_fiber     REAL NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, date)
);

-- fasts: intermittent-fasting windows. A user has at most one active fast
-- (end_at IS NULL) at a time.
CREATE TABLE IF NOT EXISTS fasts (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    start_at     TEXT NOT NULL,
    end_at       TEXT,                       -- NULL = still fasting
    target_hours REAL NOT NULL DEFAULT 16,
    completed    INTEGER NOT NULL DEFAULT 0, -- 1 once target reached at end
    created_at   TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_fasts_user_start ON fasts(user_id, start_at);
-- Fast lookup of the single in-progress fast per user.
CREATE INDEX IF NOT EXISTS idx_fasts_active ON fasts(user_id) WHERE end_at IS NULL;

-- water_logs: water consumption tracking. Each row records one water entry
-- (e.g. a glass or bottle). logged_date is generated so GetWaterToday/
-- GetWaterDailyTotals can use a plain index instead of a function-wrapped
-- date(logged_at) predicate.
CREATE TABLE IF NOT EXISTS water_logs (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount_ml   INTEGER NOT NULL,
    logged_at   TEXT NOT NULL,
    logged_date TEXT GENERATED ALWAYS AS (date(logged_at)) STORED,
    note        TEXT,
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_water_logs_user_date ON water_logs(user_id, logged_date);

-- workouts/workout_exercises: exercise session tracking with individual
-- exercises. external_id supports idempotent imports (Hevy, MyFitnessPal,
-- etc.) — NULL for manually logged workouts, the overwhelming majority.
CREATE TABLE IF NOT EXISTS workouts (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    duration_min    INTEGER NOT NULL,
    intensity       TEXT NOT NULL DEFAULT 'moderate',
    calories_burned INTEGER,
    note            TEXT,
    logged_at       TEXT NOT NULL,
    logged_date     TEXT GENERATED ALWAYS AS (date(logged_at)) STORED,
    external_id     TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_workouts_user_date ON workouts(user_id, logged_date);
CREATE UNIQUE INDEX IF NOT EXISTS idx_workouts_user_external
    ON workouts(user_id, external_id) WHERE external_id IS NOT NULL;

-- workout_exercises: `position` replaces relying on SQLite's implicit rowid
-- for exercise ordering (not a portable concept on Postgres).
CREATE TABLE IF NOT EXISTS workout_exercises (
    id         TEXT PRIMARY KEY,
    workout_id TEXT NOT NULL REFERENCES workouts(id) ON DELETE CASCADE,
    position   INTEGER NOT NULL DEFAULT 0,
    name       TEXT NOT NULL,
    sets       INTEGER,
    reps       INTEGER,
    weight_kg  REAL,
    note       TEXT
);
CREATE INDEX IF NOT EXISTS idx_workout_exercises_workout ON workout_exercises(workout_id, position);

-- sleep_logs: sleep session tracking. A sleep log has wake_at NULL while the
-- session is in progress.
CREATE TABLE IF NOT EXISTS sleep_logs (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sleep_at   TEXT NOT NULL,
    wake_at    TEXT,
    quality    TEXT NOT NULL DEFAULT 'ok',
    note       TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_sleep_logs_user_date ON sleep_logs(user_id, sleep_at);
-- Fast lookup of the single in-progress sleep session per user, mirrors idx_fasts_active.
CREATE INDEX IF NOT EXISTS idx_sleep_logs_active ON sleep_logs(user_id) WHERE wake_at IS NULL;

-- weight_log/measurement_log: UNIQUE(user_id, date) — one entry per user per
-- day; logging again the same day overwrites (upsert-by-date, not by id).
CREATE TABLE IF NOT EXISTS weight_log (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    date       TEXT NOT NULL,
    weight_kg  REAL NOT NULL,
    note       TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE (user_id, date)
);
CREATE INDEX IF NOT EXISTS idx_weight_log_user_date ON weight_log(user_id, date);

CREATE TABLE IF NOT EXISTS measurement_log (
    id             TEXT PRIMARY KEY,
    user_id        TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    date           TEXT NOT NULL,
    waist_cm       REAL NOT NULL DEFAULT 0,
    hips_cm        REAL NOT NULL DEFAULT 0,
    chest_cm       REAL NOT NULL DEFAULT 0,
    left_arm_cm    REAL NOT NULL DEFAULT 0,
    right_arm_cm   REAL NOT NULL DEFAULT 0,
    left_thigh_cm  REAL NOT NULL DEFAULT 0,
    right_thigh_cm REAL NOT NULL DEFAULT 0,
    note           TEXT NOT NULL DEFAULT '',
    created_at     TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE (user_id, date)
);
CREATE INDEX IF NOT EXISTS idx_measurement_log_user_date ON measurement_log(user_id, date);

CREATE TABLE IF NOT EXISTS progress_photos (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    date       TEXT NOT NULL,
    view       TEXT NOT NULL CHECK(view IN ('front', 'side', 'back')),
    mime_type  TEXT NOT NULL,
    data       BLOB NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_progress_photos_user_date ON progress_photos(user_id, date);

-- user_profiles: per-user profile for TDEE calculation and goal tracking.
CREATE TABLE IF NOT EXISTS user_profiles (
    user_id          TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    height_cm        REAL NOT NULL DEFAULT 0,
    birth_date       TEXT NOT NULL DEFAULT '',
    gender           TEXT NOT NULL DEFAULT '',
    activity_level   TEXT NOT NULL DEFAULT '',
    goal             TEXT NOT NULL DEFAULT '',
    target_weight_kg REAL NOT NULL DEFAULT 0,
    weekly_rate      REAL NOT NULL DEFAULT 0,
    onboarded        INTEGER NOT NULL DEFAULT 0,
    created_at       TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at       TEXT NOT NULL DEFAULT (datetime('now'))
);

-- meal_templates/meal_template_items: reusable meal shortcuts. Items are
-- relational rows (mirrors resolved_items) instead of a items_json blob, so
-- they're queryable/indexable and consistent with how meals store items.
CREATE TABLE IF NOT EXISTS meal_templates (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    last_used  TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_meal_templates_user ON meal_templates(user_id);

CREATE TABLE IF NOT EXISTS meal_template_items (
    id               TEXT PRIMARY KEY,
    template_id      TEXT    NOT NULL REFERENCES meal_templates(id) ON DELETE CASCADE,
    position         INTEGER NOT NULL DEFAULT 0,
    raw_phrase       TEXT    NOT NULL DEFAULT '',
    quantity         REAL    NOT NULL DEFAULT 0,
    unit             TEXT    NOT NULL DEFAULT '',
    normalized_grams REAL    NOT NULL DEFAULT 0,
    food_id          TEXT    NOT NULL DEFAULT '',
    food_name        TEXT    NOT NULL DEFAULT '',
    source           TEXT    NOT NULL DEFAULT '',
    match_score      REAL    NOT NULL DEFAULT 0,
    kcal             REAL    NOT NULL DEFAULT 0,
    protein          REAL    NOT NULL DEFAULT 0,
    carbs            REAL    NOT NULL DEFAULT 0,
    fat              REAL    NOT NULL DEFAULT 0,
    fiber            REAL    NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_meal_template_items_template ON meal_template_items(template_id, position);

CREATE TABLE IF NOT EXISTS template_logs (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    template_id TEXT NOT NULL REFERENCES meal_templates(id) ON DELETE CASCADE,
    logged_at   TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_template_logs_user ON template_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_template_logs_template ON template_logs(template_id);

CREATE TABLE IF NOT EXISTS password_credentials (
    user_id    TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    phc_hash   TEXT NOT NULL,          -- argon2id PHC string
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS sessions (
    id                  TEXT PRIMARY KEY,  -- SHA-256(hex) of the cookie token
    user_id             TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    csrf_token          TEXT NOT NULL,
    created_at          TEXT NOT NULL DEFAULT (datetime('now')),
    last_seen_at        TEXT NOT NULL,
    idle_expires_at     TEXT NOT NULL,
    absolute_expires_at TEXT NOT NULL,
    remember            INTEGER NOT NULL DEFAULT 0,
    ip                  TEXT,
    user_agent          TEXT
);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);

CREATE TABLE IF NOT EXISTS api_keys (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    hashed_key   TEXT NOT NULL UNIQUE,   -- SHA-256(hex) of the raw key
    label        TEXT,
    created_at   TEXT NOT NULL DEFAULT (datetime('now')),
    last_used_at TEXT,
    revoked_at   TEXT
);
CREATE INDEX IF NOT EXISTS idx_api_keys_user ON api_keys(user_id);

-- auth_audit_log: append-only event log. FKs use SET NULL (not CASCADE) so
-- the audit trail survives account/user deletion.
CREATE TABLE IF NOT EXISTS auth_audit_log (
    id         TEXT PRIMARY KEY,
    account_id TEXT REFERENCES accounts(id) ON DELETE SET NULL,
    user_id    TEXT REFERENCES users(id) ON DELETE SET NULL,
    event      TEXT NOT NULL,   -- e.g. login.success, login.fail, logout, register, apikey.create, apikey.revoke, lockout
    ip         TEXT,
    user_agent TEXT,
    meta       TEXT,            -- optional JSON
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS login_attempts (
    id         TEXT PRIMARY KEY,
    identifier TEXT NOT NULL,   -- lowercased email or "ip:<addr>"; global/anonymous by design
    succeeded  INTEGER NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_login_attempts_id ON login_attempts(identifier, created_at);

-- totp_secrets/recovery_codes: TOTP two-factor auth with encrypted-at-rest
-- secrets and one-time recovery codes.
CREATE TABLE IF NOT EXISTS totp_secrets (
    user_id      TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    secret       TEXT NOT NULL,  -- AES-256-GCM ciphertext (base64), NOT raw base32
    confirmed_at TEXT            -- NULL until enrollment is verified
);

CREATE TABLE IF NOT EXISTS recovery_codes (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash  TEXT NOT NULL,   -- SHA-256 hex of the code
    used_at    TEXT,            -- NULL → available; set on first use, row retained
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_recovery_codes_user ON recovery_codes(user_id);

-- auth_challenges: short-lived, single-use, SELECT+DELETE-consumed ceremony
-- tokens. Merges what used to be two near-identical tables (mfa_challenges,
-- webauthn_sessions) — both were opaque-ID-PK, optional-user_id, ~5-minute
-- TTL, differing only in payload shape. `kind` distinguishes them;
-- payload_json holds whatever that kind needs (mfa: {"remember":bool},
-- webauthn_ceremony: the go-webauthn SessionData verbatim).
CREATE TABLE IF NOT EXISTS auth_challenges (
    id           TEXT PRIMARY KEY,
    user_id      TEXT REFERENCES users(id) ON DELETE CASCADE,  -- NULL for discoverable WebAuthn login
    kind         TEXT NOT NULL CHECK(kind IN ('mfa', 'webauthn_ceremony')),
    payload_json TEXT NOT NULL DEFAULT '{}',
    expires_at   TEXT NOT NULL,
    created_at   TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_auth_challenges_expires ON auth_challenges(expires_at);

-- auth_verification_codes: single-use, hashed, time-bound secrets sent to a
-- user out-of-band. Merges auth_email_tokens + auth_magic_codes +
-- mfa_email_codes — all three shared this shape; `purpose` distinguishes
-- them and UNIQUE(user_id, purpose) gives "one active code per purpose"
-- (requesting a new one invalidates the outstanding one, same as the old
-- magic-code/mfa-email behavior).
CREATE TABLE IF NOT EXISTS auth_verification_codes (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose    TEXT NOT NULL CHECK(purpose IN ('email_verify', 'password_reset', 'magic_signin', 'mfa_email')),
    code_hash  TEXT NOT NULL,   -- SHA-256 hex
    attempts   INTEGER NOT NULL DEFAULT 0,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE (user_id, purpose)
);
CREATE INDEX IF NOT EXISTS idx_auth_verification_codes_expires ON auth_verification_codes(expires_at);

-- oidc_identities/oidc_states: OIDC client login — linked identities by
-- provider+subject, and short-lived state tokens for the OAuth
-- authorization code flow (PKCE+nonce).
CREATE TABLE IF NOT EXISTS oidc_identities (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider   TEXT NOT NULL,   -- e.g. "google", "authentik"
    subject    TEXT NOT NULL,   -- provider's stable subject claim
    email      TEXT,            -- email from the ID token (may differ from users.email)
    linked_at  TEXT NOT NULL DEFAULT (datetime('now')),
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(provider, subject)
);
CREATE INDEX IF NOT EXISTS idx_oidc_identities_user ON oidc_identities(user_id);

CREATE TABLE IF NOT EXISTS oidc_states (
    id            TEXT PRIMARY KEY,  -- SHA-256 hex of the random state param
    nonce         TEXT NOT NULL,
    pkce_verifier TEXT NOT NULL,     -- PKCE code verifier (plaintext, for the token exchange)
    link_user_id  TEXT REFERENCES users(id) ON DELETE CASCADE,  -- non-empty when this is a link (not sign-in) flow
    next          TEXT,              -- post-login redirect path
    expires_at    TEXT NOT NULL,
    created_at    TEXT NOT NULL DEFAULT (datetime('now'))
);

-- webauthn_credentials: passkeys. Long-lived, permanent until the user
-- deletes them.
CREATE TABLE IF NOT EXISTS webauthn_credentials (
    id              TEXT PRIMARY KEY,         -- base64url credential ID
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label           TEXT,
    credential_json TEXT NOT NULL,            -- full go-webauthn Credential, JSON
    sign_count      INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    last_used_at    TEXT                      -- NULL until first auth
);
CREATE INDEX IF NOT EXISTS idx_webauthn_credentials_user ON webauthn_credentials(user_id);

-- linking_codes: one-time codes for bot-to-dashboard account linking. Links
-- a chat account (Telegram/Discord/Matrix) to a dashboard user. Codes
-- expire after 10 minutes and are single-use — consumption is a soft-delete
-- (used_at set, row kept) because the linking SSE endpoint polls the row
-- after consumption to detect the used-transition.
CREATE TABLE IF NOT EXISTS linking_codes (
    code       TEXT PRIMARY KEY,   -- 6-char random alphanumeric
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform   TEXT NOT NULL,      -- "telegram", "discord", "matrix"
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    expires_at TEXT NOT NULL,      -- created_at + 10 minutes
    used_at    TEXT                -- NULL = unused, set on use
);

-- user_provider_keys: per-user, per-provider encrypted API keys for BYOK
-- integrations (AI providers, Hevy workout import, etc). Merges what used to
-- be two structurally-identical tables (user_ai_keys, user_hevy_keys).
CREATE TABLE IF NOT EXISTS user_provider_keys (
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider   TEXT NOT NULL,       -- "anthropic" | "openai" | "hevy"
    enc_key    TEXT NOT NULL,       -- AES-256-GCM ciphertext, base64
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (user_id, provider)
);

-- backup_config: per-user scheduled backup/export settings, local disk or S3.
CREATE TABLE IF NOT EXISTS backup_config (
    user_id            TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    enabled            INTEGER NOT NULL DEFAULT 0,
    destination        TEXT NOT NULL DEFAULT 'local', -- "local" | "s3"
    local_subdir       TEXT,
    s3_bucket          TEXT,
    s3_prefix          TEXT,
    s3_region          TEXT,
    s3_endpoint        TEXT,
    interval_hrs       INTEGER NOT NULL DEFAULT 24,
    last_run_at        TEXT NOT NULL DEFAULT '',
    last_meals_count   INTEGER NOT NULL DEFAULT 0,
    last_rollups_count INTEGER NOT NULL DEFAULT 0
);

-- chat_sessions/chat_messages: AI chat assistant conversation history.
-- Distinct from chat_routes below (proactive nudge delivery routing).
CREATE TABLE IF NOT EXISTS chat_sessions (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title      TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_chat_sessions_user ON chat_sessions(user_id);

CREATE TABLE IF NOT EXISTS chat_messages (
    id         TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    role       TEXT NOT NULL,
    content    TEXT NOT NULL,
    tool_name  TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_chat_messages_session ON chat_messages(session_id);

CREATE TABLE IF NOT EXISTS user_assistant_settings (
    user_id             TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    custom_instructions TEXT NOT NULL DEFAULT '',
    updated_at          TEXT NOT NULL DEFAULT (datetime('now'))
);

-- chat_routes: reverse routing from user_id to the chat metadata needed to
-- deliver a message proactively (chat id, channel id, room id — whichever
-- the active MessagingAdapter needs). Refreshed on every inbound message so
-- the scheduler can reach a user without waiting for them to message first.
CREATE TABLE IF NOT EXISTS chat_routes (
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel    TEXT NOT NULL,
    meta_json  TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (user_id, channel)
);

-- nudge_log: dedupe table so each scheduler rule fires at most once per user
-- per local day (or per ISO week, using local_date as an unconstrained key
-- for weekly rules). Composite PK handles idempotency naturally.
CREATE TABLE IF NOT EXISTS nudge_log (
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    local_date TEXT NOT NULL,
    rule_id    TEXT NOT NULL,
    sent_at    TEXT NOT NULL,
    PRIMARY KEY (user_id, local_date, rule_id)
);

-- nudge_rule_config: per-user overrides for macro/health/digest/weekly-budget
-- nudge rules. One row per (user, rule_id); params_json holds the
-- rule-specific shape (deliberate EAV over a closed ~10-rule-kind set so one
-- table covers every rule kind without a sparse wide schema).
CREATE TABLE IF NOT EXISTS nudge_rule_config (
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rule_id     TEXT NOT NULL,
    enabled     INTEGER NOT NULL DEFAULT 1,
    params_json TEXT NOT NULL DEFAULT '{}',
    PRIMARY KEY (user_id, rule_id)
);

-- sent_nudges: delivery/undo log. snapshot_kcal..snapshot_fiber are flat
-- columns (was one snapshot_json blob) — matches how daily_rollups already
-- stores the same Macros shape.
CREATE TABLE IF NOT EXISTS sent_nudges (
    id               TEXT PRIMARY KEY,
    user_id          TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rule_id          TEXT NOT NULL,
    sent_at          TEXT NOT NULL,
    body             TEXT NOT NULL,
    snapshot_kcal    REAL NOT NULL DEFAULT 0,
    snapshot_protein REAL NOT NULL DEFAULT 0,
    snapshot_carbs   REAL NOT NULL DEFAULT 0,
    snapshot_fat     REAL NOT NULL DEFAULT 0,
    snapshot_fiber   REAL NOT NULL DEFAULT 0,
    status           TEXT NOT NULL DEFAULT 'sent' CHECK(status IN ('sent', 'dismissed')),
    resolved_at      TEXT
);

-- pending_state: durable pending-meal store. One row per user; entire
-- PendingMeal JSON-marshalled into payload. created_at is Unix epoch
-- seconds, duplicated out of the payload so expiry can be evaluated in SQL
-- or Go without unmarshalling. Counterpart: internal/pendingstore.
CREATE TABLE IF NOT EXISTS pending_state (
    user_id    TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    created_at INTEGER NOT NULL,
    payload    BLOB NOT NULL
);

-- user_channels: maps messaging platform (channel + channel_user_id) to an
-- internal user_id so the pipeline can resolve inbound messages to users.
CREATE TABLE IF NOT EXISTS user_channels (
    channel         TEXT NOT NULL,
    channel_user_id TEXT NOT NULL,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (channel, channel_user_id)
);
