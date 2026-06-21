// Self-contained nudge: compares current vs recommended intake and weekly loss,
// with a one-click Apply that nudges daily calorie targets toward the goal.
// Renders nothing when there's no actionable suggestion.

import { useState } from 'react'
import { motion } from 'framer-motion'
import { useGoalSuggestions, useSetTargets, useTargets } from '@/lib/queries'
import { useDemo } from '@/lib/demo'
import { formatNumber } from '@/lib/format'
import type { Macros } from '@/lib/types'
import { Card, Button } from './ui'
import { SparkleIcon, CheckIcon } from './icons'

const ZERO: Macros = { Calories: 0, Protein: 0, Carbs: 0, Fat: 0, Fiber: 0 }

function Stat({ label, current, target, unit }: { label: string; current: number; target: number; unit: string }) {
  return (
    <div className="rounded-xl border border-line bg-surface-2 px-3 py-2.5">
      <div className="text-[11px] font-medium uppercase tracking-[0.12em] text-muted">{label}</div>
      <div className="mt-1 flex items-baseline gap-1.5">
        <span className="text-lg font-bold text-ink tnum">{formatNumber(current)}</span>
        <span className="text-xs text-muted">→</span>
        <span className="text-lg font-bold text-primary tnum">{formatNumber(target)}</span>
        <span className="text-xs text-muted">{unit}</span>
      </div>
    </div>
  )
}

export function GoalSuggestion() {
  const { demo } = useDemo()
  const { data } = useGoalSuggestions()
  const targets = useTargets()
  const setTargets = useSetTargets()
  const [applied, setApplied] = useState(false)

  if (!data || data.recommended_kcal <= 0 || !data.message) return null

  function apply() {
    if (!data) return
    const base = targets.data ?? ZERO
    setTargets.mutate(
      { ...base, Calories: data.recommended_kcal },
      {
        onSuccess: () => {
          setApplied(true)
          setTimeout(() => setApplied(false), 2400)
        },
      },
    )
  }

  return (
    <Card className="p-5">
      <div className="flex items-start gap-3">
        <span className="mt-0.5 text-accent">
          <SparkleIcon />
        </span>
        <div className="min-w-0 flex-1">
          <h2 className="font-semibold text-ink">Suggested adjustment</h2>
          <p className="mt-1 text-sm text-muted">{data.message}</p>

          <div className="mt-4 grid grid-cols-1 gap-2 sm:grid-cols-2">
            <Stat
              label="Weekly loss"
              current={data.current_loss_kg}
              target={data.target_loss_kg}
              unit="kg/wk"
            />
            <Stat
              label="Daily intake"
              current={data.current_intake_kcal}
              target={data.recommended_kcal}
              unit="kcal"
            />
          </div>

          <div className="mt-4 flex items-center gap-3">
            <Button onClick={apply} disabled={demo || setTargets.isPending || applied}>
              {applied ? (
                <>
                  <CheckIcon width={16} height={16} /> Applied
                </>
              ) : setTargets.isPending ? (
                'Applying…'
              ) : (
                'Apply'
              )}
            </Button>
            {demo && <span className="text-xs text-muted">unavailable</span>}
            {setTargets.isError && (
              <motion.span
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                className="text-sm font-medium text-accent"
                role="alert"
              >
                {setTargets.error instanceof Error ? setTargets.error.message : 'Failed to apply'}
              </motion.span>
            )}
          </div>
        </div>
      </div>
    </Card>
  )
}
