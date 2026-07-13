// Self-contained weekly overview: pulls its own last-7-days range, reduces it
// to WeeklyStats, and renders stat tiles, a daily-calories bar chart vs the
// target, and best/worst day mini cards. Single-level cards throughout
// (DESIGN.md: never nest a Card in a Card).

import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Bar,
  BarChart,
  CartesianGrid,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { useRange } from '@/lib/queries'
import { weeklyStats } from '@/lib/insights'
import type { DailyRollup, TrendDirection } from '@/lib/types'
import { cssVar } from '@/lib/format'
import { Card, Eyebrow, EmptyState, Spinner } from './ui'
import { AnimatedNumber } from './AnimatedNumber'

function isoDaysAgo(n: number): string {
  const d = new Date()
  d.setDate(d.getDate() - n)
  return d.toISOString().slice(0, 10)
}

function niceDate(iso: string, locale: string): string {
  // iso is YYYY-MM-DD; parse as local date to avoid TZ drift.
  const [y, m, d] = iso.split('-').map(Number)
  return new Date(y, m - 1, d).toLocaleDateString(locale, {
    weekday: 'short',
    month: 'short',
    day: 'numeric',
  })
}

// up arrow = accent (intake rising), down arrow = primary (good on a cut),
// flat = muted. Glyphs kept text-only and tasteful.
function TrendArrow({ dir }: { dir: TrendDirection }) {
  const { t } = useTranslation()
  const map: Record<TrendDirection, { glyph: string; cls: string; labelKey: string }> = {
    up: { glyph: '▲', cls: 'text-accent', labelKey: 'weeklyDashboard.trend.up' },
    down: { glyph: '▼', cls: 'text-primary', labelKey: 'weeklyDashboard.trend.down' },
    flat: { glyph: '→', cls: 'text-muted', labelKey: 'weeklyDashboard.trend.flat' },
  }
  const m = map[dir]
  const label = t(m.labelKey)
  return (
    <span className={`text-sm ${m.cls}`} aria-label={label} title={label}>
      {m.glyph}
    </span>
  )
}

function StatTile({
  label,
  value,
  unit,
  trend,
}: {
  label: string
  value: number
  unit?: string
  trend?: TrendDirection
}) {
  return (
    <Card className="p-4">
      <div className="flex items-center justify-between gap-2">
        <Eyebrow>{label}</Eyebrow>
        {trend && <TrendArrow dir={trend} />}
      </div>
      <div className="mt-2 flex items-baseline gap-1">
        <span className="text-2xl font-extrabold text-ink">
          <AnimatedNumber value={value} />
        </span>
        {unit && <span className="text-sm font-medium text-muted">{unit}</span>}
      </div>
    </Card>
  )
}

function DayCard({ title, day, locale }: { title: string; day: DailyRollup; locale: string }) {
  const consumed = Math.round(day.Consumed.Calories)
  const target = Math.round(day.Targets.Calories)
  return (
    <Card className="p-4">
      <Eyebrow>{title}</Eyebrow>
      <p className="mt-1.5 text-sm font-semibold text-ink">{niceDate(day.Date, locale)}</p>
      <p className="mt-1 text-sm text-muted tnum">
        {consumed.toLocaleString(locale)} kcal
        {target > 0 && <span className="text-muted"> / {target.toLocaleString(locale)} target</span>}
      </p>
    </Card>
  )
}

export function WeeklyDashboard() {
  const { t, i18n } = useTranslation()
  const range = useRange(isoDaysAgo(6), isoDaysAgo(0))
  const days = useMemo(() => range.data ?? [], [range.data])
  const stats = useMemo(() => weeklyStats(days), [days])

  const chartData = useMemo(
    () =>
      days.map((d) => ({
        date: d.Date.slice(5),
        calories: Math.round(d.Consumed.Calories),
        target: Math.round(d.Targets.Calories),
      })),
    [days],
  )
  const targetLine = chartData.find((d) => d.target > 0)?.target ?? 0
  const calColor = cssVar('--color-cal')

  if (range.isLoading) {
    return (
      <Card className="p-5">
        <Spinner label={t('weeklyDashboard.loadingWeek')} />
      </Card>
    )
  }

  if (stats.loggedDays === 0) {
    return (
      <EmptyState
        title={t('weeklyDashboard.emptyTitle')}
        hint={t('weeklyDashboard.emptyHint')}
      />
    )
  }

  return (
    <div className="flex flex-col gap-5">
      {/* Stat tiles */}
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
        <StatTile
          label={t('weeklyDashboard.avgCalories')}
          value={Math.round(stats.avg.Calories)}
          unit="kcal"
          trend={stats.calorieTrend}
        />
        <StatTile
          label={t('weeklyDashboard.avgProtein')}
          value={Math.round(stats.avg.Protein)}
          unit="g"
          trend={stats.proteinTrend}
        />
        <StatTile label={t('weeklyDashboard.adherence')} value={Math.round(stats.adherence * 100)} unit="%" />
      </div>

      {/* Daily calories chart */}
      <Card className="p-5">
        <Eyebrow>{t('weeklyDashboard.last7DaysCalories')}</Eyebrow>
        <div className="mt-4 h-56 w-full">
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={chartData} margin={{ top: 8, right: 8, bottom: 0, left: 8 }}>
              <CartesianGrid stroke="var(--color-line)" vertical={false} />
              <XAxis
                dataKey="date"
                tickLine={false}
                axisLine={false}
                fontSize={12}
                stroke="var(--color-muted)"
              />
              <YAxis
                tickLine={false}
                axisLine={false}
                fontSize={12}
                stroke="var(--color-muted)"
                width={56}
                tickFormatter={(v: number) => (v >= 1000 ? `${v / 1000}k` : String(v))}
              />
              <Tooltip
                cursor={{ fill: 'var(--color-surface-2)' }}
                contentStyle={{
                  background: 'var(--color-surface)',
                  border: '1px solid var(--color-line)',
                  borderRadius: 12,
                  color: 'var(--color-ink)',
                }}
              />
              {targetLine > 0 && (
                <ReferenceLine y={targetLine} stroke="var(--color-muted)" strokeDasharray="4 4" />
              )}
              <Bar dataKey="calories" fill={calColor} radius={[6, 6, 0, 0]} name={t('weeklyDashboard.calories')} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </Card>

      {/* Best / worst day */}
      <div className="grid gap-4 sm:grid-cols-2">
        {stats.bestDay ? (
          <DayCard title={t('weeklyDashboard.bestDay')} day={stats.bestDay} locale={i18n.language} />
        ) : (
          <EmptyState title={t('weeklyDashboard.notEnoughData')} />
        )}
        {stats.worstDay ? (
          <DayCard title={t('weeklyDashboard.worstDay')} day={stats.worstDay} locale={i18n.language} />
        ) : (
          <EmptyState title={t('weeklyDashboard.notEnoughData')} />
        )}
      </div>
    </div>
  )
}
