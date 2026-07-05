# Roadmap

Working list of features selected for near-term planning (2026-07-05). Each entry is a
one-liner + why; sizing/design happens when picked up for implementation.

## Committed features

1. **Alias review UI** — surface embedding-matched food aliases that got auto-written to the
   personal library (`ALIAS_WRITE_BACK_THRESHOLD`, default 0.92, see
   `internal/resolver/resolver.go`) so the user can confirm or reject them instead of a silent
   write-back. No undo path exists today.
2. **Recipe / multi-ingredient composition** — save a combo of foods (e.g. "chicken + rice +
   broccoli, my usual") as one loggable unit, distinct from the existing single-food templates
   (`internal/commands/template.go`).
3. **Weekly/monthly digest notification** — one scheduled nudge summarizing trend (avg protein,
   weight delta, adherence) instead of only the existing per-day macro/health nudges
   (`internal/scheduler/rules.go`).
4. **Health platform import/export** — Apple Health / Google Fit / Garmin sync for weight and
   workout data, since those trackers already exist in-app (`weight.go`, `workout.go`).
5. **Configurable nudge rules** — today's rules are hardcoded constants in
   `scheduler.DefaultRules()` / `DefaultHealthRules()` (fixed hour, fixed macro, fixed
   threshold). Make hour/macro/threshold per-user configurable instead of one-size-fits-all.
6. **Scheduled data export/backup** — automatic periodic dump (SQLite file copy or CSV/JSON) to
   a configured location, complementing the existing on-demand `ExportModal.tsx`.
7. **Precedence UI** — let the user reorder `NUTRITION_SOURCE` per-account instead of fixed
   `.env` order. `resolver.New()` already queries `sources []Source` in order, first match wins
   (`internal/resolver/resolver.go:66`); this exposes that ordering as a setting.
8. **CorrectMealItem via bot** — `CorrectMealItem` (fixes a misparsed meal item) exists only as a
   REST endpoint today (`internal/store/store.go:694`, `internal/api/handler.go:565`), reachable
   only through the web dashboard. Add a bot command (e.g. `/correct`) so misparses can be fixed
   without `ENABLE_DASHBOARD=true`.

## Large features (size/plan separately, not part of the batch above)

_None currently._

## Implementation groups

Grouped by theme, planned/sized/shipped as batches rather than one at a time.

**Group 2 — Food logging & resolution** (all touch `internal/resolver` or the meal-correction
path)
- Alias review UI (1)
- Precedence UI (7)
- Recipe / multi-ingredient composition (2)
- CorrectMealItem via bot (8)

**Group 3 — Scheduler & data ops** (recurring background jobs, outside the immediate
log-a-meal flow)
- Weekly/monthly digest notification (3)
- Configurable nudge rules (5)
- Health platform import/export (4)
- Scheduled data export/backup (6)

## Under consideration (not committed, revisit later)

- **Family/household multi-user sharing** — shared targets or a shared fridge/food library
  across accounts. Auth already supports multi-user (OIDC, invite mode); this would be a
  data-model layer on top (shared vs private meals/targets per household).
- **Photo storage policy** — where/how progress photos (`PhotoGrid.tsx`, `PhotoCompare.tsx`) are
  stored, size limits, retention. No decision made yet; needs its own design pass before
  implementation (storage backend, retention, whether it survives `MULTI_USER` account deletion).

## Docs work

- **README rewrite** — current README only pitches macro tracking. Needs to mention the full
  tracker suite (fasting/weight/water/workout/sleep/photos), the auth options (single-user vs
  multi-user/OIDC), and ideally the parser-tier table below.

## Parser tier requirements (reference for the README rewrite)

Two independent axes get conflated in the current docs: **parser tier** (how text is turned into
food items) and **STT** (how audio becomes text before parsing even starts).

| Feature                                      | Requires | Notes                                                                                                                                                                                                                                  |
|----------------------------------------------|----------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Deterministic meal parse                     | Tier 0   | Default. No model, no Ollama.                                                                                                                                                                                                          |
| Fuzzy/typo-tolerant food matching            | Tier 1   | Embedding nearest-neighbour (`internal/resolver/embedding`), needs Ollama + `EMBED_MODEL`.                                                                                                                                             |
| Alias auto-write-back                        | Tier 1   | Rides on the Tier 1 matcher; controlled by `ALIAS_WRITE_BACK_THRESHOLD`.                                                                                                                                                               |
| Free-form / ambiguous natural-language parse | Tier 2   | LLM via Ollama + `LLM_MODEL`.                                                                                                                                                                                                          |
| Voice message logging (STT)                  | Any tier | Whisper.cpp (`adapters/stt/whisper`) transcribes audio → text, then the transcript goes through whichever `PARSER_TIER` is configured. Works fine on Tier 0; STT and parser tier are independent knobs (`ENABLE_STT` + `WHISPER_URL`). |
| Dashboard                                    | N/A      | Independent of parser tier; gated by `ENABLE_DASHBOARD`.                                                                                                                                                                               |
| Multi-user / OIDC / passkeys                 | N/A      | Independent of parser tier; gated by `MULTI_USER` / `OIDC_PROVIDERS`.                                                                                                                                                                  |
