// History, recent meals with search, parser-tier filtering, and day grouping.

import { useMemo, useState } from 'react'
import { motion } from 'framer-motion'
import { useTranslation } from 'react-i18next'
import { useMeals } from '@/lib/queries'
import { MealCard } from '@/components/MealCard'
import { PageHeader } from '@/components/PageHeader'
import { EmptyState, Spinner } from '@/components/ui'
import { SearchIcon } from '@/components/icons'
import { stagger, fadeUp } from '@/lib/motion'
import { tierLabel } from '@/lib/format'
import type { Meal, ParserTier } from '@/lib/types'

type TierFilter = 'all' | ParserTier

function dayKey(iso: string, locale: string): string {
  return new Date(iso).toLocaleDateString(locale, { weekday: 'long', month: 'long', day: 'numeric' })
}

function relativeDayLabel(iso: string, t: (key: string) => string, locale: string): string {
  const d = new Date(iso)
  const today = new Date()
  const yest = new Date()
  yest.setDate(today.getDate() - 1)
  if (d.toDateString() === today.toDateString()) return t('history.today')
  if (d.toDateString() === yest.toDateString()) return t('history.yesterday')
  return dayKey(iso, locale)
}

export function History() {
  const { t, i18n } = useTranslation()
  const meals = useMeals(50)
  const [q, setQ] = useState('')
  const [tier, setTier] = useState<TierFilter>('all')

  const filtered = useMemo(() => {
    const list = meals.data ?? []
    const needle = q.trim().toLowerCase()
    return list.filter((m) => {
      if (tier !== 'all' && m.ParserTier !== tier) return false
      if (!needle) return true
      const hay = `${m.RawText} ${m.Items.map((i) => i.Match.Name).join(' ')}`.toLowerCase()
      return hay.includes(needle)
    })
  }, [meals.data, q, tier])

  const groups = useMemo(() => {
    const map = new Map<string, Meal[]>()
    for (const m of filtered) {
      const k = relativeDayLabel(m.At, t, i18n.language)
      if (!map.has(k)) map.set(k, [])
      map.get(k)!.push(m)
    }
    return [...map.entries()]
  }, [filtered, t, i18n.language])

  return (
    <div>
      <PageHeader eyebrow={t('history.eyebrow')} title={t('history.title')} />

      {/* Search + tier filter */}
      <div className="mb-6 flex flex-wrap items-center gap-3">
        <div className="relative min-w-56 flex-1">
          <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-muted">
            <SearchIcon width={18} height={18} />
          </span>
          <input
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder={t('history.searchPlaceholder')}
            aria-label={t('history.searchAriaLabel')}
            className="w-full rounded-full border border-line bg-surface py-2.5 pl-10 pr-4 text-ink outline-none transition focus:border-primary"
          />
        </div>
        <div className="flex gap-1 rounded-full border border-line bg-surface p-1">
          {(['all', 0, 1, 2] as TierFilter[]).map((tf) => (
            <button
              key={String(tf)}
              onClick={() => setTier(tf)}
              className={`rounded-full px-3 py-1 text-sm font-medium transition ${
                tier === tf ? 'bg-primary-soft text-primary' : 'text-muted hover:text-ink'
              }`}
            >
              {tf === 'all' ? t('history.allTier') : tierLabel(tf, t)}
            </button>
          ))}
        </div>
      </div>

      {meals.isLoading ? (
        <Spinner />
      ) : !meals.data?.length ? (
        <EmptyState title={t('history.emptyTitle')} hint={t('history.emptyHint')} />
      ) : !filtered.length ? (
        <EmptyState title={t('history.noMatchesTitle')} hint={t('history.noMatchesHint')} />
      ) : (
        <motion.div variants={stagger} initial="hidden" animate="show" className="flex flex-col gap-7">
          {groups.map(([day, dayMeals]) => (
            <div key={day}>
              <h2 className="mb-2.5 text-xs font-semibold uppercase tracking-[0.14em] text-muted">{day}</h2>
              <div className="flex flex-col gap-2.5">
                {dayMeals.map((m) => (
                  <motion.div key={m.ID} variants={fadeUp}>
                    <MealCard meal={m} linkTo={`/history/${m.ID}`} />
                  </motion.div>
                ))}
              </div>
            </div>
          ))}
        </motion.div>
      )}
    </div>
  )
}
