-- 013_oidc: OIDC client login — linked identities by provider+subject, and
-- short-lived state tokens for the OAuth authorization code flow (PKCE+nonce).

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
