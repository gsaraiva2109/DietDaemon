// Demo mode: fills the whole UI with realistic sample data so it never looks
// empty while testing, with no backend running. Toggled from the nav, persisted
// in localStorage. The query hooks (queries.ts) read `useDemo()` and return
// this sample data instead of hitting the API.

import { createContext, use, useState, type ReactNode } from 'react'
import type {
  BodyCompositionSummary,
  DailyRollup,
  FoodDetail,
  Macros,
  Meal,
  MealTemplate,
  MeasurementEntry,
  UserProfile,
  WeightEntry,
  WeightTrend,
} from './types'

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

function isoDaysAgo(n: number): string {
  const d = new Date()
  d.setDate(d.getDate() - n)
  return d.toISOString().slice(0, 10)
}

// --- Phase 2: Foods --------------------------------------------------------

function food(
  id: string,
  name: string,
  source: string,
  per100: Macros,
  category: string,
  queryCount: number,
  lastUsedDaysAgo: number,
  aliases: string[] = [],
): FoodDetail {
  return {
    food_id: id, name, source, per_100g: per100, category, brand: '', barcode: '',
    image_url: '', serving_size: 100, serving_unit: 'g', query_count: queryCount,
    last_used: isoDaysAgo(lastUsedDaysAgo),
    aliases: aliases.map((a) => ({ food_id: id, alias: a, normalized: a.toLowerCase() })),
  }
}

export const DEMO_FOODS: FoodDetail[] = [
  food('taco:1', 'Chicken breast, grilled', 'taco', m(165, 31, 0, 3.6, 0), 'protein', 42, 0, ['frango', 'frango grelhado']),
  food('taco:2', 'White rice, cooked', 'taco', m(130, 2.7, 28, 0.3, 0.4), 'grain', 38, 0, ['arroz', 'arroz branco']),
  food('taco:3', 'Black beans, cooked', 'taco', m(132, 8.9, 24, 0.5, 8.7), 'legume', 27, 1, ['feijão', 'feijao preto']),
  food('off:4', 'Egg, whole', 'openfoodfacts', m(143, 13, 1.1, 9.5, 0), 'protein', 31, 0, ['ovo', 'ovos']),
  food('taco:5', 'Banana', 'taco', m(89, 1.1, 23, 0.3, 2.6), 'fruit', 24, 1, ['banana']),
  food('food_library:6', 'Whey protein', 'food_library', m(400, 80, 8, 6, 2), 'supplement', 22, 0, ['whey', 'protein shake']),
  food('taco:7', 'Rolled oats', 'taco', m(389, 17, 66, 7, 10), 'grain', 19, 2, ['aveia', 'oats']),
  food('off:8', 'Greek yogurt, plain', 'openfoodfacts', m(59, 10, 3.6, 0.4, 0), 'dairy', 18, 1, ['iogurte', 'yogurt']),
  food('taco:9', 'Sweet potato, cooked', 'taco', m(86, 1.6, 20, 0.1, 3), 'vegetable', 15, 3, ['batata doce']),
  food('taco:10', 'Salmon, grilled', 'taco', m(208, 20, 0, 13, 0), 'protein', 13, 4, ['salmão']),
  food('off:11', 'Almonds', 'openfoodfacts', m(579, 21, 22, 50, 12), 'nuts', 11, 2, ['amêndoas', 'almonds']),
  food('taco:12', 'Broccoli, cooked', 'taco', m(35, 2.4, 7, 0.4, 3.3), 'vegetable', 10, 2, ['brócolis']),
  food('food_library:13', 'Peanut butter', 'food_library', m(588, 25, 20, 50, 6), 'spread', 9, 3, ['pasta de amendoim']),
  food('taco:14', 'Apple', 'taco', m(52, 0.3, 14, 0.2, 2.4), 'fruit', 8, 1, ['maçã', 'maca']),
  food('off:15', 'Whole wheat bread', 'openfoodfacts', m(247, 13, 41, 3.4, 7), 'grain', 7, 2, ['pão integral']),
  food('taco:16', 'Avocado', 'taco', m(160, 2, 9, 15, 7), 'fruit', 6, 4, ['abacate']),
  food('food_library:17', 'Olive oil', 'food_library', m(884, 0, 0, 100, 0), 'fat', 5, 1, ['azeite']),
  food('taco:18', 'Ground beef, lean', 'taco', m(250, 26, 0, 15, 0), 'protein', 5, 5, ['carne moída']),
]

export function demoFoodSearch(q: string): FoodDetail[] {
  const n = q.trim().toLowerCase()
  if (!n) return DEMO_FOODS
  return DEMO_FOODS.filter(
    (f) => f.name.toLowerCase().includes(n) || (f.aliases ?? []).some((a) => a.normalized.includes(n)),
  )
}

// --- Phase 3: Templates ----------------------------------------------------

export const DEMO_TEMPLATES: MealTemplate[] = [
  {
    id: 'tpl-1', user_id: 'demo', name: 'Breakfast — eggs & oats',
    created_at: hoursAgo(72), last_used: hoursAgo(20),
    items: [item('3 eggs', 150, 'Egg, whole', 'openfoodfacts', m(214, 19, 1.6, 14, 0)),
      item('oats', 60, 'Rolled oats', 'taco', m(233, 8, 40, 4.3, 6))],
  },
  {
    id: 'tpl-2', user_id: 'demo', name: 'Post-workout shake',
    created_at: hoursAgo(96), last_used: hoursAgo(5),
    items: [item('whey', 40, 'Whey protein', 'food_library', m(160, 32, 3.2, 2.4, 0.8)),
      item('banana', 120, 'Banana', 'taco', m(107, 1.3, 27, 0.4, 3.1))],
  },
  {
    id: 'tpl-3', user_id: 'demo', name: 'Lunch — chicken & rice',
    created_at: hoursAgo(120), last_used: hoursAgo(28),
    items: [item('frango', 200, 'Chicken breast, grilled', 'taco', m(330, 62, 0, 7.2, 0)),
      item('arroz', 150, 'White rice, cooked', 'taco', m(195, 4, 42, 0.5, 0.6)),
      item('feijão', 100, 'Black beans, cooked', 'taco', m(132, 8.9, 24, 0.5, 8.7))],
  },
]

// --- Phase 4: Body ---------------------------------------------------------

// 90 days of realistic weight, drifting down ~3kg with daily noise.
export const DEMO_WEIGHT: WeightEntry[] = (() => {
  const out: WeightEntry[] = []
  for (let i = 90; i >= 0; i--) {
    const base = 82 - (90 - i) * 0.033
    const noise = Math.sin(i * 1.7) * 0.35 + Math.cos(i * 0.6) * 0.2
    out.push({
      id: `w-${i}`, user_id: 'demo', date: isoDaysAgo(i),
      weight_kg: Math.round((base + noise) * 10) / 10, note: '', created_at: isoDaysAgo(i),
    })
  }
  return out
})()

export function demoWeightTrend(days: number): WeightTrend[] {
  const slice = DEMO_WEIGHT.slice(Math.max(0, DEMO_WEIGHT.length - days))
  return slice.map((e, idx) => {
    const from = Math.max(0, idx - 6)
    const window = slice.slice(from, idx + 1)
    const avg = window.reduce((s, w) => s + w.weight_kg, 0) / window.length
    return { date: e.date, weight_kg: e.weight_kg, rolling_avg: Math.round(avg * 100) / 100 }
  })
}

export function demoBodySummary(): BodyCompositionSummary {
  const trend = demoWeightTrend(14)
  const current = DEMO_WEIGHT[DEMO_WEIGHT.length - 1].weight_kg
  const start = DEMO_WEIGHT[0].weight_kg
  return {
    current_weight_kg: current, start_weight_kg: start,
    change_kg: Math.round((current - start) * 10) / 10,
    trend_direction: 'down', latest_trend_point: trend[trend.length - 1] ?? null,
  }
}

export const DEMO_MEASUREMENTS: MeasurementEntry[] = (() => {
  const out: MeasurementEntry[] = []
  for (let wk = 12; wk >= 0; wk--) {
    const t = (12 - wk) / 12
    out.push({
      id: `meas-${wk}`, user_id: 'demo', date: isoDaysAgo(wk * 7),
      waist_cm: Math.round((88 - t * 5) * 10) / 10,
      hips_cm: Math.round((102 - t * 2) * 10) / 10,
      chest_cm: Math.round((104 + t * 1) * 10) / 10,
      left_arm_cm: Math.round((36 + t * 0.8) * 10) / 10,
      right_arm_cm: Math.round((36.5 + t * 0.8) * 10) / 10,
      left_thigh_cm: Math.round((58 - t * 1.5) * 10) / 10,
      right_thigh_cm: Math.round((58.5 - t * 1.5) * 10) / 10,
      note: '', created_at: isoDaysAgo(wk * 7),
    })
  }
  return out
})()

// --- Phase 5: Profile ------------------------------------------------------

export const DEMO_PROFILE: UserProfile = {
  user_id: 'demo', height_cm: 178, birth_date: '1994-05-12', gender: 'male',
  activity_level: 'moderate', goal: 'cut', target_weight_kg: 76, weekly_rate: 0.5,
  onboarded: true, created_at: hoursAgo(2000), updated_at: hoursAgo(48),
}
