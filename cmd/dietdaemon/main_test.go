package main

import (
	"reflect"
	"testing"

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
