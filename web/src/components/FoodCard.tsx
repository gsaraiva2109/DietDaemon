// A single food summary card, name, source, a per-100g macro mini-grid, and a
// usage footnote. Whole card is clickable into the FoodDetailModal.

import { motion } from 'framer-motion'
import type { FoodDetail } from '@/lib/types'
import { Pill } from './ui'
import { fadeUp } from '@/lib/motion'
import { formatNumber, relativeTime, round } from '@/lib/format'

const SOURCE_LABEL: Record<string, string> = {
  food_library: 'Library',
  openfoodfacts: 'OpenFoodFacts',
  taco: 'TACO',
  usda: 'USDA',
}

export function sourceLabel(source: string): string {
  return SOURCE_LABEL[source] ?? source
}

// kcal + the three macros that fit a compact mini-grid.
const MINI: { key: 'Protein' | 'Carbs' | 'Fat'; label: string }[] = [
  { key: 'Protein', label: 'P' },
  { key: 'Carbs', label: 'C' },
  { key: 'Fat', label: 'F' },
]

export function FoodCard({ food, onClick }: { food: FoodDetail; onClick?: () => void }) {
  const per = food.per_100g
  return (
    <motion.button
      type="button"
      variants={fadeUp}
      onClick={onClick}
      className="group flex w-full flex-col gap-3 rounded-xl border border-line bg-surface p-4 text-left shadow-soft transition hover:shadow-lift"
    >
      <div className="flex items-start justify-between gap-2">
        <p className="min-w-0 truncate font-semibold text-ink">{food.name}</p>
        <Pill tone={food.source === 'food_library' ? 'primary' : 'neutral'}>
          {sourceLabel(food.source)}
        </Pill>
      </div>

      <dl className="grid grid-cols-4 gap-2 border-t border-line pt-3">
        <div>
          <dt className="text-[10px] uppercase tracking-[0.1em] text-muted">kcal</dt>
          <dd className="font-semibold text-ink tnum">{formatNumber(per.Calories)}</dd>
        </div>
        {MINI.map((mm) => (
          <div key={mm.key}>
            <dt className="text-[10px] uppercase tracking-[0.1em] text-muted">{mm.label}</dt>
            <dd className="font-semibold text-ink tnum">{round(per[mm.key])}</dd>
          </div>
        ))}
      </dl>

      <p className="text-[11px] text-muted">
        {food.last_used ? relativeTime(food.last_used) : 'never used'}
        {food.query_count > 0 && (
          <>
            <span className="px-1 text-line">·</span>
            {food.query_count}×
          </>
        )}
      </p>
    </motion.button>
  )
}
