# AI Chat Assistant

DietDaemon includes a conversational AI assistant exposed over an HTTP/SSE API.
Users type free-form chat ("what should I eat, I skipped breakfast"), the assistant
streams a reply token-by-token, and can invoke any DietDaemon slash-command as a
tool (log meals, `/suggest`, `/status`, etc.) via LLM tool-calling.

The frontend (built separately, using `assistant-ui`) fetches the SSE endpoint
directly and parses the event stream client-side.

## How it works

```
User text → POST /api/v1/chat/sessions/{id}/messages
                ↓
         [Assistant Router]
         (tool-calling loop, max 6 rounds)
                ↓
         [ChatAdapter] ──→ Anthropic / OpenAI / Ollama
                ↓
         SSE event stream → Frontend renders token-by-token
```

1. Frontend creates a session via `POST /api/v1/chat/sessions`, gets back an `{id}`.
2. Each user message is `POST`ed to `/api/v1/chat/sessions/{id}/messages`.
3. The handler builds a localized system prompt (i18n base + user's custom instructions),
   loads the conversation history from the DB, and calls the assistant router.
4. The router streams tokens from the LLM in real-time via SSE. When the model calls a
   tool, the router executes the corresponding DietDaemon command directly and feeds the
   result back into the conversation.
5. All messages are persisted: user messages before streaming, assistant + tool messages
   on completion.

## API endpoints

All endpoints require authentication (session cookie or bearer key).

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/chat/sessions` | Create a new chat session. Optional body: `{"title": "..."}`. Returns `{"id": "..."}`. |
| `GET` | `/api/v1/chat/sessions` | List all sessions for the authenticated user, newest first. |
| `POST` | `/api/v1/chat/sessions/{id}/messages` | Send a message and stream the response via SSE. Body: `{"text": "..."}`. |
| `GET` | `/api/v1/chat/sessions/{id}/messages` | Get message history for a session. |
| `GET` | `/api/v1/chat/settings` | Get the user's custom assistant instructions. |
| `PUT` | `/api/v1/chat/settings` | Set custom instructions. Body: `{"custom_instructions": "..."}`. |

## SSE event catalog

The `POST .../messages` endpoint returns `Content-Type: text/event-stream`. The wire
format is a simple custom protocol (not Vercel AI SDK, not any third-party protocol).

```
event: delta
data: {"text": "partial token(s)"}

event: tool-call
data: {"id": "tc_1", "name": "suggest", "args": ""}

event: tool-result
data: {"id": "tc_1", "name": "suggest", "text": "<command output>"}

event: done
data: {}

event: error
data: {"message": "..."}
```

- **delta** — a single token or short phrase from the model. The frontend appends these
  as they arrive to render streaming text.
- **tool-call** — the model requested a command execution. The frontend may show a
  loading indicator for the tool name.
- **tool-result** — the command finished. `text` contains the command's reply text, which
  the model will use to continue its response.
- **done** — the conversation turn is complete (either the model finished speaking, or
  the tool-calling loop exhausted its rounds).
- **error** — something went wrong. The message is user-facing.

## Provider support

The assistant uses the same `COMPLETION_ADAPTER` config value as the meal-parsing
pipeline. All three providers are supported.

### Anthropic

Set `COMPLETION_ADAPTER=anthropic` with `ANTHROPIC_API_KEY` and `ANTHROPIC_MODEL`.
Uses Anthropic's native streaming Messages API (`/v1/messages`) with SSE and native
tool-calling. Claude models (Sonnet, Opus, Haiku) all support tool-calling.

### OpenAI

Set `COMPLETION_ADAPTER=openai` with `OPENAI_API_KEY`, `OPENAI_MODEL`, and optionally
`OPENAI_BASE_URL` (for OpenAI-compatible proxies). Uses the streaming chat/completions
API with SSE. GPT-4o, GPT-4.1, and GPT-4-turbo support tool-calling. Also works with
OpenAI-compatible backends (vLLM, LiteLLM) that implement the `/chat/completions`
streaming endpoint with `tools`.

### Ollama (self-hosted)

Set `COMPLETION_ADAPTER=ollama` with `OLLAMA_URL` and `LLM_MODEL` (e.g. `llama3.1`).
Uses Ollama's `/api/chat` endpoint with NDJSON streaming (one JSON object per line —
different wire format than the SSE providers, handled transparently inside the adapter).

**Tool-calling is model-dependent.** Ollama passes `tools` to the model, but the model
must actually support tool-calling. Known-good models:

- `llama3.1` (8B, 70B)
- `qwen2.5` (7B, 14B, 32B, 72B)
- `mistral-nemo` (12B)
- `firefunction-v2`

Models that **don't** support tool-calling (older or smaller models like `llama3`,
`phi3`, `gemma2`) will silently ignore the `tools` field and respond with plain
conversation — no error, no tool calls. The conversation degrades gracefully: the
assistant still responds, but it won't execute commands or look up real data. This
is the expected behavior; choose a tool-calling-capable model if you want the full
assistant experience.

## System prompt

The system prompt is built per-request from two sources:

1. **i18n base prompt** — resolved from the locale bundle key `assistant.system_prompt`
   in the user's language (falls back to English). This encodes hard-to-override rules:
   always call tools for real data, never fabricate nutrition numbers, don't execute
   mutating tools unless explicitly asked.
2. **Custom instructions** — per-user text set via `PUT /api/v1/chat/settings`. Appended
   after the base prompt with a blank-line separator.

The user's locale is read from their profile (`users.locale`). If unset, defaults to `en`.

## BYOK (Bring Your Own Key)

When `AI_KEY_MODE=byok`, users can set their own Anthropic or OpenAI API key via the
existing settings endpoints (`GET/PUT /api/v1/settings/ai-key`). The chat assistant
picks up the per-user key and builds a user-scoped adapter, falling back to the
boot-configured provider for ollama (which is self-hosted and has no per-user key).

## Configuration

No new environment variables. The assistant reuses existing config:

| Variable | Used for |
|----------|----------|
| `COMPLETION_ADAPTER` | Selects the provider (`anthropic`, `openai`, `ollama`) |
| `ANTHROPIC_API_KEY` | Anthropic API key |
| `ANTHROPIC_MODEL` | Anthropic model (e.g. `claude-sonnet-4-5`) |
| `OPENAI_API_KEY` | OpenAI API key |
| `OPENAI_MODEL` | OpenAI model (e.g. `gpt-4o`) |
| `OPENAI_BASE_URL` | OpenAI-compatible base URL (default: `https://api.openai.com/v1`) |
| `OLLAMA_URL` | Ollama base URL (default: `http://localhost:11434`) |
| `LLM_MODEL` | Ollama chat model (e.g. `llama3.1`) |
| `MODEL_TIMEOUT` | HTTP client timeout for LLM requests |
| `AI_KEY_MODE` | Set to `byok` to enable per-user API keys |
| `AI_KEY_ENC_KEY` | 32-byte hex key for encrypting stored BYOK keys |
