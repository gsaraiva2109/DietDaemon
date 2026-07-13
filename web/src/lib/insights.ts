// Lightweight, honest insights derived from today's rollup and the time of
// day. "Pace" compares consumed vs the fraction of the day elapsed, so a nudge
// only fires when you're genuinely behind, matching the product's nudge model.

import type { TFunction } from 'i18next'
import type { DailyRollup, Macros, MacroKey, TrendDirection, WeeklyStats } from './types'

export interface Insight {
  tone: 'good' | 'warn' | 'info'
  text: string
}

function dayFraction(now = new Date()): number {
  // Treat a sensible eating window (08:00 to 22:00) as the pacing window.
  const h = now.getHours() + now.getMinutes() / 60
  return Math.min(1, Math.max(0, (h - 8) / 14))
}

export function greeting(t: TFunction, now = new Date()): string {
  const h = now.getHours()
  if (h < 12) return t('insights.goodMorning')
  if (h < 18) return t('insights.goodAfternoon')
  return t('insights.goodEvening')
}

export function insights(rollup: DailyRollup | null, t: TFunction): Insight[] {
  if (!rollup) return [{ tone: 'info', text: t('insights.noData') }]
  const out: Insight[] = []
  const frac = dayFraction()
  const { Consumed: c, Targets: targets } = rollup

  const protPct = targets.Protein > 0 ? c.Protein / targets.Protein : 1
  if (targets.Protein > 0 && protPct < frac - 0.15) {
    out.push({ tone: 'warn', text: t('insights.proteinBehind', { grams: Math.round(targets.Protein - c.Protein) }) })
  } else if (protPct >= 1) {
    out.push({ tone: 'good', text: t('insights.proteinTargetHit') })
  }

  const calPct = targets.Calories > 0 ? c.Calories / targets.Calories : 0
  if (targets.Calories > 0) {
    if (calPct > 1) out.push({ tone: 'warn', text: t('insights.caloriesOver', { calories: Math.round(c.Calories - targets.Calories) }) })
    else if (calPct >= frac - 0.1 && calPct < 1) out.push({ tone: 'good', text: t('insights.caloriesOnTrack') })
    else if (calPct < frac - 0.2) out.push({ tone: 'info', text: t('insights.caloriesLeft', { calories: Math.round(targets.Calories - c.Calories) }) })
  }

  const biggestGap = (['Protein', 'Carbs', 'Fat'] as MacroKey[])
    .map((k) => ({ k, gap: targets[k] > 0 ? 1 - c[k] / targets[k] : 0 }))
    .sort((a, b) => b.gap - a.gap)[0]
  if (biggestGap && biggestGap.gap > 0.5 && out.length < 3) {
    out.push({ tone: 'info', text: t('insights.biggestGap', { macro: t(`common.macro.${biggestGap.k}`) }) })
  }

  return out.length ? out.slice(0, 3) : [{ tone: 'good', text: t('insights.trackingWell') }]
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
