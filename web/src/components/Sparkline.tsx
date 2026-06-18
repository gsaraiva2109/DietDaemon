// Tiny 7-day trend sparkline. No axes, no chrome — just the shape of recent
// days, with a soft gradient fill.

import { Area, AreaChart, ResponsiveContainer } from 'recharts'

export function Sparkline({ data, color }: { data: number[]; color: string }) {
  const points = data.map((v, i) => ({ i, v }))
  const id = `spark-${color.replace(/[^a-z0-9]/gi, '')}`
  return (
    <div className="h-12 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={points} margin={{ top: 4, right: 0, bottom: 0, left: 0 }}>
          <defs>
            <linearGradient id={id} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={color} stopOpacity={0.35} />
              <stop offset="100%" stopColor={color} stopOpacity={0} />
            </linearGradient>
          </defs>
          <Area type="monotone" dataKey="v" stroke={color} strokeWidth={2} fill={`url(#${id})`} isAnimationActive />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  )
}
