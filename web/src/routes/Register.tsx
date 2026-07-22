// Register, email + password + display name, inside AuthLayout. Gated by the
// server's registration_mode: hidden entirely when 'oidc-only'. Errors stay
// generic (no per-field server detail).

import { useState, type SyntheticEvent } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from '@/lib/auth'
import { useProviders } from '@/lib/queries'
import { RateLimitError } from '@/lib/api'
import { AuthLayout } from '@/components/AuthLayout'
import { ProviderButtons } from '@/components/ProviderButtons'
import { Button, Field, FormError } from '@/components/ui'

export function Register() {
  const { t } = useTranslation()
  const { register } = useAuth()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const next = params.get('next') || '/'
  const providers = useProviders()
  // oidc-only → no password form; create an account through a provider.
  const oidcOnly = providers.data?.registration_mode === 'oidc-only'

  const [email, setEmail] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function onSubmit(e: SyntheticEvent) {
    e.preventDefault()
    if (!email.trim() || !password) return
    setBusy(true)
    setError(null)
    try {
      await register(email, password, displayName)
      navigate(next, { replace: true })
    } catch (err) {
      setError(err instanceof RateLimitError ? t('register.tooManyAttempts') : t('register.genericError'))
    } finally {
      setBusy(false)
    }
  }

  return (
    <AuthLayout
      title={t('register.title')}
      subtitle={oidcOnly ? t('register.oidcSubtitle') : t('register.subtitle')}
      footer={
        <>
          {t('register.alreadyHaveAccount')}{' '}
          <Link to="/login" className="font-medium text-primary hover:underline">
            {t('register.signIn')}
          </Link>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        {!oidcOnly && (
          <form onSubmit={onSubmit} className="flex flex-col gap-4">
            <Field
              label={t('register.displayNameLabel')}
              type="text"
              autoComplete="name"
              autoFocus
              value={displayName}
              disabled={busy}
              onChange={(e) => setDisplayName(e.target.value)}
              placeholder={t('register.displayNamePlaceholder')}
              hint={t('register.displayNameHint')}
            />
            <Field
              label={t('register.emailLabel')}
              type="email"
              autoComplete="email"
              value={email}
              disabled={busy}
              onChange={(e) => setEmail(e.target.value)}
              placeholder={t('register.emailPlaceholder')}
            />
            <Field
              label={t('register.passwordLabel')}
              type="password"
              autoComplete="new-password"
              value={password}
              disabled={busy}
              onChange={(e) => setPassword(e.target.value)}
              placeholder={t('register.passwordPlaceholder')}
              hint={t('register.passwordHint')}
            />
            <FormError>{error}</FormError>
            <Button type="submit" disabled={busy || !email.trim() || !password}>
              {busy ? t('register.creating') : t('register.createAccount')}
            </Button>
          </form>
        )}
        <ProviderButtons verb="signup" />
      </div>
    </AuthLayout>
  )
}
