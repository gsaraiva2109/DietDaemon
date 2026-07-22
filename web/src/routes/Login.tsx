// Login, email + password + remember, inside the calm AuthLayout. Errors are
// always generic (never reveal which field). A "View demo" button drops into
// demo mode (no backend). Honors ?next= to return where the guard sent us.

import { useState, type SyntheticEvent } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from '@/lib/auth'
import { useDemo, demoAvailable } from '@/lib/demo'
import { useProviders, useMagicRequest } from '@/lib/queries'
import { api, AUTH_ERROR, RateLimitError } from '@/lib/api'
import { loginWithPasskey, browserSupportsWebAuthn, isWebAuthnCancel } from '@/lib/webauthn'
import { isMfaChallenge } from '@/lib/types'
import { AuthLayout } from '@/components/AuthLayout'
import { ProviderButtons } from '@/components/ProviderButtons'
import { MagicCodeEntry } from '@/components/MagicCodeEntry'
import { Button, Field, FormError } from '@/components/ui'

export function Login() {
  const { t } = useTranslation()
  const { login, refresh } = useAuth()
  const { setDemo } = useDemo()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const next = params.get('next') || '/'
  const providers = useProviders()
  const oidcOnly = providers.data?.registration_mode === 'oidc-only'
  const canRegister = !oidcOnly

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [remember, setRemember] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  // Set when login defers to a second factor (TOTP).
  const [challengeToken, setChallengeToken] = useState<string | null>(null)
  // Set when the user requested a passwordless sign-in code.
  const [magicEmail, setMagicEmail] = useState<string | null>(null)
  const magicRequest = useMagicRequest()

  async function onSubmit(e: SyntheticEvent) {
    e.preventDefault()
    if (!email.trim() || !password) return
    setBusy(true)
    setError(null)
    try {
      const res = await login(email, password, remember)
      if (res.status === 'mfa_required') {
        setChallengeToken(res.challengeToken)
        return
      }
      navigate(next, { replace: true })
    } catch (err) {
      if (err instanceof RateLimitError) {
        const n = err.retryAfter
        setError(
          n ? t('login.tooManyAttemptsSeconds', { count: n }) : t('login.tooManyAttempts'),
        )
      } else {
        setError(AUTH_ERROR)
      }
    } finally {
      setBusy(false)
    }
  }

  function viewDemo() {
    setDemo(true)
    navigate('/', { replace: true })
  }

  async function signInWithPasskey() {
    setError(null)
    setBusy(true)
    try {
      const res = await loginWithPasskey()
      if (isMfaChallenge(res)) {
        setChallengeToken(res.challenge_token)
        return
      }
      await refresh()
      navigate(next, { replace: true })
    } catch (err) {
      if (!isWebAuthnCancel(err)) setError(t('login.passkeyFailed'))
    } finally {
      setBusy(false)
    }
  }

  async function emailMeCode() {
    const addr = email.trim().toLowerCase()
    if (!addr) {
      setError(t('login.enterEmailFirst'))
      return
    }
    setError(null)
    try {
      await magicRequest.mutateAsync(addr)
    } catch {
      // Generic, never leak whether the address exists.
    }
    setMagicEmail(addr)
  }

  if (magicEmail) {
    return (
      <AuthLayout
        title={t('login.checkEmailTitle')}
        subtitle={t('login.checkEmailSubtitle')}
      >
        <MagicCodeEntry
          email={magicEmail}
          onVerified={() => navigate(next, { replace: true })}
          onBack={() => setMagicEmail(null)}
        />
      </AuthLayout>
    )
  }

  if (challengeToken) {
    return (
      <MfaChallenge
        challengeToken={challengeToken}
        onVerified={() => navigate(next, { replace: true })}
        onBack={() => {
          setChallengeToken(null)
          setPassword('')
        }}
      />
    )
  }

  return (
    <AuthLayout
      title={t('login.welcomeBackTitle')}
      subtitle={t('login.welcomeBackSubtitle')}
      footer={
        canRegister && (
          <>
            {t('login.newHere')}{' '}
            <Link to="/register" className="font-medium text-primary hover:underline">
              {t('login.createAccount')}
            </Link>
          </>
        )
      }
    >
      <div className="flex flex-col gap-4">
        {!oidcOnly && (
          <form onSubmit={onSubmit} className="flex flex-col gap-4">
            <Field
              label={t('login.emailLabel')}
              type="email"
              autoComplete="email"
              autoFocus
              value={email}
              disabled={busy}
              onChange={(e) => setEmail(e.target.value)}
              placeholder={t('login.emailPlaceholder')}
            />
            <Field
              label={t('login.passwordLabel')}
              type="password"
              autoComplete="current-password"
              value={password}
              disabled={busy}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
            />
            <div className="flex items-center justify-between">
              <label className="flex items-center gap-2 text-sm text-muted">
                <input
                  type="checkbox"
                  checked={remember}
                  disabled={busy}
                  onChange={(e) => setRemember(e.target.checked)}
                  className="size-4 rounded border-line accent-primary"
                />
                {t('login.keepSignedIn')}
              </label>
              <Link
                to="/forgot-password"
                className="text-sm font-medium text-primary hover:underline"
              >
                {t('login.forgotPassword')}
              </Link>
            </div>
            <FormError>{error}</FormError>
            <Button type="submit" disabled={busy || !email.trim() || !password}>
              {busy ? t('login.signingIn') : t('login.signIn')}
            </Button>
            <Button
              type="button"
              variant="ghost"
              onClick={emailMeCode}
              disabled={busy || magicRequest.isPending}
            >
              {magicRequest.isPending ? t('login.sendingCode') : t('login.emailMeCode')}
            </Button>
          </form>
        )}
        <ProviderButtons verb="signin" />
        {browserSupportsWebAuthn() && (
          <Button type="button" variant="ghost" onClick={signInWithPasskey} disabled={busy}>
            {t('login.signInWithPasskey')}
          </Button>
        )}
        {demoAvailable() && (
          <Button type="button" variant="ghost" onClick={viewDemo} disabled={busy}>
            {t('login.viewDemo')}
          </Button>
        )}
      </div>
    </AuthLayout>
  )
}

// Second step of a 2FA login: TOTP, passkey, or email-OTP fallback.
export function MfaChallenge({
  challengeToken,
  onVerified,
  onBack,
}: Readonly<{
  challengeToken: string
  onVerified: () => void
  onBack: () => void
}>) {
  const { t } = useTranslation()
  const { verifyTotp, verifyMfaPasskey, verifyMfaEmail } = useAuth()
  const [code, setCode] = useState('')
  const [recovery, setRecovery] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  // Email-OTP state: null = not requested yet, 'sent' = code on the way, 'entering' = user is typing code.
  const [emailOtp, setEmailOtp] = useState<null | 'sent' | 'entering'>(null)

  async function onSubmit(e: SyntheticEvent) {
    e.preventDefault()
    if (!code.trim()) return
    setBusy(true)
    setError(null)
    try {
      if (emailOtp === 'entering') {
        await verifyMfaEmail(challengeToken, code.trim())
      } else {
        await verifyTotp(challengeToken, code.trim(), recovery)
      }
      onVerified()
    } catch (err) {
      setError(
        err instanceof RateLimitError ? t('login.tooManyAttempts') : t('login.codeMismatch'),
      )
    } finally {
      setBusy(false)
    }
  }

  async function usePasskey() {
    setError(null)
    setBusy(true)
    try {
      await verifyMfaPasskey(challengeToken)
      onVerified()
    } catch (err) {
      if (!isWebAuthnCancel(err)) setError(t('login.passkeyVerifyFailed'))
    } finally {
      setBusy(false)
    }
  }

  async function sendEmailCode() {
    setError(null)
    setBusy(true)
    try {
      await api.auth.mfa.emailSend(challengeToken)
      setEmailOtp('sent')
      setCode('')
    } catch {
      setError(t('login.sendCodeFailed'))
    } finally {
      setBusy(false)
    }
  }

  // Email-OTP code entry view.
  if (emailOtp === 'entering' || (emailOtp === 'sent')) {
    return (
      <AuthLayout
        title={t('login.emailVerificationTitle')}
        subtitle={t('login.emailVerificationSubtitle')}
        footer={
          <button type="button" onClick={onBack} className="font-medium text-primary hover:underline">
            {t('login.backToSignIn')}
          </button>
        }
      >
        <form onSubmit={onSubmit} className="flex flex-col gap-4">
          <Button
            type="button"
            variant="ghost"
            onClick={() => {
              setEmailOtp('entering')
              setCode('')
              setError(null)
            }}
            disabled={busy}
          >
            {emailOtp === 'sent' ? t('login.enterCode') : t('login.resendCode')}
          </Button>
          {emailOtp === 'entering' && (
            <>
              <Field
                label={t('login.verificationCodeLabel')}
                inputMode="numeric"
                autoComplete="one-time-code"
                autoFocus
                maxLength={6}
                value={code}
                disabled={busy}
                onChange={(e) => setCode(e.target.value.replace(/\D/g, ''))}
                placeholder="000000"
              />
              <FormError>{error}</FormError>
              <Button type="submit" disabled={busy || code.length < 6}>
                {busy ? t('login.verifying') : t('login.verify')}
              </Button>
            </>
          )}
          <button
            type="button"
            onClick={() => setEmailOtp(null)}
            className="text-sm font-medium text-muted hover:text-ink"
          >
            {t('login.backToTwoFactor')}
          </button>
        </form>
      </AuthLayout>
    )
  }

  return (
    <AuthLayout
      title={t('login.twoFactorTitle')}
      subtitle={
        recovery ? t('login.recoveryCodeSubtitle') : t('login.authenticatorSubtitle')
      }
      footer={
        <button type="button" onClick={onBack} className="font-medium text-primary hover:underline">
          {t('login.backToSignIn')}
        </button>
      }
    >
      <form onSubmit={onSubmit} className="flex flex-col gap-4">
        <Field
          label={recovery ? t('login.recoveryCodeLabel') : t('login.authCodeLabel')}
          inputMode={recovery ? 'text' : 'numeric'}
          autoComplete="one-time-code"
          autoFocus
          maxLength={recovery ? 20 : 6}
          value={code}
          disabled={busy}
          onChange={(e) =>
            setCode(recovery ? e.target.value : e.target.value.replace(/\D/g, ''))
          }
          placeholder={recovery ? 'xxxxx-xxxxx' : '000000'}
        />
        <FormError>{error}</FormError>
        <Button type="submit" disabled={busy || !code.trim()}>
          {busy ? t('login.verifying') : t('login.verify')}
        </Button>
        <button
          type="button"
          onClick={() => {
            setRecovery((v) => !v)
            setCode('')
            setError(null)
          }}
          className="text-sm font-medium text-muted hover:text-ink"
        >
          {recovery ? t('login.useAuthenticatorInstead') : t('login.useRecoveryInstead')}
        </button>
        <hr className="border-line" />
        {browserSupportsWebAuthn() && (
          <Button type="button" variant="ghost" onClick={usePasskey} disabled={busy}>
            {t('login.usePasskeyInstead')}
          </Button>
        )}
        <Button type="button" variant="ghost" onClick={sendEmailCode} disabled={busy}>
          {t('login.emailMeACode')}
        </Button>
      </form>
    </AuthLayout>
  )
}
