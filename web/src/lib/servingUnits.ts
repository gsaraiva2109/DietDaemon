// Unit math shared by the food picker (#134): grams stays authoritative,
// unit/quantity are how the user thinks about and reviews a portion.

import type { FoodDetail } from './types'

export const GRAMS_UNIT_ID = 'g'

// Generic approximate volume units (density-1.0), offered only when the food
// is volume_units_eligible. Mirrors internal/parser/normalize's unitAliases
// gram values so a logged "2 tbsp" here means the same grams the backend
// would compute for that unit elsewhere.
const GENERIC_VOLUME_UNITS: { id: string; grams: number }[] = [
  { id: 'tbsp', grams: 15 },
  { id: 'tsp', grams: 5 },
  { id: 'cup', grams: 240 },
  { id: 'oz', grams: 28.3495 },
]

export type SelectedFood = { food: FoodDetail; unitID: string; quantity: number }

// unitOptionsFor lists every way this food can be logged: grams (always
// first), its system + custom serving_units, then generic volume units if
// eligible. id is unique per food (serving_units are food-scoped already).
export function unitOptionsFor(food: FoodDetail): { id: string; label: string; grams: number }[] {
  const opts = [{ id: GRAMS_UNIT_ID, label: GRAMS_UNIT_ID, grams: 1 }]
  for (const u of food.serving_units ?? []) opts.push({ id: u.id, label: u.label, grams: u.grams })
  if (food.volume_units_eligible) {
    for (const u of GENERIC_VOLUME_UNITS) opts.push({ ...u, label: u.id })
  }
  return opts
}

export function gramsFor(s: SelectedFood): number {
  if (s.unitID === GRAMS_UNIT_ID) return s.quantity
  const unit = unitOptionsFor(s.food).find((u) => u.id === s.unitID)
  return unit ? s.quantity * unit.grams : s.quantity
}

export { GENERIC_VOLUME_UNITS }
