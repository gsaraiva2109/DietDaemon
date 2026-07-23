package config

import (
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// setEnv sets the given vars for the duration of the test and clears anything
// else config reads so tests are hermetic.
func setEnv(t *testing.T, kv map[string]string) {
	t.Helper()
	keys := []string{
		"MESSAGING_ADAPTER", "TELEGRAM_BOT_TOKEN", "DISCORD_BOT_TOKEN",
		"MATRIX_HOMESERVER_URL", "MATRIX_USER_ID", "MATRIX_TOKEN", "PARSER_TIER",
		"NUTRITION_SOURCE", "USDA_FDC_API_KEY", "TACO_DATA_PATH", "EMBED_ADAPTER", "COMPLETION_ADAPTER", "OLLAMA_URL",
		"FOOD_IMPORT_ENABLED", "FOOD_IMPORT_SOURCES", "FOOD_IMPORT_INTERVAL",
		"USDA_BULK_FILE", "USDA_BULK_DATA_TYPES", "USDA_BULK_MAX_ROWS",
		"OFF_BULK_FILE", "OFF_BULK_MIN_POPULARITY", "OFF_BULK_MAX_ROWS", "TACO_BULK_MAX_ROWS",
		"EMBED_MODEL", "LLM_MODEL", "MODEL_TIMEOUT", "OLLAMA_AUTO_PULL", "EMBED_MATCH_THRESHOLD", "ALIAS_WRITE_BACK_THRESHOLD",
		"ANTHROPIC_API_KEY", "ANTHROPIC_MODEL", "OPENAI_BASE_URL", "OPENAI_API_KEY", "OPENAI_MODEL",
		"NOTIFIER", "NTFY_URL", "NTFY_TOPIC", "GOTIFY_URL", "GOTIFY_TOKEN", "DEFAULT_TIMEZONE", "DB_PATH",
		"ENABLE_NOTIFICATIONS", "ENABLE_DASHBOARD", "ENABLE_STT", "WHISPER_URL", "HSTS_ENABLED", "CORS_ALLOWED_ORIGINS", "LOG_LEVEL",
		"MULTI_USER", "API_AUTH_TOKEN",
		"DB_DRIVER", "DATABASE_URL",
		"AUTH_REGISTRATION_MODE", "SESSION_IDLE_TTL", "SESSION_ABSOLUTE_TTL", "SESSION_REMEMBER_TTL",
		"TRUSTED_PROXIES",
		"PUBLIC_RATE_LIMIT_PER_MINUTE", "AUTH_READ_RATE_LIMIT_PER_MINUTE", "AUTH_WRITE_RATE_LIMIT_PER_MINUTE", "AUTH_EXPENSIVE_RATE_LIMIT_PER_MINUTE",
		"PUBLIC_BASE_URL", "OIDC_PROVIDERS",
		"WEBAUTHN_RP_ID", "WEBAUTHN_RP_ORIGINS", "WEBAUTHN_RP_DISPLAY_NAME",
		"EMAIL_PROVIDER", "EMAIL_FROM", "RESEND_API_KEY", "SES_REGION",
		"SMTP_HOST", "SMTP_PORT", "SMTP_USERNAME", "SMTP_PASSWORD", "SMTP_TLS",
		"PORT", "HEALTH_CHECK_PATH", "CONFIDENCE_THRESHOLD", "NUDGE_INTERVAL", "PENDING_TTL", "MESSAGE_WORKERS",
	}
	for _, k := range keys {
		t.Setenv(k, "")
	}
	for k, v := range kv {
		t.Setenv(k, v)
	}
}

func validBase() map[string]string {
	return map[string]string{
		"MESSAGING_ADAPTER":    "telegram",
		"TELEGRAM_BOT_TOKEN":   "token123",
		"PARSER_TIER":          "0",
		"NUTRITION_SOURCE":     "openfoodfacts,taco",
		"NOTIFIER":             "ntfy",
		"NTFY_URL":             "https://ntfy.sh",
		"NTFY_TOPIC":           "diet",
		"DEFAULT_TIMEZONE":     "America/Sao_Paulo",
		"DB_PATH":              "/tmp/diet.db",
		"ENABLE_NOTIFICATIONS": "true",
	}
}

// assertLoadErr sets env and asserts Load() fails with an error containing
// wantSubstr. Shared by sibling validation-branch tests that otherwise repeat
// the same setEnv/Load/Contains/Fatalf boilerplate for each case.
func assertLoadErr(t *testing.T, env map[string]string, wantSubstr string) {
	t.Helper()
	setEnv(t, env)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), wantSubstr) {
		t.Fatalf("expected error containing %q, got %v", wantSubstr, err)
	}
}

// assertLoadOK sets env and asserts Load() succeeds, returning the config.
func assertLoadOK(t *testing.T, env map[string]string) *Config {
	t.Helper()
	setEnv(t, env)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	return c
}

// assertBulkFileMustExist asserts that pointing envKey at a nonexistent path
// fails Load() with a "not found" error, and that pointing it at a real file
// succeeds and populates the field read by getField. Shared by the USDA and
// OFF bulk-file tests, which are otherwise identical apart from the env key,
// file name, and field being checked.
func assertBulkFileMustExist(t *testing.T, env map[string]string, envKey string, getField func(*Config) string) {
	t.Helper()
	env[envKey] = filepath.Join(t.TempDir(), "missing.csv")
	setEnv(t, env)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), envKey) || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected %s not-found error, got %v", envKey, err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "bulk.csv")
	if err := os.WriteFile(path, []byte("data"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	env[envKey] = path
	setEnv(t, env)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if getField(c) != path {
		t.Errorf("%s = %q, want %q", envKey, getField(c), path)
	}
}

func TestLoadValid(t *testing.T) {
	setEnv(t, validBase())
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c.ParserTier != types.TierDeterministic {
		t.Errorf("ParserTier = %d, want 0", c.ParserTier)
	}
	if got := c.NutritionSources; len(got) != 2 || got[0] != "openfoodfacts" || got[1] != "taco" {
		t.Errorf("NutritionSources = %v, want [openfoodfacts taco]", got)
	}
	if c.Location == nil || c.Location.String() != "America/Sao_Paulo" {
		t.Errorf("Location = %v, want America/Sao_Paulo", c.Location)
	}
}

func TestOllamaAutoPull(t *testing.T) {
	env := validBase()
	env["OLLAMA_AUTO_PULL"] = "true"
	setEnv(t, env)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !c.OllamaAutoPull {
		t.Error("OllamaAutoPull = false, want true")
	}
}

func TestHTTPHardeningConfig(t *testing.T) {
	env := validBase()
	env["HSTS_ENABLED"] = "true"
	env["CORS_ALLOWED_ORIGINS"] = "https://app.example.com,http://localhost:5173"
	setEnv(t, env)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !c.HSTSEnabled {
		t.Error("HSTSEnabled = false, want true")
	}
	if got, want := strings.Join(c.CORSAllowedOrigins, ","), env["CORS_ALLOWED_ORIGINS"]; got != want {
		t.Errorf("CORSAllowedOrigins = %q, want %q", got, want)
	}
}

func TestRateLimitConfig(t *testing.T) {
	env := validBase()
	env["PUBLIC_RATE_LIMIT_PER_MINUTE"] = "11"
	env["AUTH_READ_RATE_LIMIT_PER_MINUTE"] = "121"
	env["AUTH_WRITE_RATE_LIMIT_PER_MINUTE"] = "31"
	env["AUTH_EXPENSIVE_RATE_LIMIT_PER_MINUTE"] = "12"
	setEnv(t, env)
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.PublicRateLimitPerMinute != 11 || c.AuthenticatedReadRateLimitPerMinute != 121 || c.AuthenticatedWriteRateLimitPerMinute != 31 || c.AuthenticatedExpensiveRateLimitPerMinute != 12 {
		t.Fatalf("rate limits = %+v", c)
	}
}

func TestRateLimitConfigRejectsNonPositive(t *testing.T) {
	env := validBase()
	env["AUTH_WRITE_RATE_LIMIT_PER_MINUTE"] = "0"
	setEnv(t, env)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "AUTH_WRITE_RATE_LIMIT_PER_MINUTE") {
		t.Fatalf("expected rate limit validation error, got %v", err)
	}
}

func TestCORSAllowedOriginsRejectsNonOrigins(t *testing.T) {
	for _, raw := range []string{"*", "example.com", "https://app.example.com/path", "https://app.example.com/"} {
		t.Run(raw, func(t *testing.T) {
			env := validBase()
			env["CORS_ALLOWED_ORIGINS"] = raw
			setEnv(t, env)
			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), "CORS_ALLOWED_ORIGINS") {
				t.Fatalf("expected CORS_ALLOWED_ORIGINS error, got %v", err)
			}
		})
	}
}

func TestMissingTelegramTokenFails(t *testing.T) {
	env := validBase()
	env["TELEGRAM_BOT_TOKEN"] = ""
	setEnv(t, env)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "TELEGRAM_BOT_TOKEN") {
		t.Fatalf("expected TELEGRAM_BOT_TOKEN error, got %v", err)
	}
}

func TestInvalidTimezoneFails(t *testing.T) {
	env := validBase()
	env["DEFAULT_TIMEZONE"] = "Mars/Phobos"
	setEnv(t, env)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "DEFAULT_TIMEZONE") {
		t.Fatalf("expected DEFAULT_TIMEZONE error, got %v", err)
	}
}

func TestTierRequiresModel(t *testing.T) {
	env := validBase()
	env["PARSER_TIER"] = "2"
	env["OLLAMA_URL"] = ""
	setEnv(t, env)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "OLLAMA_URL") {
		t.Fatalf("expected OLLAMA_URL error for tier>0, got %v", err)
	}
}

func TestEmbedAdapterMustBeOllama(t *testing.T) {
	env := validBase()
	env["EMBED_ADAPTER"] = "anthropic"
	setEnv(t, env)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "EMBED_ADAPTER") {
		t.Fatalf("expected EMBED_ADAPTER error, got %v", err)
	}
}

func TestCompletionAdapterAnthropicRequiresAPIKey(t *testing.T) {
	env := validBase()
	env["COMPLETION_ADAPTER"] = "anthropic"
	setEnv(t, env)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "ANTHROPIC_API_KEY") {
		t.Fatalf("expected ANTHROPIC_API_KEY error, got %v", err)
	}

	env["ANTHROPIC_API_KEY"] = "sk-ant-test"
	setEnv(t, env)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c.CompletionAdapter != "anthropic" {
		t.Errorf("CompletionAdapter = %q, want anthropic", c.CompletionAdapter)
	}
}

func TestCompletionAdapterDefaultsToOllamaZeroKeys(t *testing.T) {
	setEnv(t, validBase())
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c.EmbedAdapter != "ollama" || c.CompletionAdapter != "ollama" {
		t.Errorf("EmbedAdapter/CompletionAdapter = %q/%q, want ollama/ollama", c.EmbedAdapter, c.CompletionAdapter)
	}
}

func TestBadTierFails(t *testing.T) {
	env := validBase()
	env["PARSER_TIER"] = "9"
	setEnv(t, env)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "PARSER_TIER") {
		t.Fatalf("expected PARSER_TIER error, got %v", err)
	}
}

func TestThresholdValidation(t *testing.T) {
	env := validBase()
	env["EMBED_MATCH_THRESHOLD"] = "1.5"
	setEnv(t, env)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "EMBED_MATCH_THRESHOLD") {
		t.Fatalf("expected EMBED_MATCH_THRESHOLD validation error, got %v", err)
	}

	env = validBase()
	env["ALIAS_WRITE_BACK_THRESHOLD"] = "0.0"
	setEnv(t, env)
	_, err = Load()
	if err == nil || !strings.Contains(err.Error(), "ALIAS_WRITE_BACK_THRESHOLD") {
		t.Fatalf("expected ALIAS_WRITE_BACK_THRESHOLD validation error, got %v", err)
	}
}

func TestOperationalDefaults(t *testing.T) {
	setEnv(t, validBase())
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c.Port != "8080" {
		t.Errorf("Port = %q, want 8080", c.Port)
	}
	if c.HealthCheckPath != "/data/healthy" {
		t.Errorf("HealthCheckPath = %q, want /data/healthy", c.HealthCheckPath)
	}
	if c.ConfidenceThreshold != 0.6 {
		t.Errorf("ConfidenceThreshold = %v, want 0.6", c.ConfidenceThreshold)
	}
	if c.NudgeInterval != 5*time.Minute {
		t.Errorf("NudgeInterval = %v, want 5m", c.NudgeInterval)
	}
	if c.PendingTTL != 30*time.Minute {
		t.Errorf("PendingTTL = %v, want 30m", c.PendingTTL)
	}
	if c.MessageWorkers != 4 {
		t.Errorf("MessageWorkers = %d, want 4", c.MessageWorkers)
	}
}

func TestOperationalOverrides(t *testing.T) {
	env := validBase()
	env["PORT"] = "9090"
	env["HEALTH_CHECK_PATH"] = "/tmp/alive"
	env["CONFIDENCE_THRESHOLD"] = "0.75"
	env["NUDGE_INTERVAL"] = "10m"
	env["PENDING_TTL"] = "1h"
	env["MESSAGE_WORKERS"] = "8"
	setEnv(t, env)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c.Port != "9090" || c.HealthCheckPath != "/tmp/alive" || c.ConfidenceThreshold != 0.75 ||
		c.NudgeInterval != 10*time.Minute || c.PendingTTL != time.Hour || c.MessageWorkers != 8 {
		t.Fatalf("operational overrides = %+v", c)
	}
}

func TestOperationalValidation(t *testing.T) {
	cases := []struct {
		name string
		key  string
		val  string
		want string
	}{
		{"bad port too low", "PORT", "0", "PORT"},
		{"bad port too high", "PORT", "70000", "PORT"},
		{"bad port not numeric", "PORT", "abc", "PORT"},
		{"confidence threshold too low", "CONFIDENCE_THRESHOLD", "0", "CONFIDENCE_THRESHOLD"},
		{"confidence threshold too high", "CONFIDENCE_THRESHOLD", "1.5", "CONFIDENCE_THRESHOLD"},
		{"nudge interval non-positive", "NUDGE_INTERVAL", "0s", "NUDGE_INTERVAL"},
		{"pending ttl non-positive", "PENDING_TTL", "0s", "PENDING_TTL"},
		{"message workers below one", "MESSAGE_WORKERS", "0", "MESSAGE_WORKERS"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := validBase()
			env[tc.key] = tc.val
			setEnv(t, env)
			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %s validation error, got %v", tc.want, err)
			}
		})
	}
}

func TestAuthAndMultiUser(t *testing.T) {
	env := validBase()
	env["MULTI_USER"] = "true"
	env["API_AUTH_TOKEN"] = "secure123"
	setEnv(t, env)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !c.MultiUser {
		t.Error("expected MultiUser to be true")
	}
	if c.APIAuthToken != "secure123" {
		t.Errorf("APIAuthToken = %q, want secure123", c.APIAuthToken)
	}
}

func TestDefaultDBDriverIsSQLite(t *testing.T) {
	setEnv(t, validBase())
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c.DBDriver != "sqlite" {
		t.Errorf("DBDriver = %q, want \"sqlite\"", c.DBDriver)
	}
}

func TestPostgresDriverRequiresDatabaseURL(t *testing.T) {
	env := validBase()
	env["DB_DRIVER"] = "postgres"
	env["DATABASE_URL"] = ""
	setEnv(t, env)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("expected DATABASE_URL error, got %v", err)
	}
}

func TestPostgresDriverValid(t *testing.T) {
	env := validBase()
	env["DB_DRIVER"] = "postgres"
	env["DATABASE_URL"] = "postgres://localhost/mydb"
	setEnv(t, env)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c.DBDriver != "postgres" {
		t.Errorf("DBDriver = %q, want \"postgres\"", c.DBDriver)
	}
	if c.DatabaseURL != "postgres://localhost/mydb" {
		t.Errorf("DatabaseURL = %q, want \"postgres://localhost/mydb\"", c.DatabaseURL)
	}
}

func TestInvalidDBDriverFails(t *testing.T) {
	env := validBase()
	env["DB_DRIVER"] = "mysql"
	setEnv(t, env)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "DB_DRIVER") {
		t.Fatalf("expected DB_DRIVER error, got %v", err)
	}
}

func TestSQLiteDriverDoesNotRequireDatabaseURL(t *testing.T) {
	env := validBase()
	env["DB_DRIVER"] = "sqlite"
	env["DATABASE_URL"] = ""
	setEnv(t, env)
	_, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
}

// --- TrustedProxies (clientIP spoofing fix) ---

func TestTrustedProxiesDefaultsToLoopbackOnly(t *testing.T) {
	setEnv(t, validBase())
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	prefixes := c.TrustedProxyPrefixes()
	if !containsAddr(prefixes, "127.0.0.1") || !containsAddr(prefixes, "::1") {
		t.Fatalf("expected loopback to be trusted by default, got %v", c.TrustedProxies)
	}
	if containsAddr(prefixes, "203.0.113.5") {
		t.Fatalf("public IP must not be trusted by default, got %v", c.TrustedProxies)
	}
}

func TestTrustedProxiesCustomCIDRAndBareIP(t *testing.T) {
	env := validBase()
	env["TRUSTED_PROXIES"] = "10.0.0.0/8,203.0.113.5"
	setEnv(t, env)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	prefixes := c.TrustedProxyPrefixes()
	if !containsAddr(prefixes, "10.1.2.3") {
		t.Errorf("expected 10.0.0.0/8 to cover 10.1.2.3")
	}
	if !containsAddr(prefixes, "203.0.113.5") {
		t.Errorf("expected bare IP entry to be trusted as a /32")
	}
	if containsAddr(prefixes, "203.0.113.6") {
		t.Errorf("bare IP entry must not cover a neighboring address")
	}
	if containsAddr(prefixes, "127.0.0.1") {
		t.Errorf("loopback default must not apply once TRUSTED_PROXIES is set explicitly")
	}
}

func TestTrustedProxiesInvalidEntryFails(t *testing.T) {
	env := validBase()
	env["TRUSTED_PROXIES"] = "not-an-ip"
	setEnv(t, env)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "TRUSTED_PROXIES") {
		t.Fatalf("expected TRUSTED_PROXIES validation error, got %v", err)
	}
}

func containsAddr(prefixes []netip.Prefix, addr string) bool {
	a := netip.MustParseAddr(addr)
	for _, p := range prefixes {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

// TestLoadMinimalSQLiteOnly is the whole point of issue #133: a one-shot CLI
// tool (cmd/import-foods) must not be forced to set daemon-only env vars
// (messaging, notifier, OIDC, email, ...) just to load config.
func TestLoadMinimalSQLiteOnly(t *testing.T) {
	setEnv(t, map[string]string{"DB_DRIVER": "sqlite"})
	c, err := LoadMinimal()
	if err != nil {
		t.Fatalf("LoadMinimal() error = %v", err)
	}
	if c.DBDriver != "sqlite" {
		t.Errorf("DBDriver = %q, want sqlite", c.DBDriver)
	}
}

func TestLoadMinimalPostgresRequiresDatabaseURL(t *testing.T) {
	setEnv(t, map[string]string{"DB_DRIVER": "postgres"})
	_, err := LoadMinimal()
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("expected DATABASE_URL validation error, got %v", err)
	}
}

func TestLoadMinimalPostgresWithDatabaseURL(t *testing.T) {
	setEnv(t, map[string]string{"DB_DRIVER": "postgres", "DATABASE_URL": "postgres://user:pass@host/db"})
	c, err := LoadMinimal()
	if err != nil {
		t.Fatalf("LoadMinimal() error = %v", err)
	}
	if c.DatabaseURL != "postgres://user:pass@host/db" {
		t.Errorf("DatabaseURL = %q, want postgres://user:pass@host/db", c.DatabaseURL)
	}
}

// --- Messaging adapters: discord, matrix (issue #158) ---

func TestMessagingAdapterDiscordRequiresToken(t *testing.T) {
	env := validBase()
	env["MESSAGING_ADAPTER"] = "discord"
	env["TELEGRAM_BOT_TOKEN"] = ""
	setEnv(t, env)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "DISCORD_BOT_TOKEN") {
		t.Fatalf("expected DISCORD_BOT_TOKEN error, got %v", err)
	}

	env["DISCORD_BOT_TOKEN"] = "discord-token"
	setEnv(t, env)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c.MessagingAdapter != "discord" || c.DiscordBotToken != "discord-token" {
		t.Errorf("MessagingAdapter/DiscordBotToken = %q/%q", c.MessagingAdapter, c.DiscordBotToken)
	}
}

func TestMessagingAdapterMatrixRequiresFields(t *testing.T) {
	cases := []struct {
		name    string
		missing string
		want    string
	}{
		{"missing homeserver url", "MATRIX_HOMESERVER_URL", "MATRIX_HOMESERVER_URL"},
		{"missing user id", "MATRIX_USER_ID", "MATRIX_USER_ID"},
		{"missing token", "MATRIX_TOKEN", "MATRIX_TOKEN"},
	}
	full := map[string]string{
		"MATRIX_HOMESERVER_URL": "https://matrix.example.com",
		"MATRIX_USER_ID":        "@bot:example.com",
		"MATRIX_TOKEN":          "matrix-token",
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := validBase()
			env["MESSAGING_ADAPTER"] = "matrix"
			env["TELEGRAM_BOT_TOKEN"] = ""
			for k, v := range full {
				env[k] = v
			}
			env[tc.missing] = ""
			setEnv(t, env)
			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %s error, got %v", tc.want, err)
			}
		})
	}

	env := validBase()
	env["MESSAGING_ADAPTER"] = "matrix"
	env["TELEGRAM_BOT_TOKEN"] = ""
	for k, v := range full {
		env[k] = v
	}
	setEnv(t, env)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c.MatrixHomeserverURL != full["MATRIX_HOMESERVER_URL"] || c.MatrixUserID != full["MATRIX_USER_ID"] || c.MatrixToken != full["MATRIX_TOKEN"] {
		t.Errorf("matrix fields = %+v", c)
	}
}

// --- STT (ENABLE_STT / WHISPER_URL) ---

// WHISPER_URL defaults to a non-empty value ("http://whisper:8080"), and
// getStr() falls back to the default whenever the env var is unset OR blank
// — there is no way to make WhisperURL == "" via env vars alone. That means
// the "WHISPER_URL is required when ENABLE_STT=true" branch is unreachable
// through Load(); these tests pin down the reachable behavior instead.
func TestEnableSTTWithDefaultWhisperURLPasses(t *testing.T) {
	env := validBase()
	env["ENABLE_STT"] = "true"
	setEnv(t, env)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !c.EnableSTT || c.WhisperURL != "http://whisper:8080" {
		t.Errorf("EnableSTT/WhisperURL = %v/%q, want true/http://whisper:8080", c.EnableSTT, c.WhisperURL)
	}
}

func TestEnableSTTWithCustomWhisperURLPasses(t *testing.T) {
	env := validBase()
	env["ENABLE_STT"] = "true"
	env["WHISPER_URL"] = "http://custom-whisper:9000"
	setEnv(t, env)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c.WhisperURL != "http://custom-whisper:9000" {
		t.Errorf("WhisperURL = %q, want http://custom-whisper:9000", c.WhisperURL)
	}
}

// --- Nutrition sources / TACO_DATA_PATH ---

// NUTRITION_SOURCE also defaults to a non-empty value ("openfoodfacts"), so
// forcing an empty NutritionSources slice requires a value that survives
// getStr's blank check but splits down to nothing, e.g. a bare comma.
func TestNutritionSourceEmptyListFails(t *testing.T) {
	env := validBase()
	env["NUTRITION_SOURCE"] = ","
	setEnv(t, env)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "NUTRITION_SOURCE must list at least one source") {
		t.Fatalf("expected NUTRITION_SOURCE empty-list error, got %v", err)
	}
}

func TestNutritionSourceUSDARequiresAPIKey(t *testing.T) {
	env := validBase()
	env["NUTRITION_SOURCE"] = "usda"
	setEnv(t, env)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "USDA_FDC_API_KEY is required when 'usda' is in NUTRITION_SOURCE") {
		t.Fatalf("expected USDA_FDC_API_KEY error, got %v", err)
	}

	env["USDA_FDC_API_KEY"] = "usda-key"
	setEnv(t, env)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c.USDAFDCAPIKey != "usda-key" {
		t.Errorf("USDAFDCAPIKey = %q, want usda-key", c.USDAFDCAPIKey)
	}
}

func TestTacoDataPathNotFoundFails(t *testing.T) {
	env := validBase()
	env["NUTRITION_SOURCE"] = "taco"
	env["TACO_DATA_PATH"] = filepath.Join(t.TempDir(), "missing.csv")
	setEnv(t, env)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "TACO_DATA_PATH") || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected TACO_DATA_PATH not-found error, got %v", err)
	}
}

func TestTacoDataPathFoundPasses(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "taco.csv")
	if err := os.WriteFile(path, []byte("data"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	env := validBase()
	env["NUTRITION_SOURCE"] = "taco"
	env["TACO_DATA_PATH"] = path
	setEnv(t, env)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c.TacoDataPath != path {
		t.Errorf("TacoDataPath = %q, want %q", c.TacoDataPath, path)
	}
}

// --- Food import (bulk sources / API keys / bulk files) ---

func TestFoodImportEnabledRequiresSources(t *testing.T) {
	env := validBase()
	env["FOOD_IMPORT_ENABLED"] = "true"
	setEnv(t, env)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "FOOD_IMPORT_SOURCES must list at least one source") {
		t.Fatalf("expected FOOD_IMPORT_SOURCES error, got %v", err)
	}
}

func TestFoodImportUSDASourceRequiresAPIKey(t *testing.T) {
	env := validBase()
	env["FOOD_IMPORT_ENABLED"] = "true"
	env["FOOD_IMPORT_SOURCES"] = "usda"
	setEnv(t, env)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "USDA_FDC_API_KEY is required when 'usda' is in FOOD_IMPORT_SOURCES") {
		t.Fatalf("expected USDA_FDC_API_KEY error, got %v", err)
	}

	env["USDA_FDC_API_KEY"] = "usda-key"
	setEnv(t, env)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(c.FoodImportSources) != 1 || c.FoodImportSources[0] != "usda" {
		t.Errorf("FoodImportSources = %v, want [usda]", c.FoodImportSources)
	}
}

func TestFoodImportUSDABulkFileMustExist(t *testing.T) {
	env := validBase()
	env["FOOD_IMPORT_ENABLED"] = "true"
	env["FOOD_IMPORT_SOURCES"] = "openfoodfacts"
	assertBulkFileMustExist(t, env, "USDA_BULK_FILE", func(c *Config) string { return c.USDABulkFile })
}

func TestFoodImportOFFBulkFileMustExist(t *testing.T) {
	env := validBase()
	env["FOOD_IMPORT_ENABLED"] = "true"
	env["FOOD_IMPORT_SOURCES"] = "openfoodfacts"
	assertBulkFileMustExist(t, env, "OFF_BULK_FILE", func(c *Config) string { return c.OFFBulkFile })
}

// --- Notifier=gotify ---

func TestNotifierGotifyRequiresURLAndToken(t *testing.T) {
	env := validBase()
	env["NOTIFIER"] = "gotify"
	env["NTFY_URL"] = ""
	env["NTFY_TOPIC"] = ""
	assertLoadErr(t, env, "GOTIFY_URL is required when NOTIFIER=gotify")

	env["GOTIFY_URL"] = "https://gotify.example.com"
	assertLoadErr(t, env, "GOTIFY_TOKEN is required when NOTIFIER=gotify")

	env["GOTIFY_TOKEN"] = "gotify-token"
	c := assertLoadOK(t, env)
	if c.GotifyURL != "https://gotify.example.com" || c.GotifyToken != "gotify-token" {
		t.Errorf("Gotify fields = %q/%q", c.GotifyURL, c.GotifyToken)
	}
}

// --- OIDC providers ---

func oidcValidBase() map[string]string {
	env := validBase()
	env["PUBLIC_BASE_URL"] = "https://app.example.com"
	env["OIDC_PROVIDERS"] = "google"
	env["OIDC_GOOGLE_ISSUER"] = "https://accounts.google.com"
	env["OIDC_GOOGLE_CLIENT_ID"] = "client-id"
	env["OIDC_GOOGLE_CLIENT_SECRET"] = "client-secret"
	return env
}

func TestOIDCProviderValid(t *testing.T) {
	env := oidcValidBase()
	setEnv(t, env)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(c.OIDCProviders) != 1 {
		t.Fatalf("OIDCProviders = %v, want 1 entry", c.OIDCProviders)
	}
	p := c.OIDCProviders[0]
	if p.ID != "google" || p.Name != "Google" || p.Issuer != "https://accounts.google.com" ||
		p.ClientID != "client-id" || p.ClientSecret != "client-secret" {
		t.Errorf("OIDCProviders[0] = %+v", p)
	}
	if p.RedirectURL != "https://app.example.com/api/v1/auth/oidc/google/callback" {
		t.Errorf("RedirectURL = %q", p.RedirectURL)
	}
}

func TestOIDCProvidersRequiresPublicBaseURL(t *testing.T) {
	env := oidcValidBase()
	env["PUBLIC_BASE_URL"] = ""
	setEnv(t, env)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "PUBLIC_BASE_URL is required when OIDC_PROVIDERS is set") {
		t.Fatalf("expected PUBLIC_BASE_URL error, got %v", err)
	}
}

func TestOIDCProviderRequiresIssuerClientIDAndSecret(t *testing.T) {
	cases := []struct {
		name string
		key  string
		want string
	}{
		{"missing issuer", "OIDC_GOOGLE_ISSUER", "OIDC_GOOGLE_ISSUER is required"},
		{"missing client id", "OIDC_GOOGLE_CLIENT_ID", "OIDC_GOOGLE_CLIENT_ID is required"},
		{"missing client secret", "OIDC_GOOGLE_CLIENT_SECRET", "OIDC_GOOGLE_CLIENT_SECRET is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := oidcValidBase()
			env[tc.key] = ""
			setEnv(t, env)
			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q error, got %v", tc.want, err)
			}
		})
	}
}

func TestOIDCOnlyRegistrationModeRequiresProvider(t *testing.T) {
	env := validBase()
	env["AUTH_REGISTRATION_MODE"] = "oidc-only"
	setEnv(t, env)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "AUTH_REGISTRATION_MODE is \"oidc-only\" but no OIDC_PROVIDERS configured") {
		t.Fatalf("expected oidc-only error, got %v", err)
	}

	env = oidcValidBase()
	env["AUTH_REGISTRATION_MODE"] = "oidc-only"
	setEnv(t, env)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c.RegistrationMode != "oidc-only" {
		t.Errorf("RegistrationMode = %q, want oidc-only", c.RegistrationMode)
	}
}

// --- WebAuthn / passkeys ---

// WEBAUTHN_RP_DISPLAY_NAME defaults to a non-empty value ("DietDaemon") via
// getStr, which falls back to the default whenever the env var is unset or
// blank — so "WEBAUTHN_RP_DISPLAY_NAME is required when WEBAUTHN_RP_ID is
// set" is unreachable through Load(); only the origins requirement is.
func TestWebAuthnRPIDRequiresOrigins(t *testing.T) {
	env := validBase()
	env["WEBAUTHN_RP_ID"] = "example.com"
	setEnv(t, env)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "WEBAUTHN_RP_ORIGINS is required when WEBAUTHN_RP_ID is set") {
		t.Fatalf("expected WEBAUTHN_RP_ORIGINS error, got %v", err)
	}
}

func TestWebAuthnRPOriginsMustBeValidURLs(t *testing.T) {
	env := validBase()
	env["WEBAUTHN_RP_ID"] = "example.com"
	env["WEBAUTHN_RP_ORIGINS"] = "not a url"
	setEnv(t, env)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "WEBAUTHN_RP_ORIGINS entry \"not a url\" is not a valid URL") {
		t.Fatalf("expected WEBAUTHN_RP_ORIGINS URL error, got %v", err)
	}
}

func TestWebAuthnRPIDValid(t *testing.T) {
	env := validBase()
	env["WEBAUTHN_RP_ID"] = "example.com"
	env["WEBAUTHN_RP_ORIGINS"] = "https://example.com"
	setEnv(t, env)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c.WebAuthnRPID != "example.com" || len(c.WebAuthnRPOrigins) != 1 || c.WebAuthnRPOrigins[0] != "https://example.com" {
		t.Errorf("WebAuthn fields = %q/%v", c.WebAuthnRPID, c.WebAuthnRPOrigins)
	}
	if c.WebAuthnRPDisplayName != "DietDaemon" {
		t.Errorf("WebAuthnRPDisplayName = %q, want default DietDaemon", c.WebAuthnRPDisplayName)
	}
}

// --- Mailer / email provider ---

func TestEmailProviderInvalidValueFails(t *testing.T) {
	env := validBase()
	env["EMAIL_PROVIDER"] = "sendgrid"
	setEnv(t, env)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "EMAIL_PROVIDER must be one of: resend, ses, smtp, none, got \"sendgrid\"") {
		t.Fatalf("expected EMAIL_PROVIDER error, got %v", err)
	}
}

func TestEmailProviderNoneRequiresNothing(t *testing.T) {
	setEnv(t, validBase())
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c.EmailProvider != "none" {
		t.Errorf("EmailProvider = %q, want none", c.EmailProvider)
	}
}

func TestEmailProviderRequiresFromAndPublicBaseURL(t *testing.T) {
	for _, provider := range []string{"resend", "ses", "smtp"} {
		t.Run(provider, func(t *testing.T) {
			env := validBase()
			env["EMAIL_PROVIDER"] = provider
			env["RESEND_API_KEY"] = "resend-key"
			env["SMTP_HOST"] = "smtp.example.com"
			assertLoadErr(t, env, "EMAIL_FROM is required when EMAIL_PROVIDER is not \"none\"")

			env["EMAIL_FROM"] = "noreply@example.com"
			assertLoadErr(t, env, "PUBLIC_BASE_URL is required when EMAIL_PROVIDER is not \"none\"")
		})
	}
}

func TestEmailProviderResendRequiresAPIKey(t *testing.T) {
	env := validBase()
	env["EMAIL_PROVIDER"] = "resend"
	env["EMAIL_FROM"] = "noreply@example.com"
	env["PUBLIC_BASE_URL"] = "https://app.example.com"
	setEnv(t, env)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "RESEND_API_KEY is required when EMAIL_PROVIDER=resend") {
		t.Fatalf("expected RESEND_API_KEY error, got %v", err)
	}

	env["RESEND_API_KEY"] = "resend-key"
	setEnv(t, env)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c.ResendAPIKey != "resend-key" {
		t.Errorf("ResendAPIKey = %q, want resend-key", c.ResendAPIKey)
	}
}

func TestEmailProviderSESPassesWithoutRegion(t *testing.T) {
	env := validBase()
	env["EMAIL_PROVIDER"] = "ses"
	env["EMAIL_FROM"] = "noreply@example.com"
	env["PUBLIC_BASE_URL"] = "https://app.example.com"
	setEnv(t, env)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c.EmailProvider != "ses" || c.SESRegion != "" {
		t.Errorf("EmailProvider/SESRegion = %q/%q, want ses/\"\" (SES_REGION is not validated)", c.EmailProvider, c.SESRegion)
	}
}

func TestEmailProviderSMTPRequiresHostAndValidPort(t *testing.T) {
	env := validBase()
	env["EMAIL_PROVIDER"] = "smtp"
	env["EMAIL_FROM"] = "noreply@example.com"
	env["PUBLIC_BASE_URL"] = "https://app.example.com"
	assertLoadErr(t, env, "SMTP_HOST is required when EMAIL_PROVIDER=smtp")

	env["SMTP_HOST"] = "smtp.example.com"
	env["SMTP_PORT"] = "0"
	assertLoadErr(t, env, "SMTP_PORT must be between 1 and 65535, got 0")

	env["SMTP_PORT"] = "70000"
	assertLoadErr(t, env, "SMTP_PORT must be between 1 and 65535, got 70000")

	env["SMTP_PORT"] = "2525"
	c := assertLoadOK(t, env)
	if c.SMTPHost != "smtp.example.com" || c.SMTPPort != 2525 {
		t.Errorf("SMTPHost/SMTPPort = %q/%d", c.SMTPHost, c.SMTPPort)
	}
}

func TestEmailProviderSMTPDefaultPort(t *testing.T) {
	env := validBase()
	env["EMAIL_PROVIDER"] = "smtp"
	env["EMAIL_FROM"] = "noreply@example.com"
	env["PUBLIC_BASE_URL"] = "https://app.example.com"
	env["SMTP_HOST"] = "smtp.example.com"
	setEnv(t, env)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c.SMTPPort != 587 {
		t.Errorf("SMTPPort = %d, want default 587", c.SMTPPort)
	}
}
