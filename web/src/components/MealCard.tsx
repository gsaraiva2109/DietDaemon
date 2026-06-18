// A single meal summary row — raw text, time, total calories, and provenance
// chips (parser tier + confidence). Used in the dashboard timeline and history.

import { Link } from 'react-router-dom'
import { motion } from 'framer-motion'
import type { Meal } from '@/lib/types'
import { Pill } from './ui'
import { ChevronRight } from './icons'
import { fadeUp } from '@/lib/motion'
import { clockTime, confidenceLabel, formatNumber, tierLabel } from '@/lib/format'

export function MealCard({ meal, linkTo }: { meal: Meal; linkTo?: string }) {
  const total = meal.Items.reduce((s, it) => s + (it.Macros?.Calories ?? 0), 0)
  const conf = confidenceLabel(meal.Confidence)
  const body = (
    <motion.div
      variants={fadeUp}
      className="group flex items-center gap-4 rounded-xl border border-line bg-surface px-4 py-3.5 shadow-soft transition hover:shadow-lift"
    >
      <div className="min-w-0 flex-1">
        <p className="truncate font-semibold text-ink">{meal.RawText || 'Logged meal'}</p>
        <div className="mt-1.5 flex flex-wrap items-center gap-1.5 text-xs">
          <span className="text-muted">{clockTime(meal.At)}</span>
          <span className="text-line">·</span>
          <span className="text-muted">
            {meal.Items.length} item{meal.Items.length === 1 ? '' : 's'}
          </span>
          <Pill tone={meal.ParserTier === 2 ? 'accent' : 'primary'}>{tierLabel(meal.ParserTier)}</Pill>
          {conf !== 'high' && <Pill tone="muted">{conf} confidence</Pill>}
        </div>
      </div>
      <div className="text-right">
        <div className="text-lg font-bold text-ink tnum">{formatNumber(total)}</div>
        <div className="text-[11px] uppercase tracking-[0.12em] text-muted">kcal</div>
      </div>
      {linkTo && (
        <span className="text-muted transition group-hover:translate-x-0.5 group-hover:text-ink">
          <ChevronRight />
        </span>
      )}
    </motion.div>
  )

  return linkTo ? (
    <Link to={linkTo} prefetch="intent" className="block">
      {body}
    </Link>
  ) : (
    body
  )
}
