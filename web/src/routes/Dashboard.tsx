// Today, the hero screen. Ring-focused (chosen in the prototype pass) and
// enriched: greeting + date, hero calories ring with macro satellites, streak,
// 7-day calorie sparkline, energy-split donut, honest insights, inline quick
// log, and today's meal timeline.

import { lazy, Suspense, useMemo, useState } from 'react'
import { motion } from 'framer-motion'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  useToday,
  useMeals,
  useRange,
  useBodySummary,
  useStreak,
  useWeeklyBudget,
  useActivePlan,
  usePlanDay,
  usePlanBundle,
  useSetDayOverride,
  useLogTemplate,
} from '@/lib/queries'
import {
  MACRO_META,
  type Macros,
  type MacroKey,
  type DietPlanSlotBundle,
  type DietPlanSlotOption,
  type DietPlanDayTypeBundle,
  type Meal,
} from '@/lib/types'
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
import { FlameIcon, BodyIcon, ShareIcon, CheckIcon } from '@/components/icons'
import { cssVar, formatNumber, round, sumMacros } from '@/lib/format'
import { stagger, fadeUp } from '@/lib/motion'
import { greeting, insights } from '@/lib/insights'

const ZERO: Macros = { Calories: 0, Protein: 0, Carbs: 0, Fat: 0, Fiber: 0 }
const SATELLITES: MacroKey[] = ['Protein', 'Carbs', 'Fat', 'Fiber']
const INSIGHT_TONE_CLASS = { good: 'bg-primary', warn: 'bg-accent', info: 'bg-muted' } as const
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

// ---------------------------------------------------------------------------
// Diet plan surfaces (issue #191): day-type badge/switcher, today's slot
// checklist, and a 7-day week strip. All four gate on activePlan/planDay
// existing so a user with no plan sees byte-identical output to before.
// ---------------------------------------------------------------------------

function byPosition<T extends { position: number }>(list: T[]): T[] {
  return [...list].sort((a, b) => a.position - b.position)
}

// The 7 calendar dates (Mon..Sun) of the week containing `todayISO`, the
// window the mockup's "Seg Ter Qua..." week strip shows. TargetsFor resolves
// each date independently server-side, so this is just which dates to ask
// GET /plans/day/{date} about -- no cycle-length math needed here.
function weekDatesFor(todayISO: string): string[] {
  const today = new Date(`${todayISO}T00:00:00`)
  const mondayOffset = (today.getDay() + 6) % 7 // days since Monday (Sun=0 -> 6)
  const monday = new Date(today)
  monday.setDate(today.getDate() - mondayOffset)
  return Array.from({ length: 7 }, (_, i) => {
    const d = new Date(monday)
    d.setDate(monday.getDate() + i)
    return d.toISOString().slice(0, 10)
  })
}

function timeToMinutes(hhmm: string): number {
  const [h, m] = hhmm.split(':')
  return (Number(h) || 0) * 60 + (Number(m) || 0)
}

// Bot-logged meals arrive with no slot/option link (trap #12: a wrong guess
// must never persist), so which slot's checkmark lights up is inferred here,
// at render time only, from the meal's clock time. "Nearest slot whose
// window (the midpoint between adjacent slot times) contains the meal" is
// exactly nearest-neighbour-by-absolute-time-distance on a 1-D line, cheaper
// to compute directly than building the midpoint windows first.
function nearestSlotID(slots: DietPlanSlotBundle[], mealAtISO: string): string | null {
  if (!slots.length) return null
  const meal = new Date(mealAtISO)
  const mealMinutes = meal.getHours() * 60 + meal.getMinutes()
  let best = slots[0]
  let bestDiff = Infinity
  for (const s of slots) {
    const diff = Math.abs(timeToMinutes(s.time_of_day) - mealMinutes)
    if (diff < bestDiff) {
      bestDiff = diff
      best = s
    }
  }
  return best.id
}

// computeSlotKcal sums each of today's logged meals' kcal against its
// nearest-by-time slot (see nearestSlotID above), for the slot-completion
// checklist. Pulled out of Dashboard's useMemo so its loop/branches count
// against this function's own complexity budget, not the component's.
function computeSlotKcal(slots: DietPlanSlotBundle[], meals: Meal[] | undefined): Map<string, number> {
  const map = new Map<string, number>()
  if (!slots.length || !meals?.length) return map
  const todayKey = new Date().toDateString()
  for (const meal of meals) {
    if (new Date(meal.At).toDateString() !== todayKey) continue
    // An explicit "registrar opção" log carries the real slot id; only fall
    // back to time-based inference for bot-logged meals (trap #12).
    const slotID = meal.PlanSlotID || nearestSlotID(slots, meal.At)
    if (!slotID) continue
    const kcal = sumMacros(meal.Items.map((it) => it.Macros)).Calories
    map.set(slotID, (map.get(slotID) ?? 0) + kcal)
  }
  return map
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

  // Diet plan surfaces. activePlan is a cheap 404-tolerant check; the rest
  // (planDay, planBundle) only fire once it resolves a real plan, via the
  // `enabled: Boolean(...)` gating already built into these hooks -- so a
  // user with no plan causes no extra plan-related requests either.
  const activePlan = useActivePlan()
  const planActive = Boolean(activePlan.data)
  const todayISO = isoDaysAgo(0)
  const planDay = usePlanDay(planActive ? todayISO : '')
  const planBundle = usePlanBundle(activePlan.data?.id)
  const setOverride = useSetDayOverride()
  const dayType = planDay.data?.day_type ?? null
  const weekDates = useMemo(() => weekDatesFor(todayISO), [todayISO])

  const consumed = today.data?.Consumed ?? ZERO
  const targets = today.data?.Targets ?? ZERO
  const tips = useMemo(() => insights(today.data ?? null, t), [t, today.data])
  const calorieSeries = useMemo(() => (week.data ?? []).map((d) => d.Consumed.Calories), [week.data])
  const dayStreak = streakQuery.data?.current_days ?? 0

  // Slot completion for today's checklist. Reuses the same 6-most-recent
  // `meals` query the "today's meals" list already fetches -- plenty for a
  // handful of plan slots per day; bump the limit if a day-type ever needs
  // more.
  const slotKcal = useMemo(
    () => computeSlotKcal(planDay.data?.slots ?? [], meals.data),
    [planDay.data?.slots, meals.data],
  )

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
              {dayType && (
                <div className="flex w-full flex-wrap items-center justify-center gap-2 self-start">
                  <Pill tone="primary">
                    {t('dashboard.today')} · {dayType.name}
                  </Pill>
                  {planBundle.data && planBundle.data.day_types.length > 1 && (
                    <select
                      aria-label={t('plan.switchDayTypeAria')}
                      value={dayType.id}
                      disabled={setOverride.isPending}
                      onChange={(e) => setOverride.mutate({ date: todayISO, dayTypeID: e.target.value })}
                      className="rounded-full border border-line bg-surface px-2.5 py-1 text-xs font-medium text-ink outline-none focus:border-primary"
                    >
                      {byPosition(planBundle.data.day_types).map((dt) => (
                        <option key={dt.id} value={dt.id}>
                          {dt.name}
                        </option>
                      ))}
                    </select>
                  )}
                </div>
              )}
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

          {dayType && (
            <Card className="p-5">
              <Eyebrow>
                {t('dashboard.today')} · {dayType.name}
              </Eyebrow>
              {planDay.data?.slots.length ? (
                <ul className="mt-3 flex flex-col gap-2">
                  {byPosition(planDay.data.slots).map((slot) => (
                    <SlotRow key={slot.id} slot={slot} kcal={slotKcal.get(slot.id)} />
                  ))}
                </ul>
              ) : (
                <p className="mt-3 text-sm text-muted">{t('plan.noSlotsToday')}</p>
              )}
            </Card>
          )}

          {planActive && (
            <Card className="p-5">
              <Eyebrow>{t('plan.weekStripTitle')}</Eyebrow>
              <div className="mt-3 grid grid-cols-7 gap-1.5">
                {weekDates.map((date) => (
                  <WeekStripDay
                    key={date}
                    date={date}
                    isToday={date === todayISO}
                    dayTypes={planBundle.data?.day_types ?? []}
                    onPick={(dayTypeID) => setOverride.mutate({ date, dayTypeID })}
                  />
                ))}
              </div>
            </Card>
          )}

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
                    {tips.map((tip) => (
                      <li key={tip.text} className="flex items-start gap-2.5 text-sm">
                        <span
                          className={`mt-1.5 size-2 shrink-0 rounded-full ${INSIGHT_TONE_CLASS[tip.tone]}`}
                        />
                        <span className="text-ink">{tip.text}</span>
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

          <TodayMeals meals={meals} />
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

function TodayMeals({ meals }: Readonly<{ meals: ReturnType<typeof useMeals> }>) {
  const { t } = useTranslation()
  let content = <Spinner />
  if (!meals.isLoading) {
    if (!meals.data?.length) {
      content = <EmptyState title={t('dashboard.emptyTitle')} hint={t('dashboard.emptyHint')} />
    } else {
      content = (
        <motion.div variants={stagger} initial="hidden" animate="show" className="flex flex-col gap-2.5">
          {meals.data.map((meal) => (
            <motion.div key={meal.ID} variants={fadeUp}>
              <MealCard meal={meal} linkTo={`/history/${meal.ID}`} />
            </motion.div>
          ))}
        </motion.div>
      )
    }
  }
  return (
    <section>
      <h2 className="mb-3 text-sm font-semibold uppercase tracking-[0.14em] text-muted">{t('dashboard.todaysMeals')}</h2>
      {content}
    </section>
  )
}

// WeightMiniCard shows the latest weight + recent change, linking to /body.
function WeightMiniCard({ body }: Readonly<{ body?: import('@/lib/types').BodyCompositionSummary }>) {
  const { t } = useTranslation()
  if (!body || body.current_weight_kg <= 0) return null
  let arrow = '→'
  let tone = 'text-muted'
  switch (body.trend_direction) {
    case 'up':
      arrow = '↑'
      tone = 'text-accent'
      break
    case 'down':
      arrow = '↓'
      tone = 'text-primary'
      break
  }
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

// One row of the today's-slot checklist: a checkmark + logged kcal once a
// meal has been matched to this slot (explicit "registrar" tap, or a
// bot-logged meal inferred by time -- see nearestSlotID), otherwise a
// "registrar {option}" button per prescribed option.
function SlotRow({ slot, kcal }: Readonly<{ slot: DietPlanSlotBundle; kcal: number | undefined }>) {
  const done = kcal !== undefined
  return (
    <li className="flex flex-wrap items-center gap-3 rounded-lg border border-line px-3 py-2">
      <span className={done ? 'text-primary' : 'text-muted'}>
        {done ? <CheckIcon width={16} height={16} /> : <span className="block size-4 rounded-full border border-line" />}
      </span>
      <span className="w-12 shrink-0 text-xs text-muted tnum">{slot.time_of_day}</span>
      <span className="flex-1 text-sm font-medium text-ink">{slot.label}</span>
      {kcal !== undefined ? (
        <span className="text-xs text-muted tnum">{formatNumber(round(kcal, 0))} kcal</span>
      ) : (
        <div className="flex flex-wrap items-center gap-1.5">
          {byPosition(slot.options).map((opt) => (
            <RegisterOptionButton key={opt.id} slotID={slot.id} option={opt} />
          ))}
        </div>
      )}
    </li>
  )
}

// "registrar opção N": reuses the existing log-template path (a slot option
// is backed by a meal_templates row), which is the one path allowed to
// explicitly attribute a logged meal to this slot/option (trap #12 --
// bot-logged meals never get that write, only ever an inferred display).
function RegisterOptionButton({ slotID, option }: Readonly<{ slotID: string; option: DietPlanSlotOption }>) {
  const { t } = useTranslation()
  const log = useLogTemplate()
  const [logged, setLogged] = useState(false)

  function doLog() {
    log.mutate(
      { id: option.template_id, planSlotID: slotID, planOptionID: option.id },
      {
        onSuccess: () => {
          setLogged(true)
          window.setTimeout(() => setLogged(false), 2200)
        },
      },
    )
  }

  if (logged) {
    return (
      <Pill tone="primary">
        <CheckIcon width={12} height={12} /> {t('plan.optionLogged')}
      </Pill>
    )
  }
  return (
    <Button variant="ghost" onClick={doLog} disabled={log.isPending} className="px-2.5 py-1 text-xs">
      {log.isPending ? t('plan.registering') : t('plan.registerOption', { label: option.label })}
    </Button>
  )
}

// One column of the 7-day week strip. Tapping a different day-type writes an
// override for that specific date via onPick -- works for future dates with
// no extra plumbing since TargetsFor/plan-day resolution is read-time only.
function WeekStripDay({
  date,
  isToday,
  dayTypes,
  onPick,
}: Readonly<{
  date: string
  isToday: boolean
  dayTypes: DietPlanDayTypeBundle[]
  onPick: (dayTypeID: string) => void
}>) {
  const { t, i18n } = useTranslation()
  const day = usePlanDay(date)
  const weekdayLabel = new Date(`${date}T00:00:00`).toLocaleDateString(i18n.language, { weekday: 'short' })
  const dayTypeID = day.data?.day_type?.id ?? ''

  return (
    <div className={`flex flex-col items-center gap-1 rounded-lg px-1 py-2 ${isToday ? 'bg-primary-soft' : ''}`}>
      <span className="text-[10px] font-semibold uppercase tracking-wide text-muted">{weekdayLabel}</span>
      <select
        aria-label={t('plan.weekDayAria', { date })}
        value={dayTypeID}
        disabled={!dayTypes.length}
        onChange={(e) => onPick(e.target.value)}
        className="w-full rounded-md border border-line bg-surface px-1 py-1 text-center text-[10px] font-medium text-ink outline-none focus:border-primary"
      >
        {!dayTypeID && <option value="">–</option>}
        {byPosition(dayTypes).map((dt) => (
          <option key={dt.id} value={dt.id}>
            {dt.name}
          </option>
        ))}
      </select>
    </div>
  )
}
