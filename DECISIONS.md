# DietDaemon — Architecture Decisions

This document details the architectural decisions, rationales, implementation notes, and ownership for the open design questions of DietDaemon.

---

## 1. Nutrition Food-DB Delivery + Licensing

### Decision
* **Open Food Facts (OFF)**: Queried on-demand via the public HTTP API. Compliance attribution is placed in the project `README.md`, `ATTRIBUTION.md`, the dashboard footer, and the chat bot's `/about` command response.
* **TACO (Tabela Brasileira de Composição de Alimentos)**: The default TACO dataset (`taco.csv`) is pre-processed and embedded directly into the Go binary using `go:embed`. An optional file path override can be provided via `TACO_DATA_PATH`.

### Rationale
* **OFF**: At over 3 million products, the OFF database is too large for homelab resources. Fetch-on-demand keeps the Docker image tiny and requires no local storage.
* **TACO**: The TACO dataset is small (~1000 items, ~150KB CSV), making it trivial to embed. Embedding guarantees 100% offline availability, removes network latency, and requires zero setup for Brazilian food tracking.
* **Licensing**: OFF is licensed under ODbL 1.0, requiring attribution and share-alike terms for modifications (none are published). TACO allows redistribution with attribution.

### Implementation Notes
* Create `ATTRIBUTION.md` at the repo root with the license and credit texts.
* Update `adapters/nutrition/taco/taco.go` to support `go:embed`:
  ```go
  //go:embed taco.csv
  var defaultTacoCSV []byte
  ```
  If `TACO_DATA_PATH` is empty, parse `defaultTacoCSV`.
* **Config Keys**:
  * `TACO_DATA_PATH` (default: `""` - falls back to embedded)
  * `NUTRITION_OFF_API_URL` (default: `https://world.openfoodfacts.org`)

### Owner
* `[you]` / `[deepseek]` (you supply `taco.csv` and `ATTRIBUTION.md`, deepseek writes the embed code).

---

## 2. Nutrition Source Precedence + Conflict Resolution

### Decision
* **Query Precedence**:
  1. **Local Food Library (Personal Cache)**: Always checked first (`LookupFood`).
  2. **Local/Embedded Databases**: TACO (prioritized for Portuguese/Brazilian locale) or USDA (prioritized for English locale).
  3. **External Web APIs**: Open Food Facts (OFF) queried last.
* **Default Configured Order**: `NUTRITION_SOURCE=taco,usda,openfoodfacts` (resolves TACO first for local high-quality PT entries).
* **Conflict Resolution**:
  * **Local Library Wins**: Once a food is written to the user's local library and associated with an alias, any future match to that alias resolves directly to the library macros.
  * **External Tie-Break**: Configured order in `NUTRITION_SOURCE` breaks ties (first match wins).

### Rationale
* Querying local databases before web APIs avoids unnecessary network calls and respects structured data over crowd-sourced OFF entries.
* The local personal food library is the user's single source of truth, allowing custom corrections to override any global source.

### Implementation Notes
* In `internal/resolver/resolver.go`, the default configured source order is parsed and instantiated in order.
* No changes to `Resolver` interface required; the configuration loader dictates the order.

### Owner
* `[deepseek]`

---

## 3. Read API Contract (Headless Backend)

### Decision
* **Protocol**: REST/JSON over HTTP using the Go standard library `net/http`.
* **Gate**: `ENABLE_DASHBOARD=true` (default: `false`).
* **Endpoints**:
  * `GET /api/v1/rollups/today`: Returns today's macro rollup.
  * `GET /api/v1/rollups/range?start=YYYY-MM-DD&end=YYYY-MM-DD`: Returns daily rollups for charts.
  * `GET /api/v1/meals?limit=N`: Returns a list of recent meals.
  * `GET /api/v1/meals/:id`: Returns details of a specific meal.
  * `POST /api/v1/meals/:id/items/:item_id/correct`: Corrects a resolved item's macros, updating both the meal record, the daily rollup, and the user's `food_library` cache entry for future logs.
  * `POST /api/v1/meals/log`: Submits a raw text string to the parser pipeline for standard processing and logging.

### Rationale
* The standard library `net/http` keeps the static binary lightweight (~15MB RAM) and CGO-free.
* Item correction must propagate to the `food_library` to avoid forcing the user to correct the same food entry repeatedly.

### Implementation Notes
* Register routes under a new `internal/dashboard/` or `internal/api/` package.
* Use `http.ServeMux` for routing.
* Reuse canonical types from `core/types` directly in JSON serialization.

### Owner
* `[deepseek]`

---

## 4. Auth / Multi-User

### Decision
* **Single-User Mode** (`MULTI_USER=false`, default):
  * All chat messages from any source ID map to `"default"`.
  * The REST API bypasses token verification if accessed locally, or validates against a static `API_AUTH_TOKEN` if provided.
* **Multi-User Mode** (`MULTI_USER=true`):
  * **REST API**: Simple Bearer Token authentication (`Authorization: Bearer <token>`).
  * **Inbound Messages**: Map messaging channel IDs (e.g. Telegram chat integer IDs) to a clean internal string `user_id` using a database mapping table.

### Rationale
* Single-user mode requires zero authentication setup, matching standard homelab expectations.
* Database mapping tables decouple messaging channel details from core logic, permitting multi-user setups to span multiple platforms.

### Implementation Notes
* Create database migration `005_auth.sql`:
  ```sql
  CREATE TABLE IF NOT EXISTS api_tokens (
      token      TEXT PRIMARY KEY,
      user_id    TEXT NOT NULL REFERENCES users(id),
      created_at TEXT NOT NULL
  );
  CREATE TABLE IF NOT EXISTS user_channels (
      channel         TEXT NOT NULL,
      channel_user_id TEXT NOT NULL,
      user_id         TEXT NOT NULL REFERENCES users(id),
      PRIMARY KEY (channel, channel_user_id)
  );
  ```
* **Config Keys**:
  * `MULTI_USER` (boolean, default: `false`)
  * `API_AUTH_TOKEN` (string, default: `""`)

### Owner
* `[deepseek]` (mechanical SQL and middleware); `[you]` (manual token provision / DB seeding).

---

## 5. Per-User Timezone Override

### Decision
* Store user-specific timezones in the existing `users.timezone` column.
* Precedence: Use `users.timezone` if set; otherwise, fall back to `DEFAULT_TIMEZONE` env config.
* CLI/Chat Command: `/timezone <IANA_name>` (e.g., `/timezone America/Sao_Paulo`) updates the user's timezone database record.

### Rationale
* Pre-existing column is already wired in `types.User` and database tables.
* A chat command gives users a self-service way to correct timezones without direct database manipulation.

### Implementation Notes
* Wire `/timezone` into the deterministic parser and commands router.
* Validate the timezone name with `time.LoadLocation` before saving.

### Owner
* `[deepseek]`

---

## 6. Embedding Match Policy (Phase 5 Tuning)

### Decision
* (a) **Text to Embed**: **Canonical name only** (as currently implemented).
* (b) **Provisional Thresholds**:
  * `EMBED_MATCH_THRESHOLD` = `0.80`
  * `ALIAS_WRITE_BACK_THRESHOLD` = `0.92`
* **Tuning Method**:
  * Build a CLI utility `cmd/tune/main.go` that runs a benchmark suite of typical messy food inputs (`fixtures/test_phrases.json`) against a set of target food names to calculate precision/recall across a range of thresholds.

### Rationale
* Embedding the canonical name only ensures semantic clarity and avoids polluting vectors with noisy aliases. Exact aliases are matched fast via the exact local cache lookup anyway.
* The thresholds of `0.80` and `0.92` provide a safe middle ground to balance search recall and auto-caching.

### Implementation Notes
* No changes to `internal/resolver/embedding/embedding.go`.
* **Config Keys**:
  * `EMBED_MATCH_THRESHOLD` (float, default: `0.80`)
  * `ALIAS_WRITE_BACK_THRESHOLD` (float, default: `0.92`)

### Owner
* `[you]` / `[deepseek]` (you supply the JSON fixtures; deepseek writes the tune CLI).

---

## 7. STT Scope (Phase 6)

### Decision
* **Scope**: Whisper transcribes the audio payload. The returned text is passed directly to the standard Stage A text parsing pipeline.
* **Language/Locale**: The language returned by Whisper is propagated as the `Locale` hint to the parser.
* No confidence gating or transcripts filtering is performed. If transcription fails or returns nonsense, the normal parsing pipeline fails to extract ingredients and triggers the standard clarification loop naturally.

### Rationale
* Leveraging the pre-existing clarification loop keeps the STT subsystem simple and robust without introducing custom error-prone confidence heuristics.

### Implementation Notes
* Gated by `ENABLE_STT=true` (default: `false`).
* **Config Keys**:
  * `ENABLE_STT` (boolean, default: `false`)
  * `WHISPER_URL` (string, default: `http://whisper:8080`)

### Owner
* `[deepseek]`

---

## 8. v1 Release Scope (Phase 7)

### Decision
* **Messaging Adapters**: `telegram` is fully supported. `discord` and `matrix` are experimental/reference.
* **Notification Adapters**: `ntfy` is fully supported. `gotify` is experimental/reference.
* **Nutrition Sources**: `taco` (embedded) and `openfoodfacts` (HTTP) are fully supported. `usda` is experimental.
* **Features Included**: Read/Write REST API, Dashboard (opt-in), STT (opt-in).
* **Features Deferred**: Multi-User mode is deferred to post-v1 (databases prepared, but `MULTI_USER` defaults to `false`).
* **Compliance**: All messaging/notifier adapters must pass their port-level contract test suites before being merged.

### Rationale
* Restricting release to Telegram and ntfy allows stabilizing the core modular monolith architecture before supporting broad multi-user and multi-chat ecosystems.

### Implementation Notes
* Implement unit and contract test suites under `core/ports/contract_tests/`.

### Owner
* `[deepseek]`
