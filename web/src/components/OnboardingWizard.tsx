// Calm 4-step onboarding overlay. Opens automatically for a not-yet-onboarded
// user (never in demo), and on demand via a window 'dd:onboarding' CustomEvent
// for "edit profile", in which case it pre-fills and "Skip" becomes "Cancel".

import { useEffect, useMemo, useState } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import { useTranslation } from 'react-i18next'
import {
  useProfile,
  useLogWeight,
  useUpsertProfile,
  useSetTargets,
  useTDEE,
} from '@/lib/queries'
import { useDemo } from '@/lib/demo'
import type { Macros, UserProfile } from '@/lib/types'
import { ACTIVITY_LEVELS, GOALS } from '@/lib/types'
import { Button } from './ui'
import { CloseIcon, ChevronLeft, ChevronRight, CheckIcon } from './icons'
import { TDEECard } from './TDEECard'
import { easeOut } from '@/lib/motion'

interface Draft {
  height_cm: number
  weight_kg: number // local only, feeds TDEE, not persisted to the profile
  birth_date: string
  gender: 'male' | 'female'
  activity_level: string
  goal: 'cut' | 'maintain' | 'bulk'
  target_weight_kg: number
  weekly_rate: number
}

const TOTAL_STEPS = 4

function ageFrom(birth: string): number {
  if (!birth) return 0
  const b = new Date(birth)
  if (Number.isNaN(b.getTime())) return 0
  const diff = Date.now() - b.getTime()
  return Math.floor(diff / (365.25 * 24 * 3600 * 1000))
}

function emptyDraft(): Draft {
  return {
    height_cm: 0,
    weight_kg: 0,
    birth_date: '',
    gender: 'male',
    activity_level: 'moderate',
    goal: 'cut',
    target_weight_kg: 0,
    weekly_rate: 0.5,
  }
}

// Decimal-friendly numeric input. A plain type="number" controlled by a number
// strips a trailing "." on each keystroke, so "82.5" is impossible to type.
// We hold the raw text locally, accept partial decimals while typing, and only
// resync from the prop when it changes externally (e.g. edit-mode prefill).
function NumberField({
  label,
  value,
  unit,
  onChange,
}: {
  label: string
  value: number
  unit: string
  onChange: (v: number) => void
}) {
  const [text, setText] = useState(() => (value ? String(value) : ''))

  // Resync from the prop only when it changes externally (e.g. edit-mode
  // prefill), not on our own keystrokes. Adjusting state during render with a
  // previous-value guard is the documented alternative to a sync effect.
  const [prevValue, setPrevValue] = useState(value)
  if (value !== prevValue) {
    setPrevValue(value)
    const parsed = parseFloat(text)
    const current = Number.isNaN(parsed) ? 0 : parsed
    if (current !== value) setText(value ? String(value) : '')
  }

  function handle(raw: string) {
    // Allow only digits and a single decimal point (incl. partial "0." / ".5").
    if (!/^\d*\.?\d*$/.test(raw)) return
    setText(raw)
    const parsed = parseFloat(raw)
    onChange(Number.isNaN(parsed) ? 0 : parsed)
  }

  return (
    <label className="block">
      <span className="mb-1 block text-xs font-medium text-muted">{label}</span>
      <div className="flex items-baseline gap-1">
        <input
          type="text"
          inputMode="decimal"
          value={text}
          onChange={(e) => handle(e.target.value)}
          className="w-full rounded-lg border border-line bg-bg px-3 py-2 text-ink outline-none transition focus:border-primary tnum"
        />
        <span className="text-sm text-muted">{unit}</span>
      </div>
    </label>
  )
}

export function OnboardingWizard() {
  const { t, i18n } = useTranslation()
  const { demo } = useDemo()
  const { data: profile, isLoading } = useProfile()
  const logWeight = useLogWeight()
  const upsert = useUpsertProfile()
  const setTargets = useSetTargets()

  const [step, setStep] = useState(0)
  const [dismissed, setDismissed] = useState(false)
  const [editMode, setEditMode] = useState(false)
  const [draft, setDraft] = useState<Draft>(emptyDraft)

  // Open in edit mode on the global event, pre-filling from the current profile.
  useEffect(() => {
    function open() {
      setDraft((d) => ({
        ...emptyDraft(),
        ...d,
        height_cm: profile?.height_cm || d.height_cm,
        birth_date: profile?.birth_date || d.birth_date,
        gender: (profile?.gender as Draft['gender']) || d.gender,
        activity_level: profile?.activity_level || d.activity_level,
        goal: (profile?.goal as Draft['goal']) || d.goal,
        target_weight_kg: profile?.target_weight_kg || d.target_weight_kg,
        weekly_rate: profile?.weekly_rate ?? d.weekly_rate,
      }))
      setEditMode(true)
      setDismissed(false)
      setStep(0)
    }
    window.addEventListener('dd:onboarding', open)
    return () => window.removeEventListener('dd:onboarding', open)
  }, [profile])

  function set<K extends keyof Draft>(key: K, value: Draft[K]) {
    setDraft((d) => ({ ...d, [key]: value }))
  }

  const age = ageFrom(draft.birth_date)
  const tdeeParams =
    step === TOTAL_STEPS - 1 && draft.weight_kg > 0 && draft.height_cm > 0 && age > 0
      ? {
          weight_kg: draft.weight_kg,
          height_cm: draft.height_cm,
          age,
          gender: draft.gender,
          activity: draft.activity_level,
        }
      : null
  const { data: tdee } = useTDEE(tdeeParams)

  const visible = !isLoading && !demo && !dismissed && (editMode || profile?.onboarded !== true)

  const recommended: Macros | null = useMemo(() => {
    if (!tdee) return null
    const cal =
      draft.goal === 'cut' ? tdee.cut_cal : draft.goal === 'bulk' ? tdee.bulk_cal : tdee.maintain_cal
    return { Calories: cal, Protein: tdee.protein_g, Carbs: tdee.carbs_g, Fat: tdee.fat_g, Fiber: 30 }
  }, [tdee, draft.goal])

  const stepValid = (() => {
    switch (step) {
      case 0:
        return draft.height_cm > 0 && draft.weight_kg > 0 && draft.birth_date !== '' && age > 0
      case 1:
        return Boolean(draft.activity_level)
      case 2:
        return Boolean(draft.goal) && draft.target_weight_kg > 0 && draft.weekly_rate > 0
      default:
        return true
    }
  })()

  function close() {
    setDismissed(true)
    setEditMode(false)
    setStep(0)
  }

  function profilePayload(): UserProfile {
    return {
      user_id: profile?.user_id ?? '',
      height_cm: draft.height_cm,
      birth_date: draft.birth_date,
      gender: draft.gender,
      activity_level: draft.activity_level,
      goal: draft.goal,
      target_weight_kg: draft.target_weight_kg,
      weekly_rate: draft.weekly_rate,
      onboarded: true,
      created_at: profile?.created_at ?? '',
      updated_at: profile?.updated_at ?? '',
    }
  }

  function save() {
    upsert.mutate(profilePayload(), { onSuccess: close })
    if (!editMode && draft.weight_kg > 0) {
      logWeight.mutate({ date: new Date().toISOString().slice(0, 10), weightKg: draft.weight_kg })
    }
    if (recommended) setTargets.mutate(recommended)
  }

  function skipOrCancel() {
    if (editMode) {
      close()
      return
    }
    // Mark onboarded with whatever was filled, so it won't reappear.
    upsert.mutate(profilePayload(), { onSuccess: close })
    if (!editMode && draft.weight_kg > 0) {
      logWeight.mutate({ date: new Date().toISOString().slice(0, 10), weightKg: draft.weight_kg })
    }
  }

  return (
    <AnimatePresence>
      {visible && (
      <motion.div
        key="onboarding"
        className="fixed inset-0 grid place-items-center p-4"
        style={{ zIndex: 1500 }}
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        transition={{ duration: 0.3, ease: easeOut }}
      >
        <div className="absolute inset-0 bg-ink/40 backdrop-blur-sm" style={{ zIndex: 1400 }} />
        <motion.div
          role="dialog"
          aria-modal="true"
          aria-label={t('onboardingWizard.dialogLabel')}
          initial={{ opacity: 0, scale: 0.97, y: 10 }}
          animate={{ opacity: 1, scale: 1, y: 0 }}
          exit={{ opacity: 0, scale: 0.96, y: 8 }}
          transition={{ duration: 0.3, ease: easeOut }}
          className="relative w-full max-w-lg rounded-xl border border-line bg-surface p-6 shadow-lift"
          style={{ zIndex: 1500 }}
        >
          <div className="mb-5 flex items-start justify-between">
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted">
                {editMode ? t('onboardingWizard.editProfileEyebrow') : t('onboardingWizard.welcomeEyebrow')}
              </p>
              <h2 className="mt-1 text-xl font-bold text-ink">
                {step === 0 && t('onboardingWizard.stepBodyStats')}
                {step === 1 && t('onboardingWizard.stepActivity')}
                {step === 2 && t('onboardingWizard.stepGoal')}
                {step === 3 && t('onboardingWizard.stepPlan')}
              </h2>
            </div>
            {editMode && (
              <button onClick={close} aria-label={t('onboardingWizard.close')} className="text-muted hover:text-ink">
                <CloseIcon />
              </button>
            )}
          </div>

          {/* progress dots */}
          <div className="mb-6 flex items-center gap-1.5">
            {Array.from({ length: TOTAL_STEPS }).map((_, i) => (
              <span
                key={i}
                className={`h-1.5 flex-1 rounded-full transition-colors ${
                  i <= step ? 'bg-primary' : 'bg-surface-2'
                }`}
              />
            ))}
          </div>

          <div className="min-h-[16rem]">
            <AnimatePresence mode="wait">
              <motion.div
                key={step}
                initial={{ opacity: 0, x: 16 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0, x: -16 }}
                transition={{ duration: 0.3, ease: easeOut }}
              >
                {step === 0 && (
                  <div className="space-y-4">
                    <div className="grid grid-cols-2 gap-3">
                      <NumberField
                        label={t('onboardingWizard.height')}
                        value={draft.height_cm}
                        unit="cm"
                        onChange={(v) => set('height_cm', v)}
                      />
                      <NumberField
                        label={t('onboardingWizard.currentWeight')}
                        value={draft.weight_kg}
                        unit="kg"
                        onChange={(v) => set('weight_kg', v)}
                      />
                    </div>
                    <label className="block">
                      <span className="mb-1 block text-xs font-medium text-muted">{t('onboardingWizard.dateOfBirth')}</span>
                      <input
                        type="date"
                        lang={i18n.language}
                        value={draft.birth_date}
                        onChange={(e) => set('birth_date', e.target.value)}
                        className="w-full rounded-lg border border-line bg-bg px-3 py-2 text-ink outline-none transition focus:border-primary"
                      />
                      <p className="mt-1 text-xs text-muted">
                        {t('onboardingWizard.dateFormatHint', {
                          example: new Intl.DateTimeFormat(i18n.language, { dateStyle: 'short' }).format(new Date()),
                        })}
                      </p>
                    </label>
                    <div>
                      <span className="mb-1.5 block text-xs font-medium text-muted">{t('onboardingWizard.gender')}</span>
                      <div className="flex gap-2">
                        {(['male', 'female'] as const).map((g) => (
                          <button
                            key={g}
                            onClick={() => set('gender', g)}
                            className={`flex-1 rounded-full border px-4 py-2 text-sm font-medium capitalize transition ${
                              draft.gender === g
                                ? 'border-transparent bg-primary-soft text-primary'
                                : 'border-line bg-surface text-ink hover:bg-surface-2'
                            }`}
                          >
                            {g === 'male' ? t('onboardingWizard.male') : t('onboardingWizard.female')}
                          </button>
                        ))}
                      </div>
                    </div>
                  </div>
                )}

                {step === 1 && (
                  <div className="space-y-2">
                    {ACTIVITY_LEVELS.map((a) => {
                      const active = draft.activity_level === a.value
                      return (
                        <button
                          key={a.value}
                          onClick={() => set('activity_level', a.value)}
                          className={`flex w-full items-start gap-3 rounded-xl border px-4 py-3 text-left transition ${
                            active
                              ? 'border-transparent bg-primary-soft'
                              : 'border-line bg-surface hover:bg-surface-2'
                          }`}
                        >
                          <span
                            className={`mt-0.5 grid size-5 shrink-0 place-items-center rounded-full border ${
                              active ? 'border-primary bg-primary text-primary-ink' : 'border-line'
                            }`}
                          >
                            {active && <CheckIcon width={12} height={12} />}
                          </span>
                          <span>
                            <span className={`block text-sm font-semibold ${active ? 'text-primary' : 'text-ink'}`}>
                              {t(`onboardingWizard.activity.${a.value}.label`)}
                            </span>
                            <span className="block text-xs text-muted">{t(`onboardingWizard.activity.${a.value}.hint`)}</span>
                          </span>
                        </button>
                      )
                    })}
                  </div>
                )}

                {step === 2 && (
                  <div className="space-y-4">
                    <div className="grid grid-cols-3 gap-2">
                      {GOALS.map((g) => {
                        const active = draft.goal === g.value
                        return (
                          <button
                            key={g.value}
                            onClick={() => set('goal', g.value as Draft['goal'])}
                            className={`rounded-xl border px-3 py-3 text-center transition ${
                              active
                                ? 'border-transparent bg-primary-soft text-primary'
                                : 'border-line bg-surface text-ink hover:bg-surface-2'
                            }`}
                          >
                            <span className="block text-sm font-semibold">{t(`onboardingWizard.goal.${g.value}.label`)}</span>
                            <span className="mt-0.5 block text-[11px] leading-tight text-muted">{t(`onboardingWizard.goal.${g.value}.hint`)}</span>
                          </button>
                        )
                      })}
                    </div>
                    <div className="grid grid-cols-2 gap-3">
                      <NumberField
                        label={t('onboardingWizard.targetWeight')}
                        value={draft.target_weight_kg}
                        unit="kg"
                        onChange={(v) => set('target_weight_kg', v)}
                      />
                      <NumberField
                        label={t('onboardingWizard.weeklyRate')}
                        value={draft.weekly_rate}
                        unit={t('goalSuggestion.weeklyUnit')}
                        onChange={(v) => set('weekly_rate', v)}
                      />
                    </div>
                  </div>
                )}

                {step === 3 && (
                  <div>
                    {tdee ? (
                      <TDEECard result={tdee} goal={draft.goal} />
                    ) : (
                      <p className="py-12 text-center text-sm text-muted">{t('onboardingWizard.crunchingNumbers')}</p>
                    )}
                  </div>
                )}
              </motion.div>
            </AnimatePresence>
          </div>

          <div className="mt-6 flex items-center justify-between gap-2">
            <div>
              {step > 0 && (
                <Button variant="ghost" onClick={() => setStep((s) => s - 1)}>
                  <ChevronLeft width={16} height={16} /> {t('onboardingWizard.back')}
                </Button>
              )}
            </div>
            <div className="flex items-center gap-2">
              <Button variant="ghost" onClick={skipOrCancel} disabled={upsert.isPending}>
                {editMode ? t('onboardingWizard.cancel') : t('onboardingWizard.skip')}
              </Button>
              {step < TOTAL_STEPS - 1 ? (
                <Button onClick={() => setStep((s) => s + 1)} disabled={!stepValid}>
                  {t('onboardingWizard.next')} <ChevronRight width={16} height={16} />
                </Button>
              ) : (
                <Button onClick={save} disabled={upsert.isPending}>
                  {upsert.isPending ? t('onboardingWizard.saving') : t('onboardingWizard.savePlan')}
                </Button>
              )}
            </div>
          </div>
        </motion.div>
      </motion.div>
      )}
    </AnimatePresence>
  )
}
