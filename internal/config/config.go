// Package config loads and validates DietDaemon's runtime configuration from
// environment variables (see .env.example for the full list). Validation runs
// at boot and fails fast, reporting every problem at once so a misconfigured
// self-host is obvious immediately rather than crashing deep in a pipeline.
package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// Config is the fully parsed, validated configuration. Location is derived from
// DefaultTimezone so the rest of the app never re-parses the tz string.
type Config struct {
	MessagingAdapter string
	TelegramBotToken string

	ParserTier types.ParserTier

	NutritionSources []string
	USDAFDCAPIKey    string
	TacoDataPath     string

	ModelAdapter string
	OllamaURL    string
	EmbedModel   string
	LLMModel     string
	ModelTimeout time.Duration

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

	MultiUser    bool
	APIAuthToken string

	LogLevel string
}

// Load reads configuration from the environment, applying values from a .env
// file in the working directory first (without overriding variables already set
// in the environment), then validates the result.
func Load() (*Config, error) {
	loadDotEnv(".env")

	c := &Config{
		MessagingAdapter:        getStr("MESSAGING_ADAPTER", "telegram"),
		TelegramBotToken:        getStr("TELEGRAM_BOT_TOKEN", ""),
		NutritionSources:        splitCSV(getStr("NUTRITION_SOURCE", "openfoodfacts")),
		USDAFDCAPIKey:           getStr("USDA_FDC_API_KEY", ""),
		TacoDataPath:            getStr("TACO_DATA_PATH", ""),
		ModelAdapter:            getStr("MODEL_ADAPTER", "ollama"),
		OllamaURL:               getStr("OLLAMA_URL", ""),
		EmbedModel:              getStr("EMBED_MODEL", "nomic-embed-text"),
		LLMModel:                getStr("LLM_MODEL", "llama3.1"),
		ModelTimeout:            getDuration("MODEL_TIMEOUT", 30*time.Second),
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
		MultiUser:               getBool("MULTI_USER", false),
		APIAuthToken:            getStr("API_AUTH_TOKEN", ""),
		LogLevel:                getStr("LOG_LEVEL", "info"),
	}

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

	if c.ParserTier > types.TierDeterministic {
		if c.ModelAdapter == "" {
			add("MODEL_ADAPTER is required when PARSER_TIER > 0")
		}
		if c.OllamaURL == "" {
			add("OLLAMA_URL is required when PARSER_TIER > 0")
		}
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
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func contains(ss []string, target string) bool {
	for _, s := range ss {
		if s == target {
			return true
		}
	}
	return false
}

// loadDotEnv reads simple KEY=VALUE lines from path into the environment without
// overriding variables already set. Missing file is not an error. Lines may use
// optional surrounding quotes and `#` comments.
func loadDotEnv(path string) {
	f, err := os.Open(path) // #nosec G304 -- path is always ".env", not user input
	if err != nil {
		return
	}
	defer f.Close()

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
