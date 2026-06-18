// Compact horizontal rail of the user's most-logged foods. One tap jumps
// straight to the log composer pre-filled with the food name. Embedded on both
// the Dashboard and the Foods browser, so it stays fully self-contained.

import { Link } from 'react-router-dom'
import { useFrequentFoods } from '@/lib/queries'
import type { FoodDetail } from '@/lib/types'
import { Eyebrow } from './ui'
import { formatNumber } from '@/lib/format'

export function FrequentFoods() {
  const { data } = useFrequentFoods(12)
  const foods = data ?? []
  if (!foods.length) return null

  return (
    <section>
      <Eyebrow>Frequent foods</Eyebrow>
      <div className="mt-2 flex gap-2 overflow-x-auto pb-1">
        {foods.map((f: FoodDetail) => (
          <Link
            key={f.food_id}
            to={`/log?text=${encodeURIComponent(f.name)}`}
            prefetch="intent"
            className="inline-flex shrink-0 items-center gap-2 rounded-full border border-line bg-surface px-3.5 py-1.5 text-sm text-ink shadow-soft transition hover:shadow-lift"
          >
            <span className="font-medium">{f.name}</span>
            <span className="text-xs text-muted tnum">{formatNumber(f.per_100g.Calories)} kcal/100g</span>
          </Link>
        ))}
      </div>
    </section>
  )
}
