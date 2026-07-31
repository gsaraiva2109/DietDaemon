# Roadmap

Future features only — shipped work is removed here (see git history/CHANGELOG for what's
done). Grouped by implementation complexity, not by theme. Each entry is a one-liner + why;
sizing/design happens when picked up.

## Low complexity

1. **Shareable read-only dashboard link** — read-only token scoped to one `account_id`, same
   per-account isolation multi-user login already requires. Not a new access model, just another
   token type on the existing scoped-read path.

## Medium complexity

1. **Barcode scan** — photo → barcode → OpenFoodFacts lookup. Works well for packaged
   supermarket goods; no coverage for fresh/local/artisanal food since those never had a barcode
   to begin with (not a DB gap, a barcode gap). Scope expectations accordingly. Will need a
   barcode-decode library when picked up — `gozxing` (pure-Go ZXing port) fits this repo's
   no-CGO stance (matches the `modernc.org/sqlite` choice); decide then, no dependency added now.
2. **Macro-aware recipe suggestion from on-hand ingredients** — user lists what's in the fridge,
   the matching engine (`internal/suggest`, shipped) finds combos hitting remaining macros.
3. **Photo storage policy** — see [docs/PHOTO_STORAGE.md](PHOTO_STORAGE.md) for the approved
   BLOB-in-DB storage decision, enforced limits, access-control guarantees, and future
   S3-migration trigger conditions.

## High complexity

1. **Family/household multi-user sharing** — shared targets or a shared fridge/food library
   across accounts. Auth already supports multi-user (OIDC, invite mode); this is a data-model
   layer on top (shared vs private meals/targets per household).

## Dropped / not pursuing

- **Photo food recognition (full CV, unlabeled plate photos)** — genuinely hard CV problem
  (identify + estimate portions), explicitly deferred, not touching yet.
- **Digital-menu scraper** — one bespoke scraper per restaurant site, all different, all break
  silently, ToS-gray. Not worth it for a single-user self-hosted tool.