// Lightweight, honest insights derived from today's rollup and the time of
// day. "Pace" compares consumed vs the fraction of the day elapsed, so a nudge
// only fires when you're genuinely behind — matching the product's nudge model.

import type { DailyRollup, Macros, MacroKey, TrendDirection, WeeklyStats } from './types'

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

const ZERO_MACROS: Macros = { Calories: 0, Protein: 0, Carbs: 0, Fat: 0, Fiber: 0 }

// Compare the average of the first half of the range to the second half.
function trend(values: number[]): TrendDirection {
  if (values.length < 2) return 'flat'
  const mid = Math.floor(values.length / 2)
  const avg = (xs: number[]) => (xs.length ? xs.reduce((s, x) => s + x, 0) / xs.length : 0)
  const a = avg(values.slice(0, mid))
  const b = avg(values.slice(mid))
  if (a === 0) return 'flat'
  const delta = (b - a) / a
  if (delta > 0.05) return 'up'
  if (delta < -0.05) return 'down'
  return 'flat'
}

/**
 * weeklyStats reduces a range of daily rollups to dashboard-ready aggregates:
 * macro averages over logged days, calorie adherence (within ±10% of target),
 * calorie/protein trend direction, and the best/worst day by calorie accuracy.
 */
export function weeklyStats(range: DailyRollup[]): WeeklyStats {
  const logged = range.filter((d) => d.Consumed.Calories > 0)
  if (logged.length === 0) {
    return {
      days: range, avg: ZERO_MACROS, adherence: 0, calorieTrend: 'flat',
      proteinTrend: 'flat', bestDay: null, worstDay: null, loggedDays: 0,
    }
  }

  const sum = logged.reduce<Macros>(
    (acc, d) => ({
      Calories: acc.Calories + d.Consumed.Calories,
      Protein: acc.Protein + d.Consumed.Protein,
      Carbs: acc.Carbs + d.Consumed.Carbs,
      Fat: acc.Fat + d.Consumed.Fat,
      Fiber: acc.Fiber + d.Consumed.Fiber,
    }),
    { ...ZERO_MACROS },
  )
  const n = logged.length
  const avg: Macros = {
    Calories: sum.Calories / n, Protein: sum.Protein / n, Carbs: sum.Carbs / n,
    Fat: sum.Fat / n, Fiber: sum.Fiber / n,
  }

  // Adherence: fraction of logged days whose calories land within ±10% of target.
  const onTarget = logged.filter((d) => {
    const t = d.Targets.Calories
    if (t <= 0) return false
    return Math.abs(d.Consumed.Calories - t) / t <= 0.1
  }).length
  const adherence = onTarget / n

  // Best/worst by absolute distance from the calorie target (target-relative).
  const dist = (d: DailyRollup) => {
    const t = d.Targets.Calories
    return t > 0 ? Math.abs(d.Consumed.Calories - t) / t : Infinity
  }
  const sorted = [...logged].sort((a, b) => dist(a) - dist(b))
  const bestDay = sorted[0] ?? null
  const worstDay = sorted[sorted.length - 1] ?? null

  return {
    days: range,
    avg,
    adherence,
    calorieTrend: trend(logged.map((d) => d.Consumed.Calories)),
    proteinTrend: trend(logged.map((d) => d.Consumed.Protein)),
    bestDay,
    worstDay,
    loggedDays: n,
  }
}
