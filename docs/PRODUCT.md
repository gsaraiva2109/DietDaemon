# Product

## Register

product

## Users

A single self-hosted user (the homelab owner) who is bulking and tracks daily macros. Primary context: a **desktop/homelab browser**, checked repeatedly through the day. Meals are mostly logged elsewhere (chat adapters, Telegram/Discord/Matrix); the web dashboard is where they **glance at progress and review/correct** what was logged. The job-to-be-done, in order: (1) "how much protein/calories do I still need today?", (2) review recent meals and fix a misparse, (3) check multi-day trends.

## Product Purpose

DietDaemon is a low-friction, self-hosted nutrition/macro tracker. The frontend is an **optional dashboard** (gated by `ENABLE_DASHBOARD`) sitting on top of a clean read-mostly REST API. It exists to make daily macro progress **glanceable** and to let the user audit/correct the automated parsing. Success = the user can answer "what's left to hit my targets today?" in under two seconds, and can correct a wrong food match in a few clicks. The core must keep booting fully headless without it.

## Brand Personality

Calm, quiet, focused. Three words: **restful, precise, glanceable**. The numbers do the talking; chrome gets out of the way. Tone is understated and trustworthy, a calm health companion, not a hype coach and not an instrument panel screaming for attention. Feel reference: **Apple Health / Fitness**, big friendly rings and numbers, soft cards, calm color, summaries you read at a glance.

## Anti-references

- **Generic AI-purple SaaS:** no purple gradients, no glassmorphism-on-everything, no three equal feature cards, no Inter + slate-900 default.
- **Busy MyFitnessPal-style:** no cramped data tables, no ad-dense clutter, no overwhelming nutrition-grid.
- **Dark gamer / neon dashboard:** no neon-on-black, no RGB glow, no aggressive instrumentation.
- **Corporate/enterprise admin:** no sterile gray Bootstrap/Material admin-panel feel.

## Design Principles

1. **Remaining over consumed.** The hero answer is what's *left* to hit targets today, not a wall of totals.
2. **Glance, then drill.** Every screen reads in one glance; detail is one interaction away, never forced up front.
3. **Quiet by default, motion with meaning.** Calm surfaces; animation only to explain change (a number ticking toward target), never decoration.
4. **Honest about uncertainty.** Surface parser confidence/tier and make correction obvious, the product never hides a guessed macro.
5. **Numbers do the talking.** Typography and soft data-viz carry hierarchy; ornament stays out of the way.

## Accessibility & Inclusion

WCAG **AA** baseline: body contrast ≥4.5:1 (no light-gray-on-tint body text), large text ≥3:1. Full keyboard navigation and visible focus. Every animation has a `prefers-reduced-motion` fallback (crossfade/instant). Don't rely on color alone for macro state, pair color with labels/numbers. Desktop-first, but responsive and usable on phone.
