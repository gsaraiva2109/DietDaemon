// /shared/:token, a public read-only view of another user's Today dashboard.
// No session, no app shell, anyone with the link can see it. If the primary
// rollup query fails (401/404 = revoked or never existed), the token is
// dead and nothing else would load either, so show one "invalid" state
// instead of five broken sections.

import { useParams } from 'react-router'
import { useTranslation } from 'react-i18next'
import { useSharedDashboard } from '@/lib/queries'
import { MACRO_META, type DailyRollup, type MacroKey } from '@/lib/types'
import { MacroRing } from '@/components/MacroRing'
import { MealCard } from '@/components/MealCard'
import { Card, Eyebrow, EmptyState, Pill, Spinner } from '@/components/ui'
import { LeafIcon, FlameIcon, BodyIcon } from '@/components/icons'
import { cssVar, formatNumber, round } from '@/lib/format'

const SATELLITES: MacroKey[] = ['Protein', 'Carbs', 'Fat', 'Fiber']

type SharedQueries = ReturnType<typeof useSharedDashboard>

// Recent-meals has its own loading/error/empty/list ladder, independent of
// the top-level rollup state. Kept as if/else (not a nested ternary) and out
// of DashboardContent so it's easy to read in isolation.
function RecentMeals({ meals }: Readonly<Pick<SharedQueries, 'meals'>>) {
  const { t } = useTranslation()

  if (meals.isLoading) return <Spinner />
  if (meals.isError) return <p className="text-sm text-muted">{t('sharedDashboard.sectionError')}</p>
  if (!meals.data?.length) return <p className="text-sm text-muted">{t('sharedDashboard.noMeals')}</p>
  return (
    <div className="flex flex-col gap-2.5">
      {meals.data.map((m) => (
        <MealCard key={m.ID} meal={m} />
      ))}
    </div>
  )
}

// The macro rings + budget/weight cards + recent meals, shown once the rollup
// has loaded successfully. `today` is guaranteed non-null by the caller.
function DashboardContent({
  today,
  meals,
  budget,
  bodySummary,
}: Readonly<Pick<SharedQueries, 'meals' | 'budget' | 'bodySummary'> & { today: DailyRollup }>) {
  const { t } = useTranslation()

  return (
    <>
      <Card className="flex flex-col items-center gap-7 p-7">
        <MacroRing
          consumed={today.Consumed.Calories}
          target={today.Targets.Calories}
          label={t('dashboard.calories')}
          unit="kcal"
          color={cssVar('--color-cal')}
          size={200}
          thickness={16}
        />
        <div className="grid w-full grid-cols-2 gap-5 sm:grid-cols-4">
          {SATELLITES.map((k) => (
            <div key={k} className="flex flex-col items-center gap-2">
              <MacroRing
                consumed={today.Consumed[k] ?? 0}
                target={today.Targets[k] ?? 0}
                label={t(`common.macro.${k}`)}
                unit={MACRO_META[k].unit}
                color={cssVar(MACRO_META[k].colorVar)}
                size={88}
                thickness={8}
              />
              <span className="text-sm font-medium text-muted">{t(`common.macro.${k}`)}</span>
            </div>
          ))}
        </div>
      </Card>

      <div className="grid gap-5 sm:grid-cols-2">
        {budget.data && (
          <Card className="p-5">
            <Eyebrow>{t('dashboard.weeklyBudget')}</Eyebrow>
            <div className="mt-2 flex items-baseline justify-between">
              <span className="text-sm text-muted">{t('dashboard.calories')}</span>
              <span className="text-sm font-semibold text-ink tnum">
                {formatNumber(round(budget.data.calories.effective, 0))}
              </span>
            </div>
            <div className="mt-1 flex items-baseline justify-between">
              <span className="text-sm text-muted">{t('dashboard.protein')}</span>
              <span className="text-sm font-semibold text-ink tnum">
                {formatNumber(round(budget.data.protein.effective, 0))}g
              </span>
            </div>
          </Card>
        )}
        {bodySummary.data && bodySummary.data.current_weight_kg > 0 && (
          <Card className="p-5">
            <div className="flex items-center justify-between">
              <Eyebrow>{t('dashboard.weight')}</Eyebrow>
              <span className="text-muted"><BodyIcon width={16} height={16} /></span>
            </div>
            <div className="mt-2 flex items-baseline gap-2">
              <span className="text-2xl font-extrabold text-ink tnum">
                {formatNumber(round(bodySummary.data.current_weight_kg, 1))}
              </span>
              <span className="text-sm text-muted">kg</span>
            </div>
          </Card>
        )}
        {(budget.isError || bodySummary.isError) && (
          <p className="text-sm text-muted sm:col-span-2">{t('sharedDashboard.sectionError')}</p>
        )}
      </div>

      <section>
        <h2 className="mb-3 text-sm font-semibold uppercase tracking-[0.14em] text-muted">
          {t('sharedDashboard.recentMeals')}
        </h2>
        <RecentMeals meals={meals} />
      </section>
    </>
  )
}

// Top-level state: loading, invalid link (revoked/unknown token), or loaded
// content. If/else instead of a nested ternary, and lives outside
// SharedDashboard so its branches don't add to that function's complexity.
function SharedDashboardBody({
  today,
  meals,
  budget,
  bodySummary,
}: Readonly<Pick<SharedQueries, 'today' | 'meals' | 'budget' | 'bodySummary'>>) {
  const { t } = useTranslation()

  if (today.isLoading) return <Spinner label={t('sharedDashboard.loading')} />
  if (today.isError || !today.data) {
    return <EmptyState title={t('sharedDashboard.invalidTitle')} hint={t('sharedDashboard.invalidHint')} />
  }
  return <DashboardContent today={today.data} meals={meals} budget={budget} bodySummary={bodySummary} />
}

export function SharedDashboard() {
  const { t } = useTranslation()
  const { token } = useParams<{ token: string }>()
  const { today, meals, budget, bodySummary, streak } = useSharedDashboard(token ?? '')

  return (
    <div className="min-h-[100dvh] px-6 py-10">
      <div className="mx-auto flex w-full max-w-2xl flex-col gap-6">
        <header className="flex items-center gap-3">
          <span className="grid size-11 place-items-center rounded-2xl bg-primary-soft text-primary">
            <LeafIcon width={22} height={22} />
          </span>
          <div>
            <Eyebrow>DietDaemon</Eyebrow>
            <h1 className="text-xl font-bold tracking-tight text-ink">{t('dashboard.today')}</h1>
          </div>
          {streak.data && streak.data.current_days > 0 && (
            <div className="ml-auto">
              <Pill tone="primary">
                <FlameIcon width={14} height={14} /> {t('dashboard.streakDays', { count: streak.data.current_days })}
              </Pill>
            </div>
          )}
        </header>

        <SharedDashboardBody today={today} meals={meals} budget={budget} bodySummary={bodySummary} />

        <p className="text-center text-xs text-muted">{t('sharedDashboard.footer')}</p>
      </div>
    </div>
  )
}
