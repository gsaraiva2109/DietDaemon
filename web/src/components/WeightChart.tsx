// Weight over time — raw daily points plus a smoother rolling average, with an
// optional faint calorie-intake bar overlay on a second axis. Styled to match
// Trends.tsx (grid var(--color-line), muted axes, surface tooltip).

import { useMemo } from 'react'
import {
  Bar,
  CartesianGrid,
  ComposedChart,
  Line,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { EmptyState } from './ui'
import { cssVar } from '@/lib/format'
import type { WeightTrend } from '@/lib/types'

interface IntakePoint {
  date: string
  calories: number
}

export function WeightChart({
  trend,
  intake,
}: {
  trend: WeightTrend[]
  intake?: IntakePoint[]
}) {
  const data = useMemo(() => {
    const kcalByDate = new Map<string, number>(
      (intake ?? []).map((p) => [p.date, p.calories]),
    )
    return trend.map((t) => ({
      date: t.date.slice(5),
      weight_kg: t.weight_kg,
      rolling_avg: t.rolling_avg,
      calories: kcalByDate.get(t.date) ?? null,
    }))
  }, [trend, intake])

  if (!data.length) {
    return (
      <EmptyState
        title="No weight logged yet"
        hint="Log a weigh-in above to start charting your trend."
      />
    )
  }

  const primary = cssVar('--color-primary') || cssVar('--color-cal')
  const calColor = cssVar('--color-cal')

  return (
    <div className="h-72 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <ComposedChart data={data} margin={{ top: 8, right: 8, bottom: 0, left: 8 }}>
          <CartesianGrid stroke="var(--color-line)" vertical={false} />
          <XAxis
            dataKey="date"
            tickLine={false}
            axisLine={false}
            fontSize={12}
            stroke="var(--color-muted)"
          />
          <YAxis
            yAxisId="kg"
            tickLine={false}
            axisLine={false}
            fontSize={12}
            stroke="var(--color-muted)"
            width={48}
            domain={['dataMin - 1', 'dataMax + 1']}
            tickFormatter={(v: number) => `${Math.round(v)}`}
          />
          {intake && intake.length > 0 && (
            <YAxis
              yAxisId="kcal"
              orientation="right"
              tickLine={false}
              axisLine={false}
              fontSize={12}
              stroke="var(--color-muted)"
              width={48}
              tickFormatter={(v: number) => (v >= 1000 ? `${Math.round(v / 1000)}k` : String(v))}
            />
          )}
          <Tooltip
            contentStyle={{
              background: 'var(--color-surface)',
              border: '1px solid var(--color-line)',
              borderRadius: 12,
              color: 'var(--color-ink)',
            }}
          />
          {intake && intake.length > 0 && (
            <Bar
              yAxisId="kcal"
              dataKey="calories"
              fill={calColor}
              fillOpacity={0.14}
              radius={[3, 3, 0, 0]}
              name="Calories"
              isAnimationActive={false}
            />
          )}
          <Line
            yAxisId="kg"
            type="monotone"
            dataKey="weight_kg"
            stroke="var(--color-muted)"
            strokeWidth={1}
            dot={{ r: 1.5, fill: 'var(--color-muted)' }}
            activeDot={{ r: 3 }}
            name="Weight"
          />
          <Line
            yAxisId="kg"
            type="monotone"
            dataKey="rolling_avg"
            stroke={primary}
            strokeWidth={2.5}
            dot={false}
            name="Trend"
          />
        </ComposedChart>
      </ResponsiveContainer>
    </div>
  )
}
