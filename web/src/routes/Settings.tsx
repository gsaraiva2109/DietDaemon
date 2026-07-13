// Settings, editable daily targets (PUT /targets), theme, demo, token.

import { useState } from 'react'
import { motion } from 'framer-motion'
import { Link, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useToday, useSetTargets } from '@/lib/queries'
import { useAuth } from '@/lib/auth'
import { useDemo } from '@/lib/demo'
import { languages } from '@/lib/i18n'
import { PageHeader } from '@/components/PageHeader'
import { Button, Card, Pill, Spinner } from '@/components/ui'
import { ExportModal } from '@/components/ExportModal'
import {
  ChevronRight,
  FoodsIcon,
  GoalIcon,
  DownloadIcon,
  BodyIcon,
  SettingsIcon,
  LinkIcon,
  CheckIcon,
  SparkleIcon,
  ClockIcon,
  TrashIcon,
  GlobeIcon,
} from '@/components/icons'
import { MACRO_KEYS, MACRO_META, type Macros } from '@/lib/types'

const ZERO: Macros = { Calories: 0, Protein: 0, Carbs: 0, Fat: 0, Fiber: 0 }

export function Settings() {
  const today = useToday()
  const setTargets = useSetTargets()
  const { i18n, t } = useTranslation()
  const { logout } = useAuth()
  const { demo, setDemo } = useDemo()
  const navigate = useNavigate()
  const [exporting, setExporting] = useState(false)
  const [signingOut, setSigningOut] = useState(false)

  async function signOut() {
    setSigningOut(true)
    if (demo) setDemo(false)
    await logout()
    navigate('/login', { replace: true })
  }

  // null = not yet edited; derive values from server data. Targets can carry
  // long decimals from the TDEE calc, so round to whole units for display and
  // for whatever gets saved back.
  const [draft, setDraft] = useState<Macros | null>(null)
  const raw = today.data?.Targets ?? ZERO
  const serverTargets: Macros = {
    Calories: Math.round(raw.Calories),
    Protein: Math.round(raw.Protein),
    Carbs: Math.round(raw.Carbs),
    Fat: Math.round(raw.Fat),
    Fiber: Math.round(raw.Fiber),
  }
  const values = draft ?? serverTargets

  function set(k: keyof Macros, v: number) {
    setDraft((d) => ({ ...(d ?? values), [k]: v }))
  }

  return (
    <div>
      <PageHeader eyebrow={t('nav.settings')} title={t('settings.preferencesTitle')} />

      <Card className="mb-5 p-5">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="font-semibold text-ink">{t('settings.dailyTargetsTitle')}</h2>
          {demo && <Pill tone="muted">{t('settings.readOnly')}</Pill>}
        </div>

        {today.isLoading ? (
          <Spinner />
        ) : (
          <>
            <div className="grid grid-cols-2 gap-4 sm:grid-cols-5">
              {MACRO_KEYS.map((k) => (
                <label key={k} className="block">
                  <span className="mb-1 block text-xs uppercase tracking-[0.1em] text-muted">
                    {t(`common.macro.${k}`)}
                  </span>
                  <div className="flex items-baseline gap-1">
                    <input
                      type="number"
                      min={0}
                      value={values[k]}
                      disabled={demo}
                      onChange={(e) => set(k, Number(e.target.value))}
                      className="w-full rounded-lg border border-line bg-bg px-3 py-2 text-lg font-semibold text-ink outline-none transition focus:border-primary disabled:opacity-60 tnum"
                    />
                    <span className="text-sm text-muted">{MACRO_META[k].unit}</span>
                  </div>
                </label>
              ))}
            </div>

            <div className="mt-5 flex items-center gap-3">
              <Button
                onClick={() => setTargets.mutate(values)}
                disabled={demo || setTargets.isPending}
              >
                {setTargets.isPending ? t('settings.saving') : t('settings.saveTargets')}
              </Button>
              {setTargets.isSuccess && (
                <motion.span initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="text-sm font-medium text-primary">
                  {t('settings.saved')}
                </motion.span>
              )}
              {setTargets.isError && (
                <span className="text-sm font-medium text-accent" role="alert">
                  {setTargets.error instanceof Error ? setTargets.error.message : t('settings.saveFailed')}
                </span>
              )}
            </div>
            <p className="mt-3 text-xs text-muted">
              {t('settings.targetsHintPrefix')} <code className="rounded bg-surface-2 px-1">/target</code> {t('settings.targetsHintSuffix')}
            </p>
          </>
        )}
      </Card>

      <Card className="mb-5 p-2">
        <div className="flex flex-wrap items-center gap-3 px-3 py-3">
          <span className="text-muted"><GlobeIcon width={20} height={20} /></span>
          <span className="flex-1">
            <span className="block text-sm font-medium text-ink">{t('settings.language')}</span>
            <span className="block text-xs text-muted">{t('settings.languageHint')}</span>
          </span>
          <div className="flex items-center gap-1 rounded-full border border-line bg-surface-2 p-1">
            {languages.map(({ code, label }) => (
              <button
                key={code}
                type="button"
                title={label}
                aria-label={label}
                aria-pressed={i18n.resolvedLanguage === code}
                onClick={() => i18n.changeLanguage(code)}
                className={`rounded-full px-3 py-1.5 text-xs font-semibold uppercase tracking-wide transition ${
                  i18n.resolvedLanguage === code
                    ? 'bg-primary text-primary-ink'
                    : 'text-muted hover:text-ink'
                }`}
              >
                {code}
              </button>
            ))}
          </div>
        </div>
      </Card>

      {/* Manage, links to the new feature surfaces. */}
      <Card className="mb-5 p-2">
        <RowLink to="/settings/security" Icon={SettingsIcon} label={t('settings.securityLabel')} hint={t('settings.securityHint')} />
        <RowLink to="/settings/link-bot" Icon={LinkIcon} label={t('settings.linkBotLabel')} hint={t('settings.linkBotHint')} />
        <RowLink to="/goals" Icon={GoalIcon} label={t('settings.bodyGoalsLabel')} hint={t('settings.bodyGoalsHint')} />
        <RowLink to="/settings/aliases" Icon={FoodsIcon} label={t('settings.foodAliasesLabel')} hint={t('settings.foodAliasesHint')} />
        <RowLink
          to="/settings/aliases/pending"
          Icon={CheckIcon}
          label={t('settings.pendingAliasesLabel')}
          hint={t('settings.pendingAliasesHint')}
        />
        <RowLink
          to="/settings/precedence"
          Icon={SparkleIcon}
          label={t('settings.precedenceLabel')}
          hint={t('settings.precedenceHint')}
        />
        <RowLink to="/settings/nudges" Icon={ClockIcon} label={t('settings.nudgesLabel')} hint={t('settings.nudgesHint')} />
        <RowLink to="/settings/backup" Icon={ClockIcon} label={t('settings.backupLabel')} hint={t('settings.backupHint')} />
        <RowLink to="/settings/ai-key" Icon={SettingsIcon} label={t('settings.aiKeyLabel')} hint={t('settings.aiKeyHint')} />
        <RowLink to="/settings/assistant" Icon={SparkleIcon} label={t('settings.assistantLabel')} hint={t('settings.assistantHint')} />
        <RowLink to="/settings/deleted-chats" Icon={TrashIcon} label={t('settings.deletedChatsLabel')} hint={t('settings.deletedChatsHint')} />
        <RowLink to="/settings/hevy-import" Icon={DownloadIcon} label={t('settings.hevyImportLabel')} hint={t('settings.hevyImportHint')} />
        <button
          onClick={() => window.dispatchEvent(new CustomEvent('dd:onboarding'))}
          className="flex w-full items-center gap-3 rounded-lg px-3 py-3 text-left transition hover:bg-surface-2"
        >
          <span className="text-muted"><BodyIcon width={20} height={20} /></span>
          <span className="flex-1">
            <span className="block text-sm font-medium text-ink">{t('settings.editBodyProfileLabel')}</span>
            <span className="block text-xs text-muted">{t('settings.editBodyProfileHint')}</span>
          </span>
          <ChevronRight width={18} height={18} className="text-muted" />
        </button>
        <button
          onClick={() => setExporting(true)}
          className="flex w-full items-center gap-3 rounded-lg px-3 py-3 text-left transition hover:bg-surface-2"
        >
          <span className="text-muted"><DownloadIcon width={20} height={20} /></span>
          <span className="flex-1">
            <span className="block text-sm font-medium text-ink">{t('settings.exportDataLabel')}</span>
            <span className="block text-xs text-muted">{t('settings.exportDataHint')}</span>
          </span>
          <ChevronRight width={18} height={18} className="text-muted" />
        </button>
      </Card>

      <Card className="p-5">
        <h2 className="mb-1 font-semibold text-ink">{t('settings.sessionTitle')}</h2>
        <p className="mb-4 text-sm text-muted">
          {t('settings.sessionDesc')}
        </p>
        <Button variant="ghost" onClick={signOut} disabled={signingOut}>
          {signingOut ? t('settings.signingOut') : t('settings.signOut')}
        </Button>
      </Card>

      {exporting && <ExportModal onClose={() => setExporting(false)} />}
    </div>
  )
}

function RowLink({
  to,
  Icon,
  label,
  hint,
}: {
  to: string
  Icon: typeof FoodsIcon
  label: string
  hint: string
}) {
  return (
    <Link
      to={to}
      className="flex items-center gap-3 rounded-lg px-3 py-3 transition hover:bg-surface-2"
    >
      <span className="text-muted"><Icon width={20} height={20} /></span>
      <span className="flex-1">
        <span className="block text-sm font-medium text-ink">{label}</span>
        <span className="block text-xs text-muted">{hint}</span>
      </span>
      <ChevronRight width={18} height={18} className="text-muted" />
    </Link>
  )
}
