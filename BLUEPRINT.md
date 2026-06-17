# DietDaemon — Technical Blueprint (v0)

## Context

A self-hosted, low-friction nutrition/macro tracker for a homelab. Owner is bulking and
misses meals due to high food volume; needs to log intake by sending natural text/voice
to a chat app and get back structured macros, a dashboard, and nudges when daily targets
lag. Built **open-source from day one**: provider-agnostic adapters, env-driven config,
feature-flagged subsystems, easy for the self-hosted community to run and extend.

Core design tension resolved during planning: keep the **always-on footprint tiny and the
system fully usable with zero LLM / zero GPU / zero API key**, while letting "intelligence"
(natural-language parsing, smart multilingual matching) be an **opt-in enhancement**.

### Locked decisions
- **Language:** Go for the core (single static binary, ~15–25 MB idle RAM, trivial Docker/cross-compile, low contributor barrier).
- **Topology:** Modular monolith. Clean internal interfaces, in-memory queue. One container by default; bus swappable later behind the same interface.
- **Tenancy:** Single-user first, but **schema keyed by `user_id`** throughout. Auth/multi-user is a later feature flag, no rewrite needed.
- **Parsing:** Tiered behind one `Parser` interface. Tier-0 deterministic (default, no model) → Tier-1 embeddings (opt-in) → Tier-2 generative LLM (opt-in).
- **Nutrition accuracy:** Two-stage. Stage A extracts food items + quantities. Stage B resolves macros from a **real food DB** (numbers never come from an LLM). Reproducible + auditable.
- **Models:** Optional **Ollama sidecar** via docker-compose `ai` profile, serving both embeddings and LLM over HTTP behind adapters. Disabled by default.
- **Offline multilingual matching:** unaccent + trigram/Levenshtein fuzzy match against a multilingual alias index (Open Food Facts + TACO, PT & EN). Embeddings upgrade this only when the sidecar is on.

## Architecture overview

```
[Messaging Adapter] --InboundMessage--> [Ingest] --> [in-mem queue]
   (Telegram/Discord/Matrix)                               |
                                                           v
[STT Adapter (opt)] <--audio--                       [Parse pipeline]
                                                     Stage A: Parser (Tier 0/1/2)
                                                     Stage B: Nutrition Resolver (food DB)
                                                           |
                                          (low confidence?) --> Clarification loop (back via channel)
                                                           v
                                                     [Store: meals, items, rollups]
                                                           |
                                            +--------------+--------------+
                                            v                             v
                                     [Scheduler/Rules]              [Dashboard (opt)]
                                            |                        SSR (HTMX)
                                     [Notifier Adapter]
                                     (ntfy/Gotify/webhook)
```

Everything between adapters speaks **canonical core types** — adapters only translate to/from
provider formats. Core never imports a provider SDK directly.

## Core contracts (interfaces)

Define in a `core/ports` package; adapters depend on core, never the reverse.

- `MessagingAdapter` — `Receive() <-chan InboundMessage`, `Send(Reply)`. Impls: Telegram (baseline), Discord, Matrix.
- `STTProvider` (opt) — `Transcribe(audio) (text, lang)`. Impls: Whisper (local), API.
- `Parser` — `Extract(text, locale) ([]ParsedItem, confidence)`. Impls: `deterministic` (Tier 0), `embedding` (Tier 1, via model adapter), `llm` (Tier 2, via model adapter).
- `NutritionSource` — `Resolve(ParsedItem) (FoodMatch, macros)`. Impls: local cache, Open Food Facts, USDA FDC, TACO.
- `ModelAdapter` (opt) — `Embed(text) []float32`, `Complete(prompt) text`. Impl: Ollama HTTP. Used by `embedding`/`llm` parsers.
- `Notifier` — `Notify(Notification)`. Impls: ntfy (baseline), Gotify, webhook.

### Canonical types (sketch)
- `InboundMessage{ UserID, At, Kind(text|audio|image), Payload, ChannelMeta }`
- `ParsedItem{ RawPhrase, Quantity, Unit, NormalizedGrams, Locale }`
- `Meal{ ID, UserID, At(UTC), RawText, []ResolvedItem, Confidence, ParserTier, Source }`
- `Notification{ UserID, Title, Body, Priority }`

## Parse pipeline detail

1. **Stage A — extract items+quantities** via configured `Parser` tier:
   - Tier 0: tokenizer + per-locale unit dictionary (`g, kg, ml, colher, xícara, cup, tbsp…`) → `(qty, unit, food-phrase)` chunks. Handles disciplined shorthand in any DB-covered language.
   - Tier 1: embed food-phrase, nearest-neighbor against food-DB embedding index (cross-language, typo-tolerant).
   - Tier 2: LLM untangles messy prose into discrete items+quantities.
2. **Unit normalization** → canonical grams (shared utility, reused by all tiers).
3. **Stage B — resolve macros** via `NutritionSource`, **local-first**:
   - **Look up the personal food library (local DB) first.** Only on a miss fall back to external sources (OFF/USDA/TACO), then write the result back to the library.
   - Track **per-user query frequency** per food; rank matches by frequency so repeat meals resolve instantly. Since a bulking diet repeats the same foods daily, after a few days nearly every lookup is served offline, free, and fast — external APIs become the rare cold-start path.
   - Match via fuzzy alias index by default; embedding NN if Tier 1 on.
4. **Confidence gate:** below threshold → **clarification loop**: bot asks back through the same channel; pending-meal state stored short-lived until user confirms. Never silently guess.
5. Persist `Meal` with raw text, parsed result, confidence, parser tier, source (full audit trail).

## Storage

- Single embedded/SQL store (start with SQLite for zero-dep self-host; Postgres optional via config). All tables keyed by `user_id`.
- Tables: `users`, `meals`, `resolved_items`, `food_library` (personal cache: food entry + macros + source + `query_count` + `last_used`, keyed by user_id), `daily_targets`, `daily_rollups` (materialized per user-day).
- **Personal food library** is the local-first cache from Stage B — the more you log, the less the system ever touches an external API.
- **Timezone:** store all timestamps UTC. The aggregation/"daily" timezone is set via a **docker-compose env var** (`DEFAULT_TIMEZONE`, e.g. `America/Sao_Paulo`), overridable per-user once multi-user lands. Required for correct nudges/rollups.

## Notifications + scheduler

- Scheduler is a real component, not just an adapter: per-user rules (`if local 18:00 and protein < 60% → nudge`), cron tick, **dedupe** so it never spams.
- `Notifier` adapter sends the canonical `Notification`. ntfy baseline.

## Frontend — DEFERRED (planned separately)

Build and fully test the **backend first**, then plan the frontend as its own dedicated effort.
The frontend is intended to be robust and polished, so it deserves a separate design pass — not
a rushed afterthought bolted onto this blueprint.

Constraints to honor when it's planned later:
- Optional behind a feature flag; **core must boot fully headless** (API only) without it.
- Backend exposes a clean read API (rollups vs targets, recent meals, manual correction) so any
  frontend stack can sit on top. No frontend decisions are baked into the backend.

## Extensibility / OSS ergonomics

- Adapter selection env-driven: `MESSAGING_ADAPTER=telegram`, `PARSER_TIER=0`, `NUTRITION_SOURCE=openfoodfacts,taco`, `MODEL_ADAPTER=ollama`, `NOTIFIER=ntfy`, etc.
- All secrets/regional settings in `.env`. **Boot-time config validation** fails fast with clear messages.
- **Feature flags** gate whole subsystems (dashboard, notifications, STT, each parser tier).
- **Contract test suite** per port: a contributor runs it against a new adapter; green = compliant. Lowers contribution friction.
- docker-compose profiles: default (core only) and `ai` (adds Ollama for embeddings+LLM).

## Implementation phases (suggested order)

**Backend first; frontend is a separate effort planned after the backend works and is tested.**

**Model routing legend** — to save cost: `[opus]` = design/judgment-heavy, keep on Opus; `[deepseek]`
= mechanical boilerplate, route to DeepSeek; `[you]` = manual shell/account/data work, no LLM needed.
General rule: **Opus designs the contract + the hard algorithm; DeepSeek fills implementations behind it.**

0. **Repo bootstrap:**
   - `[you]` `go mod init`, dir skeleton, `.gitignore`, `git` plumbing.
   - `[deepseek]` root `DEFERRED.md`, `.env.example`, `Dockerfile`, `docker-compose.yml` (default + `ai` profiles). Static, follows this blueprint.
1. **Skeleton + contracts:**
   - `[opus]` `core/ports` interfaces + canonical types — wrong here = costly rework. **Design boundary.**
   - `[opus]` config loader + validation (incl. `DEFAULT_TIMEZONE`), in-mem queue.
   - `[deepseek]` SQLite store CRUD + migrations, once schema/interfaces are fixed by Opus.
2. **Vertical slice (no AI):**
   - `[you]` create Telegram bot token, ntfy topic, download OFF/TACO datasets.
   - `[deepseek]` Telegram adapter + ntfy notifier (HTTP plumbing behind fixed interfaces); dataset import code.
   - `[opus]` Tier-0 parser grammar/tokenizer + local-first resolver ranking + confidence gate.
3. **Rollups + scheduler:**
   - `[opus]` scheduler dedupe + timezone-correct rollups (judgment).
   - `[deepseek]` daily-targets CRUD, rule config plumbing.
4. **Clarification loop:** `[opus]` confidence gate + pending-meal conversational state.
5. **AI tiers (opt-in):**
   - `[you]` `docker pull ollama`, `ollama pull` models.
   - `[deepseek]` Ollama `ModelAdapter` HTTP client (mechanical).
   - `[opus]` Tier-1 embedding matcher + Tier-2 LLM prompt/extraction logic.
6. **STT (opt-in):** `[deepseek]` Whisper adapter; `[opus]` only if confidence/lang-detect logic needed.
7. **Backend polish + release:** `[deepseek]` contract test scaffolding, example `.env`, README draft, Discord/Gotify reference adapters; `[opus]` review.
8. **Frontend — separate planning pass** (only after backend is validated): design + build the robust UI on top of the read API.

## Verification

- **Per-phase:** unit tests on unit-normalization + Tier-0 grammar (PT & EN fixtures: `"200g frango, 2 ovos"` and `"200g chicken, 2 eggs"` resolve to equal macros).
- **Contract tests:** run the port test suite against each adapter impl.
- **End-to-end (Phase 2):** spin core via docker-compose (default profile, no AI), send a Telegram message, assert a `Meal` row with correct macros from the food DB and an ntfy notification on target miss.
- **AI path (Phase 5):** enable `ai` profile, send messy multilingual prose, assert Tier-1/2 extraction + correct DB-grounded macros; confirm core still works with profile off.
- **Footprint check:** confirm idle core RAM ~15–25 MB with `ai` profile off (validates the light-by-default promise).

## Deferred items — tracked in root `DEFERRED.md`

Phase 0 creates a **`DEFERRED.md` at the repo root** so these are not forgotten. Seed contents:
- **Frontend:** full robust UI, planned as its own effort after backend is built + tested.
- **Food DBs:** exact DBs to bundle vs fetch-on-demand; license check (OFF = ODbL, TACO terms).
- **Embedding index storage:** in-DB vs sidecar vector store, decided once Tier-1 is built.
- **Auth / multi-user:** mechanism to add when the multi-user flag is eventually enabled.
- **Per-user timezone override:** lands with multi-user (env `DEFAULT_TIMEZONE` covers single-user now).

Keep `DEFERRED.md` updated as new "later" items surface during the build.
