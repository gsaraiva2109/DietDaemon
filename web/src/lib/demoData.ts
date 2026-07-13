// Sample data for demo mode. Split out of demo.tsx (which only exports the
// DemoProvider component + useDemo hook) so Fast Refresh works there — this
// file exports plain data/functions and is imported by queries.ts.

import type { DailyRollup, Meal, Macros, FoodDetail, MealTemplate, WeightEntry, WeightTrend, BodyCompositionSummary, MeasurementEntry, UserProfile, PendingAlias } from './types'

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

// --- demo foods ------------------------------------------------------------

function fd(name: string, source: string, p100g: Macros, cat = '', brand = ''): FoodDetail {
  return {
    food_id: `${source}:${name}`, name, source, per_100g: p100g, category: cat, brand, barcode: '',
    image_url: '', serving_size: 100, serving_unit: 'g', query_count: 3, last_used: hoursAgo(24),
    in_library: true,
  }
}

export const DEMO_FOODS: FoodDetail[] = [
  fd('Chicken breast, grilled', 'taco', m(165, 31, 0, 3.6, 0), 'Meat'),
  fd('White rice, cooked', 'taco', m(130, 2.7, 28, 0.3, 0.4), 'Grains'),
  fd('Egg, whole', 'openfoodfacts', m(143, 12.6, 1.1, 9.5, 0), 'Dairy & Eggs'),
  fd('Banana', 'taco', m(89, 1.1, 23, 0.3, 2.6), 'Fruit'),
  fd('Whey protein shake', 'food_library', m(120, 24, 3, 1.5, 0.5), 'Supplements'),
  fd('Rolled oats', 'taco', m(389, 16.9, 66, 6.9, 10.6), 'Grains'),
  fd('Latte', 'openfoodfacts', m(48, 2.5, 4.5, 2, 0), 'Beverages'),
  fd('Mixed salad', 'food_library', m(25, 1.2, 3.8, 0.3, 1.8), 'Vegetables'),
]

export function demoFoodSearch(q: string): FoodDetail[] {
  const n = q.toLowerCase()
  return DEMO_FOODS.filter((f) => f.name.toLowerCase().includes(n))
}

// --- demo templates --------------------------------------------------------

export const DEMO_TEMPLATES: MealTemplate[] = [
  {
    id: 'tpl-1', user_id: 'demo', name: 'Grilled chicken + rice',
    items: [
      item('frango grelhado', 200, 'Chicken breast, grilled', 'taco', m(330, 62, 0, 7.2, 0)),
      item('arroz', 150, 'White rice, cooked', 'taco', m(195, 4, 42, 0.5, 0.6)),
    ],
    created_at: hoursAgo(168), last_used: hoursAgo(2),
  },
  {
    id: 'tpl-2', user_id: 'demo', name: 'Protein shake + oats',
    items: [
      item('protein shake', 350, 'Whey protein shake', 'food_library', m(210, 31, 14, 3.5, 1.8)),
      item('oats', 60, 'Rolled oats', 'taco', m(233, 8, 40, 4.3, 6)),
    ],
    created_at: hoursAgo(72), last_used: hoursAgo(26),
  },
]

// --- demo weight -----------------------------------------------------------

const WEIGHT_BASE = 78.5

export const DEMO_WEIGHT: WeightEntry[] = Array.from({ length: 30 }, (_, i) => {
  const d = new Date()
  d.setDate(d.getDate() - 29 + i)
  const noise = (Math.sin(i * 0.7) * 0.4 + (Math.random() - 0.5) * 0.3)
  return {
    id: `w-${i}`, user_id: 'demo', date: d.toISOString().slice(0, 10),
    weight_kg: Math.round((WEIGHT_BASE + noise - i * 0.03) * 10) / 10,
    note: '', created_at: d.toISOString(),
  }
})

export function demoWeightTrend(days: number): WeightTrend[] {
  const out: WeightTrend[] = []
  for (let i = days - 1; i >= 0; i--) {
    const d = new Date()
    d.setDate(d.getDate() - i)
    const raw = WEIGHT_BASE - i * 0.03 + Math.sin(i * 0.7) * 0.4
    out.push({ date: d.toISOString().slice(0, 10), weight_kg: Math.round(raw * 10) / 10, rolling_avg: Math.round((raw - 0.1) * 10) / 10 })
  }
  return out
}

export function demoBodySummary(): BodyCompositionSummary {
  return {
    current_weight_kg: 77.6, start_weight_kg: WEIGHT_BASE,
    change_kg: -0.9, trend_direction: 'down',
    latest_trend_point: { date: new Date().toISOString().slice(0, 10), weight_kg: 77.6, rolling_avg: 77.8 },
  }
}

// --- demo measurements -----------------------------------------------------

export const DEMO_MEASUREMENTS: MeasurementEntry[] = [
  {
    id: 'm-1', user_id: 'demo', date: new Date(Date.now() - 7 * 864e4).toISOString().slice(0, 10),
    waist_cm: 88, hips_cm: 102, chest_cm: 104, left_arm_cm: 34, right_arm_cm: 35,
    left_thigh_cm: 56, right_thigh_cm: 57, note: '', created_at: hoursAgo(168),
  },
]

// --- demo profile ----------------------------------------------------------

export const DEMO_PROFILE: UserProfile = {
  user_id: 'demo', height_cm: 178, birth_date: '1992-03-15', gender: 'male',
  activity_level: 'moderate', goal: 'cut', target_weight_kg: 75,
  weekly_rate: 0.5, onboarded: true,
  created_at: hoursAgo(720), updated_at: hoursAgo(24),
}

// --- demo pending aliases ---------------------------------------------------

export const DEMO_PENDING_ALIASES: PendingAlias[] = [
  {
    id: 'pa-1', user_id: 'demo', phrase: 'frango na chapa', food_id: 'demo-chicken',
    food_name: 'Chicken breast, grilled', match_score: 0.94, created_at: hoursAgo(3),
  },
  {
    id: 'pa-2', user_id: 'demo', phrase: 'arrozinho', food_id: 'demo-rice',
    food_name: 'White rice, cooked', match_score: 0.92, created_at: hoursAgo(20),
  },
]

// --- demo nutrition source precedence ---------------------------------------

// Empty = not customized, same as a fresh backend user; the settings page
// falls back to NUTRITION_SOURCES' default order.
export const DEMO_PRECEDENCE: string[] = []

// --- demo AI key ------------------------------------------------------------

export const DEMO_AI_KEY = { has_key: true, provider: 'anthropic' }

// --- demo Hevy key ----------------------------------------------------------

export const DEMO_HEVY_KEY = { has_key: false }
