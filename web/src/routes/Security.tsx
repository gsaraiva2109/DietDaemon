// Security, machine API keys + change password. Single-level Cards, matching
// Settings. A freshly created key's raw secret is revealed exactly once in a
// scaleIn panel with a copy button; after that only metadata is ever shown.

import { useState, type SubmitEvent } from 'react'
import { motion } from 'framer-motion'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import {
  useApiKeys,
  useCreateApiKey,
  useRevokeApiKey,
  useChangePassword,
  useTotpDisable,
  useRegenerateRecovery,
  useChangeEmail,
} from '@/lib/queries'
import { useAuth } from '@/lib/auth'
import { useDemo } from '@/lib/demo'
import { PageHeader } from '@/components/PageHeader'
import { Button, Card, Field, FormError, Input, Pill, Spinner } from '@/components/ui'
import { TotpEnroll } from '@/components/TotpEnroll'
import { RecoveryCodes } from '@/components/RecoveryCodes'
import { LinkedAccounts } from '@/components/LinkedAccounts'
import { PasskeyManager } from '@/components/PasskeyManager'
import { CopyIcon, TrashIcon } from '@/components/icons'
import { scaleIn } from '@/lib/motion'
import type { NewApiKey } from '@/lib/types'

export function Security() {
  const { t } = useTranslation()
  const { demo } = useDemo()

  return (
    <div>
      <PageHeader eyebrow={t('nav.settings')} title={t('security.title')} />
      <TwoFactorCard demo={demo} />
      <PasskeysCard demo={demo} />
      <LinkedAccountsCard demo={demo} />
      <ApiKeysCard demo={demo} />
      <ChangeEmailCard demo={demo} />
      <ChangePasswordCard demo={demo} />
    </div>
  )
}

function ChangeEmailCard({ demo }: { demo: boolean }) {
  const { t } = useTranslation()
  const { user, refresh } = useAuth()
  const change = useChangeEmail()
  const [email, setEmail] = useState('')
  const [error, setError] = useState<string | null>(null)

  async function onSubmit(e: SubmitEvent<HTMLFormElement>) {
    e.preventDefault()
    setError(null)
    const next = email.trim().toLowerCase()
    if (!next || next === user?.email) {
      setError(t('security.emailEnterNew'))
      return
    }
    try {
      await change.mutateAsync(next)
      await refresh()
      toast.success(t('security.emailVerificationSent'))
      setEmail('')
    } catch {
      setError(t('security.emailChangeFailed'))
    }
  }

  return (
    <Card className="mb-5 p-5">
      <div className="mb-1 flex items-center justify-between">
        <h2 className="font-semibold text-ink">{t('security.emailAddress')}</h2>
        {demo && <Pill tone="muted">{t('security.readOnly')}</Pill>}
      </div>
      <p className="mb-4 text-sm text-muted">
        {t('security.emailCurrentLabel')}{' '}
        <span className="font-medium text-ink">{user?.email ?? t('security.notAvailable')}</span>.{' '}
        {t('security.emailChangeNote')}
      </p>
      <form onSubmit={onSubmit} className="flex max-w-sm flex-col gap-3">
        <Field
          label={t('security.newEmailLabel')}
          type="email"
          autoComplete="email"
          value={email}
          disabled={demo || change.isPending}
          onChange={(e) => setEmail(e.target.value)}
          placeholder={t('security.emailPlaceholder')}
          error={error ?? undefined}
        />
        <Button type="submit" disabled={demo || change.isPending || !email.trim()} className="self-start">
          {change.isPending ? t('security.saving') : t('security.changeEmailButton')}
        </Button>
      </form>
    </Card>
  )
}

function PasskeysCard({ demo }: { demo: boolean }) {
  const { t } = useTranslation()
  return (
    <Card className="mb-5 p-5">
      <div className="mb-1 flex items-center justify-between">
        <h2 className="font-semibold text-ink">{t('security.passkeysTitle')}</h2>
        {demo && <Pill tone="muted">{t('security.readOnly')}</Pill>}
      </div>
      <p className="mb-4 text-sm text-muted">
        {t('security.passkeysDesc')}
      </p>
      {demo ? (
        <p className="text-sm text-muted">{t('security.passkeysDemoNote')}</p>
      ) : (
        <PasskeyManager />
      )}
    </Card>
  )
}

function LinkedAccountsCard({ demo }: { demo: boolean }) {
  const { t } = useTranslation()
  return (
    <Card className="mb-5 p-5">
      <div className="mb-1 flex items-center justify-between">
        <h2 className="font-semibold text-ink">{t('security.linkedAccountsTitle')}</h2>
        {demo && <Pill tone="muted">{t('security.readOnly')}</Pill>}
      </div>
      <p className="mb-4 text-sm text-muted">
        {t('security.linkedAccountsDesc')}
      </p>
      {demo ? (
        <p className="text-sm text-muted">{t('security.linkedAccountsDemoNote')}</p>
      ) : (
        <LinkedAccounts />
      )}
    </Card>
  )
}

function TwoFactorCard({ demo }: { demo: boolean }) {
  const { t } = useTranslation()
  const { user, refresh } = useAuth()
  const disable = useTotpDisable()
  const regen = useRegenerateRecovery()
  const [enrolling, setEnrolling] = useState(false)
  const [recovery, setRecovery] = useState<string[] | null>(null)
  const enabled = Boolean(user?.totp_enabled)

  async function onEnrolled() {
    setEnrolling(false)
    setRecovery(null)
    await refresh()
    toast.success(t('security.twoFactorEnabledToast'))
  }

  async function onDisable() {
    try {
      await disable.mutateAsync()
      await refresh()
      toast.success(t('security.twoFactorDisabledToast'))
    } catch {
      toast.error(t('security.twoFactorDisableFailed'))
    }
  }

  async function onRegenerate() {
    try {
      const res = await regen.mutateAsync()
      setRecovery(res.recovery_codes)
    } catch {
      toast.error(t('security.recoveryRegenFailed'))
    }
  }

  return (
    <Card className="mb-5 p-5">
      <div className="mb-1 flex items-center justify-between">
        <h2 className="font-semibold text-ink">{t('security.twoFactorTitle')}</h2>
        {demo ? (
          <Pill tone="muted">{t('security.readOnly')}</Pill>
        ) : (
          <Pill tone={enabled ? 'primary' : 'muted'}>{enabled ? t('security.twoFactorOn') : t('security.twoFactorOff')}</Pill>
        )}
      </div>
      <p className="mb-4 text-sm text-muted">
        {t('security.twoFactorDesc')}
      </p>

      {demo ? (
        <p className="text-sm text-muted">{t('security.twoFactorDemoNote')}</p>
      ) : enrolling ? (
        <TotpEnroll onComplete={onEnrolled} onCancel={() => setEnrolling(false)} />
      ) : recovery ? (
        <RecoveryCodes codes={recovery} onDone={() => setRecovery(null)} />
      ) : enabled ? (
        <div className="flex flex-wrap gap-2">
          <Button variant="ghost" onClick={onRegenerate} disabled={regen.isPending}>
            {regen.isPending ? t('security.generating') : t('security.regenerateRecoveryCodes')}
          </Button>
          <Button variant="ghost" onClick={onDisable} disabled={disable.isPending}>
            {disable.isPending ? t('security.disabling') : t('security.disableTwoFactor')}
          </Button>
        </div>
      ) : (
        <Button onClick={() => setEnrolling(true)}>{t('security.enableTwoFactor')}</Button>
      )}
    </Card>
  )
}

function ApiKeysCard({ demo }: { demo: boolean }) {
  const { t, i18n } = useTranslation()
  const keys = useApiKeys()
  const create = useCreateApiKey()
  const revoke = useRevokeApiKey()
  const [label, setLabel] = useState('')
  const [fresh, setFresh] = useState<NewApiKey | null>(null)

  async function onCreate(e: SubmitEvent<HTMLFormElement>) {
    e.preventDefault()
    if (!label.trim()) return
    const key = await create.mutateAsync(label.trim())
    setFresh(key)
    setLabel('')
  }

  async function copyKey(value: string) {
    try {
      await navigator.clipboard.writeText(value)
      toast.success(t('security.copyKeySuccess'))
    } catch {
      toast.error(t('security.copyKeyFailed'))
    }
  }

  const list = keys.data ?? []
  const active = list.filter((k) => !k.revoked_at)

  return (
    <Card className="mb-5 p-5">
      <div className="mb-1 flex items-center justify-between">
        <h2 className="font-semibold text-ink">{t('security.apiKeysTitle')}</h2>
        {demo && <Pill tone="muted">{t('security.readOnly')}</Pill>}
      </div>
      <p className="mb-4 text-sm text-muted">
        {t('security.apiKeysDesc')}{' '}
        <code className="rounded bg-surface-2 px-1 tnum">Authorization: Bearer ddk_…</code>
      </p>

      {fresh && (
        <motion.div
          variants={scaleIn}
          initial="hidden"
          animate="show"
          className="mb-4 rounded-xl border border-primary/40 bg-primary-soft/50 p-4"
        >
          <p className="text-sm font-medium text-ink">
            {t('security.apiKeyCopyNowNote')}
          </p>
          <div className="mt-3 flex items-center gap-2">
            <code className="flex-1 overflow-x-auto rounded-lg border border-line bg-surface px-3 py-2 text-sm text-ink tnum">
              {fresh.key}
            </code>
            <Button type="button" onClick={() => copyKey(fresh.key)} className="shrink-0">
              <CopyIcon width={16} height={16} /> {t('security.copy')}
            </Button>
          </div>
          <button
            onClick={() => setFresh(null)}
            className="mt-3 text-xs font-medium text-muted hover:text-ink"
          >
            {t('security.dismissSavedKey')}
          </button>
        </motion.div>
      )}

      <form onSubmit={onCreate} className="mb-5 flex flex-col gap-2 sm:flex-row">
        <Input
          value={label}
          disabled={demo || create.isPending}
          onChange={(e) => setLabel(e.target.value)}
          placeholder={t('security.keyLabelPlaceholder')}
          aria-label={t('security.apiKeyLabelAria')}
        />
        <Button type="submit" disabled={demo || create.isPending || !label.trim()} className="shrink-0">
          {create.isPending ? t('security.creating') : t('security.createKeyButton')}
        </Button>
      </form>

      {keys.isLoading ? (
        <Spinner />
      ) : active.length === 0 ? (
        <p className="text-sm text-muted">{t('security.noApiKeysYet')}</p>
      ) : (
        <ul className="flex flex-col divide-y divide-line">
          {active.map((k) => (
            <li key={k.id} className="flex items-center justify-between gap-3 py-3">
              <div className="min-w-0">
                <p className="truncate text-sm font-medium text-ink">{k.label}</p>
                <p className="text-xs text-muted">
                  {t('security.createdOn')} {new Date(k.created_at).toLocaleDateString(i18n.language)}
                  {k.last_used_at && ` · ${t('security.lastUsedOn')} ${new Date(k.last_used_at).toLocaleDateString(i18n.language)}`}
                </p>
              </div>
              <button
                onClick={() => revoke.mutate(k.id)}
                disabled={revoke.isPending}
                className="grid size-9 shrink-0 place-items-center rounded-lg text-muted transition hover:bg-surface-2 hover:text-accent disabled:opacity-50"
                aria-label={t('security.revokeAria', { label: k.label })}
              >
                <TrashIcon width={18} height={18} />
              </button>
            </li>
          ))}
        </ul>
      )}
    </Card>
  )
}

function ChangePasswordCard({ demo }: { demo: boolean }) {
  const { t } = useTranslation()
  const change = useChangePassword()
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState<string | null>(null)

  async function onSubmit(e: SubmitEvent<HTMLFormElement>) {
    e.preventDefault()
    setError(null)
    if (next !== confirm) {
      setError(t('security.passwordsMismatch'))
      return
    }
    if (next.length < 8) {
      setError(t('security.newPasswordHint'))
      return
    }
    try {
      await change.mutateAsync({ current, next })
      toast.success(t('security.passwordChanged'))
      setCurrent('')
      setNext('')
      setConfirm('')
    } catch {
      setError(t('security.passwordChangeFailed'))
    }
  }

  return (
    <Card className="p-5">
      <div className="mb-4 flex items-center justify-between">
        <h2 className="font-semibold text-ink">{t('security.changePasswordTitle')}</h2>
        {demo && <Pill tone="muted">{t('security.readOnly')}</Pill>}
      </div>
      <form onSubmit={onSubmit} className="flex max-w-sm flex-col gap-4">
        <Field
          label={t('security.currentPasswordLabel')}
          type="password"
          autoComplete="current-password"
          value={current}
          disabled={demo || change.isPending}
          onChange={(e) => setCurrent(e.target.value)}
        />
        <Field
          label={t('security.newPasswordLabel')}
          type="password"
          autoComplete="new-password"
          value={next}
          disabled={demo || change.isPending}
          onChange={(e) => setNext(e.target.value)}
          hint={t('security.newPasswordHint')}
        />
        <Field
          label={t('security.confirmNewPasswordLabel')}
          type="password"
          autoComplete="new-password"
          value={confirm}
          disabled={demo || change.isPending}
          onChange={(e) => setConfirm(e.target.value)}
        />
        <FormError>{error}</FormError>
        <Button
          type="submit"
          disabled={demo || change.isPending || !current || !next}
          className="self-start"
        >
          {change.isPending ? t('security.saving') : t('security.updatePasswordButton')}
        </Button>
      </form>
    </Card>
  )
}
