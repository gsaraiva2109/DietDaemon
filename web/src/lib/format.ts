// Small presentation helpers for macros, dates, and parser provenance.

import type { Macros, MacroKey, ParserTier } from './types'

export function round(n: number, places = 0): number {
  const f = 10 ** places
  return Math.round(n * f) / f
}

export function macroValue(m: Macros, key: MacroKey): number {
  return m[key] ?? 0
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

const TIER_LABEL: Record<ParserTier, string> = {
  0: 'Exact',
  1: 'Matched',
  2: 'AI',
}
export function tierLabel(t: ParserTier): string {
  return TIER_LABEL[t] ?? 'Unknown'
}

export function confidenceLabel(c: number): 'high' | 'medium' | 'low' {
  if (c >= 0.8) return 'high'
  if (c >= 0.5) return 'medium'
  return 'low'
}

export function relativeTime(iso: string): string {
  const then = new Date(iso).getTime()
  const diffMin = Math.round((Date.now() - then) / 60000)
  if (diffMin < 1) return 'just now'
  if (diffMin < 60) return `${diffMin}m ago`
  const h = Math.round(diffMin / 60)
  if (h < 24) return `${h}h ago`
  return new Date(iso).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
}

export function clockTime(iso: string): string {
  return new Date(iso).toLocaleTimeString(undefined, { hour: 'numeric', minute: '2-digit' })
}

/** Read a DESIGN.md macro color token off the document for inline SVG fills. */
export function cssVar(name: string): string {
  if (typeof window === 'undefined') return ''
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
}
