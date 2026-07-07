# Roadmap

Future features only — shipped work is removed here (see git history/CHANGELOG for what's
done). Grouped by implementation complexity, not by theme. Each entry is a one-liner + why;
sizing/design happens when picked up.

## Low complexity

1. **Adopt sqlx incrementally** — replace hand-written `rows.Scan(&a, &b, ...)` boilerplate with
   `sqlx.Get`/`Select` struct-scanning, one function at a time. `sqlx.Rebind()` covers what
   `s.rewrite()` already does for sqlite/postgres placeholders, so this is a drop-in, not a
   rewrite — picked over sqlc/jet specifically because both of those generate per-engine code
   with no equivalent bridge for a single query targeting two dialects (see the ORM/query-layer
   discussion this session for the full comparison).
2. **Shareable read-only dashboard link** — read-only token scoped to one `account_id`, same
   per-account isolation multi-user login already requires. Not a new access model, just another
   token type on the existing scoped-read path.
3. **Import old logs (MyFitnessPal etc.)** — one-time CSV/export parser mapping to internal meal
   records. No ongoing maintenance, it's a one-shot ETL path.
4. **Confidence-colored macro numbers** — visually distinguish low-confidence/guessed values from
   resolved ones, not just a badge. Parser confidence is already computed, this is styling only.
5. **"Why is this number here" trace** — tap a logged meal's macro, see which resolver source
   (OFF/TACO/USDA) + confidence tier answered it. Drill-down UI over data already stored, no new
   computation.

## Medium complexity

1. **Barcode scan** — photo → barcode → OpenFoodFacts lookup. Works well for packaged
   supermarket goods; no coverage for fresh/local/artisanal food since those never had a barcode
   to begin with (not a DB gap, a barcode gap). Scope expectations accordingly. Will need a
   barcode-decode library when picked up — `gozxing` (pure-Go ZXing port) fits this repo's
   no-CGO stance (matches the `modernc.org/sqlite` choice); decide then, no dependency added now.
2. **Macro-aware recipe suggestion from on-hand ingredients** — user lists what's in the fridge,
   the matching engine (`internal/suggest`, shipped) finds combos hitting remaining macros.
3. **Smart reminders from historical patterns** — learn usual meal/log times from stored history,
   nudge before the user's own pattern instead of `scheduler.DefaultRules()`'s fixed hours.
   Extends the existing rules engine, not a new one.
4. **Photo storage policy** — where/how progress photos (`PhotoGrid.tsx`, `PhotoCompare.tsx`) are
   stored, size limits, retention. Needs a design pass (storage backend, retention, whether it
   survives `MULTI_USER` account deletion) before implementation.
5. **Correction feedback loop** — when `/correct` fixes a misparsed item, auto-feed that
   correction into the alias table instead of leaving the food-library fix as a separate manual
   step.
## High complexity

1. **Eating-out mode (photo menu only)** — OCR the photo (same image→text shape as the existing
   STT step), feed the transcript through the normal meal parser. Dish name → macros still needs
   an LLM rough-estimate (no nutrition source prices whole restaurant dishes), shipped as
   low-confidence per the existing "honest about uncertainty" design principle. The LLM adapter
   this depends on is already shipped (`COMPLETION_ADAPTER=ollama|anthropic|openai`). Skipping
   the digital-menu-scraper variant entirely (see Dropped). OCR step: shell out to the
   `tesseract` binary via `os/exec` rather than a cgo binding (`gosseract`), same no-CGO
   reasoning as the barcode-scan pick above — decide then, no dependency added now.
2. **Family/household multi-user sharing** — shared targets or a shared fridge/food library
   across accounts. Auth already supports multi-user (OIDC, invite mode); this is a data-model
   layer on top (shared vs private meals/targets per household).
3. **Target auto-suggestion from trend** — if weight trend contradicts the stated goal (e.g.
   "cutting" but flat 3 weeks), surface a gentle "adjust target?" prompt instead of silently
   nudging against a target that isn't working. Trend-detection isn't trivial (noise vs signal),
   and framing needs care to stay an observation about the user's own stated goal, not dietary
   advice — same territory as the dropped meal-plan generator.

## Dropped / not pursuing

- **Photo food recognition (full CV, unlabeled plate photos)** — genuinely hard CV problem
  (identify + estimate portions), explicitly deferred, not touching yet.
- **Digital-menu scraper** — one bespoke scraper per restaurant site, all different, all break
  silently, ToS-gray. Not worth it for a single-user self-hosted tool.