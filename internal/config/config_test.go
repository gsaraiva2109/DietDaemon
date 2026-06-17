package config

import (
	"strings"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// setEnv sets the given vars for the duration of the test and clears anything
// else config reads so tests are hermetic.
func setEnv(t *testing.T, kv map[string]string) {
	t.Helper()
	keys := []string{
		"MESSAGING_ADAPTER", "TELEGRAM_BOT_TOKEN", "PARSER_TIER",
		"NUTRITION_SOURCE", "USDA_FDC_API_KEY", "TACO_DATA_PATH", "MODEL_ADAPTER", "OLLAMA_URL",
		"NOTIFIER", "NTFY_URL", "NTFY_TOPIC", "DEFAULT_TIMEZONE", "DB_PATH",
		"ENABLE_NOTIFICATIONS", "ENABLE_DASHBOARD", "ENABLE_STT", "LOG_LEVEL",
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
		"TACO_DATA_PATH":       "/data/taco.csv",
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
	env["MODEL_ADAPTER"] = ""
	env["OLLAMA_URL"] = ""
	setEnv(t, env)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "OLLAMA_URL") {
		t.Fatalf("expected OLLAMA_URL error for tier>0, got %v", err)
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
