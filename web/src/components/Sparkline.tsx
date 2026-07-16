// Tiny 7-day trend sparkline. No axes, no chrome, just the shape of recent
// days, with a soft gradient fill.

import { useId } from 'react'

export function Sparkline({ data, color }: { data: number[]; color: string }) {
  const id = `spark-${useId().replace(/:/g, '')}`
  const min = Math.min(...data)
  const range = Math.max(...data) - min || 1
  const points = data
    .map((value, index) => {
      const x = data.length === 1 ? 50 : (index / (data.length - 1)) * 100
      const y = 44 - ((value - min) / range) * 40
      return `${x},${y}`
    })
    .join(' ')
  return (
    <div className="h-12 w-full">
      <svg viewBox="0 0 100 48" preserveAspectRatio="none" className="size-full" aria-hidden="true">
        <defs>
          <linearGradient id={id} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor={color} stopOpacity={0.35} />
            <stop offset="100%" stopColor={color} stopOpacity={0} />
          </linearGradient>
        </defs>
        <polygon points={`0,48 ${points} 100,48`} fill={`url(#${id})`} />
        <polyline points={points} fill="none" stroke={color} strokeWidth={2} vectorEffect="non-scaling-stroke" />
      </svg>
    </div>
  )
}
