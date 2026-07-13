# DietDaemon

[![CI](https://github.com/gsaraiva2109/dietdaemon/actions/workflows/main.yml/badge.svg)](https://github.com/gsaraiva2109/dietdaemon/actions/workflows/main.yml)
[![Go 1.26](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Self-hosted nutrition and macro tracker. Log meals by sending natural text or voice
to a chat app and get back structured macros, a dashboard, and nudges when daily
targets lag. Fasting, weight, water, workouts, and sleep ride along on the same store
and bot.

**Provider-agnostic, env-driven, feature-flagged.** Runs with zero LLM / zero GPU /
zero API key by default. Intelligence (natural-language parsing, smart matching) is
opt-in via an Ollama sidecar.

## Light by default

| Mode             | Parser                                    | RAM (idle)   | Requirements |
|------------------|-------------------------------------------|--------------|--------------|
| Default (Tier 0) | Deterministic tokenizer + unit dictionary | ~15 to 25 MB | None         |
| AI (Tier 1 to 2) | Embeddings or LLM via Ollama sidecar      | +Ollama      | GPU optional |

Core boots fully headless (API only). The dashboard is behind a feature flag.

## How it works

```
You: "200g chicken breast, 2 eggs, 150g rice"  →  Telegram/Discord/Matrix
                                                    ↓
                                         1. Parse: extract items + quantities
                                         2. Resolve: match against real food DBs
                                         3. Store: meal + macros + audit trail
                                         4. Nudge: if daily target is behind
```

1. **Stage A**, extract food items and quantities from natural language.
2. **Stage B**, resolve macros from a real food database (never from an LLM).
3. **Personal food library**, repeat meals resolve instantly from local cache.
4. **Notifications**, nudged when protein or calories lag behind daily targets.

## Beyond macros

Same bot, same store, same nudging engine, more than food:

- **Fasting** — `/fast start`, `/fast end`, with a running timer on the dashboard.
- **Weight & measurements** — logged over time, charted, trend called out.
- **Water** — logged per day, nudged if you're behind by afternoon and again by evening.
- **Workouts** — logged, nudged if none in 3 days.
- **Sleep** — logged as a range (`/sleep 23:00 07:00`).
- **Progress photos** — stored, compared side by side.

Meal templates and frequent foods turn a repeat meal ("my usual breakfast") into one command
instead of re-typing the whole thing.

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

## Auth

Single-user by default — set `API_AUTH_TOKEN` for a static bearer token, or leave it empty
for no auth on localhost. Set `MULTI_USER=true` to run it for more than one person: email +
password, magic-code login, TOTP, WebAuthn passkeys, and OIDC (Google, Authentik, Keycloak,
anything OIDC-compliant) are all available, with `AUTH_REGISTRATION_MODE` deciding who can sign
up (`invite`, `open`, `oidc-only`). Accounts don't share data — no household/shared-target mode
yet.

## Configuration

All behaviour is driven by environment variables. See `.env.example` for every option.
Key knobs:

| Variable             | Description                                                                 |
|----------------------|-----------------------------------------------------------------------------|
| `MESSAGING_ADAPTER`  | `telegram`, `discord`, `matrix`                                             |
| `PARSER_TIER`        | `0` (deterministic), `1` (embeddings), `2` (LLM)                            |
| `NUTRITION_SOURCE`   | Comma-separated: `openfoodfacts,taco,usda`                                  |
| `EMBED_ADAPTER`      | `ollama` only, backs food-matching embeddings                               |
| `COMPLETION_ADAPTER` | `ollama`, `anthropic`, `openai` (opt-in), backs Tier-2 parsing + `/suggest` |
| `ENABLE_STT`         | `true`, enables speech-to-text for audio messages                           |
| `WHISPER_URL`        | Whisper.cpp HTTP server URL                                                 |
| `NOTIFIER`           | `ntfy`, `gotify`                                                            |
| `DEFAULT_TIMEZONE`   | IANA timezone for daily rollup boundaries                                   |
| `MULTI_USER`         | `true` to allow more than one account                                       |

## Parser tiers vs. STT

Two knobs that look related but aren't: parser tier decides how *text* becomes food items,
STT decides how *voice* becomes text before the parser ever sees it. STT runs fine on Tier 0 —
`ENABLE_STT` + `WHISPER_URL` and `PARSER_TIER` are independent settings. See
[docs/STT.md](docs/STT.md) for setup and troubleshooting.

| Tier | Buys you                                          | Needs                  |
|------|---------------------------------------------------|------------------------|
| 0    | Deterministic tokenizer + unit dictionary         | Nothing                |
| 1    | Fuzzy/typo-tolerant matching, alias auto-learning | Ollama + `EMBED_MODEL` |
| 2    | Free-form / ambiguous phrasing via LLM            | Ollama + `LLM_MODEL`   |

## Architecture

Modular monolith with clean internal interfaces. Adapters translate to/from provider
formats, core never imports a provider SDK.

```
[Messaging Adapter] → [Ingest] → [Parse pipeline] → [Store (SQLite)]
      telegram                   deterministic        meals, fasting, weight
      discord                    ollama (opt)         water, workouts, sleep
      matrix                                          food library, targets
                                          ↓            pending meals (durable)
                              [Scheduler] → [Notifier]
                                                ntfy / gotify
                              [STT (opt)] → Whisper
                              [Auth (opt)] → email / OIDC / passkeys, multi-user
                              [Dashboard (opt)]
```

## License

MIT, see [LICENSE](LICENSE).
