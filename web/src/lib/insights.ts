// Lightweight, honest insights derived from today's rollup and the time of
// day. "Pace" compares consumed vs the fraction of the day elapsed, so a nudge
// only fires when you're genuinely behind — matching the product's nudge model.

import type { DailyRollup, MacroKey } from './types'

export interface Insight {
  tone: 'good' | 'warn' | 'info'
  text: string
}

function dayFraction(now = new Date()): number {
  // Treat a sensible eating window (08:00–22:00) as the pacing window.
  const h = now.getHours() + now.getMinutes() / 60
  return Math.min(1, Math.max(0, (h - 8) / 14))
}

export function greeting(now = new Date()): string {
  const h = now.getHours()
  if (h < 12) return 'Good morning'
  if (h < 18) return 'Good afternoon'
  return 'Good evening'
}

export function insights(rollup: DailyRollup | null): Insight[] {
  if (!rollup) return [{ tone: 'info', text: 'No data yet today — log a meal to get started.' }]
  const out: Insight[] = []
  const frac = dayFraction()
  const { Consumed: c, Targets: t } = rollup

  const protPct = t.Protein > 0 ? c.Protein / t.Protein : 1
  if (t.Protein > 0 && protPct < frac - 0.15) {
    out.push({ tone: 'warn', text: `Protein is behind pace — ${Math.round(t.Protein - c.Protein)}g to go.` })
  } else if (protPct >= 1) {
    out.push({ tone: 'good', text: 'Protein target hit. Nice.' })
  }

  const calPct = t.Calories > 0 ? c.Calories / t.Calories : 0
  if (t.Calories > 0) {
    if (calPct > 1) out.push({ tone: 'warn', text: `${Math.round(c.Calories - t.Calories)} kcal over target.` })
    else if (calPct >= frac - 0.1 && calPct < 1) out.push({ tone: 'good', text: 'On track for calories.' })
    else if (calPct < frac - 0.2) out.push({ tone: 'info', text: `${Math.round(t.Calories - c.Calories)} kcal left to hit your goal.` })
  }

  const biggestGap = (['Protein', 'Carbs', 'Fat'] as MacroKey[])
    .map((k) => ({ k, gap: t[k] > 0 ? 1 - c[k] / t[k] : 0 }))
    .sort((a, b) => b.gap - a.gap)[0]
  if (biggestGap && biggestGap.gap > 0.5 && out.length < 3) {
    out.push({ tone: 'info', text: `${biggestGap.k} is your biggest gap today.` })
  }

  return out.length ? out.slice(0, 3) : [{ tone: 'good', text: "You're tracking well today." }]
}

/** Consecutive days (ending today/most-recent) with any calories logged. */
export function streak(range: DailyRollup[]): number {
  let n = 0
  for (let i = range.length - 1; i >= 0; i--) {
    if (range[i].Consumed.Calories > 0) n++
    else break
  }
  return n
}
