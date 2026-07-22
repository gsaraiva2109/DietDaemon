import { describe, expect, it } from 'vitest'
import { gramsFor, unitOptionsFor, GRAMS_UNIT_ID } from '@/lib/servingUnits'
import type { FoodDetail } from '@/lib/types'

function food(overrides: Partial<FoodDetail> = {}): FoodDetail {
  return {
    food_id: 'f1',
    name: 'Egg',
    source: 'usda',
    per_100g: { Calories: 143, Protein: 13, Carbs: 1, Fat: 10, Fiber: 0 },
    category: '',
    brand: '',
    barcode: '',
    image_url: '',
    serving_size: 0,
    serving_unit: '',
    query_count: 0,
    last_used: '',
    in_library: true,
    volume_units_eligible: false,
    ...overrides,
  }
}

describe('unitOptionsFor', () => {
  it('always offers grams first', () => {
    const opts = unitOptionsFor(food())
    expect(opts[0]).toEqual({ id: GRAMS_UNIT_ID, label: 'g', grams: 1 })
  })

  it('includes system and custom serving units', () => {
    const f = food({ serving_units: [{ id: 'u1', label: '1 large egg', grams: 50, custom: false }] })
    const opts = unitOptionsFor(f)
    expect(opts).toContainEqual({ id: 'u1', label: '1 large egg', grams: 50 })
  })

  it('adds generic volume units only when volume_units_eligible', () => {
    expect(unitOptionsFor(food({ volume_units_eligible: false })).some((o) => o.id === 'tbsp')).toBe(false)
    const opts = unitOptionsFor(food({ volume_units_eligible: true }))
    expect(opts).toContainEqual({ id: 'tbsp', label: 'tbsp', grams: 15 })
    expect(opts).toContainEqual({ id: 'cup', label: 'cup', grams: 240 })
  })
})

describe('gramsFor', () => {
  it('returns quantity directly for the grams unit', () => {
    expect(gramsFor({ food: food(), unitID: GRAMS_UNIT_ID, quantity: 75 })).toBe(75)
  })

  it('multiplies quantity by the selected unit grams', () => {
    const f = food({ serving_units: [{ id: 'u1', label: '1 large egg', grams: 50, custom: false }] })
    expect(gramsFor({ food: f, unitID: 'u1', quantity: 3 })).toBe(150)
  })

  it('falls back to quantity if the unit id is unknown (e.g. stale selection)', () => {
    expect(gramsFor({ food: food(), unitID: 'missing', quantity: 42 })).toBe(42)
  })
})
