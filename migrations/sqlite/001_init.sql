-- 001_init: DietDaemon schema (SQLite).

CREATE TABLE IF NOT EXISTS accounts (
    id         TEXT PRIMARY KEY,
    created_at TEXT NOT NULL
);

-- users: tenant-scoped (account_id) with all login identity columns folded in
-- (email/password auth, WebAuthn, locale). timezone drives local-day nudge
-- scheduling.
CREATE TABLE IF NOT EXISTS users (
    id                TEXT PRIMARY KEY,
    account_id        TEXT REFERENCES accounts(id),
    timezone          TEXT NOT NULL DEFAULT '',
    email             TEXT,
    email_verified_at TEXT,
    status            TEXT NOT NULL DEFAULT 'active',
    display_name      TEXT,
    webauthn_handle   TEXT,  -- base64, NULL until first passkey
    locale            TEXT,  -- BCP-47 tag (e.g. "en", "pt-BR"); NULL = auto-detect
    created_at        TEXT NOT NULL
);
-- App lowercases email before write/compare; NULL emails are exempt (bot-only users).
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_account_email ON users(account_id, email) WHERE email IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_webauthn_handle ON users(webauthn_handle);

CREATE TABLE IF NOT EXISTS meals (
    id          TEXT PRIMARY KEY,
    user_id     TEXT    NOT NULL REFERENCES users(id),
    at_utc      TEXT    NOT NULL,
    raw_text    TEXT    NOT NULL,
    confidence  REAL    NOT NULL DEFAULT 0,
    parser_tier INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT    NOT NULL
);
-- Covers GetMealsInRange (user_id + date(at_utc) range scan).
CREATE INDEX IF NOT EXISTS idx_meals_user_at ON meals(user_id, at_utc);
CREATE INDEX IF NOT EXISTS idx_meals_user_date_at ON meals(user_id, date(at_utc));
-- Covers RecentMeals (user_id filter, created_at DESC LIMIT n).
CREATE INDEX IF NOT EXISTS idx_meals_user_created ON meals(user_id, created_at);

CREATE TABLE IF NOT EXISTS resolved_items (
    id               TEXT PRIMARY KEY,
    meal_id          TEXT    NOT NULL REFERENCES meals(id),
    raw_phrase       TEXT    NOT NULL,
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
-- Covers every "items for this meal" lookup (WHERE meal_id = ?).
CREATE INDEX IF NOT EXISTS idx_resolved_items_meal ON resolved_items(meal_id);

-- food_library: per-user food catalog, metadata columns (category/brand/
-- barcode/image/serving) folded in from old migration 006.
CREATE TABLE IF NOT EXISTS food_library (
    food_id       TEXT    NOT NULL,
    user_id       TEXT    NOT NULL,
    name          TEXT    NOT NULL,
    source        TEXT    NOT NULL DEFAULT '',
    kcal_100g     REAL    NOT NULL DEFAULT 0,
    protein_100g  REAL    NOT NULL DEFAULT 0,
    carbs_100g    REAL    NOT NULL DEFAULT 0,
    fat_100g      REAL    NOT NULL DEFAULT 0,
    fiber_100g    REAL    NOT NULL DEFAULT 0,
    query_count   INTEGER NOT NULL DEFAULT 0,
    last_used     TEXT    NOT NULL DEFAULT '',
    category      TEXT    NOT NULL DEFAULT '',
    brand         TEXT    NOT NULL DEFAULT '',
    barcode       TEXT    NOT NULL DEFAULT '',
    image_url     TEXT    NOT NULL DEFAULT '',
    serving_size  REAL    NOT NULL DEFAULT 0,
    serving_unit  TEXT    NOT NULL DEFAULT '',
    PRIMARY KEY (user_id, food_id)
);

CREATE TABLE IF NOT EXISTS food_aliases (
    user_id          TEXT NOT NULL,
    alias_normalized TEXT NOT NULL,
    food_id          TEXT NOT NULL,
    PRIMARY KEY (user_id, alias_normalized),
    FOREIGN KEY (user_id, food_id) REFERENCES food_library(user_id, food_id)
);

-- food_vectors: per-user food embedding vectors for nearest-neighbor matching
-- (Tier-1/Tier-2). vec is a little-endian float32 BLOB; dim records the
-- vector length for sanity checks on load.
CREATE TABLE IF NOT EXISTS food_vectors (
    user_id TEXT NOT NULL,
    food_id TEXT NOT NULL,
    dim     INTEGER NOT NULL,
    vec     BLOB NOT NULL,
    PRIMARY KEY (user_id, food_id),
    FOREIGN KEY (user_id, food_id) REFERENCES food_library(user_id, food_id)
);

-- food_search: FTS5 full-text index on food_library names and food_aliases.
-- No population INSERT here (fresh schema, food_library starts empty) —
-- triggers keep it in sync with all future writes.
CREATE VIRTUAL TABLE IF NOT EXISTS food_search USING fts5(
    food_id,
    user_id,
    name,
    alias
);

CREATE TRIGGER IF NOT EXISTS food_library_fts_insert AFTER INSERT ON food_library BEGIN
    INSERT INTO food_search(food_id, user_id, name, alias)
    VALUES (NEW.food_id, NEW.user_id, NEW.name, '');
END;

CREATE TRIGGER IF NOT EXISTS food_library_fts_update AFTER UPDATE ON food_library BEGIN
    DELETE FROM food_search WHERE food_id = OLD.food_id AND user_id = OLD.user_id;
    INSERT INTO food_search(food_id, user_id, name, alias)
    VALUES (NEW.food_id, NEW.user_id, NEW.name, '');
    INSERT INTO food_search(food_id, user_id, name, alias)
    SELECT fa.food_id, fa.user_id, NEW.name, fa.alias_normalized
    FROM food_aliases fa
    WHERE fa.user_id = NEW.user_id AND fa.food_id = NEW.food_id;
END;

CREATE TRIGGER IF NOT EXISTS food_library_fts_delete AFTER DELETE ON food_library BEGIN
    DELETE FROM food_search WHERE food_id = OLD.food_id AND user_id = OLD.user_id;
END;

CREATE TRIGGER IF NOT EXISTS food_aliases_fts_insert AFTER INSERT ON food_aliases BEGIN
    INSERT INTO food_search(food_id, user_id, name, alias)
    SELECT fl.food_id, fl.user_id, fl.name, NEW.alias_normalized
    FROM food_library fl
    WHERE fl.user_id = NEW.user_id AND fl.food_id = NEW.food_id;
END;

CREATE TRIGGER IF NOT EXISTS food_aliases_fts_delete AFTER DELETE ON food_aliases BEGIN
    DELETE FROM food_search
    WHERE food_id = OLD.food_id AND user_id = OLD.user_id AND alias = OLD.alias_normalized;
END;

CREATE TABLE IF NOT EXISTS daily_targets (
    user_id TEXT PRIMARY KEY,
    kcal    REAL NOT NULL DEFAULT 0,
    protein REAL NOT NULL DEFAULT 0,
    carbs   REAL NOT NULL DEFAULT 0,
    fat     REAL NOT NULL DEFAULT 0,
    fiber   REAL NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS daily_rollups (
    user_id          TEXT NOT NULL,
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

-- nudge_log: dedupe table so each scheduler rule fires at most once per user
-- per local day (or per ISO week, using local_date as an unconstrained key
-- for weekly rules). Composite PK handles idempotency naturally.
CREATE TABLE IF NOT EXISTS nudge_log (
    user_id    TEXT NOT NULL,
    local_date TEXT NOT NULL,
    rule_id    TEXT NOT NULL,
    sent_at    TEXT NOT NULL,
    PRIMARY KEY (user_id, local_date, rule_id)
);

-- pending_state: BLOB-backed durable pending meal store. One row per user;
-- entire PendingMeal JSON-marshalled into payload. created_at is Unix epoch
-- seconds, duplicated out of the payload so expiry can be evaluated in SQL
-- or Go without unmarshalling. Counterpart: internal/pendingstore.
CREATE TABLE IF NOT EXISTS pending_state (
    user_id    TEXT PRIMARY KEY,
    created_at INTEGER NOT NULL,
    payload    BLOB NOT NULL
);

-- user_channels: maps messaging platform (channel + channel_user_id) to an
-- internal user_id so the pipeline can resolve inbound messages to users.
CREATE TABLE IF NOT EXISTS user_channels (
    channel         TEXT NOT NULL,
    channel_user_id TEXT NOT NULL,
    user_id         TEXT NOT NULL REFERENCES users(id),
    PRIMARY KEY (channel, channel_user_id)
);

CREATE TABLE IF NOT EXISTS meal_templates (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id),
    name       TEXT NOT NULL,
    items_json TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL,
    last_used  TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS template_logs (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id),
    template_id TEXT NOT NULL REFERENCES meal_templates(id),
    logged_at   TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS weight_log (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id),
    date       TEXT NOT NULL,
    weight_kg  REAL NOT NULL,
    note       TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_weight_log_user_date ON weight_log(user_id, date);

CREATE TABLE IF NOT EXISTS measurement_log (
    id             TEXT PRIMARY KEY,
    user_id        TEXT NOT NULL REFERENCES users(id),
    date           TEXT NOT NULL,
    waist_cm       REAL NOT NULL DEFAULT 0,
    hips_cm        REAL NOT NULL DEFAULT 0,
    chest_cm       REAL NOT NULL DEFAULT 0,
    left_arm_cm    REAL NOT NULL DEFAULT 0,
    right_arm_cm   REAL NOT NULL DEFAULT 0,
    left_thigh_cm  REAL NOT NULL DEFAULT 0,
    right_thigh_cm REAL NOT NULL DEFAULT 0,
    note           TEXT NOT NULL DEFAULT '',
    created_at     TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_measurement_log_user_date ON measurement_log(user_id, date);

CREATE TABLE IF NOT EXISTS progress_photos (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id),
    date       TEXT NOT NULL,
    view       TEXT NOT NULL CHECK(view IN ('front', 'side', 'back')),
    mime_type  TEXT NOT NULL,
    data       BLOB NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_progress_photos_user_date ON progress_photos(user_id, date);

-- user_profiles: per-user profile for TDEE calculation and goal tracking.
CREATE TABLE IF NOT EXISTS user_profiles (
    user_id          TEXT PRIMARY KEY REFERENCES users(id),
    height_cm        REAL NOT NULL DEFAULT 0,
    birth_date       TEXT NOT NULL DEFAULT '',
    gender           TEXT NOT NULL DEFAULT '',
    activity_level   TEXT NOT NULL DEFAULT '',
    goal             TEXT NOT NULL DEFAULT '',
    target_weight_kg REAL NOT NULL DEFAULT 0,
    weekly_rate      REAL NOT NULL DEFAULT 0,
    onboarded        INTEGER NOT NULL DEFAULT 0,
    created_at       TEXT NOT NULL,
    updated_at       TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS password_credentials (
    user_id    TEXT PRIMARY KEY REFERENCES users(id),
    phc_hash   TEXT NOT NULL,          -- argon2id PHC string
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
    id                  TEXT PRIMARY KEY,  -- SHA-256(hex) of the cookie token
    user_id             TEXT NOT NULL REFERENCES users(id),
    csrf_token          TEXT NOT NULL,
    created_at          TEXT NOT NULL,
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
    user_id      TEXT NOT NULL REFERENCES users(id),
    hashed_key   TEXT NOT NULL UNIQUE,   -- SHA-256(hex) of the raw key
    label        TEXT NOT NULL,
    created_at   TEXT NOT NULL,
    last_used_at TEXT,
    revoked_at   TEXT
);
CREATE INDEX IF NOT EXISTS idx_api_keys_user ON api_keys(user_id);

CREATE TABLE IF NOT EXISTS auth_audit_log (
    id         TEXT PRIMARY KEY,
    account_id TEXT,
    user_id    TEXT,
    event      TEXT NOT NULL,   -- e.g. login.success, login.fail, logout, register, apikey.create, apikey.revoke, lockout
    ip         TEXT,
    user_agent TEXT,
    meta       TEXT,            -- optional JSON
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_audit_account ON auth_audit_log(account_id, created_at);

CREATE TABLE IF NOT EXISTS login_attempts (
    id          TEXT PRIMARY KEY,
    identifier  TEXT NOT NULL,   -- lowercased email (per-account) or "ip:<addr>"
    succeeded   INTEGER NOT NULL,
    created_at  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_login_attempts_id ON login_attempts(identifier, created_at);

-- totp_secrets/recovery_codes/mfa_challenges: TOTP two-factor auth with
-- encrypted-at-rest secrets, recovery codes, and MFA challenge tokens for
-- step-up authentication.
CREATE TABLE IF NOT EXISTS totp_secrets (
    user_id      TEXT PRIMARY KEY REFERENCES users(id),
    secret       TEXT NOT NULL,  -- AES-256-GCM ciphertext (base64), NOT raw base32
    confirmed_at TEXT            -- NULL until enrollment is verified
);

CREATE TABLE IF NOT EXISTS recovery_codes (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id),
    code_hash  TEXT NOT NULL,   -- SHA-256 hex of the code
    used_at    TEXT,            -- NULL → available; set on first use
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_recovery_codes_user ON recovery_codes(user_id);

CREATE TABLE IF NOT EXISTS mfa_challenges (
    id         TEXT PRIMARY KEY,  -- SHA-256 hex of the challenge token
    user_id    TEXT NOT NULL,
    remember   INTEGER NOT NULL DEFAULT 0,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_mfa_challenges_user ON mfa_challenges(user_id);

-- oidc_identities/oidc_states: OIDC client login — linked identities by
-- provider+subject, and short-lived state tokens for the OAuth
-- authorization code flow (PKCE+nonce).
CREATE TABLE IF NOT EXISTS oidc_identities (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id),
    provider   TEXT NOT NULL,   -- e.g. "google", "authentik"
    subject    TEXT NOT NULL,   -- provider's stable subject claim
    email      TEXT,            -- email from the ID token (may differ from users.email)
    linked_at  TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE(provider, subject)
);
CREATE INDEX IF NOT EXISTS idx_oidc_identities_user ON oidc_identities(user_id);

CREATE TABLE IF NOT EXISTS oidc_states (
    id            TEXT PRIMARY KEY,  -- SHA-256 hex of the random state param
    nonce         TEXT NOT NULL,
    pkce_verifier TEXT NOT NULL,     -- PKCE code verifier (plaintext, for the token exchange)
    link_user_id  TEXT,              -- non-empty when this is a link (not sign-in) flow
    next          TEXT,              -- post-login redirect path
    expires_at    TEXT NOT NULL,
    created_at    TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_oidc_states_expires ON oidc_states(expires_at);

-- auth_email_tokens: single-use tokens for email verification and password
-- reset. Tokens are keyed by SHA-256 hex of the random token sent to the
-- user; only the hash is persisted so a DB leak cannot replay the token.
CREATE TABLE IF NOT EXISTS auth_email_tokens (
    id         TEXT PRIMARY KEY,  -- SHA-256 hex of the random token
    user_id    TEXT NOT NULL REFERENCES users(id),
    purpose    TEXT NOT NULL,     -- 'verify' | 'reset'
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_auth_email_tokens_expires ON auth_email_tokens(expires_at);

-- auth_magic_codes: magic code table for passwordless email sign-in. Codes
-- are 6-digit low-entropy secrets, so there's an attempt cap and the
-- code_hash is always stored hashed (SHA-256). One active code per user
-- (user_id is the PK); resend overwrites via upsert.
CREATE TABLE IF NOT EXISTS auth_magic_codes (
    user_id    TEXT PRIMARY KEY REFERENCES users(id),
    code_hash  TEXT NOT NULL,   -- SHA-256 hex of the 6-digit code
    expires_at TEXT NOT NULL,
    attempts   INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_auth_magic_codes_expires ON auth_magic_codes(expires_at);

-- webauthn_credentials/webauthn_sessions/mfa_email_codes: passkeys (WebAuthn
-- credentials), ceremony session storage, and email-OTP fallback codes for
-- MFA step-up.
CREATE TABLE IF NOT EXISTS webauthn_credentials (
    id              TEXT PRIMARY KEY,         -- base64url credential ID
    user_id         TEXT NOT NULL REFERENCES users(id),
    label           TEXT NOT NULL,
    credential_json TEXT NOT NULL,            -- full go-webauthn Credential, JSON
    sign_count      INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT NOT NULL,
    last_used_at    TEXT                      -- NULL until first auth
);
CREATE INDEX IF NOT EXISTS idx_webauthn_credentials_user ON webauthn_credentials(user_id);

CREATE TABLE IF NOT EXISTS webauthn_sessions (
    id           TEXT PRIMARY KEY,            -- opaque ceremony id (random), in dd_webauthn cookie
    user_id      TEXT,                        -- NULL for discoverable login (unknown until finish)
    session_data TEXT NOT NULL,               -- go-webauthn SessionData, JSON
    expires_at   TEXT NOT NULL,
    created_at   TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_webauthn_sessions_expires ON webauthn_sessions(expires_at);

CREATE TABLE IF NOT EXISTS mfa_email_codes (
    challenge_id TEXT PRIMARY KEY,            -- = mfa_challenges.id this code is bound to
    code_hash    TEXT NOT NULL,               -- sha256 hex of the 6-digit code
    expires_at   TEXT NOT NULL,
    attempts     INTEGER NOT NULL DEFAULT 0,
    created_at   TEXT NOT NULL
);

-- linking_codes: one-time codes for bot-to-dashboard account linking. Links
-- a chat account (Telegram/Discord/Matrix) to a dashboard user. Codes
-- expire after 10 minutes and are single-use.
CREATE TABLE IF NOT EXISTS linking_codes (
    code        TEXT PRIMARY KEY,   -- 6-char random alphanumeric
    user_id     TEXT NOT NULL,      -- dashboard user ID (accounts.id)
    platform    TEXT NOT NULL,      -- "telegram", "discord", "matrix"
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    expires_at  TEXT NOT NULL,      -- created_at + 10 minutes
    used_at     TEXT                -- NULL = unused, set on use
);

-- fasts: intermittent-fasting windows. A user has at most one active fast
-- (end_at IS NULL) at a time.
CREATE TABLE IF NOT EXISTS fasts (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL REFERENCES users(id),
    start_at     TEXT NOT NULL,
    end_at       TEXT,                       -- NULL = still fasting
    target_hours REAL NOT NULL DEFAULT 16,
    completed    INTEGER NOT NULL DEFAULT 0, -- 1 once target reached at end
    created_at   TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_fasts_user_start ON fasts(user_id, start_at);
-- Fast lookup of the single in-progress fast per user.
CREATE INDEX IF NOT EXISTS idx_fasts_active ON fasts(user_id) WHERE end_at IS NULL;

-- water_logs: water consumption tracking. Each row records one water entry
-- (e.g. a glass or bottle).
CREATE TABLE IF NOT EXISTS water_logs (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id),
    amount_ml  INTEGER NOT NULL,
    logged_at  TEXT NOT NULL,
    note       TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_water_logs_user_date ON water_logs(user_id, logged_at);

-- workouts/workout_exercises: exercise session tracking with individual exercises.
CREATE TABLE IF NOT EXISTS workouts (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id),
    name            TEXT NOT NULL,
    duration_min    INTEGER NOT NULL,
    intensity       TEXT NOT NULL DEFAULT 'moderate',
    calories_burned INTEGER,
    note            TEXT,
    logged_at       TEXT NOT NULL,
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_workouts_user_date ON workouts(user_id, logged_at);

CREATE TABLE IF NOT EXISTS workout_exercises (
    id         TEXT PRIMARY KEY,
    workout_id TEXT NOT NULL REFERENCES workouts(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    sets       INTEGER,
    reps       INTEGER,
    weight_kg  REAL,
    note       TEXT
);
CREATE INDEX IF NOT EXISTS idx_workout_exercises_workout ON workout_exercises(workout_id);

-- sleep_logs: sleep session tracking. A sleep log has wake_at NULL while the
-- session is in progress.
CREATE TABLE IF NOT EXISTS sleep_logs (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id),
    sleep_at   TEXT NOT NULL,
    wake_at    TEXT,
    quality    TEXT NOT NULL DEFAULT 'ok',
    note       TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_sleep_logs_user_date ON sleep_logs(user_id, sleep_at);

-- pending_aliases: embedding-matched aliases awaiting user confirmation.
-- Replaces the silent write-back into food_aliases for near-miss matches:
-- the match is parked here until the user confirms or rejects it.
CREATE TABLE IF NOT EXISTS pending_aliases (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id),
    phrase      TEXT NOT NULL,      -- raw phrase that triggered the match
    food_id     TEXT NOT NULL,      -- candidate food_library.food_id
    match_score REAL NOT NULL,      -- embedding similarity score
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_pending_aliases_user ON pending_aliases(user_id);

-- source_precedence: per-user override of the nutrition source resolution
-- order (default order comes from the NUTRITION_SOURCE env var).
CREATE TABLE IF NOT EXISTS source_precedence (
    user_id TEXT NOT NULL REFERENCES users(id),
    source  TEXT NOT NULL,      -- source name, e.g. "openfoodfacts"
    rank    INTEGER NOT NULL,   -- 0 = tried first
    PRIMARY KEY (user_id, source)
);

-- nudge_rule_config: per-user overrides for macro/health/digest/weekly-budget
-- nudge rules. One row per (user, rule_id); params_json holds the
-- rule-specific shape so one table covers every rule kind without a sparse
-- wide schema.
CREATE TABLE IF NOT EXISTS nudge_rule_config (
    user_id     TEXT NOT NULL REFERENCES users(id),
    rule_id     TEXT NOT NULL,
    enabled     INTEGER NOT NULL DEFAULT 1,
    params_json TEXT NOT NULL DEFAULT '{}',
    PRIMARY KEY (user_id, rule_id)
);

-- backup_config: per-user scheduled backup/export settings, local disk or S3.
CREATE TABLE IF NOT EXISTS backup_config (
    user_id       TEXT PRIMARY KEY REFERENCES users(id),
    enabled       INTEGER NOT NULL DEFAULT 0,
    destination   TEXT NOT NULL DEFAULT 'local', -- "local" | "s3"
    local_subdir  TEXT NOT NULL DEFAULT '',
    s3_bucket     TEXT NOT NULL DEFAULT '',
    s3_prefix     TEXT NOT NULL DEFAULT '',
    s3_region     TEXT NOT NULL DEFAULT '',
    s3_endpoint   TEXT NOT NULL DEFAULT '',
    interval_hrs  INTEGER NOT NULL DEFAULT 24,
    last_run_at   TEXT NOT NULL DEFAULT ''
);

-- chat_routes: reverse routing from user_id to the chat metadata needed to
-- deliver a message proactively (chat id, channel id, room id — whichever
-- the active MessagingAdapter needs). Refreshed on every inbound message so
-- the scheduler can reach a user without waiting for them to message first.
CREATE TABLE IF NOT EXISTS chat_routes (
    user_id    TEXT NOT NULL REFERENCES users(id),
    channel    TEXT NOT NULL,
    meta_json  TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (user_id, channel)
);

CREATE TABLE IF NOT EXISTS sent_nudges (
    id            TEXT PRIMARY KEY,
    user_id       TEXT NOT NULL REFERENCES users(id),
    rule_id       TEXT NOT NULL,
    sent_at       TEXT NOT NULL,
    body          TEXT NOT NULL,
    snapshot_json TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'sent',
    resolved_at   TEXT
);

-- Missing indexes on foreign keys to avoid full table scans during cascading checks/deletes and joins
CREATE INDEX IF NOT EXISTS idx_users_account ON users(account_id);
CREATE INDEX IF NOT EXISTS idx_user_channels_user ON user_channels(user_id);
CREATE INDEX IF NOT EXISTS idx_meal_templates_user ON meal_templates(user_id);
CREATE INDEX IF NOT EXISTS idx_template_logs_user ON template_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_template_logs_template ON template_logs(template_id);
CREATE INDEX IF NOT EXISTS idx_auth_email_tokens_user ON auth_email_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_sent_nudges_user ON sent_nudges(user_id);
CREATE INDEX IF NOT EXISTS idx_food_aliases_food ON food_aliases(user_id, food_id);

-- Composite indexes to optimize sorting and filtering in food queries
CREATE INDEX IF NOT EXISTS idx_food_library_user_last_used ON food_library(user_id, last_used DESC);
CREATE INDEX IF NOT EXISTS idx_food_library_frequent ON food_library(user_id, query_count DESC, last_used DESC);

