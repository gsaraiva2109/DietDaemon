# Roadmap

Future features only — shipped work is removed here (see git history/CHANGELOG for what's
done). Grouped by implementation complexity, not by theme. Each entry is a one-liner + why;
sizing/design happens when picked up.

## Low complexity

None currently.

## Medium complexity

1. **Barcode scan** — photo → barcode → OpenFoodFacts lookup. Works well for packaged
   supermarket goods; no coverage for fresh/local/artisanal food since those never had a barcode
   to begin with (not a DB gap, a barcode gap). Scope expectations accordingly. Will need a
   barcode-decode library when picked up — `gozxing` (pure-Go ZXing port) fits this repo's
   no-CGO stance (matches the `modernc.org/sqlite` choice); decide then, no dependency added now.

## High complexity

1. **Family/household multi-user sharing** — shared targets or a shared fridge/food library
   across accounts. Auth already supports multi-user (OIDC, invite mode); this is a data-model
   layer on top (shared vs private meals/targets per household).

## Dropped / not pursuing

- **Photo food recognition (full CV, unlabeled plate photos)** — genuinely hard CV problem
  (identify + estimate portions), explicitly deferred, not touching yet.
- **Digital-menu scraper** — one bespoke scraper per restaurant site, all different, all break
  silently, ToS-gray. Not worth it for a single-user self-hosted tool.