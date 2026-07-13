// Code-entry step for passwordless sign-in. The user requested a code by email
// (the mock logs it to its console); here they enter it to complete sign-in.
// Includes a resend button with a short cooldown. When the account has TOTP
// enabled, the verify step returns an MFA challenge instead of a session; we
// hand off to <MfaChallenge> which issues the session on success.

import { useEffect, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from '@/lib/auth'
import { useMagicRequest, useMagicVerifyCode } from '@/lib/queries'
import { isMfaChallenge } from '@/lib/types'
import { MfaChallenge } from '@/routes/Login'
import { Button, Field, FormError } from './ui'

const RESEND_COOLDOWN = 30

export function MagicCodeEntry({
  email,
  onVerified,
  onBack,
}: {
  email: string
  onVerified: () => void
  onBack: () => void
}) {
  const { t } = useTranslation()
  const { refresh } = useAuth()
  const verify = useMagicVerifyCode()
  const resend = useMagicRequest()
  const [code, setCode] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [cooldown, setCooldown] = useState(RESEND_COOLDOWN)
  const [challengeToken, setChallengeToken] = useState<string | null>(null)

  useEffect(() => {
    if (cooldown <= 0) return
    const t = setTimeout(() => setCooldown((c) => c - 1), 1000)
    return () => clearTimeout(t)
  }, [cooldown])

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    if (!code.trim()) return
    setError(null)
    try {
      const res = await verify.mutateAsync({ email, code: code.trim() })
      if (isMfaChallenge(res)) {
        setChallengeToken(res.challenge_token)
        return
      }
      await refresh()
      onVerified()
    } catch {
      setError(t('magicCodeEntry.invalidCode'))
    }
  }

  async function onResend() {
    try {
      await resend.mutateAsync(email)
    } finally {
      setCooldown(RESEND_COOLDOWN)
    }
  }

  if (challengeToken) {
    return (
      <MfaChallenge
        challengeToken={challengeToken}
        onVerified={() => {
          refresh()
          onVerified()
        }}
        onBack={() => {
          setChallengeToken(null)
          setCode('')
        }}
      />
    )
  }

  return (
    <form onSubmit={onSubmit} className="flex flex-col gap-4">
      <Field
        label={t('magicCodeEntry.codeLabel')}
        inputMode="numeric"
        autoComplete="one-time-code"
        autoFocus
        maxLength={6}
        value={code}
        disabled={verify.isPending}
        onChange={(e) => setCode(e.target.value.replace(/\D/g, ''))}
        placeholder="000000"
        hint={t('magicCodeEntry.sentTo', { email })}
      />
      <FormError>{error}</FormError>
      <Button type="submit" disabled={verify.isPending || code.length < 6}>
        {verify.isPending ? t('magicCodeEntry.verifying') : t('magicCodeEntry.signIn')}
      </Button>
      <div className="flex items-center justify-between text-sm">
        <button type="button" onClick={onBack} className="font-medium text-muted hover:text-ink">
          {t('magicCodeEntry.useDifferentEmail')}
        </button>
        <button
          type="button"
          onClick={onResend}
          disabled={cooldown > 0 || resend.isPending}
          className="font-medium text-primary hover:underline disabled:text-muted disabled:no-underline"
        >
          {cooldown > 0 ? t('magicCodeEntry.resendIn', { seconds: cooldown }) : t('magicCodeEntry.resendCode')}
        </button>
      </div>
    </form>
  )
}
