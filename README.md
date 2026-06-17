# DietDaemon

Self-hosted nutrition and macro tracker. Log meals by sending natural text or voice
to a chat app and get back structured macros, a dashboard, and nudges when daily
targets lag.

**Provider-agnostic, env-driven, feature-flagged.** Runs with zero LLM / zero GPU /
zero API key by default. Intelligence (natural-language parsing, smart matching) is
opt-in via an Ollama sidecar.

## Light by default

| Mode | Parser | RAM (idle) | Requirements |
|---|---|---|---|
| Default (Tier 0) | Deterministic tokenizer + unit dictionary | ~15–25 MB | None |
| AI (Tier 1–2) | Embeddings or LLM via Ollama sidecar | +Ollama | GPU optional |

Core boots fully headless (API only). The dashboard is behind a feature flag.

## How it works

```
You: "200g frango, 2 ovos, 150g arroz"  →  Telegram/Discord/Matrix
                                                    ↓
                                         1. Parse: extract items + quantities
                                         2. Resolve: match against real food DBs
                                         3. Store: meal + macros + audit trail
                                         4. Nudge: if daily target is behind
```

1. **Stage A** — extract food items and quantities from natural language.
2. **Stage B** — resolve macros from a real food database (never from an LLM).
3. **Personal food library** — repeat meals resolve instantly from local cache.
4. **Notifications** — nudged when protein or calories lag behind daily targets.

## Quick start

```bash
cp .env.example .env
# Edit .env with your Telegram bot token, ntfy URL, etc.

docker compose up -d
```

For AI-powered parsing (optional):

```bash
docker compose --profile ai up -d
```

## Configuration

All behaviour is driven by environment variables. See `.env.example` for every option.
Key knobs:

| Variable | Description |
|---|---|
| `MESSAGING_ADAPTER` | `telegram`, `discord`, `matrix` |
| `PARSER_TIER` | `0` (deterministic), `1` (embeddings), `2` (LLM) |
| `NUTRITION_SOURCE` | Comma-separated: `openfoodfacts,taco` |
| `NOTIFIER` | `ntfy`, `gotify`, `webhook` |
| `DEFAULT_TIMEZONE` | IANA timezone for daily rollup boundaries |

## Architecture

Modular monolith with clean internal interfaces. Adapters translate to/from provider
formats — core never imports a provider SDK.

```
[Messaging Adapter] → [Ingest] → [Parse pipeline] → [Store]
                                          ↓
                              [Scheduler] → [Notifier]
                              [Dashboard (opt)]
```

Deferred decisions are tracked in [`DEFERRED.md`](DEFERRED.md).
Full architecture and implementation plan in [`BLUEPRINT.md`](BLUEPRINT.md).

## License

MIT — see [LICENSE](LICENSE).
