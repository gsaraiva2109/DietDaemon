// Small presentation helpers for macros, dates, and parser provenance.

import type { Macros, MacroKey, ParserTier } from './types'
import type { TFunction } from 'i18next'

export function round(n: number, places = 0): number {
  const f = 10 ** places
  return Math.round(n * f) / f
}

export function macroValue(m: Macros, key: MacroKey): number {
  return m[key] ?? 0
}

/** Scale per-100g macros to a given gram amount. */
export function scaleMacros(per100g: Macros, grams: number): Macros {
  const f = grams / 100
  return {
    Calories: per100g.Calories * f,
    Protein: per100g.Protein * f,
    Carbs: per100g.Carbs * f,
    Fat: per100g.Fat * f,
    Fiber: per100g.Fiber * f,
  }
}

/** Sum a list of macro sets into a running total. */
export function sumMacros(list: Macros[]): Macros {
  return list.reduce(
    (sum, m) => ({
      Calories: sum.Calories + m.Calories,
      Protein: sum.Protein + m.Protein,
      Carbs: sum.Carbs + m.Carbs,
      Fat: sum.Fat + m.Fat,
      Fiber: sum.Fiber + m.Fiber,
    }),
    { Calories: 0, Protein: 0, Carbs: 0, Fat: 0, Fiber: 0 },
  )
}

/** Remaining-to-target, clamped at 0 (the hero number). */
export function remaining(consumed: number, target: number): number {
  return Math.max(0, target - consumed)
}

/** Progress 0..1 toward target; 0 when no target set. */
export function progress(consumed: number, target: number): number {
  if (target <= 0) return 0
  return Math.min(1, consumed / target)
}

export function pct(consumed: number, target: number): number {
  return Math.round(progress(consumed, target) * 100)
}

export function isOverTarget(consumed: number, target: number): boolean {
  return target > 0 && consumed > target
}

export function formatNumber(n: number): string {
  return new Intl.NumberFormat(undefined, { maximumFractionDigits: 0 }).format(n)
}

export function formatGrams(n: number): string {
  return `${round(n)}g`
}

export function tierLabel(tier: ParserTier, t: TFunction): string {
  return t(`history.tier${({ 0: 'Exact', 1: 'Matched', 2: 'AI' } as const)[tier] ?? 'Unknown'}`)
}

export function confidenceLabel(c: number): 'high' | 'medium' | 'low' {
  if (c >= 0.8) return 'high'
  if (c >= 0.5) return 'medium'
  return 'low'
}

export function confidenceTier(c: number): 'high' | 'medium' | 'low' {
  if (c >= 0.85) return 'high'
  if (c >= 0.6) return 'medium'
  return 'low'
}

export function confidenceColor(c: number): string {
  if (c >= 0.85) return ''
  if (c >= 0.6) return 'text-amber-600 dark:text-amber-400'
  return 'text-red-500 dark:text-red-400'
}

export function relativeTime(iso: string, t: TFunction, locale: string): string {
  const then = new Date(iso).getTime()
  const diffMin = Math.round((Date.now() - then) / 60000)
  if (diffMin < 1) return t('common.justNow')
  if (diffMin < 60) return t('common.minutesAgo', { count: diffMin })
  const h = Math.round(diffMin / 60)
  if (h < 24) return t('common.hoursAgo', { count: h })
  return new Date(iso).toLocaleDateString(locale, { month: 'short', day: 'numeric' })
}

export function clockTime(iso: string, locale: string): string {
  return new Date(iso).toLocaleTimeString(locale, { hour: 'numeric', minute: '2-digit' })
}

/** Read a DESIGN.md macro color token off the document for inline SVG fills. */
export function cssVar(name: string): string {
  if (typeof window === 'undefined') return ''
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
}
