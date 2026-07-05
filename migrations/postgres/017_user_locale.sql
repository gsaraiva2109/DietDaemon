-- 017_user_locale: i18n language preference per user.
-- BCP-47 locale tag (e.g. "en", "pt-BR"); NULL = auto-detect from channel.

ALTER TABLE users ADD COLUMN locale TEXT;
