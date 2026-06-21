# Design

Mood: *Scandinavian winter morning, quiet light through frost; a calm clinic in sage and paper.* Calm Health / Soft Structuralism. Feel reference: Apple Health / Fitness (rings, big numbers, soft cards, glanceable). Register: product. Desktop-first, responsive. WCAG AA.

Color strategy: **Restrained**, near-neutral surfaces, sage carries the calm, one warm accent reserved for "behind target". Tokens are OKLCH. No hex. Avoid AI-purple, neon, claude-beige, corporate gray.

## Color

### Light (default)

| Role | OKLCH | Use |
|---|---|---|
| `--bg` | `oklch(0.994 0.002 150)` | App background, near-pure paper white |
| `--surface` | `oklch(0.975 0.006 155)` | Cards, panels, nav |
| `--surface-2` | `oklch(0.955 0.008 155)` | Inset wells, hover |
| `--ink` | `oklch(0.27 0.020 160)` | Body text (≥7:1 on bg) |
| `--muted` | `oklch(0.52 0.015 160)` | Secondary text (≥4.5:1 on bg) |
| `--line` | `oklch(0.90 0.006 155)` | Hairlines, dividers |
| `--primary` | `oklch(0.62 0.090 155)` | Sage, calories ring, primary actions; white text on fill |
| `--primary-soft` | `oklch(0.93 0.030 155)` | Tints, ring tracks, selected bg |
| `--accent` | `oklch(0.70 0.150 65)` | Warm amber, "behind target" emphasis; white text on fill |

### Macro hues (color-blind-safe set; ALWAYS paired with a label/number, never color alone)

| Macro | OKLCH | Note |
|---|---|---|
| Calories | `oklch(0.62 0.090 155)` | sage (primary) |
| Protein | `oklch(0.58 0.110 250)` | calm blue |
| Carbs | `oklch(0.72 0.130 75)` | amber |
| Fat | `oklch(0.64 0.110 25)` | clay/rose |
| Fiber | `oklch(0.60 0.070 180)` | muted teal |

State: `--over-target` reuses `--accent` (amber), surfaced with an icon + text, not color alone. `--on-track`/met = `--primary` (sage).

### Dark (calm, NOT neon, optional theme)

`--bg oklch(0.20 0.012 160)`, `--surface oklch(0.245 0.014 160)`, `--ink oklch(0.93 0.010 150)`, `--muted oklch(0.72 0.012 155)`, `--line oklch(0.32 0.012 160)`. Primary/accent/macro hues lift L by ~0.06 for contrast. Deep slate-sage, no pure black, no glow.

## Typography

- **Display / numbers:** one grotesque family in multiple weights, **Plus Jakarta Sans** (or Geist). NEVER Inter/Roboto/Arial. Huge glanceable macro numbers are the hero element.
- **Body:** same family, regular weight. Body line length 65 to 75ch.
- Display letter-spacing floor **−0.04em** (never tighter). Hero number `clamp()` max ≤ 6rem.
- `text-wrap: balance` on headings; `text-wrap: pretty` on prose. Tabular figures (`font-variant-numeric: tabular-nums`) for all macro numbers so they don't jump when animating.

## Space, radius, shadow

- Spacing scale (rem): 0.25 / 0.5 / 0.75 / 1 / 1.5 / 2 / 3 / 4 / 6. Generous whitespace; vary rhythm.
- Radius: `--r-sm 0.5rem`, `--r-md 0.875rem`, `--r-lg 1.25rem`, `--r-xl 1.75rem`, `--r-full 9999px`. Concentric: nested radius = outer − padding.
- Shadows: soft *diffused ambient* only, e.g. `0 1px 2px oklch(0.27 0.02 160 / 0.04), 0 8px 24px oklch(0.27 0.02 160 / 0.06)`. No harsh dark `shadow-md`, no `rgba(0,0,0,0.3)`.

## Components

- **Macro ring/arc:** the signature element. Soft track (`--primary-soft`) + colored progress arc; big centered remaining-to-target number with `tabular-nums`; small consumed/target sublabel. Apple-Health character.
- **Cards:** sparingly. Single-level only, **never nested cards**. Soft surface + hairline + ambient shadow + `--r-lg`.
- **Nav:** desktop sidebar (collapsible), bottom-bar on mobile. Quiet, not edge-to-edge sticky.
- **Pills/badges:** parser-tier + confidence chips, status pills. Rounded-full, eyebrow micro-caps for section labels.
- **Buttons:** rounded-full primary (sage, white text); quiet ghost secondaries.
- Build a semantic z-index scale (dropdown → sticky → modal-backdrop → modal → toast → tooltip).

## Motion (dial: MOTION 4 of 10, quiet, meaningful)

- Library: **Framer Motion** primary; **GSAP/ScrollTrigger** for scroll choreography on trends/history; **Lenis** smooth scroll on long pages only.
- Signature: numbers **spring/tick** toward their target on load and after logging (this is the one motion that earns its keep, it explains change). Ring arcs ease-out to fill.
- Staggered reveal on meal-history list. Ease-out exponential curves (quart/quint/expo). No bounce, no elastic, no infinite loops.
- **Every** animation has a `prefers-reduced-motion: reduce` fallback (crossfade or instant). Reveals enhance already-visible content, never gate visibility on a transition.
