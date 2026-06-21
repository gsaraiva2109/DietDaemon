// SleepCard, last night's duration + quality, plus a 7-day bar chart of hours
// slept. Indigo accent paired with the "Sleep" label and a quality Pill. Backend
// is Phase 4 (404 → empty state). Chart styling mirrors WeightChart.

import { useMemo } from 'react'
import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { useSleep } from '@/lib/queries'
import { Card, Eyebrow, Pill, Spinner } from '@/components/ui'
import { MoonIcon } from '@/components/icons'
import type { SleepLog, SleepQuality } from '@/lib/types'

const INDIGO = 'var(--color-protein)'

const QUALITY_TONE: Record<SleepQuality, 'primary' | 'neutral' | 'accent' | 'muted'> = {
  great: 'primary',
  good: 'primary',
  fair: 'neutral',
  poor: 'accent',
}

export function SleepCard() {
  const sleep = useSleep(7)
  const logs = sleep.data ?? []

  // Oldest → newest for the chart; the API returns newest-first.
  const chart = useMemo(
    () =>
      [...logs]
        .reverse()
        .map((s: SleepLog) => ({
          day: s.logged_at.slice(5, 10),
          hours: Math.round(s.duration_hours * 10) / 10,
        })),
    [logs],
  )
  const last = logs[0]

  return (
    <Card className="flex h-full flex-col gap-4 p-5">
      <header className="flex items-center justify-between">
        <div className="flex items-center gap-2" style={{ color: INDIGO }}>
          <MoonIcon width={18} height={18} />
          <Eyebrow>Sleep</Eyebrow>
        </div>
        {last && <Pill tone={QUALITY_TONE[last.quality]}>{last.quality}</Pill>}
      </header>

      {sleep.isLoading ? (
        <Spinner />
      ) : sleep.isError ? (
        <button
          onClick={() => sleep.refetch()}
          className="self-start text-sm font-medium text-accent hover:underline"
        >
          Couldn't load, retry
        </button>
      ) : logs.length === 0 ? (
        <p className="text-sm text-muted">No sleep data yet. Log a night from your chat bot.</p>
      ) : (
        <>
          <div className="flex items-baseline gap-1.5">
            <span className="text-3xl font-bold text-ink tnum">{last.duration_hours.toFixed(1)}</span>
            <span className="text-sm text-muted">hrs last night</span>
          </div>

          <div className="mt-auto h-28 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={chart} margin={{ top: 4, right: 4, bottom: 0, left: -16 }}>
                <CartesianGrid stroke="var(--color-line)" vertical={false} />
                <XAxis
                  dataKey="day"
                  tickLine={false}
                  axisLine={false}
                  fontSize={11}
                  stroke="var(--color-muted)"
                />
                <YAxis
                  tickLine={false}
                  axisLine={false}
                  fontSize={11}
                  stroke="var(--color-muted)"
                  width={32}
                  tickFormatter={(v: number) => `${Math.round(v)}`}
                />
                <Tooltip
                  contentStyle={{
                    background: 'var(--color-surface)',
                    border: '1px solid var(--color-line)',
                    borderRadius: 12,
                    color: 'var(--color-ink)',
                  }}
                />
                <Bar
                  dataKey="hours"
                  fill={INDIGO}
                  fillOpacity={0.7}
                  radius={[3, 3, 0, 0]}
                  name="Hours"
                  isAnimationActive={false}
                />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </>
      )}
    </Card>
  )
}
