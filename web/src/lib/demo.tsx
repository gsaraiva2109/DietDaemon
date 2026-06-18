// Demo mode: fills the whole UI with realistic sample data so it never looks
// empty while testing, with no backend running. Toggled from the nav, persisted
// in localStorage. The query hooks (queries.ts) read `useDemo()` and return
// this sample data instead of hitting the API.

import { createContext, use, useState, type ReactNode } from 'react'
import type { DailyRollup, Meal, Macros } from './types'

const KEY = 'dd.demo'

interface DemoValue {
  demo: boolean
  setDemo: (v: boolean) => void
}
const DemoContext = createContext<DemoValue | null>(null)

export function DemoProvider({ children }: { children: ReactNode }) {
  const [demo, set] = useState<boolean>(() => localStorage.getItem(KEY) === '1')
  function setDemo(v: boolean) {
    set(v)
    localStorage.setItem(KEY, v ? '1' : '0')
  }
  return <DemoContext value={{ demo, setDemo }}>{children}</DemoContext>
}

export function useDemo(): DemoValue {
  const ctx = use(DemoContext)
  if (!ctx) throw new Error('useDemo must be used within DemoProvider')
  return ctx
}

// --- sample data -----------------------------------------------------------

function m(c: number, p: number, cb: number, f: number, fi: number): Macros {
  return { Calories: c, Protein: p, Carbs: cb, Fat: f, Fiber: fi }
}

export const DEMO_TARGETS: Macros = m(3000, 180, 360, 90, 38)
export const DEMO_CONSUMED: Macros = m(1840, 132, 196, 51, 21)

export const DEMO_MEALS: Meal[] = [
  {
    ID: 'demo-1', UserID: 'demo', At: hoursAgo(1), RawText: '200g frango grelhado, 150g arroz, salada',
    Confidence: 0.94, ParserTier: 0, CreatedAt: hoursAgo(1),
    Items: [
      item('frango grelhado', 200, 'Chicken breast, grilled', 'taco', m(330, 62, 0, 7.2, 0)),
      item('arroz', 150, 'White rice, cooked', 'taco', m(195, 4, 42, 0.5, 0.6)),
      item('salada', 80, 'Mixed salad', 'food_library', m(20, 1, 3, 0.2, 1.4)),
    ],
  },
  {
    ID: 'demo-2', UserID: 'demo', At: hoursAgo(4), RawText: '3 eggs and a banana',
    Confidence: 0.78, ParserTier: 1, CreatedAt: hoursAgo(4),
    Items: [
      item('3 eggs', 150, 'Egg, whole', 'openfoodfacts', m(214, 19, 1.6, 14, 0)),
      item('banana', 120, 'Banana', 'taco', m(107, 1.3, 27, 0.4, 3.1)),
    ],
  },
  {
    ID: 'demo-3', UserID: 'demo', At: hoursAgo(7), RawText: 'whey protein shake with oats',
    Confidence: 0.55, ParserTier: 2, CreatedAt: hoursAgo(7),
    Items: [
      item('protein shake', 350, 'Whey protein shake', 'food_library', m(210, 31, 14, 3.5, 1.8)),
      item('oats', 60, 'Rolled oats', 'taco', m(233, 8, 40, 4.3, 6),),
    ],
  },
  {
    ID: 'demo-4', UserID: 'demo', At: hoursAgo(10), RawText: 'coffee with milk',
    Confidence: 0.88, ParserTier: 0, CreatedAt: hoursAgo(10),
    Items: [item('coffee with milk', 200, 'Latte', 'openfoodfacts', m(96, 5, 9, 4, 0))],
  },
]

export function demoRange(start: string, end: string): DailyRollup[] {
  const days: DailyRollup[] = []
  const s = new Date(start)
  const e = new Date(end)
  for (let d = new Date(s); d <= e; d.setDate(d.getDate() + 1)) {
    const seed = d.getDate()
    const w = (n: number, spread = 0.28) => Math.round(n * (1 - spread + ((seed * 37) % 100) / 100 * spread * 2))
    days.push({
      UserID: 'demo', Date: d.toISOString().slice(0, 10),
      Consumed: m(w(2850), w(168), w(330), w(84), w(33)), Targets: DEMO_TARGETS,
    })
  }
  return days
}

export function demoToday(): DailyRollup {
  return { UserID: 'demo', Date: new Date().toISOString().slice(0, 10), Consumed: DEMO_CONSUMED, Targets: DEMO_TARGETS }
}

function hoursAgo(h: number): string {
  return new Date(Date.now() - h * 3600e3).toISOString()
}

function item(phrase: string, grams: number, name: string, source: string, macros: Macros) {
  return {
    Parsed: { RawPhrase: phrase, Quantity: grams, Unit: 'g', NormalizedGrams: grams, Locale: 'pt-BR' },
    Match: { FoodID: `${source}:${name}`, Name: name, Source: source, Per100g: macros, MatchScore: 0.9 },
    Macros: macros,
  }
}
