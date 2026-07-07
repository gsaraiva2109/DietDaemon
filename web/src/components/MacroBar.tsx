// Horizontal target bar, the primary affordance for bar-based layouts.
// Always pairs color with a text label + numbers (a11y: never color alone).

import { motion } from 'framer-motion'
import { easeOut } from '@/lib/motion'
import { AnimatedNumber } from './AnimatedNumber'
import { progress, isOverTarget, remaining, confidenceTier } from '@/lib/format'

interface Props {
  consumed: number
  target: number
  label: string
  unit: string
  color: string
  confidence?: number
}

export function MacroBar({ consumed, target, label, unit, color, confidence }: Props) {
  const p = progress(consumed, target)
  const over = isOverTarget(consumed, target)
  const left = remaining(consumed, target)
  const tier = confidenceTier(confidence ?? 1)
  const opacityClass = tier === 'high' ? '' : tier === 'medium' ? 'opacity-75' : 'opacity-50'

  return (
    <div>
      <div className="mb-1.5 flex items-baseline justify-between gap-3">
        <span className="text-sm font-semibold text-ink">{label}</span>
        <span className="text-sm text-muted tnum">
          <span className="font-semibold text-ink">
            <AnimatedNumber value={Math.round(consumed)} />
          </span>{' '}
          / {Math.round(target)} {unit}
        </span>
      </div>
      <div
        className="h-2.5 w-full overflow-hidden rounded-full bg-primary-soft"
        role="progressbar"
        aria-valuenow={Math.round(p * 100)}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-label={`${label} progress`}
      >
        <motion.div
          className={`h-full rounded-full ${opacityClass}`}
          style={{ background: over ? 'var(--color-accent)' : color }}
          initial={{ width: 0 }}
          animate={{ width: `${p * 100}%` }}
          transition={{ duration: 0.9, ease: easeOut }}
        />
      </div>
      <div className="mt-1 text-xs text-muted">
        {over ? (
          <span className="font-medium text-accent">{Math.round(consumed - target)} {unit} over</span>
        ) : (
          <span>
            <span className="font-medium text-ink">{Math.round(left)} {unit}</span> left
          </span>
        )}
      </div>
    </div>
  )
}
