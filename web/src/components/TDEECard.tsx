// Visual breakdown of energy budget: BMR -> TDEE -> goal calories, plus the
// recommended macro split. One Card; numbers use tnum. Highlights the active
// goal so the wizard's final step and the Goals page read the same.

import { motion } from 'framer-motion'
import type { TDEEResult } from '@/lib/types'
import { GOALS } from '@/lib/types'
import { formatNumber, cssVar } from '@/lib/format'
import { Card, Eyebrow } from './ui'
import { fadeUp, numberSpring } from '@/lib/motion'

const GOAL_CAL: Record<string, keyof Pick<TDEEResult, 'cut_cal' | 'maintain_cal' | 'bulk_cal'>> = {
  cut: 'cut_cal',
  maintain: 'maintain_cal',
  bulk: 'bulk_cal',
}

function Bar({ label, value, max, tone }: { label: string; value: number; max: number; tone: 'muted' | 'primary' }) {
  const ratio = max > 0 ? Math.min(1, value / max) : 0
  return (
    <div>
      <div className="mb-1 flex items-baseline justify-between">
        <span className="text-xs font-medium text-muted">{label}</span>
        <span className="text-sm font-semibold text-ink tnum">{formatNumber(value)} kcal</span>
      </div>
      <div className="h-2.5 overflow-hidden rounded-full bg-surface-2">
        <motion.div
          className={`h-full rounded-full ${tone === 'primary' ? 'bg-primary' : 'bg-primary-soft'}`}
          initial={{ width: 0 }}
          animate={{ width: `${ratio * 100}%` }}
          transition={{ duration: 0.6, ease: [0.16, 1, 0.3, 1] }}
        />
      </div>
    </div>
  )
}

export function TDEECard({ result, goal }: { result: TDEEResult; goal?: string }) {
  const goalCals: Array<{ value: string; label: string; cal: number }> = GOALS.map((g) => ({
    value: g.value,
    label: g.label,
    cal: result[GOAL_CAL[g.value]] ?? 0,
  }))
  const maxBar = Math.max(result.tdee, result.bmr, ...goalCals.map((g) => g.cal), 1)

  const macros = [
    { key: 'Protein', g: result.protein_g, color: cssVar('--color-protein') },
    { key: 'Carbs', g: result.carbs_g, color: cssVar('--color-carbs') },
    { key: 'Fat', g: result.fat_g, color: cssVar('--color-fat') },
  ]
  const macroKcal = result.protein_g * 4 + result.carbs_g * 4 + result.fat_g * 9

  return (
    <Card className="p-5">
      <Eyebrow>Energy budget</Eyebrow>

      <div className="mt-4 space-y-3.5">
        <Bar label="BMR, at rest" value={result.bmr} max={maxBar} tone="muted" />
        <Bar label="TDEE, maintenance" value={result.tdee} max={maxBar} tone="primary" />
      </div>

      <div className="mt-5">
        <p className="mb-2 text-xs font-medium text-muted">Daily target by goal</p>
        <div className="grid grid-cols-3 gap-2">
          {goalCals.map((g) => {
            const active = g.value === goal
            return (
              <motion.div
                key={g.value}
                variants={fadeUp}
                initial="hidden"
                animate="show"
                className={`rounded-xl border px-3 py-3 text-center transition ${
                  active
                    ? 'border-transparent bg-primary-soft text-primary'
                    : 'border-line bg-surface-2 text-ink'
                }`}
              >
                <div className="text-[11px] font-semibold uppercase tracking-[0.12em]">{g.label}</div>
                <motion.div
                  className="mt-1 text-lg font-bold tnum"
                  initial={{ opacity: 0, y: 4 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={numberSpring}
                >
                  {formatNumber(g.cal)}
                </motion.div>
                <div className="text-[10px] uppercase tracking-[0.12em] text-muted">kcal</div>
              </motion.div>
            )
          })}
        </div>
      </div>

      <div className="mt-5">
        <p className="mb-2 text-xs font-medium text-muted">Recommended macros</p>
        <div className="mb-2 flex h-2.5 overflow-hidden rounded-full bg-surface-2">
          {macros.map((mc) => {
            const kcal = mc.key === 'Fat' ? mc.g * 9 : mc.g * 4
            const w = macroKcal > 0 ? (kcal / macroKcal) * 100 : 0
            return <span key={mc.key} style={{ width: `${w}%`, background: mc.color }} />
          })}
        </div>
        <div className="grid grid-cols-3 gap-2">
          {macros.map((mc) => (
            <div key={mc.key} className="flex items-center gap-1.5 text-sm">
              <span className="size-2.5 shrink-0 rounded-full" style={{ background: mc.color }} />
              <span className="font-medium text-ink">{mc.key}</span>
              <span className="text-muted tnum">{Math.round(mc.g)}g</span>
            </div>
          ))}
        </div>
      </div>
    </Card>
  )
}
