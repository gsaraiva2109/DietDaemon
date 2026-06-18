// Today — the hero screen. Ring-focused (chosen in the prototype pass) and
// enriched: greeting + date, hero calories ring with macro satellites, streak,
// 7-day calorie sparkline, energy-split donut, honest insights, inline quick
// log, and today's meal timeline.

import { useMemo } from 'react'
import { motion } from 'framer-motion'
import { useToday, useMeals, useRange } from '@/lib/queries'
import { MACRO_META, type Macros, type MacroKey } from '@/lib/types'
import { MacroRing } from '@/components/MacroRing'
import { MacroDonut } from '@/components/MacroDonut'
import { Sparkline } from '@/components/Sparkline'
import { MealCard } from '@/components/MealCard'
import { QuickLogCard } from '@/components/QuickLogCard'
import { Card, Eyebrow, EmptyState, Pill, Spinner } from '@/components/ui'
import { FlameIcon } from '@/components/icons'
import { cssVar } from '@/lib/format'
import { stagger, fadeUp } from '@/lib/motion'
import { greeting, insights, streak } from '@/lib/insights'

const ZERO: Macros = { Calories: 0, Protein: 0, Carbs: 0, Fat: 0, Fiber: 0 }
const SATELLITES: MacroKey[] = ['Protein', 'Carbs', 'Fat', 'Fiber']

function isoDaysAgo(n: number): string {
  const d = new Date()
  d.setDate(d.getDate() - n)
  return d.toISOString().slice(0, 10)
}

export function Dashboard() {
  const today = useToday()
  const meals = useMeals(6)
  const week = useRange(isoDaysAgo(6), isoDaysAgo(0))

  const consumed = today.data?.Consumed ?? ZERO
  const targets = today.data?.Targets ?? ZERO
  const tips = useMemo(() => insights(today.data ?? null), [today.data])
  const calorieSeries = useMemo(() => (week.data ?? []).map((d) => d.Consumed.Calories), [week.data])
  const dayStreak = useMemo(() => streak(week.data ?? []), [week.data])

  const todayLabel = new Date().toLocaleDateString(undefined, {
    weekday: 'long',
    month: 'long',
    day: 'numeric',
  })

  return (
    <div className="flex flex-col gap-6">
      {/* Greeting */}
      <header className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <Eyebrow>{todayLabel}</Eyebrow>
          <h1 className="mt-1 text-3xl font-bold tracking-tight text-ink">{greeting()}</h1>
        </div>
        {dayStreak > 0 && (
          <Pill tone="primary">
            <FlameIcon width={14} height={14} /> {dayStreak}-day streak
          </Pill>
        )}
      </header>

      {today.isLoading ? (
        <Spinner label="Loading today" />
      ) : (
        <>
          {/* Hero ring + side stats */}
          <div className="grid gap-5 lg:grid-cols-3">
            <Card className="flex flex-col items-center gap-7 p-7 lg:col-span-2">
              <MacroRing
                consumed={consumed.Calories}
                target={targets.Calories}
                label="Calories"
                unit="kcal"
                color={cssVar('--color-cal')}
                size={232}
                thickness={18}
              />
              <div className="grid w-full grid-cols-2 gap-5 sm:grid-cols-4">
                {SATELLITES.map((k) => (
                  <div key={k} className="flex flex-col items-center gap-2">
                    <MacroRing
                      consumed={consumed[k]}
                      target={targets[k]}
                      label={MACRO_META[k].label}
                      unit={MACRO_META[k].unit}
                      color={cssVar(MACRO_META[k].colorVar)}
                      size={96}
                      thickness={8}
                    />
                    <span className="text-sm font-medium text-muted">{MACRO_META[k].label}</span>
                  </div>
                ))}
              </div>
            </Card>

            <div className="flex flex-col gap-5">
              <Card className="p-5">
                <Eyebrow>Streak</Eyebrow>
                <div className="mt-2 flex items-center gap-2">
                  <span className="text-primary">
                    <FlameIcon width={28} height={28} />
                  </span>
                  <span className="text-4xl font-extrabold text-ink tnum">{dayStreak}</span>
                  <span className="mb-1 self-end text-sm text-muted">days logged</span>
                </div>
              </Card>
              <Card className="flex flex-1 flex-col p-5">
                <Eyebrow>Last 7 days · calories</Eyebrow>
                <div className="mt-auto pt-3">
                  {calorieSeries.length ? (
                    <Sparkline data={calorieSeries} color={cssVar('--color-cal')} />
                  ) : (
                    <p className="text-sm text-muted">No history yet.</p>
                  )}
                </div>
              </Card>
            </div>
          </div>

          {/* Energy split + insights */}
          <div className="grid gap-5 lg:grid-cols-3">
            <Card className="p-5">
              <Eyebrow>Energy split</Eyebrow>
              <div className="mt-4">
                <MacroDonut consumed={consumed} />
              </div>
            </Card>
            <Card className="p-5 lg:col-span-2">
              <Eyebrow>Insights</Eyebrow>
              <ul className="mt-3 flex flex-col gap-2.5">
                {tips.map((t, i) => (
                  <li key={i} className="flex items-start gap-2.5 text-sm">
                    <span
                      className={`mt-1.5 size-2 shrink-0 rounded-full ${
                        t.tone === 'good' ? 'bg-primary' : t.tone === 'warn' ? 'bg-accent' : 'bg-muted'
                      }`}
                    />
                    <span className="text-ink">{t.text}</span>
                  </li>
                ))}
              </ul>
            </Card>
          </div>

          <QuickLogCard />
        </>
      )}

      {/* Today's meals */}
      <section>
        <h2 className="mb-3 text-sm font-semibold uppercase tracking-[0.14em] text-muted">Today's meals</h2>
        {meals.isLoading ? (
          <Spinner />
        ) : !meals.data?.length ? (
          <EmptyState
            title="Nothing logged yet"
            hint="Use Quick log above, or send a meal through your chat bot. Try Demo mode to see it populated."
          />
        ) : (
          <motion.div variants={stagger} initial="hidden" animate="show" className="flex flex-col gap-2.5">
            {meals.data.map((m) => (
              <motion.div key={m.ID} variants={fadeUp}>
                <MealCard meal={m} linkTo={`/history/${m.ID}`} />
              </motion.div>
            ))}
          </motion.div>
        )}
      </section>
    </div>
  )
}
