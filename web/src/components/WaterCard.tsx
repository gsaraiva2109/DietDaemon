// WaterCard, today's hydration against the daily goal. Blue accent paired with
// the "Water" label and ml numbers (colour never the sole signal). Backend is
// Phase 4: a 404 collapses to the empty state, and quick-adds light it up once
// the endpoint ships.

import { useTranslation } from 'react-i18next'
import { useWaterToday, useLogWater } from '@/lib/queries'
import { Card, Eyebrow, Pill, Spinner } from '@/components/ui'
import { AnimatedNumber } from '@/components/AnimatedNumber'
import { DropletIcon } from '@/components/icons'

const QUICK_ADDS = [200, 500, 1000] // ml
const BLUE = 'var(--color-protein)'

export function WaterCard() {
  const { t, i18n } = useTranslation()
  const water = useWaterToday()
  const logWater = useLogWater()

  const data = water.data
  const todayMl = data?.today_ml ?? 0
  const goalMl = data?.goal_ml ?? 0
  const pct = goalMl > 0 ? Math.min(100, Math.round((todayMl / goalMl) * 100)) : 0
  const hit = goalMl > 0 && todayMl >= goalMl

  return (
    <Card className="flex h-full flex-col gap-4 p-5">
      <header className="flex items-center justify-between">
        <div className="flex items-center gap-2" style={{ color: BLUE }}>
          <DropletIcon width={18} height={18} />
          <Eyebrow>{t('waterCard.title')}</Eyebrow>
        </div>
        {hit && <Pill tone="primary">{t('waterCard.goalMet')}</Pill>}
      </header>

      {water.isLoading ? (
        <Spinner />
      ) : water.isError ? (
        <button
          onClick={() => water.refetch()}
          className="self-start text-sm font-medium text-accent hover:underline"
        >
          {t('waterCard.retry')}
        </button>
      ) : (
        <>
          <div className="flex items-baseline gap-1.5">
            <span className="text-3xl font-bold text-ink tnum">
              <AnimatedNumber value={todayMl} />
            </span>
            <span className="text-sm text-muted">
              {goalMl > 0 ? `/ ${goalMl.toLocaleString(i18n.language)} ml` : t('waterCard.mlToday')}
            </span>
          </div>

          {goalMl > 0 ? (
            <div
              className="h-1.5 w-full rounded-full bg-surface-2"
              role="progressbar"
              aria-valuenow={pct}
              aria-valuemin={0}
              aria-valuemax={100}
              aria-label={t('waterCard.ariaGoal')}
            >
              <div
                className="h-full rounded-full transition-[width] duration-500"
                style={{ width: `${pct}%`, background: BLUE }}
              />
            </div>
          ) : (
            <p className="text-sm text-muted">{t('waterCard.empty')}</p>
          )}

          <div className="mt-auto flex flex-wrap gap-2 pt-1">
            {QUICK_ADDS.map((ml) => (
              <button
                key={ml}
                onClick={() => logWater.mutate({ amountMl: ml })}
                disabled={logWater.isPending}
                className="rounded-full border border-line bg-surface px-3 py-1.5 text-sm font-medium text-ink transition hover:bg-surface-2 disabled:opacity-50"
              >
                +{ml} ml
              </button>
            ))}
          </div>
          {logWater.isError && (
            <p className="text-xs font-medium text-accent" role="alert">
              {t('waterCard.logError')}
            </p>
          )}
        </>
      )}
    </Card>
  )
}
