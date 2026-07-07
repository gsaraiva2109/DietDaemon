package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/gsaraiva2109/dietdaemon/adapters/model/anthropic"
	"github.com/gsaraiva2109/dietdaemon/adapters/model/openai"
	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/internal/auth"
)

// decryptAIKey base64-decodes and then decrypts the stored AI key ciphertext.
func decryptAIKey(encKey string, key []byte) ([]byte, error) {
	ct, err := base64.RawStdEncoding.DecodeString(encKey)
	if err != nil {
		return nil, fmt.Errorf("byok: base64 decode: %w", err)
	}
	plaintext, err := auth.Decrypt(ct, key)
	if err != nil {
		return nil, fmt.Errorf("byok: decrypt: %w", err)
	}
	return plaintext, nil
}

// buildAdapterForProvider mirrors buildCompletionAdapter in main.go but takes an
// explicit API key instead of reading it from config. Only handles anthropic/openai
// — ollama is a self-hosted adapter with no per-user key concept.
func buildAdapterForProvider(provider, apiKey, anthropicModel, openaiModel, openaiBaseURL string, timeout time.Duration) (ports.ModelAdapter, error) {
	switch provider {
	case "anthropic":
		return anthropic.New(apiKey, anthropicModel, timeout), nil
	case "openai":
		return openai.New(openaiBaseURL, apiKey, openaiModel, timeout), nil
	default:
		return nil, fmt.Errorf("unsupported BYOK provider %q", provider)
	}
}

// injectModelOverride checks AI_KEY_MODE, looks up the user's stored key, decrypts
// it, builds a per-user adapter, and injects it into ctx. Returns ctx unchanged
// when BYOK is disabled or the user has no key.
func (h *Handler) injectModelOverride(ctx context.Context, userID string) context.Context {
	if h.cfg == nil || h.cfg.AIKeyMode != "byok" {
		return ctx
	}
	provider, encKey, found, err := h.store.GetUserAIKey(ctx, userID)
	if err != nil || !found {
		return ctx
	}
	plaintext, err := decryptAIKey(encKey, h.cfg.AIKeyEncKey)
	if err != nil {
		return ctx
	}
	adapter, err := buildAdapterForProvider(
		provider,
		string(plaintext),
		h.cfg.AnthropicModel,
		h.cfg.OpenAIModel,
		h.cfg.OpenAIBaseURL,
		h.cfg.ModelTimeout,
	)
	if err != nil {
		return ctx
	}
	return ports.WithModelOverride(ctx, adapter)
}
