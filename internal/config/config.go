// Package config loads and validates DietDaemon's runtime configuration from
// environment variables (see .env.example for the full list). Validation runs
// at boot and fails fast, reporting every problem at once so a misconfigured
// self-host is obvious immediately rather than crashing deep in a pipeline.
package config

import (
	"bufio"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/netip"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/auth"
)

// OIDCProviderConfig is the parsed static configuration for one OIDC provider.
type OIDCProviderConfig struct {
	ID           string
	Name         string
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	// TrustEmail treats a non-empty email from this provider as verified even
	// when the provider omits/!email_verified. For self-hosted IdPs the operator
	// controls (e.g. Authentik), the email is authoritative. Default false.
	TrustEmail bool
}

// Config is the fully parsed, validated configuration. Location is derived from
// DefaultTimezone so the rest of the app never re-parses the tz string.
type Config struct {
	MessagingAdapter    string
	TelegramBotToken    string
	DiscordBotToken     string
	MatrixHomeserverURL string
	MatrixUserID        string
	MatrixToken         string

	ParserTier types.ParserTier

	NutritionSources []string
	USDAFDCAPIKey    string
	TacoDataPath     string

	// --- Bulk import (opt-in; disabled by default so there's no surprise startup traffic) ---
	FoodImportEnabled  bool          // FOOD_IMPORT_ENABLED, default false
	FoodImportSources  []string      // FOOD_IMPORT_SOURCES, comma-separated, default "" (empty = disabled)
	FoodImportInterval time.Duration // FOOD_IMPORT_INTERVAL, default 24h

	USDABulkFile      string   // USDA_BULK_FILE — if set, adapter uses file mode instead of live API
	USDABulkDataTypes []string // USDA_BULK_DATA_TYPES, comma-separated, default "Foundation,SR Legacy"
	USDABulkMaxRows   int      // USDA_BULK_MAX_ROWS, default 0 (unlimited)

	OFFBulkFile          string // OFF_BULK_FILE — if set, adapter uses file mode instead of live API
	OFFBulkMinPopularity int    // OFF_BULK_MIN_POPULARITY, default 0
	OFFBulkMaxRows       int    // OFF_BULK_MAX_ROWS, default 0

	TacoBulkMaxRows int // TACO_BULK_MAX_ROWS, default 0 (safety valve; TACO's dataset is already small)

	EmbedAdapter      string
	CompletionAdapter string
	OllamaURL         string
	EmbedModel        string
	LLMModel          string
	ModelTimeout      time.Duration
	OllamaAutoPull    bool

	AnthropicAPIKey string
	AnthropicModel  string

	OpenAIBaseURL string
	OpenAIAPIKey  string
	OpenAIModel   string

	EmbedMatchThreshold     float64
	AliasWriteBackThreshold float64

	Notifier  string
	NtfyURL   string
	NtfyTopic string
	NtfyToken string

	GotifyURL   string
	GotifyToken string

	DefaultTimezone string
	Location        *time.Location

	DBPath string

	EnableNotifications bool
	EnableDashboard     bool
	EnableSTT           bool
	WhisperURL          string

	// --- Scheduled backup/export ---
	BackupLocalDir      string        // base dir for the "local" destination; empty disables it
	BackupCheckInterval time.Duration // how often the backup runner checks for due users

	MultiUser    bool
	APIAuthToken string

	// --- Auth ---
	DBDriver           string
	DatabaseURL        string
	RegistrationMode   string
	SessionIdleTTL     time.Duration
	SessionAbsoluteTTL time.Duration
	SessionRememberTTL time.Duration
	CookieSecure       bool
	CookieDomain       string

	// TrustedProxies lists CIDRs (or bare IPs, treated as /32 or /128) whose
	// X-Forwarded-For / X-Real-IP headers are honored when resolving the
	// client IP (used for rate limiting, lockout, and audit logs). Requests
	// arriving from any other peer have those headers ignored — otherwise
	// any client could spoof them to dodge IP-based lockout entirely.
	// Defaults to loopback only; set this when DietDaemon sits behind a
	// reverse proxy so the real client IP is used instead of the proxy's.
	TrustedProxies []string

	// --- Auth — TOTP two-factor authentication ---
	TOTPEncKey []byte // AES-256-GCM key, 32 bytes; empty = TOTP disabled
	TOTPIssuer string // otpauth issuer label

	// --- BYOK: per-user AI API keys ---
	AIKeyEncKey []byte // AES-256-GCM key, 32 bytes; separate from TOTPEncKey for domain separation
	AIKeyMode   string // "shared" (default) or "byok"

	// --- Auth — OIDC ---
	OIDCProviders []OIDCProviderConfig
	PublicBaseURL string

	// --- Auth — Mailer / Email ---
	EmailProvider string // resend | ses | smtp | none
	EmailFrom     string // verified sender address
	ResendAPIKey  string
	SESRegion     string
	SMTPHost      string
	SMTPPort      int
	SMTPUsername  string
	SMTPPassword  string
	SMTPTLS       bool

	// --- Auth — WebAuthn / Passkeys ---
	WebAuthnRPID          string
	WebAuthnRPOrigins     []string
	WebAuthnRPDisplayName string

	LogLevel string
}

// WebAuthnConfig builds a WebAuthnConfig from the parsed configuration.
func (c *Config) WebAuthnConfig() auth.WebAuthnConfig {
	rpID := c.WebAuthnRPID
	if rpID == "" {
		rpID = hostFromBaseURL(c.PublicBaseURL)
	}
	origins := c.WebAuthnRPOrigins
	if len(origins) == 0 {
		if c.PublicBaseURL != "" {
			origins = []string{c.PublicBaseURL}
		} else {
			origins = []string{"http://localhost:8080"}
		}
	}
	displayName := c.WebAuthnRPDisplayName
	if displayName == "" {
		displayName = "DietDaemon"
	}
	return auth.WebAuthnConfig{
		RPID:          rpID,
		RPDisplayName: displayName,
		RPOrigins:     origins,
	}
}

// TrustedProxyPrefixes parses TrustedProxies into netip.Prefix values for
// runtime IP-containment checks. Entries that fail to parse are skipped —
// validate() rejects malformed entries at boot, so in practice this is
// always well-formed by the time the server is serving requests.
func (c *Config) TrustedProxyPrefixes() []netip.Prefix {
	prefixes := make([]netip.Prefix, 0, len(c.TrustedProxies))
	for _, raw := range c.TrustedProxies {
		if p, err := parseProxyEntry(raw); err == nil {
			prefixes = append(prefixes, p)
		}
	}
	return prefixes
}

// parseProxyEntry accepts either a CIDR ("10.0.0.0/8") or a bare IP
// ("127.0.0.1"), normalizing a bare IP to a single-address prefix.
func parseProxyEntry(raw string) (netip.Prefix, error) {
	if p, err := netip.ParsePrefix(raw); err == nil {
		return p, nil
	}
	addr, err := netip.ParseAddr(raw)
	if err != nil {
		return netip.Prefix{}, err
	}
	return netip.PrefixFrom(addr, addr.BitLen()), nil
}

// hostFromBaseURL extracts the host from a URL like "https://example.com".
func hostFromBaseURL(raw string) string {
	s := strings.TrimPrefix(raw, "https://")
	s = strings.TrimPrefix(s, "http://")
	if idx := strings.Index(s, ":"); idx > 0 {
		s = s[:idx]
	}
	if idx := strings.Index(s, "/"); idx > 0 {
		s = s[:idx]
	}
	return s
}

// Load reads configuration from the environment, applying values from a .env
// file in the working directory first (without overriding variables already set
// in the environment), then validates the result.
func Load() (*Config, error) {
	loadDotEnv(".env")

	c := &Config{
		MessagingAdapter:        getStr("MESSAGING_ADAPTER", "telegram"),
		TelegramBotToken:        getStr("TELEGRAM_BOT_TOKEN", ""),
		DiscordBotToken:         getStr("DISCORD_BOT_TOKEN", ""),
		MatrixHomeserverURL:     getStr("MATRIX_HOMESERVER_URL", ""),
		MatrixUserID:            getStr("MATRIX_USER_ID", ""),
		MatrixToken:             getStr("MATRIX_TOKEN", ""),
		NutritionSources:        splitCSV(getStr("NUTRITION_SOURCE", "openfoodfacts")),
		USDAFDCAPIKey:           getStr("USDA_FDC_API_KEY", ""),
		TacoDataPath:            getStr("TACO_DATA_PATH", ""),
		FoodImportEnabled:       getBool("FOOD_IMPORT_ENABLED", false),
		FoodImportSources:       splitCSV(getStr("FOOD_IMPORT_SOURCES", "")),
		FoodImportInterval:      getDuration("FOOD_IMPORT_INTERVAL", 24*time.Hour),
		USDABulkFile:            getStr("USDA_BULK_FILE", ""),
		USDABulkDataTypes:       splitCSV(getStr("USDA_BULK_DATA_TYPES", "Foundation,SR Legacy")),
		USDABulkMaxRows:         getInt("USDA_BULK_MAX_ROWS", 0),
		OFFBulkFile:             getStr("OFF_BULK_FILE", ""),
		OFFBulkMinPopularity:    getInt("OFF_BULK_MIN_POPULARITY", 0),
		OFFBulkMaxRows:          getInt("OFF_BULK_MAX_ROWS", 0),
		TacoBulkMaxRows:         getInt("TACO_BULK_MAX_ROWS", 0),
		EmbedAdapter:            getStr("EMBED_ADAPTER", "ollama"),
		CompletionAdapter:       getStr("COMPLETION_ADAPTER", ""),
		OllamaURL:               getStr("OLLAMA_URL", ""),
		EmbedModel:              getStr("EMBED_MODEL", "nomic-embed-text"),
		LLMModel:                getStr("LLM_MODEL", "llama3.1"),
		ModelTimeout:            getDuration("MODEL_TIMEOUT", 30*time.Second),
		OllamaAutoPull:          getBool("OLLAMA_AUTO_PULL", false),
		AnthropicAPIKey:         getStr("ANTHROPIC_API_KEY", ""),
		AnthropicModel:          getStr("ANTHROPIC_MODEL", "claude-haiku-4-5-20251001"),
		OpenAIBaseURL:           getStr("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		OpenAIAPIKey:            getStr("OPENAI_API_KEY", ""),
		OpenAIModel:             getStr("OPENAI_MODEL", "gpt-4o-mini"),
		EmbedMatchThreshold:     getFloat("EMBED_MATCH_THRESHOLD", 0.80),
		AliasWriteBackThreshold: getFloat("ALIAS_WRITE_BACK_THRESHOLD", 0.92),
		Notifier:                getStr("NOTIFIER", "ntfy"),
		NtfyURL:                 getStr("NTFY_URL", ""),
		NtfyTopic:               getStr("NTFY_TOPIC", ""),
		NtfyToken:               getStr("NTFY_TOKEN", ""),
		GotifyURL:               getStr("GOTIFY_URL", ""),
		GotifyToken:             getStr("GOTIFY_TOKEN", ""),
		DefaultTimezone:         getStr("DEFAULT_TIMEZONE", "UTC"),
		DBPath:                  getStr("DB_PATH", "/data/dietdaemon.db"),
		EnableNotifications:     getBool("ENABLE_NOTIFICATIONS", true),
		EnableDashboard:         getBool("ENABLE_DASHBOARD", false),
		EnableSTT:               getBool("ENABLE_STT", false),
		WhisperURL:              getStr("WHISPER_URL", "http://whisper:8080"),
		BackupLocalDir:          getStr("BACKUP_LOCAL_DIR", ""),
		BackupCheckInterval:     getDuration("BACKUP_CHECK_INTERVAL", time.Hour),
		MultiUser:               getBool("MULTI_USER", false),
		APIAuthToken:            getStr("API_AUTH_TOKEN", ""),
		DBDriver:                getStr("DB_DRIVER", "sqlite"),
		DatabaseURL:             getStr("DATABASE_URL", ""),
		RegistrationMode:        getStr("AUTH_REGISTRATION_MODE", "invite"),
		SessionIdleTTL:          getDuration("SESSION_IDLE_TTL", 168*time.Hour),
		SessionAbsoluteTTL:      getDuration("SESSION_ABSOLUTE_TTL", 720*time.Hour),
		SessionRememberTTL:      getDuration("SESSION_REMEMBER_TTL", 2160*time.Hour),
		CookieSecure:            getBool("COOKIE_SECURE", true),
		CookieDomain:            getStr("COOKIE_DOMAIN", ""),
		TrustedProxies:          splitCSV(getStr("TRUSTED_PROXIES", "127.0.0.0/8,::1/128")),
		LogLevel:                getStr("LOG_LEVEL", "info"),
		TOTPIssuer:              getStr("TOTP_ISSUER", "DietDaemon"),
	}

	// COMPLETION_ADAPTER left unset: infer from which credentials are
	// actually present instead of defaulting to ollama, so setting
	// OPENAI_API_KEY (or ANTHROPIC_API_KEY) alone is enough to switch
	// providers without also having to set COMPLETION_ADAPTER explicitly.
	if c.CompletionAdapter == "" {
		switch {
		case c.OpenAIAPIKey != "":
			c.CompletionAdapter = "openai"
		case c.AnthropicAPIKey != "":
			c.CompletionAdapter = "anthropic"
		default:
			c.CompletionAdapter = "ollama"
		}
	}

	// TOTP encryption key: optional (TOTP is unavailable without it).
	if raw := getStr("TOTP_ENC_KEY", ""); raw != "" {
		key, err := decodeKey(raw)
		if err != nil {
			return nil, fmt.Errorf("TOTP_ENC_KEY: %w", err)
		}
		c.TOTPEncKey = key
	}

	// BYOK: per-user AI API key encryption (separate from TOTP for domain separation).
	if raw := getStr("AI_KEY_ENC_KEY", ""); raw != "" {
		key, err := decodeKey(raw)
		if err != nil {
			return nil, fmt.Errorf("AI_KEY_ENC_KEY: %w", err)
		}
		c.AIKeyEncKey = key
	}
	c.AIKeyMode = strings.ToLower(getStr("AI_KEY_MODE", "shared"))

	// OIDC providers.
	c.PublicBaseURL = strings.TrimRight(getStr("PUBLIC_BASE_URL", ""), "/")
	if raw := getStr("OIDC_PROVIDERS", ""); raw != "" {
		ids := splitCSV(raw)
		for _, id := range ids {
			canon := strings.ToUpper(id)
			name := getStr("OIDC_"+canon+"_NAME", "")
			if name == "" {
				name = strings.ToUpper(id[:1]) + id[1:]
			}
			cfg := OIDCProviderConfig{
				ID:           id,
				Name:         name,
				Issuer:       getStr("OIDC_"+canon+"_ISSUER", ""),
				ClientID:     getStr("OIDC_"+canon+"_CLIENT_ID", ""),
				ClientSecret: getStr("OIDC_"+canon+"_CLIENT_SECRET", ""),
				RedirectURL:  c.PublicBaseURL + "/api/v1/auth/oidc/" + id + "/callback",
				Scopes:       splitCSV(getStr("OIDC_"+canon+"_SCOPES", "openid,email,profile")),
				TrustEmail:   getBool("OIDC_"+canon+"_TRUST_EMAIL", false),
			}
			c.OIDCProviders = append(c.OIDCProviders, cfg)
		}
	}

	// Auth mailer / email settings.
	c.EmailProvider = strings.ToLower(getStr("EMAIL_PROVIDER", "none"))
	c.EmailFrom = getStr("EMAIL_FROM", "")
	c.ResendAPIKey = getStr("RESEND_API_KEY", "")
	c.SESRegion = getStr("SES_REGION", "")
	c.SMTPHost = getStr("SMTP_HOST", "")
	c.SMTPPort = getInt("SMTP_PORT", 587)
	c.SMTPUsername = getStr("SMTP_USERNAME", "")
	c.SMTPPassword = getStr("SMTP_PASSWORD", "")
	c.SMTPTLS = getBool("SMTP_TLS", true)

	// WebAuthn / Passkeys settings.
	c.WebAuthnRPID = getStr("WEBAUTHN_RP_ID", "")
	if raw := getStr("WEBAUTHN_RP_ORIGINS", ""); raw != "" {
		c.WebAuthnRPOrigins = splitCSV(raw)
	}
	c.WebAuthnRPDisplayName = getStr("WEBAUTHN_RP_DISPLAY_NAME", "DietDaemon")

	tier, tierErr := parseTier(getStr("PARSER_TIER", "0"))
	c.ParserTier = tier

	if err := c.validate(tierErr); err != nil {
		return nil, err
	}
	return c, nil
}

// validate collects every configuration problem and returns them as one error
// so the operator can fix them in a single pass.
func (c *Config) validate(tierErr error) error {
	var problems []string
	add := func(format string, args ...any) {
		problems = append(problems, fmt.Sprintf(format, args...))
	}

	if tierErr != nil {
		add("PARSER_TIER: %v", tierErr)
	}

	if c.MessagingAdapter == "" {
		add("MESSAGING_ADAPTER is required")
	}
	if c.MessagingAdapter == "telegram" && c.TelegramBotToken == "" {
		add("TELEGRAM_BOT_TOKEN is required when MESSAGING_ADAPTER=telegram")
	}
	if c.MessagingAdapter == "discord" && c.DiscordBotToken == "" {
		add("DISCORD_BOT_TOKEN is required when MESSAGING_ADAPTER=discord")
	}
	if c.MessagingAdapter == "matrix" {
		if c.MatrixHomeserverURL == "" {
			add("MATRIX_HOMESERVER_URL is required when MESSAGING_ADAPTER=matrix")
		}
		if c.MatrixUserID == "" {
			add("MATRIX_USER_ID is required when MESSAGING_ADAPTER=matrix")
		}
		if c.MatrixToken == "" {
			add("MATRIX_TOKEN is required when MESSAGING_ADAPTER=matrix")
		}
	}

	if c.EnableSTT && c.WhisperURL == "" {
		add("WHISPER_URL is required when ENABLE_STT=true")
	}

	if len(c.NutritionSources) == 0 {
		add("NUTRITION_SOURCE must list at least one source")
	}
	if contains(c.NutritionSources, "usda") && c.USDAFDCAPIKey == "" {
		add("USDA_FDC_API_KEY is required when 'usda' is in NUTRITION_SOURCE")
	}
	if contains(c.NutritionSources, "taco") && c.TacoDataPath != "" {
		if _, err := os.Stat(c.TacoDataPath); err != nil {
			add("TACO_DATA_PATH %q not found: %v", c.TacoDataPath, err)
		}
	}

	if c.FoodImportEnabled {
		if len(c.FoodImportSources) == 0 {
			add("FOOD_IMPORT_SOURCES must list at least one source when FOOD_IMPORT_ENABLED=true")
		}
		if contains(c.FoodImportSources, "usda") && c.USDAFDCAPIKey == "" {
			add("USDA_FDC_API_KEY is required when 'usda' is in FOOD_IMPORT_SOURCES")
		}
		if c.USDABulkFile != "" {
			if _, err := os.Stat(c.USDABulkFile); err != nil {
				add("USDA_BULK_FILE %q not found: %v", c.USDABulkFile, err)
			}
		}
		if c.OFFBulkFile != "" {
			if _, err := os.Stat(c.OFFBulkFile); err != nil {
				add("OFF_BULK_FILE %q not found: %v", c.OFFBulkFile, err)
			}
		}
	}

	if c.EmbedAdapter == "" {
		add("EMBED_ADAPTER is required")
	} else if c.EmbedAdapter != "ollama" {
		add("EMBED_ADAPTER must be \"ollama\" (no other adapter offers embeddings), got %q", c.EmbedAdapter)
	}
	validCompletion := map[string]bool{"ollama": true, "anthropic": true, "openai": true}
	if c.CompletionAdapter == "" {
		add("COMPLETION_ADAPTER is required")
	} else if !validCompletion[c.CompletionAdapter] {
		add("COMPLETION_ADAPTER must be one of: ollama, anthropic, openai, got %q", c.CompletionAdapter)
	}
	if c.CompletionAdapter == "anthropic" && c.AnthropicAPIKey == "" {
		add("ANTHROPIC_API_KEY is required when COMPLETION_ADAPTER=anthropic")
	}
	if c.CompletionAdapter == "openai" && c.OpenAIBaseURL == "" {
		add("OPENAI_BASE_URL is required when COMPLETION_ADAPTER=openai")
	}

	if c.ParserTier > types.TierDeterministic && c.OllamaURL == "" {
		add("OLLAMA_URL is required when PARSER_TIER > 0")
	}

	if c.EnableNotifications {
		if c.Notifier == "" {
			add("NOTIFIER is required when ENABLE_NOTIFICATIONS=true")
		}
		if c.Notifier == "ntfy" {
			if c.NtfyURL == "" {
				add("NTFY_URL is required when NOTIFIER=ntfy")
			}
			if c.NtfyTopic == "" {
				add("NTFY_TOPIC is required when NOTIFIER=ntfy")
			}
		}
		if c.Notifier == "gotify" {
			if c.GotifyURL == "" {
				add("GOTIFY_URL is required when NOTIFIER=gotify")
			}
			if c.GotifyToken == "" {
				add("GOTIFY_TOKEN is required when NOTIFIER=gotify")
			}
		}
	}

	if c.EmbedMatchThreshold <= 0 || c.EmbedMatchThreshold > 1 {
		add("EMBED_MATCH_THRESHOLD must be between 0 and 1")
	}
	if c.AliasWriteBackThreshold <= 0 || c.AliasWriteBackThreshold > 1 {
		add("ALIAS_WRITE_BACK_THRESHOLD must be between 0 and 1")
	}

	if c.DBPath == "" {
		add("DB_PATH is required")
	}

	if loc, err := time.LoadLocation(c.DefaultTimezone); err != nil {
		add("DEFAULT_TIMEZONE %q is not a valid IANA timezone: %v", c.DefaultTimezone, err)
	} else {
		c.Location = loc
	}

	// Core auth settings.
	switch c.DBDriver {
	case "sqlite":
		// DB_PATH already validated above.
	case "postgres":
		if c.DatabaseURL == "" {
			add("DATABASE_URL is required when DB_DRIVER=postgres")
		}
	default:
		add("DB_DRIVER must be \"sqlite\" or \"postgres\", got %q", c.DBDriver)
	}
	validModes := map[string]bool{"invite": true, "open": true, "oidc-only": true}
	if !validModes[c.RegistrationMode] {
		add("AUTH_REGISTRATION_MODE must be one of: invite, open, oidc-only, got %q", c.RegistrationMode)
	}
	if c.SessionIdleTTL <= 0 {
		add("SESSION_IDLE_TTL must be positive")
	}
	if c.SessionAbsoluteTTL <= 0 {
		add("SESSION_ABSOLUTE_TTL must be positive")
	}
	if c.SessionRememberTTL <= 0 {
		add("SESSION_REMEMBER_TTL must be positive")
	}
	for _, raw := range c.TrustedProxies {
		if _, err := parseProxyEntry(raw); err != nil {
			add("TRUSTED_PROXIES entry %q is not a valid IP or CIDR: %v", raw, err)
		}
	}

	// OIDC settings.
	if len(c.OIDCProviders) > 0 && c.PublicBaseURL == "" {
		add("PUBLIC_BASE_URL is required when OIDC_PROVIDERS is set")
	}
	for _, prov := range c.OIDCProviders {
		canon := strings.ToUpper(prov.ID)
		if prov.Issuer == "" {
			add("OIDC_%s_ISSUER is required", canon)
		}
		if prov.ClientID == "" {
			add("OIDC_%s_CLIENT_ID is required", canon)
		}
		if prov.ClientSecret == "" {
			add("OIDC_%s_CLIENT_SECRET is required", canon)
		}
	}
	if c.RegistrationMode == "oidc-only" && len(c.OIDCProviders) == 0 {
		add("AUTH_REGISTRATION_MODE is \"oidc-only\" but no OIDC_PROVIDERS configured")
	}

	// WebAuthn / passkey settings.
	if c.WebAuthnRPID != "" {
		if len(c.WebAuthnRPOrigins) == 0 {
			add("WEBAUTHN_RP_ORIGINS is required when WEBAUTHN_RP_ID is set")
		}
		for _, origin := range c.WebAuthnRPOrigins {
			if u, err := url.Parse(origin); err != nil || u.Scheme == "" || u.Host == "" {
				add("WEBAUTHN_RP_ORIGINS entry %q is not a valid URL", origin)
			}
		}
		if c.WebAuthnRPDisplayName == "" {
			add("WEBAUTHN_RP_DISPLAY_NAME is required when WEBAUTHN_RP_ID is set")
		}
	}

	// Mailer / email settings.
	validProviders := map[string]bool{"resend": true, "ses": true, "smtp": true, "none": true, "": true}
	if !validProviders[c.EmailProvider] {
		add("EMAIL_PROVIDER must be one of: resend, ses, smtp, none, got %q", c.EmailProvider)
	}
	if c.EmailProvider != "none" && c.EmailProvider != "" {
		if c.EmailFrom == "" {
			add("EMAIL_FROM is required when EMAIL_PROVIDER is not \"none\"")
		}
		if c.PublicBaseURL == "" {
			add("PUBLIC_BASE_URL is required when EMAIL_PROVIDER is not \"none\" (to build verification/reset links)")
		}
	}
	if c.EmailProvider == "resend" && c.ResendAPIKey == "" {
		add("RESEND_API_KEY is required when EMAIL_PROVIDER=resend")
	}
	if c.EmailProvider == "smtp" {
		if c.SMTPHost == "" {
			add("SMTP_HOST is required when EMAIL_PROVIDER=smtp")
		}
		if c.SMTPPort <= 0 || c.SMTPPort > 65535 {
			add("SMTP_PORT must be between 1 and 65535, got %d", c.SMTPPort)
		}
	}

	if len(problems) > 0 {
		return fmt.Errorf("invalid configuration:\n  - %s", strings.Join(problems, "\n  - "))
	}
	return nil
}

func parseTier(s string) (types.ParserTier, error) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return types.TierDeterministic, fmt.Errorf("must be an integer (0, 1, or 2), got %q", s)
	}
	if n < int(types.TierDeterministic) || n > int(types.TierLLM) {
		return types.TierDeterministic, fmt.Errorf("must be 0, 1, or 2, got %d", n)
	}
	return types.ParserTier(n), nil
}

func getStr(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return def
}

func getDuration(key string, def time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	d, err := time.ParseDuration(strings.TrimSpace(v))
	if err != nil {
		return def
	}
	return d
}

func getFloat(key string, def float64) float64 {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil {
		return def
	}
	return f
}

func getInt(key string, def int) int {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return def
	}
	return n
}

func getBool(key string, def bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}

func splitCSV(s string) []string {
	var out []string
	for p := range strings.SplitSeq(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// decodeKey accepts a base64 (standard or raw, with or without padding) or hex
// encoded key and returns the decoded bytes. Returns an error if the decoded
// key is not exactly 32 bytes.
func decodeKey(raw string) ([]byte, error) {
	// Try hex first (64 hex chars = 32 bytes).
	if len(raw) == 64 {
		key, err := hex.DecodeString(raw)
		if err == nil {
			return key, nil
		}
	}

	// Try base64 variants.
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	} {
		key, err := enc.DecodeString(raw)
		if err == nil {
			if len(key) == 32 {
				return key, nil
			}
		}
	}

	return nil, fmt.Errorf("must be a 32-byte key encoded as hex (64 chars) or base64, got %d chars", len(raw))
}

func contains(ss []string, target string) bool {
	return slices.Contains(ss, target)
}

// loadDotEnv reads simple KEY=VALUE lines from path into the environment without
// overriding variables already set. Missing file is not an error. Lines may use
// optional surrounding quotes and `#` comments.
func loadDotEnv(path string) {
	f, err := os.Open(path) // #nosec G304 -- path is always ".env", not user input
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, val)
		}
	}
}
