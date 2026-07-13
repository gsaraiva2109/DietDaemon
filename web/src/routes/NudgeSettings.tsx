// Nudge settings, a settings sub-page listing every scheduler nudge rule
// (macro, health, weekly digest) with a per-rule enable toggle, inline
// editors for its tunable numbers, and a reset-to-default action. Mirrors
// Aliases.tsx (settings sub-page shell, read-only-in-demo) and Settings.tsx
// (numeric field + save affordance).

import { useState } from 'react'
import { Link } from 'react-router-dom'
import { motion } from 'framer-motion'
import { useTranslation } from 'react-i18next'
import { useNudgeRules, useSetNudgeRule, useResetNudgeRule } from '@/lib/queries'
import { useDemo } from '@/lib/demo'
import { PageHeader } from '@/components/PageHeader'
import { Button, Card, Eyebrow, EmptyState, Spinner, Toggle } from '@/components/ui'
import { ChevronLeft, GoalIcon } from '@/components/icons'
import type { NudgeRuleView } from '@/lib/types'
import { stagger, fadeUp } from '@/lib/motion'

// Which of the rule's own JSON fields are safe to tune from the UI, per
// rule "group" (macro rules; health rules further split by Domain; digest).
// Fields not listed here (Message, ID, MaxGapHours — currently unused by the
// scheduler) stay hidden rather than offering controls that do nothing.
// labelKey is looked up under nudgeSettings.fields.<labelKey> at render time
// (this object is module scope, outside any component, so it can't call
// useTranslation() itself).
const EDITABLE_FIELDS: Record<string, { key: string; labelKey: string; min?: number; max?: number; step?: number }[]> = {
  macro: [
    { key: 'AfterHour', labelKey: 'afterHour', min: 0, max: 23 },
    { key: 'MinFraction', labelKey: 'minFractionMet', min: 0, max: 1, step: 0.05 },
  ],
  water: [
    { key: 'CheckHour', labelKey: 'checkHour', min: 0, max: 23 },
    { key: 'MinDailyAmount', labelKey: 'minDailyAmount', min: 0, step: 50 },
  ],
  workout: [
    { key: 'CheckHour', labelKey: 'checkHour', min: 0, max: 23 },
    { key: 'MaxGapDays', labelKey: 'maxGapDays', min: 1, max: 14 },
  ],
  sleep: [{ key: 'CheckHour', labelKey: 'checkHour', min: 0, max: 23 }],
  fasting: [],
  digest: [{ key: 'CheckHour', labelKey: 'checkHour', min: 0, max: 23 }],
  'weekly-budget': [
    { key: 'CheckHour', labelKey: 'checkHour', min: 0, max: 23 },
    { key: 'WeeklyTargetOverride', labelKey: 'weeklyTargetOverride', min: 0, step: 50 },
    { key: 'ClampFloorPct', labelKey: 'clampFloorPct', min: 0.1, max: 1.5, step: 0.05 },
    { key: 'ClampCeilPct', labelKey: 'clampCeilPct', min: 0.1, max: 2.0, step: 0.05 },
  ],
  'smart-meal': [],
}

function titleFromID(id: string): string {
  return id
    .split('-')
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ')
}

export function NudgeSettings() {
  const { t } = useTranslation()
  const { demo } = useDemo()
  const { data, isLoading } = useNudgeRules()

  const rules = data ?? []
  const macro = rules.filter((r) => r.kind === 'macro')
  const health = rules.filter((r) => r.kind === 'health')
  const digest = rules.filter((r) => r.kind === 'digest')
  const weeklyBudget = rules.filter((r) => r.kind === 'weekly-budget')
  const smartMeal = rules.filter((r) => r.kind === 'smart-meal')

  return (
    <div>
      <Link
        to="/settings"
        className="inline-flex items-center gap-1 text-sm text-muted hover:text-ink"
      >
        <ChevronLeft width={18} height={18} /> {t('nav.settings')}
      </Link>

      <PageHeader eyebrow={t('nav.settings')} title={t('nudgeSettings.title')} />

      {demo && (
        <p className="mb-5 rounded-xl border border-line bg-surface-2 px-4 py-2.5 text-sm text-muted">
          {t('nudgeSettings.readOnly')}
        </p>
      )}

      {isLoading ? (
        <Spinner label={t('nudgeSettings.loading')} />
      ) : !rules.length ? (
        <EmptyState
          icon={<GoalIcon width={28} height={28} />}
          title={t('nudgeSettings.emptyTitle')}
          hint={t('nudgeSettings.emptyHint')}
        />
      ) : (
        <motion.div variants={stagger} initial="hidden" animate="show" className="space-y-6">
          <RuleGroup title={t('nudgeSettings.groups.macro')} rules={macro} demo={demo} />
          <RuleGroup title={t('nudgeSettings.groups.health')} rules={health} demo={demo} />
          <RuleGroup title={t('nudgeSettings.groups.weeklyBudget')} rules={weeklyBudget} demo={demo} />
          <RuleGroup title={t('nudgeSettings.groups.smartMeal')} rules={smartMeal} demo={demo} />
          <RuleGroup title={t('nudgeSettings.groups.digest')} rules={digest} demo={demo} />
        </motion.div>
      )}
    </div>
  )
}

function RuleGroup({ title, rules, demo }: { title: string; rules: NudgeRuleView[]; demo: boolean }) {
  if (!rules.length) return null
  return (
    <section>
      <Eyebrow>{title}</Eyebrow>
      <div className="mt-2 flex flex-col gap-3">
        {rules.map((r) => (
          <motion.div key={r.rule_id} variants={fadeUp}>
            <NudgeRuleRow view={r} demo={demo} />
          </motion.div>
        ))}
      </div>
    </section>
  )
}

function NudgeRuleRow({ view, demo }: { view: NudgeRuleView; demo: boolean }) {
  const { t } = useTranslation()
  const setRule = useSetNudgeRule()
  const resetRule = useResetNudgeRule()
  const [draft, setDraft] = useState<Record<string, unknown> | null>(null)

  const values = draft ?? view.rule
  const groupKey = view.kind === 'health' ? String(view.rule.Domain ?? '') : view.kind
  const fields = EDITABLE_FIELDS[groupKey] ?? []
  const message = typeof view.rule.Message === 'string' ? view.rule.Message : null
  const dirty = draft !== null

  function setField(key: string, v: number) {
    setDraft({ ...(draft ?? view.rule), [key]: v })
  }

  function save() {
    if (demo) return
    const params: Record<string, unknown> = {}
    for (const f of fields) params[f.key] = values[f.key]
    setRule.mutate(
      { rule_id: view.rule_id, enabled: view.enabled, params },
      { onSuccess: () => setDraft(null) },
    )
  }

  function toggle(next: boolean) {
    if (demo) return
    setRule.mutate({ rule_id: view.rule_id, enabled: next })
  }

  function reset() {
    if (demo) return
    resetRule.mutate(view.rule_id, { onSuccess: () => setDraft(null) })
  }

  return (
    <Card className="p-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="font-semibold text-ink">{titleFromID(view.rule_id)}</p>
          {message && <p className="mt-0.5 text-sm text-muted">{message}</p>}
        </div>
        <Toggle
          checked={view.enabled}
          onChange={toggle}
          disabled={demo || setRule.isPending}
          label={t('nudgeSettings.enableRule', { rule: titleFromID(view.rule_id) })}
        />
      </div>

      {fields.length > 0 && (
        <div className="mt-3 flex flex-wrap items-end gap-3">
          {fields.map((f) => (
            <label key={f.key} className="block">
              <span className="mb-1 block text-xs uppercase tracking-[0.1em] text-muted">
                {t(`nudgeSettings.fields.${f.labelKey}`)}
              </span>
              <input
                type="number"
                min={f.min}
                max={f.max}
                step={f.step ?? 1}
                value={Number(values[f.key] ?? 0)}
                disabled={demo}
                onChange={(e) => setField(f.key, Number(e.target.value))}
                className="w-32 rounded-lg border border-line bg-bg px-3 py-1.5 text-sm font-medium text-ink outline-none transition focus:border-primary disabled:opacity-60 tnum"
              />
            </label>
          ))}
        </div>
      )}

      {!demo && (
        <div className="mt-3 flex items-center gap-3">
          {fields.length > 0 && (
            <Button
              variant="ghost"
              className="px-4 py-1.5 text-xs"
              onClick={save}
              disabled={!dirty || setRule.isPending}
            >
              {setRule.isPending ? t('nudgeSettings.saving') : t('nudgeSettings.save')}
            </Button>
          )}
          <Button
            variant="ghost"
            className="px-4 py-1.5 text-xs"
            onClick={reset}
            disabled={resetRule.isPending}
          >
            {t('nudgeSettings.resetToDefault')}
          </Button>
        </div>
      )}
    </Card>
  )
}
