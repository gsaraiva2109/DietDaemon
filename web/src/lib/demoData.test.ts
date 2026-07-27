import { expect, test } from 'vitest'
import { demoFoodSearch, demoWeightTrend, DEMO_FOODS } from './demoData'

test('demoFoodSearch matches case-insensitively on name', () => {
  expect(demoFoodSearch('chicken')).toEqual(
    DEMO_FOODS.filter((f) => f.name.toLowerCase().includes('chicken')),
  )
  expect(demoFoodSearch('CHICKEN')).toEqual(demoFoodSearch('chicken'))
  expect(demoFoodSearch('no-such-food-xyz')).toEqual([])
})

test('demoWeightTrend returns one point per day, oldest first', () => {
  const trend = demoWeightTrend(5)
  expect(trend).toHaveLength(5)
  // Ascending dates: each entry's date is >= the previous one's.
  for (let i = 1; i < trend.length; i++) {
    expect(trend[i].date >= trend[i - 1].date).toBe(true)
  }
  for (const point of trend) {
    expect(typeof point.weight_kg).toBe('number')
    expect(typeof point.rolling_avg).toBe('number')
  }
})
