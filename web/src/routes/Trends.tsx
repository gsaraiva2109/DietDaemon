// Trends — multi-day macros vs targets. Functional baseline; richer charting
// and GSAP scroll choreography land in later tasks.

import { useMemo, useState } from 'react'
import {
  Area,
  AreaChart,
  CartesianGrid,
  Line,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { useRange } from '@/lib/queries'
import { PageHeader } from '@/components/PageHeader'
import { Card, EmptyState, Spinner } from '@/components/ui'
import { MACRO_KEYS, MACRO_META, type MacroKey } from '@/lib/types'
import { cssVar } from '@/lib/format'

function isoDaysAgo(n: number): string {
  const d = new Date()
  d.setDate(d.getDate() - n)
  return d.toISOString().slice(0, 10)
}

export function Trends() {
  const [days, setDays] = useState(14)
  const [macro, setMacro] = useState<MacroKey>('Calories')
  const start = isoDaysAgo(days - 1)
  const end = isoDaysAgo(0)
  const range = useRange(start, end)

  const data = useMemo(
    () =>
      (range.data ?? []).map((r) => ({
        date: r.Date.slice(5),
        consumed: Math.round(r.Consumed[macro]),
        target: Math.round(r.Targets[macro]),
      })),
    [range.data, macro],
  )

  const color = cssVar(MACRO_META[macro].colorVar)

  return (
    <div>
      <PageHeader eyebrow="Trends" title="Over time">
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
      </PageHeader>

      <div className="mb-5 flex flex-wrap gap-2">
        {MACRO_KEYS.map((k) => (
          <button
            key={k}
            onClick={() => setMacro(k)}
            className={`rounded-full border px-3 py-1.5 text-sm font-medium transition ${
              macro === k
                ? 'border-transparent bg-primary-soft text-primary'
                : 'border-line bg-surface text-muted hover:text-ink'
            }`}
          >
            {MACRO_META[k].label}
          </button>
        ))}
      </div>

      <Card className="p-5">
        {range.isLoading ? (
          <Spinner />
        ) : !data.length ? (
          <EmptyState title="No data in range" hint="Log meals across a few days to see trends." />
        ) : (
          <div className="h-72 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={data} margin={{ top: 8, right: 8, bottom: 0, left: 8 }}>
                <defs>
                  <linearGradient id="fill" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor={color} stopOpacity={0.25} />
                    <stop offset="100%" stopColor={color} stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid stroke="var(--color-line)" vertical={false} />
                <XAxis dataKey="date" tickLine={false} axisLine={false} fontSize={12} stroke="var(--color-muted)" />
                <YAxis tickLine={false} axisLine={false} fontSize={12} stroke="var(--color-muted)" width={56} tickFormatter={(v: number) => (v >= 1000 ? `${v / 1000}k` : String(v))} />
                <Tooltip
                  contentStyle={{
                    background: 'var(--color-surface)',
                    border: '1px solid var(--color-line)',
                    borderRadius: 12,
                    color: 'var(--color-ink)',
                  }}
                />
                {data[0]?.target > 0 && (
                  <ReferenceLine y={data[0].target} stroke="var(--color-muted)" strokeDasharray="4 4" />
                )}
                <Area
                  type="monotone"
                  dataKey="consumed"
                  stroke={color}
                  strokeWidth={2.5}
                  fill="url(#fill)"
                  name={MACRO_META[macro].label}
                />
                <Line type="monotone" dataKey="target" stroke="var(--color-muted)" strokeWidth={1} dot={false} name="Target" />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        )}
      </Card>
    </div>
  )
}
