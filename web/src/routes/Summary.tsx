// Summary — period rollup: averages, target adherence, best/hardest days, and
// per-macro average vs target. Uses GET /rollups/range.

import { useMemo, useState } from 'react'
import { motion } from 'framer-motion'
import { useRange } from '@/lib/queries'
import { PageHeader } from '@/components/PageHeader'
import { MacroBar } from '@/components/MacroBar'
import { ExportModal } from '@/components/ExportModal'
import { DownloadIcon } from '@/components/icons'
import { Button, Card, Eyebrow, EmptyState, Spinner } from '@/components/ui'
import { MACRO_KEYS, MACRO_META, type Macros, type DailyRollup } from '@/lib/types'
import { cssVar, formatNumber } from '@/lib/format'
import { stagger, fadeUp } from '@/lib/motion'

function isoDaysAgo(n: number): string {
  const d = new Date()
  d.setDate(d.getDate() - n)
  return d.toISOString().slice(0, 10)
}

const ZERO: Macros = { Calories: 0, Protein: 0, Carbs: 0, Fat: 0, Fiber: 0 }

export function Summary() {
  const [days, setDays] = useState(7)
  const [exporting, setExporting] = useState(false)
  const range = useRange(isoDaysAgo(days - 1), isoDaysAgo(0))

  const stats = useMemo(() => compute(range.data ?? []), [range.data])

  return (
    <div>
      <PageHeader eyebrow="Summary" title="Your period">
        <div className="flex items-center gap-2">
          <div className="flex gap-1 rounded-full border border-line bg-surface p-1">
            {[7, 14, 30].map((d) => (
              <button
                key={d}
                onClick={() => setDays(d)}
                className={`rounded-full px-3 py-1 text-sm font-medium transition ${
                  days === d ? 'bg-primary-soft text-primary' : 'text-muted hover:text-ink'
                }`}
              >
                {d}d
              </button>
            ))}
          </div>
          <Button variant="ghost" onClick={() => setExporting(true)} className="px-3 py-1.5 text-xs">
            <DownloadIcon width={15} height={15} /> Export
          </Button>
        </div>
      </PageHeader>

      {exporting && <ExportModal onClose={() => setExporting(false)} />}

      {range.isLoading ? (
        <Spinner />
      ) : !stats ? (
        <EmptyState title="No data in range" hint="Log meals across a few days, or turn on Demo mode." />
      ) : (
        <motion.div variants={stagger} initial="hidden" animate="show" className="flex flex-col gap-5">
          {/* Stat tiles */}
          <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
            <Tile label="Avg calories / day" value={formatNumber(stats.avg.Calories)} unit="kcal" />
            <Tile label="Avg protein / day" value={formatNumber(stats.avg.Protein)} unit="g" />
            <Tile label="Days on target" value={`${stats.onTarget}`} unit={`of ${stats.logged}`} />
            <Tile label="Calorie adherence" value={`${stats.adherence}`} unit="%" />
          </div>

          {/* Per-macro avg vs target */}
          <Card className="p-5">
            <Eyebrow>Average vs target</Eyebrow>
            <div className="mt-4 flex flex-col gap-5">
              {MACRO_KEYS.map((k) => (
                <MacroBar
                  key={k}
                  consumed={stats.avg[k]}
                  target={stats.target[k]}
                  label={MACRO_META[k].label}
                  unit={MACRO_META[k].unit}
                  color={cssVar(MACRO_META[k].colorVar)}
                />
              ))}
            </div>
          </Card>

          {/* Best / hardest day */}
          <div className="grid gap-4 sm:grid-cols-2">
            <motion.div variants={fadeUp}>
              <Card className="p-5">
                <Eyebrow>Closest to target</Eyebrow>
                <p className="mt-2 text-lg font-bold text-ink">{stats.best?.label ?? '—'}</p>
                <p className="text-sm text-muted">{stats.best ? `${formatNumber(stats.best.kcal)} kcal` : 'No data'}</p>
              </Card>
            </motion.div>
            <motion.div variants={fadeUp}>
              <Card className="p-5">
                <Eyebrow>Furthest from target</Eyebrow>
                <p className="mt-2 text-lg font-bold text-ink">{stats.worst?.label ?? '—'}</p>
                <p className="text-sm text-muted">{stats.worst ? `${formatNumber(stats.worst.kcal)} kcal` : 'No data'}</p>
              </Card>
            </motion.div>
          </div>
        </motion.div>
      )}
    </div>
  )
}

function Tile({ label, value, unit }: { label: string; value: string; unit: string }) {
  return (
    <motion.div variants={fadeUp}>
      <Card className="p-5">
        <Eyebrow>{label}</Eyebrow>
        <div className="mt-2 flex items-baseline gap-1">
          <span className="text-3xl font-extrabold text-ink tnum">{value}</span>
          <span className="text-sm text-muted">{unit}</span>
        </div>
      </Card>
    </motion.div>
  )
}

interface Stats {
  avg: Macros
  target: Macros
  logged: number
  onTarget: number
  adherence: number
  best?: { label: string; kcal: number }
  worst?: { label: string; kcal: number }
}

function compute(rollups: DailyRollup[]): Stats | null {
  const logged = rollups.filter((r) => r.Consumed.Calories > 0)
  if (!logged.length) return null

  const sum = logged.reduce<Macros>(
    (a, r) => ({
      Calories: a.Calories + r.Consumed.Calories,
      Protein: a.Protein + r.Consumed.Protein,
      Carbs: a.Carbs + r.Consumed.Carbs,
      Fat: a.Fat + r.Consumed.Fat,
      Fiber: a.Fiber + r.Consumed.Fiber,
    }),
    { ...ZERO },
  )
  const n = logged.length
  const avg: Macros = {
    Calories: sum.Calories / n,
    Protein: sum.Protein / n,
    Carbs: sum.Carbs / n,
    Fat: sum.Fat / n,
    Fiber: sum.Fiber / n,
  }
  const target = logged[logged.length - 1].Targets

  // On-target = within ±10% of the calorie target.
  let onTarget = 0
  let adherenceSum = 0
  for (const r of logged) {
    const t = r.Targets.Calories
    if (t > 0) {
      const ratio = r.Consumed.Calories / t
      if (ratio >= 0.9 && ratio <= 1.1) onTarget++
      adherenceSum += Math.min(ratio, 1)
    }
  }
  const adherence = Math.round((adherenceSum / n) * 100)

  const dayLabel = (iso: string) =>
    new Date(iso).toLocaleDateString(undefined, { weekday: 'short', month: 'short', day: 'numeric' })

  const scored = logged
    .filter((r) => r.Targets.Calories > 0)
    .map((r) => ({ label: dayLabel(r.Date), kcal: r.Consumed.Calories, dist: Math.abs(r.Consumed.Calories - r.Targets.Calories) }))
    .sort((a, b) => a.dist - b.dist)

  return {
    avg,
    target,
    logged: n,
    onTarget,
    adherence,
    best: scored[0],
    worst: scored[scored.length - 1],
  }
}
