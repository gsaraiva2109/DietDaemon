// Goals & Planning — current targets, the TDEE breakdown for the user's
// profile, and a suggested adjustment. Profile edits + recalculation flow back
// through the onboarding wizard and the targets endpoint.

import { useMemo } from 'react'
import { motion } from 'framer-motion'
import {
  useProfile,
  useTargets,
  useTDEE,
  useWeightLog,
  useSetTargets,
} from '@/lib/queries'
import { useDemo } from '@/lib/demo'
import { PageHeader } from '@/components/PageHeader'
import { Button, Card, Eyebrow, EmptyState, Spinner } from '@/components/ui'
import { MACRO_KEYS, MACRO_META } from '@/lib/types'
import type { Macros } from '@/lib/types'
import { formatNumber } from '@/lib/format'
import { GoalIcon } from '@/components/icons'
import { TDEECard } from '@/components/TDEECard'
import { GoalSuggestion } from '@/components/GoalSuggestion'
import { fadeUp } from '@/lib/motion'

function ageFrom(birth: string): number {
  if (!birth) return 0
  const b = new Date(birth)
  if (Number.isNaN(b.getTime())) return 0
  return Math.floor((Date.now() - b.getTime()) / (365.25 * 24 * 3600 * 1000))
}

function openWizard() {
  window.dispatchEvent(new CustomEvent('dd:onboarding'))
}

export function Goals() {
  const { demo } = useDemo()
  const profile = useProfile()
  const targets = useTargets()
  const weight = useWeightLog(1)
  const setTargets = useSetTargets()

  const prof = profile.data
  const latestWeight = weight.data?.[weight.data.length - 1]?.weight_kg
  const age = prof ? ageFrom(prof.birth_date) : 0
  const weightKg = latestWeight ?? prof?.target_weight_kg ?? 0

  const tdeeParams =
    prof && weightKg > 0 && prof.height_cm > 0 && age > 0
      ? {
          weight_kg: weightKg,
          height_cm: prof.height_cm,
          age,
          gender: prof.gender,
          activity: prof.activity_level,
        }
      : null
  const tdee = useTDEE(tdeeParams)

  const recommended: Macros | null = useMemo(() => {
    if (!tdee.data || !prof) return null
    const cal =
      prof.goal === 'cut'
        ? tdee.data.cut_cal
        : prof.goal === 'bulk'
          ? tdee.data.bulk_cal
          : tdee.data.maintain_cal
    return {
      Calories: cal,
      Protein: tdee.data.protein_g,
      Carbs: tdee.data.carbs_g,
      Fat: tdee.data.fat_g,
      Fiber: 30,
    }
  }, [tdee.data, prof])

  if (profile.isLoading) {
    return (
      <div>
        <PageHeader eyebrow="Goals" title="Your plan" />
        <Spinner />
      </div>
    )
  }

  const hasTargets = Boolean(targets.data && targets.data.Calories > 0)

  return (
    <div>
      <PageHeader eyebrow="Goals" title="Your plan">
        {prof && (
          <Button variant="ghost" onClick={openWizard}>
            Edit profile
          </Button>
        )}
      </PageHeader>

      <motion.div variants={fadeUp} initial="hidden" animate="show" className="space-y-5">
        {/* Current targets */}
        <Card className="p-5">
          <div className="mb-4 flex items-center justify-between">
            <Eyebrow>Daily targets</Eyebrow>
            {recommended && (
              <Button
                variant="ghost"
                onClick={() => recommended && setTargets.mutate(recommended)}
                disabled={demo || setTargets.isPending}
              >
                {setTargets.isPending ? 'Saving…' : 'Recalculate targets'}
              </Button>
            )}
          </div>

          {targets.isLoading ? (
            <Spinner />
          ) : hasTargets && targets.data ? (
            <div className="grid grid-cols-2 gap-4 sm:grid-cols-5">
              {MACRO_KEYS.map((k) => (
                <div key={k}>
                  <div className="text-xs uppercase tracking-[0.1em] text-muted">{MACRO_META[k].label}</div>
                  <div className="mt-1 flex items-baseline gap-1">
                    <span className="text-2xl font-bold text-ink tnum">
                      {formatNumber(targets.data![k])}
                    </span>
                    <span className="text-xs text-muted">{MACRO_META[k].unit}</span>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <EmptyState
              icon={<GoalIcon width={28} height={28} />}
              title="No targets set yet"
              hint="Set up your profile to calculate a personalized calorie and macro plan."
            />
          )}

          {!hasTargets && !targets.isLoading && (
            <div className="mt-4">
              <Button onClick={openWizard}>Set up your profile</Button>
            </div>
          )}
        </Card>

        {/* TDEE breakdown */}
        {tdee.data && prof ? (
          <TDEECard result={tdee.data} goal={prof.goal} />
        ) : !prof ? (
          <Card className="p-5">
            <EmptyState
              icon={<GoalIcon width={28} height={28} />}
              title="Tell us about yourself"
              hint="Add your height, weight, age, and goal to see your energy budget."
            />
            <div className="mt-4">
              <Button onClick={openWizard}>Set up your profile</Button>
            </div>
          </Card>
        ) : null}

        {/* Suggested adjustment */}
        <GoalSuggestion />
      </motion.div>
    </div>
  )
}
