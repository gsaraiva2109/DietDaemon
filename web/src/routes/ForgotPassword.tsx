// /forgot-password, request a reset link. The response is ALWAYS generic
// (never reveals whether an account exists), so we show the same confirmation
// regardless of outcome.

import { useState, type SyntheticEvent } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useForgotPassword } from '@/lib/queries'
import { AuthLayout } from '@/components/AuthLayout'
import { Button, Field } from '@/components/ui'

export function ForgotPassword() {
  const { t } = useTranslation()
  const forgot = useForgotPassword()
  const [email, setEmail] = useState('')
  const [sent, setSent] = useState(false)
  const [busy, setBusy] = useState(false)

  async function onSubmit(e: SyntheticEvent) {
    e.preventDefault()
    if (!email.trim()) return
    setBusy(true)
    try {
      await forgot.mutateAsync(email.trim().toLowerCase())
    } catch {
      // Deliberately ignore failures, never leak account existence.
    } finally {
      setBusy(false)
      setSent(true)
    }
  }

  if (sent) {
    return (
      <AuthLayout
        title={t('forgotPassword.checkEmailTitle')}
        subtitle={t('forgotPassword.checkEmailSubtitle')}
        footer={
          <Link to="/login" className="font-medium text-primary hover:underline">
            {t('forgotPassword.backToSignIn')}
          </Link>
        }
      >
        <span />
      </AuthLayout>
    )
  }

  return (
    <AuthLayout
      title={t('forgotPassword.title')}
      subtitle={t('forgotPassword.subtitle')}
      footer={
        <Link to="/login" className="font-medium text-primary hover:underline">
          {t('forgotPassword.backToSignIn')}
        </Link>
      }
    >
      <form onSubmit={onSubmit} className="flex flex-col gap-4">
        <Field
          label={t('forgotPassword.emailLabel')}
          type="email"
          autoComplete="email"
          autoFocus
          value={email}
          disabled={busy}
          onChange={(e) => setEmail(e.target.value)}
          placeholder={t('forgotPassword.emailPlaceholder')}
        />
        <Button type="submit" disabled={busy || !email.trim()}>
          {busy ? t('forgotPassword.sending') : t('forgotPassword.sendLink')}
        </Button>
      </form>
    </AuthLayout>
  )
}
