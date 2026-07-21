package config

import (
	"net/netip"
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
		"MESSAGING_ADAPTER", "TELEGRAM_BOT_TOKEN", "PARSER_TIER",
		"NUTRITION_SOURCE", "USDA_FDC_API_KEY", "TACO_DATA_PATH", "EMBED_ADAPTER", "COMPLETION_ADAPTER", "OLLAMA_URL",
		"FOOD_IMPORT_ENABLED", "FOOD_IMPORT_SOURCES", "FOOD_IMPORT_INTERVAL",
		"USDA_BULK_FILE", "USDA_BULK_DATA_TYPES", "USDA_BULK_MAX_ROWS",
		"OFF_BULK_FILE", "OFF_BULK_MIN_POPULARITY", "OFF_BULK_MAX_ROWS", "TACO_BULK_MAX_ROWS",
		"EMBED_MODEL", "LLM_MODEL", "MODEL_TIMEOUT", "OLLAMA_AUTO_PULL", "EMBED_MATCH_THRESHOLD", "ALIAS_WRITE_BACK_THRESHOLD",
		"ANTHROPIC_API_KEY", "ANTHROPIC_MODEL", "OPENAI_BASE_URL", "OPENAI_API_KEY", "OPENAI_MODEL",
		"NOTIFIER", "NTFY_URL", "NTFY_TOPIC", "DEFAULT_TIMEZONE", "DB_PATH",
		"ENABLE_NOTIFICATIONS", "ENABLE_DASHBOARD", "ENABLE_STT", "HSTS_ENABLED", "CORS_ALLOWED_ORIGINS", "LOG_LEVEL",
		"MULTI_USER", "API_AUTH_TOKEN",
		"DB_DRIVER", "DATABASE_URL",
		"TRUSTED_PROXIES",
		"PUBLIC_RATE_LIMIT_PER_MINUTE", "AUTH_READ_RATE_LIMIT_PER_MINUTE", "AUTH_WRITE_RATE_LIMIT_PER_MINUTE", "AUTH_EXPENSIVE_RATE_LIMIT_PER_MINUTE",
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
