package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/adapters/messaging/discord"
	"github.com/gsaraiva2109/dietdaemon/adapters/messaging/matrix"
	"github.com/gsaraiva2109/dietdaemon/adapters/messaging/telegram"
	"github.com/gsaraiva2109/dietdaemon/adapters/model/anthropic"
	"github.com/gsaraiva2109/dietdaemon/adapters/model/ollama"
	"github.com/gsaraiva2109/dietdaemon/adapters/model/openai"
	"github.com/gsaraiva2109/dietdaemon/adapters/notifier/gotify"
	"github.com/gsaraiva2109/dietdaemon/adapters/notifier/ntfy"
	"github.com/gsaraiva2109/dietdaemon/adapters/nutrition/openfoodfacts"
	"github.com/gsaraiva2109/dietdaemon/adapters/nutrition/taco"
	"github.com/gsaraiva2109/dietdaemon/adapters/nutrition/usda"
	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/config"
)

func TestRequiredOllamaModels(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
		want []string
	}{
		{"tier 0", config.Config{EmbedModel: "embed", LLMModel: "llm"}, nil},
		{"tier 1", config.Config{ParserTier: types.TierEmbedding, EmbedModel: "embed", LLMModel: "llm"}, []string{"embed"}},
		{"tier 2", config.Config{ParserTier: types.TierLLM, CompletionAdapter: "ollama", EmbedModel: "embed", LLMModel: "llm"}, []string{"embed", "llm"}},
		{"dashboard chat", config.Config{CompletionAdapter: "ollama", EnableDashboard: true, EmbedModel: "embed", LLMModel: "llm"}, []string{"llm"}},
		{"cloud completion", config.Config{ParserTier: types.TierLLM, CompletionAdapter: "openai", EmbedModel: "embed", LLMModel: "llm"}, []string{"embed"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := requiredOllamaModels(&tt.cfg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("requiredOllamaModels() = %v, want %v", got, tt.want)
			}
		})
	}
}

// --- Characterization tests for the 7 pure builder/switch functions run()
// delegates to. Construction never dials out (see comment above run()'s
// buildCompletionAdapter call), so these are safely unit-testable, unlike
// run() itself which opens a real DB/HTTP listener and blocks on wg.Wait().

func TestBuildEmbedAdapter(t *testing.T) {
	if got, err := buildEmbedAdapter(&config.Config{EmbedAdapter: "ollama"}); err != nil {
		t.Fatalf("ollama: unexpected error: %v", err)
	} else if _, ok := got.(*ollama.Adapter); !ok {
		t.Fatalf("ollama: got %T, want *ollama.Adapter", got)
	}

	got, err := buildEmbedAdapter(&config.Config{EmbedAdapter: "bogus"})
	if got != nil || err == nil || !strings.Contains(err.Error(), `unsupported EMBED_ADAPTER "bogus"`) {
		t.Fatalf("default: got %v, %v", got, err)
	}
}

func TestBuildCompletionAdapter(t *testing.T) {
	tests := []struct {
		name    string
		adapter string
		want    any
	}{
		{"ollama", "ollama", &ollama.Adapter{}},
		{"anthropic", "anthropic", &anthropic.Adapter{}},
		{"openai", "openai", &openai.Adapter{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildCompletionAdapter(&config.Config{CompletionAdapter: tt.adapter})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if reflect.TypeOf(got) != reflect.TypeOf(tt.want) {
				t.Fatalf("got %T, want %T", got, tt.want)
			}
		})
	}

	got, err := buildCompletionAdapter(&config.Config{CompletionAdapter: "bogus"})
	if got != nil || err == nil || !strings.Contains(err.Error(), `unsupported COMPLETION_ADAPTER "bogus"`) {
		t.Fatalf("default: got %v, %v", got, err)
	}
}

func TestBuildOCRAdapter(t *testing.T) {
	if got, err := buildOCRAdapter(&config.Config{OCRAdapter: ""}); got != nil || err != nil {
		t.Fatalf("unset: got %v, %v, want nil, nil", got, err)
	}

	tests := []struct {
		name    string
		adapter string
		want    any
	}{
		{"ollama", "ollama", &ollama.Adapter{}},
		{"anthropic", "anthropic", &anthropic.Adapter{}},
		{"openai", "openai", &openai.Adapter{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildOCRAdapter(&config.Config{OCRAdapter: tt.adapter})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if reflect.TypeOf(got) != reflect.TypeOf(tt.want) {
				t.Fatalf("got %T, want %T", got, tt.want)
			}
		})
	}

	got, err := buildOCRAdapter(&config.Config{OCRAdapter: "bogus"})
	if got != nil || err == nil || !strings.Contains(err.Error(), `unsupported OCR_ADAPTER "bogus"`) {
		t.Fatalf("default: got %v, %v", got, err)
	}
}

func TestBuildChatAdapter(t *testing.T) {
	tests := []struct {
		name    string
		adapter string
		want    any
	}{
		{"anthropic", "anthropic", &anthropic.ChatAdapter{}},
		{"openai", "openai", &openai.ChatAdapter{}},
		{"ollama", "ollama", &ollama.ChatAdapter{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildChatAdapter(&config.Config{CompletionAdapter: tt.adapter})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if reflect.TypeOf(got) != reflect.TypeOf(tt.want) {
				t.Fatalf("got %T, want %T", got, tt.want)
			}
		})
	}

	// Unlike the other six builders, an unrecognized adapter here is NOT an
	// error: it's the documented "no chat adapter configured" case (chat
	// endpoint then returns 503), so buildChatAdapter returns (nil, nil).
	got, err := buildChatAdapter(&config.Config{CompletionAdapter: "bogus"})
	if got != nil || err != nil {
		t.Fatalf("default: got %v, %v, want nil, nil", got, err)
	}
}

func TestBuildMessaging(t *testing.T) {
	tests := []struct {
		name    string
		adapter string
		want    any
	}{
		{"telegram", "telegram", &telegram.Adapter{}},
		{"discord", "discord", &discord.Adapter{}},
		{"matrix", "matrix", &matrix.Adapter{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildMessaging(&config.Config{MessagingAdapter: tt.adapter})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if reflect.TypeOf(got) != reflect.TypeOf(tt.want) {
				t.Fatalf("got %T, want %T", got, tt.want)
			}
		})
	}

	got, err := buildMessaging(&config.Config{MessagingAdapter: "bogus"})
	if got != nil || err == nil || !strings.Contains(err.Error(), `unsupported MESSAGING_ADAPTER "bogus"`) {
		t.Fatalf("default: got %v, %v", got, err)
	}
}

func TestBuildNotifier(t *testing.T) {
	tests := []struct {
		name    string
		adapter string
		want    any
	}{
		{"ntfy", "ntfy", &ntfy.Notifier{}},
		{"gotify", "gotify", &gotify.Notifier{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildNotifier(&config.Config{Notifier: tt.adapter})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if reflect.TypeOf(got) != reflect.TypeOf(tt.want) {
				t.Fatalf("got %T, want %T", got, tt.want)
			}
		})
	}

	got, err := buildNotifier(&config.Config{Notifier: "bogus"})
	if got != nil || err == nil || !strings.Contains(err.Error(), `unsupported NOTIFIER "bogus"`) {
		t.Fatalf("default: got %v, %v", got, err)
	}
}

func TestBuildSources(t *testing.T) {
	t.Run("single openfoodfacts", func(t *testing.T) {
		got, err := buildSources(&config.Config{NutritionSources: []string{"openfoodfacts"}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("len = %d, want 1", len(got))
		}
		if _, ok := got[0].(*openfoodfacts.Source); !ok {
			t.Fatalf("got[0] = %T, want *openfoodfacts.Source", got[0])
		}
	})

	t.Run("taco default data path", func(t *testing.T) {
		got, err := buildSources(&config.Config{NutritionSources: []string{"taco"}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("len = %d, want 1", len(got))
		}
		if _, ok := got[0].(*taco.Source); !ok {
			t.Fatalf("got[0] = %T, want *taco.Source", got[0])
		}
	})

	t.Run("multiple sources all accumulate, not just the last", func(t *testing.T) {
		got, err := buildSources(&config.Config{NutritionSources: []string{"openfoodfacts", "usda", "taco"}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("len = %d, want 3", len(got))
		}
		if _, ok := got[0].(*openfoodfacts.Source); !ok {
			t.Fatalf("got[0] = %T, want *openfoodfacts.Source", got[0])
		}
		if _, ok := got[1].(*usda.Source); !ok {
			t.Fatalf("got[1] = %T, want *usda.Source", got[1])
		}
		if _, ok := got[2].(*taco.Source); !ok {
			t.Fatalf("got[2] = %T, want *taco.Source", got[2])
		}
	})

	t.Run("taco missing bulk file errors", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "does-not-exist.csv")
		got, err := buildSources(&config.Config{NutritionSources: []string{"taco"}, TacoDataPath: missing})
		if got != nil || err == nil {
			t.Fatalf("got %v, %v, want nil, non-nil error", got, err)
		}
	})

	t.Run("unsupported source errors", func(t *testing.T) {
		got, err := buildSources(&config.Config{NutritionSources: []string{"bogus"}})
		if got != nil || err == nil || !strings.Contains(err.Error(), `unsupported NUTRITION_SOURCE "bogus"`) {
			t.Fatalf("got %v, %v", got, err)
		}
	})
}

func TestWriteHealthy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "healthy")
	writeHealthy(path)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	ts, err := time.Parse(time.RFC3339, string(data))
	if err != nil {
		t.Fatalf("file content %q is not RFC3339: %v", data, err)
	}
	if time.Since(ts) > time.Minute {
		t.Errorf("timestamp %v is not recent", ts)
	}
}
