// Duplicate a past meal as today's meal. Two-step picker: pick a day, then a
// meal from that day. Selecting a meal re-logs it via useDuplicateMeal.

import { useEffect, useMemo, useState } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import type { Meal } from '@/lib/types'
import { useMeals, useDuplicateMeal } from '@/lib/queries'
import { useDemo } from '@/lib/demo'
import { Spinner, EmptyState, Pill } from './ui'
import { CloseIcon, ChevronLeft, ChevronRight, CopyIcon } from './icons'
import { clockTime, formatNumber } from '@/lib/format'
import { scaleIn, stagger, fadeUp } from '@/lib/motion'

interface Props {
  onClose: () => void
}

function dayKey(iso: string): string {
  return new Date(iso).toDateString()
}

function dayLabel(iso: string): string {
  const d = new Date(iso)
  const today = new Date()
  const yest = new Date()
  yest.setDate(today.getDate() - 1)
  if (d.toDateString() === today.toDateString()) return 'Today'
  if (d.toDateString() === yest.toDateString()) return 'Yesterday'
  return d.toLocaleDateString(undefined, { weekday: 'long', month: 'long', day: 'numeric' })
}

function mealKcal(meal: Meal): number {
  return meal.Items.reduce((s, it) => s + (it.Macros?.Calories ?? 0), 0)
}

export function DuplicateMealModal({ onClose }: Props) {
  const meals = useMeals(50)
  const duplicate = useDuplicateMeal()
  const { demo } = useDemo()
  const [day, setDay] = useState<string | null>(null)

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  // Group meals by local day, preserving recency order.
  const groups = useMemo(() => {
    const map = new Map<string, Meal[]>()
    for (const m of meals.data ?? []) {
      const k = dayKey(m.At)
      if (!map.has(k)) map.set(k, [])
      map.get(k)!.push(m)
    }
    return [...map.entries()]
  }, [meals.data])

  const selectedMeals = useMemo(
    () => groups.find(([k]) => k === day)?.[1] ?? [],
    [groups, day],
  )

  function pick(meal: Meal) {
    if (demo) return
    duplicate.mutate(meal.ID, { onSuccess: onClose })
  }

  return (
    <AnimatePresence>
      <motion.div
        className="fixed inset-0 grid place-items-center p-4"
        style={{ zIndex: 1500 }}
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
      >
        <div
          className="absolute inset-0 bg-ink/30 backdrop-blur-sm"
          style={{ zIndex: 1400 }}
          onClick={onClose}
        />
        <motion.div
          role="dialog"
          aria-modal="true"
          aria-label="Duplicate a past meal"
          variants={scaleIn}
          initial="hidden"
          animate="show"
          exit="hidden"
          className="relative flex max-h-[80vh] w-full max-w-md flex-col rounded-xl border border-line bg-surface p-6 shadow-lift"
          style={{ zIndex: 1500 }}
        >
          <div className="mb-5 flex items-start justify-between">
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted">
                Duplicate meal
              </p>
              <h2 className="mt-1 text-xl font-bold text-ink">
                {day ? 'Pick a meal' : 'Pick a day'}
              </h2>
            </div>
            <button onClick={onClose} aria-label="Close" className="text-muted hover:text-ink">
              <CloseIcon />
            </button>
          </div>

          {day && (
            <button
              onClick={() => setDay(null)}
              className="mb-3 inline-flex items-center gap-1 self-start text-sm text-muted hover:text-ink"
            >
              <ChevronLeft width={18} height={18} /> Back to days
            </button>
          )}

          {demo && (
            <p className="mb-3 text-xs text-muted">Duplicating is disabled in demo mode.</p>
          )}

          <div className="-mx-1 min-h-0 flex-1 overflow-y-auto px-1">
            {meals.isLoading ? (
              <Spinner label="Loading meals" />
            ) : !groups.length ? (
              <EmptyState
                title="No meals to duplicate"
                hint="Log a meal first, then you can re-use it any day."
                icon={<CopyIcon />}
              />
            ) : !day ? (
              <motion.div
                variants={stagger}
                initial="hidden"
                animate="show"
                className="flex flex-col gap-2"
              >
                {groups.map(([k, dayMeals]) => (
                  <motion.button
                    key={k}
                    variants={fadeUp}
                    onClick={() => setDay(k)}
                    className="group flex items-center justify-between gap-3 rounded-xl border border-line bg-surface px-4 py-3 text-left shadow-soft transition hover:shadow-lift"
                  >
                    <div className="min-w-0">
                      <p className="font-semibold text-ink">{dayLabel(dayMeals[0].At)}</p>
                      <p className="mt-0.5 text-xs text-muted">
                        {dayMeals.length} meal{dayMeals.length === 1 ? '' : 's'}
                      </p>
                    </div>
                    <span className="text-muted transition group-hover:translate-x-0.5 group-hover:text-ink">
                      <ChevronRight />
                    </span>
                  </motion.button>
                ))}
              </motion.div>
            ) : (
              <motion.div
                variants={stagger}
                initial="hidden"
                animate="show"
                className="flex flex-col gap-2"
              >
                {selectedMeals.map((m) => (
                  <motion.button
                    key={m.ID}
                    variants={fadeUp}
                    onClick={() => pick(m)}
                    disabled={demo || duplicate.isPending}
                    className="flex items-center gap-4 rounded-xl border border-line bg-surface px-4 py-3 text-left shadow-soft transition hover:shadow-lift disabled:opacity-50"
                  >
                    <div className="min-w-0 flex-1">
                      <p className="truncate font-semibold text-ink">
                        {m.RawText || 'Logged meal'}
                      </p>
                      <p className="mt-0.5 text-xs text-muted">{clockTime(m.At)}</p>
                    </div>
                    <div className="text-right">
                      <div className="text-base font-bold text-ink tnum">
                        {formatNumber(mealKcal(m))}
                      </div>
                      <div className="text-[10px] uppercase tracking-[0.12em] text-muted">kcal</div>
                    </div>
                  </motion.button>
                ))}
              </motion.div>
            )}
          </div>

          {duplicate.isError && (
            <p className="mt-3 text-sm font-medium text-accent" role="alert">
              {duplicate.error instanceof Error
                ? duplicate.error.message
                : 'Failed to duplicate meal'}
            </p>
          )}

          {duplicate.isPending && (
            <div className="mt-3">
              <Pill tone="primary">Duplicating…</Pill>
            </div>
          )}
        </motion.div>
      </motion.div>
    </AnimatePresence>
  )
}
