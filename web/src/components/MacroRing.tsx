// The signature element (DESIGN.md): a soft progress arc with a big centered
// remaining-to-target number. Apple-Health character. Color is paired with a
// label everywhere it's used, never carrying meaning alone.

import { motion } from 'framer-motion'
import { useTranslation } from 'react-i18next'
import { easeOut } from '@/lib/motion'
import { AnimatedNumber } from './AnimatedNumber'
import { progress, isOverTarget } from '@/lib/format'

interface Props {
  /** value consumed so far */
  consumed: number
  /** daily target (0 = unset) */
  target: number
  label: string
  unit: string
  /** CSS color (resolved) for the progress arc */
  color: string
  size?: number
  thickness?: number
  /** show remaining (default) or consumed in the center */
  center?: 'remaining' | 'consumed'
}

export function MacroRing({
  consumed,
  target,
  label,
  unit,
  color,
  size = 200,
  thickness = 14,
  center = 'remaining',
}: Props) {
  const { t } = useTranslation()
  const r = (size - thickness) / 2
  const circ = 2 * Math.PI * r
  const p = progress(consumed, target)
  const over = isOverTarget(consumed, target)
  const centerValue = center === 'remaining' ? Math.max(0, target - consumed) : consumed
  const centerLabel = center === 'remaining' ? (over ? t('macroRing.over') : t('macroRing.left')) : t('macroRing.eaten')
  const gid = `ring-${label}-${Math.round(size)}`

  return (
    <div
      className="relative inline-grid place-items-center"
      style={{ width: size, height: size }}
      role="img"
      aria-label={t('macroRing.ariaLabel', {
        label,
        consumed: Math.round(consumed),
        target: Math.round(target),
        unit,
        percent: Math.round(p * 100),
      })}
    >
      <svg width={size} height={size} className="-rotate-90">
        <defs>
          <linearGradient id={gid} x1="0" y1="0" x2="1" y2="1">
            <stop offset="0%" stopColor={over ? 'var(--color-accent)' : color} stopOpacity={0.65} />
            <stop offset="100%" stopColor={over ? 'var(--color-accent)' : color} stopOpacity={1} />
          </linearGradient>
        </defs>
        <circle
          cx={size / 2}
          cy={size / 2}
          r={r}
          fill="none"
          stroke="var(--color-primary-soft)"
          strokeWidth={thickness}
        />
        <motion.circle
          cx={size / 2}
          cy={size / 2}
          r={r}
          fill="none"
          stroke={`url(#${gid})`}
          strokeWidth={thickness}
          strokeLinecap="round"
          strokeDasharray={circ}
          initial={{ strokeDashoffset: circ }}
          animate={{ strokeDashoffset: circ * (1 - p) }}
          transition={{ duration: 1, ease: easeOut }}
        />
      </svg>
      <div className="absolute inset-0 grid place-items-center text-center">
        <div>
          <div className="text-4xl font-bold leading-none text-ink">
            <AnimatedNumber value={Math.round(centerValue)} />
          </div>
          <div className="mt-1 text-xs font-medium uppercase tracking-[0.14em] text-muted">
            {unit} {centerLabel}
          </div>
        </div>
      </div>
    </div>
  )
}
