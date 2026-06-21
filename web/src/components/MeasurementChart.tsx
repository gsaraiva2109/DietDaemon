// Body measurements over time, one line per measurement field, with pill
// toggles to show/hide individual series. Styled to match Trends.tsx.

import { useMemo, useState } from 'react'
import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { EmptyState } from './ui'
import { cssVar } from '@/lib/format'
import { MEASUREMENT_FIELDS, type MeasurementEntry, type MeasurementField } from '@/lib/types'

// Cycle the macro color vars across the (up to) 7 measurement series.
const COLOR_VARS = [
  '--color-cal',
  '--color-protein',
  '--color-carbs',
  '--color-fat',
  '--color-fiber',
  '--color-primary',
  '--color-accent',
] as const

export function MeasurementChart({ data }: { data: MeasurementEntry[] }) {
  const [visible, setVisible] = useState<Set<MeasurementField>>(
    () => new Set(MEASUREMENT_FIELDS.map((f) => f.key)),
  )

  const rows = useMemo(
    () =>
      data.map((e) => {
        const row: Record<string, number | string> = { date: e.date.slice(5) }
        for (const f of MEASUREMENT_FIELDS) row[f.key] = e[f.key]
        return row
      }),
    [data],
  )

  function toggle(key: MeasurementField) {
    setVisible((prev) => {
      const next = new Set(prev)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      return next
    })
  }

  if (!rows.length) {
    return (
      <EmptyState
        title="No measurements yet"
        hint="Log a set of measurements above to track them over time."
      />
    )
  }

  return (
    <div>
      <div className="mb-4 flex flex-wrap gap-2">
        {MEASUREMENT_FIELDS.map((f, i) => {
          const color = cssVar(COLOR_VARS[i % COLOR_VARS.length])
          const on = visible.has(f.key)
          return (
            <button
              key={f.key}
              onClick={() => toggle(f.key)}
              className={`inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-sm font-medium transition ${
                on
                  ? 'border-line bg-surface text-ink'
                  : 'border-line bg-surface text-muted opacity-60 hover:opacity-100'
              }`}
            >
              <span
                className="size-2.5 rounded-full"
                style={{ background: on ? color : 'var(--color-muted)' }}
              />
              {f.label}
            </button>
          )
        })}
      </div>

      <div className="h-72 w-full">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={rows} margin={{ top: 8, right: 8, bottom: 0, left: 8 }}>
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
              width={48}
              domain={['dataMin - 2', 'dataMax + 2']}
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
            {MEASUREMENT_FIELDS.map((f, i) =>
              visible.has(f.key) ? (
                <Line
                  key={f.key}
                  type="monotone"
                  dataKey={f.key}
                  name={f.label}
                  stroke={cssVar(COLOR_VARS[i % COLOR_VARS.length])}
                  strokeWidth={2}
                  dot={false}
                  isAnimationActive={false}
                />
              ) : null,
            )}
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  )
}
