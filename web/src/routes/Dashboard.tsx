// Today, the hero screen. Ring-focused (chosen in the prototype pass) and
// enriched: greeting + date, hero calories ring with macro satellites, streak,
// 7-day calorie sparkline, energy-split donut, honest insights, inline quick
// log, and today's meal timeline.

import { lazy, Suspense, useMemo, useState } from 'react'
import { motion } from 'framer-motion'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useToday, useMeals, useRange, useBodySummary, useStreak, useWeeklyBudget } from '@/lib/queries'
import { MACRO_META, type Macros, type MacroKey } from '@/lib/types'
import { MacroRing } from '@/components/MacroRing'
import { Sparkline } from '@/components/Sparkline'
import { MealCard } from '@/components/MealCard'
import { QuickLogCard } from '@/components/QuickLogCard'
import { WaterCard } from '@/components/WaterCard'
import { WorkoutCard } from '@/components/WorkoutCard'
import { FastingCard } from '@/components/FastingCard'
import { FrequentFoods } from '@/components/FrequentFoods'
import { ShareCard } from '@/components/ShareCard'
import { Card, Eyebrow, EmptyState, Pill, Spinner, Button } from '@/components/ui'
import { FlameIcon, BodyIcon, ShareIcon } from '@/components/icons'
import { cssVar, formatNumber, round } from '@/lib/format'
import { stagger, fadeUp } from '@/lib/motion'
import { greeting, insights } from '@/lib/insights'

const ZERO: Macros = { Calories: 0, Protein: 0, Carbs: 0, Fat: 0, Fiber: 0 }
const SATELLITES: MacroKey[] = ['Protein', 'Carbs', 'Fat', 'Fiber']
const MacroDonut = lazy(() => import('@/components/MacroDonut').then(m => ({ default: m.MacroDonut })))
const SleepCard = lazy(() => import('@/components/SleepCard').then(m => ({ default: m.SleepCard })))
const WeeklyDashboard = lazy(() =>
  import('@/components/WeeklyDashboard').then(m => ({ default: m.WeeklyDashboard })),
)

function isoDaysAgo(n: number): string {
  const d = new Date()
  d.setDate(d.getDate() - n)
  return d.toISOString().slice(0, 10)
}

export function Dashboard() {
  const { t, i18n } = useTranslation()
  const today = useToday()
  const meals = useMeals(6)
  const week = useRange(isoDaysAgo(6), isoDaysAgo(0))
  const body = useBodySummary()
  const streakQuery = useStreak()
  const budget = useWeeklyBudget()
  const [view, setView] = useState<'day' | 'week'>('day')
  const [sharing, setSharing] = useState(false)

  const consumed = today.data?.Consumed ?? ZERO
  const targets = today.data?.Targets ?? ZERO
  const tips = useMemo(() => insights(today.data ?? null, t), [today.data, t])
  const calorieSeries = useMemo(() => (week.data ?? []).map((d) => d.Consumed.Calories), [week.data])
  const dayStreak = streakQuery.data?.current_days ?? 0

  // Weekly budget: show effective target when it differs from plain target.
  const budgetDelta = budget.data
    ? budget.data.calories.effective - budget.data.calories.plain
    : 0
  const budgetActive = Math.abs(budgetDelta) >= 1

  const todayLabel = new Date().toLocaleDateString(i18n.language, {
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
          <h1 className="mt-1 text-3xl font-bold tracking-tight text-ink">{greeting(t)}</h1>
        </div>
        <div className="flex items-center gap-2">
          {dayStreak > 0 && (
            <Pill tone="primary">
              <FlameIcon width={14} height={14} /> {t('dashboard.streakDays', { count: dayStreak })}
            </Pill>
          )}
          <div className="flex gap-1 rounded-full border border-line bg-surface p-1">
            {(['day', 'week'] as const).map((v) => (
              <button
                key={v}
                onClick={() => setView(v)}
                className={`rounded-full px-3 py-1 text-sm font-medium capitalize transition ${
                  view === v ? 'bg-primary-soft text-primary' : 'text-muted hover:text-ink'
                }`}
              >
                {t(`dashboard.view.${v}`)}
              </button>
            ))}
          </div>
          <Button
            variant="ghost"
            onClick={() => setSharing(true)}
            aria-label={t('dashboard.shareAria')}
            className="px-3 py-1.5 text-xs"
          >
            <ShareIcon width={15} height={15} /> {t('dashboard.share')}
          </Button>
        </div>
      </header>

      {today.isLoading ? (
        <Spinner label={t('dashboard.loadingToday')} />
      ) : (
        <>
          {/* Hero ring + side stats */}
          <div className="grid gap-5 lg:grid-cols-3">
            <Card className="flex flex-col items-center gap-7 p-7 lg:col-span-2">
              <MacroRing
                consumed={consumed.Calories}
                target={targets.Calories}
                label={t('dashboard.calories')}
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
                      label={t(`common.macro.${k}`)}
                      unit={MACRO_META[k].unit}
                      color={cssVar(MACRO_META[k].colorVar)}
                      size={96}
                      thickness={8}
                    />
                    <span className="text-sm font-medium text-muted">{t(`common.macro.${k}`)}</span>
                  </div>
                ))}
              </div>
            </Card>

            <div className="flex flex-col gap-5">
              <Card className="p-5">
                <Eyebrow>{t('dashboard.streak')}</Eyebrow>
                <div className="mt-2 flex items-center gap-2">
                  <span className="text-primary">
                    <FlameIcon width={28} height={28} />
                  </span>
                  <span className="text-4xl font-extrabold text-ink tnum">{dayStreak}</span>
                  <span className="mb-1 self-end text-sm text-muted">{t('dashboard.daysOnTarget')}</span>
                </div>
              </Card>
              {budgetActive && budget.data && (
                <Card className="p-5">
                  <Eyebrow>{t('dashboard.weeklyBudget')}</Eyebrow>
                  <div className="mt-2 space-y-2">
                    <div className="flex items-baseline justify-between">
                      <span className="text-sm text-muted">{t('dashboard.calories')}</span>
                      <span className="text-sm font-semibold text-ink tnum">
                        {formatNumber(round(budget.data.calories.effective, 0))}
                      </span>
                    </div>
                    <div className="flex items-baseline justify-between">
                      <span className="text-xs text-muted">{t('dashboard.vsPlain')}</span>
                      <span className={`text-xs font-medium tnum ${budgetDelta > 0 ? 'text-accent' : 'text-primary'}`}>
                        {budgetDelta > 0 ? '+' : ''}{formatNumber(round(budgetDelta, 0))} kcal
                      </span>
                    </div>
                    <div className="flex items-baseline justify-between">
                      <span className="text-sm text-muted">{t('dashboard.protein')}</span>
                      <span className="text-sm font-semibold text-ink tnum">
                        {formatNumber(round(budget.data.protein.effective, 0))}g
                      </span>
                    </div>
                  </div>
                </Card>
              )}
              <WeightMiniCard body={body.data} />
              <Card className="flex flex-1 flex-col p-5">
                <Eyebrow>{t('dashboard.last7DaysCalories')}</Eyebrow>
                <div className="mt-auto pt-3">
                  {calorieSeries.length ? (
                    <Sparkline data={calorieSeries} color={cssVar('--color-cal')} />
                  ) : (
                    <p className="text-sm text-muted">{t('dashboard.noHistoryYet')}</p>
                  )}
                </div>
              </Card>
            </div>
          </div>

          {view === 'week' ? (
            <Suspense fallback={null}>
              <WeeklyDashboard />
            </Suspense>
          ) : (
            <>
              {/* Energy split + insights */}
              <div className="grid gap-5 lg:grid-cols-3">
                <Card className="p-5">
                  <Eyebrow>{t('dashboard.energySplit')}</Eyebrow>
                  <div className="mt-4">
                    <Suspense fallback={null}>
                      <MacroDonut consumed={consumed} />
                    </Suspense>
                  </div>
                </Card>
                <Card className="p-5 lg:col-span-2">
                  <Eyebrow>{t('dashboard.insights')}</Eyebrow>
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
        </>
      )}

      {view === 'day' && (
        <>
          {/* Health, quiet secondary section, subordinate to the macro hero. */}
          <section>
            <Eyebrow>{t('dashboard.health')}</Eyebrow>
            <motion.div
              variants={stagger}
              initial="hidden"
              animate="show"
              className="mt-3 grid gap-5 md:grid-cols-2"
            >
              <motion.div variants={fadeUp}><WaterCard /></motion.div>
              <motion.div variants={fadeUp}><FastingCard /></motion.div>
              <motion.div variants={fadeUp}><WorkoutCard /></motion.div>
              <motion.div variants={fadeUp}>
                <Suspense fallback={null}>
                  <SleepCard />
                </Suspense>
              </motion.div>
            </motion.div>
          </section>

          {/* Frequent foods */}
          <FrequentFoods />

          {/* Today's meals */}
          <section>
            <h2 className="mb-3 text-sm font-semibold uppercase tracking-[0.14em] text-muted">{t('dashboard.todaysMeals')}</h2>
            {meals.isLoading ? (
              <Spinner />
            ) : !meals.data?.length ? (
              <EmptyState
                title={t('dashboard.emptyTitle')}
                hint={t('dashboard.emptyHint')}
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
        </>
      )}

      {sharing && (
        <ShareCard
          heading={t('dashboard.today')}
          subtitle={todayLabel}
          consumed={consumed}
          onClose={() => setSharing(false)}
        />
      )}
    </div>
  )
}

// WeightMiniCard shows the latest weight + recent change, linking to /body.
function WeightMiniCard({ body }: { body?: import('@/lib/types').BodyCompositionSummary }) {
  const { t } = useTranslation()
  if (!body || body.current_weight_kg <= 0) return null
  const arrow = body.trend_direction === 'up' ? '↑' : body.trend_direction === 'down' ? '↓' : '→'
  const tone = body.trend_direction === 'down' ? 'text-primary' : body.trend_direction === 'up' ? 'text-accent' : 'text-muted'
  return (
    <Link to="/body" className="block">
      <Card className="p-5 transition hover:shadow-lift">
        <div className="flex items-center justify-between">
          <Eyebrow>{t('dashboard.weight')}</Eyebrow>
          <span className="text-muted"><BodyIcon width={18} height={18} /></span>
        </div>
        <div className="mt-2 flex items-baseline gap-2">
          <span className="text-3xl font-extrabold text-ink tnum">{formatNumber(round(body.current_weight_kg, 1))}</span>
          <span className="text-sm text-muted">kg</span>
          {body.change_kg !== 0 && (
            <span className={`ml-auto text-sm font-semibold ${tone}`}>
              {arrow} {Math.abs(round(body.change_kg, 1))}kg
            </span>
          )}
        </div>
      </Card>
    </Link>
  )
}
