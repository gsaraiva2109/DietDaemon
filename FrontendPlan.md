# DietDaemon Dashboard — Feature Expansion Plan

## Context

DietDaemon dashboard is a React 19 + Vite 8 SPA embedded in the Go binary. Currently 6 routes: Dashboard (today overview with macro rings), Log Meal, History, Meal Detail, Trends, Summary. Backend API has 10 endpoints — rollups, meals CRUD, targets, food library (internal only, no public API).

**Outcome**: Full-featured nutrition dashboard that stands on its own. Users track body composition, browse foods, save meal templates, set goals with automatic target calculation, export data, and control app appearance — all through the dashboard.

> **API backend complete.** All ~30 endpoints across phases 2-6 are implemented and merged to main. Remaining work is frontend only.

## User Decisions (Confirmed)

| Decision | Choice |
|----------|--------|
| Scope | Frontend only — backend APIs already implemented |
| Body tracking | Full body composition — weight + measurements (waist, hips, chest, arms, thighs) + progress photos |
| Implementation order | Dashboard-first: Weekly dash → Food browser → Meal templates → Weight → Goals → Export |
| Food aliases | Auto-learned from usage + manual alias manager UI in settings |
| Micronutrients | Best-effort display — show whatever data exists in food sources, no RDA targets |
| User profile | Onboarding flow with skip option, editable later in settings |
| Fasting timer | Manual start/stop timer on dashboard |
| Barcode scanning | Excluded |
| PWA | Deferred — not in this plan |
| Plan detail | Full plan — all phases with implementation detail |

## Architecture Overview

```
Phase 1 (Dashboard Polish)        Phase 2 (Food Discovery)       Phase 3 (Meal Workflow)
┌──────────────────────────┐    ┌──────────────────────────┐    ┌──────────────────────────┐
│ Weekly dashboard toggle   │    │ Food browser page         │    │ Meal templates CRUD       │
│ Dark mode toggle          │    │ Frequent foods section    │    │ Copy meal from history    │
│ Keyboard shortcuts        │    │ Food alias manager UI     │    │ Template → one-tap log    │
│ Manual fasting timer      │    │                           │    │                           │
└──────────────────────────┘    └──────────────────────────┘    └──────────────────────────┘

Phase 4 (Body Tracking)          Phase 5 (Goals & Planning)     Phase 6 (Export & Share)
┌──────────────────────────┐    ┌──────────────────────────┐    ┌──────────────────────────┐
│ Weight entry + chart      │    │ Onboarding wizard         │    │ CSV/JSON export            │
│ Body measurements         │    │ TDEE calculator            │    │ Share meal/day card        │
│ Progress photos           │    │ Goal profiles (cut/maintain│    │                           │
│                           │    │            /bulk)          │    │                           │
│                           │    │ Auto-calculate targets     │    │                           │
│                           │    │ Calorie adaptation nudge   │    │                           │
└──────────────────────────┘    └──────────────────────────┘    └──────────────────────────┘
```

## Design Principles

1. **Same visual language** — reuse existing primitives (Card, Eyebrow, Pill, Button, EmptyState, Spinner) from `web/src/components/ui.tsx`. Match existing OKLCH color tokens, Plus Jakarta Sans font, soft shadows, rounded-full inputs.
2. **Same data patterns** — PascalCase types mirroring Go structs, TanStack Query with 30s polling, demo mode short-circuit, `cssVar()` for runtime color resolution.
3. **Same animation vocabulary** — `fadeUp`/`stagger` for lists, `easeOut` for transitions, `numberSpring` for animated numbers, `scaleIn` for modals.
4. **Backward compatible** — existing APIs unchanged. New features are additive. Dashboard works without opting into weight tracking or onboarding.

---

## Phase 1 — Dashboard Polish (Dashboard-First)

Goal: Enrich the existing Dashboard without new backend endpoints. Weekly view, dark mode toggle, keyboard shortcuts, manual fasting timer. All data already available via `/rollups/range`, `/meals`, and `/meals/latest`.

### Frontend — Files to Create

| File | Purpose |
|------|---------|
| `web/src/components/WeeklyDashboard.tsx` | Weekly stat tiles (avg kcal, avg protein, adherence %, trend arrows), 7-day macro bar chart, best/worst day cards |
| `web/src/components/FastingTimer.tsx` | Manual start/stop timer. Shows elapsed time. Optional "fasting goal" (e.g. 16h). Pill badge on dashboard |
| `web/src/components/ThemeToggle.tsx` | Extracted from `UtilityBar.tsx` — standalone animated sun/moon toggle button |

### Frontend — Files to Modify

| File | Change |
|------|--------|
| `web/src/routes/Dashboard.tsx` | Add day/week toggle at top. Weekly mode renders `WeeklyDashboard` below the hero ring. Add `FastingTimer` pill in the side-stats column |
| `web/src/components/UtilityBar.tsx` | Replace inline theme toggle with `ThemeToggle` component |
| `web/src/components/CommandPalette.tsx` | Add keyboard shortcuts: `⌘L` → /log, `⌘H` → /history, `⌘T` → /trends, `⌘S` → /summary, `⌘D` → / (dashboard) |
| `web/src/components/AppShell.tsx` | Add dark mode toggle to sidebar (desktop) and bottom nav (mobile) |
| `web/src/lib/types.ts` | Add `WeeklyStats` type (computed on frontend from `DailyRollup[]`) |
| `web/src/lib/insights.ts` | Add `weeklyInsights(range: DailyRollup[])` — averages, trend direction, adherence |

### Verification
```bash
cd web && npm run build               # compiles clean
make build                            # Go binary embeds updated SPA
# Dashboard shows day/week toggle. Weekly view renders stats from last 7 days.
# Cmd+K opens palette, Cmd+L navigates to /log, Cmd+H to /history.
# Fasting timer starts/stops/resets. Elapsed time shown.
# Dark mode toggle in sidebar + Cmd+K palette switches theme.
```

---

## Phase 2 — Food Discovery (Category B)

Goal: Browse and search the personal food library. See macros per 100g before logging. Manage food aliases. Surface frequently-eaten foods.

### Frontend — Files to Create

| File | Purpose |
|------|---------|
| `web/src/routes/Foods.tsx` | Food browser page. Search input + source filter pills + results grid. Each result is a `FoodCard`. Click opens detail modal |
| `web/src/components/FoodCard.tsx` | Summary card: food name, source badge, per-100g macro mini-grid (kcal/P/C/F), last-used date. Hover lifts shadow |
| `web/src/components/FoodDetailModal.tsx` | Modal: full macro breakdown, all aliases, serving info, "Log this" button → navigates to /log with pre-filled text |
| `web/src/routes/Aliases.tsx` | Settings sub-page or section: search foods, see/edit aliases per food. Save/delete alias inline |
| `web/src/components/FrequentFoods.tsx` | Horizontal scrollable row of top-20 food pills. Shows on Dashboard + Foods page |

### Frontend — Files to Modify

| File | Change |
|------|--------|
| `web/src/App.tsx` | Add route `/foods` → `Foods`, `/settings/aliases` → `Aliases` |
| `web/src/components/AppShell.tsx` | Add "Foods" nav item (desktop sidebar + mobile bottom bar) |
| `web/src/lib/api.ts` | Add `foods.list()`, `foods.search()`, `foods.frequent()`, `foods.get()`, `foods.addAlias()`, `foods.deleteAlias()` |
| `web/src/lib/queries.ts` | Add `useFoods()`, `useSearchFoods()`, `useFrequentFoods()`, `useFood()`, `useAddAlias()`, `useDeleteAlias()` |
| `web/src/lib/types.ts` | Add `FoodDetail`, `FoodAlias` types |
| `web/src/lib/demo.tsx` | Add `DEMO_FOODS` — 15-20 sample food entries |
| `web/src/routes/Dashboard.tsx` | Add `FrequentFoods` row below today's meals section |

### Verification
```bash
cd web && npm run build                   # compiles
# /foods shows searchable food list. Filter by source works.
# Click food opens detail modal with macros + aliases.
# Frequent foods shown on dashboard + foods page.
# /settings/aliases lets you add/edit/delete aliases.
```

---

## Phase 3 — Meal Workflow (Category C)

Goal: Save meals as named templates. One-tap log a template. Copy a meal from any past day. Reduce typing friction for repeat meals.

### Frontend — Files to Create

| File | Purpose |
|------|---------|
| `web/src/routes/Templates.tsx` | List saved templates. Each shows name, item count, total kcal, last used. Tap to log (confirmation dialog), swipe/button to delete |
| `web/src/components/SaveTemplateModal.tsx` | Modal: name input + preview of items being saved. Opens from any MealDetail page |
| `web/src/components/DuplicateMealModal.tsx` | Modal: pick a day, pick a meal from that day, duplicate it as today's meal |

### Frontend — Files to Modify

| File | Change |
|------|--------|
| `web/src/App.tsx` | Add routes `/templates` → `Templates` |
| `web/src/components/AppShell.tsx` | Add "Templates" nav item |
| `web/src/routes/MealDetail.tsx` | Add "Save as template" button next to existing "Add item" button |
| `web/src/routes/LogMeal.tsx` | Add "From template" quick-action section: row of recent template pills. Add "Copy from day" button that opens DuplicateMealModal |
| `web/src/lib/api.ts` | Add `templates.list()`, `templates.create()`, `templates.get()`, `templates.delete()`, `templates.log()`, `meals.duplicate()` |
| `web/src/lib/queries.ts` | Add `useTemplates()`, `useCreateTemplate()`, `useDeleteTemplate()`, `useLogTemplate()`, `useDuplicateMeal()` |
| `web/src/lib/types.ts` | Add `MealTemplate` type |
| `web/src/lib/demo.tsx` | Add `DEMO_TEMPLATES` — 3-5 sample templates |

### Verification
```bash
cd web && npm run build
# /templates shows saved templates. Click logs a meal. Delete removes template.
# MealDetail → "Save as template" creates template with current items.
# LogMeal → "From template" row lets you one-tap log a template.
# LogMeal → "Copy from day" opens picker, duplicates meal.
```

---

## Phase 4 — Body Tracking (Category A)

Goal: Log weight and body measurements. See trends over time. Correlate weight change with calorie intake. Upload progress photos.

### Frontend — Files to Create

| File | Purpose |
|------|---------|
| `web/src/routes/Body.tsx` | Body tracking hub. Sub-tabs: Weight, Measurements, Photos. Weight tab: entry form (date + kg + note), trend chart (line with rolling avg), history list. Measurements tab: form grid (waist/hips/chest/etc) + trend lines. Photos tab: upload button + timeline grid |
| `web/src/components/WeightChart.tsx` | Recharts LineChart: weight over time + 7-day rolling average line. Overlay calorie intake as bar background (dual-axis). Date range toggle (30/90/180/all) |
| `web/src/components/MeasurementChart.tsx` | Recharts LineChart: multi-line (waist, hips, chest) over time. Toggle individual lines |
| `web/src/components/PhotoGrid.tsx` | CSS grid of photo thumbnails grouped by date. Click opens full-size comparison view |
| `web/src/components/PhotoCompare.tsx` | Side-by-side photo comparison modal: "4 weeks ago" vs "today" |

### Frontend — Files to Modify

| File | Change |
|------|--------|
| `web/src/App.tsx` | Add route `/body` → `Body` |
| `web/src/components/AppShell.tsx` | Add "Body" nav item |
| `web/src/routes/Dashboard.tsx` | Add mini weight card to side-stats column. Shows latest weight + change arrow (↑0.5kg this week) |
| `web/src/lib/api.ts` | Add `body.weight.*`, `body.measurements.*`, `body.photos.*`, `body.summary()` |
| `web/src/lib/queries.ts` | Add `useWeightLog()`, `useWeightTrend()`, `useLogWeight()`, `useMeasurements()`, `useLogMeasurements()`, `usePhotos()`, `useUploadPhoto()`, `useBodySummary()` |
| `web/src/lib/types.ts` | Add `WeightEntry`, `MeasurementEntry`, `ProgressPhoto`, `WeightTrend`, `BodyCompositionSummary` types |
| `web/src/lib/demo.tsx` | Add `DEMO_WEIGHT` — 90 days of realistic weight data. Add `DEMO_MEASUREMENTS` — weekly measurements |

### Verification
```bash
cd web && npm run build
# /body/weight shows chart + entry form. Log weight, see trend line update.
# /body/measurements shows multi-line chart. Enter measurements, see trends.
# /body/photos uploads and shows grid. Compare mode works.
# Dashboard shows latest weight in side card.
```

---

## Phase 5 — Goals & Planning (Category E)

Goal: Onboarding wizard collects body stats + goal. Auto-calculates TDEE and macro targets. User can revisit and adjust anytime. Calorie adaptation suggestions based on weight trend.

### Frontend — Files to Create

| File | Purpose |
|------|---------|
| `web/src/components/OnboardingWizard.tsx` | Multi-step wizard shown on first login (when `profile.onboarded == false`). Step 1: body stats (height, weight, birth date, gender). Step 2: activity level (5 options with descriptions). Step 3: goal selection (cut/maintain/bulk) + target weight + desired rate. Step 4: calculated targets shown with "Save" and "Skip" buttons. Progress dots at top |
| `web/src/routes/Goals.tsx` | Goals & profile page. Shows current targets, TDEE breakdown, goal progress. "Edit profile" opens the wizard in edit mode. "Recalculate targets" button |
| `web/src/components/TDEECard.tsx` | Visual TDEE breakdown: BMR → TDEE → target calories. Animated bar or donut. Shows macro split |
| `web/src/components/GoalSuggestion.tsx` | Card: "You're losing 0.3kg/week at 2000 kcal. To hit your 0.5kg/week goal, try 1850 kcal." With "Apply" button that calls `/targets` |

### Frontend — Files to Modify

| File | Change |
|------|--------|
| `web/src/App.tsx` | Add onboarding check in `Gate` component: if `authed && !onboarded`, show `OnboardingWizard`. Add route `/goals` → `Goals` |
| `web/src/components/AppShell.tsx` | Add "Goals" nav item (replaces or sits alongside Settings) |
| `web/src/routes/Settings.tsx` | Add "Body profile" link → opens wizard in edit mode. Add "Recalculate targets" button |
| `web/src/lib/api.ts` | Add `profile.get()`, `profile.put()`, `tdee.calculate()`, `goals.suggestions()` |
| `web/src/lib/queries.ts` | Add `useProfile()`, `useUpsertProfile()`, `useTDEE()`, `useGoalSuggestions()` |
| `web/src/lib/types.ts` | Add `UserProfile`, `TDEEParams`, `TDEEResult`, `GoalSuggestion` types |

### Verification
```bash
cd web && npm run build
# First login shows onboarding wizard. Fill in steps, see calculated targets.
# /goals shows TDEE breakdown + current targets. Edit profile works.
# Goal suggestion appears if weight trend mismatches goal.
# "Apply" updates targets. Settings → "Recalculate" works.
```

---

## Phase 6 — Export & Share (Category F)

Goal: Download meal and rollup data as CSV/JSON. Generate shareable meal/day summary images.

### Frontend — Files to Create

| File | Purpose |
|------|---------|
| `web/src/components/ExportModal.tsx` | Modal: select type (meals/rollups), format (CSV/JSON), date range. "Download" button triggers file save |
| `web/src/components/ShareCard.tsx` | Generates a styled card image (via `html-to-image` or `html2canvas`): "Today: 2,100 kcal · 180g protein · 220g carbs · 70g fat". Macro rings mini version. Download as PNG or copy to clipboard |

### Frontend — Files to Modify

| File | Change |
|------|--------|
| `web/src/routes/Settings.tsx` | Add "Export data" button → opens `ExportModal` |
| `web/src/routes/Dashboard.tsx` | Add share button (icon) in header → generates share card |
| `web/src/routes/MealDetail.tsx` | Add share button → generates meal-specific share card |
| `web/src/routes/Summary.tsx` | Add export button in header |

### Dependencies to Add
```bash
cd web && npm install html-to-image  # for share card PNG generation
```

### Verification
```bash
cd web && npm run build
# Settings → Export → pick CSV + 30 days → file downloads with correct data.
# Dashboard → share button → PNG generated with today's macros.
# MealDetail → share button → PNG with meal items + macros.
```

---

## Implementation Order Summary

```
Phase 1: Dashboard Polish          (1-2 sessions, frontend only)
Phase 2: Food Discovery            (1-2 sessions, frontend only)
Phase 3: Meal Workflow             (1-2 sessions, frontend only)
Phase 4: Body Tracking             (2-3 sessions, frontend only)
Phase 5: Goals & Planning          (1-2 sessions, frontend only)
Phase 6: Export & Share            (1-2 sessions, frontend only)
```

All backend APIs already implemented. Each phase is independently shippable.

---

## New Dependencies Summary

| Phase | Package | Purpose |
|-------|---------|---------|
| 6 | `html-to-image` | Share card PNG generation |

No other new dependencies. All charts use existing Recharts. All state uses existing TanStack Query + React context.

---

## Verification (End-to-End)

```bash
# Frontend
cd web && npm ci && npx tsc -b --noEmit && npm run lint && npm run build

# Full build
make build   # produces bin/dietdaemon with embedded SPA
```
