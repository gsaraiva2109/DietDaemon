# Roadmap

Future features only — shipped work is removed here (see git history/CHANGELOG for what's
done). Grouped by implementation complexity, not by theme. Each entry is a one-liner + why;
sizing/design happens when picked up.

## Low complexity

1. **Split `internal/store/store.go` (2686 lines) by domain** — reorg only, no logic change:
   users.go, meals.go, templates.go, nudges.go, etc. Same functions, same SQL, just moved.
   Zero behavior risk since nothing changes but file boundaries; do after the current batch
   (phases 1-4/6/7) ships, not concurrently with in-flight edits to this file.
2. **Split `internal/api/handler.go` (2372 lines) by domain** — same reorg-only split: auth,
   templates, meals, backup, etc. into separate files. Same reasoning and same "after, not
   during" caveat as the store.go split.
3. **Adopt sqlx incrementally** — replace hand-written `rows.Scan(&a, &b, ...)` boilerplate with
   `sqlx.Get`/`Select` struct-scanning, one function at a time. `sqlx.Rebind()` covers what
   `s.rewrite()` already does for sqlite/postgres placeholders, so this is a drop-in, not a
   rewrite — picked over sqlc/jet specifically because both of those generate per-engine code
   with no equivalent bridge for a single query targeting two dialects (see the ORM/query-layer
   discussion this session for the full comparison).
4. **Inline-keyboard quick actions** — nudge messages ship with tappable buttons ("log usual
   breakfast", "log 500ml water") in Telegram/Discord. Both platforms support native message
   buttons; Matrix client support is patchy, may need a text-command fallback there.
5. **Shareable read-only dashboard link** — read-only token scoped to one `account_id`, same
   per-account isolation multi-user login already requires. Not a new access model, just another
   token type on the existing scoped-read path.
6. **Import old logs (MyFitnessPal etc.)** — one-time CSV/export parser mapping to internal meal
   records. No ongoing maintenance, it's a one-shot ETL path.
7. **Confidence-colored macro numbers** — visually distinguish low-confidence/guessed values from
   resolved ones, not just a badge. Parser confidence is already computed, this is styling only.
8. **"Why is this number here" trace** — tap a logged meal's macro, see which resolver source
   (OFF/TACO/USDA) + confidence tier answered it. Drill-down UI over data already stored, no new
   computation.
9. **Data export sanity check** — before a scheduled backup/export runs, compare row counts
   against the last successful run, catch silent corruption instead of blindly dumping whatever's
   there. Extends `internal/backup`.
10. **Rate limit auth endpoints** — login, passwordless, MFA-email, OIDC callback
    (`handler_auth.go`, `handler_passwordless.go`, `handler_mfa_email.go`, `handler_oidc.go`) have
    no throttling today, confirmed via grep. `golang.org/x/time/rate`, per-IP or per-account
    token bucket, a few lines per handler — no new infra, no external dependency risk (same x/
    trust tier as `x/crypto`/`x/oauth2` already in use).
11. **Squash migrations to one file per dialect** — no production data to preserve yet, so
    collapse `migrations/{sqlite,postgres}/*.sql` into a single `001_init.sql` each reflecting
    current schema, delete the rest, wipe dev DB files. No new dependency — the existing
    hand-rolled runner (`runMigrations()`, sorted filename + `schema_migrations` table) already
    handles a single file fine, this is a one-time reorg, not a tooling change.

## Medium complexity

1. **Recipe / multi-ingredient composition** — save a combo of foods (e.g. "chicken + rice +
   broccoli, my usual") as one loggable unit, distinct from the existing single-food templates
   (`internal/commands/template.go`). Carried over, still undone.
2. **Barcode scan** — photo → barcode → OpenFoodFacts lookup. Works well for packaged
   supermarket goods; no coverage for fresh/local/artisanal food since those never had a barcode
   to begin with (not a DB gap, a barcode gap). Scope expectations accordingly. Will need a
   barcode-decode library when picked up — `gozxing` (pure-Go ZXing port) fits this repo's
   no-CGO stance (matches the `modernc.org/sqlite` choice); decide then, no dependency added now.
3. **Rule-based macro-fit matching engine** — given remaining kcal/protein, search the food
   library/resolver for combos that fit. No LLM needed. Foundation piece — both the fridge-recipe
   suggestion below and the LLM-ranked suggestions in High reuse this matcher.
4. **Macro-aware recipe suggestion from on-hand ingredients** — user lists what's in the fridge,
   the matching engine above finds combos hitting remaining macros.
5. **Smart reminders from historical patterns** — learn usual meal/log times from stored history,
   nudge before the user's own pattern instead of `scheduler.DefaultRules()`'s fixed hours.
   Extends the existing rules engine, not a new one.
6. **Photo storage policy** — where/how progress photos (`PhotoGrid.tsx`, `PhotoCompare.tsx`) are
   stored, size limits, retention. Needs a design pass (storage backend, retention, whether it
   survives `MULTI_USER` account deletion) before implementation.
7. **Adherence streak, not just log streak** — track "days within target range" separately from
   "days logged". Logging isn't the same as hitting goals; current streak framing conflates them.
8. **End-of-day honest recap** — passive nudge at day-close summarizing pattern truth ("hit
   protein, missed water 3x this week"). Extends the existing weekly digest, not a new subsystem.
9. **Undo/edit window on nudges** — if a nudge fires on stale data (logged 2 min after it sent),
   let the notification itself be dismissed/corrected instead of just ignored.
10. **Correction feedback loop** — when `/correct` fixes a misparsed item, auto-feed that
    correction into the alias table instead of leaving the food-library fix as a separate manual
    step.
11. **Weekly rolling calorie/protein budget with auto-compensation** — daily targets today are
    fixed and reset at midnight with no memory of prior days (`GetTargets`/`SetTargets`,
    `types.DailyTargets`). For hypertrophy, one bad day shouldn't blow the week; this makes the
    *effective* daily target self-correct off the weekly goal. High detail below, this one's
    important — expanded past the usual one-liner on purpose.

    - **Formula**: `effective_target_today = (weekly_target − actual_so_far_this_week) /
      days_remaining_in_week`. Self-correcting either direction — undereating raises the
      remaining days' target, *overeating lowers it* (never tells you to eat more after a binge,
      the bug in the first draft of this idea).
    - **Window**: calendar week, Monday–Sunday. Gives a clean denominator
      (`7 − day_of_week`) and lines up with the existing weekly digest
      (`scheduler.DefaultDigestRules`, fires Sunday).
    - **Data source**: week-to-date actuals via `Store.GetRollups(ctx, userID, mondayDate,
      todayDate)` (already exists, same range-query the digest uses); baseline via
      `Store.GetTargets`. No new aggregation query needed, just a new consumer of `GetRollups`.
    - **Weekly target**: defaults to `daily_target × 7`, but should be independently settable
      (someone might want a weekly total that isn't a flat multiple, e.g. deliberately eating more
      on training days).
    - **Clamp**: percentage-based floor/ceiling on the *redistributed* number, not a flat ±kcal
      cap (a flat cap breaks differently for a 1800kcal target vs a 3500kcal one). E.g. floor
      = 85% of baseline daily target, ceiling = 120% — configurable, but must exist, an
      unclamped formula can demand an unsafe single-day swing after a big miss.
    - **Week boundary**: leftover debt/surplus resets Sunday→Monday, does not carry into next
      week. Unbounded carry-over is a guilt spiral, clashes with the "calm, quiet" brand
      (`docs/PRODUCT.md`).
    - **Scope**: run the formula independently for calories *and* protein (protein is the actual
      hypertrophy lever, calories are bulk fuel) — same formula twice, two clamp configs.
    - **Delivery**: dashboard shows the adjusted number *alongside* the plain daily target, never
      hides the original (transparency principle already in `docs/PRODUCT.md`: "honest about
      uncertainty"). Nudge copy branches on sign: behind → "catch up today, +Xkcal", ahead →
      "ease up today, −Xkcal" — never assumes direction.
    - **Opt-in**: off by default, per-user toggle (rides the same per-user config pattern
      `NudgeRuleConfig` already established for configurable nudge rules) — plenty of users want
      strict daily targets, this shouldn't change default behavior for them.
    - **Edge case**: Monday (day 1 of week) has no prior days to redistribute from, effective
      target == plain daily target, no adjustment shown.
    - **Touch points**: `internal/scheduler/rules.go` (new rule alongside `DigestRule`),
      `internal/store/store.go` (reuse `GetRollups`/`GetTargets`, no schema change expected),
      `core/types/types.go` (new config struct, e.g. `WeeklyBudgetConfig`), dashboard component
      for the adjusted-number display.
    - **Complexity**: Medium — reuses existing range-query and per-user config patterns, no new
      subsystem, no LLM, no third-party integration.

## High complexity

1. **LLM adapter (local + cloud, opt-in)** — extends `MODEL_ADAPTER` (today: `ollama`) with an
   `anthropic`/`openai`-compatible option. Opt-in only, core keeps booting with zero keys.
2. **LLM-ranked meal suggestions** — rides on the rule-based matching engine and the LLM adapter
   above: rule-based candidates, LLM ranks/phrases them, adjustable prompt.
3. **Eating-out mode (photo menu only)** — OCR the photo (same image→text shape as the existing
   STT step), feed the transcript through the normal meal parser. Dish name → macros still needs
   an LLM rough-estimate (no nutrition source prices whole restaurant dishes), shipped as
   low-confidence per the existing "honest about uncertainty" design principle. Depends on the
   LLM adapter above. Skipping the digital-menu-scraper variant entirely (see Dropped). OCR
   step: shell out to the `tesseract` binary via `os/exec` rather than a cgo binding
   (`gosseract`), same no-CGO reasoning as the barcode-scan pick above — decide then, no
   dependency added now.
4. **Health platform import/export** — Apple Health / Google Fit / Garmin sync for weight and
   workout data, since those trackers already exist in-app (`weight.go`, `workout.go`). Carried
   over, still undone — genuinely large: several distinct third-party APIs/OAuth flows, sync and
   conflict-resolution logic, not one integration.
5. **Family/household multi-user sharing** — shared targets or a shared fridge/food library
   across accounts. Auth already supports multi-user (OIDC, invite mode); this is a data-model
   layer on top (shared vs private meals/targets per household).
6. **Target auto-suggestion from trend** — if weight trend contradicts the stated goal (e.g.
   "cutting" but flat 3 weeks), surface a gentle "adjust target?" prompt instead of silently
   nudging against a target that isn't working. Trend-detection isn't trivial (noise vs signal),
   and framing needs care to stay an observation about the user's own stated goal, not dietary
   advice — same territory as the dropped meal-plan generator.

## Dropped / not pursuing

- **Photo food recognition (full CV, unlabeled plate photos)** — genuinely hard CV problem
  (identify + estimate portions), explicitly deferred, not touching yet.
- **Digital-menu scraper** — one bespoke scraper per restaurant site, all different, all break
  silently, ToS-gray. Not worth it for a single-user self-hosted tool.
